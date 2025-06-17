package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func xlayerInit() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetXlayerRpcEndpoint(),
			Type:     conf.Xlayer,
			Decimals: decimals{
				Usdt:   conf.UsdtXlayerDecimals,
				Native: -18, // xlayer.okb 小数位数
			},
			Block: block{
				RollDelayOffset: 3,
				ConfirmedOffset: 12,
			},
		}),
		duration: time.Second * 3,
		callback: evmBlockRoll,
	})
}
