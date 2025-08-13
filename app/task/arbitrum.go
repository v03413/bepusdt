package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func arbitrumInit() {
	ctx := context.Background()
	arb := evm{
		Network:  conf.Arbitrum,
		Endpoint: conf.GetArbitrumRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			ConfirmedOffset: 40,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{ctx: ctx, callback: arb.blockDispatch})
	register(task{ctx: ctx, callback: arb.blockRoll, duration: time.Second * 5})
	register(task{ctx: ctx, callback: arb.tradeConfirmHandle, duration: time.Second * 5})
}
