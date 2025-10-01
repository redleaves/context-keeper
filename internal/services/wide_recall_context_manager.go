package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// WideRecallContextManager 宽召回上下文管理器
type WideRecallContextManager struct {
	// === 核心服务 ===
	wideRecallService *WideRecallService // 宽召回服务

	// === 内存存储 ===
	sessionContexts map[string]*models.UnifiedContextModel // 会话上下文缓存
	mu              sync.RWMutex                           // 读写锁

	// === 配置 ===
	config *WideRecallContextConfig // 配置

	// === 生命周期管理 ===
	stopChan chan struct{} // 停止信号
}

// WideRecallContextConfig 宽召回上下文管理器配置
type WideRecallContextConfig struct {
	// === 置信度阈值 ===
	MemoryThreshold      float64 `json:"memory_threshold"`      // 内存更新阈值
	PersistenceThreshold float64 `json:"persistence_threshold"` // 持久化阈值

	// === 缓存配置 ===
	MaxCacheSize    int           `json:"max_cache_size"`   // 最大缓存大小
	CacheExpiry     time.Duration `json:"cache_expiry"`     // 缓存过期时间
	CleanupInterval time.Duration `json:"cleanup_interval"` // 清理间隔

	// === 性能配置 ===
	MaxConcurrency int `json:"max_concurrency"` // 最大并发数
}

// NewWideRecallContextManager 创建宽召回上下文管理器
func NewWideRecallContextManager(
	wideRecallService *WideRecallService,
	config *WideRecallContextConfig,
) *WideRecallContextManager {
	if config == nil {
		config = getDefaultWideRecallContextConfig()
	}

	manager := &WideRecallContextManager{
		wideRecallService: wideRecallService,
		sessionContexts:   make(map[string]*models.UnifiedContextModel),
		config:            config,
		stopChan:          make(chan struct{}),
	}

	// 启动定期清理
	go manager.startPeriodicCleanup()

	return manager
}

