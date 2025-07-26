package components

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/interfaces"
)

// ============================================================================
// 🔍 查询意图分析器 - 第一阶段基础实现
// ============================================================================

// BasicQueryIntentAnalyzer 基础查询意图分析器
// 设计原则：简单、稳定、可验证、可扩展
type BasicQueryIntentAnalyzer struct {
	// 基础配置
	name    string
	version string
	enabled bool

	// 分析规则
	intentRules    map[string]*IntentRule
	domainPatterns map[string]*regexp.Regexp
	keywordGroups  map[string][]string

	// 统计信息
	stats *AnalyzerStats
}

// IntentRule 意图识别规则
type IntentRule struct {
	IntentType string           `json:"intent_type"`
	Patterns   []*regexp.Regexp `json:"patterns"`
	Keywords   []string         `json:"keywords"`
	Confidence float64          `json:"confidence"`
	Priority   int              `json:"priority"`
}

// AnalyzerStats 分析器统计
type AnalyzerStats struct {
	TotalAnalyzed      int            `json:"total_analyzed"`
	IntentDistribution map[string]int `json:"intent_distribution"`
	AverageConfidence  float64        `json:"average_confidence"`
	ProcessingTime     time.Duration  `json:"processing_time"`
}

// NewBasicQueryIntentAnalyzer 创建基础意图分析器
func NewBasicQueryIntentAnalyzer() *BasicQueryIntentAnalyzer {
	analyzer := &BasicQueryIntentAnalyzer{
		name:           "BasicQueryIntentAnalyzer",
		version:        "v1.0.0",
		enabled:        true,
		intentRules:    make(map[string]*IntentRule),
		domainPatterns: make(map[string]*regexp.Regexp),
		keywordGroups:  make(map[string][]string),
		stats: &AnalyzerStats{
			IntentDistribution: make(map[string]int),
		},
	}

	// 初始化默认规则
	analyzer.initializeDefaultRules()

	return analyzer
}

// ============================================================================
// 🎯 实现 QueryIntentAnalyzer 接口
// ============================================================================

// AnalyzeIntent 分析查询意图 - 核心方法
func (bia *BasicQueryIntentAnalyzer) AnalyzeIntent(ctx context.Context, query string) (*interfaces.QueryIntent, error) {
	if !bia.enabled {
		return nil, fmt.Errorf("分析器已禁用")
	}

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("查询不能为空")
	}

	startTime := time.Now()

	// 🔍 第一步：基础意图分类
	intentType := bia.classifyIntent(query)

	// 🏗️ 第二步：领域识别
	domain := bia.identifyDomain(query)

	// 📊 第三步：复杂度评估
	complexity := bia.assessComplexity(query)

	// 🔤 第四步：关键词提取
	keywords := bia.extractKeywords(query)

	// 🏷️ 第五步：实体识别
	entities := bia.extractEntities(query)

	// 🛠️ 第六步：技术栈识别
	techStack := bia.identifyTechStack(query)

	// 🎯 第七步：计算置信度
	confidence := bia.calculateConfidence(intentType, domain, keywords)

	// 创建结果
	intent := &interfaces.QueryIntent{
		OriginalQuery: query,
		Timestamp:     time.Now(),
		IntentType:    intentType,
		Domain:        domain,
		Complexity:    complexity,
		Keywords:      keywords,
		Entities:      entities,
		TechStack:     techStack,
		Confidence:    confidence,
		Metadata: map[string]interface{}{
			"analyzer_name":    bia.name,
			"analyzer_version": bia.version,
			"processing_time":  time.Since(startTime),
			"analysis_method":  "rule_based",
		},
	}

	// 更新统计信息
	bia.updateStats(intentType, confidence, time.Since(startTime))

	return intent, nil
}

// Name 组件名称
func (bia *BasicQueryIntentAnalyzer) Name() string {
	return bia.name
}

// Version 组件版本
func (bia *BasicQueryIntentAnalyzer) Version() string {
	return bia.version
}

// IsEnabled 是否启用
func (bia *BasicQueryIntentAnalyzer) IsEnabled() bool {
	return bia.enabled
}

// Configure 配置管理
func (bia *BasicQueryIntentAnalyzer) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		bia.enabled = enabled
	}

	if name, ok := config["name"].(string); ok && name != "" {
		bia.name = name
	}

	// 可以扩展更多配置项
	return nil
}

