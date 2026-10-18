package model

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
)

// supportFiat 支持的法定货币
var supportFiat = map[Fiat]struct{}{
	CNY: {},
	USD: {},
	JPY: {},
	EUR: {},
	GBP: {},
}

// supportCrypto 支持的加密货币；Coin Id 参考来源：https://docs.coingecko.com/v3.0.1/reference/coins-list
var supportCrypto = map[Crypto]CoinId{
	USDT: "tether",
	USDC: "usd-coin",
	TRX:  "tron",
	BNB:  "binancecoin",
	ETH:  "ethereum",
	GRAM: "the-open-network",
}

// TradeType 交易类型，当下类型开始增多，现在这里统一管理、尽量收缩配置项
var registry = map[TradeType]TradeTypeConf{
	UsdtTon: {
		Alias:        "USDT・Ton",
		NetworkName:  "Ton",
		Network:      conf.Ton,
		Crypto:       USDT,
		Contract:     conf.UsdtTon,
		Decimal:      conf.UsdtTonDecimals,
		AmountRange:  usdGeneralRange,
		ExplorerFmt:  "https://tonscan.org/tx/%s#overview",
		EndpointKey:  RpcGlobalConfigUrlTon,
		AddrCaseSens: true,
	},
	UsdtPlasma: {
		Alias:       "USDT・Plasma",
		NetworkName: "Plasma",
		Network:     conf.Plasma,
		Crypto:      USDT,
		Contract:    conf.UsdtPlasma,
		Decimal:     conf.UsdtPlasmaDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://plasmascan.to/tx/%s",
		EndpointKey: RpcEndpointPlasma,
	},
	UsdtTrc20: {
		Alias:        "USDT・TRC20",
		NetworkName:  "Tron",
		Network:      conf.Tron,
		Crypto:       USDT,
		Contract:     "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // 占位，目前实际没使用
		Decimal:      conf.UsdtTronDecimals,
		AmountRange:  usdGeneralRange,
		ExplorerFmt:  "https://tronscan.org/#/transaction/%s",
		EndpointKey:  RpcEndpointTron,
		AddrCaseSens: true,
	},
	UsdtErc20: {
		Alias:       "USDT・ERC20",
		NetworkName: "Ethereum",
		Network:     conf.Ethereum,
		Crypto:      USDT,
		Contract:    conf.UsdtErc20,
		Decimal:     conf.UsdtEthDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://etherscan.io/tx/%s",
		EndpointKey: RpcEndpointEthereum,
	},
	UsdtBep20: {
		Alias:       "USDT・BEP20",
		NetworkName: "Bsc",
		Network:     conf.Bsc,
		Crypto:      USDT,
		Contract:    conf.UsdtBep20,
		Decimal:     conf.UsdtBscDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://bscscan.com/tx/%s",
		EndpointKey: RpcEndpointBsc,
	},
	UsdtAptos: {
		Alias:       "USDT・Aptos",
		NetworkName: "Aptos",
		Network:     conf.Aptos,
		Crypto:      USDT,
		Contract:    conf.UsdtAptos,
		Decimal:     conf.UsdtAptosDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://explorer.aptoslabs.com/txn/%s",
		EndpointKey: RpcEndpointAptos,
	},
	UsdtXlayer: {
		Alias:       "USDT・X Layer",
		NetworkName: "X Layer",
		Network:     conf.Xlayer,
		Crypto:      USDT,
		Contract:    conf.UsdtXlayer,
		Decimal:     conf.UsdtXlayerDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://web3.okx.com/zh-hans/explorer/x-layer/tx/%s",
		EndpointKey: RpcEndpointXlayer,
	},
	UsdtSolana: {
		Alias:        "USDT・Solana",
		NetworkName:  "Solana",
		Network:      conf.Solana,
		Crypto:       USDT,
		Contract:     conf.UsdtSolana,
		Decimal:      conf.UsdtSolanaDecimals,
		AmountRange:  usdGeneralRange,
		ExplorerFmt:  "https://solscan.io/tx/%s",
		EndpointKey:  RpcEndpointSolana,
		AddrCaseSens: true,
	},
	UsdtPolygon: {
		Alias:       "USDT・Polygon",
		NetworkName: "Polygon",
		Network:     conf.Polygon,
		Crypto:      USDT,
		Contract:    conf.UsdtPolygon,
		Decimal:     conf.UsdtPolygonDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://polygonscan.com/tx/%s",
		EndpointKey: RpcEndpointPolygon,
	},
	UsdtArbitrum: {
		Alias:       "USDT・Arbitrum",
		NetworkName: "Arbitrum",
		Network:     conf.Arbitrum,
		Crypto:      USDT,
		Contract:    conf.UsdtArbitrum,
		Decimal:     conf.UsdtArbitrumDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://arbiscan.io/tx/%s",
		EndpointKey: RpcEndpointArbitrum,
	},
	UsdcErc20: {
		Alias:       "USDC・ERC20",
		NetworkName: "Ethereum",
		Network:     conf.Ethereum,
		Crypto:      USDC,
		Contract:    conf.UsdcErc20,
		Decimal:     conf.UsdcEthDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://etherscan.io/tx/%s",
		EndpointKey: RpcEndpointEthereum,
	},
	UsdcBep20: {
		Alias:       "USDC・BEP20",
		NetworkName: "Bsc",
		Network:     conf.Bsc,
		Crypto:      USDC,
		Contract:    conf.UsdcBep20,
		Decimal:     conf.UsdcBscDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://bscscan.com/tx/%s",
		EndpointKey: RpcEndpointBsc,
	},
	UsdcXlayer: {
		Alias:       "USDC・X Layer",
		NetworkName: "X Layer",
		Network:     conf.Xlayer,
		Crypto:      USDC,
		Contract:    conf.UsdcXlayer,
		Decimal:     conf.UsdcXlayerDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://web3.okx.com/zh-hans/explorer/x-layer/tx/%s",
		EndpointKey: RpcEndpointXlayer,
	},
	UsdcPolygon: {
		Alias:       "USDC・Polygon",
		NetworkName: "Polygon",
		Network:     conf.Polygon,
		Crypto:      USDC,
		Contract:    conf.UsdcPolygon,
		Decimal:     conf.UsdcPolygonDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://polygonscan.com/tx/%s",
		EndpointKey: RpcEndpointPolygon,
	},
	UsdcArbitrum: {
		Alias:       "USDC・Arbitrum",
		NetworkName: "Arbitrum",
		Network:     conf.Arbitrum,
		Crypto:      USDC,
		Contract:    conf.UsdcArbitrum,
		Decimal:     conf.UsdcArbitrumDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://arbiscan.io/tx/%s",
		EndpointKey: RpcEndpointArbitrum,
	},
	UsdcBase: {
		Alias:       "USDC・Base",
		NetworkName: "Base",
		Network:     conf.Base,
		Crypto:      USDC,
		Contract:    conf.UsdcBase,
		Decimal:     conf.UsdcBaseDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://basescan.org/tx/%s",
		EndpointKey: RpcEndpointBase,
	},
	UsdcTrc20: {
		Alias:        "USDC・TRC20",
		NetworkName:  "Tron",
		Network:      conf.Tron,
		Crypto:       USDC,
		Contract:     "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8", // 占位，目前实际没使用
		Decimal:      conf.UsdcTronDecimals,
		AmountRange:  usdGeneralRange,
		ExplorerFmt:  "https://tronscan.org/#/transaction/%s",
		EndpointKey:  RpcEndpointTron,
		AddrCaseSens: true,
	},
	UsdcSolana: {
		Alias:        "USDC・Solana",
		NetworkName:  "Solana",
		Network:      conf.Solana,
		Crypto:       USDC,
		Contract:     conf.UsdcSolana,
		Decimal:      conf.UsdcSolanaDecimals,
		AmountRange:  usdGeneralRange,
		ExplorerFmt:  "https://solscan.io/tx/%s",
		EndpointKey:  RpcEndpointSolana,
		AddrCaseSens: true,
	},
	UsdcAptos: {
		Alias:       "USDC・Aptos",
		NetworkName: "Aptos",
		Network:     conf.Aptos,
		Crypto:      USDC,
		Contract:    conf.UsdcAptos,
		Decimal:     conf.UsdcAptosDecimals,
		AmountRange: usdGeneralRange,
		ExplorerFmt: "https://explorer.aptoslabs.com/txn/%s",
		EndpointKey: RpcEndpointAptos,
	},
	TronTrx: {
		Alias:       "Tron・Trx",
		NetworkName: "Tron",
		Network:     conf.Tron,
		Crypto:      TRX,
		Native:      true,
		Decimal:     -6,
		AmountRange: Range{
			MinAmount: decimal.NewFromFloat(0.1),
			MaxAmount: decimal.NewFromFloat(1000000),
		},
		ExplorerFmt:  "https://tronscan.org/#/transaction/%s",
		EndpointKey:  RpcEndpointTron,
		AddrCaseSens: true,
	},
	EthereumEth: {
		Alias:       "Ethereum・Eth",
		NetworkName: "Ethereum",
		Network:     conf.Ethereum,
		Crypto:      ETH,
		Native:      true,
		Decimal:     conf.EthereumEthDecimals,
		AmountRange: Range{
			MinAmount: decimal.NewFromFloat(0.000001),
			MaxAmount: decimal.NewFromFloat(1000000),
		},
		ExplorerFmt: "https://etherscan.io/tx/%s",
		EndpointKey: RpcEndpointEthereum,
	},
	BscBnb: {
		Alias:       "Bsc・Bnb",
		NetworkName: "Bsc",
		Network:     conf.Bsc,
		Crypto:      BNB,
		Native:      true,
		Decimal:     conf.BscBnbDecimals,
		AmountRange: Range{
			MinAmount: decimal.NewFromFloat(0.00001),
			MaxAmount: decimal.NewFromFloat(1000000),
		},
		ExplorerFmt: "https://bscscan.com/tx/%s",
		EndpointKey: RpcEndpointBsc,
	},
	TonGram: {
		Alias:       "Ton・Gram",
		NetworkName: "Ton",
		Network:     conf.Ton,
		Crypto:      GRAM,
		Native:      true,
		Decimal:     conf.TonTonDecimals,
		AmountRange: Range{
			MinAmount: decimal.NewFromFloat(0.00001),
			MaxAmount: decimal.NewFromFloat(1000000),
		},
		ExplorerFmt:  "https://tonscan.org/tx/%s#overview",
		EndpointKey:  RpcGlobalConfigUrlTon,
		AddrCaseSens: true,
	},
}

