package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/model"
)

func TestPendingLookbackUnixCoversEveryOrderWindow(t *testing.T) {
	type window struct {
		created, expired time.Duration // 相对当前时间
		status           int
	}
	tests := []struct {
		name    string
		orders  []window
		wantEnd time.Duration
	}{
		{
			name:    "expired and waiting",
			orders:  []window{{-40 * time.Minute, -30 * time.Minute, model.OrderStatusExpired}, {-5 * time.Minute, 5 * time.Minute, model.OrderStatusWaiting}},
			wantEnd: 0,
		},
		{
			name:    "earlier order expires later",
			orders:  []window{{-60 * time.Minute, -10 * time.Minute, model.OrderStatusExpired}, {-40 * time.Minute, -30 * time.Minute, model.OrderStatusExpired}},
			wantEnd: -10 * time.Minute,
		},
		{
			name:    "all waiting",
			orders:  []window{{-5 * time.Minute, 5 * time.Minute, model.OrderStatusWaiting}},
			wantEnd: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initTaskTestEnv(t, map[model.ConfKey]string{model.PaymentLookbackHour: "3"})
			lookbackDone.Clear()

			now := time.Now()
			for i, w := range tt.orders {
				createTestOrder(t, model.Order{
					OrderId:   fmt.Sprintf("merchant-order-%d", i),
					TradeId:   fmt.Sprintf("trade-order-%d", i),
					TradeType: model.UsdcSolana,
					Rate:      "7.00",
					Amount:    fmt.Sprintf("1.%02d", i),
					Money:     "7.00",
					Address:   "GpsvqTPP7triXkqH8mVtXEM6at9DJ41UHy5dYPTFe16H",
					Status:    w.status,
					ApiType:   model.OrderApiTypeEpusdt,
					ExpiredAt: now.Add(w.expired),
				}, now.Add(w.created))
			}

			startAt, endAt, orderIDs, ok := pendingLookbackUnix(conf.Solana)
			if !ok || len(orderIDs) != len(tt.orders) {
				t.Fatalf("ok = %v, orders = %d, want %d", ok, len(orderIDs), len(tt.orders))
			}
			if want := now.Add(tt.orders[0].created).Unix(); startAt != want {
				t.Fatalf("startAt = %d, want %d", startAt, want)
			}
			if want := now.Add(tt.wantEnd).Unix(); endAt < want-1 || endAt > want+1 {
				t.Fatalf("endAt is %s before now, want %s", now.Sub(time.Unix(endAt, 0)).Round(time.Second), -tt.wantEnd)
			}
		})
	}
}
