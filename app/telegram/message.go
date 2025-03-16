package telegram

import (
	"fmt"
	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"strconv"
	"time"
)

func SendTradeSuccMsg(order model.TradeOrders) {
	var chatId, err = strconv.ParseInt(config.GetTgBotNotifyTarget(), 10, 64)
	if err != nil {

		return
	}

	var tradeType = "USDT"
	var tradeUnit = `USDT.TRC20`
	if order.TradeType == model.OrderTradeTypeTronTrx {
		tradeType = "TRX"
		tradeUnit = "TRX"
	}

	var text = `
#收款成功 #订单交易 #` + tradeType + `
---
` + "```" + `
🚦商户订单：%v
💰请求金额：%v CNY(%v)
💲支付数额：%v ` + tradeUnit + `
✅收款地址：%s
⏱️创建时间：%s
️🎯️支付时间：%s
` + "```" + `
`
	text = fmt.Sprintf(text,
		order.OrderId,
		order.Money,
		order.TradeRate,
		order.Amount,
		help.MaskAddress(order.Address),
		order.CreatedAt.Format(time.DateTime),
		order.UpdatedAt.Format(time.DateTime),
	)
	var msg = api.NewMessage(chatId, text)
	msg.ParseMode = api.ModeMarkdown
	msg.ReplyMarkup = api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				api.NewInlineKeyboardButtonURL("📝查看交易明细", "https://tronscan.org/#/transaction/"+order.TradeHash),
			},
		},
	}

	_, _ = botApi.Send(msg)
}

func SendNotifyFailed(o model.TradeOrders, reason string) {
	var chatId = cast.ToInt64(config.GetTgBotNotifyTarget())
	if err != nil {

		return
	}

	var tradeType = "USDT"
	var tradeUnit = `USDT.TRC20`
	if o.TradeType == model.OrderTradeTypeTronTrx {
		tradeType = "TRX"
		tradeUnit = "TRX"
	}

	var text = fmt.Sprintf(`
#回调失败 #订单交易 #`+tradeType+`
---
`+"```"+`
🚦商户订单：%v
💰请求金额：%v CNY(%v)
💲支付数额：%v %s
⚖️️确认时间：%s
⏰下次回调：%s
🗒️失败原因：%s
`+"```"+`
`,
		help.Ec(o.OrderId),
		o.Money, o.TradeRate,
		o.Amount, tradeUnit,
		o.ConfirmedAt.Format(time.DateTime),
		help.CalcNextNotifyTime(o.ConfirmedAt, o.NotifyNum+1).Format(time.DateTime),
		reason,
	)

	var msg = api.NewMessage(chatId, text)
	msg.ParseMode = api.ModeMarkdown
	msg.ReplyMarkup = api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				api.NewInlineKeyboardButtonData("📝查看收款详情", fmt.Sprintf("%s|%v", cbOrderDetail, o.TradeId)),
			},
			{
				api.NewInlineKeyboardButtonData("✅标记回调成功", fmt.Sprintf("%s|%v", cbMarkNotifySucc, o.TradeId)),
			},
		},
	}

	_, _ = botApi.Send(msg)
}

func SendOtherNotify(text string) {
	var chatId, err = strconv.ParseInt(config.GetTgBotNotifyTarget(), 10, 64)
	if err != nil {

		return
	}

	var msg = api.NewMessage(chatId, text)
	msg.ParseMode = api.ModeMarkdown

	_, _ = botApi.Send(msg)
}

func SendWelcome(version string) {
	var text = `
👋 欢迎使用 Bepusdt，一款更好用的个人USDT收款网关，如果您看到此消息，说明机器人已经启动成功！

📌当前版本：` + version + `
📝发送命令 /start 可以开始使用
🎉开源地址 https://github.com/v03413/bepusdt
---
`

	SendMsg(api.NewMessage(0, text))
}
