package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func init() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetXlayerRpcEndpoint(),
			Type:     conf.Xlayer,
			Decimals: decimals{
				Usdt:   -6,  // usdt.xlayer 小数位数
				Native: -18, // xlayer.okb 小数位数
			},
		}),
		duration: time.Second,
		callback: evmBlockRoll,
	})
}
