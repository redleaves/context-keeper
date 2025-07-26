package interfaces

import (
	"context"
	"time"
)

// ============================================================================
// 🎯 Agentic RAG 核心架构接口 - 严格按照流程图设计
// ============================================================================

// ============================================================================
// 🔍 A → B: 查询意图分析器
// ============================================================================

// QueryIntentAnalyzer 查询意图分析器 - 流程图中的组件B
type QueryIntentAnalyzer interface {
	// 核心功能：分析用户查询意图
	AnalyzeIntent(ctx context.Context, query string) (*QueryIntent, error)

	// 组件管理
	Name() string
	Version() string
	IsEnabled() bool

	// 配置管理
	Configure(config map[string]interface{}) error
	HealthCheck() error
}

// QueryIntent 查询意图结构 (融入v2优秀设计)
type QueryIntent struct {
	// 基础信息
	OriginalQuery string    `json:"original_query"`
	Timestamp     time.Time `json:"timestamp"`

	// 意图分类
	IntentType string  `json:"intent_type"` // "technical", "conceptual", "procedural", "debugging"
	Domain     string  `json:"domain"`      // "programming", "architecture", "database", "frontend"
	Complexity float64 `json:"complexity"`  // 0.0-1.0 复杂度评分

	// 关键信息提取 (融入v2设计)
	Keywords  []KeywordInfo `json:"keywords"`   // 详细关键词信息 (v2优势)
	Entities  []EntityInfo  `json:"entities"`   // 详细实体信息 (v2优势)
	TechStack []string      `json:"tech_stack"` // 技术栈识别

	// 置信度和元数据
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// KeywordInfo 关键词详细信息 (融入v2设计)
type KeywordInfo struct {
	Term     string  `json:"term"`
	Weight   float64 `json:"weight"`   // v2优势：权重信息
	Category string  `json:"category"` // v2优势：分类 "technical", "domain", "action", "object"
	Source   string  `json:"source"`   // v2优势：提取来源
}

// EntityInfo 实体详细信息 (融入v2设计)
type EntityInfo struct {
	Text     string  `json:"text"`
	Type     string  `json:"type"`     // v2优势：实体类型 "PERSON", "ORG", "TECH", "TOOL"
	Score    float64 `json:"score"`    // v2优势：置信度评分
	Position [2]int  `json:"position"` // v2优势：文本位置 [start, end]
}

// ============================================================================
// 🧠 B → C: 智能决策中心
// ============================================================================

// IntelligentDecisionCenter 智能决策中心 - 流程图中的组件C（核心控制器）
type IntelligentDecisionCenter interface {
	// 核心功能：基于意图做出处理决策
	MakeDecision(ctx context.Context, intent *QueryIntent) (*ProcessingDecision, error)

	// 组件协调
	RegisterTaskPlanner(planner TaskPlanner) error
	RegisterStrategySelector(selector StrategySelector) error
	RegisterContextLayer(layer ContextAwareLayer) error

	// 决策管理
	GetDecisionHistory() []*ProcessingDecision
	OptimizeDecisionStrategy() error

	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ProcessingDecision 处理决策
type ProcessingDecision struct {
	// 决策信息
	DecisionID string       `json:"decision_id"`
	Intent     *QueryIntent `json:"intent"`
	Timestamp  time.Time    `json:"timestamp"`

	// 处理计划
	TaskPlan           *TaskPlan              `json:"task_plan"`
	SelectedStrategies []string               `json:"selected_strategies"`
	ContextInfo        map[string]interface{} `json:"context_info"`

	// 决策元数据
	DecisionReasoning string                 `json:"decision_reasoning"`
	Confidence        float64                `json:"confidence"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 📋 C → D: 任务规划模块
// ============================================================================

// TaskPlanner 任务规划模块 - 流程图中的组件D
type TaskPlanner interface {
	// 核心功能：根据意图规划处理任务
	PlanTasks(ctx context.Context, intent *QueryIntent) (*TaskPlan, error)

	// 计划管理
	ValidatePlan(plan *TaskPlan) error
	OptimizePlan(plan *TaskPlan) (*TaskPlan, error)

	// 组件信息
	Name() string
	GetCapabilities() []string
}

// TaskPlan 任务计划
type TaskPlan struct {
	// 计划信息
	PlanID       string       `json:"plan_id"`
	TargetIntent *QueryIntent `json:"target_intent"`
	CreatedAt    time.Time    `json:"created_at"`

	// 任务列表
	Tasks          []*Task  `json:"tasks"`
	ExecutionOrder []string `json:"execution_order"`

	// 执行配置
	ParallelExecution bool `json:"parallel_execution"`
	MaxRetries        int  `json:"max_retries"`
	TimeoutSeconds    int  `json:"timeout_seconds"`

	// 计划元数据
	Priority int                    `json:"priority"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Task 单个任务
type Task struct {
	TaskID          string                 `json:"task_id"`
	Type            string                 `json:"type"` // "enhance", "filter", "adapt"
	TargetComponent string                 `json:"target_component"`
	Parameters      map[string]interface{} `json:"parameters"`
	Dependencies    []string               `json:"dependencies"`
	Priority        int                    `json:"priority"`
}

// ============================================================================
// 🎮 C → E: 策略选择器
// ============================================================================

// StrategySelector 策略选择器 - 流程图中的组件E
type StrategySelector interface {
	// 核心功能：选择最适合的处理策略
	SelectStrategies(ctx context.Context, intent *QueryIntent, plan *TaskPlan) (*StrategySelection, error)

	// 策略管理
	RegisterStrategy(strategy ProcessingStrategy) error
	GetAvailableStrategies() []string
	EvaluateStrategyFitness(strategy string, intent *QueryIntent) float64

	// 学习优化
	UpdateStrategyPerformance(strategyName string, performance float64) error
	GetStrategyStatistics() map[string]*StrategyStats

	// 组件信息
	Name() string
}

// StrategySelection 策略选择结果
type StrategySelection struct {
	// 选择信息
	SelectionID string       `json:"selection_id"`
	Intent      *QueryIntent `json:"intent"`
	Timestamp   time.Time    `json:"timestamp"`

	// 选中的策略
	SelectedStrategies []*SelectedStrategy `json:"selected_strategies"`

	// 选择理由
	SelectionReasoning string  `json:"selection_reasoning"`
	OverallConfidence  float64 `json:"overall_confidence"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// SelectedStrategy 选中的策略
type SelectedStrategy struct {
	StrategyName   string                 `json:"strategy_name"`
	TargetTask     string                 `json:"target_task"`
	Priority       float64                `json:"priority"`
	ExpectedImpact float64                `json:"expected_impact"`
	Configuration  map[string]interface{} `json:"configuration"`
}

// ============================================================================
// 🌐 C → F: 上下文感知层
// ============================================================================

// ContextAwareLayer 上下文感知层 - 流程图中的组件F
type ContextAwareLayer interface {
	// 核心功能：构建和管理处理上下文
	BuildContext(ctx context.Context, intent *QueryIntent) (*ProcessingContext, error)

	// 上下文管理
	EnrichContext(context *ProcessingContext, additionalInfo map[string]interface{}) error
	GetRelevantHistory(sessionID string, limit int) ([]*QueryIntent, error)
	UpdateUserPreferences(userID string, preferences map[string]interface{}) error

	// 上下文分析
	AnalyzeContextRelevance(context *ProcessingContext, intent *QueryIntent) float64
	ExtractContextPatterns(sessionID string) (*ContextPatterns, error)

	// 组件信息
	Name() string
}

// ProcessingContext 处理上下文
type ProcessingContext struct {
	// 基础信息
	ContextID string    `json:"context_id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`

	// 历史信息
	RecentQueries   []*QueryIntent         `json:"recent_queries"`
	UserPreferences map[string]interface{} `json:"user_preferences"`
	SessionPatterns *ContextPatterns       `json:"session_patterns"`

	// 技术上下文
	TechnicalDomain  string                 `json:"technical_domain"`
	CurrentTechStack []string               `json:"current_tech_stack"`
	ProjectContext   map[string]interface{} `json:"project_context"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// ContextPatterns 上下文模式
type ContextPatterns struct {
	FrequentTopics    []string `json:"frequent_topics"`
	PreferredApproach string   `json:"preferred_approach"`
	TechnicalLevel    float64  `json:"technical_level"`
	InteractionStyle  string   `json:"interaction_style"`
	PatternConfidence float64  `json:"pattern_confidence"`
}

// ============================================================================
// ⚡ D,E,F → G: 多策略并行处理器
// ============================================================================

// MultiStrategyProcessor 多策略并行处理器 - 流程图中的组件G（关键协调器）
type MultiStrategyProcessor interface {
	// 核心功能：并行执行多种处理策略
	ProcessParallel(ctx context.Context, decision *ProcessingDecision, context *ProcessingContext) (*ProcessingResults, error)

	// 处理器管理
	RegisterEnhancer(enhancer SemanticEnhancer) error
	RegisterFilter(filter NoiseFilter) error
	RegisterAdapter(adapter DomainAdapter) error

	// 执行控制
	SetConcurrency(level int) error
	SetTimeout(duration time.Duration) error
	GetProcessingStatistics() *ProcessingStatistics
}

// ProcessingResults 处理结果集
type ProcessingResults struct {
	// 结果信息
	ResultsID      string       `json:"results_id"`
	OriginalIntent *QueryIntent `json:"original_intent"`
	ProcessedAt    time.Time    `json:"processed_at"`

	// 各组件处理结果
	EnhancementResults []*EnhancementResult `json:"enhancement_results"`
	FilterResults      []*FilterResult      `json:"filter_results"`
	AdaptationResults  []*AdaptationResult  `json:"adaptation_results"`

	// 综合信息
	ProcessingTime time.Duration `json:"processing_time"`
	OverallSuccess bool          `json:"overall_success"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 🎨 G → H: 语义增强器
// ============================================================================

// SemanticEnhancer 语义增强器 - 流程图中的组件H
type SemanticEnhancer interface {
	// 核心功能：增强查询的语义表达
	EnhanceSemantics(ctx context.Context, query string, context *ProcessingContext) (*EnhancementResult, error)

	// 增强策略
	GetEnhancementStrategies() []string
	ApplyEnhancementStrategy(strategy string, query string) (string, error)

	// 组件信息
	Name() string
	GetCapabilities() []string
}

// EnhancementResult 增强结果
type EnhancementResult struct {
	OriginalQuery    string                 `json:"original_query"`
	EnhancedQuery    string                 `json:"enhanced_query"`
	EnhancementType  string                 `json:"enhancement_type"`
	ImprovementScore float64                `json:"improvement_score"`
	AppliedChanges   []string               `json:"applied_changes"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 🧹 G → I: 噪声智能过滤器
// ============================================================================

// NoiseFilter 噪声智能过滤器 - 流程图中的组件I
type NoiseFilter interface {
	// 核心功能：智能过滤查询中的噪声
	FilterNoise(ctx context.Context, query string, context *ProcessingContext) (*FilterResult, error)

	// 过滤策略
	GetFilterStrategies() []string
	ApplyFilterStrategy(strategy string, query string) (string, error)
	DetectNoisePatterns(query string) []string

	// 学习能力
	LearnFromFeedback(original, filtered string, isGoodFilter bool) error
}

// FilterResult 过滤结果
type FilterResult struct {
	OriginalQuery    string                 `json:"original_query"`
	FilteredQuery    string                 `json:"filtered_query"`
	RemovedElements  []string               `json:"removed_elements"`
	FilterConfidence float64                `json:"filter_confidence"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 🎯 G → J: 领域自适应器
// ============================================================================

// DomainAdapter 领域自适应器 - 流程图中的组件J
type DomainAdapter interface {
	// 核心功能：根据技术领域自适应查询
	AdaptToDomain(ctx context.Context, query string, domain string, context *ProcessingContext) (*AdaptationResult, error)

	// 领域管理
	GetSupportedDomains() []string
	RegisterDomainKnowledge(domain string, knowledge *DomainKnowledge) error
	UpdateDomainContext(domain string, context map[string]interface{}) error

	// 适应性评估
	EvaluateDomainFit(query string, domain string) float64
}

// AdaptationResult 适应结果
type AdaptationResult struct {
	OriginalQuery    string                 `json:"original_query"`
	AdaptedQuery     string                 `json:"adapted_query"`
	TargetDomain     string                 `json:"target_domain"`
	AdaptationScore  float64                `json:"adaptation_score"`
	DomainTermsAdded []string               `json:"domain_terms_added"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// DomainKnowledge 领域知识
type DomainKnowledge struct {
	Domain         string                 `json:"domain"`
	TechnicalTerms []string               `json:"technical_terms"`
	CommonPatterns []string               `json:"common_patterns"`
	BestPractices  []string               `json:"best_practices"`
	RelatedDomains []string               `json:"related_domains"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 🏭 H,I,J → K: 候选查询生成器
// ============================================================================

// CandidateQueryGenerator 候选查询生成器 - 流程图中的组件K
type CandidateQueryGenerator interface {
	// 核心功能：基于处理结果生成候选查询
	GenerateCandidates(ctx context.Context, results *ProcessingResults) (*QueryCandidates, error)

	// 生成策略
	GetGenerationStrategies() []string
	SetGenerationRules(rules []*GenerationRule) error

	// 候选管理
	RankCandidates(candidates []*QueryCandidate) ([]*QueryCandidate, error)
	FilterCandidates(candidates []*QueryCandidate, criteria *FilterCriteria) ([]*QueryCandidate, error)
}

// QueryCandidates 查询候选集
type QueryCandidates struct {
	// 基础信息
	CandidatesID  string    `json:"candidates_id"`
	OriginalQuery string    `json:"original_query"`
	GeneratedAt   time.Time `json:"generated_at"`

	// 候选列表
	Candidates []*QueryCandidate `json:"candidates"`

	// 生成信息
	GenerationTime  time.Duration `json:"generation_time"`
	TotalCandidates int           `json:"total_candidates"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// QueryCandidate 查询候选
type QueryCandidate struct {
	CandidateID      string                 `json:"candidate_id"`
	Query            string                 `json:"query"`
	Score            float64                `json:"score"`
	GenerationSource string                 `json:"generation_source"`
	Transformations  []string               `json:"transformations"`
	ExpectedQuality  float64                `json:"expected_quality"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 📊 K → L: 质量评估引擎
// ============================================================================

// QualityAssessmentEngine 质量评估引擎 - 流程图中的组件L
type QualityAssessmentEngine interface {
	// 核心功能：评估候选查询质量
	AssessQuality(ctx context.Context, candidates *QueryCandidates, intent *QueryIntent) (*QualityAssessment, error)

	// 评估器管理
	RegisterEvaluator(evaluator QualityEvaluator) error
	GetAvailableEvaluators() []string
	SetEvaluationCriteria(criteria []*EvaluationCriterion) error

	// 质量标准
	DefineQualityMetrics(metrics []*QualityMetric) error
	GetQualityBenchmarks() map[string]float64
}

// QualityAssessment 质量评估
type QualityAssessment struct {
	// 评估信息
	AssessmentID string    `json:"assessment_id"`
	EvaluatedAt  time.Time `json:"evaluated_at"`

	// 候选评估
	CandidateScores []*CandidateQualityScore `json:"candidate_scores"`

	// 综合评估
	OverallMetrics *OverallQualityMetrics `json:"overall_metrics"`
	Recommendation string                 `json:"recommendation"`

	// 评估元数据
	EvaluationTime time.Duration          `json:"evaluation_time"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CandidateQualityScore 候选质量分数
type CandidateQualityScore struct {
	CandidateID         string             `json:"candidate_id"`
	OverallScore        float64            `json:"overall_score"`
	DetailedScores      map[string]float64 `json:"detailed_scores"`
	QualityRanking      int                `json:"quality_ranking"`
	StrengthsWeaknesses *QualityAnalysis   `json:"strengths_weaknesses"`
}

// ============================================================================
// 🎯 L → M: 最优查询选择器
// ============================================================================

// OptimalQuerySelector 最优查询选择器 - 流程图中的组件M
type OptimalQuerySelector interface {
	// 核心功能：选择最优查询
	SelectOptimalQuery(ctx context.Context, assessment *QualityAssessment) (*OptimalSelection, error)

	// 选择策略
	SetSelectionStrategy(strategy SelectionStrategy) error
	GetSelectionStrategies() []string

	// 选择优化
	OptimizeSelection(feedback []*SelectionFeedback) error
	GetSelectionStatistics() *SelectionStatistics
}

// OptimalSelection 最优选择结果
type OptimalSelection struct {
	// 选择信息
	SelectionID string    `json:"selection_id"`
	SelectedAt  time.Time `json:"selected_at"`

	// 选中的查询
	OptimalQuery    *QueryCandidate `json:"optimal_query"`
	SelectionReason string          `json:"selection_reason"`
	Confidence      float64         `json:"confidence"`

	// 备选查询
	AlternativeQueries []*QueryCandidate `json:"alternative_queries"`

	// 选择元数据
	SelectionTime time.Duration          `json:"selection_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ============================================================================
// 📈 M → N → O → C: 反馈学习闭环
// ============================================================================

// FeedbackLearningLayer 反馈学习层 - 流程图中的组件N
type FeedbackLearningLayer interface {
	// 核心功能：收集和处理反馈
	CollectFeedback(ctx context.Context, selection *OptimalSelection, feedback *QueryFeedback) error

	// 学习处理
	ProcessFeedbackBatch(feedbacks []*QueryFeedback) (*LearningUpdate, error)
	GenerateLearningInsights() (*LearningInsights, error)

	// 反馈分析
	AnalyzeFeedbackPatterns() (*FeedbackPatterns, error)
	GetLearningProgress() *LearningProgress
}

// StrategyOptimizer 策略优化器 - 流程图中的组件O
type StrategyOptimizer interface {
	// 核心功能：基于学习结果优化策略
	OptimizeStrategies(ctx context.Context, insights *LearningInsights) (*OptimizationResult, error)

	// 优化管理
	GetOptimizationHistory() []*OptimizationResult
	ApplyOptimizations(optimizations *OptimizationResult) error
	RollbackOptimizations(optimizationID string) error

	// 效果评估
	EvaluateOptimizationImpact(optimizationID string) (*ImpactAssessment, error)
}

// ============================================================================
// 📋 辅助数据结构
// ============================================================================

// ProcessingStrategy 处理策略接口
type ProcessingStrategy interface {
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Name() string
	Type() string
	Priority() int
}

// StrategyStats 策略统计
type StrategyStats struct {
	UsageCount   int       `json:"usage_count"`
	AverageScore float64   `json:"average_score"`
	SuccessRate  float64   `json:"success_rate"`
	LastUsed     time.Time `json:"last_used"`
}

// ProcessingStatistics 处理统计
type ProcessingStatistics struct {
	TotalProcessed int                    `json:"total_processed"`
	AverageTime    time.Duration          `json:"average_time"`
	SuccessRate    float64                `json:"success_rate"`
	ComponentStats map[string]interface{} `json:"component_stats"`
}

// GenerationRule 生成规则
type GenerationRule struct {
	RuleID     string                 `json:"rule_id"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"`
	Priority   int                    `json:"priority"`
	Parameters map[string]interface{} `json:"parameters"`
}

// FilterCriteria 过滤标准
type FilterCriteria struct {
	MinScore        float64  `json:"min_score"`
	MaxCandidates   int      `json:"max_candidates"`
	RequiredFields  []string `json:"required_fields"`
	ExcludePatterns []string `json:"exclude_patterns"`
}

// EvaluationCriterion 评估标准
type EvaluationCriterion struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
	MetricType  string  `json:"metric_type"`
}

// QualityMetric 质量指标
type QualityMetric struct {
	MetricName  string  `json:"metric_name"`
	TargetValue float64 `json:"target_value"`
	Tolerance   float64 `json:"tolerance"`
	Description string  `json:"description"`
}

// OverallQualityMetrics 整体质量指标
type OverallQualityMetrics struct {
	SemanticConsistency float64 `json:"semantic_consistency"`
	InformationRichness float64 `json:"information_richness"`
	QueryClarity        float64 `json:"query_clarity"`
	SearchOptimization  float64 `json:"search_optimization"`
	DomainRelevance     float64 `json:"domain_relevance"`
}

// QualityAnalysis 质量分析
type QualityAnalysis struct {
	Strengths    []string `json:"strengths"`
	Weaknesses   []string `json:"weaknesses"`
	Improvements []string `json:"improvements"`
	RiskFactors  []string `json:"risk_factors"`
}

// SelectionStrategy 选择策略
type SelectionStrategy interface {
	SelectBest(candidates []*CandidateQualityScore) (*CandidateQualityScore, error)
	Name() string
	Description() string
}

// SelectionFeedback 选择反馈
type SelectionFeedback struct {
	SelectionID       string    `json:"selection_id"`
	UserSatisfaction  float64   `json:"user_satisfaction"`
	ActualPerformance float64   `json:"actual_performance"`
	Issues            []string  `json:"issues"`
	Timestamp         time.Time `json:"timestamp"`
}

// SelectionStatistics 选择统计
type SelectionStatistics struct {
	TotalSelections  int                    `json:"total_selections"`
	AverageScore     float64                `json:"average_score"`
	UserSatisfaction float64                `json:"user_satisfaction"`
	StrategyStats    map[string]interface{} `json:"strategy_stats"`
}

// QueryFeedback 查询反馈 (融入v2优秀设计)
type QueryFeedback struct {
	FeedbackID       string          `json:"feedback_id"`
	OriginalQuery    string          `json:"original_query"`
	OptimalQuery     string          `json:"optimal_query"`
	UserRating       float64         `json:"user_rating"`
	RetrievalSuccess bool            `json:"retrieval_success"`
	RelevanceScore   float64         `json:"relevance_score"` // v2优势：相关性评分
	UserComments     string          `json:"user_comments"`
	Issues           []FeedbackIssue `json:"issues"`      // v2优势：结构化问题列表
	Suggestions      []string        `json:"suggestions"` // v2优势：改进建议
	Timestamp        time.Time       `json:"timestamp"`
	SessionID        string          `json:"session_id"`
	UserID           string          `json:"user_id"`
}

// FeedbackIssue 反馈问题 (来自v2设计)
type FeedbackIssue struct {
	Type        string `json:"type"`        // "semantic_loss", "over_expansion", "noise_added"
	Severity    string `json:"severity"`    // "low", "medium", "high"
	Description string `json:"description"` // 问题描述
}

// LearningUpdate 学习更新
type LearningUpdate struct {
	UpdateID          string                 `json:"update_id"`
	UpdatedComponents []string               `json:"updated_components"`
	ImprovementScore  float64                `json:"improvement_score"`
	UpdateDescription string                 `json:"update_description"`
	AppliedAt         time.Time              `json:"applied_at"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// LearningInsights 学习洞察
type LearningInsights struct {
	InsightID                 string             `json:"insight_id"`
	KeyFindings               []string           `json:"key_findings"`
	PerformanceTrends         map[string]float64 `json:"performance_trends"`
	OptimizationOpportunities []string           `json:"optimization_opportunities"`
	RiskAreas                 []string           `json:"risk_areas"`
	Recommendations           []string           `json:"recommendations"`
	GeneratedAt               time.Time          `json:"generated_at"`
}

// FeedbackPatterns 反馈模式
type FeedbackPatterns struct {
	CommonIssues    []string               `json:"common_issues"`
	SuccessFactors  []string               `json:"success_factors"`
	UserPreferences map[string]interface{} `json:"user_preferences"`
	SeasonalTrends  map[string]float64     `json:"seasonal_trends"`
}

// LearningProgress 学习进度
type LearningProgress struct {
	TotalFeedbacks     int               `json:"total_feedbacks"`
	LearningRate       float64           `json:"learning_rate"`
	ModelAccuracy      float64           `json:"model_accuracy"`
	RecentUpdates      []*LearningUpdate `json:"recent_updates"`
	NextUpdateSchedule time.Time         `json:"next_update_schedule"`
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	OptimizationID          string                 `json:"optimization_id"`
	TargetComponents        []string               `json:"target_components"`
	OptimizationDescription string                 `json:"optimization_description"`
	ParameterChanges        map[string]interface{} `json:"parameter_changes"`
	ExpectedImpact          string                 `json:"expected_impact"`
	AppliedAt               time.Time              `json:"applied_at"`
	Status                  string                 `json:"status"`
}

// ImpactAssessment 影响评估
type ImpactAssessment struct {
	OptimizationID         string    `json:"optimization_id"`
	ActualImpact           string    `json:"actual_impact"`
	PerformanceChange      float64   `json:"performance_change"`
	UserSatisfactionChange float64   `json:"user_satisfaction_change"`
	UnexpectedEffects      []string  `json:"unexpected_effects"`
	AssessmentDate         time.Time `json:"assessment_date"`
	Recommendation         string    `json:"recommendation"`
}
