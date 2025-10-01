package models

import (
	"time"
)

// UnifiedContextModel 统一上下文模型
type UnifiedContextModel struct {
	// === 核心标识 ===
	SessionID   string `json:"session_id"`   // 会话ID（生命周期绑定）
	UserID      string `json:"user_id"`      // 用户隔离
	WorkspaceID string `json:"workspace_id"` // 工作空间隔离

	// === 当前主题/痛点（核心！）===
	CurrentTopic *TopicContext `json:"current_topic"`

	// === 最近变更描述（简化版本）===
	RecentChangesSummary string `json:"recent_changes_summary,omitempty"` // 语义/需求/痛点变更的一句话描述

	// === 项目上下文（常驻信息）===
	Project *ProjectContext `json:"project"`

	// === 代码上下文 ===
	Code *CodeContext `json:"code"`

	// === 会话上下文 ===
	Conversation *ConversationContext `json:"conversation"`

	// === 元数据 ===
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TopicContext 当前主题上下文（最重要的核心）
type TopicContext struct {
	// === 核心主题信息 ===
	MainTopic     string        `json:"main_topic"`     // 主要话题
	TopicCategory TopicCategory `json:"topic_category"` // 话题分类
	UserIntent    UserIntent    `json:"user_intent"`    // 用户意图

	// === 痛点和需求 ===
	PrimaryPainPoint    string   `json:"primary_pain_point"`    // 主要痛点
	SecondaryPainPoints []string `json:"secondary_pain_points"` // 次要痛点
	ExpectedOutcome     string   `json:"expected_outcome"`      // 期望结果

	// === 上下文关键词 ===
	KeyConcepts    []ConceptInfo   `json:"key_concepts"`    // 关键概念
	TechnicalTerms []TechnicalTerm `json:"technical_terms"` // 技术术语
	BusinessTerms  []BusinessTerm  `json:"business_terms"`  // 业务术语

	// === 话题演进 ===
	TopicEvolution []TopicEvolutionStep `json:"topic_evolution"` // 话题演进步骤
	RelatedTopics  []RelatedTopic       `json:"related_topics"`  // 相关话题

	// === 元数据 ===
	TopicStartTime  time.Time `json:"topic_start_time"` // 话题开始时间
	LastUpdated     time.Time `json:"last_updated"`     // 最后更新时间
	UpdateCount     int       `json:"update_count"`     // 更新次数
	ConfidenceLevel float64   `json:"confidence_level"` // 置信度
}

// TopicCategory 话题分类
type TopicCategory string

const (
	TopicCategoryTechnical       TopicCategory = "technical"       // 技术问题
	TopicCategoryProject         TopicCategory = "project"         // 项目管理
	TopicCategoryBusiness        TopicCategory = "business"        // 业务需求
	TopicCategoryLearning        TopicCategory = "learning"        // 学习探索
	TopicCategoryTroubleshooting TopicCategory = "troubleshooting" // 问题排查
)

// UserIntent 用户意图
type UserIntent struct {
	IntentType        IntentType        `json:"intent_type"`        // 意图类型
	IntentDescription string            `json:"intent_description"` // 意图描述
	ActionRequired    []ActionItem      `json:"action_required"`    // 需要的行动
	InformationNeeded []InformationNeed `json:"information_needed"` // 需要的信息
	Priority          Priority          `json:"priority"`           // 优先级
}

// 注意：IntentType 已在 llm_driven_models.go 中定义，这里直接使用

// Priority 优先级
type Priority string

const (
	PriorityHigh   Priority = "high"   // 高优先级
	PriorityMedium Priority = "medium" // 中优先级
	PriorityLow    Priority = "low"    // 低优先级
)

// ConceptInfo 概念信息
type ConceptInfo struct {
	ConceptName     string      `json:"concept_name"`     // 概念名称
	ConceptType     ConceptType `json:"concept_type"`     // 概念类型
	Definition      string      `json:"definition"`       // 定义
	Importance      float64     `json:"importance"`       // 重要程度
	RelatedConcepts []string    `json:"related_concepts"` // 相关概念
	Source          string      `json:"source"`           // 来源
}

// ConceptType 概念类型
type ConceptType string

const (
	ConceptTypeTechnical ConceptType = "technical" // 技术概念
	ConceptTypeBusiness  ConceptType = "business"  // 业务概念
	ConceptTypeDomain    ConceptType = "domain"    // 领域概念
)

// TechnicalTerm 技术术语
type TechnicalTerm struct {
	TermName    string   `json:"term_name"`    // 术语名称
	Definition  string   `json:"definition"`   // 定义
	Category    string   `json:"category"`     // 分类
	Importance  float64  `json:"importance"`   // 重要程度
	Context     string   `json:"context"`      // 使用上下文
	RelatedAPIs []string `json:"related_apis"` // 相关API
}

// BusinessTerm 业务术语
type BusinessTerm struct {
	TermName    string  `json:"term_name"`   // 术语名称
	Definition  string  `json:"definition"`  // 定义
	Domain      string  `json:"domain"`      // 业务域
	Importance  float64 `json:"importance"`  // 重要程度
	Stakeholder string  `json:"stakeholder"` // 相关干系人
}

// ActionItem 行动项
type ActionItem struct {
	ActionType  string   `json:"action_type"` // 行动类型
	Description string   `json:"description"` // 行动描述
	Priority    Priority `json:"priority"`    // 优先级
	Deadline    string   `json:"deadline"`    // 截止时间
}

// InformationNeed 信息需求
type InformationNeed struct {
	InfoType    string   `json:"info_type"`   // 信息类型
	Description string   `json:"description"` // 信息描述
	Urgency     Priority `json:"urgency"`     // 紧急程度
	Source      string   `json:"source"`      // 期望来源
}

// TopicEvolutionStep 话题演进步骤
type TopicEvolutionStep struct {
	StepIndex       int       `json:"step_index"`       // 步骤序号
	StepDescription string    `json:"step_description"` // 步骤描述
	KeyChanges      []string  `json:"key_changes"`      // 关键变化
	Timestamp       time.Time `json:"timestamp"`        // 时间戳
	TriggerQuery    string    `json:"trigger_query"`    // 触发查询
}

// RelatedTopic 相关话题
type RelatedTopic struct {
	TopicName   string  `json:"topic_name"`   // 话题名称
	Relation    string  `json:"relation"`     // 关系类型
	Similarity  float64 `json:"similarity"`   // 相似度
	LastMention string  `json:"last_mention"` // 最后提及时间
}

// ProjectContext 项目上下文
type ProjectContext struct {
	// === 基础信息 ===
	ProjectName string      `json:"project_name"` // 项目名称
	ProjectPath string      `json:"project_path"` // 项目路径
	ProjectType ProjectType `json:"project_type"` // 项目类型
	Description string      `json:"description"`  // 项目描述

	// === 技术栈信息 ===
	PrimaryLanguage string           `json:"primary_language"` // 主要编程语言
	TechStack       []TechStackItem  `json:"tech_stack"`       // 技术栈
	Dependencies    []DependencyInfo `json:"dependencies"`     // 依赖信息
	Architecture    ArchitectureInfo `json:"architecture"`     // 架构信息

	// === 项目结构 ===
	MainComponents []ComponentInfo `json:"main_components"` // 主要组件
	KeyModules     []ModuleInfo    `json:"key_modules"`     // 关键模块
	ImportantFiles []FileInfo      `json:"important_files"` // 重要文件

	// === 项目状态 ===
	CurrentPhase     ProjectPhase     `json:"current_phase"`     // 当前阶段
	KeyFeatures      []FeatureInfo    `json:"key_features"`      // 主要功能
	CompletionStatus CompletionStatus `json:"completion_status"` // 完成状态

	// === 元数据 ===
	LastAnalyzed    time.Time `json:"last_analyzed"`    // 最后分析时间
	AnalysisVersion int       `json:"analysis_version"` // 分析版本
	ConfidenceLevel float64   `json:"confidence_level"` // 置信度
}

// ProjectType 项目类型
type ProjectType string

const (
	ProjectTypeGo         ProjectType = "go"         // Go项目
	ProjectTypeNodeJS     ProjectType = "nodejs"     // Node.js项目
	ProjectTypePython     ProjectType = "python"     // Python项目
	ProjectTypeJava       ProjectType = "java"       // Java项目
	ProjectTypeRust       ProjectType = "rust"       // Rust项目
	ProjectTypeTypeScript ProjectType = "typescript" // TypeScript项目
	ProjectTypeOther      ProjectType = "other"      // 其他类型
)

// ProjectPhase 项目阶段
type ProjectPhase string

const (
	ProjectPhasePlanning    ProjectPhase = "planning"    // 规划阶段
	ProjectPhaseDevelopment ProjectPhase = "development" // 开发阶段
	ProjectPhaseTesting     ProjectPhase = "testing"     // 测试阶段
	ProjectPhaseDeployment  ProjectPhase = "deployment"  // 部署阶段
	ProjectPhaseMaintenance ProjectPhase = "maintenance" // 维护阶段
)

// TechStackItem 技术栈项
type TechStackItem struct {
	Name       string  `json:"name"`       // 技术名称
	Type       string  `json:"type"`       // 技术类型
	Version    string  `json:"version"`    // 版本
	Importance float64 `json:"importance"` // 重要程度
}

// DependencyInfo 依赖信息
type DependencyInfo struct {
	Name        string `json:"name"`        // 依赖名称
	Version     string `json:"version"`     // 版本
	Type        string `json:"type"`        // 依赖类型
	Description string `json:"description"` // 描述
}

// ArchitectureInfo 架构信息
type ArchitectureInfo struct {
	Pattern     string   `json:"pattern"`     // 架构模式
	Layers      []string `json:"layers"`      // 架构层次
	Components  []string `json:"components"`  // 主要组件
	Description string   `json:"description"` // 架构描述
}

// ComponentInfo 组件信息
type ComponentInfo struct {
	ComponentName string          `json:"component_name"` // 组件名称
	ComponentType ComponentType   `json:"component_type"` // 组件类型
	Description   string          `json:"description"`    // 描述
	FilePaths     []string        `json:"file_paths"`     // 文件路径
	Dependencies  []string        `json:"dependencies"`   // 依赖组件
	Interfaces    []InterfaceInfo `json:"interfaces"`     // 接口信息
	Importance    float64         `json:"importance"`     // 重要程度
	LastModified  time.Time       `json:"last_modified"`  // 最后修改时间
}

// ComponentType 组件类型
type ComponentType string

const (
	ComponentTypeService   ComponentType = "service"   // 服务组件
	ComponentTypeModule    ComponentType = "module"    // 模块组件
	ComponentTypeLibrary   ComponentType = "library"   // 库组件
	ComponentTypeInterface ComponentType = "interface" // 接口组件
	ComponentTypeUtil      ComponentType = "util"      // 工具组件
)

// ModuleInfo 模块信息
type ModuleInfo struct {
	ModuleName   string    `json:"module_name"`   // 模块名称
	Description  string    `json:"description"`   // 模块描述
	FilePath     string    `json:"file_path"`     // 文件路径
	Exports      []string  `json:"exports"`       // 导出内容
	Imports      []string  `json:"imports"`       // 导入依赖
	Importance   float64   `json:"importance"`    // 重要程度
	LastModified time.Time `json:"last_modified"` // 最后修改时间
}

// FileInfo 文件信息
type FileInfo struct {
	FilePath     string    `json:"file_path"`     // 文件路径
	FileType     string    `json:"file_type"`     // 文件类型
	Description  string    `json:"description"`   // 文件描述
	Size         int64     `json:"size"`          // 文件大小
	Importance   float64   `json:"importance"`    // 重要程度
	LastModified time.Time `json:"last_modified"` // 最后修改时间
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	InterfaceName string   `json:"interface_name"` // 接口名称
	Methods       []string `json:"methods"`        // 方法列表
	Description   string   `json:"description"`    // 接口描述
	FilePath      string   `json:"file_path"`      // 定义文件路径
}

// FeatureInfo 功能信息
type FeatureInfo struct {
	FeatureName       string             `json:"feature_name"`       // 功能名称
	FeatureType       FeatureType        `json:"feature_type"`       // 功能类型
	Description       string             `json:"description"`        // 功能描述
	Status            FeatureStatus      `json:"status"`             // 功能状态
	RelatedComponents []string           `json:"related_components"` // 相关组件
	Implementation    ImplementationInfo `json:"implementation"`     // 实现信息
	Priority          Priority           `json:"priority"`           // 优先级
	CompletionRate    float64            `json:"completion_rate"`    // 完成率
}

// FeatureType 功能类型
type FeatureType string

const (
	FeatureTypeCore        FeatureType = "core"        // 核心功能
	FeatureTypeExtension   FeatureType = "extension"   // 扩展功能
	FeatureTypeIntegration FeatureType = "integration" // 集成功能
	FeatureTypeUtility     FeatureType = "utility"     // 工具功能
)

// FeatureStatus 功能状态
type FeatureStatus string

const (
	FeatureStatusCompleted  FeatureStatus = "completed"   // 已完成
	FeatureStatusInProgress FeatureStatus = "in_progress" // 进行中
	FeatureStatusPlanned    FeatureStatus = "planned"     // 已规划
	FeatureStatusBlocked    FeatureStatus = "blocked"     // 阻塞
)

// ImplementationInfo 实现信息
type ImplementationInfo struct {
	FilePaths    []string `json:"file_paths"`    // 实现文件路径
	KeyFunctions []string `json:"key_functions"` // 关键函数
	Dependencies []string `json:"dependencies"`  // 依赖项
	TestCoverage float64  `json:"test_coverage"` // 测试覆盖率
}

// CompletionStatus 完成状态
type CompletionStatus struct {
	OverallProgress float64            `json:"overall_progress"` // 总体进度
	PhaseProgress   map[string]float64 `json:"phase_progress"`   // 各阶段进度
	Milestones      []Milestone        `json:"milestones"`       // 里程碑
}

// Milestone 里程碑
type Milestone struct {
	Name        string    `json:"name"`        // 里程碑名称
	Description string    `json:"description"` // 描述
	TargetDate  time.Time `json:"target_date"` // 目标日期
	Status      string    `json:"status"`      // 状态
	Progress    float64   `json:"progress"`    // 进度
}
