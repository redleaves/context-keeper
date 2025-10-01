#!/bin/bash

# 设置基础URL
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 生成一个唯一的会话ID
SESSION_ID="test-session-$(date +%s)"
echo "使用会话ID: $SESSION_ID"

# 存储测试消息
echo "正在存储测试消息..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "content": "这是一条测试消息，用于验证ID检索功能",
    "priority": "P1",
    "metadata": {
      "type": "test_message",
      "batchId": "test-batch-1"
    }
  }')

echo "存储响应: $RESPONSE"
MEMORY_ID=$(echo $RESPONSE | grep -o '"memoryId":"[^"]*"' | cut -d'"' -f4)
echo "获得记忆ID: $MEMORY_ID"

# 等待索引更新
echo "等待索引更新..."
sleep 5

# 测试1：通过记忆ID检索
echo -e "\n测试1: 通过记忆ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "memoryId": "'"${MEMORY_ID}"'"
  }' | jq

# 测试2：通过会话ID检索
echo -e "\n测试2: 通过会话ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "query": "会话检索"
  }' | jq

# 测试3：通过批次ID检索
echo -e "\n测试3: 通过批次ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "batchId": "test-batch-1"
  }' | jq

# 存储一批对话消息
echo -e "\n存储一批对话消息..."
MESSAGES_RESPONSE=$(curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "user",
        "content": "这是用户的测试消息",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "batchId": "test-batch-2",
          "type": "test"
        }
      },
      {
        "role": "assistant",
        "content": "这是助手的回复消息",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "batchId": "test-batch-2",
          "type": "test"
        }
      }
    ]
  }')

echo "存储消息响应: $MESSAGES_RESPONSE"
MESSAGE_IDS=$(echo $MESSAGES_RESPONSE | grep -o '"messageIds":\[[^]]*\]' | cut -d':' -f2)
FIRST_MESSAGE_ID=$(echo $MESSAGE_IDS | grep -o '"[^"]*"' | head -1 | tr -d '"')

echo "第一条消息ID: $FIRST_MESSAGE_ID"

# 等待索引更新
sleep 5

# 测试4：通过消息ID检索
echo -e "\n测试4: 通过消息ID检索对话"
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "messageId": "'"${FIRST_MESSAGE_ID}"'"
  }' | jq

# 测试5：通过批次ID检索对话
echo -e "\n测试5: 通过批次ID检索对话"
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "batchId": "test-batch-2"
  }' | jq

# 测试6：使用跳过相似度阈值的选项进行检索
echo -e "\n测试6: 使用跳过相似度阈值选项进行检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"${SESSION_ID}"'",
    "query": "无关查询词",
    "skipThreshold": true
  }' | jq

echo -e "\n测试完成!" 