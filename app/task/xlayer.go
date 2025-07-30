package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func xlayerInit() {
	ctx := context.Background()
	xlayer := evm{
		Type:     conf.Xlayer,
		Endpoint: conf.GetXlayerRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			RollDelayOffset: 3,
			ConfirmedOffset: 12,
		},
		blockScanQueue: chanx.NewUnboundedChan[[]int64](ctx, 30),
	}

	register(task{ctx: ctx, callback: xlayer.blockDispatch})
	register(task{ctx: ctx, callback: xlayer.blockRoll, duration: time.Second * 3})
}
