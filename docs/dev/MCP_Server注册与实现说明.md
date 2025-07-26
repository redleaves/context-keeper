# MCP Server 注册与实现指南

## 对话内容

### 用户提问：
> 你详细说明 一下 怎么才能注册成功，就是注册一个 MCP serve，你是基于什么约定协议，或者技术文件  代码 配置  json等，详细说明一下技术细节，怎样才能被你正确成功的识别，就拿sse类型的说明吧

### 助手回答：
## MCP Server 注册与实现详细说明

要成功注册一个MCP (Memory, Cognition, and Planning) server 并被我正确识别，需要遵循以下技术规范和实现细节，特别是基于SSE (Server-Sent Events)的实现：

### 1. 基本协议与架构

MCP server 需要实现一个符合 RESTful API 规范的服务端，支持以下核心功能：
- 实体管理（创建、读取、更新、删除）
- 关系管理（创建、读取、删除）
- 知识图谱查询（搜索、过滤）

### 2. API 端点规范

需要实现以下核心API端点：

```
POST /entities            # 创建实体
GET /entities/{id}        # 获取特定实体
PUT /entities/{id}        # 更新实体
DELETE /entities/{id}     # 删除实体

POST /relations           # 创建关系
GET /relations            # 获取关系
DELETE /relations/{id}    # 删除关系

GET /graph                # 获取整个知识图谱
GET /search               # 搜索节点
```

### 3. 数据模型与JSON格式

实体(Entity)格式要求：
```json
{
  "name": "实体名称",
  "entityType": "实体类型",
  "observations": ["观察1", "观察2", "..."]
}
```

关系(Relation)格式要求：
```json
{
  "from": "源实体名称",
  "to": "目标实体名称",
  "relationType": "关系类型"
}
```

### 4. SSE实现规范

对于SSE（Server-Sent Events）实现，需要：

1. 设置正确的HTTP头：
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

2. 事件格式符合SSE规范：
```
event: entity_created
data: {"type":"entity","name":"实体名","entityType":"类型","observations":[...]}

event: relation_created
data: {"type":"relation","from":"源实体","to":"目标实体","relationType":"关系类型"}
```

3. 支持事件流的断线重连机制，通过`Last-Event-ID`头部实现。

### 5. 注册配置

需要在对接Claude的配置文件中注册MCP server的URL和身份验证信息，格式通常为：

```json
{
  "mcp_servers": [
    {
      "name": "context-keeper",
      "url": "https://your-mcp-server.com/api",
      "auth": {
        "type": "bearer",
        "token": "your-auth-token"
      },
      "capabilities": ["entities", "relations", "search", "graph"]
    }
  ]
}
```

### 6. 技术实现细节

1. **服务端技术栈**：
   - 可以使用Node.js、Python、Go等语言实现
   - 需要支持长连接和事件流
   - 推荐使用支持异步I/O的框架

2. **数据持久化**：
   - 可以使用图数据库（如Neo4j）或文档数据库（如MongoDB）
   - 需支持高效的图查询和文本搜索

3. **安全与身份验证**：
   - 支持OAuth 2.0或API密钥认证
   - 实现适当的访问控制和权限管理

### 7. 具体SSE实现示例

下面是一个基于Node.js和Express的SSE实现示例：

```javascript
const express = require('express');
const app = express();
app.use(express.json());

// SSE端点实现
app.get('/sse-events', (req, res) => {
  // 设置SSE所需的HTTP头
  res.setHeader('Content-Type', 'text/event-stream');
  res.setHeader('Cache-Control', 'no-cache');
  res.setHeader('Connection', 'keep-alive');
  
  // 客户端ID，用于重连
  const clientId = Date.now();
  
  // 发送事件的函数
  const sendEvent = (event, data) => {
    res.write(`event: ${event}\n`);
    res.write(`data: ${JSON.stringify(data)}\n\n`);
  };
  
  // 发送初始连接确认
  sendEvent('connected', { clientId });
  
  // 设置事件监听器
  const entityListener = (data) => {
    sendEvent('entity_updated', data);
  };
  
  // 注册监听器
  eventEmitter.on('entity_change', entityListener);
  
  // 处理客户端断开连接
  req.on('close', () => {
    eventEmitter.off('entity_change', entityListener);
  });
});

// 创建实体API
app.post('/entities', async (req, res) => {
  try {
    const entity = req.body;
    // 数据验证
    if (!entity.name || !entity.entityType) {
      return res.status(400).json({ error: '实体必须包含name和entityType' });
    }
    
    // 存储实体
    const savedEntity = await db.saveEntity(entity);
    
    // 触发事件
    eventEmitter.emit('entity_change', savedEntity);
    
    res.status(201).json(savedEntity);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// 其他API端点实现...

app.listen(3000, () => {
  console.log('MCP Server running on port 3000');
});
```

### 8. 验证与测试

要验证MCP server是否被正确识别和连接：

1. 使用工具如Postman测试所有API端点
2. 实现简单的测试客户端，测试SSE连接和事件接收
3. 验证实体和关系的CRUD操作是否正常
4. 检查知识图谱查询和搜索功能

通过以上规范和实现细节，你可以创建一个可以被Claude正确识别和使用的MCP server。关键在于遵循标准的RESTful API设计、正确实现SSE事件流、以及符合规定的数据格式和端点定义。