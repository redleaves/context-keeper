# Context-Keeper 使用示例

本文档提供Context-Keeper在不同场景下的使用示例，帮助开发者更好地理解和应用Context-Keeper服务。

## 目录

- [创建和管理会话](#创建和管理会话)
- [代码文件管理](#代码文件管理)
- [编辑操作记录](#编辑操作记录)
- [上下文检索和应用](#上下文检索和应用)
- [批次管理示例](#批次管理示例)
- [完整工作流示例](#完整工作流示例)
- [与Cursor集成示例](#与cursor集成示例)
- [性能优化示例](#性能优化示例)

## 创建和管理会话

### 创建新会话

当用户开始新的编程活动时，Context-Keeper会自动创建新会话。会话ID可以由客户端生成，也可以使用服务生成的ID。

```bash
# 使用客户端生成的会话ID
SESSION_ID="session-$(date +%s)"
echo "新会话ID: $SESSION_ID"

# 创建第一个关联来初始化会话
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/index.js",
    "language": "javascript",
    "content": "console.log('\''Hello World'\'');"
  }'
```

### 获取会话状态

检查会话的当前状态，包括创建时间、最后活动时间和相关统计信息。

```bash
curl -X GET "http://localhost:8081/api/mcp/context-keeper/sessionState?sessionId=session-1629384756"
```

响应示例：

```json
{
  "sessionId": "session-1629384756",
  "created": 1629384756,
  "lastActive": 1629385000,
  "status": "active",
  "stats": {
    "contextCount": 12,
    "messageCount": 8,
    "fileCount": 3,
    "editCount": 15
  }
}
```

## 代码文件管理

### 关联多个文件

在一个项目中，通常需要关联多个相关文件到同一会话。

```bash
# 关联主应用文件
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/app.js",
    "language": "javascript",
    "content": "const express = require('\''express'\'');\nconst app = express();\n\napp.get('\''/'\\'', (req, res) => {\n  res.send('\''Hello World!'\\'');\n});\n\nmodule.exports = app;"
  }'

# 关联路由文件
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/routes/index.js",
    "language": "javascript",
    "content": "const router = require('\''express'\'').Router();\n\nrouter.get('\''/users'\'', (req, res) => {\n  res.json([{ id: 1, name: '\''Alice'\'' }, { id: 2, name: '\''Bob'\'' }]);\n});\n\nmodule.exports = router;"
  }'

# 关联数据库配置文件
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/db/config.js",
    "language": "javascript",
    "content": "const mongoose = require('\''mongoose'\'');\n\nmongoose.connect('\''mongodb://localhost:27017/myapp'\'', {\n  useNewUrlParser: true,\n  useUnifiedTopology: true\n});\n\nmodule.exports = mongoose;"
  }'
```

### 处理不同编程语言的文件

Context-Keeper支持多种编程语言，能够自动提取特定语言的编程特征。

```bash
# 关联Python文件
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/scripts/process_data.py",
    "language": "python",
    "content": "import pandas as pd\n\ndef process_data(file_path):\n    df = pd.read_csv(file_path)\n    df['\''processed'\''] = df['\''raw_value'\''] * 2\n    return df\n\nif __name__ == '\''__main__'\'':\n    result = process_data('\''data.csv'\'')\n    print(f'\''Processed {len(result)} rows'\'')"
  }'

# 关联Go文件
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/cmd/worker/main.go",
    "language": "go",
    "content": "package main\n\nimport (\n    \"fmt\"\n    \"log\"\n    \"time\"\n)\n\nfunc main() {\n    log.Println(\"Worker started\")\n    for {\n        fmt.Println(\"Processing...\")\n        time.Sleep(5 * time.Second)\n    }\n}"
  }'
```

## 编辑操作记录

### 记录插入操作

当用户在文件中添加新代码时，记录插入操作。

```bash
curl -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/app.js",
    "type": "insert",
    "position": 105,
    "content": "\n\napp.post('\''/api/login'\'', (req, res) => {\n  const { username, password } = req.body;\n  // TODO: 实现用户验证逻辑\n  res.json({ success: true, token: '\''dummy-token'\'' });\n});\n"
  }'
```

### 记录修改操作

当用户修改现有代码时，记录修改操作。

```bash
curl -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/app.js",
    "type": "modify",
    "position": 42,
    "content": "  res.json({ message: '\''Welcome to MyApp API'\'' });"
  }'
```

### 记录删除操作

当用户删除代码时，记录删除操作。

```bash
curl -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "filePath": "/projects/myapp/src/routes/index.js",
    "type": "delete",
    "position": 150,
    "content": "// 此路由已不再需要\nrouter.get('\''/deprecated'\'', (req, res) => {\n  res.send('\''This route is deprecated'\'');\n});\n"
  }'
```

## 上下文检索和应用

### 基本上下文检索

根据查询检索相关上下文。

```bash
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "如何在Express应用中处理用户登录"
  }'
```

### 自定义相似度阈值的检索

调整相似度阈值以获取更多或更少的结果。

```bash
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "Mongoose数据库连接配置",
    "similarity_threshold": 0.3
  }'
```

### 跳过相似度阈值

在需要尽可能多的上下文时，可以跳过相似度阈值过滤。

```bash
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "所有API路由",
    "skip_threshold": true
  }'
```

### 获取编程特征和摘要

获取当前会话的编程特征和上下文摘要。

```bash
curl -X POST "http://localhost:8081/api/cursor/programmingContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756"
  }'
```

## 批次管理示例

### 创建批次并存储消息

在对话过程中，可以使用批次ID组织相关消息。

```bash
BATCH_ID="batch-$(date +%s)"
echo "新批次ID: $BATCH_ID"

curl -X POST "http://localhost:8081/api/mcp/context-keeper/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "messages": [
      {
        "role": "user",
        "content": "我需要在Express应用中添加一个新的API端点，用于处理文件上传。",
        "contentType": "text",
        "priority": "P1"
      },
      {
        "role": "assistant",
        "content": "在Express中处理文件上传，你可以使用multer中间件。首先，安装multer：\n```bash\nnpm install multer\n```\n\n然后在你的Express应用中配置：\n```javascript\nconst multer = require('\''multer'\'');\nconst upload = multer({ dest: '\''uploads/'\'' });\n\napp.post('\''/api/upload'\'', upload.single('\''file'\''), (req, res) => {\n  res.json({\n    success: true,\n    file: req.file\n  });\n});\n```",
        "contentType": "text",
        "priority": "P1"
      }
    ],
    "batchId": "'"$BATCH_ID"'"
  }'
```

### 基于批次ID检索消息

使用批次ID检索特定对话。

```bash
curl -X POST "http://localhost:8081/api/mcp/context-keeper/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "batchId": "'"$BATCH_ID"'"
  }'
```

### 基于批次ID存储和检索上下文

在特定批次内存储和检索上下文。

```bash
# 存储批次相关上下文
curl -X POST "http://localhost:8081/api/mcp/context-keeper/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "content": "文件上传功能已添加到API中，使用multer中间件处理多部分表单数据。",
    "type": "implementation_note",
    "priority": "P1",
    "metadata": {
      "batchId": "'"$BATCH_ID"'"
    }
  }'

# 检索批次相关上下文
curl -X POST "http://localhost:8081/api/mcp/context-keeper/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "文件上传",
    "metadata": {
      "batchId": "'"$BATCH_ID"'"
    }
  }'
```

## 完整工作流示例

以下是一个完整的工作流示例，展示Context-Keeper在实际开发过程中的使用。

```bash
#!/bin/bash

# 创建新会话
SESSION_ID="session-$(date +%s)"
echo "新会话ID: $SESSION_ID"

# 关联主应用文件
echo "关联主应用文件..."
curl -s -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/server.js",
    "language": "javascript",
    "content": "const express = require('\''express'\'');\nconst app = express();\n\napp.use(express.json());\n\napp.get('\''/'\\'', (req, res) => {\n  res.send('\''API Server'\\'');\n});\n\nconst PORT = process.env.PORT || 3000;\napp.listen(PORT, () => {\n  console.log(`Server running on port ${PORT}`);\n});"
  }'

# 记录编辑操作 - 添加用户路由
echo "记录编辑操作 - 添加用户路由..."
curl -s -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/server.js",
    "type": "insert",
    "position": 90,
    "content": "\n\n// 用户路由\napp.get('\''/api/users'\'', (req, res) => {\n  res.json([{ id: 1, name: '\''Alice'\'' }, { id: 2, name: '\''Bob'\'' }]);\n});\n"
  }'

# 关联用户模型文件
echo "关联用户模型文件..."
curl -s -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/models/User.js",
    "language": "javascript",
    "content": "const mongoose = require('\''mongoose'\'');\n\nconst userSchema = new mongoose.Schema({\n  name: { type: String, required: true },\n  email: { type: String, required: true, unique: true },\n  password: { type: String, required: true },\n  createdAt: { type: Date, default: Date.now }\n});\n\nmodule.exports = mongoose.model('\''User'\'', userSchema);"
  }'

# 记录编辑操作 - 添加用户创建路由
echo "记录编辑操作 - 添加用户创建路由..."
curl -s -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/server.js",
    "type": "insert",
    "position": 210,
    "content": "\n\n// 创建用户\napp.post('\''/api/users'\'', (req, res) => {\n  const { name, email, password } = req.body;\n  // TODO: 实现用户创建逻辑\n  res.status(201).json({ id: 3, name, email });\n});\n"
  }'

# 使用上下文检索 - 查询如何实现用户验证
echo "使用上下文检索 - 查询如何实现用户验证..."
curl -s -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "query": "如何实现Express中的用户验证"
  }' | jq

# 记录编辑操作 - 添加用户验证中间件
echo "记录编辑操作 - 添加用户验证中间件..."
curl -s -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/middleware/auth.js",
    "language": "javascript",
    "type": "insert",
    "position": 0,
    "content": "const jwt = require('\''jsonwebtoken'\'');\n\nconst auth = (req, res, next) => {\n  try {\n    const token = req.header('\''Authorization'\'').replace('\''Bearer '\'', '\'''\'');\n    const decoded = jwt.verify(token, process.env.JWT_SECRET);\n    req.userId = decoded.userId;\n    next();\n  } catch (error) {\n    res.status(401).json({ error: '\''Authentication failed'\'' });\n  }\n};\n\nmodule.exports = auth;"
  }'

# 关联认证路由文件
echo "关联认证路由文件..."
curl -s -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/routes/auth.js",
    "language": "javascript",
    "content": "const express = require('\''express'\'');\nconst router = express.Router();\nconst jwt = require('\''jsonwebtoken'\'');\n\nrouter.post('\''/login'\'', (req, res) => {\n  const { email, password } = req.body;\n  // TODO: 验证用户凭据\n  const token = jwt.sign({ userId: 1 }, process.env.JWT_SECRET, { expiresIn: '\''1h'\'' });\n  res.json({ token });\n});\n\nmodule.exports = router;"
  }'

# 记录编辑操作 - 将认证路由集成到主应用
echo "记录编辑操作 - 将认证路由集成到主应用..."
curl -s -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/myapp/src/server.js",
    "type": "insert",
    "position": 60,
    "content": "\n// 导入路由\nconst authRoutes = require('\''/routes/auth'\'');\n\n// 使用路由\napp.use('\''/api/auth'\'', authRoutes);\n"
  }'

# 获取编程特征和上下文摘要
echo "获取编程特征和上下文摘要..."
curl -s -X POST "http://localhost:8081/api/cursor/programmingContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'"
  }' | jq

echo "完整工作流示例执行完毕"
```

## 与Cursor集成示例

以下是Context-Keeper与Cursor编辑器集成的典型场景。

### Cursor自动发送的API请求

当在Cursor中编辑代码时，编辑器会自动向Context-Keeper发送以下请求：

1. 当打开新文件时，Cursor发送associateFile请求：

```json
{
  "sessionId": "cursor-session-1234567890",
  "filePath": "/path/to/file.js",
  "language": "javascript",
  "content": "// 文件内容..."
}
```

2. 当编辑文件时，Cursor发送recordEdit请求：

```json
{
  "sessionId": "cursor-session-1234567890",
  "filePath": "/path/to/file.js",
  "type": "insert",
  "position": 120,
  "content": "// 新添加的代码..."
}
```

3. 当在AI面板中提问时，Cursor发送retrieveContext请求：

```json
{
  "sessionId": "cursor-session-1234567890",
  "query": "用户在AI面板中输入的问题"
}
```

### Cursor响应处理

Context-Keeper返回的数据在Cursor中的使用方式：

1. 上下文检索结果嵌入到AI提示中：

```
【会话状态】
会话ID: cursor-session-1234567890
创建时间: 2023-05-01 10:20:30
最后活动: 2023-05-01 10:30:45

【编程上下文】
使用的编程语言:
  javascript: 4个文件
编辑操作统计:
  总编辑数: 7
  插入操作: 4
  修改操作: 3
活跃文件:
  server.js: 3次编辑
  auth.js: 1次编辑
  
用户问题: 如何实现用户登录API?
```

2. 文件关联和编辑操作响应用于日志和调试：

```
[Cursor] 文件已关联: /path/to/file.js
[Cursor] 编辑操作已记录: insert at position 120
```

## 性能优化示例

### 批量关联文件

在打开大型项目时，可以批量关联多个文件以提高效率。

```bash
#!/bin/bash

SESSION_ID="session-$(date +%s)"
echo "新会话ID: $SESSION_ID"

# 获取项目中的所有JS文件
find /projects/myapp -name "*.js" | while read file; do
  content=$(cat "$file")
  language="javascript"
  echo "关联文件: $file"
  
  curl -s -X POST "http://localhost:8081/api/cursor/associateFile" \
    -H "Content-Type: application/json" \
    -d '{
      "sessionId": "'"$SESSION_ID"'",
      "filePath": "'"$file"'",
      "language": "'"$language"'",
      "content": "'"$(echo "$content" | sed 's/"/\\"/g' | sed 's/\\/\\\\/g' | sed 's/\n/\\n/g')"'"
    }' > /dev/null
done

echo "成功关联所有JS文件到会话: $SESSION_ID"
```

### 使用特定相似度阈值优化检索

针对不同需求调整相似度阈值，提高检索效率。

```bash
# 高精度检索 - 仅返回高度相关的结果
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "JWT身份验证中间件",
    "similarity_threshold": 0.7
  }'

# 高召回率检索 - 返回更多可能相关的结果
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "数据库模型定义",
    "similarity_threshold": 0.2
  }'
```

### 限制返回内容大小

根据客户端需求限制返回内容大小，避免传输过大的数据。

```bash
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "session-1629384756",
    "query": "API路由定义",
    "limit": 500
  }'
```

---

以上示例展示了Context-Keeper在不同场景下的使用方法。这些示例可以根据实际需求进行调整和扩展。有关API的完整参考，请查看[API参考文档](api-reference.md)。 