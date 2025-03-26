package model

import (
	"fmt"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"time"
)

const (
	StatusEnable       uint8 = 1
	StatusDisable      uint8 = 0
	OtherNotifyEnable  uint8 = 1
	OtherNotifyDisable uint8 = 0
	WaChainTron              = "tron"
	WaChainPolygon           = "polygon"
)

var tradeChain = map[string]string{
	OrderTradeTypeTronTrx:     WaChainTron,
	OrderTradeTypeUsdtTrc20:   WaChainTron,
	OrderTradeTypeUsdtPolygon: WaChainPolygon,
}

type WalletAddress struct {
	ID          int64     `gorm:"integer;primaryKey;not null;comment:id"`
	Status      uint8     `gorm:"column:status;type:tinyint(1);not null;default:1;comment:地址状态 1启动 0禁止"`
	Chain       string    `gorm:"column:chain;type:varchar(16);not null;default:'tron';comment:链类型"`
	Address     string    `gorm:"column:address;type:varchar(64);not null;uniqueIndex;comment:钱包地址"`
	OtherNotify uint8     `gorm:"column:other_notify;type:tinyint(1);not null;default:0;comment:其它转账通知 1启动 0禁止"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;type:timestamp;not null;comment:更新时间"`
}

// 启动时添加初始钱包地址
func addStartWalletAddress() {
	for _, address := range conf.GetWalletAddress() {
		if !help.IsValidTronAddress(address) && !help.IsValidPolygonAddress(address) {
			fmt.Println("❌钱包地址不合法：", address)

			continue
		}

		var chain = WaChainTron
		if help.IsValidPolygonAddress(address) {
			chain = WaChainPolygon
		}

		var wa WalletAddress
		DB.Where("address = ?", address).Limit(1).Find(&wa)
		if wa.ID != 0 {

			continue
		}

		var err = DB.Create(&WalletAddress{Chain: chain, Address: address, Status: StatusEnable}).Error
		if err != nil {
			fmt.Println("❌钱包地址添加失败：", err)

			continue
		}

		fmt.Println("✅钱包地址添加成功：", chain, address)
	}
}

func (wa *WalletAddress) TableName() string {

	return "wallet_address"
}

func (wa *WalletAddress) SetStatus(status uint8) {
	wa.Status = status
	DB.Save(wa)
}

func (wa *WalletAddress) SetOtherNotify(notify uint8) {
	wa.OtherNotify = notify

	DB.Save(wa)
}

func (wa *WalletAddress) Delete() {
	DB.Delete(wa)
}

func GetAvailableAddress(address, tradeType string) []WalletAddress {
	var rows []WalletAddress
	var chain = tradeChain[tradeType]
	if address == "" {
		DB.Where("chain = ? and status = ?", chain, StatusEnable).Find(&rows)

		return rows
	}

	DB.Where("address = ?", address).Find(&rows)
	if len(rows) == 0 {
		var wa = WalletAddress{Chain: chain, Address: address, Status: StatusEnable, OtherNotify: OtherNotifyDisable}

		DB.Create(&wa)

		return []WalletAddress{wa}
	}

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
