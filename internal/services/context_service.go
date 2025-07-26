package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/aliyun"
)

// ContextService 提供上下文管理功能
type ContextService struct {
	vectorService      *aliyun.VectorService
	vectorStore        models.VectorStore // 新增：抽象向量存储接口
	sessionStore       *store.SessionStore
	userSessionManager *store.UserSessionManager
	config             *config.Config
}

// NewContextService 创建新的上下文服务
func NewContextService(vectorSvc *aliyun.VectorService, sessionStore *store.SessionStore, cfg *config.Config) *ContextService {
	// 使用同样的存储路径为UserSessionManager创建基础路径
	// 修复：直接使用sessionStore的完整路径作为基础路径，确保用户隔离存储在正确的目录下
	baseStorePath := sessionStore.GetStorePath()
	userSessionManager := store.NewUserSessionManager(baseStorePath)

	return &ContextService{
		vectorService:      vectorSvc,
		vectorStore:        nil, // 初始为nil，表示使用传统vectorService
		sessionStore:       sessionStore,
		userSessionManager: userSessionManager,
		config:             cfg,
	}
}

// SetVectorStore 设置新的向量存储接口
// 这允许ContextService动态切换到新的向量存储实现
func (s *ContextService) SetVectorStore(vectorStore models.VectorStore) {
	log.Printf("[上下文服务] 切换到新的向量存储接口")
	s.vectorStore = vectorStore
	log.Printf("[上下文服务] 向量存储接口切换完成，现在使用抽象接口")
}

// GetCurrentVectorService 获取当前使用的向量服务
// 如果设置了新的vectorStore，则返回它；否则返回传统的vectorService
func (s *ContextService) GetCurrentVectorService() interface{} {
	if s.vectorStore != nil {
		return s.vectorStore
	}
	return s.vectorService
}

// generateEmbedding 统一的向量生成接口
// 自动选择使用新接口或传统接口生成向量
func (s *ContextService) generateEmbedding(content string) ([]float32, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口生成向量")
		// 新接口返回[]float32，直接返回
		return s.vectorStore.GenerateEmbedding(content)
	}
	log.Printf("[上下文服务] 使用传统向量服务生成向量")
	// 传统接口也返回[]float32
	return s.vectorService.GenerateEmbedding(content)
}

// storeMemory 统一的记忆存储接口
// 自动选择使用新接口或传统接口存储记忆
func (s *ContextService) storeMemory(memory *models.Memory) error {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口存储记忆")
		return s.vectorStore.StoreMemory(memory)
	}
	log.Printf("[上下文服务] 使用传统向量服务存储记忆")
	return s.vectorService.StoreVectors(memory)
}

// searchByID 统一的ID搜索接口
func (s *ContextService) searchByID(ctx context.Context, id string, idType string) ([]models.SearchResult, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口按ID搜索")
		searchOptions := &models.SearchOptions{
			Limit:         10,
			SkipThreshold: true,
		}
		return s.vectorStore.SearchByID(ctx, id, searchOptions)
	}
	log.Printf("[上下文服务] 使用传统向量服务按ID搜索")
	return s.vectorService.SearchByID(id, idType)
}

// searchByText 统一的文本搜索接口
func (s *ContextService) searchByText(ctx context.Context, query string, sessionID string, options map[string]interface{}) ([]models.SearchResult, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口文本搜索")

		// 转换选项格式
		searchOptions := &models.SearchOptions{
			Limit:         10,
			SessionID:     sessionID,
			SkipThreshold: false,
			// IsBruteSearch: 不在此处设置，根据传入参数决定
		}

		if options != nil {
			if skipThreshold, ok := options["skip_threshold_filter"].(bool); ok {
				searchOptions.SkipThreshold = skipThreshold
			}
			if userFilter, ok := options["filter"].(string); ok && strings.Contains(userFilter, "userId=") {
				// 从过滤器中提取用户ID
				re := regexp.MustCompile(`userId="([^"]+)"`)
				if matches := re.FindStringSubmatch(userFilter); len(matches) > 1 {
					searchOptions.UserID = matches[1]
				}
			}
			// 处理暴力搜索参数（仅对 Vearch 有效）
			if bruteSearch, ok := options["is_brute_search"].(int); ok {
				// 只有 Vearch 类型的向量存储才支持暴力搜索
				if s.vectorStore.GetProvider() == models.VectorStoreTypeVearch {
					searchOptions.IsBruteSearch = bruteSearch
					log.Printf("[上下文服务] 检测到 Vearch 存储，启用暴力搜索参数: %d", bruteSearch)
				} else {
					log.Printf("[上下文服务] 检测到 %s 存储，忽略暴力搜索参数", s.vectorStore.GetProvider())
				}
			}
		}

		return s.vectorStore.SearchByText(ctx, query, searchOptions)
	}

	// 传统接口搜索
	log.Printf("[上下文服务] 使用传统向量服务文本搜索")

	// 生成查询向量
	queryVector, err := s.vectorService.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 执行搜索
	limit := 10
	if limitVal, ok := options["limit"].(int); ok && limitVal > 0 {
		limit = limitVal
	}

	return s.vectorService.SearchVectorsAdvanced(queryVector, sessionID, limit, options)
}

// searchBySessionID 统一的会话ID搜索接口
func (s *ContextService) searchBySessionID(ctx context.Context, sessionID string, limit int) ([]models.SearchResult, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口按会话ID搜索")
		filter := fmt.Sprintf(`session_id="%s"`, sessionID)
		searchOptions := &models.SearchOptions{
			Limit:         limit,
			SkipThreshold: true,
		}
		return s.vectorStore.SearchByFilter(ctx, filter, searchOptions)
	}
	log.Printf("[上下文服务] 使用传统向量服务按会话ID搜索")
	return s.vectorService.SearchBySessionID(sessionID, limit)
}

// countMemories 统一的记忆计数接口
func (s *ContextService) countMemories(sessionID string) (int, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口计数记忆")
		return s.vectorStore.CountMemories(sessionID)
	}
	log.Printf("[上下文服务] 使用传统向量服务计数记忆")
	return s.vectorService.CountSessionMemories(sessionID)
}

// SessionStore 返回会话存储实例
func (s *ContextService) SessionStore() *store.SessionStore {
	return s.sessionStore
}

// GetUserSessionStore 获取指定用户的会话存储
func (s *ContextService) GetUserSessionStore(userID string) (*store.SessionStore, error) {
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
func (s *ContextService) CountSessionMemories(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	// 使用统一接口计数记忆
	count, err := s.countMemories(sessionID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":     count,
		"timestamp": time.Now().Unix(),
	}, nil
}

// StoreContext 存储上下文内容
func (s *ContextService) StoreContext(ctx context.Context, req models.StoreContextRequest) (string, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 接收存储请求: 会话ID=%s, 内容长度=%d字节",
		req.SessionID, len(req.Content))

	// 创建记忆对象
	memory := models.NewMemory(req.SessionID, req.Content, req.Priority, req.Metadata)

	// 如果请求中有设置bizType，直接设置到Memory结构体中
	if req.BizType > 0 {
		log.Printf("设置业务类型: %d", req.BizType)
		memory.BizType = req.BizType
	}

	// 如果请求中有设置userId，直接设置到Memory结构体中
	if req.UserID != "" {
		log.Printf("设置用户ID: %s", req.UserID)
		memory.UserID = req.UserID
	}

	startTime := time.Now()
	// 使用统一接口生成嵌入向量
	vector, err := s.generateEmbedding(req.Content)
	if err != nil {
		log.Printf("生成嵌入向量失败: %v", err)
		return "", fmt.Errorf("生成嵌入向量失败: %w", err)
	}
	log.Printf("[上下文服务] 向量生成耗时: %v", time.Since(startTime))

	// 设置向量
	memory.Vector = vector

	// 使用统一接口存储到向量数据库
	startTime = time.Now()
	if err := s.storeMemory(memory); err != nil {
		return "", fmt.Errorf("存储向量失败: %w", err)
	}
	log.Printf("[上下文服务] 向量存储耗时: %v", time.Since(startTime))

	// 更新会话信息
	if err := s.sessionStore.UpdateSession(req.SessionID, req.Content); err != nil {
		log.Printf("[上下文服务] 警告: 更新会话信息失败: %v", err)
		// 继续执行，不返回错误
	}

	log.Printf("[上下文服务] 成功存储记忆 ID: %s, 会话: %s", memory.ID, memory.SessionID)
	log.Printf("==================================================== 存储上下文完成 ====================================================")
	return memory.ID, nil
}

