package bot

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v03413/bepusdt/app/conf"
	"strings"
)

func updateFilter(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		var allow bool
		var admin = conf.BotAdminID()

		// 只处理管理员私聊消息
		if update.Message != nil && update.Message.Chat.Type == models.ChatTypePrivate && update.Message.Chat.ID == admin {

			allow = true
		}

		// 只处理管理员回调消息
		if update.CallbackQuery != nil && update.CallbackQuery.From.ID == admin {
			ctx = context.WithValue(ctx, "args", strings.Split(update.CallbackQuery.Data, "|"))

			allow = true
		}

		if !allow {

			return
		}

		next(ctx, bot, update)
	}
}
