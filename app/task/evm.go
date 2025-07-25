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
	contentType      = "application/json"
)

var chainBlockNum sync.Map
var contractMap = map[string]string{
	conf.UsdtXlayer:  model.OrderTradeTypeUsdtXlayer,
	conf.UsdtBep20:   model.OrderTradeTypeUsdtBep20,
	conf.UsdtPolygon: model.OrderTradeTypeUsdtPolygon,
	conf.UsdtErc20:   model.OrderTradeTypeUsdtErc20,
}
var chainUsdtMap = map[string]string{
	conf.Bsc:      model.OrderTradeTypeUsdtBep20,
	conf.Xlayer:   model.OrderTradeTypeUsdtXlayer,
	conf.Polygon:  model.OrderTradeTypeUsdtPolygon,
	conf.Ethereum: model.OrderTradeTypeUsdtErc20,
	conf.Solana:   model.OrderTradeTypeUsdtSolana,
	conf.Aptos:    model.OrderTradeTypeUsdtAptos,
}

var client = &http.Client{Timeout: time.Second * 30}
var chainScanQueue = chanx.NewUnboundedChan[[]evmBlock](context.Background(), 30)

type decimals struct {
	Usdt   int32 // USDT小数位数
	Native int32 // 原生代币小数位数
}

type block struct {
	InitStartOffset int64 // 首次偏移量，第一次启动时，区块高度需要叠加此值，设置为负值可解决部分已创建但未超时(未扫描)的订单问题
	RollDelayOffset int64 // 延迟偏移量，某些RPC节点如果不延迟，会报错 block is out of range，目前发现 https://rpc.xlayer.tech/ 存在此问题
	ConfirmedOffset int64 // 确认偏移量，开启交易确认后，区块高度需要减去此值认为交易已确认
}

type evmCfg struct {
	Type     string
	Endpoint string
	Decimals decimals
	Block    block
}

type evmBlock struct {
	Network evmCfg
	Num     int64
}

func init() {
	register(task{callback: evmBlockDispatch})
}

func evmBlockRoll(ctx context.Context) {
	val := ctx.Value("cfg")
	if val == nil {
		log.Warn("evmBlockRoll: context value 'cfg' is nil")

		return
	}

	cfg, ok := val.(evmCfg)
	if !ok {
		log.Warn("evmBlockRoll: context value 'cfg' is not of type evmCfg")

		return
	}

	if rollBreak(cfg.Type) {

		return
	}

	post := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	resp, err := client.Post(cfg.Endpoint, contentType, bytes.NewBuffer(post))
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
	var now = help.HexStr2Int(res.Get("result").String()).Int64() - cfg.Block.RollDelayOffset
	if now <= 0 {

		return
	}

	if conf.GetTradeIsConfirmed() {

		now = now - cfg.Block.ConfirmedOffset
	}

	var lastBlockNumber int64
	if v, ok := chainBlockNum.Load(cfg.Type); ok {

		lastBlockNumber = v.(int64)
	}

	if now-lastBlockNumber > conf.BlockHeightMaxDiff {

		lastBlockNumber = evmBlockInitOffset(now, cfg.Block.InitStartOffset, cfg) - 1
	}

	chainBlockNum.Store(cfg.Type, now)
	if now <= lastBlockNumber { // 区块高度没有变化

		return
	}

	blocks := make([]evmBlock, 0)
	for n := lastBlockNumber + 1; n <= now; n++ {
		blocks = append(blocks, evmBlock{Num: n, Network: cfg})
		if len(blocks) >= blockParseMaxNum {
			chainScanQueue.In <- blocks
			blocks = make([]evmBlock, 0)
		}
	}

	if len(blocks) > 0 {
		chainScanQueue.In <- blocks
	}
}

func evmBlockInitOffset(now, offset int64, cfg evmCfg) int64 {
	go func() {
		var blocks []evmBlock
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for b := now; b >= now+offset; b-- {
			if rollBreak(cfg.Type) {

				return
			}

			blocks = append(blocks, evmBlock{Num: b, Network: cfg})
			if len(blocks) >= blockParseMaxNum {
				chainScanQueue.In <- blocks
				blocks = blocks[:0]
			}

			<-ticker.C
		}
		if len(blocks) > 0 {
			chainScanQueue.In <- blocks
		}
	}()

	return now
}

func evmBlockDispatch(context.Context) {
	p, err := ants.NewPoolWithFunc(8, evmBlockParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for n := range chainScanQueue.Out {
		if err := p.Invoke(n); err != nil {
			chainScanQueue.In <- n

			log.Warn("evmBlockDispatch Error invoking process block:", err)
		}
	}
}

func evmBlockParse(b any) {
	var blocks, ok = b.([]evmBlock)
	if !ok {
		log.Warn("evmBlockParse: received non-evmBlock type")

		return
	}

	first := blocks[0]
	items := make([]string, 0)
	for _, v := range blocks {
		conf.SetBlockTotal(v.Network.Type)
		items = append(items, fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":%d}`, v.Num, v.Num))
	}

	post := []byte(fmt.Sprintf(`[%s]`, strings.Join(items, ",")))
	resp, err := client.Post(first.Network.Endpoint, contentType, bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(first.Network.Type)
		chainScanQueue.In <- blocks
		log.Warn("Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(first.Network.Type)
		chainScanQueue.In <- blocks
		log.Warn("Error reading response body:", err)

		return
	}

	var list = gjson.ParseBytes(body).Array()
	for _, data := range list {
		if !data.Get("result").Exists() {
			conf.SetBlockFail(first.Network.Type)
			chainScanQueue.In <- []evmBlock{{Network: first.Network, Num: data.Get("id").Int()}}
			log.Warn(fmt.Sprintf("%s getBlockByNumber response error %s", first.Network.Type, data.String()))

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
				fmt.Println("解码错误:", err)
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
				recv, amount = parseUsdtContractTransfer(input)
			}

			if bytes.Equal(input[0:4], []byte{0x23, 0xb8, 0x72, 0xdd}) { // transfer from function ID
				from, recv, amount = parseUsdtContractTransferFrom(input)
			}

			if amount == nil {

				continue
			}

			transfers = append(transfers, transfer{
				Network:     first.Network.Type,
				FromAddress: from,
				RecvAddress: recv,
				Amount:      decimal.NewFromBigInt(amount, first.Network.Decimals.Usdt),
				TxHash:      v.Get("hash").String(),
				BlockNum:    num,
				Timestamp:   timestamp,
				TradeType:   tradeType,
			})
		}

		if len(transfers) > 0 {

			transferQueue.In <- transfers
		}

		log.Info("区块扫描完成", num, conf.GetBlockSuccRate(first.Network.Type), first.Network.Type)
	}
}

func parseUsdtContractTransfer(data []byte) (string, *big.Int) {
	if len(data) < 68 {

		return "", nil
	}

	receiver := hex.EncodeToString(data[16:36])
	amount := big.NewInt(0).SetBytes(data[36:68])

	return "0x" + receiver, amount
}

func parseUsdtContractTransferFrom(data []byte) (string, string, *big.Int) {
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