// UpdateContextWithWideRecall 使用宽召回更新上下文
func (wrcm *WideRecallContextManager) UpdateContextWithWideRecall(req *models.ContextUpdateRequest) (*models.ContextUpdateResponse, error) {
	startTime := time.Now()
	log.Printf("🔄 [宽召回上下文] 开始处理，会话ID: %s", req.SessionID)

	// === 阶段1: 获取当前上下文 ===
	currentContext := wrcm.getFromMemory(req.SessionID)
	if currentContext == nil {
		log.Printf("🆕 [宽召回上下文] 首次创建上下文")
		return wrcm.initializeContextWithWideRecall(req)
	}

	// === 阶段2: 执行宽召回检索 ===
	wideRecallReq := &models.WideRecallRequest{
		UserID:         req.UserID,
		SessionID:      req.SessionID,
		WorkspaceID:    req.WorkspaceID,
		UserQuery:      req.UserQuery,
		IntentAnalysis: nil, // 将由宽召回服务内部进行意图分析
		RetrievalConfig: &models.RetrievalConfig{
			TimelineTimeout:     5,
			KnowledgeTimeout:    5,
			VectorTimeout:       5,
			TimelineMaxResults:  20,
			KnowledgeMaxResults: 15,
			VectorMaxResults:    25,
			MinSimilarityScore:  0.6,
			MinRelevanceScore:   0.5,
			MaxRetries:          1,
			RetryInterval:       2,
		},
		RequestTime: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	wideRecallResp, err := wrcm.wideRecallService.ExecuteWideRecall(ctx, wideRecallReq)
	if err != nil {
		log.Printf("❌ [宽召回上下文] 宽召回检索失败: %v", err)
		// 降级到原有流程
		return &models.ContextUpdateResponse{
			Success:         false,
			UpdatedContext:  currentContext,
			UpdateSummary:   fmt.Sprintf("宽召回失败，降级处理: %v", err),
			ConfidenceLevel: 0.3,
			ProcessingTime:  time.Since(startTime).Milliseconds(),
		}, nil
	}

	log.Printf("📊 [宽召回上下文] 检索完成 - 总结果: %d", wideRecallResp.RetrievalResults.TotalResults)

	// === 阶段3: 执行上下文合成 ===
	synthesisReq := &models.ContextSynthesisRequest{
		UserID:           req.UserID,
		SessionID:        req.SessionID,
		WorkspaceID:      req.WorkspaceID,
		UserQuery:        req.UserQuery,
		IntentAnalysis:   nil, // 意图分析将在上下文合成过程中进行
		CurrentContext:   currentContext,
		RetrievalResults: wideRecallResp.RetrievalResults,
		SynthesisConfig: &models.SynthesisConfig{
			LLMTimeout:           40,
			MaxTokens:            4096,
			Temperature:          0.2,
			ConfidenceThreshold:  0.7,
			ConflictResolution:   "time_priority",
			InformationFusion:    "weighted_merge",
			QualityAssessment:    "comprehensive",
			UpdateThreshold:      0.4,
			PersistenceThreshold: 0.7,
		},
		RequestTime: time.Now(),
	}

	synthesisResp, err := wrcm.wideRecallService.ExecuteContextSynthesis(ctx, synthesisReq)
	if err != nil {
		log.Printf("❌ [宽召回上下文] 上下文合成失败: %v", err)
		return &models.ContextUpdateResponse{
			Success:         false,
			UpdatedContext:  currentContext,
			UpdateSummary:   fmt.Sprintf("上下文合成失败: %v", err),
			ConfidenceLevel: 0.3,
			ProcessingTime:  time.Since(startTime).Milliseconds(),
		}, nil
	}

	// === 阶段4: 根据评估结果更新上下文 ===
	if synthesisResp.EvaluationResult == nil {
		log.Printf("❌ [宽召回上下文] EvaluationResult为nil，使用降级方案")
		return &models.ContextUpdateResponse{
			Success:         false,
			UpdatedContext:  currentContext,
			UpdateSummary:   "评估结果为空，无法更新上下文",
			ConfidenceLevel: 0.3,
			ProcessingTime:  time.Since(startTime).Milliseconds(),
		}, nil
	}

	if synthesisResp.EvaluationResult.ShouldUpdate {
		updatedContext := synthesisResp.SynthesizedContext
		updatedContext.SessionID = req.SessionID
		updatedContext.UserID = req.UserID
		updatedContext.WorkspaceID = req.WorkspaceID
		updatedContext.UpdatedAt = time.Now()

		// 更新内存中的上下文
		wrcm.updateMemory(req.SessionID, updatedContext)

		// 如果置信度足够高，考虑持久化
		if synthesisResp.EvaluationResult.UpdateConfidence >= wrcm.config.PersistenceThreshold {
			go wrcm.persistContextAsync(updatedContext)
		}

		log.Printf("✅ [宽召回上下文] 成功更新，置信度: %.2f", synthesisResp.EvaluationResult.UpdateConfidence)

		return &models.ContextUpdateResponse{
			Success:         true,
			UpdatedContext:  updatedContext,
			UpdateSummary:   synthesisResp.EvaluationResult.EvaluationReason,
			ConfidenceLevel: synthesisResp.EvaluationResult.UpdateConfidence,
			ProcessingTime:  time.Since(startTime).Milliseconds(),
		}, nil
	}

	// 无需更新的情况
	log.Printf("ℹ️ [宽召回上下文] 无需更新，置信度: %.2f", synthesisResp.EvaluationResult.UpdateConfidence)

	return &models.ContextUpdateResponse{
		Success:         true,
		UpdatedContext:  currentContext,
		UpdateSummary:   "无需更新上下文",
		ConfidenceLevel: synthesisResp.EvaluationResult.UpdateConfidence,
		ProcessingTime:  time.Since(startTime).Milliseconds(),
	}, nil
}

// initializeContextWithWideRecall 使用宽召回初始化上下文
func (wrcm *WideRecallContextManager) initializeContextWithWideRecall(req *models.ContextUpdateRequest) (*models.ContextUpdateResponse, error) {
	startTime := time.Now()
	log.Printf("🆕 [宽召回上下文] 开始初始化，会话ID: %s", req.SessionID)

	// === 阶段1: 执行宽召回检索 ===
	wideRecallReq := &models.WideRecallRequest{
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		WorkspaceID: req.WorkspaceID,
		UserQuery:   req.UserQuery,
		RetrievalConfig: &models.RetrievalConfig{
			TimelineTimeout:     5,
			KnowledgeTimeout:    5,
			VectorTimeout:       5,
			TimelineMaxResults:  20,
			KnowledgeMaxResults: 15,
			VectorMaxResults:    25,
			MinSimilarityScore:  0.6,
			MinRelevanceScore:   0.5,
			MaxRetries:          1,
			RetryInterval:       2,
		},
		RequestTime: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	wideRecallResp, err := wrcm.wideRecallService.ExecuteWideRecall(ctx, wideRecallReq)
	if err != nil {
		log.Printf("❌ [宽召回上下文] 初始化时宽召回失败: %v", err)
		// 创建基础上下文
		return wrcm.createBasicContext(req, startTime)
	}

	// === 阶段2: 执行上下文合成（首次构建）===
	synthesisReq := &models.ContextSynthesisRequest{
		UserID:           req.UserID,
		SessionID:        req.SessionID,
		WorkspaceID:      req.WorkspaceID,
		UserQuery:        req.UserQuery,
		IntentAnalysis:   nil, // 意图分析将在上下文合成过程中进行
		CurrentContext:   nil, // 首次构建，无现有上下文
		RetrievalResults: wideRecallResp.RetrievalResults,
		SynthesisConfig: &models.SynthesisConfig{
			LLMTimeout:           40,
			MaxTokens:            4096,
			Temperature:          0.2,
			ConfidenceThreshold:  0.7,
			ConflictResolution:   "time_priority",
			InformationFusion:    "weighted_merge",
			QualityAssessment:    "comprehensive",
			UpdateThreshold:      0.4,
			PersistenceThreshold: 0.7,
		},
		RequestTime: time.Now(),
	}

	synthesisResp, err := wrcm.wideRecallService.ExecuteContextSynthesis(ctx, synthesisReq)
	if err != nil {
		log.Printf("❌ [宽召回上下文] 初始化时上下文合成失败: %v", err)
		return wrcm.createBasicContext(req, startTime)
	}

	// === 阶段3: 创建新的上下文 ===
	if synthesisResp.SynthesizedContext == nil {
		log.Printf("❌ [宽召回上下文] SynthesizedContext为nil，使用降级方案")
		return wrcm.createBasicContext(req, startTime)
	}

	newContext := synthesisResp.SynthesizedContext
	newContext.SessionID = req.SessionID
	newContext.UserID = req.UserID
	newContext.WorkspaceID = req.WorkspaceID
	newContext.CreatedAt = time.Now()
	newContext.UpdatedAt = time.Now()

	// 存储到内存
	wrcm.updateMemory(req.SessionID, newContext)

	// 如果置信度足够高，持久化
	if synthesisResp.EvaluationResult.UpdateConfidence >= wrcm.config.PersistenceThreshold {
		go wrcm.persistContextAsync(newContext)
	}

	log.Printf("✅ [宽召回上下文] 初始化完成，置信度: %.2f", synthesisResp.EvaluationResult.UpdateConfidence)

	return &models.ContextUpdateResponse{
		Success:         true,
		UpdatedContext:  newContext,
		UpdateSummary:   "使用宽召回成功初始化上下文",
		ConfidenceLevel: synthesisResp.EvaluationResult.UpdateConfidence,
		ProcessingTime:  time.Since(startTime).Milliseconds(),
	}, nil
}

// createBasicContext 创建基础上下文（降级方案）
func (wrcm *WideRecallContextManager) createBasicContext(req *models.ContextUpdateRequest, startTime time.Time) (*models.ContextUpdateResponse, error) {
	log.Printf("🔧 [宽召回上下文] 创建基础上下文作为降级方案")

	// 创建基础的上下文模型
	basicContext := &models.UnifiedContextModel{
		SessionID:   req.SessionID,
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
		CurrentTopic: &models.TopicContext{
			MainTopic:     extractMainTopicFromQuery(req.UserQuery),
			TopicCategory: models.TopicCategoryTechnical, // 默认为技术类
			UserIntent: models.UserIntent{
				IntentType:        models.IntentQuery, // 默认为查询类型
				IntentDescription: req.UserQuery,
				Priority:          models.PriorityMedium,
			},
			PrimaryPainPoint: "需要更多信息来理解具体问题",
			ExpectedOutcome:  "获得相关的技术支持或信息",
			KeyConcepts:      extractKeyConceptsFromQuery(req.UserQuery),
			TopicStartTime:   time.Now(),
			LastUpdated:      time.Now(),
			UpdateCount:      1,
			ConfidenceLevel:  0.3, // 低置信度
		},
		Project: &models.ProjectContext{
			ProjectName:     "未知项目",
			ProjectPath:     req.WorkspaceID,
			ProjectType:     models.ProjectTypeOther,
			Description:     "项目信息待分析",
			PrimaryLanguage: "unknown",
			CurrentPhase:    models.ProjectPhaseDevelopment,
			ConfidenceLevel: 0.2,
		},
		RecentChangesSummary: "",
		Code: &models.CodeContext{
			SessionID:         req.SessionID,
			ActiveFiles:       []models.ActiveFileInfo{},
			RecentEdits:       []models.ContextEditInfo{},
			FocusedComponents: []string{},
			KeyFunctions:      []models.FunctionInfo{},
			ImportantTypes:    []models.TypeInfo{},
		},
		Conversation: nil, // ConversationContext 暂未定义，设为nil
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 存储到内存
	wrcm.updateMemory(req.SessionID, basicContext)

	return &models.ContextUpdateResponse{
		Success:         true,
		UpdatedContext:  basicContext,
		UpdateSummary:   "创建基础上下文（降级方案）",
		ConfidenceLevel: 0.3,
		ProcessingTime:  time.Since(startTime).Milliseconds(),
	}, nil
}

// 内存管理方法
func (wrcm *WideRecallContextManager) getFromMemory(sessionID string) *models.UnifiedContextModel {
	wrcm.mu.RLock()
	defer wrcm.mu.RUnlock()
	return wrcm.sessionContexts[sessionID]
}

func (wrcm *WideRecallContextManager) updateMemory(sessionID string, context *models.UnifiedContextModel) {
	wrcm.mu.Lock()
	defer wrcm.mu.Unlock()
	wrcm.sessionContexts[sessionID] = context
}

func (wrcm *WideRecallContextManager) removeFromMemory(sessionID string) {
	wrcm.mu.Lock()
	defer wrcm.mu.Unlock()
	delete(wrcm.sessionContexts, sessionID)
}

// persistContextAsync 异步持久化上下文
func (wrcm *WideRecallContextManager) persistContextAsync(context *models.UnifiedContextModel) {
	// TODO: 实现持久化逻辑
	log.Printf("💾 [宽召回上下文] 异步持久化上下文，会话ID: %s", context.SessionID)
}

// startPeriodicCleanup 启动定期清理
func (wrcm *WideRecallContextManager) startPeriodicCleanup() {
	ticker := time.NewTicker(wrcm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wrcm.cleanupExpiredContexts()
		case <-wrcm.stopChan:
			return
		}
	}
}

// cleanupExpiredContexts 清理过期的上下文
func (wrcm *WideRecallContextManager) cleanupExpiredContexts() {
	wrcm.mu.Lock()
	defer wrcm.mu.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, context := range wrcm.sessionContexts {
		if now.Sub(context.UpdatedAt) > wrcm.config.CacheExpiry {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		delete(wrcm.sessionContexts, sessionID)
		log.Printf("🧹 [宽召回上下文] 清理过期上下文，会话ID: %s", sessionID)
	}

	if len(expiredSessions) > 0 {
		log.Printf("🧹 [宽召回上下文] 清理完成，删除 %d 个过期上下文", len(expiredSessions))
	}
}

// Stop 停止上下文管理器
func (wrcm *WideRecallContextManager) Stop() {
	close(wrcm.stopChan)
	log.Printf("🛑 [宽召回上下文] 上下文管理器已停止")
}

// 辅助函数
func extractMainTopicFromQuery(query string) string {
	// 简单的主题提取逻辑
	if len(query) > 50 {
		return query[:50] + "..."
	}
	return query
}

func extractKeyConceptsFromQuery(query string) []models.ConceptInfo {
	// 简单的关键概念提取
	return []models.ConceptInfo{
		{
			ConceptName: "用户查询",
			ConceptType: models.ConceptTypeTechnical,
			Definition:  query,
			Importance:  0.8,
			Source:      "user_input",
		},
	}
}

// getDefaultWideRecallContextConfig 获取默认配置
func getDefaultWideRecallContextConfig() *WideRecallContextConfig {
	return &WideRecallContextConfig{
		MemoryThreshold:      0.4,
		PersistenceThreshold: 0.7,
		MaxCacheSize:         1000,
		CacheExpiry:          30 * time.Minute,
		CleanupInterval:      5 * time.Minute,
		MaxConcurrency:       10,
	}
}
