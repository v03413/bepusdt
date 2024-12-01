package rate

import (
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"regexp"
)

var okxTrxCnyCalcRate = 0.0
var okxUsdtCnyCalcRate = 0.0
var okxUsdtCnyRawRate = 0.0 // okx 交易所 usdt 兑 cny原始汇率

func GetTrxCnyCalcRate(defaultRate float64) float64 {
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

func SetOkxTrxUsdtRawRate(syntax string, rawRate float64) {

	okxTrxCnyCalcRate = parseFloatRate(syntax, rawRate*okxUsdtCnyRawRate)
}

func SetOkxUsdtCnyRawRate(syntax string, rawRate float64) {
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

	switch act {
	case "~":
		return raw.Mul(base).InexactFloat64()
	case "+":
		return raw.Add(base).InexactFloat64()
	case "-":
		return raw.Sub(base).InexactFloat64()
	}

	return 0
}
