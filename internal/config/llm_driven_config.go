package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// LLMDrivenConfigManager LLM驱动配置管理器
type LLMDrivenConfigManager struct {
	configPath string
	config     *LLMDrivenFullConfig
}

// LLMDrivenFullConfig 完整的LLM驱动配置
type LLMDrivenFullConfig struct {
	// 总开关
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 功能开关
	Features struct {
		SemanticAnalysis bool `json:"semantic_analysis" yaml:"semantic_analysis"`
		MultiDimensional bool `json:"multi_dimensional" yaml:"multi_dimensional"`
		ContentSynthesis bool `json:"content_synthesis" yaml:"content_synthesis"`
		ContextUpdates   bool `json:"context_updates" yaml:"context_updates"`
		// 🔥 短期记忆LLM驱动开关（独立控制，默认关闭）
		ShortTermMemoryLLM bool `json:"short_term_memory_llm" yaml:"short_term_memory_llm"`
	} `json:"features" yaml:"features"`

	// 降级策略
	Fallback struct {
		EnableAutoFallback bool `json:"enable_auto_fallback" yaml:"enable_auto_fallback"`
		FallbackThreshold  int  `json:"fallback_threshold" yaml:"fallback_threshold"`
		MaxRetries         int  `json:"max_retries" yaml:"max_retries"`
	} `json:"fallback" yaml:"fallback"`

	// LLM配置
	LLM struct {
		Provider    string  `json:"provider" yaml:"provider"`
		Model       string  `json:"model" yaml:"model"`
		MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
		Temperature float64 `json:"temperature" yaml:"temperature"`
		Timeout     int     `json:"timeout" yaml:"timeout"`
	} `json:"llm" yaml:"llm"`

	// 存储配置
	Storage struct {
		TimelineDB struct {
			Enabled          bool   `json:"enabled" yaml:"enabled"`
			ConnectionString string `json:"connection_string" yaml:"connection_string"`
		} `json:"timeline_db" yaml:"timeline_db"`

		KnowledgeGraph struct {
			Enabled          bool   `json:"enabled" yaml:"enabled"`
			ConnectionString string `json:"connection_string" yaml:"connection_string"`
		} `json:"knowledge_graph" yaml:"knowledge_graph"`
	} `json:"storage" yaml:"storage"`

	// 🆕 智能存储配置
	SmartStorage struct {
		// 置信度阈值配置
		ConfidenceThresholds struct {
			TimelineStorage       float64 `json:"timeline_storage" yaml:"timeline_storage"`               // 时间线存储阈值，默认 0.7
			KnowledgeGraphStorage float64 `json:"knowledge_graph_storage" yaml:"knowledge_graph_storage"` // 知识图谱存储阈值，默认 0.6
			VectorStorage         float64 `json:"vector_storage" yaml:"vector_storage"`                   // 向量存储阈值，默认 0.5
			ContextOnlyThreshold  float64 `json:"context_only_threshold" yaml:"context_only_threshold"`   // 仅上下文记录阈值，默认 0.5
		} `json:"confidence_thresholds" yaml:"confidence_thresholds"`

		// 多向量配置
		MultiVector struct {
			EnabledDimensions []string           `json:"enabled_dimensions" yaml:"enabled_dimensions"` // 启用的维度
			DefaultWeights    map[string]float64 `json:"default_weights" yaml:"default_weights"`       // 默认权重配置
			MaxDimensions     int                `json:"max_dimensions" yaml:"max_dimensions"`         // 最大维度数量，默认4
		} `json:"multi_vector" yaml:"multi_vector"`

		// 存储策略配置
		Strategy struct {
			EnableFallback         bool `json:"enable_fallback" yaml:"enable_fallback"`                     // 启用降级机制
			FallbackToSingleVector bool `json:"fallback_to_single_vector" yaml:"fallback_to_single_vector"` // 降级到单向量存储
			LogAnalysisDetails     bool `json:"log_analysis_details" yaml:"log_analysis_details"`           // 记录分析详情
			EnableAsyncStorage     bool `json:"enable_async_storage" yaml:"enable_async_storage"`           // 启用异步存储
			StorageTimeoutSeconds  int  `json:"storage_timeout_seconds" yaml:"storage_timeout_seconds"`     // 存储超时时间
		} `json:"strategy" yaml:"strategy"`
	} `json:"smart_storage" yaml:"smart_storage"`

	// 性能配置
	Performance struct {
		MaxConcurrentRequests int  `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`
		RequestTimeout        int  `json:"request_timeout" yaml:"request_timeout"`
		CacheEnabled          bool `json:"cache_enabled" yaml:"cache_enabled"`
		CacheTTL              int  `json:"cache_ttl" yaml:"cache_ttl"`
	} `json:"performance" yaml:"performance"`

	// 监控配置
	Monitoring struct {
		MetricsEnabled bool   `json:"metrics_enabled" yaml:"metrics_enabled"`
		LogLevel       string `json:"log_level" yaml:"log_level"`
		AlertEnabled   bool   `json:"alert_enabled" yaml:"alert_enabled"`
	} `json:"monitoring" yaml:"monitoring"`
}

