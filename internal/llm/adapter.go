package llm

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// =============================================================================
// 适配器模式 - 统一不同LLM API的差异
// =============================================================================

// BaseAdapter 基础适配器 - 适配器模式的Adapter
type BaseAdapter struct {
	provider       LLMProvider
	config         *LLMConfig
	httpClient     *http.Client
	rateLimiter    *rate.Limiter
	circuitBreaker *CircuitBreaker
	capabilities   *LLMCapabilities
	mutex          sync.RWMutex
}

// NewBaseAdapter 创建基础适配器
func NewBaseAdapter(provider LLMProvider, config *LLMConfig) *BaseAdapter {
	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        200, // 🔥 增加全局连接数
			MaxIdleConnsPerHost: 50,  // 🔥 大幅增加每个host的连接数，支持并发
			MaxConnsPerHost:     100, // 🔥 增加每个host的最大连接数
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false, // 🔥 启用keep-alive
			DisableCompression:  false,
		},
	}

	// 创建限流器 (requests per minute -> requests per second)
	rateLimit := rate.Limit(float64(config.RateLimit) / 60.0)
	rateLimiter := rate.NewLimiter(rateLimit, config.RateLimit)

	// 创建熔断器
	circuitBreaker := NewCircuitBreaker(&CircuitBreakerConfig{
		MaxFailures:    5,
		ResetTimeout:   30 * time.Second,
		FailureTimeout: 10 * time.Second,
	})

	return &BaseAdapter{
		provider:       provider,
		config:         config,
		httpClient:     httpClient,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
	}
}

// GetProvider 获取提供商
func (ba *BaseAdapter) GetProvider() LLMProvider {
	return ba.provider
}

// GetCapabilities 获取能力
func (ba *BaseAdapter) GetCapabilities() *LLMCapabilities {
	ba.mutex.RLock()
	defer ba.mutex.RUnlock()
	return ba.capabilities
}

// SetCapabilities 设置能力
func (ba *BaseAdapter) SetCapabilities(capabilities *LLMCapabilities) {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()
	ba.capabilities = capabilities
}

// CheckRateLimit 检查限流
func (ba *BaseAdapter) CheckRateLimit(ctx context.Context) error {
	if err := ba.rateLimiter.Wait(ctx); err != nil {
		return &LLMError{
			Provider:  ba.provider,
			Code:      "RATE_LIMIT_EXCEEDED",
			Message:   "请求频率超限",
			Retryable: true,
		}
	}
	return nil
}

// CheckCircuitBreaker 检查熔断器
func (ba *BaseAdapter) CheckCircuitBreaker() error {
	if !ba.circuitBreaker.AllowRequest() {
		return &LLMError{
			Provider:  ba.provider,
			Code:      "CIRCUIT_BREAKER_OPEN",
			Message:   "熔断器开启，拒绝请求",
			Retryable: true,
		}
	}
	return nil
}

// RecordSuccess 记录成功
func (ba *BaseAdapter) RecordSuccess() {
	ba.circuitBreaker.RecordSuccess()
}

// RecordFailure 记录失败
func (ba *BaseAdapter) RecordFailure() {
	ba.circuitBreaker.RecordFailure()
}

// Close 关闭适配器
func (ba *BaseAdapter) Close() error {
	ba.httpClient.CloseIdleConnections()
	return nil
}

// =============================================================================
// 熔断器实现
// =============================================================================

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	MaxFailures    int           `json:"max_failures"`
	ResetTimeout   time.Duration `json:"reset_timeout"`
	FailureTimeout time.Duration `json:"failure_timeout"`
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// AllowRequest 是否允许请求
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.Sub(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.config.MaxFailures {
		cb.state = StateOpen
	}
}

// GetState 获取状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetFailures 获取失败次数
func (cb *CircuitBreaker) GetFailures() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failures
}
