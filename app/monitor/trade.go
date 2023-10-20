package monitor

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/config"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/notify"
	"github.com/v03413/bepusdt/app/telegram"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const tronScanApi = "https://apilist.tronscanapi.com/"
const usdtToken = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

func TradeStart() {
	log.Info("交易监控启动.")

	for range time.Tick(time.Second * 5) {
		// 列出所有等待支付的交易订单
		tradeOrders, err := model.GetTradeOrderByStatus(model.OrderStatusWaiting)
		if err != nil {
			log.Error("待支付订单获取失败", err)

			continue
		}

		var _lock = make(map[string]model.TradeOrders) // 当前所有正在等待支付的订单 Lock Key
		var _address = make(map[string]bool)           // 当前所有正在等待支付的订单 收款 Address
		for _, order := range tradeOrders {
			if time.Now().Unix() >= order.ExpiredAt.Unix() { // 订单过期
				err := order.OrderSetExpired()
				if err != nil {
					log.Error("订单过期标记失败：", err, order.OrderId)
				} else {
					log.Info("订单过期：", order.OrderId)
				}

				continue
			}

			_lock[order.Address+order.Amount] = order
			_address[order.Address] = true
		}

		// 遍历所有钱包地址最近的交易记录
		for _toAddress, _ := range _address {
			var params = url.Values{}
			var client = &http.Client{Timeout: time.Second * 5}
			req, err := http.NewRequest("GET", tronScanApi+"api/multi/search", nil)
			if err != nil {
				log.Error("处理请求创建错误", err)

				continue
			}

			var now = time.Now()
			var startTimestamp = now.Add(-time.Hour) // 当前时间向前推 3 小时
			var endTimestamp = now.Add(time.Hour)    // 当前时间向后推 1 小时

			// 整合请求参数
			params.Add("limit", "50")
			params.Add("start", "0")
			params.Add("type", "transfer")
			params.Add("secondType", "20")
			params.Add("start_timestamp", strconv.FormatInt(startTimestamp.UnixMilli(), 10)) // 起始时间
			params.Add("end_timestamp", strconv.FormatInt(endTimestamp.UnixMilli(), 10))     // 截止时间
			params.Add("toAddress", _toAddress)                                              // 接收地址
			params.Add("token", usdtToken)                                                   // USDT 通证
			params.Add("relation", "and")
			req.URL.RawQuery = params.Encode()

			// 请求交易记录
			resp, err := client.Do(req)
			if err != nil {
				log.Error("请求交易记录错误", err)

				continue
			}

			// 获取响应记录
			all, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("读取交易记录错误", err)

				continue
			}

			// 释放响应请求
			_ = resp.Body.Close()

			// 解析响应记录
			result := gjson.ParseBytes(all)
			recentTransferTotal := result.Get("total").Num

			log.Info(fmt.Sprintf("recent transfer total: %s(%v)", _toAddress, recentTransferTotal))
			if recentTransferTotal <= 0 {
				// 没有交易记录

				continue
			}

			// 遍历交易记录
			for _, transfer := range result.Get("data").Array() {
				// 计算交易金额
				var _rawAmount = transfer.Get("amount").Float()
				var _decimalAmount = decimal.NewFromFloat(_rawAmount)
				var _decimalDivisor = decimal.NewFromFloat(1000000)
				var _amount = _decimalAmount.Div(_decimalDivisor).String()

				_order, ok := _lock[_toAddress+_amount]
				if !ok || transfer.Get("contractRet").String() != "SUCCESS" {
					// 订单不存在或交易失败

					continue
				}

				// 判断时间是否有效
				var _createdAt = time.UnixMilli(transfer.Get("date_created").Int())
				if _createdAt.Unix() < _order.CreatedAt.Unix() || _createdAt.Unix() > _order.ExpiredAt.Unix() {
					// 失效交易

					continue
				}

				// 判断交易是否需要等待广播确认
				var _confirmed = transfer.Get("confirmed").Bool()
				var _tradeHash = transfer.Get("hash").String()
				var _tradeIsConfirmed = config.GetTradeConfirmed()
				var _fromAddress = transfer.Get("from_address").String()

				if (_tradeIsConfirmed && _confirmed) || !_tradeIsConfirmed {
					if _order.OrderSetSucc(_fromAddress, _tradeHash, _createdAt) == nil {
						// 通知订单支付成功
						go notify.OrderNotify(_order)

						// TG发送订单信息
						go telegram.SendTradeSuccMsg(_order)
					}
				}
			}
		}
	}
}