// NewLLMDrivenConfigManager 创建配置管理器
func NewLLMDrivenConfigManager(configPath string) *LLMDrivenConfigManager {
	return &LLMDrivenConfigManager{
		configPath: configPath,
	}
}

// LoadConfig 加载配置
func (cm *LLMDrivenConfigManager) LoadConfig() (*LLMDrivenFullConfig, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		log.Printf("🔧 [配置管理] 配置文件不存在，创建默认配置: %s", cm.configPath)
		if err := cm.createDefaultConfig(); err != nil {
			return nil, fmt.Errorf("创建默认配置失败: %w", err)
		}
	}

	// 读取配置文件
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML配置
	config := &LLMDrivenFullConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 应用环境变量覆盖
	cm.applyEnvironmentOverrides(config)

	cm.config = config
	log.Printf("✅ [配置管理] LLM驱动配置加载完成，启用状态: %v", config.Enabled)
	return config, nil
}

// createDefaultConfig 创建默认配置文件
func (cm *LLMDrivenConfigManager) createDefaultConfig() error {
	defaultConfig := &LLMDrivenFullConfig{
		Enabled: false, // 🔥 默认关闭，确保稳定性

		Features: struct {
			SemanticAnalysis bool `json:"semantic_analysis" yaml:"semantic_analysis"`
			MultiDimensional bool `json:"multi_dimensional" yaml:"multi_dimensional"`
			ContentSynthesis bool `json:"content_synthesis" yaml:"content_synthesis"`
			ContextUpdates   bool `json:"context_updates" yaml:"context_updates"`
			// 🔥 短期记忆LLM驱动开关（独立控制，默认关闭）
			ShortTermMemoryLLM bool `json:"short_term_memory_llm" yaml:"short_term_memory_llm"`
		}{
			SemanticAnalysis:   true,
			MultiDimensional:   true,
			ContentSynthesis:   true,
			ContextUpdates:     false, // 上下文更新功能暂时关闭
			ShortTermMemoryLLM: false, // 🔥 短期记忆LLM驱动默认关闭
		},

		Fallback: struct {
			EnableAutoFallback bool `json:"enable_auto_fallback" yaml:"enable_auto_fallback"`
			FallbackThreshold  int  `json:"fallback_threshold" yaml:"fallback_threshold"`
			MaxRetries         int  `json:"max_retries" yaml:"max_retries"`
		}{
			EnableAutoFallback: true,
			FallbackThreshold:  3,
			MaxRetries:         2,
		},

		LLM: struct {
			Provider    string  `json:"provider" yaml:"provider"`
			Model       string  `json:"model" yaml:"model"`
			MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
			Temperature float64 `json:"temperature" yaml:"temperature"`
			Timeout     int     `json:"timeout" yaml:"timeout"`
		}{
			Provider:    "openai",
			Model:       "gpt-4",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     30,
		},

		Storage: struct {
			TimelineDB struct {
				Enabled          bool   `json:"enabled" yaml:"enabled"`
				ConnectionString string `json:"connection_string" yaml:"connection_string"`
			} `json:"timeline_db" yaml:"timeline_db"`
			KnowledgeGraph struct {
				Enabled          bool   `json:"enabled" yaml:"enabled"`
				ConnectionString string `json:"connection_string" yaml:"connection_string"`
			} `json:"knowledge_graph" yaml:"knowledge_graph"`
		}{
			TimelineDB: struct {
				Enabled          bool   `json:"enabled" yaml:"enabled"`
				ConnectionString string `json:"connection_string" yaml:"connection_string"`
			}{
				Enabled:          false, // TimescaleDB集成暂时关闭
				ConnectionString: "",
			},
			KnowledgeGraph: struct {
				Enabled          bool   `json:"enabled" yaml:"enabled"`
				ConnectionString string `json:"connection_string" yaml:"connection_string"`
			}{
				Enabled:          false, // Neo4j集成暂时关闭
				ConnectionString: "",
			},
		},

		Performance: struct {
			MaxConcurrentRequests int  `json:"max_concurrent_requests" yaml:"max_concurrent_requests"`
			RequestTimeout        int  `json:"request_timeout" yaml:"request_timeout"`
			CacheEnabled          bool `json:"cache_enabled" yaml:"cache_enabled"`
			CacheTTL              int  `json:"cache_ttl" yaml:"cache_ttl"`
		}{
			MaxConcurrentRequests: 10,
			RequestTimeout:        60,
			CacheEnabled:          true,
			CacheTTL:              300,
		},

		Monitoring: struct {
			MetricsEnabled bool   `json:"metrics_enabled" yaml:"metrics_enabled"`
			LogLevel       string `json:"log_level" yaml:"log_level"`
			AlertEnabled   bool   `json:"alert_enabled" yaml:"alert_enabled"`
		}{
			MetricsEnabled: true,
			LogLevel:       "info",
			AlertEnabled:   false,
		},
	}

	// 确保配置目录存在
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("序列化默认配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入默认配置文件失败: %w", err)
	}

	log.Printf("✅ [配置管理] 默认配置文件已创建: %s", cm.configPath)
	return nil
}

