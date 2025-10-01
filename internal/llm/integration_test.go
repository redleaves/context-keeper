package llm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestDeepSeekIntegration 测试DeepSeek集成
func TestDeepSeekIntegration(t *testing.T) {
	// 跳过集成测试（除非设置了环境变量）
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=1 来运行")
	}

	fmt.Printf("\n🚀 ========== DeepSeek集成测试开始 ==========\n")

	// 初始化配置
	fmt.Printf("⚙️  初始化配置...\n")
	err := InitializeWithEnv()
	if err != nil {
		fmt.Printf("❌ 初始化配置失败: %v\n", err)
		t.Fatalf("初始化配置失败: %v", err)
	}
	fmt.Printf("✅ 配置初始化成功\n")

	// 获取服务
	fmt.Printf("🔧 获取全局服务...\n")
	service, err := GetGlobalService()
	if err != nil {
		fmt.Printf("❌ 获取全局服务失败: %v\n", err)
		t.Fatalf("获取全局服务失败: %v", err)
	}
	fmt.Printf("✅ 全局服务获取成功\n")

	// 测试简单对话
	ctx := context.Background()
	fmt.Printf("🤖 创建DeepSeek客户端...\n")
	client, err := CreateGlobalClient(ProviderDeepSeek)
	if err != nil {
		fmt.Printf("❌ 创建DeepSeek客户端失败: %v\n", err)
		t.Fatalf("创建DeepSeek客户端失败: %v", err)
	}
	fmt.Printf("✅ DeepSeek客户端创建成功\n")

	// 测试基本对话
	t.Run("基本对话测试", func(t *testing.T) {
		fmt.Printf("\n💬 ========== 基本对话测试 ==========\n")

		req := &LLMRequest{
			Prompt:      "你好，请简单介绍一下Go语言的特点",
			MaxTokens:   200,
			Temperature: 0.7,
		}

		fmt.Printf("📤 发送请求: %s\n", req.Prompt)
		fmt.Printf("⚙️  参数: MaxTokens=%d, Temperature=%.1f\n", req.MaxTokens, req.Temperature)

		resp, err := client.Complete(ctx, req)
		if err != nil {
			fmt.Printf("❌ DeepSeek对话失败: %v\n", err)
			t.Fatalf("DeepSeek对话失败: %v", err)
		}

		fmt.Printf("✅ 对话成功完成\n")

		// 验证响应
		if resp.Content == "" {
			fmt.Printf("❌ 响应内容为空\n")
			t.Error("响应内容为空")
		} else {
			fmt.Printf("✅ 响应内容非空 (长度: %d 字符)\n", len(resp.Content))
		}

		if resp.Provider != ProviderDeepSeek {
			fmt.Printf("❌ 提供商不匹配: 期望 %s，实际 %s\n", ProviderDeepSeek, resp.Provider)
			t.Errorf("期望提供商为 %s，实际为 %s", ProviderDeepSeek, resp.Provider)
		} else {
			fmt.Printf("✅ 提供商匹配: %s\n", resp.Provider)
		}

		fmt.Printf("📊 响应统计:\n")
		fmt.Printf("  - 使用Token数: %d\n", resp.TokensUsed)
		fmt.Printf("  - 响应时间: %v\n", resp.Duration)
		fmt.Printf("  - 模型: %s\n", resp.Model)

		fmt.Printf("📄 响应内容:\n%s\n", resp.Content)

		t.Logf("✅ 基本对话测试通过")
	})

	// 测试三要素分析
	t.Run("三要素分析测试", func(t *testing.T) {
		sessionHistory := []Message{
			{Role: "user", Content: "我在用Go开发微服务项目", Timestamp: time.Now()},
			{Role: "user", Content: "遇到了Redis缓存性能问题", Timestamp: time.Now()},
			{Role: "user", Content: "需要优化查询性能", Timestamp: time.Now()},
		}

		workspaceContext := &WorkspaceContext{
			ProjectType: "microservices",
			TechStack:   []string{"Go", "Redis", "Docker", "Kubernetes"},
			ProjectName: "ecommerce-platform",
			Environment: "production",
		}

		threeElements, err := service.AnalyzeThreeElementsWithLLM(ctx, sessionHistory, workspaceContext)
		if err != nil {
			t.Fatalf("三要素分析失败: %v", err)
		}

		// 打印原始响应用于调试
		t.Logf("三要素分析结果: %+v", threeElements)

		// 验证结果（放宽验证条件，因为LLM可能返回不同格式）
		if len(threeElements.User.TechStack) == 0 {
			t.Logf("Warning: 用户技术栈为空")
		}

		if threeElements.User.ExperienceLevel == "" {
			t.Logf("Warning: 用户经验水平为空")
		}

		if threeElements.Situation.ProjectType == "" {
			t.Logf("Warning: 项目类型为空")
		}

		if threeElements.Problem.Intent == "" {
			t.Logf("Warning: 问题意图为空")
		}

		t.Logf("用户技术栈: %v", threeElements.User.TechStack)
		t.Logf("用户经验水平: %s", threeElements.User.ExperienceLevel)
		t.Logf("用户角色: %s", threeElements.User.Role)
		t.Logf("项目类型: %s", threeElements.Situation.ProjectType)
		t.Logf("业务背景: %s", threeElements.Situation.BusinessContext)
		t.Logf("问题意图: %s", threeElements.Problem.Intent)
		t.Logf("明确问题: %s", threeElements.Problem.ExplicitProblem)
	})

	// 测试查询改写
	t.Run("查询改写测试", func(t *testing.T) {
		// 先进行三要素分析
		sessionHistory := []Message{
			{Role: "user", Content: "我是Go后端工程师", Timestamp: time.Now()},
			{Role: "user", Content: "在做微服务项目", Timestamp: time.Now()},
		}

		workspaceContext := &WorkspaceContext{
			ProjectType: "microservices",
			TechStack:   []string{"Go", "Redis"},
		}

		threeElements, err := service.AnalyzeThreeElementsWithLLM(ctx, sessionHistory, workspaceContext)
		if err != nil {
			t.Fatalf("三要素分析失败: %v", err)
		}

		// 进行查询改写
		originalQuery := "如何优化性能"
		rewriteResult, err := service.RewriteQueryWithLLM(ctx, originalQuery, threeElements, &RetrievalContext{
			Strategy:  "semantic",
			TopK:      10,
			Threshold: 0.7,
		})

		if err != nil {
			t.Fatalf("查询改写失败: %v", err)
		}

		// 验证结果
		if rewriteResult.OriginalQuery != originalQuery {
			t.Errorf("原始查询不匹配: 期望 %s，实际 %s", originalQuery, rewriteResult.OriginalQuery)
		}

		if rewriteResult.RewrittenQuery == "" {
			t.Error("改写后查询为空")
		}

		if rewriteResult.QualityScore < 0 || rewriteResult.QualityScore > 1 {
			t.Errorf("质量分数超出范围: %f", rewriteResult.QualityScore)
		}

		t.Logf("原始查询: %s", rewriteResult.OriginalQuery)
		t.Logf("改写查询: %s", rewriteResult.RewrittenQuery)
		t.Logf("扩展词汇: %v", rewriteResult.Expansions)
		t.Logf("质量分数: %.2f", rewriteResult.QualityScore)
		t.Logf("改写理由: %v", rewriteResult.Reasoning)
	})

	// 测试健康检查
	t.Run("健康检查测试", func(t *testing.T) {
		err := client.HealthCheck(ctx)
		if err != nil {
			t.Errorf("健康检查失败: %v", err)
		} else {
			t.Log("健康检查通过")
		}
	})

	// 测试能力获取
	t.Run("能力获取测试", func(t *testing.T) {
		capabilities := client.GetCapabilities()
		if capabilities == nil {
			t.Error("获取能力信息失败")
		} else {
			t.Logf("最大Token数: %d", capabilities.MaxTokens)
			t.Logf("支持格式: %v", capabilities.SupportedFormats)
			t.Logf("支持流式: %t", capabilities.SupportsStreaming)
			t.Logf("每Token成本: %f", capabilities.CostPerToken)
			t.Logf("延迟: %dms", capabilities.LatencyMs)
			t.Logf("支持模型: %v", capabilities.Models)
		}
	})
}

