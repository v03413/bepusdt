package task

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
)

type task struct {
	duration time.Duration
	callback func(ctx context.Context)
}

var (
	tasks []task
	mu    sync.Mutex
)

func Init() error {
	bscInit()
	ethInit()
	polygonInit()
	arbitrumInit()
	xlayerInit()
	baseInit()

	return nil
}

func register(t task) {
	mu.Lock()
	defer mu.Unlock()

	if t.callback == nil {

		panic("task callback cannot be nil")
	}

	tasks = append(tasks, t)
}

func inAmountRange(payAmount decimal.Decimal) bool {
	if payAmount.GreaterThan(conf.GetPaymentAmountMax()) {

		return false
	}

	if payAmount.LessThan(conf.GetPaymentAmountMin()) {

		return false
	}

	return true
}

func Start(ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()

	for _, t := range tasks {
		go func(t task) {
			if t.duration <= 0 {
				t.callback(ctx)

				return
			}

			t.callback(ctx)

			ticker := time.NewTicker(t.duration)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					t.callback(ctx)
				}
			}
		}(t)
	}
}
