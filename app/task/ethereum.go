package task

import (
	"context"
	"github.com/v03413/bepusdt/app/conf"
	"time"
)

func init() {
	register(task{
		ctx: context.WithValue(context.Background(), "cfg", evmCfg{
			Endpoint: conf.GetEthereumRpcEndpoint(),
			Type:     conf.Ethereum,
			Decimals: decimals{
				Usdt:   -6,  // usdt.erc20 小数位数
				Native: -18, // ethereum.eth 小数位数
			},
			Block: block{
				ConfirmedOffset: numConfirmedSub,
			},
		}),
		duration: time.Second * 12,
		callback: evmBlockRoll,
	})
}
