#!/bin/bash

# Context-Keeper 后台进程启动脚本
# 支持后台运行、进程守护、自动重启等功能

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取脚本所在目录和项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 配置文件路径
PID_DIR="${PROJECT_ROOT}/logs"
DAEMON_CONFIG="${PROJECT_ROOT}/config/daemon.conf"

# 创建必要的目录
mkdir -p "${PID_DIR}"
mkdir -p "${PROJECT_ROOT}/config"

# 默认配置
DEFAULT_MODE="http"
DEFAULT_PORT="8088"
DEFAULT_AUTO_RESTART="true"
DEFAULT_MAX_RESTARTS="5"
DEFAULT_RESTART_DELAY="5"

# 显示帮助信息
show_help() {
    echo -e "${BLUE}Context-Keeper 后台进程启动脚本${NC}"
    echo ""
    echo "用法: ./start-daemon.sh [选项]"
    echo ""
    echo "选项:"
    echo "  --mode MODE              服务模式 (stdio|http, 默认: http)"
    echo "  --port PORT              HTTP端口 (默认: 8088)"
    echo "  --auto-restart           启用自动重启 (默认: 启用)"
    echo "  --no-auto-restart        禁用自动重启"
    echo "  --max-restarts N         最大重启次数 (默认: 5)"
    echo "  --restart-delay N        重启延迟秒数 (默认: 5)"
    echo "  --save-config            保存配置到文件"
    echo "  --load-config            从文件加载配置"
    echo "  --install-service        安装为系统服务"
    echo "  --uninstall-service      卸载系统服务"
    echo "  --help, -h               显示帮助信息"
    echo ""
    echo "守护进程功能:"
    echo "  - 后台运行，不受终端关闭影响"
    echo "  - 自动监控服务状态"
    echo "  - 异常退出时自动重启"
    echo "  - 详细的日志记录"
    echo "  - 支持系统服务安装"
    echo ""
    echo "示例:"
    echo "  ./start-daemon.sh                        # 使用默认配置启动"
    echo "  ./start-daemon.sh --mode stdio           # 启动STDIO模式"
    echo "  ./start-daemon.sh --port 8080            # 指定端口"
    echo "  ./start-daemon.sh --no-auto-restart      # 禁用自动重启"
    echo "  ./start-daemon.sh --save-config          # 保存当前配置"
}

# 加载配置文件
load_config() {
    if [ -f "$DAEMON_CONFIG" ]; then
        echo -e "${YELLOW}加载配置文件: $DAEMON_CONFIG${NC}"
        source "$DAEMON_CONFIG"
    else
        echo -e "${YELLOW}配置文件不存在，使用默认配置${NC}"
    fi
}

# 保存配置文件
save_config() {
    echo -e "${YELLOW}保存配置到: $DAEMON_CONFIG${NC}"
    cat > "$DAEMON_CONFIG" << EOF
# Context-Keeper 守护进程配置
MODE="$MODE"
PORT="$PORT"
AUTO_RESTART="$AUTO_RESTART"
MAX_RESTARTS="$MAX_RESTARTS"
RESTART_DELAY="$RESTART_DELAY"
CREATED_TIME="$(date)"
EOF
    echo -e "${GREEN}配置已保存${NC}"
}

# 安装系统服务 (macOS)
install_macos_service() {
    local service_name="com.context-keeper.daemon"
    local plist_file="$HOME/Library/LaunchAgents/${service_name}.plist"
    local script_path="$(realpath "$0")"
    
    echo -e "${YELLOW}安装 macOS LaunchAgent 服务...${NC}"
    
    # 创建 plist 文件
    cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${service_name}</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>${script_path}</string>
        <string>--load-config</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${PROJECT_ROOT}</string>
    <key>StandardOutPath</key>
    <string>${PID_DIR}/daemon.out.log</string>
    <key>StandardErrorPath</key>
    <string>${PID_DIR}/daemon.err.log</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF
    
    # 加载服务
    launchctl load "$plist_file"
    
    echo -e "${GREEN}macOS 服务安装完成${NC}"
    echo -e "${GREEN}服务将在系统启动时自动运行${NC}"
    echo -e "${YELLOW}服务文件: $plist_file${NC}"
    echo -e "${YELLOW}使用以下命令管理服务:${NC}"
    echo "  启动: launchctl start ${service_name}"
    echo "  停止: launchctl stop ${service_name}"
    echo "  卸载: launchctl unload ${plist_file}"
}

# 卸载系统服务 (macOS)
uninstall_macos_service() {
    local service_name="com.context-keeper.daemon"
    local plist_file="$HOME/Library/LaunchAgents/${service_name}.plist"
    
    echo -e "${YELLOW}卸载 macOS LaunchAgent 服务...${NC}"
    
    if [ -f "$plist_file" ]; then
        launchctl unload "$plist_file"
        rm -f "$plist_file"
        echo -e "${GREEN}服务卸载完成${NC}"
    else
        echo -e "${YELLOW}服务未安装${NC}"
    fi
}

