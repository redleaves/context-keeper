package models

import (
	"context"
)

// VectorStore 向量存储抽象接口
// 提供统一的向量数据库操作接口，支持多厂商实现
type VectorStore interface {
	// EmbeddingProvider 文本转向量能力
	EmbeddingProvider
	// MemoryStorage 记忆存储能力
	MemoryStorage
	// VectorSearcher 向量搜索能力
	VectorSearcher
	// CollectionManager 集合管理能力
	CollectionManager
	// UserDataStorage 用户数据存储能力
	UserDataStorage

	// GetProvider 获取向量存储提供商类型
	GetProvider() VectorStoreType
}

// EmbeddingProvider 文本转向量接口
type EmbeddingProvider interface {
	// GenerateEmbedding 将文本转换为向量表示
	GenerateEmbedding(text string) ([]float32, error)

	// GetEmbeddingDimension 获取向量维度
	GetEmbeddingDimension() int
}

// MemoryStorage 记忆存储接口
type MemoryStorage interface {
	// StoreMemory 存储记忆到向量数据库
	StoreMemory(memory *Memory) error

	// StoreMessage 存储消息到向量数据库
	StoreMessage(message *Message) error

	// CountMemories 统计指定会话的记忆数量
	CountMemories(sessionID string) (int, error)

	// StoreEnhancedMemory 存储增强的多维度记忆（新增方法）
	StoreEnhancedMemory(memory *EnhancedMemory) error

	// StoreEnhancedMessage 存储增强的多维度消息（新增方法）
	StoreEnhancedMessage(message *EnhancedMessage) error
}

// VectorSearcher 向量搜索接口
type VectorSearcher interface {
	// SearchByVector 使用向量进行相似度搜索
	SearchByVector(ctx context.Context, vector []float32, options *SearchOptions) ([]SearchResult, error)

	// SearchByText 使用文本进行搜索（内部转换为向量）
	SearchByText(ctx context.Context, query string, options *SearchOptions) ([]SearchResult, error)

	// SearchByID 根据ID精确搜索
	SearchByID(ctx context.Context, id string, options *SearchOptions) ([]SearchResult, error)

	// SearchByFilter 根据过滤条件搜索
	SearchByFilter(ctx context.Context, filter string, options *SearchOptions) ([]SearchResult, error)
}

// CollectionManager 集合管理接口
type CollectionManager interface {
	// EnsureCollection 确保集合存在，不存在则创建
	EnsureCollection(collectionName string) error

	// CreateCollection 创建新集合
	CreateCollection(name string, config *CollectionConfig) error

	// DeleteCollection 删除集合
	DeleteCollection(name string) error

	// CollectionExists 检查集合是否存在
	CollectionExists(name string) (bool, error)
}

// UserDataStorage 用户数据存储接口
type UserDataStorage interface {
	// StoreUserInfo 存储用户信息
	StoreUserInfo(userInfo *UserInfo) error

	// GetUserInfo 获取用户信息
	GetUserInfo(userID string) (*UserInfo, error)

	// CheckUserExists 检查用户是否存在
	CheckUserExists(userID string) (bool, error)

	// InitUserStorage 初始化用户存储
	InitUserStorage() error
}

// SearchOptions 搜索选项配置
type SearchOptions struct {
	// Limit 结果数量限制
	Limit int `json:"limit,omitempty"`

	// SessionID 会话ID过滤
	SessionID string `json:"sessionId,omitempty"`

	// UserID 用户ID过滤
	UserID string `json:"userId,omitempty"`

	// SkipThreshold 是否跳过相似度阈值过滤
	SkipThreshold bool `json:"skipThreshold,omitempty"`

	// IsBruteSearch 是否启用暴力搜索（用于索引未训练的情况）
	IsBruteSearch int `json:"isBruteSearch,omitempty"`

	// Fields 返回的字段列表
	Fields []string `json:"fields,omitempty"`

	// ExtraFilters 额外的过滤条件
	ExtraFilters map[string]interface{} `json:"extraFilters,omitempty"`

	// SortBy 排序字段
	SortBy string `json:"sortBy,omitempty"`

	// SortOrder 排序方向 (asc/desc)
	SortOrder string `json:"sortOrder,omitempty"`
}

// CollectionConfig 集合配置
type CollectionConfig struct {
	// Dimension 向量维度
	Dimension int `json:"dimension"`

	// Metric 距离度量方式 (inner_product, euclidean) - Vearch主要支持这两种
	Metric string `json:"metric"`

	// Description 集合描述
	Description string `json:"description,omitempty"`

	// IndexType 索引类型
	IndexType string `json:"indexType,omitempty"`

	// ExtraConfig 厂商特定的额外配置
	ExtraConfig map[string]interface{} `json:"extraConfig,omitempty"`
}

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	// Provider 提供商类型 (aliyun, tencent, openai, etc.)
	Provider string `json:"provider"`

	// EmbeddingConfig 文本转向量配置
	EmbeddingConfig *EmbeddingConfig `json:"embeddingConfig"`

	// DatabaseConfig 向量数据库配置
	DatabaseConfig *DatabaseConfig `json:"databaseConfig"`

	// DefaultCollection 默认集合名称
	DefaultCollection string `json:"defaultCollection"`

	// SimilarityThreshold 默认相似度阈值
	SimilarityThreshold float64 `json:"similarityThreshold"`
}