// HealthCheck 健康检查
func (bia *BasicQueryIntentAnalyzer) HealthCheck() error {
	if !bia.enabled {
		return fmt.Errorf("分析器已禁用")
	}

	if len(bia.intentRules) == 0 {
		return fmt.Errorf("意图规则未初始化")
	}

	return nil
}

// ============================================================================
// 🔧 核心分析方法
// ============================================================================

// classifyIntent 分类查询意图
func (bia *BasicQueryIntentAnalyzer) classifyIntent(query string) string {
	query = strings.ToLower(query)

	// 🐛 调试类意图 (最高优先级)
	debugPatterns := []string{
		"调试", "debug", "错误", "error", "bug", "问题", "失败", "不工作",
		"修复", "fix", "解决", "怎么回事", "为什么不", "出了什么问题", "报错",
	}
	for _, pattern := range debugPatterns {
		if strings.Contains(query, pattern) {
			return "debugging"
		}
	}

	// 📋 过程类意图 (第二优先级 - 明确的操作指向)
	proceduralPatterns := []string{
		"怎么", "如何", "步骤", "流程", "教程", "指南", "方法",
		"操作", "使用", "设置", "建立",
	}
	for _, pattern := range proceduralPatterns {
		if strings.Contains(query, pattern) {
			return "procedural"
		}
	}

	// 📚 概念类意图 (第三优先级 - 理论和概念)
	conceptualPatterns := []string{
		"什么是", "概念", "原理", "定义", "解释", "理解", "学习",
		"区别", "比较", "优缺点", "特点", "作用", "最佳实践",
	}
	for _, pattern := range conceptualPatterns {
		if strings.Contains(query, pattern) {
			return "conceptual"
		}
	}

	// 🛠️ 技术类意图 (第四优先级 - 具体技术实现)
	technicalPatterns := []string{
		"实现", "代码", "函数", "算法", "性能", "优化",
		"配置", "部署", "安装", "搭建", "集成", "开发",
	}
	for _, pattern := range technicalPatterns {
		if strings.Contains(query, pattern) {
			return "technical"
		}
	}

	// 默认为技术类
	return "technical"
}

// identifyDomain 识别技术领域
func (bia *BasicQueryIntentAnalyzer) identifyDomain(query string) string {
	query = strings.ToLower(query)

	// 🏗️ 架构领域 (最高优先级 - 系统层面设计)
	architectureKeywords := []string{"架构", "architecture", "设计模式", "系统设计", "分布式", "高可用", "负载均衡", "服务发现", "熔断器", "监控系统"}
	if bia.containsAny(query, architectureKeywords) {
		return "architecture"
	}

	// 🗄️ 数据库领域
	if bia.containsAny(query, []string{"数据库", "database", "sql", "mysql", "postgresql", "mongodb", "redis"}) {
		return "database"
	}

	// 🌐 前端领域
	if bia.containsAny(query, []string{"前端", "frontend", "react", "vue", "javascript", "css", "html", "ui"}) {
		return "frontend"
	}

	// 🖥️ 后端领域 (移除"微服务"避免与架构冲突)
	if bia.containsAny(query, []string{"后端", "backend", "api", "服务器", "server", "service"}) {
		return "backend"
	}

	// 🔧 DevOps领域
	if bia.containsAny(query, []string{"部署", "docker", "kubernetes", "ci/cd", "运维", "devops"}) {
		return "devops"
	}

	// 默认为编程领域
	return "programming"
}

