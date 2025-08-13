package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func bscInit() {
	ctx := context.Background()
	bsc := evm{
		Network:  conf.Bsc,
		Endpoint: conf.GetBscRpcEndpoint(),
		Block: block{
			InitStartOffset: -400,
			ConfirmedOffset: 15,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{ctx: ctx, callback: bsc.blockDispatch})
	register(task{ctx: ctx, callback: bsc.blockRoll, duration: time.Second * 5})
	register(task{ctx: ctx, callback: bsc.tradeConfirmHandle, duration: time.Second * 5})
}
