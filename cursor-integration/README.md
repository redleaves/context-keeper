# Context-Keeper Cursor/VSCode 扩展插件

## 🧠 智能编程上下文管理扩展

Context-Keeper是一个专为Cursor/VSCode设计的智能编程上下文管理扩展，提供记忆存储、代码关联、WebSocket实时通信和本地指令执行能力。

---

## 📁 核心文件结构和职能分析

### 🎯 **扩展核心文件（必需）**

#### 1. `package.json` - 扩展配置清单 ⭐
**职能**: VSCode/Cursor扩展的配置清单文件
- 定义扩展入口：`"main": "./cursor-vscode-extension.js"`
- 声明激活事件：`"activationEvents": ["onStartupFinished"]`
- 注册命令、菜单、配置项
- 定义扩展依赖（ws库用于WebSocket）

**关键配置**：
```json
{
  "main": "./cursor-vscode-extension.js",
  "activationEvents": ["onStartupFinished"],
  "contributes": {
    "commands": [...],
    "configuration": {...}
  }
}
```

#### 2. `cursor-vscode-extension.js` - 扩展主入口 ⭐⭐⭐
**职能**: Cursor/VSCode扩展的**唯一正式入口文件**
- **VSCode API集成**: 状态栏、命令面板、配置界面
- **MCP客户端集成**: 调用Context-Keeper的MCP服务
- **WebSocket功能**: 自动用户ID检查、连接管理、消息处理
- **本地指令执行**: 文件写入、短期记忆存储
- **配置管理**: 完整的配置界面和持久化

**核心功能**：
```javascript
class ContextKeeperExtension {
  // VSCode扩展生命周期
  constructor(context) // 扩展初始化
  async init()         // 完整功能启动
  
  // WebSocket集成
  startUserIdCheck()           // 自动用户ID检查
  connectWebSocket(userId)     // 建立WebSocket连接
  handleWebSocketMessage()     // 处理服务端消息
  
  // 本地指令执行
  executeLocalInstructionDirect()  // 执行本地文件操作
  handleShortMemoryDirect()        // 处理短期记忆存储
  
  // 配置界面
  showStatusPanel()    // 状态面板
  openSettingsPanel()  // 设置界面
}
```

#### 3. `mcp-client.js` - MCP客户端库 ⭐⭐
**职能**: 与Context-Keeper服务端通信的MCP客户端实现
- **HTTP请求封装**: 封装所有MCP工具调用
- **会话管理**: 创建、获取会话信息
- **记忆操作**: 存储对话、检索上下文、长期记忆
- **文件关联**: 代码文件关联和编辑记录
- **健康检查**: 服务连接状态检查

**主要方法**：
```javascript
class ContextKeeperMCPClient {
  async sessionManagement()    // 会话管理
  async storeConversation()    // 存储对话
  async memorizeContext()      // 长期记忆
  async retrieveContext()      // 检索上下文
  async associateFile()        // 文件关联
  async recordEdit()           // 记录编辑
  async healthCheck()          // 健康检查
}
```

#### 4. `cursor-config-ui.html` - 完整配置界面 ⭐⭐
**职能**: 功能完整的Web配置界面
- **动态配置架构**: 支持从服务器获取配置schema
- **多种字段类型**: 文本、数字、布尔、选择器等
- **表单验证**: 必填字段、数值范围验证
- **实时连接测试**: 自动测试服务器连接状态
- **响应式设计**: 支持移动端适配
- **高级功能**: 数据清除、配置导出等

#### 5. `icon.png` - 扩展图标 ⭐
**职能**: VSCode扩展的显示图标

---

### 🧪 **测试和开发文件（可选保留）**

#### 6. `test.js` - 集成测试
**职能**: 完整的MCP功能集成测试
- 测试所有MCP工具调用
- 验证WebSocket连接
- 会话和记忆功能测试

#### 7. `verify-installation.js` - 安装验证
**职能**: 验证扩展安装是否正确

---

### 📋 **文档和配置文件（保留）**

#### 7. `USER-GUIDE.md` - 用户指南
**职能**: 详细的用户使用指南

#### 8. `EXTENSION-INSTALL.md` - 安装说明
**职能**: 扩展安装步骤说明

#### 9. `install-guide.md` - 安装指南
**职能**: 快速安装指南

---

### ❌ **废弃/冗余文件（建议删除）**

#### 历史遗留文件：
- `extension.js` - 独立WebSocket客户端启动器（已集成到主入口）
- `start-extension.js` - 简化启动器（已废弃）
- `cursor-extension.js` - 独立WebSocket逻辑文件（已集成到主入口）

#### 测试文件：
- `test-hooks.js` - 测试钩子（非核心功能）
- `simple-path-test.js` - 路径测试（开发用）
- `path-unification-test.js` - 路径统一测试（开发用）

#### 配置文件：
- `cursor_mcp_config.json` - 旧MCP配置（已被扩展配置替代）

#### 其他：
- `websocket-client.log` - 日志文件（临时文件）
- `README-VSCode-Extension.md` - 旧版README（已合并）
- `vscode-extension/` - TypeScript版本目录（未使用）
- `.DS_Store` - 系统文件

---

