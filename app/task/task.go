package task

import (
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/config"
	"time"
)

type task struct {
	Tick     time.Duration
	Callback func(tick time.Duration)
}

var scheduleList []task

var minAmount, maxAmount decimal.Decimal

func init() {
	minAmount = config.GetPaymentMinAmount()
	maxAmount = config.GetPaymentMaxAmount()
}

func Start() {
	for _, t := range scheduleList {
		go t.Callback(t.Tick)
	}
}

func RegisterSchedule(tick time.Duration, callback func(tick time.Duration)) {
	scheduleList = append(scheduleList, task{
		Tick:     tick,
		Callback: callback,
	})
}

func inPaymentAmountRange(payAmount decimal.Decimal) bool {
	if payAmount.GreaterThan(maxAmount) {

		return false
	}

	if payAmount.LessThan(minAmount) {

		return false
	}

	return true
}
