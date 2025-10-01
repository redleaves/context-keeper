package vectorstore

import (
	"context"
	"fmt"
	"log"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/pkg/aliyun"
)

// AliyunVectorStore 阿里云向量存储实现
// 内部包装原有的 aliyun.VectorService，实现 VectorStore 抽象接口
type AliyunVectorStore struct {
	// 内部包装原有的VectorService，保持其不变
	vectorService *aliyun.VectorService
	config        *models.VectorStoreConfig
}

// NewAliyunVectorStore 创建阿里云向量存储实例
func NewAliyunVectorStore(vectorService *aliyun.VectorService, config *models.VectorStoreConfig) *AliyunVectorStore {
	return &AliyunVectorStore{
		vectorService: vectorService,
		config:        config,
	}
}

// =============================================================================
// EmbeddingProvider 接口实现
// =============================================================================

// GenerateEmbedding 将文本转换为向量表示
func (a *AliyunVectorStore) GenerateEmbedding(text string) ([]float32, error) {
	log.Printf("[阿里云向量存储] 开始生成文本嵌入向量")
	return a.vectorService.GenerateEmbedding(text)
}

// GetEmbeddingDimension 获取向量维度
func (a *AliyunVectorStore) GetEmbeddingDimension() int {
	return a.vectorService.GetDimension()
}

// =============================================================================
// MemoryStorage 接口实现
// =============================================================================

// StoreMemory 存储记忆到向量数据库
func (a *AliyunVectorStore) StoreMemory(memory *models.Memory) error {
	log.Printf("[阿里云向量存储] 存储记忆: ID=%s, 会话=%s", memory.ID, memory.SessionID)
	return a.vectorService.StoreVectors(memory)
}

// StoreMessage 存储消息到向量数据库
func (a *AliyunVectorStore) StoreMessage(message *models.Message) error {
	log.Printf("[阿里云向量存储] 存储消息: ID=%s, 会话=%s", message.ID, message.SessionID)
	return a.vectorService.StoreMessage(message)
}

// CountMemories 统计指定会话的记忆数量
func (a *AliyunVectorStore) CountMemories(sessionID string) (int, error) {
	log.Printf("[阿里云向量存储] 统计会话记忆数量: 会话=%s", sessionID)
	return a.vectorService.CountSessionMemories(sessionID)
}

// StoreEnhancedMemory 存储增强的多维度记忆（新增方法）
func (a *AliyunVectorStore) StoreEnhancedMemory(memory *models.EnhancedMemory) error {
	log.Printf("[阿里云向量存储] 存储增强记忆: ID=%s, 会话=%s", memory.Memory.ID, memory.Memory.SessionID)

	// 调用底层向量服务的增强存储方法
	return a.vectorService.StoreEnhancedMemory(memory)
}

// StoreEnhancedMessage 存储增强的多维度消息（新增方法）
func (a *AliyunVectorStore) StoreEnhancedMessage(message *models.EnhancedMessage) error {
	log.Printf("[阿里云向量存储] 存储增强消息: ID=%s, 会话=%s", message.Message.ID, message.Message.SessionID)

	// 调用底层向量服务的增强存储方法
	return a.vectorService.StoreEnhancedMessage(message)
}

// =============================================================================
// VectorSearcher 接口实现
// =============================================================================

// SearchByVector 使用向量进行相似度搜索
func (a *AliyunVectorStore) SearchByVector(ctx context.Context, vector []float32, options *models.SearchOptions) ([]models.SearchResult, error) {
	log.Printf("[阿里云向量存储] 向量搜索: 维度=%d, 限制=%d", len(vector), options.Limit)

	// 构建搜索选项
	searchOptions := a.buildSearchOptions(options)

	// 调用原有的高级搜索方法
	return a.vectorService.SearchVectorsAdvanced(vector, options.SessionID, options.Limit, searchOptions)
}

// SearchByText 使用文本进行搜索（内部转换为向量）
func (a *AliyunVectorStore) SearchByText(ctx context.Context, query string, options *models.SearchOptions) ([]models.SearchResult, error) {
	log.Printf("[阿里云向量存储] 文本搜索: 查询='%s', 限制=%d", query, options.Limit)

	// 构建过滤条件
	filters := make(map[string]interface{})
	if options.ExtraFilters != nil {
		for k, v := range options.ExtraFilters {
			filters[k] = v
		}
	}

	// 调用原有的文本搜索方法
	return a.vectorService.SearchWithTextAndFilters(ctx, query, options.Limit, filters, options.SkipThreshold)
}

// SearchByID 根据ID精确搜索
func (a *AliyunVectorStore) SearchByID(ctx context.Context, id string, options *models.SearchOptions) ([]models.SearchResult, error) {
	log.Printf("[阿里云向量存储] ID搜索: ID='%s'", id)

	// 调用原有的ID搜索方法
	return a.vectorService.SearchByID(id, "id")
}

