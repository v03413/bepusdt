package task

import (
	"context"
	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func ethInit() {
	ctx := context.Background()
	eth := evm{
		Type:     conf.Ethereum,
		Endpoint: conf.GetEthereumRpcEndpoint(),
		Decimals: decimals{
			Usdt:   conf.UsdtEthDecimals,
			Native: -18,
		},
		Block: block{
			InitStartOffset: -100,
			ConfirmedOffset: 12,
		},
		blockScanQueue: chanx.NewUnboundedChan[[]int64](context.Background(), 30),
	}

	register(task{ctx: ctx, callback: eth.blockDispatch})
	register(task{ctx: ctx, callback: eth.blockRoll, duration: time.Second * 12})
}