// RetrieveContext 检索相关上下文
func (s *ContextService) RetrieveContext(ctx context.Context, req models.RetrieveContextRequest) (models.ContextResponse, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 接收检索请求: 会话ID=%s, 查询=%s, 限制=%d字节, MemoryID=%s, BatchID=%s",
		req.SessionID, req.Query, req.Limit, req.MemoryID, req.BatchID)

	if req.Limit <= 0 {
		req.Limit = 2000 // 默认长度限制
	}

	// 获取会话状态
	sessionState, err := s.sessionStore.GetSessionState(req.SessionID)
	if err != nil {
		log.Printf("[上下文服务] 警告: 获取会话状态失败: %v", err)
		sessionState = fmt.Sprintf("会话ID: %s", req.SessionID)
	}

	// 获取最近的对话历史
	recentHistory, err := s.sessionStore.GetRecentHistory(req.SessionID, 5)
	if err != nil {
		log.Printf("[上下文服务] 警告: 获取最近历史失败: %v", err)
		recentHistory = []string{}
	}

	var searchResults []models.SearchResult
	var relevantMemories []string

	// 根据请求类型选择不同的检索方式
	if req.MemoryID != "" {
		// 使用记忆ID精确检索
		startTime := time.Now()
		searchResults, err = s.searchByID(ctx, req.MemoryID, "id")
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过记忆ID检索失败: %w", err)
		}
		log.Printf("[上下文服务] 记忆ID检索耗时: %v", time.Since(startTime))

		// 从搜索结果中提取会话ID
		if len(searchResults) > 0 {
			if sessionID, ok := searchResults[0].Fields["session_id"].(string); ok && sessionID != "" {
				// 更新会话ID
				req.SessionID = sessionID

				// 重新获取会话状态
				sessionState, err = s.sessionStore.GetSessionState(sessionID)
				if err != nil {
					log.Printf("[上下文服务] 警告: 获取会话状态失败: %v", err)
					sessionState = fmt.Sprintf("会话ID: %s", sessionID)
				}

				// 重新获取最近对话历史
				recentHistory, err = s.sessionStore.GetRecentHistory(sessionID, 5)
				if err != nil {
					log.Printf("[上下文服务] 警告: 获取最近历史失败: %v", err)
					recentHistory = []string{}
				}

				log.Printf("[上下文服务] 从记忆ID %s 中提取到会话ID: %s", req.MemoryID, sessionID)
			}
		}
	} else if req.BatchID != "" {
		// 使用批次ID检索 - 直接使用ID检索方式而不是filter
		startTime := time.Now()
		// 使用专门用于批次ID检索的方法
		searchResults, err = s.searchByID(ctx, req.BatchID, "id")
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过批次ID检索失败: %w", err)
		}
		log.Printf("[上下文服务] 批次ID检索耗时: %v", time.Since(startTime))

		// 从搜索结果中提取会话ID（如果当前会话ID为空）
		if req.SessionID == "" && len(searchResults) > 0 {
			if sessionID, ok := searchResults[0].Fields["session_id"].(string); ok && sessionID != "" {
				// 更新会话ID
				req.SessionID = sessionID

				// 重新获取会话状态
				sessionState, err = s.sessionStore.GetSessionState(sessionID)
				if err != nil {
					log.Printf("[上下文服务] 警告: 获取会话状态失败: %v", err)
					sessionState = fmt.Sprintf("会话ID: %s", sessionID)
				}

				// 重新获取最近对话历史
				recentHistory, err = s.sessionStore.GetRecentHistory(sessionID, 5)
				if err != nil {
					log.Printf("[上下文服务] 警告: 获取最近历史失败: %v", err)
					recentHistory = []string{}
				}

				log.Printf("[上下文服务] 从批次ID %s 中提取到会话ID: %s", req.BatchID, sessionID)
			}
		}
	} else if req.Query != "" {
		// 检查查询内容
		if strings.TrimSpace(req.Query) == "" {
			return models.ContextResponse{}, fmt.Errorf("查询内容不能为空")
		}

		// 标准向量相似度搜索
		// 生成查询向量
		startTime := time.Now()
		queryVector, err := s.generateEmbedding(req.Query)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("生成查询向量失败: %w", err)
		}
		log.Printf("[上下文服务] 查询向量生成耗时: %v", time.Since(startTime))

		// 在向量数据库中搜索相似向量
		startTime = time.Now()

		// 使用高级向量搜索，支持跳过相似度阈值过滤
		options := make(map[string]interface{})
		if req.SkipThreshold {
			options["skip_threshold_filter"] = true
		}
		// 传递暴力搜索参数
		if req.IsBruteSearch > 0 {
			options["is_brute_search"] = req.IsBruteSearch
		}

		//options["filter"] = "" // 覆盖默认的会话ID过滤器
		// 🔥 修复：从会话ID获取用户ID，实现真正的多用户隔离
		var filterConditions []string

		// 从会话ID获取用户ID
		userID, err := s.GetUserIDFromSessionID(req.SessionID)
		if err != nil {
			log.Printf("[上下文服务] 从会话获取用户ID失败: %v，为保护数据安全，拒绝执行搜索", err)
			return models.ContextResponse{}, fmt.Errorf("安全错误: 从会话获取用户ID失败: %w", err)
		}

		if userID != "" {
			filterConditions = append(filterConditions, fmt.Sprintf(`userId="%s"`, userID))
			log.Printf("[上下文服务] 🔥 从会话%s获取用户ID: %s，添加过滤条件", req.SessionID, userID)
		} else {
			log.Printf("[上下文服务] 严重安全错误: 会话%s中未找到用户ID，为保护数据安全，拒绝执行搜索", req.SessionID)
			return models.ContextResponse{}, fmt.Errorf("安全错误: 会话中未找到用户ID，拒绝执行搜索以防止数据泄露")
		}

		// 构建最终过滤器
		if len(filterConditions) > 0 {
			//基于用户隔离数据的开关
			options["filter"] = strings.Join(filterConditions, " AND ")
			log.Printf("[上下文服务] 使用过滤条件: %s", options["filter"])
		}

		searchResults, err = s.searchByVector(ctx, queryVector, "", options)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("向量搜索失败: %w", err)
		}
		log.Printf("[上下文服务] 向量搜索耗时: %v", time.Since(startTime))
	} else {
		// 如果既没有ID也没有查询关键词，则按会话ID检索
		startTime := time.Now()
		searchResults, err = s.searchBySessionID(ctx, req.SessionID, 10)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("通过会话ID检索失败: %w", err)
		}
		log.Printf("[上下文服务] 会话ID检索耗时: %v", time.Since(startTime))
	}

	// 组装相关记忆内容 - 按相似度排序（余弦距离：越小越相似）
	//TODO  这个排序逻辑 放到存储引擎层，放到不同的实现中，每个实现的逻辑不一样
	/*sort.Slice(searchResults, func(i, j int) bool {
		return searchResults[i].Score < searchResults[j].Score
	})*/

	for _, result := range searchResults {
		if content, ok := result.Fields["content"].(string); ok {
			// 添加相似度分数
			formattedContent := fmt.Sprintf("[相似度:%.4f] %s", result.Score, content)
			relevantMemories = append(relevantMemories, formattedContent)
		}
	}

	// 构建响应
	response := models.ContextResponse{
		SessionState:      sessionState,
		ShortTermMemory:   formatMemories(recentHistory, "最近对话"),
		LongTermMemory:    formatMemories(relevantMemories, "相关历史"),
		RelevantKnowledge: "", // V1版本暂不实现
	}

	log.Printf("[上下文服务] 成功检索上下文，会话: %s, 短期记忆数: %d, 长期记忆数: %d",
		req.SessionID, len(recentHistory), len(relevantMemories))
	log.Printf("==================================================== 检索上下文完成 ====================================================")
	return response, nil
}

// SummarizeContext 生成会话摘要
func (s *ContextService) SummarizeContext(ctx context.Context, req models.SummarizeContextRequest) (string, error) {
	// 获取会话历史
	history, err := s.sessionStore.GetRecentHistory(req.SessionID, 20) // 获取更多历史用于摘要
	if err != nil {
		return "", fmt.Errorf("获取会话历史失败: %w", err)
	}

	if len(history) == 0 {
		return "会话尚无内容", nil
	}

	// V1版本简单实现: 直接返回历史记录数量和前几条内容的简单摘要
	summary := fmt.Sprintf("会话包含%d条记录。", len(history))

	// 添加最新几条记录的简单表示
	maxPreview := 3
	if len(history) < maxPreview {
		maxPreview = len(history)
	}

	recentItems := history[len(history)-maxPreview:]
	for i, item := range recentItems {
		// 截断过长内容
		if len(item) > 100 {
			item = item[:97] + "..."
		}
		summary += fmt.Sprintf("\n最近记录%d: %s", i+1, item)
	}

	// 更新会话摘要
	if err := s.sessionStore.UpdateSessionSummary(req.SessionID, summary); err != nil {
		log.Printf("[上下文服务] 警告: 更新会话摘要失败: %v", err)
		// 继续执行，不返回错误
	}

	return summary, nil
}

// 格式化记忆列表为易读字符串
func formatMemories(memories []string, title string) string {
	if len(memories) == 0 {
		return fmt.Sprintf("【%s】\n无相关内容", title)
	}

	result := fmt.Sprintf("【%s】\n", title)
	for i, memory := range memories {
		result += fmt.Sprintf("%d. %s\n", i+1, memory)
	}
	return result
}

