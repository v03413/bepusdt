# Bepusdt (Better Easy Payment Usdt)

<p align="center">
<img src="./static/img/tether.svg" width="15%" alt="tether">
</p>
<p align="center">
<a href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://img.shields.io/badge/license-GPLV3-blue" alt="license GPLV3"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/Golang-1.22-red" alt="Go version 1.21"></a>
<a href="https://github.com/gin-gonic/gin"><img src="https://img.shields.io/badge/Gin-v1.9-blue" alt="Gin Web Framework v1.9"></a>
<a href="https://github.com/go-telegram-bot-api/telegram-bot-api"><img src="https://img.shields.io/badge/Telegram Bot-v5-lightgrey" alt="Golang Telegram Bot Api-v5"></a>
<a href="https://github.com/v03413/bepusdt"><img src="https://img.shields.io/badge/Release-v1.9.21-green" alt="Release v1.9.21"></a>
</p>

## 🪧 介绍

基本就是对`Epusdt`重新造了一次轮子，移除一些非必要依赖(`Redis MySQL`)，同时加入一些新特性，让个人`USDT.TRC20`
收款更好用、部署更便捷！

## 🎉 新特性

- ✅ 具备`Epusdt`的所有特性，插件兼容无缝替换
- ✅ USDT汇率实时同步交易所，且支持在以此基础上波动
- ✅ 不依赖`MySQL Redis`环境，支持`Docker`部署
- ✅ 支持非订单交易监控通知，钱包余额变动及时通知
- ✅ 机器人支持查询当前实时汇率，计算实际浮动汇率
- ✅ 机器人支持任意地址查询 USDT、TRX余额等信息
- ✅ 订单收款成功和余额变动通知 支持指定群组推送
- ⭕️ 待实现：原生支持易支付对接，不依赖额外插件
- ⭕️ 待实现：底层直接采用区块扫描，不依赖三方API
- ⭕️ 待实现：支持更多监控通知，如TRX交易 能量变动等  

## 🛠 参数配置

Bepusdt 所有参数都是以传递环境变量的方式进行配置，大部分参数含默认值，少量配置即可直接使用！

### 参数列表

| 参数名称                 | 默认值          | 用法说明                                                                                                                                          |
|----------------------|--------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| EXPIRE_TIME          | `600`        | 订单有效期，单位秒                                                                                                                                     |
| USDT_RATE            | 空            | USDT汇率，默认留空则获取Okx交易所的汇率(每分钟同步一次)，支持多种写法，如：`7.4` 表示固定7.4、`～1.02`表示最新汇率上浮2%、`～0.97`表示最新汇率下浮3%、`+0.3`表示最新加0.3、`-0.2`表示最新减0.2，以此类推；如参数错误则使用固定值6.4 |
| AUTH_TOKEN           | `123234`     | 认证Token，对接会用到这个参数                                                                                                                             |
| LISTEN               | `:8080`      | 服务器HTTP监听地址                                                                                                                                   |
| TRADE_IS_CONFIRMED   | `0`          | 是否需要网络确认，禁用可以提高回调速度，启用则可以防止交易失败                                                                                                               |
| APP_URI              | 空            | 应用访问地址，留空则系统自动获取，前端收银台会用到，建议设置，例如：https://token-pay.example.com                                                                               |
| WALLET_ADDRESS       | 空            | 启动时需要添加的钱包地址，多个请用半角符逗号`,`分开；当然，同样也支持通过机器人添加。                                                                                                  |
| TG_BOT_TOKEN         | 无            | Telegram Bot Token，**必须设置**，否则无法使用                                                                                                            |
| TG_BOT_ADMIN_ID      | 无            | Telegram Bot 管理员ID，**必须设置**，否则无法使用                                                                                                            |
| TG_BOT_GROUP_ID      | 无            | Telegram 群组ID，设置之后机器人会将交易消息会推送到此群                                                                                                             |
| TRON_SERVER_API      | `TRON_SCAN`  | 可选`TRON_SCAN`,`TRON_GRID`，推荐`TRON_GRID`和`TRON_GRID_API_KEY`搭配使用，*更准更强更及时*                                                                     |
| TRON_SCAN_API_KEY    | 无            | TRONSCAN API KEY，如果收款地址较多推荐设置，可避免被官方QOS                                                                                                       |
| TRON_GRID_API_KEY    | 无            | TRONGRID API KEY，如果收款地址较多推荐设置，可避免被官方QOS                                                                                                       |
| PAYMENT_AMOUNT_RANGE | `0.01,99999` | 支付监控的允许数额范围(闭区间)，设置合理数值可避免一些诱导式诈骗交易提醒                                                                                                         |

**Ps：所以综上所述，必须设置的参数有`TG_BOT_TOKEN TG_BOT_ADMIN_ID`，否则无法使用！**

## 🚀 安装部署

- [Docker 安装教程（强烈推荐🔥）](./docs/docker.md)
- [https 配置教程](./docs/ssl.md)
- [Linux 手动安装教程](./docs/systemd.md)
- [Linux 时钟同步配置](./docs/systemd-timesyncd.md)
- [彩虹易支付对接教程](./docs/epay.md)

## 🤔 常见问题

### 如何获取参数 TG_BOT_ADMIN_ID

Telegram 搜索`@userinfobot`机器人并启用，返回的ID就是`TG_BOT_ADMIN_ID`

### 如何申请`TronScan`和`TronGrid`的ApiKey

目前[TronScan](https://tronscan.org/)和[TronGrid](https://www.trongrid.io/)
都可以通过邮箱注册，登录之后在用户中心创建一个ApiKey即可；默认免费套餐都是每天10W请求，对于个人收款绰绰有余。  
**❗️最近发现TronScan接口不稳定且数据不及时，可以有条件的话都推荐使用TronGrid。**

## ⚠️ 特别注意

- 订单交易强依赖时间，请确保服务器时间准确性，否则可能导致订单异常！
- 部分功能依赖网络，请确保服务器网络纯洁性，否则可能导致功能异常！

## 🙏 感谢

- https://github.com/assimon/epusdt

## 📢 声明

- 本项目仅供个人学习研究使用，任何人或组织在使用过程中请符合当地的法律法规，否则产生的任何后果责任自负。

## 🌟 Stargazers over time

[![Stargazers over time](https://starchart.cc/v03413/bepusdt.svg)](https://starchart.cc/v03413/bepusdt)
