#!/bin/bash

# 设置基础URL
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 首先存储一条测试消息
SESSION_ID="test-message-id-$(date +%s)"
echo "测试会话ID: $SESSION_ID"

# 存储消息
echo -e "\n存储测试消息"
RESPONSE=$(curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "user",
        "content": "这是一条测试消息，用于测试通过消息ID检索",
        "contentType": "text",
        "priority": "high"
      }
    ]
  }')

echo "存储响应: $RESPONSE"

# 从响应中提取消息ID - 修复提取逻辑
MESSAGE_ID=$(echo $RESPONSE | grep -o '"messageIds":\["[^"]*"' | sed 's/.*\["//;s/".*//')
echo "提取的消息ID: $MESSAGE_ID"

# 等待索引更新 (10秒)
echo "等待索引更新 (10秒)..."
sleep 10

# 测试1：通过会话ID检索对话
echo -e "\n测试1: 通过会话ID检索对话"
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'"
  }' | head -c 500

# 测试2：通过消息ID检索对话
echo -e "\n\n测试2: 通过消息ID检索对话"
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "messageId": "'"${MESSAGE_ID}"'"
  }' | head -c 500

echo -e "\n\n测试完成!" 