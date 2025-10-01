package services

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// TestJSONParsing 专门测试JSON解析问题
func TestJSONParsing(t *testing.T) {
	// 创建简单的LLM服务
	llmService := &SimpleLLMService{}

	// 创建宽召回服务
	wideRecallService := NewWideRecallService(
		&SimpleTimelineStore{},
		&SimpleKnowledgeStore{},
		&SimpleVectorStore{},
		llmService,
		&WideRecallConfig{
			LLMTimeout:           30,
			TimelineTimeout:      5,
			KnowledgeTimeout:     5,
			VectorTimeout:        5,
			TimelineMaxResults:   5,
			KnowledgeMaxResults:  5,
			VectorMaxResults:     5,
			MinSimilarityScore:   0.6,
			MinRelevanceScore:    0.5,
			ConfidenceThreshold:  0.7,
			UpdateThreshold:      0.4,
			PersistenceThreshold: 0.7,
			MaxRetries:           1,
			RetryInterval:        1,
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试上下文合成
	synthesisReq := &models.ContextSynthesisRequest{
		UserID:         "test_user",
		SessionID:      "test_session",
		WorkspaceID:    "/test/workspace",
		UserQuery:      "测试查询",
		IntentAnalysis: nil,
		CurrentContext: nil,
		RetrievalResults: &models.RetrievalResults{
			TotalResults:     3,
			TimelineResults:  []models.TimelineResult{},
			KnowledgeResults: []models.KnowledgeResult{},
			VectorResults:    []models.VectorResult{},
		},
		SynthesisConfig: &models.SynthesisConfig{
			LLMTimeout:           30,
			MaxTokens:            2048,
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

	log.Printf("🔍 [JSON调试] 开始测试上下文合成")

	resp, err := wideRecallService.ExecuteContextSynthesis(ctx, synthesisReq)
	if err != nil {
		t.Fatalf("上下文合成失败: %v", err)
	}

	log.Printf("🔍 [JSON调试] 上下文合成完成")

	// 详细检查结果
	if resp == nil {
		t.Fatal("响应为nil")
	}

	log.Printf("🔍 [JSON调试] EvaluationResult是否为nil: %t", resp.EvaluationResult == nil)
	log.Printf("🔍 [JSON调试] SynthesizedContext是否为nil: %t", resp.SynthesizedContext == nil)

	if resp.EvaluationResult != nil {
		log.Printf("✅ [JSON调试] 评估结果正常")
		log.Printf("   - ShouldUpdate: %t", resp.EvaluationResult.ShouldUpdate)
		log.Printf("   - UpdateConfidence: %.2f", resp.EvaluationResult.UpdateConfidence)
		log.Printf("   - EvaluationReason: %s", resp.EvaluationResult.EvaluationReason)
	} else {
		t.Error("❌ EvaluationResult为nil")
	}

	if resp.SynthesizedContext == nil {
		log.Printf("⚠️ [JSON调试] SynthesizedContext为nil，这是预期的（因为我们故意设置为null）")
	} else {
		log.Printf("✅ [JSON调试] SynthesizedContext正常")
	}
}
