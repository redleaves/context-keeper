package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/knowledge"
	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/timeline"
	"github.com/contextkeeper/service/internal/llm"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/aliyun"
	"github.com/google/uuid"
)

// ContextService 提供上下文管理功能
type ContextService struct {
	vectorService      *aliyun.VectorService
	vectorStore        models.VectorStore // 新增：抽象向量存储接口
	sessionStore       *store.SessionStore
	userSessionManager *store.UserSessionManager
	config             *config.Config
	llmDrivenConfig    *config.LLMDrivenConfigManager // 🆕 LLM驱动配置管理器

	// 🔥 新增：TimescaleDB时间线存储引擎
	timelineEngine *timeline.TimescaleDBEngine

	// 🔧 临时解决方案：存储最后一次分析结果
	lastAnalysisResult  *models.SmartAnalysisResult
	analysisResultMutex sync.RWMutex
}

// NewContextService 创建新的上下文服务
func NewContextService(vectorSvc *aliyun.VectorService, sessionStore *store.SessionStore, cfg *config.Config) *ContextService {
	// 使用同样的存储路径为UserSessionManager创建基础路径
	// 修复：直接使用sessionStore的完整路径作为基础路径，确保用户隔离存储在正确的目录下
	baseStorePath := sessionStore.GetStorePath()
	userSessionManager := store.NewUserSessionManager(baseStorePath)

	// 🆕 加载LLM驱动配置
	llmDrivenConfigPath := "config/llm_driven.yaml"
	llmDrivenConfig := config.NewLLMDrivenConfigManager(llmDrivenConfigPath)
	if _, err := llmDrivenConfig.LoadConfig(); err != nil {
		log.Printf("⚠️ [配置加载] LLM驱动配置加载失败，使用默认配置: %v", err)
	} else {
		log.Printf("✅ [配置加载] LLM驱动配置加载成功")
	}

	return &ContextService{
		vectorService:      vectorSvc,
		vectorStore:        nil, // 初始为nil，表示使用传统vectorService
		sessionStore:       sessionStore,
		userSessionManager: userSessionManager,
		config:             cfg,
		llmDrivenConfig:    llmDrivenConfig, // 🆕 LLM驱动配置
	}
}

// SetVectorStore 设置新的向量存储接口
// 这允许ContextService动态切换到新的向量存储实现
func (s *ContextService) SetVectorStore(vectorStore models.VectorStore) {
	log.Printf("[上下文服务] 切换到新的向量存储接口")
	s.vectorStore = vectorStore
	log.Printf("[上下文服务] 向量存储接口切换完成，现在使用抽象接口")
}

// GetVectorStore 获取向量存储接口
func (s *ContextService) GetVectorStore() models.VectorStore {
	return s.vectorStore
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

	if s.vectorService != nil {
		log.Printf("[上下文服务] 使用传统向量服务生成向量")
		// 传统接口也返回[]float32
		return s.vectorService.GenerateEmbedding(content)
	}

	log.Printf("⚠️ [上下文服务] 向量服务未配置，跳过向量生成")
	return nil, fmt.Errorf("向量服务未配置")
}

// storeMemory 统一的记忆存储接口
// 自动选择使用新接口或传统接口存储记忆
func (s *ContextService) storeMemory(memory *models.Memory) error {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口存储记忆")
		return s.vectorStore.StoreMemory(memory)
	}

	if s.vectorService != nil {
		log.Printf("[上下文服务] 使用传统向量服务存储记忆")
		return s.vectorService.StoreVectors(memory)
	}

	log.Printf("⚠️ [上下文服务] 向量服务未配置，跳过向量存储")
	return fmt.Errorf("向量服务未配置")
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
	if s.vectorService != nil {
		log.Printf("[上下文服务] 使用传统向量服务按会话ID搜索")
		return s.vectorService.SearchBySessionID(sessionID, limit)
	}

	log.Printf("⚠️ [上下文服务] 向量服务未配置，返回空结果")
	return []models.SearchResult{}, nil
}

// countMemories 统一的记忆计数接口
func (s *ContextService) countMemories(sessionID string) (int, error) {
	if s.vectorStore != nil {
		log.Printf("[上下文服务] 使用新向量存储接口计数记忆")
		return s.vectorStore.CountMemories(sessionID)
	}

	if s.vectorService != nil {
		log.Printf("[上下文服务] 使用传统向量服务计数记忆")
		return s.vectorService.CountSessionMemories(sessionID)
	}

	log.Printf("⚠️ [上下文服务] 向量服务未配置，返回0")
	return 0, nil
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

// StoreContext 存储上下文内容（向后兼容版本）
func (s *ContextService) StoreContext(ctx context.Context, req models.StoreContextRequest) (string, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 接收存储请求: 会话ID=%s, 内容长度=%d字节",
		req.SessionID, len(req.Content))

	// 🔥 开关控制：互斥的两套逻辑
	if s.config.EnableMultiDimensionalStorage {
		log.Printf("🚀 启用LLM驱动的多维度存储逻辑")
		return s.executeLLMDrivenStorage(ctx, req)
	} else {
		log.Printf("📋 使用原有的向量存储逻辑")
		return s.executeOriginalStorage(ctx, req)
	}
}

// StoreContextWithAnalysis 存储上下文内容并返回完整分析结果（扩展版本）
func (s *ContextService) StoreContextWithAnalysis(ctx context.Context, req models.StoreContextRequest) (*models.StoreContextResponse, error) {
	// 记录请求信息
	log.Printf("[上下文服务] 接收存储请求（扩展版本）: 会话ID=%s, 内容长度=%d字节",
		req.SessionID, len(req.Content))

	// 🔥 开关控制：互斥的两套逻辑
	if s.llmDrivenConfig.GetConfig().Enabled {
		log.Printf("🧠 [上下文服务] 使用LLM驱动的多维度存储逻辑（扩展版本）")

		// 执行LLM驱动存储并获取分析结果
		memoryID, err := s.executeLLMDrivenStorage(ctx, req)
		if err != nil {
			return nil, err
		}

		// 获取最后一次分析结果
		analysisResult := s.GetLastAnalysisResult()

		response := &models.StoreContextResponse{
			MemoryID: memoryID,
			Status:   "success",
		}

		if analysisResult != nil {
			response.AnalysisResult = analysisResult
			response.Confidence = analysisResult.ConfidenceAssessment.OverallConfidence

			// 根据置信度确定存储策略
			contextOnlyThreshold := s.llmDrivenConfig.GetContextOnlyThreshold()
			if response.Confidence < contextOnlyThreshold {
				response.StorageStrategy = "context_only"
			} else if response.Confidence < 0.8 {
				response.StorageStrategy = "selective_storage"
			} else {
				response.StorageStrategy = "full_storage"
			}
		}

		return response, nil
	} else {
		log.Printf("📦 [上下文服务] 使用原有的向量存储逻辑（扩展版本）")
		memoryID, err := s.executeOriginalStorage(ctx, req)
		if err != nil {
			return nil, err
		}
		return &models.StoreContextResponse{
			MemoryID: memoryID,
			Status:   "success",
		}, nil
	}
}

