package main

import (
	"fmt"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/monitor"
	"github.com/v03413/bepusdt/app/web"
	"os"
	"os/signal"
	"runtime"
)

// Version 版本号说明 1.0.0 代表主版本号.功能版本号.修订号
const Version = "1.7.17"

func main() {
	if err := model.Init(); err != nil {

		panic("数据库初始化失败：" + err.Error())
	}

	if config.GetTGBotToken() == "" || config.GetTGBotAdminId() == "" {

		panic("请配置参数 TG_BOT_TOKEN 和 TG_BOT_ADMIN_ID")
	}

	// 启动机器人
	go monitor.BotStart(Version)

	// 启动汇率监控
	go monitor.OkxUsdtRateStart()

	// 启动交易监控
	go monitor.TradeStart()

	// 启动回调监控
	go monitor.NotifyStart()

	// 启动web服务
	go web.Start()

	fmt.Println("Bepusdt 启动成功，当前版本：" + Version)
	// 优雅退出
	{
		var signals = make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, os.Kill)
		<-signals
		runtime.GC()
	}
}
