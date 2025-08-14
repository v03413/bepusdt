package task

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

const (
	blockParseMaxNum = 10 // 每次解析区块的最大数量
	evmTransferEvent = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

var chainBlockNum sync.Map
var contractMap = map[string]string{
	conf.UsdtXlayer:   model.OrderTradeTypeUsdtXlayer,
	conf.UsdtBep20:    model.OrderTradeTypeUsdtBep20,
	conf.UsdtPolygon:  model.OrderTradeTypeUsdtPolygon,
	conf.UsdtArbitrum: model.OrderTradeTypeUsdtArbitrum,
	conf.UsdtErc20:    model.OrderTradeTypeUsdtErc20,
	conf.UsdcErc20:    model.OrderTradeTypeUsdcErc20,
	conf.UsdcPolygon:  model.OrderTradeTypeUsdcPolygon,
	conf.UsdcXlayer:   model.OrderTradeTypeUsdcXlayer,
	conf.UsdcArbitrum: model.OrderTradeTypeUsdcArbitrum,
	conf.UsdcBep20:    model.OrderTradeTypeUsdcBep20,
	conf.UsdcBase:     model.OrderTradeTypeUsdcBase,
}
var networkTokenMap = map[string][]string{
	conf.Bsc:      {model.OrderTradeTypeUsdtBep20, model.OrderTradeTypeUsdcBep20},
	conf.Xlayer:   {model.OrderTradeTypeUsdtXlayer, model.OrderTradeTypeUsdcXlayer},
	conf.Polygon:  {model.OrderTradeTypeUsdtPolygon, model.OrderTradeTypeUsdcPolygon},
	conf.Arbitrum: {model.OrderTradeTypeUsdtArbitrum, model.OrderTradeTypeUsdcArbitrum},
	conf.Ethereum: {model.OrderTradeTypeUsdtErc20, model.OrderTradeTypeUsdcErc20},
	conf.Base:     {model.OrderTradeTypeUsdcBase},
	conf.Solana:   {model.OrderTradeTypeUsdtSolana, model.OrderTradeTypeUsdcSolana},
	conf.Aptos:    {model.OrderTradeTypeUsdtAptos, model.OrderTradeTypeUsdcAptos},
}
var client = &http.Client{Timeout: time.Second * 30}
var decimals = map[string]int32{
	conf.UsdtXlayer:   conf.UsdtXlayerDecimals,
	conf.UsdtBep20:    conf.UsdtBscDecimals,
	conf.UsdtPolygon:  conf.UsdtPolygonDecimals,
	conf.UsdtArbitrum: conf.UsdtArbitrumDecimals,
	conf.UsdtErc20:    conf.UsdtEthDecimals,
	conf.UsdcErc20:    conf.UsdcEthDecimals,
	conf.UsdcPolygon:  conf.UsdcPolygonDecimals,
	conf.UsdcXlayer:   conf.UsdcXlayerDecimals,
	conf.UsdcArbitrum: conf.UsdcArbitrumDecimals,
	conf.UsdcBep20:    conf.UsdcBscDecimals,
	conf.UsdcBase:     conf.UsdcBaseDecimals,
	conf.UsdcAptos:    conf.UsdcAptosDecimals,
	conf.UsdtAptos:    conf.UsdtAptosDecimals,
}

type block struct {
	InitStartOffset int64 // 首次偏移量，第一次启动时，区块高度需要叠加此值，设置为负值可解决部分已创建但未超时(未扫描)的订单问题
	RollDelayOffset int64 // 延迟偏移量，某些RPC节点如果不延迟，会报错 block is out of range，目前发现 https://rpc.xlayer.tech/ 存在此问题
	ConfirmedOffset int64 // 确认偏移量，开启交易确认后，区块高度需要减去此值认为交易已确认
}

type evm struct {
	Network        string
	Endpoint       string
	Block          block
	blockScanQueue *chanx.UnboundedChan[evmBlock]
}

type evmBlock struct {
	From int64
	To   int64
}

func init() {
	//register(task{callback: evmBlockDispatch})
}

func (e *evm) blockRoll(ctx context.Context) {
	if rollBreak(e.Network) {

		return
	}

	post := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	req, err := http.NewRequestWithContext(ctx, "POST", e.Endpoint, bytes.NewBuffer(post))
	if err != nil {
		log.Warn("Error creating request:", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Warn("Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("Error reading response body:", err)

		return
	}

	var res = gjson.ParseBytes(body)
	var now = help.HexStr2Int(res.Get("result").String()).Int64() - e.Block.RollDelayOffset
	if now <= 0 {

		return
	}

	if conf.GetTradeIsConfirmed() {

		now = now - e.Block.ConfirmedOffset
	}

	var lastBlockNumber int64
	if v, ok := chainBlockNum.Load(e.Network); ok {

		lastBlockNumber = v.(int64)
	}

	if now-lastBlockNumber > conf.BlockHeightMaxDiff {
		lastBlockNumber = e.blockInitOffset(now, e.Block.InitStartOffset) - 1
	}

	chainBlockNum.Store(e.Network, now)
	if now <= lastBlockNumber {

		return
	}

	for from := lastBlockNumber + 1; from <= now; from += blockParseMaxNum {
		to := from + blockParseMaxNum - 1
		if to > now {
			to = now
		}

		e.blockScanQueue.In <- evmBlock{From: from, To: to}
	}
}

func (e *evm) blockInitOffset(now, offset int64) int64 {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for b := now; b > now+offset; b -= blockParseMaxNum {
			if rollBreak(e.Network) {

				return
			}

			e.blockScanQueue.In <- evmBlock{From: b - blockParseMaxNum + 1, To: b}

			<-ticker.C
		}
	}()

	return now
}

func (e *evm) blockDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(2, e.getBlockByNumber)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for {
		select {
		case <-ctx.Done():
			return
		case n := <-e.blockScanQueue.Out:
			if err := p.Invoke(n); err != nil {
				e.blockScanQueue.In <- n

				log.Warn("evmBlockDispatch Error invoking process block:", err)
			}
		}
	}
}

func (e *evm) getBlockByNumber(a any) {
	b, ok := a.(evmBlock)
	if !ok {
		log.Warn("evmBlockParse Error: expected []int64, got", a)

		return
	}

	items := make([]string, 0)
	for i := b.From; i <= b.To; i++ {
		items = append(items, fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",false],"id":%d}`, i, i))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", e.Endpoint, bytes.NewBuffer([]byte(fmt.Sprintf(`[%s]`, strings.Join(items, ",")))))
	if err != nil {
		log.Warn("Error creating request:", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		conf.SetBlockFail(e.Network)
		e.blockScanQueue.In <- b
		log.Warn("eth_getBlockByNumber Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(e.Network)
		e.blockScanQueue.In <- b
		log.Warn("eth_getBlockByNumber Error reading response body:", err)

		return
	}

	timestamp := make(map[string]time.Time)
	for _, itm := range gjson.ParseBytes(body).Array() {
		if itm.Get("error").Exists() {
			conf.SetBlockFail(e.Network)
			e.blockScanQueue.In <- b
			log.Warn(fmt.Sprintf("%s eth_getBlockByNumber response error %s", e.Network, itm.Get("error").String()))

			return
		}

		timestamp[itm.Get("result.number").String()] = time.Unix(help.HexStr2Int(itm.Get("result.timestamp").String()).Int64(), 0)
	}

	transfers, err := e.parseBlockTransfer(b, timestamp)
	if err != nil {
		conf.SetBlockFail(e.Network)
		e.blockScanQueue.In <- b
		log.Warn("evmBlockParse Error parsing block transfer:", err)

		return
	}

	if len(transfers) >= 0 {

		transferQueue.In <- transfers
	}

	log.Info("区块扫描完成", b, conf.GetBlockSuccRate(e.Network), e.Network)
}

func (e *evm) parseBlockTransfer(b evmBlock, timestamp map[string]time.Time) ([]transfer, error) {
	transfers := make([]transfer, 0)
	post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"fromBlock":"0x%x","toBlock":"0x%x","topics":["%s"]}],"id":1}`, b.From, b.To, evmTransferEvent))
	resp, err := client.Post(e.Endpoint, "application/json", bytes.NewBuffer(post))
	if err != nil {

		return transfers, errors.Join(errors.New("eth_getLogs Post Error"), err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {

		return transfers, errors.Join(errors.New("eth_getLogs ReadAll Error"), err)
	}

	data := gjson.ParseBytes(body)
	if data.Get("error").Exists() {

		return transfers, errors.New(fmt.Sprintf("%s eth_getLogs response error %s", e.Network, data.Get("error").String()))
	}

	for _, itm := range data.Get("result").Array() {
		to := itm.Get("address").String()
		tradeType, ok := contractMap[to]
		if !ok {

			continue
		}

		topics := itm.Get("topics").Array()
		if len(topics) < 3 {

			continue
		}

		if topics[0].String() != evmTransferEvent { // transfer event signature

			continue
		}

		from := fmt.Sprintf("0x%s", topics[1].String()[26:])
		recv := fmt.Sprintf("0x%s", topics[2].String()[26:])
		amount, ok := big.NewInt(0).SetString(itm.Get("data").String()[2:], 16)
		if !ok || amount.Sign() <= 0 {

			continue
		}

		blockNum, err := strconv.ParseInt(itm.Get("blockNumber").String(), 0, 64)
		if err != nil {
			log.Warn("evmBlockParse Error parsing block number:", err)

			continue
		}

		transfers = append(transfers, transfer{
			Network:     e.Network,
			FromAddress: from,
			RecvAddress: recv,
			Amount:      decimal.NewFromBigInt(amount, decimals[to]),
			TxHash:      itm.Get("transactionHash").String(),
			BlockNum:    blockNum,
			Timestamp:   timestamp[itm.Get("blockNumber").String()],
			TradeType:   tradeType,
		})
	}

	return transfers, nil
}

func (e *evm) tradeConfirmHandle(ctx context.Context) {
	var orders = getConfirmingOrders(networkTokenMap[e.Network])
	var wg sync.WaitGroup

	var handle = func(o model.TradeOrders) {
		post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["%s"],"id":1}`, o.TradeHash))
		req, err := http.NewRequestWithContext(ctx, "POST", e.Endpoint, bytes.NewBuffer(post))
		if err != nil {
			log.Warn("evm tradeConfirmHandle Error creating request:", err)

			return
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Warn("evm tradeConfirmHandle Error sending request:", err)

			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warn("evm tradeConfirmHandle Error reading response body:", err)

			return
		}

		data := gjson.ParseBytes(body)
		if data.Get("error").Exists() {
			log.Warn(fmt.Sprintf("%s eth_getTransactionReceipt response error %s", e.Network, data.Get("error").String()))

			return
		}

		if data.Get("result.status").String() == "0x1" {
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

func rollBreak(network string) bool {
	token, ok := networkTokenMap[network]
	if !ok {

		return true
	}

	var count int64 = 0
	model.DB.Model(&model.TradeOrders{}).Where("status = ? and trade_type in (?)", model.OrderStatusWaiting, token).Count(&count)
	if count > 0 {

		return false
	}

	model.DB.Model(&model.WalletAddress{}).Where("other_notify = ? and trade_type in (?)", model.OtherNotifyEnable, token).Count(&count)
	if count > 0 {

		return false
	}

	return true
}
