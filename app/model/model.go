package model

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/v03413/bepusdt/app/conf"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

var DB *gorm.DB
var err error

func Init() error {
	var path = conf.GetSqlitePath()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {

		return fmt.Errorf("创建数据库目录失败：%w", err)
	}

	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {

		return fmt.Errorf("数据库初始化失败：%w", err)
	}

	if err = AutoMigrate(); err != nil {

		return fmt.Errorf("数据库结构迁移失败：%w", err)
	}

	addStartWalletAddress()

	return nil
}

func AutoMigrate() error {

	return DB.AutoMigrate(&WalletAddress{}, &TradeOrders{}, &NotifyRecord{}, &Config{}, &Webhook{})
}
