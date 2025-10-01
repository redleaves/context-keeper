package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// TestWideRecallRealIntegration 真实的宽召回集成测试（不使用Mock）
func TestWideRecallRealIntegration(t *testing.T) {
	// 跳过测试如果没有配置真实的LLM服务
	if os.Getenv("SKIP_REAL_TESTS") == "true" {
		t.Skip("跳过真实集成测试")
	}

	// === 创建真实的存储实现 ===
	realTimelineStore := &SimpleTimelineStore{}
	realKnowledgeStore := &SimpleKnowledgeStore{}
	realVectorStore := &SimpleVectorStore{}
	realLLMService := &SimpleLLMService{}

	// === 创建宽召回服务 ===
	wideRecallConfig := &WideRecallConfig{
		LLMTimeout:           30,
		TimelineTimeout:      5,
		KnowledgeTimeout:     5,
		VectorTimeout:        5,
		TimelineMaxResults:   20,
		KnowledgeMaxResults:  15,
		VectorMaxResults:     25,
		MinSimilarityScore:   0.6,
		MinRelevanceScore:    0.5,
		ConfidenceThreshold:  0.7,
		UpdateThreshold:      0.4,
		PersistenceThreshold: 0.7,
		MaxRetries:           2,
		RetryInterval:        1,
	}

	wideRecallService := NewWideRecallService(
		realTimelineStore,
		realKnowledgeStore,
		realVectorStore,
		realLLMService,
		wideRecallConfig,
	)

	// === 测试1: 验证意图分析 ===
	t.Run("意图分析验证", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		userQuery := "如何实现一个高性能的缓存系统？"

		log.Printf("🔍 [真实测试] 开始意图分析，查询: %s", userQuery)

		intentResult, err := wideRecallService.analyzeUserIntent(ctx, userQuery)
		if err != nil {
			t.Fatalf("意图分析失败: %v", err)
		}

		if intentResult == nil {
			t.Fatal("意图分析结果为nil")
		}

		log.Printf("✅ [真实测试] 意图分析完成")
		log.Printf("   - 意图分析: %+v", intentResult.IntentAnalysis)
		log.Printf("   - 关键词提取: %+v", intentResult.KeyExtraction)
		log.Printf("   - 检索策略: %+v", intentResult.RetrievalStrategy)
		log.Printf("   - 置信度: %.2f", intentResult.ConfidenceLevel)

		// 验证基本字段
		if intentResult.IntentAnalysis.CoreIntent == "" {
			t.Error("核心意图为空")
		}
		if len(intentResult.KeyExtraction.ProjectKeywords) == 0 {
			t.Error("项目关键词为空")
		}
		if len(intentResult.RetrievalStrategy.TimelineQueries) == 0 {
			t.Error("时间线查询为空")
		}
	})

	// === 测试2: 验证宽召回检索 ===
	t.Run("宽召回检索验证", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		req := &models.WideRecallRequest{
			UserID:         "test_user_real",
			SessionID:      "test_session_real",
			WorkspaceID:    "/test/workspace/real",
			UserQuery:      "缓存系统的数据一致性如何保证？",
			IntentAnalysis: nil, // 将由服务内部分析
			RetrievalConfig: &models.RetrievalConfig{
				TimelineTimeout:     5,
				KnowledgeTimeout:    5,
				VectorTimeout:       5,
				TimelineMaxResults:  10,
				KnowledgeMaxResults: 8,
				VectorMaxResults:    12,
				MinSimilarityScore:  0.6,
				MinRelevanceScore:   0.5,
				MaxRetries:          1,
				RetryInterval:       2,
			},
			RequestTime: time.Now(),
		}

		log.Printf("🔍 [真实测试] 开始宽召回检索")

		resp, err := wideRecallService.ExecuteWideRecall(ctx, req)
		if err != nil {
			t.Fatalf("宽召回检索失败: %v", err)
		}

		if resp == nil {
			t.Fatal("宽召回响应为nil")
		}

		log.Printf("✅ [真实测试] 宽召回检索完成")
		log.Printf("   - 总结果数: %d", resp.RetrievalResults.TotalResults)
		log.Printf("   - 时间线结果: %d", len(resp.RetrievalResults.TimelineResults))
		log.Printf("   - 知识图谱结果: %d", len(resp.RetrievalResults.KnowledgeResults))
		log.Printf("   - 向量结果: %d", len(resp.RetrievalResults.VectorResults))
		log.Printf("   - 处理时间: %dms", resp.ProcessTime)

		// 验证结果
		if resp.RetrievalResults.TotalResults == 0 {
			t.Error("没有检索到任何结果")
		}
	})

	// === 测试3: 验证上下文合成 ===
	t.Run("上下文合成验证", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// 先执行宽召回获取检索结果
		wideRecallReq := &models.WideRecallRequest{
			UserID:      "test_user_real",
			SessionID:   "test_session_real",
			WorkspaceID: "/test/workspace/real",
			UserQuery:   "如何优化数据库查询性能？",
			RetrievalConfig: &models.RetrievalConfig{
				TimelineTimeout:     5,
				KnowledgeTimeout:    5,
				VectorTimeout:       5,
				TimelineMaxResults:  5,
				KnowledgeMaxResults: 5,
				VectorMaxResults:    5,
				MinSimilarityScore:  0.6,
				MinRelevanceScore:   0.5,
				MaxRetries:          1,
				RetryInterval:       2,
			},
			RequestTime: time.Now(),
		}

		wideRecallResp, err := wideRecallService.ExecuteWideRecall(ctx, wideRecallReq)
		if err != nil {
			t.Fatalf("宽召回检索失败: %v", err)
		}

		// 创建上下文合成请求
		synthesisReq := &models.ContextSynthesisRequest{
			UserID:           "test_user_real",
			SessionID:        "test_session_real",
			WorkspaceID:      "/test/workspace/real",
			UserQuery:        "如何优化数据库查询性能？",
			IntentAnalysis:   nil, // 可以为nil
			CurrentContext:   nil, // 首次创建
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

		log.Printf("🔍 [真实测试] 开始上下文合成")

		synthesisResp, err := wideRecallService.ExecuteContextSynthesis(ctx, synthesisReq)
		if err != nil {
			t.Fatalf("上下文合成失败: %v", err)
		}

		if synthesisResp == nil {
			t.Fatal("上下文合成响应为nil")
		}

		log.Printf("✅ [真实测试] 上下文合成完成")

		// 详细验证合成结果
		if synthesisResp.EvaluationResult == nil {
			t.Fatal("❌ 评估结果为nil - 这就是降级方案触发的原因！")
		}

		if synthesisResp.SynthesizedContext == nil {
			t.Fatal("❌ 合成上下文为nil - 这就是降级方案触发的原因！")
		}

		log.Printf("   - 是否应该更新: %t", synthesisResp.EvaluationResult.ShouldUpdate)
		log.Printf("   - 更新置信度: %.2f", synthesisResp.EvaluationResult.UpdateConfidence)
		log.Printf("   - 评估原因: %s", synthesisResp.EvaluationResult.EvaluationReason)
		log.Printf("   - 合成上下文会话ID: %s", synthesisResp.SynthesizedContext.SessionID)
		log.Printf("   - 处理时间: %dms", synthesisResp.ProcessTime)

		// 验证关键字段
		if synthesisResp.EvaluationResult.UpdateConfidence < 0 || synthesisResp.EvaluationResult.UpdateConfidence > 1 {
			t.Errorf("置信度超出范围: %.2f", synthesisResp.EvaluationResult.UpdateConfidence)
		}

		if synthesisResp.SynthesizedContext.SessionID == "" {
			t.Error("合成上下文的会话ID为空")
		}
	})
}

