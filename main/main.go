package main

import (
	"fmt"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/bot"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/task"
	"github.com/v03413/bepusdt/app/web"
	"os"
	"os/signal"
	"runtime"
)

func init() {
	conf.Init()
	log.Init()
	bot.Init()
	model.Init()

	if conf.BotToken() == "" || conf.BotAdminID() == 0 {

		panic("请配置参数 BOT_TOKEN 和 BOT_ADMIN_ID")
	}
}

func main() {
	task.Start()

	web.Start()

	fmt.Println("Bepusdt 启动成功，当前版本：" + app.Version)

	{
		var signals = make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, os.Kill)
		<-signals
		runtime.GC()
	}
}
