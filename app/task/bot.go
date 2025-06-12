package task

import (
	"context"
	"github.com/v03413/bepusdt/app/bot"
)

var err error

func init() {
	register(task{callback: BotStart})
}

func BotStart(context.Context) {

	bot.Start()
}