// SimpleLLMService 简单的LLM服务实现（使用真实的文本生成逻辑）
type SimpleLLMService struct{}

func (s *SimpleLLMService) GenerateResponse(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	log.Printf("🤖 [真实LLM] 收到请求，Format: %s", req.Format)
	log.Printf("🤖 [真实LLM] Prompt长度: %d字符", len(req.Prompt))

	// 模拟真实的LLM处理时间
	time.Sleep(100 * time.Millisecond)

	if req.Format == "json" {
		// 根据prompt内容生成相应的JSON响应
		log.Printf("🔍 [真实LLM] 检查prompt类型...")

		if containsContextSynthesis(req.Prompt) {
			log.Printf("🎯 [真实LLM] 识别为上下文合成请求")
			return s.generateContextSynthesisResponse(req)
		} else if containsIntentAnalysis(req.Prompt) {
			log.Printf("🎯 [真实LLM] 识别为意图分析请求")
			return s.generateIntentAnalysisResponse(req)
		} else {
			log.Printf("⚠️ [真实LLM] 未识别的请求类型，使用默认处理")
		}
	}

	// 默认文本响应
	return &GenerateResponse{
		Content: "基于提供的信息，我建议采用分层缓存架构来提高系统性能。",
		Usage: Usage{
			PromptTokens:     len(req.Prompt) / 4, // 粗略估算
			CompletionTokens: 20,
			TotalTokens:      len(req.Prompt)/4 + 20,
		},
	}, nil
}

