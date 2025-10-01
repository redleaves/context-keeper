package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/llm"
	"github.com/contextkeeper/service/internal/models"
)

// SemanticAnalysisEngine 语料分析引擎
// 负责第一次LLM调用：意图识别、查询拆解、上下文理解
type SemanticAnalysisEngine struct {
	enabled         bool
	llmClient       llm.LLMClient
	config          *SemanticAnalysisConfig
	metrics         *SemanticAnalysisMetrics
	strategyFactory *AnalysisStrategyFactory
}

// SemanticAnalysisConfig 语料分析配置
type SemanticAnalysisConfig struct {
	Enabled              bool    `json:"enabled" yaml:"enabled"`
	Provider             string  `json:"provider" yaml:"provider"`
	Model                string  `json:"model" yaml:"model"`
	MaxTokens            int     `json:"max_tokens" yaml:"max_tokens"`
	Temperature          float64 `json:"temperature" yaml:"temperature"`
	TimeoutSeconds       int     `json:"timeout_seconds" yaml:"timeout_seconds"`
	EnableIntentCache    bool    `json:"enable_intent_cache" yaml:"enable_intent_cache"`
	EnableQueryExpansion bool    `json:"enable_query_expansion" yaml:"enable_query_expansion"`
	// 策略相关配置
	AnalysisStrategy         string `json:"analysis_strategy" yaml:"analysis_strategy"` // "lightweight" 或 "deepIntent"
	EnableStrategyComparison bool   `json:"enable_strategy_comparison" yaml:"enable_strategy_comparison"`
}

// SemanticAnalysisMetrics 语料分析指标
type SemanticAnalysisMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// ContextInfo 上下文信息
type ContextInfo struct {
	// 会话上下文
	RecentConversation string `json:"recentConversation"` // 最近对话内容
	SessionTopic       string `json:"sessionTopic"`       // 会话主题

	// 项目上下文
	CurrentProject   string `json:"currentProject"`   // 当前项目
	WorkspaceContext string `json:"workspaceContext"` // 工作空间上下文

	// 历史上下文
	RelevantHistory string `json:"relevantHistory"` // 相关历史记录
	UserPreferences string `json:"userPreferences"` // 用户偏好

	// 技术上下文
	TechStack   []string `json:"techStack"`   // 技术栈
	CurrentTask string   `json:"currentTask"` // 当前任务

	// 兼容性字段（临时保留）
	ShortTermMemory string `json:"shortTermMemory,omitempty"`
	LongTermMemory  string `json:"longTermMemory,omitempty"`
	SessionState    string `json:"sessionState,omitempty"`
}

// SemanticAnalysisStrategy 语料分析策略接口
type SemanticAnalysisStrategy interface {
	AnalyzeQuery(ctx context.Context, query string, contextInfo *ContextInfo) (*SemanticAnalysisResult, error)
	GetStrategyName() string
}

// ComparisonResult 对比结果
type ComparisonResult struct {
	Primary           *SemanticAnalysisResult `json:"primary,omitempty"`
	LightweightResult *SemanticAnalysisResult `json:"lightweightResult,omitempty"`
	DeepIntentResult  *SemanticAnalysisResult `json:"deepIntentResult,omitempty"`
	Errors            map[string]error        `json:"errors,omitempty"`
}

// RecallMetrics 召回效果指标
type RecallMetrics struct {
	StrategyName   string                  `json:"strategyName"`
	QueryTime      time.Time               `json:"queryTime"`
	RecallCount    int                     `json:"recallCount"`
	TokensUsed     int                     `json:"tokensUsed"`
	ResponseTime   float64                 `json:"responseTime"`
	AnalysisResult *SemanticAnalysisResult `json:"analysisResult"`
}

// SemanticAnalysisRequest 语料分析请求
type SemanticAnalysisRequest struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id"`
	Context   string `json:"context,omitempty"`
}

