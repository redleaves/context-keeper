package models

import "time"

// ContextComplexityLevel 上下文复杂度级别
type ContextComplexityLevel string

const (
	ContextLevelMinimal  ContextComplexityLevel = "minimal"  // 最小化：只包含核心字段
	ContextLevelBasic    ContextComplexityLevel = "basic"    // 基础：包含常用字段
	ContextLevelStandard ContextComplexityLevel = "standard" // 标准：包含大部分字段
	ContextLevelFull     ContextComplexityLevel = "full"     // 完整：包含所有字段
)

// SimplifiedContextModel 简化的上下文模型
type SimplifiedContextModel struct {
	// === 核心标识（必需）===
	SessionID   string `json:"session_id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`

	// === 核心主题（必需）===
	MainTopic        string  `json:"main_topic"`
	TopicCategory    string  `json:"topic_category"`
	PrimaryPainPoint string  `json:"primary_pain_point"`
	ExpectedOutcome  string  `json:"expected_outcome"`
	ConfidenceLevel  float64 `json:"confidence_level"`

	// === 关键信息（基础级别）===
	KeyConcepts     []string `json:"key_concepts,omitempty"`
	TechnicalTerms  []string `json:"technical_terms,omitempty"`
	CurrentPhase    string   `json:"current_phase,omitempty"`
	PrimaryLanguage string   `json:"primary_language,omitempty"`

	// === 扩展信息（标准级别）===
	SecondaryPainPoints []string              `json:"secondary_pain_points,omitempty"`
	ActionItems         []SimpleActionItem    `json:"action_items,omitempty"`
	RelatedTopics       []string              `json:"related_topics,omitempty"`
	ProjectInfo         *SimpleProjectInfo    `json:"project_info,omitempty"`
	RecentActivity      *SimpleRecentActivity `json:"recent_activity,omitempty"`

	// === 元数据 ===
	ComplexityLevel ContextComplexityLevel `json:"complexity_level"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// SimpleActionItem 简化的行动项
type SimpleActionItem struct {
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Type        string `json:"type"`
}

// SimpleProjectInfo 简化的项目信息
type SimpleProjectInfo struct {
	ProjectName     string `json:"project_name"`
	ProjectType     string `json:"project_type"`
	PrimaryLanguage string `json:"primary_language"`
	CurrentPhase    string `json:"current_phase"`
	Description     string `json:"description,omitempty"`
}

// SimpleRecentActivity 简化的最近活动
type SimpleRecentActivity struct {
	LastChangeTime time.Time `json:"last_change_time"`
	RecentChanges  []string  `json:"recent_changes,omitempty"`
	ActiveFiles    []string  `json:"active_files,omitempty"`
	CompletedTasks []string  `json:"completed_tasks,omitempty"`
	OngoingTasks   []string  `json:"ongoing_tasks,omitempty"`
}

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	level ContextComplexityLevel
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(level ContextComplexityLevel) *ContextBuilder {
	return &ContextBuilder{level: level}
}

// BuildFromLLMResponse 从LLM响应构建上下文
func (cb *ContextBuilder) BuildFromLLMResponse(llmResponse string, retrievalResults *RetrievalResults) (*SimplifiedContextModel, error) {
	// 根据复杂度级别决定解析哪些字段
	switch cb.level {
	case ContextLevelMinimal:
		return cb.buildMinimalContext(llmResponse)
	case ContextLevelBasic:
		return cb.buildBasicContext(llmResponse, retrievalResults)
	case ContextLevelStandard:
		return cb.buildStandardContext(llmResponse, retrievalResults)
	case ContextLevelFull:
		return cb.buildFullContext(llmResponse, retrievalResults)
	default:
		return cb.buildBasicContext(llmResponse, retrievalResults)
	}
}

// buildMinimalContext 构建最小化上下文（只要求LLM生成核心字段）
func (cb *ContextBuilder) buildMinimalContext(llmResponse string) (*SimplifiedContextModel, error) {
	// LLM只需要生成这些字段：
	// {
	//   "main_topic": "...",
	//   "topic_category": "...",
	//   "primary_pain_point": "...",
	//   "expected_outcome": "...",
	//   "confidence_level": 0.8
	// }

	// 实现JSON解析逻辑...
	return &SimplifiedContextModel{
		ComplexityLevel: ContextLevelMinimal,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// buildBasicContext 构建基础上下文
func (cb *ContextBuilder) buildBasicContext(llmResponse string, retrievalResults *RetrievalResults) (*SimplifiedContextModel, error) {
	// 先构建最小化上下文
	ctx, err := cb.buildMinimalContext(llmResponse)
	if err != nil {
		return nil, err
	}

	// 从检索结果中补充基础信息
	if retrievalResults != nil {
		ctx.KeyConcepts = cb.extractKeyConceptsFromResults(retrievalResults)
		ctx.TechnicalTerms = cb.extractTechnicalTermsFromResults(retrievalResults)
		ctx.PrimaryLanguage = cb.inferPrimaryLanguage(retrievalResults)
	}

	ctx.ComplexityLevel = ContextLevelBasic
	return ctx, nil
}

// buildStandardContext 构建标准上下文
func (cb *ContextBuilder) buildStandardContext(llmResponse string, retrievalResults *RetrievalResults) (*SimplifiedContextModel, error) {
	// 先构建基础上下文
	ctx, err := cb.buildBasicContext(llmResponse, retrievalResults)
	if err != nil {
		return nil, err
	}

	// 从检索结果中补充标准信息
	if retrievalResults != nil {
		ctx.ProjectInfo = cb.buildProjectInfoFromResults(retrievalResults)
		ctx.RecentActivity = cb.buildRecentActivityFromResults(retrievalResults)
		ctx.ActionItems = cb.extractActionItemsFromResults(retrievalResults)
	}

	ctx.ComplexityLevel = ContextLevelStandard
	return ctx, nil
}

// buildFullContext 构建完整上下文
func (cb *ContextBuilder) buildFullContext(llmResponse string, retrievalResults *RetrievalResults) (*SimplifiedContextModel, error) {
	// 先构建标准上下文
	ctx, err := cb.buildStandardContext(llmResponse, retrievalResults)
	if err != nil {
		return nil, err
	}

	// 补充完整信息
	ctx.ComplexityLevel = ContextLevelFull
	return ctx, nil
}

// 辅助方法（简化实现）
func (cb *ContextBuilder) extractKeyConceptsFromResults(results *RetrievalResults) []string {
	concepts := []string{}
	// 从检索结果中提取关键概念
	for _, kr := range results.KnowledgeResults {
		concepts = append(concepts, kr.ConceptName)
	}
	return concepts
}

func (cb *ContextBuilder) extractTechnicalTermsFromResults(results *RetrievalResults) []string {
	terms := []string{}
	// 从检索结果中提取技术术语
	for _, vr := range results.VectorResults {
		// 简单的关键词提取逻辑
		if len(vr.Tags) > 0 {
			terms = append(terms, vr.Tags...)
		}
	}
	return terms
}

func (cb *ContextBuilder) inferPrimaryLanguage(results *RetrievalResults) string {
	// 从时间线结果中推断主要编程语言
	for _, tr := range results.TimelineResults {
		if tr.EventType == "code_change" {
			// 简单的语言推断逻辑
			return "go" // 默认
		}
	}
	return "unknown"
}

func (cb *ContextBuilder) buildProjectInfoFromResults(results *RetrievalResults) *SimpleProjectInfo {
	return &SimpleProjectInfo{
		ProjectName:     "推断的项目名称",
		ProjectType:     "backend",
		PrimaryLanguage: cb.inferPrimaryLanguage(results),
		CurrentPhase:    "development",
		Description:     "基于检索结果推断的项目描述",
	}
}

func (cb *ContextBuilder) buildRecentActivityFromResults(results *RetrievalResults) *SimpleRecentActivity {
	activity := &SimpleRecentActivity{
		LastChangeTime: time.Now(),
		RecentChanges:  []string{},
		ActiveFiles:    []string{},
	}

	// 从时间线结果中提取最近活动
	for _, tr := range results.TimelineResults {
		activity.RecentChanges = append(activity.RecentChanges, tr.Title)
	}

	return activity
}

func (cb *ContextBuilder) extractActionItemsFromResults(results *RetrievalResults) []SimpleActionItem {
	items := []SimpleActionItem{}

	// 从检索结果中提取可能的行动项
	for _, tr := range results.TimelineResults {
		if tr.EventType == "discussion" {
			items = append(items, SimpleActionItem{
				Description: "跟进讨论：" + tr.Title,
				Priority:    "medium",
				Type:        "follow_up",
			})
		}
	}

	return items
}

// ConvertToUnifiedContext 将简化上下文转换为完整上下文（向后兼容）
func (sc *SimplifiedContextModel) ConvertToUnifiedContext() *UnifiedContextModel {
	unified := &UnifiedContextModel{
		SessionID:   sc.SessionID,
		UserID:      sc.UserID,
		WorkspaceID: sc.WorkspaceID,
		CreatedAt:   sc.CreatedAt,
		UpdatedAt:   sc.UpdatedAt,
	}

	// 构建TopicContext
	if sc.MainTopic != "" {
		unified.CurrentTopic = &TopicContext{
			MainTopic:        sc.MainTopic,
			PrimaryPainPoint: sc.PrimaryPainPoint,
			ExpectedOutcome:  sc.ExpectedOutcome,
			ConfidenceLevel:  sc.ConfidenceLevel,
			TopicStartTime:   sc.CreatedAt,
			LastUpdated:      sc.UpdatedAt,
			UpdateCount:      1,
		}

		// 转换关键概念
		for _, concept := range sc.KeyConcepts {
			unified.CurrentTopic.KeyConcepts = append(unified.CurrentTopic.KeyConcepts, ConceptInfo{
				ConceptName: concept,
				ConceptType: ConceptTypeTechnical,
				Importance:  0.8,
				Source:      "llm_analysis",
			})
		}
	}

	// 构建ProjectContext
	if sc.ProjectInfo != nil {
		unified.Project = &ProjectContext{
			ProjectName:     sc.ProjectInfo.ProjectName,
			ProjectType:     ProjectType(sc.ProjectInfo.ProjectType),
			PrimaryLanguage: sc.ProjectInfo.PrimaryLanguage,
			Description:     sc.ProjectInfo.Description,
			CurrentPhase:    ProjectPhase(sc.ProjectInfo.CurrentPhase),
			ConfidenceLevel: 0.7,
		}
	}

	return unified
}
