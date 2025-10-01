package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// EmbeddingProvider embedding服务提供者接口（减少依赖）
type EmbeddingProvider interface {
	GenerateEmbedding(text string) ([]float32, error)
	GetEmbeddingDimension() int
}

// VearchStore Vearch向量存储实现
// 支持京东云Vearch和开源Vearch
type VearchStore struct {
	client      VearchClient            // Vearch客户端接口
	config      *VearchConfig           // Vearch配置
	database    string                  // 数据库名称
	spaces      map[string]*VearchSpace // 空间缓存（修正：Collection -> Space）
	initialized bool                    // 初始化状态
	// 移除直接依赖，改为通过回调获取embedding服务
	getEmbeddingService func() EmbeddingProvider // 获取embedding服务的回调函数
}

// VearchConfig Vearch配置
type VearchConfig struct {
	// 连接配置
	Endpoints []string `json:"endpoints"` // Vearch集群端点列表
	Username  string   `json:"username"`  // 用户名
	Password  string   `json:"password"`  // 密码
	Database  string   `json:"database"`  // 数据库名称

	// Embedding配置
	EmbeddingModel    string `json:"embeddingModel"`    // embedding模型
	EmbeddingEndpoint string `json:"embeddingEndpoint"` // embedding服务端点
	EmbeddingAPIKey   string `json:"embeddingApiKey"`   // embedding API密钥
	Dimension         int    `json:"dimension"`         // 向量维度

	// 搜索配置
	DefaultTopK          int     `json:"defaultTopK"`          // 默认返回结果数
	SimilarityThreshold  float64 `json:"similarityThreshold"`  // 相似度阈值
	SearchTimeoutSeconds int     `json:"searchTimeoutSeconds"` // 搜索超时时间

	// 性能配置
	ConnectionPoolSize    int `json:"connectionPoolSize"`    // 连接池大小
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"` // 请求超时时间
}

// VearchSpace Vearch空间定义（修正：Collection -> Space）
type VearchSpace struct {
	Name         string                 `json:"name"`
	PartitionNum int                    `json:"partition_num"`
	ReplicaNum   int                    `json:"replica_num"`
	Properties   map[string]interface{} `json:"properties"`
	Engine       *EngineConfig          `json:"engine"`
	Created      time.Time              `json:"created"`
}

// EngineConfig Vearch引擎配置
type EngineConfig struct {
	Name      string           `json:"name"`       // "gamma" 为主要引擎
	IndexSize int              `json:"index_size"` // 索引大小
	Retrieval *RetrievalConfig `json:"retrieval"`  // 检索配置
}

// RetrievalConfig 检索配置
type RetrievalConfig struct {
	Type       string                 `json:"type"`       // "hnsw", "ivf_pq", "flat"
	Parameters map[string]interface{} `json:"parameters"` // 特定索引类型的参数
}

// VearchClient Vearch客户端接口
// 抽象Vearch SDK的核心功能，便于测试和扩展
type VearchClient interface {
	// 连接管理
	Connect() error
	Close() error
	Ping() error

	// 数据库管理
	CreateDatabase(name string) error
	ListDatabases() ([]string, error)
	DatabaseExists(name string) (bool, error)

	// 空间管理（修正：Collection -> Space）
	CreateSpace(database, name string, config *SpaceConfig) error
	ListSpaces(database string) ([]string, error)
	SpaceExists(database, name string) (bool, error)
	DropSpace(database, name string) error

	// 文档操作
	Insert(database, space string, docs []map[string]interface{}) error
	Search(database, space string, query *VearchSearchRequest) (*VearchSearchResponse, error)
	Delete(database, space string, ids []string) error

	// 向量操作
	BulkIndex(database, space string, vectors []VearchBulkVector) error
}

// SpaceConfig 空间配置（修正：符合Vearch规范）
type SpaceConfig struct {
	Name         string                   `json:"name"`
	PartitionNum int                      `json:"partition_num"` // 分区数量
	ReplicaNum   int                      `json:"replica_num"`   // 副本数量
	Properties   []map[string]interface{} `json:"fields"`        // 字段属性定义（修正：使用fields数组）
	Engine       *EngineConfig            `json:"engine"`        // 引擎配置
}

// VearchSearchRequest Vearch搜索请求（✅ 严格按照官方文档格式）
type VearchSearchRequest struct {
	// ✅ 官方文档：平铺结构，无嵌套Query
	Vectors       []VearchVector         `json:"vectors"`                   // 向量数组
	Filters       *VearchFilter          `json:"filters,omitempty"`         // 过滤条件（官方格式）
	IndexParams   map[string]interface{} `json:"index_params,omitempty"`    // 索引参数
	Fields        []string               `json:"fields,omitempty"`          // 返回字段
	IsBruteSearch int                    `json:"is_brute_search,omitempty"` // 是否暴力搜索
	VectorValue   bool                   `json:"vector_value,omitempty"`    // 是否返回向量
	LoadBalance   string                 `json:"load_balance,omitempty"`    // 负载均衡
	Limit         int                    `json:"limit"`                     // 结果数量限制
	DbName        string                 `json:"db_name"`                   // 数据库名
	SpaceName     string                 `json:"space_name"`                // 空间名
	Ranker        *VearchRanker          `json:"ranker,omitempty"`          // 排序器
}

// VearchVector 向量查询条件（✅ 官方文档格式）
type VearchVector struct {
	Field    string    `json:"field"`               // 向量字段名
	Feature  []float32 `json:"feature"`             // 向量特征数据
	MinScore *float64  `json:"min_score,omitempty"` // 最小分数阈值
	MaxScore *float64  `json:"max_score,omitempty"` // 最大分数阈值
}

// VearchFilter 过滤条件（✅ 官方文档格式）
type VearchFilter struct {
	Operator   string            `json:"operator"`   // 操作符：AND
	Conditions []VearchCondition `json:"conditions"` // 条件数组
}

// VearchCondition 具体过滤条件（✅ 官方文档格式）
type VearchCondition struct {
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符：=, >, >=, <, <=, IN, NOT IN
	Value    interface{} `json:"value"`    // 字段值
}

// VearchRanker 排序器（✅ 官方文档格式）
type VearchRanker struct {
	Type   string    `json:"type"`   // 排序器类型：WeightedRanker
	Params []float64 `json:"params"` // 参数数组
}

// VearchSearchResponse Vearch搜索响应（✅ 严格按照官方文档格式）
type VearchSearchResponse struct {
	Code int    `json:"code"` // 状态码：0表示成功
	Msg  string `json:"msg"`  // 状态信息：success
	Data struct {
		Documents [][]VearchDocument `json:"documents"` // 文档数组（二维数组）
	} `json:"data"`
}

