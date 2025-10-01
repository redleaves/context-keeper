package engines

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// MultiDimensionalRetrieverImpl 多维度检索器实现
type MultiDimensionalRetrieverImpl struct {
	// === 存储引擎 ===
	timelineStore  TimelineStore  // 时间线存储
	knowledgeStore KnowledgeStore // 知识图谱存储
	vectorStore    VectorStore    // 向量存储

	// === 配置 ===
	config *MultiDimensionalConfig

	// === 并发控制 ===
	mu sync.RWMutex
}

// MultiDimensionalConfig 多维度检索配置
type MultiDimensionalConfig struct {
	TimelineTimeout     int     // 时间线检索超时（秒）
	KnowledgeTimeout    int     // 知识图谱检索超时（秒）
	VectorTimeout       int     // 向量检索超时（秒）
	TimelineMaxResults  int     // 时间线最大结果数
	KnowledgeMaxResults int     // 知识图谱最大结果数
	VectorMaxResults    int     // 向量最大结果数
	MinSimilarityScore  float64 // 最小相似度分数
	MinRelevanceScore   float64 // 最小相关性分数
	MaxRetries          int     // 最大重试次数
	RetryInterval       int     // 重试间隔（秒）
}

// 存储接口定义（面向应用层的简单统一接口）
type TimelineStore interface {
	SearchByQuery(ctx context.Context, req *models.TimelineSearchRequest) ([]*models.TimelineEvent, error)
}

type KnowledgeStore interface {
	SearchByQuery(ctx context.Context, query string, limit int) ([]*models.KnowledgeNode, error)
}

type VectorStore interface {
	SearchByQuery(ctx context.Context, query string, limit int) ([]*models.VectorMatch, error)
}

// RetrievalResults 检索结果集合
type RetrievalResults struct {
	TimelineResults  []*models.TimelineEvent `json:"timeline_results"`
	KnowledgeResults []*models.KnowledgeNode `json:"knowledge_results"`
	VectorResults    []*models.VectorMatch   `json:"vector_results"`
	TimelineCount    int                     `json:"timeline_count"`
	KnowledgeCount   int                     `json:"knowledge_count"`
	VectorCount      int                     `json:"vector_count"`
	TotalResults     int                     `json:"total_results"`
	OverallQuality   float64                 `json:"overall_quality"`
	RetrievalTime    int64                   `json:"retrieval_time_ms"`
	Results          []interface{}           `json:"results"` // 兼容性字段
}

// TimelineRetrievalResult 时间线检索结果
type TimelineRetrievalResult struct {
	Results  []*models.TimelineEvent `json:"results"`
	Status   string                  `json:"status"`
	Duration int64                   `json:"duration_ms"`
	Error    error                   `json:"error,omitempty"`
}

// KnowledgeRetrievalResult 知识图谱检索结果
type KnowledgeRetrievalResult struct {
	Results  []*models.KnowledgeNode `json:"results"`
	Status   string                  `json:"status"`
	Duration int64                   `json:"duration_ms"`
	Error    error                   `json:"error,omitempty"`
}

// VectorRetrievalResult 向量检索结果
type VectorRetrievalResult struct {
	Results  []*models.VectorMatch `json:"results"`
	Status   string                `json:"status"`
	Duration int64                 `json:"duration_ms"`
	Error    error                 `json:"error,omitempty"`
}

// NewMultiDimensionalRetriever 创建多维度检索器
func NewMultiDimensionalRetriever(timelineStore TimelineStore, knowledgeStore KnowledgeStore, vectorStore VectorStore) *MultiDimensionalRetrieverImpl {
	return &MultiDimensionalRetrieverImpl{
		timelineStore:  timelineStore,
		knowledgeStore: knowledgeStore,
		vectorStore:    vectorStore,
		config:         getDefaultMultiDimensionalConfig(),
	}
}

