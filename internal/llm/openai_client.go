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
// OpenAI客户端实现
// =============================================================================

// OpenAIClient OpenAI适配器
type OpenAIClient struct {
	*BaseAdapter
	apiKey  string
	baseURL string
	model   string
}

// OpenAIRequest OpenAI请求格式
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// OpenAIMessage OpenAI消息格式
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse OpenAI响应格式
type OpenAIResponse struct {
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

// OpenAIErrorResponse OpenAI错误响应
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config *LLMConfig) (LLMClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	client := &OpenAIClient{
		BaseAdapter: NewBaseAdapter(ProviderOpenAI, config),
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
		CostPerToken:      0.002,
		LatencyMs:         1000,
		Models:            []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"},
	})

	return client, nil
}

// Complete 完成对话
func (oc *OpenAIClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// 1. 检查限流
	if err := oc.CheckRateLimit(ctx); err != nil {
		return nil, err
	}

	// 2. 检查熔断器
	if err := oc.CheckCircuitBreaker(); err != nil {
		return nil, err
	}

	// 3. 转换请求格式
	openaiReq := oc.convertToOpenAIFormat(req)

	// 4. 发送请求
	resp, err := oc.sendRequest(ctx, openaiReq)
	if err != nil {
		oc.RecordFailure()
		return nil, err
	}

	// 5. 转换响应格式
	oc.RecordSuccess()
	return oc.convertFromOpenAIFormat(resp, time.Since(startTime)), nil
}

// BatchComplete 批量完成（OpenAI不直接支持，串行处理）
func (oc *OpenAIClient) BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error) {
	responses := make([]*LLMResponse, len(reqs))

	for i, req := range reqs {
		resp, err := oc.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// StreamComplete 流式完成
func (oc *OpenAIClient) StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error) {
	// 简化实现，实际应该支持SSE流式响应
	ch := make(chan *LLMStreamResponse, 1)

	go func() {
		defer close(ch)

		resp, err := oc.Complete(ctx, req)
		if err != nil {
			ch <- &LLMStreamResponse{
				Error:    err,
				Provider: ProviderOpenAI,
			}
			return
		}

		ch <- &LLMStreamResponse{
			Content:  resp.Content,
			Done:     true,
			Provider: ProviderOpenAI,
		}
	}()

	return ch, nil
}

// HealthCheck 健康检查
func (oc *OpenAIClient) HealthCheck(ctx context.Context) error {
	req := &LLMRequest{
		Prompt:      "Hello",
		MaxTokens:   1,
		Temperature: 0,
	}

	_, err := oc.Complete(ctx, req)
	return err
}

// GetModel 获取模型名称
func (oc *OpenAIClient) GetModel() string {
	return oc.model
}

// convertToOpenAIFormat 转换为OpenAI格式
func (oc *OpenAIClient) convertToOpenAIFormat(req *LLMRequest) *OpenAIRequest {
	messages := []OpenAIMessage{}

	// 添加系统消息
	if req.SystemPrompt != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// 添加用户消息
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	model := req.Model
	if model == "" {
		model = oc.model
	}

	return &OpenAIRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}

// convertFromOpenAIFormat 转换OpenAI响应格式
func (oc *OpenAIClient) convertFromOpenAIFormat(resp *OpenAIResponse, duration time.Duration) *LLMResponse {
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return &LLMResponse{
		Content:    content,
		TokensUsed: resp.Usage.TotalTokens,
		Model:      resp.Model,
		Provider:   ProviderOpenAI,
		Duration:   duration,
		Metadata: map[string]interface{}{
			"id":            resp.ID,
			"finish_reason": resp.Choices[0].FinishReason,
		},
	}
}

// sendRequest 发送HTTP请求
func (oc *OpenAIClient) sendRequest(ctx context.Context, req *OpenAIRequest) (*OpenAIResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", oc.baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+oc.apiKey)

	// 发送请求
	httpResp, err := oc.httpClient.Do(httpReq)
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
		var errorResp OpenAIErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, &LLMError{
				Provider:  ProviderOpenAI,
				Code:      errorResp.Error.Code,
				Message:   errorResp.Error.Message,
				Retryable: httpResp.StatusCode >= 500,
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp OpenAIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &resp, nil
}
