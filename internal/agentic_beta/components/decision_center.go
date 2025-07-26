package components

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/interfaces"
)

// ============================================================================
// 🧠 智能决策中心 - 第二阶段核心实现 (B→C)
// ============================================================================

// BasicIntelligentDecisionCenter 基础智能决策中心
// 设计原则：简单规则引擎 + 可扩展架构 + 组件协调
type BasicIntelligentDecisionCenter struct {
	// 基础配置
	name    string
	version string
	enabled bool
	started bool

	// 决策规则引擎
	decisionRules map[string]*DecisionRule
	ruleWeights   map[string]float64

	// 组件注册表 (为后续阶段预留)
	mu                sync.RWMutex
	taskPlanners      map[string]interfaces.TaskPlanner
	strategySelectors map[string]interfaces.StrategySelector
	contextLayers     map[string]interfaces.ContextAwareLayer

	// 决策历史和统计
	decisionHistory []*interfaces.ProcessingDecision
	stats           *DecisionCenterStats

	// 配置和优化
	config map[string]interface{}
}

// DecisionRule 决策规则
type DecisionRule struct {
	RuleID      string                                                                `json:"rule_id"`
	Name        string                                                                `json:"name"`
	Condition   func(*interfaces.QueryIntent) bool                                    `json:"-"`
	Action      func(*interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) `json:"-"`
	Priority    float64                                                               `json:"priority"`
	Confidence  float64                                                               `json:"confidence"`
	Description string                                                                `json:"description"`
	Metadata    map[string]interface{}                                                `json:"metadata"`
}

// DecisionCenterStats 决策中心统计
type DecisionCenterStats struct {
	TotalDecisions     int                        `json:"total_decisions"`
	DecisionByIntent   map[string]int             `json:"decision_by_intent"`
	DecisionByDomain   map[string]int             `json:"decision_by_domain"`
	AverageProcessTime time.Duration              `json:"average_process_time"`
	SuccessRate        float64                    `json:"success_rate"`
	RuleUsageStats     map[string]*RuleUsageStats `json:"rule_usage_stats"`
	LastUpdated        time.Time                  `json:"last_updated"`
}

// RuleUsageStats 规则使用统计
type RuleUsageStats struct {
	RuleID       string        `json:"rule_id"`
	UsageCount   int           `json:"usage_count"`
	SuccessCount int           `json:"success_count"`
	AvgTime      time.Duration `json:"avg_time"`
	LastUsed     time.Time     `json:"last_used"`
}

// NewBasicIntelligentDecisionCenter 创建基础智能决策中心
func NewBasicIntelligentDecisionCenter() *BasicIntelligentDecisionCenter {
	center := &BasicIntelligentDecisionCenter{
		name:              "BasicIntelligentDecisionCenter",
		version:           "v1.0.0",
		enabled:           true,
		started:           false,
		decisionRules:     make(map[string]*DecisionRule),
		ruleWeights:       make(map[string]float64),
		taskPlanners:      make(map[string]interfaces.TaskPlanner),
		strategySelectors: make(map[string]interfaces.StrategySelector),
		contextLayers:     make(map[string]interfaces.ContextAwareLayer),
		decisionHistory:   make([]*interfaces.ProcessingDecision, 0),
		config:            make(map[string]interface{}),
		stats: &DecisionCenterStats{
			DecisionByIntent: make(map[string]int),
			DecisionByDomain: make(map[string]int),
			RuleUsageStats:   make(map[string]*RuleUsageStats),
		},
	}

	// 初始化默认决策规则
	center.initializeDefaultRules()

	return center
}

// ============================================================================
// 🎯 实现 IntelligentDecisionCenter 接口
// ============================================================================

// MakeDecision 核心决策方法 - 基于意图做出处理决策
func (bidc *BasicIntelligentDecisionCenter) MakeDecision(ctx context.Context, intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	if !bidc.enabled {
		return nil, fmt.Errorf("决策中心已禁用")
	}

	if !bidc.started {
		return nil, fmt.Errorf("决策中心未启动")
	}

	if intent == nil {
		return nil, fmt.Errorf("查询意图不能为空")
	}

	startTime := time.Now()

	// 🎯 第一步：选择适用的决策规则
	applicableRules := bidc.selectApplicableRules(intent)
	if len(applicableRules) == 0 {
		return bidc.makeDefaultDecision(intent)
	}

	// 🏆 第二步：选择最佳规则（按优先级和置信度）
	bestRule := bidc.selectBestRule(applicableRules, intent)

	// ⚡ 第三步：执行决策
	decision, err := bestRule.Action(intent)
	if err != nil {
		return bidc.makeDefaultDecision(intent)
	}

	// 📝 第四步：完善决策信息
	decision.DecisionID = bidc.generateDecisionID()
	decision.Timestamp = time.Now()
	decision.DecisionReasoning = fmt.Sprintf("使用规则: %s (置信度: %.2f)", bestRule.Name, bestRule.Confidence)
	decision.Confidence = bestRule.Confidence

	// 📊 第五步：更新统计和历史
	processingTime := time.Since(startTime)
	bidc.updateStats(intent, decision, bestRule.RuleID, processingTime)
	bidc.addToHistory(decision)

	return decision, nil
}

