package model

import (
	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/config"
	"strconv"
	"sync"
	"time"
)

const OrderStatusExpired = 3
const OrderStatusSuccess = 2
const OrderStatusWaiting = 1

const OrderNotifyStateSucc = 1
const OrderNotifyStateFail = 0

const OrderTradeTypeUsdtTrc20 = "usdt.trc20"
const OrderTradeTypeTronTrx = "tron.trx"

var _calcMutex sync.Mutex

type TradeOrders struct {
	Id          int64     `gorm:"primary_key;AUTO_INCREMENT;comment:id"`
	OrderId     string    `gorm:"type:varchar(255);not null;unique;color:blue;comment:客户订单ID"`
	TradeId     string    `gorm:"type:varchar(255);not null;unique;color:blue;comment:本地订单ID"`
	TradeHash   string    `gorm:"type:varchar(64);default:'';unique;comment:交易哈希"`
	UsdtRate    string    `gorm:"type:varchar(10);not null;comment:USDT汇率"`
	Amount      string    `gorm:"type:decimal(10,2);not null;default:0;comment:USDT交易数额"`
	Money       float64   `gorm:"type:decimal(10,2);not null;default:0;comment:订单交易金额"`
	Address     string    `gorm:"type:varchar(34);not null;comment:收款地址"`
	FromAddress string    `gorm:"type:varchar(34);not null;default:'';comment:支付地址"`
	Status      int       `gorm:"type:tinyint(1);not null;default:0;comment:交易状态 1：等待支付 2：支付成功 3：订单过期"`
	ReturnUrl   string    `gorm:"type:varchar(255);not null;default:'';comment:同步地址"`
	NotifyUrl   string    `gorm:"type:varchar(255);not null;default:'';comment:异步地址"`
	NotifyNum   int       `gorm:"type:int(11);not null;default:0;comment:回调次数"`
	NotifyState int       `gorm:"type:tinyint(1);not null;default:0;comment:回调状态 1：成功 0：失败"`
	RefBlockNum int64     `gorm:"type:bigint(20);not null;default:0;comment:交易所在区块"`
	ExpiredAt   time.Time `gorm:"type:timestamp;not null;comment:订单失效时间"`
	CreatedAt   time.Time `gorm:"autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;type:timestamp;not null;comment:更新时间"`
	ConfirmedAt time.Time `gorm:"type:timestamp;null;comment:交易确认时间"`
}

func (o *TradeOrders) OrderSetExpired() error {
	o.Status = OrderStatusExpired

	return DB.Save(o).Error
}

func (o *TradeOrders) OrderUpdateTxInfo(refBlockNum int64, fromAddress, tradeHash string, confirmedAt time.Time) error {
	o.FromAddress = fromAddress
	o.ConfirmedAt = confirmedAt
	o.TradeHash = tradeHash
	o.RefBlockNum = refBlockNum
	r := DB.Save(o)

	return r.Error
}

func (o *TradeOrders) OrderSetSucc() error {
	o.Status = OrderStatusSuccess // 标记成功

	r := DB.Save(o)

	return r.Error
}

func (o *TradeOrders) OrderSetNotifyState(state int) error {
	o.NotifyNum += 1
	o.NotifyState = state

	return DB.Save(o).Error
}

func (o *TradeOrders) GetStatusLabel() string {
	var _label = "🟢 收款成功"
	if o.Status == OrderStatusExpired {

		_label = "🔴 交易过期"
	}
	if o.Status == OrderStatusWaiting {

		_label = "🟡 等待支付"
	}

	return _label
}

func GetTradeOrder(tradeId string) (TradeOrders, bool) {
	var order TradeOrders
	var res = DB.Where("trade_id = ?", tradeId).First(&order)

	return order, res.Error == nil
}

func GetTradeOrderByStatus(Status int) ([]TradeOrders, error) {
	var orders []TradeOrders
	var res = DB.Where("status = ?", Status).Find(&orders)

	return orders, res.Error
}

func GetNotifyFailedTradeOrders() ([]TradeOrders, error) {
	var orders []TradeOrders
	var res = DB.Where("status = ?", OrderStatusSuccess).Where("notify_num > ?", 0).
		Where("notify_state = ?", OrderNotifyStateFail).Find(&orders)

	return orders, res.Error
}

// CalcTradeAmount 计算当前实际可用的交易金额
func CalcTradeAmount(wa []WalletAddress, rate, money float64) (WalletAddress, string) {
	_calcMutex.Lock()
	defer _calcMutex.Unlock()

	var _orders []TradeOrders
	var _lock = make(map[string]bool)
	DB.Where("status = ?", OrderStatusWaiting).Find(&_orders)
	for _, _order := range _orders {

		_lock[_order.Address+_order.Amount] = true
	}

	var _atom, _prec = config.GetAtomicity()
	var payAmount = strconv.FormatFloat(money/rate, 'f', _prec, 64)
	var _payAmount, _ = decimal.NewFromString(payAmount)
	for {
		for _, address := range wa {
			_key := address.Address + _payAmount.String()
			if _, ok := _lock[_key]; ok {

				continue
			}

			return address, _payAmount.String()
		}

		// 已经被占用，每次递增一个原子精度
		_payAmount = _payAmount.Add(_atom)
	}
}
