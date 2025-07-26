# Context-Keeper 用户使用指南

## 🎯 安装完成后的重要说明

**恭喜！Context-Keeper已成功安装。以下是安装后的使用指南。**

---

## 📂 安装位置说明

### 主要文件位置

```bash
~/.context-keeper/                    # 主安装目录
├── config/                          # 配置目录
│   ├── cursor-config-ui.html        # 🌐 图形化配置界面
│   └── default-config.json          # 📄 默认配置文件
├── extensions/                       # 扩展目录
│   └── cursor-extension.js          # Cursor扩展文件
├── logs/                            # 📊 日志目录
│   ├── cursor-extension.log         # 扩展运行日志
│   ├── mcp-client.log              # MCP客户端日志
│   └── error.log                   # 错误日志
├── users/                           # 👤 用户数据目录(用户隔离)
│   └── {userId}/
│       ├── sessions/               # 会话文件
│       └── histories/              # 短期记忆文件
├── mcp-client.js                   # MCP客户端核心文件
├── start-extension.js              # 扩展启动器
├── quick-start.sh                  # 快速启动脚本
└── uninstall.sh                    # 卸载脚本
```

### Cursor配置位置

```bash
~/.cursor/mcp.json                   # 🔧 Cursor MCP配置文件
```

---

## 🎛️ 配置方法

### 方法1: 图形化配置界面（推荐）

**1. 在浏览器中打开配置界面：**

```bash
# 复制以下地址到浏览器地址栏：
file:///Users/你的用户名/.context-keeper/config/cursor-config-ui.html

# 或者直接在终端打开：
open ~/.context-keeper/config/cursor-config-ui.html
```

**2. 配置界面功能：**

- ⚙️ **服务器连接设置**
  - 服务器URL: `http://localhost:8088`
  - 连接超时: `15000ms`
  
- 👤 **用户设置**
  - 用户ID: 自动生成或手动设置
  - 基础目录: `~/.context-keeper`
  
- 🤖 **自动化功能**
  - 自动捕获: ✅ 开启
  - 自动关联: ✅ 开启  
  - 自动记录: ✅ 开启
  - 捕获间隔: `30秒`
  
- 🔄 **重试配置**
  - 最大重试次数: `3次`
  - 重试延迟: `1000ms`
  - 退避倍数: `2`
  
- 📊 **日志设置**
  - 启用日志: ✅ 开启
  - 日志级别: `info`
  - 文件日志: 可选开启

**3. 配置界面操作：**

- 🧪 **测试连接** - 验证服务器连接
- 💾 **保存配置** - 保存当前设置  
- 🔄 **重置默认** - 恢复默认配置
- 🗑️ **清除数据** - 清理所有用户数据

### 方法2: 手动编辑配置文件

**编辑默认配置：**

```bash
# 使用你喜欢的编辑器编辑配置文件
nano ~/.context-keeper/config/default-config.json
# 或
code ~/.context-keeper/config/default-config.json
```

**配置文件格式：**

```json
{
  "serverConnection": {
    "serverURL": "http://localhost:8088",
    "timeout": 15000
  },
  "userSettings": {
    "userId": "your-unique-user-id",
    "baseDir": "~/.context-keeper"
  },
  "automationFeatures": {
    "autoCapture": true,
    "autoAssociate": true, 
    "autoRecord": true,
    "captureInterval": 30
  },
  "retryConfig": {
    "maxRetries": 3,
    "retryDelay": 1000,
    "backoffMultiplier": 2
  },
  "logging": {
    "enabled": true,
    "level": "info",
    "logToFile": true,
    "logFile": "~/.context-keeper/logs/cursor-extension.log"
  }
}
```

---

## 📊 日志查看和调试

### 实时日志查看

```bash
# 查看扩展运行日志
tail -f ~/.context-keeper/logs/cursor-extension.log

# 查看MCP客户端日志  
tail -f ~/.context-keeper/logs/mcp-client.log

# 查看错误日志
tail -f ~/.context-keeper/logs/error.log

# 查看所有日志
tail -f ~/.context-keeper/logs/*.log
```

### 日志级别说明

- 🔴 **error** - 错误信息
- 🟡 **warn** - 警告信息  
- 🔵 **info** - 一般信息
- 🟢 **debug** - 调试信息

### 启用调试日志

编辑配置文件，将日志级别改为`debug`：

```json
{
  "logging": {
    "enabled": true,
    "level": "debug",
    "logToFile": true
  }
}
```

### 日志文件位置

