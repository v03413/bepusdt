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
		}),
		duration: time.Second * 12,
		callback: evmBlockRoll,
	})
}
