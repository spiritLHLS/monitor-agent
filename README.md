# monitor-agent

## 守护进程安装

```bash
curl -L https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent.sh -o ecsagent.sh && chmod +x ecsagent.sh && bash ecsagent.sh
```

## 服务管理命令

```bash
systemctl status ecsagent.service   # 查看运行状态
systemctl stop ecsagent.service     # 停止服务
systemctl disable ecsagent.service  # 禁用开机自启
systemctl remove ecsagent.service   # 移除服务
```

## 仅测试运行

```bash
rm -rf ecsagent
wget https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent
chmod 777 ecsagent
ls
```