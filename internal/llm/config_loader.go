package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// 配置文件加载器
// =============================================================================

// ConfigFile 配置文件结构
type ConfigFile struct {
	LLM LLMConfigFile `yaml:"llm"`
}

// LLMConfigFile LLM配置文件结构
type LLMConfigFile struct {
	Default   DefaultConfig                `yaml:"default"`
	Providers map[string]ProviderConfig    `yaml:"providers"`
	Routing   map[string]RoutingRuleConfig `yaml:"routing"`
	Prompt    PromptConfigFile             `yaml:"prompt"`
}

// DefaultConfig 默认配置
type DefaultConfig struct {
	PrimaryProvider  string `yaml:"primary_provider"`
	FallbackProvider string `yaml:"fallback_provider"`
	CacheEnabled     bool   `yaml:"cache_enabled"`
	CacheTTL         string `yaml:"cache_ttl"`
	MaxRetries       int    `yaml:"max_retries"`
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	EnableRouting    bool   `yaml:"enable_routing"`
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	APIKey     string                 `yaml:"api_key"`
	BaseURL    string                 `yaml:"base_url"`
	Model      string                 `yaml:"model"`
	MaxRetries int                    `yaml:"max_retries"`
	Timeout    string                 `yaml:"timeout"`
	RateLimit  int                    `yaml:"rate_limit"`
	Extra      map[string]interface{} `yaml:"extra"`
}

// RoutingRuleConfig 路由规则配置
type RoutingRuleConfig struct {
	Primary    string                 `yaml:"primary"`
	Fallback   []string               `yaml:"fallback"`
	Conditions map[string]interface{} `yaml:"conditions"`
}

// PromptConfigFile Prompt配置文件
type PromptConfigFile struct {
	DefaultLanguage string            `yaml:"default_language"`
	MaxTokens       int               `yaml:"max_tokens"`
	Temperature     float64           `yaml:"temperature"`
	CustomVars      map[string]string `yaml:"custom_vars"`
}

// ConfigLoader 配置加载器
type ConfigLoader struct {
	configPath string
	config     *ConfigFile
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// LoadConfig 加载配置
func (cl *ConfigLoader) LoadConfig() error {
	// 读取配置文件
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 环境变量替换
	if err := cl.expandEnvVars(&config); err != nil {
		return fmt.Errorf("环境变量替换失败: %w", err)
	}

	cl.config = &config
	return nil
}

// expandEnvVars 展开环境变量
func (cl *ConfigLoader) expandEnvVars(config *ConfigFile) error {
	for providerName, providerConfig := range config.LLM.Providers {
		// 替换API密钥中的环境变量
		if strings.HasPrefix(providerConfig.APIKey, "${") && strings.HasSuffix(providerConfig.APIKey, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(providerConfig.APIKey, "${"), "}")
			envValue := os.Getenv(envVar)
			if envValue == "" {
				// 如果环境变量未设置，跳过该提供商（而不是报错）
				fmt.Printf("Warning: 环境变量 %s 未设置，跳过提供商 %s\n", envVar, providerName)
				delete(config.LLM.Providers, providerName)
				continue
			}
			providerConfig.APIKey = envValue
			config.LLM.Providers[providerName] = providerConfig
		}
	}
	return nil
}

// GetContextAwareLLMConfig 获取上下文感知LLM配置
func (cl *ConfigLoader) GetContextAwareLLMConfig() (*ContextAwareLLMConfig, error) {
	if cl.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	defaultConfig := cl.config.LLM.Default

	// 解析缓存TTL
	cacheTTL, err := time.ParseDuration(defaultConfig.CacheTTL)
	if err != nil {
		cacheTTL = 30 * time.Minute
	}

	return &ContextAwareLLMConfig{
		PrimaryProvider:  LLMProvider(defaultConfig.PrimaryProvider),
		FallbackProvider: LLMProvider(defaultConfig.FallbackProvider),
		CacheEnabled:     defaultConfig.CacheEnabled,
		CacheTTL:         cacheTTL,
		MaxRetries:       defaultConfig.MaxRetries,
		TimeoutSeconds:   defaultConfig.TimeoutSeconds,
		EnableRouting:    defaultConfig.EnableRouting,
	}, nil
}

