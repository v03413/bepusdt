package task

import (
	"bytes"
	"context"
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
	usdtTransfer = "0xa9059cbb" // Tether transfer function ID
	contentType  = "application/json"

	inputAddressTotal = 138 // USDT转账 input 正确总长度
	inputAddressStart = 34  // USDT转账 接收地址在input中的起始位置
	inputAddressEnd   = 74  // USDT转账 接收地址在input中的结束位置 amount在input中的起始位置
)

var chainBlockNum sync.Map
var nativeToken = map[string]string{
	conf.Bsc:      model.OrderTradeTypeBscBnb,
	conf.Xlayer:   model.OrderTradeTypeXlayerOkb,
	conf.Polygon:  model.OrderTradeTypePolygonPol,
	conf.Ethereum: model.OrderTradeTypeEthEth,
}
var contractMap = map[string]string{
	conf.UsdtXlayer:  model.OrderTradeTypeUsdtXlayer,
	conf.UsdtBep20:   model.OrderTradeTypeUsdtBep20,
	conf.UsdtPolygon: model.OrderTradeTypeUsdtPolygon,
	conf.UsdtErc20:   model.OrderTradeTypeUsdtErc20,
}

var client = &http.Client{Timeout: time.Second * 5}
var chainScanQueue = chanx.NewUnboundedChan[evmBlock](context.Background(), 30)

type decimals struct {
	Usdt   int32 // USDT小数位数
	Native int32 // 原生代币小数位数
}

type block struct {
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

	var post = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":1}`, n.Num))

	conf.SetBlockTotal(n.Network.Type)

	resp, err := client.Post(n.Network.Endpoint, contentType, bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn("Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn("Error reading response body:", err)

		return
	}

	var data = gjson.ParseBytes(body)
	if data.Get("error").Exists() {
		conf.SetBlockFail(n.Network.Type)
		chainScanQueue.In <- n
		log.Warn(fmt.Sprintf("%s getBlockByNumber response error %s %v", n.Network.Type, data.Get("error").String(), n))

		return
	}

	var result = data.Get("result")
	var timestamp = help.HexStr2Int(result.Get("timestamp").String())
	var transfers = make([]transfer, 0)
	for _, v := range result.Get("transactions").Array() {
		var recv = v.Get("to").String()
		var input = v.Get("input").String()
		var rawValue = v.Get("value").String()
		var rawAmount, ok = new(big.Int).SetString(rawValue, 0)
		if !ok {
			log.Warn("Error converting value to integer:" + " " + rawValue)

			return
		}

		var amount = decimal.NewFromBigInt(rawAmount, n.Network.Decimals.Native)
		var tradeType = getTradeType(n.Network.Type, recv, input, rawAmount)
		if tradeType == nil {

			continue
		}

		if !tradeType.Native && len(input) == inputAddressTotal { // usdt transfer
			rawAmount, ok = new(big.Int).SetString(input[inputAddressEnd:], 16)
			if !ok {
				log.Warn("Error converting amount(value)：" + input[inputAddressEnd:])

				continue
			}

			amount = decimal.NewFromBigInt(rawAmount, n.Network.Decimals.Usdt)
			recv = "0x" + input[inputAddressStart:inputAddressEnd] // USDT转账接收地址
		}

		transfers = append(transfers, transfer{
			Network:     n.Network.Type,
			FromAddress: v.Get("from").String(),
			RecvAddress: recv,
			Amount:      amount,
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
