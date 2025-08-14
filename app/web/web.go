package web

import (
	"context"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/static"
)

func Start(ctx context.Context) {
	gin.SetMode(gin.ReleaseMode)

	var listen = conf.GetListen()
	var engine = loadStatic(gin.New())

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
		payGrp.GET("/checkout-counter/:trade_id", checkoutCounter)
		payGrp.GET("/check-status/:trade_id", checkStatus)
	}

	orderGrp := engine.Group("/api/v1/order")
	{
		orderGrp.Use(signVerify)
		orderGrp.POST("/create-transaction", createTransaction)
		orderGrp.POST("/cancel-transaction", cancelTransaction)
	}

	// 易支付兼容
	{
		engine.POST("/submit.php", epaySubmit)
		engine.GET("/submit.php", epaySubmit)
	}

	var srv = &http.Server{Addr: listen, Handler: engine}

	log.Info("web server start listen", listen)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("web server error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdown); err != nil {
			log.Error("Web shutdown Error", err)

			return
		}

		log.Info("web shutdown success.")
	}()
}

// 加载静态资源
func loadStatic(engine *gin.Engine) *gin.Engine {
	var staticPath = conf.GetStaticPath()
	if staticPath != "" {
		engine.Static("/img", conf.GetStaticPath()+"/img")
		engine.Static("/css", conf.GetStaticPath()+"/css")
		engine.Static("/js", conf.GetStaticPath()+"/js")
		engine.LoadHTMLGlob(conf.GetStaticPath() + "/views/*")

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
