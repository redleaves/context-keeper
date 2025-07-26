package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/vectorstore"
)

// ContextServiceV2 使用抽象VectorStore接口的上下文服务
// 这是新版本的ContextService，使用抽象接口而不是具体的aliyun.VectorService
type ContextServiceV2 struct {
	vectorStore        models.VectorStore // 使用抽象接口
	sessionStore       *store.SessionStore
	userSessionManager *store.UserSessionManager
	config             *config.Config
}

// NewContextServiceV2 创建使用抽象接口的新版本上下文服务
func NewContextServiceV2(vectorStore models.VectorStore, sessionStore *store.SessionStore, cfg *config.Config) *ContextServiceV2 {
	// 使用同样的存储路径为UserSessionManager创建基础路径
	baseStorePath := sessionStore.GetStorePath()
	userSessionManager := store.NewUserSessionManager(baseStorePath)

	return &ContextServiceV2{
		vectorStore:        vectorStore,
		sessionStore:       sessionStore,
		userSessionManager: userSessionManager,
		config:             cfg,
	}
}

// NewContextServiceV2FromLegacy 从原有的ContextService创建新版本
// 这是一个过渡方法，用于将原有的服务升级到新的抽象接口
func NewContextServiceV2FromLegacy(legacyService *ContextService) *ContextServiceV2 {
	log.Printf("[上下文服务V2] 从原有服务创建新版本")

	// 获取原有的VectorService并包装成抽象接口
	vectorStore := vectorstore.CreateAliyunVectorStoreFromLegacyService(legacyService.vectorService)

	return &ContextServiceV2{
		vectorStore:        vectorStore,
		sessionStore:       legacyService.sessionStore,
		userSessionManager: legacyService.userSessionManager,
		config:             legacyService.config,
	}
}

// SessionStore 返回会话存储实例
func (s *ContextServiceV2) SessionStore() *store.SessionStore {
	return s.sessionStore
}

// GetUserSessionStore 获取指定用户的会话存储
func (s *ContextServiceV2) GetUserSessionStore(userID string) (*store.SessionStore, error) {
	if userID == "" {
		// 如果未提供用户ID，则尝试从缓存获取
		userID = utils.GetCachedUserID()
		if userID == "" {
			return s.sessionStore, nil // 降级到全局会话存储
		}
	}

	return s.userSessionManager.GetUserSessionStore(userID)
}

// CountSessionMemories 统计会话的记忆数量
func (s *ContextServiceV2) CountSessionMemories(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	// 通过抽象接口统计记忆数量
	count, err := s.vectorStore.CountMemories(sessionID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":     count,
		"timestamp": time.Now().Unix(),
	}, nil
}

// StoreContext 存储上下文内容
func (s *ContextServiceV2) StoreContext(ctx context.Context, req models.StoreContextRequest) (string, error) {
	log.Printf("[上下文服务V2] 接收存储请求: 会话ID=%s, 内容长度=%d字节",
		req.SessionID, len(req.Content))

	// 创建记忆对象
	memory := models.NewMemory(req.SessionID, req.Content, req.Priority, req.Metadata)

	// 设置业务类型和用户ID
	if req.BizType > 0 {
		memory.BizType = req.BizType
	}
	if req.UserID != "" {
		memory.UserID = req.UserID
	}

	startTime := time.Now()
	// 通过抽象接口生成嵌入向量
	vector, err := s.vectorStore.GenerateEmbedding(req.Content)
	if err != nil {
		log.Printf("生成嵌入向量失败: %v", err)
		return "", fmt.Errorf("生成嵌入向量失败: %w", err)
	}
	log.Printf("[上下文服务V2] 向量生成耗时: %v", time.Since(startTime))

	// 设置向量
	memory.Vector = vector

	// 通过抽象接口存储到向量数据库
	startTime = time.Now()
	if err := s.vectorStore.StoreMemory(memory); err != nil {
		return "", fmt.Errorf("存储向量失败: %w", err)
	}
	log.Printf("[上下文服务V2] 向量存储耗时: %v", time.Since(startTime))

	// 更新会话信息
	if err := s.sessionStore.UpdateSession(req.SessionID, req.Content); err != nil {
		log.Printf("[上下文服务V2] 警告: 更新会话信息失败: %v", err)
	}

	log.Printf("[上下文服务V2] 成功存储记忆 ID: %s, 会话: %s", memory.ID, memory.SessionID)
	return memory.ID, nil
}

