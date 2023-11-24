package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tidwall/gjson"
	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
	"io"
	"net/http"
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

func cbWalletAction(query *tgbotapi.CallbackQuery, address string) {
	var info = getWalletInfoByAddress(address)
	var msg = tgbotapi.NewMessage(query.Message.Chat.ID, "❌查询失败")
	if info != "" {
		msg.Text = info
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{
					tgbotapi.NewInlineKeyboardButtonURL("📝查看详细信息", "https://tronscan.org/#/address/"+address),
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

func getWalletInfoByAddress(address string) string {
	var url = "https://apilist.tronscanapi.com/api/accountv2?address=" + address
	var client = http.Client{Timeout: time.Second * 5}
	resp, err := client.Get(url)
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
☘️ 查询地址：` + address + `
💰 TRX余额：0.00 TRX
💲 USDT余额：0.00 USDT
📬 交易数量：` + result.Get("totalTransactionCount").String() + `
📈 转账数量：↑ ` + result.Get("transactions_out").String() + ` ↓ ` + result.Get("transactions_in").String() + `
📡 宽带资源：` + fmt.Sprintf("%v", netRemaining) + ` / ` + fmt.Sprintf("%v", netLimit) + ` 
🔋 能量资源：` + result.Get("bandwidth.energyRemaining").String() + ` / ` + result.Get("bandwidth.energyLimit").String() + `
⏰ 创建时间：` + dateCreated.Format(time.DateTime) + `
⏰ 最后活动：` + latestOperationTime.Format(time.DateTime) + `
`

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