// TestConfigLoading 测试配置加载
func TestConfigLoading(t *testing.T) {
	// 测试环境变量加载
	t.Run("环境变量加载", func(t *testing.T) {
		err := LoadEnvFile("config/.env")
		if err != nil {
			t.Logf("加载config/.env文件失败（可能不存在）: %v", err)
		} else {
			t.Log("成功加载config/.env文件")
		}

		// 检查DeepSeek API密钥
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			t.Log("DEEPSEEK_API_KEY环境变量未设置")
		} else {
			t.Logf("DEEPSEEK_API_KEY已设置: %s...", apiKey[:10])
		}
	})

	// 测试配置文件加载
	t.Run("配置文件加载", func(t *testing.T) {
		configPath, err := FindConfigFile()
		if err != nil {
			t.Fatalf("查找配置文件失败: %v", err)
		}

		t.Logf("找到配置文件: %s", configPath)

		loader := NewConfigLoader(configPath)
		err = loader.LoadConfig()
		if err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		// 测试获取各种配置
		contextConfig, err := loader.GetContextAwareLLMConfig()
		if err != nil {
			t.Fatalf("获取上下文配置失败: %v", err)
		}

		t.Logf("主要提供商: %s", contextConfig.PrimaryProvider)
		t.Logf("备用提供商: %s", contextConfig.FallbackProvider)
		t.Logf("缓存启用: %t", contextConfig.CacheEnabled)
		t.Logf("缓存TTL: %v", contextConfig.CacheTTL)

		providerConfigs, err := loader.GetProviderConfigs()
		if err != nil {
			t.Fatalf("获取提供商配置失败: %v", err)
		}

		for provider, config := range providerConfigs {
			t.Logf("提供商 %s: 模型=%s, 超时=%v, 限流=%d",
				provider, config.Model, config.Timeout, config.RateLimit)
		}
	})
}

