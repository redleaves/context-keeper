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
