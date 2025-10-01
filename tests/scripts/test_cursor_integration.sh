#!/bin/bash

# Cursor适配层集成测试脚本
# 测试针对Cursor编辑器的适配功能

# 设置基础URL
BASE_URL="http://localhost:8088"
SESSION_ID="cursor-test-$(date +%s)"
TIMESTAMP=$(date +%s)

echo "使用会话ID: $SESSION_ID"

# 1. 测试关联代码文件
echo -e "\n测试1：关联代码文件"
ASSOCIATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/server.js",
    "language": "javascript",
    "content": "const express = require(\"express\");\nconst app = express();\nconst port = 3000;\n\napp.get(\"/\", (req, res) => {\n  res.send(\"Hello World!\");\n});\n\napp.listen(port, () => {\n  console.log(`Example app listening at http://localhost:${port}`);\n});"
  }')

echo "$ASSOCIATE_RESPONSE"

# 2. 测试记录编辑操作
echo -e "\n测试2：记录编辑操作"
EDIT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/server.js",
    "type": "insert",
    "position": 72,
    "content": "\n\napp.post(\"/api/data\", (req, res) => {\n  res.json({ message: \"Data received\" });\n});\n"
  }')

echo "$EDIT_RESPONSE"

# 3. 测试第二次编辑操作
echo -e "\n测试3：记录第二次编辑操作"
EDIT_RESPONSE2=$(curl -s -X POST "$BASE_URL/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/server.js",
    "type": "modify",
    "position": 120,
    "content": "添加了body-parser中间件用于解析JSON请求体"
  }')

echo "$EDIT_RESPONSE2"

# 4. 测试获取编程上下文
echo -e "\n测试4：获取编程上下文"
sleep 1 # 等待一下以确保前面的操作都已处理完成
CONTEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/cursor/programmingContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "express route handler"
  }')

echo "$CONTEXT_RESPONSE"

# 5. 测试通过Cursor适配层检索上下文
echo -e "\n测试5：通过Cursor适配层检索上下文"
curl -s -X POST "$BASE_URL/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "如何在Express中添加POST路由？"
  }'

# 6. 关联第二个文件
echo -e "\n测试6：关联第二个文件"
curl -s -X POST "$BASE_URL/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/database.js",
    "language": "javascript",
    "content": "const mongoose = require(\"mongoose\");\n\nmongoose.connect(\"mongodb://localhost:27017/testdb\", {\n  useNewUrlParser: true,\n  useUnifiedTopology: true\n});\n\nconst db = mongoose.connection;\ndb.on(\"error\", console.error.bind(console, \"connection error:\"));\ndb.once(\"open\", function() {\n  console.log(\"Database connected\");\n});\n\nmodule.exports = mongoose;"
  }'

# 7. 测试Go语言特征提取功能
echo -e "\n测试7：测试Go语言特征提取功能"
curl -s -X POST "$BASE_URL/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "filePath": "/projects/test/main.go",
    "language": "go",
    "content": "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\thttp.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"Hello, Go Web API!\")\n\t})\n\n\tfmt.Println(\"Starting server at port 8080\")\n\thttp.ListenAndServe(\":8080\", nil)\n}"
  }'

# 8. 测试错误处理 - 缺少会话ID
echo -e "\n测试8：测试错误处理 - 缺少会话ID"
curl -s -X POST "$BASE_URL/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "",
    "query": "测试错误"
  }'

# 9. 最终测试 - 获取完整会话状态
echo -e "\n测试9：获取完整会话状态 - 验证所有内容都已存储并可检索"
curl -s -X POST "$BASE_URL/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'$SESSION_ID'",
    "query": "Express和MongoDB"
  }'

echo -e "\n测试完成！" 