package conf

import (
	"errors"
	"flag"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"math"
	"os"
	"strings"
	"time"
)

const (
	Bsc      = "bsc" // Binance Smart Chain
	Tron     = "tron"
	Aptos    = "aptos"
	Solana   = "solana"
	Xlayer   = "xlayer"
	Polygon  = "polygon"
	Arbitrum = "arbitrum"
	Ethereum = "ethereum"
	Base     = "base"
)

var (
	cfg  Conf
	path string
)

func Init() error {
	flag.StringVar(&path, "conf", "./conf.toml", "config file path")
	flag.Parse()

	data, err := os.ReadFile(path)
	if err != nil {

		return fmt.Errorf("配置文件加载失败：%w", err)
	}

	if err = toml.Unmarshal(data, &cfg); err != nil {

		return fmt.Errorf("配置数据解析失败：%w", err)
	}

	if BotToken() == "" || BotAdminID() == 0 {

		return errors.New("telegram bot 参数 admin_id 或 token 均不能为空")
	}

	return nil
}

func GetUsdtRate() string {

	return cfg.Pay.UsdtRate
}

func GetUsdcRate() string {

	return cfg.Pay.UsdcRate
}

func GetTrxRate() string {

	return cfg.Pay.TrxRate
}

func GetUsdtAtomicity() (decimal.Decimal, int) {
	var val = defaultUsdtAtomicity
	if cfg.Pay.UsdtAtom != 0 {

		val = cfg.Pay.UsdtAtom
	}

	var atom = decimal.NewFromFloat(val)

	return atom, cast.ToInt(math.Abs(float64(atom.Exponent())))
}

func GetUsdcAtomicity() (decimal.Decimal, int) {
	var val = defaultUsdcAtomicity
	if cfg.Pay.UsdcAtom != 0 {

		val = cfg.Pay.UsdcAtom
	}

	var atom = decimal.NewFromFloat(val)

	return atom, cast.ToInt(math.Abs(float64(atom.Exponent())))
}

func GetTrxAtomicity() (decimal.Decimal, int) {
	var val = defaultTrxAtomicity
	if cfg.Pay.TrxAtom != 0 {

		val = cfg.Pay.TrxAtom
	}

	var atom = decimal.NewFromFloat(val)

	return atom, cast.ToInt(math.Abs(float64(atom.Exponent())))
}

func GetExpireTime() time.Duration {
	if cfg.Pay.ExpireTime == 0 {

		return time.Duration(defaultExpireTime)
	}

	return time.Duration(cfg.Pay.ExpireTime)
}

func GetAuthToken() string {
	if cfg.AuthToken == "" {

		return defaultAuthToken
	}

	return cfg.AuthToken
}

func GetAppUri(host string) string {
	if cfg.AppUri != "" {

		return cfg.AppUri
	}

	return host
}

func GetStaticPath() string {

	return cfg.StaticPath
}

func GetSqlitePath() string {
	if cfg.SqlitePath != "" {

		return cfg.SqlitePath
	}

	return defaultSqlitePath
}

func GetOutputLog() string {
	if cfg.OutputLog != "" {

		return cfg.OutputLog
	}

	return defaultOutputLog
}

func GetListen() string {
	if cfg.Listen != "" {

		return cfg.Listen
	}

	return defaultListen
}

func BotToken() string {
	var token = strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if token != "" {

		return token
	}

	return cfg.Bot.Token
}

func BotAdminID() int64 {
	var id = strings.TrimSpace(os.Getenv("BOT_ADMIN_ID"))
	if id != "" {

		return cast.ToInt64(id)
	}

	return cfg.Bot.AdminID
}

func BotNotifyTarget() string {
	if cfg.Bot.GroupID != "" {

		return cfg.Bot.GroupID
	}

	return cast.ToString(cfg.Bot.AdminID)
}

func GetWalletAddress() []string {

	return cfg.Pay.WalletAddress
}

func GetTradeIsConfirmed() bool {

	return cfg.Pay.TradeIsConfirmed
}

func GetPaymentAmountMin() decimal.Decimal {
	var val = defaultPaymentMinAmount
	if cfg.Pay.PaymentAmountMin != 0 {

		val = cfg.Pay.PaymentAmountMin
	}

	return decimal.NewFromFloat(val)
}

func GetPaymentAmountMax() decimal.Decimal {
	var val float64 = defaultPaymentMaxAmount
	if cfg.Pay.PaymentAmountMax != 0 {

		val = cfg.Pay.PaymentAmountMax
	}

	return decimal.NewFromFloat(val)
}

func GetWebhookUrl() string {

	return cfg.WebhookUrl
}