// executeOriginalStorage 执行原有的向量存储逻辑
func (s *ContextService) executeOriginalStorage(ctx context.Context, req models.StoreContextRequest) (string, error) {
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

// executeLLMDrivenStorage 执行LLM驱动的多维度存储逻辑
func (s *ContextService) executeLLMDrivenStorage(ctx context.Context, req models.StoreContextRequest) (string, error) {
	log.Printf("🔥 [LLM驱动存储] 开始执行多维度存储流程")

	// 1. 直接获取已有的上下文（由查询链路维护）
	contextData, err := s.getExistingContextData(ctx, req.SessionID)
	if err != nil {
		log.Printf("⚠️ [LLM驱动存储] 获取上下文失败: %v", err)
		// 如果没有上下文，使用基础信息
		contextData = s.getBasicContextData(req.SessionID)
	}

	// 2. 结合上下文和原始内容进行智能LLM分析（一次调用）
	analysisResult, err := s.analyzeContentWithSmartLLM(contextData, req.Content)
	if err != nil {
		log.Printf("❌ [LLM驱动存储] 智能分析失败，降级到原有逻辑: %v", err)
		return s.executeOriginalStorage(ctx, req)
	}

	// 3. 执行智能存储策略
	return s.executeSmartStorage(ctx, analysisResult, req)
}

// getExistingContextData 获取已有的上下文数据（由查询链路维护）
func (s *ContextService) getExistingContextData(ctx context.Context, sessionID string) (*models.LLMDrivenContextModel, error) {
	log.Printf("🔍 [上下文获取] 尝试获取会话 %s 的上下文数据", sessionID)

	// 🔥 实现真实的上下文获取逻辑
	// 1. 从会话存储中获取会话信息
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		log.Printf("⚠️ [上下文获取] 获取会话 %s 失败: %v", sessionID, err)
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 2. 从会话历史中分析上下文模式
	contextModel, err := s.buildContextFromSession(session)
	if err != nil {
		log.Printf("❌ [上下文获取] 从会话构建上下文失败: %v", err)
		return nil, fmt.Errorf("构建上下文失败: %w", err)
	}

	log.Printf("✅ [上下文获取] 成功获取会话上下文，焦点: %s", contextModel.Core.CurrentFocus)
	return contextModel, nil
}

// buildContextFromSession 从会话信息构建上下文模型
func (s *ContextService) buildContextFromSession(session *models.Session) (*models.LLMDrivenContextModel, error) {
	log.Printf("🔧 [上下文构建] 开始从会话构建上下文模型，会话: %s", session.ID)

	// 分析会话历史，提取上下文信息（简化实现）
	currentFocus := s.extractCurrentFocus(session)
	intentCategory := s.extractIntentCategory(session)
	complexity := "medium" // 默认复杂度
	conversationThread := fmt.Sprintf("thread_%s", session.ID)

	// 构建核心上下文
	coreContext := &models.CoreContext{
		ConversationThread: conversationThread,
		CurrentFocus:       currentFocus,
		IntentCategory:     intentCategory,
		Complexity:         complexity,
	}

	// 构建维度上下文（使用引用模式）
	dimensions := &models.ContextDimensions{
		TechnicalRef:  fmt.Sprintf("tech_%s", session.ID),
		ProblemRef:    fmt.Sprintf("problem_%s", session.ID),
		ProjectRef:    fmt.Sprintf("project_%s", session.ID),
		UserRef:       fmt.Sprintf("user_%s", s.extractUserIDFromSession(session)),
		HistoricalRef: fmt.Sprintf("history_%s", session.ID),
	}

	// 构建完整的上下文模型
	contextModel := &models.LLMDrivenContextModel{
		SessionID:  session.ID,
		Core:       coreContext,
		Dimensions: dimensions,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	log.Printf("✅ [上下文构建] 上下文模型构建完成，焦点: %s, 意图: %s", currentFocus, string(intentCategory))
	return contextModel, nil
}

// extractCurrentFocus 从会话中提取当前焦点
func (s *ContextService) extractCurrentFocus(session *models.Session) string {
	// 简化实现：基于最近的消息或会话摘要
	if session.Summary != "" {
		return session.Summary
	}
	if len(session.Messages) > 0 {
		lastMessage := session.Messages[len(session.Messages)-1]
		if len(lastMessage.Content) > 100 {
			return lastMessage.Content[:100] + "..."
		}
		return lastMessage.Content
	}
	return fmt.Sprintf("会话 %s", session.ID)
}

// extractIntentCategory 从会话中提取意图类别
func (s *ContextService) extractIntentCategory(session *models.Session) models.IntentType {
	// 简化实现：基于会话内容分析
	if len(session.Messages) > 0 {
		// 这里可以添加更复杂的意图分析逻辑
		return models.IntentQuery // 默认为查询类型
	}
	return models.IntentQuery
}

// extractUserIDFromSession 从会话中提取用户ID
func (s *ContextService) extractUserIDFromSession(session *models.Session) string {
	if session.Metadata != nil {
		if userID, exists := session.Metadata["userId"]; exists {
			if userIDStr, ok := userID.(string); ok {
				return userIDStr
			}
		}
	}
	return "unknown_user"
}

// getBasicContextData 如果没有上下文，获取基础信息
func (s *ContextService) getBasicContextData(sessionID string) *models.LLMDrivenContextModel {
	log.Printf("📋 [上下文获取] 使用基础上下文数据，会话: %s", sessionID)

	return &models.LLMDrivenContextModel{
		SessionID: sessionID,
		// 基础的会话信息，不包含复杂的业务维度分析
		Core: &models.CoreContext{
			ConversationThread: "基础会话",
			CurrentFocus:       "未知",
			IntentCategory:     models.IntentCommand,
			Complexity:         "simple",
		},
	}
}

// analyzeContentWithSmartLLM 结合上下文和原始内容进行智能LLM分析（替换analyzeLLMContentWithContext）
func (s *ContextService) analyzeContentWithSmartLLM(contextData *models.LLMDrivenContextModel, content string) (*models.SmartAnalysisResult, error) {
	log.Printf("🧠 [LLM分析] 开始分析内容，会话: %s", contextData.SessionID)

	// 🔥 读取知识图谱抽取模式配置
	kgMode := s.getKnowledgeGraphExtractionMode()
	log.Printf("🕸️ [KG配置] 知识图谱抽取模式: %s", kgMode)

	// 根据配置选择执行方案
	switch kgMode {
	case "enhanced_prompt":
		return s.executeEnhancedPromptAnalysis(contextData, content)
	case "parallel_dedicated":
		return s.executeParallelAnalysis(contextData, content)
	default:
		return s.executeOriginalAnalysis(contextData, content)
	}
}

// getKnowledgeGraphExtractionMode 获取知识图谱抽取模式
func (s *ContextService) getKnowledgeGraphExtractionMode() string {
	mode := os.Getenv("KNOWLEDGE_GRAPH_EXTRACTION_MODE")
	if mode == "" {
		mode = "disabled" // 默认关闭
	}
	return mode
}

// executeOriginalAnalysis 执行原有的分析逻辑
func (s *ContextService) executeOriginalAnalysis(contextData *models.LLMDrivenContextModel, content string) (*models.SmartAnalysisResult, error) {
	funcStart := time.Now()
	log.Printf("🧠 [原有分析] 开始原有分析逻辑 - 函数开始: %s", funcStart.Format("15:04:05.000"))

	// 构建智能分析prompt
	promptStart := time.Now()
	prompt := s.buildSmartAnalysisPrompt(contextData, content)
	promptDuration := time.Since(promptStart)
	log.Printf("📝 [原有分析] 构建prompt完成: %s, 耗时: %v, 长度: %d", time.Now().Format("15:04:05.000"), promptDuration, len(prompt))

	// 🔥 参考查询链路的LLM调用模式，使用LLM工厂和标准接口
	llmProvider := s.config.MultiDimLLMProvider
	llmModel := s.config.MultiDimLLMModel
	if llmProvider == "" {
		return nil, fmt.Errorf("LLM提供商未配置")
	}

	// 创建LLM客户端（参考查询链路的实现）
	llmClient, err := s.createStandardLLMClient(llmProvider, llmModel)
	if err != nil {
		log.Printf("❌ [LLM分析] 创建LLM客户端失败: %v，降级到基础分析", err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	// 构建标准的LLM请求（参考查询链路的LLMRequest结构）
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   4000,
		Temperature: 0.1, // 低温度确保结果稳定
		Format:      "json",
		Model:       llmModel,
		Metadata: map[string]interface{}{
			"task":           "multi_dimensional_storage_analysis",
			"session_id":     contextData.SessionID,
			"content_length": len(content),
		},
	}

	// 调用LLM API（参考查询链路的调用方式）
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 修复：使用120秒超时
	defer cancel()

	// 🔥 打印LLM入参
	log.Printf("📤 [LLM分析] LLM请求入参:")
	log.Printf("     提供商: %s", llmProvider)
	log.Printf("     模型: %s", llmModel)
	log.Printf("     最大Token: %d", llmRequest.MaxTokens)
	log.Printf("     温度: %.1f", llmRequest.Temperature)
	log.Printf("     格式: %s", llmRequest.Format)
	log.Printf("     Prompt长度: %d 字符", len(llmRequest.Prompt))
	log.Printf("📝 [LLM分析] 完整Prompt内容:\n%s", llmRequest.Prompt)

	apiCallStart := time.Now()
	log.Printf("🚀 [原有分析] 开始调用LLM API: %s, 提供商: %s，模型: %s", apiCallStart.Format("15:04:05.000"), llmProvider, llmModel)
	log.Printf("🔍 [原有分析] 限流检查开始: %s", time.Now().Format("15:04:05.000"))

	llmResponse, err := llmClient.Complete(ctx, llmRequest)

	apiCallEnd := time.Now()
	apiCallDuration := apiCallEnd.Sub(apiCallStart)
	if err != nil {
		log.Printf("❌ [原有分析] LLM API调用失败: %s, 耗时: %v, 错误: %v", apiCallEnd.Format("15:04:05.000"), apiCallDuration, err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	// 🔥 打印LLM出参
	log.Printf("✅ [原有分析] LLM API调用完成: %s, 耗时: %v, Token使用: %d", apiCallEnd.Format("15:04:05.000"), apiCallDuration, llmResponse.TokensUsed)
	log.Printf("� [LLM分析] LLM响应详情:")
	log.Printf("     响应长度: %d 字符", len(llmResponse.Content))
	log.Printf("     Token使用: %d", llmResponse.TokensUsed)
	log.Printf("�📄 [LLM分析] LLM完整响应内容:\n%s", llmResponse.Content)

	// 解析LLM响应（使用新的智能分析解析）
	analysisResult, err := s.parseSmartAnalysisResponse(llmResponse.Content)
	if err != nil {
		log.Printf("❌ [智能分析] LLM响应解析失败: %v，降级到基础分析", err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	log.Printf("✅ [智能分析] 多维度分析完成，整体置信度: %.2f", analysisResult.ConfidenceAssessment.OverallConfidence)

	// 🔧 保存分析结果供LLM驱动服务使用
	s.setLastAnalysisResult(analysisResult)

	return analysisResult, nil
}

// executeEnhancedPromptAnalysis 执行方案一：增强prompt分析
func (s *ContextService) executeEnhancedPromptAnalysis(contextData *models.LLMDrivenContextModel, content string) (*models.SmartAnalysisResult, error) {
	log.Printf("🔥 [方案一] 执行增强prompt分析")

	// 构建增强的智能分析prompt（包含KG维度）
	prompt := s.buildEnhancedSmartAnalysisPrompt(contextData, content)
	log.Printf("📝 [增强分析] 构建的增强prompt长度: %d", len(prompt))

	// 🔥 使用现有的LLM调用逻辑
	llmProvider := s.config.MultiDimLLMProvider
	llmModel := s.config.MultiDimLLMModel
	if llmProvider == "" {
		return nil, fmt.Errorf("LLM提供商未配置")
	}

	// 创建LLM客户端
	llmClient, err := s.createStandardLLMClient(llmProvider, llmModel)
	if err != nil {
		log.Printf("❌ [增强分析] 创建LLM客户端失败: %v，降级到基础分析", err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	// 构建LLM请求
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   4000,
		Temperature: 0.1,
		Format:      "json",
		Model:       llmModel,
		Metadata: map[string]interface{}{
			"task":           "enhanced_knowledge_graph_analysis",
			"session_id":     contextData.SessionID,
			"content_length": len(content),
		},
	}

	// 调用LLM API
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	log.Printf("🚀 [增强分析] 调用LLM API，提供商: %s，模型: %s", llmProvider, llmModel)
	llmResponse, err := llmClient.Complete(ctx, llmRequest)
	if err != nil {
		log.Printf("❌ [增强分析] LLM API调用失败: %v，降级到基础分析", err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	log.Printf("✅ [增强分析] LLM调用完成，Token使用: %d", llmResponse.TokensUsed)
	log.Printf("📄 [增强分析] LLM响应长度: %d 字符", len(llmResponse.Content))

	// 解析增强的LLM响应（包含KG信息）
	analysisResult, err := s.parseEnhancedSmartAnalysisResponse(llmResponse.Content)
	if err != nil {
		log.Printf("❌ [增强分析] LLM响应解析失败: %v，降级到基础分析", err)
		return s.getBasicSmartAnalysisResult(content), nil
	}

	log.Printf("✅ [增强分析] 增强分析完成，整体置信度: %.2f", analysisResult.ConfidenceAssessment.OverallConfidence)
	if analysisResult.KnowledgeGraphExtraction != nil {
		log.Printf("🕸️ [增强分析] 知识图谱抽取完成，实体: %d个，关系: %d个",
			len(analysisResult.KnowledgeGraphExtraction.Entities),
			len(analysisResult.KnowledgeGraphExtraction.Relationships))
	}

	return analysisResult, nil
}

// buildSmartAnalysisPrompt 构建智能分析的prompt（替换buildStorageAnalysisPrompt）
func (s *ContextService) buildSmartAnalysisPrompt(contextData *models.LLMDrivenContextModel, content string) string {
	prompt := fmt.Sprintf(`你是一个专业的语义意图识别专家，专门负责从用户查询中进行意图拆分和语义关键词提取。

## 🎯 核心任务
1. **意图拆分**: 识别用户查询中的多个语义意图（可能包含多个步骤、动作或关注点）
2. **语义关键词提取**: 保留核心关键词，剔除干扰词、停用词，进行降噪处理
3. **置信度评估**: 客观评判语义是否清晰、信息是否充足、识别结果是否可靠

## 🧠 意图拆分原则
用户的query/command可能包含多个语义层次：
- **复合意图**: "先制定计划，再实现功能" → 拆分为"制定计划" + "功能实现"
- **层次意图**: "学习React Hook，重点关注useState" → 拆分为"React Hook学习" + "useState重点关注"
- **条件意图**: "如果性能有问题，就优化数据库查询" → 拆分为"性能问题诊断" + "数据库查询优化"

## 📊 四维度语义提取

### 1. Core Intent Vector (核心意图维度)
**目的**: 提取用户的核心意图关键词，支持多意图拆分
**处理原则**:
- 保留具体的技术词汇、功能名称、概念名称
- 剔除"我想"、"请帮我"、"了解一下"等干扰词
- 支持多个意图的并列表达

### 2. Domain Context Vector (领域上下文维度)
**目的**: 识别技术栈和业务领域的具体上下文
**处理原则**: 从具体到抽象，保留最具区分度的领域信息

### 3. Scenario Vector (场景维度)
**目的**: 识别具体的使用场景和问题背景
**处理原则**: 基于上下文推断最可能的使用场景

### 4. Completeness Vector (完整度维度)
**目的**: 评估信息完整度，识别缺失要素
**关键评估**: 语义是否清晰、信息是否充足、识别结果是否可靠

## 🎯 置信度评估标准（重要！）
请基于以下具体维度和指标进行客观评估：

1. **语义清晰度** (semantic_clarity):
   评估用户表达的明确程度，重点关注：
   - **用户痛点识别**: 能否明确识别用户遇到的具体问题或需求？
   - **场景上下文**: 能否判断用户所处的业务场景、工作环境或项目背景？
   - **诉求明确性**: 用户想要什么？期望得到什么帮助？

   评分标准：
   - 0.9+: 痛点明确、场景清晰、诉求具体（如"生产环境MySQL查询慢，需要优化方案"）
   - 0.7-0.9: 痛点相对明确、有基本场景信息（如"React项目中useState更新异步问题"）
   - 0.5-0.7: 痛点模糊但可推断、缺乏场景信息（如"代码有bug需要修复"）
   - 0.3-0.5: 痛点不明确、场景缺失（如"API有问题"、"系统出错了"）
   - <0.3: 无法识别痛点和场景（如"不行"、"有问题"、纯感叹词）

2. **信息完整度** (information_completeness):
   评估信息的充分程度：
   - **关键要素**: 是否包含时间、地点、对象、事件等关键要素？
   - **技术细节**: 对于技术问题，是否包含技术栈、环境、错误信息等？
   - **业务背景**: 对于业务问题，是否包含业务场景、流程、目标等？

   评分标准：
   - 0.9+: 包含完整的关键要素和背景信息
   - 0.7-0.9: 包含主要要素，少量细节缺失
   - 0.5-0.7: 包含基本要素，但缺乏重要背景
   - 0.3-0.5: 要素不完整，信息严重缺失
   - <0.3: 几乎无有效信息

3. **意图识别可信度** (intent_confidence):
   评估意图识别的准确性：
   - **意图明确性**: 用户的真实意图是否清晰？
   - **歧义程度**: 是否存在多种可能的解释？
   - **可操作性**: 基于当前信息是否能提供有效帮助？

   评分标准：
   - 0.9+: 意图非常明确，无歧义，可直接操作
   - 0.7-0.9: 意图相对明确，轻微歧义，基本可操作
   - 0.5-0.7: 意图模糊，存在歧义，需要澄清
   - 0.3-0.5: 意图不明确，多种解释，难以操作
   - <0.3: 无法识别有效意图

## 🚨 低质量内容识别标准
以下情况应给予极低置信度（overall_confidence < 0.4）：

**无效表达类**:
- 纯感叹词: "啊"、"哦"、"嗯"、"呃"
- 简单否定: "不行"、"不对"、"失败了"
- 模糊问题: "有问题"、"出错了"、"坏了"

**信息缺失类**:
- 仅有技术词汇无具体问题: "API"、"数据库"、"前端"
- 无上下文的求助: "帮忙"、"求助"、"怎么办"
- 过于简短无意义: 少于3个有效字符

## 🌟 高质量内容识别标准
以下情况应给予高置信度（overall_confidence > 0.7）：

**场景明确类**:
- 包含时间信息: "昨天的项目进度"、"上周完成的功能"
- 包含环境信息: "生产环境"、"测试环境"、"开发阶段"
- 包含业务背景: "电商系统"、"用户管理模块"、"支付流程"

**问题具体类**:
- 技术问题有细节: "MySQL查询响应时间从50ms增加到500ms"
- 功能需求明确: "需要实现用户登录的JWT认证"
- 错误信息完整: "React Hook useState更新后立即读取仍是旧值"

## ⏰ 时间线存储智能识别规则（重要！）

**🔥 应该存储到时间线的场景**：
1. **明确时间信息**: "昨天"、"上周"、"2024年8月"、"今天完成"等
2. **总结性内容**: "我们成功实现了..."、"项目已完成..."、"最终结论是..."
3. **里程碑事件**: "架构设计完成"、"功能上线"、"问题解决"、"重要决策"
4. **结论性表述**: "总结一下"、"综上所述"、"最终确定"、"得出结论"
5. **完成状态**: "已实现"、"已修复"、"已优化"、"已部署"

**时间标识规则**：
- **有明确时间**: 提取具体时间（如"2024-08-10"、"昨天"、"上周"）
- **无明确时间但是总结/结论/里程碑**: 使用"now"表示当前时间
- **普通讨论/询问**: 不存储到时间线

**示例判断**：
- ✅ "我们成功实现了LLM驱动的智能存储架构" → should_store: true, timeline_time: "now"
- ✅ "昨天完成了数据库优化" → should_store: true, timeline_time: "昨天"
- ✅ "项目第一阶段已完成，包括..." → should_store: true, timeline_time: "now"
- ❌ "如何实现用户登录功能？" → should_store: false
- ❌ "API调用出现错误" → should_store: false

## 已有上下文信息
**会话ID**: %s
**会话焦点**: %s
**意图类别**: %s
**复杂度**: %s

## 用户内容
%s

## 📋 输出格式

请严格按照以下JSON格式输出：

{
  "intent_analysis": {
    "core_intent_text": "核心意图关键词（支持多意图）",
    "domain_context_text": "具体技术栈和领域",
    "scenario_text": "具体使用场景",
    "intent_count": 1,
    "multi_intent_breakdown": ["意图1", "意图2"],
    "summary": "100-200字符的结构化摘要，突出关键信息和结果"
  },

  "confidence_assessment": {
    "semantic_clarity": <根据语义清晰度评估的0-1数值>,
    "information_completeness": <根据信息完整度评估的0-1数值>,
    "intent_confidence": <根据意图识别可信度评估的0-1数值>,
    "overall_confidence": <根据综合评估的0-1数值>,
    "missing_elements": ["缺失的关键要素"],// 例如：["技术栈", "环境信息"]
    "clarity_issues": ["识别出的清晰度问题"] //例如：["需求过于抽象", "缺少具体参数"]
  },

  "storage_recommendations": {
    "timeline_storage": {
      "should_store": <true/false，基于时间信息判断>,
      "reason": "<存储或不存储的具体原因>",
      "confidence_threshold": 0.7,
      "timeline_time": "<时间标识规则详见下方说明>",
      "event_type": "<根据内容特征判断的事件类型，详见下方说明>"
    },
    "knowledge_graph_storage": {
      "should_store": <true/false，基于是否包含技术概念和关系>,
      "reason": "<存储或不存储的具体原因>",
      "confidence_threshold": 0.6
    },
    "vector_storage": {
      "should_store": <true/false，基于意图清晰度>,
      "reason": "<存储或不存储的具体原因>",
      "confidence_threshold": 0.5,
      "enabled_dimensions": [<根据内容质量确定的维度列表>]
    }
  }
}

## 📝 summary字段生成规则（重要！）
请生成100-200字符的结构化摘要：

**生成原则**：
- **结构化表达**: 采用"通过X技术解决Y问题，达到Z效果"的格式
- **关键信息**: 技术栈、问题描述、解决方案、具体效果
- **量化优先**: 包含性能数据、时间节约、错误减少等具体数字
- **行动导向**: 突出已完成/正在做/计划做的具体行动

**生成示例**：
输入："团队讨论了Redis缓存策略，决定使用分布式缓存解决数据一致性问题，预计可以提升30%查询性能"
→ summary: "采用Redis分布式缓存策略解决数据一致性问题，预计提升查询性能30%，优化系统响应效率"

## 🔥 timeline_time字段规则
- **有明确时间**: 转换为标准格式（"昨天"→"2025-08-09", "上周"→"2025-08-03", 保持"2024-08-10"格式）
- **无明确时间但包含结论性内容**: 使用"now"（总结、已完成、成功实现、里程碑、决定等）
- **普通讨论/询问**: 不存储时间线
## 🏷️ event_type字段规则（重要！）
请根据内容特征判断最合适的事件类型：

**🔧 code_edit**: 包含具体代码修改、文件编辑、代码实现
- 关键词: "修改了"、"实现了"、"代码"、"文件"、"函数"、"实现"
- 示例: "修改了user.go文件的登录逻辑"

**💬 discussion**: 技术讨论、方案对比、团队交流
- 关键词: "讨论"、"交流"、"分析"、"对比"、"评估"
- 示例: "团队讨论了微服务架构的优缺点"

**🎨 design**: 架构设计、系统设计、方案设计
- 关键词: "设计"、"架构"、"方案"、"设计评审"、"确定采用"
- 示例: "完成了系统架构设计，采用微服务模式"

**🔧 problem_solve**: 问题解决、故障处理、bug修复
- 关键词: "解决"、"修复"、"故障"、"问题"、"bug"、"异常"
- 示例: "解决了数据库连接池耗尽的问题"

**📚 knowledge_share**: 知识分享、最佳实践、经验总结
- 关键词: "分享"、"最佳实践"、"经验"、"总结"、"技巧"
- 示例: "分享LLM系统设计的最佳实践"

**⚖️ decision**: 重要决策、技术选型、方案确定
- 关键词: "决定"、"选择"、"确定"、"采用"、"决策"
- 示例: "决定采用Redis作为缓存方案"

**📝 review**: 代码审查、方案评审、技术评估
- 关键词: "审查"、"评审"、"review"、"评估"、"检查"
- 示例: "完成了代码review，发现3个优化点"

**🧪 test**: 测试相关、验证、实验
- 关键词: "测试"、"验证"、"实验"、"test"、"验证"
- 示例: "完成了API接口的集成测试"

**🚀 deployment**: 部署、发布、上线
- 关键词: "部署"、"发布"、"上线"、"deploy"、"上线"
- 示例: "完成了生产环境的部署"

**📅 meeting**: 会议记录、团队会议、评审会议
- 关键词: "会议"、"meeting"、"评审会"、"讨论会"
- 示例: "参加了项目进度评审会议"

**🎯 intent_based**: 复杂业务场景、无法明确归类的内容
- 用途: 兜底分类，当无法明确归类到上述类型时使用
- 示例: 复杂的业务流程描述、多维度技术分析

现在请分析以上用户查询。`,
		contextData.SessionID,
		contextData.Core.CurrentFocus,
		string(contextData.Core.IntentCategory),
		contextData.Core.Complexity,
		content)

	return prompt
}

// buildEnhancedSmartAnalysisPrompt 构建增强的智能分析prompt（方案一：包含KG维度）
func (s *ContextService) buildEnhancedSmartAnalysisPrompt(contextData *models.LLMDrivenContextModel, content string) string {
	basePrompt := s.buildSmartAnalysisPrompt(contextData, content)

	// 🔥 在基础prompt后增加知识图谱抽取维度
	kgSupplement := `

## 🕸️ 知识图谱抽取补充（第5维度）

基于上述四维度分析，请额外提取关键实体和关系信息：

### 实体类型（6种）
- Technical: 技术、工具、框架、系统、编程语言、数据库
- Project: 项目、任务、功能、模块、工作
- Concept: 概念、模式、理念、方法、架构
- Issue: 问题、故障、优化、事件、错误
- Data: 数据、指标、参数、时间、版本、配置
- Process: 流程、操作、环境、部署、方法

### 关系类型（5种）
- USES: A使用B
- SOLVES: A解决B  
- BELONGS_TO: A属于B
- CAUSES: A导致B
- RELATED_TO: A相关B

### 输出要求
在JSON最后增加knowledge_extraction字段：

"knowledge_extraction": {
  "entities": ["实体名(类型)", "实体名(类型)", ...],
  "relations": ["源实体->关系->目标实体", "源实体->关系->目标实体", ...]
}

示例：
"knowledge_extraction": {
  "entities": ["系统(Technical)", "性能优化(Project)", "响应时间(Data)", "超时问题(Issue)"],
  "relations": ["性能优化->SOLVES->超时问题", "系统->USES->数据库"]
}`

	return basePrompt + kgSupplement
}

// executeParallelAnalysis 执行方案二：并行专门化分析
func (s *ContextService) executeParallelAnalysis(contextData *models.LLMDrivenContextModel, content string) (*models.SmartAnalysisResult, error) {
	startTime := time.Now()
	log.Printf("🔥 [方案二] 执行并行专门化分析 - 开始时间: %s", startTime.Format("15:04:05.000"))

	var wg sync.WaitGroup
	var analysisResult *models.SmartAnalysisResult
	var kgExtraction *models.KnowledgeGraphExtraction
	var analysisErr, kgErr error
	var originalDuration, kgDuration time.Duration

	wg.Add(2)

	// 原有分析（并行执行）
	go func() {
		defer wg.Done()
		originalStart := time.Now()
		log.Printf("🧠 [线程1-原有分析] 开始时间: %s, 线程ID: %p", originalStart.Format("15:04:05.000"), &originalStart)

		analysisResult, analysisErr = s.executeOriginalAnalysis(contextData, content)

		originalEnd := time.Now()
		originalDuration = originalEnd.Sub(originalStart)
		if analysisErr == nil {
			log.Printf("✅ [线程1-原有分析] 完成时间: %s, 耗时: %v", originalEnd.Format("15:04:05.000"), originalDuration)
		} else {
			log.Printf("❌ [线程1-原有分析] 失败时间: %s, 耗时: %v, 错误: %v", originalEnd.Format("15:04:05.000"), originalDuration, analysisErr)
		}
	}()

	// 专门化知识图谱分析（并行执行）
	go func() {
		defer wg.Done()
		kgStart := time.Now()
		log.Printf("🕸️ [线程2-专门KG] 开始时间: %s, 线程ID: %p", kgStart.Format("15:04:05.000"), &kgStart)

		kgExtraction, kgErr = s.executeDedicatedKGAnalysis(contextData, content)

		kgEnd := time.Now()
		kgDuration = kgEnd.Sub(kgStart)
		if kgErr == nil {
			log.Printf("✅ [线程2-专门KG] 完成时间: %s, 耗时: %v", kgEnd.Format("15:04:05.000"), kgDuration)
		} else {
			log.Printf("❌ [线程2-专门KG] 失败时间: %s, 耗时: %v, 错误: %v", kgEnd.Format("15:04:05.000"), kgDuration, kgErr)
		}
	}()

	log.Printf("⏳ [主线程] 等待两个并行任务完成...")
	// 等待两个并行任务完成
	wg.Wait()

	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)
	log.Printf("🏁 [主线程] 并行任务全部完成 - 结束时间: %s", endTime.Format("15:04:05.000"))
	log.Printf("📊 [并行统计] 总耗时: %v, 原有分析: %v, 专门KG: %v", totalDuration, originalDuration, kgDuration)
	log.Printf("🔍 [并行验证] 理论最短时间: %v, 实际时间: %v, 并行效率: %.1f%%",
		maxDuration(originalDuration, kgDuration), totalDuration,
		float64(maxDuration(originalDuration, kgDuration))/float64(totalDuration)*100)

	// 处理结果
	if analysisErr != nil {
		return nil, fmt.Errorf("原有分析失败: %w", analysisErr)
	}

	// 合并知识图谱结果（如果成功的话）
	if kgErr == nil && kgExtraction != nil {
		analysisResult.KnowledgeGraphExtraction = kgExtraction
		log.Printf("🔗 [并行合并] 成功合并知识图谱抽取结果，实体: %d个，关系: %d个",
			len(kgExtraction.Entities), len(kgExtraction.Relationships))
	} else {
		log.Printf("⚠️ [并行合并] 知识图谱分析失败，将在后续使用规则匹配降级")
	}

	log.Printf("✅ [方案二] 并行分析完成，整体置信度: %.2f, 总耗时: %v", analysisResult.ConfidenceAssessment.OverallConfidence, totalDuration)
	return analysisResult, nil
}

// maxDuration 返回两个时间间隔中的最大值
func maxDuration(d1, d2 time.Duration) time.Duration {
	if d1 > d2 {
		return d1
	}
	return d2
}

// executeDedicatedKGAnalysis 执行专门化的知识图谱分析
func (s *ContextService) executeDedicatedKGAnalysis(contextData *models.LLMDrivenContextModel, content string) (*models.KnowledgeGraphExtraction, error) {
	funcStart := time.Now()
	log.Printf("🕸️ [专门KG] 开始专门化知识图谱分析 - 函数开始: %s", funcStart.Format("15:04:05.000"))

	// 构建专门的知识图谱抽取prompt
	promptStart := time.Now()
	prompt := s.buildDedicatedKGPrompt(contextData, content)
	promptDuration := time.Since(promptStart)
	log.Printf("📝 [专门KG] 构建prompt完成: %s, 耗时: %v, 长度: %d", time.Now().Format("15:04:05.000"), promptDuration, len(prompt))

	// 创建LLM客户端
	clientStart := time.Now()
	llmProvider := s.config.MultiDimLLMProvider
	llmModel := s.config.MultiDimLLMModel
	if llmProvider == "" {
		return nil, fmt.Errorf("LLM提供商未配置")
	}

	llmClient, err := s.createStandardLLMClient(llmProvider, llmModel)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}
	clientDuration := time.Since(clientStart)
	log.Printf("🔧 [专门KG] 创建LLM客户端完成: %s, 耗时: %v", time.Now().Format("15:04:05.000"), clientDuration)

	// 构建专门的LLM请求
	requestStart := time.Now()
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   3000, // 专门化任务，token稍少
		Temperature: 0.1,
		Format:      "json",
		Model:       llmModel,
		Metadata: map[string]interface{}{
			"task":            "dedicated_knowledge_graph_extraction",
			"session_id":      contextData.SessionID,
			"content_length":  len(content),
			"skip_rate_limit": true, // 🔥 跳过限流检查，支持并行
			"parallel_call":   true, // 🔥 标记为并行调用
		},
	}
	requestDuration := time.Since(requestStart)
	log.Printf("📋 [专门KG] 构建LLM请求完成: %s, 耗时: %v", time.Now().Format("15:04:05.000"), requestDuration)

	// 调用LLM API
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	apiCallStart := time.Now()
	log.Printf("🚀 [专门KG] 开始调用LLM API: %s, 提供商: %s，模型: %s", apiCallStart.Format("15:04:05.000"), llmProvider, llmModel)
	log.Printf("🔍 [专门KG] 限流检查开始: %s", time.Now().Format("15:04:05.000"))

	llmResponse, err := llmClient.Complete(ctx, llmRequest)

	apiCallEnd := time.Now()
	apiCallDuration := apiCallEnd.Sub(apiCallStart)
	if err != nil {
		log.Printf("❌ [专门KG] LLM API调用失败: %s, 耗时: %v, 错误: %v", apiCallEnd.Format("15:04:05.000"), apiCallDuration, err)
		return nil, fmt.Errorf("专门化LLM API调用失败: %w", err)
	}

	log.Printf("✅ [专门KG] LLM API调用完成: %s, 耗时: %v, Token使用: %d", apiCallEnd.Format("15:04:05.000"), apiCallDuration, llmResponse.TokensUsed)
	log.Printf("📄 [专门KG] LLM响应长度: %d 字符", len(llmResponse.Content))

	// 解析专门的知识图谱响应
	parseStart := time.Now()
	kgExtraction, err := s.parseDedicatedKGResponse(llmResponse.Content)
	parseDuration := time.Since(parseStart)
	if err != nil {
		log.Printf("❌ [专门KG] 解析失败: %s, 耗时: %v, 错误: %v", time.Now().Format("15:04:05.000"), parseDuration, err)
		return nil, fmt.Errorf("解析专门化KG响应失败: %w", err)
	}

	funcEnd := time.Now()
	funcDuration := funcEnd.Sub(funcStart)
	log.Printf("✅ [专门KG] 函数完成: %s, 总耗时: %v, 实体: %d个，关系: %d个",
		funcEnd.Format("15:04:05.000"), funcDuration, len(kgExtraction.Entities), len(kgExtraction.Relationships))
	log.Printf("📊 [专门KG] 阶段耗时 - Prompt: %v, Client: %v, Request: %v, API: %v, Parse: %v",
		promptDuration, clientDuration, requestDuration, apiCallDuration, parseDuration)

	return kgExtraction, nil
}

// buildDedicatedKGPrompt 构建专门的知识图谱抽取prompt（方案二：高质量专门化）
func (s *ContextService) buildDedicatedKGPrompt(contextData *models.LLMDrivenContextModel, content string) string {
	return fmt.Sprintf(`你是专业的知识图谱构建专家，专门从技术文档和对话中抽取实体和关系。

## 🎯 核心任务
从用户内容中构建高质量的知识图谱，提取实体和关系信息。

## 📊 实体抽取标准（6种通用类型）

### 1. Technical（技术实体）
- 编程语言: Go, Python, Java, JavaScript, C++
- 框架工具: Spring Boot, React, Vue, Docker, Kubernetes
- 数据库: MySQL, Redis, PostgreSQL, Neo4j, MongoDB
- 技术产品: Context-Keeper, 微服务系统, API网关

### 2. Project（项目工作）
- 项目: 电商系统开发, 性能优化项目, 架构重构
- 功能: 订单支付模块, 用户管理功能, 数据分析
- 任务: 数据库优化, 接口开发, 性能调优

### 3. Concept（技术概念）
- 架构概念: 微服务架构, 分层设计, 事件驱动
- 技术概念: 并发处理, 缓存策略, 负载均衡
- 设计模式: 单例模式, 工厂模式, 观察者模式

### 4. Issue（事件问题）
- 技术问题: 性能瓶颈, 内存泄漏, 并发问题
- 系统事件: 服务故障, 数据丢失, 网络中断
- 优化事件: 性能优化, 架构升级, 代码重构

### 5. Data（数据资源）
- 性能数据: 72秒, 1000TPS, 15%%失败率, 99.9%%可用性
- 配置参数: 超时时间, 连接池大小, 缓存大小
- 版本信息: v1.0.0, 2025-08-20, 第一阶段

### 6. Process（操作流程）
- 技术操作: 数据库查询, API调用, 缓存更新
- 部署操作: 服务部署, 配置更新, 环境切换
- 开发流程: 代码审查, 测试执行, 持续集成

## 🔗 关系抽取标准（5种核心关系）

### 1. USES（使用关系）
- 技术栈: Context-Keeper USES Neo4j
- 工具链: 项目 USES Spring Boot

### 2. SOLVES（解决关系）
- 问题解决: 性能优化 SOLVES 响应慢
- 技术解决: 缓存策略 SOLVES 并发问题

### 3. BELONGS_TO（归属关系）
- 模块归属: 支付模块 BELONGS_TO 电商系统
- 功能归属: 用户登录 BELONGS_TO 用户管理

### 4. CAUSES（因果关系）
- 问题原因: 高并发 CAUSES 性能下降
- 技术因果: 内存泄漏 CAUSES 系统崩溃

### 5. RELATED_TO（相关关系）
- 概念相关: 微服务 RELATED_TO 分布式架构
- 技术相关: Docker RELATED_TO Kubernetes

## 📝 分析内容
**会话ID**: %s
**用户内容**: %s

## 📋 输出格式
请严格按照以下JSON格式输出：

{
  "entities": [
    {
      "title": "Context-Keeper",
      "type": "Technical",
      "description": "LLM驱动的上下文管理系统",
      "confidence": 0.95,
      "keywords": ["上下文", "管理", "LLM"]
    }
  ],
  "relationships": [
    {
      "source": "性能优化",
      "target": "客户端超时",
      "relation_type": "SOLVES",
      "description": "性能优化解决了客户端超时问题",
      "strength": 9,
      "confidence": 0.9,
      "evidence": "接口耗时从72秒降到22秒，客户端超时问题完全消除"
    }
  ],
  "extraction_meta": {
    "entity_count": 0,
    "relationship_count": 0,
    "overall_quality": 0.85
  }
}`,
		contextData.SessionID,
		content)
}

// parseEnhancedSmartAnalysisResponse 解析增强的智能分析响应（方案一）
func (s *ContextService) parseEnhancedSmartAnalysisResponse(response string) (*models.SmartAnalysisResult, error) {
	// 首先使用原有的解析逻辑
	analysisResult, err := s.parseSmartAnalysisResponse(response)
	if err != nil {
		return nil, err
	}

	// 🔥 额外解析knowledge_extraction字段
	cleanedResponse := s.cleanLLMResponse(response)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		log.Printf("⚠️ [增强解析] JSON解析失败，使用基础结果: %v", err)
		return analysisResult, nil
	}

	// 解析knowledge_extraction字段
	if kgData, exists := result["knowledge_extraction"]; exists {
		kgExtraction := s.parseKnowledgeExtractionData(kgData)
		if kgExtraction != nil {
			analysisResult.KnowledgeGraphExtraction = kgExtraction
			log.Printf("✅ [增强解析] 成功解析知识图谱信息，实体: %d个，关系: %d个",
				len(kgExtraction.Entities), len(kgExtraction.Relationships))
		}
	}

	return analysisResult, nil
}

// parseDedicatedKGResponse 解析专门的知识图谱响应（方案二）
func (s *ContextService) parseDedicatedKGResponse(response string) (*models.KnowledgeGraphExtraction, error) {
	cleanedResponse := s.cleanLLMResponse(response)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	kgExtraction := &models.KnowledgeGraphExtraction{}

	// 解析entities
	if entitiesRaw, exists := result["entities"]; exists {
		if entitiesList, ok := entitiesRaw.([]interface{}); ok {
			for _, entityRaw := range entitiesList {
				if entityMap, ok := entityRaw.(map[string]interface{}); ok {
					entity := models.LLMExtractedEntity{
						Title:       getStringFromMap(entityMap, "title", ""),
						Type:        getStringFromMap(entityMap, "type", ""),
						Description: getStringFromMap(entityMap, "description", ""),
						Confidence:  getFloat64FromMap(entityMap, "confidence"),
					}

					// 解析keywords
					if keywordsRaw, exists := entityMap["keywords"]; exists {
						if keywordsList, ok := keywordsRaw.([]interface{}); ok {
							for _, keyword := range keywordsList {
								if keywordStr, ok := keyword.(string); ok {
									entity.Keywords = append(entity.Keywords, keywordStr)
								}
							}
						}
					}

					if entity.Title != "" && entity.Type != "" {
						kgExtraction.Entities = append(kgExtraction.Entities, entity)
					}
				}
			}
		}
	}

	// 解析relationships
	if relationshipsRaw, exists := result["relationships"]; exists {
		if relationshipsList, ok := relationshipsRaw.([]interface{}); ok {
			for _, relationshipRaw := range relationshipsList {
				if relationshipMap, ok := relationshipRaw.(map[string]interface{}); ok {
					relationship := models.LLMExtractedRelationship{
						Source:       getStringFromMap(relationshipMap, "source", ""),
						Target:       getStringFromMap(relationshipMap, "target", ""),
						RelationType: getStringFromMap(relationshipMap, "relation_type", ""),
						Description:  getStringFromMap(relationshipMap, "description", ""),
						Strength:     int(getFloat64FromMap(relationshipMap, "strength")),
						Confidence:   getFloat64FromMap(relationshipMap, "confidence"),
						Evidence:     getStringFromMap(relationshipMap, "evidence", ""),
					}

					if relationship.Source != "" && relationship.Target != "" && relationship.RelationType != "" {
						kgExtraction.Relationships = append(kgExtraction.Relationships, relationship)
					}
				}
			}
		}
	}

	// 解析extraction_meta
	if metaRaw, exists := result["extraction_meta"]; exists {
		if metaMap, ok := metaRaw.(map[string]interface{}); ok {
			kgExtraction.ExtractionMeta = &models.ExtractionMetadata{
				EntityCount:       int(getFloat64FromMap(metaMap, "entity_count")),
				RelationshipCount: int(getFloat64FromMap(metaMap, "relationship_count")),
				OverallQuality:    getFloat64FromMap(metaMap, "overall_quality"),
				ProcessingTime:    getStringFromMap(metaMap, "processing_time", ""),
				StrategyUsed:      "parallel_dedicated",
			}
		}
	}

	log.Printf("✅ [专门解析] 解析完成，实体: %d个，关系: %d个",
		len(kgExtraction.Entities), len(kgExtraction.Relationships))

	return kgExtraction, nil
}

// parseKnowledgeExtractionData 解析knowledge_extraction数据（方案一的简化格式）
func (s *ContextService) parseKnowledgeExtractionData(kgData interface{}) *models.KnowledgeGraphExtraction {
	kgMap, ok := kgData.(map[string]interface{})
	if !ok {
		log.Printf("⚠️ [KG解析] knowledge_extraction格式错误")
		return nil
	}

	kgExtraction := &models.KnowledgeGraphExtraction{}

	// 解析简化格式的entities: ["实体名(类型)", ...]
	if entitiesRaw, exists := kgMap["entities"]; exists {
		if entitiesList, ok := entitiesRaw.([]interface{}); ok {
			for _, entityRaw := range entitiesList {
				if entityStr, ok := entityRaw.(string); ok {
					entity := s.parseEntityString(entityStr)
					if entity != nil {
						kgExtraction.Entities = append(kgExtraction.Entities, *entity)
					}
				}
			}
		}
	}

	// 解析简化格式的relations: ["源->关系->目标", ...]
	if relationsRaw, exists := kgMap["relations"]; exists {
		if relationsList, ok := relationsRaw.([]interface{}); ok {
			for _, relationRaw := range relationsList {
				if relationStr, ok := relationRaw.(string); ok {
					relationship := s.parseRelationString(relationStr)
					if relationship != nil {
						kgExtraction.Relationships = append(kgExtraction.Relationships, *relationship)
					}
				}
			}
		}
	}

	// 设置元数据
	kgExtraction.ExtractionMeta = &models.ExtractionMetadata{
		EntityCount:       len(kgExtraction.Entities),
		RelationshipCount: len(kgExtraction.Relationships),
		OverallQuality:    0.8, // 默认质量评分
		StrategyUsed:      "enhanced_prompt",
	}

	return kgExtraction
}

// parseEntityString 解析实体字符串 "实体名(类型)"
func (s *ContextService) parseEntityString(entityStr string) *models.LLMExtractedEntity {
	// 解析格式: "Context-Keeper(Technical)"
	if !strings.Contains(entityStr, "(") || !strings.Contains(entityStr, ")") {
		log.Printf("⚠️ [实体解析] 格式错误: %s", entityStr)
		return nil
	}

	parts := strings.Split(entityStr, "(")
	if len(parts) != 2 {
		return nil
	}

	title := strings.TrimSpace(parts[0])
	typeWithParen := parts[1]
	entityType := strings.TrimSpace(strings.TrimSuffix(typeWithParen, ")"))

	return &models.LLMExtractedEntity{
		Title:       title,
		Type:        entityType,
		Description: fmt.Sprintf("%s类型的%s", entityType, title),
		Confidence:  0.85, // 默认置信度
		Keywords:    []string{title},
	}
}

// parseRelationString 解析关系字符串 "源->关系->目标"
func (s *ContextService) parseRelationString(relationStr string) *models.LLMExtractedRelationship {
	// 解析格式: "性能优化->SOLVES->客户端超时"
	parts := strings.Split(relationStr, "->")
	if len(parts) != 3 {
		log.Printf("⚠️ [关系解析] 格式错误: %s", relationStr)
		return nil
	}

	source := strings.TrimSpace(parts[0])
	relationType := strings.TrimSpace(parts[1])
	target := strings.TrimSpace(parts[2])

	return &models.LLMExtractedRelationship{
		Source:       source,
		Target:       target,
		RelationType: relationType,
		Description:  fmt.Sprintf("%s%s%s", source, s.getRelationDescription(relationType), target),
		Strength:     8,   // 默认强度
		Confidence:   0.8, // 默认置信度
		Evidence:     relationStr,
	}
}

// getRelationDescription 获取关系描述
func (s *ContextService) getRelationDescription(relationType string) string {
	switch relationType {
	case "USES":
		return "使用"
	case "SOLVES":
		return "解决"
	case "BELONGS_TO":
		return "属于"
	case "CAUSES":
		return "导致"
	case "RELATED_TO":
		return "相关"
	default:
		return "关联"
	}
}

// convertLLMEntitiesToKnowledgeEntities 将LLM抽取的实体转换为KnowledgeEntity
func (s *ContextService) convertLLMEntitiesToKnowledgeEntities(llmEntities []models.LLMExtractedEntity, req models.StoreContextRequest, memoryID string) []*KnowledgeEntity {
	log.Printf("🔄 [实体转换] 开始转换LLM抽取的实体，数量: %d", len(llmEntities))

	var entities []*KnowledgeEntity
	for _, llmEntity := range llmEntities {
		entity := &KnowledgeEntity{
			Name:            llmEntity.Title,
			Type:            s.mapLLMTypeToEntityType(llmEntity.Type),
			Category:        s.getCategoryByLLMType(llmEntity.Type),
			SourceDimension: "llm_extracted",
			ConfidenceLevel: llmEntity.Confidence,
			Keywords:        llmEntity.Keywords,
			Properties: map[string]interface{}{
				"llm_extracted":     true,
				"original_type":     llmEntity.Type,
				"description":       llmEntity.Description,
				"extraction_method": "llm_analysis",
			},
			MemoryID:  memoryID,
			SessionID: req.SessionID,
			UserID:    req.UserID,
			CreatedAt: time.Now(),
		}

		entities = append(entities, entity)
		log.Printf("🎯 [实体转换] 转换实体: %s (%s -> %s, 置信度: %.2f)",
			entity.Name, llmEntity.Type, entity.Type, entity.ConfidenceLevel)
	}

	log.Printf("✅ [实体转换] 转换完成，获得%d个KnowledgeEntity", len(entities))
	return entities
}

// mapLLMTypeToEntityType 将LLM实体类型映射到现有的EntityType
func (s *ContextService) mapLLMTypeToEntityType(llmType string) EntityType {
	switch strings.ToLower(llmType) {
	case "technical":
		return EntityTypeTechnical
	case "project":
		return EntityTypeProject
	case "concept":
		return EntityTypeConcept
	case "issue":
		return EntityTypeProblem // Issue映射到Problem
	case "data", "process":
		return EntityTypeConcept // Data和Process映射到Concept
	default:
		return EntityTypeConcept // 默认映射
	}
}

// getCategoryByLLMType 根据LLM类型获取分类
func (s *ContextService) getCategoryByLLMType(llmType string) string {
	switch strings.ToLower(llmType) {
	case "technical":
		return "技术组件"
	case "project":
		return "项目模块"
	case "concept":
		return "概念定义"
	case "issue":
		return "问题事件"
	case "data":
		return "数据资源"
	case "process":
		return "流程操作"
	default:
		return "未知类型"
	}
}

// getEnvVar 获取环境变量
func (s *ContextService) getEnvVar(key string) string {
	return os.Getenv(key)
}

// 🔧 临时解决方案：分析结果管理方法
// GetLastAnalysisResult 获取最后一次分析结果
func (s *ContextService) GetLastAnalysisResult() *models.SmartAnalysisResult {
	s.analysisResultMutex.RLock()
	defer s.analysisResultMutex.RUnlock()
	return s.lastAnalysisResult
}

// setLastAnalysisResult 设置最后一次分析结果
func (s *ContextService) setLastAnalysisResult(result *models.SmartAnalysisResult) {
	s.analysisResultMutex.Lock()
	defer s.analysisResultMutex.Unlock()
	s.lastAnalysisResult = result
}

// getBasicAnalysisResult 获取基础分析结果
func (s *ContextService) getBasicAnalysisResult(content string) map[string]interface{} {
	return map[string]interface{}{
		"timeline_data": map[string]interface{}{
			"title":            "基础事件",
			"content":          content,
			"event_type":       "存储",
			"keywords":         []string{"存储", "内容"},
			"importance_score": 5,
		},
		"knowledge_graph_data": map[string]interface{}{
			"main_concepts": []interface{}{
				map[string]interface{}{"name": "内容存储", "type": "技术", "importance": 0.8},
				map[string]interface{}{"name": "数据库", "type": "技术", "importance": 0.9},
				map[string]interface{}{"name": "存储系统", "type": "系统", "importance": 0.7},
			},
			"relationships": []interface{}{
				map[string]interface{}{"from": "用户", "to": "内容存储", "relation": "执行", "strength": 0.9},
				map[string]interface{}{"from": "内容存储", "to": "数据库", "relation": "使用", "strength": 0.8},
				map[string]interface{}{"from": "数据库", "to": "存储系统", "relation": "属于", "strength": 0.7},
			},
			"domain": "存储管理",
		},
		"vector_data": map[string]interface{}{
			"content":         content,
			"semantic_tags":   []string{"存储", "内容"},
			"context_summary": "用户存储内容",
			"relevance_score": 0.7,
		},
	}
}

// createStandardLLMClient 创建标准LLM客户端（参考查询链路的实现）
func (s *ContextService) createStandardLLMClient(provider, model string) (llm.LLMClient, error) {
	log.Printf("🔧 [LLM客户端] 创建标准LLM客户端，提供商: %s，模型: %s", provider, model)

	// 获取对应的API Key
	var apiKey string
	switch provider {
	case "deepseek":
		apiKey = s.getEnvVar("DEEPSEEK_API_KEY")
	case "openai":
		apiKey = s.getEnvVar("OPENAI_API_KEY")
	case "claude":
		apiKey = s.getEnvVar("CLAUDE_API_KEY")
	case "qianwen":
		apiKey = s.getEnvVar("QIANWEN_API_KEY")
	case "ollama_local":
		// 🆕 本地模型不需要API密钥
		apiKey = ""
	default:
		return nil, fmt.Errorf("不支持的LLM提供商: %s", provider)
	}

	// 🔥 修复：本地模型不需要API密钥检查
	if apiKey == "" && provider != "ollama_local" {
		return nil, fmt.Errorf("LLM API Key未配置，提供商: %s", provider)
	}

	// 🔥 检查是否已有缓存的客户端
	factory := llm.GetGlobalFactory()
	if existingClient, exists := factory.GetClient(llm.LLMProvider(provider)); exists {
		log.Printf("♻️ [LLM客户端] 使用缓存的客户端，提供商: %s", provider)
		return existingClient, nil
	}

	// 🔥 使用真实的LLM工厂创建客户端（参考查询链路的实现）
	config := &llm.LLMConfig{
		Provider:   llm.LLMProvider(provider),
		APIKey:     apiKey,
		Model:      model,
		MaxRetries: 3,
		Timeout:    120 * time.Second, // 增加到120秒，适应复杂LLM分析
		RateLimit:  300,               // 🔥 增加到300次/分钟，支持并行调用（5次/秒）
	}

	// 🆕 设置本地模型的BaseURL和特殊配置
	if provider == "ollama_local" {
		config.BaseURL = "http://localhost:11434"
		config.RateLimit = 0              // 本地模型无限流限制
		config.Timeout = 60 * time.Second // 本地模型更快
	}

	log.Printf("🔧 [LLM客户端] 设置全局配置，限流: %d次/分钟", config.RateLimit)
	// 设置全局配置
	llm.SetGlobalConfig(llm.LLMProvider(provider), config)

	// 使用工厂创建客户端
	createStart := time.Now()
	client, err := llm.CreateGlobalClient(llm.LLMProvider(provider))
	createDuration := time.Since(createStart)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	log.Printf("✅ [LLM客户端] LLM客户端创建成功，提供商: %s, 模型: %s, 创建耗时: %v", provider, model, createDuration)
	return client, nil
}

// parseLLMResponse 解析LLM响应
func (s *ContextService) parseLLMResponse(response string) (map[string]interface{}, error) {
	log.Printf("🔍 [LLM解析] 开始解析LLM响应，长度: %d", len(response))

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	log.Printf("✅ [LLM解析] 响应解析成功，包含 %d 个数据维度", len(result))
	return result, nil
}

// parseSmartAnalysisResponse 解析智能分析响应（替换parseStorageAnalysisResponse）
func (s *ContextService) parseSmartAnalysisResponse(response string) (*models.SmartAnalysisResult, error) {
	log.Printf("🔍 [智能分析解析] 开始解析LLM响应，长度: %d", len(response))

	// 🔥 清理markdown代码块标记（处理DeepSeek等LLM返回的格式）
	cleanedResponse := s.cleanLLMResponse(response)
	log.Printf("🧹 [智能分析解析] 清理后响应长度: %d", len(cleanedResponse))

	var rawResult map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &rawResult); err != nil {
		log.Printf("❌ [智能分析解析] JSON解析失败，原始响应: %s", response)
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 构建SmartAnalysisResult
	result := &models.SmartAnalysisResult{
		RawLLMResponse: response,
	}

	// 解析intent_analysis
	if intentRaw, exists := rawResult["intent_analysis"]; exists {
		if intentMap, ok := intentRaw.(map[string]interface{}); ok {
			result.IntentAnalysis = &models.IntentAnalysisResult{
				CoreIntentText:    getStringFromMap(intentMap, "core_intent_text", ""),
				DomainContextText: getStringFromMap(intentMap, "domain_context_text", ""),
				ScenarioText:      getStringFromMap(intentMap, "scenario_text", ""),
				IntentCount:       getIntFromMap(intentMap, "intent_count"),
			}

			// 解析multi_intent_breakdown
			if breakdownRaw, exists := intentMap["multi_intent_breakdown"]; exists {
				if breakdownSlice, ok := breakdownRaw.([]interface{}); ok {
					for _, item := range breakdownSlice {
						if str, ok := item.(string); ok {
							result.IntentAnalysis.MultiIntentBreakdown = append(result.IntentAnalysis.MultiIntentBreakdown, str)
						}
					}
				}
			}
		}
	}

	// 解析confidence_assessment
	if confidenceRaw, exists := rawResult["confidence_assessment"]; exists {
		if confidenceMap, ok := confidenceRaw.(map[string]interface{}); ok {
			result.ConfidenceAssessment = &models.ConfidenceAssessment{
				SemanticClarity:         getFloat64FromMap(confidenceMap, "semantic_clarity"),
				InformationCompleteness: getFloat64FromMap(confidenceMap, "information_completeness"),
				IntentConfidence:        getFloat64FromMap(confidenceMap, "intent_confidence"),
				OverallConfidence:       getFloat64FromMap(confidenceMap, "overall_confidence"),
			}

			// 解析missing_elements
			if missingRaw, exists := confidenceMap["missing_elements"]; exists {
				if missingSlice, ok := missingRaw.([]interface{}); ok {
					for _, item := range missingSlice {
						if str, ok := item.(string); ok {
							result.ConfidenceAssessment.MissingElements = append(result.ConfidenceAssessment.MissingElements, str)
						}
					}
				}
			}

			// 解析clarity_issues
			if issuesRaw, exists := confidenceMap["clarity_issues"]; exists {
				if issuesSlice, ok := issuesRaw.([]interface{}); ok {
					for _, item := range issuesSlice {
						if str, ok := item.(string); ok {
							result.ConfidenceAssessment.ClarityIssues = append(result.ConfidenceAssessment.ClarityIssues, str)
						}
					}
				}
			}
		}
	}

	// 解析storage_recommendations
	if storageRaw, exists := rawResult["storage_recommendations"]; exists {
		if storageMap, ok := storageRaw.(map[string]interface{}); ok {
			result.StorageRecommendations = &models.StorageRecommendations{}

			// 解析timeline_storage
			if timelineRaw, exists := storageMap["timeline_storage"]; exists {
				if timelineMap, ok := timelineRaw.(map[string]interface{}); ok {
					// 🔥 解析并标准化时间格式
					rawTimelineTime := getStringFromMap(timelineMap, "timeline_time", "")
					standardizedTime := s.standardizeTimeFormat(rawTimelineTime)

					result.StorageRecommendations.TimelineStorage = &models.StorageRecommendation{
						ShouldStore:         getBoolFromMap(timelineMap, "should_store"),
						Reason:              getStringFromMap(timelineMap, "reason", ""),
						ConfidenceThreshold: getFloat64FromMap(timelineMap, "confidence_threshold"),
						TimelineTime:        standardizedTime,                                // 🔥 使用标准化后的时间
						EventType:           getStringFromMap(timelineMap, "event_type", ""), // 🆕 解析事件类型
					}
				}
			}

			// 解析knowledge_graph_storage
			if kgRaw, exists := storageMap["knowledge_graph_storage"]; exists {
				if kgMap, ok := kgRaw.(map[string]interface{}); ok {
					result.StorageRecommendations.KnowledgeGraphStorage = &models.StorageRecommendation{
						ShouldStore:         getBoolFromMap(kgMap, "should_store"),
						Reason:              getStringFromMap(kgMap, "reason", ""),
						ConfidenceThreshold: getFloat64FromMap(kgMap, "confidence_threshold"),
					}
				}
			}

			// 解析vector_storage
			if vectorRaw, exists := storageMap["vector_storage"]; exists {
				if vectorMap, ok := vectorRaw.(map[string]interface{}); ok {
					result.StorageRecommendations.VectorStorage = &models.VectorStorageRecommendation{
						StorageRecommendation: &models.StorageRecommendation{
							ShouldStore:         getBoolFromMap(vectorMap, "should_store"),
							Reason:              getStringFromMap(vectorMap, "reason", ""),
							ConfidenceThreshold: getFloat64FromMap(vectorMap, "confidence_threshold"),
						},
					}

					// 解析enabled_dimensions
					if dimensionsRaw, exists := vectorMap["enabled_dimensions"]; exists {
						if dimensionsSlice, ok := dimensionsRaw.([]interface{}); ok {
							for _, item := range dimensionsSlice {
								if str, ok := item.(string); ok {
									result.StorageRecommendations.VectorStorage.EnabledDimensions = append(result.StorageRecommendations.VectorStorage.EnabledDimensions, str)
								}
							}
						}
					}
				}
			}
		}
	}

	// 🔥 验证必要字段，缺失则返回错误让调用者降级处理
	if result.IntentAnalysis == nil {
		log.Printf("❌ [智能分析解析] 缺少intent_analysis字段，返回错误触发降级")
		return nil, fmt.Errorf("LLM响应缺少intent_analysis字段")
	}
	if result.ConfidenceAssessment == nil {
		log.Printf("❌ [智能分析解析] 缺少confidence_assessment字段，返回错误触发降级")
		return nil, fmt.Errorf("LLM响应缺少confidence_assessment字段")
	}
	if result.StorageRecommendations == nil {
		log.Printf("❌ [智能分析解析] 缺少storage_recommendations字段，返回错误触发降级")
		return nil, fmt.Errorf("LLM响应缺少storage_recommendations字段")
	}

	log.Printf("✅ [智能分析解析] 响应解析成功，整体置信度: %.2f", result.ConfidenceAssessment.OverallConfidence)
	return result, nil
}

// cleanLLMResponse 清理LLM响应中的markdown代码块标记
func (s *ContextService) cleanLLMResponse(response string) string {
	// 移除markdown代码块标记
	response = strings.TrimSpace(response)

	// 移除开头的```json或```
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	// 移除结尾的```
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	// 再次清理空白字符
	response = strings.TrimSpace(response)

	return response
}

// 辅助函数：从map中获取整数值
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, exists := m[key]; exists {
		if num, ok := val.(float64); ok {
			return int(num)
		}
		if num, ok := val.(int); ok {
			return num
		}
	}
	return 0
}

// 辅助函数：从map中获取浮点数值
func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if val, exists := m[key]; exists {
		if num, ok := val.(float64); ok {
			return num
		}
		if num, ok := val.(int); ok {
			return float64(num)
		}
	}
	return 0.0
}

// 辅助函数：从map中获取布尔值
func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// getContextOnlyThreshold 获取仅上下文记录的置信度阈值
func (s *ContextService) getContextOnlyThreshold() float64 {
	if s.llmDrivenConfig != nil {
		if config := s.llmDrivenConfig.GetConfig(); config != nil {
			return config.SmartStorage.ConfidenceThresholds.ContextOnlyThreshold
		}
	}
	return 0.5 // 默认阈值
}

// extractWorkspaceName 从会话元数据中提取工程名
func (s *ContextService) extractWorkspaceName(sessionID string) string {
	// 🔥 从会话元数据中获取实际的工作空间路径
	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		log.Printf("❌ [工程名提取] 获取会话失败: %v", err)
		return ""
	}

	// 从会话元数据中获取workspacePath
	if session.Metadata == nil {
		log.Printf("⚠️ [工程名提取] 会话元数据为空")
		return ""
	}

	workspacePath, ok := session.Metadata["workspacePath"].(string)
	if !ok || workspacePath == "" {
		log.Printf("⚠️ [工程名提取] 会话元数据中没有workspacePath")
		return ""
	}

	// 🔥 从完整路径中提取最后一级目录名作为工程名
	if strings.Contains(workspacePath, "/") {
		parts := strings.Split(workspacePath, "/")
		workspaceName := parts[len(parts)-1]
		if workspaceName != "" {
			log.Printf("🔧 [工程名提取] 从路径 %s 提取工程名: %s", workspacePath, workspaceName)
			return workspaceName
		}
	}

	// 如果路径不包含/，直接返回原路径
	log.Printf("🔧 [工程名提取] 路径不包含分隔符，直接使用: %s", workspacePath)
	return workspacePath
}

// getTimelineStorageThreshold 获取时间线存储的置信度阈值
func (s *ContextService) getTimelineStorageThreshold() float64 {
	if s.llmDrivenConfig != nil {
		if config := s.llmDrivenConfig.GetConfig(); config != nil {
			return config.SmartStorage.ConfidenceThresholds.TimelineStorage
		}
	}
	return 0.7 // 默认阈值
}

// getKnowledgeGraphStorageThreshold 获取知识图谱存储的置信度阈值
func (s *ContextService) getKnowledgeGraphStorageThreshold() float64 {
	if s.llmDrivenConfig != nil {
		if config := s.llmDrivenConfig.GetConfig(); config != nil {
			return config.SmartStorage.ConfidenceThresholds.KnowledgeGraphStorage
		}
	}
	return 0.6 // 默认阈值
}

// getVectorStorageThreshold 获取向量存储的置信度阈值
func (s *ContextService) getVectorStorageThreshold() float64 {
	if s.llmDrivenConfig != nil {
		if config := s.llmDrivenConfig.GetConfig(); config != nil {
			return config.SmartStorage.ConfidenceThresholds.VectorStorage
		}
	}
	return 0.5 // 默认阈值
}

// getEnabledDimensions 获取启用的向量维度
func (s *ContextService) getEnabledDimensions() []string {
	if s.llmDrivenConfig != nil {
		if config := s.llmDrivenConfig.GetConfig(); config != nil {
			return config.SmartStorage.MultiVector.EnabledDimensions
		}
	}
	return []string{"core_intent", "domain_context", "scenario"} // 默认维度
}

// getBasicSmartAnalysisResult 获取基础智能分析结果（降级时使用）
func (s *ContextService) getBasicSmartAnalysisResult(content string) *models.SmartAnalysisResult {
	return &models.SmartAnalysisResult{
		IntentAnalysis: &models.IntentAnalysisResult{
			CoreIntentText:       content[:min(50, len(content))], // 截取前50个字符作为核心意图
			DomainContextText:    "通用领域",
			ScenarioText:         "基础场景",
			IntentCount:          1,
			MultiIntentBreakdown: []string{content[:min(30, len(content))]},
		},
		ConfidenceAssessment: &models.ConfidenceAssessment{
			SemanticClarity:         0.3, // 低置信度
			InformationCompleteness: 0.3,
			IntentConfidence:        0.3,
			OverallConfidence:       0.3, // 整体低置信度，触发上下文记录
			MissingElements:         []string{"LLM分析失败"},
			ClarityIssues:           []string{"降级到基础分析"},
		},
		StorageRecommendations: &models.StorageRecommendations{
			TimelineStorage: &models.StorageRecommendation{
				ShouldStore:         false,
				Reason:              "LLM分析失败，无法确定时间线信息",
				ConfidenceThreshold: 0.7,
			},
			KnowledgeGraphStorage: &models.StorageRecommendation{
				ShouldStore:         false,
				Reason:              "LLM分析失败，无法确定概念关系",
				ConfidenceThreshold: 0.6,
			},
			VectorStorage: &models.VectorStorageRecommendation{
				StorageRecommendation: &models.StorageRecommendation{
					ShouldStore:         false,
					Reason:              "LLM分析失败，置信度过低",
					ConfidenceThreshold: 0.5,
				},
				EnabledDimensions: []string{}, // 空维度
			},
		},
		RawLLMResponse: "LLM分析失败，使用基础分析结果",
	}
}

// executeSmartStorage 执行智能存储策略（替换storeToMultiDimensionalEngines）
func (s *ContextService) executeSmartStorage(ctx context.Context, analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest) (string, error) {
	log.Printf("🧠 [智能存储] 开始执行智能存储决策")

	overallConfidence := analysisResult.ConfidenceAssessment.OverallConfidence
	log.Printf("📊 [智能存储] 整体置信度: %.2f", overallConfidence)

	// 生成统一的记忆ID
	memoryID := uuid.New().String()

	// 低置信度：仅记录上下文，不进行长期存储
	contextOnlyThreshold := s.getContextOnlyThreshold()
	if overallConfidence < contextOnlyThreshold {
		log.Printf("⚠️ [智能存储] 置信度过低(%.2f < %.2f)，仅记录上下文",
			overallConfidence, contextOnlyThreshold)
		return s.storeContextOnly(analysisResult, req, memoryID)
	}

	// 中高置信度：根据推荐结果选择性存储 - 🔥 并行执行
	log.Printf("✅ [智能存储] 置信度满足要求，执行并行选择性存储")

	var storageErrors []error
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// 检查存储条件
	timelineStorage := analysisResult.StorageRecommendations.TimelineStorage
	shouldStoreTimeline := timelineStorage.ShouldStore || timelineStorage.TimelineTime == "now"
	shouldStoreKnowledge := analysisResult.StorageRecommendations.KnowledgeGraphStorage.ShouldStore
	shouldStoreVector := analysisResult.StorageRecommendations.VectorStorage.ShouldStore

	log.Printf("📊 [智能存储] 并行存储计划 - 时间线:%v, 知识图谱:%v, 向量:%v",
		shouldStoreTimeline, shouldStoreKnowledge, shouldStoreVector)

	// 1. 时间线存储 (并行)
	if shouldStoreTimeline {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startTime := time.Now()

			if timelineStorage.TimelineTime == "now" {
				log.Printf("⏰ [并行-时间线] 检测到结论性内容，强制执行时间线存储 (timeline_time=now)")
			} else {
				log.Printf("⏰ [并行-时间线] 执行时间线存储 (明确时间信息)")
			}

			if err := s.storeTimelineDataToTimescaleDB(ctx, analysisResult, req, memoryID); err != nil {
				log.Printf("❌ [并行-时间线] 时间线存储失败: %v, 耗时: %v", err, time.Since(startTime))
				mutex.Lock()
				storageErrors = append(storageErrors, fmt.Errorf("时间线存储失败: %w", err))
				mutex.Unlock()
			} else {
				log.Printf("✅ [并行-时间线] 时间线存储成功, 耗时: %v", time.Since(startTime))
			}
		}()
	} else {
		log.Printf("⏰ [智能存储] 跳过时间线存储: %s", timelineStorage.Reason)
	}

	// 2. 知识图谱存储 (并行)
	if shouldStoreKnowledge {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startTime := time.Now()

			log.Printf("🕸️ [并行-知识图谱] 执行知识图谱存储")
			if err := s.storeKnowledgeDataToNeo4j(ctx, analysisResult, req, memoryID); err != nil {
				log.Printf("❌ [并行-知识图谱] 知识图谱存储失败: %v, 耗时: %v", err, time.Since(startTime))
				mutex.Lock()
				storageErrors = append(storageErrors, fmt.Errorf("知识图谱存储失败: %w", err))
				mutex.Unlock()
			} else {
				log.Printf("✅ [并行-知识图谱] 知识图谱存储成功, 耗时: %v", time.Since(startTime))
			}
		}()
	} else {
		log.Printf("🕸️ [智能存储] 跳过知识图谱存储: %s", analysisResult.StorageRecommendations.KnowledgeGraphStorage.Reason)
	}

	// 3. 多向量存储 (并行)
	if shouldStoreVector {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startTime := time.Now()

			log.Printf("🔍 [并行-向量] 执行多向量存储")
			if err := s.storeMultiVectorData(analysisResult, req, memoryID); err != nil {
				log.Printf("❌ [并行-向量] 多向量存储失败: %v, 耗时: %v", err, time.Since(startTime))
				mutex.Lock()
				storageErrors = append(storageErrors, fmt.Errorf("多向量存储失败: %w", err))
				mutex.Unlock()
			} else {
				log.Printf("✅ [并行-向量] 多向量存储成功, 耗时: %v", time.Since(startTime))
			}
		}()
	} else {
		log.Printf("🔍 [智能存储] 跳过多向量存储: %s", analysisResult.StorageRecommendations.VectorStorage.Reason)
	}

	// 等待所有并行存储完成
	log.Printf("⏳ [智能存储] 等待所有并行存储完成...")
	wg.Wait()
	log.Printf("🏁 [智能存储] 所有并行存储已完成")

	// 如果所有存储都失败，返回错误
	if len(storageErrors) > 0 && len(storageErrors) == 3 {
		return "", fmt.Errorf("所有存储引擎都失败: %v", storageErrors)
	}

	log.Printf("🎉 [智能存储] 智能存储完成，记忆ID: %s", memoryID)
	return memoryID, nil
}

