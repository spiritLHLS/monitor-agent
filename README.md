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

## Docker

```bash
# 基础使用（提示输入参数）
docker run -it ghcr.io/spiritlhls/ecsagent:latest
```

```bash
# 使用环境变量传递参数
docker run -e token="your_token" \
           -e host="your_host" \
           -e api_port="8080" \
           -e grpc_port="5555" \
           -e task_flag="special" \
           ghcr.io/spiritlhls/ecsagent:latest
```

```bash
# 不设置task_flag（使用默认值）
docker run -e token="your_token" \
           -e host="your_host" \
           -e api_port="8080" \
           -e grpc_port="5555" \
           ghcr.io/spiritlhls/ecsagent:latest
```
