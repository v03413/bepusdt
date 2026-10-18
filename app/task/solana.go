package task

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	blockapi "github.com/v03413/bepusdt/app/core"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/utils"
)

// 参考文档
//  - https://solana.com/zh/docs/rpc
//  - https://github.com/solana-program/token/blob/6d18ff73b1dd30703a30b1ca941cb0f1d18c2b2a/program/src/instruction.rs

type solana struct {
	slotConfirmedOffset int
	lastSlotNum         int
	slotQueue           *chanx.UnboundedChan[int]
	client              *http.Client
	retryMu             sync.Mutex
	retryAttempts       map[int]int
	retryScheduled      map[int]*time.Timer
}

type solanaTokenOwner struct {
	TradeType model.TradeType
	Address   string
}

var sol solana

func init() {
	sol = newSolana()
	Register(Task{Callback: sol.slotDispatch})
	Register(Task{Callback: sol.syncSlotForward, Duration: time.Second * 5})
	Register(Task{Callback: sol.tradeConfirmHandle, Duration: time.Second * 5})
	Register(Task{Callback: sol.lookbackSlots, Duration: time.Second * 15})
	Register(Task{Callback: sol.reconcileExpiredOrders, Duration: time.Minute})
}

func newSolana() solana {
	return solana{
		slotConfirmedOffset: 60,
		lastSlotNum:         0,
		slotQueue:           chanx.NewUnboundedChan[int](context.Background(), 30),
		client:              utils.NewHttpClient(),
		retryAttempts:       make(map[int]int),
		retryScheduled:      make(map[int]*time.Timer),
	}
}

func solanaGetBlockPayload(slot int) []byte {
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getBlock","params":[%d,{"encoding":"json","maxSupportedTransactionVersion":1,"transactionDetails":"full","rewards":false}]}`, slot))
}

