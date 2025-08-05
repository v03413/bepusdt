package web

import (
	"fmt"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"strings"
	"sync"
)

type orderParams struct {
	Money       float64 `json:"money"`        // 交易金额 CNY
	ApiType     string  `json:"api_type"`     // 支付API类型
	PayAddress  string  `json:"pay_address"`  // 收款地址
	OrderId     string  `json:"order_id"`     // 商户订单ID
	TradeType   string  `json:"trade_type"`   // 交易类型
	RedirectUrl string  `json:"redirect_url"` // 成功跳转地址
	NotifyUrl   string  `json:"notify_url"`   // 异步通知地址
	Name        string  `json:"name"`         // 商品名称
	Timeout     uint64  `json:"timeout"`      // 订单超时时间（秒）
	Rate        string  `json:"rate"`         // 强制指定汇率
}

type trade struct {
	TokenType model.TokenType
	Rate      float64
	Address   model.WalletAddress
	Amount    string
}

func buildOrder(p orderParams) (model.TradeOrders, error) {
	var order model.TradeOrders

	model.DB.Where("order_id = ?", p.OrderId).Find(&order)
	if order.Status == model.OrderStatusSuccess {
		return order, nil
	}

	if order.Status == model.OrderStatusWaiting {
		return rebuildOrder(order, p)
	}

	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	data, err := buildTrade(p)
	if err != nil {
		return order, err
	}

	return newOrder(p, data)
}

func rebuildOrder(t model.TradeOrders, p orderParams) (model.TradeOrders, error) {
	if p.OrderId == t.OrderId && p.TradeType == t.TradeType && p.Money == t.Money {
		return t, nil
	}

	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	data, err := buildTrade(p)
	if err != nil {
		return t, err
	}

	t.Amount = data.Amount
	t.TradeType = p.TradeType
	t.Address = data.Address.Address

	return t, model.DB.Save(&t).Error
}

func newOrder(p orderParams, data trade) (model.TradeOrders, error) {
	tradeId, err := help.GenerateTradeId()
	if err != nil {
		return model.TradeOrders{}, err
	}

	tradeOrder := model.TradeOrders{
		OrderId:     p.OrderId,
		TradeId:     tradeId,
		TradeHash:   tradeId,
		TradeType:   p.TradeType,
		TradeRate:   fmt.Sprintf("%v", data.Rate),
		Amount:      data.Amount,
		Money:       p.Money,
		Address:     data.Address.Address,
		Status:      model.OrderStatusWaiting,
		Name:        p.Name,
		ApiType:     p.ApiType,
		ReturnUrl:   p.RedirectUrl,
		NotifyUrl:   p.NotifyUrl,
		NotifyNum:   0,
		NotifyState: model.OrderNotifyStateFail,
		ExpiredAt:   model.CalcTradeExpiredAt(p.Timeout),
	}

	if err = model.DB.Create(&tradeOrder).Error; err != nil {
		log.Error("订单创建失败：", err.Error())
		return model.TradeOrders{}, err
	}

	model.PushWebhookEvent(model.WebhookEventOrderCreate, tradeOrder)
	return tradeOrder, nil
}

func buildTrade(p orderParams) (trade, error) {
	// 获取代币类型
	tokenType, err := model.GetTokenType(p.TradeType)
	if err != nil {
		return trade{}, fmt.Errorf("类型(%s)不支持：%v", p.TradeType, err)
	}

	// 获取交易汇率
	rate, err := model.GetTradeRate(tokenType, strings.TrimSpace(p.Rate))
	if err != nil {
		return trade{}, err
	}

	// 可用钱包地址
	wallet := model.GetAvailableAddress(p.PayAddress, p.TradeType)
	if len(wallet) == 0 {
		return trade{}, fmt.Errorf("类型(%s)未检测到可用钱包地址", p.TradeType)
	}

	// 计算交易金额
	address, amount := model.CalcTradeAmount(wallet, rate, p.Money, p.TradeType)

	return trade{
		TokenType: tokenType,
		Rate:      rate,
		Address:   address,
		Amount:    amount,
	}, nil
}
