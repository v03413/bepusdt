package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/model"
	"strconv"
	"strings"
	"time"
)

func SendTradeSuccMsg(order model.TradeOrders) {
	var adminChatId, err = strconv.ParseInt(config.GetTGBotAdminId(), 10, 64)
	if err != nil {

		return
	}
	var text = `
✅有新的交易支付成功
---
📝商户订单：｜%v｜
💰请求金额：｜%v｜ CNY(%v)
💲支付数额：%v USDT.TRC20
🪧收款地址：｜%s｜
⏱️创建时间：%s
️🎯️支付时间：%s
`
	text = fmt.Sprintf(strings.ReplaceAll(text, "｜", "`"), order.OrderId, order.Money, order.UsdtRate, order.Amount, order.Address,
		order.CreatedAt.Format(time.DateTime), order.UpdatedAt.Format(time.DateTime))
	var msg = tgbotapi.NewMessage(adminChatId, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, _ = botApi.Send(msg)
}
