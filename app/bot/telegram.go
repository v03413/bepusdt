package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
)

var botApi *tgbotapi.BotAPI
var err error

func Init() {
	botApi, err = tgbotapi.NewBotAPI(conf.BotToken())
	if err != nil {
		panic("TG Bot NewBotAPI Error:" + err.Error())

		return
	}

	// 注册命令
	_, err = botApi.Request(tgbotapi.NewSetMyCommands([]tgbotapi.BotCommand{
		{Command: "/" + cmdGetId, Description: "获取ID"},
		{Command: "/" + cmdStart, Description: "开始使用"},
		{Command: "/" + cmdState, Description: "收款状态"},
		{Command: "/" + cmdWallet, Description: "钱包信息"},
		{Command: "/" + cmdOrder, Description: "最近订单"},
	}...))
	if err != nil {
		panic("TG Bot Request Error:" + err.Error())

		return
	}

	fmt.Println("Bot UserName: ", botApi.Self.UserName)
}

func GetBotApi() *tgbotapi.BotAPI {

	return botApi
}

func SendMsg(msg tgbotapi.MessageConfig) {
	if msg.ChatID == 0 {

		msg.ChatID = conf.BotAdminID()
	}

	_, err = botApi.Send(msg)
	if err != nil {

		log.Warn("Bot SendMsg Error:", err.Error())
	}
}

func DeleteMsg(msgId int) {
	_, err = botApi.Send(tgbotapi.NewDeleteMessage(conf.BotAdminID(), msgId))
	if err != nil {

		log.Warn("Bot DeleteMsg Error:", err.Error())
	}
}

func EditAndSendMsg(msgId int, text string, replyMarkup tgbotapi.InlineKeyboardMarkup) {
	_, err = botApi.Send(tgbotapi.NewEditMessageTextAndMarkup(conf.BotAdminID(), msgId, text, replyMarkup))
	if err != nil {

		log.Warn("Bot EditAndSendMsg Error:", err.Error())
	}
}