// StoreMessages 存储对话消息
func (s *ContextService) StoreMessages(ctx context.Context, req models.StoreMessagesRequest) (*models.StoreMessagesResponse, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 收到存储消息请求: 会话ID=%s, 消息数=%d",
		req.SessionID, len(req.Messages))

	var messageIDs []string
	var err error

	// 获取或创建会话
	_, err = s.sessionStore.GetSession(req.SessionID)
	if err != nil {
		// 获取会话失败，但会话会在GetSession内部创建
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	start := time.Now()

	for _, msgReq := range req.Messages {
		// 创建新消息
		message := models.NewMessage(
			req.SessionID,
			msgReq.Role,
			msgReq.Content,
			msgReq.ContentType,
			msgReq.Priority,
			msgReq.Metadata,
		)

		// 生成向量表示
		vector, err := s.generateEmbedding(message.Content)
		if err != nil {
			return nil, fmt.Errorf("生成向量失败: %w", err)
		}
		message.Vector = vector

		// 存储消息
		if err := s.vectorService.StoreMessage(message); err != nil {
			return nil, fmt.Errorf("存储消息失败: %w", err)
		}

		messageIDs = append(messageIDs, message.ID)
	}

	// 更新会话最后活动时间（通过UpdateSession方法）
	if len(req.Messages) > 0 {
		lastMessage := req.Messages[len(req.Messages)-1]
		err = s.sessionStore.UpdateSession(req.SessionID, lastMessage.Content)
		if err != nil {
			log.Printf("[警告] 更新会话活动时间失败: %v", err)
		}
	}

	log.Printf("[上下文服务] 存储消息完成，共 %d 条，耗时: %v", len(messageIDs), time.Since(start))
	log.Printf("==================================================== 存储对话消息完成 ====================================================")
	return &models.StoreMessagesResponse{
		MessageIDs: messageIDs,
		Status:     "success",
	}, nil
}

// RetrieveConversation 检索对话
func (s *ContextService) RetrieveConversation(ctx context.Context, req models.RetrieveConversationRequest) (*models.ConversationResponse, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 收到检索对话请求: 会话ID=%s, 查询=%s, 限制=%d, MessageID=%s, BatchID=%s",
		req.SessionID, req.Query, req.Limit, req.MessageID, req.BatchID)

	start := time.Now()

	// 获取会话信息
	session, err := s.sessionStore.GetSession(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话信息失败: %w", err)
	}

	// 初始化响应
	response := &models.ConversationResponse{
		SessionID: req.SessionID,
		SessionInfo: &models.SessionSummary{
			ID:         session.ID,
			CreatedAt:  session.CreatedAt,
			LastActive: session.LastActive,
			Summary:    session.Summary,
		},
		Messages: []*models.Message{},
	}

	var searchResults []models.SearchResult

	// 根据请求类型选择不同的检索方式
	if req.BatchID != "" {
		// 通过批次ID检索 (metadata中的batchId字段)
		searchResults, err = s.searchByID(ctx, req.BatchID, "id")
		if err != nil {
			return nil, fmt.Errorf("通过批次ID检索失败: %w", err)
		}

		// 从搜索结果中提取会话ID（如果当前会话ID为空）
		if req.SessionID == "" && len(searchResults) > 0 {
			if sessionID, ok := searchResults[0].Fields["session_id"].(string); ok && sessionID != "" {
				// 更新会话ID和响应中的会话ID
				req.SessionID = sessionID
				response.SessionID = sessionID

				// 重新获取会话信息
				session, err = s.sessionStore.GetSession(sessionID)
				if err == nil {
					response.SessionInfo = &models.SessionSummary{
						ID:         session.ID,
						CreatedAt:  session.CreatedAt,
						LastActive: session.LastActive,
						Summary:    session.Summary,
					}
				}
				log.Printf("[上下文服务] 从批次ID %s 中提取到会话ID: %s", req.BatchID, sessionID)
			}
		}
	} else if req.Query != "" {
		// 如果有查询关键词，进行相关性搜索
		// 生成查询向量
		queryStart := time.Now()
		vector, err := s.generateEmbedding(req.Query)
		if err != nil {
			return nil, fmt.Errorf("生成查询向量失败: %w", err)
		}
		log.Printf("[上下文服务] 查询向量生成耗时: %v", time.Since(queryStart))

		// 使用高级搜索选项
		options := make(map[string]interface{})
		if req.SkipThreshold {
			options["skip_threshold_filter"] = true
		}

		// 搜索相关消息
		searchStart := time.Now()

		// 🔥 修复：从会话ID获取用户ID，实现真正的多用户隔离
		var filterConditions []string

		// 从会话ID获取用户ID
		userID, err := s.GetUserIDFromSessionID(req.SessionID)
		if err != nil {
			log.Printf("[上下文服务] 从会话获取用户ID失败: %v，为保护数据安全，拒绝执行搜索", err)
			return nil, fmt.Errorf("安全错误: 从会话获取用户ID失败: %w", err)
		}

		if userID != "" {
			filterConditions = append(filterConditions, fmt.Sprintf(`userId="%s"`, userID))
			log.Printf("[上下文服务] 🔥 从会话%s获取用户ID: %s，添加过滤条件", req.SessionID, userID)
		} else {
			log.Printf("[上下文服务] 严重安全错误: 会话%s中未找到用户ID，为保护数据安全，拒绝执行搜索", req.SessionID)
			return nil, fmt.Errorf("安全错误: 会话中未找到用户ID，拒绝执行搜索以防止数据泄露")
		}

		// 构建最终过滤器
		if len(filterConditions) > 0 {
			//基于用户隔离数据的开关
			options["filter"] = strings.Join(filterConditions, " AND ")
			log.Printf("[上下文服务] 使用过滤条件: %s", options["filter"])
		}

		results, err := s.searchByVector(ctx, vector, req.SessionID, options)
		if err != nil {
			return nil, fmt.Errorf("搜索相关消息失败: %w", err)
		}
		log.Printf("[上下文服务] 向量搜索耗时: %v", time.Since(searchStart))

		searchResults = results
	} else {
		// 没有查询关键词，获取最近对话
		limit := req.Limit
		if limit <= 0 {
			limit = 10 // 默认返回10条
		}

		// 直接使用会话ID查询获取最近消息
		searchResults, err = s.searchBySessionID(ctx, req.SessionID, limit)
		if err != nil {
			return nil, fmt.Errorf("获取最近消息失败: %w", err)
		}
	}

	// 构造消息对象
	for _, result := range searchResults {
		message := resultToMessage(result)
		response.Messages = append(response.Messages, message)
		response.RelevantIndices = append(response.RelevantIndices, len(response.Messages)-1)
	}

	// 如果需要按相关性排序
	if req.Format == "relevant" {
		// 已经按相关性排序了，无需额外处理
	} else {
		// 默认按时间顺序排序
		sortMessagesByTime(response.Messages)
		// 更新相关索引
		updateRelevantIndices(response)
	}

	// 统计各类型消息数量
	userMsgs, assistantMsgs := 0, 0
	for _, msg := range response.Messages {
		if msg.Role == models.RoleUser {
			userMsgs++
		} else if msg.Role == models.RoleAssistant {
			assistantMsgs++
		}
	}

	response.SessionInfo.MessageCount = len(response.Messages)
	response.SessionInfo.UserMessages = userMsgs
	response.SessionInfo.AgentMessages = assistantMsgs

	log.Printf("[上下文服务] 成功检索对话，会话: %s, 消息数: %d, 用户/助手: %d/%d, 耗时: %v",
		req.SessionID, len(response.Messages), userMsgs, assistantMsgs, time.Since(start))
	log.Printf("==================================================== 检索对话完成 ====================================================")
	return response, nil
}

// resultToMessage 将搜索结果转换为消息对象
func resultToMessage(result models.SearchResult) *models.Message {
	msg := &models.Message{
		ID: result.ID,
	}

	// 提取字段
	if content, ok := result.Fields["content"].(string); ok {
		msg.Content = content
	}
	if sessionID, ok := result.Fields["session_id"].(string); ok {
		msg.SessionID = sessionID
	}
	if role, ok := result.Fields["role"].(string); ok {
		msg.Role = role
	} else {
		// 兼容旧数据，如果没有role字段，尝试从metadata中获取
		if metadataStr, ok := result.Fields["metadata"].(string); ok && metadataStr != "{}" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
				if typeVal, ok := metadata[models.MetadataTypeKey].(string); ok {
					switch typeVal {
					case models.MetadataTypeUser:
						msg.Role = models.RoleUser
					case models.MetadataTypeAssistant:
						msg.Role = models.RoleAssistant
					case models.MetadataTypeSystem:
						msg.Role = models.RoleSystem
					}
				}
			}
		}

		// 如果无法确定角色，默认为用户
		if msg.Role == "" {
			msg.Role = models.RoleUser
		}
	}

	if contentType, ok := result.Fields["content_type"].(string); ok {
		msg.ContentType = contentType
	} else {
		msg.ContentType = "text" // 默认为文本
	}

	if timestamp, ok := result.Fields["timestamp"].(float64); ok {
		msg.Timestamp = int64(timestamp)
	}

	if priority, ok := result.Fields["priority"].(string); ok {
		msg.Priority = priority
	}

	// 解析元数据
	if metadataStr, ok := result.Fields["metadata"].(string); ok && metadataStr != "{}" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			msg.Metadata = metadata
		}
	}

	return msg
}

// sortMessagesByTime 按时间排序消息
func sortMessagesByTime(messages []*models.Message) {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})
}

