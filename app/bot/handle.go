package bot

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
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

		if help.IsValidPolygonAddress(text) {
			go queryPolygonAddressInfo(u.Message)
		}
	}
}

func addWalletAddress(u *models.Update) {
	var address = strings.TrimSpace(u.Message.Text)
	if !help.IsValidTronAddress(address) && !help.IsValidPolygonAddress(address) {
		SendMessage(&bot.SendMessageParams{Text: "钱包地址不合法"})

		return
	}

	var chain = model.WaChainTron
	if help.IsValidPolygonAddress(address) {

		chain = model.WaChainPolygon
	}

	var wa = model.WalletAddress{Chain: chain, Address: address, Status: model.StatusEnable, OtherNotify: model.OtherNotifyEnable}
	var r = model.DB.Create(&wa)
	if r.Error != nil {
		if r.Error.Error() == "UNIQUE constraint failed: wallet_address.address" {
			SendMessage(&bot.SendMessageParams{Text: "❌地址添加失败，地址重复！"})

			return
		}

		SendMessage(&bot.SendMessageParams{Text: "❌地址添加失败，错误信息：" + r.Error.Error()})

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

func queryPolygonAddressInfo(m *models.Message) {
	var address = strings.TrimSpace(m.Text)
	var params = bot.SendMessageParams{
		ChatID:          m.Chat.ID,
		Text:            getPolygonWalletInfo(address),
		ParseMode:       models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{MessageID: m.ID, ChatID: m.Chat.ID},
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					models.InlineKeyboardButton{Text: "📝查看详细信息", URL: "https://polygonscan.com/address/" + address},
				},
			},
		},
	}

	SendMessage(&params)
}
