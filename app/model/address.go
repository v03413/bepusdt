package model

import (
	"time"
)

const StatusEnable = 1
const StatusDisable = 0

type WalletAddress struct {
	Id        int64     `gorm:"primary_key;AUTO_INCREMENT"`
	Address   string    `gorm:"type:varchar(255);not null;unique"`
	Status    int       `gorm:"type:tinyint(1);not null;default:1"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp;not null"`
}

func (wa *WalletAddress) TableName() string {

	return "wallet_address"
}

func (wa *WalletAddress) SetStatus(status int) {
	wa.Status = status
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
