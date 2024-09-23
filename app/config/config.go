package config

import (
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/help"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultExpireTime = 600     // 订单默认有效期 10分钟
const defaultUsdtRate = 6.4       // 默认汇率
const defaultAuthToken = "123234" // 默认授权码
const defaultListen = ":8080"     // 默认监听地址
const defaultPaymentMinAmount = 0.01
const defaultPaymentMaxAmount = 99999
const defaultAtomicity = "0.01"                  // 原子精度
const defaultTronGrpcNode = "18.141.79.38:50051" // 默认GRPC节点

var runPath string

func init() {
	execPath, err := os.Executable()
	if err != nil {

		panic(err)
	}

	runPath = filepath.Dir(execPath)
}

func GetTronGrpcNode() string {
	if data := help.GetEnv("TRON_GRPC_NODE"); data != "" {

		return strings.TrimSpace(data)
	}

	return defaultTronGrpcNode
}

func GetAtomicity() (decimal.Decimal, int) {
	var _defaultAtom, _ = decimal.NewFromString(defaultAtomicity)
	var _defaultExp = cast.ToInt(math.Abs(float64(_defaultAtom.Exponent())))
	if data := help.GetEnv("USDT_ATOM"); data != "" {
		var _atom, err = decimal.NewFromString(data)
		if err != nil {

			return _defaultAtom, _defaultExp
		}

		// 如果大于0，且小数点后位数大于0
		if _atom.GreaterThan(decimal.Zero) && _atom.Exponent() < 0 {

			return _atom, cast.ToInt(math.Abs(float64(_atom.Exponent())))
		}
	}

	return _defaultAtom, _defaultExp
}

func GetPaymentMinAmount() decimal.Decimal {
	var _default = decimal.NewFromFloat(defaultPaymentMinAmount)
	var _min, _ = getPaymentRangeAmount()
	if _min == "" {

		return _default
	}

	_result, err := decimal.NewFromString(_min)
	if err == nil {

		return _result
	}

	return _default
}

func GetPaymentMaxAmount() decimal.Decimal {
	var _default = decimal.NewFromFloat(defaultPaymentMaxAmount)
	var _, _max = getPaymentRangeAmount()
	if _max == "" {

		return _default
	}

	_result, err := decimal.NewFromString(_max)
	if err == nil {

		return _result
	}

	return _default
}

func getPaymentRangeAmount() (string, string) {
	var _rangeVar string
	if _rangeVar = strings.TrimSpace(help.GetEnv("PAYMENT_AMOUNT_RANGE")); _rangeVar == "" {

		return "", ""
	}

	var _payRange = strings.Split(_rangeVar, ",")
	if len(_payRange) < 2 {

		return "", ""
	}

	return _payRange[0], _payRange[1]
}

func GetExpireTime() time.Duration {
	if ret := help.GetEnv("EXPIRE_TIME"); ret != "" {
		sec, err := strconv.Atoi(ret)
		if err == nil && sec > 0 {

			return time.Duration(sec)
		}
	}

	return defaultExpireTime
}

func GetUsdtRateRaw() string {
	if data := help.GetEnv("USDT_RATE"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetUsdtRate() (string, decimal.Decimal, float64) {
	if data := help.GetEnv("USDT_RATE"); data != "" {
		data = strings.TrimSpace(data)
		// 纯数字，固定汇率
		if help.IsNumber(data) {
			if _res, err := strconv.ParseFloat(data, 64); err == nil {

				return "", decimal.Decimal{}, _res
			}
		}

		// 动态交易所汇率，有波动
		if len(data) >= 2 {
			if match, err2 := regexp.MatchString(`^[~+-]\d+(\.\d+)?$`, data); match && err2 == nil {
				_value, err3 := strconv.ParseFloat(data[1:], 64)
				if err3 == nil {

					return string(data[0]), decimal.NewFromFloat(_value), defaultUsdtRate
				}
			}
		}
	}

	// 动态交易所汇率，无波动
	return "=", decimal.Decimal{}, defaultUsdtRate
}

func GetAuthToken() string {
	if data := help.GetEnv("AUTH_TOKEN"); data != "" {

		return strings.TrimSpace(data)
	}

	return defaultAuthToken
}

func GetListen() string {
	if data := help.GetEnv("LISTEN"); data != "" {

		return strings.TrimSpace(data)
	}

	return defaultListen
}

func GetTradeConfirmed() bool {
	if data := help.GetEnv("TRADE_IS_CONFIRMED"); data != "" {
		if data == "1" || data == "true" {

			return true
		}
	}

	return false
}

func GetAppUri(host string) string {
	if data := help.GetEnv("APP_URI"); data != "" {

		return strings.TrimSpace(data)
	}

	return host
}

func GetTGBotToken() string {
	if data := help.GetEnv("TG_BOT_TOKEN"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTGBotAdminId() string {
	if data := help.GetEnv("TG_BOT_ADMIN_ID"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTgBotGroupId() string {
	if data := help.GetEnv("TG_BOT_GROUP_ID"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTgBotNotifyTarget() string {
	var groupId = GetTgBotGroupId()
	if groupId != "" {

		return groupId
	}

	return GetTGBotAdminId()
}

func GetOutputLog() string {

	return runPath + "/bepusdt.log"
}

func GetDbPath() string {

	return runPath + "/bepusdt.db"
}

func GetTemplatePath() string {

	return runPath + "/templates/*"
}

func GetStaticPath() string {

	return runPath + "/static/"
}

func GetInitWalletAddress() []string {
	if data := help.GetEnv("WALLET_ADDRESS"); data != "" {

		return strings.Split(strings.TrimSpace(data), ",")
	}

	return []string{}
}
