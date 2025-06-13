package task

import (
	"bytes"
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
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
	usdtTransfer = "0xa9059cbb" // Tether transfer function ID
	contentType  = "application/json"
)

var chainBlockNum sync.Map
var nativeToken = map[string]string{
	conf.Bsc:      model.OrderTradeTypeBscBnb,
	conf.Polygon:  model.OrderTradeTypePolygonPol,
	conf.Ethereum: model.OrderTradeTypeEthEth,
}
var contractMap = map[string]string{
	"0x63b7e5aE00cc6053358fb9b97B361372FbA10a5e": model.OrderTradeTypeUsdtBep20,
	"0xc2132d05d31c914a87c6611c10748aeb04b58e8f": model.OrderTradeTypeUsdtPolygon,
	"0xdac17f958d2ee523a2206206994597c13d831ec7": model.OrderTradeTypeUsdtErc20,
}
var chainScanQueue = chanx.NewUnboundedChan[evmBlock](context.Background(), 30)

type evmCfg struct {
	Type     string
	Endpoint string
}

type evmBlock struct {
	Network evmCfg
	Num     uint64
}

func init() {
	register(task{callback: evmBlockDispatch})
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

func evmBlockRoll(ctx context.Context) {
	var val = ctx.Value("cfg")
	if val == nil {
		log.Warn("evmBlockRoll: context value 'cfg' is nil")

		return
	}

	var cfg, ok = val.(evmCfg)
	if !ok {
		log.Warn("evmBlockRoll: context value 'cfg' is not of type evmCfg")

		return
	}

	var url = cfg.Endpoint
	var jsonData = []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	var client = &http.Client{Timeout: time.Second * 5}

	resp, err := client.Post(url, contentType, bytes.NewBuffer(jsonData))
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
	var now = help.HexStr2Int(res.Get("result").String()).Uint64()
	if now == 0 {

		return
	}

	if conf.GetTradeIsConfirmed() {

		now = now - numConfirmedSub
	}

	var lastBlockNumber uint64
	if v, ok := chainBlockNum.Load(cfg.Type); ok {

		lastBlockNumber = v.(uint64)
	}

	// 首次启动
	if lastBlockNumber == 0 {

		lastBlockNumber = now - 1
	}

	// 区块高度没有变化
	if now <= lastBlockNumber {

		return
	}

	for n := lastBlockNumber + 1; n <= now; n++ {

		chainScanQueue.In <- evmBlock{Num: n, Network: cfg}
	}

	chainBlockNum.Store(cfg.Type, now)
}

func evmBlockParse(b any) {
	var n, ok = b.(evmBlock)
	if !ok {
		log.Warn("evmBlockParse: received non-evmBlock type")

		return
	}

	var client = &http.Client{Timeout: time.Second * 5}
	var post = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":1}`, n.Num))

	conf.SetBlockTotal(n.Network.Type)

	resp, err := client.Post(n.Network.Endpoint, contentType, bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn("Error sending request:", err)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn("Error reading response body:", err)

		return
	}

	defer resp.Body.Close()

	var data = gjson.ParseBytes(body)
	if data.Get("error").Exists() {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn("Polygon getBlockByNumber response error ", data.Get("error").String())

		return
	}

	var result = data.Get("result")
	var timestamp = help.HexStr2Int(result.Get("timestamp").String())
	var transfers = make([]transfer, 0)
	for _, v := range result.Get("transactions").Array() {
		var recv = v.Get("to").String()
		var input = v.Get("input").String()
		var value = v.Get("value").String()
		var amount, ok = new(big.Int).SetString(value, 0)
		if !ok {
			log.Warn("Error converting value to integer:" + " " + value)

			return
		}

		var tradeType = getTradeType(n.Network.Type, recv, input, amount)
		if tradeType == nil {

			continue
		}

		if !tradeType.Native && len(input) == 138 { // usdt transfer
			amount, ok = new(big.Int).SetString(input[74:], 16)
			if !ok {
				log.Warn("Error converting amount(value)：" + input[74:])

				continue
			}

			recv = "0x" + input[34:74] // USDT转账接收地址
		}

		transfers = append(transfers, transfer{
			Network:     n.Network.Type,
			FromAddress: v.Get("from").String(),
			RecvAddress: recv,
			Amount:      *amount,
			TxHash:      v.Get("hash").String(),
			BlockNum:    n.Num,
			Timestamp:   time.Unix(timestamp.Int64(), 0),
			TradeType:   tradeType.Type,
		})
	}

	log.Info("区块扫描完成", n.Num, conf.GetBlockSuccRate(n.Network.Type), n.Network.Type)

	if len(transfers) > 0 {

		transferQueue.In <- transfers
	}
}

func getTradeType(net, to, input string, value *big.Int) *model.TradeType {
	var tradeType, ok = nativeToken[net]
	if ok && input == "0x" && value.Sign() == 1 { // 原生代币

		return &model.TradeType{Type: tradeType, Native: true}
	}

	// 触发合约
	tradeType, ok = contractMap[to]
	if ok && strings.HasPrefix(input, usdtTransfer) { // USDT转账

		return &model.TradeType{Type: tradeType, Native: false}
	}

	// 其他合约数据
	return nil
}
