package task

import (
	"errors"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/rate"
	"io"
	"net/http"
	"strconv"
	"time"
)

func init() {
	RegisterSchedule(0, OkxUsdtRateStart)
	RegisterSchedule(0, OkxTrxUsdtRateStart)
}

// OkxUsdtRateStart Okx USDT_CNY 汇率监控
func OkxUsdtRateStart(time.Duration) {
	for {
		var rawRate, err = getOkxUsdtCnySellPrice()
		if err != nil {
			log.Error("Okx USDT_CNY 汇率获取失败", err)
		} else {
			rate.SetOkxUsdtCnyRate(conf.GetUsdtRate(), rawRate)
		}

		log.Info("当前 USDT_CNY 计算汇率：", rate.GetUsdtCalcRate(cast.ToFloat64(conf.DefaultUsdtCnyRate)))
		time.Sleep(time.Minute)
	}
}

// OkxTrxUsdtRateStart  Okx TRX_CNY 汇率监控
func OkxTrxUsdtRateStart(time.Duration) {
	for {
		var price, err = getOkxTrxCnyMarketPrice()
		if err != nil {
			log.Error("Okx TRX_USDT 汇率获取失败", err)
		} else {
			rate.SetOkxTrxCnyRate(conf.GetTrxRate(), price)
		}

		log.Info("当前 TRX_CNY 计算汇率：", rate.GetTrxCalcRate(cast.ToFloat64(conf.DefaultTrxCnyRate)))
		time.Sleep(time.Minute)
	}
}

// getOkxUsdtCnySellPrice  Okx  C2C快捷交易 USDT出售 实时汇率
func getOkxUsdtCnySellPrice() (float64, error) {
	var t = strconv.Itoa(int(time.Now().Unix()))
	var okxApi = "https://www.okx.com/v4/c2c/express/price?crypto=USDT&fiat=CNY&side=sell&t=" + t
	client := http.Client{Timeout: time.Second * 5}
	req, _ := http.NewRequest("GET", okxApi, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {

		return 0, errors.New("okx resp error:" + err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		return 0, errors.New("okx resp status code:" + strconv.Itoa(resp.StatusCode))
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {

		return 0, errors.New("okx resp read error:" + err.Error())
	}

	result := gjson.ParseBytes(all)
	if result.Get("error_code").Int() != 0 {

		return 0, errors.New("json parse error:" + result.Get("error_message").String())
	}

	if result.Get("data.price").Exists() {
		var _ret = result.Get("data.price").Float()
		if _ret <= 0 {
			return 0, errors.New("okx resp json data.price <= 0")
		}

		return cast.ToFloat64(_ret), nil
	}

	return 0, errors.New("okx resp json data.price not found")
}

// getOkxTrxCnyMarketPrice 获取 Trx/Cny 市场价格 https://www.okx.com/zh-hans/convert/trx-to-cny
func getOkxTrxCnyMarketPrice() (float64, error) {
	var t = strconv.Itoa(int(time.Now().Unix()))
	var okxApi = "https://www.okx.com/priapi/v3/growth/convert/currency-pair-market-movement?baseCurrency=TRX&quoteCurrency=CNY&bar=4H&limit=1&t=" + t
	client := http.Client{Timeout: time.Second * 5}
	req, _ := http.NewRequest("GET", okxApi, nil)
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("app-type", "web")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://www.okx.com/zh-hans/convert/trx-to-cny")
	req.Header.Set("sec-ch-ua", "\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"macOS\"")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("x-locale", "zh_CN")
	req.Header.Set("x-utc", "8")
	req.Header.Set("x-zkdex-env", "0")

	resp, err := client.Do(req)
	if err != nil {

		return 0, errors.New("okx resp error:" + err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		return 0, errors.New("okx resp status code:" + strconv.Itoa(resp.StatusCode))
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {

		return 0, errors.New("okx resp read error:" + err.Error())
	}

	result := gjson.ParseBytes(all)
	if result.Get("error_code").Int() != 0 {

		return 0, errors.New("json parse error:" + result.Get("error_message").String())
	}

	var list = result.Get("data.datapointList").Array()
	if len(list) == 0 {

		return 0, errors.New("okx resp json data.datapointList not found")
	}

	return list[0].Get("price").Float(), nil
}