// storeContextOnly 仅记录上下文（低置信度时使用）
func (s *ContextService) storeContextOnly(analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) (string, error) {
	log.Printf("📝 [上下文记录] 开始记录上下文信息，置信度过低")

	// 创建基础记忆对象，仅用于上下文记录
	memory := models.NewMemory(req.SessionID, req.Content, req.Priority, req.Metadata)
	memory.ID = memoryID

	// 设置业务类型和用户ID
	if req.BizType > 0 {
		memory.BizType = req.BizType
	}
	if req.UserID != "" {
		memory.UserID = req.UserID
	}

	// 在元数据中记录分析结果和置信度信息
	if memory.Metadata == nil {
		memory.Metadata = make(map[string]interface{})
	}
	memory.Metadata["context_only"] = true
	memory.Metadata["overall_confidence"] = analysisResult.ConfidenceAssessment.OverallConfidence
	memory.Metadata["missing_elements"] = analysisResult.ConfidenceAssessment.MissingElements
	memory.Metadata["clarity_issues"] = analysisResult.ConfidenceAssessment.ClarityIssues
	memory.Metadata["storage_reason"] = "置信度过低，仅记录上下文"

	// 🔥 修复：低置信度内容也需要生成基础向量才能存储
	log.Printf("🔧 [上下文记录] 为低置信度内容生成基础向量")
	vector, err := s.generateEmbedding(req.Content)
	if err != nil {
		log.Printf("❌ [上下文记录] 基础向量生成失败: %v", err)
		return "", fmt.Errorf("基础向量生成失败: %w", err)
	}
	memory.Vector = vector

	// 存储到向量数据库
	if err := s.storeMemory(memory); err != nil {
		log.Printf("❌ [上下文记录] 上下文记录失败: %v", err)
		return "", fmt.Errorf("上下文记录失败: %w", err)
	}

	log.Printf("✅ [上下文记录] 上下文记录成功，等待后续完善: %s", memoryID)
	return memoryID, nil
}

