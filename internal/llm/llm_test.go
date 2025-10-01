package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// 测试用例
// =============================================================================

// TestLLMFactory 测试LLM工厂
func TestLLMFactory(t *testing.T) {
	factory := NewLLMFactory()

	// 测试注册提供商
	providers := factory.ListProviders()
	expectedProviders := []LLMProvider{ProviderOpenAI, ProviderClaude, ProviderQianwen, ProviderDeepSeek}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	// 测试配置设置
	config := &LLMConfig{
		Provider:   ProviderOpenAI,
		APIKey:     "test-key",
		BaseURL:    "https://api.openai.com/v1",
		Model:      "gpt-3.5-turbo",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		RateLimit:  60,
	}

	factory.SetConfig(ProviderOpenAI, config)

	configuredProviders := factory.ListConfiguredProviders()
	if len(configuredProviders) != 1 {
		t.Errorf("Expected 1 configured provider, got %d", len(configuredProviders))
	}

	if configuredProviders[0] != ProviderOpenAI {
		t.Errorf("Expected OpenAI provider, got %s", configuredProviders[0])
	}
}

// TestConfigBuilder 测试配置构建器
func TestConfigBuilder(t *testing.T) {
	config, err := NewConfigBuilder(ProviderOpenAI).
		WithAPIKey("test-key").
		WithModel("gpt-4").
		WithTimeout(60 * time.Second).
		WithMaxRetries(5).
		WithRateLimit(100).
		Build()

	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if config.Provider != ProviderOpenAI {
		t.Errorf("Expected OpenAI provider, got %s", config.Provider)
	}

	if config.APIKey != "test-key" {
		t.Errorf("Expected test-key, got %s", config.APIKey)
	}

	if config.Model != "gpt-4" {
		t.Errorf("Expected gpt-4, got %s", config.Model)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected 60s timeout, got %v", config.Timeout)
	}
}

// TestPromptManager 测试Prompt管理器
func TestPromptManager(t *testing.T) {
	pm := NewPromptManager(nil)

	// 测试默认模板
	templates := pm.ListTemplates()
	if len(templates) < 2 {
		t.Errorf("Expected at least 2 templates, got %d", len(templates))
	}

	// 测试获取模板
	template, exists := pm.GetTemplate("three_elements_analysis")
	if !exists {
		t.Error("Expected three_elements_analysis template to exist")
	}

	if template.Name != "three_elements_analysis" {
		t.Errorf("Expected template name three_elements_analysis, got %s", template.Name)
	}

	// 测试构建Prompt
	context := &PromptContext{
		SessionHistory: []Message{
			{Role: "user", Content: "我在用Go开发微服务"},
			{Role: "user", Content: "Redis缓存有点慢"},
		},
		WorkspaceContext: &WorkspaceContext{
			ProjectType: "microservices",
			TechStack:   []string{"Go", "Redis", "Docker"},
		},
		AnalysisType: "three_elements",
	}

	prompt, err := pm.BuildPrompt("three_elements_analysis", context)
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	if prompt.SystemPrompt == "" {
		t.Error("Expected non-empty system prompt")
	}

	if prompt.Content == "" {
		t.Error("Expected non-empty content")
	}

	if prompt.Format != "json" {
		t.Errorf("Expected json format, got %s", prompt.Format)
	}
}

