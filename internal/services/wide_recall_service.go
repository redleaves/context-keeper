package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
	// TODO: 创建这些包的接口
	// "github.com/contextkeeper/service/internal/services/llm"
	// "github.com/contextkeeper/service/internal/storage"
)

// WideRecallService 宽召回服务
type WideRecallService struct {
	// === 存储引擎 ===
	timelineStore  TimelineStore  // 时间线存储
	knowledgeStore KnowledgeStore // 知识图谱存储
	vectorStore    VectorStore    // 向量存储

	// === LLM服务 ===
	llmService LLMService // LLM服务

	// === 配置 ===
	config *WideRecallConfig // 配置

	// === 并发控制 ===
	mu sync.RWMutex // 读写锁
}

// WideRecallConfig 宽召回配置
type WideRecallConfig struct {
	// === 超时配置 ===
	LLMTimeout       int `json:"llm_timeout"`       // LLM超时时间(秒)
	TimelineTimeout  int `json:"timeline_timeout"`  // 时间线检索超时(秒)
	KnowledgeTimeout int `json:"knowledge_timeout"` // 知识图谱检索超时(秒)
	VectorTimeout    int `json:"vector_timeout"`    // 向量检索超时(秒)

	// === 结果数量限制 ===
	TimelineMaxResults  int `json:"timeline_max_results"`  // 时间线最大结果数
	KnowledgeMaxResults int `json:"knowledge_max_results"` // 知识图谱最大结果数
	VectorMaxResults    int `json:"vector_max_results"`    // 向量最大结果数

	// === 质量阈值 ===
	MinSimilarityScore   float64 `json:"min_similarity_score"`  // 最小相似度分数
	MinRelevanceScore    float64 `json:"min_relevance_score"`   // 最小相关性分数
	ConfidenceThreshold  float64 `json:"confidence_threshold"`  // 置信度阈值
	UpdateThreshold      float64 `json:"update_threshold"`      // 更新阈值
	PersistenceThreshold float64 `json:"persistence_threshold"` // 持久化阈值

	// === 重试配置 ===
	MaxRetries    int `json:"max_retries"`    // 最大重试次数
	RetryInterval int `json:"retry_interval"` // 重试间隔(秒)
}

// NewWideRecallService 创建宽召回服务
func NewWideRecallService(
	timelineStore TimelineStore,
	knowledgeStore KnowledgeStore,
	vectorStore VectorStore,
	llmService LLMService,
	config *WideRecallConfig,
) *WideRecallService {
	if config == nil {
		config = getDefaultWideRecallConfig()
	}

	return &WideRecallService{
		timelineStore:  timelineStore,
		knowledgeStore: knowledgeStore,
		vectorStore:    vectorStore,
		llmService:     llmService,
		config:         config,
	}
}

// ExecuteWideRecall 执行宽召回检索
func (s *WideRecallService) ExecuteWideRecall(ctx context.Context, req *models.WideRecallRequest) (*models.WideRecallResponse, error) {
	startTime := time.Now()

	// === 阶段1: LLM意图分析和查询拆解 ===
	intentAnalysis, err := s.analyzeUserIntent(ctx, req.UserQuery)
	if err != nil {
		return nil, fmt.Errorf("意图分析失败: %w", err)
	}

	// === 阶段2: 并行宽召回检索 ===
	retrievalResults, err := s.executeParallelRetrieval(ctx, intentAnalysis, req)
	if err != nil {
		return nil, fmt.Errorf("并行检索失败: %w", err)
	}

	// === 构建响应 ===
	response := &models.WideRecallResponse{
		Success:          true,
		Message:          "宽召回检索成功",
		RequestID:        generateWideRecallRequestID(),
		ProcessTime:      time.Since(startTime).Milliseconds(),
		RetrievalResults: retrievalResults,
		ResponseTime:     time.Now(),
	}

	return response, nil
}

