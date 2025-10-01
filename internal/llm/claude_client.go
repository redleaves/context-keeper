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
// Claude客户端实现
// =============================================================================

// ClaudeClient Claude适配器
type ClaudeClient struct {
	*BaseAdapter
	apiKey  string
	baseURL string
	model   string
}

// ClaudeRequest Claude请求格式
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

// ClaudeMessage Claude消息格式
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse Claude响应格式
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeErrorResponse Claude错误响应
type ClaudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClaudeClient 创建Claude客户端
func NewClaudeClient(config *LLMConfig) (LLMClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	client := &ClaudeClient{
		BaseAdapter: NewBaseAdapter(ProviderClaude, config),
		apiKey:      config.APIKey,
		baseURL:     baseURL,
		model:       model,
	}

	// 设置能力
	client.SetCapabilities(&LLMCapabilities{
		MaxTokens:         4096,
		SupportedFormats:  []string{"text", "json"},
		SupportsStreaming: true,
		SupportsBatch:     false,
		CostPerToken:      0.003,
		LatencyMs:         1200,
		Models:            []string{"claude-3-sonnet-20240229", "claude-3-opus-20240229", "claude-3-haiku-20240307"},
	})

	return client, nil
}

// Complete 完成对话
func (cc *ClaudeClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// 1. 检查限流
	if err := cc.CheckRateLimit(ctx); err != nil {
		return nil, err
	}

	// 2. 检查熔断器
	if err := cc.CheckCircuitBreaker(); err != nil {
		return nil, err
	}

	// 3. 转换请求格式
	claudeReq := cc.convertToClaudeFormat(req)

	// 4. 发送请求
	resp, err := cc.sendRequest(ctx, claudeReq)
	if err != nil {
		cc.RecordFailure()
		return nil, err
	}

	// 5. 转换响应格式
	cc.RecordSuccess()
	return cc.convertFromClaudeFormat(resp, time.Since(startTime)), nil
}

// BatchComplete 批量完成
func (cc *ClaudeClient) BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error) {
	responses := make([]*LLMResponse, len(reqs))

	for i, req := range reqs {
		resp, err := cc.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// StreamComplete 流式完成
func (cc *ClaudeClient) StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error) {
	ch := make(chan *LLMStreamResponse, 1)

	go func() {
		defer close(ch)

		resp, err := cc.Complete(ctx, req)
		if err != nil {
			ch <- &LLMStreamResponse{
				Error:    err,
				Provider: ProviderClaude,
			}
			return
		}

		ch <- &LLMStreamResponse{
			Content:  resp.Content,
			Done:     true,
			Provider: ProviderClaude,
		}
	}()

	return ch, nil
}

// HealthCheck 健康检查
func (cc *ClaudeClient) HealthCheck(ctx context.Context) error {
	req := &LLMRequest{
		Prompt:      "Hello",
		MaxTokens:   1,
		Temperature: 0,
	}

	_, err := cc.Complete(ctx, req)
	return err
}

// GetModel 获取模型名称
func (cc *ClaudeClient) GetModel() string {
	return cc.model
}

// convertToClaudeFormat 转换为Claude格式
func (cc *ClaudeClient) convertToClaudeFormat(req *LLMRequest) *ClaudeRequest {
	messages := []ClaudeMessage{
		{
			Role:    "user",
			Content: req.Prompt,
		},
	}

	model := req.Model
	if model == "" {
		model = cc.model
	}

	claudeReq := &ClaudeRequest{
		Model:     model,
		MaxTokens: req.MaxTokens,
		Messages:  messages,
	}

	// Claude使用单独的system字段
	if req.SystemPrompt != "" {
		claudeReq.System = req.SystemPrompt
	}

	return claudeReq
}

// convertFromClaudeFormat 转换Claude响应格式
func (cc *ClaudeClient) convertFromClaudeFormat(resp *ClaudeResponse, duration time.Duration) *LLMResponse {
	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &LLMResponse{
		Content:    content,
		TokensUsed: resp.Usage.InputTokens + resp.Usage.OutputTokens,
		Model:      resp.Model,
		Provider:   ProviderClaude,
		Duration:   duration,
		Metadata: map[string]interface{}{
			"id":            resp.ID,
			"stop_reason":   resp.StopReason,
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	}
}

// sendRequest 发送HTTP请求
func (cc *ClaudeClient) sendRequest(ctx context.Context, req *ClaudeRequest) (*ClaudeResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", cc.baseURL+"/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", cc.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// 发送请求
	httpResp, err := cc.httpClient.Do(httpReq)
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
		var errorResp ClaudeErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, &LLMError{
				Provider:  ProviderClaude,
				Code:      errorResp.Error.Type,
				Message:   errorResp.Error.Message,
				Retryable: httpResp.StatusCode >= 500,
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp ClaudeResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &resp, nil
}