// TestCacheManager 测试缓存管理器
func TestCacheManager(t *testing.T) {
	cm := NewCacheManager()

	// 测试设置和获取
	testData := &ThreeElementsModel{
		User: UserElement{
			TechStack:       []string{"Go", "Redis"},
			ExperienceLevel: "senior",
			Role:            "backend_engineer",
		},
	}

	key := "test-key"
	cm.SetThreeElements(key, testData, 1*time.Hour)

	retrieved := cm.GetThreeElements(key)
	if retrieved == nil {
		t.Error("Expected to retrieve cached data")
	}

	if len(retrieved.User.TechStack) != 2 {
		t.Errorf("Expected 2 tech stack items, got %d", len(retrieved.User.TechStack))
	}

	// 测试过期
	cm.Set("expire-test", "test-value", 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	expired := cm.Get("expire-test")
	if expired != nil {
		t.Error("Expected expired data to be nil")
	}

	// 测试大小
	size := cm.Size()
	if size != 1 { // 只有test-key还在
		t.Errorf("Expected cache size 1, got %d", size)
	}
}

// TestCircuitBreaker 测试熔断器
func TestCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:    3,
		ResetTimeout:   100 * time.Millisecond,
		FailureTimeout: 50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// 初始状态应该是关闭的
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be closed, got %v", cb.GetState())
	}

	// 应该允许请求
	if !cb.AllowRequest() {
		t.Error("Expected to allow request in closed state")
	}

	// 记录失败
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// 应该进入开启状态
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open after failures, got %v", cb.GetState())
	}

	// 应该拒绝请求
	if cb.AllowRequest() {
		t.Error("Expected to reject request in open state")
	}

	// 等待重置超时
	time.Sleep(150 * time.Millisecond)

	// 应该进入半开状态
	if !cb.AllowRequest() {
		t.Error("Expected to allow request after reset timeout")
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be half-open, got %v", cb.GetState())
	}

	// 记录成功
	cb.RecordSuccess()

	// 应该回到关闭状态
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be closed after success, got %v", cb.GetState())
	}
}

// TestParseJSONResponse 测试JSON解析
func TestParseJSONResponse(t *testing.T) {
	// 测试正常JSON
	jsonStr := `{"user": {"tech_stack": ["Go", "Redis"], "experience_level": "senior"}}`
	var result ThreeElementsModel

	err := parseJSONResponse(jsonStr, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result.User.TechStack) != 2 {
		t.Errorf("Expected 2 tech stack items, got %d", len(result.User.TechStack))
	}

	// 测试带额外文本的JSON
	jsonWithText := `这是分析结果：
	{"user": {"tech_stack": ["Go"], "experience_level": "mid"}}
	以上是我的分析。`

	var result2 ThreeElementsModel
	err = parseJSONResponse(jsonWithText, &result2)
	if err != nil {
		t.Fatalf("Failed to parse JSON with extra text: %v", err)
	}

	if len(result2.User.TechStack) != 1 {
		t.Errorf("Expected 1 tech stack item, got %d", len(result2.User.TechStack))
	}
}

// MockLLMClient 模拟LLM客户端用于测试
type MockLLMClient struct {
	provider   LLMProvider
	responses  map[string]string
	callCount  int
	shouldFail bool
}

func NewMockLLMClient(provider LLMProvider) *MockLLMClient {
	return &MockLLMClient{
		provider:   provider,
		responses:  make(map[string]string),
		callCount:  0,
		shouldFail: false,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	m.callCount++

	if m.shouldFail {
		return nil, &LLMError{
			Provider:  m.provider,
			Code:      "MOCK_ERROR",
			Message:   "Mock error for testing",
			Retryable: true,
		}
	}

	// 根据系统提示词判断返回不同的模拟响应
	var content string
	if strings.Contains(req.SystemPrompt, "查询优化专家") {
		// 查询改写响应
		content = `{
			"rewritten_query": "Go微服务Redis缓存性能优化方案",
			"expansions": ["缓存策略", "性能监控", "集群配置"],
			"quality_score": 0.85,
			"reasoning": ["添加了技术栈相关词汇", "增加了性能优化关键词"]
		}`
	} else {
		// 三要素分析响应
		content = `{
			"user": {
				"tech_stack": ["Go", "Redis"],
				"experience_level": "senior",
				"role": "backend_engineer",
				"domain": "microservices"
			},
			"situation": {
				"project_type": "microservices",
				"tech_environment": {"env": "production"},
				"business_context": "e_commerce"
			},
			"problem": {
				"explicit_problem": "性能优化",
				"implicit_needs": ["监控", "缓存"],
				"intent": "performance_optimization"
			}
		}`
	}

	return &LLMResponse{
		Content:    content,
		TokensUsed: 100,
		Model:      "mock-model",
		Provider:   m.provider,
		Duration:   100 * time.Millisecond,
	}, nil
}

func (m *MockLLMClient) BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error) {
	responses := make([]*LLMResponse, len(reqs))
	for i, req := range reqs {
		resp, err := m.Complete(ctx, req)
		if err != nil {
			return nil, err
		}
		responses[i] = resp
	}
	return responses, nil
}

