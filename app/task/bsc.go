package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func init() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetBscRpcEndpoint(),
			Type:     conf.Bsc,
			Decimals: decimals{
				Usdt:   -18, // usdt.bep20 小数位数
				Native: -18, // bsc.bnb 小数位数
			},
			Block: block{
				ConfirmedOffset: numConfirmedSub,
			},
		}),
		duration: time.Second * 3,
		callback: evmBlockRoll,
	})
}