# 启动守护进程
start_daemon() {
    local mode="$1"
    local port="$2"
    local auto_restart="$3"
    local max_restarts="$4"
    local restart_delay="$5"
    
    echo -e "${BLUE}启动 Context-Keeper 守护进程${NC}"
    echo -e "${GREEN}模式: $mode${NC}"
    if [ "$mode" = "http" ]; then
        echo -e "${GREEN}端口: $port${NC}"
    fi
    echo -e "${GREEN}自动重启: $auto_restart${NC}"
    if [ "$auto_restart" = "true" ]; then
        echo -e "${GREEN}最大重启次数: $max_restarts${NC}"
        echo -e "${GREEN}重启延迟: ${restart_delay}秒${NC}"
    fi
    
    cd "$PROJECT_ROOT" || {
        echo -e "${RED}无法切换到项目根目录${NC}"
        exit 1
    }
    
    # 选择二进制文件
    local binary=""
    if [ "$mode" = "stdio" ]; then
        binary="./bin/context-keeper"
    elif [ "$mode" = "http" ]; then
        binary="./bin/context-keeper-http"
    else
        echo -e "${RED}无效的模式: $mode${NC}"
        exit 1
    fi
    
    # 检查二进制文件是否存在
    if [ ! -f "$binary" ]; then
        echo -e "${YELLOW}二进制文件不存在，正在编译...${NC}"
        if ! ./scripts/build/build.sh --"$mode"; then
            echo -e "${RED}编译失败，无法启动守护进程${NC}"
            exit 1
        fi
    fi
    
    # 设置环境变量
    export RUN_MODE="$mode"
    if [ "$mode" = "http" ]; then
        export HTTP_SERVER_PORT="$port"
    fi
    
    # 日志文件
    local daemon_log="$PID_DIR/daemon-$mode.log"
    local service_log="$PID_DIR/context-keeper-$mode.log"
    local pid_file="$PID_DIR/context-keeper-$mode.pid"
    
    # 启动监控循环
    local restart_count=0
    local start_time=$(date +%s)
    
    echo -e "${GREEN}守护进程已启动，监控服务状态...${NC}"
    echo -e "${YELLOW}守护进程日志: $daemon_log${NC}"
    echo -e "${YELLOW}服务日志: $service_log${NC}"
    
    # 将守护进程信息写入日志
    {
        echo "$(date): 守护进程启动"
        echo "模式: $mode"
        echo "端口: $port"
        echo "自动重启: $auto_restart"
        echo "最大重启次数: $max_restarts"
        echo "重启延迟: ${restart_delay}秒"
        echo "----------------------------------------"
    } >> "$daemon_log"
    
    while true; do
        # 启动服务
        echo "$(date): 启动服务..." >> "$daemon_log"
        
        nohup "$binary" > "$service_log" 2>&1 &
        local service_pid=$!
        echo "$service_pid" > "$pid_file"
        
        echo "$(date): 服务已启动，PID: $service_pid" >> "$daemon_log"
        
        # 监控服务状态
        while ps -p "$service_pid" > /dev/null 2>&1; do
            sleep 5
        done
        
        # 服务异常退出
        local exit_time=$(date +%s)
        local run_duration=$((exit_time - start_time))
        
        echo "$(date): 服务异常退出，运行时长: ${run_duration}秒" >> "$daemon_log"
        
        # 清理PID文件
        rm -f "$pid_file"
        
        # 检查是否需要重启
        if [ "$auto_restart" != "true" ]; then
            echo "$(date): 自动重启已禁用，守护进程退出" >> "$daemon_log"
            break
        fi
        
        restart_count=$((restart_count + 1))
        
        if [ $restart_count -gt $max_restarts ]; then
            echo "$(date): 达到最大重启次数 ($max_restarts)，守护进程退出" >> "$daemon_log"
            break
        fi
        
        echo "$(date): 准备重启服务 ($restart_count/$max_restarts)，${restart_delay}秒后重启..." >> "$daemon_log"
        sleep "$restart_delay"
        
        start_time=$(date +%s)
    done
    
    echo -e "${YELLOW}守护进程已退出${NC}"
}

# 主函数
main() {
    # 默认值
    MODE="$DEFAULT_MODE"
    PORT="$DEFAULT_PORT"
    AUTO_RESTART="$DEFAULT_AUTO_RESTART"
    MAX_RESTARTS="$DEFAULT_MAX_RESTARTS"
    RESTART_DELAY="$DEFAULT_RESTART_DELAY"
    LOAD_CONFIG_FLAG=false
    SAVE_CONFIG_FLAG=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --mode)
                MODE="$2"
                shift 2
                ;;
            --port)
                PORT="$2"
                shift 2
                ;;
            --auto-restart)
                AUTO_RESTART="true"
                shift
                ;;
            --no-auto-restart)
                AUTO_RESTART="false"
                shift
                ;;
            --max-restarts)
                MAX_RESTARTS="$2"
                shift 2
                ;;
            --restart-delay)
                RESTART_DELAY="$2"
                shift 2
                ;;
            --save-config)
                SAVE_CONFIG_FLAG=true
                shift
                ;;
            --load-config)
                LOAD_CONFIG_FLAG=true
                shift
                ;;
            --install-service)
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    install_macos_service
                    exit 0
                else
                    echo -e "${RED}系统服务安装目前仅支持 macOS${NC}"
                    exit 1
                fi
                ;;
            --uninstall-service)
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    uninstall_macos_service
                    exit 0
                else
                    echo -e "${RED}系统服务卸载目前仅支持 macOS${NC}"
                    exit 1
                fi
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}未知参数: $1${NC}"
                echo "使用 --help 查看帮助"
                exit 1
                ;;
        esac
    done
    
    # 加载配置
    if [ "$LOAD_CONFIG_FLAG" = true ]; then
        load_config
    fi
    
    # 验证参数
    if [ "$MODE" != "stdio" ] && [ "$MODE" != "http" ]; then
        echo -e "${RED}无效的模式: $MODE${NC}"
        exit 1
    fi
    
    # 保存配置
    if [ "$SAVE_CONFIG_FLAG" = true ]; then
        save_config
        exit 0
    fi
    
    # 启动守护进程
    start_daemon "$MODE" "$PORT" "$AUTO_RESTART" "$MAX_RESTARTS" "$RESTART_DELAY"
}

# 运行主函数
main "$@" 