// ExecuteContextSynthesis 执行上下文合成
func (s *WideRecallService) ExecuteContextSynthesis(ctx context.Context, req *models.ContextSynthesisRequest) (*models.ContextSynthesisResponse, error) {
	startTime := time.Now()

	// === 阶段3: LLM评估融合和上下文合成 ===
	synthesisResult, err := s.synthesizeAndEvaluateContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("上下文合成失败: %w", err)
	}

	// === 构建响应 ===
	response := &models.ContextSynthesisResponse{
		Success:            true,
		Message:            "上下文合成成功",
		RequestID:          generateWideRecallRequestID(),
		ProcessTime:        time.Since(startTime).Milliseconds(),
		EvaluationResult:   synthesisResult.EvaluationResult,
		SynthesizedContext: synthesisResult.SynthesizedContext,
		SynthesisMetadata:  synthesisResult.SynthesisMetadata,
		ResponseTime:       time.Now(),
	}

	return response, nil
}

// analyzeUserIntent 分析用户意图
func (s *WideRecallService) analyzeUserIntent(ctx context.Context, userQuery string) (*models.WideRecallIntentAnalysis, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.LLMTimeout)*time.Second)
	defer cancel()

	// 构建意图分析Prompt
	prompt := s.buildIntentAnalysisPrompt(userQuery)

	// 调用LLM进行意图分析
	llmRequest := &GenerateRequest{
		Prompt:      prompt,
		MaxTokens:   4000, // 增加token限制
		Temperature: 0.1,  // 降低温度提高一致性
		Format:      "json",
	}

	log.Printf("🤖 [方案1-R1模型] 意图分析LLM请求")
	log.Printf("📤 [意图分析请求] Prompt长度: %d字符", len(llmRequest.Prompt))
	log.Printf("📤 [意图分析请求] MaxTokens: %d, Temperature: %.1f", llmRequest.MaxTokens, llmRequest.Temperature)
	log.Printf("📤 [意图分析请求] Prompt前300字符:\n%s", llmRequest.Prompt[:min(300, len(llmRequest.Prompt))])

	response, err := s.llmService.GenerateResponse(timeoutCtx, llmRequest)
	if err != nil {
		log.Printf("❌ [意图分析响应] LLM调用失败: %v", err)
		return nil, fmt.Errorf("LLM意图分析失败: %w", err)
	}

	log.Printf("📥 [意图分析响应] 响应长度: %d字符", len(response.Content))
	log.Printf("📥 [意图分析响应] Token使用: Prompt=%d, Completion=%d, Total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	log.Printf("📥 [意图分析响应] 内容前800字符:\n%s", response.Content[:min(800, len(response.Content))])

	// 解析LLM响应
	intentAnalysis, err := s.parseIntentAnalysisResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("解析意图分析结果失败: %w", err)
	}

	return intentAnalysis, nil
}

// executeParallelRetrieval 执行并行检索
func (s *WideRecallService) executeParallelRetrieval(ctx context.Context, intentAnalysis *models.WideRecallIntentAnalysis, req *models.WideRecallRequest) (*models.RetrievalResults, error) {
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
		result := s.executeTimelineRetrieval(ctx, intentAnalysis.RetrievalStrategy.TimelineQueries, req)
		timelineResultChan <- result
	}()

	// 知识图谱检索
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := s.executeKnowledgeRetrieval(ctx, intentAnalysis.RetrievalStrategy.KnowledgeQueries, req)
		knowledgeResultChan <- result
	}()

	// 向量检索
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := s.executeVectorRetrieval(ctx, intentAnalysis.RetrievalStrategy.VectorQueries, req)
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

	// 构建汇总结果
	retrievalResults := &models.RetrievalResults{
		TimelineResults:  timelineResult.Results,
		TimelineCount:    len(timelineResult.Results),
		TimelineStatus:   timelineResult.Status,
		KnowledgeResults: knowledgeResult.Results,
		KnowledgeCount:   len(knowledgeResult.Results),
		KnowledgeStatus:  knowledgeResult.Status,
		VectorResults:    vectorResult.Results,
		VectorCount:      len(vectorResult.Results),
		VectorStatus:     vectorResult.Status,
		TotalResults:     len(timelineResult.Results) + len(knowledgeResult.Results) + len(vectorResult.Results),
		OverallQuality:   s.calculateOverallQuality(timelineResult, knowledgeResult, vectorResult),
		RetrievalTime:    timelineResult.Duration + knowledgeResult.Duration + vectorResult.Duration,
		SuccessfulDims:   s.countSuccessfulDimensions(timelineResult, knowledgeResult, vectorResult),
	}

	return retrievalResults, nil
}

