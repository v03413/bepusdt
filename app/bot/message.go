package bot

import (
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"strings"
	"time"
)

func SendTradeSuccMsg(order model.TradeOrders) {
	// 获取代币类型
	tokenType, err := model.GetTokenType(order.TradeType)
	if err != nil {
		SendMessage(&bot.SendMessageParams{Text: "❌交易类型不支持：" + order.TradeType})
		return
	}

	tradeType := string(tokenType)

	var text = `
\#收款成功 \#订单交易 \#` + tradeType + `
\-\-\-
` + "```" + `
🚦商户订单：%v
💰请求金额：%v CNY(%v)
💲支付数额：%v ` + order.TradeType + `
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
		ChatID:    conf.BotNotifyTarget(),
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					models.InlineKeyboardButton{Text: "📝查看交易明细", URL: order.GetDetailUrl()},
				},
			},
		},
	})
}

func SendNotifyFailed(o model.TradeOrders, reason string) {
	// 获取代币类型
	tokenType, err := model.GetTokenType(o.TradeType)
	if err != nil {
		SendMessage(&bot.SendMessageParams{Text: "❌交易类型不支持：" + o.TradeType})
		return
	}

	tradeType := string(tokenType)

	var text = fmt.Sprintf(`
\#回调失败 \#订单交易 \#` + tradeType + `
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
		ChatID:    conf.BotNotifyTarget(),
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
👋 欢迎使用 BEpusdt，一款更好用的个人 USDT/USDC 收款网关，如果您看到此消息，说明机器人已经启动成功！

📌当前版本：` + app.Version + `
📝发送命令 /start 可以开始使用
🎉开源地址 https://github.com/v03413/bepusdt
---
`
}
