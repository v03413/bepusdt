package task

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notifier"
	"github.com/v03413/bepusdt/app/task/notify"
	"github.com/v03413/tronprotocol/core"
)

type transfer struct {
	Network     string          `json:"network"`
	TxHash      string          `json:"tx_hash"`
	Amount      decimal.Decimal `json:"amount"`
	FromAddress string          `json:"from_address"`
	RecvAddress string          `json:"recv_address"`
	Timestamp   time.Time       `json:"timestamp"`
	TradeType   model.TradeType `json:"trade_type"`
	BlockNum    int             `json:"block_num"`
}

type resource struct {
	ID           string
	Type         core.Transaction_Contract_ContractType
	Balance      int64
	FromAddress  string
	RecvAddress  string
	Timestamp    time.Time
	ResourceCode core.ResourceCode
}

var resourceQueue = chanx.NewUnboundedChan[[]resource](context.Background(), 30) // 资源队列
var notOrderQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 非订单队列
var transferQueue = chanx.NewUnboundedChan[[]transfer](context.Background(), 30) // 交易转账队列

// lookbackDone 记录已触发过回溯的订单 ID，每个订单只回溯一次。
var lookbackDone sync.Map // key: int64 order ID, value: struct{}

const batchInterval = time.Second * 1       // 批处理缓解数据库读取压力
const orderCheckInterval = time.Second * 10 // 订单过期检查间隔

func init() {
	Register(Task{Callback: orderTransferHandle})
	Register(Task{Callback: notOrderTransferHandle})
	Register(Task{Callback: tronResourceHandle})
}

func orderTransferHandle(ctx context.Context) {
	var batch = make([]transfer, 0, 1000)
	var lastCheckTime = time.Now()
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case transfers, ok := <-transferQueue.Out:
			if !ok {
				return
			}
			batch = append(batch, transfers...)
		case <-ticker.C:
			// 每10秒强制检查一次过期订单，即使没有交易，防止无交易时订单不过期
			var shouldCheck = time.Since(lastCheckTime) >= orderCheckInterval
			if shouldCheck {
				lastCheckTime = time.Now()
			}

			if len(batch) == 0 {
				if shouldCheck {
					expireWaitingOrders()
				}

				continue
			}

			var other = make([]transfer, 0)
			var orders = getReceivableOrders()

			for _, t := range batch {
				// 判断数额是否在允许范围内
				if !model.IsAmountValid(t.TradeType, t.Amount) {
					continue
				}

				mqttPublish(t)

				key := fmt.Sprintf("%s%s", t.RecvAddress, t.TradeType)
				orderList, ok := orders[key]
				if !ok {
					other = append(other, t)
					continue
				}

				var matched bool
				for i, o := range orderList {
					if !orderTransferMatch(o, t) {
						continue
					}

					// 订单匹配 进入确认流程
					if err := o.MarkConfirming(t.BlockNum, t.FromAddress, t.TxHash, t.Timestamp, t.Amount); err != nil {
						log.Task.Warn("mark order confirming failed:", err)
						continue
					}

					// 从内存 map 中移除已匹配订单，防止同批次其他 transfer 重复匹配
					orders[key] = append(orderList[:i], orderList[i+1:]...)
					matched = true
					break
				}

				if !matched {
					other = append(other, t)
				}
			}

			if len(other) > 0 {
				notOrderQueue.In <- other
			}

			batch = batch[:0]

			if shouldCheck {
				expireWaitingOrders()
			}
		}
	}
}

func orderTransferMatch(o model.Order, t transfer) bool {
	if o.TradeType != t.TradeType || orderMatchAddress(o) != t.RecvAddress {
		return false
	}
	if !o.AddressLocked && !amountMatch(t.Amount, o.Amount, string(o.TradeType)) {
		return false
	}
	if !o.CreatedAt.Before(t.Timestamp) || !o.ExpiredAt.After(t.Timestamp) {
		return false
	}

	return true
}

func orderMatchAddress(o model.Order) string {
	if o.MatchAddress != "" {
		return o.MatchAddress
	}

	return o.Address
}

func notOrderTransferHandle(ctx context.Context) {
	var batch = make([]transfer, 0, 1000)
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case transfers, ok := <-notOrderQueue.Out:
			if !ok {
				return
			}
			batch = append(batch, transfers...)
		case <-ticker.C:
			if len(batch) == 0 {
				continue
			}

			var was = make([]model.Wallet, 0)
			model.Db.Where("other_notify = ?", model.WaOtherEnable).Find(&was)
			for _, wa := range was {
				for _, t := range batch {
					if t.RecvAddress != wa.MatchAddr && t.FromAddress != wa.MatchAddr {
						continue
					}

					if !model.IsNeedNotifyByTxid(t.TxHash) {
						continue
					}

					var record = model.NotifyRecord{Txid: t.TxHash}
					model.Db.Create(&record)

					notifier.NonOrderTransfer(model.TronTransfer(t), wa)
				}
			}

			batch = batch[:0]
		}
	}
}

func tronResourceHandle(ctx context.Context) {
	var batch = make([]resource, 0, 1000)
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case resources, ok := <-resourceQueue.Out:
			if !ok {
				return
			}
			batch = append(batch, resources...)
		case <-ticker.C:
			if len(batch) == 0 {
				continue
			}

			var was []model.Wallet
			model.Db.Where("status = ? and other_notify = ?", model.WaStatusEnable, model.WaOtherEnable).Find(&was)

			for _, wa := range was {
				if wa.GetNetwork() != conf.Tron {
					// 只有 Tron 网络目前才有资源变更通知
					continue
				}

				for _, r := range batch {
					if r.RecvAddress != wa.Address && r.FromAddress != wa.Address {
						continue
					}
					if r.ResourceCode != core.ResourceCode_ENERGY {
						continue
					}
					if !model.IsNeedNotifyByTxid(r.ID) {
						continue
					}

					var record = model.NotifyRecord{Txid: r.ID}
					model.Db.Create(&record)

					notifier.TronResourceChange(model.TronResource(r))
				}
			}

			batch = batch[:0]
		}
	}
}

