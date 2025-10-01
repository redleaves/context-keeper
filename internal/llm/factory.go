package llm

import (
	"fmt"
	"sync"
)

// =============================================================================
// 工厂模式实现 - 创建不同的LLM客户端
// =============================================================================

// LLMFactory LLM客户端工厂
type LLMFactory struct {
	configs  map[LLMProvider]*LLMConfig
	cache    map[LLMProvider]LLMClient
	creators map[LLMProvider]ClientCreator
	mutex    sync.RWMutex
}

// ClientCreator 客户端创建函数类型
type ClientCreator func(config *LLMConfig) (LLMClient, error)

// NewLLMFactory 创建LLM工厂
func NewLLMFactory() *LLMFactory {
	factory := &LLMFactory{
		configs:  make(map[LLMProvider]*LLMConfig),
		cache:    make(map[LLMProvider]LLMClient),
		creators: make(map[LLMProvider]ClientCreator),
	}

	// 注册默认的客户端创建器
	factory.registerDefaultCreators()

	return factory
}

// registerDefaultCreators 注册默认的客户端创建器
func (f *LLMFactory) registerDefaultCreators() {
	f.creators[ProviderOpenAI] = func(config *LLMConfig) (LLMClient, error) {
		return NewOpenAIClient(config)
	}

	f.creators[ProviderClaude] = func(config *LLMConfig) (LLMClient, error) {
		return NewClaudeClient(config)
	}

	f.creators[ProviderQianwen] = func(config *LLMConfig) (LLMClient, error) {
		return NewQianwenClient(config)
	}

	f.creators[ProviderDeepSeek] = func(config *LLMConfig) (LLMClient, error) {
		return NewDeepSeekClient(config)
	}

	f.creators[ProviderOllamaLocal] = func(config *LLMConfig) (LLMClient, error) {
		return NewOllamaLocalClient(config)
	}
}

// RegisterProvider 注册新的LLM提供商 - 支持扩展
func (f *LLMFactory) RegisterProvider(provider LLMProvider, creator ClientCreator) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.creators[provider] = creator
}

// SetConfig 设置提供商配置
func (f *LLMFactory) SetConfig(provider LLMProvider, config *LLMConfig) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.configs[provider] = config

	// 清除缓存，强制重新创建
	if client, exists := f.cache[provider]; exists {
		client.Close()
		delete(f.cache, provider)
	}
}

// CreateClient 创建LLM客户端 - 工厂方法
func (f *LLMFactory) CreateClient(provider LLMProvider) (LLMClient, error) {
	f.mutex.RLock()
	if client, exists := f.cache[provider]; exists {
		f.mutex.RUnlock()
		return client, nil
	}
	f.mutex.RUnlock()

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 双重检查锁定
	if client, exists := f.cache[provider]; exists {
		return client, nil
	}

	config, exists := f.configs[provider]
	if !exists {
		return nil, &LLMError{
			Provider:  provider,
			Code:      "CONFIG_NOT_FOUND",
			Message:   fmt.Sprintf("未配置的LLM提供商: %s", provider),
			Retryable: false,
		}
	}

	creator, exists := f.creators[provider]
	if !exists {
		return nil, &LLMError{
			Provider:  provider,
			Code:      "CREATOR_NOT_FOUND",
			Message:   fmt.Sprintf("不支持的LLM提供商: %s", provider),
			Retryable: false,
		}
	}

	client, err := creator(config)
	if err != nil {
		return nil, &LLMError{
			Provider:  provider,
			Code:      "CLIENT_CREATION_FAILED",
			Message:   fmt.Sprintf("创建LLM客户端失败: %v", err),
			Retryable: true,
		}
	}

	f.cache[provider] = client
	return client, nil
}

// GetClient 获取已创建的客户端
func (f *LLMFactory) GetClient(provider LLMProvider) (LLMClient, bool) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	client, exists := f.cache[provider]
	return client, exists
}

// ListProviders 列出所有支持的提供商
func (f *LLMFactory) ListProviders() []LLMProvider {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	providers := make([]LLMProvider, 0, len(f.creators))
	for provider := range f.creators {
		providers = append(providers, provider)
	}

	return providers
}

// ListConfiguredProviders 列出已配置的提供商
func (f *LLMFactory) ListConfiguredProviders() []LLMProvider {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	providers := make([]LLMProvider, 0, len(f.configs))
	for provider := range f.configs {
		providers = append(providers, provider)
	}

	return providers
}

// Close 关闭所有客户端
func (f *LLMFactory) Close() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var lastErr error
	for provider, client := range f.cache {
		if err := client.Close(); err != nil {
			lastErr = err
		}
		delete(f.cache, provider)
	}

	return lastErr
}

// =============================================================================
// 全局工厂实例
// =============================================================================

var (
	globalFactory *LLMFactory
	factoryOnce   sync.Once
)

// GetGlobalFactory 获取全局工厂实例
func GetGlobalFactory() *LLMFactory {
	factoryOnce.Do(func() {
		globalFactory = NewLLMFactory()
	})
	return globalFactory
}

// RegisterGlobalProvider 在全局工厂中注册提供商
func RegisterGlobalProvider(provider LLMProvider, creator ClientCreator) {
	GetGlobalFactory().RegisterProvider(provider, creator)
}

// SetGlobalConfig 设置全局配置
func SetGlobalConfig(provider LLMProvider, config *LLMConfig) {
	GetGlobalFactory().SetConfig(provider, config)
}

// CreateGlobalClient 创建全局客户端
func CreateGlobalClient(provider LLMProvider) (LLMClient, error) {
	return GetGlobalFactory().CreateClient(provider)
}
