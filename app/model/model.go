package model

import (
	"github.com/glebarez/sqlite"
	"github.com/v03413/bepusdt/app/conf"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

var DB *gorm.DB
var err error

func Init() {
	var path = conf.GetSqlitePath()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {

		panic("创建数据库目录失败：" + err.Error())
	}

	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {

		panic("数据库初始化失败：" + err.Error())
	}

	if err = AutoMigrate(); err != nil {

		panic("数据库结构迁移失败：" + err.Error())
	}

	addStartWalletAddress()
}

func AutoMigrate() error {

	return DB.AutoMigrate(&WalletAddress{}, &TradeOrders{}, &NotifyRecord{}, &Config{})
}
