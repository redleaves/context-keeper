#!/bin/bash

# Context-Keeper Docker 入口点脚本
# 完全模拟 ./scripts/manage.sh deploy http 的启动过程
# 支持多种运行模式和环境配置

set -e

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认值（与manage.sh保持一致）
DEFAULT_RUN_MODE="http"
DEFAULT_HTTP_PORT="8088"
DEFAULT_LOG_LEVEL="info"

echo -e "${BLUE}Context-Keeper Docker 容器启动中...${NC}"
echo -e "${YELLOW}模拟 ./scripts/manage.sh deploy http 启动过程${NC}"

# 显示环境信息
echo -e "${GREEN}运行环境信息:${NC}"
echo "  容器ID: $(hostname)"
echo "  时区: ${TZ:-UTC}"
echo "  工作目录: $(pwd)"
echo "  用户: $(whoami)"
echo "  项目根目录: ${PROJECT_ROOT:-/app}"

# 解析命令行参数（模拟manage.sh的参数解析）
RUN_MODE="${1:-${RUN_MODE:-$DEFAULT_RUN_MODE}}"
HTTP_SERVER_PORT="${HTTP_SERVER_PORT:-$DEFAULT_HTTP_PORT}"
LOG_LEVEL="${LOG_LEVEL:-$DEFAULT_LOG_LEVEL}"

# 验证运行模式（与manage.sh一致）
if [ "$RUN_MODE" != "http" ] && [ "$RUN_MODE" != "stdio" ]; then
    echo -e "${RED}错误: 无效的运行模式 '$RUN_MODE'${NC}"
    echo -e "${YELLOW}支持的模式: http, stdio${NC}"
    exit 1
fi

# 显示配置信息（模拟manage.sh的配置显示）
echo -e "${GREEN}服务配置 (模拟manage.sh deploy):${NC}"
echo "  运行模式: $RUN_MODE"
if [ "$RUN_MODE" = "http" ]; then
    echo "  HTTP端口: $HTTP_SERVER_PORT"
fi
echo "  日志级别: $LOG_LEVEL"
echo "  存储路径: ${STORAGE_PATH:-/app/data}"
echo "  PID目录: ${PID_DIR:-/app/logs}"

# 检查必要的目录（模拟manage.sh的目录检查）
echo -e "${YELLOW}检查目录权限（模拟manage.sh）...${NC}"
for dir in /app/data /app/logs /app/config; do
    if [ ! -d "$dir" ]; then
        echo -e "${YELLOW}创建目录: $dir${NC}"
        mkdir -p "$dir"
    fi
    if [ ! -w "$dir" ]; then
        echo -e "${RED}警告: 目录 $dir 不可写${NC}"
    fi
done

# 🔥 关键修复：使用与manage.sh完全相同的二进制文件选择逻辑
echo -e "${YELLOW}选择二进制文件（模拟manage.sh逻辑）...${NC}"
if [ "$RUN_MODE" = "stdio" ]; then
    # 模拟manage.sh中的路径选择：./bin/context-keeper
    BINARY="/app/bin/context-keeper"
    echo -e "${GREEN}使用 STDIO 模式二进制文件: $BINARY${NC}"
elif [ "$RUN_MODE" = "http" ]; then
    # 模拟manage.sh中的路径选择：./bin/context-keeper-http
    BINARY="/app/bin/context-keeper-http"
    echo -e "${GREEN}使用 HTTP 模式二进制文件: $BINARY${NC}"
else
    echo -e "${RED}错误: 无效的模式: $RUN_MODE${NC}"
    exit 1
fi

# 检查二进制文件（模拟manage.sh的检查逻辑）
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}错误: 二进制文件不存在: $BINARY${NC}"
    echo -e "${YELLOW}可用文件:${NC}"
    ls -la /app/bin/ 2>/dev/null || echo "  bin目录不存在"
    ls -la /app/context-keeper* 2>/dev/null || echo "  根目录下无context-keeper文件"
    exit 1
fi

if [ ! -x "$BINARY" ]; then
    echo -e "${RED}错误: 二进制文件不可执行: $BINARY${NC}"
    exit 1
fi

# 🔥 设置环境变量（完全模拟manage.sh的环境设置）
echo -e "${YELLOW}设置环境变量（模拟manage.sh）...${NC}"
export RUN_MODE="$RUN_MODE"
if [ "$RUN_MODE" = "http" ]; then
export HTTP_SERVER_PORT="$HTTP_SERVER_PORT"
fi
export LOG_LEVEL="$LOG_LEVEL"
export STORAGE_PATH="${STORAGE_PATH:-/app/data}"
export PID_DIR="${PID_DIR:-/app/logs}"

echo -e "${GREEN}环境变量设置完成:${NC}"
echo "  RUN_MODE=$RUN_MODE"
if [ "$RUN_MODE" = "http" ]; then
    echo "  HTTP_SERVER_PORT=$HTTP_SERVER_PORT"
fi
echo "  LOG_LEVEL=$LOG_LEVEL"
echo "  STORAGE_PATH=$STORAGE_PATH"

# 信号处理函数（Docker容器优雅停止）
cleanup() {
    echo -e "\n${YELLOW}接收到停止信号，服务将优雅关闭${NC}"
    exit 0
}

# 注册信号处理
trap cleanup SIGTERM SIGINT

# 🔥 显示启动信息（模拟manage.sh的启动信息）
echo -e "${BLUE}启动 Context-Keeper 服务（模拟manage.sh start）...${NC}"
echo "二进制文件: $BINARY"
echo "运行模式: $RUN_MODE"
if [ "$RUN_MODE" = "http" ]; then
    echo "监听端口: $HTTP_SERVER_PORT"
fi

# 如果是HTTP模式，显示端口信息（模拟manage.sh的HTTP模式处理）
if [ "$RUN_MODE" = "http" ]; then
    echo -e "${YELLOW}HTTP模式将在端口 $HTTP_SERVER_PORT 上监听${NC}"
    echo -e "${GREEN}健康检查端点: http://localhost:$HTTP_SERVER_PORT/health${NC}"
    echo -e "${GREEN}MCP通信端点: http://localhost:$HTTP_SERVER_PORT/mcp${NC}"
fi

# ✅ 日志输出简化：HTTP/WebSocket模式的日志已直接输出到标准输出
# 不再需要复杂的日志监控脚本，云端可以直接通过 docker logs 查看业务日志
echo -e "${GREEN}✓ 日志模式：HTTP/WebSocket服务日志直接输出到标准输出${NC}"

# 🔥 启动服务（直接运行二进制文件，不通过manage.sh）
echo -e "${GREEN}Docker模式：直接启动服务，日志输出到标准输出${NC}"

# 直接运行二进制文件，不使用nohup，让日志输出到标准输出
if [ $# -gt 1 ]; then
    shift  # 移除第一个参数（运行模式）
    exec "$BINARY" "$@"
else
    exec "$BINARY"
fi 