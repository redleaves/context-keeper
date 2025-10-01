package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// =============================================================================
// DeepSeek客户端实现
// =============================================================================

// DeepSeekClient DeepSeek适配器
type DeepSeekClient struct {
	*BaseAdapter
	apiKey  string
	baseURL string
	model   string
}

// DeepSeekRequest DeepSeek请求格式（类似OpenAI）
type DeepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

// DeepSeekMessage DeepSeek消息格式
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse DeepSeek响应格式
type DeepSeekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// DeepSeekErrorResponse DeepSeek错误响应
type DeepSeekErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewDeepSeekClient 创建DeepSeek客户端
func NewDeepSeekClient(config *LLMConfig) (LLMClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "deepseek-chat"
	}

	client := &DeepSeekClient{
		BaseAdapter: NewBaseAdapter(ProviderDeepSeek, config),
		apiKey:      config.APIKey,
		baseURL:     baseURL,
		model:       model,
	}

	// 设置能力
	client.SetCapabilities(&LLMCapabilities{
		MaxTokens:         4096,
		SupportedFormats:  []string{"text", "json", "code"},
		SupportsStreaming: true,
		SupportsBatch:     false,
		CostPerToken:      0.0014,
		LatencyMs:         900,
		Models:            []string{"deepseek-chat", "deepseek-coder"},
	})

	return client, nil
}

// Complete 完成对话
func (dc *DeepSeekClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	fmt.Printf("🤖 [DeepSeek] 开始处理请求...\n")
	fmt.Printf("📝 请求参数: MaxTokens=%d, Temperature=%.1f, Format=%s\n",
		req.MaxTokens, req.Temperature, req.Format)
	fmt.Printf("📋 系统提示词长度: %d 字符\n", len(req.SystemPrompt))
	fmt.Printf("📋 用户提示词长度: %d 字符\n", len(req.Prompt))

	// 1. 检查限流（支持并行调用跳过）
	fmt.Printf("🚦 [步骤1] 检查限流...\n")

	// 🔥 检查是否跳过限流（并行调用场景）
	skipRateLimit := false
	if req.Metadata != nil {
		if skip, exists := req.Metadata["skip_rate_limit"]; exists {
			if skipBool, ok := skip.(bool); ok && skipBool {
				skipRateLimit = true
				fmt.Printf("⚡ [并行优化] 跳过限流检查（并行调用模式）\n")
			}
		}
	}

	if !skipRateLimit {
		if err := dc.CheckRateLimit(ctx); err != nil {
			fmt.Printf("❌ 限流检查失败: %v\n", err)
			return nil, err
		}
		fmt.Printf("✅ 限流检查通过\n")
	} else {
		fmt.Printf("✅ 限流检查跳过（并行模式）\n")
	}

	// 2. 检查熔断器
	fmt.Printf("🔌 [步骤2] 检查熔断器...\n")
	if err := dc.CheckCircuitBreaker(); err != nil {
		fmt.Printf("❌ 熔断器检查失败: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ 熔断器检查通过\n")

	// 3. 转换请求格式
	fmt.Printf("🔄 [步骤3] 转换请求格式...\n")
	deepseekReq := dc.convertToDeepSeekFormat(req)
	fmt.Printf("✅ 请求格式转换完成: Model=%s, Messages=%d条\n",
		deepseekReq.Model, len(deepseekReq.Messages))

	// 4. 发送请求
	fmt.Printf("📡 [步骤4] 发送HTTP请求到DeepSeek API...\n")
	resp, err := dc.sendRequest(ctx, deepseekReq)
	if err != nil {
		fmt.Printf("❌ HTTP请求失败: %v\n", err)
		dc.RecordFailure()
		return nil, err
	}
	fmt.Printf("✅ HTTP请求成功\n")

	// 5. 转换响应格式
	fmt.Printf("🔄 [步骤5] 转换响应格式...\n")
	dc.RecordSuccess()
	result := dc.convertFromDeepSeekFormat(resp, time.Since(startTime))

	fmt.Printf("✅ DeepSeek处理完成\n")
	fmt.Printf("📊 响应统计: TokensUsed=%d, Duration=%v\n",
		result.TokensUsed, result.Duration)
	fmt.Printf("📄 响应内容长度: %d 字符\n", len(result.Content))

	return result, nil
}

// BatchComplete 批量完成
func (dc *DeepSeekClient) BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error) {
	responses := make([]*LLMResponse, len(reqs))

	for i, req := range reqs {
		resp, err := dc.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// StreamComplete 流式完成
func (dc *DeepSeekClient) StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error) {
	ch := make(chan *LLMStreamResponse, 1)

	go func() {
		defer close(ch)

		resp, err := dc.Complete(ctx, req)
		if err != nil {
			ch <- &LLMStreamResponse{
				Error:    err,
				Provider: ProviderDeepSeek,
			}
			return
		}

		ch <- &LLMStreamResponse{
			Content:  resp.Content,
			Done:     true,
			Provider: ProviderDeepSeek,
		}
	}()

	return ch, nil
}

// HealthCheck 健康检查
func (dc *DeepSeekClient) HealthCheck(ctx context.Context) error {
	req := &LLMRequest{
		Prompt:      "Hello",
		MaxTokens:   1,
		Temperature: 0,
	}

	_, err := dc.Complete(ctx, req)
	return err
}

// GetModel 获取模型名称
func (dc *DeepSeekClient) GetModel() string {
	return dc.model
}

// convertToDeepSeekFormat 转换为DeepSeek格式
func (dc *DeepSeekClient) convertToDeepSeekFormat(req *LLMRequest) *DeepSeekRequest {
	messages := []DeepSeekMessage{}

	// 添加系统消息
	if req.SystemPrompt != "" {
		messages = append(messages, DeepSeekMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// 添加用户消息
	messages = append(messages, DeepSeekMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	model := req.Model
	if model == "" {
		model = dc.model
	}

	return &DeepSeekRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}

// convertFromDeepSeekFormat 转换DeepSeek响应格式
func (dc *DeepSeekClient) convertFromDeepSeekFormat(resp *DeepSeekResponse, duration time.Duration) *LLMResponse {
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return &LLMResponse{
		Content:    content,
		TokensUsed: resp.Usage.TotalTokens,
		Model:      resp.Model,
		Provider:   ProviderDeepSeek,
		Duration:   duration,
		Metadata: map[string]interface{}{
			"id":                resp.ID,
			"finish_reason":     resp.Choices[0].FinishReason,
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
		},
	}
}

// sendRequest 发送HTTP请求
func (dc *DeepSeekClient) sendRequest(ctx context.Context, req *DeepSeekRequest) (*DeepSeekResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", dc.baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+dc.apiKey)

	// 发送请求
	httpResp, err := dc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		var errorResp DeepSeekErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, &LLMError{
				Provider:  ProviderDeepSeek,
				Code:      errorResp.Error.Code,
				Message:   errorResp.Error.Message,
				Retryable: httpResp.StatusCode >= 500,
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp DeepSeekResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &resp, nil
}