// RegisterTaskPlanner 注册任务规划器
func (bidc *BasicIntelligentDecisionCenter) RegisterTaskPlanner(planner interfaces.TaskPlanner) error {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	name := planner.Name()
	if _, exists := bidc.taskPlanners[name]; exists {
		return fmt.Errorf("任务规划器 %s 已注册", name)
	}

	bidc.taskPlanners[name] = planner
	fmt.Printf("📋 TaskPlanner registered: %s\n", name)
	return nil
}

// RegisterStrategySelector 注册策略选择器
func (bidc *BasicIntelligentDecisionCenter) RegisterStrategySelector(selector interfaces.StrategySelector) error {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	name := selector.Name()
	if _, exists := bidc.strategySelectors[name]; exists {
		return fmt.Errorf("策略选择器 %s 已注册", name)
	}

	bidc.strategySelectors[name] = selector
	fmt.Printf("🎮 StrategySelector registered: %s\n", name)
	return nil
}

// RegisterContextLayer 注册上下文感知层
func (bidc *BasicIntelligentDecisionCenter) RegisterContextLayer(layer interfaces.ContextAwareLayer) error {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	name := layer.Name()
	if _, exists := bidc.contextLayers[name]; exists {
		return fmt.Errorf("上下文感知层 %s 已注册", name)
	}

	bidc.contextLayers[name] = layer
	fmt.Printf("🌐 ContextAwareLayer registered: %s\n", name)
	return nil
}

// GetDecisionHistory 获取决策历史
func (bidc *BasicIntelligentDecisionCenter) GetDecisionHistory() []*interfaces.ProcessingDecision {
	bidc.mu.RLock()
	defer bidc.mu.RUnlock()

	// 返回副本，避免并发问题
	history := make([]*interfaces.ProcessingDecision, len(bidc.decisionHistory))
	copy(history, bidc.decisionHistory)
	return history
}

// OptimizeDecisionStrategy 优化决策策略
func (bidc *BasicIntelligentDecisionCenter) OptimizeDecisionStrategy() error {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	// 第一阶段：简单的规则权重调整
	for ruleID, stats := range bidc.stats.RuleUsageStats {
		if stats.UsageCount > 0 {
			successRate := float64(stats.SuccessCount) / float64(stats.UsageCount)

			// 根据成功率调整权重
			if successRate > 0.8 {
				bidc.ruleWeights[ruleID] = 1.2 // 提高权重
			} else if successRate < 0.5 {
				bidc.ruleWeights[ruleID] = 0.8 // 降低权重
			}
		}
	}

	fmt.Printf("🔧 决策策略已优化，调整了 %d 个规则的权重\n", len(bidc.ruleWeights))
	return nil
}

// Start 启动决策中心
func (bidc *BasicIntelligentDecisionCenter) Start(ctx context.Context) error {
	if bidc.started {
		return fmt.Errorf("决策中心已启动")
	}

	bidc.started = true
	fmt.Printf("🚀 IntelligentDecisionCenter started: %s (%s)\n", bidc.name, bidc.version)
	return nil
}

// Stop 停止决策中心
func (bidc *BasicIntelligentDecisionCenter) Stop(ctx context.Context) error {
	if !bidc.started {
		return nil
	}

	bidc.started = false
	fmt.Printf("⏹️ IntelligentDecisionCenter stopped: %s\n", bidc.name)
	return nil
}

// ============================================================================
// 🔧 核心决策逻辑
// ============================================================================

// selectApplicableRules 选择适用的决策规则
func (bidc *BasicIntelligentDecisionCenter) selectApplicableRules(intent *interfaces.QueryIntent) []*DecisionRule {
	var applicable []*DecisionRule

	for _, rule := range bidc.decisionRules {
		if rule.Condition(intent) {
			applicable = append(applicable, rule)
		}
	}

	return applicable
}

