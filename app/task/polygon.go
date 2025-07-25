package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func polygonInit() {
	ctx := context.Background()
	pol := evm{
		Type:     conf.Polygon,
		Endpoint: conf.GetPolygonRpcEndpoint(),
		Decimals: decimals{
			Usdt:   conf.UsdtPolygonDecimals,
			Native: -18, // polygon.pol 小数位数
		},
		Block: block{
			InitStartOffset: -600,
			ConfirmedOffset: 40,
		},
		blockScanQueue: chanx.NewUnboundedChan[[]int64](context.Background(), 30),
	}

	register(task{ctx: ctx, callback: pol.blockDispatch})
	register(task{ctx: ctx, callback: pol.blockRoll, duration: time.Second * 3})
}