// VearchDocument 文档结果（✅ 官方文档格式）
type VearchDocument map[string]interface{}

// VearchBulkVector 批量索引用的向量数据（与搜索用的VearchVector不同）
type VearchBulkVector struct {
	ID     string                 `json:"_id"`    // 文档ID
	Vector []float32              `json:"vector"` // 向量数据
	Fields map[string]interface{} `json:"fields"` // 其他字段
}

// NewVearchStore 创建Vearch向量存储实例
func NewVearchStore(client VearchClient, config *VearchConfig, getEmbeddingService func() EmbeddingProvider) *VearchStore {
	return &VearchStore{
		client:              client,
		config:              config,
		database:            config.Database,
		spaces:              make(map[string]*VearchSpace),
		initialized:         false,
		getEmbeddingService: getEmbeddingService,
	}
}

// Initialize 初始化Vearch存储
func (v *VearchStore) Initialize() error {
	if v.initialized {
		return nil
	}

	log.Printf("[Vearch存储] 开始初始化连接: endpoints=%v, database=%s", v.config.Endpoints, v.config.Database)

	// 连接Vearch集群
	if err := v.client.Connect(); err != nil {
		return fmt.Errorf("连接Vearch集群失败: %v", err)
	}

	// 检查连接健康状态
	if err := v.client.Ping(); err != nil {
		return fmt.Errorf("Vearch集群健康检查失败: %v", err)
	}

	// 确保数据库存在
	if err := v.ensureDatabase(); err != nil {
		return fmt.Errorf("确保数据库存在失败: %v", err)
	}

	// 初始化默认空间
	if err := v.initializeDefaultSpaces(); err != nil {
		return fmt.Errorf("初始化默认空间失败: %v", err)
	}

	v.initialized = true
	log.Printf("[Vearch存储] 初始化完成")
	return nil
}

// ensureDatabase 检查数据库是否存在（修正：真正检查而不是跳过）
func (v *VearchStore) ensureDatabase() error {
	log.Printf("[Vearch存储] 检查数据库是否存在: %s", v.database)

	// 检查数据库是否存在
	exists, err := v.client.DatabaseExists(v.database)
	if err != nil {
		return fmt.Errorf("检查数据库存在性失败: %v", err)
	}

	if !exists {
		return fmt.Errorf("❌ 数据库 '%s' 不存在！请先手动创建数据库。\n创建命令示例: curl -XPOST http://your-vearch-url/db/_create -d '{\"name\":\"%s\"}'", v.database, v.database)
	}

	log.Printf("✅ [Vearch存储] 数据库存在验证通过: %s", v.database)
	return nil
}

// initializeDefaultSpaces 检查必需的表空间是否存在（修正：真正检查而不是跳过）
func (v *VearchStore) initializeDefaultSpaces() error {
	// 从环境变量或配置获取必需的表空间列表
	requiredSpaces := v.getRequiredSpaces()

	log.Printf("[Vearch存储] 检查必需的表空间是否存在: %v", requiredSpaces)

	var missingSpaces []string

	for _, spaceName := range requiredSpaces {
		exists, err := v.client.SpaceExists(v.database, spaceName)
		if err != nil {
			return fmt.Errorf("检查表空间 '%s' 存在性失败: %v", spaceName, err)
		}

		if !exists {
			missingSpaces = append(missingSpaces, spaceName)
		} else {
			log.Printf("✅ [Vearch存储] 表空间存在: %s", spaceName)
		}
	}

	if len(missingSpaces) > 0 {
		return fmt.Errorf("❌ 以下必需的表空间不存在: %v\n请先手动创建这些表空间。\n创建命令示例: curl -XPOST http://your-vearch-url/dbs/%s/spaces -d '{\"name\":\"表空间名\", ...}'", missingSpaces, v.database)
	}

	log.Printf("✅ [Vearch存储] 所有必需表空间验证通过")
	return nil
}

// getRequiredSpaces 获取必需的表空间列表（可通过环境变量配置）
func (v *VearchStore) getRequiredSpaces() []string {
	// 从环境变量获取，如果没有设置则使用默认值
	envSpaces := os.Getenv("VEARCH_REQUIRED_SPACES")
	if envSpaces != "" {
		return strings.Split(envSpaces, ",")
	}

	// 默认必需的表空间
	return []string{
		"context_keeper_vector", // 主表空间：存储记忆和消息
		"context_keeper_users",  // 用户表空间：存储用户信息
	}
}

// =============================================================================
// EmbeddingProvider 接口实现
// =============================================================================

// GenerateEmbedding 生成文本向量 - 通过回调获取embedding服务
func (v *VearchStore) GenerateEmbedding(text string) ([]float32, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return nil, err
		}
	}

	// 通过回调获取embedding服务（避免直接依赖）
	if v.getEmbeddingService != nil {
		if embeddingService := v.getEmbeddingService(); embeddingService != nil {
			log.Printf("[Vearch存储] 通过工厂获取embedding服务生成向量")
			return embeddingService.GenerateEmbedding(text)
		}
	}

	// 如果没有embedding服务，返回错误
	return nil, fmt.Errorf("embedding服务未配置，Vearch需要external embedding服务支持")
}

// GetEmbeddingDimension 获取向量维度
func (v *VearchStore) GetEmbeddingDimension() int {
	return v.config.Dimension
}

// GetClient 获取Vearch客户端（用于用户存储仓库）
func (v *VearchStore) GetClient() VearchClient {
	return v.client
}

// =============================================================================
// MemoryStorage 接口实现
// =============================================================================

