package task

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/v03413/bepusdt/app/model"
)

func TestSolanaConfirmOffsetUsesRPCSlot(t *testing.T) {
	tests := []struct {
		name          string
		confirmations int
		want          int
	}{
		{name: "enough confirmations", confirmations: 100, want: model.OrderStatusSuccess},
		{name: "waiting for confirmations", confirmations: 25, want: model.OrderStatusConfirming},
	}

	const slot = 451914288
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/notify" {
					_, _ = w.Write([]byte("ok"))

					return
				}
				// 主网 getSignatureStatuses 对该交易的真实返回，只改 context.slot
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":{"context":{"slot":%d},"value":[{"confirmationStatus":"finalized","confirmations":null,"err":null,"slot":%d,"status":{"Ok":null}}]},"id":1}`, slot+tt.confirmations, slot)
			}))
			defer srv.Close()
			initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL, model.BlockOffsetConfirm: "1"})

			order := createTestOrder(t, model.Order{
				OrderId:     "merchant-order-1",
				TradeId:     "trade-order-1",
				TradeType:   model.UsdcSolana,
				Rate:        "7.00",
				Amount:      "390.990618",
				Money:       "2736.93",
				Address:     "F7p3dFrjRTbtRp8FRF6qHLomXbKRBzpvBLjtQcfcgmNe",
				Status:      model.OrderStatusConfirming,
				ApiType:     model.OrderApiTypeEpusdt,
				NotifyUrl:   srv.URL + "/notify",
				RefHash:     "DohbsAfxAerDTAX9XS1b9jXY5XYMXQsCnyY3qu42eEEcTeJ6UzMbFs3JTdY14QgNKQPdy6yezdgbqLbnRVj6YB6",
				RefBlockNum: slot,
				ExpiredAt:   time.Now().Add(10 * time.Minute),
			}, time.Now().Add(-time.Minute))

			s := newSolana() // lastSlotNum 为 0，相当于重启后或扫描已暂停
			s.tradeConfirmHandle(t.Context())

			if got := getTestOrder(t, order.TradeId).Status; got != tt.want {
				t.Fatalf("order status = %d, want %d", got, tt.want)
			}
			if tt.want == model.OrderStatusSuccess {
				waitOrderNotified(t, order.TradeId)
			}
		})
	}
}
