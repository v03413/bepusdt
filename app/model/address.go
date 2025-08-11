package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
)

const (
	StatusEnable       uint8 = 1
	StatusDisable      uint8 = 0
	OtherNotifyEnable  uint8 = 1
	OtherNotifyDisable uint8 = 0
)

type TokenType string

const (
	TokenTypeUSDT TokenType = "USDT"
	TokenTypeUSDC TokenType = "USDC"
	TokenTypeTRX  TokenType = "TRX"
)

// SupportTradeTypes 目前支持的收款交易类型
var SupportTradeTypes = []string{
	OrderTradeTypeTronTrx,
	OrderTradeTypeUsdtTrc20,
	OrderTradeTypeUsdtErc20,
	OrderTradeTypeUsdtBep20,
	OrderTradeTypeUsdtAptos,
	OrderTradeTypeUsdtXlayer,
	OrderTradeTypeUsdtSolana,
	OrderTradeTypeUsdtPolygon,
	OrderTradeTypeUsdtArbitrum,
	OrderTradeTypeUsdcErc20,
	OrderTradeTypeUsdcBep20,
	OrderTradeTypeUsdcXlayer,
	OrderTradeTypeUsdcPolygon,
	OrderTradeTypeUsdcArbitrum,
	OrderTradeTypeUsdcBase,
	OrderTradeTypeUsdcTrc20,
	OrderTradeTypeUsdcSolana,
	OrderTradeTypeUsdcAptos,
}

var tradeTypeTable = map[string]TokenType{
	// USDT
	OrderTradeTypeUsdtTrc20:    TokenTypeUSDT,
	OrderTradeTypeUsdtErc20:    TokenTypeUSDT,
	OrderTradeTypeUsdtBep20:    TokenTypeUSDT,
	OrderTradeTypeUsdtAptos:    TokenTypeUSDT,
	OrderTradeTypeUsdtXlayer:   TokenTypeUSDT,
	OrderTradeTypeUsdtSolana:   TokenTypeUSDT,
	OrderTradeTypeUsdtPolygon:  TokenTypeUSDT,
	OrderTradeTypeUsdtArbitrum: TokenTypeUSDT,

	// USDC
	OrderTradeTypeUsdcErc20:    TokenTypeUSDC,
	OrderTradeTypeUsdcBep20:    TokenTypeUSDC,
	OrderTradeTypeUsdcXlayer:   TokenTypeUSDC,
	OrderTradeTypeUsdcPolygon:  TokenTypeUSDC,
	OrderTradeTypeUsdcArbitrum: TokenTypeUSDC,
	OrderTradeTypeUsdcBase:     TokenTypeUSDC,
	OrderTradeTypeUsdcTrc20:    TokenTypeUSDC,
	OrderTradeTypeUsdcSolana:   TokenTypeUSDC,
	OrderTradeTypeUsdcAptos:    TokenTypeUSDC,

	// TRX
	OrderTradeTypeTronTrx: TokenTypeTRX,
}

type WalletAddress struct {
	ID          int64     `gorm:"integer;primaryKey;not null;comment:id"`
	Status      uint8     `gorm:"column:status;type:tinyint(1);not null;default:1;comment:地址状态"`
	TradeType   string    `gorm:"column:trade_type;type:varchar(20);not null;comment:交易类型"`
	Address     string    `gorm:"column:address;type:varchar(64);not null;index;comment:钱包地址"`
	OtherNotify uint8     `gorm:"column:other_notify;type:tinyint(1);not null;default:0;comment:其它通知"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;type:timestamp;not null;comment:更新时间"`
}