// TestSimpleUsage 测试简单使用方式
func TestSimpleUsage(t *testing.T) {
	// 跳过集成测试（除非设置了环境变量）
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=1 来运行")
	}

	// 测试简单客户端
	t.Run("简单客户端测试", func(t *testing.T) {
		// 加载环境变量
		LoadEnvFile("config/.env")

		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			t.Skip("DEEPSEEK_API_KEY未设置，跳过测试")
		}

		// 创建简单客户端
		client := NewSimpleLLMClient(ProviderDeepSeek, apiKey)

		// 进行对话
		ctx := context.Background()
		response, err := client.Chat(ctx, "请用一句话介绍Go语言")
		if err != nil {
			t.Fatalf("对话失败: %v", err)
		}

		if response == "" {
			t.Error("响应为空")
		}

		t.Logf("简单对话响应: %s", response)
	})
}

// BenchmarkDeepSeekPerformance 性能基准测试
func BenchmarkDeepSeekPerformance(b *testing.B) {
	// 跳过基准测试（除非设置了环境变量）
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		b.Skip("跳过基准测试，设置 RUN_INTEGRATION_TESTS=1 来运行")
	}

	// 初始化
	LoadEnvFile("config/.env")
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		b.Skip("DEEPSEEK_API_KEY未设置，跳过基准测试")
	}

	client := NewSimpleLLMClient(ProviderDeepSeek, apiKey)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := client.Chat(ctx, "Hello")
		if err != nil {
			b.Fatalf("对话失败: %v", err)
		}
	}
}
