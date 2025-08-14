package task

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/tronprotocol/api"
	"github.com/v03413/tronprotocol/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

var gasFreeUsdtTokenAddress = []byte{0xa6, 0x14, 0xf8, 0x03, 0xb6, 0xfd, 0x78, 0x09, 0x86, 0xa4, 0x2c, 0x78, 0xec, 0x9c, 0x7f, 0x77, 0xe6, 0xde, 0xd1, 0x3c}
var gasFreeOwnerAddress = []byte{0x41, 0x3b, 0x41, 0x50, 0x50, 0xb1, 0xe7, 0x9e, 0x38, 0x50, 0x7c, 0xb6, 0xe4, 0x8d, 0xac, 0xc2, 0x27, 0xaf, 0xfd, 0xd5, 0x0c}
var gasFreeContractAddress = []byte{0x41, 0x39, 0xdd, 0x12, 0xa5, 0x4e, 0x2b, 0xab, 0x7c, 0x82, 0xaa, 0x14, 0xa1, 0xe1, 0x58, 0xb3, 0x42, 0x63, 0xd2, 0xd5, 0x10}
var usdtTrc20ContractAddress = []byte{0x41, 0xa6, 0x14, 0xf8, 0x03, 0xb6, 0xfd, 0x78, 0x09, 0x86, 0xa4, 0x2c, 0x78, 0xec, 0x9c, 0x7f, 0x77, 0xe6, 0xde, 0xd1, 0x3c}
var usdcTrc20ContractAddress = []byte{0x41, 0x34, 0x87, 0xb6, 0x3d, 0x30, 0xb5, 0xb2, 0xc8, 0x7f, 0xb7, 0xff, 0xa8, 0xbc, 0xfa, 0xde, 0x38, 0xea, 0xac, 0x1a, 0xbe}
var trc20TokenDecimals = map[string]int32{
	model.OrderTradeTypeUsdtTrc20: conf.UsdtTronDecimals,
	model.OrderTradeTypeUsdcTrc20: conf.UsdcTronDecimals,
}
var grpcParams = grpc.ConnectParams{
	Backoff:           backoff.Config{BaseDelay: 1 * time.Second, MaxDelay: 30 * time.Second, Multiplier: 1.5},
	MinConnectTimeout: 1 * time.Minute,
}

type tron struct {
	blockConfirmedOffset int64
	blockInitStartOffset int64
	lastBlockNum         int64
	blockScanQueue       *chanx.UnboundedChan[int64]
}

var tr tron

func init() {
	tr = newTron()
	register(task{duration: time.Second, callback: tr.blockDispatch})
	register(task{duration: time.Second * 3, callback: tr.blockRoll})
	register(task{duration: time.Second * 5, callback: tr.tradeConfirmHandle})
}

func newTron() tron {
	return tron{
		blockConfirmedOffset: 30,   // 区块确认偏移量
		blockInitStartOffset: -400, // 大概为过去20分钟的区块高度
		lastBlockNum:         0,
		blockScanQueue:       chanx.NewUnboundedChan[int64](context.Background(), 30),
	}
}

func (t *tron) blockRoll(context.Context) {
	if t.rollBreak() {

		return
	}

	conn, err := grpc.NewClient(conf.GetTronGrpcNode(), grpc.WithConnectParams(grpcParams), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("grpc.NewClient", err)

		return
	}

	defer conn.Close()

	var client = api.NewWalletClient(conn)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	block, err1 := client.GetNowBlock2(ctx, nil)
	defer cancel()

	if err1 != nil {
		log.Warn("GetNowBlock2 超时：", err1)

		return
	}

	var now = block.BlockHeader.RawData.Number
	if conf.GetTradeIsConfirmed() {
		now = now - t.blockConfirmedOffset
	}

	// 区块高度变化过大，强制丢块重扫
	if now-t.lastBlockNum > conf.BlockHeightMaxDiff {
		t.blockInitOffset(now)
		t.lastBlockNum = now - 1
	}

	// 区块高度没有变化
	if now == t.lastBlockNum {

		return
	}

	// 待扫描区块入列
	for n := t.lastBlockNum + 1; n <= now; n++ {

		t.blockScanQueue.In <- n
	}

	t.lastBlockNum = now
}