// updateRelevantIndices 更新相关索引
func updateRelevantIndices(response *models.ConversationResponse) {
	if len(response.RelevantIndices) == 0 {
		return
	}

	// 创建ID到新位置的映射
	idToIndex := make(map[string]int)
	for i, msg := range response.Messages {
		idToIndex[msg.ID] = i
	}

	// 更新相关索引数组
	relevantIDs := make(map[string]bool)
	for _, idx := range response.RelevantIndices {
		if idx < len(response.Messages) {
			relevantIDs[response.Messages[idx].ID] = true
		}
	}

	// 重建索引数组
	response.RelevantIndices = []int{}
	for id := range relevantIDs {
		if idx, ok := idToIndex[id]; ok {
			response.RelevantIndices = append(response.RelevantIndices, idx)
		}
	}

	// 排序
	sort.Ints(response.RelevantIndices)
}

// StoreSessionMessages 存储会话消息
func (s *ContextService) StoreSessionMessages(ctx context.Context, req models.StoreMessagesRequest) (*models.StoreMessagesResponse, error) {
	log.Printf("[上下文服务] 接收消息存储请求: 会话ID=%s, 消息数量=%d", req.SessionID, len(req.Messages))

	// 转换消息格式
	messages := make([]*models.Message, 0, len(req.Messages))
	for _, msgReq := range req.Messages {
		// 创建元数据
		metadata := make(map[string]interface{})
		for k, v := range msgReq.Metadata {
			metadata[k] = v
		}

		// 批次ID放入元数据
		if req.BatchID != "" {
			metadata["batchId"] = req.BatchID
		}

		// 创建消息对象
		message := models.NewMessage(
			req.SessionID,
			msgReq.Role,
			msgReq.Content,
			msgReq.ContentType,
			msgReq.Priority,
			metadata,
		)
		messages = append(messages, message)
	}

	// 存储到用户隔离的会话
	userID := utils.GetCachedUserID()
	userSessionStore, err := s.GetUserSessionStore(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户会话存储失败: %w", err)
	}

	if err := userSessionStore.StoreMessages(req.SessionID, messages); err != nil {
		return nil, fmt.Errorf("存储消息失败: %w", err)
	}

	// 收集消息ID
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	// 创建响应
	response := &models.StoreMessagesResponse{
		MessageIDs: messageIDs,
		Status:     "success",
	}

	// 如果需要汇总并存储
	if req.SummarizeAndStore {
		// 生成摘要
		summary := s.GenerateMessagesSummary(messages)

		// 存储摘要
		var memoryID string
		var err error

		if req.BatchID != "" {
			// 使用批次ID存储
			metadata := map[string]interface{}{
				"type":      "conversation_summary",
				"batchId":   req.BatchID,
				"timestamp": time.Now().Unix(),
			}

			memory := models.NewMemory(req.SessionID, summary, "P1", metadata)

			// 生成向量表示
			startTime := time.Now()
			vector, err := s.generateEmbedding(summary)
			if err != nil {
				return response, fmt.Errorf("生成向量失败: %w", err)
			}
			log.Printf("[上下文服务] 向量生成耗时: %v", time.Since(startTime))

			// 设置向量
			memory.Vector = vector

			// 使用统一接口存储到向量数据库
			startTime = time.Now()
			if err := s.storeMemory(memory); err != nil {
				return response, fmt.Errorf("存储向量失败: %w", err)
			}
			log.Printf("[上下文服务] 向量存储耗时: %v", time.Since(startTime))

			memoryID = memory.ID
		} else {
			// 使用普通方式存储
			storeReq := models.StoreContextRequest{
				SessionID: req.SessionID,
				Content:   summary,
				Priority:  "P1",
				Metadata: map[string]interface{}{
					"type": "conversation_summary",
				},
			}

			memoryID, err = s.StoreContext(ctx, storeReq)
			if err != nil {
				return response, fmt.Errorf("存储摘要失败: %w", err)
			}
		}

		// 将内存ID添加到响应
		response.MemoryID = memoryID
	}

	log.Printf("[上下文服务] 成功存储消息: 会话=%s, 消息数量=%d, 摘要=%v",
		req.SessionID, len(messages), req.SummarizeAndStore)
	return response, nil
}

// GenerateMessagesSummary 生成消息摘要
func (s *ContextService) GenerateMessagesSummary(messages []*models.Message) string {
	// 简单实现：连接所有消息内容
	var summary strings.Builder

	// 添加用户和系统消息的内容
	for _, msg := range messages {
		if msg.Role == models.RoleUser || msg.Role == models.RoleSystem {
			// 只添加用户和系统消息
			if summary.Len() > 0 {
				summary.WriteString(" ")
			}
			summary.WriteString(msg.Content)
		}
	}

	// 如果摘要太长，可以截断
	maxLen := 1000 // 最大摘要长度
	content := summary.String()
	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}

	return content
}

// AssociateCodeFile 关联代码文件到会话
func (s *ContextService) AssociateCodeFile(ctx context.Context, req models.MCPCodeAssociationRequest) error {
	log.Printf("[上下文服务] 关联代码文件: 会话ID=%s, 文件路径=%s, 语言=%s",
		req.SessionID, req.FilePath, req.Language)

	// 存储文件关联
	if err := s.sessionStore.AssociateFile(req.SessionID, req.FilePath, req.Language, req.Content); err != nil {
		return fmt.Errorf("关联文件失败: %w", err)
	}

	// 如果提供了文件内容，可以存储为向量供后续检索
	if req.Content != "" {
		// 作为上下文存储
		metadata := map[string]interface{}{
			"type":      "code_file",
			"file_path": req.FilePath,
			"language":  req.Language,
		}

		// 只存储有意义的代码片段
		if len(req.Content) > 50 {
			storeReq := models.StoreContextRequest{
				SessionID: req.SessionID,
				Content:   req.Content,
				Priority:  "P1", // 代码文件较为重要
				Metadata:  metadata,
			}

			memoryID, err := s.StoreContext(ctx, storeReq)
			if err != nil {
				log.Printf("[上下文服务] 警告: 存储代码内容向量失败: %v", err)
				// 不返回错误，继续执行
			} else {
				// 记录向量存储ID，方便后续引用
				metadata["memory_id"] = memoryID
			}

			// 智能关联：查找与代码文件相关的对话记录
			if err := s.establishCodeContextLinks(ctx, req.SessionID, req.FilePath, req.Content, req.Language); err != nil {
				log.Printf("[上下文服务] 警告: 建立代码上下文链接失败: %v", err)
				// 不返回错误，继续执行
			}
		}
	}

	log.Printf("[上下文服务] 成功关联代码文件: 会话ID=%s, 文件路径=%s", req.SessionID, req.FilePath)
	return nil
}

// establishCodeContextLinks 建立代码与上下文的智能链接
func (s *ContextService) establishCodeContextLinks(ctx context.Context, sessionID, filePath, codeContent, language string) error {
	log.Printf("[上下文服务] 开始建立代码上下文智能链接: 会话ID=%s, 文件路径=%s", sessionID, filePath)

	// 1. 提取代码的关键特征
	features := extractCodeFeatures(codeContent, language)

	// 2. 构建搜索查询
	var searchQuery string
	if len(features) > 0 {
		// 使用提取的特征构建查询
		searchQuery = strings.Join(limitSliceLength(features, 5), " ")
	} else {
		// 使用文件路径作为备选查询
		searchQuery = fmt.Sprintf("关于 %s 的讨论", filepath.Base(filePath))
	}

	log.Printf("[上下文服务] 代码关联搜索查询: %s", searchQuery)

	// 3. 在向量数据库中搜索相关对话
	vector, err := s.generateEmbedding(searchQuery)
	if err != nil {
		return fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 搜索选项
	options := make(map[string]interface{})
	options["skip_threshold_filter"] = true

	// 🔥 修复：从会话ID获取用户ID，实现真正的多用户隔离
	userID, err := s.GetUserIDFromSessionID(sessionID)
	if err != nil {
		log.Printf("[上下文服务] 从会话获取用户ID失败: %v，为保护数据安全，拒绝执行搜索", err)
		return fmt.Errorf("安全错误: 从会话获取用户ID失败: %w", err)
	}

	if userID != "" {
		options["filter"] = fmt.Sprintf(`userId="%s"`, userID)
		log.Printf("[上下文服务] 🔥 从会话%s获取用户ID: %s，添加过滤条件", sessionID, userID)
	}

	// 执行向量搜索
	searchResults, err := s.searchByVector(ctx, vector, "", options)
	if err != nil {
		return fmt.Errorf("搜索相关对话失败: %w", err)
	}

	// 4. 处理搜索结果，建立双向引用
	var relatedDiscussions []models.DiscussionRef
	for _, result := range searchResults {
		if result.Score > 0.7 { // 过滤掉相关性较低的结果
			continue
		}

		// 确定类型
		resultType := "message"
		if typeVal, ok := result.Fields["metadata"].(string); ok {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(typeVal), &metadata); err == nil {
				if t, ok := metadata["type"].(string); ok {
					resultType = t
				}
			}
		}

		// 提取内容
		content := ""
		if c, ok := result.Fields["content"].(string); ok {
			content = c
			if len(content) > 200 {
				content = content[:197] + "..."
			}
		}

		// 创建讨论引用
		discussion := models.DiscussionRef{
			ID:        result.ID,
			Type:      resultType,
			Summary:   content,
			Timestamp: time.Now().Unix(),
			Relevance: 1.0 - result.Score, // 转换相似度为相关度
		}

		relatedDiscussions = append(relatedDiscussions, discussion)
		log.Printf("[上下文服务] 关联到相关讨论: ID=%s, 类型=%s, 相关度=%.2f",
			discussion.ID, discussion.Type, discussion.Relevance)
	}

	// 5. 更新会话的代码上下文
	if len(relatedDiscussions) > 0 {
		if err := s.sessionStore.UpdateCodeFileRelations(sessionID, filePath, relatedDiscussions); err != nil {
			return fmt.Errorf("更新代码文件关系失败: %w", err)
		}
		log.Printf("[上下文服务] 成功关联 %d 条相关讨论到文件 %s", len(relatedDiscussions), filePath)
	}

	return nil
}

