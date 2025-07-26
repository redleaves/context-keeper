package components

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestEndToEndABC 测试A→B→C完整流程
func TestEndToEndABC(t *testing.T) {
	// 🔍 A: 创建查询意图分析器
	analyzer := NewBasicQueryIntentAnalyzer()

	// 🧠 C: 创建智能决策中心
	decisionCenter := NewBasicIntelligentDecisionCenter()

	ctx := context.Background()

	// 启动决策中心
	err := decisionCenter.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer decisionCenter.Stop(ctx)

	// 测试用例：调试类查询的端到端流程
	t.Run("Debugging Query End-to-End", func(t *testing.T) {
		// A: 原始查询
		originalQuery := "这个Go程序报错了，怎么debug？"

		// A→B: 查询意图分析
		intent, err := analyzer.AnalyzeIntent(ctx, originalQuery)
		if err != nil {
			t.Fatalf("Failed to analyze intent: %v", err)
		}

		// 验证意图分析结果
		if intent.IntentType != "debugging" {
			t.Errorf("Expected debugging intent, got %s", intent.IntentType)
		}

		if intent.Domain != "programming" {
			t.Errorf("Expected programming domain, got %s", intent.Domain)
		}

		// B→C: 智能决策
		decision, err := decisionCenter.MakeDecision(ctx, intent)
		if err != nil {
			t.Fatalf("Failed to make decision: %v", err)
		}

		// 验证决策结果
		if decision.Intent != intent {
			t.Error("Expected decision to contain the analyzed intent")
		}

		// 验证调试类决策的特征
		if len(decision.TaskPlan.Tasks) < 2 {
			t.Error("Expected debugging decision to have multiple tasks")
		}

		hasDebugEnhance := false
		for _, task := range decision.TaskPlan.Tasks {
			if task.TaskID == "debug_enhance" {
				hasDebugEnhance = true
				break
			}
		}
		if !hasDebugEnhance {
			t.Error("Expected debugging decision to include debug_enhance task")
		}

		// 验证策略选择
		expectedStrategies := []string{"debug_enhancement", "error_analysis"}
		for _, expected := range expectedStrategies {
			found := false
			for _, actual := range decision.SelectedStrategies {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected strategy %s not found in decision", expected)
			}
		}
	})

	// 测试用例：过程类查询的端到端流程
	t.Run("Procedural Query End-to-End", func(t *testing.T) {
		// A: 原始查询
		originalQuery := "如何用React实现一个性能优化的组件？"

		// A→B: 查询意图分析
		intent, err := analyzer.AnalyzeIntent(ctx, originalQuery)
		if err != nil {
			t.Fatalf("Failed to analyze intent: %v", err)
		}

		// 验证意图分析结果
		if intent.IntentType != "procedural" {
			t.Errorf("Expected procedural intent, got %s", intent.IntentType)
		}

		if intent.Domain != "frontend" {
			t.Errorf("Expected frontend domain, got %s", intent.Domain)
		}

		// 验证技术栈识别
		hasReact := false
		for _, tech := range intent.TechStack {
			if tech == "react" {
				hasReact = true
				break
			}
		}
		if !hasReact {
			t.Error("Expected to detect React in tech stack")
		}

		// B→C: 智能决策
		decision, err := decisionCenter.MakeDecision(ctx, intent)
		if err != nil {
			t.Fatalf("Failed to make decision: %v", err)
		}

		// 验证过程类决策特征
		if !decision.TaskPlan.ParallelExecution {
			t.Error("Expected procedural decision to use parallel execution")
		}

		// 验证包含step_enhance任务
		hasStepEnhance := false
		for _, task := range decision.TaskPlan.Tasks {
			if task.TaskID == "step_enhance" {
				hasStepEnhance = true
				// 验证任务参数
				if strategy, ok := task.Parameters["strategy"]; ok {
					if strategy != "step_by_step" {
						t.Error("Expected step_by_step strategy for procedural query")
					}
				}
				break
			}
		}
		if !hasStepEnhance {
			t.Error("Expected procedural decision to include step_enhance task")
		}
	})

	// 测试用例：复杂查询的端到端流程
	t.Run("Complex Query End-to-End", func(t *testing.T) {
		// A: 原始查询
		originalQuery := "如何设计一个高可用的分布式微服务架构，包含负载均衡、服务发现、熔断器和监控系统？"

		// A→B: 查询意图分析
		intent, err := analyzer.AnalyzeIntent(ctx, originalQuery)
		if err != nil {
			t.Fatalf("Failed to analyze intent: %v", err)
		}

		// 验证复杂度识别
		if intent.Complexity < 0.7 {
			t.Errorf("Expected high complexity (>0.7), got %f", intent.Complexity)
		}

		// 验证领域识别
		if intent.Domain != "architecture" {
			t.Errorf("Expected architecture domain, got %s", intent.Domain)
		}

		// B→C: 智能决策
		decision, err := decisionCenter.MakeDecision(ctx, intent)
		if err != nil {
			t.Fatalf("Failed to make decision: %v", err)
		}

		// 验证复杂查询决策特征
		if decision.TaskPlan.ParallelExecution {
			t.Error("Expected complex decision to use sequential execution")
		}

		if decision.TaskPlan.TimeoutSeconds < 40 {
			t.Error("Expected complex decision to have longer timeout")
		}

		// 验证包含多个复杂处理任务
		expectedTasks := []string{"complex_enhance", "complex_filter", "complex_adapt"}
		for _, expectedTask := range expectedTasks {
			found := false
			for _, task := range decision.TaskPlan.Tasks {
				if task.TaskID == expectedTask {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected complex decision to include %s task", expectedTask)
			}
		}

		// 验证复杂查询的策略选择
		expectedStrategies := []string{"comprehensive_enhancement", "multi_faceted_search", "complex_analysis"}
		for _, expected := range expectedStrategies {
			found := false
			for _, actual := range decision.SelectedStrategies {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected strategy %s not found in complex decision", expected)
			}
		}
	})
}

// TestPerformanceABC 测试A→B→C流程的性能
func TestPerformanceABC(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	decisionCenter := NewBasicIntelligentDecisionCenter()

	ctx := context.Background()
	err := decisionCenter.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer decisionCenter.Stop(ctx)

	// 性能测试
	query := "如何优化大规模分布式系统的数据库性能？"

	start := time.Now()

	// A→B: 意图分析
	intent, err := analyzer.AnalyzeIntent(ctx, query)
	if err != nil {
		t.Fatalf("Failed to analyze intent: %v", err)
	}

	// B→C: 决策制定
	decision, err := decisionCenter.MakeDecision(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	totalTime := time.Since(start)

	// 验证端到端性能（应该在150ms以内）
	if totalTime > 150*time.Millisecond {
		t.Errorf("End-to-end processing took too long: %v", totalTime)
	}

	// 验证结果完整性
	if decision.DecisionID == "" {
		t.Error("Expected decision to have an ID")
	}

	if decision.TaskPlan == nil {
		t.Error("Expected decision to have a task plan")
	}

	if len(decision.SelectedStrategies) == 0 {
		t.Error("Expected decision to have selected strategies")
	}
}

// TestMultipleQueriesABC 测试A→B→C流程处理多个查询
func TestMultipleQueriesABC(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	decisionCenter := NewBasicIntelligentDecisionCenter()

	ctx := context.Background()
	err := decisionCenter.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer decisionCenter.Stop(ctx)

	// 测试查询列表
	testQueries := []struct {
		query              string
		expectedIntentType string
		expectedDomain     string
		expectedTaskCount  int
	}{
		{
			query:              "Python代码报错了",
			expectedIntentType: "debugging",
			expectedDomain:     "programming",
			expectedTaskCount:  2,
		},
		{
			query:              "什么是微服务架构？",
			expectedIntentType: "conceptual",
			expectedDomain:     "architecture",
			expectedTaskCount:  2,
		},
		{
			query:              "如何配置Docker容器？",
			expectedIntentType: "procedural",
			expectedDomain:     "devops",
			expectedTaskCount:  2,
		},
		{
			query:              "优化React组件性能",
			expectedIntentType: "technical",
			expectedDomain:     "frontend",
			expectedTaskCount:  2,
		},
	}

	for i, testCase := range testQueries {
		t.Run(fmt.Sprintf("Query_%d", i+1), func(t *testing.T) {
			// A→B: 意图分析
			intent, err := analyzer.AnalyzeIntent(ctx, testCase.query)
			if err != nil {
				t.Fatalf("Failed to analyze intent: %v", err)
			}

			// 验证意图分析
			if intent.IntentType != testCase.expectedIntentType {
				t.Errorf("Expected intent type %s, got %s", testCase.expectedIntentType, intent.IntentType)
			}

			if intent.Domain != testCase.expectedDomain {
				t.Errorf("Expected domain %s, got %s", testCase.expectedDomain, intent.Domain)
			}

			// B→C: 决策制定
			decision, err := decisionCenter.MakeDecision(ctx, intent)
			if err != nil {
				t.Fatalf("Failed to make decision: %v", err)
			}

			// 验证决策
			if len(decision.TaskPlan.Tasks) < testCase.expectedTaskCount {
				t.Errorf("Expected at least %d tasks, got %d", testCase.expectedTaskCount, len(decision.TaskPlan.Tasks))
			}

			if len(decision.SelectedStrategies) == 0 {
				t.Error("Expected decision to have selected strategies")
			}
		})
	}

	// 验证统计信息
	analyzerStats := analyzer.GetStats()
	if analyzerStats.TotalAnalyzed != len(testQueries) {
		t.Errorf("Expected %d analyzed queries, got %d", len(testQueries), analyzerStats.TotalAnalyzed)
	}

	decisionStats := decisionCenter.GetStats()
	if decisionStats.TotalDecisions != len(testQueries) {
		t.Errorf("Expected %d decisions, got %d", len(testQueries), decisionStats.TotalDecisions)
	}
}

// BenchmarkEndToEndABC 端到端基准测试
func BenchmarkEndToEndABC(b *testing.B) {
	analyzer := NewBasicQueryIntentAnalyzer()
	decisionCenter := NewBasicIntelligentDecisionCenter()

	ctx := context.Background()
	err := decisionCenter.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start decision center: %v", err)
	}
	defer decisionCenter.Stop(ctx)

	query := "如何使用Go语言实现高性能的Web API服务？"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// A→B: 意图分析
		intent, err := analyzer.AnalyzeIntent(ctx, query)
		if err != nil {
			b.Fatalf("Failed to analyze intent: %v", err)
		}

		// B→C: 决策制定
		_, err = decisionCenter.MakeDecision(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to make decision: %v", err)
		}
	}
}
