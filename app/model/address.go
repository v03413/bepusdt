package model

import (
	"errors"
	"fmt"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"gorm.io/gorm"
	"time"
)

const StatusEnable = 1
const StatusDisable = 0
const OtherNotifyEnable = 1
const OtherNotifyDisable = 0

type WalletAddress struct {
	Id          int64     `gorm:"integer;primaryKey;not null;comment:id"`
	Address     string    `gorm:"type:varchar(255);not null;unique;comment:钱包地址"`
	Status      int       `gorm:"type:tinyint(1);not null;default:1;comment:地址状态 1启动 0禁止"`
	OtherNotify int       `gorm:"type:tinyint(1);not null;default:1;comment:其它转账通知 1启动 0禁止"`
	CreatedAt   time.Time `gorm:"autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;type:timestamp;not null;comment:更新时间"`
}

// 启动时添加初始钱包地址
func addStartWalletAddress() {
	var _wa WalletAddress

	for _, address := range config.GetInitWalletAddress() {
		if help.IsValidTRONWalletAddress(address) {
			var _res2 = DB.Where("address = ?", address).First(&_wa)
			if errors.Is(_res2.Error, gorm.ErrRecordNotFound) {
				var _row = WalletAddress{Address: address, Status: StatusEnable}
				var _res = DB.Create(&_row)
				if _res.Error == nil && _res.RowsAffected == 1 {
					fmt.Println("✅钱包地址添加成功：", address)
				}
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
