package task

import (
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"
)

func TestSolanaGetBlockPayloadSupportsVersionOne(t *testing.T) {
	payload := string(solanaGetBlockPayload(456519571))
	if !strings.Contains(payload, `"maxSupportedTransactionVersion":1`) {
		t.Fatalf("payload does not support Solana v1 transactions: %s", payload)
	}
}

func TestValidateSolanaBlockResponse(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		wantSkipped bool
	}{
		{name: "valid empty block", body: `{"jsonrpc":"2.0","result":{"blockTime":1760380000,"transactions":[]},"id":1}`},
		{name: "null result", body: `{"jsonrpc":"2.0","result":null,"id":1}`, wantSkipped: true},
		{name: "slot skipped", body: `{"jsonrpc":"2.0","error":{"code":-32007,"message":"Slot 300000092 was skipped, or missing due to ledger jump to recent snapshot"},"id":1}`, wantSkipped: true},
		{name: "long-term storage slot skipped", body: `{"jsonrpc":"2.0","error":{"code":-32009,"message":"Slot 300000092 was skipped, or missing in long-term storage"},"id":1}`, wantSkipped: true},
		{name: "block not available", body: `{"jsonrpc":"2.0","error":{"code":-32004,"message":"Block not available for slot 456731994"},"id":1}`, wantErr: true},
		{name: "unsupported transaction version", body: `{"jsonrpc":"2.0","error":{"code":-32015,"message":"Transaction version (1) is not supported by the requesting client"},"id":1}`, wantErr: true},
		{name: "missing result", body: `{"jsonrpc":"2.0","id":1}`, wantErr: true},
		{name: "missing transactions", body: `{"jsonrpc":"2.0","result":{"blockTime":1760380000},"id":1}`, wantErr: true},
		{name: "invalid json", body: `<html>rate limited</html>`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateSolanaBlockResponse([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && (result.Type == gjson.Null) != tt.wantSkipped {
				t.Fatalf("skipped = %v, want %v", result.Type == gjson.Null, tt.wantSkipped)
			}
		})
	}
}

func TestSolanaRetryDelayIsBounded(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 2 * time.Second},
		{attempt: 2, want: 5 * time.Second},
		{attempt: 3, want: 15 * time.Second},
		{attempt: 4, want: 30 * time.Second},
		{attempt: 5, want: time.Minute},
		{attempt: 100, want: time.Minute},
	}
	for _, tt := range tests {
		if got := solanaRetryDelay(tt.attempt); got != tt.want {
			t.Fatalf("attempt %d delay = %s, want %s", tt.attempt, got, tt.want)
		}
	}
}

func TestValidSolanaLookbackRange(t *testing.T) {
	tests := []struct {
		start int64
		end   int64
		want  bool
	}{
		{start: 458225400, end: 458225500, want: true},
		{start: 0, end: 0, want: false},
		{start: 0, end: 458225500, want: false},
		{start: 458225500, end: 458225400, want: false},
	}
	for _, tt := range tests {
		if got := validSolanaLookbackRange(tt.start, tt.end); got != tt.want {
			t.Fatalf("validSolanaLookbackRange(%d, %d) = %v, want %v", tt.start, tt.end, got, tt.want)
		}
	}
}