// GetProviderConfigs 获取提供商配置
func (cl *ConfigLoader) GetProviderConfigs() (map[LLMProvider]*LLMConfig, error) {
	if cl.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	configs := make(map[LLMProvider]*LLMConfig)

	for providerName, providerConfig := range cl.config.LLM.Providers {
		// 解析超时时间
		timeout, err := time.ParseDuration(providerConfig.Timeout)
		if err != nil {
			timeout = 30 * time.Second
		}

		configs[LLMProvider(providerName)] = &LLMConfig{
			Provider:   LLMProvider(providerName),
			APIKey:     providerConfig.APIKey,
			BaseURL:    providerConfig.BaseURL,
			Model:      providerConfig.Model,
			MaxRetries: providerConfig.MaxRetries,
			Timeout:    timeout,
			RateLimit:  providerConfig.RateLimit,
			Extra:      providerConfig.Extra,
		}
	}

	return configs, nil
}

// GetPromptConfig 获取Prompt配置
func (cl *ConfigLoader) GetPromptConfig() (*PromptConfig, error) {
	if cl.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	promptConfig := cl.config.LLM.Prompt

	return &PromptConfig{
		DefaultLanguage: promptConfig.DefaultLanguage,
		MaxTokens:       promptConfig.MaxTokens,
		Temperature:     promptConfig.Temperature,
		CustomVars:      promptConfig.CustomVars,
	}, nil
}

// GetRoutingRules 获取路由规则
func (cl *ConfigLoader) GetRoutingRules() (map[string]*RoutingRule, error) {
	if cl.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	rules := make(map[string]*RoutingRule)

	for taskType, rule := range cl.config.LLM.Routing {
		fallbackProviders := make([]LLMProvider, len(rule.Fallback))
		for i, provider := range rule.Fallback {
			fallbackProviders[i] = LLMProvider(provider)
		}

		rules[taskType] = &RoutingRule{
			TaskType:          taskType,
			PreferredProvider: LLMProvider(rule.Primary),
			FallbackProviders: fallbackProviders,
			Conditions:        rule.Conditions,
		}
	}

	return rules, nil
}

// =============================================================================
// 全局配置管理器
// =============================================================================

// GlobalConfigManager 全局配置管理器
type GlobalConfigManager struct {
	loader  *ConfigLoader
	service *ContextAwareLLMService
}

var (
	globalConfigManager *GlobalConfigManager
)

// InitializeFromConfig 从配置文件初始化
func InitializeFromConfig(configPath string) error {
	// 如果没有提供路径，尝试默认路径
	if configPath == "" {
		// 尝试多个可能的路径
		possiblePaths := []string{
			"config/llm_config.yaml",
			"./config/llm_config.yaml",
			"../config/llm_config.yaml",
			"../../config/llm_config.yaml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return fmt.Errorf("未找到配置文件，请指定配置文件路径")
		}
	}

	// 创建配置加载器
	loader := NewConfigLoader(configPath)
	if err := loader.LoadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 获取配置
	contextConfig, err := loader.GetContextAwareLLMConfig()
	if err != nil {
		return fmt.Errorf("获取上下文配置失败: %w", err)
	}

	providerConfigs, err := loader.GetProviderConfigs()
	if err != nil {
		return fmt.Errorf("获取提供商配置失败: %w", err)
	}

	promptConfig, err := loader.GetPromptConfig()
	if err != nil {
		return fmt.Errorf("获取Prompt配置失败: %w", err)
	}

	// 设置全局配置
	for provider, config := range providerConfigs {
		SetGlobalConfig(provider, config)
	}

	// 创建服务
	service := NewContextAwareLLMService(contextConfig)
	service.promptManager = NewPromptManager(promptConfig)

	globalConfigManager = &GlobalConfigManager{
		loader:  loader,
		service: service,
	}

	return nil
}

// GetGlobalService 获取全局服务
func GetGlobalService() (*ContextAwareLLMService, error) {
	if globalConfigManager == nil {
		return nil, fmt.Errorf("全局配置管理器未初始化，请先调用 InitializeFromConfig")
	}
	return globalConfigManager.service, nil
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	if globalConfigManager == nil {
		return fmt.Errorf("全局配置管理器未初始化")
	}

	if err := globalConfigManager.loader.LoadConfig(); err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	// 重新初始化服务
	contextConfig, _ := globalConfigManager.loader.GetContextAwareLLMConfig()
	globalConfigManager.service.SetConfig(contextConfig)

	return nil
}

// FindConfigFile 查找配置文件
func FindConfigFile() (string, error) {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 向上查找配置文件
	for {
		configPath := filepath.Join(wd, "config", "llm_config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", fmt.Errorf("未找到配置文件 config/llm_config.yaml")
}