func init() {
	for t, c := range registry {
		cryptoAtomKeys[c.Crypto] = ConfKey(fmt.Sprintf("atom_%s", strings.ToLower(string(c.Crypto))))
		explorerUrlMap[t] = c.ExplorerFmt
		networkTradesMap[c.Network] = append(networkTradesMap[c.Network], t)
		networkEndpointMap[c.Network] = c.EndpointKey
		if c.Contract != "" {
			contractDecimalMap[c.Contract] = c.Decimal
			contractTradeMap[c.Contract] = t
		}
		if c.AmountRange != (Range{}) {
			tradeAmountRangeMap[t] = c.AmountRange
		}
	}
}

func IsSupportedTradeType(t TradeType) bool {
	_, ok := registry[t]

	return ok
}

func GetCrypto(t TradeType) (Crypto, error) {
	if c, ok := registry[t]; ok {
		return c.Crypto, nil
	}
	return "", fmt.Errorf("unsupported trade type: %s", t)
}

func GetAllAlias() map[string]string {
	var alias = make(map[string]string)

	for t, c := range registry {
		alias[string(t)] = c.Alias
	}

	return alias
}

func GetAllNetwork() map[string]string {
	var networks = make(map[string]string)
	for _, c := range registry {
		networks[string(c.Network)] = c.NetworkName
	}

	return networks
}

