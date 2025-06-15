package model

import (
	"fmt"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"strings"
	"time"
)

const (
	StatusEnable       uint8 = 1
	StatusDisable      uint8 = 0
	OtherNotifyEnable  uint8 = 1
	OtherNotifyDisable uint8 = 0
)

// SupportTradeTypes 目前支持的收款交易类型
var SupportTradeTypes = []string{
	OrderTradeTypeTronTrx,
	OrderTradeTypeUsdtTrc20,
	OrderTradeTypeUsdtErc20,
	OrderTradeTypeUsdtBep20,
	OrderTradeTypeUsdtXlayer,
	OrderTradeTypeUsdtPolygon,
}

type WalletAddress struct {
	ID          int64     `gorm:"integer;primaryKey;not null;comment:id"`
	Status      uint8     `gorm:"column:status;type:tinyint(1);not null;default:1;comment:地址状态"`
	TradeType   string    `gorm:"column:trade_type;type:varchar(20);not null;comment:交易类型"`
	Address     string    `gorm:"column:address;type:varchar(64);not null;index;comment:钱包地址"`
	OtherNotify uint8     `gorm:"column:other_notify;type:tinyint(1);not null;default:0;comment:其它转账通知"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;type:timestamp;not null;comment:更新时间"`
}

// 启动时添加初始钱包地址
func addStartWalletAddress() {
	for _, itm := range conf.GetWalletAddress() {
		fmt.Println(itm)
		var info = strings.Split(itm, ":")
		if len(info) != 2 {

			continue
		}

		var address = info[1]
		var tradeType = info[0]

		if !help.IsValidTronAddress(address) && !help.IsValidEvmAddress(address) {
			fmt.Println("❌钱包地址不合法：", address)

			continue
		}

		if help.IsValidEvmAddress(address) {

			address = strings.ToLower(address)
		}

		var wa WalletAddress
		DB.Where("address = ? and trade_type = ?", address, tradeType).Limit(1).Find(&wa)
		if wa.ID != 0 {

			continue
		}

		var err = DB.Create(&WalletAddress{TradeType: tradeType, Address: address, Status: StatusEnable}).Error
		if err != nil {
			fmt.Println("❌钱包地址添加失败：", err)

			continue
		}

		fmt.Println("✅钱包地址添加成功：", tradeType, address)
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
	var db = DB.Where("trade_type = ?", tradeType)
	if address != "" {

		db = db.Where("address = ?", address)
	}

	db.Find(&rows)

	if len(rows) == 0 && address != "" {
		var wa = WalletAddress{TradeType: tradeType, Address: address, Status: StatusEnable, OtherNotify: OtherNotifyDisable}

		DB.Create(&wa)

		return []WalletAddress{wa}
	}

	return rows
}
