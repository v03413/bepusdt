package web

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/epay"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"net/url"
	"time"
)

func signVerify(ctx *gin.Context) {
	rawData, err := ctx.GetRawData()
	if err != nil {
		log.Error(err.Error())
		ctx.JSON(400, gin.H{"error": err.Error()})
		ctx.Abort()

		return
	}

	var m = make(map[string]any)
	if err = json.Unmarshal(rawData, &m); err != nil {
		log.Error(err.Error())
		ctx.JSON(400, gin.H{"error": err.Error()})
		ctx.Abort()

		return
	}

	sign, ok := m["signature"]
	if !ok {
		log.Warn("signature not found", m)
		ctx.JSON(400, gin.H{"error": "signature not found"})
		ctx.Abort()

		return
	}

	if help.EpusdtSign(m, conf.GetAuthToken()) != sign {
		log.Warn("签名错误", m)
		ctx.JSON(400, gin.H{"error": "签名错误"})
		ctx.Abort()

		return
	}

	ctx.Set("data", m)
}

func createTransaction(ctx *gin.Context) {
	var data = ctx.GetStringMap("data")
	var tradeType = model.OrderTradeTypeUsdtTrc20
	var address = ""
	for _, key := range []string{"order_id", "amount", "notify_url", "redirect_url"} {
		if _, ok := data[key]; !ok {
			log.Warn(fmt.Sprintf("参数 %s 不存在", key), data)
			ctx.JSON(200, respFailJson(fmt.Sprintf("参数 %s 不存在", key)))

			return
		}
	}

	if v, ok := data["trade_type"]; ok {

		tradeType = model.GetTradeType(cast.ToString(v))
	}
	if v, ok := data["address"]; ok && cast.ToString(v) != "" {
		address = cast.ToString(v)
		if !help.IsValidTronAddress(address) && !help.IsValidPolygonAddress(address) {
			ctx.JSON(200, respFailJson(fmt.Sprintf("收款钱包地址(%s)不合法", address)))

			return
		}
	}

	// 解析请求地址
	var host = "http://" + ctx.Request.Host
	if ctx.Request.TLS != nil {

		host = "https://" + ctx.Request.Host
	}

	var amount = cast.ToFloat64(data["amount"])
	var orderId = cast.ToString(data["order_id"])
	var notifyUrl = cast.ToString(data["notify_url"])
	var redirectUrl = cast.ToString(data["redirect_url"])

	var order, err = buildOrder(amount, model.OrderApiTypeEpusdt, address, orderId, tradeType, redirectUrl, notifyUrl, orderId)
	if err != nil {
		ctx.JSON(200, respFailJson(fmt.Sprintf("订单创建失败：%s", err.Error())))

		return
	}

	// 返回响应数据
	ctx.JSON(200, respSuccJson(gin.H{
		"trade_id":        order.TradeId,
		"order_id":        order.OrderId,
		"status":          order.Status,
		"amount":          order.Money,
		"actual_amount":   order.Amount,
		"token":           order.Address,
		"expiration_time": uint64(order.ExpiredAt.Sub(time.Now()).Seconds()),
		"payment_url":     fmt.Sprintf("%s/pay/checkout-counter/%s", conf.GetAppUri(host), order.TradeId),
	}))
	log.Info(fmt.Sprintf("订单创建成功，商户订单号：%s", orderId))
}

func cancelTransaction(ctx *gin.Context) {
	var data = ctx.GetStringMap("data")
	tradeId, ok := data["trade_id"].(string)
	if !ok {
		ctx.JSON(200, respFailJson("参数 trade_id 不存在"))

		return
	}

	var order, ok2 = model.GetTradeOrder(tradeId)
	if !ok2 {
		ctx.JSON(200, respFailJson("订单不存在"))

		return
	}

	if order.Status != model.OrderStatusWaiting {
		ctx.JSON(200, respFailJson(fmt.Sprintf("当前订单(%s)状态不允许取消", tradeId)))

		return
	}

	if err := order.OrderSetCanceled(); err != nil {
		ctx.JSON(200, respFailJson(fmt.Sprintf("订单取消失败：%s", err.Error())))

		return
	}

	ctx.JSON(200, respSuccJson(gin.H{"trade_id": tradeId}))
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
		ctx.JSON(200, respFailJson("订单不存在"))

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
