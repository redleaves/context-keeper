package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// =============================================================================
// 🎯 统一语义相似度服务 - 策略模式设计
// =============================================================================

// SimilarityRequest 相似度计算请求
type SimilarityRequest struct {
	Text1   string                 `json:"text1"`
	Text2   string                 `json:"text2"`
	Context string                 `json:"context,omitempty"` // 上下文信息，帮助语义理解
	Options map[string]interface{} `json:"options,omitempty"`
}

// SimilarityResponse 相似度计算响应
type SimilarityResponse struct {
	Similarity     float64                `json:"similarity"`      // 主要相似度得分 [0,1]
	Method         string                 `json:"method"`          // 使用的计算方法
	Model          string                 `json:"model,omitempty"` // 使用的模型
	ProcessingTime time.Duration          `json:"processing_time"` // 处理时间
	Confidence     float64                `json:"confidence"`      // 置信度 [0,1]
	Details        SimilarityDetails      `json:"details"`         // 详细指标
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SimilarityDetails 详细相似度指标
type SimilarityDetails struct {
	SemanticSimilarity   float64 `json:"semantic_similarity"`   // 语义相似度
	LexicalSimilarity    float64 `json:"lexical_similarity"`    // 词汇相似度
	StructuralSimilarity float64 `json:"structural_similarity"` // 结构相似度
	IntentSimilarity     float64 `json:"intent_similarity"`     // 意图相似度
	DomainRelevance      float64 `json:"domain_relevance"`      // 领域相关性
	QualityScore         float64 `json:"quality_score"`         // 综合质量评分
}

// =============================================================================
// 🔮 策略接口定义
// =============================================================================

// SimilarityStrategy 相似度计算策略接口
type SimilarityStrategy interface {
	// Name 返回策略名称
	Name() string

	// IsAvailable 检查策略是否可用
	IsAvailable(ctx context.Context) bool

	// CalculateSimilarity 计算相似度
	CalculateSimilarity(ctx context.Context, req *SimilarityRequest) (*SimilarityResponse, error)

	// GetCapabilities 获取策略能力描述
	GetCapabilities() StrategyCapabilities
}

// StrategyCapabilities 策略能力描述
type StrategyCapabilities struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Speed        string   `json:"speed"`        // fast, medium, slow
	Accuracy     string   `json:"accuracy"`     // low, medium, high, very_high
	Languages    []string `json:"languages"`    // 支持的语言
	Offline      bool     `json:"offline"`      // 是否支持离线
	MaxLength    int      `json:"max_length"`   // 最大文本长度
	Cost         string   `json:"cost"`         // free, low, medium, high
	Dependencies []string `json:"dependencies"` // 外部依赖
}

// =============================================================================
// 🎯 统一相似度服务
// =============================================================================

// UnifiedSimilarityService 统一语义相似度服务
type UnifiedSimilarityService struct {
	strategies       map[string]SimilarityStrategy
	defaultStrategy  string
	fallbackStrategy string
	config           *SimilarityConfig
}

// SimilarityConfig 相似度服务配置
type SimilarityConfig struct {
	DefaultStrategy   string                 `json:"default_strategy"`
	FallbackStrategy  string                 `json:"fallback_strategy"`
	MaxRetries        int                    `json:"max_retries"`
	Timeout           time.Duration          `json:"timeout"`
	EnableFallback    bool                   `json:"enable_fallback"`
	StrategySelection map[string]interface{} `json:"strategy_selection"` // 策略选择规则
	PerformanceTarget PerformanceTarget      `json:"performance_target"`
}

// PerformanceTarget 性能目标
type PerformanceTarget struct {
	MaxLatency    time.Duration `json:"max_latency"`    // 最大延迟
	MinAccuracy   float64       `json:"min_accuracy"`   // 最小准确度要求
	PreferOffline bool          `json:"prefer_offline"` // 优先离线计算
}