// assessComplexity 评估查询复杂度
func (bia *BasicQueryIntentAnalyzer) assessComplexity(query string) float64 {
	complexity := 0.0

	// 基于查询长度
	if len(query) > 100 {
		complexity += 0.3
	} else if len(query) > 50 {
		complexity += 0.2
	} else {
		complexity += 0.1
	}

	// 基于技术术语数量
	techTermCount := bia.countTechnicalTerms(query)
	complexity += float64(techTermCount) * 0.1

	// 基于问题类型
	if strings.Contains(strings.ToLower(query), "架构") {
		complexity += 0.3
	}

	// 限制在0.0-1.0范围内
	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

// extractKeywords 提取详细关键词信息 (融入v2设计)
func (bia *BasicQueryIntentAnalyzer) extractKeywords(query string) []interfaces.KeywordInfo {
	// 简单的关键词提取：去除停用词，保留有意义的词汇
	stopWords := map[string]bool{
		"的": true, "了": true, "和": true, "是": true, "在": true,
		"有": true, "不": true, "为": true, "这": true, "个": true,
		"我": true, "你": true, "他": true, "它": true, "们": true,
		"怎么": true, "如何": true, "什么": true, "哪个": true,
	}

	// 技术术语字典（v2优势：分类和权重）
	techTerms := map[string]struct {
		category string
		weight   float64
	}{
		"数据库": {"technical", 0.9}, "SQL": {"technical", 0.8}, "API": {"technical", 0.8},
		"Redis": {"technical", 0.8}, "MongoDB": {"technical", 0.8}, "MySQL": {"technical", 0.8},
		"架构": {"domain", 0.9}, "设计": {"domain", 0.7}, "性能": {"domain", 0.8},
		"优化": {"action", 0.8}, "调试": {"action", 0.9}, "修复": {"action", 0.8},
		"服务": {"object", 0.7}, "系统": {"object", 0.7}, "项目": {"object", 0.6},
	}

	// 分词（简化版）
	words := strings.Fields(query)
	var keywords []interfaces.KeywordInfo

	for i, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 1 && !stopWords[word] {
			keywordInfo := interfaces.KeywordInfo{
				Term:     word,
				Weight:   0.5,             // 默认权重
				Category: "general",       // 默认分类
				Source:   "text_analysis", // 提取来源
			}

			// 检查是否是技术术语 (v2优势)
			if termInfo, exists := techTerms[word]; exists {
				keywordInfo.Weight = termInfo.weight
				keywordInfo.Category = termInfo.category
				keywordInfo.Source = "tech_dictionary"
			}

			// 位置权重调整 (v2优势)
			if i < len(words)/3 {
				keywordInfo.Weight += 0.1 // 前部权重加分
			}

			keywords = append(keywords, keywordInfo)
		}
	}

	return keywords
}

// extractEntities 提取详细实体信息 (融入v2设计)
func (bia *BasicQueryIntentAnalyzer) extractEntities(query string) []interfaces.EntityInfo {
	var entities []interfaces.EntityInfo

	// 技术实体字典 (v2优势：实体类型和评分)
	techEntities := map[string]struct {
		entityType string
		score      float64
	}{
		"Go": {"TECH", 0.9}, "Python": {"TECH", 0.9}, "Java": {"TECH", 0.9},
		"React": {"TECH", 0.8}, "Vue": {"TECH", 0.8}, "Angular": {"TECH", 0.8},
		"Docker": {"TOOL", 0.9}, "Kubernetes": {"TOOL", 0.9}, "Redis": {"TOOL", 0.8},
		"MySQL": {"TOOL", 0.8}, "MongoDB": {"TOOL", 0.8}, "PostgreSQL": {"TOOL", 0.8},
		"GitHub": {"ORG", 0.7}, "Google": {"ORG", 0.7}, "Microsoft": {"ORG", 0.7},
	}

	// 常见技术实体模式 (v2优势：位置信息)
	patterns := []struct {
		pattern    string
		entityType string
		score      float64
	}{
		{`[A-Z][a-z]+`, "TECH", 0.6}, // 首字母大写的词
		{`[A-Z]{2,}`, "TECH", 0.7},   // 全大写缩写
	}

	// 先检查字典中的已知实体
	words := strings.Fields(query)
	for _, word := range words {
		if entityInfo, exists := techEntities[word]; exists {
			entities = append(entities, interfaces.EntityInfo{
				Text:  word,
				Type:  entityInfo.entityType,
				Score: entityInfo.score,
				Position: [2]int{
					strings.Index(query, word),
					strings.Index(query, word) + len(word),
				},
			})
		}
	}

	// 然后使用模式匹配识别其他实体
	for _, patternInfo := range patterns {
		re := regexp.MustCompile(patternInfo.pattern)
		matches := re.FindAllString(query, -1)
		for _, match := range matches {
			// 避免重复添加已知实体
			found := false
			for _, existing := range entities {
				if existing.Text == match {
					found = true
					break
				}
			}
			if !found {
				entities = append(entities, interfaces.EntityInfo{
					Text:  match,
					Type:  patternInfo.entityType,
					Score: patternInfo.score,
					Position: [2]int{
						strings.Index(query, match),
						strings.Index(query, match) + len(match),
					},
				})
			}
		}
	}

	return entities
}

