package bot

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/go-cache"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const cbWallet = "wallet"
const cbAddress = "address_act"
const cbAddressAdd = "address_add"
const cbAddressType = "address_type"
const cbAddressEnable = "address_enable"
const cbAddressDisable = "address_disable"
const cbAddressDelete = "address_del"
const cbAddressBack = "address_back"
const cbAddressOtherNotify = "address_other_notify"
const cbOrderDetail = "order_detail"
const cbMarkNotifySucc = "mark_notify_succ"
const dbOrderNotifyRetry = "order_notify_retry"

func cbWalletAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var address = ctx.Value("args").([]string)[1]

	var text = bot.EscapeMarkdownUnescaped("暂不支持...")
	if help.IsValidTronAddress(address) {
		text = getTronWalletInfo(address)
	}

	var params = bot.SendMessageParams{ChatID: u.CallbackQuery.Message.Message.Chat.ID, ParseMode: models.ParseModeMarkdown}
	if text != "" {
		params.Text = text
	}

	DeleteMessage(ctx, b, &bot.DeleteMessageParams{
		ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
		MessageID: u.CallbackQuery.Message.Message.ID,
	})
	SendMessage(&params)
}

func cbAddressAddAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var tradeType = ctx.Value("args").([]string)[1]
	var k = fmt.Sprintf("%s_%d_trade_type", cbAddressAdd, u.CallbackQuery.Message.Message.Chat.ID)

	cache.Set(k, tradeType, -1)

	SendMessage(&bot.SendMessageParams{
		Text:   replayAddressText,
		ChatID: u.CallbackQuery.Message.Message.Chat.ID,
		ReplyMarkup: &models.ForceReply{
			ForceReply:            true,
			Selective:             true,
			InputFieldPlaceholder: fmt.Sprintf("钱包地址(%s)", tradeType),
		},
	})
}

func cbAddressTypeAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var btn [][]models.InlineKeyboardButton
	for _, v := range model.SupportTradeTypes {
		btn = append(btn, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("💎 %s", strings.ToUpper(v)),
				CallbackData: fmt.Sprintf("%s|%s", cbAddressAdd, v),
			},
		})
	}

	SendMessage(&bot.SendMessageParams{
		Text:        "*🏝️ 请选择添加的钱包地址类型：*",
		ChatID:      u.CallbackQuery.Message.Message.Chat.ID,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: btn},
	})
}

func cbAddressDelAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var id = ctx.Value("args").([]string)[1]
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 删除钱包地址
		wa.Delete()

		// 删除历史消息
		DeleteMessage(ctx, b, &bot.DeleteMessageParams{
			ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.Message.ID,
		})

		// 推送最新状态
		cmdStartHandle(ctx, b, u)
	}
}

func cbAddressAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var id = ctx.Value("args").([]string)[1]

	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		var otherTextLabel = "🟢已启用 非订单交易监控通知"
		if wa.OtherNotify != 1 {
			otherTextLabel = "🔴已禁用 非订单交易监控通知"
		}

		var text = fmt.Sprintf("> %s", wa.Address)
		if help.IsValidTronAddress(wa.Address) {
			text = getTronWalletInfo(wa.Address)
		}
		if help.IsValidEvmAddress(wa.Address) {
			text = getEvmWalletInfo(wa)
		}

		EditMessageText(ctx, b, &bot.EditMessageTextParams{
			ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.Message.ID,
			Text:      text,
			ParseMode: models.ParseModeMarkdown,
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						models.InlineKeyboardButton{Text: "✅启用", CallbackData: cbAddressEnable + "|" + id},
						models.InlineKeyboardButton{Text: "❌禁用", CallbackData: cbAddressDisable + "|" + id},
						models.InlineKeyboardButton{Text: "⛔️删除", CallbackData: cbAddressDelete + "|" + id},
						models.InlineKeyboardButton{Text: "🔙返回", CallbackData: cbAddressBack + "|" + cast.ToString(u.CallbackQuery.Message.Message.ID)},
					},
					{
						models.InlineKeyboardButton{Text: otherTextLabel, CallbackData: cbAddressOtherNotify + "|" + id},
					},
				},
			},
		})
	}
}

func cbAddressBackAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	DeleteMessage(ctx, b, &bot.DeleteMessageParams{
		ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
		MessageID: cast.ToInt(ctx.Value("args").([]string)[1]),
	})

	cmdStartHandle(ctx, b, u)
}

func cbAddressEnableAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var id = ctx.Value("args").([]string)[1]
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusEnable)

		// 删除历史消息
		DeleteMessage(ctx, b, &bot.DeleteMessageParams{
			ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.Message.ID,
		})

		// 推送最新状态
		cmdStartHandle(ctx, b, u)
	}
}

func cbAddressDisableAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var id = ctx.Value("args").([]string)[1]
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusDisable)

		// 删除历史消息
		DeleteMessage(ctx, b, &bot.DeleteMessageParams{
			ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.Message.ID,
		})

		// 推送最新状态
		cmdStartHandle(ctx, b, u)
	}
}

func cbAddressOtherNotifyAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var id = ctx.Value("args").([]string)[1]
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		if wa.OtherNotify == 1 {
			wa.SetOtherNotify(model.OtherNotifyDisable)
		} else {
			wa.SetOtherNotify(model.OtherNotifyEnable)
		}

		DeleteMessage(ctx, b, &bot.DeleteMessageParams{
			ChatID:    u.CallbackQuery.Message.Message.Chat.ID,
			MessageID: u.CallbackQuery.Message.Message.ID,
		})

		cmdStartHandle(ctx, b, u)
	}
}

func cbOrderDetailAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var args = ctx.Value("args").([]string)

	var o model.TradeOrders

	if model.DB.Where("trade_id = ?", args[1]).First(&o).Error != nil {

		return
	}

	var urlInfo, er2 = url.Parse(o.NotifyUrl)
	if er2 != nil {
		log.Error("商户网站地址解析错误：" + er2.Error())

		return
	}

	var notifyStateLabel = "✅回调成功"
	if o.NotifyState != model.OrderNotifyStateSucc {
		notifyStateLabel = "❌回调失败"
	}
	if model.OrderStatusWaiting == o.Status {
		notifyStateLabel = o.GetStatusLabel()
	}
	if model.OrderStatusExpired == o.Status {
		notifyStateLabel = "🈚️没有回调"
	}

	var site = &url.URL{Scheme: urlInfo.Scheme, Host: urlInfo.Host}
	var markup = models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				models.InlineKeyboardButton{Text: "🌏商户网站", URL: site.String()},
				models.InlineKeyboardButton{Text: "📝交易明细", URL: o.GetDetailUrl()},
			},
		},
	}

	if o.Status == model.OrderStatusSuccess && o.NotifyState == model.OrderNotifyStateFail {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []models.InlineKeyboardButton{
			{Text: "✅标记回调成功", CallbackData: cbMarkNotifySucc + "|" + o.TradeId},
			{Text: "⚡️立刻回调重试", CallbackData: dbOrderNotifyRetry + "|" + o.TradeId},
		})
	}

	var text = "```" + `
	⛵️系统订单：` + o.TradeId + `
	📌商户订单：` + o.OrderId + `
	📊交易汇率：` + o.TradeRate + `(` + conf.GetUsdtRate() + `)
	💲交易数额：` + o.Amount + `
	💰交易金额：` + fmt.Sprintf("%.2f", o.Money) + ` CNY
	💍交易类别：` + strings.ToUpper(o.TradeType) + `
	🌏商户网站：` + site.String() + `
	🔋收款状态：` + o.GetStatusLabel() + `
	🍀回调状态：` + notifyStateLabel + `
	💎️收款地址：` + help.MaskAddress(o.Address) + `
	🕒创建时间：` + o.CreatedAt.Format(time.DateTime) + `
	🕒失效时间：` + o.ExpiredAt.Format(time.DateTime) + `
	⚖️️确认时间：` + o.ConfirmedAt.Format(time.DateTime) + `
	` + "\n```"

	SendMessage(&bot.SendMessageParams{
		ChatID:      conf.BotAdminID(),
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: markup,
	})
}

func cbMarkNotifySuccAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var tradeId = ctx.Value("args").([]string)[1]

	model.DB.Model(&model.TradeOrders{}).Where("trade_id = ?", tradeId).Update("notify_state", model.OrderNotifyStateSucc)

	SendMessage(&bot.SendMessageParams{
		Text:      fmt.Sprintf("✅订单（`%s`）回调手动标记成功，后续将不会再次回调。", tradeId),
		ParseMode: models.ParseModeMarkdown,
	})
}

func dbOrderNotifyRetryAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var tradeId = ctx.Value("args").([]string)[1]

	model.DB.Model(&model.TradeOrders{}).Where("trade_id = ?", tradeId).UpdateColumn("notify_num", gorm.Expr("notify_num - ?", 1))

	SendMessage(&bot.SendMessageParams{
		Text:      fmt.Sprintf("🪧订单（`%s`）即将开始回调重试，稍后可再次查询。", tradeId),
		ParseMode: models.ParseModeMarkdown,
	})
}

func getTronWalletInfo(address string) string {
	var client = http.Client{Timeout: time.Second * 5}
	resp, err := client.Get("https://apilist.tronscanapi.com/api/accountv2?address=" + address)
	if err != nil {
		log.Error("GetWalletInfoByAddress client.Get(url)", err)

		return ""
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error("GetWalletInfoByAddress resp.StatusCode != 200", resp.StatusCode, err)

		return ""
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("GetWalletInfoByAddress io.ReadAll(resp.Body)", err)

		return ""
	}
	result := gjson.ParseBytes(all)

	var dateCreated = time.UnixMilli(result.Get("date_created").Int())
	var latestOperationTime = time.UnixMilli(result.Get("latest_operation_time").Int())
	var netRemaining = result.Get("bandwidth.netRemaining").Int() + result.Get("bandwidth.freeNetRemaining").Int()
	var netLimit = result.Get("bandwidth.netLimit").Int() + result.Get("bandwidth.freeNetLimit").Int()
	var text = `
>💰 TRX余额：0.00 TRX
>💲 USDT余额：0.00 USDT
>📬 交易数量：` + result.Get("totalTransactionCount").String() + `
>📈 转账数量：↑ ` + result.Get("transactions_out").String() + ` ↓ ` + result.Get("transactions_in").String() + `
>📡 宽带资源：` + fmt.Sprintf("%v", netRemaining) + ` / ` + fmt.Sprintf("%v", netLimit) + ` 
>🔋 能量资源：` + result.Get("bandwidth.energyRemaining").String() + ` / ` + result.Get("bandwidth.energyLimit").String() + `
>⏰ 创建时间：` + help.Ec(dateCreated.Format(time.DateTime)) + `
>⏰ 最后活动：` + help.Ec(latestOperationTime.Format(time.DateTime)) + `
>☘️ 查询地址：` + address

	for _, v := range result.Get("withPriceTokens").Array() {
		if v.Get("tokenName").String() == "trx" {
			text = strings.Replace(text, "0.00 TRX", help.Ec(fmt.Sprintf("%.2f TRX", v.Get("balance").Float()/1000000)), 1)
		}
		if v.Get("tokenName").String() == "Tether USD" {
			text = strings.Replace(text, "0.00 USDT", help.Ec(fmt.Sprintf("%.2f USDT", v.Get("balance").Float()/1000000)), 1)
		}
	}

	return text
}

func getEvmWalletInfo(wa model.WalletAddress) string {

	return fmt.Sprintf(">💲余额：%s\\(%s\\)\n>☘️地址：`%s`", help.Ec(evmUSDTBalanceOf(wa)), help.Ec(wa.TradeType), wa.Address)
}

func evmUSDTBalanceOf(wa model.WalletAddress) string {
	var jsonData = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"from":"0x0000000000000000000000000000000000000000","data":"0x70a08231000000000000000000000000%s","to":"%s"},"latest"]}`,
		time.Now().Unix(), strings.ToLower(strings.Trim(wa.Address, "0x")), strings.ToLower(wa.GetUsdtContract())))
	var client = &http.Client{Timeout: time.Second * 5}
	resp, err := client.Post(wa.GetEvmRpcEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Warn("Error Post response:", err)

		return "0.00"
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("Error reading response body:", err)

		return "0.00"
	}

	var data = gjson.ParseBytes(body)
	var result = data.Get("result").String()

	return decimal.NewFromBigInt(help.HexStr2Int(result), wa.GetUsdtDecimals()).String()
}