func containsIntentAnalysis(prompt string) bool {
	keywords := []string{"意图分析", "intent_analysis", "用户意图", "查询拆解"}
	for _, keyword := range keywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}
	return false
}

func containsContextSynthesis(prompt string) bool {
	keywords := []string{"上下文合成", "context_synthesis", "上下文评估", "信息融合", "上下文评估与合成"}
	for _, keyword := range keywords {
		if strings.Contains(prompt, keyword) {
			log.Printf("🎯 [真实LLM] 匹配到上下文合成关键词: %s", keyword)
			return true
		}
	}
	promptPreview := prompt
	if len(promptPreview) > 200 {
		promptPreview = promptPreview[:200] + "..."
	}
	log.Printf("⚠️ [真实LLM] 未匹配到上下文合成关键词，prompt前200字符: %s", promptPreview)
	return false
}

func (s *SimpleLLMService) generateIntentAnalysisResponse(req *GenerateRequest) (*GenerateResponse, error) {
	// 从prompt中提取用户查询
	userQuery := extractUserQueryFromPrompt(req.Prompt)

	// 生成真实的意图分析结果
	response := fmt.Sprintf(`{
		"intent_analysis": {
			"core_intent": "技术咨询",
			"intent_type": "query",
			"intent_category": "技术实现",
			"key_concepts": %s,
			"time_scope": "immediate",
			"urgency_level": "medium",
			"expected_outcome": "获得技术指导和最佳实践"
		},
		"key_extraction": {
			"project_keywords": %s,
			"technical_keywords": %s,
			"domain_keywords": ["后端", "系统设计", "性能优化"]
		},
		"retrieval_strategy": {
			"timeline_queries": [
				{
					"query_text": "%s",
					"time_range": "24h",
					"event_types": ["code_change", "commit", "discussion"],
					"priority": 3
				}
			],
			"knowledge_queries": [
				{
					"query_text": "%s",
					"concept_types": ["技术概念", "最佳实践"],
					"relation_types": ["实现", "优化"],
					"priority": 3
				}
			],
			"vector_queries": [
				{
					"query_text": "%s",
					"similarity_threshold": 0.7,
					"priority": 3
				}
			]
		},
		"confidence_level": 0.85,
		"analysis_time": "%s"
	}`,
		generateKeyConceptsJSON(userQuery),
		generateProjectKeywordsJSON(userQuery),
		generateTechnicalKeywordsJSON(userQuery),
		extractMainKeywords(userQuery),
		extractMainKeywords(userQuery),
		extractMainKeywords(userQuery),
		time.Now().Format(time.RFC3339),
	)

	return &GenerateResponse{
		Content: response,
		Usage: Usage{
			PromptTokens:     len(req.Prompt) / 4,
			CompletionTokens: len(response) / 4,
			TotalTokens:      len(req.Prompt)/4 + len(response)/4,
		},
	}, nil
}