// extractCodeFeatures 提取代码的关键特征
func extractCodeFeatures(codeContent string, language string) []string {
	// 简化实现：提取关键标识符
	var features []string

	// 去除注释和字符串常量
	cleanCode := removeCommentsAndStrings(codeContent, language)

	// 按语言类型选择不同的提取策略
	switch strings.ToLower(language) {
	case "go":
		// 提取函数名、结构体名等
		funcRegex := regexp.MustCompile(`func\s+(\w+)`)
		if matches := funcRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					features = append(features, match[1])
				}
			}
		}

		// 提取结构体名
		structRegex := regexp.MustCompile(`type\s+(\w+)\s+struct`)
		if matches := structRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					features = append(features, match[1])
				}
			}
		}

		// 提取接口名
		interfaceRegex := regexp.MustCompile(`type\s+(\w+)\s+interface`)
		if matches := interfaceRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					features = append(features, match[1])
				}
			}
		}

	case "javascript", "typescript", "jsx", "tsx":
		// 提取函数和类
		funcRegex := regexp.MustCompile(`(function|class)\s+(\w+)`)
		if matches := funcRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 2 {
					features = append(features, match[2])
				}
			}
		}

		// 提取导出变量
		exportRegex := regexp.MustCompile(`export\s+(const|let|var)\s+(\w+)`)
		if matches := exportRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 2 {
					features = append(features, match[2])
				}
			}
		}

	case "python":
		// 提取类名和函数名
		classRegex := regexp.MustCompile(`class\s+(\w+)`)
		if matches := classRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					features = append(features, match[1])
				}
			}
		}

		// 提取函数
		funcRegex := regexp.MustCompile(`def\s+(\w+)`)
		if matches := funcRegex.FindAllStringSubmatch(cleanCode, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					features = append(features, match[1])
				}
			}
		}

	default:
		// 通用提取标识符的策略
		// 提取可能的函数名（大驼峰命名的标识符）
		funcRegex := regexp.MustCompile(`[A-Z][a-z0-9]+([A-Z][a-z0-9]+)+`)
		if matches := funcRegex.FindAllString(cleanCode, -1); matches != nil {
			for _, match := range matches {
				features = append(features, match)
			}
		}
	}

	// 从文件路径中提取特征
	//pathFeatures := extractPathFeatures(filePath)
	//features = append(features, pathFeatures...)

	// 去重
	return uniqueStrings(features)
}

// removeCommentsAndStrings 移除代码中的注释和字符串常量
func removeCommentsAndStrings(code string, language string) string {
	// 简化实现
	// 去除单行注释
	singleLineComment := regexp.MustCompile(`//.*$`)
	multiLineComment := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	stringLiteral := regexp.MustCompile(`"[^"]*"`)

	result := code
	result = singleLineComment.ReplaceAllString(result, "")
	result = multiLineComment.ReplaceAllString(result, "")
	result = stringLiteral.ReplaceAllString(result, `""`)

	return result
}

// uniqueStrings 去除字符串数组中的重复项
func uniqueStrings(strings []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strings {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// 使用函数截取slice，避免使用可能与其他代码冲突的min函数
func limitSliceLength(slice []string, maxLen int) []string {
	if len(slice) <= maxLen {
		return slice
	}
	return slice[:maxLen]
}

// RecordEditAction 记录编辑操作
func (s *ContextService) RecordEditAction(ctx context.Context, req models.MCPEditRecordRequest) error {
	log.Printf("[上下文服务] 记录编辑操作: 会话ID=%s, 文件路径=%s, 类型=%s",
		req.SessionID, req.FilePath, req.Type)

	// 存储编辑记录
	if err := s.sessionStore.RecordEditAction(req.SessionID, req.FilePath, req.Type, req.Position, req.Content); err != nil {
		return fmt.Errorf("记录编辑操作失败: %w", err)
	}

	log.Printf("[上下文服务] 成功记录编辑操作: 会话ID=%s, 文件路径=%s", req.SessionID, req.FilePath)
	return nil
}

// GetSessionState 获取会话状态
func (s *ContextService) GetSessionState(ctx context.Context, sessionID string) (*models.MCPSessionResponse, error) {
	log.Printf("[上下文服务] 获取会话状态: 会话ID=%s", sessionID)

	// 获取会话
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 创建响应
	response := &models.MCPSessionResponse{
		SessionID:    session.ID,
		CreatedAt:    session.CreatedAt,
		LastActive:   session.LastActive,
		Status:       session.Status,
		MessageCount: len(session.Messages),
	}

	log.Printf("[上下文服务] 成功获取会话状态: 会话ID=%s, 状态=%s", sessionID, session.Status)
	return response, nil
}

// GetShortTermMemory 获取会话短期记忆
func (s *ContextService) GetShortTermMemory(ctx context.Context, sessionID string, limit int) (string, error) {
	log.Printf("[上下文服务] 获取短期记忆: 会话ID=%s, 限制=%d", sessionID, limit)

	if limit <= 0 {
		limit = 5 // 默认5条
	}

	// 获取最近消息
	messages, err := s.sessionStore.GetMessages(sessionID, limit)
	if err != nil {
		return "", fmt.Errorf("获取消息失败: %w", err)
	}

	// 格式化消息
	var result strings.Builder
	result.WriteString("【最近对话】\n")

	if len(messages) == 0 {
		result.WriteString("无相关内容")
		return result.String(), nil
	}

	for i, msg := range messages {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, msg.Content))
	}

	log.Printf("[上下文服务] 成功获取短期记忆: 会话ID=%s, 消息数量=%d", sessionID, len(messages))
	return result.String(), nil
}

// StartSessionCleanupTask 启动会话清理定时任务
func (s *ContextService) StartSessionCleanupTask(ctx context.Context, timeout time.Duration, interval time.Duration) {
	log.Printf("[上下文服务] 启动会话清理任务: 超时=%v, 间隔=%v", timeout, interval)

	// 启动一个定时器，定期执行清理和汇总任务
	ticker := time.NewTicker(interval)

	// 创建一个更长间隔的定时器，用于长期记忆汇总
	// 使用配置中的间隔倍数，避免过于频繁汇总
	summaryInterval := interval * time.Duration(s.config.SummaryIntervalMultiplier)
	summaryTicker := time.NewTicker(summaryInterval)

	log.Printf("[上下文服务] 自动汇总任务已启动，间隔=%v", summaryInterval)

	go func() {
		for {
			select {
			case <-ticker.C:
				// 1. 清理不活跃会话
				count := s.sessionStore.CleanupInactiveSessions(timeout)
				log.Printf("[上下文服务] 会话清理完成: 清理了%d个不活跃会话", count)

				// 2. 清理短期记忆 (使用配置中的保留天数)
				msgCount := s.sessionStore.CleanupShortTermMemory(s.config.ShortMemoryMaxAge)
				log.Printf("[上下文服务] 短期记忆清理完成: 清理了%d条过期消息", msgCount)

			case <-summaryTicker.C:
				// 3. 定期执行自动汇总长期记忆
				go s.AutoSummarizeToLongTermMemoryWithThreshold(ctx)

			case <-ctx.Done():
				ticker.Stop()
				summaryTicker.Stop()
				log.Printf("[上下文服务] 会话清理和汇总任务已停止")
				return
			}
		}
	}()
}