func GetAllTradeConfig() map[string]TradeTypeConf {
	var config = make(map[string]TradeTypeConf)

	for t, c := range registry {
		config[string(t)] = c
	}

	return config
}

func GetNetworkTrades(n Network) []TradeType {
	list, ok := networkTradesMap[n]
	if !ok {
		return []TradeType{}
	}

	return list
}

func GetContractTrade(addr string) (TradeType, bool) {
	t, ok := contractTradeMap[addr]

	return t, ok
}

func GetContractDecimal(addr string) int32 {
	if d, ok := contractDecimalMap[addr]; ok {

		return d
	}

	return -6
}

func GetTxUrl(t TradeType, hash string) string {
	if url, ok := explorerUrlMap[t]; ok {
		if url == "" {
			return ""
		}
		return fmt.Sprintf(url, hash)
	}

	return "https://tronscan.org/#/transaction/" + hash
}

func GetTradeContract(t TradeType) string {
	c, ok := registry[t]
	if !ok {

		return ""
	}

	return c.Contract
}

func GetTradeDecimal(t TradeType) int32 {
	c, ok := registry[t]
	if !ok {

		return -6
	}

	return c.Decimal
}

func IsAmountValid(t TradeType, d decimal.Decimal) bool {
	r, ok := tradeAmountRangeMap[t]
	if !ok {

		return false
	}

	if r.MaxAmount.Cmp(d) < 0 {

		return false
	}

	if r.MinAmount.Cmp(d) > 0 {

		return false
	}

	return true
}

func Endpoint(net Network) string {
	if endpointKey, ok := networkEndpointMap[net]; ok {
		return GetC(endpointKey)
	}
	return ""
}

func GetTradeAtomKey(t TradeType) (ConfKey, bool) {
	if c, ok := registry[t]; ok {
		atomKey, ok := cryptoAtomKeys[c.Crypto]

		return atomKey, ok
	}

	return "", false
}

func GetSupportFiat() map[Fiat]struct{} {

	return supportFiat
}

func GetSupportCrypto() map[Crypto]CoinId {

	return supportCrypto
}

func AddrCaseSens(t TradeType) bool {
	if c, ok := registry[t]; ok {

		return c.AddrCaseSens
	}

	return true
}

func GetTradeTypeByCurrencyAndNetwork(currency, network string) (TradeType, error) {
	for t, c := range registry {
		if string(c.Crypto) == currency && (string(c.Network) == network || c.NetworkName == network) {
			return t, nil
		}
	}
	return "", fmt.Errorf("no matching trade type found for currency %s on network %s", currency, network)
}