// identifyTechStack 识别技术栈
func (bia *BasicQueryIntentAnalyzer) identifyTechStack(query string) []string {
	query = strings.ToLower(query)
	techStack := []string{}

	// 编程语言
	languages := []string{"go", "golang", "python", "java", "javascript", "typescript", "rust", "c++"}
	for _, lang := range languages {
		if strings.Contains(query, lang) {
			techStack = append(techStack, lang)
		}
	}

	// 框架和库
	frameworks := []string{"react", "vue", "angular", "spring", "django", "express", "gin"}
	for _, fw := range frameworks {
		if strings.Contains(query, fw) {
			techStack = append(techStack, fw)
		}
	}

	// 数据库
	databases := []string{"mysql", "postgresql", "mongodb", "redis", "elasticsearch"}
	for _, db := range databases {
		if strings.Contains(query, db) {
			techStack = append(techStack, db)
		}
	}

	return techStack
}

// calculateConfidence 计算置信度 (融入v2设计)
func (bia *BasicQueryIntentAnalyzer) calculateConfidence(intentType, domain string, keywords []interfaces.KeywordInfo) float64 {
	confidence := 0.5 // 基础置信度

	// 基于意图类型
	if intentType != "technical" {
		confidence += 0.2 // 非默认意图类型增加置信度
	}

	// 基于领域识别
	if domain != "programming" {
		confidence += 0.2 // 非默认领域增加置信度
	}

	// 基于关键词质量 (v2优势：权重和分类)
	if len(keywords) > 3 {
		confidence += 0.1
	}

	// 基于关键词权重和分类质量 (v2优势)
	var weightSum float64
	var techTermCount int
	for _, keyword := range keywords {
		weightSum += keyword.Weight
		if keyword.Category == "technical" || keyword.Category == "domain" {
			techTermCount++
		}
	}

	// 根据平均权重调整置信度
	if len(keywords) > 0 {
		avgWeight := weightSum / float64(len(keywords))
		if avgWeight > 0.7 {
			confidence += 0.15 // 高权重关键词
		} else if avgWeight < 0.3 {
			confidence -= 0.1 // 低权重关键词
		}
	}

	// 根据技术术语比例调整置信度
	if len(keywords) > 0 {
		techRatio := float64(techTermCount) / float64(len(keywords))
		if techRatio > 0.5 {
			confidence += 0.1 // 技术术语比例高
		}
	}

	// 限制在0.0-1.0范围内
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// ============================================================================
// 🛠️ 辅助方法
// ============================================================================

// containsAny 检查字符串是否包含任意一个模式
func (bia *BasicQueryIntentAnalyzer) containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// countTechnicalTerms 计算技术术语数量
func (bia *BasicQueryIntentAnalyzer) countTechnicalTerms(query string) int {
	technicalTerms := []string{
		"api", "json", "xml", "http", "https", "rest", "graphql",
		"docker", "kubernetes", "微服务", "分布式", "集群",
		"数据库", "缓存", "队列", "消息", "事务",
	}

	count := 0
	query = strings.ToLower(query)

	for _, term := range technicalTerms {
		if strings.Contains(query, term) {
			count++
		}
	}

	return count
}

// updateStats 更新统计信息
func (bia *BasicQueryIntentAnalyzer) updateStats(intentType string, confidence float64, processingTime time.Duration) {
	bia.stats.TotalAnalyzed++
	bia.stats.IntentDistribution[intentType]++

	// 更新平均置信度
	totalConfidence := bia.stats.AverageConfidence * float64(bia.stats.TotalAnalyzed-1)
	bia.stats.AverageConfidence = (totalConfidence + confidence) / float64(bia.stats.TotalAnalyzed)

	// 更新处理时间
	bia.stats.ProcessingTime = (bia.stats.ProcessingTime + processingTime) / 2
}

// initializeDefaultRules 初始化默认规则
func (bia *BasicQueryIntentAnalyzer) initializeDefaultRules() {
	// 初始化意图规则
	bia.intentRules["debugging"] = &IntentRule{
		IntentType: "debugging",
		Keywords:   []string{"调试", "错误", "bug", "问题", "修复"},
		Confidence: 0.8,
		Priority:   1,
	}

	bia.intentRules["technical"] = &IntentRule{
		IntentType: "technical",
		Keywords:   []string{"实现", "代码", "开发", "优化", "配置"},
		Confidence: 0.7,
		Priority:   2,
	}

	// 可以扩展更多规则...
}

// GetStats 获取统计信息
func (bia *BasicQueryIntentAnalyzer) GetStats() *AnalyzerStats {
	return bia.stats
}
