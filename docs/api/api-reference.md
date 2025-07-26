# Context-Keeper API参考文档

本文档提供Context-Keeper服务所有可用的API端点的详细说明。

## 目录

- [公共端点](#公共端点)
- [Cursor专用API](#cursor专用api)
- [MCP标准API](#mcp标准api)
- [管理API](#管理api)
- [API认证](#api认证)
- [错误处理](#错误处理)
- [数据模型](#数据模型)

## 公共端点

### 健康检查

检查服务是否正常运行。

```
GET /health
```

#### 响应

```json
{
  "status": "healthy"
}
```

## Cursor专用API

### 关联代码文件

将代码文件关联到当前编程会话。

```
POST /api/cursor/associateFile
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| filePath | string | 是 | 文件路径 |
| language | string | 是 | 编程语言 |
| content | string | 是 | 文件内容 |

```json
{
  "sessionId": "cursor-session-123",
  "filePath": "/projects/app/src/main.js",
  "language": "javascript",
  "content": "console.log('Hello World');"
}
```

#### 响应

```json
{
  "status": "success",
  "message": "文件关联成功"
}
```

### 记录编辑操作

记录对代码文件的编辑操作。

```
POST /api/cursor/recordEdit
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| filePath | string | 是 | 文件路径 |
| type | string | 是 | 操作类型(insert/modify/delete) |
| position | integer | 是 | 操作位置 |
| content | string | 是 | 编辑内容 |

```json
{
  "sessionId": "cursor-session-123",
  "filePath": "/projects/app/src/main.js",
  "type": "insert",
  "position": 42,
  "content": "const greeting = 'Hello';"
}
```

#### 响应

```json
{
  "status": "success",
  "message": "编辑记录成功"
}
```

### 检索上下文

基于查询检索相关编程上下文。

```
POST /api/cursor/retrieveContext
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| query | string | 是 | 查询内容 |
| limit | integer | 否 | 返回结果数量限制，默认为2000 |
| skip_threshold | boolean | 否 | 是否跳过相似度阈值过滤，默认为false |
| similarity_threshold | float | 否 | 自定义相似度阈值，范围0-1，默认为0.35 |
| metadata | object | 否 | 元数据过滤条件 |

```json
{
  "sessionId": "cursor-session-123",
  "query": "如何使用Express处理POST请求",
  "skip_threshold": true
}
```

#### 响应

```json
{
  "session_state": "会话ID: cursor-session-123\n创建时间: 2023-05-01 10:20:30\n最后活动: 2023-05-01 10:30:45\n状态: active",
  "short_term_memory": "【最近对话】\n1. 示例代码片段...\n2. 编辑操作记录...",
  "long_term_memory": "【相关历史】\n1. [相似度:0.3142] 历史相关内容...",
  "relevant_knowledge": "【编程上下文】\n使用的编程语言:\n  javascript: 2个文件\n编辑操作统计:\n  总编辑数: 5\n  插入操作: 3\n  修改操作: 2"
}
```

### 获取编程特征和上下文摘要

获取当前会话的编程特征和上下文摘要。

```
POST /api/cursor/programmingContext
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| query | string | 否 | 可选查询参数 |

```json
{
  "sessionId": "cursor-session-123",
  "query": "express route handler"
}
```

#### 响应

```json
{
  "sessionId": "cursor-session-123",
  "associatedFiles": [
    {
      "path": "/projects/app/src/main.js",
      "language": "javascript",
      "lastEdit": 1682937645,
      "summary": "文件长度: 237字节"
    }
  ],
  "recentEdits": [
    {
      "timestamp": 1682937645,
      "filePath": "/projects/app/src/main.js",
      "type": "insert",
      "position": 42,
      "content": "const greeting = 'Hello';"
    }
  ],
  "extractedFeatures": [
    "使用的编程语言:",
    "  javascript: 1个文件",
    "编辑操作统计:",
    "  总编辑数: 1",
    "  插入操作: 1",
    "活跃文件:",
    "  main.js: 1次编辑"
  ]
}
```

## MCP标准API

### 存储上下文

将上下文信息存储到服务中。

```
POST /api/mcp/context-keeper/storeContext
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| content | string | 是 | 上下文内容 |
| type | string | 否 | 内容类型，默认为"conversation_summary" |
| priority | string | 否 | 优先级(P1/P2/P3)，默认为"P1" |
| metadata | object | 否 | 元数据 |

```json
{
  "sessionId": "mcp-session-123",
  "content": "这是需要记住的上下文信息",
  "type": "conversation_summary",
  "priority": "P1",
  "metadata": {
    "timestamp": 1682937645,
    "source": "user_message",
    "batchId": "batch-1234"
  }
}
```

#### 响应

```json
{
  "memoryId": "a3aebc0a-5c2d-43a0-a5e3-ceba8a99dc8e",
  "status": "success"
}
```

### 检索上下文

从服务中检索相关上下文。

```
POST /api/mcp/context-keeper/retrieveContext
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| query | string | 否 | 查询内容 |
| limit | integer | 否 | 返回结果数量限制，默认为2000 |
| skip_threshold | boolean | 否 | 是否跳过相似度阈值过滤，默认为false |
| similarity_threshold | float | 否 | 自定义相似度阈值，范围0-1，默认为0.35 |
| metadata | object | 否 | 元数据过滤条件 |
| memoryId | string | 否 | 特定记忆ID |

```json
{
  "sessionId": "mcp-session-123",
  "query": "相关内容查询",
  "metadata": {
    "batchId": "batch-1234"
  },
  "skip_threshold": true
}
```

#### 响应

与Cursor的retrieveContext响应格式相同。

### 汇总上下文

对上下文进行汇总。

```
POST /api/mcp/context-keeper/summarizeContext
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| maxTokens | integer | 否 | 汇总结果的最大token数，默认为1000 |

```json
{
  "sessionId": "mcp-session-123",
  "maxTokens": 500
}
```

#### 响应

```json
{
  "summary": "汇总内容...",
  "status": "success"
}
```

### 存储消息集合

将消息集合存储到服务中。

```
POST /api/mcp/context-keeper/storeMessages
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| messages | array | 是 | 消息数组 |
| batchId | string | 否 | 批次ID |

```json
{
  "sessionId": "mcp-session-123",
  "messages": [
    {
      "role": "user",
      "content": "如何在React中使用useEffect?",
      "contentType": "text",
      "priority": "P2"
    },
    {
      "role": "assistant",
      "content": "useEffect是React的一个Hook，用于在函数组件中执行副作用...",
      "contentType": "text",
      "priority": "P1"
    }
  ],
  "batchId": "batch-1234"
}
```

#### 响应

```json
{
  "messageIds": ["msg-123", "msg-124"],
  "status": "success"
}
```

### 检索对话

检索完整对话历史。

```
POST /api/mcp/context-keeper/retrieveConversation
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |
| limit | integer | 否 | 返回消息数量限制，默认为50 |
| before | string | 否 | 检索此消息ID之前的消息 |
| batchId | string | 否 | 特定批次ID |

```json
{
  "sessionId": "mcp-session-123",
  "limit": 20,
  "batchId": "batch-1234"
}
```

#### 响应

```json
{
  "messages": [
    {
      "id": "msg-123",
      "role": "user",
      "content": "如何在React中使用useEffect?",
      "timestamp": 1682937600
    },
    {
      "id": "msg-124",
      "role": "assistant",
      "content": "useEffect是React的一个Hook，用于在函数组件中执行副作用...",
      "timestamp": 1682937645
    }
  ],
  "hasMore": false
}
```

### 获取会话状态

获取特定会话的状态信息。

```
GET /api/mcp/context-keeper/sessionState?sessionId=mcp-session-123
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| sessionId | string | 是 | 会话ID |

#### 响应

```json
{
  "sessionId": "mcp-session-123",
  "created": 1682937600,
  "lastActive": 1682937645,
  "status": "active",
  "stats": {
    "contextCount": 5,
    "messageCount": 2,
    "fileCount": 1,
    "editCount": 3
  }
}
```

### 关联文件 (MCP标准)

将代码文件关联到当前会话。

```
POST /api/mcp/context-keeper/associateFile
```

与Cursor的associateFile接口参数和响应格式相同。

### 记录编辑 (MCP标准)

记录对代码文件的编辑操作。

```
POST /api/mcp/context-keeper/recordEdit
```

与Cursor的recordEdit接口参数和响应格式相同。

## 管理API

### 列出集合

列出所有向量集合。

```
GET /api/collections
```

#### 响应

```json
{
  "collections": ["context_keeper", "test_collection"],
  "default": "context_keeper"
}
```

### 创建集合

创建新的向量集合。

```
POST /api/collections
```

#### 请求参数

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|------|------|
| name | string | 是 | 集合名称 |
| dimension | integer | 否 | 向量维度，默认为1536 |
| metric | string | 否 | 相似度度量方式，默认为"cosine" |

```json
{
  "name": "new_collection",
  "dimension": 1536,
  "metric": "cosine"
}
```

#### 响应

```json
{
  "name": "new_collection",
  "status": "created"
}
```

### 获取集合详情

获取特定集合的详细信息。

```
GET /api/collections/:name
```

#### 响应

```json
{
  "name": "context_keeper",
  "dimension": 1536,
  "metric": "cosine",
  "count": 1250,
  "created": 1682937600
}
```

### 删除集合

删除特定集合。

```
DELETE /api/collections/:name
```

#### 响应

```json
{
  "name": "context_keeper",
  "status": "deleted"
}
```

## API认证

默认情况下，API没有启用认证。如启用认证（见配置文件），需在请求头中添加API密钥：

```
Authorization: Bearer your-api-key
```

## 错误处理

所有API错误将返回适当的HTTP状态码和JSON格式的错误详情：

```json
{
  "error": "错误消息",
  "details": "详细错误信息",
  "code": "ERROR_CODE"
}
```

常见错误代码：

| 错误代码 | HTTP状态码 | 描述 |
|---------|-----------|------|
| SESSION_NOT_FOUND | 404 | 会话ID不存在 |
| INVALID_REQUEST | 400 | 请求参数无效 |
| UNAUTHORIZED | 401 | 认证失败 |
| INTERNAL_ERROR | 500 | 服务器内部错误 |
| VECTOR_DB_ERROR | 503 | 向量数据库服务错误 |
| EMBEDDING_API_ERROR | 503 | 嵌入API服务错误 |

## 数据模型

### 会话 (Session)

```json
{
  "id": "string",          // 会话ID
  "created": "timestamp",  // 创建时间
  "lastActive": "timestamp", // 最后活跃时间
  "status": "string"       // 状态(active/inactive)
}
```

### 文件 (File)

```json
{
  "sessionId": "string",   // 关联的会话ID
  "path": "string",        // 文件路径
  "language": "string",    // 编程语言
  "lastEdit": "timestamp", // 最后编辑时间
  "summary": "string"      // 文件摘要
}
```

### 编辑记录 (Edit)

```json
{
  "sessionId": "string",   // 关联的会话ID
  "filePath": "string",    // 文件路径
  "type": "string",        // 操作类型(insert/modify/delete)
  "position": "integer",   // 操作位置
  "content": "string",     // 编辑内容
  "timestamp": "timestamp" // 编辑时间
}
```

### 上下文 (Context)

```json
{
  "id": "string",          // 上下文ID
  "sessionId": "string",   // 关联的会话ID
  "content": "string",     // 上下文内容
  "type": "string",        // 内容类型
  "priority": "string",    // 优先级(P1/P2/P3)
  "metadata": "object",    // 元数据
  "timestamp": "timestamp" // 创建时间
}
```

### 消息 (Message)

```json
{
  "id": "string",          // 消息ID
  "sessionId": "string",   // 关联的会话ID
  "role": "string",        // 角色(user/assistant/system)
  "content": "string",     // 消息内容
  "contentType": "string", // 内容类型(text/code/image)
  "priority": "string",    // 优先级(P1/P2/P3)
  "timestamp": "timestamp" // 创建时间
}
```

---

有关API使用的更多示例，请参考[用法示例](examples.md)文档。 