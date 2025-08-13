package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/spf13/cast"
	bot2 "github.com/v03413/bepusdt/app/bot"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/web/notify"
	"github.com/v03413/tronprotocol/core"
)

type transfer struct {
	Network     string
	TxHash      string
	Amount      decimal.Decimal
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
	register(task{callback: orderTransferHandle})
	register(task{callback: notOrderTransferHandle})
	register(task{callback: tronResourceHandle})
}

func markFinalConfirmed(o model.TradeOrders) {
	model.PushWebhookEvent(model.WebhookEventOrderPaid, o)

	o.SetSuccess()

	go notify.Handle(o)         // 通知订单支付成功
	go bot2.SendTradeSuccMsg(o) // TG发送订单信息
}

func orderTransferHandle(context.Context) {
	for transfers := range transferQueue.Out {
		var other = make([]transfer, 0)
		var orders = getAllWaitingOrders()
		for _, t := range transfers {
			// debug
			//if t.TradeType == model.OrderTradeTypeUsdcBep20 {
			//	fmt.Println(t.TradeType, t.TxHash, t.FromAddress, "=>", t.RecvAddress, t.Amount.String())
			//}

			// 判断金额是否在允许范围内
			if !inAmountRange(t.Amount) {

				continue
			}

			// 判断是否存在对应订单
			o, ok := orders[fmt.Sprintf("%s%v%s", t.RecvAddress, t.Amount.String(), t.TradeType)]
			if !ok {
				other = append(other, t)

				continue
			}

			// 有效期检测
			if !o.CreatedAt.Before(t.Timestamp) || !o.ExpiredAt.After(t.Timestamp) {

				continue
			}

			// 进入确认状态
			o.MarkConfirming(t.BlockNum, t.FromAddress, t.TxHash, t.Timestamp)
		}

		if len(other) > 0 {
			notOrderQueue.In <- other
		}
	}
}

func notOrderTransferHandle(context.Context) {
	for transfers := range notOrderQueue.Out {
		var was []model.WalletAddress

		model.DB.Where("other_notify = ?", model.OtherNotifyEnable).Find(&was)

		for _, wa := range was {
			for _, t := range transfers {
				if t.RecvAddress != wa.Address && t.FromAddress != wa.Address {

					continue
				}

				if !inAmountRange(t.Amount) {

					continue
				}

				if !model.IsNeedNotifyByTxid(t.TxHash) {

					continue
				}

				var title = "收入"
				if t.RecvAddress != wa.Address {
					title = "支出"
				}

				var text = fmt.Sprintf(
					"\\#账户%s \\#非订单交易\n\\-\\-\\-\n```\n💲交易数额：%v \n💍交易类别："+strings.ToUpper(t.TradeType)+"\n⏱️交易时间：%v\n✅接收地址：%v\n🅾️发送地址：%v```\n",
					title,
					t.Amount.String(),
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
								models.InlineKeyboardButton{Text: "📝查看交易明细", URL: model.GetDetailUrl(t.TradeType, t.TxHash)},
							},
						},
					},
				})
			}
		}
	}
}

func tronResourceHandle(context.Context) {
	for resources := range resourceQueue.Out {
		var was []model.WalletAddress
		var types = []string{model.OrderTradeTypeTronTrx, model.OrderTradeTypeUsdtTrc20}

		model.DB.Where("status = ? and other_notify = ? and trade_type in (?)", model.StatusEnable, model.OtherNotifyEnable, types).Find(&was)

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

func getAllWaitingOrders() map[string]model.TradeOrders {
	var tradeOrders = model.GetOrderByStatus(model.OrderStatusWaiting)
	var data = make(map[string]model.TradeOrders) // 当前所有正在等待支付的订单 Lock Key
	for _, order := range tradeOrders {
		if time.Now().Unix() >= order.ExpiredAt.Unix() { // 订单过期
			order.SetExpired()
			notify.Bepusdt(order)
			model.PushWebhookEvent(model.WebhookEventOrderTimeout, order)

			continue
		}

		if order.TradeType == model.OrderTradeTypeUsdtPolygon {

			order.Address = strings.ToLower(order.Address)
		}

		data[order.Address+order.Amount+order.TradeType] = order
	}

	return data
}

func getConfirmingOrders(tradeType []string) []model.TradeOrders {
	var orders = make([]model.TradeOrders, 0)
	var data = make([]model.TradeOrders, 0)
	var db = model.DB.Where("status = ?", model.OrderStatusConfirming)
	if len(tradeType) > 0 {
		db = db.Where("trade_type in (?)", tradeType)
	}

	db.Find(&orders)

	for _, order := range orders {
		if time.Now().Unix() >= order.ExpiredAt.Unix() {
			order.SetFailed()
			notify.Bepusdt(order)
			model.PushWebhookEvent(model.WebhookEventOrderFailed, order)

			continue
		}

		data = append(data, order)
	}

	return data
}
