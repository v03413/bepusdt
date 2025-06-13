package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func init() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetPolygonRpcEndpoint(),
			Type:     conf.Polygon,
			Decimals: decimals{
				Usdt:   -6,  // usdt.polygon 小数位数
				Native: -18, // polygon.pol 小数位数
			},
		}),
		duration: time.Second * 3,
		callback: evmBlockRoll,
	})
}
