package task

import (
	"strings"
	"testing"
)

func TestSolanaGetBlockPayloadSupportsVersionOne(t *testing.T) {
	payload := string(solanaGetBlockPayload(456519571))
	if !strings.Contains(payload, `"maxSupportedTransactionVersion":1`) {
		t.Fatalf("payload does not support Solana v1 transactions: %s", payload)
	}
}
