package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

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
const cbOrderList = "order_list"
const cbMarkNotifySucc = "mark_notify_succ"
const cbOrderNotifyRetry = "order_notify_retry"
const cbMarkOrderSucc = "mark_order_succ"

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
	var row []models.InlineKeyboardButton
	var format = func(v string) string {
		var text = fmt.Sprintf("💎 %s", strings.ToUpper(v))
		if strings.Contains(v, "usdt") {
			text = fmt.Sprintf("💚 %s", strings.ToUpper(v))
		}
		if strings.Contains(v, "usdc") {
			text = fmt.Sprintf("💙 %s", strings.ToUpper(v))
		}

		arr := strings.Split(text, ".")
		if len(arr) != 2 {

			return text
		}

		return fmt.Sprintf("%s.%s", arr[0], help.Capitalize(arr[1]))
	}
	for i, v := range model.SupportTradeTypes {
		row = append(row, models.InlineKeyboardButton{
			Text:         format(v),
			CallbackData: fmt.Sprintf("%s|%s", cbAddressAdd, v),
		})
		if (i+1)%2 == 0 || i == len(model.SupportTradeTypes)-1 {
			btn = append(btn, row)
			row = []models.InlineKeyboardButton{}
		}
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

		var text = fmt.Sprintf(">`%s`", wa.Address)
		if help.IsValidTronAddress(wa.Address) {
			text = getTronWalletInfo(wa.Address)
		}
		if help.IsValidEvmAddress(wa.Address) {
			text = getEvmWalletInfo(wa)
		}
		if help.IsValidAptosAddress(wa.Address) {
			text = getAptosWalletInfo(wa)
		}
		if help.IsValidSolanaAddress(wa.Address) {
			text = getSolanaWalletInfo(wa)
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
	args := ctx.Value("args").([]string)
	if len(args) < 2 {

		return
	}

	var order model.TradeOrders
	if err := model.DB.Where("trade_id = ?", args[1]).First(&order).Error; err != nil {

		return
	}

	urlInfo, err := url.Parse(order.NotifyUrl)
	if err != nil {
		log.Error("商户网站地址解析错误：" + err.Error())

		return
	}

	// 确定回调状态标签
	var notifyStateLabel string
	switch {
	case order.Status == model.OrderStatusWaiting:
		notifyStateLabel = order.GetStatusLabel()
	case order.Status == model.OrderStatusExpired:
		notifyStateLabel = "🈚️没有回调"
	case order.NotifyState == model.OrderNotifyStateSucc:
		notifyStateLabel = "✅回调成功"
	default:
		notifyStateLabel = "❌回调失败"
	}

	site := &url.URL{Scheme: urlInfo.Scheme, Host: urlInfo.Host}
	markup := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🌏商户网站", URL: site.String()},
				{Text: "📝交易明细", URL: order.GetDetailUrl()},
			},
		},
	}

	if order.Status == model.OrderStatusSuccess && order.NotifyState == model.OrderNotifyStateFail {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []models.InlineKeyboardButton{
			{Text: "✅标记回调成功", CallbackData: cbMarkNotifySucc + "|" + order.TradeId},
			{Text: "⚡️立刻回调重试", CallbackData: cbOrderNotifyRetry + "|" + order.TradeId},
		})
	}

	if (order.Status == model.OrderStatusExpired || order.Status == model.OrderStatusWaiting) && order.NotifyState == model.OrderNotifyStateFail {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []models.InlineKeyboardButton{
			{Text: "⚠️直接标记已支付（即使未收到款）", CallbackData: cbMarkOrderSucc + "|" + order.TradeId},
		})
	}

	if len(args) == 3 {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []models.InlineKeyboardButton{
			{Text: "📦返回订单列表", CallbackData: fmt.Sprintf("%s|%s", cbOrderList, args[2])},
		})
	}

	text := fmt.Sprintf("```\n"+
		"⛵️系统订单：%s\n"+
		"📌商户订单：%s\n"+
		"📊交易汇率：%s(%s)\n"+
		"💲交易数额：%s\n"+
		"💰交易金额：%.2f CNY\n"+
		"💍交易类别：%s\n"+
		"🌏商户网站：%s\n"+
		"🔋收款状态：%s\n"+
		"🍀回调状态：%s\n"+
		"💎️收款地址：%s\n"+
		"🕒创建时间：%s\n"+
		"🕒失效时间：%s\n"+
		"⚖️️确认时间：%s\n"+
		"```",
		order.TradeId,
		order.OrderId,
		order.TradeRate, conf.GetUsdtRate(),
		order.Amount,
		order.Money,
		strings.ToUpper(order.TradeType),
		site.String(),
		order.GetStatusLabel(),
		notifyStateLabel,
		help.MaskAddress(order.Address),
		order.CreatedAt.Format(time.DateTime),
		order.ExpiredAt.Format(time.DateTime),
		order.ConfirmedAt.Format(time.DateTime))

	EditMessageText(ctx, b, &bot.EditMessageTextParams{
		ChatID:      u.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   u.CallbackQuery.Message.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: markup,
	})
}

func cbOrderListAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	page := cast.ToInt(ctx.Value("args").([]string)[1])
	buttons := buildOrderListWithNavigation(page)

	EditMessageText(ctx, b, &bot.EditMessageTextParams{
		ChatID:      u.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   u.CallbackQuery.Message.Message.ID,
		Text:        orderListText,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
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

func dbMarkOrderSuccAction(ctx context.Context, b *bot.Bot, u *models.Update) {
	var tradeId = ctx.Value("args").([]string)[1]

	model.DB.Model(&model.TradeOrders{}).Where("trade_id = ?", tradeId).UpdateColumn("status", model.OrderStatusSuccess)

	SendMessage(&bot.SendMessageParams{
		Text:      fmt.Sprintf("🪧订单（`%s`）已经标记为收款成功，稍后可再次查询。", tradeId),
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

func getAptosWalletInfo(wa model.WalletAddress) string {

	return fmt.Sprintf(">💲余额：%s\\(%s\\)\n>☘️地址：`%s`", help.Ec(aptTokenBalanceOf(wa)), help.Ec(wa.TradeType), wa.Address)
}

func getSolanaWalletInfo(wa model.WalletAddress) string {

	return fmt.Sprintf(">💲余额：%s\\(%s\\)\n>☘️地址：`%s`", help.Ec(solTokenBalanceOf(wa)), help.Ec(wa.TradeType), wa.Address)
}

func getEvmWalletInfo(wa model.WalletAddress) string {

	return fmt.Sprintf(">💲余额：%s\\(%s\\)\n>☘️地址：`%s`", help.Ec(evmTokenBalanceOf(wa)), help.Ec(wa.TradeType), wa.Address)
}

func solTokenBalanceOf(wa model.WalletAddress) string {
	var jsonData = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getTokenAccountsByOwner","params":["%s",{"mint": "%s"},{"commitment":"finalized","encoding":"jsonParsed"}]}`,
		wa.Address, wa.GetTokenContract()))

	var client = &http.Client{Timeout: time.Second * 5}
	resp, err := client.Post(conf.GetSolanaRpcEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Warn("Error Post response:", err)

		return "0.00"
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error("solTokenBalanceOf resp.StatusCode != 200", resp.StatusCode, err)

		return "0.00"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("solTokenBalanceOf io.ReadAll(resp.Body)", err)

		return "0.00"
	}

	sum := new(big.Int)
	values := gjson.GetBytes(body, "result.value").Array()
	for _, v := range values {
		amountStr := v.Get("account.data.parsed.info.tokenAmount.amount").String()

		if amountStr == "" {
			continue
		}
		if a, ok := new(big.Int).SetString(amountStr, 10); ok {
			sum.Add(sum, a)
		}
	}

	return decimal.NewFromBigInt(sum, wa.GetTokenDecimals()).String()
}

func aptTokenBalanceOf(wa model.WalletAddress) string {
	var client = http.Client{Timeout: time.Second * 5}
	resp, err := client.Get(fmt.Sprintf("%sv1/accounts/%s/balance/%s", conf.GetAptosRpcNode(), wa.Address, strings.ToLower(wa.GetTokenContract())))
	if err != nil {
		log.Error("getAptosWalletInfo client.Get(url)", err)

		return "0.00"
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error("getAptosWalletInfo resp.StatusCode != 200", resp.StatusCode, err)

		return "0.00"
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("getAptosWalletInfo io.ReadAll(resp.Body)", err)

		return "0.00"
	}

	result, _ := new(big.Int).SetString(string(all), 10)

	return decimal.NewFromBigInt(result, wa.GetTokenDecimals()).String()
}

func evmTokenBalanceOf(wa model.WalletAddress) string {
	var jsonData = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"from":"0x0000000000000000000000000000000000000000","data":"0x70a08231000000000000000000000000%s","to":"%s"},"latest"]}`,
		time.Now().Unix(), strings.ToLower(strings.Trim(wa.Address, "0x")), strings.ToLower(wa.GetTokenContract())))
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

	return decimal.NewFromBigInt(help.HexStr2Int(result), wa.GetTokenDecimals()).String()
}