// NewUnifiedSimilarityService 创建统一相似度服务
func NewUnifiedSimilarityService(config *SimilarityConfig) *UnifiedSimilarityService {
	if config == nil {
		config = &SimilarityConfig{
			DefaultStrategy:  "enhanced_local",
			FallbackStrategy: "basic_local",
			MaxRetries:       3,
			Timeout:          30 * time.Second,
			EnableFallback:   true,
			PerformanceTarget: PerformanceTarget{
				MaxLatency:    500 * time.Millisecond,
				MinAccuracy:   0.7,
				PreferOffline: true,
			},
		}
	}

	service := &UnifiedSimilarityService{
		strategies:       make(map[string]SimilarityStrategy),
		defaultStrategy:  config.DefaultStrategy,
		fallbackStrategy: config.FallbackStrategy,
		config:           config,
	}

	// 注册所有可用策略
	service.registerStrategies()

	return service
}

// registerStrategies 注册所有策略实现
func (s *UnifiedSimilarityService) registerStrategies() {
	// 1. 本地增强策略（当前使用的修复版Jaccard）
	s.RegisterStrategy("enhanced_local", NewEnhancedLocalStrategy())

	// 2. 基础本地策略（简单Jaccard）
	s.RegisterStrategy("basic_local", NewBasicLocalStrategy())

	// 3. FastEmbed本地策略（需要ONNX Runtime）
	s.RegisterStrategy("fastembed_local", NewFastEmbedStrategy())

	// 4. HuggingFace在线策略（需要API Token）
	s.RegisterStrategy("huggingface_online", NewHuggingFaceStrategy())
}

// RegisterStrategy 注册策略
func (s *UnifiedSimilarityService) RegisterStrategy(name string, strategy SimilarityStrategy) {
	s.strategies[name] = strategy
	log.Printf("[相似度服务] 注册策略: %s", name)
}

// GetAvailableStrategies 获取可用策略列表
func (s *UnifiedSimilarityService) GetAvailableStrategies(ctx context.Context) map[string]StrategyCapabilities {
	available := make(map[string]StrategyCapabilities)

	for name, strategy := range s.strategies {
		if strategy.IsAvailable(ctx) {
			available[name] = strategy.GetCapabilities()
		}
	}

	return available
}

