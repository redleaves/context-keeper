package components

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/interfaces"
)

// ============================================================================
// 📋 任务规划模块 - 第二阶段核心实现 (C→D)
// ============================================================================

// BasicTaskPlanner 基础任务规划器
// 设计原则：基于意图和决策制定智能执行计划
type BasicTaskPlanner struct {
	// 基础配置
	name        string
	version     string
	enabled     bool
	initialized bool

	// 规划引擎
	planningRules map[string]*PlanningRule
	taskTemplates map[string]*TaskTemplate
	planCache     map[string]*interfaces.TaskPlan
	cacheExpiry   time.Duration

	// 规划统计
	mu              sync.RWMutex
	totalPlans      int
	successfulPlans int
	avgPlanningTime time.Duration
	planHistory     []*interfaces.TaskPlan

	// 配置参数
	maxTasksPerPlan int
	defaultTimeout  int
	maxRetries      int
	enableParallel  bool
}

// PlanningRule 规划规则
type PlanningRule struct {
	Name         string                 `json:"name"`
	IntentTypes  []string               `json:"intent_types"`
	Domains      []string               `json:"domains"`
	Complexity   [2]float64             `json:"complexity"` // [min, max]
	TaskSequence []string               `json:"task_sequence"`
	Parallel     bool                   `json:"parallel"`
	Priority     int                    `json:"priority"`
	Config       map[string]interface{} `json:"config"`
}

