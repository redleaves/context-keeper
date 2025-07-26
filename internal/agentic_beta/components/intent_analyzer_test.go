package components

import (
	"context"
	"testing"
	"time"
)

func TestBasicQueryIntentAnalyzer(t *testing.T) {
	// 创建分析器实例
	analyzer := NewBasicQueryIntentAnalyzer()

	// 验证基础信息
	if analyzer.Name() != "BasicQueryIntentAnalyzer" {
		t.Errorf("Expected name BasicQueryIntentAnalyzer, got %s", analyzer.Name())
	}

	if analyzer.Version() != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", analyzer.Version())
	}

	if !analyzer.IsEnabled() {
		t.Error("Expected analyzer to be enabled")
	}

	// 健康检查
	if err := analyzer.HealthCheck(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestAnalyzeIntent_DebugQuery(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 测试调试类查询
	query := "这个Go程序报错了，怎么debug？"

	intent, err := analyzer.AnalyzeIntent(ctx, query)
	if err != nil {
		t.Fatalf("Failed to analyze intent: %v", err)
	}

	// 验证结果
	if intent.OriginalQuery != query {
		t.Errorf("Expected original query %s, got %s", query, intent.OriginalQuery)
	}

	if intent.IntentType != "debugging" {
		t.Errorf("Expected intent type debugging, got %s", intent.IntentType)
	}

	if intent.Domain != "programming" {
		t.Errorf("Expected domain programming, got %s", intent.Domain)
	}

	// 验证技术栈识别
	hasGo := false
	for _, tech := range intent.TechStack {
		if tech == "go" {
			hasGo = true
			break
		}
	}
	if !hasGo {
		t.Error("Expected to detect 'go' in tech stack")
	}

	// 验证置信度
	if intent.Confidence <= 0 || intent.Confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", intent.Confidence)
	}

	// 验证元数据
	if intent.Metadata["analyzer_name"] != "BasicQueryIntentAnalyzer" {
		t.Error("Expected analyzer_name in metadata")
	}
}

func TestAnalyzeIntent_TechnicalQuery(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 测试技术类查询
	query := "如何用React实现一个性能优化的组件？"

	intent, err := analyzer.AnalyzeIntent(ctx, query)
	if err != nil {
		t.Fatalf("Failed to analyze intent: %v", err)
	}

	// 验证意图类型
	if intent.IntentType != "procedural" {
		t.Errorf("Expected intent type procedural, got %s", intent.IntentType)
	}

	// 验证领域识别
	if intent.Domain != "frontend" {
		t.Errorf("Expected domain frontend, got %s", intent.Domain)
	}

	// 验证技术栈
	hasReact := false
	for _, tech := range intent.TechStack {
		if tech == "react" {
			hasReact = true
			break
		}
	}
	if !hasReact {
		t.Error("Expected to detect 'react' in tech stack")
	}

	// 验证关键词提取
	if len(intent.Keywords) == 0 {
		t.Error("Expected to extract keywords")
	}
}

func TestAnalyzeIntent_DatabaseQuery(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 测试数据库相关查询
	query := "MySQL数据库索引优化的最佳实践是什么？"

	intent, err := analyzer.AnalyzeIntent(ctx, query)
	if err != nil {
		t.Fatalf("Failed to analyze intent: %v", err)
	}

	// 验证领域识别
	if intent.Domain != "database" {
		t.Errorf("Expected domain database, got %s", intent.Domain)
	}

	// 验证意图类型
	if intent.IntentType != "conceptual" {
		t.Errorf("Expected intent type conceptual, got %s", intent.IntentType)
	}

	// 验证技术栈
	hasMySQL := false
	for _, tech := range intent.TechStack {
		if tech == "mysql" {
			hasMySQL = true
			break
		}
	}
	if !hasMySQL {
		t.Error("Expected to detect 'mysql' in tech stack")
	}
}

