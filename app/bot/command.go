package bot

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/spf13/cast"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/model"
	"github.com/v03413/bepusdt/app/rate"
	"time"
)

const cmdGetId = "id"
const cmdStart = "start"
const cmdState = "state"
const cmdWallet = "wallet"
const cmdOrder = "order"

const replayAddressText = "🚚 请发送需要添加的钱包地址"

func cmdGetIdHandle(ctx context.Context, b *bot.Bot, u *models.Update) {

	SendMessage(&bot.SendMessageParams{
		ChatID:    u.Message.Chat.ID,
		Text:      "您的ID: " + fmt.Sprintf("`%v`（点击复制）", u.Message.Chat.ID),
		ParseMode: models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: u.Message.ID,
		},
	})
}

func cmdStartHandle(ctx context.Context, b *bot.Bot, u *models.Update) {
	var was []model.WalletAddress
	var btn [][]models.InlineKeyboardButton
	if model.DB.Find(&was).Error == nil {
		for _, wa := range was {
			var text = fmt.Sprintf("[✅已启用] %s", help.MaskAddress(wa.Address))
			if wa.Status == model.StatusDisable {
				text = fmt.Sprintf("[❌已禁用] %s", help.MaskAddress(wa.Address))
			}

			btn = append(btn, []models.InlineKeyboardButton{
				{Text: text, CallbackData: fmt.Sprintf("%s|%v", cbAddress, wa.ID)},
			})

		}
	}

	var chatID any
	if u.Message != nil {
		chatID = u.Message.Chat.ID
	}
	if u.CallbackQuery != nil {
		chatID = u.CallbackQuery.Message.Message.Chat.ID
	}

	btn = append(btn, []models.InlineKeyboardButton{{Text: "👛 收款地址添加", CallbackData: cbAddressType}})

	SendMessage(&bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "🌟点击钱包 按提示进行操作",
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: btn},
	})
}

func cmdStateHandle(ctx context.Context, b *bot.Bot, u *models.Update) {
	var rows []model.TradeOrders
	model.DB.Where("created_at > ?", time.Now().Format(time.DateOnly)).Find(&rows)
	var succ uint64
	var money, trx, uTrc20, uErc20, uBep20, uXlayer, uPol float64
	for _, o := range rows {
		if o.Status != model.OrderStatusSuccess {

			continue
		}

		succ++
		money += o.Money

		var amount = cast.ToFloat64(o.Amount)
		if o.TradeType == model.OrderTradeTypeTronTrx {
			trx += amount
		}
		if o.TradeType == model.OrderTradeTypeUsdtBep20 {
			uBep20 += amount
		}
		if o.TradeType == model.OrderTradeTypeUsdtTrc20 {
			uTrc20 += amount
		}
		if o.TradeType == model.OrderTradeTypeUsdtErc20 {
			uErc20 += amount
		}
		if o.TradeType == model.OrderTradeTypeUsdtPolygon {
			uPol += amount
		}
		if o.TradeType == model.OrderTradeTypeUsdtXlayer {
			uXlayer += amount
		}
	}

	var base = "```" + `
🎁今日成功订单：%d
💎今日总数订单：%d
💰今日收款汇总
	- %.2f CNY
	- %.2f TRX
	- %.2f USDT.Trc20
	- %.2f USDT.Erc20
	- %.2f USDT.Bep20
	- %.2f USDT.Xlayer
	- %.2f USDT.Polygon
🌟扫块成功数据
	- Bsc %s
	- Tron %s
	- Xlayer %s
	- Polygon %s
	- Ethereum %s
-----------------------
🪧基准汇率(TRX)：%v
🪧基准汇率(USDT)：%v
✅订单汇率(TRX)：%v
✅订单汇率(USDT)：%v
-----------------------
` + "```" + `
>基准汇率：来源于交易所的原始数据。
>订单汇率：订单创建过程中实际使用的汇率。
>扫块成功数据：如果该值过低，说明您的服务器与区块链网络连接不稳定，请尝试更换区块节点。
`

	var text = fmt.Sprintf(base,
		succ,
		len(rows),
		money,
		trx,
		uTrc20,
		uErc20,
		uBep20,
		uXlayer,
		uPol,
		conf.GetBlockSuccRate(conf.Bsc),
		conf.GetBlockSuccRate(conf.Tron),
		conf.GetBlockSuccRate(conf.Xlayer),
		conf.GetBlockSuccRate(conf.Polygon),
		conf.GetBlockSuccRate(conf.Ethereum),
		cast.ToString(rate.GetOkxTrxRawRate()),
		cast.ToString(rate.GetOkxUsdtRawRate()),
		cast.ToString(rate.GetTrxCalcRate(conf.DefaultTrxCnyRate)),
		cast.ToString(rate.GetUsdtCalcRate(conf.DefaultUsdtCnyRate)),
	)

	SendMessage(&bot.SendMessageParams{
		ChatID:    u.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

func cmdWalletHandle(ctx context.Context, b *bot.Bot, u *models.Update) {
	var was []model.WalletAddress
	var btn [][]models.InlineKeyboardButton
	if model.DB.Find(&was).Error == nil {
		for _, wa := range was {
			var text = fmt.Sprintf("[✅已启用] %s", help.MaskAddress(wa.Address))
			if wa.Status == model.StatusDisable {
				text = fmt.Sprintf("[❌已禁用] %s", help.MaskAddress(wa.Address))
			}

			btn = append(btn, []models.InlineKeyboardButton{
				{
					Text:         text,
					CallbackData: fmt.Sprintf("%s|%v", cbWallet, wa.Address),
				},
			})
		}
	}

	SendMessage(&bot.SendMessageParams{
		ChatID:      u.Message.Chat.ID,
		Text:        "*\\>\\>请选择需要查询的钱包地址*",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: btn},
	})
}

func cmdOrderHandle(ctx context.Context, b *bot.Bot, u *models.Update) {
	var orders []model.TradeOrders
	var btn [][]models.InlineKeyboardButton
	if model.DB.Order("id desc").Limit(8).Find(&orders).Error == nil {
		for _, o := range orders {
			btn = append(btn, []models.InlineKeyboardButton{
				{
					Text:         fmt.Sprintf("%s %s 💰%.2f", o.GetStatusEmoji(), o.OrderId, o.Money),
					CallbackData: fmt.Sprintf("%s|%v", cbOrderDetail, o.TradeId),
				},
			})
		}
	}

	SendMessage(&bot.SendMessageParams{
		ChatID:      u.Message.Chat.ID,
		Text:        "*下面是最近的8个订单，点击可查看详细信息*\n```\n🟢 收款成功 🔴 交易过期 \n🟡 等待支付 ⚪️ 订单取消\n```",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: btn},
	})
}
