package bot

import (
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"strconv"
	"strings"
	"time"
)

func SendTradeSuccMsg(order model.TradeOrders) {
	var chatId, err = strconv.ParseInt(conf.BotNotifyTarget(), 10, 64)
	if err != nil {

		return
	}

	var url = fmt.Sprintf("https://tronscan.org/#/transaction/%s", order.TradeHash)

	var tradeType = "USDT"
	var tradeUnit = `USDT.TRC20`
	if order.TradeType == model.OrderTradeTypeTronTrx {
		tradeType = "TRX"
		tradeUnit = "TRX"
	}
	if order.TradeType == model.OrderTradeTypeUsdtPolygon {
		tradeType = "USDT"
		tradeUnit = "USDT.Polygon"
		url = fmt.Sprintf("https://polygonscan.com/tx/%s", order.TradeHash)
	}

	var text = `
\#收款成功 \#订单交易 \#` + tradeType + `
\-\-\-
` + "```" + `
🚦商户订单：%v
💰请求金额：%v CNY(%v)
💲支付数额：%v ` + tradeUnit + `
💎交易哈希：%s
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
		help.MaskHash(order.TradeHash),
		help.MaskAddress(order.Address),
		order.CreatedAt.Format(time.DateTime),
		order.UpdatedAt.Format(time.DateTime),
	)

	SendMessage(&bot.SendMessageParams{
		Text:      text,
		ChatID:    chatId,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					models.InlineKeyboardButton{Text: "📝查看交易明细", URL: url},
				},
			},
		},
	})
}

func SendNotifyFailed(o model.TradeOrders, reason string) {
	var chatId = cast.ToInt64(conf.BotNotifyTarget())
	if err != nil {

		return
	}

	var tradeType = "USDT"
	if o.TradeType == model.OrderTradeTypeTronTrx {
		tradeType = "TRX"
	}

	var text = fmt.Sprintf(`
\#回调失败 \#订单交易 \#`+tradeType+`
\-\-\-
`+"```"+`
🚦商户订单：%v
💲支付数额：%v
💰请求金额：%v CNY(%v)
💍交易类别：%s
⚖️️确认时间：%s
⏰下次回调：%s
🗒️失败原因：%s
`+"```"+`
`,
		help.Ec(o.OrderId),
		o.Amount,
		o.Money, o.TradeRate,
		strings.ToUpper(o.TradeType),
		o.ConfirmedAt.Format(time.DateTime),
		help.CalcNextNotifyTime(o.ConfirmedAt, o.NotifyNum+1).Format(time.DateTime),
		reason,
	)

	SendMessage(&bot.SendMessageParams{
		Text:      text,
		ChatID:    chatId,
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					models.InlineKeyboardButton{Text: "📝查看收款详情", CallbackData: fmt.Sprintf("%s|%v", cbOrderDetail, o.TradeId)},
					models.InlineKeyboardButton{Text: "✅标记回调成功", CallbackData: fmt.Sprintf("%s|%v", cbMarkNotifySucc, o.TradeId)},
				},
			},
		},
	})
}

func Welcome() string {
	return `
👋 欢迎使用 Bepusdt，一款更好用的个人USDT收款网关，如果您看到此消息，说明机器人已经启动成功！

📌当前版本：` + app.Version + `
📝发送命令 /start 可以开始使用
🎉开源地址 https://github.com/v03413/bepusdt
---
`
}
