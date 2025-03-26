package task

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/bot"
	"github.com/v03413/bepusdt/app/log"
	"time"
)

var err error

func init() {
	RegisterSchedule(0, BotStart)
}

func BotStart(time.Duration) {
	var version = app.Version
	var botApi = bot.GetBotApi()
	if botApi == nil {

		return
	}

	_, err = botApi.MakeRequest("deleteWebhook", tgbotapi.Params{})
	if err != nil {

		log.Error("TG Bot deleteWebhook Error:", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botApi.GetUpdatesChan(u)
	if err != nil {
		log.Error("TG Bot GetUpdatesChan Error:", err)

		return
	}

	bot.SendWelcome(version)

	// 监听消息
	for _u := range updates {
		if _u.Message != nil {
			if !_u.FromChat().IsPrivate() {

				continue
			}

			bot.HandleMessage(_u.Message)
		}
		if _u.CallbackQuery != nil {

			bot.HandleCallback(_u.CallbackQuery)
		}
	}
}