// CalculateSimilarity 计算语义相似度
func (s *UnifiedSimilarityService) CalculateSimilarity(ctx context.Context, req *SimilarityRequest) (*SimilarityResponse, error) {
	startTime := time.Now()

	// 日志：记录请求基本信息
	log.Printf("🎯 [相似度服务] 开始计算相似度")
	log.Printf("📝 [相似度服务] 文本1: '%s' (长度: %d)", truncateText(req.Text1, 50), len(req.Text1))
	log.Printf("📝 [相似度服务] 文本2: '%s' (长度: %d)", truncateText(req.Text2, 50), len(req.Text2))
	if req.Context != "" {
		log.Printf("🔍 [相似度服务] 上下文: %s", req.Context)
	}

	// 1. 选择最佳策略 - 支持强制指定策略
	var strategyName string
	if req.Options != nil {
		if forceStrategy, ok := req.Options["force_strategy"].(string); ok && forceStrategy != "" {
			log.Printf("🔧 [相似度服务] 强制使用策略: %s", forceStrategy)
			strategyName = forceStrategy
		} else {
			strategyName = s.selectBestStrategy(ctx, req)
			log.Printf("🤖 [相似度服务] 智能选择策略: %s", strategyName)
		}
	} else {
		strategyName = s.selectBestStrategy(ctx, req)
		log.Printf("🤖 [相似度服务] 智能选择策略: %s", strategyName)
	}

	strategy, exists := s.strategies[strategyName]
	if !exists {
		log.Printf("❌ [相似度服务] 策略不存在: %s", strategyName)
		return nil, fmt.Errorf("策略不存在: %s", strategyName)
	}

	// 获取策略能力信息
	capabilities := strategy.GetCapabilities()
	log.Printf("📊 [相似度服务] 策略能力: %s (速度:%s, 精度:%s, 离线:%v)",
		capabilities.Name, capabilities.Speed, capabilities.Accuracy, capabilities.Offline)

	// 2. 检查策略可用性
	if !strategy.IsAvailable(ctx) {
		log.Printf("⚠️ [相似度服务] 策略 %s 不可用", strategyName)
		if s.config.EnableFallback && s.fallbackStrategy != strategyName {
			log.Printf("🔄 [相似度服务] 策略 %s 不可用，降级到 %s", strategyName, s.fallbackStrategy)
			strategyName = s.fallbackStrategy
			strategy = s.strategies[strategyName]

			if strategy == nil || !strategy.IsAvailable(ctx) {
				log.Printf("❌ [相似度服务] 所有策略都不可用")
				return nil, fmt.Errorf("所有策略都不可用")
			}
			// 重新获取降级策略的能力信息
			capabilities = strategy.GetCapabilities()
			log.Printf("📊 [相似度服务] 降级策略能力: %s (速度:%s, 精度:%s, 离线:%v)",
				capabilities.Name, capabilities.Speed, capabilities.Accuracy, capabilities.Offline)
		} else {
			log.Printf("❌ [相似度服务] 策略不可用且未启用降级: %s", strategyName)
			return nil, fmt.Errorf("策略不可用: %s", strategyName)
		}
	} else {
		log.Printf("✅ [相似度服务] 策略 %s 可用，开始执行计算", strategyName)
	}

	// 3. 执行计算
	log.Printf("🚀 [相似度服务] 调用策略 %s 进行相似度计算", strategyName)
	strategyStartTime := time.Now()

	result, err := strategy.CalculateSimilarity(ctx, req)
	strategyExecutionTime := time.Since(strategyStartTime)

	if err != nil {
		log.Printf("❌ [相似度服务] 策略 %s 执行失败: %v (耗时: %v)", strategyName, err, strategyExecutionTime)

		// 重试机制
		if s.config.EnableFallback && s.fallbackStrategy != strategyName {
			log.Printf("🔄 [相似度服务] 策略 %s 执行失败，降级到 %s: %v", strategyName, s.fallbackStrategy, err)
			fallbackStrategy := s.strategies[s.fallbackStrategy]
			if fallbackStrategy != nil && fallbackStrategy.IsAvailable(ctx) {
				log.Printf("🚀 [相似度服务] 使用降级策略 %s 重新计算", s.fallbackStrategy)
				fallbackStartTime := time.Now()

				result, err = fallbackStrategy.CalculateSimilarity(ctx, req)
				fallbackExecutionTime := time.Since(fallbackStartTime)

				if err == nil {
					result.Method = s.fallbackStrategy + "_fallback"
					log.Printf("✅ [相似度服务] 降级策略 %s 执行成功 (耗时: %v)", s.fallbackStrategy, fallbackExecutionTime)
				} else {
					log.Printf("❌ [相似度服务] 降级策略 %s 也失败: %v (耗时: %v)", s.fallbackStrategy, err, fallbackExecutionTime)
				}
			}
		}

		if err != nil {
			log.Printf("💥 [相似度服务] 所有策略都失败，返回错误")
			return nil, fmt.Errorf("相似度计算失败: %v", err)
		}
	} else {
		log.Printf("✅ [相似度服务] 策略 %s 执行成功 (耗时: %v)", strategyName, strategyExecutionTime)
		log.Printf("📈 [相似度服务] 相似度结果: %.4f (置信度: %.4f, 方法: %s)",
			result.Similarity, result.Confidence, result.Method)
	}

	// 4. 添加服务级别的元数据
	totalProcessingTime := time.Since(startTime)
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["service_processing_time"] = totalProcessingTime
	result.Metadata["strategy_selected"] = strategyName
	result.Metadata["request_id"] = generateRequestID()
	result.Metadata["strategy_execution_time"] = strategyExecutionTime

	// 日志：输出最终结果
	log.Printf("🎉 [相似度服务] 计算完成! 最终相似度: %.4f", result.Similarity)
	log.Printf("📊 [相似度服务] 详细指标: 语义相似度=%.4f, 质量分数=%.4f",
		result.Details.SemanticSimilarity, result.Details.QualityScore)
	log.Printf("⏱️ [相似度服务] 总耗时: %v (策略执行: %v)", totalProcessingTime, strategyExecutionTime)

	return result, nil
}

