package llm

import (
	"context"
	"fmt"
	"time"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// 上下文感知的LLM服务
// =============================================================================

// ContextAwareLLMService 上下文感知的LLM服务
type ContextAwareLLMService struct {
	factory       *LLMFactory
	promptManager *PromptManager
	cacheManager  *CacheManager
	config        *ContextAwareLLMConfig
}

// ContextAwareLLMConfig 上下文感知LLM配置
type ContextAwareLLMConfig struct {
	PrimaryProvider  LLMProvider   `json:"primary_provider"`
	FallbackProvider LLMProvider   `json:"fallback_provider"`
	CacheEnabled     bool          `json:"cache_enabled"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	MaxRetries       int           `json:"max_retries"`
	TimeoutSeconds   int           `json:"timeout_seconds"`
	EnableRouting    bool          `json:"enable_routing"`
}

// LLMTask LLM任务
type LLMTask struct {
	Type           string                 `json:"type"`
	Prompt         *Prompt                `json:"prompt"`
	ExpectedFormat string                 `json:"expected_format"`
	MaxTokens      int                    `json:"max_tokens"`
	Temperature    float64                `json:"temperature"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewContextAwareLLMService 创建上下文感知LLM服务
func NewContextAwareLLMService(config *ContextAwareLLMConfig) *ContextAwareLLMService {
	if config == nil {
		config = &ContextAwareLLMConfig{
			PrimaryProvider:  ProviderOpenAI,
			FallbackProvider: ProviderClaude,
			CacheEnabled:     true,
			CacheTTL:         30 * time.Minute,
			MaxRetries:       3,
			TimeoutSeconds:   30,
			EnableRouting:    true,
		}
	}

	return &ContextAwareLLMService{
		factory:       GetGlobalFactory(),
		promptManager: NewPromptManager(nil),
		cacheManager:  NewCacheManager(),
		config:        config,
	}
}

// AnalyzeThreeElementsWithLLM 使用LLM分析三要素
func (cas *ContextAwareLLMService) AnalyzeThreeElementsWithLLM(
	ctx context.Context,
	sessionHistory []Message,
	workspaceContext *WorkspaceContext,
) (*ThreeElementsModel, error) {

	fmt.Printf("\n🔍 [三要素分析] 开始分析...\n")
	fmt.Printf("📝 输入参数:\n")
	fmt.Printf("  - 会话历史条数: %d\n", len(sessionHistory))
	for i, msg := range sessionHistory {
		fmt.Printf("    [%d] %s: %s\n", i+1, msg.Role, msg.Content)
	}
	if workspaceContext != nil {
		fmt.Printf("  - 工作空间: %s (%s)\n", workspaceContext.ProjectName, workspaceContext.ProjectType)
		fmt.Printf("  - 技术栈: %v\n", workspaceContext.TechStack)
		fmt.Printf("  - 环境: %s\n", workspaceContext.Environment)
	}

	// 1. 构建上下文感知的Prompt
	fmt.Printf("🛠️  [步骤1] 构建Prompt...\n")
	prompt, err := cas.promptManager.BuildPrompt("three_elements_analysis", &PromptContext{
		SessionHistory:   sessionHistory,
		WorkspaceContext: workspaceContext,
		AnalysisType:     "three_elements",
	})
	if err != nil {
		fmt.Printf("❌ 构建Prompt失败: %v\n", err)
		return nil, fmt.Errorf("构建Prompt失败: %w", err)
	}

	fmt.Printf("✅ Prompt构建成功\n")
	fmt.Printf("📋 系统提示词: %s\n", prompt.SystemPrompt[:min(200, len(prompt.SystemPrompt))]+"...")
	fmt.Printf("📋 用户提示词: %s\n", prompt.Content[:min(300, len(prompt.Content))]+"...")

	// 2. 检查缓存
	fmt.Printf("🗄️  [步骤2] 检查缓存...\n")
	if cas.config.CacheEnabled {
		if cached := cas.cacheManager.GetThreeElements(prompt.Hash()); cached != nil {
			fmt.Printf("✅ 缓存命中，直接返回结果\n")
			return cached, nil
		}
		fmt.Printf("⚪ 缓存未命中，继续LLM调用\n")
	} else {
		fmt.Printf("⚪ 缓存已禁用\n")
	}

	// 3. 调用LLM
	fmt.Printf("🤖 [步骤3] 调用LLM...\n")
	task := &LLMTask{
		Type:           "three_elements_analysis",
		Prompt:         prompt,
		ExpectedFormat: "json",
		MaxTokens:      500,
		Temperature:    0.1, // 低温度确保一致性
	}
	fmt.Printf("📊 任务参数: Type=%s, MaxTokens=%d, Temperature=%.1f\n",
		task.Type, task.MaxTokens, task.Temperature)

	result, err := cas.callLLMWithFallback(ctx, task)
	if err != nil {
		fmt.Printf("❌ LLM调用失败: %v\n", err)
		return nil, fmt.Errorf("LLM分析失败: %w", err)
	}

	fmt.Printf("✅ LLM调用成功\n")
	fmt.Printf("📈 响应统计: Provider=%s, TokensUsed=%d, Duration=%v\n",
		result.Provider, result.TokensUsed, result.Duration)
	fmt.Printf("📄 原始响应内容:\n%s\n", result.Content)

	// 4. 解析结果
	fmt.Printf("🔧 [步骤4] 解析JSON响应...\n")
	var threeElements ThreeElementsModel
	if err := parseJSONResponse(result.Content, &threeElements); err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		fmt.Printf("📄 原始内容: %s\n", result.Content)
		return nil, fmt.Errorf("解析LLM结果失败: %w, 原始内容: %s", err, result.Content)
	}

	fmt.Printf("✅ JSON解析成功\n")
	fmt.Printf("👤 用户要素: TechStack=%v, Level=%s, Role=%s, Domain=%s\n",
		threeElements.User.TechStack, threeElements.User.ExperienceLevel,
		threeElements.User.Role, threeElements.User.Domain)
	fmt.Printf("🏢 情景要素: ProjectType=%s, BusinessContext=%s\n",
		threeElements.Situation.ProjectType, threeElements.Situation.BusinessContext)
	fmt.Printf("❓ 问题要素: Intent=%s, ExplicitProblem=%s, ImplicitNeeds=%v\n",
		threeElements.Problem.Intent, threeElements.Problem.ExplicitProblem,
		threeElements.Problem.ImplicitNeeds)

	// 5. 缓存结果
	fmt.Printf("💾 [步骤5] 缓存结果...\n")
	if cas.config.CacheEnabled {
		cas.cacheManager.SetThreeElements(prompt.Hash(), &threeElements, cas.config.CacheTTL)
		fmt.Printf("✅ 结果已缓存，TTL=%v\n", cas.config.CacheTTL)
	} else {
		fmt.Printf("⚪ 缓存已禁用\n")
	}

	fmt.Printf("🎉 [三要素分析] 完成！\n\n")
	return &threeElements, nil
}

// RewriteQueryWithLLM 使用LLM进行查询改写
func (cas *ContextAwareLLMService) RewriteQueryWithLLM(
	ctx context.Context,
	originalQuery string,
	threeElements *ThreeElementsModel,
	retrievalContext *RetrievalContext,
) (*QueryRewriteResult, error) {

	fmt.Printf("\n✏️  [查询改写] 开始改写...\n")
	fmt.Printf("📝 输入参数:\n")
	fmt.Printf("  - 原始查询: %s\n", originalQuery)
	if threeElements != nil {
		fmt.Printf("  - 用户技术栈: %v\n", threeElements.User.TechStack)
		fmt.Printf("  - 用户角色: %s\n", threeElements.User.Role)
		fmt.Printf("  - 项目类型: %s\n", threeElements.Situation.ProjectType)
		fmt.Printf("  - 问题意图: %s\n", threeElements.Problem.Intent)
	}
	if retrievalContext != nil {
		fmt.Printf("  - 检索策略: %s\n", retrievalContext.Strategy)
		fmt.Printf("  - TopK: %d\n", retrievalContext.TopK)
		fmt.Printf("  - 阈值: %.2f\n", retrievalContext.Threshold)
	}

	// 1. 构建查询改写的Prompt
	fmt.Printf("🛠️  [步骤1] 构建查询改写Prompt...\n")
	prompt, err := cas.promptManager.BuildPrompt("query_rewrite", &PromptContext{
		OriginalQuery:    originalQuery,
		ThreeElements:    threeElements,
		RetrievalContext: retrievalContext,
		AnalysisType:     "query_rewrite",
	})
	if err != nil {
		fmt.Printf("❌ 构建Prompt失败: %v\n", err)
		return nil, fmt.Errorf("构建查询改写Prompt失败: %w", err)
	}

	fmt.Printf("✅ Prompt构建成功\n")
	fmt.Printf("📋 系统提示词: %s\n", prompt.SystemPrompt[:min(200, len(prompt.SystemPrompt))]+"...")
	fmt.Printf("📋 用户提示词: %s\n", prompt.Content[:min(300, len(prompt.Content))]+"...")

	// 2. 调用LLM
	fmt.Printf("🤖 [步骤2] 调用LLM进行查询改写...\n")
	task := &LLMTask{
		Type:           "query_rewrite",
		Prompt:         prompt,
		ExpectedFormat: "json",
		MaxTokens:      300,
		Temperature:    0.2, // 稍高温度增加创造性
	}
	fmt.Printf("📊 任务参数: Type=%s, MaxTokens=%d, Temperature=%.1f\n",
		task.Type, task.MaxTokens, task.Temperature)

	result, err := cas.callLLMWithFallback(ctx, task)
	if err != nil {
		fmt.Printf("❌ LLM调用失败: %v\n", err)
		return nil, fmt.Errorf("LLM查询改写失败: %w", err)
	}

	fmt.Printf("✅ LLM调用成功\n")
	fmt.Printf("📈 响应统计: Provider=%s, TokensUsed=%d, Duration=%v\n",
		result.Provider, result.TokensUsed, result.Duration)
	fmt.Printf("📄 原始响应内容:\n%s\n", result.Content)

	// 3. 解析和验证结果
	fmt.Printf("🔧 [步骤3] 解析JSON响应...\n")
	var rewriteResult QueryRewriteResult
	if err := parseJSONResponse(result.Content, &rewriteResult); err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		fmt.Printf("📄 原始内容: %s\n", result.Content)
		return nil, fmt.Errorf("解析查询改写结果失败: %w", err)
	}

	// 4. 设置原始查询
	rewriteResult.OriginalQuery = originalQuery

	fmt.Printf("✅ JSON解析成功\n")
	fmt.Printf("📝 改写结果:\n")
	fmt.Printf("  - 原始查询: %s\n", rewriteResult.OriginalQuery)
	fmt.Printf("  - 改写查询: %s\n", rewriteResult.RewrittenQuery)
	fmt.Printf("  - 扩展词汇: %v\n", rewriteResult.Expansions)
	fmt.Printf("  - 质量分数: %.2f\n", rewriteResult.QualityScore)
	fmt.Printf("  - 改写理由: %v\n", rewriteResult.Reasoning)

	fmt.Printf("🎉 [查询改写] 完成！\n\n")
	return &rewriteResult, nil
}

