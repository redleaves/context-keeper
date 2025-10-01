package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/engines"
	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/knowledge"
	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/timeline"
	"github.com/contextkeeper/service/internal/llm"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
)

// LLMDrivenContextService LLM驱动的上下文服务
// 直接替代AgenticContextService，基于ContextService构建
type LLMDrivenContextService struct {
	// 基础服务（直接包装ContextService）
	contextService *ContextService

	// LLM驱动组件
	semanticAnalyzer   *engines.SemanticAnalysisEngine
	multiRetriever     MultiDimensionalRetriever
	contentSynthesizer ContentSynthesisEngine

	// 🆕 上下文管理器（关键闭环组件）
	contextManager *UnifiedContextManager

	// 配置和开关
	config  *LLMDrivenConfig
	enabled bool
	metrics *LLMDrivenMetrics
}

// LLMDrivenConfig LLM驱动服务配置
type LLMDrivenConfig struct {
	// 总开关
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 功能开关
	SemanticAnalysis bool `json:"semantic_analysis" yaml:"semantic_analysis"`
	MultiDimensional bool `json:"multi_dimensional" yaml:"multi_dimensional"`
	ContentSynthesis bool `json:"content_synthesis" yaml:"content_synthesis"`
	// 🔥 短期记忆LLM驱动开关（独立控制，默认关闭）
	ShortTermMemoryLLM bool `json:"short_term_memory_llm" yaml:"short_term_memory_llm"`

	// 降级策略
	AutoFallback      bool `json:"auto_fallback" yaml:"auto_fallback"`
	FallbackThreshold int  `json:"fallback_threshold" yaml:"fallback_threshold"`

	// LLM配置
	LLM struct {
		Provider    string  `json:"provider" yaml:"provider"`
		Model       string  `json:"model" yaml:"model"`
		MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
		Temperature float64 `json:"temperature" yaml:"temperature"`
	} `json:"llm" yaml:"llm"`
}

// LLMDrivenMetrics LLM驱动服务监控指标
type LLMDrivenMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	LLMDrivenRequests int64         `json:"llm_driven_requests"`
	FallbackRequests  int64         `json:"fallback_requests"`
	SuccessRate       float64       `json:"success_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorCount        int64         `json:"error_count"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// 接口定义
type SemanticAnalysisEngine interface {
	AnalyzeQuery(ctx context.Context, query string, sessionID string) (*engines.SemanticAnalysisResult, error)
	SetEnabled(enabled bool)
	GetMetrics() interface{}
}

type MultiDimensionalRetriever interface {
	ParallelRetrieve(ctx context.Context, queries *RetrievalQueries) (*RetrievalResults, error)
	GetTimelineAdapter() TimelineAdapter                                                                         // 🆕 新增方法
	DirectTimelineQuery(ctx context.Context, req *models.TimelineSearchRequest) ([]*models.TimelineEvent, error) // 🆕 直接时间线查询
}

type ContentSynthesisEngine interface {
	SynthesizeResponse(ctx context.Context, query string, analysis *engines.SemanticAnalysisResult, retrieval *RetrievalResults) (models.ContextResponse, error)
}

// 🆕 TimelineAdapter 时间线适配器接口
type TimelineAdapter interface {
	Retrieve(ctx context.Context, req *TimelineRetrievalRequest) ([]*models.TimelineEvent, error)
}

