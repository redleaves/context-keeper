package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/lib/pq"
)

// UnifiedContextManager 统一上下文管理器实现
type UnifiedContextManager struct {
	// 内存存储（按SessionID索引）
	sessionContexts map[string]*models.UnifiedContextModel
	mutex           sync.RWMutex

	// 依赖服务
	contextService *ContextService
	sessionManager *store.SessionStore
	llmService     LLMService

	// 配置
	memoryThreshold float64
	maxContextAge   time.Duration
	cleanupInterval time.Duration

	// 生命周期管理
	stopChan      chan struct{}
	cleanupTicker *time.Ticker
}

// 注意：LLMService 已在 interfaces.go 中定义，这里直接使用

// NewUnifiedContextManager 创建新的统一上下文管理器
func NewUnifiedContextManager(
	contextService *ContextService,
	sessionManager *store.SessionStore,
	llmService LLMService,
) *UnifiedContextManager {
	ucm := &UnifiedContextManager{
		sessionContexts: make(map[string]*models.UnifiedContextModel),
		contextService:  contextService,
		sessionManager:  sessionManager,
		llmService:      llmService,
		memoryThreshold: 0.7,            // 默认阈值
		maxContextAge:   24 * time.Hour, // 24小时过期
		cleanupInterval: 1 * time.Hour,  // 每小时清理一次
		stopChan:        make(chan struct{}),
	}

	// 启动定期清理
	ucm.startCleanupRoutine()

	log.Printf("✅ [统一上下文管理器] 初始化完成，内存阈值: %.2f", ucm.memoryThreshold)
	return ucm
}

// GetContext 获取上下文
func (ucm *UnifiedContextManager) GetContext(sessionID string) (*models.UnifiedContextModel, error) {
	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()

	context, exists := ucm.sessionContexts[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话上下文不存在: %s", sessionID)
	}

	log.Printf("📖 [上下文获取] 会话ID: %s, 上下文年龄: %v",
		sessionID, time.Since(context.UpdatedAt))

	return context, nil
}

// UpdateContext 更新上下文
func (ucm *UnifiedContextManager) UpdateContext(req *models.ContextUpdateRequest) (*models.ContextUpdateResponse, error) {
	// TODO: 重新实现，集成宽召回服务
	return &models.ContextUpdateResponse{
		Success:         false,
		UpdatedContext:  nil,
		UpdateSummary:   "宽召回服务正在开发中",
		ConfidenceLevel: 0.0,
		ProcessingTime:  0,
	}, fmt.Errorf("宽召回服务正在开发中")
}

