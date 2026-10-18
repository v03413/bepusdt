package task

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/v03413/bepusdt/app/model"
)

// 模拟重试定时器到期并再次扫描失败：取消定时器，下一次 scheduleSlotRetry 计为新的失败
func (s *solana) expireSlotRetryTimer(slot int) {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()

	if timer, ok := s.retryScheduled[slot]; ok {
		timer.Stop()
		delete(s.retryScheduled, slot)
	}
}

func (s *solana) stopSlotRetryTimers() {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()

	for slot, timer := range s.retryScheduled {
		timer.Stop()
		delete(s.retryScheduled, slot)
	}
}

func TestSolanaSlotRetryGivesUpAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32001,"message":"Block 457000000 cleaned up, does not exist on node. First available block: 457100000"},"id":1}`))
	}))
	defer srv.Close()
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL})

	const slot = 457000000
	s := newSolana()
	t.Cleanup(s.stopSlotRetryTimers)
	for i := 1; i <= solanaSlotMaxRetries; i++ {
		s.slotParse(slot)
		if got := s.pendingSlotRetryCount(); got != 1 {
			t.Fatalf("after %d failures pending retries = %d, want 1", i, got)
		}
		s.expireSlotRetryTimer(slot)
	}

	s.slotParse(slot)
	if got := s.pendingSlotRetryCount(); got != 0 {
		t.Fatalf("pending retries = %d after giving up, forward sync stays paused", got)
	}
	s.retryMu.Lock()
	_, scheduled := s.retryScheduled[slot]
	s.retryMu.Unlock()
	if scheduled {
		t.Fatalf("slot still scheduled after giving up")
	}
}