// applyEnvironmentOverrides 应用环境变量覆盖
func (cm *LLMDrivenConfigManager) applyEnvironmentOverrides(config *LLMDrivenFullConfig) {
	// 总开关
	if val := os.Getenv("LLM_DRIVEN_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Enabled = enabled
			log.Printf("🔧 [配置管理] 环境变量覆盖 - LLM驱动启用状态: %v", enabled)
		}
	}

	// 功能开关
	if val := os.Getenv("LLM_DRIVEN_SEMANTIC_ANALYSIS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.SemanticAnalysis = enabled
		}
	}

	if val := os.Getenv("LLM_DRIVEN_MULTI_DIMENSIONAL"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.MultiDimensional = enabled
		}
	}

	if val := os.Getenv("LLM_DRIVEN_CONTENT_SYNTHESIS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.ContentSynthesis = enabled
		}
	}

	// LLM配置
	if val := os.Getenv("LLM_PROVIDER"); val != "" {
		config.LLM.Provider = val
	}

	if val := os.Getenv("LLM_MODEL"); val != "" {
		config.LLM.Model = val
	}

	if val := os.Getenv("LLM_MAX_TOKENS"); val != "" {
		if tokens, err := strconv.Atoi(val); err == nil {
			config.LLM.MaxTokens = tokens
		}
	}

	if val := os.Getenv("LLM_TEMPERATURE"); val != "" {
		if temp, err := strconv.ParseFloat(val, 64); err == nil {
			config.LLM.Temperature = temp
		}
	}
}

// SaveConfig 保存配置
func (cm *LLMDrivenConfigManager) SaveConfig(config *LLMDrivenFullConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	cm.config = config
	log.Printf("✅ [配置管理] 配置已保存: %s", cm.configPath)
	return nil
}

// GetConfig 获取当前配置
func (cm *LLMDrivenConfigManager) GetConfig() *LLMDrivenFullConfig {
	return cm.config
}

// ValidateConfig 验证配置
func (cm *LLMDrivenConfigManager) ValidateConfig(config *LLMDrivenFullConfig) error {
	// 验证LLM配置
	if config.Enabled {
		if config.LLM.Provider == "" {
			return fmt.Errorf("LLM提供商不能为空")
		}

		if config.LLM.Model == "" {
			return fmt.Errorf("LLM模型不能为空")
		}

		if config.LLM.MaxTokens <= 0 {
			return fmt.Errorf("LLM最大Token数必须大于0")
		}

		if config.LLM.Temperature < 0 || config.LLM.Temperature > 2 {
			return fmt.Errorf("LLM温度参数必须在0-2之间")
		}
	}

	// 验证降级配置
	if config.Fallback.FallbackThreshold <= 0 {
		return fmt.Errorf("降级阈值必须大于0")
	}

	// 验证性能配置
	if config.Performance.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("最大并发请求数必须大于0")
	}

	return nil
}

// ReloadConfig 重新加载配置
func (cm *LLMDrivenConfigManager) ReloadConfig() (*LLMDrivenFullConfig, error) {
	log.Printf("🔄 [配置管理] 重新加载配置...")
	return cm.LoadConfig()
}

// GetConfigSummary 获取配置摘要
func (cm *LLMDrivenConfigManager) GetConfigSummary() map[string]interface{} {
	if cm.config == nil {
		return map[string]interface{}{
			"status": "not_loaded",
		}
	}

	return map[string]interface{}{
		"enabled":                 cm.config.Enabled,
		"semantic_analysis":       cm.config.Features.SemanticAnalysis,
		"multi_dimensional":       cm.config.Features.MultiDimensional,
		"content_synthesis":       cm.config.Features.ContentSynthesis,
		"auto_fallback":           cm.config.Fallback.EnableAutoFallback,
		"llm_provider":            cm.config.LLM.Provider,
		"llm_model":               cm.config.LLM.Model,
		"timeline_db_enabled":     cm.config.Storage.TimelineDB.Enabled,
		"knowledge_graph_enabled": cm.config.Storage.KnowledgeGraph.Enabled,
		"metrics_enabled":         cm.config.Monitoring.MetricsEnabled,
	}
}

// GetContextOnlyThreshold 获取仅上下文记录的置信度阈值
func (cm *LLMDrivenConfigManager) GetContextOnlyThreshold() float64 {
	if cm.config == nil {
		return 0.7 // 默认阈值
	}
	return cm.config.SmartStorage.ConfidenceThresholds.ContextOnlyThreshold
}
