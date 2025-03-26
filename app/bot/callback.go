package bot

import (
	"bytes"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/help"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"gorm.io/gorm"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const cbWallet = "wallet"
const cbAddress = "address"
const cbAddressAdd = "address_add"
const cbAddressEnable = "address_enable"
const cbAddressDisable = "address_disable"
const cbAddressDelete = "address_del"
const cbAddressOtherNotify = "address_other_notify"
const cbOrderDetail = "order_detail"
const cbMarkNotifySucc = "mark_notify_succ"
const dbOrderNotifyRetry = "order_notify_retry"

func cbWalletAction(query *tgbotapi.CallbackQuery, address string) {
	var info = "暂不支持..."
	if strings.HasPrefix(address, "T") {
		info = getTronWalletInfo(address)
	}
	if help.IsValidPolygonAddress(address) {
		info = getPolygonWalletInfo(address)
	}

	var uri = "https://tronscan.org/#/address/" + address
	if help.IsValidPolygonAddress(address) {

		uri = "https://polygonscan.com/address/" + address
	}

	var msg = tgbotapi.NewMessage(query.Message.Chat.ID, "❌查询失败")
	if info != "" {
		msg.Text = info
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonURL("📝查看详细信息", uri),
				},
			},
		}
	}

	DeleteMsg(query.Message.MessageID)
	_, _ = botApi.Send(msg)
}

func cbAddressAddHandle(query *tgbotapi.CallbackQuery) {
	var msg = tgbotapi.NewMessage(query.Message.Chat.ID, replayAddressText)
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true, InputFieldPlaceholder: "输入钱包地址"}

	_, _ = botApi.Send(msg)
}

func cbAddressAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		var otherTextLabel = "✅已启用 非订单交易监控通知"
		if wa.OtherNotify != 1 {
			otherTextLabel = "❌已禁用 非订单交易监控通知"
		}

		EditAndSendMsg(query.Message.MessageID, wa.Address, tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonData("✅启用", cbAddressEnable+"|"+id),
					tgbotapi.NewInlineKeyboardButtonData("❌禁用", cbAddressDisable+"|"+id),
					tgbotapi.NewInlineKeyboardButtonData("⛔️删除", cbAddressDelete+"|"+id),
				},
				{
					tgbotapi.NewInlineKeyboardButtonData(otherTextLabel, cbAddressOtherNotify+"|"+id),
				},
			},
		})
	}
}

func cbAddressEnableAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusEnable)

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressDisableAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 修改地址状态
		wa.SetStatus(model.StatusDisable)

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressDeleteAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		// 删除钱包地址
		wa.Delete()

		// 删除历史消息
		DeleteMsg(query.Message.MessageID)

		// 推送最新状态
		cmdStartHandle()
	}
}

func cbAddressOtherNotifyAction(query *tgbotapi.CallbackQuery, id string) {
	var wa model.WalletAddress
	if model.DB.Where("id = ?", id).First(&wa).Error == nil {
		if wa.OtherNotify == 1 {
			wa.SetOtherNotify(model.OtherNotifyDisable)
		} else {
			wa.SetOtherNotify(model.OtherNotifyEnable)
		}

		DeleteMsg(query.Message.MessageID)

		cmdStartHandle()
	}
}

func cbOrderDetailAction(tradeId string) {
	var o model.TradeOrders

	if model.DB.Where("trade_id = ?", tradeId).First(&o).Error != nil {

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
	var markup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonURL("🌏商户网站", site.String()),
				tgbotapi.NewInlineKeyboardButtonURL("📝交易明细", o.GetTxDetailUrl()),
			},
		},
	}
	if o.NotifyState == model.OrderNotifyStateFail {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅标记回调成功", cbMarkNotifySucc+"|"+o.TradeId),
		})
		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⚡️立刻回调重试", dbOrderNotifyRetry+"|"+o.TradeId),
		})
	}
	var msg = tgbotapi.NewMessage(0, "```"+`
⛵️系统订单：`+o.TradeId+`
📌商户订单：`+o.OrderId+`
📊交易汇率：`+o.TradeRate+`(`+conf.GetUsdtRate()+`)
💲交易数额：`+o.Amount+`
💰交易金额：`+fmt.Sprintf("%.2f", o.Money)+` CNY
💍交易类别：`+strings.ToUpper(o.TradeType)+fmt.Sprintf("(%s)", o.GetTradeChain())+` 
🌏商户网站：`+site.String()+`
🔋收款状态：`+o.GetStatusLabel()+`
🍀回调状态：`+notifyStateLabel+`
💎️收款地址：`+help.MaskAddress(o.Address)+`
🕒创建时间：`+o.CreatedAt.Format(time.DateTime)+`
🕒失效时间：`+o.ExpiredAt.Format(time.DateTime)+`
⚖️️确认时间：`+o.ConfirmedAt.Format(time.DateTime)+`
`+"\n```")
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup

	SendMsg(msg)
}

