package components

import (
	"context"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/interfaces"
)

// KeywordConfig 关键词配置
type KeywordConfig struct {
	Weight   float64 `json:"weight"`
	Category string  `json:"category"`
	Source   string  `json:"source"`
}

// EntityConfig 实体配置
type EntityConfig struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
}

// DefaultKeywordConfig 默认关键词配置
var DefaultKeywordConfig = KeywordConfig{
	Weight:   0.7,
	Category: "general",
	Source:   "test_helper",
}

// DefaultEntityConfig 默认实体配置
var DefaultEntityConfig = EntityConfig{
	Type:  "TECH",
	Score: 0.8,
}

// createKeywords 创建关键词信息（支持自定义配置）
func createKeywords(terms []string, configs ...KeywordConfig) []interfaces.KeywordInfo {
	var keywords []interfaces.KeywordInfo

	for i, term := range terms {
		config := DefaultKeywordConfig
		if i < len(configs) {
			config = configs[i] // 使用对应位置的配置
		} else if len(configs) == 1 {
			config = configs[0] // 所有关键词使用同一配置
		}

		keywords = append(keywords, interfaces.KeywordInfo{
			Term:     term,
			Weight:   config.Weight,
			Category: config.Category,
			Source:   config.Source,
		})
	}
	return keywords
}

// createEntities 创建实体信息（支持自定义配置）
func createEntities(texts []string, configs ...EntityConfig) []interfaces.EntityInfo {
	var entities []interfaces.EntityInfo

	for i, text := range texts {
		config := DefaultEntityConfig
		if i < len(configs) {
			config = configs[i] // 使用对应位置的配置
		} else if len(configs) == 1 {
			config = configs[0] // 所有实体使用同一配置
		}

		entities = append(entities, interfaces.EntityInfo{
			Text:     text,
			Type:     config.Type,
			Score:    config.Score,
			Position: [2]int{i * 10, (i + 1) * 10}, // 基于索引的位置计算
		})
	}
	return entities
}

// createTechKeywords 创建技术关键词（预设配置）
func createTechKeywords(terms ...string) []interfaces.KeywordInfo {
	techConfig := KeywordConfig{
		Weight:   0.9,
		Category: "technical",
		Source:   "tech_dictionary",
	}
	return createKeywords(terms, techConfig)
}

// createDomainKeywords 创建领域关键词（预设配置）
func createDomainKeywords(terms ...string) []interfaces.KeywordInfo {
	domainConfig := KeywordConfig{
		Weight:   0.8,
		Category: "domain",
		Source:   "domain_analysis",
	}
	return createKeywords(terms, domainConfig)
}

// createActionKeywords 创建动作关键词（预设配置）
func createActionKeywords(terms ...string) []interfaces.KeywordInfo {
	actionConfig := KeywordConfig{
		Weight:   0.7,
		Category: "action",
		Source:   "action_detection",
	}
	return createKeywords(terms, actionConfig)
}

func TestBasicIntelligentDecisionCenter(t *testing.T) {
	// 创建决策中心实例
	center := NewBasicIntelligentDecisionCenter()

	// 验证基础信息
	if center.name != "BasicIntelligentDecisionCenter" {
		t.Errorf("Expected name BasicIntelligentDecisionCenter, got %s", center.name)
	}

	if center.version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", center.version)
	}

	if !center.enabled {
		t.Error("Expected decision center to be enabled")
	}

	if center.started {
		t.Error("Expected decision center to be stopped initially")
	}
}

func TestDecisionCenterLifecycle(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	// 测试启动
	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}

	if !center.started {
		t.Error("Expected decision center to be started")
	}

	// 测试重复启动
	err = center.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already started center")
	}

	// 测试停止
	err = center.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop decision center: %v", err)
	}

	if center.started {
		t.Error("Expected decision center to be stopped")
	}
}

func TestMakeDecision_DebuggingIntent(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	// 启动决策中心
	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 创建调试类查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "这个Go程序报错了，怎么debug？",
		Timestamp:     time.Now(),
		IntentType:    "debugging",
		Domain:        "programming",
		Complexity:    0.5,
		Keywords:      createKeywords([]string{"Go", "程序", "报错", "debug"}),
		TechStack:     []string{"go"},
		Confidence:    0.8,
	}

	// 执行决策
	decision, err := center.MakeDecision(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	// 验证决策结果
	if decision.Intent != intent {
		t.Error("Expected decision to contain the original intent")
	}

	if decision.DecisionID == "" {
		t.Error("Expected decision to have an ID")
	}

	if decision.TaskPlan == nil {
		t.Error("Expected decision to have a task plan")
	}

	if len(decision.SelectedStrategies) == 0 {
		t.Error("Expected decision to have selected strategies")
	}

	// 验证任务计划
	if len(decision.TaskPlan.Tasks) == 0 {
		t.Error("Expected task plan to have tasks")
	}

	// 验证调试类决策的特定任务
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

	// 验证决策历史
	history := center.GetDecisionHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 decision in history, got %d", len(history))
	}
}

