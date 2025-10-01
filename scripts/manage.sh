#!/bin/bash

# Context-Keeper 服务管理脚本
# 提供编译、启动、停止、重启、状态检查等功能

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 获取脚本所在目录和项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# PID文件路径
PID_DIR="${PROJECT_ROOT}/logs"
STDIO_PID_FILE="${PID_DIR}/context-keeper-stdio.pid"
HTTP_PID_FILE="${PID_DIR}/context-keeper-http.pid"

# 创建必要的目录
mkdir -p "${PID_DIR}"

# 显示帮助信息
show_help() {
    echo -e "${BLUE}Context-Keeper 服务管理工具${NC}"
    echo ""
    echo "用法: ./manage.sh <命令> [选项]"
    echo ""
    echo "命令:"
    echo "  build                     编译所有版本"
    echo "  build --stdio            仅编译STDIO版本"
    echo "  build --http             仅编译HTTP版本"
    echo "  start <模式> [选项]       启动服务"
    echo "  stop [模式]              停止服务"
    echo "  restart <模式> [选项]     重启服务"
    echo "  status                   检查服务状态"
    echo "  logs <模式>              查看服务日志"
    echo "  clean                    清理编译产物和PID文件"
    echo "  deploy <模式> [选项]     一键部署（停止->编译->启动）"
    echo ""
    echo "模式:"
    echo "  stdio                    STDIO模式（用于MCP通信）"
    echo "  http                     HTTP模式（用于网络通信）"
    echo ""
    echo "启动选项:"
    echo "  --port PORT              HTTP模式端口（默认: 8088）"
    echo "  --foreground             前台运行（默认后台运行）"
    echo ""
    echo "示例:"
    echo "  ./manage.sh build                    # 编译所有版本"
    echo "  ./manage.sh start stdio              # 启动STDIO模式（后台）"
    echo "  ./manage.sh start http --port 8080   # 启动HTTP模式在8080端口"
    echo "  ./manage.sh stop http               # 停止HTTP服务"
    echo "  ./manage.sh status                  # 查看所有服务状态"
    echo "  ./manage.sh deploy http             # 一键部署HTTP服务"
    echo "  ./manage.sh logs stdio              # 查看STDIO服务日志"
}

# 检查进程是否运行
is_running() {
    local mode=$1
    local pid_file=""

    if [ "$mode" = "stdio" ]; then
        pid_file="$STDIO_PID_FILE"
    elif [ "$mode" = "http" ]; then
        pid_file="$HTTP_PID_FILE"
    else
        return 1
    fi

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            return 0  # 运行中
        else
            rm -f "$pid_file"  # 清理无效的PID文件
            return 1  # 未运行
        fi
    fi
    return 1  # 未运行
}

# 获取进程PID
get_pid() {
    local mode=$1
    local pid_file=""

    if [ "$mode" = "stdio" ]; then
        pid_file="$STDIO_PID_FILE"
    elif [ "$mode" = "http" ]; then
        pid_file="$HTTP_PID_FILE"
    else
        return 1
    fi

    if [ -f "$pid_file" ]; then
        cat "$pid_file"
    fi
}

# 进程二进制路径（用于校验与兜底匹配）
expected_binary_path() {
    local mode=$1
    if [ "$mode" = "stdio" ]; then
        echo "$PROJECT_ROOT/bin/context-keeper"
    elif [ "$mode" = "http" ]; then
        echo "$PROJECT_ROOT/bin/context-keeper-http"
    else
        echo ""
    fi
}

