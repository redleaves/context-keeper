package models

import "time"

// SmartAnalysisResult 智能分析结果（替换现有的interface{}）
type SmartAnalysisResult struct {
	// 意图分析结果
	IntentAnalysis *IntentAnalysisResult `json:"intent_analysis"`

	// 置信度评估
	ConfidenceAssessment *ConfidenceAssessment `json:"confidence_assessment"`

	// 存储建议
	StorageRecommendations *StorageRecommendations `json:"storage_recommendations"`

	// 🆕 时间回忆查询字段 - 与其他字段互斥
	TimelineRecall *TimelineRecall `json:"timeline_recall,omitempty"`

	// 🆕 知识图谱抽取结果（可选）
	KnowledgeGraphExtraction *KnowledgeGraphExtraction `json:"knowledge_graph_extraction,omitempty"`

	// 原始LLM响应（用于调试）
	RawLLMResponse string `json:"raw_llm_response,omitempty"`
}

// IntentAnalysisResult 意图分析结果
type IntentAnalysisResult struct {
	CoreIntentText       string   `json:"core_intent_text"`       // 核心意图关键词
	DomainContextText    string   `json:"domain_context_text"`    // 领域上下文
	ScenarioText         string   `json:"scenario_text"`          // 场景描述
	IntentCount          int      `json:"intent_count"`           // 意图数量
	MultiIntentBreakdown []string `json:"multi_intent_breakdown"` // 多意图拆分
	Summary              string   `json:"summary"`                // 结构化摘要（100-200字符）
}

// ConfidenceAssessment 置信度评估
type ConfidenceAssessment struct {
	SemanticClarity         float64  `json:"semantic_clarity"`         // 语义清晰度 0-1
	InformationCompleteness float64  `json:"information_completeness"` // 信息完整度 0-1
	IntentConfidence        float64  `json:"intent_confidence"`        // 意图识别可信度 0-1
	OverallConfidence       float64  `json:"overall_confidence"`       // 整体置信度 0-1
	MissingElements         []string `json:"missing_elements"`         // 缺失要素
	ClarityIssues           []string `json:"clarity_issues"`           // 清晰度问题
}

// StorageRecommendations 存储建议
type StorageRecommendations struct {
	TimelineStorage       *StorageRecommendation       `json:"timeline_storage"`
	KnowledgeGraphStorage *StorageRecommendation       `json:"knowledge_graph_storage"`
	VectorStorage         *VectorStorageRecommendation `json:"vector_storage"`
}