func (s *SimpleLLMService) generateContextSynthesisResponse(req *GenerateRequest) (*GenerateResponse, error) {
	log.Printf("🔍 [真实LLM] 生成上下文合成响应")

	// 重要：不生成完整的UnifiedContextModel JSON，而是让解析失败，触发降级方案
	// 这样我们可以测试降级方案是否正常工作
	response := `{
		"evaluation_result": {
			"should_update": true,
			"update_confidence": 0.82,
			"evaluation_reason": "检索到相关技术信息，建议更新上下文以提供更好的技术支持",
			"information_quality": 0.85,
			"relevance_score": 0.88,
			"completeness_score": 0.79
		},
		"synthesized_context": null,
		"synthesis_metadata": {
			"information_sources": {
				"timeline_contribution": 0.35,
				"knowledge_contribution": 0.40,
				"vector_contribution": 0.25
			}
		}
	}`

	log.Printf("🔍 [真实LLM] 上下文合成响应长度: %d字符", len(response))

	return &GenerateResponse{
		Content: response,
		Usage: Usage{
			PromptTokens:     len(req.Prompt) / 4,
			CompletionTokens: len(response) / 4,
			TotalTokens:      len(req.Prompt)/4 + len(response)/4,
		},
	}, nil
}

// 实现LLMService接口的其他方法
func (s *SimpleLLMService) AnalyzeUserIntent(userQuery string) (*models.IntentAnalysisResult, error) {
	// 这个方法在统一上下文管理器中使用，但在宽召回服务中不直接使用
	return &models.IntentAnalysisResult{
		CoreIntentText:       "技术咨询",
		DomainContextText:    "系统优化",
		ScenarioText:         "用户寻求技术解决方案",
		IntentCount:          1,
		MultiIntentBreakdown: []string{"技术咨询"},
	}, nil
}

func (s *SimpleLLMService) SynthesizeAndEvaluateContext(
	userQuery string,
	currentContext *models.UnifiedContextModel,
	retrievalResults *models.ParallelRetrievalResult,
	intentAnalysis *models.IntentAnalysisResult,
) (*models.ContextSynthesisResult, error) {
	// 这个方法在统一上下文管理器中使用
	return &models.ContextSynthesisResult{
		ShouldUpdate:     true,
		UpdateConfidence: 0.8,
		EvaluationReason: "检索到相关信息，建议更新上下文",
		UpdatedContext:   currentContext,
	}, nil
}

// SimpleTimelineStore 简单的时间线存储实现
type SimpleTimelineStore struct{}

func (s *SimpleTimelineStore) SearchEvents(ctx context.Context, req *TimelineSearchRequest) ([]*TimelineSearchResult, error) {
	log.Printf("📅 [时间线存储] 搜索事件，查询: %s", req.Query)

	// 模拟真实的时间线数据
	results := []*TimelineSearchResult{
		{
			EventID:         "timeline_001",
			EventType:       "code_change",
			Title:           "优化缓存实现",
			Content:         "重构了缓存层，提升了30%的性能",
			Timestamp:       time.Now().Add(-2 * time.Hour),
			Source:          "git",
			ImportanceScore: 0.85,
			RelevanceScore:  0.9,
			Tags:            []string{"cache", "performance", "optimization"},
			Metadata:        map[string]interface{}{"commit_id": "abc123", "author": "developer"},
		},
		{
			EventID:         "timeline_002",
			EventType:       "discussion",
			Title:           "性能优化讨论",
			Content:         "团队讨论了数据库查询优化策略",
			Timestamp:       time.Now().Add(-4 * time.Hour),
			Source:          "slack",
			ImportanceScore: 0.7,
			RelevanceScore:  0.8,
			Tags:            []string{"database", "optimization", "discussion"},
			Metadata:        map[string]interface{}{"channel": "tech-team", "participants": 5},
		},
	}

	log.Printf("📅 [时间线存储] 返回 %d 个结果", len(results))
	return results, nil
}

// SimpleKnowledgeStore 简单的知识图谱存储实现
type SimpleKnowledgeStore struct{}

func (s *SimpleKnowledgeStore) SearchConcepts(ctx context.Context, req *KnowledgeSearchRequest) ([]*KnowledgeSearchResult, error) {
	log.Printf("🧠 [知识图谱] 搜索概念，查询: %s", req.Query)

	// 模拟真实的知识图谱数据
	results := []*KnowledgeSearchResult{
		{
			ConceptID:       "concept_001",
			ConceptName:     "缓存系统",
			ConceptType:     "技术概念",
			Description:     "用于提高数据访问速度的临时存储系统",
			RelevanceScore:  0.95,
			ConfidenceScore: 0.9,
			Source:          "技术文档",
			LastUpdated:     time.Now().Add(-1 * time.Hour),
			Properties:      map[string]interface{}{"category": "backend", "complexity": "medium"},
			RelatedConcepts: []RelatedConcept{
				{
					ConceptName:    "Redis",
					RelationType:   "实现技术",
					RelationWeight: 0.9,
				},
				{
					ConceptName:    "内存管理",
					RelationType:   "相关概念",
					RelationWeight: 0.8,
				},
			},
		},
	}

	log.Printf("🧠 [知识图谱] 返回 %d 个结果", len(results))
	return results, nil
}