func (s *solana) syncSlotForward(ctx context.Context) {
	if syncBreak(conf.Solana, s.slotQueue.Len()) {

		return
	}
	if s.pendingSlotRetryCount() > 0 {
		return
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", model.Endpoint(conf.Solana), bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","id":1,"method":"getSlot"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Task.Warn("syncSlotForward Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Task.Warn("syncSlotForward Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Task.Warn("syncSlotForward Error reading response body:", err)

		return
	}

	now := int(gjson.GetBytes(body, "result").Int())
	if now <= 0 {
		log.Task.Warn("syncSlotForward Error: invalid slot number:", now)

		return
	}

	if now-s.lastSlotNum > cast.ToInt(model.GetC(model.BlockHeightMaxDiff)) { // 区块高度变化过大，强制丢块重扫
		s.lastSlotNum = now
	}

	if now <= s.lastSlotNum { // 区块高度没有变化或节点返回旧高度

		return
	}

	// 只填充队列的可用容量；消费后由下一轮继续追赶，避免长时间故障恢复时瞬间过度扫描。
	for n := s.lastSlotNum + 1; n <= now && s.slotQueue.Len() < blockQueueLimit; n++ {
		s.slotQueue.In <- n
		s.lastSlotNum = n
	}
}

func (s *solana) slotDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(3, s.slotParse)
	if err != nil {
		log.Task.Warn("Error creating pool:", err)

		return
	}

	defer p.Release()

	for {
		select {
		case slot := <-s.slotQueue.Out:
			if err := p.Invoke(slot); err != nil {
				s.slotQueue.In <- slot
				log.Task.Warn("slotDispatch Error invoking process slot:", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Task.Warn("slotDispatch context done:", err)
			}

			return
		}
	}
}

func (s *solana) slotParse(n any) {
	slot := n.(int)
	post := solanaGetBlockPayload(slot)
	network := conf.Solana

	resp, err := s.client.Post(model.Endpoint(conf.Solana), "application/json", bytes.NewBuffer(post))
	if err != nil {
		conf.RecordFailure(network)
		s.scheduleSlotRetry(slot, 0)
		log.Task.Warn("slotParse Error sending request:", err)

		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		conf.RecordFailure(network)
		s.scheduleSlotRetry(slot, 0)
		log.Task.Warn("slotParse Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.RecordFailure(network)
		s.scheduleSlotRetry(slot, 0)
		log.Task.Warn("slotParse Error reading response body:", err)

		return
	}

	result, err := validateSolanaBlockResponse(body)
	if err != nil {
		conf.RecordFailure(network)
		s.scheduleSlotRetry(slot, 0)
		log.Task.Warn("slotParse Error JSON-RPC response:", err)

		return
	}

	conf.RecordSuccess(network, cast.ToString(slot))
	s.resetSlotRetry(slot)
	if result.Type == gjson.Null {
		log.Task.Info(fmt.Sprintf("区块已跳过(Solana) %d 成功率：%s", slot, conf.GetSuccessRate(network)))

		return
	}

	timestamp := time.Unix(result.Get("blockTime").Int(), 0)

	for _, trans := range result.Get("transactions").Array() {
		if transfers := s.parseTransaction(trans, slot, timestamp); len(transfers) > 0 {
			transferQueue.In <- transfers
		}
	}

	log.Task.Info(fmt.Sprintf("区块扫描完成(Solana) %d 成功率：%s", slot, conf.GetSuccessRate(network)))
}

// parseTransaction 解析单笔交易中的 USDT/USDC 转账；getBlock 返回的交易项与 getTransaction 的结果结构相同，都可以传入
func (s *solana) parseTransaction(trans gjson.Result, slot int, timestamp time.Time) []transfer {
	if !solanaTxSucceeded(trans.Get("meta.err")) {

		return nil
	}

	hash := trans.Get("transaction.signatures.0").String()

	// 解析账号索引
	accountKeys := make([]string, 0)
	for _, key := range trans.Get("transaction.message.accountKeys").Array() {
		accountKeys = append(accountKeys, key.String())
	}
	for _, v := range []string{"readonly", "writable"} {
		for _, key := range trans.Get("meta.loadedAddresses." + v).Array() {
			accountKeys = append(accountKeys, key.String())
		}
	}

	// 查找SPL Token索引
	splTokenIndex := int64(-1)
	for i, v := range accountKeys {
		if v == conf.SolSplToken {
			splTokenIndex = int64(i)

			break
		}
	}

	// SPL Token的Mint地址，即不包含 Token 交易信息
	if splTokenIndex == -1 {

		return nil
	}

	// 解析 Token 账户 【Token Wallet => Owner Wallet】
	tokenAccountMap := make(map[string]solanaTokenOwner)
	for _, v := range []string{"postTokenBalances", "preTokenBalances"} {
		for _, itm := range trans.Get("meta." + v).Array() {
			tradeType, ok := model.GetContractTrade(itm.Get("mint").String())
			if !ok || itm.Get("programId").String() != conf.SolSplToken {

				continue
			}

			tokenAccountMap[accountKeys[itm.Get("accountIndex").Int()]] = solanaTokenOwner{
				TradeType: tradeType,
				Address:   itm.Get("owner").String(),
			}
		}
	}

	transArr := make([]transfer, 0)

	// 解析外部指令
	for _, instr := range trans.Get("transaction.message.instructions").Array() {
		if instr.Get("programIdIndex").Int() != splTokenIndex {

			continue
		}

		transArr = append(transArr, s.parseTransfer(instr, accountKeys, tokenAccountMap))
	}

	// 解析内部指令
	for _, itm := range trans.Get("meta.innerInstructions").Array() {
		for _, instr := range itm.Get("instructions").Array() {
			if instr.Get("programIdIndex").Int() != splTokenIndex {

				continue
			}

			transArr = append(transArr, s.parseTransfer(instr, accountKeys, tokenAccountMap))
		}
	}

	// 过滤无关交易
	result := make([]transfer, 0)
	for _, t := range transArr {
		if t.FromAddress == "" || t.RecvAddress == "" || t.Amount.IsZero() {

			continue
		}

		t.TxHash = hash
		t.Network = conf.Solana
		t.BlockNum = slot
		t.Timestamp = timestamp

		result = append(result, t)
	}

	return result
}

// 执行失败的交易同样会被打包进区块，指令和代币余额也照常保留，但转账已经回滚；只有 err 为 null 才算执行成功
func solanaTxSucceeded(err gjson.Result) bool {
	return err.Exists() && err.Type == gjson.Null
}

// getSignatureStatuses 对执行失败的交易同样返回 finalized，必须同时检查 err
func solanaSignatureFinalized(status gjson.Result) bool {
	return status.Get("confirmationStatus").String() == "finalized" && solanaTxSucceeded(status.Get("err"))
}

// Solana JSON-RPC 错误码，参考 https://github.com/anza-xyz/agave/blob/master/rpc-client-api/src/custom_error.rs
const (
	solanaErrSlotSkipped                = -32007
	solanaErrLongTermStorageSlotSkipped = -32009
)

func validateSolanaBlockResponse(body []byte) (gjson.Result, error) {
	if !gjson.ValidBytes(body) {
		return gjson.Result{}, fmt.Errorf("invalid JSON response")
	}

	data := gjson.ParseBytes(body)
	if rpcErr := data.Get("error"); rpcErr.Exists() {
		switch rpcErr.Get("code").Int() {
		case solanaErrSlotSkipped, solanaErrLongTermStorageSlotSkipped: // 被跳过的 slot 没有区块，按空 slot 处理
			return gjson.Result{Type: gjson.Null}, nil
		}

		return gjson.Result{}, fmt.Errorf("RPC error: %s", rpcErr.Raw)
	}

	result := data.Get("result")
	if !result.Exists() {
		return gjson.Result{}, fmt.Errorf("missing result")
	}
	if result.Type != gjson.Null && !result.Get("transactions").Exists() {
		return gjson.Result{}, fmt.Errorf("missing result.transactions")
	}

	return result, nil
}

func (s *solana) scheduleSlotRetry(slot int, delay time.Duration) {
	s.retryMu.Lock()
	if _, ok := s.retryScheduled[slot]; ok {
		s.retryMu.Unlock()

		return
	}

	attempt := s.retryAttempts[slot] + 1
	s.retryAttempts[slot] = attempt
	if delay <= 0 {
		delay = solanaRetryDelay(attempt)
	}

	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		s.retryMu.Lock()
		if s.retryScheduled[slot] != timer {
			s.retryMu.Unlock()

			return
		}
		delete(s.retryScheduled, slot)
		s.retryMu.Unlock()

		if s.slotQueue.Len() >= blockQueueLimit {
			log.Task.Warn("Solana 重试延后，当前区块消费堆积数量：", s.slotQueue.Len())
			s.scheduleSlotRetry(slot, delay)

			return
		}

		s.slotQueue.In <- slot
	})
	s.retryScheduled[slot] = timer
	s.retryMu.Unlock()
}

