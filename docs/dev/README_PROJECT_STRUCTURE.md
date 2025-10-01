# Context-Keeper 项目结构说明

本文档描述了Context-Keeper项目的目录结构和组织方式。

## 项目概述

Context-Keeper是一个基于MCP协议的编程上下文管理系统，为AI编程助手提供高效的上下文管理功能。

## 目录结构优化建议

经过代码审查和分析，建议进一步优化项目结构以提高可维护性和代码组织：

```
context-keeper/
├── bin/                  # 编译后的二进制文件
├── cmd/                  # 命令行应用入口
│   └── server/           # 主服务器入口
├── config/               # 配置文件及模板
│   ├── .env              # 环境变量配置
│   ├── .env.example      # 环境变量模板
│   └── manifests/        # MCP等第三方集成清单文件
├── docs/                 # 项目文档
│   ├── api/              # API文档
│   ├── usage/            # 使用说明
│   └── dev/              # 开发者文档
├── internal/             # 内部私有代码，不对外暴露
│   ├── api/              # API处理程序
│   ├── config/           # 配置管理
│   ├── models/           # 数据模型
│   ├── protocol/         # 协议实现
│   ├── services/         # 核心业务逻辑
│   ├── store/            # 数据存储
│   └── utils/            # 内部工具函数
├── pkg/                  # 可被外部项目导入的代码包
│   ├── aliyun/           # 阿里云服务接口
│   ├── embeddings/       # 嵌入向量工具
│   └── mcp/              # MCP协议实现
├── scripts/              # 构建与管理脚本
│   ├── build/            # 构建脚本
│   ├── deploy/           # 部署脚本
│   └── utils/            # 工具脚本
├── tests/                # 测试代码和测试数据
│   ├── integration/      # 集成测试
│   ├── scripts/          # 测试脚本
│   └── unit/             # 单元测试
├── vendor/               # 依赖项（如果使用vendor）
├── _archive/             # 已归档但可能有用的代码
│   ├── stdio/            # STDIO协议相关代码
│   └── legacy-bin/       # 旧版二进制文件
├── _legacy/              # 旧版代码，逐步迁移
├── logs/                 # 日志文件目录
├── data/                 # 数据存储目录
├── .gitignore            # Git忽略配置
├── go.mod                # Go模块定义
├── go.sum                # Go依赖校验
├── Dockerfile            # Docker构建文件
├── README.md             # 项目主readme
├── README_CODE_REFACTORING.md  # 代码重构说明
└── README_PROJECT_STRUCTURE.md # 项目结构说明（本文件）
```

## 优化重点

1. **配置文件集中管理**：
   - 将所有配置文件集中到`config/`目录
   - 按照功能和用途分类存放

2. **文档完善与分类**：
   - 按照API文档、使用说明和开发者文档分类
   - 保持文档与代码同步更新

3. **代码模块化**：
   - 内部代码按功能划分更细致的模块
   - 确保模块之间接口清晰，依赖关系明确

4. **归档代码整理**：
   - 按功能和用途分类存档代码
   - 保留完整的上下文信息，便于未来参考

5. **测试代码组织**：
   - 明确区分单元测试和集成测试
   - 为关键组件提供专门的测试用例

## 下一步工作

1. 迁移配置文件到指定目录
2. 整理文档结构
3. 重构内部代码组织
4. 规范化API接口
5. 完善测试覆盖率

## 代码组织说明

项目代码组织遵循以下原则：

1. **高内聚, 低耦合**: 相关功能放在同一模块，减少模块间依赖
2. **关注点分离**: 清晰分离协议实现、业务逻辑和数据处理
3. **兼容性保证**: 保留关键文件的原位置以确保系统稳定运行
4. **清晰分类**: 测试代码、部署脚本和核心功能明确分开

## 主要组件说明

### 1. 二进制文件

项目主要使用一个二进制文件：

- **context-keeper**: 主程序可执行文件，使用SSE协议，既可以通过`start.sh`脚本独立启动，也可以通过`restart_context_keeper.sh`脚本部署到Cursor

### 2. 通信协议实现

目前主要使用SSE协议实现，STDIO协议实现已归档。核心协议代码：

- **SSE协议**: 内部实现在`internal/api/sse_handler.go`
- **API处理**: 通用API处理在`internal/api/handlers.go`
- **Cursor集成**: Cursor特定API在`internal/api/cursor_handlers.go`

### 3. 配置管理

配置通过多种方式加载，按优先级顺序：

1. 命令行参数
2. 环境变量
3. 配置文件（位于`config/`目录）
4. 默认值

### 4. 存储和数据

数据存储采用双层架构：
- 本地文件系统存储短期记忆和会话状态
- 向量数据库存储长期记忆，支持语义检索

## 运行方式

### 标准服务启动

使用`scripts/deploy/start.sh`脚本启动服务：

```bash
./scripts/deploy/start.sh
```

### Cursor集成

使用`scripts/deploy/restart_context_keeper.sh`脚本部署到Cursor：

```bash
./scripts/deploy/restart_context_keeper.sh
```

## 注意事项

1. 根目录下保留了 `.env` 和 `.env.example` 的原始版本，以保证系统稳定运行
2. `_legacy` 目录包含不再主动使用但可能被引用的代码，不要删除
3. `_archive` 目录包含历史代码和不再使用的实现，可以不随项目发布 