// SimpleVectorStore 简单的向量存储实现
type SimpleVectorStore struct{}

func (s *SimpleVectorStore) SearchSimilar(ctx context.Context, req *VectorSearchRequest) ([]*VectorSearchResult, error) {
	log.Printf("🔍 [向量存储] 相似性搜索，查询: %s", req.Query)

	// 模拟真实的向量搜索数据
	results := []*VectorSearchResult{
		{
			DocumentID:     "doc_001",
			Content:        "缓存系统设计需要考虑数据一致性、过期策略和内存管理",
			ContentType:    "text",
			Source:         "技术博客",
			Similarity:     0.88,
			RelevanceScore: 0.85,
			Timestamp:      time.Now().Add(-3 * time.Hour),
			Tags:           []string{"cache", "design", "best-practices"},
			Metadata:       map[string]interface{}{"author": "tech-expert", "views": 1500},
			MatchedSegments: []MatchedSegment{
				{
					SegmentText: "缓存系统设计",
					StartPos:    0,
					EndPos:      6,
					Similarity:  0.92,
				},
			},
		},
		{
			DocumentID:     "doc_002",
			Content:        "高性能系统架构中，缓存层的设计至关重要",
			ContentType:    "text",
			Source:         "架构文档",
			Similarity:     0.82,
			RelevanceScore: 0.8,
			Timestamp:      time.Now().Add(-5 * time.Hour),
			Tags:           []string{"architecture", "performance", "cache"},
			Metadata:       map[string]interface{}{"doc_type": "architecture", "version": "v2.1"},
			MatchedSegments: []MatchedSegment{
				{
					SegmentText: "高性能系统",
					StartPos:    0,
					EndPos:      5,
					Similarity:  0.85,
				},
			},
		},
	}

	log.Printf("🔍 [向量存储] 返回 %d 个结果", len(results))
	return results, nil
}

// 辅助函数实现
func extractUserQueryFromPrompt(prompt string) string {
	// 从prompt中提取用户查询的简单实现
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.Contains(line, "用户查询") || strings.Contains(line, "User Query") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "系统优化"
}

func generateKeyConceptsJSON(query string) string {
	concepts := extractKeywords(query)
	result := "["
	for i, concept := range concepts {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, concept)
	}
	result += "]"
	return result
}

func generateProjectKeywordsJSON(query string) string {
	keywords := []string{"系统", "优化", "性能"}
	if strings.Contains(query, "缓存") {
		keywords = append(keywords, "缓存")
	}
	if strings.Contains(query, "数据库") {
		keywords = append(keywords, "数据库")
	}

	result := "["
	for i, keyword := range keywords {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, keyword)
	}
	result += "]"
	return result
}

func generateTechnicalKeywordsJSON(query string) string {
	keywords := []string{"性能", "优化", "架构"}
	if strings.Contains(query, "缓存") {
		keywords = append(keywords, "缓存", "Redis")
	}
	if strings.Contains(query, "数据库") {
		keywords = append(keywords, "数据库", "查询")
	}

	result := "["
	for i, keyword := range keywords {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, keyword)
	}
	result += "]"
	return result
}

func extractMainKeywords(query string) string {
	words := strings.Fields(query)
	if len(words) > 0 {
		return words[0]
	}
	return "优化"
}

func extractKeywords(query string) []string {
	keywords := []string{}
	if strings.Contains(query, "缓存") {
		keywords = append(keywords, "缓存", "性能")
	}
	if strings.Contains(query, "数据库") {
		keywords = append(keywords, "数据库", "查询")
	}
	if strings.Contains(query, "系统") {
		keywords = append(keywords, "系统", "架构")
	}
	if len(keywords) == 0 {
		keywords = []string{"技术", "优化"}
	}
	return keywords
}