// storeMultiVectorData 存储多向量数据（一条记录，多个向量字段）
func (s *ContextService) storeMultiVectorData(analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) error {
	log.Printf("🔍 [多向量存储] 开始处理多向量数据")

	intentAnalysis := analysisResult.IntentAnalysis

	// 创建基础记忆对象
	memory := models.NewMemory(req.SessionID, req.Content, req.Priority, req.Metadata)
	memory.ID = memoryID

	// 设置业务类型和用户ID
	if req.BizType > 0 {
		memory.BizType = req.BizType
	}
	if req.UserID != "" {
		memory.UserID = req.UserID
	}

	// 创建多向量数据对象
	multiVectorData := &models.MultiVectorData{
		QualityScore: analysisResult.ConfidenceAssessment,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// 根据启用的维度生成对应的向量
	enabledDimensions := analysisResult.StorageRecommendations.VectorStorage.EnabledDimensions
	log.Printf("🎯 [多向量存储] 启用的维度: %v", enabledDimensions)

	vectorCount := 0
	for _, dimension := range enabledDimensions {
		switch dimension {
		case "core_intent", "core_intent_text", "Core Intent Vector":
			if intentAnalysis.CoreIntentText != "" {
				log.Printf("🔍 [多向量存储] 生成核心意图向量: %s", intentAnalysis.CoreIntentText)
				vector, err := s.generateEmbedding(intentAnalysis.CoreIntentText)
				if err == nil {
					multiVectorData.CoreIntentVector = vector
					multiVectorData.CoreIntentText = intentAnalysis.CoreIntentText
					multiVectorData.CoreIntentWeight = 0.5 // 最高权重
					vectorCount++
					log.Printf("✅ [多向量存储] 核心意图向量生成成功，维度: %d", len(vector))
				} else {
					log.Printf("⚠️ [多向量存储] 核心意图向量生成失败: %v", err)
				}
			}
		case "domain_context", "domain_context_text", "Domain Context Vector":
			if intentAnalysis.DomainContextText != "" {
				log.Printf("🔍 [多向量存储] 生成领域上下文向量: %s", intentAnalysis.DomainContextText)
				vector, err := s.generateEmbedding(intentAnalysis.DomainContextText)
				if err == nil {
					multiVectorData.DomainContextVector = vector
					multiVectorData.DomainContextText = intentAnalysis.DomainContextText
					multiVectorData.DomainContextWeight = 0.3
					vectorCount++
					log.Printf("✅ [多向量存储] 领域上下文向量生成成功，维度: %d", len(vector))
				} else {
					log.Printf("⚠️ [多向量存储] 领域上下文向量生成失败: %v", err)
				}
			}
		case "scenario", "scenario_text", "Scenario Vector":
			if intentAnalysis.ScenarioText != "" {
				log.Printf("🔍 [多向量存储] 生成场景向量: %s", intentAnalysis.ScenarioText)
				vector, err := s.generateEmbedding(intentAnalysis.ScenarioText)
				if err == nil {
					multiVectorData.ScenarioVector = vector
					multiVectorData.ScenarioText = intentAnalysis.ScenarioText
					multiVectorData.ScenarioWeight = 0.15
					vectorCount++
					log.Printf("✅ [多向量存储] 场景向量生成成功，维度: %d", len(vector))
				} else {
					log.Printf("⚠️ [多向量存储] 场景向量生成失败: %v", err)
				}
			}
		}
	}

	if vectorCount == 0 {
		return fmt.Errorf("没有生成任何维度的向量")
	}

	// 设置多向量数据到记忆对象
	memory.MultiVectorData = multiVectorData

	// 同时设置主向量（使用核心意图向量，如果存在的话）
	if multiVectorData.CoreIntentVector != nil {
		memory.Vector = multiVectorData.CoreIntentVector
	} else if multiVectorData.DomainContextVector != nil {
		memory.Vector = multiVectorData.DomainContextVector
	} else if multiVectorData.ScenarioVector != nil {
		memory.Vector = multiVectorData.ScenarioVector
	}

	// 在元数据中标记多向量信息
	if memory.Metadata == nil {
		memory.Metadata = make(map[string]interface{})
	}
	memory.Metadata["multi_vector"] = true
	memory.Metadata["vector_count"] = vectorCount
	memory.Metadata["enabled_dimensions"] = enabledDimensions
	memory.Metadata["overall_confidence"] = analysisResult.ConfidenceAssessment.OverallConfidence

	// 存储到向量数据库（一条记录，多个向量字段）
	if err := s.storeMemory(memory); err != nil {
		return fmt.Errorf("多向量记忆存储失败: %w", err)
	}

	log.Printf("🎉 [多向量存储] 多向量数据存储完成，总计 %d 个维度", vectorCount)
	return nil
}

// storeTimelineDataToTimescaleDB 存储时间线数据到TimescaleDB
func (s *ContextService) storeTimelineDataToTimescaleDB(ctx context.Context, analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) error {
	log.Printf("⏰ [TimescaleDB存储] 开始存储时间线数据")

	// 🔥 处理时间标识
	var eventTime time.Time
	timelineTime := analysisResult.StorageRecommendations.TimelineStorage.TimelineTime

	if timelineTime == "now" || timelineTime == "" {
		// 当前时间
		eventTime = time.Now()
		log.Printf("⏰ [时间处理] 使用当前时间: %s", eventTime.Format("2006-01-02 15:04:05"))
	} else {
		// 尝试解析具体时间（这里可以扩展更多时间格式的解析）
		if parsedTime, err := parseTimeString(timelineTime); err == nil {
			eventTime = parsedTime
			log.Printf("⏰ [时间处理] 解析时间成功: %s -> %s", timelineTime, eventTime.Format("2006-01-02 15:04:05"))
		} else {
			// 解析失败，使用当前时间
			eventTime = time.Now()
			log.Printf("⚠️ [时间处理] 时间解析失败，使用当前时间: %s", eventTime.Format("2006-01-02 15:04:05"))
		}
	}

	// 基于智能分析结果构建时间线数据
	log.Printf("📊 [TimescaleDB存储] 基于智能分析结果存储时间线")
	log.Printf("🔧 [TimescaleDB存储] 会话: %s, 用户: %s, 记忆ID: %s, 事件时间: %s",
		req.SessionID, req.UserID, memoryID, eventTime.Format("2006-01-02 15:04:05"))

	// 🔥 调用真正的TimescaleDB存储实现
	log.Printf("🔥 [TimescaleDB存储] 调用真实的TimescaleDB存储")

	// 构建时间线数据
	timelineData := map[string]interface{}{
		"session_id":    req.SessionID,
		"user_id":       req.UserID,
		"memory_id":     memoryID,
		"content":       req.Content,
		"priority":      req.Priority,
		"metadata":      req.Metadata,
		"event_time":    eventTime,
		"analysis_data": analysisResult,
		"timeline_time": analysisResult.StorageRecommendations.TimelineStorage.TimelineTime,
	}

	// 调用真实的TimescaleDB存储
	return s.storeToRealTimescaleDB(ctx, timelineData, req, memoryID)
}

// standardizeTimeFormat 标准化时间格式
func (s *ContextService) standardizeTimeFormat(rawTime string) string {
	// 🔥 只有"now"保持原样，其他都转换为具体时间戳格式
	if rawTime == "now" {
		return "now" // 保持now标识
	}

	if rawTime == "" {
		return "" // 空值保持原样
	}

	now := time.Now()

	// 🔥 处理相对时间表述，全部转换为具体日期格式
	switch rawTime {
	case "昨天", "yesterday":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "今天", "today":
		return now.Format("2006-01-02")
	case "前天", "day before yesterday":
		return now.AddDate(0, 0, -2).Format("2006-01-02")
	case "上周", "last week":
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	case "上个月", "last month":
		return now.AddDate(0, -1, 0).Format("2006-01-02")
	default:
		// 尝试解析已有的标准格式
		if parsedTime, err := time.Parse("2006-01-02", rawTime); err == nil {
			return parsedTime.Format("2006-01-02") // 确保格式统一
		}
		// 如果无法解析，尝试其他常见格式
		formats := []string{
			"2006/01/02",
			"2006-1-2",
			"2006/1/2",
		}
		for _, format := range formats {
			if parsedTime, err := time.Parse(format, rawTime); err == nil {
				return parsedTime.Format("2006-01-02")
			}
		}
		// 🔥 如果都无法解析，返回当前日期（降级处理）
		log.Printf("⚠️ [时间标准化] 无法解析时间格式: %s，使用当前日期", rawTime)
		return now.Format("2006-01-02")
	}
}

// parseTimeString 解析时间字符串
func parseTimeString(timeStr string) (time.Time, error) {
	// 支持的时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"01-02",
		"15:04:05",
		"15:04",
	}

	// 处理相对时间词汇
	now := time.Now()
	switch timeStr {
	case "昨天", "yesterday":
		return now.AddDate(0, 0, -1), nil
	case "今天", "today":
		return now, nil
	case "明天", "tomorrow":
		return now.AddDate(0, 0, 1), nil
	case "上周", "last week":
		return now.AddDate(0, 0, -7), nil
	case "下周", "next week":
		return now.AddDate(0, 0, 7), nil
	case "上个月", "last month":
		return now.AddDate(0, -1, 0), nil
	case "下个月", "next month":
		return now.AddDate(0, 1, 0), nil
	}

	// 尝试解析具体时间格式
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			// 如果只有日期没有年份，使用当前年份
			if format == "01-02" {
				t = t.AddDate(now.Year()-1, 0, 0)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间字符串: %s", timeStr)
}

// storeToRealTimescaleDB 存储到真实的TimescaleDB
func (s *ContextService) storeToRealTimescaleDB(ctx context.Context, timelineData map[string]interface{}, req models.StoreContextRequest, memoryID string) error {
	log.Printf("🔥 [真实TimescaleDB] 开始连接TimescaleDB并存储数据")

	// 获取TimescaleDB配置
	timescaleConfig := s.getTimescaleDBConfig()
	if timescaleConfig == nil {
		return fmt.Errorf("❌ [真实TimescaleDB] TimescaleDB配置加载失败或未启用")
	}

	// 创建TimescaleDB引擎
	timelineEngine, err := s.createTimescaleDBEngine(timescaleConfig)
	if err != nil {
		log.Printf("❌ [真实TimescaleDB] 创建TimescaleDB引擎失败: %v", err)
		return fmt.Errorf("创建TimescaleDB引擎失败: %w", err)
	}
	defer timelineEngine.Close()

	// 转换LLM分析结果为TimescaleDB事件
	event, err := s.convertToTimelineEvent(timelineData, req, memoryID)
	if err != nil {
		log.Printf("❌ [真实TimescaleDB] 转换时间线事件失败: %v", err)
		return fmt.Errorf("转换时间线事件失败: %w", err)
	}

	// 存储到TimescaleDB
	eventID, err := timelineEngine.StoreEvent(ctx, event)
	if err != nil {
		log.Printf("❌ [真实TimescaleDB] 存储时间线事件失败: %v", err)
		return fmt.Errorf("存储时间线事件失败: %w", err)
	}

	log.Printf("✅ [真实TimescaleDB] 时间线事件存储成功 - EventID: %s, MemoryID: %s", eventID, memoryID)
	return nil
}

// getTimescaleDBConfig 获取TimescaleDB配置
func (s *ContextService) getTimescaleDBConfig() *timeline.TimescaleDBConfig {
	// 使用统一配置管理器加载配置
	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		log.Printf("❌ 加载数据库配置失败: %v", err)
		return nil // 不提供降级方案，强制报错
	}

	if !dbConfig.TimescaleDB.Enabled {
		log.Printf("⚠️ TimescaleDB未启用")
		return nil
	}

	// 转换配置格式
	return &timeline.TimescaleDBConfig{
		Host:        dbConfig.TimescaleDB.Host,
		Port:        dbConfig.TimescaleDB.Port,
		Database:    dbConfig.TimescaleDB.Database,
		Username:    dbConfig.TimescaleDB.Username,
		Password:    dbConfig.TimescaleDB.Password,
		SSLMode:     dbConfig.TimescaleDB.SSLMode,
		MaxConns:    dbConfig.TimescaleDB.MaxConns,
		MaxIdleTime: dbConfig.TimescaleDB.MaxIdleTime,
	}
}

