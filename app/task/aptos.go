package task

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

type aptos struct {
	versionChunkSize       int64
	versionConfirmedOffset int64
	versionInitStartOffset int64
	lastVersion            int64
	versionQueue           *chanx.UnboundedChan[version]
}

type version struct {
	Start int64
	Limit int64
}

var apt aptos

var aptDecimals = map[string]int32{
	model.OrderTradeTypeUsdtAptos: conf.UsdtAptosDecimals,
	model.OrderTradeTypeUsdcAptos: conf.UsdcAptosDecimals,
}

type aptEvent struct {
	Type    string
	Action  string
	Amount  decimal.Decimal
	Address string
}

type aptAmount struct {
	Amount string
	Type   string
}

func init() {
	apt = newAptos()
	register(task{callback: apt.versionDispatch})
	register(task{callback: apt.versionRoll, duration: time.Second * 3})
	register(task{callback: apt.tradeConfirmHandle, duration: time.Second * 5})
}

func newAptos() aptos {
	return aptos{
		versionChunkSize:       100, // 目前好像最大就只能100
		versionConfirmedOffset: 1000,
		versionInitStartOffset: -100 * 500,
		lastVersion:            0,
		versionQueue:           chanx.NewUnboundedChan[version](context.Background(), 30),
	}
}