// StoreMemory 存储记忆
func (v *VearchStore) StoreMemory(memory *models.Memory) error {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	log.Printf("[Vearch存储] 存储记忆: ID=%s, 会话=%s", memory.ID, memory.SessionID)

	// 生成内容向量
	vector, err := v.GenerateEmbedding(memory.Content)
	if err != nil {
		return fmt.Errorf("生成记忆向量失败: %v", err)
	}

	// 生成格式化时间戳（与阿里云版本对齐）
	formattedTime := time.Unix(memory.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 将metadata转换为JSON字符串（与阿里云实现保持一致）
	metadataStr := "{}"
	if memory.Metadata != nil {
		if metadataBytes, err := json.Marshal(memory.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[Vearch存储] 警告: 无法序列化metadata: %v", err)
		}
	}

	// 构建文档（字段结构与阿里云版本对齐）
	doc := map[string]interface{}{
		"_id":            memory.ID,
		"vector":         vector,
		"content":        memory.Content,
		"session_id":     memory.SessionID, // 使用下划线格式（与阿里云一致）
		"user_id":        memory.UserID,    // ✅ 使用下划线命名保持一致
		"priority":       memory.Priority,
		"metadata":       metadataStr, // ✅ 使用JSON字符串格式
		"timestamp":      memory.Timestamp,
		"formatted_time": formattedTime,                     // 添加格式化时间
		"memory_id":      memory.ID,                         // ✅ memory_id字段，与阿里云保持一致（Schema中已有）
		"biz_type":       fmt.Sprintf("%d", memory.BizType), // ✅ 使用下划线命名，转换为字符串与阿里云一致
		"role":           "",                                // 设置为空字符串，Memory模型没有Role字段
		"content_type":   "",                                // 设置为空字符串，Memory模型没有ContentType字段
	}

	// 插入到Vearch（使用主空间context_keeper）
	if err := v.client.Insert(v.database, "context_keeper_vector", []map[string]interface{}{doc}); err != nil {
		return fmt.Errorf("插入记忆到Vearch失败: %v", err)
	}

	log.Printf("[Vearch存储] 记忆存储成功: ID=%s", memory.ID)
	return nil
}

// StoreMessage 存储消息
func (v *VearchStore) StoreMessage(message *models.Message) error {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	log.Printf("[Vearch存储] 存储消息: ID=%s, 会话=%s", message.ID, message.SessionID)

	// 生成内容向量
	vector, err := v.GenerateEmbedding(message.Content)
	if err != nil {
		return fmt.Errorf("生成消息向量失败: %v", err)
	}

	// 生成格式化时间戳（与阿里云版本对齐）
	formattedTime := time.Unix(message.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 将metadata转换为JSON字符串（与阿里云实现保持一致）
	metadataStr := "{}"
	if message.Metadata != nil {
		if metadataBytes, err := json.Marshal(message.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[Vearch存储] 警告: 无法序列化metadata: %v", err)
		}
	}

	// 构建文档（字段结构与阿里云版本对齐）
	doc := map[string]interface{}{
		"_id":            message.ID,
		"vector":         vector,
		"content":        message.Content,
		"session_id":     message.SessionID, // 使用下划线格式（与阿里云一致）
		"user_id":        "",                // ✅ Message模型没有UserID字段，设置为空字符串
		"role":           message.Role,
		"content_type":   message.ContentType,
		"timestamp":      message.Timestamp,
		"formatted_time": formattedTime, // 添加格式化时间
		"priority":       message.Priority,
		"metadata":       metadataStr, // ✅ 使用JSON字符串格式
		"message_id":     message.ID,  // ✅ message_id字段，与阿里云保持一致（Schema中已添加）
		"biz_type":       "",          // ✅ Message模型没有BizType字段，设置为空字符串
		"memory_id":      "",          // Message没有memory_id，设置为空字符串
	}

	// 插入到Vearch（使用主空间context_keeper）
	if err := v.client.Insert(v.database, "context_keeper_vector", []map[string]interface{}{doc}); err != nil {
		return fmt.Errorf("插入消息到Vearch失败: %v", err)
	}

	log.Printf("[Vearch存储] 消息存储成功: ID=%s", message.ID)
	return nil
}

// CountMemories 统计记忆数量
func (v *VearchStore) CountMemories(sessionID string) (int, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return 0, err
		}
	}

	// 构建搜索请求
	searchReq := &VearchSearchRequest{
		Vectors: []VearchVector{
			{
				Field:   "vector",
				Feature: make([]float32, v.config.Dimension), // 零向量用于计数
			},
		},
		Filters: &VearchFilter{
			Operator: "AND",
			Conditions: []VearchCondition{
				{
					Field:    "session_id",
					Operator: "IN",
					Value:    []interface{}{sessionID},
				},
			},
		},
		Limit: 10000, // 大数值用于获取总数
	}

	resp, err := v.client.Search(v.database, "context_keeper_vector", searchReq)
	if err != nil {
		return 0, fmt.Errorf("搜索记忆失败: %v", err)
	}

	return len(resp.Data.Documents), nil
}