func cbMarkNotifySuccAction(tradeId string) {

	model.DB.Model(&model.TradeOrders{}).Where("trade_id = ?", tradeId).Update("notify_state", model.OrderNotifyStateSucc)

	var msg = tgbotapi.NewMessage(0, fmt.Sprintf("✅订单（`%s`）回调状态手动标记成功，后续将不会再次回调。", tradeId))

	msg.ParseMode = tgbotapi.ModeMarkdownV2

	SendMsg(msg)
}

func dbOrderNotifyRetryAction(tradeId string) {
	model.DB.Model(&model.TradeOrders{}).Where("trade_id = ?", tradeId).UpdateColumn("notify_num", gorm.Expr("notify_num - ?", 1))

	var msg = tgbotapi.NewMessage(0, fmt.Sprintf("🪧订单（`%s`）即将开始回调重试，稍后可再次查询。", tradeId))

	msg.ParseMode = tgbotapi.ModeMarkdownV2

	SendMsg(msg)
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
	var text = "```" + `
☘️ 查询地址：` + address + `
💰 TRX余额：0.00 TRX
💲 USDT余额：0.00 USDT
📬 交易数量：` + result.Get("totalTransactionCount").String() + `
📈 转账数量：↑ ` + result.Get("transactions_out").String() + ` ↓ ` + result.Get("transactions_in").String() + `
📡 宽带资源：` + fmt.Sprintf("%v", netRemaining) + ` / ` + fmt.Sprintf("%v", netLimit) + ` 
🔋 能量资源：` + result.Get("bandwidth.energyRemaining").String() + ` / ` + result.Get("bandwidth.energyLimit").String() + `
⏰ 创建时间：` + dateCreated.Format(time.DateTime) + `
⏰ 最后活动：` + latestOperationTime.Format(time.DateTime) + "\n```"

	for _, v := range result.Get("withPriceTokens").Array() {
		if v.Get("tokenName").String() == "trx" {
			text = strings.Replace(text, "0.00 TRX", fmt.Sprintf("%.2f TRX", v.Get("balance").Float()/1000000), 1)
		}
		if v.Get("tokenName").String() == "Tether USD" {

			text = strings.Replace(text, "0.00 USDT", fmt.Sprintf("%.2f USDT", v.Get("balance").Float()/1000000), 1)
		}
	}

	return text
}

func getPolygonWalletInfo(address string) string {
	var usdt = polygonBalanceOf("0xc2132d05d31c914a87c6611c10748aeb04b58e8f", address)
	var pol = polygonBalanceOf("0x0000000000000000000000000000000000001010", address)

	return fmt.Sprintf("```"+`
💰POL 余额：%s
💲USDT余额：%s
☘️查询地址：`+address+`
`+"```",
		decimal.NewFromBigInt(pol, -18).Round(4).String(),
		help.Ec(decimal.NewFromBigInt(usdt, -6).String()))
}

func polygonBalanceOf(contract, address string) *big.Int {
	var jsonData = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"eth_call","params":[{"from":"0x0000000000000000000000000000000000000000","data":"0x70a08231000000000000000000000000%s","to":"%s"},"latest"]}`,
		time.Now().Unix(), strings.ToLower(strings.Trim(address, "0x")), strings.ToLower(contract)))
	var client = &http.Client{Timeout: time.Second * 5}
	resp, err := client.Post(conf.GetPolygonRpcEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Warn("Error Post response:", err)

		return big.NewInt(0)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("Error reading response body:", err)

		return big.NewInt(0)
	}

	var data = gjson.ParseBytes(body)
	var result = data.Get("result").String()

	return help.HexStr2Int(result)
}