// synthesizeAndEvaluateContext 合成和评估上下文
func (s *WideRecallService) synthesizeAndEvaluateContext(ctx context.Context, req *models.ContextSynthesisRequest) (*ContextSynthesisResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.LLMTimeout)*time.Second)
	defer cancel()

	// 构建上下文合成Prompt
	prompt := s.buildContextSynthesisPrompt(req)

	// 调用LLM进行上下文合成
	llmRequest := &GenerateRequest{
		Prompt:      prompt,
		MaxTokens:   8000, // 大幅增加token限制以支持复杂的UnifiedContextModel
		Temperature: 0.1,  // 降低温度提高一致性
		Format:      "json",
	}

	log.Printf("🤖 [方案1-R1模型] 上下文合成LLM请求")
	log.Printf("📤 [上下文合成请求] Prompt长度: %d字符", len(llmRequest.Prompt))
	log.Printf("📤 [上下文合成请求] MaxTokens: %d, Temperature: %.1f", llmRequest.MaxTokens, llmRequest.Temperature)
	log.Printf("📤 [上下文合成请求] Prompt前500字符:\n%s", llmRequest.Prompt[:min(500, len(llmRequest.Prompt))])

	response, err := s.llmService.GenerateResponse(timeoutCtx, llmRequest)
	if err != nil {
		log.Printf("❌ [上下文合成响应] LLM调用失败: %v", err)
		return nil, fmt.Errorf("LLM上下文合成失败: %w", err)
	}

	log.Printf("📥 [上下文合成响应] 响应长度: %d字符", len(response.Content))
	log.Printf("📥 [上下文合成响应] Token使用: Prompt=%d, Completion=%d, Total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	log.Printf("📥 [上下文合成响应] 内容前1000字符:\n%s", response.Content[:min(1000, len(response.Content))])

	// 解析LLM响应
	synthesisResult, err := s.parseContextSynthesisResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("解析上下文合成结果失败: %w", err)
	}

	return synthesisResult, nil
}

// 内部结果结构
type TimelineRetrievalResult struct {
	Results  []models.TimelineResult `json:"results"`
	Status   string                  `json:"status"`
	Duration int64                   `json:"duration"`
	Error    error                   `json:"error,omitempty"`
}

type KnowledgeRetrievalResult struct {
	Results  []models.KnowledgeResult `json:"results"`
	Status   string                   `json:"status"`
	Duration int64                    `json:"duration"`
	Error    error                    `json:"error,omitempty"`
}

type VectorRetrievalResult struct {
	Results  []models.VectorResult `json:"results"`
	Status   string                `json:"status"`
	Duration int64                 `json:"duration"`
	Error    error                 `json:"error,omitempty"`
}

type ContextSynthesisResult struct {
	EvaluationResult   *models.EvaluationResult    `json:"evaluation_result"`
	SynthesizedContext *models.UnifiedContextModel `json:"synthesized_context"`
	SynthesisMetadata  *models.SynthesisMetadata   `json:"synthesis_metadata"`
}

// getDefaultWideRecallConfig 获取默认配置
func getDefaultWideRecallConfig() *WideRecallConfig {
	return &WideRecallConfig{
		LLMTimeout:           40, // 40秒
		TimelineTimeout:      5,  // 5秒
		KnowledgeTimeout:     5,  // 5秒
		VectorTimeout:        5,  // 5秒
		TimelineMaxResults:   20,
		KnowledgeMaxResults:  15,
		VectorMaxResults:     25,
		MinSimilarityScore:   0.6,
		MinRelevanceScore:    0.5,
		ConfidenceThreshold:  0.7,
		UpdateThreshold:      0.4,
		PersistenceThreshold: 0.7,
		MaxRetries:           1,
		RetryInterval:        2,
	}
}

