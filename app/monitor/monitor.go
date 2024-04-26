package monitor

import (
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/config"
)

var _paymentMinAmount, _paymentMaxAmount decimal.Decimal

func init() {
	_paymentMinAmount = config.GetPaymentMinAmount()
	_paymentMaxAmount = config.GetPaymentMaxAmount()
}

func inPaymentAmountRange(payAmount decimal.Decimal) bool {
	if payAmount.GreaterThan(_paymentMaxAmount) {

		return false
	}

	if payAmount.LessThan(_paymentMinAmount) {

		return false
	}

	return true
}
