package models

import (
	"time"
)

// ConversationContext 会话上下文
type ConversationContext struct {
	// === 对话历史摘要 ===
	ConversationSummary ConversationSummary `json:"conversation_summary"` // 对话摘要
	KeyTopics           []TopicSummary      `json:"key_topics"`           // 关键话题
	TopicTransitions    []TopicTransition   `json:"topic_transitions"`    // 话题转换

	// === 决策和结论 ===
	KeyDecisions         []Decision   `json:"key_decisions"`         // 关键决策
	ImportantConclusions []Conclusion `json:"important_conclusions"` // 重要结论
	ActionItems          []ActionItem `json:"action_items"`          // 行动项

	// === 问题和疑虑 ===
	UnresolvedQuestions   []Question      `json:"unresolved_questions"`   // 未解决问题
	OngoingDiscussions    []Discussion    `json:"ongoing_discussions"`    // 进行中讨论
	PendingClarifications []Clarification `json:"pending_clarifications"` // 待澄清事项

	// === 用户特征 ===
	UserPreferences    UserPreferences    `json:"user_preferences"`    // 用户偏好
	CommunicationStyle CommunicationStyle `json:"communication_style"` // 沟通风格
	ExpertiseLevel     ExpertiseLevel     `json:"expertise_level"`     // 专业水平

	// === 对话流程 ===
	ConversationFlow   []ConversationStep `json:"conversation_flow"`    // 对话流程
	CurrentPhase       ConversationPhase  `json:"current_phase"`        // 当前阶段
	NextExpectedAction string             `json:"next_expected_action"` // 下一步预期

	// === 上下文质量 ===
	ContextCompleteness float64          `json:"context_completeness"` // 上下文完整度
	InformationGaps     []InformationGap `json:"information_gaps"`     // 信息缺口
	ConfidenceLevel     float64          `json:"confidence_level"`     // 置信度

	// === 元数据 ===
	LastUpdated     time.Time     `json:"last_updated"`     // 最后更新
	MessageCount    int           `json:"message_count"`    // 消息数量
	SessionDuration time.Duration `json:"session_duration"` // 会话时长
}

// ConversationSummary 对话摘要
type ConversationSummary struct {
	OverallSummary    string             `json:"overall_summary"`    // 总体摘要
	MainThemes        []string           `json:"main_themes"`        // 主要主题
	KeyInsights       []string           `json:"key_insights"`       // 关键洞察
	ImportantMentions []ImportantMention `json:"important_mentions"` // 重要提及
	EmotionalTone     EmotionalTone      `json:"emotional_tone"`     // 情感基调
	UrgencyLevel      UrgencyLevel       `json:"urgency_level"`      // 紧急程度
}

// TopicSummary 话题摘要
type TopicSummary struct {
	TopicName    string        `json:"topic_name"`    // 话题名称
	StartTime    time.Time     `json:"start_time"`    // 开始时间
	EndTime      time.Time     `json:"end_time"`      // 结束时间
	Duration     time.Duration `json:"duration"`      // 持续时间
	MessageCount int           `json:"message_count"` // 消息数量
	KeyPoints    []string      `json:"key_points"`    // 关键点
	Resolution   string        `json:"resolution"`    // 解决方案
	Importance   float64       `json:"importance"`    // 重要程度
}

// TopicTransition 话题转换
type TopicTransition struct {
	FromTopic      string    `json:"from_topic"`      // 源话题
	ToTopic        string    `json:"to_topic"`        // 目标话题
	TransitionTime time.Time `json:"transition_time"` // 转换时间
	TransitionType string    `json:"transition_type"` // 转换类型
	TriggerMessage string    `json:"trigger_message"` // 触发消息
	Reason         string    `json:"reason"`          // 转换原因
}

// Decision 决策信息
type Decision struct {
	DecisionID          string       `json:"decision_id"`          // 决策ID
	DecisionTitle       string       `json:"decision_title"`       // 决策标题
	DecisionDescription string       `json:"decision_description"` // 决策描述
	DecisionType        DecisionType `json:"decision_type"`        // 决策类型
	DecisionMaker       string       `json:"decision_maker"`       // 决策者

	// 决策背景
	Context      string             `json:"context"`      // 决策背景
	Alternatives []Alternative      `json:"alternatives"` // 备选方案
	Criteria     []DecisionCriteria `json:"criteria"`     // 决策标准

	// 决策结果
	ChosenOption    string `json:"chosen_option"`    // 选择方案
	Rationale       string `json:"rationale"`        // 决策理由
	ExpectedOutcome string `json:"expected_outcome"` // 预期结果

	// 元数据
	DecisionTime    time.Time      `json:"decision_time"`    // 决策时间
	ConfidenceLevel float64        `json:"confidence_level"` // 置信度
	Impact          DecisionImpact `json:"impact"`           // 影响评估
}

// DecisionType 决策类型
type DecisionType string

