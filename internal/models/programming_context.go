package models

// 编程上下文相关的数据结构定义

// ProgrammingContext 编程上下文，包含代码文件、编辑历史和相关片段等
type ProgrammingContext struct {
	SessionID         string                `json:"sessionId"`
	AssociatedFiles   []CodeFileInfo        `json:"associatedFiles,omitempty"`
	RecentEdits       []EditInfo            `json:"recentEdits,omitempty"`
	RelevantSnippets  []CodeSnippet         `json:"relevantSnippets,omitempty"`
	DesignDecisions   []DecisionSummary     `json:"designDecisions,omitempty"` // 新增：设计决策列表
	LinkedSessions    []SessionReference    `json:"linkedSessions,omitempty"`  // 新增：关联会话
	RelatedContexts   []ContextReference    `json:"relatedContexts,omitempty"` // 新增：相关上下文引用
	ExtractedFeatures []string              `json:"extractedFeatures,omitempty"`
	Statistics        ProgrammingStatistics `json:"statistics,omitempty"` // 新增：编程统计信息
	Knowledge         KnowledgeGraph        `json:"knowledge,omitempty"`  // 新增：知识图谱
}

// CodeFileInfo 代码文件信息
type CodeFileInfo struct {
	Path               string          `json:"path"`
	Language           string          `json:"language,omitempty"`
	LastEdit           int64           `json:"lastEdit,omitempty"`
	Summary            string          `json:"summary,omitempty"`
	Dependencies       []string        `json:"dependencies,omitempty"`       // 新增：文件依赖
	RelatedDiscussions []DiscussionRef `json:"relatedDiscussions,omitempty"` // 新增：相关讨论引用
	Tags               []string        `json:"tags,omitempty"`               // 新增：标签
	Importance         float64         `json:"importance,omitempty"`         // 新增：重要性评分
}

// DiscussionRef 讨论引用
type DiscussionRef struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"` // message, memory, decision
	Summary   string  `json:"summary"`
	Timestamp int64   `json:"timestamp"`
	Relevance float64 `json:"relevance,omitempty"`
}

// EditInfo 编辑操作信息
type EditInfo struct {
	ID               string   `json:"id,omitempty"` // 新增：编辑ID
	Timestamp        int64    `json:"timestamp"`
	FilePath         string   `json:"filePath"`
	Type             string   `json:"type"` // insert, delete, modify
	Position         int      `json:"position"`
	Content          string   `json:"content,omitempty"`
	RelatedDecisions []string `json:"relatedDecisions,omitempty"` // 新增：关联的决策ID
	Tags             []string `json:"tags,omitempty"`             // 新增：标签
	Impact           int      `json:"impact,omitempty"`           // 新增：变更影响范围
	Purpose          string   `json:"purpose,omitempty"`          // 新增：变更目的
	RelatedEdits     []string `json:"relatedEdits,omitempty"`     // 新增：关联的其他编辑
}

// CodeSnippet 代码片段
type CodeSnippet struct {
	Content   string  `json:"content"`
	FilePath  string  `json:"filePath,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Context   string  `json:"context,omitempty"`   // 新增：片段上下文
	LineStart int     `json:"lineStart,omitempty"` // 新增：起始行号
	LineEnd   int     `json:"lineEnd,omitempty"`   // 新增：结束行号
	Function  string  `json:"function,omitempty"`  // 新增：所属函数
	Type      string  `json:"type,omitempty"`      // 新增：代码类型(函数定义、变量声明等)
}

// DecisionSummary 设计决策摘要
type DecisionSummary struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Category     string   `json:"category,omitempty"`
	Timestamp    int64    `json:"timestamp"`
	RelatedEdits []string `json:"relatedEdits,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Impact       string   `json:"impact,omitempty"`       // 新增：影响范围
	Alternatives []string `json:"alternatives,omitempty"` // 新增：备选方案
	Dependencies []string `json:"dependencies,omitempty"` // 新增：依赖的其他决策
	Status       string   `json:"status,omitempty"`       // 新增：决策状态(已实施、待实施等)
}

// SessionReference 会话引用
type SessionReference struct {
	SessionID    string   `json:"sessionId"`
	Relationship string   `json:"relationship"`
	Description  string   `json:"description,omitempty"`
	Timestamp    int64    `json:"timestamp"`
	Topics       []string `json:"topics,omitempty"`   // 新增：主题标签
	Strength     float64  `json:"strength,omitempty"` // 新增：关联强度
}

// ContextReference 上下文引用
type ContextReference struct {
	Type           string   `json:"type"` // conversation, code, decision, document, requirement
	Content        string   `json:"content"`
	SourceID       string   `json:"sourceId,omitempty"`
	Timestamp      int64    `json:"timestamp"`
	RelevanceScore float64  `json:"relevanceScore,omitempty"`
	Category       string   `json:"category,omitempty"`  // 新增：分类
	Author         string   `json:"author,omitempty"`    // 新增：作者
	ExpiresAt      int64    `json:"expiresAt,omitempty"` // 新增：过期时间
	Tags           []string `json:"tags,omitempty"`      // 新增：标签
}

// ProgrammingStatistics 编程统计信息
type ProgrammingStatistics struct {
	TotalFiles          int                `json:"totalFiles"`
	TotalEdits          int                `json:"totalEdits"`
	LanguageUsage       map[string]int     `json:"languageUsage,omitempty"`       // 语言使用情况
	EditsByFile         map[string]int     `json:"editsByFile,omitempty"`         // 按文件统计的编辑数
	ActivityByDay       map[string]int     `json:"activityByDay,omitempty"`       // 按日期统计的活动数
	DecisionsByCategory map[string]int     `json:"decisionsByCategory,omitempty"` // 按类别统计的决策数
	EditPatterns        []EditPattern      `json:"editPatterns,omitempty"`        // 新增：编辑模式
	CriticalPaths       []string           `json:"criticalPaths,omitempty"`       // 新增：关键路径文件
	Complexity          map[string]float64 `json:"complexity,omitempty"`          // 新增：复杂度指标
}

// EditPattern 编辑模式
type EditPattern struct {
	Pattern       string   `json:"pattern"`
	Frequency     int      `json:"frequency"`
	FilesAffected []string `json:"filesAffected,omitempty"`
	Description   string   `json:"description,omitempty"`
}

// KnowledgeGraph 知识图谱
type KnowledgeGraph struct {
	Nodes []KnowledgeNode `json:"nodes,omitempty"`
	Edges []KnowledgeEdge `json:"edges,omitempty"`
}

// 注意：KnowledgeNode和KnowledgeEdge现在使用unified_models.go中的统一定义
// 这里不再重复定义，请导入并使用统一模型

// ContextType 上下文类型常量
const (
	ContextTypeConversation = "conversation"
	ContextTypeCode         = "code"
	ContextTypeDecision     = "decision"
	ContextTypeDocument     = "document"
	ContextTypeRequirement  = "requirement"
	ContextTypeIssue        = "issue"
	ContextTypeTest         = "test"
	ContextTypeDeployment   = "deployment"
)

// RelationshipType 关系类型常量
const (
	RelationshipImplements    = "implements"
	RelationshipDependsOn     = "depends_on"
	RelationshipRefactorsFrom = "refactors_from"
	RelationshipInheritsFrom  = "inherits_from"
	RelationshipReferences    = "references"
	RelationshipTests         = "tests"
	RelationshipUsedBy        = "used_by"
	RelationshipAlternativeTo = "alternative_to"
)
