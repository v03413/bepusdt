package config

import (
	"errors"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/help"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultExpireTime = 600     // 订单默认有效期 10分钟
const defaultUsdtRate = 6.4       // 默认汇率
const defaultAuthToken = "123234" // 默认授权码
const defaultListen = ":8080"     // 默认监听地址

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

func GetUsdtRate() (float64, error) {
	// 固定汇率
	if data := help.GetEnv("USDT_RATE"); data != "" {
		rate, err := strconv.ParseFloat(data, 64)
		if err == nil && rate > 0 {

			return rate, nil
		}
	}

	// Okx C2C快捷交易 实时汇率
	var t = strconv.Itoa(int(time.Now().Unix()))
	var okxApi = "https://www.okx.com/v4/c2c/express/price?crypto=USDT&fiat=CNY&side=sell&t=" + t
	client := http.Client{Timeout: time.Second}
	req, _ := http.NewRequest("GET", okxApi, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {

		return defaultUsdtRate, errors.New("okx resp error:" + err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		return defaultUsdtRate, errors.New("okx resp status code:" + strconv.Itoa(resp.StatusCode))
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {

		return defaultUsdtRate, errors.New("okx resp read error:" + err.Error())
	}

	result := gjson.ParseBytes(all)
	if result.Get("error_code").Int() != 0 {

		return defaultUsdtRate, errors.New("json parse error:" + result.Get("error_message").String())
	}

	if result.Get("data.price").Exists() {

		return result.Get("data.price").Float(), nil
	}

	// 默认汇率
	return defaultUsdtRate, errors.New("okx resp json data.price not found")
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

func GetOutputLog() string {

	return runPath + "/bepusdt.log"
}

func GetDbPath() string {

	return runPath + "/bepusdt.db"
}

func GetTradeConfirmed() bool {
	if data := help.GetEnv("TRADE_IS_CONFIRMED"); data != "" {

		return true
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
