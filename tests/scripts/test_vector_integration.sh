#!/bin/bash

# 向量搜索与会话管理集成测试脚本
# 测试向量存储与会话管理功能的集成

# 设置基础URL
BASE_URL="http://localhost:8088/api"
SESSION_ID="vector-test-$(date +%s)"
TIMESTAMP=$(date +%s)

echo "使用会话ID: $SESSION_ID"

# 1. 测试向量存储 - 直接存储一条内容
echo -e "\n测试1：直接存储内容到向量数据库"
VECTOR_STORE_RESPONSE=$(curl -s -X POST "$BASE_URL/vector/store" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Node.js是一个基于Chrome V8引擎的JavaScript运行环境，使开发者能够使用JavaScript来编写服务器端代码。它采用事件驱动、非阻塞I/O模型，使其轻量且高效，特别适合处理数据密集型的实时应用程序。",
    "metadata": {
      "sessionId": "'$SESSION_ID'",
      "contentType": "knowledge",
      "timestamp": "'$TIMESTAMP'",
      "topic": "Node.js基础"
    }
  }')

echo "$VECTOR_STORE_RESPONSE"

# 提取返回的ID
VECTOR_ID=$(echo "$VECTOR_STORE_RESPONSE" | grep -o '"id":[^,}]*' | cut -d':' -f2 | tr -d '\" ')
echo "向量ID: $VECTOR_ID"

# 2. 通过会话管理API存储消息，并验证它们同时存入向量数据库
echo -e "\n测试2：通过会话管理API存储消息"
SESSION_STORE_RESPONSE=$(curl -s -X POST "$BASE_URL/mcp/context-keeper/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "messages": [
      {
        "role": "user",
        "content": "Express.js和Koa.js有什么区别？",
        "contentType": "text",
        "priority": "P2"
      },
      {
        "role": "assistant",
        "content": "Express.js和Koa.js都是基于Node.js的Web框架，但有以下主要区别：\n\n1. **中间件系统**：\n   - Express使用回调函数的中间件系统\n   - Koa使用更现代的基于async/await的中间件系统，避免回调地狱\n\n2. **API设计**：\n   - Express API更丰富，内置了更多功能\n   - Koa API更精简，核心功能少，主要依赖第三方中间件\n\n3. **错误处理**：\n   - Koa的错误处理更优雅，支持try/catch捕获异步错误\n   - Express需要使用next(err)传递错误\n\n4. **请求/响应对象**：\n   - Koa的ctx对象集成了请求和响应对象\n   - Express分别使用req和res对象\n\n5. **生态系统**：\n   - Express生态系统更成熟和广泛\n   - Koa更现代但生态系统相对小一些",
        "contentType": "text",
        "priority": "P1"
      }
    ]
  }')

echo "$SESSION_STORE_RESPONSE"

# 3. 通过向量搜索API验证内容是否已存储
echo -e "\n测试3：向量搜索API验证"
sleep 1  # 给系统一点时间来索引数据
curl -s -X POST "$BASE_URL/vector/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Node.js Web框架比较",
    "limit": 5,
    "filters": {
      "sessionId": "'$SESSION_ID'"
    }
  }'

# 4. 通过会话管理API检索上下文，验证从向量数据库读取
echo -e "\n测试4：通过会话管理API检索上下文"
curl -s -X POST "$BASE_URL/mcp/context-keeper/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "Node.js框架对比"
  }'

# 5. 测试通过向量ID直接获取内容
echo -e "\n测试5：通过向量ID获取内容"
if [ ! -z "$VECTOR_ID" ]; then
  curl -s -X GET "$BASE_URL/vector/get?id=$VECTOR_ID"
else
  echo "跳过测试：未获取到向量ID"
fi

# 6. 测试会话管理和向量存储的联合验证 - 存储批量消息并汇总
echo -e "\n测试6：存储批量消息并汇总"
BATCH_ID="batch-$TIMESTAMP"
BATCH_RESPONSE=$(curl -s -X POST "$BASE_URL/mcp/context-keeper/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "summarizeAndStore": true,
    "batchId": "'$BATCH_ID'",
    "messages": [
      {
        "role": "user",
        "content": "Node.js中如何实现文件上传功能？",
        "contentType": "text",
        "priority": "P1"
      },
      {
        "role": "assistant",
        "content": "在Node.js中实现文件上传功能，可以使用Express框架配合multer中间件。以下是基本步骤：\n\n1. 安装必要的包：\n```bash\nnpm install express multer\n```\n\n2. 创建服务器并配置multer：\n```javascript\nconst express = require(\"express\");\nconst multer = require(\"multer\");\nconst path = require(\"path\");\nconst app = express();\n\n// 配置存储选项\nconst storage = multer.diskStorage({\n  destination: function(req, file, cb) {\n    cb(null, \"uploads/\");  // 上传文件存储路径\n  },\n  filename: function(req, file, cb) {\n    cb(null, file.fieldname + \"-\" + Date.now() + path.extname(file.originalname));\n  }\n});\n\nconst upload = multer({ storage: storage });\n\n// 处理单文件上传\napp.post(\"/upload\", upload.single(\"file\"), (req, res) => {\n  if (!req.file) {\n    return res.status(400).send(\"No file uploaded\");\n  }\n  res.send({\n    message: \"文件上传成功\",\n    file: req.file\n  });\n});\n\n// 处理多文件上传\napp.post(\"/upload-multiple\", upload.array(\"files\", 5), (req, res) => {\n  if (!req.files || req.files.length === 0) {\n    return res.status(400).send(\"No files uploaded\");\n  }\n  res.send({\n    message: \"多文件上传成功\",\n    files: req.files\n  });\n});\n\napp.listen(3000, () => {\n  console.log(\"服务器运行在 http://localhost:3000\");\n});\n```\n\n3. 创建HTML表单：\n```html\n<!DOCTYPE html>\n<html>\n<head>\n  <title>文件上传示例</title>\n</head>\n<body>\n  <h2>单文件上传</h2>\n  <form action=\"/upload\" method=\"POST\" enctype=\"multipart/form-data\">\n    <input type=\"file\" name=\"file\" />\n    <input type=\"submit\" value=\"上传\" />\n  </form>\n  \n  <h2>多文件上传</h2>\n  <form action=\"/upload-multiple\" method=\"POST\" enctype=\"multipart/form-data\">\n    <input type=\"file\" name=\"files\" multiple />\n    <input type=\"submit\" value=\"上传\" />\n  </form>\n</body>\n</html>\n```\n\n此外，确保创建uploads目录用于存储上传的文件。如果需要处理文件验证、大小限制等，可以在multer配置中添加相应的选项。",
        "contentType": "text",
        "priority": "P1"
      }
    ]
  }')

echo "$BATCH_RESPONSE"

# 7. 通过批次ID测试检索
echo -e "\n测试7：通过批次ID检索"
curl -s -X POST "$BASE_URL/vector/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "文件上传",
    "limit": 5,
    "filters": {
      "batchId": "'$BATCH_ID'"
    }
  }'

# 8. 测试会话状态 - 验证会话信息
echo -e "\n测试8：验证会话状态"
curl -s -X GET "$BASE_URL/mcp/context-keeper/sessionState?sessionId=$SESSION_ID"

echo -e "\n测试完成！" 