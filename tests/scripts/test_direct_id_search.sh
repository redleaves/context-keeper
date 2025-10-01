#!/bin/bash

# 设置基础URL
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 使用之前存储的记忆ID进行测试
MEMORY_ID="c238fc9f-8f3f-4877-9b31-ba50b9c84fc4"
echo "测试记忆ID: $MEMORY_ID"

# 测试1：直接通过记忆ID检索
echo -e "\n测试1: 通过记忆ID检索"
curl -s -X POST "${BASE_URL}/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "memoryId": "'"${MEMORY_ID}"'"
  }' | head -c 500

echo -e "\n\n测试完成!" 