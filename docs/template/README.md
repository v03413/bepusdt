# 修改默认模板

BEpusdt 所有的静态文件都存放在 `static` 目录，而模板文件则存放在 `views` 目录下。

当前目录结构：

```
➜  bepusdt git:(main) ✗ tree ./static
./static
├── css
│   ├── layer.css
│   ├── main.min.css
│   └── usdt.min.css
├── embep.go
├── font
├── img
│   ├── cloud.png
│   ├── Detector.js
│   ├── favicon.ico
│   ├── icon-1.png
│   ├── icon-2.png
│   ├── icon-3.png
│   ├── icon-4.png
│   ├── key.png
│   ├── lock_icon_copy.png
│   ├── logo.png
│   ├── puff.svg
│   ├── RequestAnimationFrame.js
│   ├── tether.svg
│   ├── ThreeExtras.js
│   ├── ThreeWebGL.js
│   ├── tick.png
│   ├── tron_trx.svg
│   ├── trx.png
│   └── user_icon_copy.png
├── js
│   ├── clipboard.min.js
│   ├── jquery.min.js
│   ├── jquery.qrcode_1.0.min.js
│   ├── layer.min.js
│   └── usdt.min.js
└── views
    ├── index.html
    ├── tron.trx.html
    ├── usdt.aptos.html
    ├── usdt.bep20.html
    ├── usdt.erc20.html
    ├── usdt.polygon.html
    ├── usdt.solana.html
    ├── usdt.trc20.html
    └── usdt.xlayer.html

```

能够看到，`static/views` 对应了不同支付交易类型的页面模板，模板内容可以自定义，但是模板文件名不能修改，
必须和交易类型保持一致，否则系统无法识别。

当你修改之后，通过配置文件里面的`static_path`参数指定静态文件目录即可，最好是绝对路径；至于其它 css js img 等文件，自行调整。
