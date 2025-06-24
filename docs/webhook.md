# Webhook

自版本号`v1.20.6`开始，系统开始支持一个简易的Webhook功能。当系统发生特定事件时，向指定的URL发送HTTP POST请求。  
该功能默认关闭，当配置文件`webhook_url`参数不为空时，系统会自动开启Webhook功能，发生Post请求时：
`Content-Type: application/json`，请求体为JSON格式。

## 事件类型

目前已知事件：https://github.com/v03413/BEpusdt/blob/525f0f407915b89ed7bccd14c84f32d22d389df1/app/model/webhook.go#L19:L22

## 请求数据

```json
{
  "event": "event_type",
  "data": {
    "key1": "value1",
    "key2": "value2"
  }
}
```

## 请求说明

当事件发生时会自动触发一个Post请求，响应状态码必须为`200`，否则会认为失败；失败之后会以`2 4 8 16...`指数间隔分钟数重试，最大重试次数为
`10`次。