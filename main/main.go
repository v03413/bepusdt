package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/bot"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/task"
	"github.com/v03413/bepusdt/app/web"
)

type Initializer func() error

var initializers = []Initializer{conf.Init, log.Init, bot.Init, model.Init, task.Init}

func init() {
	for _, initFunc := range initializers {
		if err := initFunc(); err != nil {

			panic(fmt.Sprintf("初始化失败: %v", err))
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task.Start(ctx)
	web.Start(ctx)

	fmt.Println("BEpusdt 启动成功，当前版本：" + app.Version)

	{
		var signals = make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, os.Kill)
		<-signals
		cancel()
		runtime.GC()
	}
}
