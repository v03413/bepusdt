package task

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"filippo.io/edwards25519"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

// 订单过期后，按收款账户的链上历史对账一次。
// 扫块每个 slot 只有一次机会，跳块、放弃重试的 slot、重启、回溯范围不完整都会让有效期内的付款漏掉；
// 收款账户的签名列表是独立的链上历史，可以把这些付款找回来。

const (
	solanaReconcileDelay     = time.Minute   // 过期后等待交易 finalized 再对账
	solanaReconcileWindow    = 2 * time.Hour // 只对最近过期的订单对账
	solanaSignaturePageLimit = 1000

	solanaAssociatedTokenProgram = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
)

// solanaReconciled 记录已完成对账的订单 ID；RPC 失败的订单不记录，下一轮重试
var solanaReconciled sync.Map // key: int64 order ID, value: struct{}

func (s *solana) reconcileExpiredOrders(ctx context.Context) {
	now := time.Now()
	var orders []model.Order
	model.Db.Where("status = ? and trade_type in (?)", model.OrderStatusExpired, model.GetNetworkTrades(conf.Solana)).
		Where("expired_at > ? and expired_at < ?", now.Add(-solanaReconcileWindow), now.Add(-solanaReconcileDelay)).
		Find(&orders)

	for _, o := range orders {
		if ctx.Err() != nil {

			return
		}
		if _, done := solanaReconciled.Load(o.ID); done {

			continue
		}
		if err := s.reconcileOrder(ctx, o); err != nil {
			log.Task.Warn(fmt.Sprintf("Solana 订单对账失败，稍后重试 %s：%v", o.TradeId, err))

			continue
		}

		solanaReconciled.Store(o.ID, struct{}{})
	}
}

func (s *solana) reconcileOrder(ctx context.Context, o model.Order) error {
	accounts, err := s.tokenAccounts(ctx, orderMatchAddress(o), model.GetTradeContract(o.TradeType))
	if err != nil {

		return err
	}

	for _, account := range accounts {
		signatures, err := s.orderSignatures(ctx, account, o)
		if err != nil {

			return err
		}

		for _, signature := range signatures {
			var used int64
			model.Db.Model(&model.Order{}).Where("ref_hash = ?", signature).Count(&used)
			if used > 0 {

				continue
			}

			trans, err := s.rpcCall(ctx, "getTransaction", signature,
				map[string]any{"encoding": "json", "maxSupportedTransactionVersion": 1, "commitment": "finalized"})
			if err != nil {

				return err
			}
			if trans.Type == gjson.Null {

				return fmt.Errorf("getTransaction %s: transaction not found", signature)
			}

			timestamp := time.Unix(trans.Get("blockTime").Int(), 0)
			for _, t := range s.parseTransaction(trans, int(trans.Get("slot").Int()), timestamp) {
				if !orderTransferMatch(o, t) {

					continue
				}
				if err := o.MarkConfirming(t.BlockNum, t.FromAddress, t.TxHash, t.Timestamp, t.Amount); err != nil {

					return err
				}

				log.Task.Info(fmt.Sprintf("Solana 对账找回付款 %s：%s", o.TradeId, t.TxHash))

				return nil
			}
		}
	}

	return nil
}

// tokenAccounts 返回钱包在该 mint 下的代币账户：总是包含关联代币账户（ATA），节点支持 getTokenAccountsByOwner 时再加上其他代币账户。
// 部分节点（如 publicnode 免费节点）不提供 getTokenAccountsByOwner，此时只检查 ATA，绝大多数付款都转入 ATA。
func (s *solana) tokenAccounts(ctx context.Context, owner, mint string) ([]string, error) {
	ata, err := solanaAssociatedTokenAccount(owner, mint)
	if err != nil {

		return nil, err
	}

	accounts := []string{ata}
	result, err := s.rpcCall(ctx, "getTokenAccountsByOwner", owner, map[string]any{"mint": mint},
		map[string]any{"encoding": "base64", "dataSlice": map[string]int{"offset": 0, "length": 0}, "commitment": "finalized"})
	if err != nil {

		return accounts, nil
	}
	for _, account := range result.Get("value.#.pubkey").Array() {
		if account.String() != ata {
			accounts = append(accounts, account.String())
		}
	}

	return accounts, nil
}

// solanaAssociatedTokenAccount 按 Associated Token Account 程序的规则推导地址：
// seeds = [钱包, Token 程序, mint]，从 bump 255 往下找第一个不在 ed25519 曲线上的哈希
func solanaAssociatedTokenAccount(owner, mint string) (string, error) {
	seeds := [][]byte{base58.Decode(owner), base58.Decode(conf.SolSplToken), base58.Decode(mint)}
	for _, seed := range seeds {
		if len(seed) != 32 {

			return "", fmt.Errorf("invalid Solana address: %s / %s", owner, mint)
		}
	}

	program := base58.Decode(solanaAssociatedTokenProgram)
	for bump := 255; bump >= 0; bump-- {
		h := sha256.New()
		for _, seed := range seeds {
			h.Write(seed)
		}
		h.Write([]byte{byte(bump)})
		h.Write(program)
		h.Write([]byte("ProgramDerivedAddress"))

		addr := h.Sum(nil)
		if _, err := new(edwards25519.Point).SetBytes(addr); err != nil {

			return base58.Encode(addr), nil
		}
	}

	return "", fmt.Errorf("no associated token account for %s / %s", owner, mint)
}

// orderSignatures 返回代币账户在订单有效期内执行成功的交易签名；签名按时间倒序分页返回
func (s *solana) orderSignatures(ctx context.Context, account string, o model.Order) ([]string, error) {
	var result []string
	opts := map[string]any{"limit": solanaSignaturePageLimit, "commitment": "finalized"}
	for {
		page, err := s.rpcCall(ctx, "getSignaturesForAddress", account, opts)
		if err != nil {

			return nil, err
		}

		items := page.Array()
		for _, itm := range items {
			if !itm.Get("blockTime").Exists() || itm.Get("blockTime").Type == gjson.Null {

				continue
			}

			at := time.Unix(itm.Get("blockTime").Int(), 0)
			if !o.CreatedAt.Before(at) { // 已早于订单创建时间

				return result, nil
			}
			if o.ExpiredAt.After(at) && solanaTxSucceeded(itm.Get("err")) {
				result = append(result, itm.Get("signature").String())
			}
		}
		if len(items) < solanaSignaturePageLimit {

			return result, nil
		}

		opts["before"] = items[len(items)-1].Get("signature").String()
	}
}

func (s *solana) rpcCall(ctx context.Context, method string, params ...any) (gjson.Result, error) {
	post, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {

		return gjson.Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", model.Endpoint(conf.Solana), bytes.NewBuffer(post))
	if err != nil {

		return gjson.Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {

		return gjson.Result{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {

		return gjson.Result{}, fmt.Errorf("%s response status code %d", method, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {

		return gjson.Result{}, err
	}
	if !gjson.ValidBytes(body) {

		return gjson.Result{}, fmt.Errorf("%s invalid JSON response", method)
	}

	data := gjson.ParseBytes(body)
	if rpcErr := data.Get("error"); rpcErr.Exists() {

		return gjson.Result{}, fmt.Errorf("%s RPC error: %s", method, rpcErr.Raw)
	}
	if !data.Get("result").Exists() {

		return gjson.Result{}, fmt.Errorf("%s missing result", method)
	}

	return data.Get("result"), nil
}
