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
// 千问客户端实现
// =============================================================================

// QianwenClient 千问适配器
type QianwenClient struct {
	*BaseAdapter
	apiKey  string
	baseURL string
	model   string
}

// QianwenRequest 千问请求格式
type QianwenRequest struct {
	Model      string            `json:"model"`
	Input      QianwenInput      `json:"input"`
	Parameters QianwenParameters `json:"parameters"`
}

// QianwenInput 千问输入格式
type QianwenInput struct {
	Messages []QianwenMessage `json:"messages"`
}

// QianwenMessage 千问消息格式
type QianwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QianwenParameters 千问参数
type QianwenParameters struct {
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// QianwenResponse 千问响应格式
type QianwenResponse struct {
	Output struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// QianwenErrorResponse 千问错误响应
type QianwenErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewQianwenClient 创建千问客户端
func NewQianwenClient(config *LLMConfig) (LLMClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Qianwen API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}

	model := config.Model
	if model == "" {
		model = "qwen-turbo"
	}

	client := &QianwenClient{
		BaseAdapter: NewBaseAdapter(ProviderQianwen, config),
		apiKey:      config.APIKey,
		baseURL:     baseURL,
		model:       model,
	}

	// 设置能力
	client.SetCapabilities(&LLMCapabilities{
		MaxTokens:         2048,
		SupportedFormats:  []string{"text", "json"},
		SupportsStreaming: true,
		SupportsBatch:     false,
		CostPerToken:      0.001,
		LatencyMs:         800,
		Models:            []string{"qwen-turbo", "qwen-plus", "qwen-max"},
	})

	return client, nil
}

// Complete 完成对话
func (qc *QianwenClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// 1. 检查限流
	if err := qc.CheckRateLimit(ctx); err != nil {
		return nil, err
	}

	// 2. 检查熔断器
	if err := qc.CheckCircuitBreaker(); err != nil {
		return nil, err
	}

	// 3. 转换请求格式
	qianwenReq := qc.convertToQianwenFormat(req)

	// 4. 发送请求
	resp, err := qc.sendRequest(ctx, qianwenReq)
	if err != nil {
		qc.RecordFailure()
		return nil, err
	}

	// 5. 转换响应格式
	qc.RecordSuccess()
	return qc.convertFromQianwenFormat(resp, time.Since(startTime)), nil
}

// BatchComplete 批量完成
func (qc *QianwenClient) BatchComplete(ctx context.Context, reqs []*LLMRequest) ([]*LLMResponse, error) {
	responses := make([]*LLMResponse, len(reqs))

	for i, req := range reqs {
		resp, err := qc.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// StreamComplete 流式完成
func (qc *QianwenClient) StreamComplete(ctx context.Context, req *LLMRequest) (<-chan *LLMStreamResponse, error) {
	ch := make(chan *LLMStreamResponse, 1)

	go func() {
		defer close(ch)

		resp, err := qc.Complete(ctx, req)
		if err != nil {
			ch <- &LLMStreamResponse{
				Error:    err,
				Provider: ProviderQianwen,
			}
			return
		}

		ch <- &LLMStreamResponse{
			Content:  resp.Content,
			Done:     true,
			Provider: ProviderQianwen,
		}
	}()

	return ch, nil
}

// HealthCheck 健康检查
func (qc *QianwenClient) HealthCheck(ctx context.Context) error {
	req := &LLMRequest{
		Prompt:      "Hello",
		MaxTokens:   1,
		Temperature: 0,
	}

	_, err := qc.Complete(ctx, req)
	return err
}

// GetModel 获取模型名称
func (qc *QianwenClient) GetModel() string {
	return qc.model
}

// convertToQianwenFormat 转换为千问格式
func (qc *QianwenClient) convertToQianwenFormat(req *LLMRequest) *QianwenRequest {
	messages := []QianwenMessage{}

	// 添加系统消息
	if req.SystemPrompt != "" {
		messages = append(messages, QianwenMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// 添加用户消息
	messages = append(messages, QianwenMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	model := req.Model
	if model == "" {
		model = qc.model
	}

	return &QianwenRequest{
		Model: model,
		Input: QianwenInput{
			Messages: messages,
		},
		Parameters: QianwenParameters{
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        0.8,
		},
	}
}

// convertFromQianwenFormat 转换千问响应格式
func (qc *QianwenClient) convertFromQianwenFormat(resp *QianwenResponse, duration time.Duration) *LLMResponse {
	return &LLMResponse{
		Content:    resp.Output.Text,
		TokensUsed: resp.Usage.TotalTokens,
		Model:      qc.model,
		Provider:   ProviderQianwen,
		Duration:   duration,
		Metadata: map[string]interface{}{
			"request_id":    resp.RequestID,
			"finish_reason": resp.Output.FinishReason,
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	}
}

// sendRequest 发送HTTP请求
func (qc *QianwenClient) sendRequest(ctx context.Context, req *QianwenRequest) (*QianwenResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", qc.baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+qc.apiKey)

	// 发送请求
	httpResp, err := qc.httpClient.Do(httpReq)
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
		var errorResp QianwenErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, &LLMError{
				Provider:  ProviderQianwen,
				Code:      errorResp.Code,
				Message:   errorResp.Message,
				Retryable: httpResp.StatusCode >= 500,
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp QianwenResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &resp, nil
}