func (t *tron) blockDispatch(context.Context) {
	p, err := ants.NewPoolWithFunc(3, t.blockParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for n := range t.blockScanQueue.Out {
		if err := p.Invoke(n); err != nil {
			t.blockScanQueue.In <- n

			log.Warn("Tron Error invoking process block:", err)
		}
	}
}

func (t *tron) blockParse(n any) {
	var num = n.(int64)
	var node = conf.GetTronGrpcNode()
	var conn *grpc.ClientConn
	var err error
	if conn, err = grpc.NewClient(node, grpc.WithConnectParams(grpcParams), grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		log.Error("grpc.NewClient", err)

		return
	}

	defer conn.Close()
	var client = api.NewWalletClient(conn)

	conf.SetBlockTotal(conf.Tron)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	bok, err2 := client.GetBlockByNum2(ctx, &api.NumberMessage{Num: num})
	cancel()
	if err2 != nil {
		conf.SetBlockFail(conf.Tron)
		t.blockScanQueue.In <- num
		log.Warn("GetBlockByNum2 Error", err2)

		return
	}

	var resources = make([]resource, 0)
	var transfers = make([]transfer, 0)
	var timestamp = time.UnixMilli(bok.GetBlockHeader().GetRawData().GetTimestamp())
	for _, trans := range bok.GetTransactions() {
		if !trans.Result.Result {

			continue
		}

		var itm = trans.GetTransaction()
		var id = hex.EncodeToString(trans.Txid)
		for _, contract := range itm.GetRawData().GetContract() {
			// 资源代理 DelegateResourceContract
			if contract.GetType() == core.Transaction_Contract_DelegateResourceContract {
				var foo = &core.DelegateResourceContract{}
				err := contract.GetParameter().UnmarshalTo(foo)
				if err != nil {

					continue
				}

				resources = append(resources, resource{
					ID:           id,
					Type:         core.Transaction_Contract_DelegateResourceContract,
					Balance:      foo.Balance,
					ResourceCode: foo.Resource,
					FromAddress:  t.base58CheckEncode(foo.OwnerAddress),
					RecvAddress:  t.base58CheckEncode(foo.ReceiverAddress),
					Timestamp:    timestamp,
				})
			}

			// 资源回收 UnDelegateResourceContract
			if contract.GetType() == core.Transaction_Contract_UnDelegateResourceContract {
				var foo = &core.UnDelegateResourceContract{}
				err := contract.GetParameter().UnmarshalTo(foo)
				if err != nil {

					continue
				}

				resources = append(resources, resource{
					ID:           id,
					Type:         core.Transaction_Contract_UnDelegateResourceContract,
					Balance:      foo.Balance,
					ResourceCode: foo.Resource,
					FromAddress:  t.base58CheckEncode(foo.OwnerAddress),
					RecvAddress:  t.base58CheckEncode(foo.ReceiverAddress),
					Timestamp:    timestamp,
				})
			}

			// TRX转账交易
			if contract.GetType() == core.Transaction_Contract_TransferContract {
				var foo = &core.TransferContract{}
				err := contract.GetParameter().UnmarshalTo(foo)
				if err != nil {

					continue
				}

				transfers = append(transfers, transfer{
					Network:     conf.Tron,
					TxHash:      id,
					Amount:      decimal.NewFromBigInt(new(big.Int).SetInt64(foo.Amount), -6),
					FromAddress: t.base58CheckEncode(foo.OwnerAddress),
					RecvAddress: t.base58CheckEncode(foo.ToAddress),
					Timestamp:   timestamp,
					TradeType:   model.OrderTradeTypeTronTrx,
					BlockNum:    cast.ToInt64(num),
				})
			}

			// 触发智能合约
			if contract.GetType() == core.Transaction_Contract_TriggerSmartContract {
				var foo = &core.TriggerSmartContract{}
				if err := contract.GetParameter().UnmarshalTo(foo); err != nil {

					continue
				}

				data := foo.GetData()

				// Gas Free 钱包 合约授权转账
				if bytes.Equal(foo.OwnerAddress, gasFreeOwnerAddress) && bytes.Equal(foo.ContractAddress, gasFreeContractAddress) {
					from, receiver, amount := t.gasFreePermitTransfer(data)
					if amount != nil {
						transfers = append(transfers, transfer{
							Network:     conf.Tron,
							TxHash:      id,
							Amount:      decimal.NewFromBigInt(amount, conf.UsdtTronDecimals),
							FromAddress: from,
							RecvAddress: receiver,
							Timestamp:   timestamp,
							TradeType:   model.OrderTradeTypeUsdtTrc20,
							BlockNum:    cast.ToInt64(num),
						})
					}
				}

				// trc20 合约解析
				var tradeType = "None"
				if bytes.Equal(foo.GetContractAddress(), usdtTrc20ContractAddress) {
					tradeType = model.OrderTradeTypeUsdtTrc20
				} else if bytes.Equal(foo.GetContractAddress(), usdcTrc20ContractAddress) {
					tradeType = model.OrderTradeTypeUsdcTrc20
				}

				exp, ok := trc20TokenDecimals[tradeType]
				if !ok {

					continue
				}

				if bytes.Equal(data[:4], []byte{0xa9, 0x05, 0x9c, 0xbb}) { //  a9059cbb transfer
					receiver, amount := t.parseTrc20ContractTransfer(data)
					if amount != nil {
						transfers = append(transfers, transfer{
							Network:     conf.Tron,
							TxHash:      id,
							Amount:      decimal.NewFromBigInt(amount, exp),
							FromAddress: t.base58CheckEncode(foo.OwnerAddress),
							RecvAddress: receiver,
							Timestamp:   timestamp,
							TradeType:   tradeType,
							BlockNum:    cast.ToInt64(num),
						})
					}
				}
				if bytes.Equal(data[:4], []byte{0x23, 0xb8, 0x72, 0xdd}) { //  transferFrom (23b872dd)
					from, to, amount := t.parseTrc20ContractTransferFrom(data)
					if amount != nil {
						transfers = append(transfers, transfer{
							Network:     conf.Tron,
							TxHash:      id,
							Amount:      decimal.NewFromBigInt(amount, exp),
							FromAddress: from,
							RecvAddress: to,
							Timestamp:   timestamp,
							TradeType:   tradeType,
							BlockNum:    cast.ToInt64(num),
						})
					}
				}
			}
		}
	}

	if len(transfers) > 0 {
		transferQueue.In <- transfers
	}

	if len(resources) > 0 {
		resourceQueue.In <- resources
	}

	log.Info("区块扫描完成", num, conf.GetBlockSuccRate(conf.Tron), conf.Tron)
}

func (t *tron) blockInitOffset(now int64) {
	if now == 0 || t.lastBlockNum != 0 {

		return
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		endOffset := now + t.blockInitStartOffset
		defer ticker.Stop()

		for num := now; num >= endOffset; {
			if t.rollBreak() {

				return
			}

			for i := 0; i < 10 && num >= endOffset; i++ {
				t.blockScanQueue.In <- num
				num--
			}

			<-ticker.C
		}
	}()
}

func (t *tron) parseTrc20ContractTransfer(data []byte) (string, *big.Int) {
	if len(data) != 68 {

		return "", nil
	}

	receiver := t.base58CheckEncode(append([]byte{0x41}, data[16:36]...))
	amount := big.NewInt(0).SetBytes(data[36:68])

	return receiver, amount
}

func (t *tron) parseTrc20ContractTransferFrom(data []byte) (string, string, *big.Int) {
	if len(data) != 100 {

		return "", "", nil
	}

	from := t.base58CheckEncode(append([]byte{0x41}, data[16:36]...))
	to := t.base58CheckEncode(append([]byte{0x41}, data[48:68]...))
	amount := big.NewInt(0).SetBytes(data[68:100])

	return from, to, amount
}

func (t *tron) gasFreePermitTransfer(data []byte) (string, string, *big.Int) {
	// https://tronscan.org/#/contract/TFFAMQLZybALaLb4uxHA9RBE7pxhUAjF3U/code?func=Tab-proxywrite-F3proxyNonePayable
	if len(data) != 420 {

		return "", "", nil
	}

	if !bytes.Equal(data[:4], []byte{0x6f, 0x21, 0xb8, 0x98}) {
		// not permitTransfer (6f21b898) function

		return "", "", nil
	}

	if !bytes.Equal(data[16:36], gasFreeUsdtTokenAddress) {
		// not gas free usdt token address

		return "", "", nil
	}

	user := t.base58CheckEncode(append([]byte{0x41}, data[48:68]...))
	receiver := t.base58CheckEncode(append([]byte{0x41}, data[80:100]...))
	amount := big.NewInt(0).SetBytes(data[100:132])

	return user, receiver, amount
}

func (t *tron) tradeConfirmHandle(ctx context.Context) {
	var orders = getConfirmingOrders([]string{model.OrderTradeTypeTronTrx, model.OrderTradeTypeUsdtTrc20, model.OrderTradeTypeUsdcTrc20})

	var wg sync.WaitGroup

	var handle = func(o model.TradeOrders) {
		conn, err := grpc.NewClient(conf.GetTronGrpcNode(), grpc.WithConnectParams(grpcParams), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Error("grpc.NewClient", err)

			return
		}

		defer conn.Close()

		var c = api.NewWalletClient(conn)

		idBytes, err := hex.DecodeString(o.TradeHash)
		if err != nil {
			log.Error("hex.DecodeString", err)

			return
		}

		if o.TradeType == model.OrderTradeTypeTronTrx {
			trans, err := c.GetTransactionById(ctx, &api.BytesMessage{Value: idBytes})
			if err != nil {
				log.Error("GetTransactionById", err)

				return
			}

			if trans.GetRet()[0].ContractRet == core.Transaction_Result_SUCCESS {
				markFinalConfirmed(o)
			}

			return
		}

		info, err := c.GetTransactionInfoById(ctx, &api.BytesMessage{Value: idBytes})
		if err != nil {
			log.Error("GetTransactionInfoById", err)

			return
		}

		if info.GetReceipt().GetResult() == core.Transaction_Result_SUCCESS {
			markFinalConfirmed(o)
		}
	}

	for _, order := range orders {
		wg.Add(1)
		go func() {
			defer wg.Done()

			handle(order)
		}()
	}

	wg.Wait()
}

func (t *tron) base58CheckEncode(input []byte) string {
	checksum := chainhash.DoubleHashB(input)
	checksum = checksum[:4]

	input = append(input, checksum...)

	return base58.Encode(input)
}

func (t *tron) rollBreak() bool {
	var count int64 = 0
	trade := []string{model.OrderTradeTypeTronTrx, model.OrderTradeTypeUsdtTrc20, model.OrderTradeTypeUsdcTrc20}
	model.DB.Model(&model.TradeOrders{}).Where("status = ? and trade_type in (?)", model.OrderStatusWaiting, trade).Count(&count)
	if count > 0 {

		return false
	}

	model.DB.Model(&model.WalletAddress{}).Where("other_notify = ? and trade_type in (?)", model.OtherNotifyEnable, trade).Count(&count)
	if count > 0 {

		return false
	}

	return true
}
