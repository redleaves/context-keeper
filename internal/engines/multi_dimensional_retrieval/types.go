package multi_dimensional_retrieval

// SemanticAnalysisResult 语义分析结果（避免循环导入）
type SemanticAnalysisResult struct {
	Intent     string                 `json:"intent"`
	Confidence float64                `json:"confidence"`
	Categories []string               `json:"categories"`
	Keywords   []string               `json:"keywords"`
	Entities   []Entity               `json:"entities"`
	Queries    *MultiDimensionalQuery `json:"queries"`
	TokenUsage int                    `json:"token_usage"`
	Metadata   map[string]interface{} `json:"metadata"`

	// 🔥 新增：LLM intent_analysis 提取的关键概念
	KeyConcepts []string `json:"key_concepts,omitempty"`
}

// Entity 实体信息
type Entity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

// MultiDimensionalQuery 多维度查询（重新定义避免循环导入）
type MultiDimensionalQuery struct {
	ContextQueries   []string `json:"context_queries"`
	TimelineQueries  []string `json:"timeline_queries"`
	KnowledgeQueries []string `json:"knowledge_queries"`
	VectorQueries    []string `json:"vector_queries"`
}

// ContextInfo 上下文信息（避免循环导入）
type ContextInfo struct {
	// 会话上下文
	RecentConversation string `json:"recentConversation"`
	SessionTopic       string `json:"sessionTopic"`

	// 项目上下文
	CurrentProject   string `json:"currentProject"`
	WorkspaceContext string `json:"workspaceContext"`

	// 历史上下文
	RelevantHistory string `json:"relevantHistory"`
	UserPreferences string `json:"userPreferences"`

	// 技术上下文
	TechStack   []string `json:"techStack"`
	CurrentTask string   `json:"currentTask"`

	// 兼容性字段
	ShortTermMemory string `json:"shortTermMemory,omitempty"`
	LongTermMemory  string `json:"longTermMemory,omitempty"`
	SessionState    string `json:"sessionState,omitempty"`
}
