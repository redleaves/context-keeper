package multi_dimensional_storage

import (
	"time"
)

// MultiDimensionalStorageConfig 多维度存储配置
type MultiDimensionalStorageConfig struct {
	Enabled          bool          `json:"enabled"`            // 总开关
	TimelineEnabled  bool          `json:"timeline_enabled"`   // 时间线存储开关
	KnowledgeEnabled bool          `json:"knowledge_enabled"`  // 知识图谱存储开关
	VectorEnabled    bool          `json:"vector_enabled"`     // 向量存储开关
	FallbackToLegacy bool          `json:"fallback_to_legacy"` // 失败时回退到原有逻辑
	LLMProvider      string        `json:"llm_provider"`       // LLM提供商
	LLMModel         string        `json:"llm_model"`          // LLM模型
	MaxRetries       int           `json:"max_retries"`        // 最大重试次数
	Timeout          time.Duration `json:"timeout"`            // 超时时间
}

// StorageRequest 存储请求
type StorageRequest struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	WorkspaceID string                 `json:"workspace_id"`
	Query       string                 `json:"query"`    // 原始查询/命令
	Context     string                 `json:"context"`  // 当前上下文
	Metadata    map[string]interface{} `json:"metadata"` // 额外元数据
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id"`
}

// LLMAnalysisResult LLM分析结果
type LLMAnalysisResult struct {
	TimelineData          *TimelineData          `json:"timeline_data"`
	KnowledgeGraphData    *KnowledgeGraphData    `json:"knowledge_graph_data"`
	VectorData            *VectorData            `json:"vector_data"`
	StorageRecommendation *StorageRecommendation `json:"storage_recommendation"`
}

// TimelineData 时间线数据
type TimelineData struct {
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	EventType       string   `json:"event_type"`
	Keywords        []string `json:"keywords"`
	ImportanceScore int      `json:"importance_score"`
	TechStack       []string `json:"tech_stack"`
	ProjectContext  string   `json:"project_context"`
}

// KnowledgeGraphData 知识图谱数据
type KnowledgeGraphData struct {
	Concepts      []Concept      `json:"concepts"`
	Relationships []Relationship `json:"relationships"`
}

// Concept 概念
type Concept struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Importance float64                `json:"importance"`
}

// Relationship 关系
type Relationship struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Type        string  `json:"type"`
	Strength    float64 `json:"strength"`
	Description string  `json:"description"`
}

// VectorData 向量数据
type VectorData struct {
	Content        string   `json:"content"`
	SemanticTags   []string `json:"semantic_tags"`
	ContextSummary string   `json:"context_summary"`
	RelevanceScore float64  `json:"relevance_score"`
}

// StorageRecommendation 存储推荐
type StorageRecommendation struct {
	TimelinePriority  float64 `json:"timeline_priority"`
	KnowledgePriority float64 `json:"knowledge_priority"`
	VectorPriority    float64 `json:"vector_priority"`
	Reasoning         string  `json:"reasoning"`
}

// StorageResult 存储结果
type StorageResult struct {
	Success         bool                   `json:"success"`
	TimelineStored  bool                   `json:"timeline_stored"`
	KnowledgeStored bool                   `json:"knowledge_stored"`
	VectorStored    bool                   `json:"vector_stored"`
	StoredIDs       map[string]string      `json:"stored_ids"` // 各存储引擎返回的ID
	Errors          []string               `json:"errors"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	LLMAnalysisTime time.Duration          `json:"llm_analysis_time"`
	StorageTime     time.Duration          `json:"storage_time"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// StorageEngine 存储引擎接口
type StorageEngine interface {
	// Store 存储数据
	Store(request *StorageRequest, analysisResult *LLMAnalysisResult) (*StorageResult, error)

	// IsEnabled 检查是否启用
	IsEnabled() bool

	// GetConfig 获取配置
	GetConfig() *MultiDimensionalStorageConfig

	// Close 关闭引擎
	Close() error
}

// LLMAnalyzer LLM分析器接口
type LLMAnalyzer interface {
	// Analyze 分析请求，返回结构化数据
	Analyze(request *StorageRequest) (*LLMAnalysisResult, error)

	// IsAvailable 检查LLM是否可用
	IsAvailable() bool
}

// TimelineStorageAdapter 时间线存储适配器接口
type TimelineStorageAdapter interface {
	StoreTimelineData(userID, sessionID string, data *TimelineData) (string, error)
}

// KnowledgeStorageAdapter 知识图谱存储适配器接口
type KnowledgeStorageAdapter interface {
	StoreKnowledgeData(userID, sessionID string, data *KnowledgeGraphData) (string, error)
}

// VectorStorageAdapter 向量存储适配器接口
type VectorStorageAdapter interface {
	StoreVectorData(userID, sessionID string, data *VectorData) (string, error)
}

// QualityValidator 质量验证器接口
type QualityValidator interface {
	// ValidateTimelineData 验证时间线数据质量
	ValidateTimelineData(data *TimelineData) (bool, []string)

	// ValidateKnowledgeData 验证知识图谱数据质量
	ValidateKnowledgeData(data *KnowledgeGraphData) (bool, []string)

	// ValidateVectorData 验证向量数据质量
	ValidateVectorData(data *VectorData) (bool, []string)
}

// StorageMetrics 存储指标
type StorageMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulStores    int64         `json:"successful_stores"`
	FailedStores        int64         `json:"failed_stores"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	TimelineStoreCount  int64         `json:"timeline_store_count"`
	KnowledgeStoreCount int64         `json:"knowledge_store_count"`
	VectorStoreCount    int64         `json:"vector_store_count"`
	LLMAnalysisErrors   int64         `json:"llm_analysis_errors"`
	StorageErrors       int64         `json:"storage_errors"`
}

// PromptTemplate Prompt模板
type PromptTemplate struct {
	Template    string            `json:"template"`
	Variables   map[string]string `json:"variables"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
}
