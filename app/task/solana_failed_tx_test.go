package task

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/model"
)

func scanSolanaTestBlock(t *testing.T, block []byte) []transfer {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(block)
	}))
	defer srv.Close()
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL})

	s := newSolana()
	s.slotParse(solanaFixtureSlot(block))

	return drainTransferQueue()
}

func TestSolanaTxSucceeded(t *testing.T) {
	tests := []struct {
		name string
		tx   string
		want bool
	}{
		{name: "succeeded", tx: `{"meta":{"err":null}}`, want: true},
		{name: "instruction error", tx: `{"meta":{"err":{"InstructionError":[1,{"Custom":6001}]}}}`},
		{name: "missing err", tx: `{"meta":{}}`},
		{name: "missing meta", tx: `{"meta":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := solanaTxSucceeded(gjson.Get(tt.tx, "meta.err")); got != tt.want {
				t.Fatalf("solanaTxSucceeded = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSolanaSlotParseSkipsFailedTransactions(t *testing.T) {
	tests := []struct {
		fixture string
		// 把 meta.err 改为 null 后应解析出的转账数，证明没有转账只是因为交易失败
		transfersIfSucceeded int
	}{
		{fixture: "failed_v1_transfer.json", transfersIfSucceeded: 3},
		{fixture: "failed_v0_inner_transfer.json", transfersIfSucceeded: 1},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			block := readSolanaFixture(t, tt.fixture)
			if got := scanSolanaTestBlock(t, block); len(got) != 0 {
				t.Fatalf("failed transaction produced %d transfers: %+v", len(got), got)
			}

			errRaw := gjson.GetBytes(block, "result.transactions.0.meta.err").Raw
			succeeded := bytes.Replace(block, []byte(`"err":`+errRaw), []byte(`"err":null`), 1)
			if bytes.Equal(succeeded, block) {
				t.Fatalf("fixture has no meta.err to clear")
			}
			if got := scanSolanaTestBlock(t, succeeded); len(got) != tt.transfersIfSucceeded {
				t.Fatalf("transfers = %d, want %d", len(got), tt.transfersIfSucceeded)
			}
		})
	}
}

func TestSolanaSlotParseKeepsSucceededTransactions(t *testing.T) {
	got := scanSolanaTestBlock(t, readSolanaFixture(t, "success_v0_inner_transfer.json"))
	if len(got) != 1 {
		t.Fatalf("transfers = %d, want 1", len(got))
	}

	tr := got[0]
	if tr.TxHash != "DohbsAfxAerDTAX9XS1b9jXY5XYMXQsCnyY3qu42eEEcTeJ6UzMbFs3JTdY14QgNKQPdy6yezdgbqLbnRVj6YB6" ||
		tr.TradeType != model.UsdcSolana ||
		tr.RecvAddress != "F7p3dFrjRTbtRp8FRF6qHLomXbKRBzpvBLjtQcfcgmNe" ||
		tr.Amount.String() != "390.990618" {
		t.Fatalf("unexpected transfer: %+v", tr)
	}
}

// 主网 getSignatureStatuses 的真实返回
const (
	solanaFailedSignatureStatus    = `{"confirmationStatus":"finalized","confirmations":null,"err":{"InstructionError":[1,{"Custom":6001}]},"slot":458281527,"status":{"Err":{"InstructionError":[1,{"Custom":6001}]}}}`
	solanaSucceededSignatureStatus = `{"confirmationStatus":"finalized","confirmations":null,"err":null,"slot":451914288,"status":{"Ok":null}}`
)

func TestSolanaSignatureFinalized(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "finalized", status: solanaSucceededSignatureStatus, want: true},
		{name: "finalized but failed", status: solanaFailedSignatureStatus},
		{name: "confirmed", status: `{"confirmationStatus":"confirmed","confirmations":12,"err":null,"slot":451914288,"status":{"Ok":null}}`},
		{name: "not found", status: `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := solanaSignatureFinalized(gjson.Parse(tt.status)); got != tt.want {
				t.Fatalf("solanaSignatureFinalized = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSolanaTradeConfirmIgnoresFailedTransaction(t *testing.T) {
	var notified atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/notify" {
			notified.Store(true)
			_, _ = w.Write([]byte("ok"))

			return
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":{"context":{"slot":458281967},"value":[%s]},"id":1}`, solanaFailedSignatureStatus)
	}))
	defer srv.Close()
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL})

	order := createTestOrder(t, model.Order{
		OrderId:     "merchant-order-1",
		TradeId:     "trade-order-1",
		TradeType:   model.UsdcSolana,
		Rate:        "7.00",
		Amount:      "225.966818",
		Money:       "1581.77",
		Address:     "GpsvqTPP7triXkqH8mVtXEM6at9DJ41UHy5dYPTFe16H",
		Status:      model.OrderStatusConfirming,
		ApiType:     model.OrderApiTypeEpusdt,
		NotifyUrl:   srv.URL + "/notify",
		RefHash:     "53bZ2UeqMGryYjbjKgB2nAo8umAMsPTP4dzBhR94SdV7f4YaxDvWqEdL5WQ46JSHsmLph3hBmEdHzTMdnYKMsbRz",
		RefBlockNum: 458281527,
		ExpiredAt:   time.Now().Add(10 * time.Minute),
	}, time.Now().Add(-time.Minute))

	s := newSolana()
	s.tradeConfirmHandle(t.Context())

	if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusConfirming {
		t.Fatalf("order status = %d after failed transaction, want confirming", got.Status)
	}
	time.Sleep(200 * time.Millisecond)
	if notified.Load() {
		t.Fatalf("callback sent for failed transaction")
	}
}
