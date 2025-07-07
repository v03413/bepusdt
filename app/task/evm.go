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
	blockParseMaxNum = 10 // 每次解析区块的最大数量

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

var client = &http.Client{Timeout: time.Second * 10}
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

	// 首次启动
	if lastBlockNumber == 0 {
		lastBlockNumber = evmBlockInitOffset(now, cfg.Block.InitStartOffset, cfg)
	}

	// 区块高度没有变化
	if now <= lastBlockNumber {

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

	chainBlockNum.Store(cfg.Type, now)
}

func evmBlockInitOffset(now, offset int64, cfg evmCfg) int64 {
	go func() {
		var blocks []evmBlock
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for b := now; b >= now+offset; b-- {
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
			recv := v.Get("to").String()
			input := v.Get("input").String()
			rawValue := v.Get("value").String()
			rawAmount, ok := new(big.Int).SetString(rawValue, 0)
			if !ok {
				log.Warn("Error converting value to integer:" + " " + rawValue)

				continue
			}

			amount := decimal.NewFromBigInt(rawAmount, first.Network.Decimals.Native)
			tradeType := getTradeType(first.Network.Type, recv, input, rawAmount)
			if tradeType == nil {

				continue
			}

			if !tradeType.Native && len(input) == inputAddressTotal { // usdt transfer
				rawAmount, ok = new(big.Int).SetString(input[inputAddressEnd:], 16)
				if !ok {
					log.Warn("Error converting amount(value)：" + input[inputAddressEnd:])

					continue
				}

				amount = decimal.NewFromBigInt(rawAmount, first.Network.Decimals.Usdt)
				recv = "0x" + input[inputAddressStart:inputAddressEnd] // USDT转账接收地址
			}

			transfers = append(transfers, transfer{
				Network:     first.Network.Type,
				FromAddress: v.Get("from").String(),
				RecvAddress: recv,
				Amount:      amount,
				TxHash:      v.Get("hash").String(),
				BlockNum:    num,
				Timestamp:   timestamp,
				TradeType:   tradeType.Type,
			})
		}

		if len(transfers) > 0 {

			transferQueue.In <- transfers
		}

		log.Info("区块扫描完成", num, conf.GetBlockSuccRate(first.Network.Type), first.Network.Type)
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
