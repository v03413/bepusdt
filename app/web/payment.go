package web

import (
	"github.com/v03413/bepusdt/app/model"
)

type Payment struct {
	Coin           string
	Network        string
	NetworkDisplay string
	NetworkSuffix  string
	WarningCoin    string
}

var paymentMap = map[string]Payment{
	model.OrderTradeTypeTronTrx: {
		Coin:           "TRX",
		Network:        "Tron",
		NetworkDisplay: "Tron",
		NetworkSuffix:  "",
		WarningCoin:    "USDT",
	},
	model.OrderTradeTypeUsdtTrc20: {
		Coin:           "USDT",
		Network:        "TRC20",
		NetworkDisplay: "波场 (TRON)",
		NetworkSuffix:  "TRC20",
		WarningCoin:    "TRX",
	},
	model.OrderTradeTypeUsdtErc20: {
		Coin:           "USDT",
		Network:        "ERC20",
		NetworkDisplay: "以太坊 (Ethereum)",
		NetworkSuffix:  "ERC20",
		WarningCoin:    "ETH",
	},
	model.OrderTradeTypeUsdtBep20: {
		Coin:           "USDT",
		Network:        "BEP20",
		NetworkDisplay: "币安智能链 (BSC)",
		NetworkSuffix:  "BEP20",
		WarningCoin:    "BNB",
	},
	model.OrderTradeTypeUsdtPolygon: {
		Coin:           "USDT",
		Network:        "Polygon",
		NetworkDisplay: "Polygon",
		NetworkSuffix:  "",
		WarningCoin:    "MATIC",
	},
	model.OrderTradeTypeUsdtArbitrum: {
		Coin:           "USDT",
		Network:        "Arbitrum",
		NetworkDisplay: "Arbitrum One",
		NetworkSuffix:  "",
		WarningCoin:    "ARB",
	},
	model.OrderTradeTypeUsdtXlayer: {
		Coin:           "USDT",
		Network:        "X Layer",
		NetworkDisplay: "X Layer",
		NetworkSuffix:  "",
		WarningCoin:    "OKB",
	},
	model.OrderTradeTypeUsdtSolana: {
		Coin:           "USDT",
		Network:        "Solana",
		NetworkDisplay: "Solana",
		NetworkSuffix:  "",
		WarningCoin:    "SOL",
	},
	model.OrderTradeTypeUsdtAptos: {
		Coin:           "USDT",
		Network:        "Aptos",
		NetworkDisplay: "Aptos",
		NetworkSuffix:  "",
		WarningCoin:    "APT",
	},
	model.OrderTradeTypeUsdcTrc20: {
		Coin:           "USDC",
		Network:        "TRC20",
		NetworkDisplay: "波场 (TRON)",
		NetworkSuffix:  "TRC20",
		WarningCoin:    "TRX",
	},
	model.OrderTradeTypeUsdcErc20: {
		Coin:           "USDC",
		Network:        "ERC20",
		NetworkDisplay: "以太坊 (Ethereum)",
		NetworkSuffix:  "ERC20",
		WarningCoin:    "ETH",
	},
	model.OrderTradeTypeUsdcBep20: {
		Coin:           "USDC",
		Network:        "BEP20",
		NetworkDisplay: "币安智能链 (BSC)",
		NetworkSuffix:  "BEP20",
		WarningCoin:    "BNB",
	},
	model.OrderTradeTypeUsdcPolygon: {
		Coin:           "USDC",
		Network:        "Polygon",
		NetworkDisplay: "Polygon",
		NetworkSuffix:  "",
		WarningCoin:    "MATIC",
	},
	model.OrderTradeTypeUsdcArbitrum: {
		Coin:           "USDC",
		Network:        "Arbitrum",
		NetworkDisplay: "Arbitrum One",
		NetworkSuffix:  "",
		WarningCoin:    "ARB",
	},
	model.OrderTradeTypeUsdcXlayer: {
		Coin:           "USDC",
		Network:        "X Layer",
		NetworkDisplay: "X Layer",
		NetworkSuffix:  "",
		WarningCoin:    "OKB",
	},
	model.OrderTradeTypeUsdcBase: {
		Coin:           "USDC",
		Network:        "Base",
		NetworkDisplay: "Base",
		NetworkSuffix:  "",
		WarningCoin:    "ETH",
	},
	model.OrderTradeTypeUsdcSolana: {
		Coin:           "USDC",
		Network:        "Solana",
		NetworkDisplay: "Solana",
		NetworkSuffix:  "",
		WarningCoin:    "SOL",
	},
	model.OrderTradeTypeUsdcAptos: {
		Coin:           "USDC",
		Network:        "Aptos",
		NetworkDisplay: "Aptos",
		NetworkSuffix:  "",
		WarningCoin:    "APT",
	},
}

func GetPayment(tradeType string) (Payment, bool) {
	config, exists := paymentMap[tradeType]
	return config, exists
}
