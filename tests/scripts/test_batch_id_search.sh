#!/bin/bash

# 批次ID检索测试脚本
# 测试通过批次ID对存储的上下文进行过滤

BASE_URL="http://localhost:8088"
SESSION_ID="batch-test-$(date +%s)"
BATCH_ID="test-batch-$(date +%s)"

echo "测试批次ID: $BATCH_ID"

# 1. 存储带有批次ID的上下文
curl -s -X POST "$BASE_URL/api/mcp/context-keeper/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "content": "这是一条测试批次ID功能的消息，它属于批次'$BATCH_ID'",
    "type": "conversation_summary",
    "priority": "P1",
    "metadata": {
      "batchId": "'$BATCH_ID'",
      "timestamp": '$(date +%s)',
      "source": "batch_test"
    }
  }' > /dev/null

# 等待索引更新
echo "等待2秒让索引更新..."
sleep 2

# 2. 通过批次ID检索
RESULT=$(curl -s -X POST "$BASE_URL/api/mcp/context-keeper/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "批次ID测试",
    "metadata": {
      "batchId": "'$BATCH_ID'"
    },
    "skip_threshold": true
  }')

# 检查结果
if echo "$RESULT" | grep -q "$BATCH_ID"; then
  echo "成功：批次ID检索正常工作"
  echo "$RESULT"
else
  echo "{"details":"通过批次ID检索失败: $RESULT","error":"检索上下文失败"}"
fi

echo "测试完成!" 