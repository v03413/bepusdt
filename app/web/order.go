package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/rate"
	"time"
)

// CreateTransaction 创建订单
func CreateTransaction(ctx *gin.Context) {
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
		ctx.JSON(200, RespFailJson(fmt.Errorf("参数错误")))
		return
	}

	// 获取兑换汇率
	var calcRate = rate.GetUsdtCalcRate(config.DefaultUsdtCnyRate)
	if tradeType == model.OrderTradeTypeTronTrx {

		calcRate = rate.GetTrxCnyCalcRate(config.DefaultTrxCnyRate)
	}

	// 获取钱包地址
	var wallet = model.GetAvailableAddress()
	if len(wallet) == 0 {
		log.Error("订单创建失败：还没有配置收款地址")
		ctx.JSON(200, RespFailJson(fmt.Errorf("还没有配置收款地址")))

		return
	}

	// 计算交易金额
	address, amount := model.CalcTradeAmount(wallet, calcRate, money, tradeType)

	// 解析请求地址
	var host = "http://" + ctx.Request.Host
	if ctx.Request.TLS != nil {
		host = "https://" + ctx.Request.Host
	}

	// 创建交易订单
	var tradeId = uuid.New().String()
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
		ReturnUrl:   redirectUrl,
		NotifyUrl:   notifyUrl,
		NotifyNum:   0,
		NotifyState: model.OrderNotifyStateFail,
		ExpiredAt:   expiredAt,
	}
	var res = model.DB.Create(&tradeOrder)
	if res.Error != nil {
		log.Error("订单创建失败：", res.Error.Error())
		ctx.JSON(200, RespFailJson(fmt.Errorf("订单创建失败")))

		return
	}

	// 返回响应数据
	ctx.JSON(200, RespSuccJson(gin.H{
		"trade_id":        tradeId,
		"order_id":        orderId,
		"amount":          money,
		"actual_amount":   amount,
		"token":           address.Address,
		"expiration_time": int64(expiredAt.Sub(time.Now()).Seconds()),
		"payment_url":     fmt.Sprintf("%s/pay/checkout-counter/%s", config.GetAppUri(host), tradeId),
	}))
	log.Info(fmt.Sprintf("订单创建成功，商户订单号：%s", orderId))
}
