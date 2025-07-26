package query_rewriter_v2

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// 🧩 乐高积木式组件注册管理器
// ============================================================================

// ComponentRegistry 组件注册中心 - 乐高积木式管理
type ComponentRegistry struct {
	mu         sync.RWMutex
	components map[string]QueryRewriterComponent

	// 分类注册表
	intentAnalyzers   map[string]QueryIntentAnalyzer
	strategySelectors map[string]StrategySelector
	contextLayers     map[string]ContextAwareLayer
	strategies        map[string]RewriteStrategy
	qualityEngines    map[string]QualityAssessmentEngine
	feedbackLearners  map[string]FeedbackLearner

	// 生命周期管理
	started    bool
	startOrder []string
	stopOrder  []string

	// 健康监控
	healthMonitor *ComponentHealthMonitor

	// 配置管理
	configs map[string]map[string]interface{}
}

// NewComponentRegistry 创建组件注册中心
func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{
		components:        make(map[string]QueryRewriterComponent),
		intentAnalyzers:   make(map[string]QueryIntentAnalyzer),
		strategySelectors: make(map[string]StrategySelector),
		contextLayers:     make(map[string]ContextAwareLayer),
		strategies:        make(map[string]RewriteStrategy),
		qualityEngines:    make(map[string]QualityAssessmentEngine),
		feedbackLearners:  make(map[string]FeedbackLearner),
		configs:           make(map[string]map[string]interface{}),
	}

	registry.healthMonitor = NewComponentHealthMonitor(registry)

	return registry
}

// ============================================================================
// 🔌 组件注册方法
// ============================================================================

// RegisterComponent 注册通用组件
func (cr *ComponentRegistry) RegisterComponent(component QueryRewriterComponent) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	name := component.Name()
	if _, exists := cr.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}

	// 注册到通用表
	cr.components[name] = component

	// 注册到分类表
	if err := cr.registerByType(component); err != nil {
		delete(cr.components, name)
		return fmt.Errorf("failed to register component by type: %w", err)
	}

	fmt.Printf("🔌 Component registered: %s (%s)\n", name, component.Version())
	return nil
}

// registerByType 按类型注册组件
func (cr *ComponentRegistry) registerByType(component QueryRewriterComponent) error {
	name := component.Name()

	switch comp := component.(type) {
	case QueryIntentAnalyzer:
		cr.intentAnalyzers[name] = comp
		fmt.Printf("  📊 Registered as QueryIntentAnalyzer\n")

	case StrategySelector:
		cr.strategySelectors[name] = comp
		fmt.Printf("  🎮 Registered as StrategySelector\n")

	case ContextAwareLayer:
		cr.contextLayers[name] = comp
		fmt.Printf("  🌐 Registered as ContextAwareLayer\n")

	case RewriteStrategy:
		cr.strategies[name] = comp
		fmt.Printf("  🔧 Registered as RewriteStrategy\n")

	case QualityAssessmentEngine:
		cr.qualityEngines[name] = comp
		fmt.Printf("  📊 Registered as QualityAssessmentEngine\n")

	case FeedbackLearner:
		cr.feedbackLearners[name] = comp
		fmt.Printf("  📈 Registered as FeedbackLearner\n")

	default:
		fmt.Printf("  ⚪ Registered as generic component\n")
	}

	return nil
}

// ============================================================================
// 🔍 组件查找方法
// ============================================================================

// GetComponent 获取通用组件
func (cr *ComponentRegistry) GetComponent(name string) (QueryRewriterComponent, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	component, exists := cr.components[name]
	if !exists {
		return nil, fmt.Errorf("component %s not found", name)
	}

	return component, nil
}

// GetIntentAnalyzer 获取意图分析器
func (cr *ComponentRegistry) GetIntentAnalyzer(name string) (QueryIntentAnalyzer, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	analyzer, exists := cr.intentAnalyzers[name]
	if !exists {
		return nil, fmt.Errorf("intent analyzer %s not found", name)
	}

	return analyzer, nil
}

// GetStrategySelector 获取策略选择器
func (cr *ComponentRegistry) GetStrategySelector(name string) (StrategySelector, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	selector, exists := cr.strategySelectors[name]
	if !exists {
		return nil, fmt.Errorf("strategy selector %s not found", name)
	}

	return selector, nil
}

// GetContextLayer 获取上下文感知层
func (cr *ComponentRegistry) GetContextLayer(name string) (ContextAwareLayer, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	layer, exists := cr.contextLayers[name]
	if !exists {
		return nil, fmt.Errorf("context layer %s not found", name)
	}

	return layer, nil
}

// GetStrategy 获取改写策略
func (cr *ComponentRegistry) GetStrategy(name string) (RewriteStrategy, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	strategy, exists := cr.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", name)
	}

	return strategy, nil
}