// AutoSummarizeToLongTermMemoryWithThreshold 带阈值的自动汇总到长期记忆
// 只有满足特定条件的会话才会被汇总，避免无谓的资源消耗
func (s *ContextService) AutoSummarizeToLongTermMemoryWithThreshold(ctx context.Context) {
	log.Printf("[上下文服务] 开始基于阈值的自动汇总")

	// 获取所有会话（包括活跃和即将过期的会话）
	sessions := s.sessionStore.GetSessionList()

	var summarizedCount int
	var skippedCount int
	var expiredProcessedCount int

	now := time.Now()
	sessionTimeout := time.Duration(s.config.SessionTimeout) * time.Minute

	for _, session := range sessions {
		// 🔥 修复：处理活跃会话和即将过期的会话
		isActive := session.Status == "active"
		isAboutToExpire := isActive && now.Sub(session.LastActive) > sessionTimeout*80/100                         // 超过80%会话超时时间
		isRecentlyExpired := session.Status == "archived" && now.Sub(session.LastActive) <= sessionTimeout*120/100 // 过期后20%时间内

		if !isActive && !isRecentlyExpired {
			continue // 跳过太久的过期会话
		}

		// 🔥 修复：基于游标获取未汇总的消息
		lastSummaryCursor := int64(0)
		if session.Metadata != nil {
			if cursorVal, ok := session.Metadata["last_summary_cursor"].(float64); ok {
				lastSummaryCursor = int64(cursorVal)
			}
		}

		// 获取未汇总的消息（从游标位置开始）
		var messages []*models.Message
		var err error

		if lastSummaryCursor > 0 {
			// 获取游标之后的消息
			messages, err = s.getMessagesAfterCursor(session.ID, lastSummaryCursor)
		} else {
			// 首次汇总，获取所有消息
			messages, err = s.sessionStore.GetMessages(session.ID, s.config.MaxMessageCount)
		}

		if err != nil || len(messages) < s.config.MinMessageCount {
			// 消息太少，不值得汇总
			skippedCount++
			continue
		}

		// 检查汇总条件
		lastSumTime := int64(0)
		if session.Metadata != nil {
			if lastSumTimeVal, ok := session.Metadata["last_summary_time"].(float64); ok {
				lastSumTime = int64(lastSumTimeVal)
			}
		}

		currentTime := time.Now().Unix()
		hoursSinceLastSum := (currentTime - lastSumTime) / 3600

		// 判断是否满足汇总条件:
		// 1. 从未汇总过，或者距离上次汇总超过指定小时数
		// 2. 消息数量达到或超过触发阈值
		// 3. 会话即将过期且有未汇总内容（🔥 新增）
		needSummary := lastSumTime == 0 || hoursSinceLastSum >= int64(s.config.MinTimeSinceLastSummary)
		messageTrigger := len(messages) >= s.config.MaxMessageCount
		urgentSummary := isAboutToExpire || isRecentlyExpired // 🔥 紧急汇总

		if needSummary || messageTrigger || urgentSummary {
			// 生成摘要
			summary := s.GenerateEnhancedSummary(messages)
			if summary == "" {
				continue
			}

			// 确定触发类型
			var triggerType string
			var triggerReasons []string

			if needSummary {
				triggerReasons = append(triggerReasons, "time")
			}
			if messageTrigger {
				triggerReasons = append(triggerReasons, "message_count")
			}
			if urgentSummary {
				if isAboutToExpire {
					triggerReasons = append(triggerReasons, "about_to_expire")
				}
				if isRecentlyExpired {
					triggerReasons = append(triggerReasons, "recently_expired")
				}
			}

			triggerType = strings.Join(triggerReasons, "+")

			// 存储到长期记忆
			req := models.StoreContextRequest{
				SessionID: session.ID,
				Content:   summary,
				Priority:  "P1", // 汇总内容优先级高
				Metadata: map[string]interface{}{
					"type":           "auto_summary",
					"timestamp":      currentTime,
					"message_count":  len(messages),
					"trigger_type":   triggerType,
					"cursor_start":   lastSummaryCursor,
					"cursor_end":     s.getLastMessageTimestamp(messages),
					"session_status": session.Status,
				},
			}

			memoryID, err := s.StoreContext(ctx, req)
			if err != nil {
				log.Printf("[上下文服务] 警告: 自动汇总存储失败: %v", err)
				continue
			}

			// 🔥 更新会话元数据，记录汇总游标和时间
			if session.Metadata == nil {
				session.Metadata = make(map[string]interface{})
			}
			session.Metadata["last_summary_time"] = currentTime
			session.Metadata["last_summary_id"] = memoryID
			session.Metadata["last_summary_cursor"] = s.getLastMessageTimestamp(messages) // 🔥 记录游标

			// 保存更新后的会话
			if err := s.sessionStore.SaveSession(session); err != nil {
				log.Printf("[上下文服务] 警告: 更新会话元数据失败: %v", err)
			}

			log.Printf("[上下文服务] 会话 %s 自动汇总完成, 消息数: %d, 距上次汇总: %d小时, 触发类型: %s, 生成长期记忆 ID: %s",
				session.ID, len(messages), hoursSinceLastSum, triggerType, memoryID)

			if isRecentlyExpired {
				expiredProcessedCount++
			}
			summarizedCount++
		} else {
			skippedCount++
		}
	}

	log.Printf("[上下文服务] 自动汇总完成: 总共汇总 %d 个会话, 跳过 %d 个会话, 处理过期会话 %d 个",
		summarizedCount, skippedCount, expiredProcessedCount)
}

// 🔥 新增：获取游标之后的消息
func (s *ContextService) getMessagesAfterCursor(sessionID string, cursor int64) ([]*models.Message, error) {
	// 获取所有消息
	allMessages, err := s.sessionStore.GetMessages(sessionID, 0) // 0表示获取所有消息
	if err != nil {
		return nil, err
	}

	// 过滤出游标之后的消息
	var newMessages []*models.Message
	for _, msg := range allMessages {
		if msg.Timestamp > cursor {
			newMessages = append(newMessages, msg)
		}
	}

	return newMessages, nil
}

// 🔥 新增：获取最后一条消息的时间戳作为游标
func (s *ContextService) getLastMessageTimestamp(messages []*models.Message) int64 {
	if len(messages) == 0 {
		return time.Now().Unix()
	}

	maxTimestamp := int64(0)
	for _, msg := range messages {
		if msg.Timestamp > maxTimestamp {
			maxTimestamp = msg.Timestamp
		}
	}

	return maxTimestamp
}

// GenerateEnhancedSummary 生成增强的消息摘要
func (s *ContextService) GenerateEnhancedSummary(messages []*models.Message) string {
	if len(messages) == 0 {
		return ""
	}

	// 筛选重要消息
	var importantMessages []*models.Message
	for _, msg := range messages {
		// 优先选择用户问题和关键决策
		if msg.Role == models.RoleUser || msg.Priority == "P0" || msg.Priority == "P1" {
			importantMessages = append(importantMessages, msg)
		}
	}

	if len(importantMessages) == 0 {
		importantMessages = messages // 如果没有筛选出重要消息，使用全部消息
	}

	// 分析消息主题
	var topics []string
	var decisions []string
	var questions []string

	for _, msg := range importantMessages {
		content := strings.ToLower(msg.Content)

		// 简单的关键词检测，实际实现可以更复杂
		if strings.Contains(content, "决定") || strings.Contains(content, "决策") ||
			strings.Contains(content, "确定") || strings.Contains(content, "选择") {
			decisions = append(decisions, msg.Content)
		}

		if strings.HasSuffix(content, "?") || strings.HasSuffix(content, "？") ||
			strings.Contains(content, "如何") || strings.Contains(content, "为什么") {
			questions = append(questions, msg.Content)
		}

		// 提取可能的主题关键词 (简化实现)
		words := strings.Fields(content)
		for _, word := range words {
			if len(word) >= 4 && !strings.Contains("的了是在和与或但如果因为所以可能这那", word) {
				topics = append(topics, word)
				break // 每条消息只提取一个主题词
			}
		}
	}

	// 构建摘要
	var summary strings.Builder

	// 添加时间范围
	startTime := time.Unix(messages[0].Timestamp, 0).Format("2006-01-02 15:04:05")
	endTime := time.Unix(messages[len(messages)-1].Timestamp, 0).Format("2006-01-02 15:04:05")
	summary.WriteString(fmt.Sprintf("对话时间范围: %s 至 %s\n\n", startTime, endTime))

	// 添加主题
	if len(topics) > 0 {
		summary.WriteString("讨论主题: ")
		limit := 5
		if len(topics) < limit {
			limit = len(topics)
		}
		for i, topic := range topics[:limit] {
			if i > 0 {
				summary.WriteString(", ")
			}
			summary.WriteString(topic)
		}
		summary.WriteString("\n\n")
	}

	// 添加关键决策
	if len(decisions) > 0 {
		summary.WriteString("关键决策:\n")
		limit := 3
		if len(decisions) < limit {
			limit = len(decisions)
		}
		for i, decision := range decisions[:limit] {
			summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, decision))
		}
		summary.WriteString("\n")
	}

	// 添加重要问题
	if len(questions) > 0 {
		summary.WriteString("重要问题:\n")
		limit := 3
		if len(questions) < limit {
			limit = len(questions)
		}
		for i, question := range questions[:limit] {
			summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, question))
		}
		summary.WriteString("\n")
	}

	// 内容概要
	summary.WriteString("内容概要: ")
	var contentSummary string

	// 连接首条和末条消息，加上中间重要消息
	if len(messages) >= 2 {
		contentSummary = messages[0].Content + " ... " + messages[len(messages)-1].Content

		// 如果有重要消息，加上一条
		for _, msg := range messages {
			if msg.Priority == "P0" || msg.Priority == "P1" {
				contentSummary += " ... " + msg.Content
				break
			}
		}
	} else if len(messages) == 1 {
		contentSummary = messages[0].Content
	}

	// 截断过长内容
	if len(contentSummary) > 500 {
		contentSummary = contentSummary[:500] + "..."
	}

	summary.WriteString(contentSummary)

	return summary.String()
}

