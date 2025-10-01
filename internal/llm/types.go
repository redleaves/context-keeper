package llm

import (
	"context"
	"time"
)

// =============================================================================
// 核心类型定义
// =============================================================================

// LLMProvider LLM提供商类型
type LLMProvider string

const (
	ProviderOpenAI      LLMProvider = "openai"
	ProviderClaude      LLMProvider = "claude"
	ProviderQianwen     LLMProvider = "qianwen"
	ProviderDeepSeek    LLMProvider = "deepseek"
	ProviderOllamaLocal LLMProvider = "ollama_local"
)

// LLMRequest 统一的LLM请求结构
type LLMRequest struct {
	Prompt       string                 `json:"prompt"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTokens    int                    `json:"max_tokens"`
	Temperature  float64                `json:"temperature"`
	Format       string                 `json:"format,omitempty"` // "json", "text", "code"
	Model        string                 `json:"model,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LLMResponse 统一的LLM响应结构
type LLMResponse struct {
	Content    string                 `json:"content"`
	TokensUsed int                    `json:"tokens_used"`
	Model      string                 `json:"model"`
	Provider   LLMProvider            `json:"provider"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// LLMStreamResponse 流式响应结构
type LLMStreamResponse struct {
	Content  string      `json:"content"`
	Delta    string      `json:"delta"`
	Done     bool        `json:"done"`
	Provider LLMProvider `json:"provider"`
	Error    error       `json:"error,omitempty"`
}

// LLMCapabilities LLM能力描述
type LLMCapabilities struct {
	MaxTokens         int      `json:"max_tokens"`
	SupportedFormats  []string `json:"supported_formats"`
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsBatch     bool     `json:"supports_batch"`
	CostPerToken      float64  `json:"cost_per_token"`
	LatencyMs         int      `json:"latency_ms"`
	Models            []string `json:"models"`
}

// LLMConfig LLM配置
type LLMConfig struct {
	Provider   LLMProvider            `json:"provider"`
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url"`
	Model      string                 `json:"model"`
	MaxRetries int                    `json:"max_retries"`
	Timeout    time.Duration          `json:"timeout"`
	RateLimit  int                    `json:"rate_limit"` // requests per minute
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// LLMError LLM错误类型
type LLMError struct {
	Provider  LLMProvider `json:"provider"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Retryable bool        `json:"retryable"`
}

func (e *LLMError) Error() string {
	return e.Message
}

// RoutingRule 路由规则
type RoutingRule struct {
	TaskType          string                 `json:"task_type"`
	PreferredProvider LLMProvider            `json:"preferred_provider"`
	FallbackProviders []LLMProvider          `json:"fallback_providers"`
	Conditions        map[string]interface{} `json:"conditions"`
}

// LLMClient 核心LLM客户端接口 - 策略模式的Strategy接口
type LLMClient interface {
	// 单次完成
	Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// 批量完成
	BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error)

	// 流式完成
	StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 获取提供商信息
	GetProvider() LLMProvider

	// 获取模型名称
	GetModel() string

	// 获取模型能力
	GetCapabilities() *LLMCapabilities

	// 关闭客户端
	Close() error
}
