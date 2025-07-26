#!/bin/bash

# ⚠️  DEPRECATED - 此脚本已弃用 ⚠️ 
# 
# 📢 重要通知：
# 从二期开始，Context-Keeper HTTP/WebSocket 模式的日志已直接输出到标准输出。
# 云端部署可以直接通过 `docker logs` 命令查看业务日志，不再需要此监控脚本。
#
# 🔄 历史背景：
# 此脚本是为解决一期 STDIO 协议的日志冲突问题而创建的临时方案。
# 由于 STDIO 协议使用标准输出进行 MCP 通信，日志被重定向到文件以避免干扰。
# 二期改用 HTTP 协议后，此限制已不存在。
#
# 📋 当前状态：
# - HTTP 模式：日志直接输出到 stdout ✅
# - WebSocket 模式：日志直接输出到 stdout ✅  
# - STDIO 模式：日志仍需输出到文件（避免协议冲突）⚠️
#
# 💡 建议操作：
# 1. 云端部署使用 HTTP/WebSocket 模式
# 2. 使用 `docker logs` 查看实时日志
# 3. 移除对此脚本的依赖
#
# 保留此文件仅用于：
# - 文档参考
# - STDIO 模式兜底（如有需要）
# - 迁移期间的向后兼容

echo "⚠️ [DEPRECATED] docker-log-monitor.sh 已弃用"
echo "📢 HTTP/WebSocket 模式的日志已直接输出到标准输出"
echo "💡 建议：使用 'docker logs' 命令查看业务日志"
echo "🔄 此脚本将在 10 秒后继续执行（仅用于兼容性）"
echo ""

for i in {10..1}; do
    echo -n "⏱️  $i "
    sleep 1
done
echo ""
echo "继续执行原逻辑..."
echo ""

# Context-Keeper 日志监控脚本
# 将业务日志文件的内容实时输出到标准输出，方便云端查看
# 🔥 优化：根据启动模式智能选择监控的日志文件

LOG_BASE_DIR="/app/data/logs"
CONTAINER_LOG_DIR="/home/appuser/Library/Application Support/context-keeper/logs"

echo "🔍 [日志监控] 启动日志文件监控..."
echo "📁 [日志监控] 主日志目录: $LOG_BASE_DIR"  
echo "📁 [日志监控] 容器日志目录: $CONTAINER_LOG_DIR"

# 创建必要的日志目录
mkdir -p "$LOG_BASE_DIR"
mkdir -p "$(dirname "$CONTAINER_LOG_DIR")"

# 创建符号链接，将容器内日志目录链接到统一位置
if [ ! -L "$CONTAINER_LOG_DIR" ]; then
    rm -rf "$CONTAINER_LOG_DIR"
    ln -sf "$LOG_BASE_DIR" "$CONTAINER_LOG_DIR"
    echo "🔗 [日志监控] 已创建日志目录符号链接"
fi

# 🔥 智能检测启动模式，确定要监控的日志文件
detect_run_mode() {
    local mode="${RUN_MODE:-}"
    
    # 如果环境变量未设置，尝试从进程中检测
    if [ -z "$mode" ]; then
        if pgrep -f "context-keeper-http" >/dev/null 2>&1; then
            mode="http"
        elif pgrep -f "context-keeper-websocket" >/dev/null 2>&1; then
            mode="websocket"
        elif pgrep -f "context-keeper.*stdio" >/dev/null 2>&1; then
            mode="stdio"
        else
            # 默认假设为HTTP模式（Docker默认模式）
            mode="http"
        fi
    fi
    
    echo "$mode"
}

# 根据模式确定日志文件
get_log_files() {
    local mode="$1"
    local files=()
    
    case "$mode" in
        "http")
            files+=("$LOG_BASE_DIR/context-keeper-streamable-http.log")
            echo "📋 [日志监控] HTTP模式：监控streamable-http日志"
            ;;
        "websocket")
            files+=("$LOG_BASE_DIR/context-keeper-websocket.log")
            echo "📋 [日志监控] WebSocket模式：监控websocket日志"
            ;;
        "stdio")
            files+=("$LOG_BASE_DIR/context-keeper-debug.log")
            echo "📋 [日志监控] STDIO模式：监控debug日志"
            ;;
        *)
            # 兜底方案：监控所有可能的日志文件
            files+=(
                "$LOG_BASE_DIR/context-keeper-streamable-http.log"
                "$LOG_BASE_DIR/context-keeper-websocket.log" 
                "$LOG_BASE_DIR/context-keeper-debug.log"
            )
            echo "⚠️ [日志监控] 未知模式 '$mode'：监控所有日志文件"
            ;;
    esac
    
    printf '%s\n' "${files[@]}"
}

# 检测运行模式
echo "🔍 [日志监控] 检测运行模式..."
RUN_MODE=$(detect_run_mode)
echo "🎯 [日志监控] 检测到运行模式: $RUN_MODE"

# 等待日志文件出现并开始监控
echo "⏳ [日志监控] 等待日志文件生成..."
sleep 5

# 获取要监控的日志文件列表
mapfile -t LOG_FILES < <(get_log_files "$RUN_MODE")

echo "📝 [日志监控] 将监控以下日志文件:"
for file in "${LOG_FILES[@]}"; do
    echo "  - $file"
done

# 启动tail进程监控各个日志文件
TAIL_PIDS=()

for log_file in "${LOG_FILES[@]}"; do
    if [ -f "$log_file" ]; then
        echo "📄 [日志监控] 开始监控: $log_file"
        # 添加日志文件标识前缀，便于区分不同日志来源
        tail -f "$log_file" | sed "s/^/[$(basename "$log_file")] /" &
        TAIL_PIDS+=($!)
    else
        echo "⚠️ [日志监控] 日志文件不存在，将持续检查: $log_file"
        # 异步等待文件出现
        (
            while [ ! -f "$log_file" ]; do
                sleep 2
            done
            echo "✅ [日志监控] 发现新日志文件: $log_file"
            tail -f "$log_file" | sed "s/^/[$(basename "$log_file")] /"
        ) &
        TAIL_PIDS+=($!)
    fi
done

echo "🚀 [日志监控] 已启动 ${#TAIL_PIDS[@]} 个日志监控进程（$RUN_MODE 模式）"

# 优雅关闭处理
cleanup() {
    echo "🛑 [日志监控] 收到停止信号，正在关闭监控进程..."
    for pid in "${TAIL_PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    exit 0
}

trap cleanup SIGTERM SIGINT

# 保持脚本运行
wait 