// SearchContext 根据会话ID和查询搜索上下文
func (s *ContextService) SearchContext(ctx context.Context, sessionID, query string) ([]string, error) {
	// 获取会话
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 简单实现：返回匹配查询的内容（实际实现应该调用向量搜索）
	var results []string

	// 1. 检查会话中的关联代码文件
	if session.CodeContext != nil {
		for path, _ := range session.CodeContext {
			if strings.Contains(path, query) {
				results = append(results, fmt.Sprintf("发现相关文件: %s", path))
			}
		}
	}

	// 2. 检查会话中的编辑历史
	if session.EditHistory != nil {
		for _, edit := range session.EditHistory {
			if strings.Contains(edit.Content, query) {
				results = append(results, fmt.Sprintf("文件 %s 的编辑记录包含相关内容", edit.FilePath))
			}
		}
	}

	// 3. 检查会话中的消息
	if session.Messages != nil {
		for _, msg := range session.Messages {
			if strings.Contains(msg.Content, query) {
				results = append(results, fmt.Sprintf("消息 [%s] 包含相关内容", msg.Role))
			}
		}
	}

	// 如果没有找到结果，返回一个友好的消息
	if len(results) == 0 {
		results = append(results, fmt.Sprintf("未找到与 '%s' 相关的内容", query))
	}

	return results, nil
}

func (s *ContextService) AssociateFile(ctx context.Context, req models.AssociateFileRequest) error {
	// TODO: 实现关联文件逻辑
	return nil
}

func (s *ContextService) RecordEdit(ctx context.Context, req models.RecordEditRequest) error {
	// TODO: 实现记录编辑逻辑
	return nil
}

// SummarizeToLongTermMemory 根据用户指令汇总当前会话内容到长期记忆
func (s *ContextService) SummarizeToLongTermMemory(ctx context.Context, req models.SummarizeToLongTermRequest) (string, error) {
	log.Printf("[上下文服务] 接收用户触发汇总请求: 会话ID=%s, 自定义描述=%s",
		req.SessionID, req.CustomDescription)

	// 获取会话消息
	messages, err := s.sessionStore.GetMessages(req.SessionID, 100) // 最多获取100条记录
	if err != nil {
		return "", fmt.Errorf("获取会话消息失败: %w", err)
	}

	if len(messages) == 0 {
		return "", fmt.Errorf("没有找到可汇总的消息")
	}

	// 生成摘要
	summary := s.GenerateEnhancedSummary(messages)

	// 如果用户提供了自定义描述，将其添加到摘要顶部
	if req.CustomDescription != "" {
		summary = fmt.Sprintf("用户重要标记: %s\n\n%s", req.CustomDescription, summary)
	}

	// 准备元数据
	metadata := map[string]interface{}{
		"type":          "user_triggered_summary",
		"timestamp":     time.Now().Unix(),
		"message_count": len(messages),
	}

	// 如果提供了标签，添加到元数据
	if req.Tags != nil && len(req.Tags) > 0 {
		metadata["tags"] = req.Tags
	}

	// 存储到长期记忆
	storeReq := models.StoreContextRequest{
		SessionID: req.SessionID,
		Content:   summary,
		Priority:  "P0", // 用户指定的内容优先级最高
		Metadata:  metadata,
	}

	// 存储到向量数据库
	memoryID, err := s.StoreContext(ctx, storeReq)
	if err != nil {
		return "", fmt.Errorf("存储长期记忆失败: %w", err)
	}

	log.Printf("[上下文服务] 用户触发汇总完成，生成长期记忆ID: %s", memoryID)

	return memoryID, nil
}

// RetrieveTodos 获取待办事项列表
func (s *ContextService) RetrieveTodos(ctx context.Context, request models.RetrieveTodosRequest) (*models.RetrieveTodosResponse, error) {
	log.Printf("开始检索待办事项: sessionID=%s, userID=%s, status=%s",
		request.SessionID, request.UserID, request.Status)

	limit := request.Limit
	if limit <= 0 {
		limit = 20 // 默认查询20条
	}

	// 构建直接查询bizType字段的条件，而不是从metadata中查询
	filter := fmt.Sprintf(`bizType=%d`, models.BizTypeTodo)

	// 如果有用户ID，添加到查询条件
	if request.UserID != "" {
		filter += fmt.Sprintf(" AND userId=\"%s\"", request.UserID)
	}

	// 查询所有待办事项
	log.Printf("执行待办事项查询: filter=%s, limit=%d", filter, limit)
	results, err := s.vectorService.SearchByFilter(filter, limit)
	if err != nil {
		log.Printf("查询待办事项失败: %v", err)
		return nil, fmt.Errorf("查询待办事项失败: %v", err)
	}

	log.Printf("成功检索到 %d 个待办事项", len(results))

	// 处理结果
	var todoItems []*models.TodoItem
	for _, result := range results {
		// 提取待办事项字段
		todoItem, err := extractTodoItem(result)
		if err != nil {
			log.Printf("警告: 跳过无效的待办事项记录: %v", err)
			continue
		}

		// 根据状态过滤
		if request.Status != "all" && todoItem.Status != request.Status {
			continue
		}

		todoItems = append(todoItems, todoItem)
	}

	// 创建响应
	response := &models.RetrieveTodosResponse{
		Items:  todoItems,
		Total:  len(todoItems),
		Status: "success",
	}

	// 如果有用户ID，添加到响应中
	if request.UserID != "" {
		response.UserID = request.UserID
	}

	log.Printf("完成待办事项查询，返回 %d 个结果", len(todoItems))

	return response, nil
}

// extractTodoItem 从搜索结果中提取待办事项
func extractTodoItem(result models.SearchResult) (*models.TodoItem, error) {
	// 记录详细的日志，帮助调试
	fieldsJSON, _ := json.Marshal(result.Fields)
	log.Printf("提取待办事项字段: %s", string(fieldsJSON))

	// 从Fields中提取内容
	content, ok := result.Fields["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("缺少内容字段")
	}

	// 创建待办事项
	todoItem := &models.TodoItem{
		ID:      result.ID,
		Content: content,
		Status:  "pending", // 默认状态
	}

	// 直接从结果字段中获取userId，不再从metadata中获取
	if userId, ok := result.Fields["userId"].(string); ok {
		todoItem.UserID = userId
	}

	// 获取元数据
	var metadata map[string]interface{}
	if metadataRaw, ok := result.Fields["metadata"]; ok {
		metadata, _ = metadataRaw.(map[string]interface{})
	}

	// 从metadata中提取其他信息
	if metadata != nil {
		// 优先级
		if priority, ok := metadata["priority"].(string); ok {
			todoItem.Priority = priority
		} else {
			todoItem.Priority = "P2" // 默认优先级
		}

		// 状态
		if status, ok := metadata["status"].(string); ok {
			todoItem.Status = status
		}

		// 创建时间
		if createdAt, ok := metadata["timestamp"].(float64); ok {
			todoItem.CreatedAt = int64(createdAt)
		} else {
			todoItem.CreatedAt = time.Now().Unix() // 默认为当前时间
		}

		// 完成时间
		if completedAt, ok := metadata["completedAt"].(float64); ok {
			todoItem.CompletedAt = int64(completedAt)
		}

		// 保存原始元数据
		todoItem.Metadata = metadata
	}

	return todoItem, nil
}

