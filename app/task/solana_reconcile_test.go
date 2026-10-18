package task

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/model"
)

const (
	reconcileTestOwner     = "F7p3dFrjRTbtRp8FRF6qHLomXbKRBzpvBLjtQcfcgmNe"
	reconcileTestSignature = "DohbsAfxAerDTAX9XS1b9jXY5XYMXQsCnyY3qu42eEEcTeJ6UzMbFs3JTdY14QgNKQPdy6yezdgbqLbnRVj6YB6"
	reconcileTestSlot      = 451914288
)

type reconcileTestRPC struct {
	blockTime      int64  // 交易时间，同时用于签名列表和 getTransaction
	signatureErr   string // 签名列表里的 err 字段
	paidAccount    string // 签名列表里有这笔付款的代币账户，默认是收款钱包的 ATA
	extraAccount   string // getTokenAccountsByOwner 额外返回的代币账户；为空时该方法报错，模拟 publicnode 免费节点
	failSignatures atomic.Bool
	getTransaction atomic.Int64
}

// 按 JSON-RPC 方法返回；getTransaction 使用主网交易 success_v0_inner_transfer.json，只改 blockTime
func newReconcileTestServer(t *testing.T, rpc *reconcileTestRPC) *httptest.Server {
	t.Helper()

	ata, err := solanaAssociatedTokenAccount(reconcileTestOwner, model.GetTradeContract(model.UsdcSolana))
	if err != nil {
		t.Fatalf("derive ATA: %v", err)
	}
	if rpc.paidAccount == "" {
		rpc.paidAccount = ata
	}

	trans := gjson.GetBytes(readSolanaFixture(t, "success_v0_inner_transfer.json"), "result.transactions.0").Raw
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/notify" {
			_, _ = w.Write([]byte("ok"))

			return
		}

		body, _ := io.ReadAll(r.Body)
		switch gjson.GetBytes(body, "method").String() {
		case "getTokenAccountsByOwner":
			if rpc.extraAccount == "" {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Indexed requests require a personal token"},"id":1}`))

				return
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":{"context":{"slot":%d},"value":[{"pubkey":"%s","account":{}},{"pubkey":"%s","account":{}}]},"id":1}`, reconcileTestSlot+1000, ata, rpc.extraAccount)
		case "getSignaturesForAddress":
			if rpc.failSignatures.Load() {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32005,"message":"Node is behind"},"id":1}`))

				return
			}
			if gjson.GetBytes(body, "params.0").String() != rpc.paidAccount {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":[],"id":1}`))

				return
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":[{"signature":"%s","slot":%d,"err":%s,"memo":null,"blockTime":%d,"confirmationStatus":"finalized"}],"id":1}`, reconcileTestSignature, reconcileTestSlot, rpc.signatureErr, rpc.blockTime)
		case "getSignatureStatuses":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":{"context":{"slot":%d},"value":[{"confirmationStatus":"finalized","confirmations":null,"err":null,"slot":%d,"status":{"Ok":null}}]},"id":1}`, reconcileTestSlot+1000, reconcileTestSlot)
		case "getTransaction":
			rpc.getTransaction.Add(1)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","result":{"slot":%d,"blockTime":%d,%s,"id":1}`, reconcileTestSlot, rpc.blockTime, trans[1:])
		default:
			t.Errorf("unexpected method: %s", body)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

func createReconcileTestOrder(t *testing.T, tradeID, amount string) model.Order {
	t.Helper()

	return createTestOrder(t, model.Order{
		NotifyUrl: model.GetC(model.RpcEndpointSolana) + "/notify",
		OrderId:   "merchant-" + tradeID,
		TradeId:   tradeID,
		TradeType: model.UsdcSolana,
		Rate:      "7.00",
		Amount:    amount,
		Money:     "2736.93",
		Address:   reconcileTestOwner,
		Status:    model.OrderStatusExpired,
		ApiType:   model.OrderApiTypeEpusdt,
		ExpiredAt: time.Now().Add(-5 * time.Minute),
	}, time.Now().Add(-15*time.Minute))
}

func initReconcileTest(t *testing.T, rpc *reconcileTestRPC) *solana {
	t.Helper()

	srv := newReconcileTestServer(t, rpc)
	initTaskTestEnv(t, map[model.ConfKey]string{model.RpcEndpointSolana: srv.URL})
	solanaReconciled.Clear()
	s := newSolana()

	return &s
}

func TestSolanaReconcileFindsMissedPayment(t *testing.T) {
	rpc := &reconcileTestRPC{blockTime: time.Now().Add(-10 * time.Minute).Unix(), signatureErr: "null"}
	s := initReconcileTest(t, rpc)
	order := createReconcileTestOrder(t, "trade-order-1", "390.990618")

	s.reconcileExpiredOrders(t.Context())

	got := getTestOrder(t, order.TradeId)
	if got.Status != model.OrderStatusConfirming || got.RefHash != reconcileTestSignature || got.RefBlockNum != reconcileTestSlot ||
		got.FromAddress != "GMCJvYGf5Ex2ARiMquaBDqU6iKM8uiEQkB8jCnoNfHpC" {
		t.Fatalf("order = status %d ref %s slot %d from %s, want confirming with the payment", got.Status, got.RefHash, got.RefBlockNum, got.FromAddress)
	}
	if got.ConfirmedAt == nil || got.ConfirmedAt.Unix() != rpc.blockTime {
		t.Fatalf("confirmed_at = %v, want the transaction time", got.ConfirmedAt)
	}

	// 之后按原流程确认并回调：订单虽已过期，但付款时间在有效期内
	s.tradeConfirmHandle(t.Context())
	if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusSuccess {
		t.Fatalf("order status = %d after confirm, want success", got.Status)
	}
	waitOrderNotified(t, order.TradeId)
}