func (s *solana) resetSlotRetry(slot int) {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()

	delete(s.retryAttempts, slot)
	if timer, ok := s.retryScheduled[slot]; ok {
		timer.Stop()
		delete(s.retryScheduled, slot)
	}
}

func (s *solana) pendingSlotRetryCount() int {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()

	return len(s.retryAttempts)
}

func solanaRetryDelay(attempt int) time.Duration {
	delays := [...]time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute}
	if attempt < 1 {
		attempt = 1
	}
	if attempt > len(delays) {
		attempt = len(delays)
	}

	return delays[attempt-1]
}

func (s *solana) parseTransfer(instr gjson.Result, accountKeys []string, tokenAccountMap map[string]solanaTokenOwner) transfer {
	accounts := instr.Get("accounts").Array()
	trans := transfer{}
	if len(accounts) < 3 { // from to singer，至少存在3个账户索引，如果是多签则 > 3

		return trans
	}

	data := base58.Decode(instr.Get("data").String())
	dLen := len(data)
	if dLen < 9 {

		return trans
	}

	isTransfer := data[0] == 3 && dLen == 9
	isTransferChecked := data[0] == 12 && dLen == 10
	if !isTransfer && !isTransferChecked {

		return trans
	}

	var exp int32 = -6
	if isTransferChecked {
		exp = int32(data[9]) * -1
	}

	from, ok := tokenAccountMap[accountKeys[accounts[0].Int()]]
	if !ok {

		return trans
	}

	trans.FromAddress = from.Address
	trans.RecvAddress = tokenAccountMap[accountKeys[accounts[1].Int()]].Address
	if isTransferChecked {
		trans.RecvAddress = tokenAccountMap[accountKeys[accounts[2].Int()]].Address
	}

	buf := make([]byte, 8)
	copy(buf[:], data[1:9])
	number := binary.LittleEndian.Uint64(buf)
	b := new(big.Int)
	b.SetUint64(number)
	trans.TradeType = from.TradeType
	trans.Amount = decimal.NewFromBigInt(b, exp)

	return trans
}

func (s *solana) tradeConfirmHandle(ctx context.Context) {
	var orders = getConfirmingOrders(model.GetNetworkTrades(conf.Solana))
	var wg sync.WaitGroup

	var handle = func(o model.Order) {
		if model.GetC(model.BlockOffsetConfirm) == "1" {
			if s.lastSlotNum == 0 {
				return
			}
			if s.lastSlotNum-o.RefBlockNum < s.slotConfirmedOffset {
				return
			}
		}

		post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getSignatureStatuses","params":[["%s"],{"searchTransactionHistory":true}]}`, o.RefHash))
		req, _ := http.NewRequestWithContext(ctx, "POST", model.Endpoint(conf.Solana), bytes.NewBuffer(post))
		resp, err := s.client.Do(req)
		if err != nil {
			log.Task.Warn("solana tradeConfirmHandle Error sending request:", err)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Task.Warn("solana tradeConfirmHandle Error response status code:", resp.StatusCode)

			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Task.Warn("solana tradeConfirmHandle Error reading response body:", err)

			return
		}

		data := gjson.ParseBytes(body)
		if data.Get("error").Exists() {
			log.Task.Warn("solana tradeConfirmHandle Error:", data.Get("error").String())

			return
		}

		if solanaSignatureFinalized(data.Get("result.value.0")) {

			markFinalConfirmed(o)
		}
	}

	for _, order := range orders {
		wg.Add(1)
		go func() {
			defer wg.Done()

			handle(order)
		}()
	}

	wg.Wait()
}

func (s *solana) lookbackSlots(ctx context.Context) {
	if syncBreak(conf.Solana, s.slotQueue.Len()) {
		return
	}

	startAt, endAt, ok := getLookbackUnix(conf.Solana)
	if !ok {
		return
	}

	start, end := blockapi.New().GetBoundaryHeights(startAt, endAt, conf.Solana)
	for i := int(start); i <= int(end); i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if syncBreak(conf.Solana, s.slotQueue.Len()) {
			return
		}
		s.slotQueue.In <- i
		time.Sleep(time.Millisecond * 200)
	}
}
