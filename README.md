**❗️特别声明：本项目乃作者研究学习的开源项目，不提供任何形式的收费服务(谨防诈骗)；使用过程中遇见问题请提`issue`
或群里交流，针对信息完整的问题优先回复，作者很~~忙~~(懒)开源项目，请自重！**

---  

# BEpusdt (Better Easy Payment Usdt)

<p align="center">
<img src="./static/img/tether.svg" width="15%" alt="tether">
</p>
<p align="center">
<a href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://img.shields.io/badge/license-GPLV3-blue" alt="license GPLV3"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/Golang-1.23-red" alt="Go version 1.23"></a>
<a href="https://github.com/gin-gonic/gin"><img src="https://img.shields.io/badge/Gin-v1.9-blue" alt="Gin Web Framework v1.9"></a>
<a href="https://github.com/go-telegram/bot"><img src="https://img.shields.io/badge/Go_Telegram_Bot-v1.15-blue" alt="Golang Telegram Bot"></a>
<a href="https://github.com/v03413/bepusdt"><img src="https://img.shields.io/github/v/release/v03413/bepusdt" alt="Release v1.19.1"></a>
</p>

## 🪧 介绍

基本就是对`Epusdt`重新造了一次轮子，移除一些非必要依赖(`Redis MySQL`)，同时加入一些新特性，让个人`USDT.TRC20`
收款更好用、部署更便捷！

## 🎉 新特性

### 🌟 目前支持收款类型

🔥 主流网络：TRON(`trx` `usdt.trc20`) Ethereum(`usdt.erc20`) BSC(`usdt.bep20`)   
⚡ 其他网络：Polygon(`usdt.polygon`) X-Layer(`usdt.xlayer`) Solana(`usdt.solana`) Aptos(`usdt.aptos`)

- ✅ 完全兼容 `Epusdt`，插件可无缝替换
- ✅ 实时同步 USDT 汇率，支持自定义浮动
- ✅ 无`MySQL Redis`依赖，支持`Docker`部署
- ✅ 支持非订单交易监控，钱包余额变动通知
- ✅ 支持自定义 USDT 支付精度与递增颗粒度
- ✅ 底层区块扫描，无第三方 API，秒级响应
- ✅ 支持波场能量代理与能量回收监控通知
- ✅ 原生支持易支付对接，无三方插件依赖
- ✅ 支持创建订单时 自定义超时 钱包地址
- ✅ 完整支持`等待支付 支付超时`回调
- ✅ 支持事件`WebHook` 方便多方协同

## 🚀 安装部署

- 安装：[Docker](./docs/docker.md)・[Linux](./docs/systemd.md)・[1Panel](./docs/1panel/README.md)・[宝塔](./docs/bt_panel/README.md)
- 对接：[独角数卡](./docs/dujiaoka.md)・[异次元](./docs/acg-faka.md)・[萌次元](./docs/mcy-shop.md)・[彩虹易支付](https://github.com/v03413/Epay-BEpusdt)
- 开发：[API对接](./docs/api.md)・[订单回调](./docs/notify-epusdt.md)・[Webhook 事件](./docs/webhook.md)
- 其它：[https 配置](./docs/ssl.md)・[Linux 时钟同步](./docs/systemd-timesyncd.md)・[前端模板修改](./docs/template/README.md)

## 🖼 功能截图

![screenshot](./docs/images/screenshot.png)

## 🤔 常见问题

### 如何获取参数 TG_BOT_ADMIN_ID

Telegram 搜索加入群`@BEpusdtChat`，随后发送命令`/info`，返回的`ID`就是`TG_BOT_ADMIN_ID`

## ⚠️ 特别注意

- 订单交易强依赖时间，请确保服务器时间准确性，否则可能导致订单异常！
- 部分功能依赖网络，请确保服务器网络纯洁性，否则可能导致功能异常！

## 🏝️ 交流反馈

- Telegram 群组：[https://t.me/BEpusdtChat](https://t.me/BEpusdtChat)
- Telegram 频道：[https://t.me/BEpusdtChannel](https://t.me/BEpusdtChannel)

## 🙏 感谢

- https://github.com/assimon/epusdt

## 📢 声明

- 本项目仅供个人学习研究使用，任何人或组织在使用过程中请符合当地的法律法规，否则产生的任何后果责任自负。

## 🌟 Stargazers over time

[![Stargazers over time](https://starchart.cc/v03413/bepusdt.svg)](https://starchart.cc/v03413/bepusdt)
