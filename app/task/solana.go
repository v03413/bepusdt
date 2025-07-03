package task

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
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
	solanaBlockConfirmedOffset = 30
	solanaBlockInitStartOffset = -30
)

var solanaLastBlockNum int64
var solanaSlotQueue = chanx.NewUnboundedChan[int64](context.Background(), 30)
var solanaUsdtMint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
var solanaSplToken = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"

// 参考文档
//  - https://solana.com/zh/docs/rpc
//  - https://github.com/solana-program/token/blob/main/program/src/instruction.rs

func init() {
	register(task{callback: solanaSlotDispatch})
	register(task{callback: solanaSlotRoll, duration: time.Second * 5})
}

func solanaSlotRoll(context.Context) {
	post := []byte(`{"jsonrpc":"2.0","id":1,"method":"getSlot"}`)

	resp, err := client.Post(conf.GetSolanaRpcEndpoint(), contentType, bytes.NewBuffer(post))
	if err != nil {
		log.Warn("solanaSlotRoll Error sending request:", err)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Warn("solanaSlotRoll Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("solanaSlotRoll Error reading response body:", err)

		return
	}

	slot := gjson.GetBytes(body, "result").Int()
	if slot <= 0 {
		log.Warn("solanaSlotRoll Error: invalid slot number:", slot)

		return
	}

	if conf.GetTradeIsConfirmed() {

		slot = slot - solanaBlockConfirmedOffset
	}

	// 首次启动
	if solanaLastBlockNum == 0 {

		solanaLastBlockNum = slot + solanaBlockInitStartOffset
	}

	// 区块高度没有变化
	if slot <= solanaLastBlockNum {

		return
	}

	// 待扫描区块入列
	for n := solanaLastBlockNum + 1; n <= slot; n++ {

		solanaSlotQueue.In <- n
	}

	solanaLastBlockNum = slot
}

func solanaSlotDispatch(ctx context.Context) {
	p, err := ants.NewPoolWithFunc(8, solanaBlockParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for {
		select {
		case slot := <-solanaSlotQueue.Out:
			if err := p.Invoke(slot); err != nil {
				solanaSlotQueue.In <- slot
				log.Warn("solanaSlotDispatch Error invoking process slot:", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Warn("solanaSlotDispatch context done:", err)
			}

			return
		}
	}
}

func solanaBlockParse(s any) {
	slot := s.(int64)
	post := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getBlock","params":[%d,{"encoding":"json","maxSupportedTransactionVersion":0,"transactionDetails":"full","rewards":false}]}`, slot))
	network := conf.Solana

	conf.SetBlockTotal(network)
	resp, err := client.Post(conf.GetSolanaRpcEndpoint(), contentType, bytes.NewBuffer(post))
	if err != nil {
		conf.SetBlockFail(network)
		log.Warn("solanaBlockParse Error sending request:", err)

		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		conf.SetBlockFail(network)
		log.Warn("solanaBlockParse Error response status code:", resp.StatusCode)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		conf.SetBlockFail(network)
		solanaSlotQueue.In <- slot
		log.Warn("solanaBlockParse Error reading response body:", err)

		return
	}

	timestamp := time.Unix(gjson.GetBytes(body, "result.blockTime").Int(), 0)

	for _, trans := range gjson.GetBytes(body, "result.transactions").Array() {
		hash := trans.Get("transaction.signatures.0").String()

		// 解析账号索引
		accountKeys := make([]string, 0)
		for _, key := range trans.Get("transaction.message.accountKeys").Array() {
			accountKeys = append(accountKeys, key.String())
		}
		for _, v := range []string{"readonly", "writable"} {
			for _, key := range trans.Get("meta.loadedAddresses." + v).Array() {

				accountKeys = append(accountKeys, key.String())
			}
		}

		// 查找SPL Token索引
		splTokenIndex := int64(-1)
		for i, v := range accountKeys {
			if v == solanaSplToken {
				splTokenIndex = int64(i)

				break
			}
		}

		// SPL Token的Mint地址，即不包含USDT交易信息
		if splTokenIndex == -1 {

			continue
		}

		// 解析 USDT Token 账户 【Token Address => Owner Address】
		usdtTokenAccountMap := make(map[string]string)
		for _, v := range []string{"postTokenBalances", "preTokenBalances"} {
			for _, itm := range trans.Get("meta." + v).Array() {
				if itm.Get("mint").String() != solanaUsdtMint || itm.Get("programId").String() != solanaSplToken {

					continue
				}

				usdtTokenAccountMap[accountKeys[itm.Get("accountIndex").Int()]] = itm.Get("owner").String()
			}
		}

		transArr := make([]transfer, 0)

		// 解析外部指令
		for _, instr := range trans.Get("transaction.message.instructions").Array() {
			if instr.Get("programIdIndex").Int() != splTokenIndex {

				continue
			}

			transArr = append(transArr, parseTransfer(instr, accountKeys, usdtTokenAccountMap))
		}

		// 解析内部指令
		innerInstructions := trans.Get("meta.innerInstructions").Array()
		if len(innerInstructions) == 0 {

			continue
		}

		for _, itm := range innerInstructions {
			for _, instr := range itm.Get("instructions").Array() {
				if instr.Get("programIdIndex").Int() != splTokenIndex {
					// 不是SPL Token的指令，即不会包含合约代表 transfer 的指令

					continue
				}

				transArr = append(transArr, parseTransfer(instr, accountKeys, usdtTokenAccountMap))
			}
		}

		// 过滤无关交易
		result := make([]transfer, 0)
		for _, t := range transArr {
			if t.FromAddress == "" || t.RecvAddress == "" || t.Amount.IsZero() {

				continue
			}

			t.TxHash = hash
			t.Network = conf.Solana
			t.BlockNum = slot
			t.Timestamp = timestamp
			t.TradeType = model.OrderTradeTypeUsdtSolana

			result = append(result, t)
		}

		if len(result) > 0 {
			transferQueue.In <- result
		}
	}

	log.Info("区块扫描完成", slot, conf.GetBlockSuccRate(network), network)
}

func parseTransfer(instr gjson.Result, accountKeys []string, usdtTokenAccountMap map[string]string) transfer {
	accounts := instr.Get("accounts").Array()
	trans := transfer{}
	if len(accounts) < 3 { // from to singer，至少存在3个账户索引，如果是多签则 > 3

		return trans
	}
	data := base58.Decode(instr.Get("data").String())
	if len(data) != 9 { // data 必然是9个字节

		return trans
	}

	// not transfer && transferChecked instruction
	if data[0] != 3 && data[0] != 12 {

		return trans
	}

	from, ok := usdtTokenAccountMap[accountKeys[accounts[0].Int()]]
	if !ok {

		return trans
	}

	trans.FromAddress = from
	trans.RecvAddress = usdtTokenAccountMap[accountKeys[accounts[1].Int()]]

	buf := make([]byte, 8)
	copy(buf[:], data[1:9])
	number := binary.LittleEndian.Uint64(buf)
	b := new(big.Int)
	b.SetUint64(number)
	trans.Amount = decimal.NewFromBigInt(b, -6) // USDT的精度是6位小数

	return trans
}