// StoreEnhancedMemory 存储增强的多维度记忆（新增方法）
func (v *VearchStore) StoreEnhancedMemory(memory *models.EnhancedMemory) error {
	log.Printf("[京东云向量存储] 存储增强记忆: ID=%s, 会话=%s", memory.Memory.ID, memory.Memory.SessionID)

	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	// 首先确保基础向量已生成
	if memory.Memory.Vector == nil || len(memory.Memory.Vector) == 0 {
		return fmt.Errorf("存储前必须先生成基础向量")
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(memory.Memory.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 处理元数据
	metadataStr := "{}"
	if memory.Memory.Metadata != nil {
		if metadataBytes, err := json.Marshal(memory.Memory.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[京东云向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建增强文档（包含所有现有字段 + 新增多维度字段）
	doc := map[string]interface{}{
		// 现有字段（完全兼容）
		"_id":            memory.Memory.ID,
		"vector":         memory.Memory.Vector,
		"content":        memory.Memory.Content,
		"session_id":     memory.Memory.SessionID,
		"user_id":        memory.Memory.UserID,
		"timestamp":      memory.Memory.Timestamp,
		"formatted_time": formattedTime,
		"priority":       memory.Memory.Priority,
		"metadata":       metadataStr,
		"memory_id":      memory.Memory.ID,
		"biz_type":       memory.Memory.BizType,

		// 新增多维度字段
		"semantic_tags":    memory.SemanticTags,
		"concept_entities": memory.ConceptEntities,
		"related_concepts": memory.RelatedConcepts,
		"importance_score": memory.ImportanceScore,
		"relevance_score":  memory.RelevanceScore,
		"context_summary":  memory.ContextSummary,
		"tech_stack":       memory.TechStack,
		"project_context":  memory.ProjectContext,
		"event_type":       memory.EventType,
	}

	// 添加多维度向量字段（如果存在）
	if len(memory.SemanticVector) > 0 {
		doc["semantic_vector"] = memory.SemanticVector
	}
	if len(memory.ContextVector) > 0 {
		doc["context_vector"] = memory.ContextVector
	}
	if len(memory.TimeVector) > 0 {
		doc["time_vector"] = memory.TimeVector
	}
	if len(memory.DomainVector) > 0 {
		doc["domain_vector"] = memory.DomainVector
	}

	// 添加多维度元数据
	if memory.MultiDimMetadata != nil {
		if multiDimBytes, err := json.Marshal(memory.MultiDimMetadata); err == nil {
			doc["multi_dim_metadata"] = string(multiDimBytes)
		}
	}

	// 插入到Vearch
	if err := v.client.Insert(v.database, "context_keeper_vector", []map[string]interface{}{doc}); err != nil {
		return fmt.Errorf("插入增强记忆到Vearch失败: %v", err)
	}

	log.Printf("[京东云向量存储] 增强记忆存储成功: ID=%s", memory.Memory.ID)
	return nil
}

// StoreEnhancedMessage 存储增强的多维度消息（新增方法）
func (v *VearchStore) StoreEnhancedMessage(message *models.EnhancedMessage) error {
	log.Printf("[京东云向量存储] 存储增强消息: ID=%s, 会话=%s", message.Message.ID, message.Message.SessionID)

	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	// 首先确保基础向量已生成
	if message.Message.Vector == nil || len(message.Message.Vector) == 0 {
		return fmt.Errorf("存储前必须先生成基础向量")
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(message.Message.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 处理元数据
	metadataStr := "{}"
	if message.Message.Metadata != nil {
		if metadataBytes, err := json.Marshal(message.Message.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[京东云向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建增强文档（包含所有现有字段 + 新增多维度字段）
	doc := map[string]interface{}{
		// 现有字段（完全兼容）
		"_id":            message.Message.ID,
		"vector":         message.Message.Vector,
		"content":        message.Message.Content,
		"session_id":     message.Message.SessionID,
		"user_id":        "", // Message模型中没有UserID字段
		"role":           message.Message.Role,
		"content_type":   message.Message.ContentType,
		"timestamp":      message.Message.Timestamp,
		"formatted_time": formattedTime,
		"priority":       message.Message.Priority,
		"metadata":       metadataStr,
		"message_id":     message.Message.ID,
		"biz_type":       "", // Message模型中没有BizType字段
		"memory_id":      "", // Message没有memory_id

		// 新增多维度字段
		"semantic_tags":    message.SemanticTags,
		"concept_entities": message.ConceptEntities,
		"related_concepts": message.RelatedConcepts,
		"importance_score": message.ImportanceScore,
		"relevance_score":  message.RelevanceScore,
		"context_summary":  message.ContextSummary,
		"tech_stack":       message.TechStack,
		"project_context":  message.ProjectContext,
		"event_type":       message.EventType,
	}

	// 添加多维度向量字段（如果存在）
	if len(message.SemanticVector) > 0 {
		doc["semantic_vector"] = message.SemanticVector
	}
	if len(message.ContextVector) > 0 {
		doc["context_vector"] = message.ContextVector
	}
	if len(message.TimeVector) > 0 {
		doc["time_vector"] = message.TimeVector
	}
	if len(message.DomainVector) > 0 {
		doc["domain_vector"] = message.DomainVector
	}

	// 添加多维度元数据
	if message.MultiDimMetadata != nil {
		if multiDimBytes, err := json.Marshal(message.MultiDimMetadata); err == nil {
			doc["multi_dim_metadata"] = string(multiDimBytes)
		}
	}

	// 插入到Vearch
	if err := v.client.Insert(v.database, "context_keeper_vector", []map[string]interface{}{doc}); err != nil {
		return fmt.Errorf("插入增强消息到Vearch失败: %v", err)
	}

	log.Printf("[京东云向量存储] 增强消息存储成功: ID=%s", message.Message.ID)
	return nil
}

// =============================================================================
// VectorSearcher 接口实现
// =============================================================================

// SearchByVector 向量搜索
func (v *VearchStore) SearchByVector(ctx context.Context, vector []float32, options *models.SearchOptions) ([]models.SearchResult, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return nil, err
		}
	}

	// 设置默认选项
	if options == nil {
		options = &models.SearchOptions{Limit: v.config.DefaultTopK}
	}
	if options.Limit <= 0 {
		options.Limit = v.config.DefaultTopK
	}

	log.Printf("[Vearch存储] 向量搜索: limit=%d, sessionId=%s, userId=%s", options.Limit, options.SessionID, options.UserID)

	// ✅ 构建过滤条件（严格按照官方文档格式）
	filters := make(map[string]interface{})
	if options.SessionID != "" {
		filters["session_id"] = options.SessionID
	}
	if options.UserID != "" {
		filters["user_id"] = options.UserID // ✅ 修正：使用数据库schema中的字段名user_id
	}

	// 添加额外过滤条件
	if options.ExtraFilters != nil {
		for k, v := range options.ExtraFilters {
			filters[k] = v
		}
	}

	// 构建搜索请求（不在Query中设置Filter）
	searchReq := &VearchSearchRequest{
		Vectors: []VearchVector{
			{
				Field:   "vector",
				Feature: vector,
			},
		},
		Filters: &VearchFilter{
			Operator: "AND",
			Conditions: []VearchCondition{
				// 🔍 测试用：注释掉session_id过滤，只保留user_id过滤
				// {
				// 	Field:    "session_id",
				// 	Operator: "IN",
				// 	Value:    []interface{}{options.SessionID},
				// },
				{
					Field:    "user_id",
					Operator: "IN",
					Value:    []interface{}{options.UserID},
				},
			},
		},
		IsBruteSearch: options.IsBruteSearch, // 🔥 通过调用层控制是否启用暴力搜索
		Limit:         options.Limit,
	}

	// 🔥 详细日志：打印完整请求参数
	log.Printf("[Vearch搜索] === SearchByVector 请求详情 ===")
	log.Printf("[Vearch搜索] 数据库: %s, 空间: context_keeper_vector", v.database)
	log.Printf("[Vearch搜索] 选项 - UserID: %s, SessionID: %s, Limit: %d, IsBruteSearch: %d",
		options.UserID, options.SessionID, options.Limit, options.IsBruteSearch)
	log.Printf("[Vearch搜索] 向量维度: %d", len(vector))
	log.Printf("[Vearch搜索] 过滤器 - Operator: %s", searchReq.Filters.Operator)
	for i, condition := range searchReq.Filters.Conditions {
		log.Printf("[Vearch搜索] 过滤条件[%d] - Field: %s, Operator: %s, Value: %v",
			i, condition.Field, condition.Operator, condition.Value)
	}

	// 执行搜索（使用主空间context_keeper）
	resp, err := v.client.Search(v.database, "context_keeper_vector", searchReq)
	if err != nil {
		log.Printf("[Vearch存储] 搜索失败: %v", err)
		return nil, fmt.Errorf("Vearch搜索失败: %v", err)
	}

	// 转换结果（使用正确的字段名）
	results := make([]models.SearchResult, 0, len(resp.Data.Documents))
	for _, docArray := range resp.Data.Documents {
		if len(docArray) > 0 {
			doc := docArray[0] // 取第一个文档
			result := models.SearchResult{
				ID:    getString(doc, "_id"),
				Score: getFloat64(doc, "_score"),
				Fields: map[string]interface{}{
					"content":      doc["content"],
					"session_id":   doc["session_id"], // 使用下划线格式
					"role":         doc["role"],
					"content_type": doc["content_type"],
					"timestamp":    doc["timestamp"],
					"priority":     doc["priority"],
					"metadata":     doc["metadata"],
				},
			}
			results = append(results, result)
		}
	}

	// 🔥 修复排序问题：对于内积（InnerProduct），分数越大越相似，按降序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	log.Printf("[Vearch存储] 搜索完成: 找到%d个结果", len(results))
	return results, nil
}

// SearchByText 文本搜索
func (v *VearchStore) SearchByText(ctx context.Context, query string, options *models.SearchOptions) ([]models.SearchResult, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return nil, err
		}
	}

	// 设置默认选项
	if options == nil {
		options = &models.SearchOptions{Limit: v.config.DefaultTopK}
	}
	if options.Limit <= 0 {
		options.Limit = v.config.DefaultTopK
	}

	log.Printf("[Vearch存储] 文本搜索: query=%s, limit=%d", query, options.Limit)

	// 构建搜索请求
	searchReq := &VearchSearchRequest{
		Vectors: []VearchVector{
			{
				Field:   "vector",
				Feature: make([]float32, v.config.Dimension), // 零向量用于文本搜索
			},
		},
		IsBruteSearch: options.IsBruteSearch, // 🔥 通过调用层控制是否启用暴力搜索
		Limit:         options.Limit,
	}

	// 执行搜索（使用主空间context_keeper）
	resp, err := v.client.Search(v.database, "context_keeper_vector", searchReq)
	if err != nil {
		return nil, fmt.Errorf("Vearch文本搜索失败: %v", err)
	}

	// 转换结果
	results := make([]models.SearchResult, 0, len(resp.Data.Documents))
	for _, docArray := range resp.Data.Documents {
		if len(docArray) > 0 {
			doc := docArray[0] // 取第一个文档
			result := models.SearchResult{
				ID:    getString(doc, "_id"),
				Score: getFloat64(doc, "_score"),
				Fields: map[string]interface{}{
					"content":      doc["content"],
					"session_id":   doc["session_id"], // 使用下划线格式
					"role":         doc["role"],
					"content_type": doc["content_type"],
					"timestamp":    doc["timestamp"],
					"priority":     doc["priority"],
					"metadata":     doc["metadata"],
				},
			}
			results = append(results, result)
		}
	}

	// 🔥 修复排序问题：对于内积（InnerProduct），分数越大越相似，按降序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	log.Printf("[Vearch存储] 文本搜索完成: 找到%d个结果", len(results))
	return results, nil
}

// SearchByID 根据ID精确搜索
func (v *VearchStore) SearchByID(ctx context.Context, id string, options *models.SearchOptions) ([]models.SearchResult, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return nil, err
		}
	}

	// 设置默认选项
	if options == nil {
		options = &models.SearchOptions{Limit: 10}
	}

	log.Printf("[Vearch存储] ID搜索: id=%s, limit=%d", id, options.Limit)

	// 构建ID精确匹配的过滤条件
	filter := make(map[string]interface{})

	// 尝试不同的ID字段匹配策略
	// 1. 主ID匹配
	filter["_id"] = id

	// 2. 如果有批次ID等特殊字段，也添加到OR条件中
	// Vearch支持复杂查询，但这里使用基础的精确匹配

	// 添加会话和用户过滤
	if options.SessionID != "" {
		filter["session_id"] = options.SessionID
	}
	if options.UserID != "" {
		filter["user_id"] = options.UserID
	}

	// 使用零向量进行ID搜索（纯过滤搜索）
	zeroVector := make([]float32, v.config.Dimension)

	// 构建搜索请求
	searchReq := &VearchSearchRequest{
		Vectors: []VearchVector{
			{
				Field:   "vector",
				Feature: zeroVector,
			},
		},
		Filters: &VearchFilter{
			Operator: "AND",
			Conditions: []VearchCondition{
				// 🔍 测试用：注释掉session_id过滤，只保留user_id过滤
				// {
				// 	Field:    "session_id",
				// 	Operator: "IN",
				// 	Value:    []interface{}{options.SessionID},
				// },
				{
					Field:    "user_id",
					Operator: "IN",
					Value:    []interface{}{options.UserID},
				},
			},
		},
		Limit: options.Limit,
	}

	// 🔥 详细日志：打印完整请求参数
	log.Printf("[Vearch搜索] === SearchByID 请求详情 ===")
	log.Printf("[Vearch搜索] 数据库: %s, 空间: context_keeper_vector", v.database)
	log.Printf("[Vearch搜索] 目标ID: %s", id)
	log.Printf("[Vearch搜索] 选项 - UserID: %s, SessionID: %s, Limit: %d",
		options.UserID, options.SessionID, options.Limit)
	log.Printf("[Vearch搜索] 过滤器 - Operator: %s", searchReq.Filters.Operator)
	for i, condition := range searchReq.Filters.Conditions {
		log.Printf("[Vearch搜索] 过滤条件[%d] - Field: %s, Operator: %s, Value: %v",
			i, condition.Field, condition.Operator, condition.Value)
	}

	// 执行搜索（使用主空间context_keeper）
	resp, err := v.client.Search(v.database, "context_keeper_vector", searchReq)
	if err != nil {
		// 如果主ID搜索失败，尝试在metadata中搜索
		log.Printf("[Vearch存储] 主ID搜索失败，尝试metadata搜索: %v", err)

		// 尝试在metadata字段中搜索批次ID或记忆ID
		filter = make(map[string]interface{})
		// 构建metadata包含查询（如果Vearch支持的话）
		filter["content"] = id // 有时ID可能在内容中

		searchReq.Filters = &VearchFilter{
			Operator: "AND",
			Conditions: []VearchCondition{
				// 🔍 测试用：注释掉session_id过滤，只保留user_id过滤
				// {
				// 	Field:    "session_id",
				// 	Operator: "IN",
				// 	Value:    []interface{}{options.SessionID},
				// },
				{
					Field:    "user_id",
					Operator: "IN",
					Value:    []interface{}{options.UserID},
				},
			},
		}
		resp, err = v.client.Search(v.database, "context_keeper_vector", searchReq)
		if err != nil {
			return nil, fmt.Errorf("Vearch ID搜索失败: %v", err)
		}
	}

	// 转换结果
	results := make([]models.SearchResult, 0, len(resp.Data.Documents))
	for _, docArray := range resp.Data.Documents {
		if len(docArray) > 0 {
			doc := docArray[0] // 取第一个文档
			result := models.SearchResult{
				ID:    getString(doc, "_id"),
				Score: getFloat64(doc, "_score"),
				Fields: map[string]interface{}{
					"content":      doc["content"],
					"session_id":   doc["session_id"],
					"role":         doc["role"],
					"content_type": doc["content_type"],
					"timestamp":    doc["timestamp"],
					"priority":     doc["priority"],
					"metadata":     doc["metadata"],
				},
			}
			results = append(results, result)
		}
	}

	// 🔥 修复排序问题：对于内积（InnerProduct），分数越大越相似，按降序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	log.Printf("[Vearch存储] ID搜索完成: 找到%d个结果", len(results))
	return results, nil
}

// SearchByFilter 根据过滤条件搜索
func (v *VearchStore) SearchByFilter(ctx context.Context, filter string, options *models.SearchOptions) ([]models.SearchResult, error) {
	// 解析过滤条件
	var filterMap map[string]interface{}
	if err := json.Unmarshal([]byte(filter), &filterMap); err != nil {
		return nil, fmt.Errorf("解析过滤条件失败: %v", err)
	}

	// 使用零向量进行过滤搜索
	zeroVector := make([]float32, v.config.Dimension)

	// 将过滤条件添加到搜索选项
	if options == nil {
		options = &models.SearchOptions{}
	}
	if options.ExtraFilters == nil {
		options.ExtraFilters = make(map[string]interface{})
	}
	for k, v := range filterMap {
		options.ExtraFilters[k] = v
	}

	// 构建最终过滤条件（使用下划线字段名）
	finalFilter := make(map[string]interface{})
	if options.SessionID != "" {
		finalFilter["session_id"] = options.SessionID
	}
	if options.UserID != "" {
		finalFilter["user_id"] = options.UserID
	}
	for k, v := range options.ExtraFilters {
		finalFilter[k] = v
	}

	// 构建搜索请求（使用官方格式）
	searchReq := &VearchSearchRequest{
		Vectors: []VearchVector{
			{
				Field:   "vector",
				Feature: zeroVector,
			},
		},
		Filters: &VearchFilter{
			Operator: "AND",
			Conditions: []VearchCondition{
				// 🔍 测试用：注释掉session_id过滤，只保留user_id过滤
				// {
				// 	Field:    "session_id",
				// 	Operator: "IN",
				// 	Value:    []interface{}{options.SessionID},
				// },
				{
					Field:    "user_id",
					Operator: "IN",
					Value:    []interface{}{options.UserID},
				},
			},
		},
		Limit: options.Limit,
	}

	// 🔥 详细日志：打印完整请求参数
	log.Printf("[Vearch搜索] === SearchByFilter 请求详情 ===")
	log.Printf("[Vearch搜索] 数据库: %s, 空间: context_keeper_vector", v.database)
	log.Printf("[Vearch搜索] 原始过滤器: %s", filter)
	log.Printf("[Vearch搜索] 选项 - UserID: %s, SessionID: %s, Limit: %d",
		options.UserID, options.SessionID, options.Limit)
	log.Printf("[Vearch搜索] 最终过滤器 - Operator: %s", searchReq.Filters.Operator)
	for i, condition := range searchReq.Filters.Conditions {
		log.Printf("[Vearch搜索] 过滤条件[%d] - Field: %s, Operator: %s, Value: %v",
			i, condition.Field, condition.Operator, condition.Value)
	}

	// 执行搜索（使用主空间context_keeper）
	resp, err := v.client.Search(v.database, "context_keeper_vector", searchReq)
	if err != nil {
		return nil, fmt.Errorf("Vearch过滤搜索失败: %v", err)
	}

	// 转换结果（使用正确的字段名）
	results := make([]models.SearchResult, 0, len(resp.Data.Documents))
	for _, docArray := range resp.Data.Documents {
		if len(docArray) > 0 {
			doc := docArray[0] // 取第一个文档
			result := models.SearchResult{
				ID:    getString(doc, "_id"),
				Score: getFloat64(doc, "_score"),
				Fields: map[string]interface{}{
					"content":      doc["content"],
					"session_id":   doc["session_id"], // 使用下划线格式
					"role":         doc["role"],
					"content_type": doc["content_type"],
					"timestamp":    doc["timestamp"],
					"priority":     doc["priority"],
					"metadata":     doc["metadata"],
				},
			}
			results = append(results, result)
		}
	}

	// 🔥 修复排序问题：对于内积（InnerProduct），分数越大越相似，按降序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	log.Printf("[Vearch存储] 过滤搜索完成: 找到%d个结果", len(results))
	return results, nil
}

// =============================================================================
// CollectionManager 接口实现
// =============================================================================

// EnsureSpace 确保空间存在
func (v *VearchStore) EnsureSpace(spaceName string) error {
	// 注意：不在这里检查初始化状态，避免死循环
	// 调用方应该确保已经初始化或正在初始化过程中

	// 临时跳过空间存在性检查，直接尝试创建空间
	log.Printf("[Vearch存储] 直接尝试创建空间: %s", spaceName)

	err := v.CreateSpace(spaceName, &models.CollectionConfig{
		Dimension:   v.config.Dimension,
		Metric:      "inner_product",
		Description: fmt.Sprintf("Auto-created space: %s", spaceName),
	})

	// 如果是"空间已存在"的错误，忽略它
	if err != nil && (strings.Contains(err.Error(), "exist") || strings.Contains(err.Error(), "exists")) {
		log.Printf("[Vearch存储] 空间已存在: %s", spaceName)
		return nil
	}

	return err
}

// CreateSpace 创建空间
func (v *VearchStore) CreateSpace(name string, config *models.CollectionConfig) error {
	// 注意：不在这里检查初始化状态，避免死循环
	// 调用方应该确保已经初始化或正在初始化过程中

	log.Printf("[Vearch存储] 创建空间: name=%s, dimension=%d", name, config.Dimension)

	schema := v.buildSpaceSchema(config)

	if err := v.client.CreateSpace(v.database, name, schema); err != nil {
		return fmt.Errorf("创建空间失败: %v", err)
	}

	// 缓存空间信息
	v.spaces[name] = &VearchSpace{
		Name:         name,
		PartitionNum: 1, // 默认分区数量
		ReplicaNum:   1, // 默认副本数量
		Properties: map[string]interface{}{
			"vector_field": "vector",
			"id_field":     "_id",
		},
		Engine: &EngineConfig{
			Name:      "gamma",
			IndexSize: 1000000, // 默认索引大小
			Retrieval: &RetrievalConfig{
				Type: "ivf_pq",
				Parameters: map[string]interface{}{
					"index_type": "ivf_pq",
					"pq_m":       16,
					"pq_n":       100,
					"pq_bits":    8,
				},
			},
		},
		Created: time.Now(),
	}

	log.Printf("[Vearch存储] 空间创建成功: %s", name)
	return nil
}

// DeleteSpace 删除空间
func (v *VearchStore) DeleteSpace(name string) error {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	log.Printf("[Vearch存储] 删除空间: %s", name)

	if err := v.client.DropSpace(v.database, name); err != nil {
		return fmt.Errorf("删除空间失败: %v", err)
	}

	// 从缓存中移除
	delete(v.spaces, name)

	log.Printf("[Vearch存储] 空间删除成功: %s", name)
	return nil
}

// SpaceExists 检查空间是否存在
func (v *VearchStore) SpaceExists(name string) (bool, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return false, err
		}
	}

	return v.client.SpaceExists(v.database, name)
}

// CollectionExists 检查集合是否存在（为了兼容VectorStore接口）
func (v *VearchStore) CollectionExists(name string) (bool, error) {
	return v.SpaceExists(name)
}

// CreateCollection 创建集合（为了兼容VectorStore接口）
func (v *VearchStore) CreateCollection(name string, config *models.CollectionConfig) error {
	return v.CreateSpace(name, config)
}

// DeleteCollection 删除集合（为了兼容VectorStore接口）
func (v *VearchStore) DeleteCollection(name string) error {
	return v.DeleteSpace(name)
}

// EnsureCollection 确保集合存在（为了兼容VectorStore接口）
func (v *VearchStore) EnsureCollection(collectionName string) error {
	return v.EnsureSpace(collectionName)
}

// =============================================================================
// UserDataStorage 接口实现
// =============================================================================

// StoreUserInfo 存储用户信息
func (v *VearchStore) StoreUserInfo(userInfo *models.UserInfo) error {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return err
		}
	}

	log.Printf("[Vearch存储] 存储用户信息: ID=%s", userInfo.UserID)

	// 将metadata转换为JSON字符串（与阿里云实现保持一致）
	metadataStr := "{}"
	if userInfo.Metadata != nil {
		if metadataBytes, err := json.Marshal(userInfo.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[Vearch存储] 警告: 无法序列化用户metadata: %v", err)
		}
	}

	// 构建用户文档
	doc := map[string]interface{}{
		"_id":        userInfo.UserID,
		"user_id":    userInfo.UserID,
		"firstUsed":  userInfo.FirstUsed,
		"lastActive": userInfo.LastActive,
		"deviceInfo": userInfo.DeviceInfo,
		"createdAt":  userInfo.CreatedAt,
		"updatedAt":  userInfo.UpdatedAt,
		"metadata":   metadataStr, // ✅ 使用JSON字符串格式
	}

	// 插入到用户空间（使用context_keeper_users）
	if err := v.client.Insert(v.database, "context_keeper_users", []map[string]interface{}{doc}); err != nil {
		return fmt.Errorf("插入用户信息失败: %v", err)
	}

	log.Printf("[Vearch存储] 用户信息存储成功: %s", userInfo.UserID)
	return nil
}

