# Docker 部署教程

- 确保`Docker`已经安装成功
- 确保配置文件已准备
  参考 [conf.example.toml](https://github.com/v03413/bepusdt/blob/main/conf.example.toml)

## 启动命令

```bash
docker run -d --restart=unless-stopped \
-p 8080:8080 \
-v [配置文件]:/usr/local/bepusdt/conf.toml \
v03413/bepusdt:latest

# 最新滚动开发版镜像：v03413/bepusdt:beta 有需要可自行替换。
# 安装成功，Telegram 机器人会收到启动信息，并且访问 http://[你的IP]:8080 能正常打开。  
```

⚠️ 注意：默认数据库文件路径是`/var/lib/bepusdt/sqlite.db`，如果需要持久化，可先通过配置文件参数`sqlite_path`
指定路径，然后将该路径映射到宿主机。