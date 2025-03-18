package monitor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"io"
	"math/big"
	"net/http"
	"time"
)

type polygonUsdtTransfer struct {
	From      string
	To        string
	Amount    string
	Hash      string
	BlockNum  int64
	Timestamp int64
}

var polygonLastBlockNumber int64
var polygonBlockScanQueue = chanx.NewUnboundedChan[int64](context.Background(), 30)
var polygonUsdtTransferQueue = chanx.NewUnboundedChan[polygonUsdtTransfer](context.Background(), 30)

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
		case t := <-polygonUsdtTransferQueue.Out:
			fmt.Println(t.Hash, t.Timestamp, t.From, t.To, t.Amount)
		}
	}
}

func polygonProcessBlock(n any) {
	var num = n.(int64)
	var url = config.GetPolygonRpcEndpoint()
	var client = &http.Client{Timeout: time.Second * 5}
	var data = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x",true],"id":1}`, num))

	resp, err := client.Post(url, contentType, bytes.NewBuffer(data))
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

	var result = gjson.ParseBytes(body).Get("result")
	var timestamp = help.HexStr2Int(result.Get("timestamp").String())
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

		polygonUsdtTransferQueue.In <- polygonUsdtTransfer{
			From:      v.Get("from").String(),
			To:        "0x" + input[34:74],
			Amount:    decimal.NewFromInt(amount.Int64()).Div(decimal.NewFromInt(1e6)).String(),
			Hash:      v.Get("hash").String(),
			BlockNum:  num,
			Timestamp: timestamp,
		}
	}
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