// 启动时添加初始钱包地址
func addStartWalletAddress() {
	for _, itm := range conf.GetWalletAddress() {
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

func (wa *WalletAddress) GetTokenContract() string {
	switch wa.TradeType {
	case OrderTradeTypeUsdtPolygon:
		return conf.UsdtPolygon
	case OrderTradeTypeUsdtArbitrum:
		return conf.UsdtArbitrum
	case OrderTradeTypeUsdtErc20:
		return conf.UsdtErc20
	case OrderTradeTypeUsdtBep20:
		return conf.UsdtBep20
	case OrderTradeTypeUsdtXlayer:
		return conf.UsdtXlayer
	case OrderTradeTypeUsdtAptos:
		return conf.UsdtAptos
	case OrderTradeTypeUsdtSolana:
		return conf.UsdtSolana
	case OrderTradeTypeUsdcErc20:
		return conf.UsdcErc20
	case OrderTradeTypeUsdcBep20:
		return conf.UsdcBep20
	case OrderTradeTypeUsdcXlayer:
		return conf.UsdcXlayer
	case OrderTradeTypeUsdcPolygon:
		return conf.UsdcPolygon
	case OrderTradeTypeUsdcArbitrum:
		return conf.UsdcArbitrum
	case OrderTradeTypeUsdcBase:
		return conf.UsdcBase
	case OrderTradeTypeUsdcAptos:
		return conf.UsdcAptos
	case OrderTradeTypeUsdcSolana:
		return conf.UsdcSolana
	default:
		return ""
	}
}

func (wa *WalletAddress) GetTokenDecimals() int32 {
	switch wa.TradeType {
	case OrderTradeTypeUsdtPolygon:
		return conf.UsdtPolygonDecimals
	case OrderTradeTypeUsdtArbitrum:
		return conf.UsdtArbitrumDecimals
	case OrderTradeTypeUsdtErc20:
		return conf.UsdtEthDecimals
	case OrderTradeTypeUsdtBep20:
		return conf.UsdtBscDecimals
	case OrderTradeTypeUsdtAptos:
		return conf.UsdtAptosDecimals
	case OrderTradeTypeUsdtXlayer:
		return conf.UsdtXlayerDecimals
	case OrderTradeTypeUsdtSolana:
		return conf.UsdtSolanaDecimals
	case OrderTradeTypeUsdcErc20:
		return conf.UsdcEthDecimals
	case OrderTradeTypeUsdcBep20:
		return conf.UsdcBscDecimals
	case OrderTradeTypeUsdcXlayer:
		return conf.UsdcXlayerDecimals
	case OrderTradeTypeUsdcPolygon:
		return conf.UsdcPolygonDecimals
	case OrderTradeTypeUsdcArbitrum:
		return conf.UsdcArbitrumDecimals
	case OrderTradeTypeUsdcBase:
		return conf.UsdcBaseDecimals
	case OrderTradeTypeUsdcSolana:
		return conf.UsdcSolanaDecimals
	case OrderTradeTypeUsdcAptos:
		return conf.UsdcAptosDecimals
	default:
		return -6
	}
}

func (wa *WalletAddress) GetEvmRpcEndpoint() string {
	switch wa.TradeType {
	case OrderTradeTypeUsdtPolygon:
		return conf.GetPolygonRpcEndpoint()
	case OrderTradeTypeUsdtArbitrum:
		return conf.GetArbitrumRpcEndpoint()
	case OrderTradeTypeUsdtErc20:
		return conf.GetEthereumRpcEndpoint()
	case OrderTradeTypeUsdtBep20:
		return conf.GetBscRpcEndpoint()
	case OrderTradeTypeUsdtXlayer:
		return conf.GetXlayerRpcEndpoint()
	case OrderTradeTypeUsdcErc20:
		return conf.GetEthereumRpcEndpoint()
	case OrderTradeTypeUsdcBep20:
		return conf.GetBscRpcEndpoint()
	case OrderTradeTypeUsdcXlayer:
		return conf.GetXlayerRpcEndpoint()
	case OrderTradeTypeUsdcPolygon:
		return conf.GetPolygonRpcEndpoint()
	case OrderTradeTypeUsdcArbitrum:
		return conf.GetArbitrumRpcEndpoint()
	case OrderTradeTypeUsdcBase:
		return conf.GetBaseRpcEndpoint()
	default:
		return ""
	}
}

func GetTokenType(tradeType string) (TokenType, error) {
	if f, ok := tradeTypeTable[tradeType]; ok {
		return f, nil
	}
	return "", fmt.Errorf("unsupported trade type: %s", tradeType)
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
