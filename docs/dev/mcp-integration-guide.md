# Context-Keeper MCP集成指南

Context-Keeper是一个符合MCP（Memory Context Protocol）标准的编程上下文管理服务，可以无缝集成到支持MCP协议的开发工具中，如Cursor编辑器。本指南提供Context-Keeper的功能清单和集成步骤。

## 目录

- [功能清单](#功能清单)
- [安装前提条件](#安装前提条件)
- [在Cursor中配置Context-Keeper](#在cursor中配置context-keeper)
- [验证安装](#验证安装)
- [集成测试](#集成测试)
- [故障排除](#故障排除)
- [高级配置](#高级配置)

## 功能清单

Context-Keeper作为MCP服务器提供以下核心功能：

### 1. 会话管理

- **会话创建与识别**：自动创建和管理编程会话，每个会话有唯一标识符。
- **会话状态跟踪**：记录会话创建时间、最后活动时间和状态。
- **会话统计**：提供会话内上下文数量、消息数量、文件数量和编辑操作数量等统计信息。

### 2. 文件关联

- **代码文件关联**：将源代码文件与会话关联，支持多种编程语言自动识别。
- **文件内容索引**：索引文件内容，便于后续搜索和上下文提取。
- **文件特征提取**：自动分析文件结构，提取关键编程特征（如函数定义、类声明、导入语句等）。

### 3. 编辑跟踪

- **实时编辑记录**：记录对代码的插入、修改和删除操作。
- **编辑位置感知**：精确记录编辑操作在文件中的位置。
- **编辑历史保存**：保存编辑历史，便于理解代码演化过程。

### 4. 上下文检索

- **语义相似度搜索**：基于查询内容，返回语义相关的上下文。
- **相似度阈值控制**：可调整相似度阈值，平衡搜索精度和召回率。
- **元数据过滤**：支持基于元数据（如批次ID、时间戳等）过滤上下文。

### 5. 向量存储

- **高效向量索引**：使用高性能向量数据库存储和检索嵌入向量。
- **集合管理**：支持多个向量集合的创建和管理。
- **灵活的度量方式**：支持多种向量相似度计算方法（余弦相似度、欧氏距离等）。

### 6. 批次管理

- **批次ID标记**：使用批次ID组织相关上下文和消息。
- **批次过滤**：支持基于批次ID检索特定对话或上下文。
- **批次状态追踪**：记录和管理批次的创建和更新状态。

### 7. 对话管理

- **消息存储**：存储用户和助手间的对话消息。
- **多种内容类型**：支持文本、代码和图像等不同类型的消息内容。
- **优先级管理**：支持消息优先级设置，影响上下文检索排序。

### 8. 上下文格式化

- **MCP兼容响应**：以MCP协议格式返回上下文，包括会话状态、短期记忆、长期记忆和相关知识。
- **编程特征汇总**：汇总编程相关特征，提供代码理解辅助。
- **自适应输出限制**：控制输出的token数量，避免超出LLM上下文窗口。

### 9. API扩展

- **标准MCP端点**：提供完整的MCP标准API端点。
- **Cursor专用API**：提供针对Cursor编辑器优化的专用API。
- **管理API**：提供服务运维和管理功能的API。

## 安装前提条件

在将Context-Keeper与Cursor集成前，请确保满足以下条件：

1. **安装Context-Keeper服务**：
   - 已完成Context-Keeper服务的安装与配置
   - 服务正常运行（可通过访问 `http://<host>:<port>/health` 检查）
   - 已配置并测试向量数据库连接
   - 已配置文本嵌入服务API密钥

2. **Cursor编辑器**：
   - 安装最新版本的Cursor编辑器
   - 确保Cursor支持MCP集成（1.8.0或更高版本）

3. **网络连接**：
   - 确保Cursor编辑器可以访问Context-Keeper服务的API端点
   - 如有防火墙，请开放相应端口

## 在Cursor中配置Context-Keeper

### 步骤1：启动Context-Keeper服务

首先确保Context-Keeper服务正在运行：

```bash
cd /path/to/context-keeper
./start.sh
```

验证服务健康状态：

```bash
curl http://localhost:8081/health
```

应返回：

```json
{
  "status": "healthy"
}
```

### 步骤2：配置Cursor的MCP集成

1. 打开Cursor编辑器
2. 选择菜单: 设置 > 集成 > MCP集成
3. 点击"添加MCP服务"
4. 输入以下信息:
   - 名称: Context-Keeper
   - 服务URL: http://localhost:8081 (或您部署的URL)
   - 服务类型: 上下文管理
   - API密钥: 如果您启用了认证，输入API密钥；否则留空
5. 点击"测试连接"确保连接成功
6. 点击"保存"完成配置

### 步骤3：启用Context-Keeper服务

1. 在MCP服务列表中找到Context-Keeper
2. 将其状态切换为"启用"
3. 如需设为默认上下文提供者，点击"设为默认"

## 验证安装

### 基本功能验证

1. **创建新项目**：在Cursor中创建或打开一个项目
2. **检查会话关联**：打开开发者控制台 (菜单: 视图 > 开发者 > 切换开发者工具)，查看网络请求中是否有对Context-Keeper API的调用
3. **验证文件关联**：打开一个代码文件，检查日志是否显示文件已成功关联
4. **测试编辑跟踪**：修改代码文件，检查是否有对`/api/cursor/recordEdit`的API调用
5. **验证上下文检索**：在Cursor的AI面板中提问与代码相关的问题，查看是否返回相关上下文

### 集成测试脚本

对于更全面的测试，可以使用以下集成测试脚本：

```bash
./test_cursor_integration.sh
```

该脚本将执行一系列测试，验证Context-Keeper与Cursor的集成。

## 集成测试

Context-Keeper提供了一套完整的集成测试脚本，用于验证与Cursor的集成：

### 测试文件关联

```bash
curl -X POST "http://localhost:8081/api/cursor/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "test-session-123",
    "filePath": "/projects/test/server.js",
    "language": "javascript",
    "content": "const express = require('\''express'\'');\nconst app = express();\n\napp.get('\''/'\\'', (req, res) => {\n  res.send('\''Hello World!'\\'');\n});\n\napp.listen(3000, () => {\n  console.log('\''Server running on port 3000'\\'');\n});"
  }'
```

### 测试编辑记录

```bash
curl -X POST "http://localhost:8081/api/cursor/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "test-session-123",
    "filePath": "/projects/test/server.js",
    "type": "insert",
    "position": 121,
    "content": "\n\napp.post('\''/api/data'\'', (req, res) => {\n  // 处理数据\n  res.json({ success: true });\n});\n"
  }'
```

### 测试上下文检索

```bash
curl -X POST "http://localhost:8081/api/cursor/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "test-session-123",
    "query": "如何在Express中实现POST请求处理"
  }'
```

## 故障排除

### 常见问题

1. **连接失败**
   - 检查Context-Keeper服务是否在运行
   - 验证服务URL是否正确
   - 检查防火墙设置是否允许连接

2. **文件关联失败**
   - 检查请求参数格式是否正确
   - 确保sessionId格式符合要求
   - 查看服务日志获取详细错误信息

3. **上下文检索无结果**
   - 验证相似度阈值设置是否过高
   - 确保已有足够文件关联和编辑记录
   - 尝试使用skip_threshold参数绕过相似度过滤

4. **参数格式问题**
   - 确保使用正确的参数名（注意大小写，如sessionId而非sessionID）
   - 检查JSON格式是否有效
   - 参考API文档核对参数要求

### 日志检查

查看Context-Keeper服务日志：

```bash
cat logs/service.log
```

查看Cursor开发者工具中的网络请求和控制台输出。

## 高级配置

### 自定义相似度阈值

```json
{
  "sessionId": "session-123",
  "query": "查询内容",
  "similarity_threshold": 0.4
}
```

### 配置批次ID过滤

```json
{
  "sessionId": "session-123",
  "query": "查询内容",
  "metadata": {
    "batchId": "batch-456"
  }
}
```

### 跳过相似度过滤

```json
{
  "sessionId": "session-123",
  "query": "查询内容",
  "skip_threshold": true
}
```

### 限制返回内容大小

```json
{
  "sessionId": "session-123",
  "query": "查询内容",
  "limit": 1000
}
```

---

有关Context-Keeper更多详细信息，请参考[API参考文档](api-reference.md)和[部署指南](deployment.md)。 