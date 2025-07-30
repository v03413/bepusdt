package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func arbitrumInit() {
	ctx := context.Background()
	arb := evm{
		Type:     conf.Arbitrum,
		Endpoint: conf.GetArbitrumRpcEndpoint(),
		Block: block{
			InitStartOffset: -600,
			ConfirmedOffset: 40,
		},
		blockScanQueue: chanx.NewUnboundedChan[[]int64](ctx, 30),
	}

	register(task{ctx: ctx, callback: arb.blockDispatch})
	register(task{ctx: ctx, callback: arb.blockRoll, duration: time.Second * 3})
}