func (ucm *UnifiedContextManager) UpdateContextWithWideRecallOld(req *models.ContextUpdateRequest) (*models.ContextUpdateResponse, error) {
	startTime := time.Now()

	log.Printf("🔄 [上下文更新] 开始处理，会话ID: %s, 查询: %s",
		req.SessionID, truncateString(req.UserQuery, 50))

	// === 阶段1: 获取当前上下文 ===
	currentContext := ucm.getFromMemory(req.SessionID)
	if currentContext == nil {
		log.Printf("🆕 [上下文更新] 首次创建上下文，会话ID: %s", req.SessionID)
		return ucm.initializeContext(req)
	}

	// === 阶段2: 意图分析和宽召回准备 ===
	intentAnalysis, err := ucm.llmService.AnalyzeUserIntent(req.UserQuery)
	if err != nil {
		log.Printf("❌ [上下文更新] 意图分析失败: %v", err)
		return nil, fmt.Errorf("意图分析失败: %w", err)
	}

	log.Printf("🎯 [意图分析] 核心意图: %s, 场景: %s",
		intentAnalysis.CoreIntentText, intentAnalysis.ScenarioText)

	// === 阶段3: 并行宽召回检索 ===
	searchQueries := ucm.generateSearchQueries(intentAnalysis, req.UserQuery)
	retrievalResults, err := ucm.parallelWideRecall(searchQueries, req.UserID, req.WorkspaceID)
	if err != nil {
		log.Printf("⚠️ [上下文更新] 宽召回检索失败，继续处理: %v", err)
		// 创建空的检索结果，不中断流程
		retrievalResults = &models.ParallelRetrievalResult{}
	}

	log.Printf("🔍 [宽召回] 检索结果: 时间线%d条, 知识图谱%d条, 向量%d条",
		retrievalResults.TimelineCount, retrievalResults.KnowledgeCount, retrievalResults.VectorCount)

	// === 阶段4: LLM驱动的上下文合成与评估（一体化）===
	synthesisResult, err := ucm.llmService.SynthesizeAndEvaluateContext(
		req.UserQuery,
		currentContext,
		retrievalResults,
		intentAnalysis,
	)
	if err != nil {
		log.Printf("❌ [上下文更新] 上下文合成评估失败: %v", err)
		return nil, fmt.Errorf("上下文合成评估失败: %w", err)
	}

	log.Printf("📊 [合成评估] 是否更新: %t, 置信度: %.2f, 原因: %s",
		synthesisResult.ShouldUpdate, synthesisResult.UpdateConfidence,
		truncateString(synthesisResult.EvaluationReason, 100))

	// === 阶段5: 根据评估结果决定更新策略 ===
	processingTime := time.Since(startTime).Milliseconds()

	if synthesisResult.ShouldUpdate {
		if synthesisResult.UpdateConfidence >= ucm.memoryThreshold {
			// 高置信度：更新内存 + 持久化
			updatedContext := synthesisResult.UpdatedContext
			updatedContext.UpdatedAt = time.Now()

			ucm.updateMemory(req.SessionID, updatedContext)

			// 并行持久化（如果需要长期记忆存储）
			go ucm.persistContextIfNeeded(updatedContext, synthesisResult.UpdateConfidence)

			log.Printf("✅ [上下文更新] 高置信度更新完成，会话ID: %s, 置信度: %.2f",
				req.SessionID, synthesisResult.UpdateConfidence)

			return &models.ContextUpdateResponse{
				Success:         true,
				UpdatedContext:  updatedContext,
				UpdateSummary:   synthesisResult.EvaluationReason,
				ConfidenceLevel: synthesisResult.UpdateConfidence,
				ProcessingTime:  processingTime,
			}, nil
		} else {
			// 低置信度：仅更新内存中的临时信息
			ucm.updateTemporaryInfo(currentContext, synthesisResult)

			log.Printf("⚠️ [上下文更新] 低置信度更新，会话ID: %s, 置信度: %.2f",
				req.SessionID, synthesisResult.UpdateConfidence)

			return &models.ContextUpdateResponse{
				Success:         true,
				UpdatedContext:  currentContext,
				UpdateSummary:   fmt.Sprintf("低置信度更新(%.2f)，仅更新临时信息", synthesisResult.UpdateConfidence),
				ConfidenceLevel: synthesisResult.UpdateConfidence,
				ProcessingTime:  processingTime,
			}, nil
		}
	}

	log.Printf("ℹ️ [上下文更新] 无需更新，会话ID: %s, 置信度: %.2f",
		req.SessionID, synthesisResult.UpdateConfidence)

	return &models.ContextUpdateResponse{
		Success:         true,
		UpdatedContext:  currentContext,
		UpdateSummary:   fmt.Sprintf("无需更新上下文，置信度: %.2f", synthesisResult.UpdateConfidence),
		ConfidenceLevel: synthesisResult.UpdateConfidence,
		ProcessingTime:  processingTime,
	}, nil
}

// CleanupContext 清理上下文（会话结束时）
func (ucm *UnifiedContextManager) CleanupContext(sessionID string) error {
	ucm.mutex.Lock()
	defer ucm.mutex.Unlock()

	if _, exists := ucm.sessionContexts[sessionID]; exists {
		delete(ucm.sessionContexts, sessionID)
		log.Printf("🧹 [上下文清理] 清理会话上下文: %s", sessionID)
		return nil
	}

	return fmt.Errorf("会话上下文不存在: %s", sessionID)
}

// getFromMemory 从内存获取上下文
func (ucm *UnifiedContextManager) getFromMemory(sessionID string) *models.UnifiedContextModel {
	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()

	return ucm.sessionContexts[sessionID]
}

