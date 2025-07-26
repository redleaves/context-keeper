package query_rewriter_v2

import (
	"context"
	"time"
)

// ============================================================================
// 🧩 乐高积木式接口定义 - Query Rewriter 2.0
// ============================================================================

// QueryRewriterComponent 所有可插拔组件的基础接口
type QueryRewriterComponent interface {
	// 组件标识
	Name() string
	Version() string

	// 生命周期管理
	Initialize(config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// 健康检查
	HealthCheck() ComponentHealth
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status     string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Message    string                 `json:"message"`
	Metrics    map[string]interface{} `json:"metrics"`
	LastUpdate time.Time              `json:"last_update"`
}

// ============================================================================
// 🎯 查询意图分析器接口
// ============================================================================

// QueryIntentAnalyzer 查询意图分析器 - 可插拔组件
type QueryIntentAnalyzer interface {
	QueryRewriterComponent

	// 分析查询意图
	AnalyzeIntent(ctx context.Context, query string, context QueryContext) (*QueryIntent, error)

	// 批量分析
	AnalyzeBatch(ctx context.Context, queries []string, context QueryContext) ([]*QueryIntent, error)

	// 学习用户查询模式
	LearnPattern(feedback *IntentAnalysisFeedback) error
}

// QueryIntent 查询意图
type QueryIntent struct {
	// 基础信息
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`

	// 意图分类
	Type       string  `json:"type"`       // "technical", "conceptual", "procedural", "debugging"
	Domain     string  `json:"domain"`     // "programming", "architecture", "database", "frontend"
	Complexity float64 `json:"complexity"` // 0.0-1.0

	// 关键信息
	Keywords []Keyword              `json:"keywords"`
	Entities []NamedEntity          `json:"entities"`
	Context  map[string]interface{} `json:"context"`

	// 置信度
	Confidence float64 `json:"confidence"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// Keyword 关键词信息
type Keyword struct {
	Term     string  `json:"term"`
	Weight   float64 `json:"weight"`
	Category string  `json:"category"` // "technical", "domain", "action", "object"
	Source   string  `json:"source"`   // 提取来源
}

// NamedEntity 命名实体
type NamedEntity struct {
	Text     string  `json:"text"`
	Type     string  `json:"type"` // "PERSON", "ORG", "TECH", "TOOL"
	Score    float64 `json:"score"`
	Position [2]int  `json:"position"` // [start, end]
}

// ============================================================================
// 🧠 智能决策中心接口
// ============================================================================

// IntelligentDecisionCenter 智能决策中心 - 核心控制器
type IntelligentDecisionCenter interface {
	QueryRewriterComponent

	// 决策处理
	MakeDecision(ctx context.Context, intent *QueryIntent, context QueryContext) (*RewriteDecision, error)

	// 注册组件
	RegisterComponent(component QueryRewriterComponent) error
	UnregisterComponent(name string) error

	// 策略管理
	RegisterStrategy(strategy RewriteStrategy) error
	GetStrategy(name string) (RewriteStrategy, error)
	ListStrategies() []string

	// 配置管理
	UpdateConfig(config map[string]interface{}) error
	GetConfig() map[string]interface{}
}

// RewriteDecision 改写决策
type RewriteDecision struct {
	// 决策基础信息
	Intent     *QueryIntent `json:"intent"`
	Timestamp  time.Time    `json:"timestamp"`
	DecisionID string       `json:"decision_id"`

	// 执行计划
	Plan *RewritePlan `json:"plan"`

	// 选中的策略
	Strategies []string `json:"strategies"`

	// 决策元数据
	Confidence float64                `json:"confidence"`
	Reasoning  string                 `json:"reasoning"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// RewritePlan 改写计划
type RewritePlan struct {
	PlanID     string          `json:"plan_id"`
	Phases     []*RewritePhase `json:"phases"`
	Parallel   bool            `json:"parallel"` // 是否并行执行
	Timeout    time.Duration   `json:"timeout"`
	MaxRetries int             `json:"max_retries"`
}

// RewritePhase 改写阶段
type RewritePhase struct {
	Name         string                 `json:"name"`
	Strategy     string                 `json:"strategy"`
	Priority     float64                `json:"priority"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies"` // 依赖的前置阶段
}

// ============================================================================
// 🎮 策略选择器接口
// ============================================================================

// StrategySelector 策略选择器 - 可插拔组件
type StrategySelector interface {
	QueryRewriterComponent

	// 选择策略
	SelectStrategies(ctx context.Context, intent *QueryIntent, context QueryContext) ([]StrategySelection, error)

	// 评估策略适用性
	EvaluateApplicability(strategy string, intent *QueryIntent) float64

	// 更新策略权重
	UpdateStrategyWeights(feedback *StrategyFeedback) error
}

// StrategySelection 策略选择结果
type StrategySelection struct {
	StrategyName  string                 `json:"strategy_name"`
	Applicability float64                `json:"applicability"`
	Priority      float64                `json:"priority"`
	Config        map[string]interface{} `json:"config"`
	Reasoning     string                 `json:"reasoning"`
}

// ============================================================================
// 🌐 上下文感知层接口
// ============================================================================

// ContextAwareLayer 上下文感知层 - 可插拔组件
type ContextAwareLayer interface {
	QueryRewriterComponent

	// 构建查询上下文
	BuildContext(ctx context.Context, query string, sessionID string) (*QueryContext, error)

	// 更新上下文
	UpdateContext(ctx context.Context, queryContext *QueryContext, result *RewriteResult) error

	// 获取相关上下文
	GetRelevantContext(ctx context.Context, intent *QueryIntent) (*RelevantContext, error)
}

// QueryContext 查询上下文
type QueryContext struct {
	// 会话信息
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`

	// 历史信息
	RecentQueries []string       `json:"recent_queries"`
	QueryHistory  []*QueryIntent `json:"query_history"`

	// 用户偏好
	UserPreferences map[string]interface{} `json:"user_preferences"`

	// 领域信息
	Domain           string                 `json:"domain"`
	TechnicalContext map[string]interface{} `json:"technical_context"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// RelevantContext 相关上下文
type RelevantContext struct {
	HistoricalPatterns []QueryPattern         `json:"historical_patterns"`
	DomainKnowledge    []DomainConcept        `json:"domain_knowledge"`
	SemanticRelations  []SemanticRelation     `json:"semantic_relations"`
	UserPreferences    map[string]interface{} `json:"user_preferences"`
}

// ============================================================================
// 🔧 改写策略接口
// ============================================================================

// RewriteStrategy 改写策略 - 可插拔组件
type RewriteStrategy interface {
	QueryRewriterComponent

	// 策略信息
	StrategyType() string
	Applicability(intent *QueryIntent) float64

	// 执行改写
	Rewrite(ctx context.Context, query string, intent *QueryIntent, context QueryContext) (*RewriteCandidate, error)

	// 批量处理
	RewriteBatch(ctx context.Context, queries []string, intents []*QueryIntent, context QueryContext) ([]*RewriteCandidate, error)

	// 学习反馈
	Learn(feedback *StrategyFeedback) error

	// 配置更新
	UpdateConfig(config map[string]interface{}) error
}

// RewriteCandidate 改写候选
type RewriteCandidate struct {
	// 基础信息
	OriginalQuery  string    `json:"original_query"`
	RewrittenQuery string    `json:"rewritten_query"`
	Strategy       string    `json:"strategy"`
	Timestamp      time.Time `json:"timestamp"`

	// 改写详情
	Changes      []QueryChange `json:"changes"`
	QualityScore float64       `json:"quality_score"`
	Confidence   float64       `json:"confidence"`

	// 性能信息
	ProcessingTime time.Duration `json:"processing_time"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// QueryChange 查询变更
type QueryChange struct {
	Type      string `json:"type"`     // "add", "remove", "replace", "enhance"
	Position  [2]int `json:"position"` // [start, end]
	Original  string `json:"original"`
	Modified  string `json:"modified"`
	Reasoning string `json:"reasoning"`
}

// ============================================================================
// 📊 质量评估引擎接口
// ============================================================================

// QualityAssessmentEngine 质量评估引擎 - 可插拔组件
type QualityAssessmentEngine interface {
	QueryRewriterComponent

	// 评估单个候选
	Evaluate(ctx context.Context, candidate *RewriteCandidate, intent *QueryIntent, context QueryContext) (*QualityAssessment, error)

	// 批量评估
	EvaluateBatch(ctx context.Context, candidates []*RewriteCandidate, intent *QueryIntent, context QueryContext) ([]*QualityAssessment, error)

	// 注册评估器
	RegisterEvaluator(evaluator QualityEvaluator) error

	// 获取评估器列表
	ListEvaluators() []string
}

// QualityEvaluator 质量评估器 - 可插拔子组件
type QualityEvaluator interface {
	// 评估器信息
	Name() string
	Weight() float64

	// 执行评估
	Evaluate(original, rewritten string, intent *QueryIntent, context QueryContext) (*EvaluationScore, error)

	// 配置更新
	UpdateConfig(config map[string]interface{}) error
}

// ============================================================================
// 📈 反馈学习层接口
// ============================================================================

// FeedbackLearner 反馈学习器 - 可插拔组件
type FeedbackLearner interface {
	QueryRewriterComponent

	// 处理反馈
	ProcessFeedback(ctx context.Context, feedback *RewriteFeedback) error

	// 批量处理反馈
	ProcessBatchFeedback(ctx context.Context, feedbacks []*RewriteFeedback) error

	// 获取学习统计
	GetLearningStats() *LearningStats

	// 导出学习模型
	ExportModel() ([]byte, error)

	// 导入学习模型
	ImportModel(data []byte) error
}

// ============================================================================
// 📋 数据结构定义
// ============================================================================

// QualityAssessment 质量评估结果
type QualityAssessment struct {
	CandidateID    string                      `json:"candidate_id"`
	OverallScore   float64                     `json:"overall_score"`
	DetailedScores map[string]*EvaluationScore `json:"detailed_scores"`
	Recommendation string                      `json:"recommendation"`
	Confidence     float64                     `json:"confidence"`
	ProcessingTime time.Duration               `json:"processing_time"`
}

// EvaluationScore 评估分数
type EvaluationScore struct {
	Score     float64                `json:"score"`     // 0.0-1.0
	Weight    float64                `json:"weight"`    // 权重
	Reasoning string                 `json:"reasoning"` // 评估理由
	Metadata  map[string]interface{} `json:"metadata"`
}

// RewriteResult 最终改写结果
type RewriteResult struct {
	// 基础信息
	OriginalQuery string           `json:"original_query"`
	FinalQuery    string           `json:"final_query"`
	Intent        *QueryIntent     `json:"intent"`
	Decision      *RewriteDecision `json:"decision"`

	// 候选和评估
	Candidates        []*RewriteCandidate `json:"candidates"`
	SelectedCandidate *RewriteCandidate   `json:"selected_candidate"`
	QualityAssessment *QualityAssessment  `json:"quality_assessment"`

	// 性能信息
	TotalProcessingTime time.Duration            `json:"total_processing_time"`
	ComponentTiming     map[string]time.Duration `json:"component_timing"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// RewriteFeedback 改写反馈
type RewriteFeedback struct {
	// 基础信息
	ResultID  string    `json:"result_id"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`

	// 反馈内容
	UserRating       float64 `json:"user_rating"`       // 1.0-5.0
	RetrievalSuccess bool    `json:"retrieval_success"` // 检索是否成功
	RelevanceScore   float64 `json:"relevance_score"`   // 相关性评分

	// 详细反馈
	Comments    string          `json:"comments"`
	Issues      []FeedbackIssue `json:"issues"`
	Suggestions []string        `json:"suggestions"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// 其他辅助数据结构
type QueryPattern struct {
	Pattern   string  `json:"pattern"`
	Frequency int     `json:"frequency"`
	Success   float64 `json:"success"`
}

type DomainConcept struct {
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Relations []string `json:"relations"`
	Weight    float64  `json:"weight"`
}

type SemanticRelation struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Relation string  `json:"relation"`
	Strength float64 `json:"strength"`
}

type StrategyFeedback struct {
	StrategyName string                 `json:"strategy_name"`
	Performance  float64                `json:"performance"`
	Issues       []string               `json:"issues"`
	Improvements []string               `json:"improvements"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type IntentAnalysisFeedback struct {
	PredictedIntent *QueryIntent           `json:"predicted_intent"`
	ActualIntent    *QueryIntent           `json:"actual_intent"`
	Accuracy        float64                `json:"accuracy"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type FeedbackIssue struct {
	Type        string `json:"type"`     // "semantic_loss", "over_expansion", "noise_added"
	Severity    string `json:"severity"` // "low", "medium", "high"
	Description string `json:"description"`
}

type LearningStats struct {
	TotalFeedbacks      int                    `json:"total_feedbacks"`
	AverageRating       float64                `json:"average_rating"`
	SuccessRate         float64                `json:"success_rate"`
	StrategyPerformance map[string]float64     `json:"strategy_performance"`
	RecentTrends        map[string]interface{} `json:"recent_trends"`
	LastUpdated         time.Time              `json:"last_updated"`
}
