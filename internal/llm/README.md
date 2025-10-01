# LLM 模块使用说明

## 📋 目录结构

```
internal/llm/
├── README.md                    # 本文档
├── types.go                     # 核心类型定义
├── factory.go                   # LLM工厂模式实现
├── deepseek_client.go          # DeepSeek客户端实现
├── openai_client.go            # OpenAI客户端实现
├── claude_client.go            # Claude客户端实现
├── qianwen_client.go           # 千问客户端实现
├── circuit_breaker.go          # 熔断器实现
├── prompt_manager.go           # Prompt工程管理
├── context_aware_service.go    # 上下文感知服务
├── cache_manager.go            # 缓存管理器
├── config_loader.go            # 配置文件加载器
├── env_loader.go               # 环境变量加载器
├── llm_test.go                 # 单元测试
├── integration_test.go         # 集成测试
└── example_usage.go            # 使用示例
```

## ⚙️ 配置说明

### 1. 环境变量配置

在 `config/.env` 文件中添加LLM相关配置：

```bash
# =================================
# LLM API Keys 配置
# =================================

# DeepSeek API Key (主要用于测试)
DEEPSEEK_API_KEY=sk-31206448be1f4e6980ca7450cc8a21cb

# 其他LLM提供商API密钥（可选）
OPENAI_API_KEY=your_openai_api_key_here
CLAUDE_API_KEY=your_claude_api_key_here
QIANWEN_API_KEY=your_qianwen_api_key_here
```

### 2. YAML配置文件

在 `config/llm_config.yaml` 中配置LLM服务：

```yaml
llm:
  default:
    primary_provider: "deepseek"
    fallback_provider: "openai"
    cache_enabled: true
    cache_ttl: "30m"
    max_retries: 3
    timeout_seconds: 30
    enable_routing: true

  providers:
    deepseek:
      api_key: "${DEEPSEEK_API_KEY}"
      base_url: "https://api.deepseek.com/v1"
      model: "deepseek-chat"
      max_retries: 3
      timeout: "30s"
      rate_limit: 60
```

## 🚀 使用方式

### 方式1：简单使用（推荐新手）

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/contextkeeper/service/internal/llm"
)

func main() {
    // 1. 创建简单客户端
    client := llm.NewSimpleLLMClient(llm.ProviderDeepSeek, "your-api-key")
    
    // 2. 进行对话
    ctx := context.Background()
    response, err := client.Chat(ctx, "请介绍一下Go语言")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("AI回复:", response)
}
```

### 方式2：配置文件使用（推荐生产环境）

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/contextkeeper/service/internal/llm"
)

func main() {
    // 1. 从配置文件初始化
    err := llm.InitializeWithEnv()
    if err != nil {
        log.Fatal("初始化失败:", err)
    }
    
    // 2. 获取全局服务
    service, err := llm.GetGlobalService()
    if err != nil {
        log.Fatal("获取服务失败:", err)
    }
    
    // 3. 使用三要素分析
    ctx := context.Background()
    sessionHistory := []llm.Message{
        {Role: "user", Content: "我在用Go开发微服务"},
        {Role: "user", Content: "遇到了性能问题"},
    }
    
    workspaceContext := &llm.WorkspaceContext{
        ProjectType: "microservices",
        TechStack:   []string{"Go", "Redis", "Docker"},
        ProjectName: "my-project",
        Environment: "production",
    }
    
    threeElements, err := service.AnalyzeThreeElementsWithLLM(
        ctx, sessionHistory, workspaceContext)
    if err != nil {
        log.Fatal("三要素分析失败:", err)
    }
    
    fmt.Printf("用户技术栈: %v\n", threeElements.User.TechStack)
    fmt.Printf("用户经验水平: %s\n", threeElements.User.ExperienceLevel)
    fmt.Printf("项目类型: %s\n", threeElements.Situation.ProjectType)
    fmt.Printf("问题意图: %s\n", threeElements.Problem.Intent)
}
```

### 方式3：高级使用（自定义配置）

