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

	var listen = config.GetListen()
	var engine = loadStatic(gin.New())

	// Init gin
	{
		engine.Use(gin.LoggerWithWriter(log.GetWriter()), gin.Recovery())
		engine.Use(func(ctx *gin.Context) {
			ctx.Writer.Header().Set("Payment-Gateway", "https://github.com/v03413/bepusdt")
		})
		engine.GET("/", func(c *gin.Context) {
			c.HTML(200, "index.html", gin.H{"title": "一款更易用的USDT收款网关", "url": "https://github.com/v03413/bepusdt"})
		})
	}

	payGrp := engine.Group("/pay")
	{
		// 收银台
		payGrp.GET("/checkout-counter/:trade_id", checkoutCounter)
		// 状态检测
		payGrp.GET("/check-status/:trade_id", checkStatus)
	}

	orderGrp := engine.Group("/api/v1/order")
	{
		orderGrp.Use(func(ctx *gin.Context) {
			rawData, err := ctx.GetRawData()
			if err != nil {
				log.Error(err.Error())
				ctx.JSON(400, gin.H{"error": err.Error()})
				ctx.Abort()

				return
			}

			m := make(map[string]any)
			err = json.Unmarshal(rawData, &m)
			if err != nil {
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

			if help.GenerateSignature(m, config.GetAuthToken()) != sign {
				log.Warn("签名错误", m)
				ctx.JSON(400, gin.H{"error": "签名错误"})
				ctx.Abort()

				return
			}

			ctx.Set("data", m)
		})

		orderGrp.POST("/create-transaction", createTransaction) // 创建订单
		orderGrp.POST("/cancel-transaction", cancelTransaction) // 取消订单
	}

	// 易支付兼容
	engine.POST("/submit.php", epaySubmit)

	log.Info("WEB尝试启动 Listen: ", listen)
	go func() {
		err := engine.Run(listen)
		if err != nil {

			log.Error("Web启动失败", err)
		}
	}()
}

// 加载静态资源
func loadStatic(engine *gin.Engine) *gin.Engine {
	var staticPath = config.GetStaticPath()
	if staticPath != "" {
		engine.Static("/img", config.GetStaticPath()+"/img")
		engine.Static("/css", config.GetStaticPath()+"/css")
		engine.Static("/js", config.GetStaticPath()+"/js")
		engine.LoadHTMLGlob(config.GetStaticPath() + "/views/*")

		return engine
	}

	engine.StaticFS("/img", http.FS(subFs(static.Img, "img")))
	engine.StaticFS("/css", http.FS(subFs(static.Css, "css")))
	engine.StaticFS("/js", http.FS(subFs(static.Js, "js")))
	engine.SetHTMLTemplate(template.Must(template.New("").ParseFS(static.Views, "views/*.html")))

	return engine
}

func subFs(src fs.FS, dir string) fs.FS {
	subFS, _ := fs.Sub(src, dir)

	return subFS
}