// EmbeddingConfig 文本转向量配置
type EmbeddingConfig struct {
	// APIEndpoint API端点
	APIEndpoint string `json:"apiEndpoint"`

	// APIKey API密钥
	APIKey string `json:"apiKey"`

	// Model 模型名称
	Model string `json:"model"`

	// Dimension 向量维度
	Dimension int `json:"dimension"`

	// ExtraParams 额外参数
	ExtraParams map[string]interface{} `json:"extraParams,omitempty"`
}

// DatabaseConfig 向量数据库配置
type DatabaseConfig struct {
	// Endpoint 数据库端点
	Endpoint string `json:"endpoint"`

	// APIKey API密钥
	APIKey string `json:"apiKey"`

	// Collection 集合名称
	Collection string `json:"collection"`

	// Metric 距离度量方式
	Metric string `json:"metric"`

	// ExtraParams 额外参数
	ExtraParams map[string]interface{} `json:"extraParams,omitempty"`
}

// VectorStoreType 向量存储类型枚举
type VectorStoreType string

const (
	// VectorStoreTypeAliyun 阿里云向量存储
	VectorStoreTypeAliyun VectorStoreType = "aliyun"

	// VectorStoreTypeVearch 京东云Vearch/开源Vearch向量存储
	VectorStoreTypeVearch VectorStoreType = "vearch"

	// VectorStoreTypeTencent 腾讯云向量存储
	VectorStoreTypeTencent VectorStoreType = "tencent"

	// VectorStoreTypeOpenAI OpenAI向量存储
	VectorStoreTypeOpenAI VectorStoreType = "openai"

	// VectorStoreTypePinecone Pinecone向量存储
	VectorStoreTypePinecone VectorStoreType = "pinecone"

	// VectorStoreTypeWeaviate Weaviate向量存储
	VectorStoreTypeWeaviate VectorStoreType = "weaviate"

	// VectorStoreTypeLocal 本地向量存储
	VectorStoreTypeLocal VectorStoreType = "local"
)

// String 返回向量存储类型的字符串表示
func (vt VectorStoreType) String() string {
	return string(vt)
}

// IsValid 检查向量存储类型是否有效
func (vt VectorStoreType) IsValid() bool {
	switch vt {
	case VectorStoreTypeAliyun, VectorStoreTypeVearch, VectorStoreTypeTencent, VectorStoreTypeOpenAI,
		VectorStoreTypePinecone, VectorStoreTypeWeaviate, VectorStoreTypeLocal:
		return true
	default:
		return false
	}
}

// GetSupportedVectorStoreTypes 获取支持的向量存储类型列表
func GetSupportedVectorStoreTypes() []VectorStoreType {
	return []VectorStoreType{
		VectorStoreTypeAliyun,
		VectorStoreTypeVearch,
		VectorStoreTypeTencent,
		VectorStoreTypeOpenAI,
		VectorStoreTypePinecone,
		VectorStoreTypeWeaviate,
		VectorStoreTypeLocal,
	}
}

// EnhancedMemory 增强的多维度记忆模型（兼容现有Memory模型）
type EnhancedMemory struct {
	// 继承现有Memory的所有字段（完全兼容）
	*Memory

	// 新增多维度字段
	SemanticVector   []float32              `json:"semantic_vector,omitempty"`    // 语义向量
	ContextVector    []float32              `json:"context_vector,omitempty"`     // 上下文向量
	TimeVector       []float32              `json:"time_vector,omitempty"`        // 时间向量
	DomainVector     []float32              `json:"domain_vector,omitempty"`      // 领域向量
	SemanticTags     []string               `json:"semantic_tags,omitempty"`      // 语义标签
	ConceptEntities  []string               `json:"concept_entities,omitempty"`   // 概念实体
	RelatedConcepts  []string               `json:"related_concepts,omitempty"`   // 相关概念
	ImportanceScore  float64                `json:"importance_score,omitempty"`   // 重要性评分
	RelevanceScore   float64                `json:"relevance_score,omitempty"`    // 相关性评分
	ContextSummary   string                 `json:"context_summary,omitempty"`    // 上下文摘要
	TechStack        []string               `json:"tech_stack,omitempty"`         // 技术栈
	ProjectContext   string                 `json:"project_context,omitempty"`    // 项目上下文
	EventType        string                 `json:"event_type,omitempty"`         // 事件类型
	MultiDimMetadata map[string]interface{} `json:"multi_dim_metadata,omitempty"` // 多维度元数据
}

