#!/bin/bash

# Context-Keeper 一键停止脚本
# 停止服务器 + 本地扩展客户端

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🛑 Context-Keeper 一键停止${NC}"
echo ""

# 1. 停止本地扩展客户端
echo -e "${YELLOW}🔌 停止本地扩展客户端...${NC}"

# 通过进程名查找并停止
if pgrep -f "start-extension.js" > /dev/null; then
    pkill -f "start-extension.js"
    echo -e "${GREEN}✅ 本地扩展客户端已停止${NC}"
else
    echo -e "${YELLOW}⚠️  本地扩展客户端未运行${NC}"
fi

# 清理PID文件
if [ -f "$HOME/.context-keeper/logs/extension.pid" ]; then
    rm -f "$HOME/.context-keeper/logs/extension.pid"
fi

echo ""

# 2. 停止服务器
echo -e "${YELLOW}📡 停止服务器...${NC}"
./scripts/manage.sh stop all

echo ""
echo -e "${GREEN}🎉 Context-Keeper 已完全停止${NC}" 