// QueryRewriteResult 查询改写结果
type QueryRewriteResult struct {
	OriginalQuery   string   `json:"original_query"`
	RewrittenQuery  string   `json:"rewritten_query"`
	Expansions      []string `json:"expansions"`
	QualityScore    float64  `json:"quality_score"`
	Reasoning       []string `json:"reasoning"`
	RewriteStrategy string   `json:"rewrite_strategy"`
}

// callLLMWithFallback 带降级的LLM调用
func (cas *ContextAwareLLMService) callLLMWithFallback(
	ctx context.Context,
	task *LLMTask,
) (*LLMResponse, error) {

	fmt.Printf("🔄 [LLM调用] 开始带降级的LLM调用...\n")

	// 1. 选择提供商
	fmt.Printf("🎯 [步骤1] 选择提供商...\n")
	provider := cas.selectProvider(task)
	fmt.Printf("✅ 选择的提供商: %s (任务类型: %s)\n", provider, task.Type)

	// 2. 尝试主要提供商
	fmt.Printf("🚀 [步骤2] 尝试主要提供商: %s\n", provider)
	if result, err := cas.callLLMWithRetry(ctx, provider, task); err == nil {
		fmt.Printf("✅ 主要提供商调用成功\n")
		return result, nil
	} else {
		fmt.Printf("❌ 主要提供商调用失败: %v\n", err)
	}

	// 3. 降级到备用提供商
	if cas.config.FallbackProvider != "" && cas.config.FallbackProvider != provider {
		fmt.Printf("🔄 [步骤3] 降级到备用提供商: %s\n", cas.config.FallbackProvider)
		if result, err := cas.callLLMWithRetry(ctx, cas.config.FallbackProvider, task); err == nil {
			fmt.Printf("✅ 备用提供商调用成功\n")
			return result, nil
		} else {
			fmt.Printf("❌ 备用提供商调用失败: %v\n", err)
		}
	} else {
		fmt.Printf("⚪ 无备用提供商或与主要提供商相同\n")
	}

	fmt.Printf("💥 所有LLM提供商都不可用\n")
	return nil, fmt.Errorf("所有LLM提供商都不可用")
}

