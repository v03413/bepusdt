package model

import (
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB
var _err error

func Init() error {
	var dbPath = config.GetDbPath()
	if !help.IsExist(dbPath) {
		DB, _err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if _err != nil {

			return _err
		}

		DB.Exec(installSql)

		return nil
	}

	DB, _err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})

	return _err
}