// createTimescaleDBEngine 创建TimescaleDB引擎
func (s *ContextService) createTimescaleDBEngine(config *timeline.TimescaleDBConfig) (*timeline.TimescaleDBEngine, error) {
	return timeline.NewTimescaleDBEngine(config)
}

// convertToTimelineEvent 转换LLM分析结果为TimescaleDB事件
func (s *ContextService) convertToTimelineEvent(timelineData map[string]interface{}, req models.StoreContextRequest, memoryID string) (*timeline.TimelineEvent, error) {
	// 🔥 从timelineData中提取LLM分析结果
	analysisResult, ok := timelineData["analysis_data"].(*models.SmartAnalysisResult)
	if !ok {
		log.Printf("⚠️ [时间线转换] 无法提取LLM分析结果，使用基础数据")
	}

	// 🔥 智能生成title和summary
	title, summary := s.extractTitleSummary(req.Content, analysisResult)

	// 🔥 确定事件类型 - 优先使用LLM判断的类型
	eventType := "intent_based" // 默认类型
	if analysisResult != nil && analysisResult.StorageRecommendations != nil &&
		analysisResult.StorageRecommendations.TimelineStorage != nil {
		// 🔥 使用LLM分析的事件类型
		llmEventType := analysisResult.StorageRecommendations.TimelineStorage.EventType
		if llmEventType != "" {
			eventType = llmEventType
			log.Printf("🏷️ [事件类型] 使用LLM判断的事件类型: %s", eventType)
		} else {
			log.Printf("⚠️ [事件类型] LLM未返回事件类型，使用默认: %s", eventType)
		}
	}

	// 🔥 提取关键词 - 修复：使用MultiIntentBreakdown并确保格式一致性
	var keywords []string
	if analysisResult != nil && analysisResult.IntentAnalysis != nil {
		// ✅ 优先方式：使用LLM分析的多意图拆分作为关键词
		if len(analysisResult.IntentAnalysis.MultiIntentBreakdown) > 0 {
			// 🔧 关键词预处理：确保格式一致性
			for _, rawKeyword := range analysisResult.IntentAnalysis.MultiIntentBreakdown {
				// 处理可能的长字符串：如果包含逗号，则拆分
				if strings.Contains(rawKeyword, ",") {
					// 拆分长字符串
					parts := strings.Split(rawKeyword, ",")
					for _, part := range parts {
						part = strings.TrimSpace(part)
						if len(part) > 0 && len(part) <= 20 { // 限制关键词长度
							keywords = append(keywords, part)
						}
					}
				} else {
					// 直接使用短关键词
					keyword := strings.TrimSpace(rawKeyword)
					if len(keyword) > 0 && len(keyword) <= 20 { // 限制关键词长度
						keywords = append(keywords, keyword)
					}
				}

				// 限制总关键词数量
				if len(keywords) >= 8 {
					break
				}
			}

			log.Printf("✅ [关键词提取] 从MultiIntentBreakdown提取到 %d 个关键词: %v", len(keywords), keywords)
		}
	}

	// 🔥 计算重要性分数
	importanceScore := 0.5 // 默认分数
	if analysisResult != nil && analysisResult.ConfidenceAssessment != nil {
		importanceScore = analysisResult.ConfidenceAssessment.OverallConfidence
	}

	// 🔥 处理事件时间
	eventTime, ok := timelineData["event_time"].(time.Time)
	if !ok {
		eventTime = time.Now()
	}

	// 创建时间线事件
	// 🔥 优化：从完整路径提取工程名
	workspaceName := s.extractWorkspaceName(req.SessionID)

	event := &timeline.TimelineEvent{
		ID:              memoryID,
		UserID:          req.UserID,
		SessionID:       req.SessionID,
		WorkspaceID:     workspaceName, // 使用工程名而非完整路径
		Timestamp:       eventTime,
		EventType:       eventType,
		Title:           title,
		Content:         req.Content,
		Summary:         &summary, // 使用LLM生成的摘要
		Keywords:        keywords,
		ImportanceScore: importanceScore,
		RelevanceScore:  0.8,                    // 默认相关性分数
		Intent:          &eventType,             // 🔥 修复：使用指针类型
		Categories:      []string{req.Priority}, // 使用优先级作为分类
	}

	log.Printf("🔧 [时间线转换] 事件转换完成 - 标题: %s, 类型: %s, 重要性: %.2f",
		event.Title, event.EventType, event.ImportanceScore)

	return event, nil
}

// extractTitleSummary 智能提取title和summary
func (s *ContextService) extractTitleSummary(content string, analysisResult *models.SmartAnalysisResult) (string, string) {
	// 🔥 使用LLM生成结果，失败就简单兜底
	if analysisResult != nil && analysisResult.IntentAnalysis != nil {
		intentAnalysis := analysisResult.IntentAnalysis

		// Title: 复用core_intent_text，为空就用简单截取
		title := intentAnalysis.CoreIntentText
		if title == "" {
			title = s.simpleTitle(content)
		}

		// Summary: 使用LLM生成的summary，为空就用content
		summary := intentAnalysis.Summary
		if summary == "" {
			summary = content
		}

		log.Printf("🎯 Title/Summary - Title: %s, Summary: %s", title, summary)
		return title, summary
	}

	// 🔥 LLM失败就简单处理
	log.Printf("⚠️ LLM分析失败，简单处理")
	return s.simpleTitle(content), content
}

// simpleTitle 简单标题提取
func (s *ContextService) simpleTitle(content string) string {
	runes := []rune(content)
	if len(runes) > 30 {
		return string(runes[:27]) + "..."
	}
	return string(runes)
}

// storeKnowledgeDataToNeo4j 存储知识图谱数据到Neo4j
func (s *ContextService) storeKnowledgeDataToNeo4j(ctx context.Context, analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) error {
	log.Printf("🕸️ [Neo4j存储] 开始存储知识图谱数据")

	// 构建知识图谱数据
	knowledgeData := map[string]interface{}{
		"session_id":    req.SessionID,
		"user_id":       req.UserID,
		"memory_id":     memoryID,
		"content":       req.Content,
		"priority":      req.Priority,
		"metadata":      req.Metadata,
		"analysis_data": analysisResult,
		"created_at":    time.Now(),
	}

	// 调用真实的Neo4j存储
	return s.storeToRealNeo4j(ctx, knowledgeData, req, memoryID)
}

