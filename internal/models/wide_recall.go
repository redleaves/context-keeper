package models

import (
	"time"
)

// WideRecallRequest 宽召回请求
type WideRecallRequest struct {
	// === 基础信息 ===
	UserID      string `json:"user_id"`      // 用户ID
	SessionID   string `json:"session_id"`   // 会话ID
	WorkspaceID string `json:"workspace_id"` // 工作空间ID
	UserQuery   string `json:"user_query"`   // 用户查询

	// === 意图分析结果 ===
	IntentAnalysis *WideRecallIntentAnalysis `json:"intent_analysis"` // 意图分析结果

	// === 检索配置 ===
	RetrievalConfig *RetrievalConfig `json:"retrieval_config"` // 检索配置

	// === 元数据 ===
	RequestTime time.Time `json:"request_time"` // 请求时间
}

// WideRecallResponse 宽召回响应
type WideRecallResponse struct {
	// === 基础信息 ===
	Success     bool   `json:"success"`      // 是否成功
	Message     string `json:"message"`      // 响应消息
	RequestID   string `json:"request_id"`   // 请求ID
	ProcessTime int64  `json:"process_time"` // 处理时间(毫秒)

	// === 检索结果 ===
	RetrievalResults *RetrievalResults `json:"retrieval_results"` // 检索结果

	// === 元数据 ===
	ResponseTime time.Time `json:"response_time"` // 响应时间
}

// WideRecallIntentAnalysis 宽召回意图分析结果
type WideRecallIntentAnalysis struct {
	// === 意图分析 ===
	IntentAnalysis WideRecallIntentInfo `json:"intent_analysis"` // 意图分析

	// === 关键词提取 ===
	KeyExtraction KeyExtraction `json:"key_extraction"` // 关键词提取

	// === 检索策略 ===
	RetrievalStrategy WideRecallStrategy `json:"retrieval_strategy"` // 检索策略

	// === 元数据 ===
	ConfidenceLevel float64   `json:"confidence_level"` // 置信度
	AnalysisTime    time.Time `json:"analysis_time"`    // 分析时间
}

// WideRecallIntentInfo 宽召回意图信息
type WideRecallIntentInfo struct {
	CoreIntent      string     `json:"core_intent"`      // 核心意图
	IntentType      IntentType `json:"intent_type"`      // 意图类型（使用现有定义）
	IntentCategory  string     `json:"intent_category"`  // 意图分类
	KeyConcepts     []string   `json:"key_concepts"`     // 关键概念
	TimeScope       string     `json:"time_scope"`       // 时间范围
	UrgencyLevel    Priority   `json:"urgency_level"`    // 紧急程度
	ExpectedOutcome string     `json:"expected_outcome"` // 期望结果
}

// KeyExtraction 关键词提取
type KeyExtraction struct {
	ProjectKeywords   []string `json:"project_keywords"`   // 项目关键词
	TechnicalKeywords []string `json:"technical_keywords"` // 技术关键词
	BusinessKeywords  []string `json:"business_keywords"`  // 业务关键词
	TimeKeywords      []string `json:"time_keywords"`      // 时间关键词
	ActionKeywords    []string `json:"action_keywords"`    // 动作关键词
}

// WideRecallStrategy 宽召回检索策略
type WideRecallStrategy struct {
	TimelineQueries  []TimelineQuery  `json:"timeline_queries"`  // 时间线查询
	KnowledgeQueries []KnowledgeQuery `json:"knowledge_queries"` // 知识图谱查询
	VectorQueries    []VectorQuery    `json:"vector_queries"`    // 向量查询
}

// TimelineQuery 时间线查询
type TimelineQuery struct {
	QueryText  string   `json:"query_text"`  // 查询文本
	TimeRange  string   `json:"time_range"`  // 时间范围
	EventTypes []string `json:"event_types"` // 事件类型
	Priority   int      `json:"priority"`    // 优先级(1-5)
}

// KnowledgeQuery 知识图谱查询
type KnowledgeQuery struct {
	QueryText     string   `json:"query_text"`     // 查询文本
	ConceptTypes  []string `json:"concept_types"`  // 概念类型
	RelationTypes []string `json:"relation_types"` // 关系类型
	Priority      int      `json:"priority"`       // 优先级(1-5)
}

// VectorQuery 向量查询
type VectorQuery struct {
	QueryText           string  `json:"query_text"`           // 查询文本
	SemanticFocus       string  `json:"semantic_focus"`       // 语义焦点
	SimilarityThreshold float64 `json:"similarity_threshold"` // 相似度阈值
	Priority            int     `json:"priority"`             // 优先级(1-5)
}

