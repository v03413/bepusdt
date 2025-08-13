package task

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

// 参考文档
//  - https://solana.com/zh/docs/rpc
//  - https://github.com/solana-program/token/blob/6d18ff73b1dd30703a30b1ca941cb0f1d18c2b2a/program/src/instruction.rs

type solana struct {
	slotConfirmedOffset int64
	slotInitStartOffset int64
	lastSlotNum         int64
	slotQueue           *chanx.UnboundedChan[int64]
}

type solanaTokenOwner struct {
	TradeType string
	Address   string
}

var sol solana

var solSplToken = map[string]string{
	conf.UsdtSolana: model.OrderTradeTypeUsdtSolana,
	conf.UsdcSolana: model.OrderTradeTypeUsdcSolana,
}

func init() {
	sol = newSolana()
	register(task{callback: sol.slotDispatch})
	register(task{callback: sol.slotRoll, duration: time.Second * 5})
	register(task{callback: sol.tradeConfirmHandle, duration: time.Second * 5})
}

func newSolana() solana {
	return solana{
		slotConfirmedOffset: 60,
		slotInitStartOffset: -600,
		lastSlotNum:         0,
		slotQueue:           chanx.NewUnboundedChan[int64](context.Background(), 30),
	}
}

func (s *solana) slotRoll(context.Context) {
	if rollBreak(conf.Solana) {

		return
	}

	post := []byte(`{"jsonrpc":"2.0","id":1,"method":"getSlot"}`)

	resp, err := client.Post(conf.GetSolanaRpcEndpoint(), "application/json", bytes.NewBuffer(post))
	if err != nil {
		log.Warn("slotRoll Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Warn("slotRoll Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("slotRoll Error reading response body:", err)

		return
	}

	now := gjson.GetBytes(body, "result").Int()
	if now <= 0 {
		log.Warn("slotRoll Error: invalid slot number:", now)

		return
	}

	if conf.GetTradeIsConfirmed() {

		now = now - s.slotConfirmedOffset
	}

	if now-s.lastSlotNum > conf.BlockHeightMaxDiff { // 区块高度变化过大，强制丢块重扫
		s.lastSlotNum = now
		s.slotInitOffset(now)
	}

	if now == s.lastSlotNum { // 区块高度没有变化

		return
	}

	for n := s.lastSlotNum + 1; n <= now; n++ {
		// 待扫描区块入列

		s.slotQueue.In <- n
	}

	s.lastSlotNum = now
}

func (s *solana) slotDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(3, s.slotParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for {
		select {
		case slot := <-s.slotQueue.Out:
			if err := p.Invoke(slot); err != nil {
				s.slotQueue.In <- slot
				log.Warn("slotDispatch Error invoking process slot:", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Warn("slotDispatch context done:", err)
			}

			return
		}
	}
}

func (s *solana) slotInitOffset(now int64) {
	if now == 0 || s.lastSlotNum != 0 {

		return
	}

	go func() {
		ticker := time.NewTicker(time.Millisecond * 300)
		defer ticker.Stop()

		for num := now; num >= now+s.slotInitStartOffset; num-- {
			if rollBreak(conf.Solana) {

				return
			}

			s.slotQueue.In <- num

			<-ticker.C
		}
	}()
}

func (s *solana) slotParse(n any) {
	slot := n.(int64)
	post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getBlock","params":[%d,{"encoding":"json","maxSupportedTransactionVersion":0,"transactionDetails":"full","rewards":false}]}`, slot))
	network := conf.Solana

	conf.SetBlockTotal(network)
	resp, err := client.Post(conf.GetSolanaRpcEndpoint(), "application/json", bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(network)
		log.Warn("slotParse Error sending request:", err)

		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		conf.SetBlockFail(network)
		log.Warn("slotParse Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(network)
		s.slotQueue.In <- slot
		log.Warn("slotParse Error reading response body:", err)

		return
	}

	timestamp := time.Unix(gjson.GetBytes(body, "result.blockTime").Int(), 0)

	for _, trans := range gjson.GetBytes(body, "result.transactions").Array() {
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

			continue
		}

		// 解析 Token 账户 【Token Address => Owner Address】
		tokenAccountMap := make(map[string]solanaTokenOwner)
		for _, v := range []string{"postTokenBalances", "preTokenBalances"} {
			for _, itm := range trans.Get("meta." + v).Array() {
				tradeType, ok := solSplToken[itm.Get("mint").String()]
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

		if len(result) > 0 {
			transferQueue.In <- result
		}
	}

	log.Info("区块扫描完成", slot, conf.GetBlockSuccRate(network), network)
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
	var orders = getConfirmingOrders(networkTokenMap[conf.Solana])
	var wg sync.WaitGroup
	var ctx2, cancel = context.WithTimeout(context.Background(), time.Second*6)
	defer cancel()

	var handle = func(o model.TradeOrders) {
		post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getSignatureStatuses","params":[["%s"],{"searchTransactionHistory":true}]}`, o.TradeHash))

		resp, err := client.Post(conf.GetSolanaRpcEndpoint(), "application/json", bytes.NewBuffer(post))
		if err != nil {
			log.Warn("solana tradeConfirmHandle Error sending request:", err)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Warn("solana tradeConfirmHandle Error response status code:", resp.StatusCode)

			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warn("solana tradeConfirmHandle Error reading response body:", err)

			return
		}

		data := gjson.ParseBytes(body)
		if data.Get("error").Exists() {
			log.Warn("solana tradeConfirmHandle Error:", data.Get("error").String())

			return
		}

		if data.Get("result.value.0.confirmationStatus").String() == "finalized" {

			markFinalConfirmed(o)
		}
	}

	for _, order := range orders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case <-ctx2.Done():
				return
			default:
				handle(order)
			}
		}()
	}

	wg.Wait()
}