func TestSolanaReconcileSkipsUnmatchedPayments(t *testing.T) {
	inWindow := time.Now().Add(-10 * time.Minute).Unix()
	tests := []struct {
		name               string
		blockTime          int64
		signatureErr       string
		amount             string
		usedByOtherOrder   bool
		wantGetTransaction int64
	}{
		{name: "failed transaction", blockTime: inWindow, signatureErr: `{"InstructionError":[1,{"Custom":6001}]}`, amount: "390.990618"},
		{name: "paid before order created", blockTime: time.Now().Add(-30 * time.Minute).Unix(), signatureErr: "null", amount: "390.990618"},
		{name: "paid after expiry", blockTime: time.Now().Add(-2 * time.Minute).Unix(), signatureErr: "null", amount: "390.990618"},
		{name: "signature used by another order", blockTime: inWindow, signatureErr: "null", amount: "390.990618", usedByOtherOrder: true},
		{name: "amount mismatch", blockTime: inWindow, signatureErr: "null", amount: "1.23", wantGetTransaction: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpc := &reconcileTestRPC{blockTime: tt.blockTime, signatureErr: tt.signatureErr}
			s := initReconcileTest(t, rpc)
			order := createReconcileTestOrder(t, "trade-order-1", tt.amount)
			if tt.usedByOtherOrder {
				other := createReconcileTestOrder(t, "trade-order-2", "100")
				other.Status = model.OrderStatusSuccess
				other.RefHash = reconcileTestSignature
				model.Db.Save(&other)
			}

			s.reconcileExpiredOrders(t.Context())

			if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusExpired || got.RefHash != "" {
				t.Fatalf("order = status %d ref %s, want unchanged", got.Status, got.RefHash)
			}
			if got := rpc.getTransaction.Load(); got != tt.wantGetTransaction {
				t.Fatalf("getTransaction calls = %d, want %d", got, tt.wantGetTransaction)
			}
			if _, done := solanaReconciled.Load(order.ID); !done {
				t.Fatalf("order not marked as reconciled")
			}
		})
	}
}

func TestSolanaReconcileRetriesOnRPCError(t *testing.T) {
	rpc := &reconcileTestRPC{blockTime: time.Now().Add(-10 * time.Minute).Unix(), signatureErr: "null"}
	s := initReconcileTest(t, rpc)
	order := createReconcileTestOrder(t, "trade-order-1", "390.990618")

	rpc.failSignatures.Store(true)
	s.reconcileExpiredOrders(t.Context())
	if _, done := solanaReconciled.Load(order.ID); done {
		t.Fatalf("order marked as reconciled after RPC error")
	}

	rpc.failSignatures.Store(false)
	s.reconcileExpiredOrders(t.Context())
	if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusConfirming {
		t.Fatalf("order status = %d after retry, want confirming", got.Status)
	}
}

func TestSolanaReconcileOnlyRecentlyExpiredOrders(t *testing.T) {
	rpc := &reconcileTestRPC{blockTime: time.Now().Add(-10 * time.Minute).Unix(), signatureErr: "null"}
	s := initReconcileTest(t, rpc)
	order := createReconcileTestOrder(t, "trade-order-1", "390.990618")
	order.ExpiredAt = time.Now().Add(-30 * time.Second) // 还没到对账延迟，交易可能尚未 finalized
	model.Db.Save(&order)

	s.reconcileExpiredOrders(t.Context())
	if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusExpired {
		t.Fatalf("order status = %d, want expired until the reconcile delay passes", got.Status)
	}
}

func TestSolanaReconcileChecksOtherTokenAccounts(t *testing.T) {
	const other = "ReconcileTestTokenAccount111111111111111111"
	rpc := &reconcileTestRPC{blockTime: time.Now().Add(-10 * time.Minute).Unix(), signatureErr: "null", paidAccount: other, extraAccount: other}
	s := initReconcileTest(t, rpc)
	order := createReconcileTestOrder(t, "trade-order-1", "390.990618")

	s.reconcileExpiredOrders(t.Context())
	if got := getTestOrder(t, order.TradeId); got.Status != model.OrderStatusConfirming || got.RefHash != reconcileTestSignature {
		t.Fatalf("order = status %d ref %s, want the payment to the non-ATA token account", got.Status, got.RefHash)
	}
}

func TestSolanaAssociatedTokenAccount(t *testing.T) {
	tests := []struct {
		owner, mint, want string
	}{
		// 主网交易中的代币账户（success_legacy_transfer.json、success_legacy_transfer_checked.json）
		{owner: "Mdv6q9cauwiPLTn43SXuLtitAsDbNHtT3sFbknNvr6b", mint: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", want: "Hwnrd47xhiULAjwLThEnuxnVTCwJyr5phkfwDyB58ns1"},
		{owner: "8tjDqRuhYN1PZcZkwLy9sRTUb1hbJNyijJeAESxG5zxv", mint: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", want: "9un7p7bo7Y4TBwVdsQZTWtLZtkvW9hAUzdeV7mardWo5"},
	}
	for _, tt := range tests {
		if got, err := solanaAssociatedTokenAccount(tt.owner, tt.mint); err != nil || got != tt.want {
			t.Fatalf("ATA(%s, %s) = %s, %v; want %s", tt.owner, tt.mint, got, err, tt.want)
		}
	}

	if _, err := solanaAssociatedTokenAccount("not-an-address", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"); err == nil {
		t.Fatalf("invalid owner accepted")
	}
}