// selectBestRule 选择最佳规则
func (bidc *BasicIntelligentDecisionCenter) selectBestRule(rules []*DecisionRule, intent *interfaces.QueryIntent) *DecisionRule {
	if len(rules) == 0 {
		return nil
	}

	if len(rules) == 1 {
		return rules[0]
	}

	// 选择优先级最高、置信度最高的规则
	best := rules[0]
	bestScore := bidc.calculateRuleScore(best, intent)

	for _, rule := range rules[1:] {
		score := bidc.calculateRuleScore(rule, intent)
		if score > bestScore {
			best = rule
			bestScore = score
		}
	}

	return best
}

// calculateRuleScore 计算规则评分
func (bidc *BasicIntelligentDecisionCenter) calculateRuleScore(rule *DecisionRule, intent *interfaces.QueryIntent) float64 {
	// 基础评分 = 优先级 * 置信度
	baseScore := rule.Priority * rule.Confidence

	// 应用权重调整
	if weight, exists := bidc.ruleWeights[rule.RuleID]; exists {
		baseScore *= weight
	}

	// 可以根据意图特征进一步调整
	if intent.Complexity > 0.7 {
		baseScore *= 1.1 // 复杂查询稍微提高评分
	}

	return baseScore
}

// makeDefaultDecision 制作默认决策
func (bidc *BasicIntelligentDecisionCenter) makeDefaultDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	// 创建基础任务计划
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "default_task_1",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "basic"},
				Priority:        1,
			},
		},
		ExecutionOrder:    []string{"default_task_1"},
		ParallelExecution: false,
		MaxRetries:        3,
		TimeoutSeconds:    30,
		Priority:          1,
		Metadata:          map[string]interface{}{"source": "default_decision"},
	}

	decision := &interfaces.ProcessingDecision{
		DecisionID:         bidc.generateDecisionID(),
		Intent:             intent,
		Timestamp:          time.Now(),
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"basic_enhancement"},
		ContextInfo:        map[string]interface{}{"default": true},
		DecisionReasoning:  "默认决策：未找到匹配的决策规则",
		Confidence:         0.6,
		Metadata: map[string]interface{}{
			"decision_type": "default",
			"rule_id":       "default",
		},
	}

	return decision, nil
}

// ============================================================================
// 🛠️ 辅助方法
// ============================================================================

