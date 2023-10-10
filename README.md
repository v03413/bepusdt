## Bepusdt (Better Easy Payment Usdt)

## 项目介绍

`Epusdt`
这个项目大家都知道，而我的想法也非常简单，想要一款更简单更易用的USDT收款工具，Docker一键部署直接使用，不依赖`MySQL Redis`
，所以就有了`Bepusdt`这个项目。

本来一开始的想法是基于`Epusdt`仓库开分支，但是去掉`MySQL Redis`的工作量不如直接重新写一个，同时兼容`Epusdt`
的API接口，保证那些对接了`Epusdt`的插件可以无缝切换到`Bepusdt`。