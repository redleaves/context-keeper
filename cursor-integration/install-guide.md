# 🚀 Context-Keeper Cursor扩展安装指南

## ✅ 前置条件检查

Context-Keeper服务已正在运行：
- 端口：8088
- 状态：健康
- 连接：正常

## 📋 安装步骤

### 步骤1：在Cursor中安装扩展

1. **打开Cursor IDE**
2. **打开命令面板**：
   - 按 `Cmd+Shift+P` (macOS) 或 `Ctrl+Shift+P` (Windows/Linux)
3. **输入安装命令**：
   ```
   Developer: Install Extension from Location
   ```
4. **选择扩展目录**：
   ```
   /Users/weixiaofeng12/coding/context-keeper/cursor-integration
   ```
5. **确认安装**

### 步骤2：验证安装

安装成功后，你会看到：

✅ **状态栏指示器**：
- 右下角显示：`🧠 CK: 初始化中...`
- 几秒后变为：`🧠 CK: 已连接`

✅ **命令面板功能**：
- 按 `Cmd+Shift+P`
- 搜索 "Context-Keeper"
- 应该看到13个可用命令

✅ **侧边栏面板**：
- 左侧资源管理器中出现 "Context-Keeper" 面板

### 步骤3：测试功能

1. **查看状态**：
   - 点击状态栏的 `🧠 CK:` 图标
   - 打开状态面板

2. **测试连接**：
   - 命令面板 → "Context-Keeper: 测试连接"
   - 应该显示连接成功

3. **查看设置**：
   - 命令面板 → "Context-Keeper: 打开设置"
   - 配置服务器连接等选项

## 🎯 预期效果

安装成功后，扩展会：

- ✅ 自动连接到本地Context-Keeper服务
- ✅ 在状态栏显示实时连接状态  
- ✅ 提供完整的命令面板集成
- ✅ 自动监听文件变化
- ✅ 自动记录编程上下文

## 🔧 故障排除

### 如果状态栏显示"连接失败"：
1. 确认Context-Keeper服务正在运行
2. 检查端口8088是否可访问
3. 查看扩展日志：View → Output → Context-Keeper

### 如果找不到命令：
1. 重新加载窗口：Developer: Reload Window
2. 确认扩展已正确安装

### 如果扩展未启动：
1. 检查VSCode/Cursor版本 ≥ 1.74.0
2. 确认所有必需文件存在

## 📞 测试验证

运行以下命令验证后端连接：
```bash
curl http://localhost:8088/health
```

应该返回：
```json
{"mode":"streamable-http","status":"healthy"}
```

---

**安装完成后，Context-Keeper将成为你编程工作流程的智能助手！** 🎉 