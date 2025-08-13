package model

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/smallnest/chanx"
	"github.com/v03413/bepusdt/app/conf"
)

const (
	WebhookStatusWait = 0
	WebhookStatusSucc = 1
	WebhookStatusFail = -1
)

const (
	WebhookEventOrderCreate  = "order.create"  // 订单创建
	WebhookEventOrderPaid    = "order.paid"    // 订单支付
	WebhookEventOrderTimeout = "order.timeout" // 订单超时
	WebhookEventOrderCancel  = "order.cancel"  // 订单取消
	WebhookEventOrderFailed  = "order.failed"  // 订单失败
)

var WebhookHandleQueue = chanx.NewUnboundedChan[Webhook](context.Background(), 30)

type Webhook struct {
	ID        int64           `gorm:"column:id;type:INTEGER PRIMARY KEY AUTOINCREMENT;" json:"id"`
	Status    int8            `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	Num       int             `gorm:"column:num;type:int(11);not null;default:0" json:"hook_num"`
	Url       string          `gorm:"column:url;type:varchar(255);not null;default:''" json:"url"`
	Event     string          `gorm:"column:event;type:varchar(64);not null;default:''" json:"event"`
	Data      json.RawMessage `gorm:"column:data;type:json;not null" json:"data"`
	CreatedAt time.Time       `gorm:"autoCreateTime;type:timestamp;not null;comment:创建时间"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime;type:timestamp;not null;comment:更新时间"`
}

func (Webhook) TableName() string {

	return "bep_webhook"
}

func (w Webhook) PostData() string {
	var data = make(map[string]any)

	data["event"] = w.Event
	data["data"] = w.Data

	jsonData, _ := json.Marshal(data)

	return string(jsonData)
}

func (w Webhook) SetStatus(status int8) {
	if status == WebhookStatusSucc {
		DB.Model(&Webhook{}).Where("id = ?", w.ID).Update("status", status)

		return
	}
	if status == WebhookStatusWait {

		return
	}

	w.Num = w.Num + 1
	if w.Num > conf.NotifyMaxRetry {

		w.Status = WebhookStatusFail
	}

	DB.Save(&w)
}

func PushWebhookEvent(event string, data any) {
	go func() {
		var url = conf.GetWebhookUrl()
		if url == "" {

			return
		}

		bytes, _ := json.Marshal(data)

		var w = Webhook{Status: WebhookStatusWait, Url: url, Event: event, Data: bytes}
		if err = DB.Create(&w).Error; err == nil {

			WebhookHandleQueue.In <- w
		}
	}()
}

func ListWaitWebhooks() {
	var webhooks = make([]Webhook, 0)
	DB.Where("status = ?", WebhookStatusWait).Find(&webhooks)

	for _, w := range webhooks {
		var next = w.CreatedAt.Add(time.Minute * time.Duration(math.Pow(2, float64(w.Num))))
		if time.Now().Unix() >= next.Unix() {

			WebhookHandleQueue.In <- w
		}
	}
}