// ParallelRetrieve 并行检索（直接复制WideRecallService.executeParallelRetrieval的逻辑）
func (mdr *MultiDimensionalRetrieverImpl) ParallelRetrieve(ctx context.Context, queries *models.MultiDimensionalQuery) (*RetrievalResults, error) {
	log.Printf("🔍 [多维度检索] 开始并行检索...")

	// 创建结果通道
	timelineResultChan := make(chan *TimelineRetrievalResult, 1)
	knowledgeResultChan := make(chan *KnowledgeRetrievalResult, 1)
	vectorResultChan := make(chan *VectorRetrievalResult, 1)

	// 启动并行检索
	var wg sync.WaitGroup

	// 时间线检索
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := mdr.executeTimelineRetrieval(ctx, queries)
		timelineResultChan <- result
	}()

	// 知识图谱检索
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := mdr.executeKnowledgeRetrieval(ctx, queries.KnowledgeQueries, queries.UserID)
		knowledgeResultChan <- result
	}()

	// 向量检索
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := mdr.executeVectorRetrieval(ctx, queries.VectorQueries, queries.UserID)
		vectorResultChan <- result
	}()

	// 等待所有检索完成
	wg.Wait()
	close(timelineResultChan)
	close(knowledgeResultChan)
	close(vectorResultChan)

	// 收集结果
	timelineResult := <-timelineResultChan
	knowledgeResult := <-knowledgeResultChan
	vectorResult := <-vectorResultChan

	// 构建汇总结果（与WideRecallService保持一致）
	retrievalResults := &RetrievalResults{
		TimelineResults:  timelineResult.Results,
		TimelineCount:    len(timelineResult.Results),
		KnowledgeResults: knowledgeResult.Results,
		KnowledgeCount:   len(knowledgeResult.Results),
		VectorResults:    vectorResult.Results,
		VectorCount:      len(vectorResult.Results),
		TotalResults:     len(timelineResult.Results) + len(knowledgeResult.Results) + len(vectorResult.Results),
		OverallQuality:   mdr.calculateOverallQuality(timelineResult, knowledgeResult, vectorResult),
		RetrievalTime:    timelineResult.Duration + knowledgeResult.Duration + vectorResult.Duration,
		Results:          []interface{}{}, // 兼容性字段
	}

	log.Printf("✅ [多维度检索] 并行检索完成，总结果: %d, 耗时: %dms",
		retrievalResults.TotalResults, retrievalResults.RetrievalTime)

	return retrievalResults, nil
}

// executeTimelineRetrieval 执行时间线检索（直接复制WideRecallService的逻辑）
func (mdr *MultiDimensionalRetrieverImpl) executeTimelineRetrieval(ctx context.Context, retrievalQueries *models.MultiDimensionalQuery) *TimelineRetrievalResult {
	startTime := time.Now()
	queries := retrievalQueries.TimelineQueries
	userID := retrievalQueries.UserID

	// 🔥 获取LLM分析的关键概念
	keyConcepts := retrievalQueries.KeyConcepts

	log.Printf("📅 [时间线检索] 开始执行，查询数量: %d", len(queries))

	// 🔥 详细日志：打印输入参数
	log.Printf("📥 [时间线检索-入参] UserID: %s", userID)
	log.Printf("📥 [时间线检索-入参] 查询列表: %v", queries)
	log.Printf("📥 [时间线检索-入参] LLM关键概念: %v", keyConcepts)
	log.Printf("📥 [时间线检索-入参] 超时设置: %d秒", mdr.config.TimelineTimeout)
	log.Printf("📥 [时间线检索-入参] 最大结果数: %d", mdr.config.TimelineMaxResults)

	if mdr.timelineStore == nil {
		log.Printf("⚠️ [时间线检索] 时间线存储未初始化，返回空结果")
		return &TimelineRetrievalResult{
			Results:  []*models.TimelineEvent{},
			Status:   "skipped",
			Duration: time.Since(startTime).Milliseconds(),
		}
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(mdr.config.TimelineTimeout)*time.Second)
	defer cancel()

	var allResults []*models.TimelineEvent
	status := "success"

	// 执行每个时间线查询
	for i, query := range queries {
		if query == "" {
			log.Printf("📝 [时间线检索] 跳过空查询[%d]", i)
			continue
		}

		log.Printf("🔍 [时间线检索] 执行查询[%d]: %s", i, query)
		queryStartTime := time.Now()

		// 🔥 构建时间线搜索请求
		searchReq := &models.TimelineSearchRequest{
			Query:       query,
			Limit:       mdr.config.TimelineMaxResults,
			KeyConcepts: keyConcepts, // 🔥 关键修复：使用LLM分析的关键概念
			UserID:      userID,
			WorkspaceID: retrievalQueries.WorkspaceID,
		}

		results, err := mdr.timelineStore.SearchByQuery(timeoutCtx, searchReq)
		queryDuration := time.Since(queryStartTime)

		if err != nil {
			log.Printf("❌ [时间线检索] 查询[%d]失败: %v, 耗时: %v", i, err, queryDuration)
			status = "partial_failure"
			continue
		}

		log.Printf("✅ [时间线检索] 查询[%d]成功: 获得%d个结果, 耗时: %v", i, len(results), queryDuration)

		// 🔥 详细日志：打印每个查询的结果概要
		for j, result := range results {
			log.Printf("   📄 [结果%d-%d] ID: %s, 标题: %s, 时间: %s",
				i, j, result.ID, result.Title, result.Timestamp.Format("2006-01-02 15:04:05"))
		}

		allResults = append(allResults, results...)
	}

	// 如果没有任何结果且发生错误，标记为失败
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	duration := time.Since(startTime).Milliseconds()

	// 🔥 详细日志：打印输出参数
	log.Printf("📤 [时间线检索-出参] 总结果数: %d", len(allResults))
	log.Printf("📤 [时间线检索-出参] 执行状态: %s", status)
	log.Printf("📤 [时间线检索-出参] 总耗时: %dms", duration)

	log.Printf("✅ [时间线检索] 完成，获得 %d 个结果，耗时: %dms", len(allResults), duration)

	return &TimelineRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: duration,
	}
}

