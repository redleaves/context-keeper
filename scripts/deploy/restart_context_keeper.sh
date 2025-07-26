#!/bin/bash

# 定义颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 设置工作目录为项目根目录
cd "$(dirname "$0")/../.." || exit 1
WORKSPACE_DIR=$(pwd)

# 显示彩色消息的函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 默认运行模式为stdio
RUN_MODE="stdio"
HTTP_SERVER_PORT="8088"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    http|--http|sse|--sse)
      RUN_MODE="http"
      shift
      ;;
    stdio|--stdio)
      RUN_MODE="stdio"
      shift
      ;;
    port|--port)
      HTTP_SERVER_PORT="$2"
      shift 2
      ;;
    daemon|--daemon|-d)
      DAEMON_MODE=true
      shift
      ;;
    help|--help|-h)
      echo "用法: $0 [选项]"
      echo "选项:"
      echo "  stdio, --stdio    使用STDIO模式启动（默认，适用于MCP通信）"
      echo "  http, --http      使用HTTP/SSE模式启动（适用于网络通信）"
      echo "  sse, --sse        HTTP/SSE模式的别名"
      echo "  port, --port PORT 指定HTTP服务器端口（默认: 8088）"
      echo "  daemon, --daemon  以守护进程模式运行"
      echo "  help, --help      显示此帮助信息"
      exit 0
      ;;
    *)
      log_error "未知参数: $1"
      echo "使用 --help 查看帮助"
      exit 1
      ;;
  esac
done

# 定义路径
CURSOR_DIR="$HOME/.cursor"
SOURCE_DIR="$WORKSPACE_DIR"
STORAGE_DIR="/tmp/context-keeper"
LOG_FILE="$SOURCE_DIR/logs/context-keeper-service.log"

# 选择正确的二进制文件名
if [ "$RUN_MODE" = "http" ]; then
    BINARY_NAME="context-keeper-http"
else
    BINARY_NAME="context-keeper"
fi

SERVICE_BIN="$CURSOR_DIR/$BINARY_NAME"
BUILD_BIN="$SOURCE_DIR/bin/$BINARY_NAME"

log_info "工作目录: $WORKSPACE_DIR"
log_info "开始重启 Context-Keeper 服务..."
log_info "运行模式: $RUN_MODE"

# 创建存储目录
log_info "确保存储目录存在: $STORAGE_DIR"
mkdir -p "$STORAGE_DIR"

# 确保日志目录存在
mkdir -p "$SOURCE_DIR/logs"

# 停止现有服务
log_info "停止运行中的 Context-Keeper 服务..."
pkill -f "context-keeper" 2>/dev/null || true
sleep 1

# 重新编译
log_info "重新编译服务 ($BINARY_NAME)..."
if [ "$RUN_MODE" = "http" ]; then
    ./scripts/build/build.sh --http
else
    ./scripts/build/build.sh --stdio
fi

if [ $? -ne 0 ]; then
    log_error "编译失败!"
    exit 1
fi

log_info "编译成功: $BUILD_BIN"

# 复制到Cursor目录
log_info "将新的二进制文件复制到 Cursor 目录: $SERVICE_BIN"
cp "$BUILD_BIN" "$SERVICE_BIN"

if [ $? -ne 0 ]; then
    log_error "复制失败!"
    exit 1
fi

# 确保执行权限
chmod +x "$SERVICE_BIN"
chmod +x "$BUILD_BIN"

# 验证MCP配置
if [ ! -f "$CURSOR_DIR/mcp.json" ]; then
    log_warn "未找到 MCP 配置文件: $CURSOR_DIR/mcp.json"
    log_info "创建默认配置..."
    
    # 创建配置文件
    cat > "$CURSOR_DIR/mcp.json" << EOF
{
  "mcpServers": {
    "context-keeper": {
      "command": "$SERVICE_BIN",
      "env": {
        "EMBEDDING_API_URL": "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings",
        "EMBEDDING_API_KEY": "sk-25be9b8a195145fb994f1d9b6ac26c82",
        "VECTOR_DB_URL": "https://vrs-cn-hic4apgaa00049.dashvector.cn-hangzhou.aliyuncs.com",
        "VECTOR_DB_API_KEY": "sk-ADaEA4GfQwNmWjbmZaavNiMwd6oUQ65FAD248FC2911EF83462E5ACEF33073",
        "VECTOR_DB_COLLECTION": "context_keeper",
        "VECTOR_DB_DIMENSION": "1536",
        "VECTOR_DB_METRIC": "cosine",
        "SIMILARITY_THRESHOLD": "0.35",
        "STORAGE_PATH": "$STORAGE_DIR",
        "DEBUG": "true",
        "RUN_MODE": "$RUN_MODE",
        "HTTP_SERVER_PORT": "$HTTP_SERVER_PORT"
      }
    }
  }
}
EOF
else
    log_info "MCP 配置文件已存在: $CURSOR_DIR/mcp.json"
    # 更新运行模式到现有的MCP配置
    if command -v jq &> /dev/null; then
        TEMP_FILE=$(mktemp)
        jq ".mcpServers[\"context-keeper\"].env.RUN_MODE = \"$RUN_MODE\" | .mcpServers[\"context-keeper\"].env.HTTP_SERVER_PORT = \"$HTTP_SERVER_PORT\"" "$CURSOR_DIR/mcp.json" > "$TEMP_FILE" && mv "$TEMP_FILE" "$CURSOR_DIR/mcp.json"
        log_info "已更新MCP配置文件中的运行模式为 $RUN_MODE"
    else
        log_warn "未找到jq工具，无法自动更新MCP配置，请手动设置RUN_MODE环境变量"
    fi
