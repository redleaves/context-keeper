#!/bin/bash

echo "测试批次ID直接字段匹配"

# 定义API端点
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 添加颜色支持
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 测试方法一: 使用 batchId 字段直接匹配
echo -e "${GREEN}方法一: 使用 batchId = 'test-batch-2' 过滤条件${NC}"
curl -s -X POST "${BASE_URL}/direct-filter" \
  -H "Content-Type: application/json" \
  -d '{
    "filter": "batchId = \"test-batch-2\"",
    "topk": 10,
    "include_vector": false
  }'

echo -e "\n"

# 测试方法二: 使用 metadata.batchId 路径匹配
echo -e "${GREEN}方法二: 使用 metadata.batchId = 'test-batch-2' 过滤条件${NC}"
curl -s -X POST "${BASE_URL}/direct-filter" \
  -H "Content-Type: application/json" \
  -d '{
    "filter": "metadata.batchId = \"test-batch-2\"",
    "topk": 10,
    "include_vector": false
  }'

echo -e "\n测试完成!"
