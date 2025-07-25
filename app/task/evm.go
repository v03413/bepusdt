package task

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	blockParseMaxNum = 10 // 每次解析区块的最大数量
)

var chainBlockNum sync.Map
var contractMap = map[string]string{
	conf.UsdtXlayer:   model.OrderTradeTypeUsdtXlayer,
	conf.UsdtBep20:    model.OrderTradeTypeUsdtBep20,
	conf.UsdtPolygon:  model.OrderTradeTypeUsdtPolygon,
	conf.UsdtArbitrum: model.OrderTradeTypeUsdtArbitrum,
	conf.UsdtErc20:    model.OrderTradeTypeUsdtErc20,
}
var chainUsdtMap = map[string]string{
	conf.Bsc:      model.OrderTradeTypeUsdtBep20,
	conf.Xlayer:   model.OrderTradeTypeUsdtXlayer,
	conf.Polygon:  model.OrderTradeTypeUsdtPolygon,
	conf.Arbitrum:  model.OrderTradeTypeUsdtArbitrum,
	conf.Ethereum: model.OrderTradeTypeUsdtErc20,
	conf.Solana:   model.OrderTradeTypeUsdtSolana,
	conf.Aptos:    model.OrderTradeTypeUsdtAptos,
}
var client = &http.Client{Timeout: time.Second * 30}

type decimals struct {
	Usdt   int32 // USDT小数位数
	Native int32 // 原生代币小数位数
}

type block struct {
	InitStartOffset int64 // 首次偏移量，第一次启动时，区块高度需要叠加此值，设置为负值可解决部分已创建但未超时(未扫描)的订单问题
	RollDelayOffset int64 // 延迟偏移量，某些RPC节点如果不延迟，会报错 block is out of range，目前发现 https://rpc.xlayer.tech/ 存在此问题
	ConfirmedOffset int64 // 确认偏移量，开启交易确认后，区块高度需要减去此值认为交易已确认
}

type evm struct {
	Type           string
	Endpoint       string
	Decimals       decimals
	Block          block
	blockScanQueue *chanx.UnboundedChan[[]int64]
}

func init() {
	//register(task{callback: evmBlockDispatch})
}

