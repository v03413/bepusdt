package model

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/task/rate"
)

const (
	OrderNotifyStateSucc = 1 // 回调成功
	OrderNotifyStateFail = 0 // 回调失败

	OrderStatusWaiting    = 1 // 等待支付
	OrderStatusSuccess    = 2 // 交易确认成功
	OrderStatusExpired    = 3 // 订单过期
	OrderStatusCanceled   = 4 // 订单取消
	OrderStatusConfirming = 5 // 等待交易确认
	OrderStatusFailed     = 6 // 交易确认失败

	OrderTradeTypeTronTrx      = "tron.trx"
	OrderTradeTypeUsdtTrc20    = "usdt.trc20"
	OrderTradeTypeUsdcTrc20    = "usdc.trc20"
	OrderTradeTypeUsdtPolygon  = "usdt.polygon"
	OrderTradeTypeUsdcPolygon  = "usdc.polygon"
	OrderTradeTypeUsdtArbitrum = "usdt.arbitrum"
	OrderTradeTypeUsdcArbitrum = "usdc.arbitrum"
	OrderTradeTypeUsdtErc20    = "usdt.erc20"
	OrderTradeTypeUsdcErc20    = "usdc.erc20"
	OrderTradeTypeUsdtBep20    = "usdt.bep20"
	OrderTradeTypeUsdcBep20    = "usdc.bep20"
	OrderTradeTypeUsdtXlayer   = "usdt.xlayer"
	OrderTradeTypeUsdcXlayer   = "usdc.xlayer"
	OrderTradeTypeUsdcBase     = "usdc.base"
	OrderTradeTypeUsdtSolana   = "usdt.solana"
	OrderTradeTypeUsdcSolana   = "usdc.solana"
	OrderTradeTypeUsdtAptos    = "usdt.aptos"
	OrderTradeTypeUsdcAptos    = "usdc.aptos"
)

const (
	OrderApiTypeEpusdt = "epusdt" // epusdt
	OrderApiTypeEpay   = "epay"   // 彩虹易支付
)

var calcMutex sync.Mutex

type TradeOrders struct {
	Id          int64     `gorm:"primary_key;AUTO_INCREMENT;comment:id"`
	OrderId     string    `gorm:"column:order_id;type:varchar(128);not null;index;comment:商户ID"`
	TradeId     string    `gorm:"column:trade_id;type:varchar(128);not null;uniqueIndex;comment:本地ID"`
	TradeType   string    `gorm:"column:trade_type;type:varchar(20);not null;comment:交易类型"`
	TradeHash   string    `gorm:"column:trade_hash;type:varchar(130);default:'';unique;comment:交易哈希"`
	TradeRate   string    `gorm:"column:trade_rate;type:varchar(10);not null;comment:交易汇率"`
	Amount      string    `gorm:"type:decimal(10,2);not null;default:0;comment:交易数额"`
	Money       float64   `gorm:"type:decimal(10,2);not null;default:0;comment:订单交易金额"`
	Address     string    `gorm:"column:address;type:varchar(64);not null;comment:收款地址"`
	FromAddress string    `gorm:"type:varchar(34);not null;default:'';comment:支付地址"`
	Status      int       `gorm:"type:tinyint(1);not null;default:1;index;comment:交易状态"`
	Name        string    `gorm:"type:varchar(64);not null;default:'';comment:商品名称"`
	ApiType     string    `gorm:"type:varchar(20);not null;default:'epusdt';comment:API类型"`
	ReturnUrl   string    `gorm:"type:varchar(255);not null;default:'';comment:同步地址"`
	NotifyUrl   string    `gorm:"type:varchar(255);not null;default:'';comment:异步地址"`
	NotifyNum   int       `gorm:"column:notify_num;type:int(11);not null;default:0;comment:回调次数"`
	NotifyState int       `gorm:"column:notify_state;type:tinyint(1);not null;default:0;comment:回调状态 1：成功 0：失败"`
	RefBlockNum int64     `gorm:"type:bigint(20);not null;default:0;comment:交易所在区块"`
	ExpiredAt   time.Time `gorm:"column:expired_at;type:timestamp;not null;comment:失效时间"`
	CreatedAt   time.Time `gorm:"autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;type:timestamp;not null;comment:更新时间"`
	ConfirmedAt time.Time `gorm:"type:timestamp;null;comment:交易确认时间"`
}

func (o *TradeOrders) SetCanceled() error {
	o.Status = OrderStatusCanceled

	return DB.Save(o).Error
}

func (o *TradeOrders) SetExpired() {
	o.Status = OrderStatusExpired

	DB.Save(o)
}

func (o *TradeOrders) SetSuccess() {
	o.Status = OrderStatusSuccess

	DB.Save(o)
}

func (o *TradeOrders) SetFailed() {
	o.Status = OrderStatusFailed

	DB.Save(o)
}

func (o *TradeOrders) MarkConfirming(blockNum int64, from, hash string, at time.Time) {
	o.FromAddress = from
	o.ConfirmedAt = at
	o.TradeHash = hash
	o.RefBlockNum = blockNum
	o.Status = OrderStatusConfirming

	DB.Save(o)
}

func (o *TradeOrders) SetNotifyState(state int) error {
	o.NotifyNum += 1
	o.NotifyState = state

	return DB.Save(o).Error
}