// selectBestStrategy 选择最佳策略
func (s *UnifiedSimilarityService) selectBestStrategy(ctx context.Context, req *SimilarityRequest) string {
	// 🔥 优化策略选择逻辑：优先使用高精度语义理解策略
	textLength := len(req.Text1) + len(req.Text2)

	// 1. 🎯 优先级1：FastEmbed本地策略（最高精度，支持真正语义理解）
	//    适用于大部分场景，特别是中英文同义词识别
	if strategy, exists := s.strategies["fastembed_local"]; exists && strategy.IsAvailable(ctx) {
		log.Printf("🚀 [策略选择] 优先使用FastEmbed策略 (文本长度: %d)", textLength)
		return "fastembed_local"
	}

	// 2. 🌐 优先级2：HuggingFace在线策略（超长文本或复杂语义）
	if textLength >= 500 || s.isComplexSemantics(req) {
		if strategy, exists := s.strategies["huggingface_online"]; exists && strategy.IsAvailable(ctx) {
			log.Printf("🌐 [策略选择] 使用HuggingFace在线策略 (文本长度: %d, 复杂语义: %v)", textLength, s.isComplexSemantics(req))
			return "huggingface_online"
		}
	}

	// 3. 🔧 优先级3：增强本地策略（FastEmbed不可用时的高质量备选）
	if textLength < 500 {
		log.Printf("🔧 [策略选择] 使用增强本地策略 (文本长度: %d)", textLength)
		return "enhanced_local"
	}

	// 4. ⚡ 优先级4：基础本地策略（最后兜底）
	log.Printf("⚡ [策略选择] 降级到默认策略: %s", s.defaultStrategy)
	return s.defaultStrategy
}

// isComplexSemantics 判断是否为复杂语义
func (s *UnifiedSimilarityService) isComplexSemantics(req *SimilarityRequest) bool {
	// 简单的复杂度判断
	complexIndicators := []string{
		"技术", "算法", "架构", "设计模式", "数据库", "API",
		"machine learning", "artificial intelligence", "deep learning",
		"microservices", "distributed", "scalability",
	}

	text := strings.ToLower(req.Text1 + " " + req.Text2)
	for _, indicator := range complexIndicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}

	return false
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("sim_%d", time.Now().UnixNano())
}

// truncateText 截断文本用于日志显示
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// =============================================================================
// 🔧 便利方法
// =============================================================================

// QuickSimilarity 快速相似度计算（使用默认配置）
func (s *UnifiedSimilarityService) QuickSimilarity(text1, text2 string) (float64, error) {
	ctx := context.Background()
	req := &SimilarityRequest{
		Text1: text1,
		Text2: text2,
	}

	result, err := s.CalculateSimilarity(ctx, req)
	if err != nil {
		return 0, err
	}

	return result.Similarity, nil
}

// EvaluateQueryRewrite 评估查询改写质量
func (s *UnifiedSimilarityService) EvaluateQueryRewrite(originalQuery, rewrittenQuery string) (*QueryRewriteEvaluation, error) {
	ctx := context.Background()
	req := &SimilarityRequest{
		Text1:   originalQuery,
		Text2:   rewrittenQuery,
		Context: "query_rewrite_evaluation",
	}

	result, err := s.CalculateSimilarity(ctx, req)
	if err != nil {
		return nil, err
	}

	return &QueryRewriteEvaluation{
		SemanticSimilarity: result.Similarity,
		QualityScore:       result.Details.QualityScore,
		Method:             result.Method,
		IsGoodRewrite:      result.Similarity >= 0.7 && result.Similarity <= 0.95, // 保持语义但有所改进
		Recommendation:     s.generateRewriteRecommendation(result),
	}, nil
}

// QueryRewriteEvaluation 查询改写评估结果
type QueryRewriteEvaluation struct {
	SemanticSimilarity float64 `json:"semantic_similarity"`
	QualityScore       float64 `json:"quality_score"`
	Method             string  `json:"method"`
	IsGoodRewrite      bool    `json:"is_good_rewrite"`
	Recommendation     string  `json:"recommendation"`
}

// generateRewriteRecommendation 生成改写建议
func (s *UnifiedSimilarityService) generateRewriteRecommendation(result *SimilarityResponse) string {
	similarity := result.Similarity

	if similarity >= 0.95 {
		return "改写效果较小，可能不需要改写"
	} else if similarity >= 0.8 {
		return "改写效果良好，保持了原意并有所优化"
	} else if similarity >= 0.6 {
		return "改写幅度较大，请检查是否保持了原始意图"
	} else {
		return "改写差异过大，可能偏离了原始意图"
	}
}
