package main

import (
	"fmt"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/model"
)

func Init() error {
	if err := model.Init(); err != nil {

		return fmt.Errorf("数据库初始化失败：" + err.Error())
	}

	if config.GetTGBotToken() == "" || config.GetTGBotAdminId() == "" {

		return fmt.Errorf("请配置参数 TG_BOT_TOKEN 和 TG_BOT_ADMIN_ID")
	}

	return nil
}
