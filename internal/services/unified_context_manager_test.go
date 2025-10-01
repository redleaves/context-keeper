package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
)

// TestUnifiedContextManager 测试统一上下文管理器
func TestUnifiedContextManager(t *testing.T) {
	// 创建模拟依赖
	sessionManager := &store.SessionStore{} // 简化的会话管理器
	llmService := NewMockLLMService()
	contextService := (*ContextService)(nil) // 暂时使用nil，测试中不依赖真实的ContextService

	// 创建上下文管理器
	ucm := NewUnifiedContextManager(contextService, sessionManager, llmService)
	defer ucm.Stop()

	// 测试用例1: 初始化上下文
	t.Run("初始化上下文", func(t *testing.T) {
		req := &models.ContextUpdateRequest{
			SessionID:   "test-session-1",
			UserQuery:   "如何实现Go语言的并发编程？",
			UserID:      "test-user",
			WorkspaceID: "/Users/test/go-project",
			QueryType:   models.QueryTypeTechnical,
			StartTime:   time.Now(),
		}

		resp, err := ucm.UpdateContext(req)
		if err != nil {
			t.Fatalf("初始化上下文失败: %v", err)
		}

		if !resp.Success {
			t.Errorf("期望成功，但得到失败")
		}

		if resp.UpdatedContext == nil {
			t.Errorf("期望得到更新的上下文，但为nil")
		}

		if resp.UpdatedContext.SessionID != req.SessionID {
			t.Errorf("期望会话ID为 %s，但得到 %s", req.SessionID, resp.UpdatedContext.SessionID)
		}

		t.Logf("✅ 初始化上下文成功: %s", resp.UpdateSummary)
	})

	// 测试用例2: 更新现有上下文
	t.Run("更新现有上下文", func(t *testing.T) {
		req := &models.ContextUpdateRequest{
			SessionID:   "test-session-1", // 使用相同的会话ID
			UserQuery:   "goroutine和channel的最佳实践是什么？",
			UserID:      "test-user",
			WorkspaceID: "/Users/test/go-project",
			QueryType:   models.QueryTypeTechnical,
			StartTime:   time.Now(),
		}

		resp, err := ucm.UpdateContext(req)
		if err != nil {
			t.Fatalf("更新上下文失败: %v", err)
		}

		if !resp.Success {
			t.Errorf("期望成功，但得到失败")
		}

		t.Logf("✅ 更新上下文成功: %s", resp.UpdateSummary)
	})

	// 测试用例3: 获取上下文
	t.Run("获取上下文", func(t *testing.T) {
		context, err := ucm.GetContext("test-session-1")
		if err != nil {
			t.Fatalf("获取上下文失败: %v", err)
		}

		if context == nil {
			t.Errorf("期望得到上下文，但为nil")
		}

		if context.SessionID != "test-session-1" {
			t.Errorf("期望会话ID为 test-session-1，但得到 %s", context.SessionID)
		}

		t.Logf("✅ 获取上下文成功: 主题=%s", context.CurrentTopic.MainTopic)
	})

	// 测试用例4: 清理上下文
	t.Run("清理上下文", func(t *testing.T) {
		err := ucm.CleanupContext("test-session-1")
		if err != nil {
			t.Fatalf("清理上下文失败: %v", err)
		}

		// 验证上下文已被清理
		_, err = ucm.GetContext("test-session-1")
		if err == nil {
			t.Errorf("期望获取上下文失败，但成功了")
		}

		t.Logf("✅ 清理上下文成功")
	})
}

// TestMockLLMService 测试模拟LLM服务
func TestMockLLMService(t *testing.T) {
	llmService := NewMockLLMService()

	// 测试意图分析
	t.Run("意图分析", func(t *testing.T) {
		query := "如何优化Go程序的性能？"
		result, err := llmService.AnalyzeUserIntent(query)
		if err != nil {
			t.Fatalf("意图分析失败: %v", err)
		}

		if result.CoreIntentText == "" {
			t.Errorf("期望得到核心意图，但为空")
		}

		t.Logf("✅ 意图分析成功: 核心意图=%s, 场景=%s",
			result.CoreIntentText, result.ScenarioText)
	})

	// 测试上下文合成
	t.Run("上下文合成", func(t *testing.T) {
		query := "实现一个HTTP服务器"
		intentAnalysis := &models.IntentAnalysisResult{
			CoreIntentText:    "实现",
			DomainContextText: "Go语言开发",
			ScenarioText:      "代码开发场景",
			IntentCount:       1,
		}

		result, err := llmService.SynthesizeAndEvaluateContext(
			query, nil, nil, intentAnalysis)
		if err != nil {
			t.Fatalf("上下文合成失败: %v", err)
		}

		if !result.ShouldUpdate {
			t.Errorf("期望需要更新上下文，但得到不需要")
		}

		if result.UpdatedContext == nil {
			t.Errorf("期望得到更新的上下文，但为nil")
		}

		t.Logf("✅ 上下文合成成功: 是否更新=%t, 置信度=%.2f",
			result.ShouldUpdate, result.UpdateConfidence)
	})
}

// BenchmarkContextUpdate 上下文更新性能测试
func BenchmarkContextUpdate(b *testing.B) {
	sessionManager := &store.SessionStore{}
	llmService := NewMockLLMService()
	contextService := (*ContextService)(nil) // 暂时使用nil，测试中不依赖真实的ContextService
	ucm := NewUnifiedContextManager(contextService, sessionManager, llmService)
	defer ucm.Stop()

	req := &models.ContextUpdateRequest{
		SessionID:   "bench-session",
		UserQuery:   "性能测试查询",
		UserID:      "bench-user",
		WorkspaceID: "/Users/test/bench-project",
		QueryType:   models.QueryTypeGeneral,
		StartTime:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.SessionID = "bench-session-" + string(rune(i))
		_, err := ucm.UpdateContext(req)
		if err != nil {
			b.Fatalf("上下文更新失败: %v", err)
		}
	}
}

// TestContextManagerConcurrency 并发测试
func TestContextManagerConcurrency(t *testing.T) {
	sessionManager := &store.SessionStore{}
	llmService := NewMockLLMService()
	contextService := (*ContextService)(nil) // 暂时使用nil，测试中不依赖真实的ContextService
	ucm := NewUnifiedContextManager(contextService, sessionManager, llmService)
	defer ucm.Stop()

	// 并发创建多个会话的上下文
	const numGoroutines = 10
	const numRequests = 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numRequests; j++ {
				req := &models.ContextUpdateRequest{
					SessionID:   fmt.Sprintf("concurrent-session-%d-%d", goroutineID, j),
					UserQuery:   fmt.Sprintf("并发测试查询 %d-%d", goroutineID, j),
					UserID:      fmt.Sprintf("user-%d", goroutineID),
					WorkspaceID: "/Users/test/concurrent-project",
					QueryType:   models.QueryTypeGeneral,
					StartTime:   time.Now(),
				}

				_, err := ucm.UpdateContext(req)
				if err != nil {
					t.Errorf("并发上下文更新失败: %v", err)
					return
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	t.Logf("✅ 并发测试完成: %d个goroutine，每个%d个请求", numGoroutines, numRequests)
}