```bash
~/.context-keeper/logs/
├── cursor-extension.log     # 扩展主日志
├── mcp-client.log          # MCP通信日志
├── error.log               # 错误专用日志
└── debug.log               # 调试日志（如果启用）
```

---

## 🚀 快速启动和测试

### 1. 快速启动检查

```bash
# 运行快速启动脚本
~/.context-keeper/quick-start.sh
```

输出示例：
```
🧠 Context-Keeper 快速启动

🔍 检查Context-Keeper服务器状态...
✅ 服务器运行正常
🔗 测试MCP连接...
✅ MCP连接正常

🎉 Context-Keeper已准备就绪！
```

### 2. 手动测试连接

```bash
# 测试服务器健康状态
curl http://localhost:8088/health

# 测试MCP端点
curl http://localhost:8088/mcp/capabilities
```

### 3. 运行集成测试

```bash
# 进入项目目录
cd /path/to/context-keeper/cursor-integration

# 运行完整测试
node test.js

# 运行业务流程测试
node business-flow-test.js
```

---

## 🔧 Cursor集成状态

### 检查MCP配置

```bash
# 查看Cursor MCP配置
cat ~/.cursor/mcp.json

# 检查配置格式是否正确
jq . ~/.cursor/mcp.json
```

### 重启Cursor后的验证

1. **重启Cursor编辑器**
2. **检查MCP服务器状态** - Cursor应该自动连接到Context-Keeper
3. **在Cursor中使用MCP工具** - 应该能看到context-keeper相关工具

---

## 📋 用户数据查看

### 会话数据

```bash
# 查看所有用户
ls ~/.context-keeper/users/

# 查看特定用户的会话
ls ~/.context-keeper/users/your-user-id/sessions/

# 查看会话内容
cat ~/.context-keeper/users/your-user-id/sessions/session-xxx.json
```

### 短期记忆数据

```bash
# 查看用户的对话历史
ls ~/.context-keeper/users/your-user-id/histories/

# 查看特定会话的对话记录
cat ~/.context-keeper/users/your-user-id/histories/session-xxx.json
```

---

## 🛠️ 故障排除

### 常见问题

**1. 找不到配置界面**
```bash
# 确认文件存在
ls -la ~/.context-keeper/config/cursor-config-ui.html

# 重新安装
cd /path/to/context-keeper/cursor-integration
./install.sh
```

**2. 日志文件不生成**
```bash
# 确保日志目录存在
mkdir -p ~/.context-keeper/logs

# 检查配置文件中的日志设置
cat ~/.context-keeper/config/default-config.json | grep -A 5 "logging"
```

**3. MCP连接失败**
```bash
# 检查服务器状态
curl http://localhost:8088/health

# 检查Cursor MCP配置
cat ~/.cursor/mcp.json

# 重启Context-Keeper服务器
# (在项目根目录)
./scripts/deploy/start.sh --http --port 8088 --background
```

**4. 权限问题**
```bash
# 修复文件权限
chmod -R 755 ~/.context-keeper
chmod +x ~/.context-keeper/*.sh
```

### 获取支持

如果遇到问题：

1. **查看日志文件** - 查找错误信息
2. **运行诊断脚本** - `~/.context-keeper/quick-start.sh`
3. **重新安装** - 运行安装脚本
4. **联系支持** - 提交GitHub Issue

---

## 📈 高级用法

### 自定义用户ID

```bash
# 方法1: 通过配置文件
# 编辑 ~/.context-keeper/config/default-config.json
# 设置 "userId": "your-custom-id"

# 方法2: 通过环境变量
export CONTEXT_KEEPER_USER_ID="your-custom-id"
```

### 多用户环境

Context-Keeper支持多用户隔离，每个用户的数据存储在独立目录：

```bash
~/.context-keeper/users/
├── user1/
│   ├── sessions/
│   └── histories/
├── user2/
│   ├── sessions/
│   └── histories/
└── ...
```

### 数据备份

```bash
# 备份用户数据
cp -r ~/.context-keeper/users ~/.context-keeper/backup-$(date +%Y%m%d)

# 备份配置
cp ~/.cursor/mcp.json ~/.cursor/mcp.json.backup
cp ~/.context-keeper/config/default-config.json ~/.context-keeper/config/default-config.json.backup
```

---

## 🎉 享受智能编程体验！

现在你已经完全了解了Context-Keeper的安装位置、配置方法和日志查看。开始使用Cursor进行智能编程吧！

**重要提醒：**
- 🔄 记得重启Cursor以加载MCP配置
- 🌐 使用图形化配置界面更加方便
- 📊 定期查看日志了解系统状态
- 💾 定期备份重要数据 