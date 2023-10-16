# Linux 手动安装

准备服务器，debian11+，架构目前只对amd64做了测试，其他架构请自行测试；以此执行以下命令：

```bash
cd /tmp

# 下载最新版本
wget -O ./bepusdt.zip https://github.com/v03413/bepusdt/releases/latest/download/bepusdt-linux-amd64.zip

# 解压
unzip ./bepusdt.zip

# 移动安装至系统目录
mv ./bepusdt /usr/local

# 配置软件自启
mv /usr/local/bepusdt/bepusdt.service /etc/systemd/system
systemctl enable bepusdt.service

# 配置软件参数，请根据实际情况修改文件
vi /usr/local/bepusdt/Environment.conf

# 启动软件
systemctl start bepusdt.service

# 查看软件状态（看到 Active: active (running) 即成功启动）
systemctl status bepusdt.service
```