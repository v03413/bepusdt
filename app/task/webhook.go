package task

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

func init() {
	register(task{callback: webhookRoll})
}

func webhookRoll(ctx context.Context) {
	var w model.Webhook
	var ticker = time.NewTicker(time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			model.ListWaitWebhooks()
		case w = <-model.WebhookHandleQueue.Out:
			go webhookHandle(w)
		}
	}
}

func webhookHandle(w model.Webhook) {
	var req, err = http.NewRequest("POST", w.Url, strings.NewReader(w.PostData()))
	if err != nil {

		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Powered-By", "https://github.com/v03413/bepusdt")
	req.Header.Set("User-Agent", "BEpusdt/"+app.Version)
	resp, err := client.Do(req)
	if err != nil {
		log.Warn("Webhook request failed:", err.Error())

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.SetStatus(model.WebhookStatusFail)
		log.Warn("Webhook request failed with status code:", resp.StatusCode)

		return
	}

	w.SetStatus(model.WebhookStatusSucc)

	log.Info("Webhook request success:", w.Event, "to", w.Url)
}
