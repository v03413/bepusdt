# 独角数卡对接教程

由于新版`BEpusdt`开始支持不同区块链的USDT、超时回调等新特性，独角自带的`Epusdt`插件已经不再适配，无法直接使用，必须对原有插件进行替换。

- ‼️重要的事情说三遍，独角自带的`Epusdt`插件无法直接使用，使用必出问题！
- ‼️重要的事情说三遍，独角自带的`Epusdt`插件无法直接使用，使用必出问题！
- ‼️重要的事情说三遍，独角自带的`Epusdt`插件无法直接使用，使用必出问题！

因此我对独角数卡进行了适配调整，并对原项目发起了[PR](https://github.com/assimon/dujiaoka/pull/424)，只能希望独角能尽快合并吧。

---

当你看到这篇文章时，我不确定PR是否已经合并，所以你最好下载此文件 [EpusdtController.php](https://raw.githubusercontent.com/v03413/dujiaoka/refs/heads/master/app/Http/Controllers/Pay/EpusdtController.php)
然后在你安装好的独角数卡网站目录`app/Http/Controllers/Pay/`替换原有的同名文件。

确保`BEpusdt`已经安装成功，独角数卡后台支付对接按照以下图片格式填写：
> 独角后台 -> 支付配置 -> 新增
>
![独角数卡](./dujiaoka/1.png)

**重点说下几个参数：**

- 商户ID：搭建`BEpusdt`时候的参数`AUTH_TOKEN`
- 支付标识：即收款交易类型，可选`tron.trx usdt.trc20 usdt.erc20`
  等，[可选列表参考](https://github.com/v03413/BEpusdt/blob/62758f24689a81853a215d122a6012bac8364a82/app/model/orders.go#L25:L30)！
- 商户密钥：固定格式为`https://token.pay.com/api/v1/order/create-transaction` ，域名请自行替换
- 支付处理路由：固定填写`pay/epusdt`

其它参数按照实际情况填写保存，即可完成对接。