fi

# 设置环境变量
export RUN_MODE="$RUN_MODE"
export HTTP_SERVER_PORT="$HTTP_SERVER_PORT"

# 选择运行模式
if [ "$DAEMON_MODE" = true ]; then
    # 清空日志文件
    > "$LOG_FILE"

    # 后台模式：启动服务并重定向输出到日志文件
    log_info "以守护进程模式启动服务 ($RUN_MODE)..."
    nohup env EMBEDDING_API_URL="https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings" \
          EMBEDDING_API_KEY="sk-25be9b8a195145fb994f1d9b6ac26c82" \
          VECTOR_DB_URL="https://vrs-cn-hic4apgaa00049.dashvector.cn-hangzhou.aliyuncs.com" \
          VECTOR_DB_API_KEY="sk-ADaEA4GfQwNmWjbmZaavNiMwd6oUQ65FAD248FC2911EF83462E5ACEF33073" \
          VECTOR_DB_COLLECTION="context_keeper" \
          VECTOR_DB_DIMENSION="1536" \
          VECTOR_DB_METRIC="cosine" \
          SIMILARITY_THRESHOLD="0.35" \
          STORAGE_PATH="$STORAGE_DIR" \
          DEBUG="true" \
          RUN_MODE="$RUN_MODE" \
          HTTP_SERVER_PORT="$HTTP_SERVER_PORT" \
          "$SERVICE_BIN" > "$LOG_FILE" 2>&1 &

    # 获取进程ID
    PID=$!
    log_info "服务已启动，进程ID: $PID"

    # 等待服务启动
    sleep 2

    # 显示启动日志
    log_info "服务启动日志 (最近20行):"
    echo "---------------------------------------------------------"
    tail -n 20 "$LOG_FILE"
    echo "---------------------------------------------------------"

    # 检查服务是否正在运行
    if ps -p $PID > /dev/null 2>&1; then
        log_info "服务正在运行，可以重启Cursor应用程序来加载新的服务。"
        if [ "$RUN_MODE" = "http" ]; then
            log_info "HTTP服务地址: http://localhost:$HTTP_SERVER_PORT"
        fi
        log_info "完整日志文件位置: $LOG_FILE"
        log_info "使用以下命令查看实时日志: tail -f $LOG_FILE"
    else
        log_error "服务启动失败，请检查日志: $LOG_FILE"
    fi
else
    # 前台模式：直接运行服务，显示所有输出
    log_info "以前台模式启动服务 ($RUN_MODE)... 按 Ctrl+C 停止"
    if [ "$RUN_MODE" = "http" ]; then
        log_info "HTTP服务将在 http://localhost:$HTTP_SERVER_PORT 上运行"
    else
        log_info "STDIO服务将通过标准输入输出通信"
    fi
    log_info "在另一个终端中，请重启Cursor应用程序来测试服务。"
    echo "---------------------------------------------------------"
    EMBEDDING_API_URL="https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings" \
    EMBEDDING_API_KEY="sk-25be9b8a195145fb994f1d9b6ac26c82" \
    VECTOR_DB_URL="https://vrs-cn-hic4apgaa00049.dashvector.cn-hangzhou.aliyuncs.com" \
    VECTOR_DB_API_KEY="sk-ADaEA4GfQwNmWjbmZaavNiMwd6oUQ65FAD248FC2911EF83462E5ACEF33073" \
    VECTOR_DB_COLLECTION="context_keeper" \
    VECTOR_DB_DIMENSION="1536" \
    VECTOR_DB_METRIC="cosine" \
    SIMILARITY_THRESHOLD="0.35" \
    STORAGE_PATH="$STORAGE_DIR" \
    DEBUG="true" \
    RUN_MODE="$RUN_MODE" \
    HTTP_SERVER_PORT="$HTTP_SERVER_PORT" \
    "$SERVICE_BIN"
    echo "---------------------------------------------------------"
fi

log_info "完成!" 