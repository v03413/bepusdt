package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/model"
)

const cbAddress = "address"
const cbAddressAdd = "address_add"
const cbAddressEnable = "address_enable"
const cbAddressDisable = "address_disable"
const cbAddressDelete = "address_del"
const cbAddressOtherNotify = "address_other_notify"

func cbAddressAddHandle(query *tgbotapi.CallbackQuery) {
	var msg = tgbotapi.NewMessage(query.Message.Chat.ID, replayAddressText)
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true, InputFieldPlaceholder: "输入钱包地址"}

	_, _ = botApi.Send(msg)
}

func cbAddressAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		var otherTextLabel = "✅已启用 非订单交易监控通知"
		if wa.OtherNotify != 1 {
			otherTextLabel = "❌已禁用 非订单交易监控通知"
		}

		EditAndSendMsg(query.Message.MessageID, wa.Address, tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonData("✅启用", cbAddressEnable+"|"+id),
					tgbotapi.NewInlineKeyboardButtonData("❌禁用", cbAddressDisable+"|"+id),
					tgbotapi.NewInlineKeyboardButtonData("⛔️删除", cbAddressDelete+"|"+id),
				},
				{
					tgbotapi.NewInlineKeyboardButtonData(otherTextLabel, cbAddressOtherNotify+"|"+id),
				},
			},
		})
	}
}

func cbAddressEnableAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusEnable)

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressDisableAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusDisable)

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressDeleteAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 删除钱包地址
		wa.Delete()

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressOtherNotifyAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		if wa.OtherNotify == 1 {
			wa.SetOtherNotify(model.OtherNotifyDisable)
		} else {
			wa.SetOtherNotify(model.OtherNotifyEnable)
		}

		DeleteMsg(query.Message.MessageID)

		cmdStartHandle()
	}
}