// storeToRealNeo4j 存储到真实的Neo4j
func (s *ContextService) storeToRealNeo4j(ctx context.Context, knowledgeData map[string]interface{}, req models.StoreContextRequest, memoryID string) error {
	log.Printf("🔥 [真实Neo4j] 开始连接Neo4j并存储数据")

	// 获取Neo4j配置
	neo4jConfig := s.getNeo4jConfig()

	// 创建Neo4j引擎
	knowledgeEngine, err := s.createNeo4jEngine(neo4jConfig)
	if err != nil {
		log.Printf("❌ [真实Neo4j] 创建Neo4j引擎失败: %v", err)
		return fmt.Errorf("创建Neo4j引擎失败: %w", err)
	}
	defer knowledgeEngine.Close(ctx)

	// 转换LLM分析结果为Neo4j概念和关系
	concepts, relationships, err := s.convertToKnowledgeGraph(knowledgeData, req, memoryID)
	if err != nil {
		log.Printf("❌ [真实Neo4j] 转换知识图谱失败: %v", err)
		return fmt.Errorf("转换知识图谱失败: %w", err)
	}

	// 存储概念到Neo4j
	for _, concept := range concepts {
		if err := knowledgeEngine.CreateConcept(ctx, concept); err != nil {
			log.Printf("❌ [真实Neo4j] 存储概念失败: %v", err)
			return fmt.Errorf("存储概念失败: %w", err)
		}
	}

	// 存储关系到Neo4j
	for _, relationship := range relationships {
		if err := knowledgeEngine.CreateRelationship(ctx, relationship); err != nil {
			log.Printf("❌ [真实Neo4j] 存储关系失败: %v", err)
			return fmt.Errorf("存储关系失败: %w", err)
		}
	}

	log.Printf("✅ [真实Neo4j] 知识图谱存储成功 - 概念: %d, 关系: %d, MemoryID: %s",
		len(concepts), len(relationships), memoryID)
	return nil
}

// getNeo4jConfig 获取Neo4j配置
func (s *ContextService) getNeo4jConfig() *knowledge.Neo4jConfig {
	// 使用统一配置管理器加载配置
	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		log.Printf("❌ 加载数据库配置失败: %v", err)
		return nil // 不提供降级方案，强制报错
	}

	if !dbConfig.Neo4j.Enabled {
		log.Printf("⚠️ Neo4j未启用")
		return nil
	}

	// 转换配置格式
	return &knowledge.Neo4jConfig{
		URI:                     dbConfig.Neo4j.URI,
		Username:                dbConfig.Neo4j.Username,
		Password:                dbConfig.Neo4j.Password,
		Database:                dbConfig.Neo4j.Database,
		MaxConnectionPoolSize:   dbConfig.Neo4j.MaxConnectionPoolSize,
		ConnectionTimeout:       dbConfig.Neo4j.ConnectionTimeout,
		MaxTransactionRetryTime: dbConfig.Neo4j.MaxTransactionRetryTime,
	}
}

// createNeo4jEngine 创建Neo4j引擎
func (s *ContextService) createNeo4jEngine(config *knowledge.Neo4jConfig) (*knowledge.Neo4jEngine, error) {
	return knowledge.NewNeo4jEngine(config)
}

// convertToKnowledgeGraph 转换LLM分析结果为Neo4j概念和关系 - 规则解析方式
func (s *ContextService) convertToKnowledgeGraph(knowledgeData map[string]interface{}, req models.StoreContextRequest, memoryID string) ([]*knowledge.Concept, []*knowledge.Relationship, error) {
	log.Printf("🧠 [知识图谱转换] 开始规则解析LLM分析结果")

	// 从knowledgeData中获取LLM分析结果
	analysisDataRaw, exists := knowledgeData["analysis_data"]
	if !exists {
		return nil, nil, fmt.Errorf("缺少LLM分析结果")
	}

	analysisResult, ok := analysisDataRaw.(*models.SmartAnalysisResult)
	if !ok {
		return nil, nil, fmt.Errorf("LLM分析结果格式错误")
	}

	// 🔥 规则解析：从四维度文本中提取实体关键词
	entities := s.extractEntitiesFromAnalysisResult(analysisResult, req, memoryID)

	// 🔥 规则构建：基于实体关键词构建预定义关系
	relationships := s.buildPredefinedRelationships(entities, analysisResult, req, memoryID)

	// 转换为Neo4j存储格式
	concepts := s.convertEntitiesToConcepts(entities, req, memoryID)
	neo4jRelations := s.convertToNeo4jRelationships(relationships, req, memoryID)

	log.Printf("🔄 [知识图谱转换] 规则转换完成 - 实体: %d, 概念: %d, 关系: %d",
		len(entities), len(concepts), len(neo4jRelations))

	return concepts, neo4jRelations, nil
}

// extractEntitiesFromAnalysisResult 从LLM分析结果中抽取实体（优先使用LLM抽取结果）
func (s *ContextService) extractEntitiesFromAnalysisResult(analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) []*KnowledgeEntity {
	log.Printf("🔍 [实体解析] 开始抽取实体，模式: %s", s.getKnowledgeGraphExtractionMode())

	// 🔥 优先使用LLM抽取的知识图谱信息
	if analysisResult.KnowledgeGraphExtraction != nil && len(analysisResult.KnowledgeGraphExtraction.Entities) > 0 {
		log.Printf("✅ [实体解析] 使用LLM抽取的知识图谱信息，实体数: %d", len(analysisResult.KnowledgeGraphExtraction.Entities))
		return s.convertLLMEntitiesToKnowledgeEntities(analysisResult.KnowledgeGraphExtraction.Entities, req, memoryID)
	}

	// 降级：使用原有的规则匹配逻辑
	log.Printf("⚠️ [实体解析] LLM未提供知识图谱抽取结果，降级到规则匹配")
	return s.extractEntitiesWithRuleMatching(analysisResult, req, memoryID)
}

// extractEntitiesWithRuleMatching 使用规则匹配的原有逻辑
func (s *ContextService) extractEntitiesWithRuleMatching(analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) []*KnowledgeEntity {
	log.Printf("🔍 [规则匹配] 开始规则解析LLM分析结果中的实体")

	var entities []*KnowledgeEntity
	intentAnalysis := analysisResult.IntentAnalysis

	// 1. 从CoreIntentText中解析技术和项目实体
	if intentAnalysis.CoreIntentText != "" {
		coreEntities := s.parseEntitiesFromText(intentAnalysis.CoreIntentText, "core_intent", req, memoryID)
		entities = append(entities, coreEntities...)
		log.Printf("✅ [实体解析] 核心意图实体: %d个", len(coreEntities))
	}

	// 2. 从DomainContextText中解析技术实体
	if intentAnalysis.DomainContextText != "" {
		domainEntities := s.parseEntitiesFromText(intentAnalysis.DomainContextText, "domain_context", req, memoryID)
		entities = append(entities, domainEntities...)
		log.Printf("✅ [实体解析] 领域上下文实体: %d个", len(domainEntities))
	}

	// 3. 从ScenarioText中解析问题和概念实体
	if intentAnalysis.ScenarioText != "" {
		scenarioEntities := s.parseEntitiesFromText(intentAnalysis.ScenarioText, "scenario", req, memoryID)
		entities = append(entities, scenarioEntities...)
		log.Printf("✅ [实体解析] 场景实体: %d个", len(scenarioEntities))
	}

	// 4. 去重和过滤
	filteredEntities := s.deduplicateEntitiesRuleBased(entities)

	log.Printf("🎯 [实体解析] 规则解析完成 - 原始: %d, 过滤后: %d", len(entities), len(filteredEntities))
	return filteredEntities
}

// parseEntitiesFromText 从文本中规则解析实体 (不调用LLM)
func (s *ContextService) parseEntitiesFromText(text, dimension string, req models.StoreContextRequest, memoryID string) []*KnowledgeEntity {
	log.Printf("🔧 [规则解析] 从%s维度解析实体: %s", dimension, text[:min(50, len(text))])

	var entities []*KnowledgeEntity

	// 🔥 技术实体识别规则 (基于关键词匹配)
	technicalKeywords := []string{
		// 编程语言
		"Go", "Python", "JavaScript", "Java", "Rust", "C++", "C#",
		// 数据库
		"TimescaleDB", "Neo4j", "PostgreSQL", "MySQL", "Redis", "MongoDB",
		// 框架工具
		"Gin", "Docker", "Kubernetes", "React", "Vue", "Angular",
		// 技术概念
		"微服务", "API", "向量数据库", "LLM", "机器学习", "深度学习",
		"RESTful", "GraphQL", "gRPC", "WebSocket",
	}

	// 🔥 项目实体识别规则
	projectKeywords := []string{
		"Context-Keeper", "检索引擎", "存储服务", "分析引擎",
		"多维检索", "上下文管理", "会话管理", "记忆管理",
		"存储链路", "检索链路", "智能分析",
	}

	// 🔥 概念实体识别规则
	conceptKeywords := []string{
		"设计模式", "架构模式", "分层架构", "事件驱动",
		"用户会话", "上下文感知", "智能化", "个性化",
		"多维度存储", "知识图谱", "向量存储", "时间线存储",
	}

	// 🔥 问题实体识别规则
	problemKeywords := []string{
		"性能瓶颈", "内存泄漏", "并发问题", "数据一致性",
		"P0致命问题", "缺陷", "错误", "故障", "异常",
		"优化", "修复", "改进", "解决",
	}

	// 规则匹配实体
	entities = append(entities, s.matchEntitiesByKeywords(text, technicalKeywords, EntityTypeTechnical, dimension, req, memoryID)...)
	entities = append(entities, s.matchEntitiesByKeywords(text, projectKeywords, EntityTypeProject, dimension, req, memoryID)...)
	entities = append(entities, s.matchEntitiesByKeywords(text, conceptKeywords, EntityTypeConcept, dimension, req, memoryID)...)
	entities = append(entities, s.matchEntitiesByKeywords(text, problemKeywords, EntityTypeProblem, dimension, req, memoryID)...)

	log.Printf("✅ [规则解析] %s维度解析完成，实体数: %d", dimension, len(entities))
	return entities
}

// matchEntitiesByKeywords 通过关键词匹配实体
func (s *ContextService) matchEntitiesByKeywords(text string, keywords []string, entityType EntityType, dimension string, req models.StoreContextRequest, memoryID string) []*KnowledgeEntity {
	var entities []*KnowledgeEntity

	textLower := strings.ToLower(text)

	for _, keyword := range keywords {
		keywordLower := strings.ToLower(keyword)
		if strings.Contains(textLower, keywordLower) {
			// 计算置信度 (基于匹配度和上下文)
			confidence := s.calculateEntityConfidence(text, keyword, dimension)

			entity := &KnowledgeEntity{
				Name:            keyword,
				Type:            entityType,
				Category:        s.getCategoryByType(entityType),
				SourceDimension: dimension,
				ConfidenceLevel: confidence,
				Keywords:        []string{keyword},
				Properties: map[string]interface{}{
					"match_method":      "keyword_match",
					"source_text":       text,
					"context_relevance": confidence,
				},
				MemoryID:  memoryID,
				SessionID: req.SessionID,
				UserID:    req.UserID,
				CreatedAt: time.Now(),
			}

			entities = append(entities, entity)
			log.Printf("🎯 [关键词匹配] 发现实体: %s (%s, 置信度: %.2f)", keyword, entityType, confidence)
		}
	}

	return entities
}

// calculateEntityConfidence 计算实体置信度
func (s *ContextService) calculateEntityConfidence(text, keyword, dimension string) float64 {
	// 基础置信度
	baseConfidence := 0.7

	// 完全匹配加分
	if strings.Contains(text, keyword) {
		baseConfidence += 0.1
	}

	// 关键词长度加分 (更长的关键词更可信)
	if len(keyword) > 10 {
		baseConfidence += 0.1
	}

	// 维度相关性加分
	switch dimension {
	case "core_intent":
		baseConfidence += 0.05 // 核心意图维度权重高
	case "domain_context":
		baseConfidence += 0.1 // 领域上下文维度最可信
	}

	// 上下文丰富度加分
	if len(text) > 50 {
		baseConfidence += 0.05
	}

	// 确保在0-1范围内
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// getCategoryByType 根据实体类型获取分类
func (s *ContextService) getCategoryByType(entityType EntityType) string {
	switch entityType {
	case EntityTypeTechnical:
		return "技术组件"
	case EntityTypeProject:
		return "项目模块"
	case EntityTypeConcept:
		return "概念定义"
	case EntityTypeProblem:
		return "问题识别"
	case EntityTypePerson:
		return "人员角色"
	default:
		return "未知类型"
	}
}

// buildPredefinedRelationships 基于规则构建预定义关系 (不调用LLM)
func (s *ContextService) buildPredefinedRelationships(entities []*KnowledgeEntity, analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) []*KnowledgeRelationship {
	log.Printf("🔗 [关系构建] 开始规则构建实体关系，实体数量: %d", len(entities))

	if len(entities) < 2 {
		log.Printf("ℹ️ [关系构建] 实体数量不足，跳过关系构建")
		return []*KnowledgeRelationship{}
	}

	var relationships []*KnowledgeRelationship

	// 🔥 规则1: 技术USES关系 (技术实体之间的使用关系)
	relationships = append(relationships, s.buildTechnicalUsesRelations(entities, req, memoryID)...)

	// 🔥 规则2: 项目COMPOSED_OF关系 (项目由组件组成)
	relationships = append(relationships, s.buildProjectCompositionRelations(entities, req, memoryID)...)

	// 🔥 规则3: 解决SOLVES关系 (优化解决问题)
	relationships = append(relationships, s.buildProblemSolvingRelations(entities, req, memoryID)...)

	// 🔥 规则4: 概念RELATED_TO关系 (概念之间的相关性) - 简化实现
	relationships = append(relationships, s.buildConceptRelatedRelations(entities, req, memoryID)...)

	// 过滤低质量关系
	filteredRelationships := s.filterLowQualityRelationships(relationships)

	log.Printf("✅ [关系构建] 规则关系构建完成 - 原始: %d, 过滤后: %d", len(relationships), len(filteredRelationships))
	return filteredRelationships
}

// buildTechnicalUsesRelations 构建技术USES关系
func (s *ContextService) buildTechnicalUsesRelations(entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) []*KnowledgeRelationship {
	var relations []*KnowledgeRelationship

	// 查找技术实体
	var techEntities []*KnowledgeEntity
	for _, entity := range entities {
		if entity.Type == EntityTypeTechnical {
			techEntities = append(techEntities, entity)
		}
	}

	// 构建技术使用关系的规则
	techUsageRules := map[string][]string{
		"Context-Keeper": {"Go", "TimescaleDB", "Neo4j", "Gin"},
		"Go":             {"Gin", "Docker"},
		"API":            {"RESTful", "GraphQL"},
		"存储服务":           {"TimescaleDB", "Neo4j"},
	}

	for _, sourceEntity := range techEntities {
		if usedTechs, exists := techUsageRules[sourceEntity.Name]; exists {
			for _, targetEntity := range techEntities {
				for _, usedTech := range usedTechs {
					if strings.Contains(strings.ToLower(targetEntity.Name), strings.ToLower(usedTech)) {
						relation := &KnowledgeRelationship{
							ID:              fmt.Sprintf("uses_%s_%s_%s", memoryID, sourceEntity.Name, targetEntity.Name),
							SourceEntity:    sourceEntity.Name,
							TargetEntity:    targetEntity.Name,
							RelationType:    RelationUSES,
							Strength:        0.8,
							ConfidenceLevel: 0.85,
							EvidenceText:    fmt.Sprintf("%s使用%s技术", sourceEntity.Name, targetEntity.Name),
							MemoryID:        memoryID,
							SessionID:       req.SessionID,
							UserID:          req.UserID,
							CreatedAt:       time.Now(),
						}
						relations = append(relations, relation)
						log.Printf("🔗 [USES关系] %s -> %s", sourceEntity.Name, targetEntity.Name)
					}
				}
			}
		}
	}

	return relations
}

// buildProjectCompositionRelations 构建项目组成关系
func (s *ContextService) buildProjectCompositionRelations(entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) []*KnowledgeRelationship {
	var relations []*KnowledgeRelationship

	// 查找项目实体和组件实体
	var projectEntities []*KnowledgeEntity
	var componentEntities []*KnowledgeEntity

	for _, entity := range entities {
		if entity.Type == EntityTypeProject {
			projectEntities = append(projectEntities, entity)
		}
		if entity.Type == EntityTypeTechnical || entity.Type == EntityTypeConcept {
			componentEntities = append(componentEntities, entity)
		}
	}

	// 构建项目组成关系
	for _, project := range projectEntities {
		for _, component := range componentEntities {
			// 检查是否在同一个上下文中提及
			if s.areEntitiesRelatedInContext(project.Name, component.Name, req.Content) {
				relation := &KnowledgeRelationship{
					ID:              fmt.Sprintf("composed_%s_%s_%s", memoryID, project.Name, component.Name),
					SourceEntity:    project.Name,
					TargetEntity:    component.Name,
					RelationType:    RelationCOMPOSED_OF,
					Strength:        0.75,
					ConfidenceLevel: 0.8,
					EvidenceText:    fmt.Sprintf("%s包含%s组件", project.Name, component.Name),
					MemoryID:        memoryID,
					SessionID:       req.SessionID,
					UserID:          req.UserID,
					CreatedAt:       time.Now(),
				}
				relations = append(relations, relation)
				log.Printf("🔗 [COMPOSED_OF关系] %s -> %s", project.Name, component.Name)
			}
		}
	}

	return relations
}

// buildProblemSolvingRelations 构建问题解决关系
func (s *ContextService) buildProblemSolvingRelations(entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) []*KnowledgeRelationship {
	var relations []*KnowledgeRelationship

	// 查找问题实体和解决方案实体
	var problemEntities []*KnowledgeEntity
	var solutionEntities []*KnowledgeEntity

	for _, entity := range entities {
		if entity.Type == EntityTypeProblem {
			problemEntities = append(problemEntities, entity)
		}
		if entity.Type == EntityTypeTechnical || entity.Type == EntityTypeProject {
			solutionEntities = append(solutionEntities, entity)
		}
	}

	// 检查解决关系的关键词
	solvingKeywords := []string{"解决", "修复", "优化", "改进", "处理"}

	for _, solution := range solutionEntities {
		for _, problem := range problemEntities {
			// 检查是否存在解决关系的语义
			if s.containsSolvingRelation(req.Content, solution.Name, problem.Name, solvingKeywords) {
				relation := &KnowledgeRelationship{
					ID:              fmt.Sprintf("solves_%s_%s_%s", memoryID, solution.Name, problem.Name),
					SourceEntity:    solution.Name,
					TargetEntity:    problem.Name,
					RelationType:    RelationSOLVES,
					Strength:        0.85,
					ConfidenceLevel: 0.9,
					EvidenceText:    fmt.Sprintf("%s解决了%s", solution.Name, problem.Name),
					MemoryID:        memoryID,
					SessionID:       req.SessionID,
					UserID:          req.UserID,
					CreatedAt:       time.Now(),
				}
				relations = append(relations, relation)
				log.Printf("🔗 [SOLVES关系] %s -> %s", solution.Name, problem.Name)
			}
		}
	}

	return relations
}

// areEntitiesRelatedInContext 检查两个实体是否在上下文中相关
func (s *ContextService) areEntitiesRelatedInContext(entity1, entity2, content string) bool {
	contentLower := strings.ToLower(content)
	entity1Lower := strings.ToLower(entity1)
	entity2Lower := strings.ToLower(entity2)

	// 检查两个实体是否都出现在内容中
	if !strings.Contains(contentLower, entity1Lower) || !strings.Contains(contentLower, entity2Lower) {
		return false
	}

	// 检查两个实体在文本中的距离 (简单的距离计算)
	pos1 := strings.Index(contentLower, entity1Lower)
	pos2 := strings.Index(contentLower, entity2Lower)

	if pos1 == -1 || pos2 == -1 {
		return false
	}

	// 如果两个实体距离在100字符内，认为相关
	distance := pos1 - pos2
	if distance < 0 {
		distance = -distance
	}

	return distance <= 100
}

// containsSolvingRelation 检查是否包含解决关系
func (s *ContextService) containsSolvingRelation(content, solution, problem string, solvingKeywords []string) bool {
	contentLower := strings.ToLower(content)

	// 检查是否包含解决关系的关键词
	for _, keyword := range solvingKeywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			// 进一步检查解决方案和问题是否在附近
			return s.areEntitiesRelatedInContext(solution, problem, content)
		}
	}

	return false
}

