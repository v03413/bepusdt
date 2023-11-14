package model

import (
	"time"
)

type NotifyRecord struct {
	Txid      string    `gorm:"primary_key;type:varchar(64);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp;not null"`
}

func (nr *NotifyRecord) TableName() string {

	return "notify_record"
}

func IsNeedNotifyByTxid(txid string) bool {
	var row NotifyRecord
	var res = DB.Where("txid = ?", txid).Limit(1).Find(&row)
	if res.RowsAffected > 0 {

		return false
	}

	var row2 TradeOrders
	var res2 = DB.Where("trade_hash = ?", txid).Limit(1).Find(&row2)
	if res2.RowsAffected > 0 {

		return false
	}

	return true
}