// StorageRecommendation 存储建议
type StorageRecommendation struct {
	ShouldStore         bool    `json:"should_store"`
	Reason              string  `json:"reason"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	// 🔥 时间线存储专用字段
	TimelineTime string `json:"timeline_time,omitempty"` // 时间标识：具体时间或'now'表示当前时间
	EventType    string `json:"event_type,omitempty"`    // 🆕 事件类型：design, code_edit, problem_solve等
}

// VectorStorageRecommendation 向量存储建议
type VectorStorageRecommendation struct {
	*StorageRecommendation
	EnabledDimensions []string `json:"enabled_dimensions"` // 启用的维度
}

// MultiVectorData 多向量数据
type MultiVectorData struct {
	// 四维度向量字段
	CoreIntentVector    []float32 `json:"core_intent_vector,omitempty"`    // 核心意图向量
	DomainContextVector []float32 `json:"domain_context_vector,omitempty"` // 领域上下文向量
	ScenarioVector      []float32 `json:"scenario_vector,omitempty"`       // 场景向量
	CompletenessVector  []float32 `json:"completeness_vector,omitempty"`   // 完整度向量

	// 对应的精炼文本
	CoreIntentText    string `json:"core_intent_text,omitempty"`    // 核心意图文本
	DomainContextText string `json:"domain_context_text,omitempty"` // 领域上下文文本
	ScenarioText      string `json:"scenario_text,omitempty"`       // 场景文本
	CompletenessText  string `json:"completeness_text,omitempty"`   // 完整度文本

	// 维度权重
	CoreIntentWeight    float64 `json:"core_intent_weight,omitempty"`    // 核心意图权重
	DomainContextWeight float64 `json:"domain_context_weight,omitempty"` // 领域上下文权重
	ScenarioWeight      float64 `json:"scenario_weight,omitempty"`       // 场景权重
	CompletenessWeight  float64 `json:"completeness_weight,omitempty"`   // 完整度权重

	// 质量评分
	QualityScore *ConfidenceAssessment `json:"quality_score,omitempty"`

	// 元数据
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MultiVectorSearchOptions 多向量检索选项
type MultiVectorSearchOptions struct {
	// 维度选择
	EnabledDimensions []string `json:"enabled_dimensions"`

	// 维度权重
	DimensionWeights map[string]float64 `json:"dimension_weights"`

	// 检索策略
	SearchStrategy string `json:"search_strategy"` // "intent_focused", "domain_focused", "balanced"

	// 置信度过滤
	MinConfidence float64 `json:"min_confidence"`

	// 其他选项
	Limit int `json:"limit"`
}

// SmartStorageConfig 智能存储配置
type SmartStorageConfig struct {
	// 置信度阈值配置
	ConfidenceThresholds *ConfidenceThresholds `json:"confidence_thresholds"`

	// 多向量配置
	MultiVectorConfig *MultiVectorConfig `json:"multi_vector_config"`

	// 存储策略配置
	StorageStrategyConfig *StorageStrategyConfig `json:"storage_strategy_config"`
}

// ConfidenceThresholds 置信度阈值配置
type ConfidenceThresholds struct {
	TimelineStorage       float64 `json:"timeline_storage"`        // 默认 0.7
	KnowledgeGraphStorage float64 `json:"knowledge_graph_storage"` // 默认 0.6
	VectorStorage         float64 `json:"vector_storage"`          // 默认 0.5
	ContextOnlyThreshold  float64 `json:"context_only_threshold"`  // 默认 0.5
}

// MultiVectorConfig 多向量配置
type MultiVectorConfig struct {
	EnabledDimensions []string           `json:"enabled_dimensions"` // ["core_intent", "domain_context", "scenario"]
	DefaultWeights    map[string]float64 `json:"default_weights"`    // 默认权重配置
	MaxDimensions     int                `json:"max_dimensions"`     // 最大维度数量，默认4
}

// StorageStrategyConfig 存储策略配置
type StorageStrategyConfig struct {
	EnableFallback         bool `json:"enable_fallback"`           // 启用降级机制
	FallbackToSingleVector bool `json:"fallback_to_single_vector"` // 降级到单向量存储
	LogAnalysisDetails     bool `json:"log_analysis_details"`      // 记录分析详情
	EnableAsyncStorage     bool `json:"enable_async_storage"`      // 启用异步存储
	StorageTimeoutSeconds  int  `json:"storage_timeout_seconds"`   // 存储超时时间
}

// 🆕 知识图谱抽取相关数据结构

// KnowledgeGraphExtraction 知识图谱抽取结果
type KnowledgeGraphExtraction struct {
	Entities       []LLMExtractedEntity       `json:"entities"`
	Relationships  []LLMExtractedRelationship `json:"relationships"`
	ExtractionMeta *ExtractionMetadata        `json:"extraction_meta,omitempty"`
}

// LLMExtractedEntity LLM抽取的实体
type LLMExtractedEntity struct {
	Title       string   `json:"title"`
	Type        string   `json:"type"` // Technical/Project/Concept/Issue/Data/Process
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Keywords    []string `json:"keywords,omitempty"`
}

// LLMExtractedRelationship LLM抽取的关系
type LLMExtractedRelationship struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	RelationType string  `json:"relation_type"` // USES/SOLVES/BELONGS_TO/CAUSES/RELATED_TO
	Description  string  `json:"description"`
	Strength     int     `json:"strength"` // 1-10评分
	Confidence   float64 `json:"confidence"`
	Evidence     string  `json:"evidence"` // 支持证据
}

// ExtractionMetadata 抽取元数据
type ExtractionMetadata struct {
	EntityCount       int     `json:"entity_count"`
	RelationshipCount int     `json:"relationship_count"`
	OverallQuality    float64 `json:"overall_quality"`
	ProcessingTime    string  `json:"processing_time,omitempty"`
	StrategyUsed      string  `json:"strategy_used,omitempty"`
}

// 🆕 TimelineRecall 时间回忆查询结构
type TimelineRecall struct {
	StartTime string `json:"start_time"` // YYYY-MM-DD HH:mm:ss
	EndTime   string `json:"end_time"`   // YYYY-MM-DD HH:mm:ss
}