// generateDecisionID 生成决策ID
func (bidc *BasicIntelligentDecisionCenter) generateDecisionID() string {
	return fmt.Sprintf("decision_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%1000000)
}

// generatePlanID 生成计划ID
func (bidc *BasicIntelligentDecisionCenter) generatePlanID() string {
	return fmt.Sprintf("plan_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%1000000)
}

// updateStats 更新统计信息
func (bidc *BasicIntelligentDecisionCenter) updateStats(intent *interfaces.QueryIntent, decision *interfaces.ProcessingDecision, ruleID string, processingTime time.Duration) {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	// 更新基础统计
	bidc.stats.TotalDecisions++
	bidc.stats.DecisionByIntent[intent.IntentType]++
	bidc.stats.DecisionByDomain[intent.Domain]++

	// 更新平均处理时间
	if bidc.stats.TotalDecisions == 1 {
		bidc.stats.AverageProcessTime = processingTime
	} else {
		bidc.stats.AverageProcessTime = (bidc.stats.AverageProcessTime*time.Duration(bidc.stats.TotalDecisions-1) + processingTime) / time.Duration(bidc.stats.TotalDecisions)
	}

	// 更新规则使用统计
	if _, exists := bidc.stats.RuleUsageStats[ruleID]; !exists {
		bidc.stats.RuleUsageStats[ruleID] = &RuleUsageStats{
			RuleID: ruleID,
		}
	}

	ruleStats := bidc.stats.RuleUsageStats[ruleID]
	ruleStats.UsageCount++
	ruleStats.SuccessCount++ // 第一阶段假设都成功
	ruleStats.AvgTime = (ruleStats.AvgTime*time.Duration(ruleStats.UsageCount-1) + processingTime) / time.Duration(ruleStats.UsageCount)
	ruleStats.LastUsed = time.Now()

	// 更新成功率（第一阶段简化为100%）
	bidc.stats.SuccessRate = 1.0
	bidc.stats.LastUpdated = time.Now()
}

// addToHistory 添加到历史记录
func (bidc *BasicIntelligentDecisionCenter) addToHistory(decision *interfaces.ProcessingDecision) {
	bidc.mu.Lock()
	defer bidc.mu.Unlock()

	// 保持历史记录数量限制（最多100条）
	maxHistory := 100
	bidc.decisionHistory = append(bidc.decisionHistory, decision)

	if len(bidc.decisionHistory) > maxHistory {
		bidc.decisionHistory = bidc.decisionHistory[1:]
	}
}

// initializeDefaultRules 初始化默认决策规则
func (bidc *BasicIntelligentDecisionCenter) initializeDefaultRules() {
	// 🐛 调试类查询规则
	bidc.decisionRules["debug_rule"] = &DecisionRule{
		RuleID:      "debug_rule",
		Name:        "调试类查询处理",
		Condition:   func(intent *interfaces.QueryIntent) bool { return intent.IntentType == "debugging" },
		Action:      bidc.createDebugDecision,
		Priority:    0.9,
		Confidence:  0.8,
		Description: "针对调试类查询的专门处理",
		Metadata:    map[string]interface{}{"type": "debugging"},
	}

	// 📋 过程类查询规则
	bidc.decisionRules["procedural_rule"] = &DecisionRule{
		RuleID:      "procedural_rule",
		Name:        "过程类查询处理",
		Condition:   func(intent *interfaces.QueryIntent) bool { return intent.IntentType == "procedural" },
		Action:      bidc.createProceduralDecision,
		Priority:    0.8,
		Confidence:  0.7,
		Description: "针对过程类查询的步骤化处理",
		Metadata:    map[string]interface{}{"type": "procedural"},
	}

	// 📚 概念类查询规则
	bidc.decisionRules["conceptual_rule"] = &DecisionRule{
		RuleID:      "conceptual_rule",
		Name:        "概念类查询处理",
		Condition:   func(intent *interfaces.QueryIntent) bool { return intent.IntentType == "conceptual" },
		Action:      bidc.createConceptualDecision,
		Priority:    0.7,
		Confidence:  0.6,
		Description: "针对概念类查询的理论性处理",
		Metadata:    map[string]interface{}{"type": "conceptual"},
	}

	// 🛠️ 技术类查询规则
	bidc.decisionRules["technical_rule"] = &DecisionRule{
		RuleID:      "technical_rule",
		Name:        "技术类查询处理",
		Condition:   func(intent *interfaces.QueryIntent) bool { return intent.IntentType == "technical" },
		Action:      bidc.createTechnicalDecision,
		Priority:    0.6,
		Confidence:  0.5,
		Description: "针对技术类查询的实现性处理",
		Metadata:    map[string]interface{}{"type": "technical"},
	}

	// 🔥 高复杂度查询规则
	bidc.decisionRules["complex_rule"] = &DecisionRule{
		RuleID:      "complex_rule",
		Name:        "高复杂度查询处理",
		Condition:   func(intent *interfaces.QueryIntent) bool { return intent.Complexity > 0.7 },
		Action:      bidc.createComplexDecision,
		Priority:    0.95,
		Confidence:  0.85,
		Description: "针对高复杂度查询的特殊处理",
		Metadata:    map[string]interface{}{"type": "complex"},
	}

	fmt.Printf("🔧 Initialized %d default decision rules\n", len(bidc.decisionRules))
}

// ============================================================================
// 🎯 具体决策创建方法
// ============================================================================

// createDebugDecision 创建调试类决策
func (bidc *BasicIntelligentDecisionCenter) createDebugDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "debug_enhance",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "debug_focused", "priority": "high"},
				Priority:        1,
			},
			{
				TaskID:          "debug_filter",
				Type:            "filter",
				TargetComponent: "noise_filter",
				Parameters:      map[string]interface{}{"strategy": "debug_noise_removal"},
				Priority:        2,
			},
		},
		ExecutionOrder:    []string{"debug_enhance", "debug_filter"},
		ParallelExecution: false,
		MaxRetries:        2,
		TimeoutSeconds:    20,
		Priority:          2,
		Metadata:          map[string]interface{}{"intent_type": "debugging"},
	}

	return &interfaces.ProcessingDecision{
		Intent:             intent,
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"debug_enhancement", "error_analysis"},
		ContextInfo:        map[string]interface{}{"focus": "problem_solving"},
		Metadata:           map[string]interface{}{"rule": "debug_rule"},
	}, nil
}

// createProceduralDecision 创建过程类决策
func (bidc *BasicIntelligentDecisionCenter) createProceduralDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "step_enhance",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "step_by_step", "detail_level": "high"},
				Priority:        1,
			},
			{
				TaskID:          "procedure_adapt",
				Type:            "adapt",
				TargetComponent: "domain_adapter",
				Parameters:      map[string]interface{}{"focus": "procedural_knowledge"},
				Priority:        2,
			},
		},
		ExecutionOrder:    []string{"step_enhance", "procedure_adapt"},
		ParallelExecution: true,
		MaxRetries:        3,
		TimeoutSeconds:    25,
		Priority:          1,
		Metadata:          map[string]interface{}{"intent_type": "procedural"},
	}

	return &interfaces.ProcessingDecision{
		Intent:             intent,
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"step_enhancement", "tutorial_search"},
		ContextInfo:        map[string]interface{}{"focus": "how_to_guide"},
		Metadata:           map[string]interface{}{"rule": "procedural_rule"},
	}, nil
}