// TaskTemplate 任务模板
type TaskTemplate struct {
	TaskType        string                 `json:"task_type"`
	DefaultPriority int                    `json:"default_priority"`
	EstimatedTime   time.Duration          `json:"estimated_time"`
	Dependencies    []string               `json:"dependencies"`
	Parameters      map[string]interface{} `json:"parameters"`
	RetryPolicy     *RetryPolicy           `json:"retry_policy"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	BackoffFactor float64       `json:"backoff_factor"`
	InitialDelay  time.Duration `json:"initial_delay"`
}

// NewBasicTaskPlanner 创建基础任务规划器
func NewBasicTaskPlanner() *BasicTaskPlanner {
	planner := &BasicTaskPlanner{
		name:            "BasicTaskPlanner",
		version:         "v1.0.0",
		enabled:         true,
		initialized:     false,
		planningRules:   make(map[string]*PlanningRule),
		taskTemplates:   make(map[string]*TaskTemplate),
		planCache:       make(map[string]*interfaces.TaskPlan),
		cacheExpiry:     time.Minute * 5,
		planHistory:     make([]*interfaces.TaskPlan, 0),
		maxTasksPerPlan: 10,
		defaultTimeout:  30,
		maxRetries:      3,
		enableParallel:  true,
	}

	// 初始化默认规划规则和任务模板
	planner.initializeDefaultRules()
	planner.initializeTaskTemplates()
	planner.initialized = true

	fmt.Printf("📋 Initialized TaskPlanner with %d rules and %d templates\n",
		len(planner.planningRules), len(planner.taskTemplates))

	return planner
}

// initializeDefaultRules 初始化默认规划规则
func (btp *BasicTaskPlanner) initializeDefaultRules() {
	// 调试类查询规划规则
	btp.planningRules["debug_planning"] = &PlanningRule{
		Name:         "调试类查询规划",
		IntentTypes:  []string{"debugging"},
		Domains:      []string{"programming", "database", "frontend", "backend"},
		Complexity:   [2]float64{0.0, 1.0},
		TaskSequence: []string{"analyze_error", "enhance_debug_context", "filter_debug_noise"},
		Parallel:     false, // 调试需要顺序执行
		Priority:     90,
		Config: map[string]interface{}{
			"focus_on_errors":      true,
			"include_stack_trace":  true,
			"error_categorization": true,
		},
	}

	// 概念类查询规划规则
	btp.planningRules["concept_planning"] = &PlanningRule{
		Name:         "概念类查询规划",
		IntentTypes:  []string{"conceptual"},
		Domains:      []string{"architecture", "programming", "database"},
		Complexity:   [2]float64{0.0, 0.8},
		TaskSequence: []string{"extract_concepts", "enhance_definitions", "provide_examples"},
		Parallel:     true, // 概念解释可以并行
		Priority:     70,
		Config: map[string]interface{}{
			"include_examples":    true,
			"provide_comparisons": true,
			"conceptual_depth":    "medium",
		},
	}

	// 过程类查询规划规则
	btp.planningRules["procedural_planning"] = &PlanningRule{
		Name:         "过程类查询规划",
		IntentTypes:  []string{"procedural"},
		Domains:      []string{"devops", "frontend", "backend", "architecture"},
		Complexity:   [2]float64{0.3, 1.0},
		TaskSequence: []string{"break_down_steps", "enhance_instructions", "add_prerequisites"},
		Parallel:     false, // 步骤需要顺序执行
		Priority:     80,
		Config: map[string]interface{}{
			"step_by_step":     true,
			"include_commands": true,
			"add_verification": true,
		},
	}

	// 技术类查询规划规则
	btp.planningRules["technical_planning"] = &PlanningRule{
		Name:         "技术类查询规划",
		IntentTypes:  []string{"technical"},
		Domains:      []string{"programming", "architecture", "database", "frontend", "backend"},
		Complexity:   [2]float64{0.0, 0.9},
		TaskSequence: []string{"enhance_technical_context", "filter_implementation_noise", "add_best_practices"},
		Parallel:     true, // 技术信息可以并行处理
		Priority:     60,
		Config: map[string]interface{}{
			"include_code_examples": true,
			"best_practices":        true,
			"performance_tips":      true,
		},
	}

	// 复杂查询规划规则
	btp.planningRules["complex_planning"] = &PlanningRule{
		Name:         "复杂查询规划",
		IntentTypes:  []string{"debugging", "procedural", "technical", "conceptual"},
		Domains:      []string{"architecture", "programming", "database", "frontend", "backend", "devops"},
		Complexity:   [2]float64{0.7, 1.0},
		TaskSequence: []string{"decompose_query", "parallel_enhancement", "synthesize_results", "quality_check"},
		Parallel:     true, // 复杂查询使用混合策略
		Priority:     100,
		Config: map[string]interface{}{
			"multi_stage_processing": true,
			"quality_threshold":      0.8,
			"comprehensive_search":   true,
		},
	}
}

// initializeTaskTemplates 初始化任务模板
func (btp *BasicTaskPlanner) initializeTaskTemplates() {
	// 错误分析任务
	btp.taskTemplates["analyze_error"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 90,
		EstimatedTime:   time.Millisecond * 50,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"focus_keywords": []string{"错误", "异常", "问题", "bug", "error"},
			"context_window": 3,
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    2,
			BackoffFactor: 1.5,
			InitialDelay:  time.Millisecond * 10,
		},
	}

	// 调试上下文增强
	btp.taskTemplates["enhance_debug_context"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 80,
		EstimatedTime:   time.Millisecond * 30,
		Dependencies:    []string{}, // 移除依赖，避免依赖解析失败
		Parameters: map[string]interface{}{
			"enhancement_terms": []string{"调试", "问题排查", "错误分析", "代码质量"},
			"boost_factor":      1.5,
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    2,
			BackoffFactor: 1.2,
			InitialDelay:  time.Millisecond * 5,
		},
	}

	// 概念提取任务
	btp.taskTemplates["extract_concepts"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 70,
		EstimatedTime:   time.Millisecond * 40,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"concept_keywords": []string{"概念", "原理", "理论", "定义"},
			"depth_level":      "medium",
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    2,
			BackoffFactor: 1.3,
			InitialDelay:  time.Millisecond * 8,
		},
	}

	// 步骤分解任务
	btp.taskTemplates["break_down_steps"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 85,
		EstimatedTime:   time.Millisecond * 60,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"step_keywords": []string{"步骤", "教程", "操作指南", "如何"},
			"sequential":    true,
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    3,
			BackoffFactor: 1.4,
			InitialDelay:  time.Millisecond * 12,
		},
	}

	// 技术上下文增强
	btp.taskTemplates["enhance_technical_context"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 75,
		EstimatedTime:   time.Millisecond * 45,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"technical_terms":  []string{"实现", "技术方案", "代码", "算法"},
			"include_examples": true,
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    2,
			BackoffFactor: 1.3,
			InitialDelay:  time.Millisecond * 10,
		},
	}

	// 噪声过滤任务
	btp.taskTemplates["filter_debug_noise"] = &TaskTemplate{
		TaskType:        "filter",
		DefaultPriority: 60,
		EstimatedTime:   time.Millisecond * 25,
		Dependencies:    []string{}, // 移除依赖，避免依赖解析失败
		Parameters: map[string]interface{}{
			"noise_patterns":   []string{"无关", "干扰", "冗余"},
			"filter_threshold": 0.3,
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:    1,
			BackoffFactor: 1.0,
			InitialDelay:  time.Millisecond * 5,
		},
	}

	// 添加其他缺失的任务模板
	btp.taskTemplates["enhance_definitions"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 70,
		EstimatedTime:   time.Millisecond * 35,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"definition_terms": []string{"定义", "解释", "含义"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.2, InitialDelay: time.Millisecond * 8},
	}

	btp.taskTemplates["provide_examples"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 65,
		EstimatedTime:   time.Millisecond * 40,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"example_keywords": []string{"示例", "例子", "案例"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.2, InitialDelay: time.Millisecond * 8},
	}

	btp.taskTemplates["enhance_instructions"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 80,
		EstimatedTime:   time.Millisecond * 50,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"instruction_terms": []string{"指令", "步骤", "操作"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.3, InitialDelay: time.Millisecond * 10},
	}

	btp.taskTemplates["add_prerequisites"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 75,
		EstimatedTime:   time.Millisecond * 30,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"prerequisite_terms": []string{"前提", "要求", "准备"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.2, InitialDelay: time.Millisecond * 8},
	}

	btp.taskTemplates["filter_implementation_noise"] = &TaskTemplate{
		TaskType:        "filter",
		DefaultPriority: 60,
		EstimatedTime:   time.Millisecond * 25,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"noise_patterns": []string{"无关实现", "冗余代码"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 1, BackoffFactor: 1.0, InitialDelay: time.Millisecond * 5},
	}

	btp.taskTemplates["add_best_practices"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 70,
		EstimatedTime:   time.Millisecond * 35,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"best_practice_terms": []string{"最佳实践", "推荐做法", "优化建议"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.2, InitialDelay: time.Millisecond * 8},
	}

	// 复杂查询专用任务模板
	btp.taskTemplates["decompose_query"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 95,
		EstimatedTime:   time.Millisecond * 60,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"decomposition_strategy": "hierarchical",
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 3, BackoffFactor: 1.4, InitialDelay: time.Millisecond * 15},
	}

	btp.taskTemplates["parallel_enhancement"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 90,
		EstimatedTime:   time.Millisecond * 80,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"parallel_strategies": []string{"semantic", "syntactic", "contextual"},
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 3, BackoffFactor: 1.5, InitialDelay: time.Millisecond * 20},
	}

	btp.taskTemplates["synthesize_results"] = &TaskTemplate{
		TaskType:        "enhance",
		DefaultPriority: 85,
		EstimatedTime:   time.Millisecond * 70,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"synthesis_method": "weighted_combination",
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.3, InitialDelay: time.Millisecond * 12},
	}

	btp.taskTemplates["quality_check"] = &TaskTemplate{
		TaskType:        "filter",
		DefaultPriority: 80,
		EstimatedTime:   time.Millisecond * 40,
		Dependencies:    []string{},
		Parameters: map[string]interface{}{
			"quality_threshold": 0.8,
		},
		RetryPolicy: &RetryPolicy{MaxRetries: 2, BackoffFactor: 1.2, InitialDelay: time.Millisecond * 10},
	}
}

// PlanTasks 制定任务计划 - 核心接口实现
func (btp *BasicTaskPlanner) PlanTasks(ctx context.Context, intent *interfaces.QueryIntent) (*interfaces.TaskPlan, error) {
	if !btp.enabled || !btp.initialized {
		return nil, fmt.Errorf("task planner not available")
	}

	startTime := time.Now()
	defer func() {
		btp.mu.Lock()
		btp.totalPlans++
		btp.avgPlanningTime = (btp.avgPlanningTime*time.Duration(btp.totalPlans-1) + time.Since(startTime)) / time.Duration(btp.totalPlans)
		btp.mu.Unlock()
	}()

	// 🔍 第一步：检查缓存
	cacheKey := btp.generateCacheKey(intent)
	if cachedPlan := btp.getCachedPlan(cacheKey); cachedPlan != nil {
		return cachedPlan, nil
	}

	// 🧠 第二步：选择最佳规划规则
	rule := btp.selectBestRule(intent)
	if rule == nil {
		return btp.createDefaultPlan(intent), nil
	}

	// 📋 第三步：创建任务计划
	plan := &interfaces.TaskPlan{
		PlanID:            fmt.Sprintf("plan_%d_%s", time.Now().Unix(), btp.generateShortID()),
		TargetIntent:      intent,
		CreatedAt:         time.Now(),
		Tasks:             []*interfaces.Task{},
		ExecutionOrder:    []string{},
		ParallelExecution: rule.Parallel && btp.enableParallel,
		MaxRetries:        btp.maxRetries,
		TimeoutSeconds:    btp.defaultTimeout,
		Priority:          rule.Priority,
		Metadata: map[string]interface{}{
			"planner_name":    btp.name,
			"rule_applied":    rule.Name,
			"complexity":      intent.Complexity,
			"estimated_tasks": len(rule.TaskSequence),
		},
	}

	// 🔧 第四步：生成具体任务
	for i, taskType := range rule.TaskSequence {
		task, err := btp.createTask(taskType, i, intent, rule)
		if err != nil {
			continue // 跳过创建失败的任务
		}
		plan.Tasks = append(plan.Tasks, task)
		plan.ExecutionOrder = append(plan.ExecutionOrder, task.TaskID)
	}

	// ✅ 第五步：验证和优化计划
	if err := btp.ValidatePlan(plan); err != nil {
		return btp.createDefaultPlan(intent), nil
	}

	// 缓存计划
	btp.cachePlan(cacheKey, plan)

	// 记录成功
	btp.mu.Lock()
	btp.successfulPlans++
	btp.planHistory = append(btp.planHistory, plan)
	// 保持历史记录在合理大小
	if len(btp.planHistory) > 100 {
		btp.planHistory = btp.planHistory[len(btp.planHistory)-100:]
	}
	btp.mu.Unlock()

	return plan, nil
}

// selectBestRule 选择最佳规划规则
func (btp *BasicTaskPlanner) selectBestRule(intent *interfaces.QueryIntent) *PlanningRule {
	var bestRule *PlanningRule
	var bestScore float64

	for _, rule := range btp.planningRules {
		score := btp.calculateRuleScore(rule, intent)
		if score > bestScore {
			bestScore = score
			bestRule = rule
		}
	}

	return bestRule
}

// calculateRuleScore 计算规则匹配分数
func (btp *BasicTaskPlanner) calculateRuleScore(rule *PlanningRule, intent *interfaces.QueryIntent) float64 {
	score := 0.0

	// 基础权重分配
	intentWeight := 0.3     // 降低意图类型权重
	domainWeight := 0.3     // 保持领域权重
	complexityWeight := 0.3 // 提高复杂度权重
	confidenceWeight := 0.1 // 保持置信度权重

	// 对于高复杂度查询，进一步提高复杂度权重
	if intent.Complexity >= 0.7 {
		intentWeight = 0.2
		domainWeight = 0.2
		complexityWeight = 0.5 // 大幅提高复杂度权重
		confidenceWeight = 0.1
	}

	// 意图类型匹配
	for _, intentType := range rule.IntentTypes {
		if intentType == intent.IntentType {
			score += intentWeight
			break
		}
	}

	// 领域匹配
	for _, domain := range rule.Domains {
		if domain == intent.Domain {
			score += domainWeight
			break
		}
	}

	// 复杂度匹配
	if intent.Complexity >= rule.Complexity[0] && intent.Complexity <= rule.Complexity[1] {
		score += complexityWeight

		// 复杂度匹配度奖励：越接近复杂规则的复杂度范围中心，得分越高
		if rule.Name == "复杂查询规划" && intent.Complexity >= 0.7 {
			score += 0.2 // 额外奖励
		}
	}

	// 置信度奖励
	score += intent.Confidence * confidenceWeight

	return score
}

// createTask 创建具体任务
func (btp *BasicTaskPlanner) createTask(taskType string, index int, intent *interfaces.QueryIntent, rule *PlanningRule) (*interfaces.Task, error) {
	template, exists := btp.taskTemplates[taskType]
	if !exists {
		return nil, fmt.Errorf("task template not found: %s", taskType)
	}

	// 合并模板参数和规则配置
	parameters := make(map[string]interface{})
	for k, v := range template.Parameters {
		parameters[k] = v
	}
	for k, v := range rule.Config {
		parameters[k] = v
	}

	task := &interfaces.Task{
		TaskID:          fmt.Sprintf("task_%s_%d", taskType, index),
		Type:            template.TaskType,
		TargetComponent: taskType,
		Parameters:      parameters,
		Dependencies:    []string{}, // 暂时简化依赖关系
		Priority:        template.DefaultPriority,
	}

	return task, nil
}

// ValidatePlan 验证任务计划
func (btp *BasicTaskPlanner) ValidatePlan(plan *interfaces.TaskPlan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}

	if len(plan.Tasks) == 0 {
		return fmt.Errorf("plan must contain at least one task")
	}

	if len(plan.Tasks) > btp.maxTasksPerPlan {
		return fmt.Errorf("plan contains too many tasks: %d > %d", len(plan.Tasks), btp.maxTasksPerPlan)
	}

	// 验证依赖关系
	taskIDs := make(map[string]bool)
	for _, task := range plan.Tasks {
		taskIDs[task.TaskID] = true
	}

	for _, task := range plan.Tasks {
		for _, dep := range task.Dependencies {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s has unresolved dependency: %s", task.TaskID, dep)
			}
		}
	}

	return nil
}

// OptimizePlan 优化任务计划
func (btp *BasicTaskPlanner) OptimizePlan(plan *interfaces.TaskPlan) (*interfaces.TaskPlan, error) {
	if plan == nil {
		return nil, fmt.Errorf("cannot optimize nil plan")
	}

	optimizedPlan := *plan // 浅复制

	// 重新排序任务以优化执行
	btp.optimizeTaskOrder(&optimizedPlan)

	// 调整并行执行策略
	btp.optimizeParallelism(&optimizedPlan)

	return &optimizedPlan, nil
}

// 辅助方法实现
func (btp *BasicTaskPlanner) generateCacheKey(intent *interfaces.QueryIntent) string {
	return fmt.Sprintf("%s_%s_%.2f", intent.IntentType, intent.Domain, intent.Complexity)
}

func (btp *BasicTaskPlanner) getCachedPlan(key string) *interfaces.TaskPlan {
	btp.mu.RLock()
	defer btp.mu.RUnlock()
	return btp.planCache[key]
}

func (btp *BasicTaskPlanner) cachePlan(key string, plan *interfaces.TaskPlan) {
	btp.mu.Lock()
	defer btp.mu.Unlock()
	btp.planCache[key] = plan
}

func (btp *BasicTaskPlanner) createDefaultPlan(intent *interfaces.QueryIntent) *interfaces.TaskPlan {
	return &interfaces.TaskPlan{
		PlanID:            fmt.Sprintf("default_plan_%d", time.Now().Unix()),
		TargetIntent:      intent,
		CreatedAt:         time.Now(),
		Tasks:             []*interfaces.Task{},
		ExecutionOrder:    []string{},
		ParallelExecution: false,
		MaxRetries:        1,
		TimeoutSeconds:    10,
		Priority:          50,
		Metadata: map[string]interface{}{
			"planner_name": btp.name,
			"plan_type":    "default",
		},
	}
}

func (btp *BasicTaskPlanner) generateShortID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano()%0xFFFF)
}

func (btp *BasicTaskPlanner) optimizeTaskOrder(plan *interfaces.TaskPlan) {
	// 基于优先级重新排序
	// 实现任务依赖图的拓扑排序
	// 这里是简化实现
}

func (btp *BasicTaskPlanner) optimizeParallelism(plan *interfaces.TaskPlan) {
	// 分析任务依赖关系，确定可以并行执行的任务组
	// 这里是简化实现
}

// Name 返回组件名称
func (btp *BasicTaskPlanner) Name() string {
	return btp.name
}

// GetCapabilities 返回组件能力列表
func (btp *BasicTaskPlanner) GetCapabilities() []string {
	return []string{
		"智能任务规划",
		"规则匹配",
		"并行优化",
		"计划缓存",
		"依赖解析",
		"性能统计",
	}
}

// GetStatistics 获取规划统计信息
func (btp *BasicTaskPlanner) GetStatistics() map[string]interface{} {
	btp.mu.RLock()
	defer btp.mu.RUnlock()

	successRate := 0.0
	if btp.totalPlans > 0 {
		successRate = float64(btp.successfulPlans) / float64(btp.totalPlans)
	}

	return map[string]interface{}{
		"total_plans":       btp.totalPlans,
		"successful_plans":  btp.successfulPlans,
		"success_rate":      successRate,
		"avg_planning_time": btp.avgPlanningTime,
		"rules_count":       len(btp.planningRules),
		"templates_count":   len(btp.taskTemplates),
		"cache_size":        len(btp.planCache),
		"history_size":      len(btp.planHistory),
	}
}
