package config

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/help"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultExpireTime = 600     // 订单默认有效期 10分钟
const DefaultUsdtCnyRate = 6.4    // 默认USDT汇率
const DefaultTrxCnyRate = 0.95    // 默认TRX汇率
const defaultAuthToken = "123234" // 默认授权码
const defaultListen = ":8080"     // 默认监听地址
const defaultPaymentMinAmount = 0.01
const defaultPaymentMaxAmount = 99999
const defaultUsdtAtomicity = "0.01" // 原子精度
const defaultTrxAtomicity = "0.01"
const defaultTronGrpcNode = "18.141.79.38:50051" // 默认GRPC节点

var runPath string

var (
	BlockScanSucc  uint64
	BlockScanTotal uint64
)

func init() {
	execPath, err := os.Executable()
	if err != nil {

		panic(err)
	}

	runPath = filepath.Dir(execPath)
}

func GetBlockScanSuccRate() string {
	if BlockScanTotal == 0 {

		return "100.00%"
	}

	return fmt.Sprintf("%.2f%%", cast.ToFloat64(BlockScanSucc/BlockScanTotal)*100)
}

func GetTronGrpcNode() string {
	if data := help.GetEnv("TRON_GRPC_NODE"); data != "" {

		return strings.TrimSpace(data)
	}

	return defaultTronGrpcNode
}

func GetUsdtAtomicity() (decimal.Decimal, int) {
	var defaultAtom, _ = decimal.NewFromString(defaultUsdtAtomicity)
	var defaultExp = cast.ToInt(math.Abs(float64(defaultAtom.Exponent())))
	if data := help.GetEnv("USDT_ATOM"); data != "" {
		var atom, exp, err = parseAtomicity(data)
		if err == nil {

			return atom, exp
		}
	}

	return defaultAtom, defaultExp
}

func GetTrxAtomicity() (decimal.Decimal, int) {
	var defaultAtom, _ = decimal.NewFromString(defaultTrxAtomicity)
	var defaultExp = cast.ToInt(math.Abs(float64(defaultAtom.Exponent())))
	if data := help.GetEnv("TRX_ATOM"); data != "" {
		var atom, exp, err = parseAtomicity(data)
		if err == nil {

			return atom, exp
		}
	}

	return defaultAtom, defaultExp
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

func GetUsdtRate() string {
	if data := help.GetEnv("USDT_RATE"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTrxRate() string {
	if data := help.GetEnv("TRX_RATE"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
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

func GetStaticPath() string {
	if data := help.GetEnv("STATIC_PATH"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetInitWalletAddress() []string {
	if data := help.GetEnv("WALLET_ADDRESS"); data != "" {

		return strings.Split(strings.TrimSpace(data), ",")
	}

	return []string{}
}

func parseAtomicity(data string) (decimal.Decimal, int, error) {
	var atom, err = decimal.NewFromString(data)
	if err != nil {

		return decimal.Zero, 0, err
	}

	// 如果大于0，且小数点后位数大于0
	if atom.GreaterThan(decimal.Zero) && atom.Exponent() < 0 {

		return atom, cast.ToInt(math.Abs(float64(atom.Exponent()))), nil
	}

	return decimal.Zero, 0, errors.New("原子精度参数不合法")
}
