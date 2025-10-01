#!/bin/bash

echo "创建带有批次ID的测试记录"

# 定义API端点
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 创建一个测试会话ID，使用当前时间戳确保唯一性
SESSION_ID="test-session-$(date +%s)"
BATCH_ID="test-batch-2"

echo "使用会话ID: ${SESSION_ID} 和批次ID: ${BATCH_ID}"

# 发送存储上下文请求，包含批次ID元数据
curl -s -X POST "${BASE_URL}/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "content": "这是一条测试记录，用于验证批次ID搜索功能。批次ID为'"${BATCH_ID}"'。",
    "metadata": {
      "batchId": "'"${BATCH_ID}"'",
      "testTime": "'"$(date)"'"
    },
    "priority": "P1"
  }'

echo -e "\n记录创建完成!"