func TestMakeDecision_ProceduralIntent(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 创建过程类查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "如何用React实现一个性能优化的组件？",
		Timestamp:     time.Now(),
		IntentType:    "procedural",
		Domain:        "frontend",
		Complexity:    0.6,
		Keywords:      createTechKeywords("React", "实现", "性能", "优化", "组件"),
		TechStack:     []string{"react"},
		Confidence:    0.7,
	}

	decision, err := center.MakeDecision(ctx, intent)
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
			break
		}
	}
	if !hasStepEnhance {
		t.Error("Expected procedural decision to include step_enhance task")
	}
}

func TestMakeDecision_ComplexIntent(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 创建高复杂度查询意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "如何设计一个高可用的分布式微服务架构，包含负载均衡、服务发现、熔断器和监控系统？",
		Timestamp:     time.Now(),
		IntentType:    "technical",
		Domain:        "architecture",
		Complexity:    0.9, // 高复杂度
		Keywords:      createDomainKeywords("设计", "高可用", "分布式", "微服务", "架构"),
		TechStack:     []string{},
		Confidence:    0.8,
	}

	decision, err := center.MakeDecision(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	// 验证复杂查询特征
	if decision.TaskPlan.ParallelExecution {
		t.Error("Expected complex decision to use sequential execution")
	}

	if decision.TaskPlan.TimeoutSeconds < 40 {
		t.Error("Expected complex decision to have longer timeout")
	}

	// 验证包含complex_enhance任务
	hasComplexEnhance := false
	for _, task := range decision.TaskPlan.Tasks {
		if task.TaskID == "complex_enhance" {
			hasComplexEnhance = true
			break
		}
	}
	if !hasComplexEnhance {
		t.Error("Expected complex decision to include complex_enhance task")
	}

	// 验证包含更多任务
	if len(decision.TaskPlan.Tasks) < 3 {
		t.Error("Expected complex decision to have more tasks")
	}
}

func TestMakeDecision_DefaultRule(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 创建一个不匹配任何规则的意图
	intent := &interfaces.QueryIntent{
		OriginalQuery: "test query",
		Timestamp:     time.Now(),
		IntentType:    "unknown", // 不匹配任何规则
		Domain:        "unknown",
		Complexity:    0.1, // 低复杂度
		Keywords:      createKeywords([]string{"test"}),
		TechStack:     []string{},
		Confidence:    0.5,
	}

	decision, err := center.MakeDecision(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	// 验证默认决策
	if decision.DecisionReasoning != "默认决策：未找到匹配的决策规则" {
		t.Error("Expected default decision reasoning")
	}

	if decision.Confidence != 0.6 {
		t.Errorf("Expected default confidence 0.6, got %f", decision.Confidence)
	}
}

func TestComponentRegistration(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()

	// 测试组件注册（使用mock组件）
	mockTaskPlanner := &mockTaskPlanner{name: "MockTaskPlanner"}
	mockStrategySelector := &mockStrategySelector{name: "MockStrategySelector"}
	mockContextLayer := &mockContextLayer{name: "MockContextLayer"}

	// 注册组件
	err := center.RegisterTaskPlanner(mockTaskPlanner)
	if err != nil {
		t.Fatalf("Failed to register task planner: %v", err)
	}

	err = center.RegisterStrategySelector(mockStrategySelector)
	if err != nil {
		t.Fatalf("Failed to register strategy selector: %v", err)
	}

	err = center.RegisterContextLayer(mockContextLayer)
	if err != nil {
		t.Fatalf("Failed to register context layer: %v", err)
	}

	// 测试重复注册
	err = center.RegisterTaskPlanner(mockTaskPlanner)
	if err == nil {
		t.Error("Expected error when registering duplicate task planner")
	}
}

func TestDecisionStatistics(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 初始统计
	stats := center.GetStats()
	if stats.TotalDecisions != 0 {
		t.Errorf("Expected 0 decisions initially, got %d", stats.TotalDecisions)
	}

	// 执行几个决策
	intents := []*interfaces.QueryIntent{
		{IntentType: "debugging", Domain: "programming", Complexity: 0.5, Confidence: 0.8},
		{IntentType: "procedural", Domain: "frontend", Complexity: 0.6, Confidence: 0.7},
		{IntentType: "technical", Domain: "backend", Complexity: 0.4, Confidence: 0.6},
	}

	for _, intent := range intents {
		intent.OriginalQuery = "test query"
		intent.Timestamp = time.Now()
		intent.Keywords = createKeywords([]string{"test"})
		intent.TechStack = []string{}

		_, err := center.MakeDecision(ctx, intent)
		if err != nil {
			t.Fatalf("Failed to make decision: %v", err)
		}
	}

	// 验证统计更新
	stats = center.GetStats()
	if stats.TotalDecisions != len(intents) {
		t.Errorf("Expected %d decisions, got %d", len(intents), stats.TotalDecisions)
	}

	// 验证意图分布
	if stats.DecisionByIntent["debugging"] != 1 {
		t.Error("Expected 1 debugging decision")
	}

	if stats.DecisionByIntent["procedural"] != 1 {
		t.Error("Expected 1 procedural decision")
	}

	// 验证领域分布
	if stats.DecisionByDomain["programming"] != 1 {
		t.Error("Expected 1 programming domain decision")
	}

	// 验证规则使用统计
	if len(stats.RuleUsageStats) == 0 {
		t.Error("Expected rule usage statistics")
	}
}

func TestOptimizeDecisionStrategy(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	err := center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	// 执行一些决策以生成统计数据
	intent := &interfaces.QueryIntent{
		OriginalQuery: "test optimization",
		Timestamp:     time.Now(),
		IntentType:    "debugging",
		Domain:        "programming",
		Complexity:    0.5,
		Keywords:      createActionKeywords("test"),
		TechStack:     []string{},
		Confidence:    0.8,
	}

	_, err = center.MakeDecision(ctx, intent)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	// 测试策略优化
	err = center.OptimizeDecisionStrategy()
	if err != nil {
		t.Fatalf("Failed to optimize decision strategy: %v", err)
	}

	// 验证权重已调整
	if len(center.ruleWeights) == 0 {
		t.Error("Expected rule weights to be adjusted")
	}
}

func TestErrorConditions(t *testing.T) {
	center := NewBasicIntelligentDecisionCenter()
	ctx := context.Background()

	// 测试未启动状态下的决策
	intent := &interfaces.QueryIntent{
		OriginalQuery: "test",
		IntentType:    "technical",
		Domain:        "programming",
		Complexity:    0.5,
		Confidence:    0.8,
	}

	_, err := center.MakeDecision(ctx, intent)
	if err == nil {
		t.Error("Expected error when making decision on stopped center")
	}

	// 启动后测试nil意图
	err = center.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start decision center: %v", err)
	}
	defer center.Stop(ctx)

	_, err = center.MakeDecision(ctx, nil)
	if err == nil {
		t.Error("Expected error when making decision with nil intent")
	}
}

