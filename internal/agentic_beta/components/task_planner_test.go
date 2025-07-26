package components

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/interfaces"
)

func TestBasicTaskPlanner(t *testing.T) {
	// 创建任务规划器实例
	planner := NewBasicTaskPlanner()

	// 验证基础信息
	if planner.name != "BasicTaskPlanner" {
		t.Errorf("Expected name BasicTaskPlanner, got %s", planner.name)
	}

	if planner.version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", planner.version)
	}

	if !planner.enabled {
		t.Error("Expected task planner to be enabled")
	}

	if !planner.initialized {
		t.Error("Expected task planner to be initialized")
	}

	// 验证规则和模板数量
	if len(planner.planningRules) == 0 {
		t.Error("Expected planning rules to be initialized")
	}

	if len(planner.taskTemplates) == 0 {
		t.Error("Expected task templates to be initialized")
	}
}

func TestPlanTasks_DebuggingQuery(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 创建调试类查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "这个Go程序报错了，怎么debug？",
		Timestamp:     time.Now(),
		IntentType:    "debugging",
		Domain:        "programming",
		Complexity:    0.6,
		Keywords:      createKeywords([]string{"Go", "程序", "报错", "debug"}),
		TechStack:     []string{"go"},
		Confidence:    0.8,
	}

	// 执行任务规划
	plan, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 验证计划基础信息
	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	if plan.TargetIntent != intent {
		t.Error("Plan should reference the original intent")
	}

	// 验证调试类查询的任务序列
	expectedTasks := []string{"analyze_error", "enhance_debug_context", "filter_debug_noise"}
	if len(plan.Tasks) != len(expectedTasks) {
		t.Errorf("Expected %d tasks, got %d", len(expectedTasks), len(plan.Tasks))
	}

	// 验证任务顺序（调试类应该是顺序执行）
	if plan.ParallelExecution {
		t.Error("Debugging queries should execute sequentially")
	}

	// 验证高优先级
	if plan.Priority < 80 {
		t.Errorf("Debug plan should have high priority, got %d", plan.Priority)
	}

	t.Logf("✅ Debug plan created: %d tasks, priority %d, parallel=%v",
		len(plan.Tasks), plan.Priority, plan.ParallelExecution)
}

func TestPlanTasks_ConceptualQuery(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 创建概念类查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "什么是微服务架构？",
		Timestamp:     time.Now(),
		IntentType:    "conceptual",
		Domain:        "architecture",
		Complexity:    0.5,
		Keywords:      createDomainKeywords("什么是", "微服务", "架构"),
		TechStack:     []string{},
		Confidence:    0.9,
	}

	// 执行任务规划
	plan, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 验证概念类查询可以并行执行
	if !plan.ParallelExecution {
		t.Error("Conceptual queries should allow parallel execution")
	}

	// 验证中等优先级
	if plan.Priority < 60 || plan.Priority > 80 {
		t.Errorf("Conceptual plan priority should be medium (60-80), got %d", plan.Priority)
	}

	t.Logf("✅ Conceptual plan created: %d tasks, priority %d, parallel=%v",
		len(plan.Tasks), plan.Priority, plan.ParallelExecution)
}

func TestPlanTasks_ProceduralQuery(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 创建过程类查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "如何配置Docker容器？",
		Timestamp:     time.Now(),
		IntentType:    "procedural",
		Domain:        "devops",
		Complexity:    0.4,
		Keywords:      createActionKeywords("如何", "配置", "Docker", "容器"),
		TechStack:     []string{"docker"},
		Confidence:    0.85,
	}

	// 执行任务规划
	plan, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 验证过程类查询是顺序执行
	if plan.ParallelExecution {
		t.Error("Procedural queries should execute sequentially")
	}

	// 验证较高优先级
	if plan.Priority < 75 {
		t.Errorf("Procedural plan should have high priority, got %d", plan.Priority)
	}

	t.Logf("✅ Procedural plan created: %d tasks, priority %d, parallel=%v",
		len(plan.Tasks), plan.Priority, plan.ParallelExecution)
}

