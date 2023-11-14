package usdt

import "math"

var _latestUsdtRate = 0.0
var _okxLastUsdtRate = 0.0

func SetLatestRate(rate float64) {
	// 取绝对值

	_latestUsdtRate = math.Abs(rate)
}

func GetLatestRate() float64 {

	return _latestUsdtRate
}

func SetOkxLatestRate(okxRate float64) {

	_okxLastUsdtRate = okxRate
}

func GetOkxLastRate() float64 {

	return _okxLastUsdtRate
}