// executeKnowledgeRetrieval 执行知识图谱检索（直接复制WideRecallService的逻辑）
func (mdr *MultiDimensionalRetrieverImpl) executeKnowledgeRetrieval(ctx context.Context, queries []string, userID string) *KnowledgeRetrievalResult {
	startTime := time.Now()
	log.Printf("🧠 [知识图谱检索] 开始执行，查询数量: %d", len(queries))

	// 🔥 详细日志：打印输入参数
	log.Printf("📥 [知识图谱检索-入参] UserID: %s", userID)
	log.Printf("📥 [知识图谱检索-入参] 查询列表: %v", queries)
	log.Printf("📥 [知识图谱检索-入参] 超时设置: %d秒", mdr.config.KnowledgeTimeout)
	log.Printf("📥 [知识图谱检索-入参] 最大结果数: %d", mdr.config.KnowledgeMaxResults)

	if mdr.knowledgeStore == nil {
		log.Printf("⚠️ [知识图谱检索] 知识图谱存储未初始化，返回空结果")
		return &KnowledgeRetrievalResult{
			Results:  []*models.KnowledgeNode{},
			Status:   "skipped",
			Duration: time.Since(startTime).Milliseconds(),
		}
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(mdr.config.KnowledgeTimeout)*time.Second)
	defer cancel()

	var allResults []*models.KnowledgeNode
	status := "success"

	// 执行每个知识图谱查询
	for i, query := range queries {
		if query == "" {
			log.Printf("📝 [知识图谱检索] 跳过空查询[%d]", i)
			continue
		}

		log.Printf("🔍 [知识图谱检索] 执行查询[%d]: %s", i, query)
		queryStartTime := time.Now()

		results, err := mdr.knowledgeStore.SearchByQuery(timeoutCtx, query, mdr.config.KnowledgeMaxResults)
		queryDuration := time.Since(queryStartTime)

		if err != nil {
			log.Printf("❌ [知识图谱检索] 查询[%d]失败: %v, 耗时: %v", i, err, queryDuration)
			status = "partial_failure"
			continue
		}

		log.Printf("✅ [知识图谱检索] 查询[%d]成功: 获得%d个结果, 耗时: %v", i, len(results), queryDuration)

		// 🔥 详细日志：打印每个查询的结果概要
		for j, result := range results {
			log.Printf("   🧠 [结果%d-%d] ID: %s, 名称: %s, 类型: %s",
				i, j, result.ID, result.Name, result.Type)
		}

		allResults = append(allResults, results...)
	}

	// 如果没有任何结果且发生错误，标记为失败
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	duration := time.Since(startTime).Milliseconds()

	// 🔥 详细日志：打印输出参数
	log.Printf("📤 [知识图谱检索-出参] 总结果数: %d", len(allResults))
	log.Printf("📤 [知识图谱检索-出参] 执行状态: %s", status)
	log.Printf("📤 [知识图谱检索-出参] 总耗时: %dms", duration)

	log.Printf("✅ [知识图谱检索] 完成，获得 %d 个结果，耗时: %dms", len(allResults), duration)

	return &KnowledgeRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: duration,
	}
}