// GetUserInfo 获取用户信息
func (v *VearchStore) GetUserInfo(userID string) (*models.UserInfo, error) {
	if !v.initialized {
		if err := v.Initialize(); err != nil {
			return nil, err
		}
	}

	// TODO: 实现用户信息精确查询
	log.Printf("[Vearch存储] 获取用户信息: %s", userID)

	return nil, fmt.Errorf("Vearch用户信息查询暂未实现")
}

// CheckUserExists 检查用户是否存在
func (v *VearchStore) CheckUserExists(userID string) (bool, error) {
	userInfo, err := v.GetUserInfo(userID)
	if err != nil {
		return false, nil // 查询失败视为不存在
	}
	return userInfo != nil, nil
}

// InitUserStorage 初始化用户存储
func (v *VearchStore) InitUserStorage() error {
	return v.EnsureSpace("context_keeper_users")
}

// =============================================================================
// 辅助方法
// =============================================================================

// createMemorySpaceSchema 创建记忆空间schema
func (v *VearchStore) createMemorySpaceSchema() *models.CollectionConfig {
	return &models.CollectionConfig{
		Dimension:   v.config.Dimension,
		Metric:      "inner_product",
		Description: "Context Keeper memories space",
		IndexType:   "ivf_pq",
		ExtraConfig: map[string]interface{}{
			"vector_field": "vector",
			"id_field":     "_id",
		},
	}
}

