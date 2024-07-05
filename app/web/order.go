package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/usdt"
	"time"
)

// CreateTransaction 创建订单
func CreateTransaction(ctx *gin.Context) {
	_data, _ := ctx.Get("data")
	data := _data.(map[string]any)
	_orderId, ok1 := data["order_id"].(string)
	_money, ok2 := data["amount"].(float64)
	_notifyUrl, ok3 := data["notify_url"].(string)
	_redirectUrl, ok4 := data["redirect_url"].(string)
	// ---
	if !ok1 || !ok2 || !ok3 || !ok4 {
		log.Warn("参数错误", data)
		ctx.JSON(200, RespFailJson(fmt.Errorf("参数错误")))
		return
	}

	// 获取兑换汇率
	rate := usdt.GetLatestRate()

	// 获取钱包地址
	var wallet = model.GetAvailableAddress()
	if len(wallet) == 0 {
		log.Error("订单创建失败：还没有配置收款地址")
		ctx.JSON(200, RespFailJson(fmt.Errorf("还没有配置收款地址")))

		return
	}

	// 计算交易金额
	address, _amount := model.CalcTradeAmount(wallet, rate, _money)

	// 解析请求地址
	var _host = "http://" + ctx.Request.Host
	if ctx.Request.TLS != nil {
		_host = "https://" + ctx.Request.Host
	}

	// 创建交易订单
	var _tradeId = help.GenerateTradeId()
	var _expiredAt = time.Now().Add(config.GetExpireTime() * time.Second)
	var _orderData = model.TradeOrders{
		OrderId:     _orderId,
		TradeId:     _tradeId,
		TradeHash:   _tradeId, // 这里默认填充一个本地交易ID，等支付成功后再更新为实际交易哈希
		UsdtRate:    fmt.Sprintf("%v", rate),
		Amount:      _amount,
		Money:       _money,
		Address:     address.Address,
		Status:      model.OrderStatusWaiting,
		ReturnUrl:   _redirectUrl,
		NotifyUrl:   _notifyUrl,
		NotifyNum:   0,
		NotifyState: model.OrderNotifyStateFail,
		ExpiredAt:   _expiredAt,
	}
	var res = model.DB.Create(&_orderData)
	if res.Error != nil {
		log.Error("订单创建失败：", res.Error.Error())
		ctx.JSON(200, RespFailJson(fmt.Errorf("订单创建失败")))

		return
	}

	// 返回响应数据
	ctx.JSON(200, RespSuccJson(gin.H{
		"trade_id":        _tradeId,
		"order_id":        _orderId,
		"amount":          _money,
		"actual_amount":   _amount,
		"token":           address.Address,
		"expiration_time": _expiredAt.Second(),
		"payment_url":     fmt.Sprintf("%s/pay/checkout-counter/%s", config.GetAppUri(_host), _tradeId),
	}))
	log.Info(fmt.Sprintf("订单创建成功，商户订单号：%s", _orderId))
}