// executeVectorRetrieval 执行向量检索
func (mdr *MultiDimensionalRetrieverImpl) executeVectorRetrieval(ctx context.Context, queries []string, userID string) *VectorRetrievalResult {
	startTime := time.Now()
	log.Printf("🔍 [向量检索] 开始执行，查询数量: %d", len(queries))

	// 🔥 详细日志：打印输入参数
	log.Printf("📥 [向量检索-入参] UserID: %s", userID)
	log.Printf("📥 [向量检索-入参] 查询列表: %v", queries)
	log.Printf("📥 [向量检索-入参] 超时设置: %d秒", mdr.config.VectorTimeout)
	log.Printf("📥 [向量检索-入参] 最大结果数: %d", mdr.config.VectorMaxResults)

	if mdr.vectorStore == nil {
		log.Printf("⚠️ [向量检索] 向量存储未初始化，返回空结果")
		return &VectorRetrievalResult{
			Results:  []*models.VectorMatch{},
			Status:   "skipped",
			Duration: time.Since(startTime).Milliseconds(),
		}
	}

	// 创建带超时的上下文（复用WideRecallService的超时逻辑）
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(mdr.config.VectorTimeout)*time.Second)
	defer cancel()

	var allResults []*models.VectorMatch
	status := "success"

	// 执行每个向量查询（复用WideRecallService的查询逻辑）
	for i, query := range queries {
		if query == "" {
			log.Printf("📝 [向量检索] 跳过空查询[%d]", i)
			continue
		}

		log.Printf("🔍 [向量检索] 执行查询[%d]: %s", i, query)
		queryStartTime := time.Now()

		results, err := mdr.vectorStore.SearchByQuery(timeoutCtx, query, mdr.config.VectorMaxResults)
		queryDuration := time.Since(queryStartTime)

		if err != nil {
			log.Printf("❌ [向量检索] 查询[%d]失败: %v, 耗时: %v", i, err, queryDuration)
			status = "partial_failure" // 复用WideRecallService的错误处理逻辑
			continue
		}

		log.Printf("✅ [向量检索] 查询[%d]成功: 获得%d个结果, 耗时: %v", i, len(results), queryDuration)

		// 🔥 详细日志：打印每个查询的结果概要
		for j, result := range results {
			log.Printf("   🎯 [结果%d-%d] ID: %s, 相似度: %.4f, 内容: %s",
				i, j, result.ID, result.Score, truncateString(result.Content, 50))
		}

		allResults = append(allResults, results...)
	}

	// 如果没有任何结果且发生错误，标记为失败（复用WideRecallService的状态判断逻辑）
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	duration := time.Since(startTime).Milliseconds()

	// 🔥 详细日志：打印输出参数
	log.Printf("📤 [向量检索-出参] 总结果数: %d", len(allResults))
	log.Printf("📤 [向量检索-出参] 执行状态: %s", status)
	log.Printf("📤 [向量检索-出参] 总耗时: %dms", duration)

	log.Printf("✅ [向量检索] 完成，获得 %d 个结果，耗时: %dms", len(allResults), duration)

	return &VectorRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: duration,
	}
}

