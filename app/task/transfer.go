package task

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/smallnest/chanx"
	"github.com/spf13/cast"
	bot2 "github.com/v03413/bepusdt/app/bot"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"github.com/v03413/tronprotocol/core"
	"strings"
	"time"
)

type transfer struct {
	TxHash      string
	Amount      float64
	FromAddress string
	RecvAddress string
	Timestamp   time.Time
	TradeType   string
	BlockNum    int64
}

type resource struct {
	ID           string
	Type         core.Transaction_Contract_ContractType
	Balance      int64
	FromAddress  string
	RecvAddress  string
	Timestamp    time.Time
	ResourceCode core.ResourceCode
}

var resourceQueue = chanx.NewUnboundedChan[[]resource](context.Background(), 30) // 资源队列
var notOrderQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 非订单队列
var transferQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 交易转账队列

func init() {
	RegisterSchedule(time.Second, orderTransferHandle)
	RegisterSchedule(time.Second, notOrderTransferHandle)
	RegisterSchedule(time.Second, tronResourceHandle)
}

func orderTransferHandle(time.Duration) {
	for transfers := range transferQueue.Out {
		var other = make([]transfer, 0)
		var orders = getAllWaitingOrders()
		for _, t := range transfers {
			// 计算交易金额
			var amount = parseTransAmount(t.Amount)

			// 判断金额是否在允许范围内
			if !inPaymentAmountRange(amount) {

				continue
			}

			// 判断是否存在对应订单
			o, ok := orders[fmt.Sprintf("%s%v%s", t.RecvAddress, amount.String(), t.TradeType)]
			if !ok {
				other = append(other, t)

				continue
			}

			// 有效期检测
			if !o.CreatedAt.Before(t.Timestamp) || !o.ExpiredAt.After(t.Timestamp) {

				continue
			}

			// 标记成功
			o.MarkSuccess(t.BlockNum, t.FromAddress, t.TxHash, t.Timestamp)

			go notify.Handle(o)         // 通知订单支付成功
			go bot2.SendTradeSuccMsg(o) // TG发送订单信息
		}

		if len(other) > 0 {
			notOrderQueue.In <- other
		}
	}
}

func notOrderTransferHandle(time.Duration) {
	for transfers := range notOrderQueue.Out {
		var was []model.WalletAddress

		model.DB.Where("status = ? and other_notify = ?", model.StatusEnable, model.OtherNotifyEnable).Find(&was)

		for _, wa := range was {
			if wa.Chain == model.WaChainPolygon {
				wa.Address = strings.ToLower(wa.Address)
			}

			for _, t := range transfers {
				if t.RecvAddress != wa.Address && t.FromAddress != wa.Address {

					continue
				}

				var amount = parseTransAmount(t.Amount)
				if !inPaymentAmountRange(amount) {

					continue
				}

				if !model.IsNeedNotifyByTxid(t.TxHash) {

					continue
				}

				var url = "https://tronscan.org/#/transaction/" + t.TxHash
				if t.TradeType == model.OrderTradeTypeUsdtPolygon {
					url = "https://polygonscan.com/tx/" + t.TxHash
				}

				var title = "收入"
				if t.RecvAddress != wa.Address {
					title = "支出"
				}

				var text = fmt.Sprintf(
					"\\#账户%s \\#非订单交易\n\\-\\-\\-\n```\n💲交易数额：%v \n💍交易类别："+strings.ToUpper(t.TradeType)+"\n⏱️交易时间：%v\n✅接收地址：%v\n🅾️发送地址：%v```\n",
					title,
					amount.String(),
					t.Timestamp.Format(time.DateTime),
					help.MaskAddress(t.RecvAddress),
					help.MaskAddress(t.FromAddress),
				)

				var record = model.NotifyRecord{Txid: t.TxHash}
				model.DB.Create(&record)

				go bot2.SendMessage(&bot.SendMessageParams{
					ChatID:    conf.BotNotifyTarget(),
					Text:      text,
					ParseMode: models.ParseModeMarkdown,
					ReplyMarkup: models.InlineKeyboardMarkup{
						InlineKeyboard: [][]models.InlineKeyboardButton{
							{
								models.InlineKeyboardButton{Text: "📝查看交易明细", URL: url},
							},
						},
					},
				})
			}
		}
	}
}

func tronResourceHandle(time.Duration) {
	for resources := range resourceQueue.Out {
		var was []model.WalletAddress

		model.DB.Where("status = ? and other_notify = ? and chain = ?", model.StatusEnable, model.OtherNotifyEnable, model.WaChainTron).Find(&was)

		for _, wa := range was {
			for _, t := range resources {
				if t.RecvAddress != wa.Address && t.FromAddress != wa.Address {

					continue
				}

				if t.ResourceCode != core.ResourceCode_ENERGY {

					continue
				}

				var url = "https://tronscan.org/#/transaction/" + t.ID
				if !model.IsNeedNotifyByTxid(t.ID) {

					continue
				}

				var title = "代理"
				if t.Type == core.Transaction_Contract_UnDelegateResourceContract {
					title = "回收"
				}

				var text = fmt.Sprintf(
					"\\#资源动态 \\#能量"+title+"\n\\-\\-\\-\n```\n🔋质押数量："+cast.ToString(t.Balance/1000000)+"\n⏱️交易时间：%v\n✅操作地址：%v\n🅾️资源来源：%v```\n",
					t.Timestamp.Format(time.DateTime),
					help.MaskAddress(t.RecvAddress),
					help.MaskAddress(t.FromAddress),
				)

				var record = model.NotifyRecord{Txid: t.ID}
				model.DB.Create(&record)

				go bot2.SendMessage(&bot.SendMessageParams{
					ChatID:    conf.BotNotifyTarget(),
					Text:      text,
					ParseMode: models.ParseModeMarkdown,
					ReplyMarkup: models.InlineKeyboardMarkup{
						InlineKeyboard: [][]models.InlineKeyboardButton{
							{
								models.InlineKeyboardButton{Text: "📝查看交易明细", URL: url},
							},
						},
					},
				})
			}
		}
	}
}