// 🆕 TimelineRetrievalRequest 时间线检索请求
type TimelineRetrievalRequest struct {
	UserID      string     `json:"user_id"`
	WorkspaceID string     `json:"workspace_id"`
	Query       string     `json:"query"`
	Limit       int        `json:"limit"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
}

// 数据结构定义
type SemanticAnalysisResult struct {
	Intent         models.IntentType             `json:"intent"`
	Confidence     float64                       `json:"confidence"`
	Categories     []string                      `json:"categories"`
	Keywords       []string                      `json:"keywords"`
	Entities       []models.Entity               `json:"entities"`
	Queries        *models.MultiDimensionalQuery `json:"queries"`
	ProcessingTime time.Duration                 `json:"processing_time"`
	TokenUsage     int                           `json:"token_usage"`
	Metadata       map[string]interface{}        `json:"metadata"`
}

type RetrievalQueries = models.MultiDimensionalQuery

type RetrievalResults struct {
	Results []interface{} `json:"results"`
	Sources []string      `json:"sources"`
}

// MultiDimensionalRetrieverAdapter 多维度检索器适配器
type MultiDimensionalRetrieverAdapter struct {
	Impl *engines.MultiDimensionalRetrieverImpl
}

// SimpleTimelineAdapter 简单时间线适配器
type SimpleTimelineAdapter struct {
	impl *engines.MultiDimensionalRetrieverImpl
}

// Retrieve 实现TimelineAdapter接口
func (adapter *SimpleTimelineAdapter) Retrieve(ctx context.Context, req *TimelineRetrievalRequest) ([]*models.TimelineEvent, error) {
	log.Printf("📥 [SimpleTimelineAdapter] 查询参数: %+v", req)

	// 🔥 真正实现：调用多维检索器的DirectTimelineQuery方法
	if adapter.impl != nil {
		// 构建时间线搜索请求
		searchReq := &models.TimelineSearchRequest{
			UserID:      req.UserID,
			WorkspaceID: req.WorkspaceID,
			Query:       req.Query, // 空查询，纯时间范围过滤
			Limit:       req.Limit,
			StartTime:   req.StartTime, // 🔥 关键：传递时间范围
			EndTime:     req.EndTime,   // 🔥 关键：传递时间范围
		}

		log.Printf("🔍 [SimpleTimelineAdapter] 调用多维检索器的DirectTimelineQuery")

		// 🔥 关键：调用新增的DirectTimelineQuery方法，会真实执行SQL查询
		events, err := adapter.impl.DirectTimelineQuery(ctx, searchReq)
		if err != nil {
			log.Printf("❌ [SimpleTimelineAdapter] 直接时间线查询失败: %v", err)
			return nil, err
		}

		log.Printf("✅ [SimpleTimelineAdapter] 查询成功，返回 %d 个事件", len(events))
		return events, nil
	}

	log.Printf("❌ [SimpleTimelineAdapter] 多维检索器未初始化")
	return []*models.TimelineEvent{}, nil
}

// convertTimelineEventToModel 转换时间线事件到模型
func convertTimelineEventToModel(event *timeline.TimelineEvent) *models.TimelineEvent {
	return &models.TimelineEvent{
		ID:              event.ID,
		UserID:          event.UserID,
		SessionID:       event.SessionID,
		WorkspaceID:     event.WorkspaceID,
		Timestamp:       event.Timestamp,
		EventType:       event.EventType,
		Title:           event.Title,
		Content:         event.Content,
		Summary:         event.Summary,
		ImportanceScore: event.ImportanceScore,
		RelevanceScore:  event.RelevanceScore,
		CreatedAt:       event.CreatedAt,
		UpdatedAt:       event.UpdatedAt,
	}
}

// GetTimelineAdapter 实现MultiDimensionalRetriever接口
func (adapter *MultiDimensionalRetrieverAdapter) GetTimelineAdapter() TimelineAdapter {
	return &SimpleTimelineAdapter{impl: adapter.Impl}
}

// 🆕 DirectTimelineQuery 实现接口方法，直接时间线查询
func (adapter *MultiDimensionalRetrieverAdapter) DirectTimelineQuery(ctx context.Context, req *models.TimelineSearchRequest) ([]*models.TimelineEvent, error) {
	log.Printf("🔍 [适配器] 调用底层DirectTimelineQuery")
	return adapter.Impl.DirectTimelineQuery(ctx, req)
}

// ParallelRetrieve 实现接口方法
func (adapter *MultiDimensionalRetrieverAdapter) ParallelRetrieve(ctx context.Context, queries *RetrievalQueries) (*RetrievalResults, error) {
	// 从上下文提取用户与工作空间信息，移除硬编码
	userID, _ := ctx.Value("user_id").(string)
	workspaceID, _ := ctx.Value("workspacePath").(string)

	// 🔥 修复：填充用户、工作空间和LLM分析信息
	queries.UserID = userID
	queries.WorkspaceID = workspaceID
	// TODO: 填充KeyConcepts字段从上层LLM分析结果

	// 调用实际实现
	engineResults, err := adapter.Impl.ParallelRetrieve(ctx, queries)
	if err != nil {
		return nil, err
	}

	// 转换结果格式
	results := &RetrievalResults{
		Results: make([]interface{}, 0),
		Sources: []string{},
	}

	// 添加时间线结果
	for _, result := range engineResults.TimelineResults {
		results.Results = append(results.Results, result)
		results.Sources = append(results.Sources, "timeline")
	}

	// 添加知识图谱结果
	for _, result := range engineResults.KnowledgeResults {
		results.Results = append(results.Results, result)
		results.Sources = append(results.Sources, "knowledge")
	}

	// 添加向量结果
	for _, result := range engineResults.VectorResults {
		results.Results = append(results.Results, result)
		results.Sources = append(results.Sources, "vector")
	}

	return results, nil
}

// ContentSynthesisEngineAdapter 内容合成引擎适配器
type ContentSynthesisEngineAdapter struct {
	impl *engines.ContentSynthesisEngineImpl
}

// SynthesizeResponse 实现接口方法
func (adapter *ContentSynthesisEngineAdapter) SynthesizeResponse(ctx context.Context, query string, analysis *engines.SemanticAnalysisResult, retrieval *RetrievalResults) (models.ContextResponse, error) {
	// 转换检索结果格式
	engineRetrieval := &engines.RetrievalResults{
		TimelineResults:  []*models.TimelineEvent{},
		KnowledgeResults: []*models.KnowledgeNode{},
		VectorResults:    []*models.VectorMatch{},
		TimelineCount:    0,
		KnowledgeCount:   0,
		VectorCount:      0,
		TotalResults:     len(retrieval.Results),
		OverallQuality:   0.8,
		RetrievalTime:    0,
		Results:          retrieval.Results,
	}

	// 从Results中提取具体类型的结果
	for i, result := range retrieval.Results {
		source := ""
		if i < len(retrieval.Sources) {
			source = retrieval.Sources[i]
		}

		switch source {
		case "timeline":
			if event, ok := result.(*models.TimelineEvent); ok {
				engineRetrieval.TimelineResults = append(engineRetrieval.TimelineResults, event)
				engineRetrieval.TimelineCount++
			}
		case "knowledge":
			if node, ok := result.(*models.KnowledgeNode); ok {
				engineRetrieval.KnowledgeResults = append(engineRetrieval.KnowledgeResults, node)
				engineRetrieval.KnowledgeCount++
			}
		case "vector":
			if match, ok := result.(*models.VectorMatch); ok {
				engineRetrieval.VectorResults = append(engineRetrieval.VectorResults, match)
				engineRetrieval.VectorCount++
			}
		}
	}

	// 调用实际实现
	return adapter.impl.SynthesizeResponse(ctx, query, analysis, engineRetrieval)
}

// NewLLMDrivenContextServiceWithEngines 创建带存储引擎的LLM驱动上下文服务
func NewLLMDrivenContextServiceWithEngines(contextService *ContextService, storageEngines map[string]interface{}) *LLMDrivenContextService {
	cfg := loadLLMDrivenConfig()

	service := &LLMDrivenContextService{
		contextService: contextService,
		config:         cfg,
		enabled:        cfg.Enabled,
		metrics: &LLMDrivenMetrics{
			LastUpdated: time.Now(),
		},
	}

	// 初始化LLM驱动组件（如果启用）
	if cfg.Enabled {
		service.initializeLLMComponentsWithEngines(storageEngines)
	}

	log.Printf("🚀 [LLM驱动服务] 初始化完成（带存储引擎），状态: %v", cfg.Enabled)
	return service
}

// loadLLMDrivenConfig 加载LLM驱动配置
// 统一使用环境变量作为唯一配置源，简化开关逻辑
func loadLLMDrivenConfig() *LLMDrivenConfig {
	log.Printf("🔧 [配置加载] 开始加载LLM驱动配置，仅从环境变量读取")

	// 设置默认值
	cfg := &LLMDrivenConfig{
		// 🔥 主开关：默认启用，由环境变量控制
		Enabled: getEnvAsBool("LLM_DRIVEN_ENABLED", true),

		// 功能开关：默认启用子功能
		SemanticAnalysis:   getEnvAsBool("LLM_DRIVEN_SEMANTIC_ANALYSIS", true),
		MultiDimensional:   getEnvAsBool("LLM_DRIVEN_MULTI_DIMENSIONAL", true),
		ContentSynthesis:   getEnvAsBool("LLM_DRIVEN_CONTENT_SYNTHESIS", true),
		ShortTermMemoryLLM: getEnvAsBool("LLM_DRIVEN_SHORT_TERM_MEMORY", false),

		// 容错设置
		AutoFallback:      getEnvAsBool("LLM_DRIVEN_AUTO_FALLBACK", true),
		FallbackThreshold: getEnvAsInt("LLM_DRIVEN_FALLBACK_THRESHOLD", 3),
	}

	// LLM配置
	cfg.LLM.Provider = getEnv("LLM_PROVIDER", "deepseek")
	cfg.LLM.Model = getEnv("LLM_MODEL", "deepseek-chat")
	cfg.LLM.MaxTokens = getEnvAsInt("LLM_MAX_TOKENS", 4000)
	cfg.LLM.Temperature = getEnvAsFloat("LLM_TEMPERATURE", 0.3)

	log.Printf("🎯 [配置加载] LLM驱动配置加载完成")
	log.Printf("   🔑 主开关: enabled=%v", cfg.Enabled)
	log.Printf("   🧠 语料分析: %v", cfg.SemanticAnalysis)
	log.Printf("   📊 多维检索: %v", cfg.MultiDimensional)
	log.Printf("   ✨ 内容合成: %v", cfg.ContentSynthesis)
	log.Printf("   🤖 LLM提供商: %s/%s", cfg.LLM.Provider, cfg.LLM.Model)

	return cfg
}

// 从环境变量获取字符串值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 从环境变量获取整数值
func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}

// 从环境变量获取布尔值
func getEnvAsBool(key string, defaultValue bool) bool {
	strValue := getEnv(key, "")
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value
	}
	return defaultValue
}

// 从环境变量获取浮点值
func getEnvAsFloat(key string, defaultValue float64) float64 {
	strValue := getEnv(key, "")
	if value, err := strconv.ParseFloat(strValue, 64); err == nil {
		return value
	}
	return defaultValue
}

// initializeLLMComponentsWithEngines 初始化LLM组件（带存储引擎）
func (lds *LLMDrivenContextService) initializeLLMComponentsWithEngines(storageEngines map[string]interface{}) {
	log.Printf("🧠 [LLM驱动服务] 初始化LLM组件（带存储引擎）...")

	// 创建LLM客户端
	llmClient, err := lds.createLLMClient()
	if err != nil {
		log.Printf("❌ [LLM驱动服务] LLM客户端创建失败: %v", err)
		return
	}
	log.Printf("✅ [LLM驱动服务] LLM客户端创建成功，提供商: %s, 模型: %s", lds.config.LLM.Provider, lds.config.LLM.Model)

	// 语料分析引擎
	if lds.config.SemanticAnalysis {
		semanticConfig := &engines.SemanticAnalysisConfig{
			Enabled:              true,
			Provider:             lds.config.LLM.Provider,
			Model:                lds.config.LLM.Model,
			MaxTokens:            lds.config.LLM.MaxTokens,
			Temperature:          lds.config.LLM.Temperature,
			TimeoutSeconds:       30,
			EnableIntentCache:    true,
			EnableQueryExpansion: true,
		}
		lds.semanticAnalyzer = engines.NewSemanticAnalysisEngine(semanticConfig, llmClient)
		log.Printf("✅ [LLM驱动服务] 语料分析引擎已初始化")
	}

	// 多维度检索引擎（从storageEngines中获取已创建的实例）
	if lds.config.MultiDimensional {
		if multiRetriever, exists := storageEngines["multi_retriever"]; exists && multiRetriever != nil {
			lds.multiRetriever = multiRetriever.(MultiDimensionalRetriever)
			log.Printf("✅ [LLM驱动服务] 多维度检索引擎已连接: %T", multiRetriever)

			// 🔥 延迟赋值：直接基于MultiDimensionalRetriever的vectorAdapter进行赋值
			if adapter, ok := lds.multiRetriever.(*MultiDimensionalRetrieverAdapter); ok {
				if vectorStore := lds.contextService.GetVectorStore(); vectorStore != nil {
					adapter.Impl.SetVectorStoreEngine(vectorStore)
					log.Printf("✅ [延迟赋值] 成功设置MultiDimensionalRetriever的vectorAdapter.Engine")
				} else {
					log.Printf("⚠️ [延迟赋值] contextService.GetVectorStore()为nil")
				}
			}
		} else {
			log.Printf("⚠️ [LLM驱动服务] 多维度检索引擎未找到，多维度检索功能将不可用")
		}
	}

	// 内容合成引擎
	if lds.config.ContentSynthesis {
		// 初始化内容合成引擎
		contentSynthesizerImpl := engines.NewContentSynthesisEngine(llmClient)
		// 创建适配器
		lds.contentSynthesizer = &ContentSynthesisEngineAdapter{impl: contentSynthesizerImpl}
		log.Printf("✅ [LLM驱动服务] 内容合成引擎已初始化")
	}

	log.Printf("🎯 [LLM驱动服务] LLM组件初始化完成（带存储引擎）")
}

// ReinitializeVectorEngine 重新初始化向量引擎（用于延迟赋值）
func (lds *LLMDrivenContextService) ReinitializeVectorEngine() {
	log.Printf("🔧 [LLM驱动服务] 开始重新初始化向量引擎...")

	// 检查LLM驱动是否启用
	if !lds.enabled {
		log.Printf("📝 [向量引擎重初始化] LLM驱动功能已禁用，无需初始化向量引擎")
		return
	}

	// 检查多维度检索器是否存在
	if lds.multiRetriever == nil {
		log.Printf("⚠️ [向量引擎重初始化] 多维度检索器为nil，跳过")
		return
	}

	// 🔥 重新执行延迟赋值逻辑
	if adapter, ok := lds.multiRetriever.(*MultiDimensionalRetrieverAdapter); ok {
		if vectorStore := lds.contextService.GetVectorStore(); vectorStore != nil {
			adapter.Impl.SetVectorStoreEngine(vectorStore)
			log.Printf("✅ [向量引擎重初始化] 成功重新设置MultiDimensionalRetriever的vectorAdapter.Engine")
		} else {
			log.Printf("❌ [向量引擎重初始化] contextService.GetVectorStore()仍然为nil")
		}
	} else {
		log.Printf("⚠️ [向量引擎重初始化] 多维度检索器类型断言失败: %T", lds.multiRetriever)
	}

	log.Printf("🎯 [LLM驱动服务] 向量引擎重新初始化完成")
}

// createLLMClient 创建LLM客户端
func (lds *LLMDrivenContextService) createLLMClient() (llm.LLMClient, error) {
	// 根据Provider获取对应的API密钥
	var apiKey string
	switch lds.config.LLM.Provider {
	case "deepseek":
		apiKey = getEnv("DEEPSEEK_API_KEY", "")
	case "openai":
		apiKey = getEnv("OPENAI_API_KEY", "")
	case "claude":
		apiKey = getEnv("CLAUDE_API_KEY", "")
	case "qianwen":
		apiKey = getEnv("QIANWEN_API_KEY", "")
	case "ollama_local":
		// 🆕 本地模型不需要API密钥
		apiKey = ""
	default:
		return nil, fmt.Errorf("不支持的LLM提供商: %s", lds.config.LLM.Provider)
	}

	// 🔥 修复：本地模型不需要API密钥检查
	if apiKey == "" && lds.config.LLM.Provider != "ollama_local" {
		return nil, fmt.Errorf("LLM API密钥未配置，提供商: %s", lds.config.LLM.Provider)
	}

	// 设置LLM配置（基础配置）
	config := &llm.LLMConfig{
		Provider:   llm.LLMProvider(lds.config.LLM.Provider),
		APIKey:     apiKey,
		Model:      lds.config.LLM.Model,
		MaxRetries: 3,
		Timeout:    120 * time.Second, // 增加到120秒，适应复杂LLM分析
		RateLimit:  60,                // 每分钟60次请求
	}

	// 🆕 设置本地模型的BaseURL
	if lds.config.LLM.Provider == "ollama_local" {
		config.BaseURL = "http://localhost:11434"
		config.RateLimit = 0              // 本地模型无限流限制
		config.Timeout = 60 * time.Second // 本地模型更快
	}

	// 设置全局配置
	llm.SetGlobalConfig(llm.LLMProvider(lds.config.LLM.Provider), config)

	// 使用工厂创建客户端
	client, err := llm.CreateGlobalClient(llm.LLMProvider(lds.config.LLM.Provider))
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	log.Printf("✅ [LLM驱动服务] LLM客户端创建成功，提供商: %s, 模型: %s",
		lds.config.LLM.Provider, lds.config.LLM.Model)

	// 🔥 执行健康检查验证模型可用性
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		log.Printf("⚠️ [LLM驱动服务] 模型健康检查失败: %v", err)
		// 不返回错误，允许服务启动但记录警告
	} else {
		log.Printf("✅ [LLM驱动服务] 模型健康检查通过")
	}

	return client, nil
}

// RetrieveContext 实现ContextServiceInterface接口 - 核心方法
func (lds *LLMDrivenContextService) RetrieveContext(ctx context.Context, req models.RetrieveContextRequest) (models.ContextResponse, error) {
	lds.metrics.TotalRequests++
	lds.metrics.LastUpdated = time.Now()

	// 🔥 关键开关：LLM驱动 vs 基础服务
	if !lds.enabled {
		log.Printf("🔄 [LLM驱动服务] LLM驱动功能已禁用，使用基础ContextService")
		lds.metrics.FallbackRequests++
		return lds.contextService.RetrieveContext(ctx, req)
	}

	log.Printf("🚀 [LLM驱动服务] 启用LLM驱动智能化流程，查询: %s", req.Query)
	lds.metrics.LLMDrivenRequests++

	startTime := time.Now()

	// 执行LLM驱动的智能化流程
	response, err := lds.executeLLMDrivenFlow(ctx, req)
	if err != nil {
		lds.metrics.ErrorCount++

		// 自动降级到基础服务
		if lds.config.AutoFallback {
			log.Printf("⚠️ [LLM驱动服务] LLM驱动流程失败，自动降级到基础ContextService: %v", err)
			lds.metrics.FallbackRequests++
			return lds.contextService.RetrieveContext(ctx, req)
		}

		return models.ContextResponse{}, fmt.Errorf("LLM驱动流程失败: %w", err)
	}

	// 更新性能指标
	latency := time.Since(startTime)
	lds.updateMetrics(latency, true)

	log.Printf("✅ [LLM驱动服务] LLM驱动流程完成，耗时: %v", latency)
	return response, nil
}

// 基于llm驱动的【宽召回+精排序】
func (lds *LLMDrivenContextService) executeLLMDrivenFlow(ctx context.Context, req models.RetrieveContextRequest) (models.ContextResponse, error) {
	// Phase 1: 第一次LLM调用 - 语料分析
	if lds.config.SemanticAnalysis && lds.semanticAnalyzer != nil {
		log.Printf("🎯 [LLM驱动服务] 执行语料分析...")
		analysisResult, err := lds.semanticAnalyzer.AnalyzeQuery(ctx, req.Query, req.SessionID)
		if err != nil {
			return models.ContextResponse{}, fmt.Errorf("语料分析失败: %w", err)
		}
		log.Printf("✅ [LLM驱动服务] 语料分析完成，识别意图: %s", analysisResult.Intent)

		// 🆕 Phase 1.5: 检查是否为时间回忆查询 - 优先处理
		if lds.checkTimelineRecallQuery(analysisResult) {
			log.Printf("🕒 [时间回忆模式] 检测到时间范围查询，使用专用时间线检索")
			return lds.handleTimelineRecallQuery(ctx, req, analysisResult)
		}

		// 🆕 工程感知集成：如果提供了ProjectAnalysis，融合到检索请求中  TODO 待定这个逻辑
		if req.ProjectAnalysis != "" {
			log.Printf("🔧 [工程感知] 检测到项目分析信息，长度: %d字符", len(req.ProjectAnalysis))
			// 将工程感知信息添加到检索查询中，增强上下文理解
			analysisResult.Queries.ContextQueries = append(analysisResult.Queries.ContextQueries,
				"项目上下文: "+req.ProjectAnalysis)
			log.Printf("🔧 [工程感知] 已将项目分析融合到上下文检索中")
		}

		// Phase 2: 多维度并行检索
		if lds.config.MultiDimensional && lds.multiRetriever != nil {
			log.Printf("🔍 [LLM驱动服务] 执行多维度检索...")

			// 🔥 关键修复：从LLM分析结果中提取关键概念
			if analysisResult.Keywords != nil {
				analysisResult.Queries.KeyConcepts = analysisResult.Keywords
				log.Printf("🔥 [LLM驱动服务] 已设置关键概念: %v", analysisResult.Keywords)
			}

			retrievalResults, err := lds.multiRetriever.ParallelRetrieve(ctx, analysisResult.Queries)
			if err != nil {
				return models.ContextResponse{}, fmt.Errorf("多维度检索失败: %w", err)
			}
			log.Printf("✅ [LLM驱动服务] 多维度检索完成，获得 %d 个结果", len(retrievalResults.Results))

			// Phase 3: 第二次LLM调用 - 内容合成
			if lds.config.ContentSynthesis && lds.contentSynthesizer != nil {
				log.Printf("🧠 [LLM驱动服务] 执行内容合成...")

				// 🆕 传递工程感知信息到合成引擎
				enrichedCtx := ctx
				if req.ProjectAnalysis != "" {
					enrichedCtx = context.WithValue(ctx, "project_analysis", req.ProjectAnalysis)
					log.Printf("🔧 [工程感知] 将项目分析信息传递给内容合成引擎")
				}

				response, err := lds.contentSynthesizer.SynthesizeResponse(enrichedCtx, req.Query, analysisResult, retrievalResults)
				if err != nil {
					return models.ContextResponse{}, fmt.Errorf("内容合成失败: %w", err)
				}
				log.Printf("✅ [LLM驱动服务] 内容合成完成")
				return response, nil
			}
		}
	}

	// 如果某些组件未启用，降级到基础服务
	log.Printf("⚠️ [LLM驱动服务] LLM驱动组件未完全启用，降级到基础服务")
	return lds.contextService.RetrieveContext(ctx, req)
}

// StoreContext 代理到基础ContextService
func (lds *LLMDrivenContextService) StoreContext(ctx context.Context, req models.StoreContextRequest) (string, error) {
	return lds.contextService.StoreContext(ctx, req)
}

// RetrieveConversation 代理到基础ContextService
func (lds *LLMDrivenContextService) RetrieveConversation(ctx context.Context, req models.RetrieveConversationRequest) (*models.ConversationResponse, error) {
	return lds.contextService.RetrieveConversation(ctx, req)
}

// GetProgrammingContext 代理到基础ContextService
func (lds *LLMDrivenContextService) GetProgrammingContext(ctx context.Context, sessionID string, query string) (*models.ProgrammingContext, error) {
	return lds.contextService.GetProgrammingContext(ctx, sessionID, query)
}

// GetContextService 获取基础ContextService（用于MCP工具等需要直接访问基础服务的场景）
func (lds *LLMDrivenContextService) GetContextService() *ContextService {
	return lds.contextService
}

// SetContextManager 设置统一上下文管理器
func (lds *LLMDrivenContextService) SetContextManager(manager *UnifiedContextManager) {
	lds.contextManager = manager
}

// ============================================================================
// 🔄 代理方法 - 完全兼容AgenticContextService接口
// ============================================================================

// SummarizeContext 总结上下文
func (lds *LLMDrivenContextService) SummarizeContext(ctx context.Context, req models.SummarizeContextRequest) (string, error) {
	return lds.contextService.SummarizeContext(ctx, req)
}

// StoreSessionMessages 存储会话消息 - LLM驱动的智能存储
func (lds *LLMDrivenContextService) StoreSessionMessages(ctx context.Context, req models.StoreMessagesRequest) (*models.StoreMessagesResponse, error) {
	// 🔥 检查短期记忆LLM驱动开关（独立控制）
	if !lds.enabled || !lds.config.ShortTermMemoryLLM {
		log.Printf("🔄 [短期记忆存储] LLM驱动功能未启用或短期记忆LLM开关关闭，使用原有本地存储逻辑")
		return lds.contextService.StoreSessionMessages(ctx, req)
	}

	log.Printf("🧠 [LLM驱动存储] 开始智能存储分析: 会话=%s, 消息数=%d", req.SessionID, len(req.Messages))

	// 对每条消息进行智能分析和存储
	var allMessageIDs []string
	var smartAnalysisResults []map[string]interface{}

	for i, msgReq := range req.Messages {
		log.Printf("🔍 [LLM驱动存储] 分析消息 %d/%d: %s", i+1, len(req.Messages), msgReq.Content[:min(50, len(msgReq.Content))])

		// 执行智能存储决策
		result, err := lds.executeSmartStorage(ctx, req.SessionID, msgReq.Content, msgReq.Priority)
		if err != nil {
			log.Printf("❌ [LLM驱动存储] 智能存储失败: %v", err)
			// 降级到基础存储
			return lds.contextService.StoreSessionMessages(ctx, req)
		}

		// 收集结果
		if result.MessageIDs != nil {
			allMessageIDs = append(allMessageIDs, result.MessageIDs...)
		}

		// 收集智能分析结果
		smartAnalysisResults = append(smartAnalysisResults, map[string]interface{}{
			"messageIndex":    i,
			"content":         msgReq.Content,
			"confidence":      result.Confidence,
			"storageStrategy": result.StorageStrategy,
			"intentAnalysis":  result.IntentAnalysis,
			"qualityScore":    result.QualityScore,
		})
	}

	// 构建增强的响应
	response := &models.StoreMessagesResponse{
		MessageIDs: allMessageIDs,
		Status:     "success",
		Metadata: map[string]interface{}{
			"llm_driven":         true,
			"smart_analysis":     smartAnalysisResults,
			"total_messages":     len(req.Messages),
			"analysis_timestamp": time.Now().Unix(),
		},
	}

	log.Printf("✅ [LLM驱动存储] 智能存储完成: 消息数=%d, 总ID数=%d", len(req.Messages), len(allMessageIDs))
	return response, nil
}

// executeSmartStorage 执行智能存储决策（适配版本）
func (lds *LLMDrivenContextService) executeSmartStorage(ctx context.Context, sessionID, content, priority string) (*SmartStorageResult, error) {
	log.Printf("🧠 [智能存储决策] 开始分析内容: %s", content[:min(50, len(content))])

	// 🔥 直接调用基础服务的智能存储逻辑（包含LLM分析）
	req := models.StoreContextRequest{
		SessionID: sessionID,
		Content:   content,
		Priority:  priority,
		Metadata: map[string]interface{}{
			"source":    "llm_driven_storage",
			"timestamp": time.Now().Unix(),
		},
	}

	// 🔥 使用新的扩展接口获取完整分析结果
	response, err := lds.contextService.StoreContextWithAnalysis(ctx, req)
	if err != nil {
		log.Printf("❌ [智能存储决策] 存储失败: %v", err)
		return nil, fmt.Errorf("智能存储执行失败: %w", err)
	}

	// 构建智能存储结果
	result := &SmartStorageResult{
		MessageIDs:      []string{response.MemoryID},
		Confidence:      response.Confidence,
		StorageStrategy: response.StorageStrategy,
		IntentAnalysis:  content,
		QualityScore:    response.Confidence,
		AnalysisResult:  response.AnalysisResult, // 包含完整的分析结果
	}

	log.Printf("✅ [智能存储决策] 完成，记忆ID: %s, 置信度: %.2f", response.MemoryID, result.Confidence)
	return result, nil
}

// SmartStorageResult 智能存储结果
type SmartStorageResult struct {
	MessageIDs      []string                    `json:"messageIds"`
	Confidence      float64                     `json:"confidence"`
	StorageStrategy string                      `json:"storageStrategy"`
	IntentAnalysis  string                      `json:"intentAnalysis"`
	QualityScore    float64                     `json:"qualityScore"`
	AnalysisResult  *models.SmartAnalysisResult `json:"analysisResult,omitempty"` // 完整的分析结果
}

// GetSessionState 获取会话状态
func (lds *LLMDrivenContextService) GetSessionState(ctx context.Context, sessionID string) (*models.MCPSessionResponse, error) {
	return lds.contextService.GetSessionState(ctx, sessionID)
}

// IsMultiDimensionalEnabled 检查多维度存储是否启用
func (lds *LLMDrivenContextService) IsMultiDimensionalEnabled() bool {
	return lds.enabled && lds.multiRetriever != nil
}

// GetMultiDimensionalEngine 获取多维度检索引擎（用于MCP工具）
func (lds *LLMDrivenContextService) GetMultiDimensionalEngine() interface{} {
	if lds.multiRetriever != nil {
		return lds.multiRetriever
	}
	return nil
}

// SearchContext 搜索上下文
func (lds *LLMDrivenContextService) SearchContext(ctx context.Context, sessionID, query string) ([]string, error) {
	return lds.contextService.SearchContext(ctx, sessionID, query)
}

// AssociateFile 关联文件
func (lds *LLMDrivenContextService) AssociateFile(ctx context.Context, req models.AssociateFileRequest) error {
	return lds.contextService.AssociateFile(ctx, req)
}

// RecordEdit 记录编辑
func (lds *LLMDrivenContextService) RecordEdit(ctx context.Context, req models.RecordEditRequest) error {
	return lds.contextService.RecordEdit(ctx, req)
}

// GetUserIDFromSessionID 从会话ID获取用户ID
func (lds *LLMDrivenContextService) GetUserIDFromSessionID(sessionID string) (string, error) {
	return lds.contextService.GetUserIDFromSessionID(sessionID)
}

// ============================================================================
// 🔧 存储引擎适配器 - 快速修复检索链路
// ============================================================================

// TimelineStoreAdapter 时间线存储适配器
type TimelineStoreAdapter struct {
	Engine interface{}
}

func (adapter *TimelineStoreAdapter) SearchByQuery(ctx context.Context, req *models.TimelineSearchRequest) ([]*models.TimelineEvent, error) {
	// 🔥 优先使用请求对象中的字段，Context作为备用
	userID := req.UserID
	if userID == "" {
		userID, _ = ctx.Value("user_id").(string)
	}

	workspaceID := req.WorkspaceID
	if workspaceID == "" {
		workspaceID, _ = ctx.Value("workspacePath").(string)
	}

	log.Printf("🔍 [时间线适配器] 执行查询: %s, 用户: %s, 限制: %d", req.Query, userID, req.Limit)
	log.Printf("🔍 [时间线适配器] LLM关键概念: %v", req.KeyConcepts)
	log.Printf("🔍 [时间线适配器] 引擎状态: %T", adapter.Engine)

	// 快速实现：返回空结果但记录详细信息，避免nil panic
	if adapter.Engine == nil {
		log.Printf("⚠️ [时间线适配器] 引擎为nil，返回空结果")
		return []*models.TimelineEvent{}, nil
	}

	// 🔥 修复：实现真实的TimescaleDB查询适配
	// 使用正确的引擎接口进行类型断言
	if timelineEngine, ok := adapter.Engine.(*timeline.TimescaleDBEngine); ok {
		log.Printf("🔧 [时间线适配器] 检测到TimescaleDB引擎，构建查询参数")

		// 🔥 修复：使用正确的TimelineQuery结构
		timelineQuery := &timeline.TimelineQuery{
			UserID:      userID,
			SearchText:  req.Query,
			Keywords:    req.KeyConcepts, // 🔥 关键修复：使用LLM分析的关键概念
			Limit:       req.Limit,
			WorkspaceID: utils.ExtractWorkspaceNameFromPath(workspaceID), // 🔥 修复：使用公共工具函数
			OrderBy:     "timestamp",
			// 设置最小相关性过滤
			MinRelevance: 0.1,
		}

		// 🔥 关键修复：正确处理时间范围参数
		if req.StartTime != nil && req.EndTime != nil {
			timelineQuery.StartTime = *req.StartTime
			timelineQuery.EndTime = *req.EndTime
			log.Printf("🕒 [时间线适配器] 使用直接时间范围: %s - %s",
				req.StartTime.Format("2006-01-02 15:04:05"),
				req.EndTime.Format("2006-01-02 15:04:05"))
		} else {
			// 没有指定时间范围时才使用默认的30天窗口
			timelineQuery.TimeWindow = "30 days"
			log.Printf("⏰ [时间线适配器] 使用默认时间窗口: 30 days")
		}

		log.Printf("📥 [时间线适配器] 查询参数: %+v", timelineQuery)

		// 🔥 使用正确的引擎接口调用
		result, err := timelineEngine.RetrieveEvents(ctx, timelineQuery)
		if err != nil {
			log.Printf("❌ [时间线适配器] 查询失败: %v", err)
			return []*models.TimelineEvent{}, nil // 返回空结果而不是错误，保持检索链路稳定
		}

		// 🔥 修复：转换TimelineResult到models.TimelineEvent
		events := convertTimelineResultToEvents(result)
		log.Printf("✅ [时间线适配器] 查询成功，获得%d个结果", len(events))
		return events, nil

	} else {
		log.Printf("⚠️ [时间线适配器] 引擎类型不匹配: %T，返回空结果", adapter.Engine)
		return []*models.TimelineEvent{}, nil
	}
}

// 🆕 checkTimelineRecallQuery 检查是否为时间回忆查询
func (lds *LLMDrivenContextService) checkTimelineRecallQuery(analysisResult *engines.SemanticAnalysisResult) bool {
	// 检查语料分析结果中是否包含TimelineRecall字段
	if analysisResult != nil && analysisResult.SmartAnalysis != nil && analysisResult.SmartAnalysis.TimelineRecall != nil {
		startTime := analysisResult.SmartAnalysis.TimelineRecall.StartTime
		endTime := analysisResult.SmartAnalysis.TimelineRecall.EndTime
		if startTime != "" && endTime != "" {
			log.Printf("✅ [时间回忆检测] 发现时间回忆查询: %s - %s", startTime, endTime)
			return true
		}
	}
	return false
}

// 🆕 handleTimelineRecallQuery 处理时间回忆查询
func (lds *LLMDrivenContextService) handleTimelineRecallQuery(ctx context.Context, req models.RetrieveContextRequest, analysisResult *engines.SemanticAnalysisResult) (models.ContextResponse, error) {
	timelineRecall := analysisResult.SmartAnalysis.TimelineRecall

	// 解析时间范围
	startTime, err := time.Parse("2006-01-02 15:04:05", timelineRecall.StartTime)
	if err != nil {
		log.Printf("❌ [时间回忆] 时间解析失败 StartTime: %s, error: %v", timelineRecall.StartTime, err)
		return models.ContextResponse{}, fmt.Errorf("时间解析失败: %w", err)
	}

	endTime, err := time.Parse("2006-01-02 15:04:05", timelineRecall.EndTime)
	if err != nil {
		log.Printf("❌ [时间回忆] 时间解析失败 EndTime: %s, error: %v", timelineRecall.EndTime, err)
		return models.ContextResponse{}, fmt.Errorf("时间解析失败: %w", err)
	}

	log.Printf("🔍 [时间回忆] 查询时间范围: %s 到 %s", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// 🔥 从统一拦截器注入的context中直接获取用户ID和工作空间ID
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		log.Printf("❌ [时间回忆] 从context获取用户ID失败，统一拦截器可能未生效")
		return models.ContextResponse{}, fmt.Errorf("获取用户ID失败：context中缺少user_id")
	}
	log.Printf("✅ [时间回忆] 从context获取用户ID: %s", userID)

	workspacePath, ok := ctx.Value("workspacePath").(string)
	if !ok || workspacePath == "" {
		log.Printf("❌ [时间回忆] 从context获取工作空间路径失败，统一拦截器可能未生效")
		return models.ContextResponse{}, fmt.Errorf("获取工作空间失败：context中缺少workspacePath")
	}

	// 从工作空间路径提取workspace名称
	workspaceID := utils.ExtractWorkspaceNameFromPath(workspacePath)
	log.Printf("✅ [时间回忆] 从context获取工作空间: path=%s, id=%s", workspacePath, workspaceID)

	// 🔥 纯时间线查询：用户ID + 工作空间 + 时间范围，不使用关键词过滤
	timelineQuery := &timeline.TimelineQuery{
		UserID:      userID,      // 🔥 从上下文正确获取用户ID
		WorkspaceID: workspaceID, // 🔥 正确提取的workspace ID
		StartTime:   startTime,
		EndTime:     endTime,
		OrderBy:     "timestamp DESC, importance_score DESC",
		Limit:       20,
		// 🔥 关键：不设置Keywords和SearchText，纯时间范围查询
	}

	// 直接查询时间线引擎
	if lds.multiRetriever != nil {
		events, err := lds.queryTimelineDirectly(ctx, timelineQuery)
		if err != nil {
			log.Printf("❌ [时间回忆] 时间线查询失败: %v", err)
			return models.ContextResponse{}, fmt.Errorf("时间线查询失败: %w", err)
		}

		log.Printf("✅ [时间回忆] 查询成功，获得 %d 个时间线事件", len(events))

		// 🔥 格式化时间线数据为精简的JSON格式，只包含必要字段
		var timelineData []map[string]interface{}
		for _, event := range events {
			// 只返回必要的字段：title, content, summary, related_files, related_concepts,
			// parent_event_id, intent, keywords, relevance_score, created_at
			eventData := map[string]interface{}{
				"title":            event.Title,
				"content":          event.Content,
				"summary":          event.Summary,
				"related_files":    event.RelatedFiles,
				"related_concepts": event.RelatedConcepts,
				"parent_event_id":  event.ParentEventID,
				"intent":           event.Intent,
				"keywords":         event.Keywords,
				"relevance_score":  event.RelevanceScore,
				"created_at":       event.CreatedAt,
			}
			timelineData = append(timelineData, eventData)
		}

		// 将时间线数据序列化为JSON字符串
		timelineJSON, err := json.MarshalIndent(timelineData, "", "  ")
		if err != nil {
			log.Printf("❌ [时间回忆] JSON序列化失败: %v", err)
			return models.ContextResponse{}, fmt.Errorf("数据格式化失败: %w", err)
		}

		log.Printf("✅ [时间回忆] 时间线数据已格式化为JSON，共%d条记录", len(timelineData))

		// 🔥 使用原有的ContextResponse结构，将时间线数据填充到LongTermMemory字段
		return models.ContextResponse{
			SessionState:      "",
			ShortTermMemory:   "暂无",
			LongTermMemory:    string(timelineJSON), // 🔥 关键：时间线数据填充到LongTermMemory
			RelevantKnowledge: "暂无",
		}, nil
	}

	return models.ContextResponse{}, fmt.Errorf("时间线适配器不可用")
}

// 🆕 queryTimelineDirectly 直接查询时间线（专用于时间回忆）
func (lds *LLMDrivenContextService) queryTimelineDirectly(ctx context.Context, query *timeline.TimelineQuery) ([]*models.TimelineEvent, error) {
	log.Printf("📥 [时间回忆直查] 查询参数: UserID=%s, WorkspaceID=%s, 时间范围=%s到%s",
		query.UserID, query.WorkspaceID,
		query.StartTime.Format("2006-01-02 15:04:05"),
		query.EndTime.Format("2006-01-02 15:04:05"))

	// 🔥 直接调用多维检索器的DirectTimelineQuery方法
	events, err := lds.multiRetriever.DirectTimelineQuery(ctx, &models.TimelineSearchRequest{
		UserID:      query.UserID,
		WorkspaceID: query.WorkspaceID,
		Query:       "", // 🔥 空查询，纯时间范围过滤
		Limit:       query.Limit,
		StartTime:   &query.StartTime, // 🔥 关键：传递时间范围
		EndTime:     &query.EndTime,   // 🔥 关键：传递时间范围
	})

	if err != nil {
		log.Printf("❌ [时间回忆直查] 直接查询失败: %v", err)
		return nil, err
	}

	log.Printf("✅ [时间回忆直查] 查询成功，返回 %d 个事件", len(events))
	return events, nil
}

// extractWorkspaceFromQuery 从查询中提取工作空间信息
func extractWorkspaceFromQuery(query string) string {
	// 🔥 关键修复：添加项目上下文识别
	if containsKeywords(query, []string{"context-keeper", "Context-Keeper", "上下文", "记忆", "检索"}) {
		return "context-keeper" // 返回项目名称作为工作空间ID
	}
	return "default"
}

// extractTimeWindowFromQuery 从查询中提取时间窗口
func extractTimeWindowFromQuery(query string) string {
	// 🔥 关键修复：添加时间维度识别
	if containsKeywords(query, []string{"昨天", "yesterday"}) {
		return "1 day"
	}
	if containsKeywords(query, []string{"上周", "last week"}) {
		return "1 week"
	}
	if containsKeywords(query, []string{"最近", "recent"}) {
		return "3 days"
	}
	return "" // 不限制时间窗口
}

// containsKeywords 检查查询是否包含关键词
func containsKeywords(query string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(query), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// 🔥 新增：正确的结果转换方法
func convertTimelineResultToEvents(result *timeline.TimelineResult) []*models.TimelineEvent {
	if result == nil || len(result.Events) == 0 {
		log.Printf("⚠️ [时间线转换] 无结果数据")
		return []*models.TimelineEvent{}
	}

	events := make([]*models.TimelineEvent, len(result.Events))
	for i, event := range result.Events {
		events[i] = &models.TimelineEvent{
			ID:          event.ID,
			UserID:      event.UserID,
			SessionID:   event.SessionID,
			WorkspaceID: event.WorkspaceID,
			Timestamp:   event.Timestamp,
			EventType:   event.EventType,
			Title:       event.Title,
			Content:     event.Content,
			Summary:     event.Summary,
			// 转换相关文件和概念
			RelatedFiles:    convertStringArray(event.RelatedFiles),
			RelatedConcepts: convertStringArray(event.RelatedConcepts),
			// 其他字段
			Intent:          event.Intent,
			Keywords:        convertStringArray(event.Keywords),
			Entities:        convertToEntityArray(event.Entities),
			Categories:      convertStringArray(event.Categories),
			ImportanceScore: event.ImportanceScore,
			RelevanceScore:  event.RelevanceScore,
			CreatedAt:       event.CreatedAt,
			UpdatedAt:       event.UpdatedAt,
		}
	}

	log.Printf("✅ [时间线转换] 成功转换 %d 个事件", len(events))
	return events
}

// 辅助方法：转换字符串数组
func convertStringArray(pqArray interface{}) []string {
	if pqArray == nil {
		return []string{}
	}

	// 处理pq.StringArray类型
	if arr, ok := pqArray.([]string); ok {
		return arr
	}

	// 处理其他可能的类型
	return []string{}
}

// 辅助方法：转换实体数组
func convertToEntityArray(entities interface{}) models.EntityArray {
	if entities == nil {
		return models.EntityArray{}
	}

	// 如果已经是models.EntityArray类型
	if arr, ok := entities.(models.EntityArray); ok {
		return arr
	}

	// 如果是timeline.EntityArray类型，需要转换
	if timelineEntities, ok := entities.(timeline.EntityArray); ok {
		result := make(models.EntityArray, len(timelineEntities))
		for i, entity := range timelineEntities {
			result[i] = models.Entity{
				Text:       entity.Text,
				Type:       entity.Type,
				Confidence: entity.Confidence,
			}
		}
		return result
	}

	// 处理其他可能的类型
	return models.EntityArray{}
}

// 🔥 新增：转换知识图谱结果到模型节点
func convertKnowledgeResultToNodes(result *knowledge.KnowledgeResult) []*models.KnowledgeNode {
	if result == nil || len(result.Nodes) == 0 {
		log.Printf("⚠️ [知识图谱转换] 无结果数据")
		return []*models.KnowledgeNode{}
	}

	nodes := make([]*models.KnowledgeNode, len(result.Nodes))
	for i, node := range result.Nodes {
		nodes[i] = &models.KnowledgeNode{
			ID:          node.ID,
			Name:        node.Name,
			Labels:      node.Labels,
			Category:    node.Category,
			Description: node.Description,
			Keywords:    node.Keywords,
			Score:       node.Score,
			// 转换属性
			Properties: convertPropertiesToMap(node.Properties),
		}
	}

	log.Printf("✅ [知识图谱转换] 成功转换 %d 个节点", len(nodes))
	return nodes
}

// 辅助方法：转换属性
func convertPropertiesToMap(properties interface{}) map[string]interface{} {
	if properties == nil {
		return map[string]interface{}{}
	}

	// 如果已经是map类型
	if propMap, ok := properties.(map[string]interface{}); ok {
		return propMap
	}

	// 处理其他可能的类型
	return map[string]interface{}{}
}

// 🔥 完善：转换知识图谱关系为标准格式
func convertRelationshipsToModels(relationships []knowledge.KnowledgeRelationship, nodeID string) []map[string]interface{} {
	var result []map[string]interface{}

	for _, rel := range relationships {
		// 只包含与当前节点相关的关系
		if rel.StartNodeID == nodeID || rel.EndNodeID == nodeID {
			relationship := map[string]interface{}{
				"id":            rel.ID,
				"type":          rel.Type,
				"start_node_id": rel.StartNodeID,
				"end_node_id":   rel.EndNodeID,
				"strength":      rel.Strength,
				"description":   rel.Description,
				"properties":    convertPropertiesToMap(rel.Properties),
				// 添加关系方向指示
				"direction": getRelationshipDirection(rel, nodeID),
				// 添加关系权重评估
				"weight_category": categorizeRelationshipWeight(rel.Strength),
			}
			result = append(result, relationship)
		}
	}

	log.Printf("🔗 [关系转换] 为节点 %s 转换了 %d 个关系", nodeID, len(result))
	return result
}

// 获取关系方向
func getRelationshipDirection(rel knowledge.KnowledgeRelationship, nodeID string) string {
	if rel.StartNodeID == nodeID {
		return "outgoing" // 出度关系
	} else if rel.EndNodeID == nodeID {
		return "incoming" // 入度关系
	}
	return "unknown"
}

// 分类关系权重
func categorizeRelationshipWeight(strength float64) string {
	switch {
	case strength >= 0.8:
		return "strong"
	case strength >= 0.5:
		return "medium"
	case strength >= 0.2:
		return "weak"
	default:
		return "minimal"
	}
}

// 从查询中提取分类
func extractCategoriesFromQuery(query string) []string {
	// 简单的分类提取逻辑，可以后续优化
	categories := []string{}

	// 技术相关关键词
	techKeywords := []string{"代码", "编程", "开发", "架构", "设计", "算法", "数据库", "API", "框架"}
	for _, keyword := range techKeywords {
		if strings.Contains(query, keyword) {
			categories = append(categories, "technology")
			break
		}
	}

	// 业务相关关键词
	businessKeywords := []string{"需求", "业务", "流程", "管理", "产品", "用户"}
	for _, keyword := range businessKeywords {
		if strings.Contains(query, keyword) {
			categories = append(categories, "business")
			break
		}
	}

	// 默认分类
	if len(categories) == 0 {
		categories = append(categories, "general")
	}

	return categories
}

// 从查询中提取关键词
func extractKeywordsFromQuery(query string) []string {
	// 简单的关键词提取，按空格分割并过滤停用词
	words := strings.Fields(query)
	keywords := []string{}

	// 停用词列表
	stopWords := map[string]bool{
		"的": true, "是": true, "和": true, "在": true, "有": true, "这": true, "那": true,
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	}

	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 1 && !stopWords[strings.ToLower(word)] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// 🔥 新增：查询去重和优化，区分不同数据源的查询特点
func deduplicateAndOptimizeQueries(queries []string, queryType string) []string {
	if len(queries) == 0 {
		return queries
	}

	log.Printf("🔄 [查询优化] 处理 %s 类型查询，原始数量: %d", queryType, len(queries))

	// 第一步：基础去重
	seenQueries := make(map[string]bool)
	uniqueQueries := make([]string, 0)

	for _, query := range queries {
		normalizedQuery := strings.TrimSpace(strings.ToLower(query))
		if normalizedQuery != "" && !seenQueries[normalizedQuery] {
			seenQueries[normalizedQuery] = true
			uniqueQueries = append(uniqueQueries, query)
		}
	}

	// 第二步：根据查询类型进行特化优化
	optimizedQueries := optimizeQueriesByType(uniqueQueries, queryType)

	// 第三步：语义去重（移除过于相似的查询）
	finalQueries := semanticDeduplication(optimizedQueries)

	log.Printf("✅ [查询优化] %s 类型查询优化完成: %d -> %d", queryType, len(queries), len(finalQueries))
	return finalQueries
}

// 根据查询类型进行特化优化
func optimizeQueriesByType(queries []string, queryType string) []string {
	optimized := make([]string, 0)

	for _, query := range queries {
		optimizedQuery := ""

		switch queryType {
		case "context":
			// 上下文查询：关注当前会话和近期活动
			optimizedQuery = enhanceContextQuery(query)

		case "timeline":
			// 时间线查询：增加时间维度和顺序性
			optimizedQuery = enhanceTimelineQuery(query)

		case "knowledge":
			// 知识图谱查询：关注概念和关系
			optimizedQuery = enhanceKnowledgeQuery(query)

		case "vector":
			// 向量查询：保持原始语义
			optimizedQuery = enhanceVectorQuery(query)

		default:
			optimizedQuery = query
		}

		if optimizedQuery != "" && optimizedQuery != query {
			log.Printf("🔧 [查询特化] %s: %s -> %s", queryType, query, optimizedQuery)
		}

		optimized = append(optimized, optimizedQuery)
	}

	return optimized
}

// 增强上下文查询
func enhanceContextQuery(query string) string {
	// 上下文查询关注当前会话状态和用户意图
	if !strings.Contains(query, "当前") && !strings.Contains(query, "current") {
		return "当前会话 " + query
	}
	return query
}

// 增强时间线查询
func enhanceTimelineQuery(query string) string {
	// 🔧 修复：不再添加"历史活动"前缀，直接返回原查询
	// 原因：添加前缀导致包含不存在词汇，PostgreSQL AND逻辑查询失败
	return query
}

// 增强知识图谱查询
func enhanceKnowledgeQuery(query string) string {
	// 知识图谱查询关注概念和关系
	if !containsConceptKeywords(query) {
		return "相关概念 " + query
	}
	return query
}

// 增强向量查询
func enhanceVectorQuery(query string) string {
	// 向量查询保持原始语义，用于语义相似性匹配
	return query
}

// 检查是否包含时间关键词
func containsTimeKeywords(query string) bool {
	timeKeywords := []string{"最近", "历史", "之前", "时间", "当时", "过去", "earlier", "recent", "history", "time"}
	queryLower := strings.ToLower(query)
	for _, keyword := range timeKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

// 检查是否包含概念关键词
func containsConceptKeywords(query string) bool {
	conceptKeywords := []string{"概念", "关系", "相关", "类似", "关联", "concept", "related", "similar", "connection"}
	queryLower := strings.ToLower(query)
	for _, keyword := range conceptKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	return false
}

// 语义去重：移除过于相似的查询
func semanticDeduplication(queries []string) []string {
	if len(queries) <= 1 {
		return queries
	}

	deduplicated := make([]string, 0)

	for i, query1 := range queries {
		isDuplicate := false

		for j := 0; j < i; j++ {
			query2 := queries[j]
			// 简单的语义相似性检查：计算词汇重叠度
			similarity := calculateQuerySimilarity(query1, query2)
			if similarity > 0.8 { // 相似度阈值
				log.Printf("🔍 [语义去重] 移除相似查询: '%s' (与 '%s' 相似度: %.2f)", query1, query2, similarity)
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			deduplicated = append(deduplicated, query1)
		}
	}

	return deduplicated
}

// 计算查询相似性
func calculateQuerySimilarity(query1, query2 string) float64 {
	words1 := extractKeywordsFromQuery(query1)
	words2 := extractKeywordsFromQuery(query2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// 计算词汇交集
	intersection := 0
	word2Set := make(map[string]bool)
	for _, word := range words2 {
		word2Set[strings.ToLower(word)] = true
	}

	for _, word := range words1 {
		if word2Set[strings.ToLower(word)] {
			intersection++
		}
	}

	// 计算Jaccard相似性
	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// KnowledgeStoreAdapter 知识图谱存储适配器
type KnowledgeStoreAdapter struct {
	Engine interface{}
}

func (adapter *KnowledgeStoreAdapter) SearchByQuery(ctx context.Context, query string, limit int) ([]*models.KnowledgeNode, error) {
	// 🔥 从Context获取基础信息，而非参数传递
	userID, _ := ctx.Value("user_id").(string)

	log.Printf("🔍 [知识图谱适配器] 执行查询: %s, 用户: %s, 限制: %d", query, userID, limit)
	log.Printf("🔍 [知识图谱适配器] 引擎状态: %T", adapter.Engine)

	// 快速实现：返回空结果但记录详细信息，避免nil panic
	if adapter.Engine == nil {
		log.Printf("⚠️ [知识图谱适配器] 引擎为nil，返回空结果")
		return []*models.KnowledgeNode{}, nil
	}

	// 🔥 修复：实现真实的Neo4j查询适配
	if knowledgeEngine, ok := adapter.Engine.(*knowledge.Neo4jEngine); ok {
		log.Printf("🔧 [知识图谱适配器] 检测到Neo4j引擎，构建查询参数")

		// 🔥 修复：使用正确的KnowledgeQuery结构
		knowledgeQuery := &knowledge.KnowledgeQuery{
			UserID:     userID,
			SearchText: query,
			Limit:      limit,
			// 设置查询类型和范围
			QueryType:   "search", // 搜索查询
			Categories:  extractCategoriesFromQuery(query),
			Keywords:    extractKeywordsFromQuery(query),
			MaxDepth:    3,   // 最大深度3层
			MinStrength: 0.1, // 最小关系强度
		}

		log.Printf("📥 [知识图谱适配器] 查询参数: %+v", knowledgeQuery)

		// 🔥 使用正确的引擎接口调用
		result, err := knowledgeEngine.ExpandKnowledge(ctx, knowledgeQuery)
		if err != nil {
			log.Printf("❌ [知识图谱适配器] 查询失败: %v", err)
			return []*models.KnowledgeNode{}, nil // 返回空结果而不是错误，保持检索链路稳定
		}

		// 🔥 修复：转换KnowledgeResult到models.KnowledgeNode
		nodes := convertKnowledgeResultToNodes(result)
		log.Printf("✅ [知识图谱适配器] 查询成功，获得%d个结果", len(nodes))
		return nodes, nil

	} else {
		log.Printf("⚠️ [知识图谱适配器] 引擎类型不匹配: %T，返回空结果", adapter.Engine)
		return []*models.KnowledgeNode{}, nil
	}
}

// VectorStoreAdapter 向量存储适配器
type VectorStoreAdapter struct {
	Engine interface{}
}

func (adapter *VectorStoreAdapter) SearchByQuery(ctx context.Context, query string, limit int) ([]*models.VectorMatch, error) {
	// 🔥 从Context获取基础信息，而非参数传递
	userID, _ := ctx.Value("user_id").(string)

	log.Printf("🔍 [向量适配器] 执行查询: %s, 用户: %s, 限制: %d", query, userID, limit)
	log.Printf("🔍 [向量适配器] 引擎状态: %T", adapter.Engine)

	// 快速实现：返回空结果但记录详细信息，避免nil panic
	if adapter.Engine == nil {
		log.Printf("⚠️ [向量适配器] 引擎为nil，返回空结果")
		return []*models.VectorMatch{}, nil
	}

	// 🔥 修复：使用统一的类型断言模式
	if vectorStore, ok := adapter.Engine.(models.VectorStore); ok {
		log.Printf("🔧 [向量适配器] 检测到VectorStore引擎，构建查询参数")

		// 构建SearchOptions
		options := &models.SearchOptions{
			Limit:  limit,
			UserID: userID,
		}

		log.Printf("📥 [向量适配器] 查询参数: %+v", options)

		// 🔥 使用正确的接口调用vectorStore.SearchByText
		results, err := vectorStore.SearchByText(ctx, query, options)
		if err != nil {
			log.Printf("❌ [向量适配器] 查询失败: %v", err)
			return []*models.VectorMatch{}, nil // 返回空结果而不是错误，保持检索链路稳定
		}

		// 🔥 转换SearchResult[]到VectorMatch[]
		matches := convertSearchResultsToVectorMatches(results)
		log.Printf("✅ [向量适配器] 查询成功，获得%d个结果", len(matches))
		return matches, nil

	} else {
		log.Printf("⚠️ [向量适配器] 引擎类型不匹配: %T，返回空结果", adapter.Engine)
		return []*models.VectorMatch{}, nil
	}
}

// SetEngine 设置向量存储的Engine（用于延迟赋值）
func (adapter *VectorStoreAdapter) SetEngine(engine interface{}) {
	adapter.Engine = engine
	log.Printf("✅ [向量适配器] Engine已设置: %T", engine)
}

// convertSearchResultsToVectorMatches 转换SearchResult到VectorMatch
func convertSearchResultsToVectorMatches(results []models.SearchResult) []*models.VectorMatch {
	matches := make([]*models.VectorMatch, 0, len(results))

	for _, result := range results {
		match := &models.VectorMatch{
			ID:    result.ID,
			Score: result.Score,
		}

		// 从Fields中提取Content和Metadata
		if result.Fields != nil {
			if content, ok := result.Fields["content"].(string); ok {
				match.Content = content
			}
			if title, ok := result.Fields["title"].(string); ok {
				match.Title = title
			}
			// 其他字段作为Metadata
			match.Metadata = make(map[string]interface{})
			for k, v := range result.Fields {
				if k != "content" && k != "title" {
					match.Metadata[k] = v
				}
			}
		}

		matches = append(matches, match)
	}

	log.Printf("🔄 [结果转换] 转换了 %d 个SearchResult到VectorMatch", len(matches))
	return matches
}

// buildEnhancedQuery 构建包含项目上下文的增强查询
func buildEnhancedQuery(originalQuery, userID string) string {
	// 🔥 关键修复：在查询中添加项目上下文信息
	projectContext := "context-keeper项目 "

	// 如果查询中已经包含项目信息，则不重复添加
	if containsKeywords(originalQuery, []string{"context-keeper", "Context-Keeper"}) {
		return originalQuery
	}

	// 添加项目上下文
	return projectContext + originalQuery
}

// buildProjectContextFilter 构建项目上下文过滤器
func buildProjectContextFilter(userID, query string) string {
	// 🔥 关键修复：构建包含用户和项目信息的过滤器
	var filterParts []string

	// 用户过滤（必须）
	if userID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`userId="%s"`, userID))
	}

	// 项目上下文过滤（如果查询涉及特定项目）
	if containsKeywords(query, []string{"context-keeper", "Context-Keeper", "上下文", "记忆"}) {
		// 可以添加项目相关的过滤条件，比如workspace_id或project_name
		// filterParts = append(filterParts, `project="context-keeper"`)
	}

	if len(filterParts) > 0 {
		return strings.Join(filterParts, " AND ")
	}

	return ""
}

// convertToVectorMatches 转换搜索结果为向量匹配格式
func convertToVectorMatches(results []models.SearchResult) []*models.VectorMatch {
	var matches []*models.VectorMatch

	for _, result := range results {
		// 🔥 修复：从Fields中提取内容，因为SearchResult的内容在Fields中
		content := ""
		title := ""
		if result.Fields != nil {
			if c, ok := result.Fields["content"].(string); ok {
				content = c
			}
			if t, ok := result.Fields["title"].(string); ok {
				title = t
			}
		}

		match := &models.VectorMatch{
			ID:      result.ID,
			Content: content,
			Title:   title,
			Score:   result.Score,
			// 可以添加更多字段映射
			Metadata: result.Fields, // 保留原始字段信息
		}
		matches = append(matches, match)
	}

	log.Printf("🔄 [向量适配器] 转换了%d个搜索结果为向量匹配", len(matches))
	return matches
}

// GetUserSessionStore 获取用户会话存储
func (lds *LLMDrivenContextService) GetUserSessionStore(userID string) (*store.SessionStore, error) {
	return lds.contextService.GetUserSessionStore(userID)
}

// SessionStore 返回会话存储实例
func (lds *LLMDrivenContextService) SessionStore() *store.SessionStore {
	return lds.contextService.SessionStore()
}

// SummarizeToLongTermMemory 总结到长期记忆
func (lds *LLMDrivenContextService) SummarizeToLongTermMemory(ctx context.Context, req models.SummarizeToLongTermRequest) (string, error) {
	return lds.contextService.SummarizeToLongTermMemory(ctx, req)
}

// RetrieveTodos 获取待办事项
func (lds *LLMDrivenContextService) RetrieveTodos(ctx context.Context, req models.RetrieveTodosRequest) (*models.RetrieveTodosResponse, error) {
	return lds.contextService.RetrieveTodos(ctx, req)
}

// StartSessionCleanupTask 启动会话清理任务（代理到底层ContextService）
func (lds *LLMDrivenContextService) StartSessionCleanupTask(ctx context.Context, timeout time.Duration, interval time.Duration) {
	lds.contextService.StartSessionCleanupTask(ctx, timeout, interval)
}

// 运行时控制接口
func (lds *LLMDrivenContextService) EnableLLMDriven(enabled bool) {
	lds.enabled = enabled
	if enabled {
		log.Printf("✅ [LLM驱动服务] LLM驱动功能已启用")
	} else {
		log.Printf("⚪ [LLM驱动服务] LLM驱动功能已禁用，将使用基础ContextService")
	}
}

// GetMetrics 获取监控指标
func (lds *LLMDrivenContextService) GetMetrics() *LLMDrivenMetrics {
	return lds.metrics
}

// GetStatus 获取服务状态
func (lds *LLMDrivenContextService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":             lds.enabled,
		"semantic_analysis":   lds.config.SemanticAnalysis,
		"multi_dimensional":   lds.config.MultiDimensional,
		"content_synthesis":   lds.config.ContentSynthesis,
		"total_requests":      lds.metrics.TotalRequests,
		"llm_driven_requests": lds.metrics.LLMDrivenRequests,
		"fallback_requests":   lds.metrics.FallbackRequests,
		"success_rate":        lds.metrics.SuccessRate,
		"error_count":         lds.metrics.ErrorCount,
		"last_updated":        lds.metrics.LastUpdated,
	}
}

// updateMetrics 更新性能指标
func (lds *LLMDrivenContextService) updateMetrics(latency time.Duration, success bool) {
	// 更新平均延迟
	if lds.metrics.LLMDrivenRequests > 0 {
		lds.metrics.AverageLatency = (lds.metrics.AverageLatency*time.Duration(lds.metrics.LLMDrivenRequests-1) + latency) / time.Duration(lds.metrics.LLMDrivenRequests)
	} else {
		lds.metrics.AverageLatency = latency
	}

	// 更新成功率
	if success && lds.metrics.LLMDrivenRequests > 0 {
		lds.metrics.SuccessRate = float64(lds.metrics.LLMDrivenRequests-lds.metrics.ErrorCount) / float64(lds.metrics.LLMDrivenRequests)
	}

	lds.metrics.LastUpdated = time.Now()
}
