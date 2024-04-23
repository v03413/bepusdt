package config

import (
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/help"
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
const TronServerApiScan = "TRON_SCAN"
const TronServerApiGrid = "TRON_GRID"

var runPath string

func init() {
	execPath, err := os.Executable()
	if err != nil {

		panic(err)
	}

	runPath = filepath.Dir(execPath)
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

func GetTronServerApi() string {
	if data := help.GetEnv("TRON_SERVER_API"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTronScanApiKey() string {
	if data := help.GetEnv("TRON_SCAN_API_KEY"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func GetTronGridApiKey() string {
	if data := help.GetEnv("TRON_GRID_API_KEY"); data != "" {

		return strings.TrimSpace(data)
	}

	return ""
}

func IsTronScanApi() bool {
	if GetTronServerApi() == TronServerApiScan {

		return true
	}

	return GetTronServerApi() != TronServerApiGrid
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