# 校验指定 PID 是否为给定模式的本服务进程
pid_belongs_to_mode() {
    local mode=$1
    local pid=$2
    local expected
    expected="$(expected_binary_path "$mode")"
    if [ -z "$expected" ] || [ -z "$pid" ]; then
        return 1
    fi
    # 读取命令行
    local cmd
    cmd=$(ps -o command= -p "$pid" 2>/dev/null || true)
    if [ -z "$cmd" ]; then
        return 1
    fi
    # 允许多种形式匹配：绝对路径、相对路径、二进制名
    local expected_base
    expected_base=$(basename "$expected")
    if echo "$cmd" | grep -Fq "$expected"; then
        return 0
    fi
    if echo "$cmd" | grep -Fq "./bin/$expected_base"; then
        return 0
    fi
    if echo "$cmd" | grep -Eq "(^|[ /])$expected_base([[:space:]]|$)"; then
        return 0
    fi
    return 1
}

# 兜底：按二进制路径匹配查找进程 PID 列表
find_pids_by_pattern() {
    local mode=$1
    local expected
    expected="$(expected_binary_path "$mode")"
    if [ -z "$expected" ]; then
        return 1
    fi
    # 优先使用 pgrep，若不可用则用 ps+grep
    if command -v pgrep >/dev/null 2>&1; then
        pgrep -f "$expected" 2>/dev/null || true
    else
        ps aux | grep -F "$expected" | grep -v grep | awk '{print $2}' || true
    fi
}

# 杀掉一组 PID（优雅->强制）
kill_pids_list() {
    local pids=($@)
    if [ ${#pids[@]} -eq 0 ]; then
        return 0
    fi
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    # 等待最多 10 秒
    for i in $(seq 1 10); do
        local alive=0
        for pid in "${pids[@]}"; do
            if ps -p "$pid" >/dev/null 2>&1; then
                alive=1
                break
            fi
        done
        [ $alive -eq 0 ] && break
        sleep 1
    done
    # 仍存活则强杀
    for pid in "${pids[@]}"; do
        if ps -p "$pid" >/dev/null 2>&1; then
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
}

# 编译函数
build_service() {
    local build_type="all"

    # 解析编译参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --stdio)
                build_type="stdio"
                shift
                ;;
            --http)
                build_type="http"
                shift
                ;;
            --all)
                build_type="all"
                shift
                ;;
            *)
                echo -e "${RED}未知编译参数: $1${NC}"
                return 1
                ;;
        esac
    done

    echo -e "${BLUE}开始编译 Context-Keeper...${NC}"

    cd "$PROJECT_ROOT" || {
        echo -e "${RED}无法切换到项目根目录${NC}"
        return 1
    }

    # 调用构建脚本
    if [ "$build_type" = "all" ]; then
        ./scripts/build/build.sh --all
    else
        ./scripts/build/build.sh --"$build_type"
    fi

    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}编译完成！${NC}"
    else
        echo -e "${RED}编译失败！${NC}"
    fi

    return $exit_code
}