// RetrieveContext 检索相关上下文
func (s *ContextServiceV2) RetrieveContext(ctx context.Context, req models.RetrieveContextRequest) (models.ContextResponse, error) {
	log.Printf("[上下文服务V2] 接收检索请求: 会话ID=%s, 查询=%s", req.SessionID, req.Query)

	if req.Limit <= 0 {
		req.Limit = 2000
	}

	// 获取会话状态
	sessionState, err := s.sessionStore.GetSessionState(req.SessionID)
	if err != nil {
		log.Printf("[上下文服务V2] 警告: 获取会话状态失败: %v", err)
		sessionState = fmt.Sprintf("会话ID: %s", req.SessionID)
	}

	// 获取最近的对话历史
	recentHistory, err := s.sessionStore.GetRecentHistory(req.SessionID, 5)
	if err != nil {
		log.Printf("[上下文服务V2] 警告: 获取最近历史失败: %v", err)
		recentHistory = []string{}
	}

	var searchResults []models.SearchResult
	var relevantMemories []string

	// 构建搜索选项
	searchOptions := &models.SearchOptions{
		Limit:         10,
		SessionID:     req.SessionID,
		SkipThreshold: req.SkipThreshold,
	}

	// 根据请求类型选择不同的检索方式
	if req.MemoryID != "" {
		// 使用记忆ID精确检索
		startTime := time.Now()
		searchResults, err = s.vectorStore.SearchByID(ctx, req.MemoryID, searchOptions)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过记忆ID检索失败: %w", err)
		}
		log.Printf("[上下文服务V2] 记忆ID检索耗时: %v", time.Since(startTime))
	} else if req.BatchID != "" {
		// 使用批次ID检索
		startTime := time.Now()
		searchResults, err = s.vectorStore.SearchByID(ctx, req.BatchID, searchOptions)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过批次ID检索失败: %w", err)
		}
		log.Printf("[上下文服务V2] 批次ID检索耗时: %v", time.Since(startTime))
	} else if req.Query != "" {
		// 标准文本搜索
		startTime := time.Now()

		// 添加用户ID过滤
		if userID, _, _ := utils.GetUserID(); userID != "" {
			searchOptions.UserID = userID
		}

		searchResults, err = s.vectorStore.SearchByText(ctx, req.Query, searchOptions)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("文本搜索失败: %w", err)
		}
		log.Printf("[上下文服务V2] 文本搜索耗时: %v", time.Since(startTime))
	} else {
		// 按会话ID检索
		startTime := time.Now()
		filter := fmt.Sprintf(`session_id="%s"`, req.SessionID)
		searchOptions.Limit = 10
		searchResults, err = s.vectorStore.SearchByFilter(ctx, filter, searchOptions)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过会话ID检索失败: %w", err)
		}
		log.Printf("[上下文服务V2] 会话ID检索耗时: %v", time.Since(startTime))
	}

	// 组装相关记忆内容
	for _, result := range searchResults {
		if content, ok := result.Fields["content"].(string); ok {
			formattedContent := fmt.Sprintf("[相似度:%.4f] %s", result.Score, content)
			relevantMemories = append(relevantMemories, formattedContent)
		}
	}

	// 构建响应
	response := models.ContextResponse{
		SessionState:      sessionState,
		ShortTermMemory:   formatMemories(recentHistory, "最近对话"),
		LongTermMemory:    formatMemories(relevantMemories, "相关历史"),
		RelevantKnowledge: "",
	}

	log.Printf("[上下文服务V2] 成功检索上下文，会话: %s, 短期记忆数: %d, 长期记忆数: %d",
		req.SessionID, len(recentHistory), len(relevantMemories))
	return response, nil
}

// GetVectorStore 获取向量存储实例
func (s *ContextServiceV2) GetVectorStore() models.VectorStore {
	return s.vectorStore
}

// MigrateFromLegacyService 从原有服务迁移的辅助方法
func MigrateFromLegacyService(legacyService *ContextService) *ContextServiceV2 {
	log.Printf("[迁移助手] 开始从原有服务迁移到新版本")

	newService := NewContextServiceV2FromLegacy(legacyService)

	log.Printf("[迁移助手] 迁移完成，新服务已就绪")
	return newService
}