// deduplicateEntitiesRuleBased 基于规则去重实体
func (s *ContextService) deduplicateEntitiesRuleBased(entities []*KnowledgeEntity) []*KnowledgeEntity {
	log.Printf("🔧 [实体去重] 开始规则去重，原始实体数: %d", len(entities))

	// 使用map进行去重，key为 name+type 组合
	entityMap := make(map[string]*KnowledgeEntity)

	for _, entity := range entities {
		if entity == nil || entity.Name == "" {
			continue
		}

		// 长度过滤：实体名称不超过20字符
		if len(entity.Name) > 20 {
			log.Printf("⚠️ [实体过滤] 过滤过长实体: %s (%d字符)", entity.Name, len(entity.Name))
			continue
		}

		// 置信度过滤：>= 0.7
		if entity.ConfidenceLevel < 0.7 {
			log.Printf("⚠️ [实体过滤] 过滤低置信度实体: %s (%.2f)", entity.Name, entity.ConfidenceLevel)
			continue
		}

		// 构建去重key
		key := fmt.Sprintf("%s_%s", entity.Name, entity.Type)

		// 如果已存在，保留置信度更高的
		if existing, exists := entityMap[key]; exists {
			if entity.ConfidenceLevel > existing.ConfidenceLevel {
				entityMap[key] = entity
				log.Printf("🔄 [实体去重] 更新实体: %s (置信度: %.2f -> %.2f)",
					entity.Name, existing.ConfidenceLevel, entity.ConfidenceLevel)
			}
		} else {
			entityMap[key] = entity
		}
	}

	// 转换为数组
	var filteredEntities []*KnowledgeEntity
	for _, entity := range entityMap {
		filteredEntities = append(filteredEntities, entity)
	}

	log.Printf("✅ [实体去重] 规则去重完成，过滤后实体数: %d", len(filteredEntities))
	return filteredEntities
}

// KnowledgeEntity 知识实体结构
type KnowledgeEntity struct {
	Name            string                 `json:"name"`
	Type            EntityType             `json:"type"`
	Category        string                 `json:"category"`
	SourceDimension string                 `json:"source_dimension"`
	ConfidenceLevel float64                `json:"confidence_level"`
	Keywords        []string               `json:"keywords"`
	Properties      map[string]interface{} `json:"properties"`
	MemoryID        string                 `json:"memory_id"`
	SessionID       string                 `json:"session_id"`
	UserID          string                 `json:"user_id"`
	CreatedAt       time.Time              `json:"created_at"`
}

// EntityType 实体类型枚举
type EntityType string

const (
	EntityTypeTechnical EntityType = "technical" // 技术实体
	EntityTypeProject   EntityType = "project"   // 项目实体
	EntityTypeConcept   EntityType = "concept"   // 概念实体
	EntityTypeProblem   EntityType = "problem"   // 问题实体
	EntityTypePerson    EntityType = "person"    // 人员实体
)

// extractEntitiesFromText 从单个文本中抽取实体
func (s *ContextService) extractEntitiesFromText(text, dimension string, req models.StoreContextRequest, memoryID string) ([]*KnowledgeEntity, error) {
	log.Printf("🔍 [实体抽取] 从%s维度抽取实体: %s", dimension, text[:min(50, len(text))])

	// 构建实体抽取的专用LLM Prompt
	prompt := s.buildEntityExtractionPrompt(text, dimension, req.Content)

	// 调用LLM进行实体抽取
	llmClient, err := s.createStandardLLMClient(s.config.MultiDimLLMProvider, s.config.MultiDimLLMModel)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   2000,
		Temperature: 0.1, // 低温度确保结果稳定
		Format:      "json",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	llmResponse, err := llmClient.Complete(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM实体抽取失败: %w", err)
	}

	// 解析LLM响应
	entities, err := s.parseEntityExtractionResponse(llmResponse.Content, dimension, req, memoryID)
	if err != nil {
		return nil, fmt.Errorf("解析实体抽取结果失败: %w", err)
	}

	log.Printf("✅ [实体抽取] %s维度抽取完成，获得%d个实体", dimension, len(entities))
	return entities, nil
}

// buildEntityExtractionPrompt 构建实体抽取的LLM Prompt
func (s *ContextService) buildEntityExtractionPrompt(text, dimension, originalContent string) string {
	return fmt.Sprintf(`你是专业的知识图谱实体抽取专家，需要从给定文本中抽取细粒度的实体。

## 🎯 抽取任务
从以下文本中抽取**单个关键词或短语**级别的实体，按照5大类型进行分类。

## 📊 实体分类标准 (严格按照Context-Keeper设计方案)

### 1. 技术实体 (technical)
- **编程语言**: Go, Python, JavaScript, Java, Rust
- **框架工具**: Gin, Neo4j, Docker, Kubernetes, React
- **技术概念**: 微服务, API, 向量数据库, LLM, 机器学习
- **技术方法**: 算法, 数据结构, 设计模式, 架构模式

### 2. 项目实体 (project)  
- **项目名称**: Context-Keeper, 具体模块名, 子系统名
- **组件服务**: 检索引擎, 存储服务, 分析引擎
- **功能模块**: 多维检索, 上下文管理, 会话管理

### 3. 概念实体 (concept)
- **设计模式**: 单例模式, 工厂模式, 观察者模式
- **架构概念**: 分层架构, 事件驱动, 微服务架构
- **业务概念**: 用户会话, 记忆管理, 上下文感知

### 4. 问题实体 (problem)
- **技术问题**: 性能瓶颈, 内存泄漏, 并发问题
- **业务问题**: 需求变更, 用户体验, 功能缺陷
- **实现问题**: 接口设计, 数据一致性, 错误处理

### 5. 人员实体 (person)
- **角色**: 开发者, 架构师, 产品经理, 用户
- **具体人员**: 如果文本中提及具体姓名

## 🔍 抽取规则
1. **粒度控制**: 每个实体应该是单个关键词或短语(≤20字符)
2. **数量限制**: 每个维度最多抽取8-10个实体
3. **质量过滤**: 置信度 ≥ 0.7 的实体才保留
4. **去噪处理**: 剔除停用词、连接词、无意义词汇

## 📝 当前抽取维度
**维度**: %s
**待抽取文本**: %s

## 📋 输出格式
请严格按照以下JSON格式输出：

{
  "entities": [
    {
      "name": "实体名称",
      "type": "technical/project/concept/problem/person",
      "category": "具体分类",
      "confidence_level": 0.8,
      "keywords": ["关键词1", "关键词2"],
      "properties": {
        "source_dimension": "%s",
        "extraction_reason": "抽取原因",
        "context_relevance": 0.9
      }
    }
  ]
}

## 🎯 抽取示例
文本: "我们成功完成了Context-Keeper存储链路的优化工作，修复了P0致命问题，现在TimescaleDB和Neo4j都能真实存储数据了"

抽取结果:
{
  "entities": [
    {
      "name": "Context-Keeper",
      "type": "project",
      "category": "项目名称",
      "confidence_level": 0.95,
      "keywords": ["Context-Keeper", "项目"],
      "properties": {
        "source_dimension": "core_intent",
        "extraction_reason": "明确提及的项目名称",
        "context_relevance": 0.9
      }
    },
    {
      "name": "存储链路",
      "type": "concept",
      "category": "技术概念",
      "confidence_level": 0.9,
      "keywords": ["存储", "链路", "数据流"],
      "properties": {
        "source_dimension": "core_intent",
        "extraction_reason": "核心技术概念",
        "context_relevance": 0.95
      }
    },
    {
      "name": "TimescaleDB",
      "type": "technical",
      "category": "数据库技术",
      "confidence_level": 0.95,
      "keywords": ["TimescaleDB", "时序数据库"],
      "properties": {
        "source_dimension": "domain_context",
        "extraction_reason": "具体技术组件",
        "context_relevance": 0.9
      }
    },
    {
      "name": "Neo4j",
      "type": "technical", 
      "category": "图数据库",
      "confidence_level": 0.95,
      "keywords": ["Neo4j", "图数据库"],
      "properties": {
        "source_dimension": "domain_context",
        "extraction_reason": "具体技术组件",
        "context_relevance": 0.9
      }
    }
  ]
}`, dimension, text, dimension)
}

// parseEntityExtractionResponse 解析实体抽取响应
func (s *ContextService) parseEntityExtractionResponse(response, dimension string, req models.StoreContextRequest, memoryID string) ([]*KnowledgeEntity, error) {
	log.Printf("🔍 [实体解析] 开始解析%s维度的实体抽取结果", dimension)

	// 清理响应格式
	cleanedResponse := s.cleanLLMResponse(response)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	entitiesRaw, exists := result["entities"]
	if !exists {
		return nil, fmt.Errorf("响应中缺少entities字段")
	}

	entitiesList, ok := entitiesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("entities字段格式错误")
	}

	var entities []*KnowledgeEntity
	for _, entityRaw := range entitiesList {
		entityMap, ok := entityRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// 解析实体信息
		entity := &KnowledgeEntity{
			Name:            getStringFromMap(entityMap, "name", ""),
			Type:            EntityType(getStringFromMap(entityMap, "type", "")),
			Category:        getStringFromMap(entityMap, "category", ""),
			SourceDimension: dimension,
			ConfidenceLevel: getFloat64FromMap(entityMap, "confidence_level"),
			MemoryID:        memoryID,
			SessionID:       req.SessionID,
			UserID:          req.UserID,
			CreatedAt:       time.Now(),
		}

		// 解析关键词
		if keywordsRaw, exists := entityMap["keywords"]; exists {
			if keywordsList, ok := keywordsRaw.([]interface{}); ok {
				for _, keyword := range keywordsList {
					if keywordStr, ok := keyword.(string); ok {
						entity.Keywords = append(entity.Keywords, keywordStr)
					}
				}
			}
		}

		// 解析属性
		if propertiesRaw, exists := entityMap["properties"]; exists {
			if propertiesMap, ok := propertiesRaw.(map[string]interface{}); ok {
				entity.Properties = propertiesMap
			}
		}

		// 质量过滤：置信度 >= 0.7
		if entity.ConfidenceLevel >= 0.7 && entity.Name != "" {
			entities = append(entities, entity)
		} else {
			log.Printf("⚠️ [实体过滤] 过滤低质量实体: %s (置信度: %.2f)", entity.Name, entity.ConfidenceLevel)
		}
	}

	log.Printf("✅ [实体解析] %s维度解析完成，有效实体: %d个", dimension, len(entities))
	return entities, nil
}

// buildKnowledgeRelationships 构建实体间的18种关系类型
func (s *ContextService) buildKnowledgeRelationships(entities []*KnowledgeEntity, analysisResult *models.SmartAnalysisResult, req models.StoreContextRequest, memoryID string) ([]*KnowledgeRelationship, error) {
	log.Printf("🔗 [关系构建] 开始构建实体关系，实体数量: %d", len(entities))

	if len(entities) < 2 {
		log.Printf("ℹ️ [关系构建] 实体数量不足，跳过关系构建")
		return []*KnowledgeRelationship{}, nil
	}

	// 构建关系抽取的LLM Prompt
	prompt := s.buildRelationshipExtractionPrompt(entities, analysisResult, req.Content)

	// 调用LLM进行关系抽取
	llmClient, err := s.createStandardLLMClient(s.config.MultiDimLLMProvider, s.config.MultiDimLLMModel)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   3000,
		Temperature: 0.1,
		Format:      "json",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	llmResponse, err := llmClient.Complete(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM关系抽取失败: %w", err)
	}

	// 解析关系响应
	relationships, err := s.parseRelationshipExtractionResponse(llmResponse.Content, entities, req, memoryID)
	if err != nil {
		return nil, fmt.Errorf("解析关系抽取结果失败: %w", err)
	}

	log.Printf("✅ [关系构建] 关系构建完成，关系数量: %d", len(relationships))
	return relationships, nil
}

// KnowledgeRelationship 知识关系结构
type KnowledgeRelationship struct {
	ID              string                 `json:"id"`
	SourceEntity    string                 `json:"source_entity"`
	TargetEntity    string                 `json:"target_entity"`
	RelationType    RelationType           `json:"relation_type"`
	Strength        float64                `json:"strength"`
	ConfidenceLevel float64                `json:"confidence_level"`
	EvidenceText    string                 `json:"evidence_text"`
	Properties      map[string]interface{} `json:"properties"`
	MemoryID        string                 `json:"memory_id"`
	SessionID       string                 `json:"session_id"`
	UserID          string                 `json:"user_id"`
	CreatedAt       time.Time              `json:"created_at"`
}

// RelationType 关系类型枚举 (18种关系类型)
type RelationType string

const (
	// 技术关系 (6种)
	RelationUSES            RelationType = "USES"            // A使用B
	RelationIMPLEMENTS      RelationType = "IMPLEMENTS"      // A实现B
	RelationDEPENDS_ON      RelationType = "DEPENDS_ON"      // A依赖B
	RelationINTEGRATES_WITH RelationType = "INTEGRATES_WITH" // A集成B
	RelationBASED_ON        RelationType = "BASED_ON"        // A基于B
	RelationREPLACED_BY     RelationType = "REPLACED_BY"     // A被B替代

	// 功能关系 (6种)
	RelationSOLVES      RelationType = "SOLVES"      // A解决B
	RelationCAUSES      RelationType = "CAUSES"      // A导致B
	RelationBELONGS_TO  RelationType = "BELONGS_TO"  // A属于B
	RelationAPPLIED_TO  RelationType = "APPLIED_TO"  // A应用于B
	RelationTESTED_IN   RelationType = "TESTED_IN"   // A在B中测试
	RelationDEPLOYED_IN RelationType = "DEPLOYED_IN" // A部署在B

	// 语义关系 (4种)
	RelationRELATED_TO  RelationType = "RELATED_TO"  // A相关B
	RelationSIMILAR_TO  RelationType = "SIMILAR_TO"  // A类似B
	RelationEXTENDS     RelationType = "EXTENDS"     // A扩展B
	RelationCOMPOSED_OF RelationType = "COMPOSED_OF" // A由B组成

	// 协作关系 (2种)
	RelationWORKS_ON  RelationType = "WORKS_ON"  // A工作于B
	RelationEXPERT_IN RelationType = "EXPERT_IN" // A专家于B
)

// buildRelationshipExtractionPrompt 构建关系抽取的LLM Prompt
func (s *ContextService) buildRelationshipExtractionPrompt(entities []*KnowledgeEntity, analysisResult *models.SmartAnalysisResult, originalContent string) string {
	// 构建实体列表
	entityList := ""
	for i, entity := range entities {
		entityList += fmt.Sprintf("%d. %s (%s) - %s\n", i+1, entity.Name, entity.Type, entity.Category)
	}

	return fmt.Sprintf(`你是专业的知识图谱关系分析专家，需要分析实体间的语义关系。

## 🎯 分析任务
基于给定的实体列表和原始文本，识别实体间的语义关系，严格按照18种关系类型进行分类。

## 📊 18种关系类型定义

### 技术关系 (6种)
- **USES**: A使用B (Go语言 USES Gin框架)
- **IMPLEMENTS**: A实现B (Context-Keeper IMPLEMENTS 多维检索)
- **DEPENDS_ON**: A依赖B (检索引擎 DEPENDS_ON 向量数据库)
- **INTEGRATES_WITH**: A集成B (API服务 INTEGRATES_WITH 数据库)
- **BASED_ON**: A基于B (新架构 BASED_ON 微服务模式)
- **REPLACED_BY**: A被B替代 (旧方案 REPLACED_BY 新方案)

### 功能关系 (6种)
- **SOLVES**: A解决B (性能优化 SOLVES 响应缓慢)
- **CAUSES**: A导致B (多用户并发 CAUSES 性能瓶颈)
- **BELONGS_TO**: A属于B (检索引擎 BELONGS_TO Context-Keeper)
- **APPLIED_TO**: A应用于B (优化策略 APPLIED_TO 具体项目)
- **TESTED_IN**: A在B中测试 (新功能 TESTED_IN 测试环境)
- **DEPLOYED_IN**: A部署在B (服务 DEPLOYED_IN 生产环境)

### 语义关系 (4种)
- **RELATED_TO**: A相关B (微服务 RELATED_TO 分布式架构)
- **SIMILAR_TO**: A类似B (Redis SIMILAR_TO 内存数据库)
- **EXTENDS**: A扩展B (新模块 EXTENDS 现有框架)
- **COMPOSED_OF**: A由B组成 (系统 COMPOSED_OF 多个服务)

### 协作关系 (2种)
- **WORKS_ON**: A工作于B (开发者 WORKS_ON 项目)
- **EXPERT_IN**: A专家于B (架构师 EXPERT_IN 微服务设计)

## 🔍 关系强度计算 (多因子加权)
基于以下4个因子计算关系强度 (0.0-1.0):
1. **共现频率** (0.3权重): 实体在同一文本中出现的频率
2. **语义距离** (0.2权重): 实体在文本中的位置距离
3. **关系类型** (0.3权重): 不同关系类型的基础强度
4. **上下文相关性** (0.2权重): 基于上下文的相关性评估

强度范围:
- 0.8-1.0: 强关系 (直接使用、实现关系)
- 0.6-0.8: 中关系 (相关、属于关系)  
- 0.3-0.6: 弱关系 (偶然提及、间接关联)
- <0.3: 过滤掉

## 📋 分析数据

### 实体列表
%s

### 原始文本
%s

## 📋 输出格式
请严格按照以下JSON格式输出：

{
  "relationships": [
    {
      "source_entity": "实体名称1",
      "target_entity": "实体名称2", 
      "relation_type": "USES/IMPLEMENTS/DEPENDS_ON/...",
      "strength": 0.85,
      "confidence_level": 0.9,
      "evidence_text": "支持该关系的原始文本片段",
      "properties": {
        "co_occurrence_frequency": 2,
        "semantic_distance": 0.3,
        "relation_base_strength": 0.8,
        "context_relevance": 0.9
      }
    }
  ]
}

## 🎯 关系识别示例
实体: ["Context-Keeper", "存储链路", "TimescaleDB", "Neo4j", "优化工作"]

可能的关系:
1. Context-Keeper COMPOSED_OF 存储链路 (强度: 0.9)
2. 存储链路 USES TimescaleDB (强度: 0.85)
3. 存储链路 USES Neo4j (强度: 0.85)  
4. 优化工作 APPLIED_TO Context-Keeper (强度: 0.8)
5. 优化工作 SOLVES P0致命问题 (强度: 0.9)

请基于给定的实体和文本，识别所有有意义的关系。`, entityList, originalContent)
}

