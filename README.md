# Bepusdt (Better Easy Payment Usdt)

## 🪧 介绍

基本就是对`Epusdt`重新造了一次轮子，移除一些非必要依赖(`Redis MySQL`)，同时加入一些新特性，让个人`USDT.TRC20`
收款更好用、部署更便捷！

## 🎉 新特性

- [x] `Epusdt`插件可直接使用，接口兼容无缝替换
- [x] 对接交易所(`Okx`)，USDT动态汇率，订单实时更新
- [ ] 不依赖`MySQL Redis`环境，支持`Docker`部署

## 🛠 参数配置

Bepusdt 所有参数都是以传递环境变量的方式进行配置，大部分参数含默认值，少量配置即可直接使用！

### 参数列表

- EXPIRE_TIME 默认`600`，订单有效期，单位秒
- USDT_RATE USDT汇率 默认留空则自动获取`Okx`C2C最新出售交易汇率(每分钟同步一次)，支持多种写法，如：`7.4`
  表示固定7.4、`～1.02`表示最新汇率上浮2%、`～0.97`表示最新汇率下浮3%、`+0.3`表示最新加`0.3`、`-0.2`
  表示最新减`0.2`，以此类推；如果最终解析失败则使用固定值`6.4`
- AUTH_TOKEN 默认`123234` 认证TOKEN，自己设置一个合适口令即可
- LISTEN 默认`:8080` HTTP监听地址
- TRADE_IS_CONFIRMED 默认`0`表示不启用 是否需要等待网络确认，不启用可以提高监控速度，启用则可以防止交易失败
- APP_URI 本应用的访问地址，留空则系统自动获取，前端收银台会用到，建议设置，例如：`https://token-pay.example.com`
- TG_BOT_TOKEN Telegram Bot Token，必须设置，否则无法使用
- TG_BOT_ADMIN_ID Telegram Bot 管理员ID，必须设置，否则无法使用

**Ps：所以综上所述，必须设置的参数有`TG_BOT_TOKEN TG_BOT_ADMIN_ID`，否则无法使用！**

## 🚀 安装部署

### Docker

```shell
# 完善中...
```

### 手动安装

```shell
# 完善中...
```

## 🤔 常见问题

### 如何获取参数 TG_BOT_ADMIN_ID

Telegram 搜索`@userinfobot`机器人并启用，返回的ID就是`TG_BOT_ADMIN_ID`

## ⚠️ 特别注意

- 订单交易强依赖时间，请确保服务器时间准确性，否则可能导致订单异常！
- 部分功能依赖网络，请确保服务器网络纯洁性，否则可能导致功能异常！

## 🙏 感谢

- https://github.com/assimon/epusdt

## 📢 声明

- 本项目仅供个人学习研究使用，任何人或组织在使用过程中请符合当地的法律法规，否则产生的任何后果责任自负。