package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func baseInit() {
	ctx := context.Background()
	base := evm{
		Network:  conf.Base,
		Endpoint: conf.GetBaseRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			ConfirmedOffset: 40,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{ctx: ctx, callback: base.blockDispatch})
	register(task{ctx: ctx, callback: base.blockRoll, duration: time.Second * 5})
	register(task{ctx: ctx, callback: base.tradeConfirmHandle, duration: time.Second * 5})
}
