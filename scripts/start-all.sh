#!/bin/bash

# Context-Keeper 一键启动脚本
# 启动服务器 + 本地扩展客户端

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 Context-Keeper 一键启动${NC}"
echo ""

# 1. 启动服务器
echo -e "${YELLOW}📡 启动服务器...${NC}"
./scripts/manage.sh deploy http websocket

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 服务器启动失败${NC}"
    exit 1
fi

echo ""

# 2. 启动本地扩展客户端
echo -e "${YELLOW}🔌 启动本地扩展客户端...${NC}"

if [ -f "$HOME/.context-keeper/start-extension.js" ]; then
    cd "$HOME/.context-keeper"
    
    # 检查是否已经在运行
    if pgrep -f "start-extension.js" > /dev/null; then
        echo -e "${YELLOW}⚠️  本地扩展客户端已在运行中${NC}"
    else
        # 后台启动
        nohup node start-extension.js > logs/extension.log 2>&1 &
        EXTENSION_PID=$!
        echo -e "${GREEN}✅ 本地扩展客户端已启动 (PID: $EXTENSION_PID)${NC}"
        
        # 保存PID
        echo $EXTENSION_PID > logs/extension.pid
        
        # 等待一下确保启动成功
        sleep 2
        
        if ps -p $EXTENSION_PID > /dev/null; then
            echo -e "${GREEN}✅ 本地扩展客户端运行正常${NC}"
        else
            echo -e "${RED}❌ 本地扩展客户端启动失败${NC}"
            cat logs/extension.log
        fi
    fi
else
    echo -e "${YELLOW}⚠️  未找到本地扩展客户端，请先运行安装脚本${NC}"
    echo -e "${YELLOW}   安装命令: cd cursor-integration && ./install.sh${NC}"
fi

echo ""

# 3. 显示状态
echo -e "${BLUE}📊 系统状态:${NC}"
cd "$PROJECT_ROOT"
./scripts/manage.sh status

echo ""
echo -e "${GREEN}🎉 Context-Keeper 启动完成！${NC}"
echo ""
echo -e "${YELLOW}📖 接下来的步骤:${NC}"
echo "1. 确保 Cursor 已重启（以加载 MCP 配置）"
echo "2. 在 Cursor 中开始使用 Context-Keeper 工具"
echo "3. 查看日志: tail -f ~/.context-keeper/logs/extension.log"
echo ""
echo -e "${YELLOW}🛑 停止服务:${NC}"
echo "./scripts/stop-all.sh" 