// EnhancedMessage 增强的多维度消息模型（兼容现有Message模型）
type EnhancedMessage struct {
	// 继承现有Message的所有字段（完全兼容）
	*Message

	// 新增多维度字段（与EnhancedMemory保持一致）
	SemanticVector   []float32              `json:"semantic_vector,omitempty"`
	ContextVector    []float32              `json:"context_vector,omitempty"`
	TimeVector       []float32              `json:"time_vector,omitempty"`
	DomainVector     []float32              `json:"domain_vector,omitempty"`
	SemanticTags     []string               `json:"semantic_tags,omitempty"`
	ConceptEntities  []string               `json:"concept_entities,omitempty"`
	RelatedConcepts  []string               `json:"related_concepts,omitempty"`
	ImportanceScore  float64                `json:"importance_score,omitempty"`
	RelevanceScore   float64                `json:"relevance_score,omitempty"`
	ContextSummary   string                 `json:"context_summary,omitempty"`
	TechStack        []string               `json:"tech_stack,omitempty"`
	ProjectContext   string                 `json:"project_context,omitempty"`
	EventType        string                 `json:"event_type,omitempty"`
	MultiDimMetadata map[string]interface{} `json:"multi_dim_metadata,omitempty"`
}

// MultiDimensionalVectors 多维度向量结构
type MultiDimensionalVectors struct {
	// 多维度向量
	SemanticVector []float32 `json:"semantic_vector,omitempty"` // 语义向量
	ContextVector  []float32 `json:"context_vector,omitempty"`  // 上下文向量
	TimeVector     []float32 `json:"time_vector,omitempty"`     // 时间向量
	DomainVector   []float32 `json:"domain_vector,omitempty"`   // 领域向量

	// 分析结果
	SemanticTags    []string `json:"semantic_tags,omitempty"`    // 语义标签
	ConceptEntities []string `json:"concept_entities,omitempty"` // 概念实体
	RelatedConcepts []string `json:"related_concepts,omitempty"` // 相关概念
	ImportanceScore float64  `json:"importance_score,omitempty"` // 重要性评分
	RelevanceScore  float64  `json:"relevance_score,omitempty"`  // 相关性评分
	ContextSummary  string   `json:"context_summary,omitempty"`  // 上下文摘要
	TechStack       []string `json:"tech_stack,omitempty"`       // 技术栈
	ProjectContext  string   `json:"project_context,omitempty"`  // 项目上下文
	EventType       string   `json:"event_type,omitempty"`       // 事件类型
}

// LLMAnalysisResult LLM分析结果
type LLMAnalysisResult struct {
	// 分析摘要
	SemanticSummary string `json:"semantic_summary"` // 语义摘要
	ContextSummary  string `json:"context_summary"`  // 上下文摘要
	TimeFeatures    string `json:"time_features"`    // 时间特征
	DomainFeatures  string `json:"domain_features"`  // 领域特征

	// 提取的信息
	Keywords        []string `json:"keywords"`         // 关键词
	ConceptEntities []string `json:"concept_entities"` // 概念实体
	RelatedConcepts []string `json:"related_concepts"` // 相关概念
	TechStack       []string `json:"tech_stack"`       // 技术栈

	// 评分
	ImportanceScore float64 `json:"importance_score"` // 重要性评分
	RelevanceScore  float64 `json:"relevance_score"`  // 相关性评分

	// 分类
	EventType      string `json:"event_type"`      // 事件类型
	ProjectContext string `json:"project_context"` // 项目上下文
}

// MultiDimensionalAnalysisResult 多维度分析结果（重新设计）
type MultiDimensionalAnalysisResult struct {
	TimelineData       *TimelineData       `json:"timeline_data"`
	KnowledgeGraphData *KnowledgeGraphData `json:"knowledge_graph_data"`
	VectorData         *VectorData         `json:"vector_data"`
	MetaAnalysis       *MetaAnalysis       `json:"meta_analysis"`
}

// TimelineData 时间线故事性数据
type TimelineData struct {
	StoryTitle      string   `json:"story_title"`
	StorySummary    string   `json:"story_summary"`
	KeyEvents       []string `json:"key_events"`
	TimeSequence    string   `json:"time_sequence"`
	Outcome         string   `json:"outcome"`
	LessonsLearned  string   `json:"lessons_learned"`
	ImportanceLevel int      `json:"importance_level"`
}

// KnowledgeGraphData 知识图谱数据
type KnowledgeGraphData struct {
	MainConcepts    []Concept      `json:"main_concepts"`
	Relationships   []Relationship `json:"relationships"`
	Domain          string         `json:"domain"`
	ComplexityLevel string         `json:"complexity_level"`
}

// Concept 概念实体
type Concept struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Importance float64 `json:"importance"`
}

// Relationship 关系
type Relationship struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Relation string  `json:"relation"`
	Strength float64 `json:"strength"`
}

// VectorData 向量数据
type VectorData struct {
	SemanticCore     string   `json:"semantic_core"`
	ContextInfo      string   `json:"context_info"`
	SearchKeywords   []string `json:"search_keywords"`
	SemanticTags     []string `json:"semantic_tags"`
	RelevanceContext string   `json:"relevance_context"`
}

// MetaAnalysis 元分析数据
type MetaAnalysis struct {
	ContentType    string   `json:"content_type"`
	Priority       string   `json:"priority"`
	TechStack      []string `json:"tech_stack"`
	BusinessValue  float64  `json:"business_value"`
	ReusePotential float64  `json:"reuse_potential"`
}