// selectProvider 选择提供商
func (cas *ContextAwareLLMService) selectProvider(task *LLMTask) LLMProvider {
	if !cas.config.EnableRouting {
		return cas.config.PrimaryProvider
	}

	// 检查首选提供商是否可用
	checkProvider := func(provider LLMProvider) bool {
		_, err := cas.factory.CreateClient(provider)
		return err == nil
	}

	// 根据任务类型智能选择提供商
	var preferredProvider LLMProvider
	switch task.Type {
	case "three_elements_analysis":
		// Claude更适合分析任务，但如果不可用则使用DeepSeek
		if checkProvider(ProviderClaude) {
			preferredProvider = ProviderClaude
		} else if checkProvider(ProviderDeepSeek) {
			preferredProvider = ProviderDeepSeek
		} else {
			preferredProvider = cas.config.PrimaryProvider
		}
	case "query_rewrite":
		// 千问速度更快，但如果不可用则使用DeepSeek
		if checkProvider(ProviderQianwen) {
			preferredProvider = ProviderQianwen
		} else if checkProvider(ProviderDeepSeek) {
			preferredProvider = ProviderDeepSeek
		} else {
			preferredProvider = cas.config.PrimaryProvider
		}
	case "code_analysis":
		// DeepSeek更适合代码分析
		preferredProvider = ProviderDeepSeek
	default:
		preferredProvider = cas.config.PrimaryProvider
	}

	// 确保选择的提供商可用
	if checkProvider(preferredProvider) {
		return preferredProvider
	}

	// 如果首选不可用，返回主要提供商
	return cas.config.PrimaryProvider
}