const (
	DecisionTypeTechnical     DecisionType = "technical"
	DecisionTypeBusiness      DecisionType = "business"
	DecisionTypeArchitectural DecisionType = "architectural"
	DecisionTypeOperational   DecisionType = "operational"
)

// Alternative 备选方案
type Alternative struct {
	OptionName  string   `json:"option_name"` // 方案名称
	Description string   `json:"description"` // 方案描述
	Pros        []string `json:"pros"`        // 优点
	Cons        []string `json:"cons"`        // 缺点
	Cost        float64  `json:"cost"`        // 成本
	Risk        float64  `json:"risk"`        // 风险
	Feasibility float64  `json:"feasibility"` // 可行性
}

// DecisionCriteria 决策标准
type DecisionCriteria struct {
	CriteriaName string  `json:"criteria_name"` // 标准名称
	Weight       float64 `json:"weight"`        // 权重
	Description  string  `json:"description"`   // 描述
}

// DecisionImpact 决策影响
type DecisionImpact struct {
	ImpactScope     string   `json:"impact_scope"`      // 影响范围
	AffectedAreas   []string `json:"affected_areas"`    // 影响区域
	RiskLevel       string   `json:"risk_level"`        // 风险级别
	TimeToImplement string   `json:"time_to_implement"` // 实施时间
	ResourcesNeeded []string `json:"resources_needed"`  // 所需资源
}

// Conclusion 结论
type Conclusion struct {
	ConclusionID string    `json:"conclusion_id"` // 结论ID
	Title        string    `json:"title"`         // 标题
	Description  string    `json:"description"`   // 描述
	Evidence     []string  `json:"evidence"`      // 证据
	Confidence   float64   `json:"confidence"`    // 置信度
	Implications []string  `json:"implications"`  // 影响
	NextSteps    []string  `json:"next_steps"`    // 下一步
	ConcludedAt  time.Time `json:"concluded_at"`  // 得出时间
}

// Question 问题
type Question struct {
	QuestionID     string    `json:"question_id"`     // 问题ID
	Question       string    `json:"question"`        // 问题内容
	Context        string    `json:"context"`         // 问题背景
	QuestionType   string    `json:"question_type"`   // 问题类型
	Importance     float64   `json:"importance"`      // 重要程度
	Urgency        Priority  `json:"urgency"`         // 紧急程度
	RelatedTopics  []string  `json:"related_topics"`  // 相关话题
	AskedAt        time.Time `json:"asked_at"`        // 提问时间
	ExpectedAnswer string    `json:"expected_answer"` // 期望答案
}

// Discussion 讨论
type Discussion struct {
	DiscussionID  string    `json:"discussion_id"` // 讨论ID
	Topic         string    `json:"topic"`         // 讨论话题
	Participants  []string  `json:"participants"`  // 参与者
	StartTime     time.Time `json:"start_time"`    // 开始时间
	LastActivity  time.Time `json:"last_activity"` // 最后活动
	Status        string    `json:"status"`        // 状态
	KeyPoints     []string  `json:"key_points"`    // 关键点
	Agreements    []string  `json:"agreements"`    // 达成的共识
	Disagreements []string  `json:"disagreements"` // 分歧
	NextActions   []string  `json:"next_actions"`  // 下一步行动
}

// Clarification 澄清事项
type Clarification struct {
	ClarificationID string    `json:"clarification_id"` // 澄清ID
	Subject         string    `json:"subject"`          // 主题
	Question        string    `json:"question"`         // 需要澄清的问题
	Context         string    `json:"context"`          // 背景
	Importance      float64   `json:"importance"`       // 重要程度
	RequestedAt     time.Time `json:"requested_at"`     // 请求时间
	Deadline        time.Time `json:"deadline"`         // 截止时间
	Status          string    `json:"status"`           // 状态
}

// UserPreferences 用户偏好
type UserPreferences struct {
	// === 技术偏好 ===
	PreferredLanguages  []string `json:"preferred_languages"`  // 偏好语言
	PreferredFrameworks []string `json:"preferred_frameworks"` // 偏好框架
	PreferredTools      []string `json:"preferred_tools"`      // 偏好工具

	// === 沟通偏好 ===
	DetailLevel           DetailLevel      `json:"detail_level"`            // 详细程度
	ExplanationStyle      ExplanationStyle `json:"explanation_style"`       // 解释风格
	CodeExamplePreference CodeExamplePref  `json:"code_example_preference"` // 代码示例偏好

	// === 工作模式 ===
	WorkingStyle           WorkingStyle        `json:"working_style"`            // 工作风格
	ProblemSolvingApproach ProblemSolvingStyle `json:"problem_solving_approach"` // 问题解决方式
	LearningPreference     LearningStyle       `json:"learning_preference"`      // 学习偏好

	// === 元数据 ===
	PreferenceSource PreferenceSource `json:"preference_source"` // 偏好来源
	ConfidenceLevel  float64          `json:"confidence_level"`  // 置信度
	LastUpdated      time.Time        `json:"last_updated"`      // 最后更新
}

