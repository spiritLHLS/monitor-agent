#!/usr/bin/env bash
# 2025.01.16
_red() { echo -e "\033[31m\033[01m$@\033[0m"; }
_green() { echo -e "\033[32m\033[01m$@\033[0m"; }
_yellow() { echo -e "\033[33m\033[01m$@\033[0m"; }
_blue() { echo -e "\033[36m\033[01m$@\033[0m"; }
reading() { read -rp "$(_green "$1")" "$2"; }

# 检查服务状态
check_service() {
    if systemctl is-active --quiet ecsagent.service; then
        return 0  # 服务正在运行
    else
        return 1  # 服务未运行
    fi
}

# 停止并清理现有服务
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

# 处理命令行参数
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
    -use-cf)
        use_cf="$2"
        shift 2
        ;;
    -cf-service)
        cf_service="$2"
        shift 2
        ;;
    *)
        _red "未知的选项: $1"
        exit 1
        ;;
    esac
done

# 检查必要参数
[ -z "$token" ] && reading "主控Token：" token
[ -z "$host" ] && reading "主控IPV4/域名：" host
[ -z "$api_port" ] && reading "主控API端口：" api_port
[ -z "$grpc_port" ] && reading "主控gRPC端口：" grpc_port
[ -z "$use_cf" ] && use_cf="false"  # 默认不使用 CF 服务
[ -z "$cf_service" ] && cf_service="http://127.0.0.1:8000"  # 默认 CF 服务地址

# 检查现有服务并处理
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

# 下载新文件
_blue "正在下载最新版本..."
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent -o /usr/local/bin/ecsagent
curl -s https://raw.githubusercontent.com/spiritLHLS/monitor-agent/main/ecsagent.service -o /etc/systemd/system/ecsagent.service

# 设置权限
chmod +x /usr/local/bin/ecsagent
chmod +x /etc/systemd/system/ecsagent.service

# 更新服务配置
if [ -f "/etc/systemd/system/ecsagent.service" ]; then
    new_exec_start="ExecStart=/usr/local/bin/ecsagent -token ${token} -host ${host} -grpc-port ${grpc_port} -api-port ${api_port} -use-cf ${use_cf} -cf-service ${cf_service}"
    file_path="/etc/systemd/system/ecsagent.service"
    line_number=6
    sed -i "${line_number}s|.*|${new_exec_start}|" "$file_path"
fi

# 启动服务
systemctl daemon-reload
systemctl start ecsagent.service
systemctl enable ecsagent.service

# 检查安装结果
if check_service; then
    _green "ECS Agent 安装成功并已启动！"
else
    _red "ECS Agent 安装可能存在问题，请检查日志"
    systemctl status ecsagent.service
fi

# 使用说明
_blue "使用示例："
echo "# 使用默认设置（启用 CF 服务）"
echo "#./ecsagent.sh -token xxx -host xxx -grpc-port xxx -api-port xxx"
echo "# 禁用 CF 服务"
echo "#./ecsagent.sh -token xxx -host xxx -grpc-port xxx -api-port xxx -use-cf false"
echo "# 指定自定义 CF 服务地址"
echo "#./ecsagent.sh -token xxx -host xxx -grpc-port xxx -api-port xxx -cf-service http://custom-server:8000"