package llm

import (
	"encoding/json"
	"sync"
	"time"
)

// =============================================================================
// 缓存管理器
// =============================================================================

// CacheManager 缓存管理器
type CacheManager struct {
	cache map[string]*CacheItem
	mutex sync.RWMutex
}

// CacheItem 缓存项
type CacheItem struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	cm := &CacheManager{
		cache: make(map[string]*CacheItem),
	}

	// 启动清理协程
	go cm.startCleanup()

	return cm
}

// Set 设置缓存
func (cm *CacheManager) Set(key string, value interface{}, ttl time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache[key] = &CacheItem{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
}

// Get 获取缓存
func (cm *CacheManager) Get(key string) interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	item, exists := cm.cache[key]
	if !exists {
		return nil
	}

	if time.Now().After(item.ExpiresAt) {
		delete(cm.cache, key)
		return nil
	}

	return item.Data
}

// GetThreeElements 获取三要素缓存
func (cm *CacheManager) GetThreeElements(key string) *ThreeElementsModel {
	data := cm.Get(key)
	if data == nil {
		return nil
	}

	if threeElements, ok := data.(*ThreeElementsModel); ok {
		return threeElements
	}

	return nil
}

// SetThreeElements 设置三要素缓存
func (cm *CacheManager) SetThreeElements(key string, threeElements *ThreeElementsModel, ttl time.Duration) {
	cm.Set(key, threeElements, ttl)
}

// Delete 删除缓存
func (cm *CacheManager) Delete(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.cache, key)
}

// Clear 清空缓存
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache = make(map[string]*CacheItem)
}

// Size 获取缓存大小
func (cm *CacheManager) Size() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.cache)
}

// startCleanup 启动清理协程
func (cm *CacheManager) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanup()
	}
}

// cleanup 清理过期缓存
func (cm *CacheManager) cleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	for key, item := range cm.cache {
		if now.After(item.ExpiresAt) {
			delete(cm.cache, key)
		}
	}
}

// =============================================================================
// 工具函数
// =============================================================================

// parseJSONResponse 解析JSON响应
func parseJSONResponse(content string, target interface{}) error {
	// 尝试直接解析
	if err := json.Unmarshal([]byte(content), target); err == nil {
		return nil
	}

	// 尝试提取JSON部分（处理LLM可能返回的额外文本）
	jsonStart := -1
	jsonEnd := -1
	braceCount := 0

	for i, char := range content {
		if char == '{' {
			if jsonStart == -1 {
				jsonStart = i
			}
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 && jsonStart != -1 {
				jsonEnd = i + 1
				break
			}
		}
	}

	if jsonStart != -1 && jsonEnd != -1 {
		jsonContent := content[jsonStart:jsonEnd]
		return json.Unmarshal([]byte(jsonContent), target)
	}

	return json.Unmarshal([]byte(content), target)
}

// validateLLMConfig 验证LLM配置
func validateLLMConfig(config *LLMConfig) error {
	if config.Provider == "" {
		return &LLMError{
			Code:    "INVALID_CONFIG",
			Message: "Provider is required",
		}
	}

	if config.APIKey == "" {
		return &LLMError{
			Code:    "INVALID_CONFIG",
			Message: "API key is required",
		}
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	if config.RateLimit <= 0 {
		config.RateLimit = 60
	}

	return nil
}

// buildDefaultModel 构建默认模型名称
func buildDefaultModel(provider LLMProvider) string {
	switch provider {
	case ProviderOpenAI:
		return "gpt-3.5-turbo"
	case ProviderClaude:
		return "claude-3-sonnet-20240229"
	case ProviderQianwen:
		return "qwen-turbo"
	case ProviderDeepSeek:
		return "deepseek-chat"
	default:
		return ""
	}
}

// buildDefaultBaseURL 构建默认基础URL
func buildDefaultBaseURL(provider LLMProvider) string {
	switch provider {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderClaude:
		return "https://api.anthropic.com/v1"
	case ProviderQianwen:
		return "https://dashscope.aliyuncs.com/api/v1"
	case ProviderDeepSeek:
		return "https://api.deepseek.com/v1"
	default:
		return ""
	}
}

// =============================================================================
// 配置构建器
// =============================================================================

// ConfigBuilder 配置构建器
type ConfigBuilder struct {
	config *LLMConfig
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder(provider LLMProvider) *ConfigBuilder {
	return &ConfigBuilder{
		config: &LLMConfig{
			Provider:   provider,
			BaseURL:    buildDefaultBaseURL(provider),
			Model:      buildDefaultModel(provider),
			MaxRetries: 3,
			Timeout:    30 * time.Second,
			RateLimit:  60,
			Extra:      make(map[string]interface{}),
		},
	}
}

// WithAPIKey 设置API密钥
func (cb *ConfigBuilder) WithAPIKey(apiKey string) *ConfigBuilder {
	cb.config.APIKey = apiKey
	return cb
}

// WithBaseURL 设置基础URL
func (cb *ConfigBuilder) WithBaseURL(baseURL string) *ConfigBuilder {
	cb.config.BaseURL = baseURL
	return cb
}

// WithModel 设置模型
func (cb *ConfigBuilder) WithModel(model string) *ConfigBuilder {
	cb.config.Model = model
	return cb
}

// WithTimeout 设置超时
func (cb *ConfigBuilder) WithTimeout(timeout time.Duration) *ConfigBuilder {
	cb.config.Timeout = timeout
	return cb
}

// WithMaxRetries 设置最大重试次数
func (cb *ConfigBuilder) WithMaxRetries(maxRetries int) *ConfigBuilder {
	cb.config.MaxRetries = maxRetries
	return cb
}

// WithRateLimit 设置限流
func (cb *ConfigBuilder) WithRateLimit(rateLimit int) *ConfigBuilder {
	cb.config.RateLimit = rateLimit
	return cb
}

// WithExtra 设置额外配置
func (cb *ConfigBuilder) WithExtra(key string, value interface{}) *ConfigBuilder {
	cb.config.Extra[key] = value
	return cb
}

// Build 构建配置
func (cb *ConfigBuilder) Build() (*LLMConfig, error) {
	if err := validateLLMConfig(cb.config); err != nil {
		return nil, err
	}
	return cb.config, nil
}
