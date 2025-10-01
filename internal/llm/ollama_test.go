package llm

import (
	"context"
	"testing"
	"time"
)

// TestOllamaLocalClient 测试Ollama本地客户端
func TestOllamaLocalClient(t *testing.T) {
	// 创建配置
	config := &LLMConfig{
		Provider:   ProviderOllamaLocal,
		BaseURL:    "http://localhost:11434",
		Model:      "codeqwen:7b",
		MaxRetries: 3,
		Timeout:    60 * time.Second,
		RateLimit:  0,
	}

	// 创建客户端
	client, err := NewOllamaLocalClient(config)
	if err != nil {
		t.Fatalf("Failed to create Ollama client: %v", err)
	}
	defer client.Close()

	// 测试健康检查
	ctx := context.Background()
	if err := client.HealthCheck(ctx); err != nil {
		t.Skipf("Ollama service not available, skipping test: %v", err)
	}

	// 测试基本对话
	req := &LLMRequest{
		Prompt:      "Hello, write a simple Go function to add two numbers.",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Failed to complete request: %v", err)
	}

	// 验证响应
	if resp.Content == "" {
		t.Error("Response content is empty")
	}

	if resp.Provider != ProviderOllamaLocal {
		t.Errorf("Expected provider %s, got %s", ProviderOllamaLocal, resp.Provider)
	}

	if resp.Model != "codeqwen:7b" {
		t.Errorf("Expected model codeqwen:7b, got %s", resp.Model)
	}

	t.Logf("Response: %s", resp.Content)
	t.Logf("Tokens used: %d", resp.TokensUsed)
	t.Logf("Duration: %v", resp.Duration)
}

// TestOllamaModelSwitching 测试模型切换
func TestOllamaModelSwitching(t *testing.T) {
	testModels := []string{
		"codeqwen:7b",
		"deepseek-coder:6.7b-instruct",
	}

	for _, model := range testModels {
		t.Run(model, func(t *testing.T) {
			config := &LLMConfig{
				Provider:   ProviderOllamaLocal,
				BaseURL:    "http://localhost:11434",
				Model:      model,
				MaxRetries: 3,
				Timeout:    60 * time.Second,
				RateLimit:  0,
			}

			client, err := NewOllamaLocalClient(config)
			if err != nil {
				t.Fatalf("Failed to create client for %s: %v", model, err)
			}
			defer client.Close()

			// 测试健康检查
			ctx := context.Background()
			if err := client.HealthCheck(ctx); err != nil {
				t.Skipf("Ollama service not available for %s, skipping: %v", model, err)
			}

			// 验证模型名称
			if client.GetModel() != model {
				t.Errorf("Expected model %s, got %s", model, client.GetModel())
			}

			// 测试简单请求
			req := &LLMRequest{
				Prompt:      "What is 1+1?",
				MaxTokens:   50,
				Temperature: 0.1,
			}

			resp, err := client.Complete(ctx, req)
			if err != nil {
				t.Fatalf("Failed to complete request with %s: %v", model, err)
			}

			if resp.Content == "" {
				t.Errorf("Empty response from %s", model)
			}

			t.Logf("Model %s response: %s", model, resp.Content)
		})
	}
}

// TestOllamaFactory 测试工厂创建
func TestOllamaFactory(t *testing.T) {
	factory := NewLLMFactory()

	config := &LLMConfig{
		Provider:   ProviderOllamaLocal,
		BaseURL:    "http://localhost:11434",
		Model:      "codeqwen:7b",
		MaxRetries: 3,
		Timeout:    60 * time.Second,
		RateLimit:  0,
	}

	factory.SetConfig(ProviderOllamaLocal, config)

	client, err := factory.CreateClient(ProviderOllamaLocal)
	if err != nil {
		t.Fatalf("Failed to create client via factory: %v", err)
	}
	defer client.Close()

	// 验证客户端类型
	if client.GetProvider() != ProviderOllamaLocal {
		t.Errorf("Expected provider %s, got %s", ProviderOllamaLocal, client.GetProvider())
	}

	// 测试能力获取
	capabilities := client.GetCapabilities()
	if capabilities == nil {
		t.Error("Capabilities should not be nil")
	}

	if capabilities.CostPerToken != 0.0 {
		t.Errorf("Local model should have zero cost, got %f", capabilities.CostPerToken)
	}

	t.Logf("Capabilities: %+v", capabilities)
}
