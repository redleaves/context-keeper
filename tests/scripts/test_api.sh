#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # 无颜色

# 设置工作目录为项目根目录
cd "$(dirname "$0")/../.." || exit 1
WORKSPACE_DIR=$(pwd)
echo -e "${GREEN}工作目录: ${WORKSPACE_DIR}${NC}"

# 测试API端点
API_BASE="http://localhost:8088/api"
SESSION_ID="test-session-$(date +%s)"

echo -e "${GREEN}开始API测试...${NC}"
echo -e "${YELLOW}使用会话ID: ${SESSION_ID}${NC}"

# 会话管理API测试
echo -e "\n${GREEN}1. 测试会话管理API${NC}"
curl -s -X POST "${API_BASE}/session/create" \
     -H "Content-Type: application/json" \
     -d "{\"sessionId\":\"${SESSION_ID}\",\"metadata\":{\"test\":true}}" | jq

# 上下文存储API测试
echo -e "\n${GREEN}2. 测试上下文存储API${NC}"
curl -s -X POST "${API_BASE}/context/store" \
     -H "Content-Type: application/json" \
     -d "{\"sessionId\":\"${SESSION_ID}\",\"content\":\"这是一条测试上下文信息。\"}" | jq

# 上下文检索API测试
echo -e "\n${GREEN}3. 测试上下文检索API${NC}"
curl -s -X POST "${API_BASE}/context/retrieve" \
     -H "Content-Type: application/json" \
     -d "{\"sessionId\":\"${SESSION_ID}\",\"query\":\"测试上下文\"}" | jq

# 消息存储API测试
echo -e "\n${GREEN}4. 测试消息存储API${NC}"
curl -s -X POST "${API_BASE}/conversation/store" \
     -H "Content-Type: application/json" \
     -d "{\"sessionId\":\"${SESSION_ID}\",\"messages\":[{\"role\":\"user\",\"content\":\"这是用户消息\"},{\"role\":\"assistant\",\"content\":\"这是助手回复\"}]}" | jq

# 消息检索API测试
echo -e "\n${GREEN}5. 测试消息检索API${NC}"
curl -s -X POST "${API_BASE}/conversation/retrieve" \
     -H "Content-Type: application/json" \
     -d "{\"sessionId\":\"${SESSION_ID}\",\"query\":\"用户消息\"}" | jq

# 获取会话信息API测试
echo -e "\n${GREEN}6. 测试获取会话信息API${NC}"
curl -s -X GET "${API_BASE}/session/${SESSION_ID}" | jq

# 使用测试数据测试摘要存储
echo -e "\n${GREEN}7. 测试摘要存储API${NC}"
curl -s -X POST "${API_BASE}/conversation/store" \
     -H "Content-Type: application/json" \
     -d @tests/data/test_memory_store.json | jq

echo -e "\n${GREEN}API测试完成${NC}" 