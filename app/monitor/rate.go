package monitor

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/usdt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// OkxUsdtRateStart Okx USDT 汇率监控，避免频繁请求
func OkxUsdtRateStart() {
	var _act, _value, _defaultRate = config.GetUsdtRate()
	for {
		if _act == "" {
			usdt.SetLatestRate(_defaultRate)

			log.Info("固定汇率", usdt.GetLatestRate())
		} else {
			_okxRate, _okxErr := getOkxUsdtCnySellPrice()
			if _okxErr == nil { // 获取成功
				usdt.SetOkxLatestRate(_okxRate.InexactFloat64())

				switch _act {
				case "~":
					usdt.SetLatestRate(_okxRate.Mul(_value).InexactFloat64())
				case "+":
					usdt.SetLatestRate(_okxRate.Add(_value).InexactFloat64())
				case "-":
					usdt.SetLatestRate(_okxRate.Sub(_value).InexactFloat64())
				default:
					usdt.SetLatestRate(_okxRate.InexactFloat64())
				}

				log.Info(fmt.Sprintf("okx rate: %v act(%v) value(%v) 最终实际汇率：%v", _okxRate, _act, _value, usdt.GetLatestRate()))
			}
		}

		time.Sleep(time.Minute)
	}
}

// getOkxUsdtCnySellPrice  Okx  C2C快捷交易 USDT出售 实时汇率
func getOkxUsdtCnySellPrice() (decimal.Decimal, error) {
	var _zero = decimal.NewFromInt(0)
	var t = strconv.Itoa(int(time.Now().Unix()))
	var okxApi = "https://www.okx.com/v4/c2c/express/price?crypto=USDT&fiat=CNY&side=sell&t=" + t
	client := http.Client{Timeout: time.Second * 5}
	req, _ := http.NewRequest("GET", okxApi, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {

		return _zero, errors.New("okx resp error:" + err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		return _zero, errors.New("okx resp status code:" + strconv.Itoa(resp.StatusCode))
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {

		return _zero, errors.New("okx resp read error:" + err.Error())
	}

	result := gjson.ParseBytes(all)
	if result.Get("error_code").Int() != 0 {

		return _zero, errors.New("json parse error:" + result.Get("error_message").String())
	}

	if result.Get("data.price").Exists() {
		var _ret = result.Get("data.price").Float()
		if _ret <= 0 {
			return _zero, errors.New("okx resp json data.price <= 0")
		}

		return decimal.NewFromFloat(_ret), nil
	}

	return _zero, errors.New("okx resp json data.price not found")
}
