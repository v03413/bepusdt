# Docker 部署教程

- 确保服务器已经安装`Docker`
- 确保配置文件已准备
  参考 [conf.example.toml](https://raw.githubusercontent.com/v03413/bepusdt/refs/heads/main/conf.example.toml)

## 启动命令

```bash
docker run -d --restart=always \
-p 8080:8080 \
-v [配置文件路径]:/opt/bepusdt/conf.toml \
v03413/bepusdt:latest \
-conf /opt/bepusdt/conf.toml

# 执行后如果一切正常，Telegram 机器人会收到启动信息，并且应该可以通过浏览器访问到服务。  
```

⚠️ 注意：默认数据库文件路径是`var/lib/bepusdt/sqlite.db`，如果需要持久化数据，可先通过配置文件参数`sqlite_path`
指定路径，然后将该路径映射到宿主机。