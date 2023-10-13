# Docker安装教程

保证你的服务器已经安装了`Docker`和`Docker Compose`，如果没有请自行安装。

直接执行命令，注意下面的参数需要修改为你自己合适的参数，按照相同的格式进行修改。

```bash
docker run -d --restart=always \
-p 8080:8080 \
-e TG_BOT_TOKEN=6361745888:AAFaX_T9XLe4hvF7vRLf1dvolQcuAkw6888 \
-e TG_BOT_ADMIN_ID=1641035888 \
-e USDT_RATE=~0.98 \
v03413/bepusdt:latest
```

执行成功后，访问`http://你的服务器IP:8080`如果能正常打开，则安装成功，你就可以使用机器人添加钱包地址了。