// DetailLevel 详细程度
type DetailLevel string

const (
	DetailLevelHigh   DetailLevel = "high"
	DetailLevelMedium DetailLevel = "medium"
	DetailLevelLow    DetailLevel = "low"
)

// ExplanationStyle 解释风格
type ExplanationStyle string

const (
	ExplanationStyleTechnical  ExplanationStyle = "technical"
	ExplanationStyleConceptual ExplanationStyle = "conceptual"
	ExplanationStylePractical  ExplanationStyle = "practical"
)

// CodeExamplePref 代码示例偏好
type CodeExamplePref string

const (
	CodeExamplePrefMany CodeExamplePref = "many"
	CodeExamplePrefFew  CodeExamplePref = "few"
	CodeExamplePrefNone CodeExamplePref = "none"
)

// WorkingStyle 工作风格
type WorkingStyle string

const (
	WorkingStyleMethodical WorkingStyle = "methodical"
	WorkingStyleAgile      WorkingStyle = "agile"
	WorkingStyleExplorer   WorkingStyle = "explorer"
)

// ProblemSolvingStyle 问题解决风格
type ProblemSolvingStyle string

const (
	ProblemSolvingStyleAnalytical ProblemSolvingStyle = "analytical"
	ProblemSolvingStyleIntuitive  ProblemSolvingStyle = "intuitive"
	ProblemSolvingStyleIterative  ProblemSolvingStyle = "iterative"
)

// LearningStyle 学习风格
type LearningStyle string

const (
	LearningStyleVisual      LearningStyle = "visual"
	LearningStyleAuditory    LearningStyle = "auditory"
	LearningStyleKinesthetic LearningStyle = "kinesthetic"
)

// PreferenceSource 偏好来源
type PreferenceSource string

const (
	PreferenceSourceExplicit PreferenceSource = "explicit"
	PreferenceSourceInferred PreferenceSource = "inferred"
	PreferenceSourceDefault  PreferenceSource = "default"
)

// CommunicationStyle 沟通风格
type CommunicationStyle struct {
	Formality     string `json:"formality"`      // 正式程度
	Directness    string `json:"directness"`     // 直接程度
	Patience      string `json:"patience"`       // 耐心程度
	Encouragement string `json:"encouragement"`  // 鼓励程度
	QuestionStyle string `json:"question_style"` // 提问风格
	FeedbackStyle string `json:"feedback_style"` // 反馈风格
}

// ExpertiseLevel 专业水平
type ExpertiseLevel struct {
	OverallLevel    string            `json:"overall_level"`    // 总体水平
	DomainExpertise map[string]string `json:"domain_expertise"` // 领域专业度
	LearningSpeed   string            `json:"learning_speed"`   // 学习速度
	Experience      string            `json:"experience"`       // 经验水平
}

// ConversationStep 对话步骤
type ConversationStep struct {
	StepIndex    int           `json:"step_index"`     // 步骤序号
	StepType     string        `json:"step_type"`      // 步骤类型
	Description  string        `json:"description"`    // 描述
	Timestamp    time.Time     `json:"timestamp"`      // 时间戳
	Duration     time.Duration `json:"duration"`       // 持续时间
	Outcome      string        `json:"outcome"`        // 结果
	NextStepHint string        `json:"next_step_hint"` // 下一步提示
}

// ConversationPhase 对话阶段
type ConversationPhase string

const (
	ConversationPhaseInitial     ConversationPhase = "initial"
	ConversationPhaseExploration ConversationPhase = "exploration"
	ConversationPhaseAnalysis    ConversationPhase = "analysis"
	ConversationPhaseResolution  ConversationPhase = "resolution"
	ConversationPhaseWrapup      ConversationPhase = "wrapup"
)

// ImportantMention 重要提及
type ImportantMention struct {
	MentionType string    `json:"mention_type"` // 提及类型
	Content     string    `json:"content"`      // 内容
	Context     string    `json:"context"`      // 上下文
	Importance  float64   `json:"importance"`   // 重要程度
	Timestamp   time.Time `json:"timestamp"`    // 时间戳
}

// EmotionalTone 情感基调
type EmotionalTone struct {
	Sentiment  string   `json:"sentiment"`  // 情感倾向
	Intensity  float64  `json:"intensity"`  // 强度
	Confidence float64  `json:"confidence"` // 置信度
	Keywords   []string `json:"keywords"`   // 关键词
}

// UrgencyLevel 紧急程度
type UrgencyLevel struct {
	Level        string   `json:"level"`        // 级别
	Indicators   []string `json:"indicators"`   // 指标
	Deadline     string   `json:"deadline"`     // 截止时间
	Consequences string   `json:"consequences"` // 后果
}

// InformationGap 信息缺口（在context_extensions.go中已定义，这里引用）
// 注意：InformationGap 已在其他文件中定义，这里不重复定义
