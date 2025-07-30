package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func bscInit() {
	ctx := context.Background()
	bsc := evm{
		Type:     conf.Bsc,
		Endpoint: conf.GetBscRpcEndpoint(),
		Block: block{
			InitStartOffset: -400,
			ConfirmedOffset: 15,
		},
		blockScanQueue: chanx.NewUnboundedChan[[]int64](ctx, 30),
	}

	register(task{ctx: ctx, callback: bsc.blockDispatch})
	register(task{ctx: ctx, callback: bsc.blockRoll, duration: time.Second * 3})
}
