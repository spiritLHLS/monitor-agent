#!/usr/bin/env bash
# 2025.08.20


_red() { echo -e "\033[31m\033[01m$@\033[0m"; }
_green() { echo -e "\033[32m\033[01m$@\033[0m"; }
_yellow() { echo -e "\033[33m\033[01m$@\033[0m"; }
_blue() { echo -e "\033[36m\033[01m$@\033[0m"; }
reading() { read -rp "$(_green "$1")" "$2"; }

check_service() {
    if systemctl is-active --quiet ecsagent.service; then
        return 0  # 服务正在运行
    else
        return 1  # 服务未运行
    fi
}

cleanup_service() {
    _yellow "检测到现有服务，正在停止并清理..."
    systemctl stop ecsagent.service >/dev/null 2>&1
    systemctl disable ecsagent.service >/dev/null 2>&1
    rm -f /usr/local/bin/ecsagent
    rm -f /etc/systemd/system/ecsagent.service
    systemctl daemon-reload
}

cd /root >/dev/null 2>&1
if [ ! -d /usr/local/bin ]; then
    mkdir -p /usr/local/bin
fi
while [ "$#" -gt 0 ]; do
    case "$1" in
    -token)
        token="$2"
        shift 2
        ;;
    -host)
        host="$2"
        shift 2
        ;;
    -api-port)
        api_port="$2"
        shift 2
        ;;
    -grpc-port)
        grpc_port="$2"
        shift 2
        ;;
    -task-flag)
        task_flag="$2"
        shift 2
        ;;
    *)
        _red "未知的选项: $1"
        exit 1
        ;;
    esac
done
[ -z "$token" ] && reading "主控Token：" token
[ -z "$host" ] && reading "主控IPV4/域名：" host
[ -z "$api_port" ] && reading "主控API端口：" api_port
[ -z "$grpc_port" ] && reading "主控gRPC端口：" grpc_port
if check_service; then
    _yellow "发现现有 ECS Agent 服务正在运行"
    reading "是否重新安装？[y/N] " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        cleanup_service
        _green "已清理原有服务，继续安装..."
    else
        _blue "已取消安装"
        exit 0
    fi
else
    if [ -f "/etc/systemd/system/ecsagent.service" ] || [ -f "/usr/local/bin/ecsagent" ]; then
        cleanup_service
        _green "已清理残留文件，继续安装..."
    fi
fi
_blue "正在下载最新版本..."
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent -o /usr/local/bin/ecsagent
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent.service -o /etc/systemd/system/ecsagent.service
chmod +x /usr/local/bin/ecsagent
chmod +x /etc/systemd/system/ecsagent.service
exec_start="/usr/local/bin/ecsagent -token ${token} -host ${host} -grpc-port ${grpc_port} -api-port ${api_port}"
if [ -n "$task_flag" ]; then
    exec_start="${exec_start} -task-flag ${task_flag}"
fi
if [ -f "/etc/systemd/system/ecsagent.service" ]; then
    new_exec_start="ExecStart=${exec_start}"
    file_path="/etc/systemd/system/ecsagent.service"
    line_number=6
    sed -i "${line_number}s|.*|${new_exec_start}|" "$file_path"
    _green "服务配置已更新"
fi
systemctl daemon-reload
systemctl start ecsagent.service
systemctl enable ecsagent.service
sleep 3
if check_service; then
    _green "ECS Agent 安装成功并已启动！"
    echo
    _blue "当前配置："
    echo "  Token: ${token:0:8}****"
    echo "  Host: $host"
    echo "  gRPC Port: $grpc_port"
    echo "  API Port: $api_port"
    if [ -n "$task_flag" ]; then
        echo "  Task Flag: $task_flag"
    else
        echo "  Task Flag: 默认（普通任务）"
    fi
    echo
    _green "服务状态："
    systemctl status ecsagent.service --no-pager -l
else
    _red "ECS Agent 安装可能存在问题，请检查日志"
    echo
    _yellow "错误日志："
    systemctl status ecsagent.service --no-pager -l
    echo
    _yellow "详细日志："
    journalctl -u ecsagent.service --no-pager -l -n 20
fi
echo