func (m *MockLLMClient) StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error) {
	ch := make(chan *LLMStreamResponse, 1)
	go func() {
		defer close(ch)
		resp, err := m.Complete(ctx, req)
		if err != nil {
			ch <- &LLMStreamResponse{Error: err, Provider: m.provider}
			return
		}
		ch <- &LLMStreamResponse{Content: resp.Content, Done: true, Provider: m.provider}
	}()
	return ch, nil
}

func (m *MockLLMClient) HealthCheck(ctx context.Context) error {
	if m.shouldFail {
		return &LLMError{Provider: m.provider, Code: "HEALTH_CHECK_FAILED", Message: "Health check failed"}
	}
	return nil
}

func (m *MockLLMClient) GetProvider() LLMProvider {
	return m.provider
}

func (m *MockLLMClient) GetCapabilities() *LLMCapabilities {
	return &LLMCapabilities{
		MaxTokens:         4096,
		SupportedFormats:  []string{"text", "json"},
		SupportsStreaming: true,
		SupportsBatch:     false,
		CostPerToken:      0.001,
		LatencyMs:         100,
		Models:            []string{"mock-model"},
	}
}

func (m *MockLLMClient) Close() error {
	return nil
}

// TestContextAwareLLMService 测试上下文感知LLM服务
func TestContextAwareLLMService(t *testing.T) {
	// 创建模拟工厂
	factory := NewLLMFactory()

	// 注册模拟客户端
	factory.RegisterProvider(ProviderOpenAI, func(config *LLMConfig) (LLMClient, error) {
		return NewMockLLMClient(ProviderOpenAI), nil
	})

	// 设置配置
	config := &LLMConfig{
		Provider:   ProviderOpenAI,
		APIKey:     "test-key",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		RateLimit:  60,
	}
	factory.SetConfig(ProviderOpenAI, config)

	// 创建服务
	service := NewContextAwareLLMService(&ContextAwareLLMConfig{
		PrimaryProvider:  ProviderOpenAI,
		FallbackProvider: ProviderClaude,
		CacheEnabled:     true,
		CacheTTL:         30 * time.Minute,
		MaxRetries:       3,
		TimeoutSeconds:   30,
	})
	service.factory = factory

	// 测试三要素分析
	ctx := context.Background()
	sessionHistory := []Message{
		{Role: "user", Content: "我在用Go开发微服务", Timestamp: time.Now()},
		{Role: "user", Content: "Redis缓存性能有问题", Timestamp: time.Now()},
	}

	workspaceContext := &WorkspaceContext{
		ProjectType: "microservices",
		TechStack:   []string{"Go", "Redis", "Docker"},
		ProjectName: "ecommerce-platform",
		Environment: "production",
	}

	threeElements, err := service.AnalyzeThreeElementsWithLLM(ctx, sessionHistory, workspaceContext)
	if err != nil {
		t.Fatalf("Failed to analyze three elements: %v", err)
	}

	if threeElements == nil {
		t.Fatal("Expected three elements result, got nil")
	}

	if len(threeElements.User.TechStack) == 0 {
		t.Error("Expected non-empty tech stack")
	}

	if threeElements.User.ExperienceLevel == "" {
		t.Error("Expected non-empty experience level")
	}

	// 测试查询改写
	originalQuery := "如何优化Redis性能"
	rewriteResult, err := service.RewriteQueryWithLLM(ctx, originalQuery, threeElements, &RetrievalContext{
		Strategy:  "semantic",
		TopK:      10,
		Threshold: 0.7,
	})

	if err != nil {
		t.Fatalf("Failed to rewrite query: %v", err)
	}

	if rewriteResult.OriginalQuery != originalQuery {
		t.Errorf("Expected original query %s, got %s", originalQuery, rewriteResult.OriginalQuery)
	}

	if rewriteResult.RewrittenQuery == "" {
		t.Error("Expected non-empty rewritten query")
	}
}

