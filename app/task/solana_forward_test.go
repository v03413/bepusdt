package task

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/v03413/bepusdt/app/model"
)

func newSolanaSlotTestServer(t *testing.T, head *atomic.Int64) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":%d,"id":1}`, head.Load())
	}))
	t.Cleanup(srv.Close)

	return srv
}

func createSolanaWaitingOrder(t *testing.T) {
	t.Helper()

	createTestOrder(t, model.Order{
		OrderId:   "merchant-order-1",
		TradeId:   "trade-order-1",
		TradeType: model.UsdcSolana,
		Rate:      "7.00",
		Amount:    "1.23",
		Money:     "8.61",
		Address:   "GpsvqTPP7triXkqH8mVtXEM6at9DJ41UHy5dYPTFe16H",
		Status:    model.OrderStatusWaiting,
		ApiType:   model.OrderApiTypeEpusdt,
		ExpiredAt: time.Now().Add(10 * time.Minute),
	}, time.Now().Add(-time.Minute))
}

// 落后阈值设为 10，用 20 个 slot 的落后量验证，队列不会超过 chanx 的初始容量
const solanaTestMaxDiff = 10

func TestSolanaForwardCatchesUpWhileOrdersActive(t *testing.T) {
	var head atomic.Int64
	srv := newSolanaSlotTestServer(t, &head)
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL, model.BlockHeightMaxDiff: fmt.Sprint(solanaTestMaxDiff)})
	createSolanaWaitingOrder(t)

	const last = 458000000
	s := newSolana()
	s.lastSlotNum = last
	head.Store(last + 2*solanaTestMaxDiff)
	s.syncSlotForward(t.Context())

	if s.lastSlotNum != last+2*solanaTestMaxDiff {
		t.Fatalf("lastSlotNum = %d, want %d", s.lastSlotNum, last+2*solanaTestMaxDiff)
	}
	select {
	case got := <-s.slotQueue.Out:
		if got != last+1 {
			t.Fatalf("first queued slot = %d, want %d: slots were skipped", got, last+1)
		}
	case <-time.After(time.Second):
		t.Fatalf("no slot queued: slots %d-%d were skipped", last+1, last+2*solanaTestMaxDiff)
	}
}

func TestSolanaForwardSkipsAfterIdle(t *testing.T) {
	var head atomic.Int64
	srv := newSolanaSlotTestServer(t, &head)
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL, model.BlockHeightMaxDiff: fmt.Sprint(solanaTestMaxDiff)})

	const last = 458000000
	s := newSolana()
	s.lastSlotNum = last
	head.Store(last + 2*solanaTestMaxDiff)
	s.syncSlotForward(t.Context()) // 没有订单，暂停扫描
	if s.lastSlotNum != last {
		t.Fatalf("lastSlotNum = %d while idle, want %d", s.lastSlotNum, last)
	}

	createSolanaWaitingOrder(t)
	s.syncSlotForward(t.Context())
	if s.lastSlotNum != last+2*solanaTestMaxDiff || s.slotQueue.Len() != 0 {
		t.Fatalf("lastSlotNum = %d, queued = %d; want a jump to %d after idle", s.lastSlotNum, s.slotQueue.Len(), last+2*solanaTestMaxDiff)
	}
}

func TestSolanaForwardSkipsOnStartup(t *testing.T) {
	var head atomic.Int64
	srv := newSolanaSlotTestServer(t, &head)
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL, model.BlockHeightMaxDiff: fmt.Sprint(solanaTestMaxDiff)})
	createSolanaWaitingOrder(t)

	s := newSolana()
	head.Store(458000000)
	s.syncSlotForward(t.Context())
	if s.lastSlotNum != 458000000 || s.slotQueue.Len() != 0 {
		t.Fatalf("lastSlotNum = %d, queued = %d; want a jump to the latest slot on startup", s.lastSlotNum, s.slotQueue.Len())
	}
}