func TestPlanTasks_ComplexQuery(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 创建复杂查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "如何设计高可用的分布式微服务架构？",
		Timestamp:     time.Now(),
		IntentType:    "procedural",
		Domain:        "architecture",
		Complexity:    0.9, // 高复杂度
		Keywords:      createDomainKeywords("设计", "高可用", "分布式", "微服务", "架构"),
		TechStack:     []string{},
		Confidence:    0.8,
	}

	// 执行任务规划
	plan, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 复杂查询应该使用复杂规划规则
	if plan.Priority != 100 {
		t.Errorf("Complex query should have highest priority (100), got %d", plan.Priority)
	}

	// 复杂查询应该支持并行
	if !plan.ParallelExecution {
		t.Error("Complex queries should support parallel execution")
	}

	// 应该有更多的任务
	if len(plan.Tasks) < 3 {
		t.Errorf("Complex plan should have at least 3 tasks, got %d", len(plan.Tasks))
	}

	t.Logf("✅ Complex plan created: %d tasks, priority %d, parallel=%v",
		len(plan.Tasks), plan.Priority, plan.ParallelExecution)
}

func TestPlanTasks_CacheHit(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 创建相同的查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "测试缓存",
		Timestamp:     time.Now(),
		IntentType:    "technical",
		Domain:        "programming",
		Complexity:    0.3,
		Keywords:      createKeywords([]string{"测试", "缓存"}),
		TechStack:     []string{},
		Confidence:    0.7,
	}

	// 第一次规划
	plan1, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 第二次规划（应该命中缓存）
	plan2, err := planner.PlanTasks(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to plan tasks: %v", err)
	}

	// 验证缓存命中（应该返回相同的计划）
	if plan1.PlanID != plan2.PlanID {
		t.Error("Second planning should hit cache and return same plan")
	}

	t.Logf("✅ Cache hit verified: plan ID %s", plan1.PlanID)
}

func TestValidatePlan(t *testing.T) {
	planner := NewBasicTaskPlanner()

	// 测试有效计划
	validPlan := &interfaces.TaskPlan{
		PlanID: "test_plan",
		Tasks: []*interfaces.Task{
			{TaskID: "task1", Type: "enhance", Dependencies: []string{}},
			{TaskID: "task2", Type: "filter", Dependencies: []string{"task1"}},
		},
		ExecutionOrder: []string{"task1", "task2"},
	}

	if err := planner.ValidatePlan(validPlan); err != nil {
		t.Errorf("Valid plan should pass validation: %v", err)
	}

	// 测试空计划
	emptyPlan := &interfaces.TaskPlan{
		PlanID: "empty_plan",
		Tasks:  []*interfaces.Task{},
	}

	if err := planner.ValidatePlan(emptyPlan); err == nil {
		t.Error("Empty plan should fail validation")
	}

	// 测试nil计划
	if err := planner.ValidatePlan(nil); err == nil {
		t.Error("Nil plan should fail validation")
	}

	// 测试依赖关系错误的计划
	invalidPlan := &interfaces.TaskPlan{
		PlanID: "invalid_plan",
		Tasks: []*interfaces.Task{
			{TaskID: "task1", Type: "enhance", Dependencies: []string{"nonexistent"}},
		},
	}

	if err := planner.ValidatePlan(invalidPlan); err == nil {
		t.Error("Plan with invalid dependencies should fail validation")
	}
}

func TestOptimizePlan(t *testing.T) {
	planner := NewBasicTaskPlanner()

	// 创建测试计划
	originalPlan := &interfaces.TaskPlan{
		PlanID: "test_plan",
		Tasks: []*interfaces.Task{
			{TaskID: "task1", Type: "enhance", Priority: 50},
			{TaskID: "task2", Type: "filter", Priority: 80},
		},
		ParallelExecution: false,
	}

	// 优化计划
	optimizedPlan, err := planner.OptimizePlan(originalPlan)
	if err != nil {
		t.Errorf("Failed to optimize plan: %v", err)
	}

	if optimizedPlan == nil {
		t.Error("Optimized plan should not be nil")
	}

	// 测试nil计划优化
	_, err = planner.OptimizePlan(nil)
	if err == nil {
		t.Error("Optimizing nil plan should return error")
	}
}