func markFinalConfirmed(o model.Order) {
	o.SetSuccess()
	notifyOrderSuccess(o)
}

func receivableOrderStatuses() []int {
	return []int{model.OrderStatusWaiting, model.OrderStatusExpired, model.OrderStatusConfirming}
}

func getReceivableOrders() map[string][]model.Order {
	var orders []model.Order
	db := model.Db.Where("status in (?)", receivableOrderStatuses()).
		Where("expired_at > ?", time.Now().Add(model.GetLookbackHour())).
		Order("created_at asc")
	db.Find(&orders)

	data := make(map[string][]model.Order)
	for _, t := range orders {
		key := orderMatchAddress(t) + string(t.TradeType)
		data[key] = append(data[key], t)
	}

	return data
}

func hasLookbackOrders(tradeType []model.TradeType) bool {
	var count int64
	db := model.Db.Model(&model.Order{}).
		Where("status in (?)", receivableOrderStatuses()).
		Where("expired_at > ?", time.Now().Add(model.GetLookbackHour()))
	if len(tradeType) > 0 {
		db = db.Where("trade_type in (?)", tradeType)
	}

	db.Count(&count)

	return count > 0
}

func getLookbackUnix(network model.Network) (startAt, endAt int64, ok bool) {
	startAt, endAt, orderIDs, ok := pendingLookbackUnix(network)
	if ok {
		markLookbackDone(orderIDs)
	}

	return startAt, endAt, ok
}

func pendingLookbackUnix(network model.Network) (startAt, endAt int64, orderIDs []int64, ok bool) {
	trade := model.GetNetworkTrades(network)
	if len(trade) == 0 {
		return
	}

	lookback := time.Now().Add(model.GetLookbackHour())
	var all []model.Order
	model.Db.Model(&model.Order{}).
		Where("status in (?) and trade_type in (?)", receivableOrderStatuses(), trade).
		Where("expired_at > ?", lookback).
		Order("created_at asc").
		Find(&all)

	// 过滤掉已经回溯过的订单
	pending := make([]model.Order, 0, len(all))
	for _, o := range all {
		if _, done := lookbackDone.Load(o.ID); !done {
			pending = append(pending, o)
		}
	}
	if len(pending) == 0 {
		return
	}
	orderIDs = make([]int64, 0, len(pending))
	for _, o := range pending {
		orderIDs = append(orderIDs, o.ID)
	}

	// 起点：最早的创建时间（已按 created_at asc 排序）
	startAt = pending[0].CreatedAt.Time().Unix()

	// 终点：覆盖每个订单的有效期，取各订单 min(过期时间, 当前时间) 的最大值
	now := time.Now()
	endAt = startAt
	for _, o := range pending {
		end := o.ExpiredAt
		if end.After(now) {
			end = now
		}
		if end.Unix() > endAt {
			endAt = end.Unix()
		}
	}

	ok = true

	return
}

func markLookbackDone(orderIDs []int64) {
	for _, orderID := range orderIDs {
		lookbackDone.Store(orderID, struct{}{})
	}
}

func expireWaitingOrders() {
	for _, t := range model.GetOrderByStatus(model.OrderStatusWaiting) {
		if time.Now().Unix() < t.ExpiredAt.Unix() {
			continue
		}

		t.SetExpired()
		notify.Bepusdt(t)
	}
}

func getConfirmingOrders(tradeType []model.TradeType) []model.Order {
	var orders = make([]model.Order, 0)
	var data = make([]model.Order, 0)
	var db = model.Db.Where("status = ?", model.OrderStatusConfirming)
	if len(tradeType) > 0 {
		db = db.Where("trade_type in (?)", tradeType)
	}

	db.Find(&orders)

	for _, order := range orders {
		if time.Now().Unix() >= order.ExpiredAt.Unix() {
			if order.ConfirmedAt == nil || order.ConfirmedAt.IsZero() || !order.ConfirmedAt.Before(order.ExpiredAt) {
				order.SetFailed()
				notify.Bepusdt(order)

				continue
			}
		}

		data = append(data, order)
	}

	return data
}

func amountMatch(amount decimal.Decimal, target, tradeType string) bool {
	mode := model.GetC(model.PaymentMatchMode)
	switch model.MatchMode(mode) {
	case model.Classic:
		return amount.String() == target
	case model.HasPrefix:
		s := amount.String()
		if !strings.HasPrefix(s, target) {
			return false
		}
		rest := s[len(target):]
		if rest == "" {
			return true
		}

		return strings.Contains(target, ".") || strings.HasPrefix(rest, ".")
	case model.RoundOff:
		t, err := decimal.NewFromString(target)
		if err != nil {
			log.Warn(err.Error())

			return false
		}

		_, precision := model.GetAtomicity(model.TradeType(tradeType)) // 标准精度
		precision2 := abs(t.Exponent())                                // 实际精度
		if precision2 != precision {
			precision = precision2
		}

		a := amount.Round(precision)
		t = t.Round(precision)

		return a.Equal(t)
	}

	return false
}

func abs(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}
