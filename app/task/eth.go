package task

import (
	"context"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

func ethInit() {
	ctx := context.Background()
	eth := evm{
		Network:  conf.Ethereum,
		Endpoint: conf.GetEthereumRpcEndpoint(),
		Block: block{
			InitStartOffset: -100,
			ConfirmedOffset: 12,
		},
		blockScanQueue: chanx.NewUnboundedChan[evmBlock](ctx, 30),
	}

	register(task{callback: eth.blockDispatch})
	register(task{callback: eth.blockRoll, duration: time.Second * 12})
	register(task{callback: eth.tradeConfirmHandle, duration: time.Second * 5})
}