// callLLMWithRetry 带重试的LLM调用
func (cas *ContextAwareLLMService) callLLMWithRetry(
	ctx context.Context,
	provider LLMProvider,
	task *LLMTask,
) (*LLMResponse, error) {

	fmt.Printf("🔁 [重试调用] 开始重试调用 %s...\n", provider)

	client, err := cas.factory.CreateClient(provider)
	if err != nil {
		fmt.Printf("❌ 创建客户端失败: %v\n", err)
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	fmt.Printf("✅ 客户端创建成功: %s\n", provider)

	var lastErr error

	for attempt := 0; attempt < cas.config.MaxRetries; attempt++ {
		fmt.Printf("🎯 [尝试 %d/%d] 调用 %s...\n", attempt+1, cas.config.MaxRetries, provider)

		// 设置超时
		timeoutCtx, cancel := context.WithTimeout(ctx,
			time.Duration(cas.config.TimeoutSeconds)*time.Second)

		// 构建请求
		req := &LLMRequest{
			Prompt:       task.Prompt.Content,
			SystemPrompt: task.Prompt.SystemPrompt,
			MaxTokens:    task.MaxTokens,
			Temperature:  task.Temperature,
			Format:       task.ExpectedFormat,
		}

		fmt.Printf("📤 发送请求: MaxTokens=%d, Temperature=%.1f, Timeout=%ds\n",
			req.MaxTokens, req.Temperature, cas.config.TimeoutSeconds)

		// 调用LLM
		startTime := time.Now()
		result, err := client.Complete(timeoutCtx, req)
		duration := time.Since(startTime)
		cancel()

		if err == nil {
			fmt.Printf("✅ 调用成功! 耗时: %v\n", duration)
			return result, nil
		}

		fmt.Printf("❌ 调用失败: %v (耗时: %v)\n", err, duration)
		lastErr = err

		// 指数退避
		if attempt < cas.config.MaxRetries-1 {
			backoff := time.Duration(1<<attempt) * time.Second
			fmt.Printf("⏳ 等待 %v 后重试...\n", backoff)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				fmt.Printf("❌ 上下文取消\n")
				return nil, ctx.Err()
			}
		}
	}

	fmt.Printf("💥 重试%d次后仍然失败: %v\n", cas.config.MaxRetries, lastErr)
	return nil, fmt.Errorf("重试%d次后仍然失败: %w", cas.config.MaxRetries, lastErr)
}

// SetConfig 设置配置
func (cas *ContextAwareLLMService) SetConfig(config *ContextAwareLLMConfig) {
	cas.config = config
}

// GetConfig 获取配置
func (cas *ContextAwareLLMService) GetConfig() *ContextAwareLLMConfig {
	return cas.config
}

// Close 关闭服务
func (cas *ContextAwareLLMService) Close() error {
	return cas.factory.Close()
}

// =============================================================================
// 简单的LLM客户端接口 - 供外部调用
// =============================================================================

// SimpleLLMClient 简单的LLM客户端接口
type SimpleLLMClient struct {
	service *ContextAwareLLMService
}

// NewSimpleLLMClient 创建简单LLM客户端
func NewSimpleLLMClient(provider LLMProvider, apiKey string) *SimpleLLMClient {
	// 设置配置
	config := &LLMConfig{
		Provider:   provider,
		APIKey:     apiKey,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		RateLimit:  60,
	}

	SetGlobalConfig(provider, config)

	// 创建服务
	service := NewContextAwareLLMService(&ContextAwareLLMConfig{
		PrimaryProvider: provider,
	})

	return &SimpleLLMClient{
		service: service,
	}
}

// Chat 简单对话接口
func (slc *SimpleLLMClient) Chat(ctx context.Context, message string) (string, error) {
	client, err := slc.service.factory.CreateClient(slc.service.config.PrimaryProvider)
	if err != nil {
		return "", err
	}

	req := &LLMRequest{
		Prompt:      message,
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