// UpdateMemory 更新内存中的上下文（公开方法）
func (ucm *UnifiedContextManager) UpdateMemory(sessionID string, context *models.UnifiedContextModel) {
	ucm.mutex.Lock()
	defer ucm.mutex.Unlock()

	ucm.sessionContexts[sessionID] = context
}

// updateMemory 更新内存中的上下文（私有方法，保持兼容性）
func (ucm *UnifiedContextManager) updateMemory(sessionID string, context *models.UnifiedContextModel) {
	ucm.UpdateMemory(sessionID, context)
}

// startCleanupRoutine 启动定期清理例程
func (ucm *UnifiedContextManager) startCleanupRoutine() {
	ucm.cleanupTicker = time.NewTicker(ucm.cleanupInterval)

	go func() {
		for {
			select {
			case <-ucm.cleanupTicker.C:
				ucm.performCleanup()
			case <-ucm.stopChan:
				ucm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// performCleanup 执行清理
func (ucm *UnifiedContextManager) performCleanup() {
	ucm.mutex.Lock()
	defer ucm.mutex.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, context := range ucm.sessionContexts {
		if now.Sub(context.UpdatedAt) > ucm.maxContextAge {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		delete(ucm.sessionContexts, sessionID)
		log.Printf("🧹 [定期清理] 清理过期上下文: %s", sessionID)
	}

	if len(expiredSessions) > 0 {
		log.Printf("🧹 [定期清理] 清理了 %d 个过期上下文", len(expiredSessions))
	}
}

// Stop 停止上下文管理器
func (ucm *UnifiedContextManager) Stop() {
	close(ucm.stopChan)
	log.Printf("🛑 [统一上下文管理器] 已停止")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// initializeContext 初始化上下文
func (ucm *UnifiedContextManager) initializeContext(req *models.ContextUpdateRequest) (*models.ContextUpdateResponse, error) {
	log.Printf("🆕 [上下文初始化] 开始初始化，会话ID: %s", req.SessionID)

	// === 阶段1: 意图分析和宽召回准备 ===
	intentAnalysis, err := ucm.llmService.AnalyzeUserIntent(req.UserQuery)
	if err != nil {
		log.Printf("❌ [上下文初始化] 意图分析失败: %v", err)
		return nil, fmt.Errorf("意图分析失败: %w", err)
	}

	// === 阶段2: 并行宽召回检索 ===
	searchQueries := ucm.generateSearchQueries(intentAnalysis, req.UserQuery)
	retrievalResults, err := ucm.parallelWideRecall(searchQueries, req.UserID, req.WorkspaceID)
	if err != nil {
		log.Printf("⚠️ [上下文初始化] 宽召回检索失败，继续处理: %v", err)
		retrievalResults = &models.ParallelRetrievalResult{}
	}

	// === 阶段3: 项目信息分析（从workspace路径）===
	projectInfo := ucm.analyzeProjectFromWorkspace(req.WorkspaceID)

	// === 阶段4: LLM驱动的上下文合成与评估（一体化）===
	synthesisResult, err := ucm.llmService.SynthesizeAndEvaluateContext(
		req.UserQuery,
		nil, // 首次构建，无现有上下文
		retrievalResults,
		intentAnalysis,
	)
	if err != nil {
		log.Printf("❌ [上下文初始化] 上下文合成失败: %v", err)
		return nil, fmt.Errorf("上下文合成失败: %w", err)
	}

	// === 阶段5: 构建初始上下文模型 ===
	context := synthesisResult.UpdatedContext
	if context == nil {
		// 如果LLM没有返回上下文，创建基础上下文
		context = ucm.createBasicContext(req.SessionID, req.UserID, req.WorkspaceID, intentAnalysis, projectInfo)
	} else {
		context.SessionID = req.SessionID
		context.UserID = req.UserID
		context.WorkspaceID = req.WorkspaceID
	}

	context.CreatedAt = time.Now()
	context.UpdatedAt = time.Now()

	// === 阶段6: 存储到内存 ===
	ucm.updateMemory(req.SessionID, context)

	log.Printf("✅ [上下文初始化] 完成，会话ID: %s, 置信度: %.2f",
		req.SessionID, synthesisResult.UpdateConfidence)

	return &models.ContextUpdateResponse{
		Success:         true,
		UpdatedContext:  context,
		UpdateSummary:   "初始化上下文完成",
		ConfidenceLevel: synthesisResult.UpdateConfidence,
		ProcessingTime:  time.Since(req.StartTime).Milliseconds(),
	}, nil
}

// parallelWideRecall 并行宽召回检索
func (ucm *UnifiedContextManager) parallelWideRecall(queries []models.SearchQuery, userID string, workspaceID string) (*models.ParallelRetrievalResult, error) {
	startTime := time.Now()
	log.Printf("🔍 [宽召回] 开始并行检索，查询数量: %d", len(queries))

	// 初始化结果
	result := &models.ParallelRetrievalResult{
		TimelineResults:  make([]*models.TimelineEvent, 0),
		KnowledgeResults: make([]*models.KnowledgeNode, 0),
		VectorResults:    make([]*models.VectorMatch, 0),
		TimelineCount:    0,
		KnowledgeCount:   0,
		VectorCount:      0,
	}

	// 使用WaitGroup进行并行检索
	var wg sync.WaitGroup
	var mutex sync.Mutex

	// 为每种检索类型创建goroutine
	for _, query := range queries {
		wg.Add(1)
		go func(q models.SearchQuery) {
			defer wg.Done()

			switch q.QueryType {
			case "timeline":
				timelineResults := ucm.searchTimeline(q, userID, workspaceID)
				mutex.Lock()
				result.TimelineResults = append(result.TimelineResults, timelineResults...)
				result.TimelineCount += len(timelineResults)
				mutex.Unlock()

			case "knowledge":
				knowledgeResults := ucm.searchKnowledge(q, userID, workspaceID)
				mutex.Lock()
				result.KnowledgeResults = append(result.KnowledgeResults, knowledgeResults...)
				result.KnowledgeCount += len(knowledgeResults)
				mutex.Unlock()

			case "vector":
				vectorResults := ucm.searchVector(q, userID, workspaceID)
				mutex.Lock()
				result.VectorResults = append(result.VectorResults, vectorResults...)
				result.VectorCount += len(vectorResults)
				mutex.Unlock()
			}
		}(query)
	}

	// 等待所有检索完成
	wg.Wait()

	result.TotalRetrievalTime = time.Since(startTime).Milliseconds()

	log.Printf("🔍 [宽召回] 检索完成，时间线%d条, 知识图谱%d条, 向量%d条, 耗时%dms",
		result.TimelineCount, result.KnowledgeCount, result.VectorCount, result.TotalRetrievalTime)

	return result, nil
}

// analyzeProjectFromWorkspace 从工作空间分析项目信息
func (ucm *UnifiedContextManager) analyzeProjectFromWorkspace(workspaceID string) *models.ProjectContext {
	// TODO: 实现真正的项目分析
	// 目前返回基础项目信息
	log.Printf("📁 [项目分析] 分析工作空间: %s", workspaceID)

	return &models.ProjectContext{
		ProjectName:     extractProjectNameFromPath(workspaceID),
		ProjectPath:     workspaceID,
		ProjectType:     models.ProjectTypeGo, // 默认Go项目
		Description:     "项目描述待分析",
		PrimaryLanguage: "Go",
		LastAnalyzed:    time.Now(),
		ConfidenceLevel: 0.5,
	}
}

// createBasicContext 创建基础上下文
func (ucm *UnifiedContextManager) createBasicContext(sessionID, userID, workspaceID string, intentAnalysis *models.IntentAnalysisResult, projectInfo *models.ProjectContext) *models.UnifiedContextModel {
	return &models.UnifiedContextModel{
		SessionID:   sessionID,
		UserID:      userID,
		WorkspaceID: workspaceID,
		CurrentTopic: &models.TopicContext{
			MainTopic:       intentAnalysis.CoreIntentText,
			TopicCategory:   models.TopicCategoryTechnical,
			TopicStartTime:  time.Now(),
			LastUpdated:     time.Now(),
			ConfidenceLevel: 0.7, // 默认置信度
		},
		Project: projectInfo,
		Code: &models.CodeContext{
			SessionID:       sessionID,
			LastAnalyzed:    time.Now(),
			ConfidenceLevel: 0.5,
		},
		Conversation: &models.ConversationContext{
			LastUpdated:     time.Now(),
			MessageCount:    1,
			ConfidenceLevel: 0.5,
		},
	}
}

// updateTemporaryInfo 更新临时信息
func (ucm *UnifiedContextManager) updateTemporaryInfo(context *models.UnifiedContextModel, synthesisResult *models.ContextSynthesisResult) {
	// 更新时间戳
	context.UpdatedAt = time.Now()

	// 更新对话计数
	if context.Conversation != nil {
		context.Conversation.MessageCount++
		context.Conversation.LastUpdated = time.Now()
	}

	log.Printf("📝 [临时更新] 更新临时信息，会话ID: %s", context.SessionID)
}

// persistContextIfNeeded 根据需要持久化上下文
func (ucm *UnifiedContextManager) persistContextIfNeeded(context *models.UnifiedContextModel, confidence float64) {
	// TODO: 实现持久化逻辑
	log.Printf("💾 [持久化] 上下文持久化，会话ID: %s, 置信度: %.2f", context.SessionID, confidence)
}

// extractProjectNameFromPath 从路径提取项目名称
func extractProjectNameFromPath(path string) string {
	// 简单实现：取路径的最后一部分
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-project"
}

// searchTimeline 搜索时间线数据
func (ucm *UnifiedContextManager) searchTimeline(query models.SearchQuery, userID string, workspaceID string) []*models.TimelineEvent {
	log.Printf("🕒 [时间线检索] 查询: %s", query.QueryText)

	// 基于ContextService的时间线检索能力
	if ucm.contextService != nil {
		// 调用ContextService的检索功能
		ctx := context.Background()
		searchOptions := map[string]interface{}{
			"query_type": "timeline",
			"user_id":    userID,
			"workspace":  workspaceID,
			"keywords":   query.Keywords,
		}

		results, err := ucm.contextService.searchByText(ctx, query.QueryText, "", searchOptions)
		if err != nil {
			log.Printf("⚠️ [时间线检索] 检索失败: %v", err)
			return []*models.TimelineEvent{}
		}

		// 转换搜索结果为时间线事件
		timelineEvents := make([]*models.TimelineEvent, 0)
		for _, result := range results {
			// 从Fields中提取标题和内容
			title := "搜索结果"
			content := ""
			if result.Fields != nil {
				if t, ok := result.Fields["title"].(string); ok {
					title = t
				}
				if c, ok := result.Fields["content"].(string); ok {
					content = c
				}
			}

			event := &models.TimelineEvent{
				ID:              result.ID,
				EventType:       "search_result",
				Title:           title,
				Content:         content,
				Timestamp:       time.Now(), // 实际应该从result中获取
				Keywords:        pq.StringArray(query.Keywords),
				ImportanceScore: result.Score,
				UserID:          userID,
				WorkspaceID:     workspaceID,
			}
			timelineEvents = append(timelineEvents, event)
		}

		log.Printf("🕒 [时间线检索] 找到 %d 条结果", len(timelineEvents))
		return timelineEvents
	}

	return []*models.TimelineEvent{}
}

// searchKnowledge 搜索知识图谱数据
func (ucm *UnifiedContextManager) searchKnowledge(query models.SearchQuery, userID string, workspaceID string) []*models.KnowledgeNode {
	log.Printf("🧠 [知识检索] 查询: %s", query.QueryText)

	// 基于ContextService的知识检索能力
	if ucm.contextService != nil {
		ctx := context.Background()
		searchOptions := map[string]interface{}{
			"query_type": "knowledge",
			"user_id":    userID,
			"workspace":  workspaceID,
			"keywords":   query.Keywords,
		}

		results, err := ucm.contextService.searchByText(ctx, query.QueryText, "", searchOptions)
		if err != nil {
			log.Printf("⚠️ [知识检索] 检索失败: %v", err)
			return []*models.KnowledgeNode{}
		}

		// 转换搜索结果为知识节点
		knowledgeNodes := make([]*models.KnowledgeNode, 0)
		for _, result := range results {
			// 从Fields中提取标签和内容
			label := "搜索概念"
			content := ""
			if result.Fields != nil {
				if l, ok := result.Fields["label"].(string); ok {
					label = l
				}
				if c, ok := result.Fields["content"].(string); ok {
					content = c
				}
			}

			// 构建元数据
			metadata := map[string]interface{}{
				"score":     result.Score,
				"keywords":  query.Keywords,
				"user_id":   userID,
				"workspace": workspaceID,
			}

			node := &models.KnowledgeNode{
				ID:         result.ID,
				Type:       "search_concept",
				Name:       label,
				Content:    content,
				Properties: metadata,
			}
			knowledgeNodes = append(knowledgeNodes, node)
		}

		log.Printf("🧠 [知识检索] 找到 %d 条结果", len(knowledgeNodes))
		return knowledgeNodes
	}

	return []*models.KnowledgeNode{}
}

// searchVector 搜索向量数据
func (ucm *UnifiedContextManager) searchVector(query models.SearchQuery, userID string, workspaceID string) []*models.VectorMatch {
	log.Printf("🔍 [向量检索] 查询: %s", query.QueryText)

	// 基于ContextService的向量检索能力
	if ucm.contextService != nil {
		ctx := context.Background()
		searchOptions := map[string]interface{}{
			"query_type": "vector",
			"user_id":    userID,
			"workspace":  workspaceID,
			"keywords":   query.Keywords,
		}

		results, err := ucm.contextService.searchByText(ctx, query.QueryText, "", searchOptions)
		if err != nil {
			log.Printf("⚠️ [向量检索] 检索失败: %v", err)
			return []*models.VectorMatch{}
		}

		// 转换搜索结果为向量匹配
		vectorMatches := make([]*models.VectorMatch, 0)
		for _, result := range results {
			// 从Fields中提取内容和来源
			content := ""
			source := "search_result"
			if result.Fields != nil {
				if c, ok := result.Fields["content"].(string); ok {
					content = c
				}
				if s, ok := result.Fields["source"].(string); ok {
					source = s
				}
			}

			match := &models.VectorMatch{
				ID:          result.ID,
				Content:     content,
				SourceType:  source,
				Score:       result.Score,
				Timestamp:   time.Now(), // 实际应该从result中获取
				UserID:      userID,
				WorkspaceID: workspaceID,
			}
			vectorMatches = append(vectorMatches, match)
		}

		log.Printf("🔍 [向量检索] 找到 %d 条结果", len(vectorMatches))
		return vectorMatches
	}

	return []*models.VectorMatch{}
}

// generateSearchQueries 生成搜索查询
func (ucm *UnifiedContextManager) generateSearchQueries(intentAnalysis *models.IntentAnalysisResult, userQuery string) []models.SearchQuery {
	queries := make([]models.SearchQuery, 0)

	// 基于核心意图生成查询
	if intentAnalysis.CoreIntentText != "" {
		queries = append(queries, models.SearchQuery{
			QueryText: intentAnalysis.CoreIntentText,
			QueryType: "timeline",
			Keywords:  []string{intentAnalysis.CoreIntentText},
			Priority:  1,
		})

		queries = append(queries, models.SearchQuery{
			QueryText: intentAnalysis.CoreIntentText,
			QueryType: "knowledge",
			Keywords:  []string{intentAnalysis.CoreIntentText},
			Priority:  1,
		})
	}

	// 基于场景生成查询
	if intentAnalysis.ScenarioText != "" {
		queries = append(queries, models.SearchQuery{
			QueryText: intentAnalysis.ScenarioText,
			QueryType: "vector",
			Keywords:  []string{intentAnalysis.ScenarioText},
			Priority:  2,
		})
	}

	// 基于原始查询生成向量查询
	queries = append(queries, models.SearchQuery{
		QueryText: userQuery,
		QueryType: "vector",
		Keywords:  strings.Fields(userQuery),
		Priority:  3,
	})

	log.Printf("🔍 [查询生成] 生成了 %d 个搜索查询", len(queries))
	return queries
}
