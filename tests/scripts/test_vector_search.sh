#!/bin/bash

# 上下文向量检索测试脚本
# 主要测试向量搜索功能的准确性和阈值控制

BASE_URL="http://localhost:8088"
SESSION_ID="vector-test-$(date +%s)"
TIMESTAMP=$(date +%s)

echo "测试文字向量检索功能"

# 1. 首先关联一个文件，并写入一些内容
echo "步骤1: 关联文件并创建内容"
curl -s -X POST "$BASE_URL/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/vector_test.js",
    "language": "javascript",
    "content": "// 向量数据库测试\nconst vectorService = {\n  addDocument: async (collection, document, embedding) => {\n    console.log(`向量集合${collection}添加文档`);\n    return { id: \"doc-123\", status: \"success\" };\n  },\n  searchSimilar: async (collection, query, options) => {\n    console.log(`查询集合${collection}，阈值=${options.threshold}`);\n    return { hits: [ /* 相似文档 */ ] };\n  }\n};"
  }' > /dev/null

# 2. 向会话中添加一条关于向量搜索的消息
echo "步骤2: 添加向量搜索相关消息"
curl -s -X POST "$BASE_URL/api/mcp/context-keeper/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "content": "向量搜索是一种基于语义相似度的内容检索方法，它将文本转换为多维向量，然后计算向量间的距离或相似度来确定内容关联性。在上下文保持服务中，相似度阈值是一个重要参数，它决定了返回结果的相关度。",
    "type": "conversation_summary",
    "priority": "P1"
  }' > /dev/null

# 等待索引更新
echo "等待1秒钟让索引更新..."
sleep 1

# 3. 测试标准向量检索
echo "测试: 向量语义检索"
curl -s -X POST "$BASE_URL/api/mcp/context-keeper/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "语义相似度检索的原理是什么？",
    "skip_threshold": true
  }'

# 4. 测试带显式阈值的向量检索
echo -e "\n\n测试: 自定义阈值向量检索"
curl -s -X POST "$BASE_URL/api/mcp/context-keeper/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "向量距离和相似度计算",
    "similarity_threshold": 0.8,
    "skip_threshold": true
  }'

echo -e "\n测试完成!"