## 🚀 扩展加载机制

### Cursor/VSCode 加载流程：

1. **Cursor启动** → 扫描扩展目录
2. **读取 `package.json`** → 发现 `"main": "./cursor-vscode-extension.js"`
3. **激活条件检查** → `"activationEvents": ["onStartupFinished"]`
4. **调用 `activate()` 函数** → 加载扩展主入口
5. **扩展完整初始化**：
   ```
   ├── 创建状态栏和输出通道
   ├── 注册所有命令（显示状态、打开设置等）
   ├── 加载配置文件
   ├── 初始化MCP客户端
   ├── 🔥 启动WebSocket集成
   │   ├── 开始用户ID检查循环（每5秒）
   │   ├── 自动建立WebSocket连接
   │   └── 开始接收服务端指令
   └── 设置文件监听器
   ```

---

## ⚙️ 功能特性

### 🎯 **界面集成**
- **状态栏显示**: 右下角显示连接状态
- **命令面板**: `Ctrl+Shift+P` → "Context-Keeper"
- **配置界面**: 内置WebView配置面板
- **日志查看**: 专用输出通道

### 🔌 **WebSocket实时通信**
- **自动连接**: 检测用户ID后自动建立连接
- **心跳保活**: 30秒心跳机制
- **自动重连**: 连接断开后自动重试
- **指令接收**: 实时接收服务端指令

### 💾 **本地指令执行**
- **文件操作**: 自动创建目录和写入文件
- **路径展开**: 支持 `${HOME}`、`${USER}` 等模板
- **短期记忆**: 本地存储会话记忆
- **执行反馈**: 指令执行结果实时回传

### 🧠 **MCP集成**
- **会话管理**: 自动创建和管理编程会话
- **长期记忆**: 重要信息持久化存储
- **上下文检索**: 智能搜索历史记忆
- **文件关联**: 自动关联讨论的代码文件

---

## 🛠️ 使用说明

### 安装扩展
1. 将插件目录链接到VSCode扩展目录
2. 重启Cursor/VSCode
3. 检查右下角状态栏显示

### 配置服务器
1. `Ctrl+Shift+P` → "Context-Keeper: 打开设置"
2. 设置服务器URL（默认：http://localhost:8088）
3. 保存配置

### 查看状态
1. `Ctrl+Shift+P` → "Context-Keeper: 显示状态"
2. `Ctrl+Shift+P` → "Context-Keeper: 显示日志"

---

## 🔍 调试和故障排除

### 状态检查
- **状态栏颜色含义**:
  - 🟢 绿色: WebSocket已连接
  - 🟡 黄色: 正在连接
  - 🟠 橙色: 用户未初始化
  - 🔴 红色: 连接错误

### 日志查看
```javascript
// 查看详细日志
输出面板 → 选择 "Context-Keeper"
```

### 常见问题
1. **状态栏显示"用户未初始化"**: 需要首先通过MCP客户端进行用户初始化
2. **WebSocket连接失败**: 检查服务器是否运行在正确端口
3. **MCP工具调用失败**: 检查服务器URL配置是否正确

---

## 📦 构建和发布

### 开发模式
```bash
# 安装依赖
npm install

# 运行测试
node test.js
```

### 打包扩展
```bash
# 使用vsce打包
npm install -g vsce
vsce package
```

---

## 🤝 技术架构

```
Cursor/VSCode
    ↓ (activate)
cursor-vscode-extension.js (主入口)
    ├── VSCode API → 界面集成
    ├── mcp-client.js → MCP通信
    └── WebSocket → 实时通信
            ↓
    Context-Keeper Server
```

---

**重要**: 本扩展的核心是 `cursor-vscode-extension.js`，所有功能都已集成到这个单一入口文件中，确保扩展的稳定性和维护性。 

## 配置说明

### 基本配置

编辑 `cursor_mcp_config.json` 文件进行配置：

```json
{
  "extensions": {
    "contextKeeper": {
      "config": {
        "serverURL": "http://localhost:8088",
        "websocketURL": "",
        "autoCapture": true,
        "autoAssociate": true,
        "autoRecord": true
      }
    }
  }
}
```

### 配置项说明

- **serverURL**: Context-Keeper 服务器的 HTTP 地址
- **websocketURL**: WebSocket 服务器地址，用于实时指令推送
  - 留空则自动根据 serverURL 生成（推荐）
  - 手动配置示例：`ws://localhost:8088/ws` 或 `wss://your-domain.com/ws`
- **autoCapture**: 自动捕获代码上下文
- **autoAssociate**: 自动关联文件
- **autoRecord**: 自动记录编辑操作

## 使用说明

1. 确保 Context-Keeper 服务器正在运行
2. 配置正确的服务器地址
3. 插件会自动连接并开始工作

## 故障排除

### WebSocket 连接问题

如果 WebSocket 连接失败：

1. 检查 `websocketURL` 配置是否正确
2. 确认服务器支持 WebSocket 连接
3. 查看输出面板的连接日志

### 服务器连接问题

1. 确认 `serverURL` 地址正确
2. 检查服务器是否启动
3. 验证网络连接

## 更多信息

详细文档请参考项目主目录的 README.md 文件。 