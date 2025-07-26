package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FeatureFlag 特性开关
type FeatureFlag struct {
	Name        string      `json:"name"`
	Enabled     bool        `json:"enabled"`
	Description string      `json:"description"`
	Config      interface{} `json:"config,omitempty"`
}

// AgenticConfig Agentic RAG配置
type AgenticConfig struct {
	// 🔥 新增：组件化配置 - 支持独立插拔
	Components ComponentsConfig `json:"components"`

	// 第一阶段：智能检索决策引擎
	RetrievalDecision struct {
		Enabled             bool    `json:"enabled"`
		ConfidenceThreshold float64 `json:"confidence_threshold"`
		MaxRetries          int     `json:"max_retries"`
	} `json:"retrieval_decision"`

	// 查询优化
	QueryOptimization struct {
		Enabled       bool `json:"enabled"`
		HydeEnabled   bool `json:"hyde_enabled"`
		Query2Doc     bool `json:"query2doc_enabled"`
		MaxExpansions int  `json:"max_expansions"`
	} `json:"query_optimization"`

	// 质量评估
	QualityEvaluation struct {
		Enabled            bool    `json:"enabled"`
		RelevanceThreshold float64 `json:"relevance_threshold"`
		DiversityWeight    float64 `json:"diversity_weight"`
	} `json:"quality_evaluation"`

	// 实验性功能
	Experimental struct {
		MultiHopReasoning  bool `json:"multi_hop_reasoning"`
		SemanticCache      bool `json:"semantic_cache"`
		QueryDecomposition bool `json:"query_decomposition"`
	} `json:"experimental"`
}

// 🔥 新增：组件化配置
type ComponentsConfig struct {
	// 检索过滤器 - 可独立插拔（你说的拦截逻辑）
	RetrievalFilter struct {
		Enabled  bool                   `json:"enabled"`
		Rules    []string               `json:"rules"`    // 启用的规则列表
		Priority int                    `json:"priority"` // 组件优先级
		Config   map[string]interface{} `json:"config"`   // 自定义配置
	} `json:"retrieval_filter"`

	// 🔥 查询改写器 - 重点研究对象
	QueryRewriter struct {
		Enabled          bool                   `json:"enabled"`
		KeywordExtractor bool                   `json:"keyword_extractor"` // 关键词提取
		NoiseReducer     bool                   `json:"noise_reducer"`     // 噪声去除
		ContextEnricher  bool                   `json:"context_enricher"`  // 上下文丰富
		Priority         int                    `json:"priority"`
		Config           map[string]interface{} `json:"config"`
	} `json:"query_rewriter"`

	// 🔥 迭代检索器 - 多次调整机制
	IterativeRetrieval struct {
		Enabled       bool                   `json:"enabled"`
		MaxIterations int                    `json:"max_iterations"` // 最大迭代次数
		QualityCheck  bool                   `json:"quality_check"`  // 质量评估
		RewriteRules  bool                   `json:"rewrite_rules"`  // 改写规则
		Priority      int                    `json:"priority"`
		Config        map[string]interface{} `json:"config"`
	} `json:"iterative_retrieval"`

	// 质量评估器
	QualityEvaluator struct {
		Enabled  bool                   `json:"enabled"`
		Priority int                    `json:"priority"`
		Config   map[string]interface{} `json:"config"`
	} `json:"quality_evaluator"`
}

// FeatureFlagManager 特性开关管理器
type FeatureFlagManager struct {
	mu     sync.RWMutex
	flags  map[string]*FeatureFlag
	config *AgenticConfig
}

// NewFeatureFlagManager 创建特性开关管理器
func NewFeatureFlagManager() *FeatureFlagManager {
	manager := &FeatureFlagManager{
		flags:  make(map[string]*FeatureFlag),
		config: &AgenticConfig{},
	}

	// 设置默认配置
	manager.setDefaults()

	// 尝试从文件加载配置
	if err := manager.LoadFromFile("config/agentic.json"); err != nil {
		// 文件不存在时使用默认配置
		manager.saveDefaults()
	}

	return manager
}

