package monitor

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"github.com/v03413/bepusdt/app/telegram"
	"strconv"
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

var transferQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 交易转账队列
var notOrderQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 非订单队列

func init() {
	RegisterSchedule(time.Second, orderTransferHandle)
	RegisterSchedule(time.Second, notOrderTransferHandle)
}

func orderTransferHandle(time.Duration) {
	for {
		select {
		case transfers := <-transferQueue.Out:
			var other = make([]transfer, 0)
			var orders = getAllWaitingOrders()
			for _, t := range transfers {
				// 计算交易金额
				var amount, quant = parseTransAmount(t.Amount)

				// 判断金额是否在允许范围内
				if !inPaymentAmountRange(amount) {

					continue
				}

				// 判断是否存在对应订单
				o, ok := orders[fmt.Sprintf("%s%v%s", t.RecvAddress, quant, t.TradeType)]
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

				go notify.Handle(o)             // 通知订单支付成功
				go telegram.SendTradeSuccMsg(o) // TG发送订单信息
			}

			if len(other) > 0 {
				notOrderQueue.In <- other
			}
		}
	}
}

func notOrderTransferHandle(time.Duration) {
	for {
		select {
		case transfers := <-notOrderQueue.Out:
			handleOtherNotify(transfers)
		}
	}
}

func handleOtherNotify(transfers []transfer) {
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

			var amount, quant = parseTransAmount(t.Amount)
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
				"#账户%s #非订单交易\n---\n```\n💲交易数额：%v \n💍交易类别："+strings.ToUpper(t.TradeType)+"\n⏱️交易时间：%v\n✅接收地址：%v\n🅾️发送地址：%v```\n",
				title,
				quant,
				t.Timestamp.Format(time.DateTime),
				help.MaskAddress(t.RecvAddress),
				help.MaskAddress(t.FromAddress),
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
						tgbotapi.NewInlineKeyboardButtonURL("📝查看交易明细", url),
					},
				},
			}

			var record = model.NotifyRecord{Txid: t.TxHash}
			model.DB.Create(&record)

			go telegram.SendMsg(msg)
		}
	}
}
