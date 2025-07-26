package interfaces

import (
	"context"
	"time"
)

// ============================================================================
// 🧩 查询处理器接口 - 轻量级、可插拔、向后兼容
// ============================================================================

// QueryProcessor 查询处理器核心接口
// 设计原则：简单、稳定、可扩展
type QueryProcessor interface {
	// 处理查询
	Process(ctx context.Context, query string, options ProcessOptions) (*ProcessResult, error)

	// 组件信息
	Name() string
	Version() string
	Priority() int

	// 适用性检查
	IsApplicable(query string) bool

	// 配置管理
	Configure(config map[string]interface{}) error

	// 健康检查
	IsHealthy() bool
}

// ProcessOptions 处理选项
type ProcessOptions struct {
	// 上下文信息
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`

	// 处理偏好
	EnableOptimization bool `json:"enable_optimization"`
	SkipCache          bool `json:"skip_cache"`

	// 扩展参数
	Metadata map[string]interface{} `json:"metadata"`
}

// ProcessResult 处理结果
type ProcessResult struct {
	// 处理结果
	OriginalQuery  string `json:"original_query"`
	ProcessedQuery string `json:"processed_query"`

	// 处理信息
	ProcessorName  string        `json:"processor_name"`
	ProcessingTime time.Duration `json:"processing_time"`
	QualityScore   float64       `json:"quality_score"`

	// 变更记录
	Changes []ChangeRecord `json:"changes"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// ChangeRecord 变更记录
type ChangeRecord struct {
	Type     string `json:"type"`     // "enhance", "remove", "replace"
	Position int    `json:"position"` // 变更位置
	Original string `json:"original"` // 原始内容
	Modified string `json:"modified"` // 修改后内容
	Reason   string `json:"reason"`   // 变更原因
}

// ============================================================================
// 🔧 查询增强器接口 - 专门负责查询优化
// ============================================================================

// QueryEnhancer 查询增强器接口
type QueryEnhancer interface {
	// 增强查询
	Enhance(ctx context.Context, query string, context EnhanceContext) (string, error)

	// 组件信息
	Name() string
	Type() string // "noise_removal", "term_enhancement", "context_enrichment"

	// 适用性评估
	IsApplicable(query string) bool
	ApplicabilityScore(query string) float64

	// 配置管理
	UpdateConfig(config map[string]interface{}) error
}

// EnhanceContext 增强上下文
type EnhanceContext struct {
	// 历史信息
	RecentQueries   []string               `json:"recent_queries"`
	UserPreferences map[string]interface{} `json:"user_preferences"`

	// 技术上下文
	Domain    string   `json:"domain"`
	TechStack []string `json:"tech_stack"`

	// 会话信息
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
}

// ============================================================================
// 📊 质量评估器接口 - 负责评估处理质量
// ============================================================================

// QualityEvaluator 质量评估器接口
type QualityEvaluator interface {
	// 评估质量
	Evaluate(original, processed string, context EvaluateContext) (*QualityScore, error)

	// 评估器信息
	Name() string
	Weight() float64

	// 批量评估
	EvaluateBatch(pairs []QueryPair, context EvaluateContext) ([]*QualityScore, error)
}

// QueryPair 查询对
type QueryPair struct {
	Original  string `json:"original"`
	Processed string `json:"processed"`
}

// EvaluateContext 评估上下文
type EvaluateContext struct {
	// 评估标准
	Criteria []string `json:"criteria"`

	// 用户反馈
	UserFeedback *UserFeedback `json:"user_feedback"`

	// 历史数据
	HistoricalData map[string]interface{} `json:"historical_data"`
}

// QualityScore 质量分数
type QualityScore struct {
	// 综合评分
	Overall float64 `json:"overall"` // 0.0-1.0

	// 分项评分
	SemanticKeeping float64 `json:"semantic_keeping"` // 语义保持度
	Enhancement     float64 `json:"enhancement"`      // 增强效果
	Clarity         float64 `json:"clarity"`          // 清晰度
	Searchability   float64 `json:"searchability"`    // 可搜索性

	// 评估信息
	EvaluatorName string  `json:"evaluator_name"`
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`

	// 建议
	Suggestions []string `json:"suggestions"`
}

// ============================================================================
// 📈 反馈收集器接口 - 负责收集和处理反馈
// ============================================================================

// FeedbackCollector 反馈收集器接口
type FeedbackCollector interface {
	// 收集反馈
	CollectFeedback(feedback *UserFeedback) error

	// 获取统计信息
	GetStatistics(timeRange TimeRange) (*FeedbackStatistics, error)

	// 获取改进建议
	GetImprovementSuggestions() ([]*ImprovementSuggestion, error)
}

// UserFeedback 用户反馈
type UserFeedback struct {
	// 基础信息
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`

	// 查询信息
	OriginalQuery  string `json:"original_query"`
	ProcessedQuery string `json:"processed_query"`
	ProcessorName  string `json:"processor_name"`

	// 反馈内容
	Rating           float64 `json:"rating"`     // 1.0-5.0
	Usefulness       float64 `json:"usefulness"` // 0.0-1.0
	RetrievalSuccess bool    `json:"retrieval_success"`

	// 详细反馈
	Comments    string   `json:"comments"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// FeedbackStatistics 反馈统计
type FeedbackStatistics struct {
	// 基础统计
	TotalFeedbacks int     `json:"total_feedbacks"`
	AverageRating  float64 `json:"average_rating"`
	SuccessRate    float64 `json:"success_rate"`

	// 处理器性能
	ProcessorStats map[string]*ProcessorStat `json:"processor_stats"`

	// 趋势数据
	TrendData map[string]interface{} `json:"trend_data"`

	// 更新时间
	LastUpdated time.Time `json:"last_updated"`
}

// ProcessorStat 处理器统计
type ProcessorStat struct {
	Name             string  `json:"name"`
	UsageCount       int     `json:"usage_count"`
	AverageRating    float64 `json:"average_rating"`
	SuccessRate      float64 `json:"success_rate"`
	PerformanceScore float64 `json:"performance_score"`
}

// ImprovementSuggestion 改进建议
type ImprovementSuggestion struct {
	// 建议信息
	Type        string `json:"type"`     // "parameter_adjust", "strategy_change", "new_component"
	Priority    string `json:"priority"` // "high", "medium", "low"
	Description string `json:"description"`

	// 目标组件
	TargetComponent string `json:"target_component"`

	// 具体建议
	Recommendation map[string]interface{} `json:"recommendation"`

	// 预期影响
	ExpectedImpact string  `json:"expected_impact"`
	Confidence     float64 `json:"confidence"`
}