// setDefaults 设置默认配置
func (fm *FeatureFlagManager) setDefaults() {
	// 第一阶段默认关闭，需要手动启用
	fm.config.RetrievalDecision.Enabled = false
	fm.config.RetrievalDecision.ConfidenceThreshold = 0.6
	fm.config.RetrievalDecision.MaxRetries = 2

	fm.config.QueryOptimization.Enabled = false
	fm.config.QueryOptimization.HydeEnabled = false
	fm.config.QueryOptimization.Query2Doc = false
	fm.config.QueryOptimization.MaxExpansions = 3

	fm.config.QualityEvaluation.Enabled = false
	fm.config.QualityEvaluation.RelevanceThreshold = 0.7
	fm.config.QualityEvaluation.DiversityWeight = 0.3

	// 🔥 新增：组件化配置默认值
	// 检索过滤器配置
	fm.config.Components.RetrievalFilter.Enabled = false
	fm.config.Components.RetrievalFilter.Rules = []string{"status_confirmation", "emotional_feedback", "ultra_short_query"}
	fm.config.Components.RetrievalFilter.Priority = 100
	fm.config.Components.RetrievalFilter.Config = map[string]interface{}{
		"confidence_threshold": 0.6,
		"min_query_length":     3,
	}

	// 查询改写器配置
	fm.config.Components.QueryRewriter.Enabled = false
	fm.config.Components.QueryRewriter.KeywordExtractor = true
	fm.config.Components.QueryRewriter.NoiseReducer = true
	fm.config.Components.QueryRewriter.ContextEnricher = true
	fm.config.Components.QueryRewriter.Priority = 90
	fm.config.Components.QueryRewriter.Config = map[string]interface{}{
		"max_iterations":    3,
		"quality_threshold": 0.7,
	}

	// 迭代检索器配置
	fm.config.Components.IterativeRetrieval.Enabled = false
	fm.config.Components.IterativeRetrieval.MaxIterations = 3
	fm.config.Components.IterativeRetrieval.QualityCheck = true
	fm.config.Components.IterativeRetrieval.RewriteRules = true
	fm.config.Components.IterativeRetrieval.Priority = 80
	fm.config.Components.IterativeRetrieval.Config = map[string]interface{}{
		"quality_threshold": 0.6,
		"timeout_ms":        30000,
	}

	// 质量评估器配置
	fm.config.Components.QualityEvaluator.Enabled = false
	fm.config.Components.QualityEvaluator.Priority = 70
	fm.config.Components.QualityEvaluator.Config = map[string]interface{}{
		"metrics":       []string{"relevance", "diversity", "completeness"},
		"feedback_loop": true,
	}

	// 实验性功能默认关闭
	fm.config.Experimental.MultiHopReasoning = false
	fm.config.Experimental.SemanticCache = false
	fm.config.Experimental.QueryDecomposition = false

	// 注册特性开关
	fm.registerFlag("retrieval_decision", "智能检索决策引擎", fm.config.RetrievalDecision.Enabled)
	fm.registerFlag("query_optimization", "查询优化器", fm.config.QueryOptimization.Enabled)
	fm.registerFlag("quality_evaluation", "质量评估器", fm.config.QualityEvaluation.Enabled)

	// 🔥 注册组件级开关
	fm.registerFlag("retrieval_filter", "检索过滤器", fm.config.Components.RetrievalFilter.Enabled)
	fm.registerFlag("query_rewriter", "查询改写器", fm.config.Components.QueryRewriter.Enabled)
	fm.registerFlag("iterative_retrieval", "迭代检索器", fm.config.Components.IterativeRetrieval.Enabled)
	fm.registerFlag("quality_evaluator", "质量评估器", fm.config.Components.QualityEvaluator.Enabled)
	fm.registerFlag("multi_hop_reasoning", "多跳推理", fm.config.Experimental.MultiHopReasoning)
}

// registerFlag 注册特性开关
func (fm *FeatureFlagManager) registerFlag(name, description string, enabled bool) {
	fm.flags[name] = &FeatureFlag{
		Name:        name,
		Enabled:     enabled,
		Description: description,
	}
}

// IsEnabled 检查特性是否启用
func (fm *FeatureFlagManager) IsEnabled(flagName string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if flag, exists := fm.flags[flagName]; exists {
		return flag.Enabled
	}
	return false
}

// EnableFeature 启用特性
func (fm *FeatureFlagManager) EnableFeature(flagName string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if flag, exists := fm.flags[flagName]; exists {
		flag.Enabled = true

		// 同步到配置结构
		switch flagName {
		case "retrieval_decision":
			fm.config.RetrievalDecision.Enabled = true
		case "query_optimization":
			fm.config.QueryOptimization.Enabled = true
		case "quality_evaluation":
			fm.config.QualityEvaluation.Enabled = true
		case "multi_hop_reasoning":
			fm.config.Experimental.MultiHopReasoning = true
		}

		return fm.SaveToFile("config/agentic.json")
	}
	return fmt.Errorf("特性开关 %s 不存在", flagName)
}

// DisableFeature 禁用特性
func (fm *FeatureFlagManager) DisableFeature(flagName string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if flag, exists := fm.flags[flagName]; exists {
		flag.Enabled = false

		// 同步到配置结构
		switch flagName {
		case "retrieval_decision":
			fm.config.RetrievalDecision.Enabled = false
		case "query_optimization":
			fm.config.QueryOptimization.Enabled = false
		case "quality_evaluation":
			fm.config.QualityEvaluation.Enabled = false
		case "multi_hop_reasoning":
			fm.config.Experimental.MultiHopReasoning = false
		}

		return fm.SaveToFile("config/agentic.json")
	}
	return fmt.Errorf("特性开关 %s 不存在", flagName)
}

// GetConfig 获取配置
func (fm *FeatureFlagManager) GetConfig() *AgenticConfig {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.config
}

// LoadFromFile 从文件加载配置
func (fm *FeatureFlagManager) LoadFromFile(filepath string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, fm.config)
}

// SaveToFile 保存配置到文件
func (fm *FeatureFlagManager) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(fm.config, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll("config", 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// saveDefaults 保存默认配置
func (fm *FeatureFlagManager) saveDefaults() {
	fm.SaveToFile("config/agentic.json")
}

// ListFlags 列出所有特性开关
func (fm *FeatureFlagManager) ListFlags() map[string]*FeatureFlag {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make(map[string]*FeatureFlag)
	for k, v := range fm.flags {
		result[k] = &FeatureFlag{
			Name:        v.Name,
			Enabled:     v.Enabled,
			Description: v.Description,
			Config:      v.Config,
		}
	}
	return result
}
