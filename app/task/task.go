package task

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
	"sync"
	"time"
)

type contextKey struct{}

type task struct {
	ctx      context.Context
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
	if t.ctx == nil {

		t.ctx = context.Background()
	}

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

func Start() {
	mu.Lock()
	defer mu.Unlock()

	for _, t := range tasks {
		go func(t task) {
			if t.duration <= 0 {
				t.callback(t.ctx)

				return
			}

			t.callback(t.ctx)

			ticker := time.NewTicker(t.duration)
			defer ticker.Stop()

			for {
				select {
				case <-t.ctx.Done():
					return
				case <-ticker.C:
					t.callback(t.ctx)
				}
			}
		}(t)
	}
}
