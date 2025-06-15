**❗️特别声明：本项目乃作者研究学习的开源项目，不提供任何形式的收费服务(谨防诈骗)；使用过程中遇见问题请提`issue`
或群里交流，针对信息完整的问题优先回复，作者很~~忙~~(懒)开源项目，请自重！**

---  

# Bepusdt (Better Easy Payment Usdt)

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

**🌟 目前支持收款类型：trx usdt.trc20 usdt.bep20(币安) usdt.erc20(以太) usdt.polygon usdt.xlayer(OKX)**

- ✅ 完全兼容 `Epusdt`，插件可无缝替换
- ✅ 实时同步 USDT 汇率，支持自定义浮动
- ✅ 无`MySQL Redis`依赖，支持`Docker`部署
- ✅ 支持非订单交易监控，钱包余额变动通知
- ✅ 支持自定义 USDT 支付精度与递增颗粒度
- ✅ 底层区块扫描，无第三方 API，秒级响应
- ✅ 支持波场能量代理与能量回收监控通知
- ✅ 原生支持易支付对接，无需第三方插件
- ✅ 支持创建订单时 自定义钱包地址 超时
- ✅ 订单支持“等待支付”“支付超时”状态回调

## 🚀 安装部署

- [Docker 安装教程（推荐🔥）](./docs/docker.md)
- [异次元发卡对接教程 🌟](./docs/acg-faka.md)
- [萌次元商城系统对接 🌟](./docs/mcy-shop.md)
- [独角数卡对接教程 🌟](./docs/dujiaoka.md)
- [https 配置教程](./docs/ssl.md)
- [Linux 手动安装教程](./docs/systemd.md)
- [Linux 时钟同步配置](./docs/systemd-timesyncd.md)
- [彩虹易支付对接教程](./docs/epay.md)
- [API 对接开发签名算法](./docs/sign.md)
- [对接 回调通知说明](./docs/notify-epusdt.md)

## 🤔 常见问题

### 如何获取参数 TG_BOT_ADMIN_ID

Telegram 搜索加入群`@BEpusdtChat`，随后发送命令`/info`，返回的`ID`就是`TG_BOT_ADMIN_ID`

## ⚠️ 特别注意

- 订单交易强依赖时间，请确保服务器时间准确性，否则可能导致订单异常！
- 部分功能依赖网络，请确保服务器网络纯洁性，否则可能导致功能异常！

## 📚 接口文档

<details>
<summary>创建订单</summary>  

### 请求地址

```http
POST /api/v1/order/create-transaction
```

### 请求数据

```json
{
  "address": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",  // 可根据实际情况传入收款地址，亦可留空
  "trade_type": "usdt.trc20",  // usdt.trc20(默认) tron.trx 可选列表 https://github.com/v03413/bepusdt/blob/16d8df2cc5acc3d41c1c014ecb5370bd97f7e955/app/model/orders.go#L25:L30
  "order_id": "787240927112940881",   // 商户订单编号
  "amount": 28.88,   // 请求支付金额，CNY
  "signature":"123456abcd", // 签名
  "notify_url": "https://example.com/callback",   // 回调地址
  "redirect_url": "https://example.com/callback", // 支付成功跳转地址
  "timeout": 1200 // 超时时间(秒) 最低60；留空则取配置文件 expire_time，还是没有取默认600
}
```

### 响应内容

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "trade_id": "b3d2477c-d945-41da-96b7-f925bbd1b415", // 本地交易ID
    "order_id": "787240927112940881", // 商户订单编号
    "amount": "28.88", // 请求支付金额，CNY
    "actual_amount": "10", // 实际支付数额 usdt or trx
    "token": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // 收款地址
    "expiration_time": 1200, // 订单有效期，秒
    "payment_url": "https://example.com//pay/checkout-counter/b3d2477c-d945-41da-96b7-f925bbd1b415"  // 收银台地址
  },
  "request_id": ""
}

```

</details>

<details>
<summary>取消订单</summary>  

商户端系统可以通过此接口取消订单，取消后，系统将不再监控此订单，同时释放对应金额占用。

### 请求地址

```http
POST /api/v1/order/cancel-transaction
```

### 请求数据

```json
{
  "trade_id": "0TJV0br98YbNTQe7nQ",   // 交易ID
  "signature":"123456abcd" // 签名内容
}
```

### 响应内容

```json
{
  "data": {
    "trade_id": "0TJV0br98YbNTQe7nQ"
  },
  "message": "success",
  "request_id": "",
  "status_code": 200
}
```

</details>

<details>
<summary>回调通知</summary>

```json
{
  "trade_id": "b3d2477c-d945-41da-96b7-f925bbd1b415",
  "order_id": "787240927112940881",
  "amount": 28.88,
  "actual_amount": 10,
  "token": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "block_transaction_id": "12ef6267b42e43959795cf31808d0cc72b3d0a48953ed19c61d4b6665a341d10",
  "signature": "123456abcd",
  "status": 2   //  1:等待支付  2:支付成功  3:支付超时
}
```

</details>

## 🏝️ 交流反馈

- Telegram 频道：[https://t.me/BepusdtChannel](https://t.me/BepusdtChannel)
- Telegram 群组：[https://t.me/BepusdtChat](https://t.me/BepusdtChat)

## 🙏 感谢

- https://github.com/assimon/epusdt

## 📢 声明

- 本项目仅供个人学习研究使用，任何人或组织在使用过程中请符合当地的法律法规，否则产生的任何后果责任自负。

## 🌟 Stargazers over time

[![Stargazers over time](https://starchart.cc/v03413/bepusdt.svg)](https://starchart.cc/v03413/bepusdt)