// GetQualityEngine 获取质量评估引擎
func (cr *ComponentRegistry) GetQualityEngine(name string) (QualityAssessmentEngine, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	engine, exists := cr.qualityEngines[name]
	if !exists {
		return nil, fmt.Errorf("quality engine %s not found", name)
	}

	return engine, nil
}

// GetFeedbackLearner 获取反馈学习器
func (cr *ComponentRegistry) GetFeedbackLearner(name string) (FeedbackLearner, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	learner, exists := cr.feedbackLearners[name]
	if !exists {
		return nil, fmt.Errorf("feedback learner %s not found", name)
	}

	return learner, nil
}

// ============================================================================
// 📋 组件列表方法
// ============================================================================

// ListAllComponents 列出所有组件
func (cr *ComponentRegistry) ListAllComponents() []ComponentInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	components := make([]ComponentInfo, 0, len(cr.components))

	for name, component := range cr.components {
		health := component.HealthCheck()

		info := ComponentInfo{
			Name:    name,
			Version: component.Version(),
			Type:    cr.getComponentType(component),
			Status:  health.Status,
			Health:  health,
		}

		components = append(components, info)
	}

	// 按名称排序
	sort.Slice(components, func(i, j int) bool {
		return components[i].Name < components[j].Name
	})

	return components
}

// ListByType 按类型列出组件
func (cr *ComponentRegistry) ListByType(componentType string) []string {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	var names []string

	switch componentType {
	case "intent_analyzer":
		for name := range cr.intentAnalyzers {
			names = append(names, name)
		}
	case "strategy_selector":
		for name := range cr.strategySelectors {
			names = append(names, name)
		}
	case "context_layer":
		for name := range cr.contextLayers {
			names = append(names, name)
		}
	case "strategy":
		for name := range cr.strategies {
			names = append(names, name)
		}
	case "quality_engine":
		for name := range cr.qualityEngines {
			names = append(names, name)
		}
	case "feedback_learner":
		for name := range cr.feedbackLearners {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

// ============================================================================
// 🔄 生命周期管理
// ============================================================================

// StartAll 启动所有组件
func (cr *ComponentRegistry) StartAll(ctx context.Context) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.started {
		return fmt.Errorf("components already started")
	}

	// 按依赖顺序启动
	startOrder := cr.calculateStartOrder()

	for _, name := range startOrder {
		component := cr.components[name]

		fmt.Printf("🚀 Starting component: %s\n", name)

		if err := component.Start(ctx); err != nil {
			fmt.Printf("❌ Failed to start component %s: %v\n", name, err)
			// 回滚已启动的组件
			cr.stopStartedComponents(startOrder[:cr.findIndex(startOrder, name)])
			return fmt.Errorf("failed to start component %s: %w", name, err)
		}

		fmt.Printf("✅ Component started: %s\n", name)
	}

	cr.started = true
	cr.startOrder = startOrder
	cr.stopOrder = cr.reverseOrder(startOrder)

	// 启动健康监控
	cr.healthMonitor.Start(ctx)

	fmt.Printf("🎉 All components started successfully\n")
	return nil
}

// StopAll 停止所有组件
func (cr *ComponentRegistry) StopAll(ctx context.Context) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.started {
		return nil
	}

	// 停止健康监控
	cr.healthMonitor.Stop()

	// 按逆序停止组件
	for _, name := range cr.stopOrder {
		component := cr.components[name]

		fmt.Printf("⏹️ Stopping component: %s\n", name)

		if err := component.Stop(ctx); err != nil {
			fmt.Printf("⚠️ Failed to stop component %s: %v\n", name, err)
			// 继续停止其他组件
		} else {
			fmt.Printf("✅ Component stopped: %s\n", name)
		}
	}

	cr.started = false
	fmt.Printf("🏁 All components stopped\n")
	return nil
}

// ============================================================================
// 🔧 配置管理
// ============================================================================

// UpdateComponentConfig 更新组件配置
func (cr *ComponentRegistry) UpdateComponentConfig(name string, config map[string]interface{}) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	component, exists := cr.components[name]
	if !exists {
		return fmt.Errorf("component %s not found", name)
	}

	// 更新配置
	if err := component.Initialize(config); err != nil {
		return fmt.Errorf("failed to update component config: %w", err)
	}

	// 保存配置
	cr.configs[name] = make(map[string]interface{})
	for k, v := range config {
		cr.configs[name][k] = v
	}

	fmt.Printf("🔧 Component config updated: %s\n", name)
	return nil
}

// GetComponentConfig 获取组件配置
func (cr *ComponentRegistry) GetComponentConfig(name string) (map[string]interface{}, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	config, exists := cr.configs[name]
	if !exists {
		return nil, fmt.Errorf("component %s config not found", name)
	}

	// 返回配置副本
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}

	return result, nil
}

