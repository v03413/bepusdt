package task

import (
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

type task struct {
	Tick     time.Duration
	Callback func(tick time.Duration)
}

var scheduleList []task

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
	if payAmount.GreaterThan(conf.GetPaymentAmountMax()) {

		return false
	}

	if payAmount.LessThan(conf.GetPaymentAmountMin()) {

		return false
	}

	return true
}
