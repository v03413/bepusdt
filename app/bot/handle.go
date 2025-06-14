package bot

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/go-cache"
	"strings"
)

func defaultHandle(ctx context.Context, bot *bot.Bot, u *models.Update) {
	if u.Message.ReplyToMessage != nil && u.Message.ReplyToMessage.Text == replayAddressText {
		addWalletAddress(u)

		return
	}

	// 私聊消息
	if u.Message != nil && u.Message.Chat.Type == models.ChatTypePrivate {
		var text = u.Message.Text
		if help.IsValidTronAddress(text) {
			go queryTronAddressInfo(u.Message)
		}
	}
}

func addWalletAddress(u *models.Update) {
	var address = strings.TrimSpace(u.Message.Text)
	if !help.IsValidTronAddress(address) && !help.IsValidEvmAddress(address) {
		SendMessage(&bot.SendMessageParams{Text: "钱包地址不合法"})

		return
	}

	var tradeType, _ = cache.Get(fmt.Sprintf("%s_%d_trade_type", cbAddressAdd, u.Message.Chat.ID))

	var wa = model.WalletAddress{TradeType: tradeType.(string), Address: address, Status: model.StatusEnable, OtherNotify: model.OtherNotifyEnable}
	var r = model.DB.Create(&wa)
	if r.Error != nil {
		SendMessage(&bot.SendMessageParams{Text: "❌地址添加失败，" + r.Error.Error()})

		return
	}

	SendMessage(&bot.SendMessageParams{Text: "✅添加且成功启用"})

	// 推送最新状态
	cmdStartHandle(context.Background(), api, u)
}

func queryTronAddressInfo(m *models.Message) {
	var address = strings.TrimSpace(m.Text)
	var params = bot.SendMessageParams{
		ChatID:    m.Chat.ID,
		Text:      getTronWalletInfo(address),
		ParseMode: models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: m.ID,
			ChatID:    m.Chat.ID,
		},
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					models.InlineKeyboardButton{Text: "📝查看详细信息", URL: "https://tronscan.org/#/address/" + address},
				},
			},
		},
	}

	SendMessage(&params)
}
