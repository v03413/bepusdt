package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func polygonInit() {
	ctx := context.Background()
	pol := evm{
		Network:  conf.Polygon,
		Endpoint: conf.GetPolygonRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			ConfirmedOffset: 40,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{callback: pol.blockDispatch})
	register(task{callback: pol.blockRoll, duration: time.Second * 5})
	register(task{callback: pol.tradeConfirmHandle, duration: time.Second * 5})
}
