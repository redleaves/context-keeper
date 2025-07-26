# Context-Keeper 安装指南

Context-Keeper 是一个符合MCP（Model Context Protocol）协议的上下文服务，为编程助手和代码理解场景提供强大的上下文管理能力。本文档将指导您如何安装和配置 Context-Keeper。

## 目录

- [支持的平台](#支持的平台)
- [安装步骤](#安装步骤)
  - [Windows](#windows)
  - [macOS/Linux](#macoslinux)
- [配置 Cursor](#配置-cursor)
- [验证安装](#验证安装)
- [常见问题](#常见问题)
- [故障排除](#故障排除)

## 支持的平台

Context-Keeper 支持以下操作系统:

- Windows 10/11
- macOS (Intel 和 Apple Silicon)
- Linux (x86_64/amd64)

## 安装步骤

### Windows

1. 解压下载的安装包到一个临时目录
2. 双击运行 `install.bat`
3. 按照提示选择安装目录（或直接按回车使用默认目录）
4. 等待安装完成

### macOS/Linux

1. 解压下载的安装包到一个临时目录
2. 打开终端，进入解压后的目录
3. 运行安装脚本：
   ```bash
   chmod +x install.sh
   ./install.sh
   ```
4. 按照提示选择安装目录（或直接按回车使用默认目录）
5. 等待安装完成

## 配置 Cursor

安装完成后，您需要配置 Cursor 编辑器以使用 Context-Keeper:

1. 打开 Cursor 编辑器
2. 打开设置（Windows/Linux: Ctrl+, | macOS: Cmd+,）
3. 搜索 "MCP" 或 "Model Context Protocol"
4. 在 MCP 服务器配置部分，添加 Context-Keeper 执行文件的路径：
   - Windows: `C:\Users\<用户名>\.context-keeper\bin\context-keeper.exe`
   - macOS/Linux: `/Users/<用户名>/.context-keeper/bin/context-keeper`
   
   (注意：路径可能因您的安装位置而异，请使用安装脚本最后显示的路径)

5. 保存设置并重启 Cursor

## 验证安装

要验证 Context-Keeper 是否安装正确，可以运行以下命令：

**Windows:**
```
%USERPROFILE%\.context-keeper\bin\context-keeper.exe --version
```

**macOS/Linux:**
```
~/.context-keeper/bin/context-keeper --version
```

如果安装成功，您将看到版本信息输出。

## 常见问题

### Q: 我如何卸载 Context-Keeper？

A: 只需删除安装目录：
- Windows: `%USERPROFILE%\.context-keeper`
- macOS/Linux: `~/.context-keeper`

### Q: Context-Keeper 需要什么系统要求？

A: Context-Keeper 是一个轻量级应用程序，不需要特殊的系统要求。它不依赖任何外部运行时（如 Go 环境）。

### Q: 数据存储在哪里？

A: 所有数据默认存储在安装目录下的 `data` 文件夹中。

### Q: 如何升级 Context-Keeper？

A: 下载新版本的安装包，然后重新运行安装脚本。您的数据和配置将被保留。

## 故障排除

### 问题: Cursor 无法连接到 Context-Keeper

**解决方案**:
1. 确认 Context-Keeper 可执行文件路径在 Cursor 设置中配置正确
2. 检查 Context-Keeper 是否可以独立运行（使用上述验证命令）
3. 确保您有权限执行该文件
4. 检查 Cursor 日志中是否有错误信息

### 问题: 安装失败

**解决方案**:
1. 确保您有足够的权限创建安装目录
2. 检查是否有防病毒软件阻止安装
3. 尝试以管理员/超级用户身份运行安装脚本

---

如果您遇到任何问题，请联系支持团队或提交 GitHub issue。

感谢使用 Context-Keeper！ 