// SearchByFilter 根据过滤条件搜索
func (a *AliyunVectorStore) SearchByFilter(ctx context.Context, filter string, options *models.SearchOptions) ([]models.SearchResult, error) {
	log.Printf("[阿里云向量存储] 过滤搜索: 过滤条件='%s', 限制=%d", filter, options.Limit)

	// 调用原有的过滤搜索方法
	return a.vectorService.SearchByFilter(filter, options.Limit)
}

// =============================================================================
// CollectionManager 接口实现
// =============================================================================

// EnsureCollection 确保集合存在，不存在则创建
func (a *AliyunVectorStore) EnsureCollection(collectionName string) error {
	log.Printf("[阿里云向量存储] 确保集合存在: %s", collectionName)
	return a.vectorService.EnsureCollection()
}

// CreateCollection 创建新集合
func (a *AliyunVectorStore) CreateCollection(name string, config *models.CollectionConfig) error {
	log.Printf("[阿里云向量存储] 创建集合: %s, 维度=%d, 度量=%s", name, config.Dimension, config.Metric)
	return a.vectorService.CreateCollection(name, config.Dimension, config.Metric)
}

// DeleteCollection 删除集合
func (a *AliyunVectorStore) DeleteCollection(name string) error {
	log.Printf("[阿里云向量存储] 删除集合: %s", name)
	return a.vectorService.DeleteCollection(name)
}

// CollectionExists 检查集合是否存在
func (a *AliyunVectorStore) CollectionExists(name string) (bool, error) {
	log.Printf("[阿里云向量存储] 检查集合存在性: %s", name)
	return a.vectorService.CheckCollectionExists(name)
}

// =============================================================================
// UserDataStorage 接口实现
// =============================================================================

// StoreUserInfo 存储用户信息
func (a *AliyunVectorStore) StoreUserInfo(userInfo *models.UserInfo) error {
	log.Printf("[阿里云向量存储] 存储用户信息: 用户ID=%s", userInfo.UserID)
	return a.vectorService.StoreUserInfo(userInfo)
}

// GetUserInfo 获取用户信息
func (a *AliyunVectorStore) GetUserInfo(userID string) (*models.UserInfo, error) {
	log.Printf("[阿里云向量存储] 获取用户信息: 用户ID=%s", userID)
	return a.vectorService.GetUserInfo(userID)
}

// CheckUserExists 检查用户是否存在
func (a *AliyunVectorStore) CheckUserExists(userID string) (bool, error) {
	log.Printf("[阿里云向量存储] 检查用户存在性: 用户ID=%s", userID)
	return a.vectorService.CheckUserIDUniqueness(userID)
}

// InitUserStorage 初始化用户存储
func (a *AliyunVectorStore) InitUserStorage() error {
	log.Printf("[阿里云向量存储] 初始化用户存储")
	return a.vectorService.InitUserCollection()
}

// =============================================================================
// 内部辅助方法
// =============================================================================

// buildSearchOptions 构建搜索选项，将抽象选项转换为阿里云特定选项
func (a *AliyunVectorStore) buildSearchOptions(options *models.SearchOptions) map[string]interface{} {
	searchOptions := make(map[string]interface{})

	// 跳过阈值过滤
	if options.SkipThreshold {
		searchOptions["skip_threshold"] = true
		searchOptions["skip_threshold_filter"] = true
	}

	// 用户ID过滤
	if options.UserID != "" {
		searchOptions["filter"] = fmt.Sprintf(`userId="%s"`, options.UserID)
	}

	// 额外过滤条件
	if options.ExtraFilters != nil {
		// 如果已有filter，需要组合
		existingFilter, hasExisting := searchOptions["filter"].(string)
		var filterConditions []string

		if hasExisting {
			filterConditions = append(filterConditions, existingFilter)
		}

		// 添加额外过滤条件
		for key, value := range options.ExtraFilters {
			switch v := value.(type) {
			case string:
				filterConditions = append(filterConditions, fmt.Sprintf(`%s="%s"`, key, v))
			case int, int64, float32, float64:
				filterConditions = append(filterConditions, fmt.Sprintf(`%s=%v`, key, v))
			case bool:
				filterConditions = append(filterConditions, fmt.Sprintf(`%s=%t`, key, v))
			}
		}

		// 组合所有过滤条件
		if len(filterConditions) > 0 {
			searchOptions["filter"] = joinFilters(filterConditions, " AND ")
		}
	}

	return searchOptions
}

// joinFilters 连接过滤条件
func joinFilters(conditions []string, separator string) string {
	if len(conditions) == 0 {
		return ""
	}
	if len(conditions) == 1 {
		return conditions[0]
	}

	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result += separator + conditions[i]
	}
	return result
}

// GetConfig 获取配置信息
func (a *AliyunVectorStore) GetConfig() *models.VectorStoreConfig {
	return a.config
}

// GetProvider 获取提供商类型
func (a *AliyunVectorStore) GetProvider() models.VectorStoreType {
	return models.VectorStoreTypeAliyun
}

// IsHealthy 检查存储健康状态
func (a *AliyunVectorStore) IsHealthy() bool {
	// 尝试检查集合是否存在来验证连接健康状态
	_, err := a.vectorService.CheckCollectionExists(a.config.DefaultCollection)
	return err == nil
}