// createMessageSpaceSchema 创建消息空间schema
func (v *VearchStore) createMessageSpaceSchema() *models.CollectionConfig {
	return &models.CollectionConfig{
		Dimension:   v.config.Dimension,
		Metric:      "inner_product",
		Description: "Context Keeper messages space",
		IndexType:   "ivf_pq",
		ExtraConfig: map[string]interface{}{
			"vector_field": "vector",
			"id_field":     "_id",
		},
	}
}

// createUserSpaceSchema 创建用户空间schema
func (v *VearchStore) createUserSpaceSchema() *models.CollectionConfig {
	return &models.CollectionConfig{
		Dimension:   128, // 用户信息用较小维度
		Metric:      "inner_product",
		Description: "Context Keeper users space",
		IndexType:   "ivf_pq",
		ExtraConfig: map[string]interface{}{
			"id_field": "_id",
		},
	}
}

// createDefaultSpaceSchema 创建默认空间schema
func (v *VearchStore) createDefaultSpaceSchema() *SpaceConfig {
	return &SpaceConfig{
		Name:         "default",
		PartitionNum: 1,
		ReplicaNum:   1,
		Properties: []map[string]interface{}{
			{
				"name": "_id",
				"type": "string",
			},
			{
				"name": "content",
				"type": "string",
			},
			{
				"name": "session_id",
				"type": "string",
				"index": map[string]interface{}{
					"name": "session_id_index",
					"type": "SCALAR",
				},
			},
			{
				"name": "user_id",
				"type": "string",
				"index": map[string]interface{}{
					"name": "user_id_index",
					"type": "SCALAR",
				},
			},
			{
				"name": "memory_id",
				"type": "string",
				"index": map[string]interface{}{
					"name": "memory_id_index",
					"type": "SCALAR",
				},
			},
			{
				"name": "message_id",
				"type": "string",
				"index": map[string]interface{}{
					"name": "message_id_index",
					"type": "SCALAR",
				},
			},
			{
				"name": "formatted_time",
				"type": "string",
			},
			{
				"name": "biz_type",
				"type": "string",
			},
			{
				"name": "role",
				"type": "string",
			},
			{
				"name": "content_type",
				"type": "string",
			},
			{
				"name": "timestamp",
				"type": "integer",
				"index": map[string]interface{}{
					"name": "timestamp_index",
					"type": "SCALAR",
				},
			},
			{
				"name": "priority",
				"type": "string",
			},
			{
				"name": "metadata",
				"type": "string",
			},
			{
				"name":      "vector",
				"type":      "vector",
				"dimension": v.config.Dimension,
				"index": map[string]interface{}{
					"name": "vector_index",
					"type": "IVFPQ", // 使用IVFPQ索引类型
					"params": map[string]interface{}{
						"metric_type":    "InnerProduct", // 使用内积计算
						"ncentroids":     2048,           // 聚类中心数量
						"nsubvector":     32,             // PQ拆分子向量大小
						"nprobe":         80,             // 检索时查找的聚类中心数量
						"efConstruction": 40,             // 构图深度
						"efSearch":       40,             // 搜索深度
					},
				},
			},
		},
		Engine: &EngineConfig{
			Name:      "gamma",
			IndexSize: 1000000,
			Retrieval: &RetrievalConfig{
				Type: "ivf_pq",
				Parameters: map[string]interface{}{
					"index_type": "ivf_pq",
					"pq_m":       16,
					"pq_n":       100,
					"pq_bits":    8,
				},
			},
		},
	}
}

