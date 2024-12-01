package web

import (
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/static"
	"html/template"
	"io/fs"
	"net/http"
)

func Start() {
	gin.SetMode(gin.ReleaseMode)

	listen := config.GetListen()
	r := gin.New()

	// ----- 静态资源 -----
	r.StaticFS("/img", http.FS(subFs(static.Img, "img")))
	r.StaticFS("/css", http.FS(subFs(static.Css, "css")))
	r.StaticFS("/js", http.FS(subFs(static.Js, "js")))
	tmpl := template.Must(template.New("").ParseFS(static.Views, "views/*.html"))
	r.SetHTMLTemplate(tmpl)
	// ----- 静态资源 -----

	r.Use(gin.LoggerWithWriter(log.GetWriter()), gin.Recovery())
	r.Use(func(ctx *gin.Context) {
		// 解析请求地址
		var _host = "http://" + ctx.Request.Host
		if ctx.Request.TLS != nil {
			_host = "https://" + ctx.Request.Host
		}
		_host = config.GetAppUri(_host)

		ctx.Set("HTTP_HOST", _host)
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "一款更易用的USDT收款网关",
			"url":   "https://github.com/v03413/bepusdt",
		})
	})

	// ==== 支付相关=====
	payRoute := r.Group("/pay")
	{
		// 收银台
		payRoute.GET("/checkout-counter/:trade_id", CheckoutCounter)
		// 状态检测
		payRoute.GET("/check-status/:trade_id", CheckStatus)
	}

	// 创建订单
	orderRoute := r.Group("/api/v1/order")
	{
		orderRoute.Use(func(ctx *gin.Context) {
			rawData, err := ctx.GetRawData()
			if err != nil {
				log.Error(err.Error())
				ctx.JSON(400, gin.H{"error": err.Error()})
				ctx.Abort()
			}

			m := make(map[string]any)
			err = json.Unmarshal(rawData, &m)
			if err != nil {
				log.Error(err.Error())
				ctx.JSON(400, gin.H{"error": err.Error()})
				ctx.Abort()
			}

			sign, ok := m["signature"]
			if !ok {
				log.Warn("signature not found", m)
				ctx.JSON(400, gin.H{"error": "signature not found"})
				ctx.Abort()
			}

			if help.GenerateSignature(m, config.GetAuthToken()) != sign {
				log.Warn("签名错误", m)
				ctx.JSON(400, gin.H{"error": "签名错误"})
				ctx.Abort()
			}

			ctx.Set("data", m)
		})

		orderRoute.POST("/create-transaction", CreateTransaction)
	}

	log.Info("WEB尝试启动 Listen: ", listen)
	go func() {
		err := r.Run(listen)
		if err != nil {

			log.Error("Web启动失败", err)
		}
	}()
}

func subFs(src fs.FS, dir string) fs.FS {
	subFS, _ := fs.Sub(src, dir)

	return subFS
}
