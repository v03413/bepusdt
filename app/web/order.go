package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/epay"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/rate"
	"net/url"
	"sync"
	"time"
)

// createTransaction 创建订单
func createTransaction(ctx *gin.Context) {
	_data, _ := ctx.Get("data")
	data := _data.(map[string]any)
	orderId, ok1 := data["order_id"].(string)
	money, ok2 := data["amount"].(float64)
	notifyUrl, ok3 := data["notify_url"].(string)
	redirectUrl, ok4 := data["redirect_url"].(string)
	tradeType, _ := data["trade_type"].(string)
	if tradeType != model.OrderTradeTypeTronTrx {
		tradeType = model.OrderTradeTypeUsdtTrc20
	}
	// ---
	if !ok1 || !ok2 || !ok3 || !ok4 {
		log.Warn("参数错误", data)
		ctx.JSON(200, respFailJson(fmt.Errorf("参数错误")))
		return
	}

	// 解析请求地址
	var host = "http://" + ctx.Request.Host
	if ctx.Request.TLS != nil {
		host = "https://" + ctx.Request.Host
	}

	var order, err = buildOrder(money, model.OrderApiTypeEpusdt, orderId, tradeType, redirectUrl, notifyUrl, orderId)
	if err != nil {
		ctx.JSON(200, respFailJson(fmt.Errorf("订单创建失败：%w", err)))

		return
	}

	// 返回响应数据
	ctx.JSON(200, respSuccJson(gin.H{
		"trade_id":        order.TradeId,
		"order_id":        orderId,
		"amount":          money,
		"actual_amount":   order.Amount,
		"token":           order.Address,
		"expiration_time": int64(order.ExpiredAt.Sub(time.Now()).Seconds()),
		"payment_url":     fmt.Sprintf("%s/pay/checkout-counter/%s", config.GetAppUri(host), order.TradeId),
	}))
	log.Info(fmt.Sprintf("订单创建成功，商户订单号：%s", orderId))
}

func buildOrder(money float64, apiType, orderId, tradeType, redirectUrl, notifyUrl, name string) (model.TradeOrders, error) {
	var lock sync.Mutex
	var order model.TradeOrders

	// 暂时先强制使用互斥锁，后续有需求的话再考虑优化
	lock.Lock()
	defer lock.Unlock()

	// 获取兑换汇率
	var calcRate = rate.GetUsdtCalcRate(config.DefaultUsdtCnyRate)
	if tradeType == model.OrderTradeTypeTronTrx {

		calcRate = rate.GetTrxCalcRate(config.DefaultTrxCnyRate)
	}

	// 获取钱包地址
	var wallet = model.GetAvailableAddress()
	if len(wallet) == 0 {
		log.Error("订单创建失败：还没有配置收款地址")

		return order, fmt.Errorf("还没有配置收款地址")
	}

	// 计算交易金额
	address, amount := model.CalcTradeAmount(wallet, calcRate, money, tradeType)
	tradeId, err := help.GenerateTradeId()
	if err != nil {

		return order, err
	}

	// 创建交易订单
	var expiredAt = time.Now().Add(config.GetExpireTime() * time.Second)
	var tradeOrder = model.TradeOrders{
		OrderId:     orderId,
		TradeId:     tradeId,
		TradeHash:   tradeId, // 这里默认填充一个本地交易ID，等支付成功后再更新为实际交易哈希
		TradeType:   tradeType,
		TradeRate:   fmt.Sprintf("%v", calcRate),
		Amount:      amount,
		Money:       money,
		Address:     address.Address,
		Status:      model.OrderStatusWaiting,
		Name:        name,
		ApiType:     apiType,
		ReturnUrl:   redirectUrl,
		NotifyUrl:   notifyUrl,
		NotifyNum:   0,
		NotifyState: model.OrderNotifyStateFail,
		ExpiredAt:   expiredAt,
	}
	var res = model.DB.Create(&tradeOrder)
	if res.Error != nil {
		log.Error("订单创建失败：", res.Error.Error())

		return order, res.Error
	}

	return tradeOrder, nil
}

func checkoutCounter(ctx *gin.Context) {
	var tradeId = ctx.Param("trade_id")
	var order, ok = model.GetTradeOrder(tradeId)
	if !ok {
		ctx.String(200, "订单不存在")

		return
	}

	uri, err := url.ParseRequestURI(order.ReturnUrl)
	if err != nil {
		ctx.String(200, "同步地址错误")
		log.Error("同步地址解析错误", err.Error())

		return
	}

	ctx.HTML(200, order.TradeType+".html", gin.H{
		"http_host":  uri.Host,
		"amount":     order.Amount,
		"address":    order.Address,
		"expire":     int64(order.ExpiredAt.Sub(time.Now()).Seconds()),
		"return_url": order.ReturnUrl,
		"usdt_rate":  order.TradeRate,
		"trade_id":   tradeId,
		"trade_type": order.TradeType,
	})
}

func checkStatus(ctx *gin.Context) {
	var tradeId = ctx.Param("trade_id")
	var order, ok = model.GetTradeOrder(tradeId)
	if !ok {
		ctx.JSON(200, respFailJson(fmt.Errorf("订单不存在")))

		return
	}

	var returnUrl string
	if order.Status == model.OrderStatusSuccess {
		returnUrl = order.ReturnUrl
		if order.ApiType == model.OrderApiTypeEpay {
			// 易支付兼容
			returnUrl = fmt.Sprintf("%s?%s", returnUrl, epay.BuildNotifyParams(order))
		}
	}

	// 返回响应数据
	ctx.JSON(200, gin.H{"trade_id": tradeId, "status": order.Status, "return_url": returnUrl})
}