func TestAnalyzeIntent_ComplexityAssessment(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	testCases := []struct {
		query              string
		expectedComplexity float64
		name               string
	}{
		{
			query:              "Hi",
			expectedComplexity: 0.2, // 短查询，低复杂度
			name:               "Simple query",
		},
		{
			query:              "如何设计一个高可用的分布式微服务架构，包含负载均衡、服务发现、熔断器和监控系统？",
			expectedComplexity: 0.8, // 长查询，多技术术语，高复杂度
			name:               "Complex query",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent, err := analyzer.AnalyzeIntent(ctx, tc.query)
			if err != nil {
				t.Fatalf("Failed to analyze intent: %v", err)
			}

			// 允许一定的误差范围
			if intent.Complexity < tc.expectedComplexity-0.2 || intent.Complexity > tc.expectedComplexity+0.2 {
				t.Errorf("Expected complexity around %f, got %f", tc.expectedComplexity, intent.Complexity)
			}
		})
	}
}

func TestConfiguration(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()

	// 测试配置更新
	config := map[string]interface{}{
		"enabled": false,
		"name":    "CustomAnalyzer",
	}

	err := analyzer.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}

	// 验证配置已更新
	if analyzer.IsEnabled() {
		t.Error("Expected analyzer to be disabled")
	}

	if analyzer.Name() != "CustomAnalyzer" {
		t.Errorf("Expected name CustomAnalyzer, got %s", analyzer.Name())
	}

	// 验证禁用状态下的健康检查
	if err := analyzer.HealthCheck(); err == nil {
		t.Error("Expected health check to fail when disabled")
	}
}

func TestEmptyQuery(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 测试空查询
	_, err := analyzer.AnalyzeIntent(ctx, "")
	if err == nil {
		t.Error("Expected error for empty query")
	}

	// 测试只有空格的查询
	_, err = analyzer.AnalyzeIntent(ctx, "   ")
	if err == nil {
		t.Error("Expected error for whitespace-only query")
	}
}

func TestStatistics(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 初始统计信息
	stats := analyzer.GetStats()
	if stats.TotalAnalyzed != 0 {
		t.Errorf("Expected 0 analyzed queries initially, got %d", stats.TotalAnalyzed)
	}

	// 分析几个查询
	queries := []string{
		"Go语言的错误处理",
		"React组件优化",
		"MySQL索引设计",
	}

	for _, query := range queries {
		_, err := analyzer.AnalyzeIntent(ctx, query)
		if err != nil {
			t.Fatalf("Failed to analyze query '%s': %v", query, err)
		}
	}

	// 验证统计信息更新
	stats = analyzer.GetStats()
	if stats.TotalAnalyzed != len(queries) {
		t.Errorf("Expected %d analyzed queries, got %d", len(queries), stats.TotalAnalyzed)
	}

	// 验证意图分布
	if len(stats.IntentDistribution) == 0 {
		t.Error("Expected intent distribution to be populated")
	}

	// 验证平均置信度
	if stats.AverageConfidence <= 0 || stats.AverageConfidence > 1 {
		t.Errorf("Expected average confidence between 0 and 1, got %f", stats.AverageConfidence)
	}
}

func TestPerformance(t *testing.T) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()

	// 性能测试
	query := "如何优化大规模分布式系统的数据库性能？"

	start := time.Now()
	_, err := analyzer.AnalyzeIntent(ctx, query)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to analyze intent: %v", err)
	}

	// 验证处理时间合理（应该在100ms以内）
	if duration > 100*time.Millisecond {
		t.Errorf("Analysis took too long: %v", duration)
	}
}

// 基准测试
func BenchmarkAnalyzeIntent(b *testing.B) {
	analyzer := NewBasicQueryIntentAnalyzer()
	ctx := context.Background()
	query := "如何使用Go语言实现高性能的Web API服务？"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeIntent(ctx, query)
		if err != nil {
			b.Fatalf("Failed to analyze intent: %v", err)
		}
	}
}
