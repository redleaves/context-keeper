#!/bin/bash

# 会话管理和MCP协议支持功能测试脚本
# 用于测试会话管理、代码文件关联和编辑记录等功能

# 设置基础URL
BASE_URL="http://localhost:8088/api/mcp/context-keeper"

# 生成会话ID
SESSION_ID="test-session-$(date +%s)"
echo "使用会话ID: $SESSION_ID"

# 1. 测试存储消息
echo -e "\n测试1：存储消息"
STORE_RESPONSE=$(curl -s -X POST "$BASE_URL/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "messages": [
      {
        "role": "user",
        "content": "请帮我实现一个简单的Node.js服务器",
        "contentType": "text",
        "priority": "P2"
      },
      {
        "role": "assistant",
        "content": "好的，下面是一个简单的Node.js HTTP服务器实现:\n\n```javascript\nconst http = require(\"http\");\nconst server = http.createServer((req, res) => {\n  res.writeHead(200, {\"Content-Type\": \"text/plain\"});\n  res.end(\"Hello World\");\n});\nserver.listen(3000, () => {\n  console.log(\"Server running at http://localhost:3000/\");\n});\n```\n\n你可以将这段代码保存为app.js，然后运行 `node app.js` 启动服务器。",
        "contentType": "text",
        "priority": "P1"
      }
    ]
  }')

echo "$STORE_RESPONSE"

# 提取消息ID
MESSAGE_IDS=$(echo "$STORE_RESPONSE" | grep -o '"messageIds":\[[^]]*\]' | cut -d':' -f2)
echo "消息ID: $MESSAGE_IDS"

# 2. 测试存储并汇总消息（带批次ID）
echo -e "\n测试2：存储并汇总消息（带批次ID）"
BATCH_ID="batch-$(date +%s)"
SUMMARIZE_RESPONSE=$(curl -s -X POST "$BASE_URL/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "summarizeAndStore": true,
    "batchId": "'$BATCH_ID'",
    "messages": [
      {
        "role": "user",
        "content": "这个服务器能处理POST请求吗？",
        "contentType": "text",
        "priority": "P2"
      },
      {
        "role": "assistant",
        "content": "刚才的例子只处理了基本请求，没有区分GET和POST。下面是支持POST请求的版本:\n\n```javascript\nconst http = require(\"http\");\n\nconst server = http.createServer((req, res) => {\n  if (req.method === \"POST\") {\n    let body = \"\";\n    req.on(\"data\", chunk => {\n      body += chunk.toString();\n    });\n    req.on(\"end\", () => {\n      res.writeHead(200, {\"Content-Type\": \"application/json\"});\n      res.end(JSON.stringify({ message: \"POST请求成功\", data: body }));\n    });\n  } else {\n    res.writeHead(200, {\"Content-Type\": \"text/plain\"});\n    res.end(\"Hello World\");\n  }\n});\n\nserver.listen(3000, () => {\n  console.log(\"Server running at http://localhost:3000/\");\n});\n```\n\n这个版本可以处理POST请求，并将提交的数据返回给客户端。",
        "contentType": "text",
        "priority": "P1"
      }
    ]
  }')

echo "$SUMMARIZE_RESPONSE"

# 提取内存ID
MEMORY_ID=$(echo "$SUMMARIZE_RESPONSE" | grep -o '"memoryId":"[^"]*"' | cut -d':' -f2 | tr -d '\"')
echo "记忆ID: $MEMORY_ID"

# 3. 测试会话状态查询
echo -e "\n测试3：会话状态查询"
curl -s -X GET "$BASE_URL/sessionState?sessionId=$SESSION_ID"

# 4. 测试代码文件关联
echo -e "\n测试4：代码文件关联"
curl -s -X POST "$BASE_URL/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/server/app.js",
    "language": "javascript",
    "content": "const http = require(\"http\");\n\nconst server = http.createServer((req, res) => {\n  if (req.method === \"POST\") {\n    let body = \"\";\n    req.on(\"data\", chunk => {\n      body += chunk.toString();\n    });\n    req.on(\"end\", () => {\n      res.writeHead(200, {\"Content-Type\": \"application/json\"});\n      res.end(JSON.stringify({ message: \"POST请求成功\", data: body }));\n    });\n  } else {\n    res.writeHead(200, {\"Content-Type\": \"text/plain\"});\n    res.end(\"Hello World\");\n  }\n});\n\nserver.listen(3000, () => {\n  console.log(\"Server running at http://localhost:3000/\");\n});"
  }'

# 5. 测试编辑记录
echo -e "\n测试5：编辑记录"
curl -s -X POST "$BASE_URL/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/server/app.js",
    "type": "modify",
    "position": 450,
    "content": "加入了错误处理逻辑"
  }'

# 6. 测试上下文检索（包含短期和长期记忆）
echo -e "\n测试6：上下文检索"
curl -s -X POST "$BASE_URL/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "Node.js服务器POST请求示例"
  }'

# 7. 测试通过批次ID检索
echo -e "\n测试7：通过批次ID检索"
curl -s -X POST "$BASE_URL/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "batchId": "'$BATCH_ID'"
  }'

# 8. 测试通过记忆ID检索
echo -e "\n测试8：通过记忆ID检索"
if [ ! -z "$MEMORY_ID" ]; then
  curl -s -X POST "$BASE_URL/retrieveContext" \
    -H "Content-Type: application/json" \
    -d '{
      "memoryId": "'$MEMORY_ID'"
    }'
else
  echo "跳过测试：未获取到记忆ID"
fi

echo -e "\n测试完成！" 