// generateWideRecallRequestID 生成宽召回请求ID
func generateWideRecallRequestID() string {
	return fmt.Sprintf("wr_%d", time.Now().UnixNano())
}

// executeTimelineRetrieval 执行时间线检索
func (s *WideRecallService) executeTimelineRetrieval(ctx context.Context, queries []models.TimelineQuery, req *models.WideRecallRequest) *TimelineRetrievalResult {
	startTime := time.Now()

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimelineTimeout)*time.Second)
	defer cancel()

	var allResults []models.TimelineResult
	status := "success"

	// 执行每个时间线查询
	for _, query := range queries {
		results, err := s.timelineStore.SearchEvents(timeoutCtx, &TimelineSearchRequest{
			UserID:      req.UserID,
			WorkspaceID: req.WorkspaceID,
			Query:       query.QueryText,
			TimeRange:   query.TimeRange,
			EventTypes:  query.EventTypes,
			MaxResults:  s.config.TimelineMaxResults,
		})

		if err != nil {
			status = "partial_failure"
			continue
		}

		// 转换结果格式
		for _, result := range results {
			timelineResult := models.TimelineResult{
				EventID:         result.EventID,
				EventType:       result.EventType,
				Title:           result.Title,
				Content:         result.Content,
				Timestamp:       result.Timestamp,
				Source:          result.Source,
				ImportanceScore: result.ImportanceScore,
				RelevanceScore:  result.RelevanceScore,
				Tags:            result.Tags,
				Metadata:        result.Metadata,
			}
			allResults = append(allResults, timelineResult)
		}
	}

	// 如果没有任何结果且发生错误，标记为失败
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	return &TimelineRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: time.Since(startTime).Milliseconds(),
	}
}

// executeKnowledgeRetrieval 执行知识图谱检索
func (s *WideRecallService) executeKnowledgeRetrieval(ctx context.Context, queries []models.KnowledgeQuery, req *models.WideRecallRequest) *KnowledgeRetrievalResult {
	startTime := time.Now()

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.KnowledgeTimeout)*time.Second)
	defer cancel()

	var allResults []models.KnowledgeResult
	status := "success"

	// 执行每个知识图谱查询
	for _, query := range queries {
		results, err := s.knowledgeStore.SearchConcepts(timeoutCtx, &KnowledgeSearchRequest{
			UserID:        req.UserID,
			WorkspaceID:   req.WorkspaceID,
			Query:         query.QueryText,
			ConceptTypes:  query.ConceptTypes,
			RelationTypes: query.RelationTypes,
			MaxResults:    s.config.KnowledgeMaxResults,
		})

		if err != nil {
			status = "partial_failure"
			continue
		}

		// 转换结果格式
		for _, result := range results {
			knowledgeResult := models.KnowledgeResult{
				ConceptID:       result.ConceptID,
				ConceptName:     result.ConceptName,
				ConceptType:     result.ConceptType,
				Description:     result.Description,
				RelatedConcepts: convertRelatedConcepts(result.RelatedConcepts),
				Properties:      result.Properties,
				RelevanceScore:  result.RelevanceScore,
				ConfidenceScore: result.ConfidenceScore,
				Source:          result.Source,
				LastUpdated:     result.LastUpdated,
			}
			allResults = append(allResults, knowledgeResult)
		}
	}

	// 如果没有任何结果且发生错误，标记为失败
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	return &KnowledgeRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: time.Since(startTime).Milliseconds(),
	}
}

