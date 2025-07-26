# Context-Keeper VSCode 扩展

## 🎯 为什么要创建真正的VSCode扩展？

经过深入调研，我们发现当前的`cursor-extension.js`只是一个普通的Node.js模块，**没有真正利用编辑器的扩展机制**。

### 问题分析

1. **Cursor基于VSCode构建** - 是VSCode的Fork版本，完全支持VSCode Extension API
2. **当前方案的局限性**：
   - 只是普通Node.js模块，无法访问编辑器内部API
   - 无法使用VSCode的事件钩子机制
   - 无法注册语言功能(Language Features)
   - 无法与编辑器UI深度集成

3. **Microsoft的封锁策略**：
   - 微软开始执行许可证限制，阻止非官方编辑器使用其扩展
   - 社区通过修改`product.json`绕过限制，但每次更新都会重置

## 🚀 新的技术方案

### 架构设计
```
┌─────────────────────────────────────┐
│           VSCode/Cursor             │
├─────────────────────────────────────┤
│     Context-Keeper Extension       │ ← 真正的VSCode扩展
├─────────────────────────────────────┤
│       HTTP Client (axios)          │
├─────────────────────────────────────┤
│    Context-Keeper MCP Server       │ ← 保留现有服务器
└─────────────────────────────────────┘
```

### 核心优势

1. **利用VSCode Extension API**：
   - `vscode.workspace.onDidSaveTextDocument` - 文件保存钩子
   - `vscode.window.onDidChangeActiveTextEditor` - 编辑器切换钩子  
   - `vscode.languages.registerHoverProvider` - 悬停提示
   - `vscode.languages.registerCodeActionsProvider` - 代码建议

2. **深度编辑器集成**：
   - 状态栏显示连接状态
   - 右键菜单添加"存储记忆"选项
   - 快捷键支持 (Ctrl+Shift+M)
   - 智能代码补全基于历史上下文

3. **跨编辑器兼容**：
   - 同时支持VSCode和Cursor
   - 可发布到OpenVSX市场(开源替代)
   - 避免Microsoft许可证限制

## 📦 文件结构

```
vscode-extension/
├── package.json          # 扩展配置和依赖
├── tsconfig.json         # TypeScript配置
├── src/
│   └── extension.ts      # 主扩展文件
├── out/                  # 编译输出
└── README.md            # 说明文档
```

## 🛠️ 核心功能实现

### 1. 自动化钩子
```typescript
// 文件保存时自动记录
vscode.workspace.onDidSaveTextDocument(async (document) => {
    await this.recordFileEdit(document);
});

// 文件打开时自动关联  
vscode.workspace.onDidOpenTextDocument(async (document) => {
    await this.associateFile(document.uri.fsPath);
});
```

### 2. 智能语言功能
```typescript
// 悬停时显示相关记忆
vscode.languages.registerHoverProvider({ scheme: 'file' }, {
    async provideHover(document, position) {
        const word = document.getWordRangeAtPosition(position);
        const memories = await this.searchMemories(word);
        return new vscode.Hover(memories);
    }
});
```

### 3. UI集成
```typescript
// 状态栏显示
this.statusBarItem = vscode.window.createStatusBarItem();
this.statusBarItem.text = "$(brain) Context Keeper";
this.statusBarItem.command = 'contextKeeper.showDashboard';

// 右键菜单
"menus": {
    "editor/context": [{
        "command": "contextKeeper.storeMemory",
        "when": "editorHasSelection"
    }]
}
```

## 🎮 用户体验改进

### 1. 自然的工作流集成
- **保存文件** → 自动记录变更
- **选择代码** → 右键"存储为重要记忆"
- **悬停变量** → 显示相关历史记忆
- **快捷键** → 快速查询上下文

### 2. 可视化界面  
- 状态栏实时显示连接状态
- Webview面板显示查询结果
- 侧边栏显示记忆列表
- 设置页面配置服务器连接

### 3. 智能提示
- 基于历史上下文的代码补全
- 悬停显示相关设计决策
- 错误时提示类似问题的解决方案

## 🚀 安装和使用

### 开发环境
```bash
cd vscode-extension
npm install
npm run compile
```

### 调试扩展
1. 在VSCode中打开该目录
2. 按F5启动调试会话
3. 在新窗口中测试扩展功能

### 打包发布
```bash
npm install -g vsce
vsce package
```

## 📈 与现有MCP服务器的关系

### 保留现有架构
- **MCP服务器**：继续作为数据处理和存储中心
- **VSCode扩展**：作为用户界面和事件捕获层
- **HTTP通信**：扩展通过HTTP与MCP服务器通信

### 优势互补
```
VSCode扩展负责：           MCP服务器负责：
- 编辑器事件捕获           - 数据处理和存储  
- UI交互和显示            - 向量搜索和检索
- 语言功能提供            - 会话管理
- 用户体验优化            - API兼容性
```

## 🎯 下一步计划

1. **完善扩展功能**
   - 实现所有核心API调用
   - 添加错误处理和重试机制  
   - 优化用户体验

2. **测试和验证**
   - 在VSCode和Cursor中测试
   - 验证与现有MCP服务器的兼容性
   - 性能测试和优化

3. **发布和分发**
   - 发布到OpenVSX市场
   - 创建安装说明
   - 用户反馈收集

这个方案**真正利用了编辑器的扩展机制**，解决了当前只是普通Node.js模块的局限性，同时保持了与现有MCP服务器架构的兼容性。 