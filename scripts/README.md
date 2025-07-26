# Context-Keeper 脚本管理体系

本目录包含 Context-Keeper 项目的完整脚本管理体系，支持编译、启动、停止、重启和监控等功能。

## 🚀 快速开始

### 一键启动服务
```bash
# 启动HTTP模式（推荐）
./scripts/manage.sh deploy http

# 启动STDIO模式
./scripts/manage.sh deploy stdio

# 指定端口启动HTTP模式
./scripts/manage.sh deploy http --port 8080
```

### 检查服务状态
```bash
./scripts/manage.sh status
```

### 查看服务日志
```bash
# 查看HTTP服务日志
./scripts/manage.sh logs http

# 查看STDIO服务日志
./scripts/manage.sh logs stdio
```

## 📁 脚本目录结构

```
scripts/
├── manage.sh              # 主管理脚本（推荐使用）
├── start-daemon.sh        # 守护进程启动脚本
├── build/
│   └── build.sh          # 编译脚本
├── deploy/
│   ├── start.sh          # 传统启动脚本
│   ├── logs.sh           # 日志查看脚本
│   └── restart_context_keeper.sh
└── utils/
    └── store_conversation.sh
```

## 🛠️ 主要脚本说明

### 1. manage.sh - 主管理脚本

这是推荐使用的主要管理工具，提供完整的服务生命周期管理。

**功能特性：**
- ✅ 独立的编译、启动、停止操作
- ✅ 一键部署（停止→编译→启动）
- ✅ 后台进程管理（默认后台运行）
- ✅ PID文件管理和进程监控
- ✅ 服务状态检查（CPU、内存、运行时间）
- ✅ 日志查看和管理
- ✅ 清理功能

**使用方法：**
```bash
# 编译
./scripts/manage.sh build                    # 编译所有版本
./scripts/manage.sh build --stdio          # 仅编译STDIO版本
./scripts/manage.sh build --http           # 仅编译HTTP版本

# 启动服务
./scripts/manage.sh start http             # 启动HTTP模式（后台）
./scripts/manage.sh start stdio            # 启动STDIO模式（后台）
./scripts/manage.sh start http --port 8080 --foreground  # 前台运行

# 停止服务
./scripts/manage.sh stop http              # 停止HTTP服务
./scripts/manage.sh stop stdio             # 停止STDIO服务
./scripts/manage.sh stop                   # 停止所有服务

# 重启服务
./scripts/manage.sh restart http           # 重启HTTP服务
./scripts/manage.sh restart stdio --port 8080  # 重启并修改端口

# 一键部署
./scripts/manage.sh deploy http            # 一键部署HTTP服务
./scripts/manage.sh deploy stdio           # 一键部署STDIO服务

# 状态和日志
./scripts/manage.sh status                 # 查看所有服务状态
./scripts/manage.sh logs http              # 查看HTTP服务日志
./scripts/manage.sh logs stdio 100         # 查看STDIO服务最近100行日志

# 清理
./scripts/manage.sh clean                  # 清理编译产物和日志
```

### 2. start-daemon.sh - 守护进程脚本

专门用于守护进程启动，支持自动重启和系统服务安装。

**功能特性：**
- ✅ 守护进程模式运行
- ✅ 自动监控和重启
- ✅ 配置文件保存/加载
- ✅ macOS系统服务安装
- ✅ 详细的守护进程日志

**使用方法：**
```bash
# 启动守护进程
./scripts/start-daemon.sh                  # 使用默认配置
./scripts/start-daemon.sh --mode stdio     # 指定模式
./scripts/start-daemon.sh --port 8080      # 指定端口
./scripts/start-daemon.sh --no-auto-restart # 禁用自动重启

# 配置管理
./scripts/start-daemon.sh --save-config    # 保存当前配置
./scripts/start-daemon.sh --load-config    # 加载配置文件

# 系统服务（macOS）
./scripts/start-daemon.sh --install-service    # 安装系统服务
./scripts/start-daemon.sh --uninstall-service  # 卸载系统服务
```

### 3. build.sh - 编译脚本

独立的编译脚本，支持多种编译模式。

```bash
./scripts/build/build.sh --all     # 编译所有版本
./scripts/build/build.sh --stdio   # 仅编译STDIO版本
./scripts/build/build.sh --http    # 仅编译HTTP版本
```

## 🔧 后台进程管理