// executeVectorRetrieval 执行向量检索
func (s *WideRecallService) executeVectorRetrieval(ctx context.Context, queries []models.VectorQuery, req *models.WideRecallRequest) *VectorRetrievalResult {
	startTime := time.Now()

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.VectorTimeout)*time.Second)
	defer cancel()

	var allResults []models.VectorResult
	status := "success"

	// 执行每个向量查询
	for _, query := range queries {
		results, err := s.vectorStore.SearchSimilar(timeoutCtx, &VectorSearchRequest{
			UserID:              req.UserID,
			WorkspaceID:         req.WorkspaceID,
			Query:               query.QueryText,
			SimilarityThreshold: query.SimilarityThreshold,
			MaxResults:          s.config.VectorMaxResults,
		})

		if err != nil {
			status = "partial_failure"
			continue
		}

		// 转换结果格式
		for _, result := range results {
			vectorResult := models.VectorResult{
				DocumentID:      result.DocumentID,
				Content:         result.Content,
				ContentType:     result.ContentType,
				Source:          result.Source,
				Similarity:      result.Similarity,
				RelevanceScore:  result.RelevanceScore,
				Timestamp:       result.Timestamp,
				Tags:            result.Tags,
				Metadata:        result.Metadata,
				MatchedSegments: convertMatchedSegments(result.MatchedSegments),
			}
			allResults = append(allResults, vectorResult)
		}
	}

	// 如果没有任何结果且发生错误，标记为失败
	if len(allResults) == 0 && status == "partial_failure" {
		status = "failure"
	}

	return &VectorRetrievalResult{
		Results:  allResults,
		Status:   status,
		Duration: time.Since(startTime).Milliseconds(),
	}
}

// 辅助函数
func convertRelatedConcepts(storageRelated []RelatedConcept) []models.RelatedConcept {
	var result []models.RelatedConcept
	for _, related := range storageRelated {
		result = append(result, models.RelatedConcept{
			ConceptName:    related.ConceptName,
			RelationType:   related.RelationType,
			RelationWeight: related.RelationWeight,
		})
	}
	return result
}

func convertMatchedSegments(storageSegments []MatchedSegment) []models.MatchedSegment {
	var result []models.MatchedSegment
	for _, segment := range storageSegments {
		result = append(result, models.MatchedSegment{
			SegmentText: segment.SegmentText,
			StartPos:    segment.StartPos,
			EndPos:      segment.EndPos,
			Similarity:  segment.Similarity,
		})
	}
	return result
}

// calculateOverallQuality 计算总体质量
func (s *WideRecallService) calculateOverallQuality(timeline *TimelineRetrievalResult, knowledge *KnowledgeRetrievalResult, vector *VectorRetrievalResult) float64 {
	var totalScore float64
	var totalWeight float64

	// 时间线质量评分
	if timeline.Status == "success" && len(timeline.Results) > 0 {
		timelineScore := 0.0
		for _, result := range timeline.Results {
			timelineScore += (result.ImportanceScore + result.RelevanceScore) / 2
		}
		timelineScore /= float64(len(timeline.Results))
		totalScore += timelineScore * 0.3 // 30%权重
		totalWeight += 0.3
	}

	// 知识图谱质量评分
	if knowledge.Status == "success" && len(knowledge.Results) > 0 {
		knowledgeScore := 0.0
		for _, result := range knowledge.Results {
			knowledgeScore += (result.RelevanceScore + result.ConfidenceScore) / 2
		}
		knowledgeScore /= float64(len(knowledge.Results))
		totalScore += knowledgeScore * 0.3 // 30%权重
		totalWeight += 0.3
	}

	// 向量检索质量评分
	if vector.Status == "success" && len(vector.Results) > 0 {
		vectorScore := 0.0
		for _, result := range vector.Results {
			vectorScore += (result.Similarity + result.RelevanceScore) / 2
		}
		vectorScore /= float64(len(vector.Results))
		totalScore += vectorScore * 0.4 // 40%权重
		totalWeight += 0.4
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalScore / totalWeight
}

// countSuccessfulDimensions 统计成功的维度数量
func (s *WideRecallService) countSuccessfulDimensions(timeline *TimelineRetrievalResult, knowledge *KnowledgeRetrievalResult, vector *VectorRetrievalResult) int {
	count := 0
	if timeline.Status == "success" {
		count++
	}
	if knowledge.Status == "success" {
		count++
	}
	if vector.Status == "success" {
		count++
	}
	return count
}
