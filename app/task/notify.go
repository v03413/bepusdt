package task

import (
	"context"
	"time"

	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/web/notify"
)

func init() {
	register(task{duration: time.Second * 3, callback: notifyRetry})
	register(task{duration: time.Second * 30, callback: notifyRoll})
}

// notifyRetry 回调失败重试
func notifyRetry(context.Context) {
	tradeOrders, err := model.GetNotifyFailedTradeOrders()
	if err != nil {
		log.Error("待回调订单获取失败", err)

		return
	}

	for _, order := range tradeOrders {
		var next = help.CalcNextNotifyTime(order.ConfirmedAt, order.NotifyNum)
		if time.Now().Unix() >= next.Unix() {

			go notify.Handle(order)
		}
	}
}

func notifyRoll(context.Context) {
	for _, o := range model.GetOrderByStatus(model.OrderStatusWaiting) {
		notify.Bepusdt(o)
	}
}