# 启动服务
start_service() {
    local mode="$1"
    shift
    local port=8088
    local foreground=false

    if [ -z "$mode" ]; then
        echo -e "${RED}请指定启动模式: stdio 或 http${NC}"
        return 1
    fi

    # 解析启动参数
    local background=false
    while [[ $# -gt 0 ]]; do
        case $1 in
            --port)
                port="$2"
                shift 2
                ;;
            --foreground)
                foreground=true
                shift
                ;;
            --background|-bg)
                background=true
                shift
                ;;
            *)
                echo -e "${RED}未知启动参数: $1${NC}"
                return 1
                ;;
        esac
    done

    # 检查是否已经运行（并校验 PID 是否属于本服务）
    if is_running "$mode"; then
        local pid=$(get_pid "$mode")
        if pid_belongs_to_mode "$mode" "$pid"; then
            echo -e "${YELLOW}$mode 模式的服务已在运行中 (PID: $pid)${NC}"
            return 0
        else
            echo -e "${YELLOW}检测到无效的PID文件（PID=$pid非本服务），已清理，继续启动...${NC}"
            if [ "$mode" = "stdio" ]; then
                rm -f "$STDIO_PID_FILE"
            else
                rm -f "$HTTP_PID_FILE"
            fi
        fi
    fi

    cd "$PROJECT_ROOT" || {
        echo -e "${RED}无法切换到项目根目录${NC}"
        return 1
    }

    # 选择二进制文件
    local binary=""
    if [ "$mode" = "stdio" ]; then
        binary="./bin/context-keeper"
    elif [ "$mode" = "http" ]; then
        binary="./bin/context-keeper-http"
    else
        echo -e "${RED}无效的模式: $mode${NC}"
        return 1
    fi

    # 检查二进制文件是否存在
    if [ ! -f "$binary" ]; then
        echo -e "${YELLOW}二进制文件不存在，正在编译...${NC}"
        if ! build_service --"$mode"; then
            echo -e "${RED}编译失败，无法启动服务${NC}"
            return 1
        fi
    fi

    # 设置环境变量
    export RUN_MODE="$mode"
    if [ "$mode" = "http" ]; then
        export HTTP_SERVER_PORT="$port"
    fi

    echo -e "${GREEN}启动 $mode 模式的服务...${NC}"

    # 设置日志和PID文件路径
    local log_file="$PID_DIR/context-keeper-$mode.log"
    local pid_file=""

    if [ "$mode" = "stdio" ]; then
        pid_file="$STDIO_PID_FILE"
    else
        pid_file="$HTTP_PID_FILE"
    fi

    # 启动进程
    if [ "$mode" = "stdio" ]; then
        # STDIO模式：必须重定向到文件，避免日志干扰MCP协议通信
        nohup "$binary" > "$log_file" 2>&1 &
        local pid=$!
        echo -e "${YELLOW}STDIO模式：日志重定向到文件 $log_file${NC}"
    elif [ "$background" = "true" ]; then
        # HTTP模式后台运行
        nohup "$binary" > "$log_file" 2>&1 &
        local pid=$!
        echo -e "${YELLOW}HTTP模式后台运行：日志输出到 $log_file${NC}"
    else
        # HTTP模式前台运行，显示实时日志
        echo -e "${GREEN}HTTP模式前台运行：显示实时日志，按 Ctrl+C 停止服务${NC}"
        echo -e "${BLUE}如需后台运行，请使用: $0 deploy $mode --background${NC}"
        echo "----------------------------------------"
        # 也写入一个临时 PID 文件用于 stop 兜底：记录当前 shell 的 PID（父进程），便于 stop 发现无效后走兜底流程
        echo "$$" > "$pid_file"
        "$binary" &
        wait $!
        rm -f "$pid_file"
        return 0
    fi

    # 保存PID（仅后台模式需要）
    if [ -n "$pid" ]; then
        echo "$pid" > "$pid_file"
    fi

    # 等待服务启动（仅后台模式需要）
    if [ -n "$pid" ]; then
        sleep 2
    fi

    # 检查进程是否还在运行（仅后台模式）
    if [ -n "$pid" ] && ps -p "$pid" > /dev/null 2>&1; then
        echo -e "${GREEN}服务启动成功！${NC}"
        echo -e "${GREEN}模式: $mode${NC}"
        echo -e "${GREEN}PID: $pid${NC}"
        if [ "$mode" = "http" ]; then
            echo -e "${GREEN}端口: $port${NC}"
            echo -e "${GREEN}访问地址: http://localhost:$port${NC}"
        fi
        echo -e "${YELLOW}日志文件: $log_file${NC}"
        # 显示最近的日志
        echo -e "\n${YELLOW}最近的日志:${NC}"
        tail -n 10 "$log_file"
    elif [ -n "$pid" ]; then
        # 兜底：尝试通过二进制路径匹配确认是否已经启动
        mapfile -t started_pids < <(find_pids_by_pattern "$mode")
        if [ ${#started_pids[@]} -gt 0 ]; then
            echo -e "${GREEN}服务已启动（通过模式匹配确认），PID: ${started_pids[0]}${NC}"
            echo -e "${YELLOW}日志文件: $log_file${NC}"
            echo -e "\n${YELLOW}最近的日志:${NC}"
            tail -n 10 "$log_file"
        else
            echo -e "${RED}服务启动失败${NC}"
            rm -f "$pid_file"
            if [ -f "$log_file" ]; then
                echo -e "${RED}错误日志:${NC}"
                tail -n 100 "$log_file"
            fi
            return 1
        fi
    fi
}

# 停止服务
stop_service() {
    local mode="$1"

    if [ -z "$mode" ]; then
        # 停止所有服务
        local stopped=false
        for m in stdio http; do
            if is_running "$m"; then
                stop_service "$m"
                stopped=true
            fi
        done
        if [ "$stopped" = false ]; then
            echo -e "${YELLOW}没有运行中的服务${NC}"
        fi
        return 0
    fi

    if ! is_running "$mode"; then
        echo -e "${YELLOW}$mode 模式的服务未运行，尝试兜底匹配${NC}"
    fi

    local pid=$(get_pid "$mode")
    local pids_to_kill=()

    if [ -n "$pid" ] && pid_belongs_to_mode "$mode" "$pid"; then
        echo -e "${YELLOW}停止 $mode 模式的服务 (PID: $pid)...${NC}"
        pids_to_kill+=("$pid")
    else
        # PID 文件无效或不存在，尝试兜底匹配
        echo -e "${YELLOW}未找到有效 PID，尝试按二进制路径兜底匹配进程...${NC}"
        mapfile -t found_pids < <(find_pids_by_pattern "$mode")
        if [ ${#found_pids[@]} -gt 0 ]; then
            echo -e "${YELLOW}匹配到如下进程: ${found_pids[*]}${NC}"
            pids_to_kill+=("${found_pids[@]}")
        else
            echo -e "${YELLOW}$mode 模式的服务未运行${NC}"
        fi
    fi

    # 执行优雅停止 -> 强制停止
    if [ ${#pids_to_kill[@]} -gt 0 ]; then
        kill_pids_list "${pids_to_kill[@]}"
    fi

    # 清理PID文件
    if [ "$mode" = "stdio" ]; then
        rm -f "$STDIO_PID_FILE"
    elif [ "$mode" = "http" ]; then
        rm -f "$HTTP_PID_FILE"
    fi

    echo -e "${GREEN}$mode 模式的服务已停止${NC}"
}

# 重启服务
restart_service() {
    local mode="$1"
    shift

    if [ -z "$mode" ]; then
        echo -e "${RED}请指定重启模式: stdio 或 http${NC}"
        return 1
    fi

    echo -e "${BLUE}重启 $mode 模式的服务...${NC}"

    # 停止服务
    stop_service "$mode"

    # 等待一下确保完全停止
    sleep 1

    # 启动服务
    start_service "$mode" "$@"
}

# 检查服务状态
check_status() {
    echo -e "${BLUE}Context-Keeper 服务状态:${NC}"
    echo ""

    local any_running=false

    for mode in stdio http; do
        if is_running "$mode"; then
            local pid=$(get_pid "$mode")
            local memory=$(ps -o rss= -p "$pid" 2>/dev/null | awk '{print $1/1024}' 2>/dev/null)
            local cpu=$(ps -o %cpu= -p "$pid" 2>/dev/null | awk '{print $1}' 2>/dev/null)
            local start_time=$(ps -o lstart= -p "$pid" 2>/dev/null)

            echo -e "${GREEN}✓ $mode 模式: 运行中${NC}"
            echo -e "  PID: $pid"
            if [ -n "$memory" ]; then
                echo -e "  内存: ${memory}MB"
            fi
            if [ -n "$cpu" ]; then
                echo -e "  CPU: ${cpu}%"
            fi
            if [ -n "$start_time" ]; then
                echo -e "  启动时间: $start_time"
            fi

            if [ "$mode" = "http" ]; then
                local port=$(ps -o command= -p "$pid" | grep -o "HTTP_SERVER_PORT=[0-9]*" | cut -d= -f2)
                if [ -n "$port" ]; then
                    echo -e "  端口: $port"
                fi
            fi

            any_running=true
        else
            echo -e "${RED}✗ $mode 模式: 未运行${NC}"
        fi
        echo ""
    done

    if [ "$any_running" = false ]; then
        echo -e "${YELLOW}所有服务都未运行${NC}"
    fi
}

# 查看日志
view_logs() {
    local mode="$1"
    local lines="${2:-50}"

    if [ -z "$mode" ]; then
        echo -e "${RED}请指定模式: stdio 或 http${NC}"
        return 1
    fi

    local log_file="$PID_DIR/context-keeper-$mode.log"

    if [ ! -f "$log_file" ]; then
        echo -e "${YELLOW}日志文件不存在: $log_file${NC}"
        return 1
    fi

    echo -e "${BLUE}$mode 模式的服务日志 (最近 $lines 行):${NC}"
    echo "日志文件: $log_file"
    echo "----------------------------------------"
    tail -n "$lines" "$log_file"
}

# 清理文件
clean_files() {
    echo -e "${YELLOW}清理编译产物和临时文件...${NC}"

    cd "$PROJECT_ROOT" || return 1

    # 清理二进制文件
    rm -f ./bin/context-keeper ./bin/context-keeper-http
    rm -f ./context-keeper ./context-keeper-http ./context-keeper-stdio
    rm -f ./test-*

    # 清理日志文件
    rm -f "$PID_DIR"/*.log
    rm -f "$PID_DIR"/*.pid

    echo -e "${GREEN}清理完成${NC}"
}

# 一键部署
deploy_service() {
    local mode="$1"
    shift

    if [ -z "$mode" ]; then
        echo -e "${RED}请指定部署模式: stdio 或 http${NC}"
        return 1
    fi

    echo -e "${BLUE}一键部署 $mode 模式的服务...${NC}"

    # 1. 停止现有服务
    if is_running "$mode"; then
        echo -e "${YELLOW}停止现有服务...${NC}"
        stop_service "$mode"
    fi

    # 2. 编译服务
    echo -e "${YELLOW}编译服务...${NC}"
    if ! build_service --"$mode"; then
        echo -e "${RED}编译失败，部署中止${NC}"
        return 1
    fi

    # 3. 启动服务
    echo -e "${YELLOW}启动服务...${NC}"
    # 如果是 http 模式且未显式 --foreground，则默认后台运行，保证 PID 文件可用
    local args=("$@")
    local explicitly_foreground=false
    for a in "${args[@]}"; do
        if [ "$a" = "--foreground" ]; then
            explicitly_foreground=true
            break
        fi
    done
    if [ "$mode" = "http" ] && [ "$explicitly_foreground" = false ]; then
        args+=("--background")
    fi
    if start_service "$mode" "${args[@]}"; then
        echo -e "${GREEN}部署成功！${NC}"
    else
        echo -e "${RED}启动失败，部署失败${NC}"
        return 1
    fi
}

# 主函数
main() {
    if [ $# -eq 0 ]; then
        show_help
        return 0
    fi

    local command="$1"
    shift

    case "$command" in
        build)
            build_service "$@"
            ;;
        start)
            start_service "$@"
            ;;
        stop)
            stop_service "$@"
            ;;
        restart)
            restart_service "$@"
            ;;
        status)
            check_status
            ;;
        logs)
            view_logs "$@"
            ;;
        clean)
            clean_files
            ;;
        deploy)
            deploy_service "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}未知命令: $command${NC}"
            echo "使用 './manage.sh help' 查看帮助"
            return 1
            ;;
    esac
}

# 运行主函数
main "$@"