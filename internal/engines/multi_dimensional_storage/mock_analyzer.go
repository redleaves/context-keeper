package multi_dimensional_storage

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// MockLLMAnalyzer 模拟LLM分析器实现
type MockLLMAnalyzer struct {
	config *MultiDimensionalStorageConfig
}

// NewMockLLMAnalyzer 创建模拟LLM分析器
func NewMockLLMAnalyzer(config *MultiDimensionalStorageConfig) (*MockLLMAnalyzer, error) {
	analyzer := &MockLLMAnalyzer{
		config: config,
	}

	log.Printf("✅ 模拟LLM分析器初始化完成")
	return analyzer, nil
}

// Analyze 分析请求，返回结构化数据
func (a *MockLLMAnalyzer) Analyze(request *StorageRequest) (*LLMAnalysisResult, error) {
	startTime := time.Now()

	log.Printf("🔍 开始模拟LLM分析 - 用户: %s, 会话: %s", request.UserID, request.SessionID)
	log.Printf("📝 查询内容: %s", request.Query[:min(100, len(request.Query))])

	// 模拟分析延迟
	time.Sleep(100 * time.Millisecond)

	// 基于关键词生成模拟分析结果
	result := a.generateMockAnalysis(request)

	analysisTime := time.Since(startTime)
	log.Printf("✅ 模拟LLM分析完成 - 耗时: %v", analysisTime)
	log.Printf("📊 分析结果: 时间线优先级=%.2f, 知识图谱优先级=%.2f, 向量优先级=%.2f",
		result.StorageRecommendation.TimelinePriority,
		result.StorageRecommendation.KnowledgePriority,
		result.StorageRecommendation.VectorPriority)

	return result, nil
}

// IsAvailable 检查LLM是否可用
func (a *MockLLMAnalyzer) IsAvailable() bool {
	return true
}

// generateMockAnalysis 生成模拟分析结果
func (a *MockLLMAnalyzer) generateMockAnalysis(request *StorageRequest) *LLMAnalysisResult {
	query := strings.ToLower(request.Query)

	// 1. 生成时间线数据
	timelineData := &TimelineData{
		Title:           a.extractTitle(request.Query),
		Content:         request.Query,
		EventType:       a.detectEventType(query),
		Keywords:        a.extractKeywords(query),
		ImportanceScore: a.calculateImportance(query),
		TechStack:       a.extractTechStack(query),
		ProjectContext:  a.extractProjectContext(request.Context),
	}

	// 2. 生成知识图谱数据
	knowledgeData := &KnowledgeGraphData{
		Concepts:      a.extractConcepts(query),
		Relationships: a.extractRelationships(query),
	}

	// 3. 生成向量数据
	vectorData := &VectorData{
		Content:        a.cleanContent(request.Query),
		SemanticTags:   a.generateSemanticTags(query),
		ContextSummary: a.generateContextSummary(request.Query, request.Context),
		RelevanceScore: 0.9,
	}

	// 4. 生成存储推荐
	recommendation := &StorageRecommendation{
		TimelinePriority:  a.calculateTimelinePriority(query),
		KnowledgePriority: a.calculateKnowledgePriority(query),
		VectorPriority:    a.calculateVectorPriority(query),
		Reasoning:         a.generateReasoning(query),
	}

	return &LLMAnalysisResult{
		TimelineData:          timelineData,
		KnowledgeGraphData:    knowledgeData,
		VectorData:            vectorData,
		StorageRecommendation: recommendation,
	}
}

// extractTitle 提取标题
func (a *MockLLMAnalyzer) extractTitle(query string) string {
	if len(query) <= 50 {
		return query
	}
	return query[:47] + "..."
}

// detectEventType 检测事件类型
func (a *MockLLMAnalyzer) detectEventType(query string) string {
	if strings.Contains(query, "问题") || strings.Contains(query, "解决") {
		return "problem_solve"
	}
	if strings.Contains(query, "学习") || strings.Contains(query, "了解") {
		return "knowledge_share"
	}
	if strings.Contains(query, "选择") || strings.Contains(query, "决策") {
		return "decision"
	}
	if strings.Contains(query, "讨论") || strings.Contains(query, "分析") {
		return "discussion"
	}
	return "discussion"
}

// extractKeywords 提取关键词
func (a *MockLLMAnalyzer) extractKeywords(query string) []string {
	keywords := []string{}

	techKeywords := []string{"Redis", "TimescaleDB", "Neo4j", "Milvus", "Pinecone", "Weaviate",
		"QPS", "性能", "优化", "集群", "缓存", "数据库", "向量", "检索"}

	for _, keyword := range techKeywords {
		if strings.Contains(query, keyword) {
			keywords = append(keywords, keyword)
		}
	}

	if len(keywords) == 0 {
		keywords = []string{"技术讨论"}
	}

	return keywords
}

// calculateImportance 计算重要性
func (a *MockLLMAnalyzer) calculateImportance(query string) int {
	score := 5 // 基础分数

	if strings.Contains(query, "问题") || strings.Contains(query, "故障") {
		score += 3
	}
	if strings.Contains(query, "性能") || strings.Contains(query, "优化") {
		score += 2
	}
	if strings.Contains(query, "决策") || strings.Contains(query, "选择") {
		score += 2
	}

	if score > 10 {
		score = 10
	}

	return score
}

