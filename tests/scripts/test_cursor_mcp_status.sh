#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 设置工作目录为项目根目录
cd "$(dirname "$0")/../.." || exit 1
WORKSPACE_DIR=$(pwd)
echo -e "${GREEN}工作目录: ${WORKSPACE_DIR}${NC}"

echo -e "${YELLOW}测试Context-Keeper MCP连接状态...${NC}"

# 验证SSE连接
echo -e "\n${YELLOW}1. 验证SSE连接...${NC}"
echo "尝试连接到SSE端点，将在5秒后自动终止..."
curl -N -H "Accept: text/event-stream" http://localhost:8088/sse & 
PID=$!
sleep 5
kill $PID 2>/dev/null

# 检查服务器日志
echo -e "\n${YELLOW}2. 检查最新的10行服务器日志...${NC}"
tail -n 10 logs/service.log

# 查看服务状态
echo -e "\n${YELLOW}3. 验证服务健康状态...${NC}"
HEALTH_RESPONSE=$(curl -s http://localhost:8088/health)
echo "健康状态: $HEALTH_RESPONSE"

# 查看所有路由
echo -e "\n${YELLOW}4. 获取所有API路由...${NC}"
curl -s http://localhost:8088/api/routes

# 验证服务器能否接收正常的API请求
echo -e "\n${YELLOW}5. 验证服务能否接收正常API请求...${NC}"
# 创建测试会话ID
TEST_SESSION="test-cursor-mcp-$(date +%s)"

# 尝试关联文件
ASSOCIATE_RESPONSE=$(curl -s -X POST "http://localhost:8088/api/mcp/context-keeper/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$TEST_SESSION"'",
    "filePath": "/test/mcp_test.js",
    "content": "console.log(\"Testing MCP connection\");"
  }')
echo "文件关联响应: $ASSOCIATE_RESPONSE"

echo -e "\n${GREEN}测试完成！${NC}"
echo -e "${YELLOW}如果看到文件关联成功的响应，表示API服务正常运行。${NC}"
echo -e "${YELLOW}如果看到SSE连接输出了manifest事件，表示SSE连接正常。${NC}"
echo -e "${YELLOW}如果服务日志显示MCP连接请求，表示Cursor正在尝试连接。${NC}" 