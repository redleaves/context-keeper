package llm

import (
	"context"
	"fmt"
	"log"
	"time"
)

// =============================================================================
// 使用示例 - 这些函数展示了如何使用LLM模块
// =============================================================================

// ExampleSimpleUsage 示例1：简单使用方式
func ExampleSimpleUsage() {
	fmt.Println("=== 示例1：简单使用方式 ===")

	// 1. 创建简单客户端 - 只需要传入提供商和API密钥
	client := NewSimpleLLMClient(ProviderDeepSeek, "your-api-key")

	// 2. 直接对话
	ctx := context.Background()
	response, err := client.Chat(ctx, "请用一句话介绍Go语言")
	if err != nil {
		log.Printf("对话失败: %v", err)
		return
	}

	fmt.Printf("AI回复: %s\n", response)
}

// ExampleConfigFileUsage 示例2：配置文件使用方式（推荐）
func ExampleConfigFileUsage() {
	fmt.Println("\n=== 示例2：配置文件使用方式 ===")

	// 1. 从配置文件初始化（会自动加载config/.env和config/llm_config.yaml）
	err := InitializeWithEnv()
	if err != nil {
		log.Printf("初始化失败: %v", err)
		return
	}
	fmt.Println("✅ 配置初始化成功")

	// 2. 获取全局服务
	service, err := GetGlobalService()
	if err != nil {
		log.Printf("获取服务失败: %v", err)
		return
	}
	fmt.Println("✅ 获取全局服务成功")

	// 3. 使用基本LLM客户端
	client, err := CreateGlobalClient(ProviderDeepSeek)
	if err != nil {
		log.Printf("创建客户端失败: %v", err)
		return
	}

	ctx := context.Background()
	req := &LLMRequest{
		Prompt:      "解释一下什么是微服务架构",
		MaxTokens:   200,
		Temperature: 0.7,
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		log.Printf("LLM调用失败: %v", err)
		return
	}

	fmt.Printf("✅ LLM调用成功\n")
	fmt.Printf("📄 响应内容: %s\n", resp.Content)
	fmt.Printf("📊 Token使用: %d\n", resp.TokensUsed)
	fmt.Printf("⏱️  耗时: %v\n", resp.Duration)

	// 4. 使用高级功能 - 三要素分析
	fmt.Println("\n--- 三要素分析示例 ---")
	sessionHistory := []Message{
		{Role: "user", Content: "我在用Go开发微服务项目", Timestamp: time.Now()},
		{Role: "user", Content: "遇到了Redis缓存性能问题", Timestamp: time.Now()},
	}

	workspaceContext := &WorkspaceContext{
		ProjectType: "microservices",
		TechStack:   []string{"Go", "Redis", "Docker"},
		ProjectName: "ecommerce-platform",
		Environment: "production",
	}

	threeElements, err := service.AnalyzeThreeElementsWithLLM(ctx, sessionHistory, workspaceContext)
	if err != nil {
		log.Printf("三要素分析失败: %v", err)
		return
	}

	fmt.Printf("✅ 三要素分析完成\n")
	fmt.Printf("👤 用户技术栈: %v\n", threeElements.User.TechStack)
	fmt.Printf("👤 用户经验水平: %s\n", threeElements.User.ExperienceLevel)
	fmt.Printf("👤 用户角色: %s\n", threeElements.User.Role)
	fmt.Printf("🏢 项目类型: %s\n", threeElements.Situation.ProjectType)
	fmt.Printf("🏢 业务背景: %s\n", threeElements.Situation.BusinessContext)
	fmt.Printf("❓ 问题意图: %s\n", threeElements.Problem.Intent)
	fmt.Printf("❓ 明确问题: %s\n", threeElements.Problem.ExplicitProblem)

	// 5. 使用查询改写
	fmt.Println("\n--- 查询改写示例 ---")
	originalQuery := "如何优化性能"
	rewriteResult, err := service.RewriteQueryWithLLM(ctx, originalQuery, threeElements, &RetrievalContext{
		Strategy:  "semantic",
		TopK:      10,
		Threshold: 0.7,
	})

	if err != nil {
		log.Printf("查询改写失败: %v", err)
		return
	}

	fmt.Printf("✅ 查询改写完成\n")
	fmt.Printf("📝 原始查询: %s\n", rewriteResult.OriginalQuery)
	fmt.Printf("📝 改写查询: %s\n", rewriteResult.RewrittenQuery)
	fmt.Printf("📝 扩展词汇: %v\n", rewriteResult.Expansions)
	fmt.Printf("📊 质量分数: %.2f\n", rewriteResult.QualityScore)
}

