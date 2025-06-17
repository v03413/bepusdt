# Docker 部署教程

## 准备工作

1. **安装 Docker：** 确保服务器已经安装 Docker，这里不阐述安装细节。
2. **准备配置文件：**
   通过这个链接下载示例配置文件：[conf.example.toml](https://github.com/v03413/bepusdt/blob/main/conf.example.toml)
   下载后，请按照实际需求编辑文件，如果不清楚参数含义请勿修改！
3. **推荐服务器：** 推荐使用新加坡服务器(Debian11+)，性能无硬性要求，主要确保网络通畅；
   推荐尽量知名厂商(例如：Aws Gcp Digitalocean 等)，一方面安全更有保障，其次一些私人VPS总是出现一些奇奇怪怪的问题！

## 部署命令

```bash
docker run -d --restart=unless-stopped \
-p 8080:8080 \
-v [配置文件路径]:/usr/local/bepusdt/conf.toml \
v03413/bepusdt:latest
```

### 说明：

- 请将 `[配置文件路径]` 替换为你的配置文件的实际存放路径。
- `-p 8080:8080` 表示将容器的 8080 端口映射到宿主机的 8080 端口，使得应用可以通过宿主机的端口访问。
- `--restart=unless-stopped` 确保你的容器在遇到问题时可以自动重启。

版本说明：`v03413/bepusdt:latest` 为最新发行版镜像，`v03413/bepusdt:nightly`为每日构建的开发版。

## 验证部署

部署完成后，你的 Telegram 机器人会收到一条通知消息，说明应用已经启动。你可以通过访问 `http://[你的IP]:8080` 来检查应用是否正常运行。

## 数据持久化（可选）

如果你想保存应用数据，确保在配置文件中设置了数据库的路径：

```toml
sqlite_path = "你的数据库文件路径"
```

然后在运行 Docker 命令时，将数据库文件路径映射到宿主机，例如：

```bash
-v 你的数据库文件路径:/var/lib/bepusdt/sqlite.db
```

这样即使容器重新启动或删除，数据也不会丢失。
