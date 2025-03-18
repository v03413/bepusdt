package monitor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"github.com/v03413/bepusdt/app/telegram"
	"io"
	"math/big"
	"net/http"
	"time"
)

type polygonUsdtTransfer struct {
	From      string
	To        string
	Amount    float64
	Hash      string
	BlockNum  int64
	Timestamp time.Time
	TradeType string
}

var polygonLastBlockNumber int64
var polygonBlockScanQueue = chanx.NewUnboundedChan[int64](context.Background(), 30)
var polygonUsdtTransferQueue = chanx.NewUnboundedChan[[]polygonUsdtTransfer](context.Background(), 30)

const usdtPolygonContractAddress = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
const usdtPolygonTransferMethodID = "0xa9059cbb" // Function: transfer(address recipient, uint256 amount)
const contentType = "application/json"

func init() {
	RegisterSchedule(time.Second*3, polygonBlockNumber)
	RegisterSchedule(time.Second, polygonBlockScan)
	RegisterSchedule(time.Second, polygonUsdtTransferHandle)
}

func polygonUsdtTransferHandle(time.Duration) {
	for {
		select {
		case transfers := <-polygonUsdtTransferQueue.Out:
			var orders = getAllWaitingOrders()
			for _, t := range transfers {
				// 计算交易金额
				var amount, quant = parseTransAmount(t.Amount)

				// 判断金额是否在允许范围内
				if !inPaymentAmountRange(amount) {

					continue
				}

				// 判断是否存在对应订单
				order, is := orders[fmt.Sprintf("%s%v%s", t.To, quant, t.TradeType)]
				if !is {

					continue
				}

				// 有效期检测
				if !order.CreatedAt.Before(t.Timestamp) || !order.ExpiredAt.After(t.Timestamp) {

					continue
				}

				// 更新信息
				order.OrderUpdateTxInfo(t.BlockNum, t.From, t.Hash, t.Timestamp)

				// 标记成功
				order.MarkSuccess()

				go notify.Handle(order)             // 通知订单支付成功
				go telegram.SendTradeSuccMsg(order) // TG发送订单信息
			}
		}
	}
}

func polygonProcessBlock(n any) {
	var num = n.(int64)
	var url = config.GetPolygonRpcEndpoint()
	var client = &http.Client{Timeout: time.Second * 5}
	var post = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":1}`, num))

	resp, err := client.Post(url, contentType, bytes.NewBuffer(post))
	if err != nil {
		polygonBlockScanQueue.In <- num
		log.Warn("Error sending request:", err)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		polygonBlockScanQueue.In <- num
		log.Warn("Error reading response body:", err)

		return
	}

	defer resp.Body.Close()

	var data = gjson.ParseBytes(body)
	if data.Get("error").Exists() {
		polygonBlockScanQueue.In <- num
		log.Warn("Polygon getBlockByNumber response error ", data.Get("error").String())

		return
	}

	var result = data.Get("result")
	var timestamp = help.HexStr2Int(result.Get("timestamp").String())
	var transfers = make([]polygonUsdtTransfer, 0)
	for _, v := range result.Get("transactions").Array() {
		if v.Get("to").String() != usdtPolygonContractAddress {

			continue
		}

		var input = v.Get("input").String()
		var methodID = input[:10]
		if methodID != usdtPolygonTransferMethodID {

			continue
		}

		amount, ok := new(big.Int).SetString(input[74:], 16)
		if !ok {
			log.Warn("Error converting amount" + input[74:])

			continue
		}

		transfers = append(transfers, polygonUsdtTransfer{
			From:      v.Get("from").String(),
			To:        "0x" + input[34:74],
			Amount:    float64(amount.Int64()),
			Hash:      v.Get("hash").String(),
			BlockNum:  num,
			Timestamp: time.Unix(timestamp, 0),
			TradeType: model.OrderTradeTypeUsdtPolygon,
		})
	}

	log.Info("区块扫描完成", num, "POLYGON")

	if len(transfers) == 0 {

		return
	}

	polygonUsdtTransferQueue.In <- transfers
}

func polygonBlockScan(time.Duration) {
	p, err := ants.NewPoolWithFunc(8, polygonProcessBlock)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for n := range polygonBlockScanQueue.Out {
		if err := p.Invoke(n); err != nil {
			polygonBlockScanQueue.In <- n

			log.Warn("Error invoking process block:", err)
		}
	}
}

func polygonBlockNumber(d time.Duration) {
	var url = config.GetPolygonRpcEndpoint()
	var jsonData = []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	var client = &http.Client{Timeout: time.Second * 5}

	for range time.Tick(d) {
		resp, err := client.Post(url, contentType, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Warn("Error sending request:", err)

			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warn("Error reading response body:", err)

			continue
		}

		_ = resp.Body.Close()

		var res = gjson.ParseBytes(body)
		var number = help.HexStr2Int(res.Get("result").String())
		if number == 0 {

			continue
		}

		if config.GetTradeConfirmed() { // 暂且认为30个区块之前的交易已经被全网确认

			number = number - 30
		}

		// 首次启动
		if polygonLastBlockNumber == 0 {

			polygonLastBlockNumber = number - 1
		}

		// 区块高度没有变化
		if number <= polygonLastBlockNumber {

			continue
		}

		for n := polygonLastBlockNumber + 1; n <= number; n++ {

			polygonBlockScanQueue.In <- n
		}

		polygonLastBlockNumber = number
	}
}
