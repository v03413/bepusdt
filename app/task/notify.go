package task

import (
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"time"
)

func init() {
	RegisterSchedule(time.Second*3, NotifyStart)
}

func NotifyStart(duration time.Duration) {
	log.Info("回调监控启动.")
	for range time.Tick(duration) {
		tradeOrders, err := model.GetNotifyFailedTradeOrders()
		if err != nil {
			log.Error("待回调订单获取失败", err)

			continue
		}

		for _, order := range tradeOrders {
			var next = help.CalcNextNotifyTime(order.ConfirmedAt, order.NotifyNum)
			if time.Now().Unix() >= next.Unix() {

				go notify.Handle(order)
			}
		}
	}
}
