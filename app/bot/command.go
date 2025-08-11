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
	"github.com/v03413/bepusdt/app/task/rate"
	"math"
	"time"
)

const cmdGetId = "id"
const cmdStart = "start"
const cmdState = "state"
const cmdOrder = "order"

const replayAddressText = "🚚 请发送需要添加的钱包地址"
const orderListText = "*现有订单列表，点击可查看详细信息，不同颜色对应着不同支付状态！*\n>🟢收款成功 🔴交易过期 🟡等待支付 ⚪️订单取消\n>🌟按钮内容 订单创建时间 订单号末八位 交易金额"
const orderPageSize = 8

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
			var text = fmt.Sprintf("[✅已启用] %s %s", help.MaskAddress2(wa.Address), wa.TradeType)
			if wa.Status == model.StatusDisable {
				text = fmt.Sprintf("[❌已禁用] %s %s", help.MaskAddress2(wa.Address), wa.TradeType)
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
	var money float64

	var types []string
	model.DB.Model(&model.WalletAddress{}).Distinct("trade_type").Where("status = ?", model.StatusEnable).Pluck("trade_type", &types)

	// 动态统计各类型金额
	typeAmounts := make(map[string]float64)
	for _, t := range types {
		typeAmounts[t] = 0
	}

	for _, o := range rows {
		if o.Status != model.OrderStatusSuccess {

			continue
		}
		succ++
		money += o.Money

		// 只统计启用类型的金额
		if _, exists := typeAmounts[o.TradeType]; exists {
			typeAmounts[o.TradeType] += cast.ToFloat64(o.Amount)
		}
	}

	// 构建基础统计信息
	var text = "```\n"
	text += fmt.Sprintf("🎁今日成功订单：%d\n", succ)
	text += fmt.Sprintf("💎今日总数订单：%d\n", len(rows))
	text += "💰今日收款汇总\n"
	text += fmt.Sprintf(" - %.2f CNY\n", money)

	// 动态显示启用类型的收款汇总
	typeDisplayNames := map[string]string{
		model.OrderTradeTypeTronTrx:      "TRX",
		model.OrderTradeTypeUsdtTrc20:    "USDT.Trc20",
		model.OrderTradeTypeUsdtErc20:    "USDT.Erc20",
		model.OrderTradeTypeUsdtBep20:    "USDT.Bep20",
		model.OrderTradeTypeUsdtAptos:    "USDT.Aptos",
		model.OrderTradeTypeUsdtXlayer:   "USDT.Xlayer",
		model.OrderTradeTypeUsdtSolana:   "USDT.Solana",
		model.OrderTradeTypeUsdtPolygon:  "USDT.Polygon",
		model.OrderTradeTypeUsdtArbitrum: "USDT.Arbitrum",
		model.OrderTradeTypeUsdcTrc20:    "USDC.Trc20",
		model.OrderTradeTypeUsdcErc20:    "USDC.Erc20",
		model.OrderTradeTypeUsdcBep20:    "USDC.Bep20",
		model.OrderTradeTypeUsdcAptos:    "USDC.Aptos",
		model.OrderTradeTypeUsdcXlayer:   "USDC.Xlayer",
		model.OrderTradeTypeUsdcSolana:   "USDC.Solana",
		model.OrderTradeTypeUsdcPolygon:  "USDC.Polygon",
		model.OrderTradeTypeUsdcArbitrum: "USDC.Arbitrum",
		model.OrderTradeTypeUsdcBase:     "USDC.Base",
	}

	for _, t := range types {
		if displayName, exists := typeDisplayNames[t]; exists {
			text += fmt.Sprintf(" - %.2f %s\n", typeAmounts[t], displayName)
		}
	}

	// 动态显示扫块成功数据
	text += "🌟扫块成功数据\n"
	blockchainMap := map[string]string{
		model.OrderTradeTypeUsdtBep20:    conf.Bsc,
		model.OrderTradeTypeTronTrx:      conf.Tron,
		model.OrderTradeTypeUsdtTrc20:    conf.Tron,
		model.OrderTradeTypeUsdtAptos:    conf.Aptos,
		model.OrderTradeTypeUsdtXlayer:   conf.Xlayer,
		model.OrderTradeTypeUsdtSolana:   conf.Solana,
		model.OrderTradeTypeUsdtPolygon:  conf.Polygon,
		model.OrderTradeTypeUsdtArbitrum: conf.Arbitrum,
		model.OrderTradeTypeUsdtErc20:    conf.Ethereum,
		model.OrderTradeTypeUsdcBase:     conf.Base,
	}

	blockchainNames := map[string]string{
		conf.Bsc:      "Bsc",
		conf.Tron:     "Tron",
		conf.Aptos:    "Aptos",
		conf.Xlayer:   "Xlayer",
		conf.Solana:   "Solana",
		conf.Polygon:  "Polygon",
		conf.Arbitrum: "Arbitrum",
		conf.Ethereum: "Ethereum",
		conf.Base:     "Base",
	}

	// 收集需要显示的区块链
	blockchainSet := make(map[string]bool)
	for _, t := range types {
		if blockchain, exists := blockchainMap[t]; exists {
			blockchainSet[blockchain] = true
		}
	}

	// 将区块链转换为切片并按名字长度排序
	var blockchains []string
	for blockchain := range blockchainSet {
		blockchains = append(blockchains, blockchain)
	}

	// 按区块链名字长度排序，名字越长排越后
	for i := 0; i < len(blockchains)-1; i++ {
		for j := 0; j < len(blockchains)-1-i; j++ {
			name1 := blockchainNames[blockchains[j]]
			name2 := blockchainNames[blockchains[j+1]]
			if len(name1) > len(name2) {
				blockchains[j], blockchains[j+1] = blockchains[j+1], blockchains[j]
			}
		}
	}

	// 按排序后的顺序显示区块链数据
	for _, blockchain := range blockchains {
		text += fmt.Sprintf(" - %s %s\n", blockchainNames[blockchain], conf.GetBlockSuccRate(blockchain))
	}

	text += "-----------------------\n"
	text += fmt.Sprintf("🪧基准汇率(TRX)：%v\n", cast.ToString(rate.GetOkxTrxRawRate()))
	text += fmt.Sprintf("🪧基准汇率(USDT)：%v\n", cast.ToString(rate.GetOkxUsdtRawRate()))
	text += fmt.Sprintf("🪧基准汇率(USDC)：%v\n", cast.ToString(rate.GetOkxUsdcRawRate()))
	text += fmt.Sprintf("✅订单汇率(TRX)：%v\n", cast.ToString(rate.GetTrxCalcRate()))
	text += fmt.Sprintf("✅订单汇率(USDT)：%v\n", cast.ToString(rate.GetUsdtCalcRate()))
	text += fmt.Sprintf("✅订单汇率(USDC)：%v\n", cast.ToString(rate.GetUsdcCalcRate()))
	text += "-----------------------\n"
	text += "```\n"
	text += ">基准汇率：来源于交易所的原始数据。\n"
	text += ">订单汇率：订单创建过程中实际使用的汇率。\n"
	text += ">扫块成功数据：如果该值过低，说明您的服务器与区块链网络连接不稳定，请尝试更换区块节点。"

	SendMessage(&bot.SendMessageParams{
		ChatID:    u.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

func cmdOrderHandle(ctx context.Context, b *bot.Bot, u *models.Update) {
	buttons := buildOrderListWithNavigation(1)
	if buttons == nil {
		SendMessage(&bot.SendMessageParams{
			ChatID:    u.Message.Chat.ID,
			Text:      "*订单列表暂时为空！*",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	SendMessage(&bot.SendMessageParams{
		ChatID:      u.Message.Chat.ID,
		Text:        orderListText,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
}

func buildOrderListWithNavigation(page int) [][]models.InlineKeyboardButton {
	buttons, total := buildOrderButtons(page, orderPageSize)
	if buttons == nil {
		return nil
	}
	return append(buttons, buildPageNavigation(page, total, orderPageSize)...)
}

func buildOrderButtons(page, size int) ([][]models.InlineKeyboardButton, int) {
	var total int64
	model.DB.Model(&model.TradeOrders{}).Count(&total)
	if total == 0 {
		return nil, 0
	}

	var orders []model.TradeOrders
	model.DB.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&orders)

	buttons := make([][]models.InlineKeyboardButton, 0, len(orders))
	for _, o := range orders {
		orderId := o.OrderId
		if len(orderId) > 8 {
			orderId = orderId[len(orderId)-8:]
		}
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text:         fmt.Sprintf("%s〚%s〛%s 💰%.2f", o.GetStatusEmoji(), o.CreatedAt.Format("1/2 15:04"), orderId, o.Money),
			CallbackData: fmt.Sprintf("%s|%v|%d", cbOrderDetail, o.TradeId, page),
		}})
	}

	return buttons, int(total)
}

func buildPageNavigation(page, total, size int) [][]models.InlineKeyboardButton {
	totalPage := int(math.Ceil(float64(total) / float64(size)))

	prevBtn := models.InlineKeyboardButton{Text: "🏠首页", CallbackData: "-"}
	if page > 1 {
		prevBtn = models.InlineKeyboardButton{Text: "⬅️上一页", CallbackData: fmt.Sprintf("%s|%d", cbOrderList, page-1)}
	}

	nextBtn := models.InlineKeyboardButton{Text: "🔙末页", CallbackData: "-"}
	if page < totalPage {
		nextBtn = models.InlineKeyboardButton{Text: "➡️下一页", CallbackData: fmt.Sprintf("%s|%d", cbOrderList, page+1)}
	}

	return [][]models.InlineKeyboardButton{{
		prevBtn,
		{Text: fmt.Sprintf("📄第[%d/%d]页", page, totalPage), CallbackData: "-"},
		nextBtn,
	}}
}