// GetProgrammingContext 获取编程上下文
func (s *ContextService) GetProgrammingContext(ctx context.Context, sessionID string, query string) (*models.ProgrammingContext, error) {
	log.Printf("[上下文服务] 获取编程上下文: 会话ID=%s, 查询=%s", sessionID, query)

	// 创建响应
	result := &models.ProgrammingContext{
		SessionID: sessionID,
	}

	// 获取会话
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 1. 获取关联文件信息
	if session.CodeContext != nil {
		for path, file := range session.CodeContext {
			// 转换为CodeFileInfo
			fileInfo := models.CodeFileInfo{
				Path:     file.Path,
				Language: file.Language,
				LastEdit: file.LastEdit,
				Summary:  file.Summary,
			}

			// 获取文件关联的讨论
			discussions, err := s.sessionStore.GetCodeFileRelations(sessionID, path)
			if err == nil && len(discussions) > 0 {
				fileInfo.RelatedDiscussions = discussions
				fileInfo.Importance = float64(len(discussions)) * 0.2
				if fileInfo.Importance > 1.0 {
					fileInfo.Importance = 1.0
				}
			}

			result.AssociatedFiles = append(result.AssociatedFiles, fileInfo)
		}
	}

	// 按最后编辑时间排序文件
	sort.Slice(result.AssociatedFiles, func(i, j int) bool {
		return result.AssociatedFiles[i].LastEdit > result.AssociatedFiles[j].LastEdit
	})

	// 2. 获取编辑历史
	if session.EditHistory != nil {
		for _, edit := range session.EditHistory {
			editInfo := models.EditInfo{
				ID:        edit.ID,
				Timestamp: edit.Timestamp,
				FilePath:  edit.FilePath,
				Type:      edit.Type,
				Position:  edit.Position,
				Content:   edit.Content,
			}

			// 添加标签
			if edit.Metadata != nil {
				if tags, ok := edit.Metadata["tags"].([]string); ok {
					editInfo.Tags = tags
				}
			}

			// 关联决策
			if edit.DecisionIDs != nil {
				editInfo.RelatedDecisions = edit.DecisionIDs
			}

			result.RecentEdits = append(result.RecentEdits, editInfo)
		}

		// 限制返回的编辑历史数量，只显示最近的20条
		if len(result.RecentEdits) > 20 {
			result.RecentEdits = result.RecentEdits[len(result.RecentEdits)-20:]
		}
	}

	// 3. 构建统计信息
	stats := models.ProgrammingStatistics{
		TotalFiles: len(result.AssociatedFiles),
		TotalEdits: len(result.RecentEdits),
	}

	// 语言使用统计
	languageUsage := make(map[string]int)
	for _, file := range result.AssociatedFiles {
		if file.Language != "" {
			languageUsage[file.Language]++
		}
	}
	stats.LanguageUsage = languageUsage

	// 按文件统计编辑数
	editsByFile := make(map[string]int)
	for _, edit := range result.RecentEdits {
		editsByFile[edit.FilePath]++
	}
	stats.EditsByFile = editsByFile

	// 按日期统计活动数
	activityByDay := make(map[string]int)
	for _, edit := range result.RecentEdits {
		day := time.Unix(edit.Timestamp, 0).Format("2006-01-02")
		activityByDay[day]++
	}
	stats.ActivityByDay = activityByDay

	// 4. 如果有特定查询，尝试查找相关代码片段
	if query != "" {
		// 生成查询向量
		queryVector, err := s.generateEmbedding(query)
		if err != nil {
			log.Printf("[上下文服务] 警告: 生成查询向量失败: %v", err)
		} else {
			// 搜索选项
			options := make(map[string]interface{})
			options["skip_threshold_filter"] = true

			// 设置过滤器
			options["filter"] = `metadata.type="code_file"`

			// 执行向量搜索
			searchResults, err := s.searchByVector(ctx, queryVector, "", options)
			if err == nil && len(searchResults) > 0 {
				for _, searchResult := range searchResults {
					if searchResult.Score > 0.8 { // 过滤相关性很低的结果
						continue
					}

					// 解析代码内容
					content, ok := searchResult.Fields["content"].(string)
					if !ok || content == "" {
						continue
					}

					// 获取文件路径
					filePath := ""
					if metadataStr, ok := searchResult.Fields["metadata"].(string); ok {
						var metadata map[string]interface{}
						if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
							if path, ok := metadata["file_path"].(string); ok {
								filePath = path
							}
						}
					}

					// 创建代码片段
					snippet := models.CodeSnippet{
						Content:  content,
						FilePath: filePath,
						Score:    searchResult.Score,
						Context:  fmt.Sprintf("相关度:%.2f", 1.0-searchResult.Score),
					}

					result.RelevantSnippets = append(result.RelevantSnippets, snippet)
				}
			}
		}
	}

	// 5. 查找并关联设计决策（如果有）
	// 将Metadata中的决策提取出来
	if session.Metadata != nil {
		if decisions, ok := session.Metadata["design_decisions"].([]interface{}); ok {
			for _, decisionData := range decisions {
				if decisionMap, ok := decisionData.(map[string]interface{}); ok {
					decision := models.DecisionSummary{
						ID:        getStringFromMap(decisionMap, "id", ""),
						Title:     getStringFromMap(decisionMap, "title", ""),
						Timestamp: getInt64FromMap(decisionMap, "timestamp", 0),
						Category:  getStringFromMap(decisionMap, "category", ""),
					}

					// 提取描述
					if desc, ok := decisionMap["description"].(string); ok {
						decision.Description = desc
					}

					// 提取相关编辑ID
					if edits, ok := decisionMap["related_edits"].([]interface{}); ok {
						for _, edit := range edits {
							if editID, ok := edit.(string); ok {
								decision.RelatedEdits = append(decision.RelatedEdits, editID)
							}
						}
					}

					result.DesignDecisions = append(result.DesignDecisions, decision)
				}
			}
		}
	}

	// 6. 查找关联会话
	if session.Metadata != nil {
		if linkedSessions, ok := session.Metadata["linked_sessions"].([]interface{}); ok {
			for _, linkData := range linkedSessions {
				if linkMap, ok := linkData.(map[string]interface{}); ok {
					link := models.SessionReference{
						SessionID:    getStringFromMap(linkMap, "session_id", ""),
						Relationship: getStringFromMap(linkMap, "relationship", ""),
						Description:  getStringFromMap(linkMap, "description", ""),
						Timestamp:    getInt64FromMap(linkMap, "timestamp", 0),
					}

					// 提取主题
					if topics, ok := linkMap["topics"].([]interface{}); ok {
						for _, topic := range topics {
							if t, ok := topic.(string); ok {
								link.Topics = append(link.Topics, t)
							}
						}
					}

					result.LinkedSessions = append(result.LinkedSessions, link)
				}
			}
		}
	}

	// 设置统计信息
	result.Statistics = stats

	log.Printf("[上下文服务] 成功获取编程上下文: 文件数=%d, 编辑数=%d, 决策数=%d",
		len(result.AssociatedFiles), len(result.RecentEdits), len(result.DesignDecisions))

	return result, nil
}

// getStringFromMap 从map中获取字符串值，如果不存在则返回默认值
func getStringFromMap(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// getInt64FromMap 从map中获取int64值，如果不存在则返回默认值
func getInt64FromMap(m map[string]interface{}, key string, defaultValue int64) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return defaultValue
}

// searchByVector 统一的向量搜索接口
func (s *ContextService) searchByVector(ctx context.Context, queryVector []float32, sessionID string, options map[string]interface{}) ([]models.SearchResult, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口向量搜索")

		// 转换选项格式
		searchOptions := &models.SearchOptions{
			Limit:         10,
			SessionID:     sessionID,
			SkipThreshold: false,
			// IsBruteSearch: 不在此处设置，根据传入参数决定
		}

		if options != nil {
			if skipThreshold, ok := options["skip_threshold_filter"].(bool); ok {
				searchOptions.SkipThreshold = skipThreshold
			}
			if userFilter, ok := options["filter"].(string); ok && strings.Contains(userFilter, "userId=") {
				log.Printf("[上下文服务] 🔍 检测到用户过滤器: %s", userFilter)
				// 从过滤器中提取用户ID
				re := regexp.MustCompile(`userId="([^"]+)"`)
				if matches := re.FindStringSubmatch(userFilter); len(matches) > 1 {
					searchOptions.UserID = matches[1]
					log.Printf("[上下文服务] ✅ 成功提取用户ID: %s", searchOptions.UserID)
				} else {
					log.Printf("[上下文服务] ⚠️  无法从过滤器中提取用户ID: %s", userFilter)
				}
			} else {
				log.Printf("[上下文服务] ⚠️  未检测到用户过滤器，options: %+v", options)
			}
			// 处理暴力搜索参数（仅对 Vearch 有效）
			if bruteSearch, ok := options["is_brute_search"].(int); ok {
				// 只有 Vearch 类型的向量存储才支持暴力搜索
				if s.vectorStore.GetProvider() == models.VectorStoreTypeVearch {
					searchOptions.IsBruteSearch = bruteSearch
					log.Printf("[上下文服务] 检测到 Vearch 存储，启用暴力搜索参数: %d", bruteSearch)
				} else {
					log.Printf("[上下文服务] 检测到 %s 存储，忽略暴力搜索参数", s.vectorStore.GetProvider())
				}
			}
		}

		// 🔥 详细日志：打印最终搜索选项
		log.Printf("[上下文服务] 🚀 调用向量存储搜索: UserID=%s, SessionID=%s, Limit=%d, IsBruteSearch=%d",
			searchOptions.UserID, searchOptions.SessionID, searchOptions.Limit, searchOptions.IsBruteSearch)

		// 使用新接口的向量搜索
		return s.vectorStore.SearchByVector(ctx, queryVector, searchOptions)
	}

	// 传统接口向量搜索
	log.Printf("[上下文服务] 使用传统向量服务向量搜索")

	// 执行搜索
	limit := 10
	if limitVal, ok := options["limit"].(int); ok && limitVal > 0 {
		limit = limitVal
	}

	return s.vectorService.SearchVectorsAdvanced(queryVector, sessionID, limit, options)
}

// GetUserIDFromSessionID 从会话ID获取用户ID - 简化版本
// 直接使用ContextService的SessionStore获取session，然后从metadata中获取userId
func (s *ContextService) GetUserIDFromSessionID(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("会话ID不能为空")
	}

	// 直接使用ContextService的SessionStore获取会话
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("获取会话失败: %w", err)
	}

	if session == nil {
		return "", fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 从metadata中获取userId
	if session.Metadata != nil {
		if userID, ok := session.Metadata["userId"].(string); ok && userID != "" {
			log.Printf("[会话用户ID获取] 成功从会话%s获取用户ID: %s", sessionID, userID)
			return userID, nil
		}
	}

	return "", fmt.Errorf("会话%s中未找到用户ID", sessionID)
}
