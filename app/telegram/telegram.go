package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/config"
	"strconv"
)

var botApi *tgbotapi.BotAPI
var err error

func init() {
	var token = config.GetTGBotToken()
	if token == "" {

		return
	}

	botApi, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		panic("TG Bot NewBotAPI Error:" + err.Error())

		return
	}

	// 注册命令
	_, err = botApi.Request(tgbotapi.NewSetMyCommands([]tgbotapi.BotCommand{
		{Command: "/" + cmdGetId, Description: "获取ID"},
		{Command: "/" + cmdStart, Description: "开始使用"},
		{Command: "/" + cmdUsdt, Description: "实时汇率"},
		{Command: "/" + cmdWallet, Description: "钱包信息"},
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
	if msg.ChatID != 0 {
		_, _ = botApi.Send(msg)

		return
	}

	var chatId, err = strconv.ParseInt(config.GetTGBotAdminId(), 10, 64)
	if err == nil {
		msg.ChatID = chatId
		_, _ = botApi.Send(msg)
	}
}

func DeleteMsg(msgId int) {
	var chatId, err = strconv.ParseInt(config.GetTGBotAdminId(), 10, 64)
	if err == nil {
		_, _ = botApi.Send(tgbotapi.NewDeleteMessage(chatId, msgId))
	}
}

func EditAndSendMsg(msgId int, text string, replyMarkup tgbotapi.InlineKeyboardMarkup) {
	var chatId, err = strconv.ParseInt(config.GetTGBotAdminId(), 10, 64)
	if err == nil {
		_, _ = botApi.Send(tgbotapi.NewEditMessageTextAndMarkup(chatId, msgId, text, replyMarkup))
	}
}
