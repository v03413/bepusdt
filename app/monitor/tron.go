package monitor

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/telegram"
	"github.com/v03413/tronprotocol/api"
	"github.com/v03413/tronprotocol/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// 暂且认为交易所在区块高度和当前区块高度差值超过20，说明此交易已经被网络确认
const blockHeightNumConfirmedSub = 20

// usdt trc20 contract address 41a614f803b6fd780986a42c78ec9c7f77e6ded13c TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
var usdtTrc20ContractAddress = []byte{0x41, 0xa6, 0x14, 0xf8, 0x03, 0xb6, 0xfd, 0x78, 0x09, 0x86, 0xa4, 0x2c, 0x78, 0xec, 0x9c, 0x7f, 0x77, 0xe6, 0xde, 0xd1, 0x3c}

var currentBlockHeight int64

type resource struct {
	ID           string
	Type         core.Transaction_Contract_ContractType
	Balance      int64
	FromAddress  string
	RecvAddress  string
	Timestamp    time.Time
	ResourceCode core.ResourceCode
}

type usdtTrc20TransferRaw struct {
	RecvAddress string
	Amount      float64
}

func init() {
	RegisterSchedule(time.Second*3, tronBlockScan)
}

// tronBlockScan 区块扫描
func tronBlockScan(duration time.Duration) {
	var node = config.GetTronGrpcNode()
	log.Info("区块扫描启动：", node)

	reParams := grpc.ConnectParams{
		Backoff:           backoff.Config{BaseDelay: 1 * time.Second, MaxDelay: 30 * time.Second, Multiplier: 1.5},
		MinConnectTimeout: 1 * time.Minute,
	}

	conn, err := grpc.NewClient(node, grpc.WithConnectParams(reParams), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {

		log.Error("grpc.NewClient", err)
	}

	defer conn.Close()

	var client = api.NewWalletClient(conn)

	for range time.Tick(duration) { // 3秒产生一个区块
		atomic.AddUint64(&config.BlockScanTotal, 1)

		var ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		nowBlock, err1 := client.GetNowBlock2(ctx, nil)
		cancel()
		if err1 != nil {
			log.Warn("GetNowBlock 超时：", err1)

			continue
		}

		atomic.AddUint64(&config.BlockScanSucc, 1)

		var nowBlockHeight = nowBlock.BlockHeader.RawData.Number
		if config.GetTradeConfirmed() {
			nowBlockHeight = nowBlockHeight - blockHeightNumConfirmedSub
		}

		if currentBlockHeight == 0 { // 初始化当前区块高度

			currentBlockHeight = nowBlockHeight - 1
		}

		// 连续区块
		var sub = nowBlockHeight - currentBlockHeight
		if sub == 1 {
			parseBlockTrans(nowBlock, nowBlockHeight)

			continue
		}

		// 如果当前区块高度和上次扫描的区块高度差值超过1，说明存在区块丢失
		var endBlockHeight = nowBlockHeight
		var startBlockHeight = currentBlockHeight + 1

		// 扫描丢失的区块
		var ctx2, cancel2 = context.WithTimeout(context.Background(), time.Second*3)
		blocks, err2 := client.GetBlockByLimitNext2(ctx2, &api.BlockLimit{StartNum: startBlockHeight, EndNum: endBlockHeight})
		cancel2()
		if err2 != nil {
			log.Warn("GetBlockByLimitNext2 超时：", err2)

			continue
		}

		// 扫描丢失区块
		for _, block := range blocks.GetBlock() {
			parseBlockTrans(block, block.BlockHeader.RawData.Number)
		}
	}
}

// parseBlockTrans 解析区块交易
func parseBlockTrans(block *api.BlockExtention, nowHeight int64) {
	currentBlockHeight = nowHeight

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
					TxHash:      id,
					Amount:      float64(foo.Amount),
					FromAddress: base58CheckEncode(foo.OwnerAddress),
					RecvAddress: base58CheckEncode(foo.ToAddress),
					Timestamp:   timestamp,
					TradeType:   model.OrderTradeTypeTronTrx,
					BlockNum:    nowHeight,
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

				transItem.TradeType = model.OrderTradeTypeUsdtTrc20
				transItem.Amount = trc20Contract.Amount
				transItem.RecvAddress = trc20Contract.RecvAddress
				transItem.BlockNum = nowHeight

				transfers = append(transfers, transItem)
			}
		}
	}

	if len(transfers) > 0 {
		transferQueue.In <- transfers
	}

	if len(resources) > 0 {
		handleResourceNotify(resources)
	}

	log.Info("区块扫描完成", nowHeight, "TRON")
}

// parseUsdtTrc20Contract 解析usdt trc20合约
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

	return usdtTrc20TransferRaw{RecvAddress: toAddress, Amount: float64(amount)}
}

// handleResourceNotify 处理资源通知
func handleResourceNotify(items []resource) {
	var ads []model.WalletAddress
	var tx = model.DB.Where("status = ? and other_notify = ?", model.StatusEnable, model.OtherNotifyEnable).Find(&ads)
	if tx.RowsAffected <= 0 {

		return
	}

	for _, wa := range ads {
		for _, trans := range items {
			if trans.RecvAddress != wa.Address && trans.FromAddress != wa.Address {

				continue
			}

			if trans.ResourceCode != core.ResourceCode_ENERGY {

				continue
			}

			var detailUrl = "https://tronscan.org/#/transaction/" + trans.ID
			if !model.IsNeedNotifyByTxid(trans.ID) {
				// 不需要额外通知

				continue
			}

			var title = "代理"
			if trans.Type == core.Transaction_Contract_UnDelegateResourceContract {
				title = "回收"
			}

			var text = fmt.Sprintf(
				"#资源动态 #能量"+title+"\n---\n```\n🔋质押数量："+cast.ToString(trans.Balance/1000000)+"\n⏱️交易时间：%v\n✅操作地址：%v\n🅾️资源来源：%v```\n",
				trans.Timestamp.Format(time.DateTime),
				help.MaskAddress(trans.RecvAddress),
				help.MaskAddress(trans.FromAddress),
			)

			var chatId, err = strconv.ParseInt(config.GetTgBotNotifyTarget(), 10, 64)
			if err != nil {

				continue
			}

			var msg = tgbotapi.NewMessage(chatId, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						tgbotapi.NewInlineKeyboardButtonURL("📝查看交易明细", detailUrl),
					},
				},
			}

			var _record = model.NotifyRecord{Txid: trans.ID}
			model.DB.Create(&_record)

			go telegram.SendMsg(msg)
		}
	}
}

func base58CheckEncode(input []byte) string {
	checksum := chainhash.DoubleHashB(input)
	checksum = checksum[:4]

	input = append(input, checksum...)

	return base58.Encode(input)
}

// 列出所有待支付交易订单
func getAllWaitingOrders() map[string]model.TradeOrders {
	var tradeOrders = model.GetTradeOrderByStatus(model.OrderStatusWaiting)
	var data = make(map[string]model.TradeOrders) // 当前所有正在等待支付的订单 Lock Key
	for _, order := range tradeOrders {
		if time.Now().Unix() >= order.ExpiredAt.Unix() { // 订单过期
			order.OrderSetExpired()

			continue
		}

		if order.TradeType == model.OrderTradeTypeUsdtPolygon {

			order.Address = strings.ToLower(order.Address)
		}

		data[order.Address+order.Amount+order.TradeType] = order
	}

	return data
}

// 解析交易金额
func parseTransAmount(amount float64) (decimal.Decimal, string) {
	var result = decimal.NewFromFloat(amount).Div(decimal.NewFromFloat(1000000))

	return result, result.String()
}