func (e *evm) blockRoll(ctx context.Context) {
	if rollBreak(e.Type) {

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
	if v, ok := chainBlockNum.Load(e.Type); ok {

		lastBlockNumber = v.(int64)
	}

	if now-lastBlockNumber > conf.BlockHeightMaxDiff {

		lastBlockNumber = e.blockInitOffset(now, e.Block.InitStartOffset) - 1
	}

	chainBlockNum.Store(e.Type, now)
	if now <= lastBlockNumber { // 区块高度没有变化

		return
	}

	blocks := make([]int64, 0)
	for n := lastBlockNumber + 1; n <= now; n++ {
		blocks = append(blocks, n)
		if len(blocks) >= blockParseMaxNum {
			e.blockScanQueue.In <- blocks
			blocks = make([]int64, 0)
		}
	}

	if len(blocks) > 0 {
		e.blockScanQueue.In <- blocks
	}
}

func (e *evm) blockInitOffset(now, offset int64) int64 {
	go func() {
		var blocks []int64
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for b := now; b >= now+offset; b-- {
			if rollBreak(e.Type) {

				return
			}

			blocks = append(blocks, b)
			if len(blocks) >= blockParseMaxNum {
				e.blockScanQueue.In <- blocks
				blocks = blocks[:0]
			}

			<-ticker.C
		}
		if len(blocks) > 0 {
			e.blockScanQueue.In <- blocks
		}
	}()

	return now
}

func (e *evm) blockDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(2, e.blockParse)
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

func (e *evm) blockParse(a any) {
	blocks, ok := a.([]int64)
	if !ok {
		log.Warn("evmBlockParse Error: expected []int64, got", a)

		return
	}

	items := make([]string, 0)
	for _, v := range blocks {
		conf.SetBlockTotal(e.Type)
		items = append(items, fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":%d}`, v, v))
	}

	post := []byte(fmt.Sprintf(`[%s]`, strings.Join(items, ",")))
	resp, err := client.Post(e.Endpoint, "application/json", bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(e.Type)
		e.blockScanQueue.In <- blocks
		log.Warn("Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(e.Type)
		e.blockScanQueue.In <- blocks
		log.Warn("Error reading response body:", err)

		return
	}

	var list = gjson.ParseBytes(body).Array()
	for _, data := range list {
		if !data.Get("result").Exists() {
			conf.SetBlockFail(e.Type)
			e.blockScanQueue.In <- []int64{data.Get("id").Int()}
			log.Warn(fmt.Sprintf("%s getBlockByNumber response error %s", e.Type, data.String()))

			continue
		}

		num := data.Get("id").Int()
		result := data.Get("result")
		timestamp := time.Unix(help.HexStr2Int(result.Get("timestamp").String()).Int64(), 0)
		transfers := make([]transfer, 0)
		for _, v := range result.Get("transactions").Array() {
			to := v.Get("to").String()
			input, err := hex.DecodeString(strings.TrimPrefix(v.Get("input").String(), "0x"))
			if err != nil {
				log.Warn("evmBlockParse Error:", err)

				return
			}

			tradeType, ok := contractMap[to]
			if !ok { // 暂时忽略掉EVM原生代币的转账，后续... 暂时也没计划

				continue
			}

			var recv string
			var from = v.Get("from").String()
			var amount *big.Int
			if bytes.Equal(input[0:4], []byte{0xa9, 0x05, 0x9c, 0xbb}) { // transfer function ID
				recv, amount = e.parseUsdtContractTransfer(input)
			}

			if bytes.Equal(input[0:4], []byte{0x23, 0xb8, 0x72, 0xdd}) { // transfer from function ID
				from, recv, amount = e.parseUsdtContractTransferFrom(input)
			}

			if amount == nil {

				continue
			}

			transfers = append(transfers, transfer{
				Network:     e.Type,
				FromAddress: from,
				RecvAddress: recv,
				Amount:      decimal.NewFromBigInt(amount, e.Decimals.Usdt),
				TxHash:      v.Get("hash").String(),
				BlockNum:    num,
				Timestamp:   timestamp,
				TradeType:   tradeType,
			})
		}

		if len(transfers) > 0 {

			transferQueue.In <- transfers
		}

		log.Info("区块扫描完成", num, conf.GetBlockSuccRate(e.Type), e.Type)
	}

}

func (e *evm) parseUsdtContractTransfer(data []byte) (string, *big.Int) {
	if len(data) < 68 {

		return "", nil
	}

	receiver := hex.EncodeToString(data[16:36])
	amount := big.NewInt(0).SetBytes(data[36:68])

	return "0x" + receiver, amount
}

func (e *evm) parseUsdtContractTransferFrom(data []byte) (string, string, *big.Int) {
	if len(data) < 100 {

		return "", "", nil
	}

	from := hex.EncodeToString(data[16:36])
	to := hex.EncodeToString(data[48:68])
	amount := big.NewInt(0).SetBytes(data[68:100])

	return "0x" + from, "0x" + to, amount
}

func rollBreak(network string) bool {
	usdt, ok := chainUsdtMap[network]
	if !ok {

		return true
	}

	var count int64 = 0
	model.DB.Model(&model.TradeOrders{}).Where("status = ? and trade_type = ?", model.OrderStatusWaiting, usdt).Count(&count)
	if count > 0 {

		return false
	}

	model.DB.Model(&model.WalletAddress{}).Where("other_notify = ? and trade_type = ?", model.OtherNotifyEnable, usdt).Count(&count)
	if count > 0 {

		return false
	}

	return true
}