// ExampleAdvancedUsage 示例3：高级自定义使用
func ExampleAdvancedUsage() {
	fmt.Println("\n=== 示例3：高级自定义使用 ===")

	// 1. 使用配置构建器创建自定义配置
	config, err := NewConfigBuilder(ProviderDeepSeek).
		WithAPIKey("your-deepseek-api-key").
		WithModel("deepseek-chat").
		WithTimeout(60 * time.Second).
		WithMaxRetries(5).
		WithRateLimit(100).
		Build()
	if err != nil {
		log.Printf("构建配置失败: %v", err)
		return
	}

	fmt.Println("✅ 自定义配置创建成功")

	// 2. 设置全局配置
	SetGlobalConfig(ProviderDeepSeek, config)

	// 3. 创建上下文感知服务
	_ = NewContextAwareLLMService(&ContextAwareLLMConfig{
		PrimaryProvider:  ProviderDeepSeek,
		FallbackProvider: ProviderOpenAI,
		CacheEnabled:     true,
		CacheTTL:         30 * time.Minute,
		MaxRetries:       3,
		TimeoutSeconds:   30,
		EnableRouting:    true,
	})

	fmt.Println("✅ 上下文感知服务创建成功")

	// 4. 创建自定义Prompt模板
	promptManager := NewPromptManager(nil)
	customTemplate := &PromptTemplate{
		Name: "code_review",
		SystemPrompt: `你是一个专业的代码审查专家。请对提供的代码进行详细审查，包括：
1. 代码质量和可读性
2. 潜在的bug和安全问题
3. 性能优化建议
4. 最佳实践建议`,
		UserTemplate: `请审查以下{{.Language}}代码：

` + "```" + `{{.Language}}
{{.Code}}
` + "```" + `

项目背景：{{.ProjectContext}}
审查重点：{{.ReviewFocus}}`,
		Variables:    []string{"Language", "Code", "ProjectContext", "ReviewFocus"},
		OutputFormat: "markdown",
		Version:      "v1.0.0",
		Description:  "代码审查模板",
	}

	promptManager.RegisterTemplate(customTemplate)
	fmt.Println("✅ 自定义Prompt模板注册成功")

	// 5. 使用自定义模板
	prompt, err := promptManager.BuildPrompt("code_review", &PromptContext{
		Metadata: map[string]interface{}{
			"Language":       "Go",
			"Code":           "func main() {\n    fmt.Println(\"Hello World\")\n}",
			"ProjectContext": "微服务项目",
			"ReviewFocus":    "性能和安全",
		},
	})
	if err != nil {
		log.Printf("构建Prompt失败: %v", err)
		return
	}

	fmt.Printf("✅ 自定义Prompt构建成功\n")
	fmt.Printf("📋 系统提示词: %s\n", prompt.SystemPrompt[:100]+"...")
	fmt.Printf("📋 用户提示词: %s\n", prompt.Content[:100]+"...")
}

// ExampleErrorHandling 示例4：错误处理
func ExampleErrorHandling() {
	fmt.Println("\n=== 示例4：错误处理 ===")

	// 1. 创建配置（故意使用错误的API密钥）
	config := &LLMConfig{
		Provider:   ProviderDeepSeek,
		APIKey:     "invalid-key",
		MaxRetries: 2,
		Timeout:    10 * time.Second,
		RateLimit:  60,
	}

	SetGlobalConfig(ProviderDeepSeek, config)

	// 2. 尝试调用
	client, err := CreateGlobalClient(ProviderDeepSeek)
	if err != nil {
		log.Printf("创建客户端失败: %v", err)
		return
	}

	ctx := context.Background()
	response, err := client.Complete(ctx, &LLMRequest{
		Prompt:      "Hello",
		MaxTokens:   10,
		Temperature: 0.5,
	})

	// 3. 处理LLM特定错误
	if err != nil {
		if llmErr, ok := err.(*LLMError); ok {
			fmt.Printf("❌ LLM错误详情:\n")
			fmt.Printf("  - 提供商: %s\n", llmErr.Provider)
			fmt.Printf("  - 错误代码: %s\n", llmErr.Code)
			fmt.Printf("  - 错误信息: %s\n", llmErr.Message)
			fmt.Printf("  - 可重试: %t\n", llmErr.Retryable)

			if llmErr.Retryable {
				fmt.Println("💡 这是可重试的错误，可以稍后再试")
			} else {
				fmt.Println("💡 这是不可重试的错误（如API密钥错误）")
			}
		} else {
			fmt.Printf("❌ 其他错误: %v\n", err)
		}
		return
	}

	fmt.Printf("✅ 调用成功: %s\n", response.Content)
}

// RunAllExamples 运行所有示例
func RunAllExamples() {
	fmt.Println("🚀 ========== LLM模块使用示例 ==========")

	// 注意：这些示例需要有效的API密钥才能正常运行
	// 在实际使用时，请确保config/.env文件中设置了正确的API密钥

	ExampleSimpleUsage()
	ExampleConfigFileUsage()
	ExampleAdvancedUsage()
	ExampleErrorHandling()

	fmt.Println("\n🎉 ========== 所有示例运行完成 ==========")
}
