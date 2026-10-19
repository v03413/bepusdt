package task

import (
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/model"
)

// 预期结果由独立的解码脚本从主网交易得出：只统计两端代币账户都在 token balances 中的 USDT/USDC transfer、transferChecked
func TestSolanaParseTransactionMainnetFixtures(t *testing.T) {
	type want struct {
		from, to, amount string
		tradeType        model.TradeType
	}
	tests := []struct {
		fixture string
		want    []want
	}{
		{fixture: "success_legacy_transfer.json", want: []want{
			{from: "218mr4Lpgn5nFX4vBmgSDUSj4uvtTnx5jmwC5cZFHfSn", to: "Mdv6q9cauwiPLTn43SXuLtitAsDbNHtT3sFbknNvr6b", amount: "5.36", tradeType: model.UsdtSolana},
		}},
		{fixture: "success_legacy_transfer_checked.json", want: []want{
			{from: "8HrXbX3MdyEpgwiA1MkqR5max8gugvpX7ay4DFDd53aH", to: "8tjDqRuhYN1PZcZkwLy9sRTUb1hbJNyijJeAESxG5zxv", amount: "0.0024", tradeType: model.UsdcSolana},
		}},
		{fixture: "success_transfer_creates_token_account.json", want: []want{ // 同一笔交易里先创建收款方代币账户（非 ATA）
			{from: "Bj7j6d8sybC7rLTmGMCB4K4p6yC1v4RQvdAptDzTiURv", to: "8YKSSNbwE72ftktZS3V1NXvd8ekHzVfLk7GkBsvy8E2i", amount: "2560.021449", tradeType: model.UsdcSolana},
		}},
		{fixture: "success_v0_inner_transfer.json", want: []want{ // 地址查找表 + 内部指令
			{from: "GMCJvYGf5Ex2ARiMquaBDqU6iKM8uiEQkB8jCnoNfHpC", to: "F7p3dFrjRTbtRp8FRF6qHLomXbKRBzpvBLjtQcfcgmNe", amount: "390.990618", tradeType: model.UsdcSolana},
		}},
		{fixture: "success_v1_transfer.json", want: []want{
			{from: "2LAYGGSiF3QfyQEG1Gb3UGAsuzvD21zjHEW88v2XZ7cW", to: "FDRRDey8oQeRMgM5msEzy4yayKccuN2k47UdSnNNVVGQ", amount: "103.347262", tradeType: model.UsdcSolana},
			{from: "FDRRDey8oQeRMgM5msEzy4yayKccuN2k47UdSnNNVVGQ", to: "6EVgUURuvprazVdpd717E9uqruHcqRPbhRKDm5Czdidu", amount: "103.347262", tradeType: model.UsdcSolana},
		}},
		{fixture: "failed_v1_transfer.json"},
		{fixture: "failed_v0_inner_transfer.json"},
	}

	s := &solana{}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			block := readSolanaFixture(t, tt.fixture)
			trans := gjson.GetBytes(block, "result.transactions.0")
			slot := solanaFixtureSlot(block)
			timestamp := time.Unix(gjson.GetBytes(block, "result.blockTime").Int(), 0)

			got := s.parseTransaction(trans, slot, timestamp)
			if len(got) != len(tt.want) {
				t.Fatalf("transfers = %d, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, w := range tt.want {
				g := got[i]
				if g.FromAddress != w.from || g.RecvAddress != w.to || g.Amount.String() != w.amount || g.TradeType != w.tradeType {
					t.Fatalf("transfer %d = %s -> %s %s %s, want %s -> %s %s %s", i, g.FromAddress, g.RecvAddress, g.Amount, g.TradeType, w.from, w.to, w.amount, w.tradeType)
				}
				if g.TxHash != trans.Get("transaction.signatures.0").String() || g.Network != conf.Solana || g.BlockNum != slot || !g.Timestamp.Equal(timestamp) {
					t.Fatalf("transfer %d metadata = %s %s %d %s", i, g.TxHash, g.Network, g.BlockNum, g.Timestamp)
				}
			}
		})
	}
}
