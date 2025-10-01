package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/llm"
	"github.com/contextkeeper/service/internal/models"
)

// ContentSynthesisEngineImpl 内容合成引擎实现
type ContentSynthesisEngineImpl struct {
	// === LLM客户端 ===
	llmClient llm.LLMClient

	// === 配置 ===
	config *ContentSynthesisConfig
}

// ContentSynthesisConfig 内容合成配置
type ContentSynthesisConfig struct {
	LLMTimeout           int     // LLM调用超时（秒）
	MaxTokens            int     // 最大Token数
	Temperature          float64 // 温度参数
	ConfidenceThreshold  float64 // 置信度阈值
	ConflictResolution   string  // 冲突解决策略
	InformationFusion    string  // 信息融合策略
	QualityAssessment    string  // 质量评估策略
	UpdateThreshold      float64 // 更新阈值
	PersistenceThreshold float64 // 持久化阈值
}

// NewContentSynthesisEngine 创建内容合成引擎
func NewContentSynthesisEngine(llmClient llm.LLMClient) *ContentSynthesisEngineImpl {
	return &ContentSynthesisEngineImpl{
		llmClient: llmClient,
		config:    getDefaultContentSynthesisConfig(),
	}
}

// SynthesizeResponse 合成响应（实现接口）
func (cse *ContentSynthesisEngineImpl) SynthesizeResponse(ctx context.Context, query string, analysis *SemanticAnalysisResult, retrieval *RetrievalResults) (models.ContextResponse, error) {
	startTime := time.Now()
	log.Printf("🧠 [内容合成] 开始合成响应...")
	log.Printf("📤 [内容合成] 用户查询: %s", query[:min(100, len(query))])
	log.Printf("📊 [内容合成] 检索结果: 时间线=%d, 知识图谱=%d, 向量=%d",
		retrieval.TimelineCount, retrieval.KnowledgeCount, retrieval.VectorCount)

	// 构建上下文合成请求
	synthesisReq := &models.ContextSynthesisRequest{
		UserQuery:        query,
		IntentAnalysis:   convertToIntentAnalysis(analysis),
		RetrievalResults: convertToRetrievalResults(retrieval),
		SynthesisConfig: &models.SynthesisConfig{
			LLMTimeout:           cse.config.LLMTimeout,
			MaxTokens:            cse.config.MaxTokens,
			Temperature:          cse.config.Temperature,
			ConfidenceThreshold:  cse.config.ConfidenceThreshold,
			ConflictResolution:   cse.config.ConflictResolution,
			InformationFusion:    cse.config.InformationFusion,
			QualityAssessment:    cse.config.QualityAssessment,
			UpdateThreshold:      cse.config.UpdateThreshold,
			PersistenceThreshold: cse.config.PersistenceThreshold,
		},
		RequestTime: startTime,
	}

	// 执行上下文合成与评估
	synthesisResp, err := cse.synthesizeAndEvaluateContext(ctx, synthesisReq)
	if err != nil {
		return models.ContextResponse{}, fmt.Errorf("上下文合成失败: %w", err)
	}

	// 🔥 修复：使用LLM合成的真实内容而非硬编码字符串
	response := models.ContextResponse{
		SessionState:      "active",
		ShortTermMemory:   cse.extractContentByDimension(synthesisResp, "short_term_memory"),
		LongTermMemory:    cse.extractContentByDimension(synthesisResp, "long_term_memory"),
		RelevantKnowledge: cse.extractContentByDimension(synthesisResp, "relevant_knowledge"),
	}

	processingTime := time.Since(startTime).Milliseconds()
	confidence := 0.8 // 默认置信度
	if synthesisResp.EvaluationResult != nil {
		confidence = synthesisResp.EvaluationResult.UpdateConfidence
	}

	log.Printf("✅ [内容合成] 合成完成，耗时: %dms, 置信度: %.2f",
		processingTime, confidence)

	return response, nil
}

