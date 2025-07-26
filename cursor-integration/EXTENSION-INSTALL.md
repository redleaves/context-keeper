xi# Context-Keeper VSCode/Cursor 扩展安装指南

## 🚀 快速安装

### 方法1：开发模式安装（推荐用于测试）

1. **克隆或下载扩展文件**：
   ```bash
   # 确保你在 context-keeper/cursor-integration 目录下
   cd /path/to/context-keeper/cursor-integration
   ```

2. **启动Context-Keeper服务**：
   ```bash
   # 返回主目录启动服务
   cd ..
   ./context-keeper-http  # 或使用 scripts/start.sh
   ```

3. **在VSCode/Cursor中安装扩展**：
   - 打开 VSCode/Cursor
   - 按 `Ctrl+Shift+P` (Windows/Linux) 或 `Cmd+Shift+P` (Mac)
   - 输入 "Developer: Install Extension from Location"
   - 选择 `cursor-integration` 文件夹

### 方法2：打包安装

1. **安装VSCE工具**：
   ```bash
   npm install -g vsce
   ```

2. **打包扩展**：
   ```bash
   cd cursor-integration
   vsce package
   ```

3. **安装生成的.vsix文件**：
   - 在VSCode/Cursor中：Extensions -> Install from VSIX...
   - 选择生成的 `context-keeper-1.0.0.vsix` 文件

## 🎛️ 扩展功能

### 状态栏集成
- 右下角会显示 "🧠 CK: [状态]" 
- 点击可快速查看连接状态

### 命令面板功能
按 `Ctrl+Shift+P` 输入以下命令：

- **Context-Keeper: 显示状态面板** - 查看详细状态和统计
- **Context-Keeper: 打开设置** - 配置服务器连接和功能
- **Context-Keeper: 测试连接** - 验证与服务器的连接
- **Context-Keeper: 重置配置** - 恢复默认设置

### 侧边栏面板
- 在左侧资源管理器中有独立的 Context-Keeper 面板
- 显示实时状态和快速操作按钮

## ⚙️ 配置选项

在 VSCode/Cursor 设置中搜索 "context-keeper"：

### 服务器连接
- `context-keeper.serverURL`: 服务器地址 (默认: http://localhost:8088)
- `context-keeper.timeout`: 连接超时时间 (默认: 15000ms)

### 用户设置
- `context-keeper.userId`: 用户ID (留空自动生成)

### 自动化功能
- `context-keeper.autoCapture`: 自动捕获编程活动 (默认: true)
- `context-keeper.autoAssociate`: 自动关联相关文件 (默认: true)
- `context-keeper.autoRecord`: 自动记录代码编辑 (默认: true)
- `context-keeper.captureInterval`: 捕获间隔秒数 (默认: 30)

### 日志设置
- `context-keeper.logging.enabled`: 启用日志记录 (默认: true)
- `context-keeper.logging.level`: 日志级别 (默认: info)

## 🔧 故障排除

### 扩展无法启动
1. 检查 Context-Keeper 服务是否运行：
   ```bash
   curl http://localhost:8088/health
   ```

2. 查看扩展日志：
   - View -> Output -> 选择 "Context-Keeper"

### 连接失败
1. 确认服务器URL正确
2. 检查防火墙设置
3. 验证端口8088未被占用

### 功能异常
1. 重启扩展：
   - 命令面板 -> "Developer: Reload Window"

2. 重置配置：
   - 命令面板 -> "Context-Keeper: 重置配置"

## 📱 使用示例

### 基本工作流程

1. **启动**：
   - Context-Keeper 服务自动启动
   - 扩展连接并显示绿色状态

2. **编程**：
   - 正常编写代码
   - 扩展自动捕获和关联文件

3. **查看状态**：
   - 点击状态栏图标查看统计信息
   - 在状态面板中查看详细数据

4. **配置调整**：
   - 通过设置面板调整自动化功能
   - 实时生效，无需重启

## 🎯 核心优势

- ✅ **真正的IDE内置体验** - 无需外部浏览器
- ✅ **自动化捕获** - 无感知的上下文管理
- ✅ **实时监控** - 状态栏和面板实时更新
- ✅ **配置灵活** - 所有功能可自定义开关
- ✅ **完整集成** - 命令、菜单、视图全覆盖

## 🚨 注意事项

- 扩展需要 Context-Keeper 后端服务运行
- 首次使用会自动生成用户ID
- 所有数据存储在本地 `~/.context-keeper` 目录
- 扩展与 MCP 协议完全兼容

---

**问题反馈**: 如有问题请检查日志输出或重启服务 