// RetrievalConfig 检索配置
type RetrievalConfig struct {
	// === 超时配置 ===
	TimelineTimeout  int `json:"timeline_timeout"`  // 时间线检索超时(秒)
	KnowledgeTimeout int `json:"knowledge_timeout"` // 知识图谱检索超时(秒)
	VectorTimeout    int `json:"vector_timeout"`    // 向量检索超时(秒)

	// === 结果数量限制 ===
	TimelineMaxResults  int `json:"timeline_max_results"`  // 时间线最大结果数
	KnowledgeMaxResults int `json:"knowledge_max_results"` // 知识图谱最大结果数
	VectorMaxResults    int `json:"vector_max_results"`    // 向量最大结果数

	// === 质量阈值 ===
	MinSimilarityScore float64 `json:"min_similarity_score"` // 最小相似度分数
	MinRelevanceScore  float64 `json:"min_relevance_score"`  // 最小相关性分数

	// === 重试配置 ===
	MaxRetries    int `json:"max_retries"`    // 最大重试次数
	RetryInterval int `json:"retry_interval"` // 重试间隔(秒)
}

// RetrievalResults 检索结果
type RetrievalResults struct {
	// === 时间线检索结果 ===
	TimelineResults []TimelineResult `json:"timeline_results"` // 时间线结果
	TimelineCount   int              `json:"timeline_count"`   // 时间线结果数量
	TimelineStatus  string           `json:"timeline_status"`  // 时间线检索状态

	// === 知识图谱检索结果 ===
	KnowledgeResults []KnowledgeResult `json:"knowledge_results"` // 知识图谱结果
	KnowledgeCount   int               `json:"knowledge_count"`   // 知识图谱结果数量
	KnowledgeStatus  string            `json:"knowledge_status"`  // 知识图谱检索状态

	// === 向量检索结果 ===
	VectorResults []VectorResult `json:"vector_results"` // 向量结果
	VectorCount   int            `json:"vector_count"`   // 向量结果数量
	VectorStatus  string         `json:"vector_status"`  // 向量检索状态

	// === 汇总信息 ===
	TotalResults   int     `json:"total_results"`   // 总结果数量
	OverallQuality float64 `json:"overall_quality"` // 总体质量评分
	RetrievalTime  int64   `json:"retrieval_time"`  // 检索耗时(毫秒)
	SuccessfulDims int     `json:"successful_dims"` // 成功的维度数量
}

// TimelineResult 时间线检索结果
type TimelineResult struct {
	EventID         string                 `json:"event_id"`         // 事件ID
	EventType       string                 `json:"event_type"`       // 事件类型
	Title           string                 `json:"title"`            // 标题
	Content         string                 `json:"content"`          // 内容
	Timestamp       time.Time              `json:"timestamp"`        // 时间戳
	Source          string                 `json:"source"`           // 来源
	ImportanceScore float64                `json:"importance_score"` // 重要性评分
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	Tags            []string               `json:"tags"`             // 标签
	Metadata        map[string]interface{} `json:"metadata"`         // 元数据
}

// KnowledgeResult 知识图谱检索结果
type KnowledgeResult struct {
	ConceptID       string                 `json:"concept_id"`       // 概念ID
	ConceptName     string                 `json:"concept_name"`     // 概念名称
	ConceptType     string                 `json:"concept_type"`     // 概念类型
	Description     string                 `json:"description"`      // 描述
	RelatedConcepts []RelatedConcept       `json:"related_concepts"` // 相关概念
	Properties      map[string]interface{} `json:"properties"`       // 属性
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	ConfidenceScore float64                `json:"confidence_score"` // 置信度评分
	Source          string                 `json:"source"`           // 来源
	LastUpdated     time.Time              `json:"last_updated"`     // 最后更新时间
}

// RelatedConcept 相关概念
type RelatedConcept struct {
	ConceptName    string  `json:"concept_name"`    // 概念名称
	RelationType   string  `json:"relation_type"`   // 关系类型
	RelationWeight float64 `json:"relation_weight"` // 关系权重
}

// VectorResult 向量检索结果
type VectorResult struct {
	DocumentID      string                 `json:"document_id"`      // 文档ID
	Content         string                 `json:"content"`          // 内容
	ContentType     string                 `json:"content_type"`     // 内容类型
	Source          string                 `json:"source"`           // 来源
	Similarity      float64                `json:"similarity"`       // 相似度
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	Timestamp       time.Time              `json:"timestamp"`        // 时间戳
	Tags            []string               `json:"tags"`             // 标签
	Metadata        map[string]interface{} `json:"metadata"`         // 元数据
	MatchedSegments []MatchedSegment       `json:"matched_segments"` // 匹配片段
}

// MatchedSegment 匹配片段
type MatchedSegment struct {
	SegmentText string  `json:"segment_text"` // 片段文本
	StartPos    int     `json:"start_pos"`    // 开始位置
	EndPos      int     `json:"end_pos"`      // 结束位置
	Similarity  float64 `json:"similarity"`   // 相似度
}

