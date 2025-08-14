package task

import (
	"context"

	"github.com/v03413/bepusdt/app/bot"
)

func init() {
	register(task{callback: botStart})
}

func botStart(ctx context.Context) {
	bot.Start(ctx)
}