// synthesizeAndEvaluateContext 执行上下文合成与评估（核心逻辑）
func (cse *ContentSynthesisEngineImpl) synthesizeAndEvaluateContext(ctx context.Context, req *models.ContextSynthesisRequest) (*models.ContextSynthesisResponse, error) {
	startTime := time.Now()
	log.Printf("🔄 [上下文合成] 开始执行上下文合成与评估...")

	// 构建上下文合成Prompt
	prompt := cse.buildContextSynthesisPrompt(req)
	log.Printf("📝 [上下文合成] Prompt构建完成，长度: %d", len(prompt))

	// 🔥 检查是否有检索结果，决定合成策略
	hasRetrievalData := req.RetrievalResults != nil &&
		(len(req.RetrievalResults.TimelineResults) > 0 ||
			len(req.RetrievalResults.KnowledgeResults) > 0 ||
			len(req.RetrievalResults.VectorResults) > 0)

	if !hasRetrievalData {
		log.Printf("⚠️ [上下文合成] 宽召回无数据，启用项目上下文合成策略")
		// TODO: 实现基于项目结构、代码分析、git提交记录的合成逻辑
		return cse.synthesizeFromProjectContext(ctx, req)
	}

	// 调用LLM进行上下文合成
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   req.SynthesisConfig.MaxTokens,
		Temperature: req.SynthesisConfig.Temperature,
		Format:      "json",
		// 🔥 修复：从llmClient获取模型名称，不再硬编码
		Model: cse.llmClient.GetModel(),
		Metadata: map[string]interface{}{
			"task":     "context_synthesis",
			"strategy": "evaluation_and_synthesis",
		},
	}

	log.Printf("🚀 [上下文合成] 发送LLM请求...")
	log.Printf("📤 [LLM请求] 模型: %s, MaxTokens: %d, Temperature: %.2f",
		llmRequest.Model, llmRequest.MaxTokens, llmRequest.Temperature)

	// 🔥 调试：输出完整的prompt内容
	log.Printf("📝 [完整Prompt内容] ===========================================")
	log.Printf("%s", llmRequest.Prompt)
	log.Printf("📝 [完整Prompt结束] ===========================================")

	llmResponse, err := cse.llmClient.Complete(ctx, llmRequest)
	if err != nil {
		log.Printf("❌ [LLM请求] 调用失败: %v", err)
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	log.Printf("✅ [LLM响应] 调用成功，Token使用: %d", llmResponse.TokensUsed)
	// 🔥 调试：输出完整的响应内容
	log.Printf("📥 [完整响应内容] ===========================================")
	log.Printf("%s", llmResponse.Content)
	log.Printf("📥 [完整响应结束] ===========================================")

	// 解析LLM响应
	evaluationResult, err := cse.parseContextSynthesisResponse(llmResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	// 构建统一上下文和用户响应
	synthesizedContext, userResponse := cse.buildSynthesizedContext(ctx, llmResponse.Content, evaluationResult)
	if synthesizedContext == nil {
		return nil, fmt.Errorf("上下文构建失败: 缺少必需的上下文信息")
	}

	// 构建合成响应
	synthesisResp := &models.ContextSynthesisResponse{
		Success:            true,
		Message:            "上下文合成与评估完成",
		RequestID:          generateSynthesisRequestID(),
		ProcessTime:        time.Since(startTime).Milliseconds(),
		EvaluationResult:   evaluationResult,
		SynthesizedContext: synthesizedContext, // 🔥 使用真正构建的上下文
		UserResponse:       userResponse,       // 🔥 新增用户响应
		ResponseTime:       time.Now(),
	}

	log.Printf("🎯 [上下文合成] 评估完成 - 是否需要更新: %t, 置信度: %.2f",
		evaluationResult.ShouldUpdate, evaluationResult.UpdateConfidence)

	// 🔥 完善：记录合成上下文概要，确保LLM价值可追踪
	if synthesizedContext != nil {
		log.Printf("📋 [合成上下文] 会话: %s, 用户: %s, 工作空间: %s",
			synthesizedContext.SessionID, synthesizedContext.UserID, synthesizedContext.WorkspaceID)
		if synthesizedContext.CurrentTopic != nil {
			log.Printf("📊 [主题上下文] 主题: %s, 置信度: %.2f",
				synthesizedContext.CurrentTopic.MainTopic, synthesizedContext.CurrentTopic.ConfidenceLevel)
		}
		log.Printf("🕐 [时间戳] 创建: %v, 更新: %v",
			synthesizedContext.CreatedAt, synthesizedContext.UpdatedAt)
	}

	return synthesisResp, nil
}

// convertToIntentAnalysis 转换意图分析结果
func convertToIntentAnalysis(analysis *SemanticAnalysisResult) *models.WideRecallIntentAnalysis {
	if analysis == nil {
		return nil
	}

	return &models.WideRecallIntentAnalysis{
		IntentAnalysis: models.WideRecallIntentInfo{
			CoreIntent:      string(analysis.Intent),
			IntentType:      analysis.Intent,
			IntentCategory:  "technical", // 简化处理
			KeyConcepts:     analysis.Keywords,
			TimeScope:       "recent",
			UrgencyLevel:    models.PriorityMedium,
			ExpectedOutcome: "获取相关信息",
		},
		KeyExtraction: models.KeyExtraction{
			TechnicalKeywords: analysis.Keywords,
			ProjectKeywords:   []string{},
			BusinessKeywords:  []string{},
			TimeKeywords:      []string{},
			ActionKeywords:    []string{},
		},
		RetrievalStrategy: models.WideRecallStrategy{
			TimelineQueries:  convertToTimelineQueries(analysis.Queries.TimelineQueries),
			KnowledgeQueries: convertToKnowledgeQueries(analysis.Queries.KnowledgeQueries),
			VectorQueries:    convertToVectorQueries(analysis.Queries.VectorQueries),
		},
		ConfidenceLevel: analysis.Confidence,
		AnalysisTime:    time.Now(),
	}
}

// convertToRetrievalResults 转换检索结果
func convertToRetrievalResults(retrieval *RetrievalResults) *models.RetrievalResults {
	if retrieval == nil {
		return &models.RetrievalResults{}
	}

	// 转换时间线结果
	timelineResults := make([]models.TimelineResult, len(retrieval.TimelineResults))
	for i, event := range retrieval.TimelineResults {
		timelineResults[i] = models.TimelineResult{
			EventID:         event.ID,
			EventType:       event.EventType,
			Title:           event.Title,
			Content:         event.Content,
			Timestamp:       event.Timestamp,
			Source:          "timeline",
			ImportanceScore: 0.8,
			RelevanceScore:  0.8,
			Tags:            []string{},
			Metadata:        map[string]interface{}{},
		}
	}

	// 转换知识图谱结果
	knowledgeResults := make([]models.KnowledgeResult, len(retrieval.KnowledgeResults))
	for i, node := range retrieval.KnowledgeResults {
		knowledgeResults[i] = models.KnowledgeResult{
			ConceptID:       node.ID,
			ConceptName:     node.Name,
			ConceptType:     node.Type,
			Description:     node.Description,
			RelatedConcepts: []models.RelatedConcept{},
			Properties:      map[string]interface{}{},
			RelevanceScore:  0.8,
			ConfidenceScore: 0.8,
			Source:          "knowledge",
			LastUpdated:     time.Now(),
		}
	}

	// 转换向量结果
	vectorResults := make([]models.VectorResult, len(retrieval.VectorResults))
	for i, match := range retrieval.VectorResults {
		vectorResults[i] = models.VectorResult{
			DocumentID:      match.ID,
			Content:         match.Content,
			ContentType:     "text",
			Source:          "vector",
			Similarity:      match.Score,
			RelevanceScore:  match.Score,
			Timestamp:       time.Now(),
			Tags:            []string{},
			Metadata:        match.Metadata,
			MatchedSegments: []models.MatchedSegment{},
		}
	}

	return &models.RetrievalResults{
		TimelineResults:  timelineResults,
		TimelineCount:    len(timelineResults),
		TimelineStatus:   "success",
		KnowledgeResults: knowledgeResults,
		KnowledgeCount:   len(knowledgeResults),
		KnowledgeStatus:  "success",
		VectorResults:    vectorResults,
		VectorCount:      len(vectorResults),
		VectorStatus:     "success",
		TotalResults:     len(timelineResults) + len(knowledgeResults) + len(vectorResults),
		OverallQuality:   retrieval.OverallQuality,
		RetrievalTime:    retrieval.RetrievalTime,
		SuccessfulDims:   3,
	}
}

// getDefaultContentSynthesisConfig 获取默认配置
func getDefaultContentSynthesisConfig() *ContentSynthesisConfig {
	return &ContentSynthesisConfig{
		LLMTimeout:           60,   // 60秒
		MaxTokens:            8000, // 8000 tokens
		Temperature:          0.1,  // 低温度，更确定性
		ConfidenceThreshold:  0.7,  // 70%置信度阈值
		ConflictResolution:   "time_priority",
		InformationFusion:    "weighted_merge",
		QualityAssessment:    "comprehensive",
		UpdateThreshold:      0.4, // 40%更新阈值
		PersistenceThreshold: 0.7, // 70%持久化阈值
	}
}

// generateSynthesisRequestID 生成合成请求ID
func generateSynthesisRequestID() string {
	return fmt.Sprintf("cs_%d", time.Now().UnixNano())
}

// buildContextSynthesisPrompt 构建上下文合成Prompt（生成TopicContext和RecentChangesSummary）
func (cse *ContentSynthesisEngineImpl) buildContextSynthesisPrompt(req *models.ContextSynthesisRequest) string {
	// 构建检索结果信息
	retrievalResultsStr := cse.buildRetrievalResultsString(req.RetrievalResults)

	return fmt.Sprintf(`## 上下文分析与合成任务

你是一个专业的上下文分析专家，基于用户查询和检索到的相关信息，分析并提取核心的主题上下文信息。

### 用户查询
**用户问题**: %s

### 检索到的相关信息
%s

## 分析要求

### 1. TopicContext 分析（核心重点）
请深度分析用户的核心主题，包括：
- **MainTopic**: 用户关注的核心主题（简洁明确）
- **TopicCategory**: 主题分类（technical/project/business/learning/troubleshooting）
- **UserIntent**: 用户意图分析
  - IntentType: 意图类型（query/command/conversation/analysis/creation/modification）
  - IntentDescription: 意图的详细描述
  - Priority: 优先级（high/medium/low）
- **PrimaryPainPoint**: 用户的主要痛点问题
- **ExpectedOutcome**: 用户期望的结果
- **KeyConcepts**: 关键概念列表（每个概念包含名称和重要性0-1）

### 2. 变更感知分析（轻量化）
如果发现用户查询体现了明显的语义变化、需求变化或关键要素变化，请用一句话描述这种变化。如果没有明显变化，输出空字符串。

---

### 3. 用户响应生成（重要）
基于分析结果，生成高质量的用户响应：
- **用户意图理解**：准确理解用户真正想要什么，结合检索信息进行整合
- **解决方案提供**：提供实用、针对性的解决方案或答案

---

## 输出格式要求
请严格按照以下JSON格式输出，确保字段名称和结构完全匹配：

{
  "topic_context": {
    "main_topic": "用户关注的核心主题",
    "topic_category": "technical",
    "user_intent": {
      "intent_type": "query",
      "intent_description": "用户意图的详细描述",
      "priority": "medium"
    },
    "primary_pain_point": "用户的主要痛点",
    "expected_outcome": "用户期望的结果",
    "key_concepts": [
      {
        "concept_name": "概念名称",
        "importance": 0.8
      }
    ],
    "confidence_level": 0.8
  },
  "recent_changes_summary": "语义/需求/痛点变更的一句话描述，无变更则为空字符串",
  "user_response": {
    "user_intent": "用户真实意图分析 + 从宽召回多维信息中筛选的相关信息整合",
    "solution": "基于分析提供的实用针对性解决方案"
  }
}

请只返回JSON，不要包含其他文本。`, req.UserQuery, retrievalResultsStr)
}

// buildRetrievalResultsString 构建检索结果字符串
func (cse *ContentSynthesisEngineImpl) buildRetrievalResultsString(results *models.RetrievalResults) string {
	if results == nil {
		return "无检索结果"
	}

	resultStr := fmt.Sprintf(`**时间线检索结果** (%d条):
`, results.TimelineCount)

	// 显示前3条时间线结果
	for i, result := range results.TimelineResults {
		if i >= 3 {
			resultStr += "...\n"
			break
		}
		resultStr += fmt.Sprintf("- [%s] %s: %s\n",
			result.Timestamp.Format("2006-01-02 15:04"),
			result.EventType,
			result.Title)
	}

	resultStr += fmt.Sprintf(`
**知识图谱检索结果** (%d条):
`, results.KnowledgeCount)

	// 显示前3条知识图谱结果
	for i, result := range results.KnowledgeResults {
		if i >= 3 {
			resultStr += "...\n"
			break
		}
		resultStr += fmt.Sprintf("- %s (%s): %s\n",
			result.ConceptName,
			result.ConceptType,
			result.Description)
	}

	resultStr += fmt.Sprintf(`
**向量检索结果** (%d条):
`, results.VectorCount)

	// 显示前3条向量结果
	for i, result := range results.VectorResults {
		if i >= 3 {
			resultStr += "...\n"
			break
		}
		content := result.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		resultStr += fmt.Sprintf("- 相似度%.2f: %s\n",
			result.Similarity,
			content)
	}

	return resultStr
}

// parseContextSynthesisResponse 解析上下文合成响应
func (cse *ContentSynthesisEngineImpl) parseContextSynthesisResponse(content string) (*models.EvaluationResult, error) {
	// 清理响应内容
	content = strings.TrimSpace(content)

	// 尝试提取JSON部分
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	}

	content = strings.TrimSpace(content)

	// 解析JSON
	var rawResult struct {
		ShouldUpdate     bool    `json:"should_update"`
		UpdateConfidence float64 `json:"update_confidence"`
		SemanticChanges  []struct {
			Dimension      string   `json:"dimension"`
			ChangeType     string   `json:"change_type"`
			OldSemantic    string   `json:"old_semantic"`
			NewSemantic    string   `json:"new_semantic"`
			ChangeStrength float64  `json:"change_strength"`
			Evidence       []string `json:"evidence"`
		} `json:"semantic_changes"`
		InformationGaps []struct {
			Dimension            string   `json:"dimension"`
			MissingAspects       []string `json:"missing_aspects"`
			Importance           float64  `json:"importance"`
			CanFillFromRetrieval bool     `json:"can_fill_from_retrieval"`
		} `json:"information_gaps"`
		NewInformation []struct {
			Dimension   string  `json:"dimension"`
			Content     string  `json:"content"`
			Source      string  `json:"source"`
			Reliability float64 `json:"reliability"`
			Relevance   float64 `json:"relevance"`
		} `json:"new_information"`
		UpdateDimensions []string `json:"update_dimensions"`
		UpdateActions    []struct {
			ActionType        string `json:"action_type"`
			TargetDimension   string `json:"target_dimension"`
			ActionDescription string `json:"action_description"`
			Priority          int    `json:"priority"`
		} `json:"update_actions"`
		EvaluationReason  string `json:"evaluation_reason"`
		ConfidenceFactors []struct {
			Factor      string  `json:"factor"`
			Impact      float64 `json:"impact"`
			Description string  `json:"description"`
		} `json:"confidence_factors"`
		OverallConfidence float64 `json:"overall_confidence"`
	}

	if err := json.Unmarshal([]byte(content), &rawResult); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w, 内容: %s", err, content)
	}

	// 转换为简化的EvaluationResult结构
	result := &models.EvaluationResult{
		ShouldUpdate:     rawResult.ShouldUpdate,
		UpdateConfidence: rawResult.UpdateConfidence,
		EvaluationReason: rawResult.EvaluationReason,
		SemanticChanges:  make([]models.WideRecallSemanticChange, len(rawResult.SemanticChanges)),
	}

	// 转换语义变化
	for i, change := range rawResult.SemanticChanges {
		result.SemanticChanges[i] = models.WideRecallSemanticChange{
			Dimension:         change.Dimension,
			ChangeType:        change.ChangeType,
			ChangeDescription: fmt.Sprintf("%s -> %s", change.OldSemantic, change.NewSemantic),
			Evidence:          change.Evidence,
		}
	}

	return result, nil
}

