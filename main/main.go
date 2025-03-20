package main

import (
	"fmt"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/task"
	"github.com/v03413/bepusdt/app/web"
	"os"
	"os/signal"
	"runtime"
)

func main() {
	if err := Init(); err != nil {

		panic(err)
	}

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
