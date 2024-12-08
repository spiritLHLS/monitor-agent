#!/usr/bin/env bash
# 2024.12.08

_red() { echo -e "\033[31m\033[01m$@\033[0m"; }
_green() { echo -e "\033[32m\033[01m$@\033[0m"; }
_yellow() { echo -e "\033[33m\033[01m$@\033[0m"; }
_blue() { echo -e "\033[36m\033[01m$@\033[0m"; }
reading() { read -rp "$(_green "$1")" "$2"; }

cd /root >/dev/null 2>&1
if [ ! -d /usr/local/bin ]; then
    mkdir -p /usr/local/bin
fi

while [ "$#" -gt 0 ]; do
    case "$1" in
    -token)
        # 处理 -token 选项
        token="$2"
        shift 2
        ;;
    -host)
        # 处理 -host 选项
        host="$2"
        shift 2
        ;;
    -api-port)
        # 处理 API 端口选项
        api_port="$2"
        shift 2
        ;;
    -grpc-port)
        # 处理 gRPC 端口选项
        grpc_port="$2"
        shift 2
        ;;
    *)
        echo "未知的选项: $1"
        exit 1
        ;;
    esac
done

[ -z $token ] && reading "主控Token：" token
[ -z $host ] && reading "主控IPV4/域名：" host
[ -z $api_port ] && reading "主控API端口：" api_port
[ -z $grpc_port ] && reading "主控gRPC端口：" grpc_port

rm -rf /usr/local/bin/ecsagent
rm -rf /etc/systemd/system/ecsagent.service
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent -o /usr/local/bin/ecsagent
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent.service -o /etc/systemd/system/ecsagent.service
chmod +x /usr/local/bin/ecsagent
chmod +x /etc/systemd/system/ecsagent.service

if [ -f "/etc/systemd/system/ecsagent.service" ]; then
    new_exec_start="ExecStart=/usr/local/bin/ecsagent -token ${token} -host ${host} -grpc-port ${grpc_port} -api-port ${api_port}"
    file_path="/etc/systemd/system/ecsagent.service"
    line_number=6
    sed -i "${line_number}s|.*|${new_exec_start}|" "$file_path"
fi

systemctl daemon-reload
systemctl start ecsagent.service
systemctl enable ecsagent.service