// parseRelationshipExtractionResponse 解析关系抽取响应
func (s *ContextService) parseRelationshipExtractionResponse(response string, entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) ([]*KnowledgeRelationship, error) {
	log.Printf("🔍 [关系解析] 开始解析关系抽取结果")

	// 清理响应格式
	cleanedResponse := s.cleanLLMResponse(response)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	relationshipsRaw, exists := result["relationships"]
	if !exists {
		return nil, fmt.Errorf("响应中缺少relationships字段")
	}

	relationshipsList, ok := relationshipsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("relationships字段格式错误")
	}

	var relationships []*KnowledgeRelationship
	for _, relRaw := range relationshipsList {
		relMap, ok := relRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// 解析关系信息
		relationship := &KnowledgeRelationship{
			ID:              fmt.Sprintf("rel_%s_%d", memoryID, len(relationships)),
			SourceEntity:    getStringFromMap(relMap, "source_entity", ""),
			TargetEntity:    getStringFromMap(relMap, "target_entity", ""),
			RelationType:    RelationType(getStringFromMap(relMap, "relation_type", "")),
			Strength:        getFloat64FromMap(relMap, "strength"),
			ConfidenceLevel: getFloat64FromMap(relMap, "confidence_level"),
			EvidenceText:    getStringFromMap(relMap, "evidence_text", ""),
			MemoryID:        memoryID,
			SessionID:       req.SessionID,
			UserID:          req.UserID,
			CreatedAt:       time.Now(),
		}

		// 解析属性
		if propertiesRaw, exists := relMap["properties"]; exists {
			if propertiesMap, ok := propertiesRaw.(map[string]interface{}); ok {
				relationship.Properties = propertiesMap
			}
		}

		// 质量过滤：置信度 >= 0.6，强度 >= 0.3
		if relationship.ConfidenceLevel >= 0.6 && relationship.Strength >= 0.3 &&
			relationship.SourceEntity != "" && relationship.TargetEntity != "" {
			relationships = append(relationships, relationship)
		} else {
			log.Printf("⚠️ [关系过滤] 过滤低质量关系: %s -> %s (置信度: %.2f, 强度: %.2f)",
				relationship.SourceEntity, relationship.TargetEntity,
				relationship.ConfidenceLevel, relationship.Strength)
		}
	}

	log.Printf("✅ [关系解析] 关系解析完成，有效关系: %d个", len(relationships))
	return relationships, nil
}

// deduplicateAndFilterEntities 去重和质量过滤实体
func (s *ContextService) deduplicateAndFilterEntities(entities []*KnowledgeEntity) []*KnowledgeEntity {
	log.Printf("🔧 [实体去重] 开始去重和过滤，原始实体数: %d", len(entities))

	// 使用map进行去重，key为 name+type 组合
	entityMap := make(map[string]*KnowledgeEntity)

	for _, entity := range entities {
		if entity == nil || entity.Name == "" {
			continue
		}

		// 长度过滤：实体名称不超过20字符
		if len(entity.Name) > 20 {
			log.Printf("⚠️ [实体过滤] 过滤过长实体: %s (%d字符)", entity.Name, len(entity.Name))
			continue
		}

		// 置信度过滤：>= 0.7
		if entity.ConfidenceLevel < 0.7 {
			log.Printf("⚠️ [实体过滤] 过滤低置信度实体: %s (%.2f)", entity.Name, entity.ConfidenceLevel)
			continue
		}

		// 构建去重key
		key := fmt.Sprintf("%s_%s", entity.Name, entity.Type)

		// 如果已存在，保留置信度更高的
		if existing, exists := entityMap[key]; exists {
			if entity.ConfidenceLevel > existing.ConfidenceLevel {
				entityMap[key] = entity
				log.Printf("🔄 [实体去重] 更新实体: %s (置信度: %.2f -> %.2f)",
					entity.Name, existing.ConfidenceLevel, entity.ConfidenceLevel)
			}
		} else {
			entityMap[key] = entity
		}
	}

	// 转换为数组
	var filteredEntities []*KnowledgeEntity
	for _, entity := range entityMap {
		filteredEntities = append(filteredEntities, entity)
	}

	log.Printf("✅ [实体去重] 去重完成，过滤后实体数: %d", len(filteredEntities))
	return filteredEntities
}

// convertEntitiesToConcepts 将KnowledgeEntity转换为Neo4j Concept格式
func (s *ContextService) convertEntitiesToConcepts(entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) []*knowledge.Concept {
	log.Printf("🔄 [格式转换] 开始转换实体为Neo4j概念格式")

	var concepts []*knowledge.Concept
	for _, entity := range entities {
		concept := &knowledge.Concept{
			Name:        entity.Name,
			Description: fmt.Sprintf("%s实体，来源: %s维度", entity.Category, entity.SourceDimension),
			Category:    string(entity.Type),
			Keywords:    entity.Keywords,
			Importance:  entity.ConfidenceLevel,
			CreatedAt:   entity.CreatedAt,
			UpdatedAt:   entity.CreatedAt,
		}

		// 将扩展信息编码到Description中 (因为Concept模型没有Properties字段)
		concept.Description = fmt.Sprintf("%s实体，来源: %s维度，置信度: %.2f，记忆ID: %s",
			entity.Category, entity.SourceDimension, entity.ConfidenceLevel, entity.MemoryID)

		concepts = append(concepts, concept)
	}

	log.Printf("✅ [格式转换] 实体转换完成，概念数: %d", len(concepts))
	return concepts
}

// convertToNeo4jRelationships 将KnowledgeRelationship转换为Neo4j Relationship格式
func (s *ContextService) convertToNeo4jRelationships(relationships []*KnowledgeRelationship, req models.StoreContextRequest, memoryID string) []*knowledge.Relationship {
	log.Printf("🔄 [格式转换] 开始转换关系为Neo4j关系格式")

	var neo4jRelations []*knowledge.Relationship
	for _, rel := range relationships {
		neo4jRel := &knowledge.Relationship{
			FromName:    rel.SourceEntity,
			ToName:      rel.TargetEntity,
			Type:        string(rel.RelationType),
			Strength:    rel.Strength,
			Description: fmt.Sprintf("关系: %s, 证据: %s", rel.RelationType, rel.EvidenceText[:min(100, len(rel.EvidenceText))]),
			CreatedAt:   rel.CreatedAt,
			UpdatedAt:   rel.CreatedAt,
		}

		// 将扩展信息编码到Description中 (因为Relationship模型没有Properties字段)
		evidenceText := rel.EvidenceText
		if len(evidenceText) > 100 {
			evidenceText = evidenceText[:100] + "..."
		}
		neo4jRel.Description = fmt.Sprintf("关系: %s, 强度: %.2f, 置信度: %.2f, 证据: %s, 记忆ID: %s",
			rel.RelationType, rel.Strength, rel.ConfidenceLevel, evidenceText, rel.MemoryID)

		neo4jRelations = append(neo4jRelations, neo4jRel)
	}

	log.Printf("✅ [格式转换] 关系转换完成，Neo4j关系数: %d", len(neo4jRelations))
	return neo4jRelations
}

// storeMultiDimensionalVectorData 存储多维度向量数据
func (s *ContextService) storeMultiDimensionalVectorData(ctx context.Context, analysisResult interface{}, req models.StoreContextRequest, memoryID string) error {
	log.Printf("🔍 [多维度向量] 开始处理多维度向量数据")

	// 解析分析结果
	resultMap, ok := analysisResult.(map[string]interface{})
	if !ok {
		return fmt.Errorf("分析结果格式错误")
	}

	// 提取vector_data
	vectorDataRaw, exists := resultMap["vector_data"]
	if !exists {
		return fmt.Errorf("分析结果中缺少vector_data字段")
	}

	vectorData, ok := vectorDataRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("vector_data格式错误")
	}

	log.Printf("📊 [多维度向量] 提取到的向量数据: %+v", vectorData)

	// 🔥 多维度向量生成：基于LLM分析结果的不同维度生成多个向量
	var vectors []models.DimensionalVector

	// 1. 内容向量 - 基于精炼的内容
	if content, exists := vectorData["content"].(string); exists && content != "" {
		log.Printf("🔍 [多维度向量] 生成内容向量，内容: %s", content[:min(100, len(content))])
		contentVector, err := s.generateEmbedding(content)
		if err != nil {
			log.Printf("⚠️ [多维度向量] 内容向量生成失败: %v", err)
		} else {
			vectors = append(vectors, models.DimensionalVector{
				Dimension: "content",
				Vector:    contentVector,
				Source:    content,
				Weight:    1.0,
			})
			log.Printf("✅ [多维度向量] 内容向量生成成功，维度: %d", len(contentVector))
		}
	}

	// 2. 语义标签向量 - 基于语义标签
	if tagsRaw, exists := vectorData["semantic_tags"]; exists {
		if tags, ok := tagsRaw.([]interface{}); ok && len(tags) > 0 {
			var tagStrings []string
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					tagStrings = append(tagStrings, tagStr)
				}
			}
			if len(tagStrings) > 0 {
				tagsText := strings.Join(tagStrings, ", ")
				log.Printf("🔍 [多维度向量] 生成语义标签向量，标签: %s", tagsText)
				tagsVector, err := s.generateEmbedding(tagsText)
				if err != nil {
					log.Printf("⚠️ [多维度向量] 语义标签向量生成失败: %v", err)
				} else {
					vectors = append(vectors, models.DimensionalVector{
						Dimension: "semantic_tags",
						Vector:    tagsVector,
						Source:    tagsText,
						Weight:    0.8,
					})
					log.Printf("✅ [多维度向量] 语义标签向量生成成功，维度: %d", len(tagsVector))
				}
			}
		}
	}

	// 3. 上下文摘要向量 - 基于上下文摘要
	if summary, exists := vectorData["context_summary"].(string); exists && summary != "" {
		log.Printf("🔍 [多维度向量] 生成上下文摘要向量，摘要: %s", summary[:min(100, len(summary))])
		summaryVector, err := s.generateEmbedding(summary)
		if err != nil {
			log.Printf("⚠️ [多维度向量] 上下文摘要向量生成失败: %v", err)
		} else {
			vectors = append(vectors, models.DimensionalVector{
				Dimension: "context_summary",
				Vector:    summaryVector,
				Source:    summary,
				Weight:    0.9,
			})
			log.Printf("✅ [多维度向量] 上下文摘要向量生成成功，维度: %d", len(summaryVector))
		}
	}

	if len(vectors) == 0 {
		return fmt.Errorf("没有生成任何维度的向量")
	}

	log.Printf("🎯 [多维度向量] 总共生成了 %d 个维度的向量", len(vectors))

	// 🔥 存储多维度向量到向量数据库
	return s.storeMultiDimensionalVectors(ctx, vectors, req, memoryID)
}

// storeMultiDimensionalVectors 存储多维度向量到向量数据库
func (s *ContextService) storeMultiDimensionalVectors(ctx context.Context, vectors []models.DimensionalVector, req models.StoreContextRequest, memoryID string) error {
	log.Printf("💾 [多维度向量存储] 开始存储 %d 个维度的向量", len(vectors))

	// 🔥 策略1：为每个维度创建独立的记忆对象存储
	for i, dimVector := range vectors {
		log.Printf("📦 [多维度向量存储] 存储第 %d 个维度: %s", i+1, dimVector.Dimension)

		// 为每个维度创建独立的记忆对象
		memory := models.NewMemory(req.SessionID, dimVector.Source, req.Priority, req.Metadata)
		memory.ID = fmt.Sprintf("%s_%s", memoryID, dimVector.Dimension) // 使用维度后缀
		memory.Vector = dimVector.Vector

		// 设置业务类型和用户ID
		if req.BizType > 0 {
			memory.BizType = req.BizType
		}
		if req.UserID != "" {
			memory.UserID = req.UserID
		}

		// 在元数据中标记维度信息
		if memory.Metadata == nil {
			memory.Metadata = make(map[string]interface{})
		}
		memory.Metadata["dimension"] = dimVector.Dimension
		memory.Metadata["dimension_weight"] = dimVector.Weight
		memory.Metadata["original_memory_id"] = memoryID
		memory.Metadata["multi_dimensional"] = true

		// 存储到向量数据库
		if err := s.storeMemory(memory); err != nil {
			log.Printf("❌ [多维度向量存储] 维度 %s 存储失败: %v", dimVector.Dimension, err)
			return fmt.Errorf("维度 %s 存储失败: %w", dimVector.Dimension, err)
		} else {
			log.Printf("✅ [多维度向量存储] 维度 %s 存储成功，ID: %s", dimVector.Dimension, memory.ID)
		}
	}

	// 🔥 策略2：同时存储一个主记忆对象（使用内容维度的向量作为主向量）
	if len(vectors) > 0 {
		log.Printf("📦 [多维度向量存储] 存储主记忆对象")

		// 找到内容维度的向量作为主向量
		var mainVector []float32
		var mainContent string = req.Content

		for _, dimVector := range vectors {
			if dimVector.Dimension == "content" {
				mainVector = dimVector.Vector
				mainContent = dimVector.Source
				break
			}
		}

		// 如果没有内容维度，使用第一个维度
		if mainVector == nil {
			mainVector = vectors[0].Vector
			mainContent = vectors[0].Source
		}

		// 创建主记忆对象
		mainMemory := models.NewMemory(req.SessionID, mainContent, req.Priority, req.Metadata)
		mainMemory.ID = memoryID // 使用原始ID
		mainMemory.Vector = mainVector

		if req.BizType > 0 {
			mainMemory.BizType = req.BizType
		}
		if req.UserID != "" {
			mainMemory.UserID = req.UserID
		}

		// 在元数据中标记多维度信息
		if mainMemory.Metadata == nil {
			mainMemory.Metadata = make(map[string]interface{})
		}
		mainMemory.Metadata["multi_dimensional"] = true
		mainMemory.Metadata["dimension_count"] = len(vectors)
		mainMemory.Metadata["main_dimension"] = "content"

		// 存储主记忆对象
		if err := s.storeMemory(mainMemory); err != nil {
			log.Printf("❌ [多维度向量存储] 主记忆对象存储失败: %v", err)
			return fmt.Errorf("主记忆对象存储失败: %w", err)
		} else {
			log.Printf("✅ [多维度向量存储] 主记忆对象存储成功，ID: %s", mainMemory.ID)
		}
	}

	log.Printf("🎉 [多维度向量存储] 多维度向量存储完成，总计 %d 个维度 + 1 个主对象", len(vectors))
	return nil
}

// storeToMultiDimensionalEngines 并行存储到不同的存储引擎
func (s *ContextService) storeToMultiDimensionalEngines(ctx context.Context, analysisResult interface{}, req models.StoreContextRequest) (string, error) {
	log.Printf("💾 [多维度存储] 开始并行存储到不同引擎")

	// 生成统一的记忆ID（使用UUID格式）
	memoryID := uuid.New().String()
	log.Printf("📊 [多维度存储] 分析结果: %+v", analysisResult)

	// 1. 存储时间线数据到TimescaleDB
	if s.config.MultiDimTimelineEnabled {
		log.Printf("⏰ [时间线存储] 存储时间线数据到TimescaleDB")

		// 🔥 实现真实的TimescaleDB存储（暂时注释，使用新的智能存储）
		// timelineErr := s.storeTimelineDataToTimescaleDB(ctx, analysisResult, req, memoryID)
		timelineErr := fmt.Errorf("旧方法已废弃，使用新的智能存储")
		if timelineErr != nil {
			log.Printf("❌ [时间线存储] TimescaleDB存储失败: %v", timelineErr)
		} else {
			log.Printf("✅ [时间线存储] 时间线数据存储成功: %s", memoryID)
		}
	} else {
		log.Printf("⏰ [时间线存储] 时间线存储已禁用")
	}

	// 2. 存储知识图谱数据到Neo4j
	if s.config.MultiDimKnowledgeEnabled {
		log.Printf("🕸️ [知识图谱存储] 存储知识图谱数据到Neo4j")

		// 🔥 实现真实的Neo4j存储（暂时注释，使用新的智能存储）
		// knowledgeErr := s.storeKnowledgeDataToNeo4j(ctx, analysisResult, req, memoryID)
		knowledgeErr := fmt.Errorf("旧方法已废弃，使用新的智能存储")
		if knowledgeErr != nil {
			log.Printf("❌ [知识图谱存储] Neo4j存储失败: %v", knowledgeErr)
		} else {
			log.Printf("✅ [知识图谱存储] 知识图谱数据存储成功: %s", memoryID)
		}
	} else {
		log.Printf("🕸️ [知识图谱存储] 知识图谱存储已禁用")
	}

	// 3. 存储多维度向量数据 - 🔥 修复：使用LLM分析结果中的多维度向量数据
	if s.config.MultiDimVectorEnabled {
		log.Printf("🔍 [向量存储] 存储多维度向量数据到向量数据库")

		// 🔥 从LLM分析结果中提取向量数据并进行多维度向量生成
		err := s.storeMultiDimensionalVectorData(ctx, analysisResult, req, memoryID)
		if err != nil {
			log.Printf("❌ [向量存储] 多维度向量存储失败: %v，降级到单一向量存储", err)

			// 降级：使用原始内容生成单一向量
			memory := models.NewMemory(req.SessionID, req.Content, req.Priority, req.Metadata)
			memory.ID = memoryID
			if req.BizType > 0 {
				memory.BizType = req.BizType
			}
			if req.UserID != "" {
				memory.UserID = req.UserID
			}

			vector, vectorErr := s.generateEmbedding(req.Content)
			if vectorErr != nil {
				log.Printf("❌ [向量存储] 降级向量生成也失败: %v", vectorErr)
			} else {
				memory.Vector = vector
				if storeErr := s.storeMemory(memory); storeErr != nil {
					log.Printf("❌ [向量存储] 降级向量存储失败: %v", storeErr)
				} else {
					log.Printf("✅ [向量存储] 降级向量存储成功: %s", memoryID)
				}
			}
		} else {
			log.Printf("✅ [向量存储] 多维度向量数据存储成功: %s", memoryID)
		}
	} else {
		log.Printf("🔍 [向量存储] 向量存储已禁用")
	}

	// 更新会话信息
	if err := s.sessionStore.UpdateSession(req.SessionID, req.Content); err != nil {
		log.Printf("⚠️ [多维度存储] 更新会话信息失败: %v", err)
		// 继续执行，不返回错误
	}

	log.Printf("🎉 [多维度存储] 多维度存储完成: memoryID=%s", memoryID)
	return memoryID, nil
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
			log.Printf("⚠️ [上下文服务] 生成查询向量失败: %v，降级到会话ID检索", err)
			// 降级到会话ID检索
			searchResults, err = s.searchBySessionID(ctx, req.SessionID, req.Limit)
			if err != nil {
				return models.ContextResponse{}, fmt.Errorf("降级检索失败: %w", err)
			}
			log.Printf("[上下文服务] 降级检索耗时: %v", time.Since(startTime))
		} else {
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
		}
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

// buildConceptRelatedRelations 构建概念相关关系
func (s *ContextService) buildConceptRelatedRelations(entities []*KnowledgeEntity, req models.StoreContextRequest, memoryID string) []*KnowledgeRelationship {
	var relations []*KnowledgeRelationship

	// 查找概念实体
	var conceptEntities []*KnowledgeEntity
	for _, entity := range entities {
		if entity.Type == EntityTypeConcept {
			conceptEntities = append(conceptEntities, entity)
		}
	}

	// 构建概念间的相关关系
	for i, concept1 := range conceptEntities {
		for j, concept2 := range conceptEntities {
			if i >= j { // 避免重复和自关联
				continue
			}

			// 检查两个概念是否在上下文中相关
			if s.areEntitiesRelatedInContext(concept1.Name, concept2.Name, req.Content) {
				relation := &KnowledgeRelationship{
					ID:              fmt.Sprintf("related_%s_%s_%s", memoryID, concept1.Name, concept2.Name),
					SourceEntity:    concept1.Name,
					TargetEntity:    concept2.Name,
					RelationType:    RelationRELATED_TO,
					Strength:        0.7,
					ConfidenceLevel: 0.75,
					EvidenceText:    fmt.Sprintf("%s与%s相关", concept1.Name, concept2.Name),
					MemoryID:        memoryID,
					SessionID:       req.SessionID,
					UserID:          req.UserID,
					CreatedAt:       time.Now(),
				}
				relations = append(relations, relation)
				log.Printf("🔗 [RELATED_TO关系] %s -> %s", concept1.Name, concept2.Name)
			}
		}
	}

	return relations
}

// filterLowQualityRelationships 过滤低质量关系
func (s *ContextService) filterLowQualityRelationships(relationships []*KnowledgeRelationship) []*KnowledgeRelationship {
	var filtered []*KnowledgeRelationship

	for _, rel := range relationships {
		// 过滤条件：置信度 >= 0.6，强度 >= 0.3
		if rel.ConfidenceLevel >= 0.6 && rel.Strength >= 0.3 {
			filtered = append(filtered, rel)
		} else {
			log.Printf("⚠️ [关系过滤] 过滤低质量关系: %s -> %s (置信度: %.2f, 强度: %.2f)",
				rel.SourceEntity, rel.TargetEntity, rel.ConfidenceLevel, rel.Strength)
		}
	}

	return filtered
}