func (a *aptos) versionRoll(context.Context) {
	if rollBreak(conf.Aptos) {

		return
	}

	resp, err := client.Get(conf.GetAptosRpcNode() + "/v1")
	if err != nil {
		log.Warn("aptos versionRoll Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Warn("aptos versionRoll Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("aptos versionRoll Error reading response body:", err)

		return
	}

	now := gjson.GetBytes(body, "ledger_version").Int()
	if now <= 0 {
		log.Warn("versionRoll Error: invalid ledger_version:", now)

		return
	}

	if conf.GetTradeIsConfirmed() {

		now = now - a.versionConfirmedOffset
	}

	if now-a.lastVersion > 10000 {
		a.versionInitOffset(now)
		a.lastVersion = now - a.versionChunkSize
	}

	var sub = now - a.lastVersion
	if now == 0 {

		return
	}

	if sub <= a.versionChunkSize {
		a.versionQueue.In <- version{Start: a.lastVersion, Limit: sub}
	} else {
		chunks := (sub + a.versionChunkSize - 1) / a.versionChunkSize
		for i := int64(0); i < chunks; i++ {
			limit := a.versionChunkSize
			start := a.lastVersion + a.versionChunkSize*i
			if i == chunks-1 {
				limit = sub % a.versionChunkSize
				if limit == 0 {
					limit = a.versionChunkSize
				}
			}

			a.versionQueue.In <- version{Start: start, Limit: limit}
		}
	}

	a.lastVersion = now
}

func (a *aptos) versionDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(3, a.versionParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for {
		select {
		case n := <-a.versionQueue.Out:
			if err := p.Invoke(n); err != nil {
				a.versionQueue.In <- n
				log.Warn("versionDispatch Error invoking process slot:", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Warn("versionDispatch context done:", err)
			}

			return
		}
	}
}

func (a *aptos) versionInitOffset(now int64) {
	if now == 0 || a.lastVersion != 0 {

		return
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		var end = now + a.versionInitStartOffset
		for s := now - a.versionChunkSize; s >= end; s = s - a.versionChunkSize {
			if rollBreak(conf.Aptos) {

				return
			}

			a.versionQueue.In <- version{Start: s, Limit: a.versionChunkSize}

			<-ticker.C
		}
	}()
}

// 由于 aptos 网络特性，交易数据中不会显示存在交易转账 from => to 的对应关系，
// 所以目前此解析函数存在大量循环嵌套解析，逻辑较为复杂，希望未来有更好的方式进行解析 慢慢优化
func (a *aptos) versionParse(n any) {
	p := n.(version)

	var net = conf.Aptos
	var url = fmt.Sprintf("%sv1/transactions?start=%d&limit=%d", conf.GetAptosRpcNode(), p.Start, p.Limit)

	conf.SetBlockTotal(net)
	resp, err := client.Get(url)
	if err != nil {
		conf.SetBlockFail(net)
		log.Warn("versionParse Error sending request:", err)

		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		conf.SetBlockFail(net)
		log.Warn("versionParse Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(net)
		a.versionQueue.In <- p
		log.Warn("versionParse Error reading response body:", err)

		return
	}

	if !gjson.ValidBytes(body) {
		conf.SetBlockFail(net)
		a.versionQueue.In <- p
		log.Warn("versionParse Error: invalid JSON response body")

		return
	}

	transfers := make([]transfer, 0)
	for _, trans := range gjson.ParseBytes(body).Array() {
		tsNano := trans.Get("timestamp").Int() * 1000
		timestamp := time.Unix(tsNano/1e9, tsNano%1e9)
		ver := trans.Get("version").Int()
		hash := trans.Get("hash").String()
		addrOwner := make(map[string]string)                                         // [address] => owner address
		addrType := make(map[string]string)                                          // [address] => tradeType
		amtAddrMap := map[string]map[aptAmount]string{"deposit": {}, "withdraw": {}} // [amount] => address
		aptEvents := make([]aptEvent, 0)
		trans.Get("changes").ForEach(func(_, v gjson.Result) bool {
			if v.Get("type").String() != "write_resource" {

				return true
			}

			data := v.Get("data")
			if data.Get("type").String() == "0x1::fungible_asset::FungibleStore" {
				addr := v.Get("address").String()
				switch data.Get("data.metadata.inner").String() {
				case conf.UsdtAptos:
					addrType[addr] = model.OrderTradeTypeUsdtAptos
				case conf.UsdcAptos:
					addrType[addr] = model.OrderTradeTypeUsdcAptos
				}
			}
			if data.Get("type").String() == "0x1::object::ObjectCore" {
				addrOwner[v.Get("address").String()] = data.Get("data.owner").String()
			}

			return true
		})
		trans.Get("events").ForEach(func(_, v gjson.Result) bool {
			amount := v.Get("data.amount").String()
			amt, err := decimal.NewFromString(amount)
			if err != nil {

				return true
			}

			address := v.Get("data.store").String()
			switch v.Get("type").String() {
			case "0x1::fungible_asset::Deposit":
				aptEvents = append(aptEvents, aptEvent{Amount: amt, Address: address, Action: "deposit"})
				amtAddrMap["deposit"][aptAmount{Amount: amount, Type: addrType[address]}] = address
			case "0x1::fungible_asset::Withdraw":
				amtAddrMap["withdraw"][aptAmount{Amount: amount, Type: addrType[address]}] = address
				aptEvents = append(aptEvents, aptEvent{Amount: amt, Address: address, Action: "withdraw"})
			}
			return true
		})

		// 针对 一个withdraw 对应 一个deposit 且数额相同的情况
		for amt, to := range amtAddrMap["deposit"] {
			from, ok := amtAddrMap["withdraw"][amt]
			if !ok {

				continue
			}

			amount, ok := new(big.Int).SetString(amt.Amount, 10)
			if !ok {

				continue
			}

			tradeType, ok := addrType[to]
			if !ok {

				continue
			}

			transfers = append(transfers, transfer{
				Network:     net,
				TxHash:      hash,
				Amount:      decimal.NewFromBigInt(amount, aptDecimals[tradeType]),
				FromAddress: a.padAddressLeadingZeros(addrOwner[from]),
				RecvAddress: a.padAddressLeadingZeros(addrOwner[to]),
				Timestamp:   timestamp,
				TradeType:   tradeType,
				BlockNum:    ver,
			})
		}

		// 针对 一个withdraw 对应 多个deposit(数额累计等于 withdraw) 的情况
		processEvents := func(tradeType string, events []aptEvent) ([]aptEvent, map[string]string) {
			deposits := make([]aptEvent, 0)
			withdraws := make(map[decimal.Decimal]aptEvent)
			fromMap := make(map[string]string)

			// 分类事件
			for _, e := range events {
				if addrType[e.Address] == tradeType {
					if e.Action == "deposit" {
						deposits = append(deposits, e)
					}
					if e.Action == "withdraw" {
						withdraws[e.Amount] = e
					}
				}
			}

			// 穷举计算匹配关系，只穷举 A + B = C 的情况，实际上还存在 A + B + C + ... = D
			// 大部分这种情况都是合约 swap 等交易，非普通人1对1转账，所以选择忽视
			for k1, e1 := range deposits {
				for k2, e2 := range deposits {
					if k1 == k2 {
						continue
					}
					for sum, e3 := range withdraws {
						if e1.Amount.Add(e2.Amount).Equal(sum) {
							fromMap[e1.Address] = e3.Address
						}
					}
				}
			}

			return deposits, fromMap
		}
		generateTransfers := func(deposits []aptEvent, fromMap map[string]string, tradeType string, decimals int32) {
			for _, to := range deposits {
				if from, ok := fromMap[to.Address]; ok {
					transfers = append(transfers, transfer{
						Network:     net,
						TxHash:      hash,
						Amount:      decimal.NewFromBigInt(to.Amount.BigInt(), decimals),
						FromAddress: a.padAddressLeadingZeros(addrOwner[from]),
						RecvAddress: a.padAddressLeadingZeros(addrOwner[to.Address]),
						Timestamp:   timestamp,
						TradeType:   tradeType,
						BlockNum:    ver,
					})
				}
			}
		}

		// 处理 USDT
		usdtDeposits, usdtFrom := processEvents(model.OrderTradeTypeUsdtAptos, aptEvents)
		generateTransfers(usdtDeposits, usdtFrom, model.OrderTradeTypeUsdtAptos, aptDecimals[model.OrderTradeTypeUsdtAptos])

		// 处理 USDC
		usdcDeposits, usdcFrom := processEvents(model.OrderTradeTypeUsdcAptos, aptEvents)
		generateTransfers(usdcDeposits, usdcFrom, model.OrderTradeTypeUsdcAptos, aptDecimals[model.OrderTradeTypeUsdcAptos])
	}

	if len(transfers) > 0 {

		transferQueue.In <- transfers
	}

	log.Info("区块扫描完成", fmt.Sprintf("%d.%d", p.Start, p.Limit), conf.GetBlockSuccRate(net), net)
}

func (a *aptos) padAddressLeadingZeros(addr string) string {
	addr = strings.TrimPrefix(addr, "0x")
	addr = strings.Repeat("0", 64-len(addr)) + addr

	return "0x" + addr
}

func (a *aptos) tradeConfirmHandle(ctx context.Context) {
	var orders = getConfirmingOrders(networkTokenMap[conf.Aptos])
	var wg sync.WaitGroup
	var ctx2, cancel = context.WithTimeout(context.Background(), time.Second*6)
	defer cancel()

	var handle = func(o model.TradeOrders) {
		resp, err := client.Get(conf.GetAptosRpcNode() + "v1/transactions/by_hash/" + o.TradeHash)
		if err != nil {
			log.Warn("aptos tradeConfirmHandle Error sending request:", err)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Warn("aptos tradeConfirmHandle Error response status code:", resp.StatusCode)

			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warn("aptos tradeConfirmHandle Error reading response body:", err)

			return
		}

		data := gjson.ParseBytes(body)
		if data.Get("error_code").Exists() {
			log.Warn("aptos tradeConfirmHandle Error:", data.Get("message").String())

			return
		}

		if data.Get("version").String() != "" &&
			data.Get("success").Bool() &&
			data.Get("vm_status").String() == "Executed successfully" {

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
