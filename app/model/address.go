package model

import (
	"fmt"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"time"
)

const StatusEnable = 1
const StatusDisable = 0
const OtherNotifyEnable = 1
const OtherNotifyDisable = 0

type WalletAddress struct {
	Id          int64     `gorm:"primary_key;AUTO_INCREMENT"`
	Address     string    `gorm:"type:varchar(255);not null;unique"`
	Status      int       `gorm:"type:tinyint(1);not null;default:1"`
	OtherNotify int       `gorm:"type:tinyint(1);not null;default:1"`
	CreatedAt   time.Time `gorm:"autoCreateTime;type:timestamp;not null"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;type:timestamp;not null"`
}

// 启动时添加初始钱包地址
func addStartWalletAddress() {
	for _, address := range config.GetInitWalletAddress() {
		if help.IsValidTRONWalletAddress(address) {
			var _row = WalletAddress{Address: address, Status: StatusEnable}
			var _res = DB.Create(&_row)
			if _res.Error == nil && _res.RowsAffected == 1 {

				fmt.Println("✅钱包地址添加成功：", address)
			}
		}
	}
}

func (wa *WalletAddress) TableName() string {

	return "wallet_address"
}

func (wa *WalletAddress) SetStatus(status int) {
	wa.Status = status
	DB.Save(wa)
}

func (wa *WalletAddress) SetOtherNotify(notify int) {
	wa.OtherNotify = notify

	DB.Save(wa)
}

func (wa *WalletAddress) Delete() {
	DB.Delete(wa)
}

func GetAvailableAddress() []WalletAddress {
	var rows []WalletAddress

	DB.Where("status = ?", StatusEnable).Find(&rows)

	return rows
}

func GetOtherNotify(address string) bool {
	var row WalletAddress
	var res = DB.Where("status = ? and address = ?", StatusEnable, address).First(&row)
	if res.Error != nil {

		return false
	}

	return row.OtherNotify == 1
}