// ============================================================================
// 🏥 健康监控
// ============================================================================

// ComponentHealthMonitor 组件健康监控器
type ComponentHealthMonitor struct {
	registry *ComponentRegistry
	ticker   *time.Ticker
	stopChan chan struct{}
	running  bool
}

// NewComponentHealthMonitor 创建健康监控器
func NewComponentHealthMonitor(registry *ComponentRegistry) *ComponentHealthMonitor {
	return &ComponentHealthMonitor{
		registry: registry,
		stopChan: make(chan struct{}),
	}
}

// Start 启动健康监控
func (chm *ComponentHealthMonitor) Start(ctx context.Context) {
	if chm.running {
		return
	}

	chm.ticker = time.NewTicker(30 * time.Second) // 30秒检查一次
	chm.running = true

	go chm.monitor(ctx)

	fmt.Printf("🏥 Component health monitor started\n")
}

// Stop 停止健康监控
func (chm *ComponentHealthMonitor) Stop() {
	if !chm.running {
		return
	}

	chm.running = false
	close(chm.stopChan)

	if chm.ticker != nil {
		chm.ticker.Stop()
	}

	fmt.Printf("🏥 Component health monitor stopped\n")
}

// monitor 执行健康监控
func (chm *ComponentHealthMonitor) monitor(ctx context.Context) {
	for {
		select {
		case <-chm.ticker.C:
			chm.checkAllComponents()
		case <-chm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAllComponents 检查所有组件健康状态
func (chm *ComponentHealthMonitor) checkAllComponents() {
	chm.registry.mu.RLock()
	components := make(map[string]QueryRewriterComponent)
	for name, component := range chm.registry.components {
		components[name] = component
	}
	chm.registry.mu.RUnlock()

	unhealthyCount := 0

	for name, component := range components {
		health := component.HealthCheck()

		switch health.Status {
		case "unhealthy":
			fmt.Printf("🚨 Component unhealthy: %s - %s\n", name, health.Message)
			unhealthyCount++
		case "degraded":
			fmt.Printf("⚠️ Component degraded: %s - %s\n", name, health.Message)
		}
	}

	if unhealthyCount > 0 {
		fmt.Printf("🚨 Health check summary: %d unhealthy components\n", unhealthyCount)
	}
}

// ============================================================================
// 🛠️ 辅助方法
// ============================================================================

// ComponentInfo 组件信息
type ComponentInfo struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Health  ComponentHealth `json:"health"`
}

// getComponentType 获取组件类型
func (cr *ComponentRegistry) getComponentType(component QueryRewriterComponent) string {
	switch component.(type) {
	case QueryIntentAnalyzer:
		return "intent_analyzer"
	case StrategySelector:
		return "strategy_selector"
	case ContextAwareLayer:
		return "context_layer"
	case RewriteStrategy:
		return "strategy"
	case QualityAssessmentEngine:
		return "quality_engine"
	case FeedbackLearner:
		return "feedback_learner"
	default:
		return "generic"
	}
}

// calculateStartOrder 计算启动顺序
func (cr *ComponentRegistry) calculateStartOrder() []string {
	// 简化实现：按类型优先级排序
	order := make([]string, 0, len(cr.components))

	// 优先级：上下文感知层 -> 意图分析器 -> 策略选择器 -> 策略 -> 质量引擎 -> 反馈学习器
	priorities := map[string]int{
		"context_layer":     1,
		"intent_analyzer":   2,
		"strategy_selector": 3,
		"strategy":          4,
		"quality_engine":    5,
		"feedback_learner":  6,
		"generic":           7,
	}

	components := make([]struct {
		name     string
		priority int
	}, 0, len(cr.components))

	for name, component := range cr.components {
		componentType := cr.getComponentType(component)
		priority := priorities[componentType]

		components = append(components, struct {
			name     string
			priority int
		}{name, priority})
	}

	// 按优先级排序
	sort.Slice(components, func(i, j int) bool {
		return components[i].priority < components[j].priority
	})

	for _, comp := range components {
		order = append(order, comp.name)
	}

	return order
}

// stopStartedComponents 停止已启动的组件
func (cr *ComponentRegistry) stopStartedComponents(names []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := len(names) - 1; i >= 0; i-- {
		if component, exists := cr.components[names[i]]; exists {
			component.Stop(ctx)
		}
	}
}

// reverseOrder 反转顺序
func (cr *ComponentRegistry) reverseOrder(order []string) []string {
	reversed := make([]string, len(order))
	for i, name := range order {
		reversed[len(order)-1-i] = name
	}
	return reversed
}

// findIndex 查找索引
func (cr *ComponentRegistry) findIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
