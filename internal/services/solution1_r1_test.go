package services

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// TestSolution1_DeepSeekR1Model 方案1：测试DeepSeek-R1模型的效果
func TestSolution1_DeepSeekR1Model(t *testing.T) {
	log.Printf("🚀 [方案1测试] 开始测试DeepSeek-R1模型处理复杂UnifiedContextModel的能力")

	// === 创建真实的服务实例 ===
	realTimelineStore := &SimpleTimelineStore{}
	realKnowledgeStore := &SimpleKnowledgeStore{}
	realVectorStore := &SimpleVectorStore{}
	realLLMService := &SimpleLLMService{} // 这个会使用配置中的deepseek-reasoner模型

	// === 创建宽召回服务 ===
	wideRecallConfig := &WideRecallConfig{
		LLMTimeout:           60, // 增加超时时间，R1模型推理时间较长
		TimelineTimeout:      5,
		KnowledgeTimeout:     5,
		VectorTimeout:        5,
		TimelineMaxResults:   10,
		KnowledgeMaxResults:  8,
		VectorMaxResults:     12,
		MinSimilarityScore:   0.6,
		MinRelevanceScore:    0.5,
		ConfidenceThreshold:  0.7,
		UpdateThreshold:      0.4,
		PersistenceThreshold: 0.7,
		MaxRetries:           1,
		RetryInterval:        2,
	}

	wideRecallService := NewWideRecallService(
		realTimelineStore,
		realKnowledgeStore,
		realVectorStore,
		realLLMService,
		wideRecallConfig,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 增加总超时时间
	defer cancel()

	// === 测试复杂的上下文合成 ===
	t.Run("R1模型复杂上下文合成测试", func(t *testing.T) {
		// 先执行宽召回获取丰富的检索结果
		wideRecallReq := &models.WideRecallRequest{
			UserID:      "test_user_r1",
			SessionID:   "test_session_r1",
			WorkspaceID: "/test/workspace/r1",
			UserQuery:   "我需要设计一个高并发的微服务架构，包括缓存层、数据库分片、消息队列和监控系统，请帮我分析技术选型和架构设计",
			RetrievalConfig: &models.RetrievalConfig{
				TimelineTimeout:     5,
				KnowledgeTimeout:    5,
				VectorTimeout:       5,
				TimelineMaxResults:  15,
				KnowledgeMaxResults: 12,
				VectorMaxResults:    18,
				MinSimilarityScore:  0.6,
				MinRelevanceScore:   0.5,
				MaxRetries:          1,
				RetryInterval:       2,
			},
			RequestTime: time.Now(),
		}

		log.Printf("🔍 [方案1测试] 开始宽召回检索...")

		wideRecallResp, err := wideRecallService.ExecuteWideRecall(ctx, wideRecallReq)
		if err != nil {
			t.Fatalf("宽召回检索失败: %v", err)
		}

		log.Printf("✅ [方案1测试] 宽召回检索完成，总结果数: %d", wideRecallResp.RetrievalResults.TotalResults)

		// 创建复杂的上下文合成请求
		synthesisReq := &models.ContextSynthesisRequest{
			UserID:           "test_user_r1",
			SessionID:        "test_session_r1",
			WorkspaceID:      "/test/workspace/r1",
			UserQuery:        "我需要设计一个高并发的微服务架构，包括缓存层、数据库分片、消息队列和监控系统，请帮我分析技术选型和架构设计",
			IntentAnalysis:   nil, // 让服务内部处理
			CurrentContext:   nil, // 首次创建
			RetrievalResults: wideRecallResp.RetrievalResults,
			SynthesisConfig: &models.SynthesisConfig{
				LLMTimeout:           60,   // R1模型需要更长时间
				MaxTokens:            8000, // 增加token限制
				Temperature:          0.1,  // 降低温度提高一致性
				ConfidenceThreshold:  0.7,
				ConflictResolution:   "time_priority",
				InformationFusion:    "weighted_merge",
				QualityAssessment:    "comprehensive",
				UpdateThreshold:      0.4,
				PersistenceThreshold: 0.7,
			},
			RequestTime: time.Now(),
		}

		log.Printf("🤖 [方案1测试] 开始R1模型上下文合成...")
		log.Printf("📊 [方案1测试] 输入数据规模:")
		log.Printf("   - 用户查询长度: %d字符", len(synthesisReq.UserQuery))
		log.Printf("   - 检索结果总数: %d", synthesisReq.RetrievalResults.TotalResults)
		log.Printf("   - 时间线结果: %d", len(synthesisReq.RetrievalResults.TimelineResults))
		log.Printf("   - 知识图谱结果: %d", len(synthesisReq.RetrievalResults.KnowledgeResults))
		log.Printf("   - 向量结果: %d", len(synthesisReq.RetrievalResults.VectorResults))

		synthesisResp, err := wideRecallService.ExecuteContextSynthesis(ctx, synthesisReq)
		if err != nil {
			t.Fatalf("❌ [方案1测试] 上下文合成失败: %v", err)
		}

		// === 详细验证R1模型的输出质量 ===
		log.Printf("🎯 [方案1测试] R1模型上下文合成结果分析:")

		if synthesisResp.EvaluationResult == nil {
			t.Fatal("❌ [方案1测试] EvaluationResult为nil - R1模型未能生成评估结果")
		}

		log.Printf("✅ [方案1测试] 评估结果正常:")
		log.Printf("   - 是否应该更新: %t", synthesisResp.EvaluationResult.ShouldUpdate)
		log.Printf("   - 更新置信度: %.3f", synthesisResp.EvaluationResult.UpdateConfidence)
		log.Printf("   - 评估原因: %s", synthesisResp.EvaluationResult.EvaluationReason)
		log.Printf("   - 语义变化数量: %d", len(synthesisResp.EvaluationResult.SemanticChanges))

		if synthesisResp.SynthesizedContext == nil {
			log.Printf("❌ [方案1测试] SynthesizedContext为nil - 这是关键问题！")
			log.Printf("🔍 [方案1测试] 分析原因:")
			log.Printf("   - R1模型可能无法生成如此复杂的UnifiedContextModel结构")
			log.Printf("   - 需要检查LLM响应的JSON格式是否正确")
			log.Printf("   - 可能需要简化目标结构或改进prompt")

			// 这里不直接失败，而是记录问题继续分析
			t.Errorf("❌ [方案1测试] R1模型未能生成完整的上下文结构")
		} else {
			log.Printf("🎉 [方案1测试] R1模型成功生成完整上下文!")
			log.Printf("✅ [方案1测试] 合成上下文详情:")
			log.Printf("   - 会话ID: %s", synthesisResp.SynthesizedContext.SessionID)
			log.Printf("   - 用户ID: %s", synthesisResp.SynthesizedContext.UserID)
			log.Printf("   - 工作空间ID: %s", synthesisResp.SynthesizedContext.WorkspaceID)

			if synthesisResp.SynthesizedContext.CurrentTopic != nil {
				log.Printf("   - 主题: %s", synthesisResp.SynthesizedContext.CurrentTopic.MainTopic)
				log.Printf("   - 主要痛点: %s", synthesisResp.SynthesizedContext.CurrentTopic.PrimaryPainPoint)
				log.Printf("   - 期望结果: %s", synthesisResp.SynthesizedContext.CurrentTopic.ExpectedOutcome)
				log.Printf("   - 关键概念数量: %d", len(synthesisResp.SynthesizedContext.CurrentTopic.KeyConcepts))
				log.Printf("   - 置信度: %.3f", synthesisResp.SynthesizedContext.CurrentTopic.ConfidenceLevel)
			}

			if synthesisResp.SynthesizedContext.Project != nil {
				log.Printf("   - 项目名称: %s", synthesisResp.SynthesizedContext.Project.ProjectName)
				log.Printf("   - 项目类型: %s", string(synthesisResp.SynthesizedContext.Project.ProjectType))
				log.Printf("   - 主要语言: %s", synthesisResp.SynthesizedContext.Project.PrimaryLanguage)
				log.Printf("   - 当前阶段: %s", string(synthesisResp.SynthesizedContext.Project.CurrentPhase))
			}
		}

		// === 性能分析 ===
		log.Printf("📊 [方案1测试] 性能分析:")
		log.Printf("   - 总处理时间: %dms", synthesisResp.ProcessTime)
		log.Printf("   - 是否超过预期时间(30s): %t", synthesisResp.ProcessTime > 30000)

		// === 结论 ===
		if synthesisResp.SynthesizedContext != nil {
			log.Printf("🎯 [方案1结论] ✅ R1模型能够处理复杂的UnifiedContextModel")
			log.Printf("🎯 [方案1结论] ✅ 生成质量较高，结构完整")
			log.Printf("🎯 [方案1结论] ⚠️  处理时间较长，需要优化")
		} else {
			log.Printf("🎯 [方案1结论] ❌ R1模型无法稳定生成完整的UnifiedContextModel")
			log.Printf("🎯 [方案1结论] 📋 建议：需要尝试方案2（并行拆解）或方案3（简化模型）")
		}
	})
}