// convertToTimelineQueries 转换时间线查询
func convertToTimelineQueries(queries []string) []models.TimelineQuery {
	result := make([]models.TimelineQuery, len(queries))
	for i, query := range queries {
		result[i] = models.TimelineQuery{
			QueryText:  query,
			TimeRange:  "recent",
			EventTypes: []string{"code_change", "discussion", "commit"},
			Priority:   3,
		}
	}
	return result
}

// convertToKnowledgeQueries 转换知识图谱查询
func convertToKnowledgeQueries(queries []string) []models.KnowledgeQuery {
	result := make([]models.KnowledgeQuery, len(queries))
	for i, query := range queries {
		result[i] = models.KnowledgeQuery{
			QueryText:     query,
			ConceptTypes:  []string{"技术概念", "最佳实践"},
			RelationTypes: []string{"实现", "优化"},
			Priority:      3,
		}
	}
	return result
}

// convertToVectorQueries 转换向量查询
func convertToVectorQueries(queries []string) []models.VectorQuery {
	result := make([]models.VectorQuery, len(queries))
	for i, query := range queries {
		result[i] = models.VectorQuery{
			QueryText:           query,
			SemanticFocus:       "技术实现",
			SimilarityThreshold: 0.7,
			Priority:            3,
		}
	}
	return result
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// synthesizeFromProjectContext 基于项目上下文合成响应（宽召回无数据时的降级策略）
func (cse *ContentSynthesisEngineImpl) synthesizeFromProjectContext(ctx context.Context, req *models.ContextSynthesisRequest) (*models.ContextSynthesisResponse, error) {
	log.Printf("🏗️ [项目上下文合成] 启动项目上下文合成策略...")

	// 构建基于项目的合成Prompt
	projectPrompt := cse.buildProjectContextPrompt(req)
	log.Printf("📝 [项目上下文合成] 项目Prompt构建完成，长度: %d", len(projectPrompt))

	// 调用LLM进行项目上下文合成
	llmRequest := &llm.LLMRequest{
		Prompt:      projectPrompt,
		MaxTokens:   req.SynthesisConfig.MaxTokens,
		Temperature: req.SynthesisConfig.Temperature,
		Format:      "json",
		// 🔥 修复：从llmClient获取模型名称，不再硬编码
		Model: cse.llmClient.GetModel(),
		Metadata: map[string]interface{}{
			"task":     "project_context_synthesis",
			"strategy": "project_based_fallback",
		},
	}

	log.Printf("🚀 [项目上下文合成] 发送LLM请求...")
	log.Printf("📤 [LLM请求-项目] 模型: %s, MaxTokens: %d", llmRequest.Model, llmRequest.MaxTokens)
	// 🔥 修复Prompt预览截断问题 - 显示完整内容
	log.Printf("📤 [LLM请求-项目] Prompt完整内容:\n%s", projectPrompt)

	llmResponse, err := cse.llmClient.Complete(ctx, llmRequest)
	if err != nil {
		log.Printf("❌ [项目上下文合成] LLM调用失败: %v", err)
		return nil, fmt.Errorf("项目上下文合成失败: %w", err)
	}

	log.Printf("✅ [项目上下文合成] LLM调用成功，Token使用: %d", llmResponse.TokensUsed)
	log.Printf("📥 [LLM响应-项目] 响应长度: %d字符", len(llmResponse.Content))
	log.Printf("📥 [LLM响应-项目] 原始内容: %s", llmResponse.Content[:min(500, len(llmResponse.Content))])

	// 解析项目上下文合成响应
	evaluationResult, err := cse.parseContextSynthesisResponse(llmResponse.Content)
	if err != nil {
		log.Printf("❌ [项目上下文合成] 响应解析失败: %v", err)
		return nil, fmt.Errorf("项目上下文响应解析失败: %w", err)
	}

	log.Printf("✅ [项目上下文合成] 解析成功，置信度: %.2f", evaluationResult.UpdateConfidence)

	return &models.ContextSynthesisResponse{
		Success:            true,
		Message:            "基于项目上下文的合成完成",
		RequestID:          generateSynthesisRequestID(),
		ProcessTime:        time.Since(req.RequestTime).Milliseconds(),
		EvaluationResult:   evaluationResult,
		SynthesizedContext: nil, // TODO: 实现项目上下文模型构建
		ResponseTime:       time.Now(),
	}, nil
}

// buildProjectContextPrompt 构建项目上下文合成Prompt
func (cse *ContentSynthesisEngineImpl) buildProjectContextPrompt(req *models.ContextSynthesisRequest) string {
	// 🔥 从CurrentContext中获取ProjectContext信息
	projectInfo := ""
	if req.CurrentContext != nil && req.CurrentContext.Project != nil {
		project := req.CurrentContext.Project

		// 构建项目信息描述 - 基于精简后的ProjectContext模型

		// 格式化技术栈信息
		techStackStr := "未分析"
		if len(project.TechStack) > 0 {
			techStackItems := make([]string, len(project.TechStack))
			for i, tech := range project.TechStack {
				if tech.Version != "" {
					techStackItems[i] = fmt.Sprintf("%s(%s)", tech.Name, tech.Version)
				} else {
					techStackItems[i] = tech.Name
				}
			}
			techStackStr = strings.Join(techStackItems, ", ")
		}

		// 格式化依赖信息
		dependenciesStr := "无依赖信息"
		if len(project.Dependencies) > 0 {
			dependenciesStr = fmt.Sprintf("%d个依赖项", len(project.Dependencies))
		}

		// 格式化组件信息
		componentsStr := "无组件信息"
		if len(project.MainComponents) > 0 {
			componentsStr = fmt.Sprintf("%d个主要组件", len(project.MainComponents))
		}

		// 格式化特性信息
		featuresStr := "无功能信息"
		if len(project.KeyFeatures) > 0 {
			featuresStr = fmt.Sprintf("%d个主要功能", len(project.KeyFeatures))
		}

		projectInfo = fmt.Sprintf(`## 🏗️ 工程感知信息

### 项目基础信息
- **项目名称**: %s
- **项目类型**: %s
- **主要语言**: %s
- **项目描述**: %s

### 技术栈信息  
- **技术栈**: %s
- **架构模式**: %s
- **依赖信息**: %s

### 项目结构
- **主要组件**: %s
- **关键模块**: %d个
- **重要文件**: %d个

### 项目状态
- **当前阶段**: %s
- **主要功能**: %s
- **完成进度**: %.1f%%
- **分析置信度**: %.1f%%`,
			project.ProjectName,
			string(project.ProjectType),
			project.PrimaryLanguage,
			project.Description,
			techStackStr,
			project.Architecture.Pattern,
			dependenciesStr,
			componentsStr,
			len(project.KeyModules),
			len(project.ImportantFiles),
			string(project.CurrentPhase),
			featuresStr,
			project.CompletionStatus.OverallProgress*100,
			project.ConfidenceLevel*100)
	} else {
		// 如果没有ProjectContext，使用默认信息
		projectInfo = `## 🏗️ 工程感知信息

### 项目基础信息
- **项目名称**: Context-Keeper上下文记忆管理系统
- **项目类型**: Go语言后端服务
- **主要语言**: Go
- **项目描述**: 智能上下文记忆管理系统

### 技术栈信息  
- **技术栈**: Go + Gin + TimescaleDB + Neo4j + 向量数据库
- **架构模式**: 分层架构 + LLM驱动
- **主要框架**: Gin Web框架
- **数据库**: TimescaleDB + Neo4j + Vearch

⚠️ **注意**: ProjectContext信息缺失，建议通过工程感知分析获取完整项目信息`
	}

	return fmt.Sprintf(`你是一个智能编程助手。用户查询了"%s"，但是从记忆中没有找到相关的历史信息。

请基于以下工程感知信息来回答用户的问题：

%s

## 🎯 当前用户意图分析
- **查询意图**: %s
- **用户查询**: %s

## ✅ 回答要求
1. **深度利用工程感知信息** - 基于项目的技术栈、架构、当前状态等信息回答
2. **具体可执行** - 提供具体的代码示例、配置建议或操作步骤
3. **结合当前痛点** - 如果查询与当前痛点相关，优先给出解决方案
4. **技术最佳实践** - 尽量结合项目的技术栈给出最佳实践建议
5. **保持准确性** - 确保回答与项目实际情况匹配

请以JSON格式返回：
{
  "should_update": true,
  "update_confidence": 0.8,
  "synthesis_result": "基于工程感知信息的详细回答内容",
  "reasoning": "基于项目实际情况的推理过程"
}`,
		req.UserQuery,
		projectInfo,
		getIntentFromAnalysis(req.IntentAnalysis),
		req.UserQuery)
}

// 辅助函数：从意图分析中提取信息
func getIntentFromAnalysis(analysis *models.WideRecallIntentAnalysis) string {
	if analysis != nil {
		return analysis.IntentAnalysis.CoreIntent
	}
	return "未知意图"
}

func getKeywordsFromAnalysis(analysis *models.WideRecallIntentAnalysis) []string {
	if analysis != nil {
		return analysis.KeyExtraction.TechnicalKeywords
	}
	return []string{}
}

func getConfidenceFromAnalysis(analysis *models.WideRecallIntentAnalysis) float64 {
	if analysis != nil {
		return analysis.ConfidenceLevel
	}
	return 0.5
}

// 🔥 新增：从LLM合成响应中提取指定维度的内容
func (cse *ContentSynthesisEngineImpl) extractContentByDimension(synthesisResp *models.ContextSynthesisResponse, dimension string) string {
	if synthesisResp == nil {
		return generateFallbackContent(dimension)
	}

	// 🔥 修复：优先从UserResponse中提取实际内容
	if synthesisResp.UserResponse != nil {
		switch dimension {
		case "short_term_memory":
			// 使用用户意图分析作为短期记忆
			if synthesisResp.UserResponse.UserIntent != "" {
				log.Printf("✅ [内容提取] 维度 %s 提取到LLM合成内容: %s", dimension, synthesisResp.UserResponse.UserIntent[:min(100, len(synthesisResp.UserResponse.UserIntent))])
				return synthesisResp.UserResponse.UserIntent
			}
		case "long_term_memory":
			// 使用解决方案作为长期记忆
			if synthesisResp.UserResponse.Solution != "" {
				log.Printf("✅ [内容提取] 维度 %s 提取到LLM合成内容: %s", dimension, synthesisResp.UserResponse.Solution[:min(100, len(synthesisResp.UserResponse.Solution))])
				return synthesisResp.UserResponse.Solution
			}
		case "relevant_knowledge":
			// 合并用户意图和解决方案作为相关知识
			var contentParts []string
			if synthesisResp.UserResponse.UserIntent != "" {
				contentParts = append(contentParts, fmt.Sprintf("🎯 意图分析: %s", synthesisResp.UserResponse.UserIntent))
			}
			if synthesisResp.UserResponse.Solution != "" {
				contentParts = append(contentParts, fmt.Sprintf("💡 解决方案: %s", synthesisResp.UserResponse.Solution))
			}
			if len(contentParts) > 0 {
				result := strings.Join(contentParts, "\n\n")
				log.Printf("✅ [内容提取] 维度 %s 提取到LLM合成内容: %s", dimension, result[:min(100, len(result))])
				return result
			}
		}
	}

	// 🔥 兜底：从EvaluationResult中提取内容（原有逻辑保留）
	if synthesisResp.EvaluationResult != nil {
		var contentParts []string

		// 如果有语义变化，添加变化描述
		for _, change := range synthesisResp.EvaluationResult.SemanticChanges {
			if change.Dimension == dimension || dimension == "relevant_knowledge" {
				contentParts = append(contentParts, change.ChangeDescription)
			}
		}

		// 通过评估原因提取相关内容
		if synthesisResp.EvaluationResult.EvaluationReason != "" {
			switch dimension {
			case "short_term_memory":
				contentParts = append(contentParts, fmt.Sprintf("最新上下文评估: %s", synthesisResp.EvaluationResult.EvaluationReason))
			case "long_term_memory":
				if synthesisResp.EvaluationResult.ShouldUpdate {
					contentParts = append(contentParts, fmt.Sprintf("需要长期记忆更新: %s", synthesisResp.EvaluationResult.EvaluationReason))
				}
			case "relevant_knowledge":
				contentParts = append(contentParts, fmt.Sprintf("知识评估结果: %s", synthesisResp.EvaluationResult.EvaluationReason))
			}
		}

		// 如果有内容，合并返回
		if len(contentParts) > 0 {
			result := strings.Join(contentParts, "; ")
			log.Printf("✅ [内容提取] 维度 %s 提取到LLM合成内容: %s", dimension, result[:min(100, len(result))])
			return result
		}
	}

	fallback := generateFallbackContent(dimension)
	log.Printf("⚠️ [内容提取] 维度 %s 无LLM内容，使用后备内容: %s", dimension, fallback)
	return fallback
}

// 生成后备内容（当LLM合成失败时使用）
func generateFallbackContent(dimension string) string {
	switch dimension {
	case "short_term_memory":
		return "暂未找到短期记忆"
	case "long_term_memory":
		return "暂未找到长期记忆摘要"
	case "relevant_knowledge":
		return "未检索相关知识"
	default:
		return "未检索到内容摘要"
	}
}

// 🔥 新增：构建真正的合成上下文

// buildSynthesizedContext 从LLM输出构建结构化上下文和用户响应
func (cse *ContentSynthesisEngineImpl) buildSynthesizedContext(
	ctx context.Context,
	llmContent string,
	evaluationResult *models.EvaluationResult,
) (*models.UnifiedContextModel, *models.UserResponseSynthesis) {
	log.Printf("🔧 [上下文构建] 开始构建SynthesizedContext")

	// 🔥 从上下文中获取基础信息
	// 注意：统一拦截器注入的字段名与期望不同，需要适配
	sessionID, _ := ctx.Value("session_id").(string)
	userID, _ := ctx.Value("user_id").(string)
	workspaceID, _ := ctx.Value("workspacePath").(string)

	// 检查基础信息是否完整 - sessionID 可以从请求中获取，不强制要求从context
	if userID == "" || workspaceID == "" {
		log.Printf("⚠️ [上下文构建] 基础信息不完整: sessionID=%s, userID=%s, workspaceID=%s",
			sessionID, userID, workspaceID)
		return nil, nil
	}

	// 🆕 获取工程感知信息
	projectAnalysis, _ := ctx.Value("project_analysis").(string)

	// 构建ProjectContext（工程感知独立性）
	var projectContext *models.ProjectContext
	if projectAnalysis != "" {
		projectContext = cse.buildProjectContextFromAnalysis(projectAnalysis, workspaceID)
		log.Printf("🔧 [工程感知] 已构建ProjectContext，项目: %s", projectContext.ProjectName)
	}

	// 🔥 从LLM JSON输出解析TopicContext、RecentChangesSummary和UserResponse
	topicContext, recentChangesSummary, userResponse, err := cse.parseContextSynthesisJSON(llmContent)
	if err != nil {
		log.Printf("❌ [上下文构建] TopicContext解析失败: %v", err)
		log.Printf("🔧 [容错处理] 创建包含ProjectContext的最小统一上下文")

		unified := &models.UnifiedContextModel{
			// === 核心标识 ===
			SessionID:   sessionID,
			UserID:      userID,
			WorkspaceID: workspaceID,

			// === 项目上下文（工程感知保留）===
			Project: projectContext, // 🔥 关键：保留ProjectContext

			// === 时间戳 ===
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// 返回最小上下文和空的用户响应
		return unified, nil
	}

	// 构建基础UnifiedContextModel
	unified := &models.UnifiedContextModel{
		// === 核心标识 ===
		SessionID:   sessionID,
		UserID:      userID,
		WorkspaceID: workspaceID,

		// === 当前主题（核心）===
		CurrentTopic: topicContext,

		// === 项目上下文（工程感知）===
		Project: projectContext,

		// === 最近变更描述 ===
		RecentChangesSummary: recentChangesSummary, // 🔥 从LLM解析获取

		// === 时间戳 ===
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Printf("✅ [上下文构建] SynthesizedContext构建完成 - 主题: %s",
		topicContext.MainTopic)

	return unified, userResponse
}

// buildProjectContextFromAnalysis 从工程分析结果构建ProjectContext
func (cse *ContentSynthesisEngineImpl) buildProjectContextFromAnalysis(
	projectAnalysis string,
	workspaceID string,
) *models.ProjectContext {
	// 简单解析工程分析结果，构建ProjectContext
	// 这里可以根据实际的analysisPrompt格式进行更复杂的解析

	return &models.ProjectContext{
		ProjectName:     extractProjectName(workspaceID),          // 从工作空间ID提取项目名
		ProjectPath:     workspaceID,                              // 工作空间路径
		Description:     projectAnalysis,                          // 工程分析作为描述
		PrimaryLanguage: extractPrimaryLanguage(projectAnalysis),  // 主要编程语言
		TechStack:       extractTechStack(projectAnalysis),        // 技术栈
		Architecture:    extractArchitectureInfo(projectAnalysis), // 架构信息
		Dependencies:    extractDependencyInfo(projectAnalysis),   // 依赖信息
		LastAnalyzed:    time.Now(),                               // 最后分析时间
		ConfidenceLevel: 0.7,                                      // 默认置信度
	}
}

// 辅助函数：从工作空间ID提取项目名
func extractProjectName(workspaceID string) string {
	// 简单实现：从路径中提取最后一段作为项目名
	parts := strings.Split(workspaceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-project"
}

// 辅助函数：从分析结果中提取主要编程语言
func extractPrimaryLanguage(analysis string) string {
	// 简单实现：查找最可能的主要语言
	analysisLower := strings.ToLower(analysis)

	if strings.Contains(analysisLower, "go") || strings.Contains(analysisLower, "golang") {
		return "Go"
	} else if strings.Contains(analysisLower, "python") {
		return "Python"
	} else if strings.Contains(analysisLower, "node.js") || strings.Contains(analysisLower, "javascript") {
		return "JavaScript"
	} else if strings.Contains(analysisLower, "java") {
		return "Java"
	}

	return "unknown"
}

// 辅助函数：从分析结果中提取技术栈信息
func extractTechStack(analysis string) []models.TechStackItem {
	// 简单实现：查找常见技术关键词并构建TechStackItem
	techStack := []models.TechStackItem{}
	keywords := map[string]string{
		"go": "language", "python": "language", "node.js": "runtime",
		"react": "frontend", "vue": "frontend", "gin": "framework",
		"docker": "containerization", "kubernetes": "orchestration",
		"redis": "cache", "mysql": "database", "postgresql": "database",
	}

	analysisLower := strings.ToLower(analysis)
	for tech, techType := range keywords {
		if strings.Contains(analysisLower, tech) {
			techStack = append(techStack, models.TechStackItem{
				Name:       tech,
				Type:       techType,
				Version:    "unknown",
				Importance: 0.8,
			})
		}
	}

	return techStack
}

// 辅助函数：从分析结果中提取架构信息
func extractArchitectureInfo(analysis string) models.ArchitectureInfo {
	// 简单实现：查找架构关键词
	analysisLower := strings.ToLower(analysis)

	if strings.Contains(analysisLower, "microservice") {
		return models.ArchitectureInfo{
			Pattern:     "microservices",
			Layers:      []string{"API层", "服务层", "数据层"},
			Components:  []string{"API网关", "服务注册", "配置中心"},
			Description: "微服务架构",
		}
	} else if strings.Contains(analysisLower, "monolith") {
		return models.ArchitectureInfo{
			Pattern:     "monolithic",
			Layers:      []string{"表示层", "业务层", "数据层"},
			Components:  []string{"Web服务器", "应用服务器", "数据库"},
			Description: "单体架构",
		}
	}

	return models.ArchitectureInfo{
		Pattern:     "unknown",
		Layers:      []string{"unknown"},
		Components:  []string{"unknown"},
		Description: "未知架构",
	}
}

// 辅助函数：从分析结果中提取依赖信息
func extractDependencyInfo(analysis string) []models.DependencyInfo {
	// 简单实现：根据分析内容构建依赖信息
	dependencies := []models.DependencyInfo{}

	analysisLower := strings.ToLower(analysis)
	if strings.Contains(analysisLower, "gin") {
		dependencies = append(dependencies, models.DependencyInfo{
			Name:        "gin",
			Version:     "unknown",
			Type:        "framework",
			Description: "Go web框架",
		})
	}
	if strings.Contains(analysisLower, "mysql") {
		dependencies = append(dependencies, models.DependencyInfo{
			Name:        "mysql",
			Version:     "unknown",
			Type:        "database",
			Description: "关系型数据库",
		})
	}

	return dependencies
}

// parseContextSynthesisJSON 解析LLM输出的上下文合成JSON
func (cse *ContentSynthesisEngineImpl) parseContextSynthesisJSON(llmContent string) (*models.TopicContext, string, *models.UserResponseSynthesis, error) {
	// 🔥 定义与LLM输出对应的JSON结构
	type LLMTopicContext struct {
		MainTopic     string `json:"main_topic"`
		TopicCategory string `json:"topic_category"`
		UserIntent    struct {
			IntentType        string `json:"intent_type"`
			IntentDescription string `json:"intent_description"`
			Priority          string `json:"priority"`
		} `json:"user_intent"`
		PrimaryPainPoint string `json:"primary_pain_point"`
		ExpectedOutcome  string `json:"expected_outcome"`
		KeyConcepts      []struct {
			ConceptName string  `json:"concept_name"`
			Importance  float64 `json:"importance"`
		} `json:"key_concepts"`
		ConfidenceLevel float64 `json:"confidence_level"`
	}

	type LLMUserResponse struct {
		UserIntent string `json:"user_intent"`
		Solution   string `json:"solution"`
	}

	type LLMContextSynthesis struct {
		TopicContext         LLMTopicContext `json:"topic_context"`
		RecentChangesSummary string          `json:"recent_changes_summary"`
		UserResponse         LLMUserResponse `json:"user_response"`
	}

	// 🔥 清理markdown代码块标记
	content := strings.TrimSpace(llmContent)

	// 移除```json和```标记
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	}

	content = strings.TrimSpace(content)
	log.Printf("🔧 [JSON清理] 清理后内容: %s", content[:min(200, len(content))])

	// 🔥 解析JSON
	var llmResult LLMContextSynthesis
	if err := json.Unmarshal([]byte(content), &llmResult); err != nil {
		return nil, "", nil, fmt.Errorf("JSON解析失败: %w, 清理后内容: %s", err, content[:min(300, len(content))])
	}

	// 🔥 直接使用LLM输出的字符串值，无需映射
	topicCategory := models.TopicCategory(llmResult.TopicContext.TopicCategory)
	intentType := models.IntentType(llmResult.TopicContext.UserIntent.IntentType)
	priority := models.Priority(llmResult.TopicContext.UserIntent.Priority)

	// 构建UserIntent
	userIntent := models.UserIntent{
		IntentType:        intentType,
		IntentDescription: llmResult.TopicContext.UserIntent.IntentDescription,
		ActionRequired:    []models.ActionItem{},      // 本次不包含
		InformationNeeded: []models.InformationNeed{}, // 本次不包含
		Priority:          priority,
	}

	// 构建KeyConcepts
	var keyConcepts []models.ConceptInfo
	for _, concept := range llmResult.TopicContext.KeyConcepts {
		keyConcepts = append(keyConcepts, models.ConceptInfo{
			ConceptName: concept.ConceptName,
			Importance:  concept.Importance,
		})
	}

	// 🔥 构建完整的TopicContext
	topicContext := &models.TopicContext{
		// === 核心主题信息 ===
		MainTopic:     llmResult.TopicContext.MainTopic,
		TopicCategory: topicCategory,
		UserIntent:    userIntent,

		// === 痛点和需求 ===
		PrimaryPainPoint:    llmResult.TopicContext.PrimaryPainPoint,
		SecondaryPainPoints: []string{}, // 本次不包含
		ExpectedOutcome:     llmResult.TopicContext.ExpectedOutcome,

		// === 上下文关键词 ===
		KeyConcepts:    keyConcepts,
		TechnicalTerms: []models.TechnicalTerm{}, // 本次不包含
		BusinessTerms:  []models.BusinessTerm{},  // 本次不包含

		// === 话题演进 ===
		TopicEvolution: []models.TopicEvolutionStep{}, // 本次不包含
		RelatedTopics:  []models.RelatedTopic{},       // 本次不包含

		// === 元数据 ===
		TopicStartTime:  time.Now(),
		LastUpdated:     time.Now(),
		UpdateCount:     1,
		ConfidenceLevel: llmResult.TopicContext.ConfidenceLevel,
	}

	// 🔥 构建UserResponseSynthesis
	userResponse := &models.UserResponseSynthesis{
		UserIntent: llmResult.UserResponse.UserIntent,
		Solution:   llmResult.UserResponse.Solution,
	}

	return topicContext, llmResult.RecentChangesSummary, userResponse, nil
}