// Mock implementations for testing
type mockTaskPlanner struct {
	name string
}

func (m *mockTaskPlanner) Name() string              { return m.name }
func (m *mockTaskPlanner) GetCapabilities() []string { return []string{"mock"} }
func (m *mockTaskPlanner) PlanTasks(ctx context.Context, intent *interfaces.QueryIntent) (*interfaces.TaskPlan, error) {
	return nil, nil
}
func (m *mockTaskPlanner) ValidatePlan(plan *interfaces.TaskPlan) error { return nil }
func (m *mockTaskPlanner) OptimizePlan(plan *interfaces.TaskPlan) (*interfaces.TaskPlan, error) {
	return plan, nil
}

type mockStrategySelector struct {
	name string
}

func (m *mockStrategySelector) Name() string { return m.name }
func (m *mockStrategySelector) SelectStrategies(ctx context.Context, intent *interfaces.QueryIntent, plan *interfaces.TaskPlan) (*interfaces.StrategySelection, error) {
	return nil, nil
}
func (m *mockStrategySelector) RegisterStrategy(strategy interfaces.ProcessingStrategy) error {
	return nil
}
func (m *mockStrategySelector) GetAvailableStrategies() []string { return []string{} }
func (m *mockStrategySelector) EvaluateStrategyFitness(strategy string, intent *interfaces.QueryIntent) float64 {
	return 0.5
}
func (m *mockStrategySelector) UpdateStrategyPerformance(strategyName string, performance float64) error {
	return nil
}
func (m *mockStrategySelector) GetStrategyStatistics() map[string]*interfaces.StrategyStats {
	return nil
}

type mockContextLayer struct {
	name string
}

func (m *mockContextLayer) Name() string { return m.name }
func (m *mockContextLayer) BuildContext(ctx context.Context, intent *interfaces.QueryIntent) (*interfaces.ProcessingContext, error) {
	return nil, nil
}
func (m *mockContextLayer) EnrichContext(context *interfaces.ProcessingContext, additionalInfo map[string]interface{}) error {
	return nil
}
func (m *mockContextLayer) GetRelevantHistory(sessionID string, limit int) ([]*interfaces.QueryIntent, error) {
	return nil, nil
}
func (m *mockContextLayer) UpdateUserPreferences(userID string, preferences map[string]interface{}) error {
	return nil
}
func (m *mockContextLayer) AnalyzeContextRelevance(context *interfaces.ProcessingContext, intent *interfaces.QueryIntent) float64 {
	return 0.5
}
func (m *mockContextLayer) ExtractContextPatterns(sessionID string) (*interfaces.ContextPatterns, error) {
	return nil, nil
}
