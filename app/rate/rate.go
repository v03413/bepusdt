package rate

import (
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"math"
	"regexp"
)

var okxTrxCnyCalcRate = 0.0
var okxUsdtCnyCalcRate = 0.0
var okxUsdtCnyRawRate = conf.DefaultUsdtCnyRate // okx 交易所 usdt 兑 cny原始汇率
var okxTrxCnyRawRate = conf.DefaultTrxCnyRate   // okx 交易所 trx/cny 市场价
var okxRatePrecision = 2                        // 汇率保留位数，强迫症，另一方面两位小数足以覆盖大部分CNY使用场景

func GetTrxCalcRate(defaultRate float64) float64 {
	if okxTrxCnyCalcRate > 0 {

		return okxTrxCnyCalcRate
	}

	return defaultRate
}

func GetUsdtCalcRate(defaultRate float64) float64 {
	if okxUsdtCnyCalcRate > 0 {

		return okxUsdtCnyCalcRate
	}

	return defaultRate
}

func GetOkxUsdtRawRate() float64 {

	return okxUsdtCnyRawRate
}

func GetOkxTrxRawRate() float64 {

	return okxTrxCnyRawRate
}

func SetOkxTrxCnyRate(syntax string, rawRate float64) {
	rawRate = round(rawRate, okxRatePrecision)
	okxTrxCnyRawRate = rawRate
	okxTrxCnyCalcRate = parseFloatRate(syntax, rawRate)
}

func SetOkxUsdtCnyRate(syntax string, rawRate float64) {
	rawRate = round(rawRate, okxRatePrecision)
	okxUsdtCnyRawRate = rawRate
	okxUsdtCnyCalcRate = parseFloatRate(syntax, rawRate)
}

func parseFloatRate(syntax string, rawVal float64) float64 {
	if syntax == "" {

		return rawVal
	}

	if help.IsNumber(syntax) {

		return cast.ToFloat64(syntax)
	}

	match, err := regexp.MatchString(`^[~+-]\d+(\.\d+)?$`, syntax)
	if !match || err != nil {
		log.Error("浮动语法解析错误", err)

		return 0
	}

	var act = syntax[0:1]
	var raw = decimal.NewFromFloat(rawVal)
	var base = decimal.NewFromFloat(cast.ToFloat64(syntax[1:]))
	var result float64 = 0

	switch act {
	case "~":
		result = raw.Mul(base).InexactFloat64()
	case "+":
		result = raw.Add(base).InexactFloat64()
	case "-":
		result = raw.Sub(base).InexactFloat64()
	}

	return round(result, okxRatePrecision)
}

func round(val float64, precision int) float64 {
	// Round 四舍五入，ROUND_HALF_UP 模式实现
	// 返回将 val 根据指定精度 precision（十进制小数点后数字的数目）进行四舍五入的结果。precision 也可以是负数或零。

	if precision == 0 {
		return math.Round(val)
	}

	p := math.Pow10(precision)
	if precision < 0 {
		return math.Floor(val*p+0.5) * math.Pow10(-precision)
	}

	return math.Floor(val*p+0.5) / p
}
