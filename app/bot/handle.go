package bot

import (
	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"strings"
)

func HandleMessage(msg *api.Message) {
	if msg.IsCommand() {
		botCommandHandle(msg)

		return
	}

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.Text == replayAddressText {

		addWalletAddress(msg)
	}

	if msg.Text != "" {
		if help.IsValidTronAddress(msg.Text) {
			go queryTronAddressInfo(msg, msg.Text)
		}

		if help.IsValidPolygonAddress(msg.Text) {
			go queryPolygonAddressInfo(msg, msg.Text)
		}
	}
}

func HandleCallback(query *api.CallbackQuery) {
	if query.From.ID != conf.BotAdminID() {

		return
	}

	var args []string
	var act = query.Data
	if strings.Contains(query.Data, "|") {
		args = strings.Split(query.Data, "|")
		act = args[0]
	}

	switch act {
	case cbWallet:
		go cbWalletAction(query, args[1])
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
	case cbOrderDetail:
		go cbOrderDetailAction(args[1])
	case cbMarkNotifySucc:
		go cbMarkNotifySuccAction(args[1])
	case dbOrderNotifyRetry:
		go dbOrderNotifyRetryAction(args[1])
	}
}

func addWalletAddress(msg *api.Message) {
	var address = strings.TrimSpace(msg.Text)
	// 简单检测地址是否合法
	if !help.IsValidTronAddress(address) && !help.IsValidPolygonAddress(address) {
		SendMsg(api.NewMessage(msg.Chat.ID, "钱包地址不合法"))

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
			SendMsg(api.NewMessage(msg.Chat.ID, "❌地址添加失败，地址重复！"))

			return
		}

		SendMsg(api.NewMessage(msg.Chat.ID, "❌地址添加失败，错误信息："+r.Error.Error()))

		return
	}

	SendMsg(api.NewMessage(msg.Chat.ID, "✅添加且成功启用"))
	cmdStartHandle()
}

func botCommandHandle(msg *api.Message) {
	if msg.Command() == cmdGetId {

		go cmdGetIdHandle(msg)
	}

	if msg.Chat.ID != conf.BotAdminID() {

		return
	}

	switch msg.Command() {
	case cmdStart:
		go cmdStartHandle()
	case cmdState:
		go cmdStateHandle()
	case cmdWallet:
		go cmdWalletHandle()
	case cmdOrder:
		go cmdOrderHandle()
	}
}

func queryTronAddressInfo(msg *api.Message, address string) {
	var info = getTronWalletInfo(address)
	var reply = api.NewMessage(msg.Chat.ID, "❌查询失败")
	if info != "" {
		reply.ReplyToMessageID = msg.MessageID
		reply.Text = info
		reply.ParseMode = api.ModeMarkdownV2
		reply.ReplyMarkup = api.InlineKeyboardMarkup{
			InlineKeyboard: [][]api.InlineKeyboardButton{
				{
					api.NewInlineKeyboardButtonURL("📝查看详细信息", "https://tronscan.org/#/address/"+address),
				},
			},
		}
	}

	_, _ = botApi.Send(reply)
}

func queryPolygonAddressInfo(msg *api.Message, address string) {
	var reply = api.NewMessage(msg.Chat.ID, getPolygonWalletInfo(address))

	reply.ReplyToMessageID = msg.MessageID
	reply.ParseMode = api.ModeMarkdownV2
	reply.ReplyMarkup = api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				api.NewInlineKeyboardButtonURL("📝查看详细信息", "https://polygonscan.com/address/"+address),
			},
		},
	}

	_, _ = botApi.Send(reply)
}
