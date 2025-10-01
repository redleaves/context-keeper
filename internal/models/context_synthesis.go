package models

import (
	"time"
)

// ContextSynthesisResult 上下文合成结果（包含评估）
type ContextSynthesisResult struct {
	// === 合成结果 ===
	UpdatedContext *UnifiedContextModel `json:"updated_context"` // 更新后的上下文

	// === 评估结果（内嵌在合成过程中）===
	ShouldUpdate     bool    `json:"should_update"`     // 是否需要更新
	UpdateConfidence float64 `json:"update_confidence"` // 更新置信度
	EvaluationReason string  `json:"evaluation_reason"` // 评估原因

	// === 变化分析 ===
	SemanticChanges []SemanticChange     `json:"semantic_changes"` // 语义变化
	InformationGaps []InformationGap     `json:"information_gaps"` // 信息缺口
	NewInformation  []NewInformationItem `json:"new_information"`  // 新增信息

	// === 更新策略 ===
	UpdateDimensions []string       `json:"update_dimensions"` // 需要更新的维度
	UpdateActions    []UpdateAction `json:"update_actions"`    // 具体更新动作

	// === 置信度因子 ===
	ConfidenceFactors []ConfidenceFactor `json:"confidence_factors"` // 置信度因子

	// === 合成元数据 ===
	SynthesisTime      time.Time          `json:"synthesis_time"`      // 合成时间
	InformationSources InformationSources `json:"information_sources"` // 信息来源贡献度
	SynthesisNotes     string             `json:"synthesis_notes"`     // 合成说明
}

// SemanticChange 语义变化
type SemanticChange struct {
	Dimension      string   `json:"dimension"`       // 变化维度：topic/project/code/conversation
	ChangeType     string   `json:"change_type"`     // 变化类型：shift/expand/refine/contradict
	OldSemantic    string   `json:"old_semantic"`    // 原语义
	NewSemantic    string   `json:"new_semantic"`    // 新语义
	ChangeStrength float64  `json:"change_strength"` // 变化强度 0-1
	Evidence       []string `json:"evidence"`        // 变化证据
}

// InformationGap 信息缺口
type InformationGap struct {
	Dimension            string   `json:"dimension"`               // 缺失维度
	MissingAspects       []string `json:"missing_aspects"`         // 缺失方面
	Importance           float64  `json:"importance"`              // 重要程度
	CanFillFromRetrieval bool     `json:"can_fill_from_retrieval"` // 能否从召回结果填充
}

// NewInformationItem 新增信息项
type NewInformationItem struct {
	Dimension   string  `json:"dimension"`   // 信息维度
	Content     string  `json:"content"`     // 信息内容
	Source      string  `json:"source"`      // 信息来源：query/timeline/knowledge/vector
	Reliability float64 `json:"reliability"` // 可靠性
	Relevance   float64 `json:"relevance"`   // 相关性
}

// UpdateAction 更新动作
type UpdateAction struct {
	ActionType        string `json:"action_type"`        // 动作类型：create/update/merge/refine
	TargetDimension   string `json:"target_dimension"`   // 目标维度
	ActionDescription string `json:"action_description"` // 具体动作描述
	Priority          int    `json:"priority"`           // 优先级 1-5
}

// ConfidenceFactor 置信度因子
type ConfidenceFactor struct {
	Factor      string  `json:"factor"`      // 影响置信度的因子
	Impact      float64 `json:"impact"`      // 正负影响值
	Description string  `json:"description"` // 因子说明
}

// InformationSources 信息来源贡献度
type InformationSources struct {
	TimelineContribution  float64 `json:"timeline_contribution"`  // 时间线贡献度
	KnowledgeContribution float64 `json:"knowledge_contribution"` // 知识图谱贡献度
	VectorContribution    float64 `json:"vector_contribution"`    // 向量检索贡献度
	ContextContribution   float64 `json:"context_contribution"`   // 现有上下文贡献度
}

// ParallelRetrievalResult 并行检索结果
type ParallelRetrievalResult struct {
	TimelineResults  []*TimelineEvent `json:"timeline_results"`
	KnowledgeResults []*KnowledgeNode `json:"knowledge_results"`
	VectorResults    []*VectorMatch   `json:"vector_results"`

	// 检索元数据
	TimelineCount      int   `json:"timeline_count"`
	KnowledgeCount     int   `json:"knowledge_count"`
	VectorCount        int   `json:"vector_count"`
	TotalRetrievalTime int64 `json:"total_retrieval_time_ms"`
}

// 注意：TimelineEvent现在使用unified_models.go中的统一定义
// 这里不再重复定义，直接使用统一模型

// 注意：KnowledgeNode 已在 programming_context.go 中定义，这里直接使用

// 注意：VectorMatch现在使用unified_models.go中的统一定义
// 这里不再重复定义，直接使用统一模型

// 注意：IntentAnalysisResult 已在 smart_analysis.go 中定义，这里直接使用
// 注意：Entity 已在 llm_driven_models.go 中定义，这里直接使用

// SearchQuery 搜索查询
type SearchQuery struct {
	QueryText string                 `json:"query_text"` // 查询文本
	QueryType string                 `json:"query_type"` // 查询类型：timeline/knowledge/vector
	Keywords  []string               `json:"keywords"`   // 关键词
	Filters   map[string]interface{} `json:"filters"`    // 过滤条件
	Priority  int                    `json:"priority"`   // 优先级
}

// 注意：RetrievalStrategy 已在 llm_driven_models.go 中定义，这里直接使用

// ContextUpdateRequest 上下文更新请求
type ContextUpdateRequest struct {
	SessionID   string                 `json:"session_id"`
	UserQuery   string                 `json:"user_query"`
	UserID      string                 `json:"user_id"`
	WorkspaceID string                 `json:"workspace_id"`
	QueryType   QueryType              `json:"query_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	StartTime   time.Time              `json:"start_time"` // 请求开始时间
}

// QueryType 查询类型
type QueryType string

const (
	QueryTypeGeneral   QueryType = "general"
	QueryTypeTechnical QueryType = "technical"
	QueryTypeProject   QueryType = "project"
	QueryTypeCode      QueryType = "code"
	QueryTypeReview    QueryType = "review"
)

// ContextUpdateResponse 上下文更新响应
type ContextUpdateResponse struct {
	Success         bool                   `json:"success"`
	UpdatedContext  *UnifiedContextModel   `json:"updated_context"`
	UpdateSummary   string                 `json:"update_summary"`
	ConfidenceLevel float64                `json:"confidence_level"`
	ProcessingTime  int64                  `json:"processing_time_ms"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ContextManager 上下文管理器接口
type ContextManager interface {
	// 获取上下文
	GetContext(sessionID string) (*UnifiedContextModel, error)

	// 更新上下文
	UpdateContext(req *ContextUpdateRequest) (*ContextUpdateResponse, error)

	// 清理上下文（会话结束时）
	CleanupContext(sessionID string) error

	// 并行宽召回
	ParallelWideRecall(queries []SearchQuery, userID string, workspaceID string) (*ParallelRetrievalResult, error)

	// LLM意图分析
	AnalyzeUserIntent(userQuery string) (*IntentAnalysisResult, error)

	// LLM上下文合成与评估
	SynthesizeAndEvaluateContext(
		userQuery string,
		currentContext *UnifiedContextModel,
		retrievalResults *ParallelRetrievalResult,
		intentAnalysis *IntentAnalysisResult,
	) (*ContextSynthesisResult, error)
}
