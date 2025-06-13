package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func init() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetXlayerRpcEndpoint(),
			Type:     conf.Xlayer,
		}),
		duration: time.Second,
		callback: evmBlockRoll,
	})
}