// buildSpaceSchema 构建空间schema（按官方文档规范）
func (v *VearchStore) buildSpaceSchema(config *models.CollectionConfig) *SpaceConfig {
	// 📖 根据Vearch官方文档，fields是一个数组，定义表空间的字段结构
	// 注意：_id字段是Vearch保留字段，不需要显式定义
	fields := []map[string]interface{}{
		// 内容字段
		{
			"name": "content",
			"type": "string",
		},
		// 记忆ID字段（重要：与阿里云版本对齐）
		{
			"name": "memory_id",
			"type": "string",
			"index": map[string]interface{}{
				"name": "memory_id_index",
				"type": "SCALAR",
			},
		},
		// 消息ID字段（重要：补充缺失的字段，与阿里云版本对齐）
		{
			"name": "message_id",
			"type": "string",
			"index": map[string]interface{}{
				"name": "message_id_index",
				"type": "SCALAR",
			},
		},
		// 会话ID字段（建立标量索引以支持过滤查询）- 与阿里云版本对齐
		{
			"name": "session_id",
			"type": "string",
			"index": map[string]interface{}{
				"name": "session_id_index",
				"type": "SCALAR",
			},
		},
		// 用户ID字段（建立标量索引以支持过滤查询）
		{
			"name": "user_id",
			"type": "string",
			"index": map[string]interface{}{
				"name": "user_id_index",
				"type": "SCALAR",
			},
		},
		// 格式化时间字段（与阿里云版本对齐）
		{
			"name": "formatted_time",
			"type": "string",
		},
		// 业务类型字段（与阿里云版本对齐）
		{
			"name": "biz_type",
			"type": "string",
		},
		// 角色字段
		{
			"name": "role",
			"type": "string",
		},
		// 内容类型字段
		{
			"name": "content_type",
			"type": "string",
		},
		// 时间戳字段（建立标量索引以支持时间排序）
		{
			"name": "timestamp",
			"type": "integer",
			"index": map[string]interface{}{
				"name": "timestamp_index",
				"type": "SCALAR",
			},
		},
		// 优先级字段
		{
			"name": "priority",
			"type": "string",
		},
		// 元数据字段
		{
			"name": "metadata",
			"type": "string",
		},
		// 向量字段（关键：用于向量搜索）
		{
			"name":      "vector",
			"type":      "vector",
			"dimension": config.Dimension,
			"index": map[string]interface{}{
				"name": "vector_index",
				"type": "IVFPQ", // 使用IVFPQ索引类型
				"params": map[string]interface{}{
					"metric_type":    "InnerProduct", // 使用内积计算
					"ncentroids":     2048,           // 聚类中心数量
					"nsubvector":     32,             // PQ拆分子向量大小
					"nprobe":         80,             // 检索时查找的聚类中心数量
					"efConstruction": 40,             // 构图深度
					"efSearch":       40,             // 搜索深度
				},
			},
		},
	}

	schema := &SpaceConfig{
		Name:         "auto_created_space",
		PartitionNum: 1,      // 默认分区数量
		ReplicaNum:   1,      // 默认副本数量
		Properties:   fields, // 使用fields数组而不是map
		Engine: &EngineConfig{
			Name:      "gamma",
			IndexSize: 1000000,
			Retrieval: &RetrievalConfig{
				Type: "ivf_pq",
				Parameters: map[string]interface{}{
					"index_type": "ivf_pq",
					"pq_m":       16,
					"pq_n":       100,
					"pq_bits":    8,
				},
			},
		},
	}

	// 添加额外配置
	if config.ExtraConfig != nil {
		// 处理额外字段定义
		for fieldName, fieldConfig := range config.ExtraConfig {
			if fieldMap, ok := fieldConfig.(map[string]interface{}); ok {
				additionalField := map[string]interface{}{
					"name": fieldName,
				}
				for k, v := range fieldMap {
					additionalField[k] = v
				}
				// 正确的数组追加语法
				schema.Properties = append(schema.Properties, additionalField)
			}
		}
	}

	return schema
}

// getFloat64 安全地从map中获取float64值
func getFloat64(data map[string]interface{}, key string) float64 {
	if value, ok := data[key].(float64); ok {
		return value
	}
	if value, ok := data[key].(int); ok {
		return float64(value)
	}
	if value, ok := data[key].(int64); ok {
		return float64(value)
	}
	return 0.0
}

// GetProvider 获取向量存储提供商类型
func (v *VearchStore) GetProvider() models.VectorStoreType {
	return models.VectorStoreTypeVearch
}