```go
package main

import (
    "context"
    "time"
    
    "github.com/contextkeeper/service/internal/llm"
)

func main() {
    // 1. 创建自定义配置
    config, err := llm.NewConfigBuilder(llm.ProviderDeepSeek).
        WithAPIKey("your-api-key").
        WithModel("deepseek-chat").
        WithTimeout(60 * time.Second).
        WithMaxRetries(5).
        Build()
    if err != nil {
        panic(err)
    }
    
    // 2. 设置全局配置
    llm.SetGlobalConfig(llm.ProviderDeepSeek, config)
    
    // 3. 创建客户端
    client, err := llm.CreateGlobalClient(llm.ProviderDeepSeek)
    if err != nil {
        panic(err)
    }
    
    // 4. 发送请求
    ctx := context.Background()
    req := &llm.LLMRequest{
        Prompt:      "Hello, world!",
        MaxTokens:   100,
        Temperature: 0.7,
    }
    
    resp, err := client.Complete(ctx, req)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("响应: %s\n", resp.Content)
    fmt.Printf("Token使用: %d\n", resp.TokensUsed)
    fmt.Printf("耗时: %v\n", resp.Duration)
}
```

## 🧪 测试说明

### 1. 运行单元测试

```bash
# 运行所有单元测试
go test ./internal/llm -v

# 运行特定测试
go test ./internal/llm -v -run TestLLMFactory
```

### 2. 运行集成测试

```bash
# 设置环境变量并运行集成测试
RUN_INTEGRATION_TESTS=1 go test ./internal/llm -v -run TestDeepSeekIntegration

# 只运行基本对话测试
RUN_INTEGRATION_TESTS=1 go test ./internal/llm -v -run TestDeepSeekIntegration/基本对话测试

# 只运行三要素分析测试
RUN_INTEGRATION_TESTS=1 go test ./internal/llm -v -run TestDeepSeekIntegration/三要素分析测试
```

### 3. 运行性能基准测试

```bash
# 运行基准测试
RUN_INTEGRATION_TESTS=1 go test ./internal/llm -bench=. -v
```

## 📊 核心功能

### 1. 支持的LLM提供商

- ✅ **DeepSeek**: 代码理解和生成
- ✅ **OpenAI**: GPT系列模型
- ✅ **Claude**: Anthropic的Claude模型
- ✅ **千问**: 阿里云千问模型

### 2. 核心特性

- 🔄 **自动降级**: 主要提供商失败时自动切换到备用提供商
- 🔁 **重试机制**: 指数退避重试策略
- 🚦 **限流控制**: 基于令牌桶的限流
- 🔌 **熔断器**: 防止级联故障
- 💾 **智能缓存**: 基于内容哈希的缓存机制
- 🎯 **智能路由**: 根据任务类型选择最适合的提供商
- 📝 **Prompt工程**: 模板化Prompt管理

### 3. 高级功能

- **三要素分析**: 分析用户、情景、问题三要素
- **查询改写**: 基于上下文智能改写查询
- **上下文感知**: 结合会话历史和工作空间信息

## 🔧 故障排除

### 1. 常见错误

**错误**: `config/.env文件不存在`
**解决**: 确保在项目根目录的config文件夹中有.env文件

**错误**: `环境变量 DEEPSEEK_API_KEY 未设置`
**解决**: 在config/.env文件中设置正确的API密钥

**错误**: `所有LLM提供商都不可用`
**解决**: 检查网络连接和API密钥是否正确

### 2. 调试技巧

1. **启用详细日志**: 运行测试时会自动显示详细的执行日志
2. **检查配置**: 使用`TestConfigLoading`测试验证配置是否正确加载
3. **单独测试**: 使用`TestSimpleUsage`测试单个提供商是否工作正常

## 📈 性能优化

1. **缓存策略**: 启用缓存可以显著减少重复请求
2. **并发控制**: 合理设置限流参数避免API限制
3. **超时设置**: 根据网络环境调整超时时间
4. **模型选择**: 根据任务类型选择合适的模型

## 🔮 扩展指南

### 添加新的LLM提供商

1. 实现`LLMClient`接口
2. 在工厂中注册新提供商
3. 添加配置支持
4. 编写测试用例

### 自定义Prompt模板

1. 创建`PromptTemplate`实例
2. 使用`PromptManager.RegisterTemplate()`注册
3. 通过`BuildPrompt()`使用模板
