package task

import (
	"bytes"
	"context"
	"encoding/hex"
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
	"math/big"
	"strconv"
	"time"
)

// Tron区块确认偏移量
const tronBlockConfirmedOffset = 30

// usdt trc20 contract address 41a614f803b6fd780986a42c78ec9c7f77e6ded13c TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
var usdtTrc20ContractAddress = []byte{0x41, 0xa6, 0x14, 0xf8, 0x03, 0xb6, 0xfd, 0x78, 0x09, 0x86, 0xa4, 0x2c, 0x78, 0xec, 0x9c, 0x7f, 0x77, 0xe6, 0xde, 0xd1, 0x3c}
var tronLastBlockNum int64
var tronBlockScanQueue = chanx.NewUnboundedChan[int64](context.Background(), 30)
var params = grpc.ConnectParams{
	Backoff:           backoff.Config{BaseDelay: 1 * time.Second, MaxDelay: 30 * time.Second, Multiplier: 1.5},
	MinConnectTimeout: 1 * time.Minute,
}

type usdtTrc20TransferRaw struct {
	RecvAddress string
	Amount      int64
}

func init() {
	register(task{duration: time.Second * 3, callback: tronBlockRoll}) // 大概3秒产生一个区块
	register(task{duration: time.Second, callback: tronBlockDispatch})
}

func tronBlockDispatch(context.Context) {
	p, err := ants.NewPoolWithFunc(8, tronBlockParse)
	if err != nil {
		panic(err)

		return
	}

	defer p.Release()

	for n := range tronBlockScanQueue.Out {
		if err := p.Invoke(n); err != nil {
			tronBlockScanQueue.In <- n

			log.Warn("Tron Error invoking process block:", err)
		}
	}
}

func tronBlockRoll(context.Context) {
	var node = conf.GetTronGrpcNode()

	conn, err := grpc.NewClient(node, grpc.WithConnectParams(params), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {

		log.Error("grpc.NewClient", err)
	}

	defer conn.Close()

	var client = api.NewWalletClient(conn)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	block, err1 := client.GetNowBlock2(ctx, nil)
	defer cancel()

	if err1 != nil {
		log.Warn("GetNowBlock2 超时：", err1)

		return
	}

	var now = block.BlockHeader.RawData.Number
	if conf.GetTradeIsConfirmed() {

		now = now - tronBlockConfirmedOffset
	}

	// 首次启动
	if tronLastBlockNum == 0 {

		tronLastBlockNum = now - 1
	}

	// 区块高度没有变化
	if now <= tronLastBlockNum {

		return
	}

	// 待扫描区块入列
	for n := tronLastBlockNum + 1; n <= now; n++ {

		tronBlockScanQueue.In <- n
	}

	tronLastBlockNum = now
}

func tronBlockParse(n any) {
	var num = n.(int64)
	var node = conf.GetTronGrpcNode()
	var conn *grpc.ClientConn
	var err error
	if conn, err = grpc.NewClient(node, grpc.WithConnectParams(params), grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {

		log.Error("grpc.NewClient", err)
	}

	defer conn.Close()
	var client = api.NewWalletClient(conn)

	conf.SetBlockTotal(conf.Tron)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	block, err1 := client.GetBlockByNum2(ctx, &api.NumberMessage{Num: num})
	cancel()
	if err1 != nil {
		conf.SetBlockFail(conf.Tron)
		tronBlockScanQueue.In <- num
		log.Warn("GetBlockByNum Error", err1)

		return
	}

	var resources = make([]resource, 0)
	var transfers = make([]transfer, 0)
	var timestamp = time.UnixMilli(block.GetBlockHeader().GetRawData().GetTimestamp())
	for _, v := range block.GetTransactions() {
		if !v.Result.Result {

			continue
		}

		var itm = v.GetTransaction()
		var id = hex.EncodeToString(v.Txid)
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
					FromAddress:  base58CheckEncode(foo.OwnerAddress),
					RecvAddress:  base58CheckEncode(foo.ReceiverAddress),
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
					FromAddress:  base58CheckEncode(foo.OwnerAddress),
					RecvAddress:  base58CheckEncode(foo.ReceiverAddress),
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
					FromAddress: base58CheckEncode(foo.OwnerAddress),
					RecvAddress: base58CheckEncode(foo.ToAddress),
					Timestamp:   timestamp,
					TradeType:   model.OrderTradeTypeTronTrx,
					BlockNum:    cast.ToInt64(num),
				})

				continue
			}

			// 触发智能合约
			if contract.GetType() == core.Transaction_Contract_TriggerSmartContract {
				var foo = &core.TriggerSmartContract{}
				err := contract.GetParameter().UnmarshalTo(foo)
				if err != nil {

					continue
				}

				var transItem = transfer{Timestamp: timestamp, TxHash: id, FromAddress: base58CheckEncode(foo.OwnerAddress)}
				var reader = bytes.NewReader(foo.GetData())
				if !bytes.Equal(foo.GetContractAddress(), usdtTrc20ContractAddress) { // usdt trc20 contract

					continue
				}

				// 解析合约数据
				var trc20Contract = parseUsdtTrc20Contract(reader)
				if trc20Contract.Amount == 0 {

					continue
				}

				transItem.Network = conf.Tron
				transItem.TradeType = model.OrderTradeTypeUsdtTrc20
				transItem.Amount = decimal.NewFromBigInt(new(big.Int).SetInt64(trc20Contract.Amount), -6)
				transItem.RecvAddress = trc20Contract.RecvAddress
				transItem.BlockNum = cast.ToInt64(num)

				transfers = append(transfers, transItem)
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

func parseUsdtTrc20Contract(reader *bytes.Reader) usdtTrc20TransferRaw {
	var funcName = make([]byte, 4)
	_, err = reader.Read(funcName)
	if err != nil {
		// 读取funcName失败

		return usdtTrc20TransferRaw{}
	}
	if !bytes.Equal(funcName, []byte{0xa9, 0x05, 0x9c, 0xbb}) { // a9059cbb transfer(address,uint256)
		// funcName不匹配transfer

		return usdtTrc20TransferRaw{}
	}

	var addressBytes = make([]byte, 20)
	_, err = reader.ReadAt(addressBytes, 4+12)
	if err != nil {
		// 读取toAddress失败

		return usdtTrc20TransferRaw{}
	}

	var toAddress = base58CheckEncode(append([]byte{0x41}, addressBytes...))
	var value = make([]byte, 32)
	_, err = reader.ReadAt(value, 36)
	if err != nil {
		// 读取value失败

		return usdtTrc20TransferRaw{}
	}

	var amount, _ = strconv.ParseInt(hex.EncodeToString(value), 16, 64)

	return usdtTrc20TransferRaw{RecvAddress: toAddress, Amount: amount}
}

func base58CheckEncode(input []byte) string {
	checksum := chainhash.DoubleHashB(input)
	checksum = checksum[:4]

	input = append(input, checksum...)

	return base58.Encode(input)
}
