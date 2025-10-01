# Context-Keeper 代码重构说明

## 重构目标

本次代码重构的主要目标是：

1. **提高代码组织的合理性**：按照高内聚、低耦合的原则重新组织项目结构
2. **分离关注点**：清晰分离协议实现、业务逻辑和数据处理
3. **保持兼容性**：保证现有功能不受影响，系统能够稳定运行
4. **提高可维护性**：整理测试脚本和工具脚本，使项目结构更清晰

## 重构内容

### 1. 目录结构优化

#### 1.1 二进制文件整理

可执行文件移动到 `bin/` 目录并统一命名：
- `context-keeper` -> `bin/context-keeper` (主服务二进制文件)
- 早期版本二进制文件已移至`_archive/bin/`

#### 1.2 脚本文件整理

脚本文件按功能分类：
- 部署脚本: `scripts/deploy/`
  - `start.sh`
  - `restart_context_keeper.sh`
- 工具脚本: `scripts/utils/`
  - `store_conversation.sh`

#### 1.3 测试文件整理

测试相关文件集中管理：
- 测试脚本: `tests/scripts/`
  - 所有 `test_*.sh` 文件
- 测试数据: `tests/data/`
  - `test_memory_store.json`
  - `real_conversation.json`
- 测试代码: `tests/`
  - `test_store.go`

#### 1.4 配置文件整理

配置文件集中到 `config/` 目录(同时保留原位置以保证兼容性)：
- `.env` -> `config/.env`
- `.env.example` -> `config/.env.example`
- `config-template.json` -> `config/config-template.json`
- `claude_desktop_config.json` -> `config/claude_desktop_config.json`
- `context-keeper-manifest.json` -> `config/context-keeper-manifest.json`

#### 1.5 日志文件整理

日志文件移动到专门目录：
- `service.log` -> `logs/service.log`

### 2. 内部结构优化

#### 2.1 协议实现分离

创建专门的协议实现目录：
- `internal/protocol/sse/` - 用于SSE协议相关代码
- `internal/protocol/stdio/` - 用于备用协议相关代码

#### 2.2 历史代码保存

不再主动使用的代码进行适当归档：
- 历史代码备份: `_legacy/` - 保留在项目中的历史代码
- 存档代码: `_archive/` - 不再使用的完整实现，可以不随项目发布

### 3. 脚本路径更新

已更新脚本中的路径引用，使其适应新的目录结构：

- `scripts/deploy/start.sh` - 更新了路径引用和工作目录设置
- `scripts/deploy/restart_context_keeper.sh` - 更新了编译输出路径和日志文件位置
- `tests/scripts/test_api.sh` - 更新了测试数据路径和工作目录设置
- `tests/scripts/test_cursor_mcp_status.sh` - 更新了日志文件路径

### 4. 代码修改

为了适应新的目录结构，同时保持向后兼容性，我们对代码进行了以下修改：

#### 4.1 配置加载逻辑更新

修改了 `internal/config/config.go` 中的环境变量加载逻辑，增加了对新路径的支持：

```go
// 尝试加载.env文件，优先尝试新的目录结构，然后兼容原来的结构
envPaths := []string{
    "config/.env",
    ".env",
}

loaded := false
for _, path := range envPaths {
    if _, err := os.Stat(path); err == nil {
        if err := godotenv.Load(path); err == nil {
            log.Printf("成功加载.env文件: %s", path)
            loaded = true
            break
        }
    }
}

if !loaded {
    log.Printf("警告: 未找到.env文件，尝试使用系统环境变量")
}
```

#### 4.2 配置模板路径更新

在 `config/config-template.json` 中更新了日志文件的相对路径，确保从配置目录能正确找到日志目录：

```json
"log": {
  "level": "info",
  "path": "../logs/service.log",
  "max_size_mb": 100,
  "max_backups": 5
}
```

#### 4.3 测试脚本工作目录设置

在所有测试脚本中添加了工作目录设置逻辑，确保无论从哪里调用脚本都能正确找到相关文件：

```bash
# 设置工作目录为项目根目录
cd "$(dirname "$0")/../.." || exit 1
WORKSPACE_DIR=$(pwd)
echo -e "${GREEN}工作目录: ${WORKSPACE_DIR}${NC}"
```

#### 4.4 二进制文件命名统一

- 统一使用 `context-keeper` 作为二进制文件名
- 更新 `restart_context_keeper.sh` 脚本中的引用，使用统一命名约定
- 将早期版本二进制文件 `context_keeper_bin` 归档到 `_archive/bin/` 目录

## 兼容性说明

为确保系统稳定运行，我们采取了以下兼容性措施：

1. **保留关键文件原位置**：环境变量文件(`.env`, `.env.example`)保留在根目录，同时复制到config目录
2. **原地复制而非移动**：对于可能被硬编码引用的文件，采用复制而非直接移动
3. **添加工作目录检测**：在脚本中添加了工作目录检测和设置，确保无论从哪里调用都能正确运行
4. **路径参数化**：将硬编码的路径改为相对路径和变量引用，提高灵活性
5. **多路径加载逻辑**：在配置加载时尝试多个可能的路径，实现向后兼容
6. **代码归档分级**：对不同程度不再使用的代码采用不同的归档策略，保留必要的历史代码

## 后续优化建议

1. **进一步模块化**：将业务逻辑按功能进一步拆分为更小的模块
2. **接口统一**：规范化API接口，使其更符合RESTful设计原则
3. **配置中心化**：实现中心化的配置管理，减少配置散落
4. **测试自动化**：添加自动化测试框架，提高测试覆盖率
5. **文档更新**：更新API文档和部署文档，反映新的项目结构 