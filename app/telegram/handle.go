package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"strings"
)

func HandleMessage(msg *tgbotapi.Message) {
	if msg.IsCommand() {
		botCommandHandle(msg)

		return
	}

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.Text == replayAddressText {
		addWalletAddress(msg)
	}
}

func HandleCallback(query *tgbotapi.CallbackQuery) {
	if fmt.Sprintf("%v", query.From.ID) != config.GetTGBotAdminId() {

		return
	}

	var args []string
	var act = query.Data
	if strings.Contains(query.Data, "|") {
		args = strings.Split(query.Data, "|")
		act = args[0]
	}

	switch act {
	case cbAddressAdd:
		go cbAddressAddHandle(query)
	case cbAddress:
		go cbAddressAction(query, args[1])
	case cbAddressEnable:
		go cbAddressEnableAction(query, args[1])
	case cbAddressDisable:
		go cbAddressDisableAction(query, args[1])
	case cbAddressDelete:
		go cbAddressDeleteAction(query, args[1])
	case cbAddressOtherNotify:
		go cbAddressOtherNotifyAction(query, args[1])
	}

}

func addWalletAddress(msg *tgbotapi.Message) {
	var address = strings.TrimSpace(msg.Text)
	// 简单检测地址是否合法
	if !help.IsValidTRONWalletAddress(address) {
		SendMsg(tgbotapi.NewMessage(msg.Chat.ID, "钱包地址不合法"))

		return
	}

	var wa = model.WalletAddress{Address: address, Status: model.StatusEnable}
	var r = model.DB.Create(&wa)
	if r.Error != nil {
		if r.Error.Error() == "UNIQUE constraint failed: wallet_address.address" {
			SendMsg(tgbotapi.NewMessage(msg.Chat.ID, "❌地址添加失败，地址重复！"))

			return
		}

		SendMsg(tgbotapi.NewMessage(msg.Chat.ID, "❌地址添加失败，错误信息："+r.Error.Error()))

		return
	}

	SendMsg(tgbotapi.NewMessage(msg.Chat.ID, "✅添加且成功启用"))
	cmdStartHandle()
}

func botCommandHandle(_msg *tgbotapi.Message) {
	if _msg.Command() == cmdGetId {

		go cmdGetIdHandle(_msg)
	}

	if fmt.Sprintf("%v", _msg.Chat.ID) != config.GetTGBotAdminId() {

		return
	}

	switch _msg.Command() {
	case cmdStart:
		go cmdStartHandle()
	case cmdUsdt:
		go cmdUsdtHandle()
	}
}