func (o *TradeOrders) GetStatusLabel() string {
	var label = "🟢收款成功"
	if o.Status == OrderStatusExpired {

		label = "🔴交易过期"
	}
	if o.Status == OrderStatusWaiting {

		label = "🟡等待支付"
	}
	if o.Status == OrderStatusCanceled {

		label = "⚪️订单取消"
	}

	return label
}

func (o *TradeOrders) GetStatusEmoji() string {
	var label = "🟢"
	if o.Status == OrderStatusExpired {

		label = "🔴"
	}
	if o.Status == OrderStatusWaiting {

		label = "🟡"
	}
	if o.Status == OrderStatusCanceled {

		label = "⚪️"
	}

	return label
}

func (o *TradeOrders) GetDetailUrl() string {

	return GetDetailUrl(o.TradeType, o.TradeHash)
}

func GetDetailUrl(tradeType, hash string) string {
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtErc20, OrderTradeTypeUsdcErc20}) {
		return "https://etherscan.io/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtBep20, OrderTradeTypeUsdcBep20}) {
		return "https://bscscan.com/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtXlayer, OrderTradeTypeUsdcXlayer}) {
		return "https://web3.okx.com/zh-hans/explorer/x-layer/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtPolygon, OrderTradeTypeUsdcPolygon}) {
		return "https://polygonscan.com/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtArbitrum, OrderTradeTypeUsdcArbitrum}) {
		return "https://arbiscan.io/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdcBase}) {
		return "https://basescan.org/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtSolana, OrderTradeTypeUsdcSolana}) {
		return "https://solscan.io/tx/" + hash
	}
	if help.InStrings(tradeType, []string{OrderTradeTypeUsdtAptos, OrderTradeTypeUsdcAptos}) {
		return fmt.Sprintf("https://explorer.aptoslabs.com/txn/%s?network=mainnet", hash)
	}

	return "https://tronscan.org/#/transaction/" + hash
}

func GetTradeOrder(tradeId string) (TradeOrders, bool) {
	var order TradeOrders
	var res = DB.Where("trade_id = ?", tradeId).Take(&order)

	return order, res.Error == nil
}

func GetOrderByStatus(Status int) []TradeOrders {
	var orders = make([]TradeOrders, 0)

	DB.Where("status = ?", Status).Find(&orders)

	return orders
}

func GetNotifyFailedTradeOrders() ([]TradeOrders, error) {
	var orders []TradeOrders
	var res = DB.Where("status = ?", OrderStatusSuccess).
		Where("notify_num <= ?", conf.NotifyMaxRetry).
		Where("notify_state = ?", OrderNotifyStateFail).Find(&orders)

	return orders, res.Error
}

// CalcTradeAmount 计算当前实际可用的交易金额
func CalcTradeAmount(wa []WalletAddress, rate, money float64, tradeType string) (WalletAddress, string) {
	calcMutex.Lock()
	defer calcMutex.Unlock()

	var orders []TradeOrders
	var lock = make(map[string]bool)
	DB.Where("status = ? and trade_type = ?", OrderStatusWaiting, tradeType).Find(&orders)
	for _, order := range orders {

		lock[order.Address+order.Amount] = true
	}

	var atom, prec = getTokenAtomicityByTradeType(tradeType)

	var payAmount, _ = decimal.NewFromString(strconv.FormatFloat(money/rate, 'f', prec, 64))
	for {
		for _, address := range wa {
			_key := address.Address + payAmount.String()
			if _, ok := lock[_key]; ok {

				continue
			}

			return address, payAmount.String()
		}

		// 已经被占用，每次递增一个原子精度
		payAmount = payAmount.Add(atom)
	}
}

func CalcTradeExpiredAt(sec uint64) time.Time {
	timeout := conf.GetExpireTime() * time.Second
	if sec >= 60 {
		timeout = time.Duration(sec) * time.Second
	}

	return time.Now().Add(timeout)
}

func GetTradeRate(token TokenType, param string) (float64, error) {
	if param != "" {
		switch token {
		case TokenTypeUSDT:
			return rate.ParseFloatRate(param, rate.GetOkxUsdtRawRate()), nil
		case TokenTypeUSDC:
			return rate.ParseFloatRate(param, rate.GetOkxUsdcRawRate()), nil
		case TokenTypeTRX:
			return rate.ParseFloatRate(param, rate.GetOkxTrxRawRate()), nil
		}

		return 0, fmt.Errorf("(%s)交易汇率计算获取失败：%s", token, param)
	}

	switch token {
	case TokenTypeUSDT:
		return rate.GetUsdtCalcRate(), nil
	case TokenTypeUSDC:
		return rate.GetUsdcCalcRate(), nil
	case TokenTypeTRX:
		return rate.GetTrxCalcRate(), nil
	}

	return 0, fmt.Errorf("(%s)交易汇率获取失败", token)
}

func getTokenAtomicityByTradeType(tradeType string) (decimal.Decimal, int) {
	switch tradeType {
	case OrderTradeTypeTronTrx:
		return conf.GetTrxAtomicity()
	case OrderTradeTypeUsdtTrc20:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtErc20:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtBep20:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtAptos:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtXlayer:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtSolana:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtPolygon:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdtArbitrum:
		return conf.GetUsdtAtomicity()
	case OrderTradeTypeUsdcTrc20:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcErc20:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcBep20:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcAptos:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcXlayer:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcSolana:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcPolygon:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcArbitrum:
		return conf.GetUsdcAtomicity()
	case OrderTradeTypeUsdcBase:
		return conf.GetUsdcAtomicity()
	default:
		return conf.GetUsdtAtomicity()
	}
}