// truncateString 截断字符串用于日志显示
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// calculateOverallQuality 计算总体质量
func (mdr *MultiDimensionalRetrieverImpl) calculateOverallQuality(timeline *TimelineRetrievalResult, knowledge *KnowledgeRetrievalResult, vector *VectorRetrievalResult) float64 {
	var totalScore float64
	var totalWeight float64

	// 时间线质量评分
	if timeline.Status == "success" && len(timeline.Results) > 0 {
		timelineScore := 0.8              // 基础分数
		totalScore += timelineScore * 0.3 // 30%权重
		totalWeight += 0.3
	}

	// 知识图谱质量评分
	if knowledge.Status == "success" && len(knowledge.Results) > 0 {
		knowledgeScore := 0.8              // 基础分数
		totalScore += knowledgeScore * 0.3 // 30%权重
		totalWeight += 0.3
	}

	// 向量检索质量评分
	if vector.Status == "success" && len(vector.Results) > 0 {
		vectorScore := 0.8              // 基础分数
		totalScore += vectorScore * 0.4 // 40%权重
		totalWeight += 0.4
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalScore / totalWeight
}

// deduplicateTimelineResults 去重时间线结果
func (mdr *MultiDimensionalRetrieverImpl) deduplicateTimelineResults(results []*models.TimelineEvent) []*models.TimelineEvent {
	seen := make(map[string]bool)
	var unique []*models.TimelineEvent

	for _, result := range results {
		if result == nil {
			continue
		}

		// 使用ID作为去重键
		key := result.ID
		if key == "" {
			// 如果没有ID，使用标题+时间戳作为键
			key = fmt.Sprintf("%s_%d", result.Title, result.Timestamp.Unix())
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// deduplicateKnowledgeResults 去重知识图谱结果
func (mdr *MultiDimensionalRetrieverImpl) deduplicateKnowledgeResults(results []*models.KnowledgeNode) []*models.KnowledgeNode {
	seen := make(map[string]bool)
	var unique []*models.KnowledgeNode

	for _, result := range results {
		if result == nil {
			continue
		}

		// 使用ID作为去重键
		key := result.ID
		if key == "" {
			// 如果没有ID，使用名称作为键
			key = result.Name
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// deduplicateVectorResults 去重向量结果
func (mdr *MultiDimensionalRetrieverImpl) deduplicateVectorResults(results []*models.VectorMatch) []*models.VectorMatch {
	seen := make(map[string]bool)
	var unique []*models.VectorMatch

	for _, result := range results {
		if result == nil {
			continue
		}

		// 使用ID作为去重键
		key := result.ID
		if key == "" {
			// 如果没有ID，使用内容的前100个字符作为键
			content := result.Content
			if len(content) > 100 {
				content = content[:100]
			}
			key = content
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// SetVectorStoreEngine 设置向量存储的Engine（用于延迟赋值）
func (mdr *MultiDimensionalRetrieverImpl) SetVectorStoreEngine(engine interface{}) {
	if vectorAdapter, ok := mdr.vectorStore.(interface{ SetEngine(interface{}) }); ok {
		vectorAdapter.SetEngine(engine)
		log.Printf("✅ [多维度检索器] 成功设置vectorStore的Engine")
	} else {
		log.Printf("⚠️ [多维度检索器] vectorStore不支持SetEngine方法")
	}
}

// 🆕 DirectTimelineQuery 直接时间线查询（专用于时间回忆）
func (mdr *MultiDimensionalRetrieverImpl) DirectTimelineQuery(ctx context.Context, req *models.TimelineSearchRequest) ([]*models.TimelineEvent, error) {
	log.Printf("🔍 [直接时间线查询] 开始执行，参数: %+v", req)

	if mdr.timelineStore == nil {
		log.Printf("❌ [直接时间线查询] 时间线存储未初始化")
		return []*models.TimelineEvent{}, fmt.Errorf("时间线存储未初始化")
	}

	// 🔥 直接调用时间线存储的SearchByQuery方法（会打印SQL和参数）
	log.Printf("🔍 [直接时间线查询] 调用timelineStore.SearchByQuery")
	events, err := mdr.timelineStore.SearchByQuery(ctx, req)
	if err != nil {
		log.Printf("❌ [直接时间线查询] 查询失败: %v", err)
		return nil, err
	}

	log.Printf("✅ [直接时间线查询] 查询成功，返回 %d 个事件", len(events))
	return events, nil
}

// 🆕 GetTimelineAdapter 获取时间线适配器（专用于时间回忆查询）
func (mdr *MultiDimensionalRetrieverImpl) GetTimelineAdapter() interface{} {
	log.Printf("🔧 [多维检索器] 返回多维检索器实例作为时间线适配器")
	return mdr // 直接返回自己，因为DirectTimelineQuery方法已经实现了
}

// getDefaultMultiDimensionalConfig 获取默认配置
func getDefaultMultiDimensionalConfig() *MultiDimensionalConfig {
	return &MultiDimensionalConfig{
		TimelineTimeout:     5, // 5秒
		KnowledgeTimeout:    5, // 5秒
		VectorTimeout:       5, // 5秒
		TimelineMaxResults:  20,
		KnowledgeMaxResults: 15,
		VectorMaxResults:    25,
		MinSimilarityScore:  0.6,
		MinRelevanceScore:   0.5,
		MaxRetries:          1,
		RetryInterval:       2,
	}
}
