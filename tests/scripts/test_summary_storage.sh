#!/bin/bash

# 设置基础URL
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 生成一个唯一的会话ID
SESSION_ID="session-$(date +%s)"
echo "使用会话ID: $SESSION_ID"

# 存储对话总结到向量数据库
echo "正在存储对话总结..."

# 对话总结内容 - 生成为单行文本，避免JSON格式问题
SUMMARY="Context-Keeper项目开发总结：已完成V1基础版本(向量存储与检索、会话管理、MCP协议支持)；计划V2版本从向量数据库迁移至MongoDB；当前将多轮对话内容汇总提取关键信息后存储到向量数据库；V2计划按照消息集合形式存储完整对话历史；API实现包括handleStoreMessages和handleRetrieveConversation处理函数；存储策略调整为V1阶段汇总存储、V2计划消息集合存储；保留现有消息集合相关代码逻辑；选择MongoDB作为V2阶段存储引擎(优势：开源免费、支持压缩、文档存储灵活、支持TTL索引)；当前savepoint功能效率不高，缺乏一键存储，多轮会话存储可能导致上下文碎片化；改进方案包括实时增量存储、一键完整存储选项、智能混合存储策略、提高消息队列处理能力、优化向量生成和存储过程。"

# 存储对话总结
RESPONSE=$(curl -s -X POST "${BASE_URL}/storeContext" \
  -H "Content-Type: application/json" \
  -d "{
    \"sessionId\": \"${SESSION_ID}\",
    \"content\": \"${SUMMARY}\",
    \"priority\": \"P1\",
    \"metadata\": {
      \"type\": \"conversation_summary\",
      \"timestamp\": $(date +%s),
      \"batchId\": \"batch-$(date +%s)\"
    }
  }")

echo "存储响应: $RESPONSE"
CONTEXT_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "获得上下文ID: $CONTEXT_ID"

# 等待索引更新
echo "等待2秒让索引更新..."
sleep 2

# 验证1：通过关键词检索
echo -e "\n验证1：通过关键词检索"
KEYWORDS=("MongoDB" "savepoint" "消息集合" "API" "V2")

for KEYWORD in "${KEYWORDS[@]}"; do
  echo -e "\n检索关键词: '$KEYWORD'"
  curl -s -X POST "${BASE_URL}/retrieveContext" \
    -H "Content-Type: application/json" \
    -d "{
      \"sessionId\": \"${SESSION_ID}\",
      \"query\": \"${KEYWORD}\",
      \"limit\": 1000
    }" | jq .
done

# 验证2：通过会话ID检索
echo -e "\n验证2：通过会话ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d "{
    \"sessionId\": \"${SESSION_ID}\",
    \"query\": \"*\",
    \"limit\": 2000
  }" | jq .

# 获取记忆ID
MEMORY_ID=$(echo "$RESPONSE" | grep -o '"memoryId":"[^"]*"' | cut -d'"' -f4)
if [ ! -z "$MEMORY_ID" ]; then
  echo -e "\n验证3：通过记忆ID检索"
  echo "使用记忆ID: $MEMORY_ID"
  curl -s -X POST "${BASE_URL}/retrieveContext" \
    -H "Content-Type: application/json" \
    -d "{
      \"memoryId\": \"${MEMORY_ID}\",
      \"limit\": 2000
    }" | jq .
fi

# 验证4：基于批次ID检索
BATCH_ID="batch-$(date +%s)"
echo -e "\n存储具有显式批次ID的第二条记录..."
curl -s -X POST "${BASE_URL}/storeContext" \
  -H "Content-Type: application/json" \
  -d "{
    \"sessionId\": \"${SESSION_ID}\",
    \"content\": \"这是批次${BATCH_ID}的附加内容，用于测试批次ID功能。\",
    \"priority\": \"P2\",
    \"metadata\": {
      \"type\": \"conversation_summary\",
      \"timestamp\": $(date +%s),
      \"batchId\": \"${BATCH_ID}\"
    }
  }" | jq .

sleep 2

echo -e "\n验证4：通过批次ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d "{
    \"sessionId\": \"${SESSION_ID}\",
    \"query\": \"${BATCH_ID}\",
    \"limit\": 2000
  }" | jq .

echo -e "\n测试完成" 