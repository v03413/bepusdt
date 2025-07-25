package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func arbitrumInit() {
	ctx := context.Background()
	pol := evm{
		Type:     conf.Arbitrum,
		Endpoint: conf.GetArbitrumRpcEndpoint(),
		Decimals: decimals{
			Usdt:   conf.UsdtArbitrumDecimals,
			Native: -18, // arbitrum.arb 小数位数
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