// TestSimpleLLMClient 测试简单LLM客户端
func TestSimpleLLMClient(t *testing.T) {
	// 注册模拟客户端
	RegisterGlobalProvider(ProviderOpenAI, func(config *LLMConfig) (LLMClient, error) {
		return NewMockLLMClient(ProviderOpenAI), nil
	})

	// 创建简单客户端
	client := NewSimpleLLMClient(ProviderOpenAI, "test-key")

	// 测试对话
	ctx := context.Background()
	response, err := client.Chat(ctx, "Hello, how are you?")
	if err != nil {
		t.Fatalf("Failed to chat: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Chat response: %s", response)
}

// TestLLMClientCreation 测试LLM客户端创建
func TestLLMClientCreation(t *testing.T) {
	testCases := []struct {
		name     string
		provider LLMProvider
		config   *LLMConfig
		wantErr  bool
	}{
		{
			name:     "Valid OpenAI config",
			provider: ProviderOpenAI,
			config: &LLMConfig{
				Provider:   ProviderOpenAI,
				APIKey:     "test-key",
				BaseURL:    "https://api.openai.com/v1",
				Model:      "gpt-3.5-turbo",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				RateLimit:  60,
			},
			wantErr: false,
		},
		{
			name:     "Missing API key",
			provider: ProviderOpenAI,
			config: &LLMConfig{
				Provider: ProviderOpenAI,
				BaseURL:  "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name:     "Valid Claude config",
			provider: ProviderClaude,
			config: &LLMConfig{
				Provider:   ProviderClaude,
				APIKey:     "test-key",
				BaseURL:    "https://api.anthropic.com/v1",
				Model:      "claude-3-sonnet-20240229",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				RateLimit:  60,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var client LLMClient
			var err error

			switch tc.provider {
			case ProviderOpenAI:
				client, err = NewOpenAIClient(tc.config)
			case ProviderClaude:
				client, err = NewClaudeClient(tc.config)
			case ProviderQianwen:
				client, err = NewQianwenClient(tc.config)
			case ProviderDeepSeek:
				client, err = NewDeepSeekClient(tc.config)
			}

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				if client == nil {
					t.Error("Expected client, got nil")
				}

				if client != nil {
					if client.GetProvider() != tc.provider {
						t.Errorf("Expected provider %s, got %s", tc.provider, client.GetProvider())
					}

					capabilities := client.GetCapabilities()
					if capabilities == nil {
						t.Error("Expected capabilities, got nil")
					}

					// 清理
					client.Close()
				}
			}
		})
	}
}

// BenchmarkLLMFactory 基准测试LLM工厂
func BenchmarkLLMFactory(b *testing.B) {
	factory := NewLLMFactory()
	config := &LLMConfig{
		Provider:   ProviderOpenAI,
		APIKey:     "test-key",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		RateLimit:  60,
	}
	factory.SetConfig(ProviderOpenAI, config)

	// 注册模拟客户端
	factory.RegisterProvider(ProviderOpenAI, func(config *LLMConfig) (LLMClient, error) {
		return NewMockLLMClient(ProviderOpenAI), nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := factory.CreateClient(ProviderOpenAI)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}
		_ = client
	}
}

// BenchmarkPromptBuilding 基准测试Prompt构建
func BenchmarkPromptBuilding(b *testing.B) {
	pm := NewPromptManager(nil)
	context := &PromptContext{
		SessionHistory: []Message{
			{Role: "user", Content: "我在用Go开发微服务"},
			{Role: "user", Content: "Redis缓存有点慢"},
		},
		WorkspaceContext: &WorkspaceContext{
			ProjectType: "microservices",
			TechStack:   []string{"Go", "Redis", "Docker"},
		},
		AnalysisType: "three_elements",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		prompt, err := pm.BuildPrompt("three_elements_analysis", context)
		if err != nil {
			b.Fatalf("Failed to build prompt: %v", err)
		}
		_ = prompt
	}
}