// extractTechStack 提取技术栈
func (a *MockLLMAnalyzer) extractTechStack(query string) []string {
	techStack := []string{}

	technologies := map[string]string{
		"redis":       "Redis",
		"timescaledb": "TimescaleDB",
		"neo4j":       "Neo4j",
		"milvus":      "Milvus",
		"pinecone":    "Pinecone",
		"weaviate":    "Weaviate",
	}

	for key, tech := range technologies {
		if strings.Contains(query, key) {
			techStack = append(techStack, tech)
		}
	}

	return techStack
}

// extractProjectContext 提取项目上下文
func (a *MockLLMAnalyzer) extractProjectContext(context string) string {
	if context == "" {
		return "技术研究"
	}
	if len(context) > 100 {
		return context[:97] + "..."
	}
	return context
}

// extractConcepts 提取概念
func (a *MockLLMAnalyzer) extractConcepts(query string) []Concept {
	concepts := []Concept{}

	conceptMap := map[string]string{
		"redis":       "缓存数据库",
		"timescaledb": "时间序列数据库",
		"neo4j":       "图数据库",
		"milvus":      "向量数据库",
		"性能优化":        "技术方法",
		"集群":          "架构模式",
	}

	for keyword, conceptType := range conceptMap {
		if strings.Contains(query, keyword) {
			concepts = append(concepts, Concept{
				Name:       keyword,
				Type:       conceptType,
				Properties: map[string]interface{}{"source": "mock_analysis"},
				Importance: 0.8,
			})
		}
	}

	return concepts
}

// extractRelationships 提取关系
func (a *MockLLMAnalyzer) extractRelationships(query string) []Relationship {
	relationships := []Relationship{}

	if strings.Contains(query, "redis") && strings.Contains(query, "集群") {
		relationships = append(relationships, Relationship{
			Source:      "Redis",
			Target:      "集群",
			Type:        "USED_WITH",
			Strength:    0.9,
			Description: "Redis使用集群架构",
		})
	}

	if strings.Contains(query, "性能") && strings.Contains(query, "优化") {
		relationships = append(relationships, Relationship{
			Source:      "性能优化",
			Target:      "系统性能",
			Type:        "IMPROVES",
			Strength:    0.8,
			Description: "性能优化提升系统性能",
		})
	}

	return relationships
}

// cleanContent 清理内容
func (a *MockLLMAnalyzer) cleanContent(query string) string {
	// 简单的内容清理
	return strings.TrimSpace(query)
}

// generateSemanticTags 生成语义标签
func (a *MockLLMAnalyzer) generateSemanticTags(query string) []string {
	tags := []string{}

	if strings.Contains(query, "问题") || strings.Contains(query, "故障") {
		tags = append(tags, "问题解决")
	}
	if strings.Contains(query, "学习") {
		tags = append(tags, "学习记录")
	}
	if strings.Contains(query, "决策") || strings.Contains(query, "选择") {
		tags = append(tags, "技术决策")
	}
	if strings.Contains(query, "性能") || strings.Contains(query, "优化") {
		tags = append(tags, "性能优化")
	}

	if len(tags) == 0 {
		tags = []string{"技术讨论"}
	}

	return tags
}

// generateContextSummary 生成上下文摘要
func (a *MockLLMAnalyzer) generateContextSummary(query, context string) string {
	if context == "" {
		return fmt.Sprintf("技术讨论：%s", a.extractTitle(query))
	}
	return fmt.Sprintf("上下文：%s", context[:min(100, len(context))])
}

// calculateTimelinePriority 计算时间线优先级
func (a *MockLLMAnalyzer) calculateTimelinePriority(query string) float64 {
	priority := 0.5

	if strings.Contains(query, "问题") || strings.Contains(query, "解决") {
		priority += 0.3
	}
	if strings.Contains(query, "学习") || strings.Contains(query, "记录") {
		priority += 0.2
	}

	if priority > 1.0 {
		priority = 1.0
	}

	return priority
}

// calculateKnowledgePriority 计算知识图谱优先级
func (a *MockLLMAnalyzer) calculateKnowledgePriority(query string) float64 {
	priority := 0.4

	techTerms := []string{"redis", "timescaledb", "neo4j", "milvus", "技术", "架构"}
	for _, term := range techTerms {
		if strings.Contains(query, term) {
			priority += 0.15
		}
	}

	if priority > 1.0 {
		priority = 1.0
	}

	return priority
}

// calculateVectorPriority 计算向量优先级
func (a *MockLLMAnalyzer) calculateVectorPriority(query string) float64 {
	// 向量存储适合所有类型的内容
	return 0.8
}

// generateReasoning 生成推荐理由
func (a *MockLLMAnalyzer) generateReasoning(query string) string {
	if strings.Contains(query, "问题") {
		return "包含问题解决过程，适合时间线存储"
	}
	if strings.Contains(query, "技术") {
		return "包含技术概念，适合知识图谱存储"
	}
	return "通用技术内容，适合向量存储"
}