// createConceptualDecision 创建概念类决策
func (bidc *BasicIntelligentDecisionCenter) createConceptualDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "concept_enhance",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "conceptual_expansion"},
				Priority:        1,
			},
			{
				TaskID:          "theory_adapt",
				Type:            "adapt",
				TargetComponent: "domain_adapter",
				Parameters:      map[string]interface{}{"focus": "theoretical_knowledge"},
				Priority:        2,
			},
		},
		ExecutionOrder:    []string{"concept_enhance", "theory_adapt"},
		ParallelExecution: true,
		MaxRetries:        2,
		TimeoutSeconds:    30,
		Priority:          1,
		Metadata:          map[string]interface{}{"intent_type": "conceptual"},
	}

	return &interfaces.ProcessingDecision{
		Intent:             intent,
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"concept_enhancement", "definition_search"},
		ContextInfo:        map[string]interface{}{"focus": "theoretical_understanding"},
		Metadata:           map[string]interface{}{"rule": "conceptual_rule"},
	}, nil
}

// createTechnicalDecision 创建技术类决策
func (bidc *BasicIntelligentDecisionCenter) createTechnicalDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "tech_enhance",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "technical_terms"},
				Priority:        1,
			},
			{
				TaskID:          "tech_adapt",
				Type:            "adapt",
				TargetComponent: "domain_adapter",
				Parameters:      map[string]interface{}{"domain": intent.Domain},
				Priority:        2,
			},
		},
		ExecutionOrder:    []string{"tech_enhance", "tech_adapt"},
		ParallelExecution: true,
		MaxRetries:        3,
		TimeoutSeconds:    35,
		Priority:          1,
		Metadata:          map[string]interface{}{"intent_type": "technical"},
	}

	return &interfaces.ProcessingDecision{
		Intent:             intent,
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"technical_enhancement", "implementation_search"},
		ContextInfo:        map[string]interface{}{"focus": "practical_implementation"},
		Metadata:           map[string]interface{}{"rule": "technical_rule"},
	}, nil
}

// createComplexDecision 创建复杂查询决策
func (bidc *BasicIntelligentDecisionCenter) createComplexDecision(intent *interfaces.QueryIntent) (*interfaces.ProcessingDecision, error) {
	taskPlan := &interfaces.TaskPlan{
		PlanID:       bidc.generatePlanID(),
		TargetIntent: intent,
		CreatedAt:    time.Now(),
		Tasks: []*interfaces.Task{
			{
				TaskID:          "complex_enhance",
				Type:            "enhance",
				TargetComponent: "semantic_enhancer",
				Parameters:      map[string]interface{}{"strategy": "comprehensive", "depth": "high"},
				Priority:        1,
			},
			{
				TaskID:          "complex_filter",
				Type:            "filter",
				TargetComponent: "noise_filter",
				Parameters:      map[string]interface{}{"strategy": "advanced_noise_removal"},
				Priority:        2,
			},
			{
				TaskID:          "complex_adapt",
				Type:            "adapt",
				TargetComponent: "domain_adapter",
				Parameters:      map[string]interface{}{"multi_domain": true, "depth": "high"},
				Priority:        3,
			},
		},
		ExecutionOrder:    []string{"complex_enhance", "complex_filter", "complex_adapt"},
		ParallelExecution: false, // 复杂查询按顺序处理
		MaxRetries:        3,
		TimeoutSeconds:    45,
		Priority:          3,
		Metadata:          map[string]interface{}{"intent_type": "complex", "complexity": intent.Complexity},
	}

	return &interfaces.ProcessingDecision{
		Intent:             intent,
		TaskPlan:           taskPlan,
		SelectedStrategies: []string{"comprehensive_enhancement", "multi_faceted_search", "complex_analysis"},
		ContextInfo:        map[string]interface{}{"focus": "comprehensive_analysis", "requires_deep_understanding": true},
		Metadata:           map[string]interface{}{"rule": "complex_rule"},
	}, nil
}

// GetStats 获取统计信息
func (bidc *BasicIntelligentDecisionCenter) GetStats() *DecisionCenterStats {
	bidc.mu.RLock()
	defer bidc.mu.RUnlock()
	return bidc.stats
}
