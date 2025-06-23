package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func bscInit() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetBscRpcEndpoint(),
			Type:     conf.Bsc,
			Decimals: decimals{
				Usdt:   conf.UsdtBscDecimals,
				Native: -18, // bsc.bnb 小数位数
			},
			Block: block{
				InitStartOffset: -400,
				ConfirmedOffset: 15,
			},
		}),
		duration: time.Second * 3,
		callback: evmBlockRoll,
	})
}
