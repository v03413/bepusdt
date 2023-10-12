package monitor

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/telegram"
)

var err error

func BotStart() {
	var BotApi = telegram.GetBotApi()
	if BotApi == nil {

		return
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := BotApi.GetUpdatesChan(u)
	if err != nil {
		log.Error("TG Bot GetUpdatesChan Error:", err)

		return
	}

	// 监听消息
	for _u := range updates {
		if _u.Message != nil {
			if !_u.FromChat().IsPrivate() {

				continue
			}

			telegram.HandleMessage(_u.Message)
		}
		if _u.CallbackQuery != nil {

			telegram.HandleCallback(_u.CallbackQuery)
		}
	}
}
