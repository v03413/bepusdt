package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/epay"
	"github.com/v03413/bepusdt/app/model"
	"net/http"
)

// epaySubmit 【兼容】易支付提交
func epaySubmit(ctx *gin.Context) {
	if err := ctx.Request.ParseForm(); err != nil {
		ctx.String(200, "参数解析错误："+err.Error())

		return
	}

	var data = make(map[string]string)
	for k, v := range ctx.Request.PostForm {
		if len(v) == 0 {
			data[k] = ""

			continue
		}

		data[k] = v[0]
	}

	if data["pid"] != epay.Pid {
		ctx.String(200, "Bepusdt 易支付兼容模式，商户号【PID】必须固定为"+epay.Pid)

		return
	}

	if epay.Sign(data, config.GetAuthToken()) != data["sign"] {
		ctx.String(200, "签名错误")

		return
	}

	var order, err = buildOrder(cast.ToFloat64(data["money"]), model.OrderApiTypeEpay, data["out_trade_no"], data["type"], data["return_url"], data["notify_url"], data["name"])
	if err != nil {
		ctx.String(200, fmt.Sprintf("订单创建失败：%v", err))

		return
	}

	// 解析请求地址
	var host = "http://" + ctx.Request.Host
	if ctx.Request.TLS != nil {
		host = "https://" + ctx.Request.Host
	}

	ctx.Redirect(http.StatusFound, fmt.Sprintf("%s/pay/checkout-counter/%s", config.GetAppUri(host), order.TradeId))
}