### 启动后台服务
```bash
# 方法1：使用管理脚本（推荐）
./scripts/manage.sh start http

# 方法2：使用守护进程脚本
./scripts/start-daemon.sh --mode http
```

### 服务不受终端关闭影响的原理
1. **nohup命令**：忽略SIGHUP信号
2. **后台运行**：使用`&`将进程放入后台
3. **PID文件管理**：记录进程ID用于后续管理
4. **日志重定向**：将输出重定向到日志文件

### 进程监控和管理
```bash
# 检查服务状态
./scripts/manage.sh status

# 查看进程详情
ps aux | grep context-keeper

# 手动停止进程
./scripts/manage.sh stop http
```

## 📊 日志管理

### 日志文件位置
```
logs/
├── context-keeper-http.log      # HTTP服务日志
├── context-keeper-stdio.log     # STDIO服务日志
├── context-keeper-http.pid      # HTTP服务PID文件
├── context-keeper-stdio.pid     # STDIO服务PID文件
├── daemon-http.log              # HTTP守护进程日志
└── daemon-stdio.log             # STDIO守护进程日志
```

### 日志查看命令
```bash
# 使用管理脚本查看日志
./scripts/manage.sh logs http              # 最近50行
./scripts/manage.sh logs http 100          # 最近100行

# 直接查看日志文件
tail -f logs/context-keeper-http.log       # 实时查看
tail -n 100 logs/context-keeper-http.log   # 查看最近100行
```

## 🔄 常用操作场景

### 场景1：开发调试
```bash
# 前台运行便于调试
./scripts/manage.sh start http --foreground

# 或者后台运行+实时查看日志
./scripts/manage.sh start http
tail -f logs/context-keeper-http.log
```

### 场景2：生产部署
```bash
# 一键部署
./scripts/manage.sh deploy http --port 8088

# 验证部署
./scripts/manage.sh status

# 设置守护进程（可选）
./scripts/start-daemon.sh --mode http --port 8088 --save-config
```

### 场景3：服务重启
```bash
# 简单重启
./scripts/manage.sh restart http

# 完整重新部署
./scripts/manage.sh deploy http
```

### 场景4：问题排查
```bash
# 检查服务状态
./scripts/manage.sh status

# 查看最近日志
./scripts/manage.sh logs http

# 重启服务
./scripts/manage.sh restart http

# 如果问题持续，清理后重新部署
./scripts/manage.sh clean
./scripts/manage.sh deploy http
```

## 🛡️ 系统服务安装（macOS）

### 安装为系统服务
```bash
# 1. 保存配置
./scripts/start-daemon.sh --mode http --port 8088 --save-config

# 2. 安装系统服务
./scripts/start-daemon.sh --install-service
```

### 管理系统服务
```bash
# 启动服务
launchctl start com.context-keeper.daemon

# 停止服务
launchctl stop com.context-keeper.daemon

# 卸载服务
./scripts/start-daemon.sh --uninstall-service
```

## ⚠️ 注意事项

1. **端口冲突**：启动HTTP模式前会自动检查端口占用
2. **权限要求**：脚本需要执行权限，使用`chmod +x scripts/*.sh`
3. **依赖检查**：启动前会自动检查并编译所需的二进制文件
4. **日志轮转**：建议定期清理日志文件或设置日志轮转
5. **资源监控**：可通过`manage.sh status`监控服务资源使用情况

## 🔍 故障排除

### 常见问题

1. **编译失败**
   ```bash
   # 检查Go环境
   go version
   
   # 清理后重新编译
   ./scripts/manage.sh clean
   ./scripts/manage.sh build
   ```

2. **启动失败**
   ```bash
   # 查看详细错误日志
   ./scripts/manage.sh logs http
   
   # 检查端口占用
   lsof -i:8088
   ```

3. **服务异常退出**
   ```bash
   # 使用守护进程自动重启
   ./scripts/start-daemon.sh --mode http
   ```

4. **无法停止服务**
   ```bash
   # 强制停止所有相关进程
   pkill -f context-keeper
   
   # 清理PID文件
   rm -f logs/*.pid
   ```

## 📝 更多帮助

- 查看管理脚本帮助：`./scripts/manage.sh help`
- 查看守护进程帮助：`./scripts/start-daemon.sh --help`
- 查看编译脚本帮助：`./scripts/build/build.sh --help` 