// SemanticAnalysisResult 语料分析结果
type SemanticAnalysisResult struct {
	// 意图分析
	Intent     models.IntentType `json:"intent"`
	Confidence float64           `json:"confidence"`
	Categories []string          `json:"categories"`
	Keywords   []string          `json:"keywords"`
	Entities   []models.Entity   `json:"entities"`

	// 查询拆解
	Queries *models.MultiDimensionalQuery `json:"queries"`

	// 🆕 智能分析结果（包含时间回忆字段）
	SmartAnalysis *models.SmartAnalysisResult `json:"smart_analysis,omitempty"`

	// 元数据
	ProcessingTime time.Duration          `json:"processing_time"`
	TokenUsage     int                    `json:"token_usage"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewSemanticAnalysisEngine 创建语料分析引擎
func NewSemanticAnalysisEngine(config *SemanticAnalysisConfig, llmClient llm.LLMClient) *SemanticAnalysisEngine {
	if config == nil {
		config = getDefaultSemanticAnalysisConfig()
	}

	// 创建策略工厂
	strategyFactory := NewAnalysisStrategyFactory(llmClient, config)

	engine := &SemanticAnalysisEngine{
		enabled:         config.Enabled,
		llmClient:       llmClient,
		config:          config,
		strategyFactory: strategyFactory,
		metrics: &SemanticAnalysisMetrics{
			LastUpdated: time.Now(),
		},
	}

	log.Printf("🎯 [语料分析引擎] 初始化完成，状态: %v, 策略: %s, 对比模式: %v",
		config.Enabled, config.AnalysisStrategy, config.EnableStrategyComparison)
	return engine
}

// getDefaultSemanticAnalysisConfig 获取默认配置
func getDefaultSemanticAnalysisConfig() *SemanticAnalysisConfig {
	return &SemanticAnalysisConfig{
		Enabled:                  true,
		Provider:                 "openai",
		Model:                    "gpt-4",
		MaxTokens:                2000,
		Temperature:              0.1,
		TimeoutSeconds:           30,
		EnableIntentCache:        true,
		EnableQueryExpansion:     true,
		AnalysisStrategy:         "lightweight", // 默认使用轻量策略
		EnableStrategyComparison: false,         // 默认不开启对比模式
	}
}

// AnalyzeQuery 分析查询语料（保持向后兼容）
func (sae *SemanticAnalysisEngine) AnalyzeQuery(ctx context.Context, query string, sessionID string) (*SemanticAnalysisResult, error) {
	// 使用新的策略方法，传入空的上下文信息
	return sae.AnalyzeQueryWithStrategy(ctx, query, sessionID, nil)
}

// AnalyzeQueryWithStrategy 使用策略分析查询语料
func (sae *SemanticAnalysisEngine) AnalyzeQueryWithStrategy(ctx context.Context, query string, sessionID string, contextInfo *ContextInfo) (*SemanticAnalysisResult, error) {
	startTime := time.Now()
	sae.metrics.TotalRequests++

	// 检查是否启用
	if !sae.enabled {
		return nil, fmt.Errorf("语料分析引擎未启用")
	}

	log.Printf("🎯 [语料分析引擎] 开始分析查询: %s", query)

	// 检查是否开启对比模式
	if sae.config.EnableStrategyComparison {
		return sae.analyzeWithComparison(ctx, query, sessionID, contextInfo, startTime)
	}

	// 正常模式，使用配置的策略
	strategy := sae.strategyFactory.GetStrategy(sae.config.AnalysisStrategy)
	result, err := strategy.AnalyzeQuery(ctx, query, contextInfo)
	if err != nil {
		sae.metrics.FailedRequests++
		return nil, fmt.Errorf("策略分析失败: %w", err)
	}

	// 补充元数据
	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	result.Metadata = map[string]interface{}{
		"session_id":   sessionID,
		"strategy":     sae.config.AnalysisStrategy,
		"llm_model":    sae.config.Model,
		"llm_provider": sae.config.Provider,
		"timestamp":    time.Now(),
	}

	// 更新指标
	sae.updateMetrics(processingTime, true)
	sae.metrics.SuccessfulRequests++

	log.Printf("✅ [语料分析引擎] 分析完成，策略: %s, 意图: %s, 置信度: %.2f, 耗时: %v",
		sae.config.AnalysisStrategy, result.Intent, result.Confidence, processingTime)

	return result, nil
}

// analyzeWithComparison 对比分析模式
func (sae *SemanticAnalysisEngine) analyzeWithComparison(ctx context.Context, query string, sessionID string, contextInfo *ContextInfo, startTime time.Time) (*SemanticAnalysisResult, error) {
	log.Printf("🔄 [语料分析引擎] 开启对比模式，同时运行两种策略")

	// 获取两种策略
	lightweightStrategy := sae.strategyFactory.GetStrategy("lightweight")
	deepIntentStrategy := sae.strategyFactory.GetStrategy("deepIntent")

	// 串行执行两种策略（避免限流）
	var lightweightResult, deepIntentResult *SemanticAnalysisResult
	var lightweightErr, deepIntentErr error

	log.Printf("🔍 [对比模式] 开始执行轻量策略...")
	lightweightResult, lightweightErr = lightweightStrategy.AnalyzeQuery(ctx, query, contextInfo)

	if lightweightErr == nil {
		log.Printf("✅ [轻量策略] 执行成功，等待3秒后执行深度策略...")
		time.Sleep(3 * time.Second) // 避免限流
	}

	log.Printf("🎯 [对比模式] 开始执行深度策略...")
	deepIntentResult, deepIntentErr = deepIntentStrategy.AnalyzeQuery(ctx, query, contextInfo)

	// 记录对比结果
	sae.logComparisonResults(query, lightweightResult, deepIntentResult, lightweightErr, deepIntentErr)

	// 选择主要结果（优先使用配置的策略）
	var primaryResult *SemanticAnalysisResult
	var primaryErr error

	if sae.config.AnalysisStrategy == "deepIntent" {
		primaryResult, primaryErr = deepIntentResult, deepIntentErr
	} else {
		primaryResult, primaryErr = lightweightResult, lightweightErr
	}

	if primaryErr != nil {
		sae.metrics.FailedRequests++
		return nil, fmt.Errorf("主策略分析失败: %w", primaryErr)
	}

	// 补充元数据
	processingTime := time.Since(startTime)
	primaryResult.ProcessingTime = processingTime
	primaryResult.Metadata = map[string]interface{}{
		"session_id":      sessionID,
		"strategy":        sae.config.AnalysisStrategy,
		"comparison_mode": true,
		"llm_model":       sae.config.Model,
		"llm_provider":    sae.config.Provider,
		"timestamp":       time.Now(),
	}

	// 更新指标
	sae.updateMetrics(processingTime, true)
	sae.metrics.SuccessfulRequests++

	log.Printf("✅ [语料分析引擎] 对比分析完成，主策略: %s, 意图: %s, 置信度: %.2f, 耗时: %v",
		sae.config.AnalysisStrategy, primaryResult.Intent, primaryResult.Confidence, processingTime)

	return primaryResult, nil
}

// logComparisonResults 记录对比结果
func (sae *SemanticAnalysisEngine) logComparisonResults(query string, lightweightResult, deepIntentResult *SemanticAnalysisResult, lightweightErr, deepIntentErr error) {
	log.Printf("📊 [策略对比] 查询: %s", query)

	if lightweightErr != nil {
		log.Printf("❌ [轻量策略] 执行失败: %v", lightweightErr)
	} else {
		log.Printf("✅ [轻量策略] 意图: %s, 置信度: %.2f, Token: %d",
			lightweightResult.Intent, lightweightResult.Confidence, lightweightResult.TokenUsage)
		log.Printf("🔍 [轻量策略] 关键词数量: %d, 实体数量: %d",
			len(lightweightResult.Keywords), len(lightweightResult.Entities))
	}

	if deepIntentErr != nil {
		log.Printf("❌ [深度策略] 执行失败: %v", deepIntentErr)
	} else {
		log.Printf("✅ [深度策略] 意图: %s, 置信度: %.2f, Token: %d",
			deepIntentResult.Intent, deepIntentResult.Confidence, deepIntentResult.TokenUsage)
		log.Printf("🎯 [深度策略] 关键词数量: %d, 实体数量: %d",
			len(deepIntentResult.Keywords), len(deepIntentResult.Entities))
	}

	// 如果两个策略都成功，进行详细对比
	if lightweightErr == nil && deepIntentErr == nil {
		log.Printf("🔄 [策略对比] 意图一致性: %v", lightweightResult.Intent == deepIntentResult.Intent)
		log.Printf("🔄 [策略对比] 置信度差异: %.3f",
			float64(deepIntentResult.Confidence-lightweightResult.Confidence))
		log.Printf("🔄 [策略对比] Token使用差异: %d",
			deepIntentResult.TokenUsage-lightweightResult.TokenUsage)
	}
}

// buildAnalysisPrompt 构建分析Prompt
func (sae *SemanticAnalysisEngine) buildAnalysisPrompt(query string, sessionID string) string {
	prompt := fmt.Sprintf(`你是一个专业的语料分析专家，请分析用户的查询并返回结构化的JSON结果。

用户查询: "%s"
会话ID: %s

请分析以下内容并返回JSON格式的结果：

1. 意图识别 (intent): 从以下类型中选择最匹配的
   - query: 查询意图
   - command: 命令意图  
   - conversation: 对话意图
   - analysis: 分析意图
   - creation: 创建意图
   - modification: 修改意图

2. 置信度 (confidence): 0-1之间的数值

3. 分类标签 (categories): 相关的分类标签数组

4. 关键词 (keywords): 提取的关键词数组

5. 实体识别 (entities): 识别的实体，每个实体包含text, type, confidence

6. 多维度查询拆解 (queries):
   - context_queries: 上下文相关的查询
   - timeline_queries: 时间线相关的查询  
   - knowledge_queries: 知识图谱相关的查询
   - vector_queries: 向量检索相关的查询

返回格式示例：
{
  "intent": "query",
  "confidence": 0.95,
  "categories": ["技术", "编程"],
  "keywords": ["API", "调用", "方法"],
  "entities": [
    {"text": "API", "type": "技术概念", "confidence": 0.9}
  ],
  "queries": {
    "context_queries": ["API调用方法", "接口使用"],
    "timeline_queries": ["最近的API变更"],
    "knowledge_queries": ["API相关概念"],
    "vector_queries": ["API调用示例"]
  }
}

请只返回JSON，不要包含其他文本：`, query, sessionID)

	return prompt
}

// parseLLMResponse 解析LLM响应
func (sae *SemanticAnalysisEngine) parseLLMResponse(content string) (*SemanticAnalysisResult, error) {
	// 清理响应内容
	content = strings.TrimSpace(content)

	// 尝试提取JSON部分
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	}

	content = strings.TrimSpace(content)

	// 解析JSON
	var rawResult struct {
		Intent     string   `json:"intent"`
		Confidence float64  `json:"confidence"`
		Categories []string `json:"categories"`
		Keywords   []string `json:"keywords"`
		Entities   []struct {
			Text       string  `json:"text"`
			Type       string  `json:"type"`
			Confidence float64 `json:"confidence"`
		} `json:"entities"`
		Queries struct {
			ContextQueries   []string `json:"context_queries"`
			TimelineQueries  []string `json:"timeline_queries"`
			KnowledgeQueries []string `json:"knowledge_queries"`
			VectorQueries    []string `json:"vector_queries"`
		} `json:"queries"`
	}

	if err := json.Unmarshal([]byte(content), &rawResult); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w, 内容: %s", err, content)
	}

	// 转换为结果结构
	result := &SemanticAnalysisResult{
		Intent:     models.IntentType(rawResult.Intent),
		Confidence: rawResult.Confidence,
		Categories: rawResult.Categories,
		Keywords:   rawResult.Keywords,
		Entities:   make([]models.Entity, len(rawResult.Entities)),
		Queries: &models.MultiDimensionalQuery{
			ContextQueries:   rawResult.Queries.ContextQueries,
			TimelineQueries:  rawResult.Queries.TimelineQueries,
			KnowledgeQueries: rawResult.Queries.KnowledgeQueries,
			VectorQueries:    rawResult.Queries.VectorQueries,
		},
	}

	// 转换实体
	for i, entity := range rawResult.Entities {
		result.Entities[i] = models.Entity{
			Text:       entity.Text,
			Type:       entity.Type,
			Confidence: entity.Confidence,
		}
	}

	return result, nil
}

// updateMetrics 更新指标
func (sae *SemanticAnalysisEngine) updateMetrics(latency time.Duration, success bool) {
	if sae.metrics.TotalRequests > 0 {
		sae.metrics.AverageLatency = (sae.metrics.AverageLatency*time.Duration(sae.metrics.TotalRequests-1) + latency) / time.Duration(sae.metrics.TotalRequests)
	} else {
		sae.metrics.AverageLatency = latency
	}

	sae.metrics.LastUpdated = time.Now()
}

// GetMetrics 获取指标
func (sae *SemanticAnalysisEngine) GetMetrics() *SemanticAnalysisMetrics {
	return sae.metrics
}

// SetEnabled 设置启用状态
func (sae *SemanticAnalysisEngine) SetEnabled(enabled bool) {
	sae.enabled = enabled
	log.Printf("🎯 [语料分析引擎] 状态更新: %v", enabled)
}
