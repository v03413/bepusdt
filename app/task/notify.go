package task

import (
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"time"
)

func init() {
	RegisterSchedule(time.Second*3, notifyRetry)
	RegisterSchedule(time.Second*30, notifyRoll)
}

// notifyRetry 回调失败重试
func notifyRetry(d time.Duration) {
	for range time.Tick(d) {
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

func notifyRoll(d time.Duration) {
	for range time.Tick(d) {
		for _, o := range model.GetOrderByStatus(model.OrderStatusWaiting) {
			notify.Bepusdt(o)
		}
	}
}
