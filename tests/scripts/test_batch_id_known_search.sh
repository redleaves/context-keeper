#!/bin/bash

echo "测试批次ID: test-batch-2"

# 发送请求到API端点
curl -s -X POST http://localhost:8088/api/mcp/context-keeper/retrieveContext \
  -H "Content-Type: application/json" \
  -d '{"batchId": "test-batch-2", "sessionId": "test-session"}'

echo -e "\n测试完成!"
