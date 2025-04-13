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
	"sync/atomic"
	"time"
)

var polygonLastBlockNumber int64
var polygonBlockScanQueue = chanx.NewUnboundedChan[int64](context.Background(), 30)

const usdtPolygonContractAddress = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
const usdtPolygonTransferMethodID = "0xa9059cbb" // Function: transfer(address recipient, uint256 amount)
const contentType = "application/json"

func init() {
	RegisterSchedule(time.Second*3, polygonBlockNumber)
	RegisterSchedule(time.Second, polygonBlockScan)
}

func polygonProcessBlock(n any) {
	var num = n.(int64)
	var url = conf.GetPolygonRpcEndpoint()
	var client = &http.Client{Timeout: time.Second * 5}
	var post = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":1}`, num))

	atomic.AddUint64(&conf.PolygonBlockScanTotal, 1)

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
	var transfers = make([]transfer, 0)
	for _, v := range result.Get("transactions").Array() {
		if v.Get("to").String() != usdtPolygonContractAddress {

			continue
		}

		var input = v.Get("input").String()
		if len(input) < 10 {
			// 我也不明白为什么这里会出现 len < 10 情况，只是有人反馈，暂时屏蔽避免panic https://github.com/v03413/bepusdt/issues/66

			continue
		}

		var methodID = input[:10]
		if methodID != usdtPolygonTransferMethodID {

			continue
		}

		amount, ok := new(big.Int).SetString(input[74:], 16)
		if !ok {
			log.Warn("Error converting amount" + input[74:])

			continue
		}

		transfers = append(transfers, transfer{
			FromAddress: v.Get("from").String(),
			RecvAddress: "0x" + input[34:74],
			Amount:      float64(amount.Int64()),
			TxHash:      v.Get("hash").String(),
			BlockNum:    num,
			Timestamp:   time.Unix(timestamp.Int64(), 0),
			TradeType:   model.OrderTradeTypeUsdtPolygon,
		})
	}

	atomic.AddUint64(&conf.PolygonBlockScanSucc, 1)

	log.Info("区块扫描完成", num, conf.GetPolygonScanSuccRate(), "POLYGON")

	if len(transfers) == 0 {

		return
	}

	transferQueue.In <- transfers
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
	var url = conf.GetPolygonRpcEndpoint()
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
		var now = help.HexStr2Int(res.Get("result").String()).Int64()
		if now == 0 {

			continue
		}

		if conf.GetTradeIsConfirmed() {

			now = now - numConfirmedSub
		}

		// 首次启动
		if polygonLastBlockNumber == 0 {

			polygonLastBlockNumber = now - 1
		}

		// 区块高度没有变化
		if now <= polygonLastBlockNumber {

			continue
		}

		for n := polygonLastBlockNumber + 1; n <= now; n++ {

			polygonBlockScanQueue.In <- n
		}

		polygonLastBlockNumber = now
	}
}
