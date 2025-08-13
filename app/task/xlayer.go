package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func xlayerInit() {
	ctx := context.Background()
	xlayer := evm{
		Network:  conf.Xlayer,
		Endpoint: conf.GetXlayerRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			RollDelayOffset: 3,
			ConfirmedOffset: 12,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{ctx: ctx, callback: xlayer.blockDispatch})
	register(task{ctx: ctx, callback: xlayer.blockRoll, duration: time.Second * 3})
	register(task{ctx: ctx, callback: xlayer.tradeConfirmHandle, duration: time.Second * 5})
}