// ContextSynthesisRequest 上下文合成请求
type ContextSynthesisRequest struct {
	// === 基础信息 ===
	UserID      string `json:"user_id"`      // 用户ID
	SessionID   string `json:"session_id"`   // 会话ID
	WorkspaceID string `json:"workspace_id"` // 工作空间ID
	UserQuery   string `json:"user_query"`   // 用户查询

	// === 输入数据 ===
	IntentAnalysis   *WideRecallIntentAnalysis `json:"intent_analysis"`   // 意图分析结果
	CurrentContext   *UnifiedContextModel      `json:"current_context"`   // 当前上下文
	RetrievalResults *RetrievalResults         `json:"retrieval_results"` // 检索结果

	// === 合成配置 ===
	SynthesisConfig *SynthesisConfig `json:"synthesis_config"` // 合成配置

	// === 元数据 ===
	RequestTime time.Time `json:"request_time"` // 请求时间
}

// ContextSynthesisResponse 上下文合成响应
type ContextSynthesisResponse struct {
	// === 基础信息 ===
	Success     bool   `json:"success"`      // 是否成功
	Message     string `json:"message"`      // 响应消息
	RequestID   string `json:"request_id"`   // 请求ID
	ProcessTime int64  `json:"process_time"` // 处理时间(毫秒)

	// === 评估结果 ===
	EvaluationResult *EvaluationResult `json:"evaluation_result"` // 评估结果

	// === 合成结果 ===
	SynthesizedContext *UnifiedContextModel `json:"synthesized_context"` // 合成后的上下文

	// === 用户响应合成 ===
	UserResponse *UserResponseSynthesis `json:"user_response"` // 用户响应合成结果

	// === 合成元数据 ===
	SynthesisMetadata *SynthesisMetadata `json:"synthesis_metadata"` // 合成元数据

	// === 元数据 ===
	ResponseTime time.Time `json:"response_time"` // 响应时间
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	ShouldUpdate     bool                       `json:"should_update"`     // 是否应该更新
	UpdateConfidence float64                    `json:"update_confidence"` // 更新置信度
	EvaluationReason string                     `json:"evaluation_reason"` // 评估原因
	SemanticChanges  []WideRecallSemanticChange `json:"semantic_changes"`  // 语义变化
}

// WideRecallSemanticChange 宽召回语义变化
type WideRecallSemanticChange struct {
	Dimension         string   `json:"dimension"`          // 变化维度
	ChangeType        string   `json:"change_type"`        // 变化类型
	ChangeDescription string   `json:"change_description"` // 变化描述
	Evidence          []string `json:"evidence"`           // 证据
}

// SynthesisConfig 合成配置
type SynthesisConfig struct {
	// === LLM配置 ===
	LLMTimeout          int     `json:"llm_timeout"`          // LLM超时时间(秒)
	MaxTokens           int     `json:"max_tokens"`           // 最大token数
	Temperature         float64 `json:"temperature"`          // 温度参数
	ConfidenceThreshold float64 `json:"confidence_threshold"` // 置信度阈值

	// === 合成策略 ===
	ConflictResolution string `json:"conflict_resolution"` // 冲突解决策略
	InformationFusion  string `json:"information_fusion"`  // 信息融合策略
	QualityAssessment  string `json:"quality_assessment"`  // 质量评估策略

	// === 更新策略 ===
	UpdateThreshold      float64 `json:"update_threshold"`      // 更新阈值
	PersistenceThreshold float64 `json:"persistence_threshold"` // 持久化阈值
}

// SynthesisMetadata 合成元数据
type SynthesisMetadata struct {
	// === 信息来源贡献度 ===
	InformationSources WideRecallInformationSources `json:"information_sources"` // 信息来源

	// === 质量评估 ===
	QualityAssessment QualityAssessment `json:"quality_assessment"` // 质量评估

	// === 合成说明 ===
	SynthesisNotes string `json:"synthesis_notes"` // 合成过程说明
}

// WideRecallInformationSources 宽召回信息来源
type WideRecallInformationSources struct {
	TimelineContribution  float64 `json:"timeline_contribution"`  // 时间线贡献度
	KnowledgeContribution float64 `json:"knowledge_contribution"` // 知识图谱贡献度
	VectorContribution    float64 `json:"vector_contribution"`    // 向量检索贡献度
}

// QualityAssessment 质量评估
type QualityAssessment struct {
	OverallQuality       float64  `json:"overall_quality"`       // 总体质量
	InformationConflicts []string `json:"information_conflicts"` // 信息冲突
	InformationGaps      []string `json:"information_gaps"`      // 信息缺口
}

// UserResponseSynthesis 用户响应合成结构
type UserResponseSynthesis struct {
	UserIntent string `json:"user_intent"` // 用户真实意图分析 + 筛选整合的相关信息
	Solution   string `json:"solution"`    // LLM提供的实用针对性解决方案
}
