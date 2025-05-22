package task

import (
	"github.com/v03413/bepusdt/app/bot"
	"time"
)

var err error

func init() {
	RegisterSchedule(0, BotStart)
}

func BotStart(time.Duration) {

	bot.Start()
}
