package web

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/model"
	"net/http"
	"sort"
)

type epayForm struct {
	Pid        string `form:"pid" binding:"required"`
	Type       string `form:"type" binding:"required"`
	NotifyUrl  string `form:"notify_url" binding:"required"`
	ReturnUrl  string `form:"return_url" binding:"required"`
	OutTradeNo string `form:"out_trade_no" binding:"required"`
	Name       string `form:"name" binding:"required"`
	Money      string `form:"money" binding:"required"`
	Sign       string `form:"sign" binding:"required"`
}

// EpaySubmit 【兼容】易支付提交
func EpaySubmit(ctx *gin.Context) {
	var f epayForm
	if err := ctx.ShouldBind(&f); err != nil {
		ctx.String(200, "参数错误："+err.Error())

		return
	}

	if EpaySign(f, config.GetAuthToken()) != f.Sign {
		ctx.String(200, "签名错误")

		return
	}

	var order, err = buildOrder(cast.ToFloat64(f.Money), model.OrderApiTypeEpay, f.OutTradeNo, f.Type, f.ReturnUrl, f.NotifyUrl, f.Name)
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

func EpaySign(p epayForm, key string) string {
	params := map[string]string{
		"pid":          p.Pid,
		"type":         p.Type,
		"notify_url":   p.NotifyUrl,
		"return_url":   p.ReturnUrl,
		"out_trade_no": p.OutTradeNo,
		"name":         p.Name,
		"money":        p.Money,
	}

	// 提取 keys 并排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	// 构建签名字符串
	signStr := ""
	for _, k := range keys {
		if k != "sign" && k != "sign_type" && params[k] != "" {
			signStr += fmt.Sprintf("%s=%s&", k, params[k])
		}
	}
	signStr = signStr[:len(signStr)-1] // 移除最后一个 '&'
	signStr += key                     // 添加密钥

	// 计算 MD5
	hash := md5.New()
	hash.Write([]byte(signStr))
	md5sum := hex.EncodeToString(hash.Sum(nil))

	return md5sum
}
