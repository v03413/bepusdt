package task

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"io"
	"math/big"
	"time"
)

const (
	aptosPayloadType = "entry_function_payload"
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

func init() {
	apt = newAptos()
	register(task{callback: apt.versionDispatch})
	register(task{callback: apt.versionRoll, duration: time.Second * 3})
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

func (a *aptos) versionParse(n any) {
	p := n.(version)

	var network = conf.Aptos
	var url = fmt.Sprintf("%sv1/transactions?start=%d&limit=%d", conf.GetAptosRpcNode(), p.Start, p.Limit)

	conf.SetBlockTotal(network)
	resp, err := client.Get(url)
	if err != nil {
		conf.SetBlockFail(network)
		log.Warn("versionParse Error sending request:", err)

		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		conf.SetBlockFail(network)
		log.Warn("versionParse Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(network)
		a.versionQueue.In <- p
		log.Warn("versionParse Error reading response body:", err)

		return
	}

	if !gjson.ValidBytes(body) {
		conf.SetBlockFail(network)
		a.versionQueue.In <- p
		log.Warn("versionParse Error: invalid JSON response body")

		return
	}

	for _, trans := range gjson.ParseBytes(body).Array() {
		tsNano := trans.Get("timestamp").Int() * 1000
		timestamp := time.Unix(tsNano/1e9, tsNano%1e9)
		function := trans.Get("payload.function").String()
		typeName := trans.Get("payload.type").String()
		if typeName != aptosPayloadType {

			continue
		}

		hash := trans.Get("hash").String()
		sender := trans.Get("sender").String()
		ver := trans.Get("version").Int()
		args := trans.Get("payload.arguments").Array()

		switch function {
		case "0x1::primary_fungible_store::transfer":
			a.parsePrimaryFungibleStoreTransfer(network, hash, sender, ver, timestamp, args)
		case "0x1::aptos_account::batch_transfer_fungible_assets":
			a.parseAptosAccountBatchTransferFungibleAssets(network, hash, sender, ver, timestamp, args)
		case "0x1::aptos_account::transfer_fungible_assets":
			a.parseAptosAccountTransferFungibleAssets(network, hash, sender, ver, timestamp, args)
		case "0x1::fungible_asset::transfer":
			// 待定
		case "0x1::coin::transfer":
			// 待定
		}
	}

	log.Info("区块扫描完成", fmt.Sprintf("%d.%d", p.Start, p.Limit), conf.GetBlockSuccRate(network), network)
}

func (a *aptos) parsePrimaryFungibleStoreTransfer(net, hash, sender string, ver int64, t time.Time, args []gjson.Result) {
	if args[0].Get("inner").String() != conf.UsdtAptos {

		return
	}

	rawAmount := new(big.Int)
	rawAmount.SetString(args[2].String(), 10)

	transferQueue.In <- []transfer{{
		Network:     net,
		TxHash:      hash,
		Amount:      decimal.NewFromBigInt(rawAmount, conf.UsdtAptosDecimals),
		FromAddress: sender,
		RecvAddress: args[1].String(),
		Timestamp:   t,
		TradeType:   model.OrderTradeTypeUsdtAptos,
		BlockNum:    ver,
	}}
}

func (a *aptos) parseAptosAccountBatchTransferFungibleAssets(net, hash, sender string, ver int64, t time.Time, args []gjson.Result) {
	if args[0].Get("inner").String() != conf.UsdtAptos {

		return
	}

	var result = make([]transfer, 0)
	for i, recv := range args[1].Array() {
		rawAmount := new(big.Int)
		rawAmount.SetString(args[2].Get(fmt.Sprintf("%d", i)).String(), 10)
		if rawAmount == nil || rawAmount.Sign() <= 0 {
			log.Warn("parseAptosAccountBatchTransferFungibleAssets Error: invalid amount for receiver", recv.String(), "in transaction", hash)

			continue
		}

		result = append(result, transfer{
			Network:     net,
			TxHash:      hash,
			Amount:      decimal.NewFromBigInt(rawAmount, conf.UsdtAptosDecimals),
			FromAddress: sender,
			RecvAddress: recv.String(),
			Timestamp:   t,
			TradeType:   model.OrderTradeTypeUsdtAptos,
			BlockNum:    ver,
		})
	}

	if len(result) > 0 {
		transferQueue.In <- result
	}
}

func (a *aptos) parseAptosAccountTransferFungibleAssets(net, hash, sender string, ver int64, t time.Time, args []gjson.Result) {
	if args[0].Get("inner").String() != conf.UsdtAptos {

		return
	}

	rawAmount := new(big.Int)
	rawAmount.SetString(args[2].String(), 10)

	transferQueue.In <- []transfer{
		{
			Network:     net,
			TxHash:      hash,
			Amount:      decimal.NewFromBigInt(rawAmount, conf.UsdtAptosDecimals),
			FromAddress: sender,
			RecvAddress: args[1].String(),
			Timestamp:   t,
			TradeType:   model.OrderTradeTypeUsdtAptos,
			BlockNum:    ver,
		},
	}
}
