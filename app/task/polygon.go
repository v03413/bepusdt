package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func polygonInit() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetPolygonRpcEndpoint(),
			Type:     conf.Polygon,
			Decimals: decimals{
				Usdt:   conf.UsdtPolygonDecimals,
				Native: -18, // polygon.pol 小数位数
			},
			Block: block{
				ConfirmedOffset: 40,
			},
		}),
		duration: time.Second * 3,
		callback: evmBlockRoll,
	})
}