func TestPlannerStatistics(t *testing.T) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	// 执行几次规划以生成统计数据
	intents := []*interfaces.QueryIntent{
		{IntentType: "debugging", Domain: "programming", Complexity: 0.5, Confidence: 0.8},
		{IntentType: "conceptual", Domain: "architecture", Complexity: 0.3, Confidence: 0.9},
		{IntentType: "technical", Domain: "frontend", Complexity: 0.7, Confidence: 0.6},
	}

	for _, intent := range intents {
		intent.OriginalQuery = "test query"
		intent.Timestamp = time.Now()
		intent.Keywords = createKeywords([]string{"test"})
		intent.TechStack = []string{}

		_, err := planner.PlanTasks(ctx, intent)
		if err != nil {
			t.Fatalf("Failed to plan tasks: %v", err)
		}
	}

	// 获取统计信息
	stats := planner.GetStatistics()

	// 验证统计数据
	if totalPlans, ok := stats["total_plans"].(int); !ok || totalPlans != 3 {
		t.Errorf("Expected total_plans to be 3, got %v", stats["total_plans"])
	}

	if successfulPlans, ok := stats["successful_plans"].(int); !ok || successfulPlans != 3 {
		t.Errorf("Expected successful_plans to be 3, got %v", stats["successful_plans"])
	}

	if successRate, ok := stats["success_rate"].(float64); !ok || successRate != 1.0 {
		t.Errorf("Expected success_rate to be 1.0, got %v", stats["success_rate"])
	}

	t.Logf("✅ Statistics: %+v", stats)
}

func TestPlannerCapabilities(t *testing.T) {
	planner := NewBasicTaskPlanner()

	// 测试Name方法
	if name := planner.Name(); name != "BasicTaskPlanner" {
		t.Errorf("Expected name BasicTaskPlanner, got %s", name)
	}

	// 测试GetCapabilities方法
	capabilities := planner.GetCapabilities()
	if len(capabilities) == 0 {
		t.Error("Planner should have capabilities")
	}

	expectedCaps := []string{"智能任务规划", "规则匹配", "并行优化", "计划缓存", "依赖解析", "性能统计"}
	if len(capabilities) != len(expectedCaps) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCaps), len(capabilities))
	}

	t.Logf("✅ Capabilities: %v", capabilities)
}

func TestRuleScoring(t *testing.T) {
	planner := NewBasicTaskPlanner()

	// 创建测试意图
	intent := &interfaces.QueryIntent{
		IntentType: "debugging",
		Domain:     "programming",
		Complexity: 0.5,
		Confidence: 0.8,
	}

	// 获取调试规则
	debugRule := planner.planningRules["debug_planning"]
	if debugRule == nil {
		t.Fatal("Debug planning rule not found")
	}

	// 计算匹配分数
	score := planner.calculateRuleScore(debugRule, intent)

	// 调试类查询应该高度匹配调试规则
	expectedScore := 0.4 + 0.3 + 0.2 + 0.08 // intent + domain + complexity + confidence
	if score < expectedScore*0.9 {          // 允许一些误差
		t.Errorf("Debug rule score too low: got %.2f, expected around %.2f", score, expectedScore)
	}

	t.Logf("✅ Debug rule score: %.2f", score)
}

// 基准测试
func BenchmarkPlanTasks(b *testing.B) {
	planner := NewBasicTaskPlanner()
	ctx := context.Background()

	intent := &interfaces.QueryIntent{
		OriginalQuery: "测试性能",
		IntentType:    "technical",
		Domain:        "programming",
		Complexity:    0.5,
		Keywords:      createKeywords([]string{"测试", "性能"}),
		Confidence:    0.7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次使用不同的查询避免缓存命中
		intent.OriginalQuery = fmt.Sprintf("测试性能 %d", i)
		_, err := planner.PlanTasks(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to plan tasks: %v", err)
		}
	}
}
