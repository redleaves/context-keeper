package retrieval

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// QueryRewriter 查询改写器 - 核心查询增强组件
type QueryRewriter struct {
	config        *QueryRewriterConfig
	enabled       bool
	rewriteChains []RewriteChain
}

// QueryRewriterConfig 查询改写器配置
type QueryRewriterConfig struct {
	// 关键词提取
	KeywordExtraction KeywordExtractionConfig `json:"keyword_extraction"`
	// 噪声去除
	NoiseReduction NoiseReductionConfig `json:"noise_reduction"`
	// 上下文丰富
	ContextEnrichment ContextEnrichmentConfig `json:"context_enrichment"`
	// 迭代设置
	MaxIterations    int     `json:"max_iterations"`
	QualityThreshold float64 `json:"quality_threshold"`
}

// 🔥 关键词提取配置
type KeywordExtractionConfig struct {
	Enabled     bool     `json:"enabled"`
	Methods     []string `json:"methods"`    // ["tfidf", "keyword_density", "named_entity", "technical_terms"]
	MinWeight   float64  `json:"min_weight"` // 最小权重阈值
	MaxKeywords int      `json:"max_keywords"`
}

// 🔥 噪声去除配置
type NoiseReductionConfig struct {
	Enabled       bool     `json:"enabled"`
	StopWords     []string `json:"stop_words"`     // 停用词
	FillerPhrases []string `json:"filler_phrases"` // 填充短语
	MinLength     int      `json:"min_length"`     // 最小查询长度
}

// 🔥 上下文丰富配置
type ContextEnrichmentConfig struct {
	Enabled          bool     `json:"enabled"`
	DomainTerms      []string `json:"domain_terms"`      // 领域术语
	SynonymExpansion bool     `json:"synonym_expansion"` // 同义词扩展
	RelatedConcepts  bool     `json:"related_concepts"`  // 相关概念
}

// RewriteChain 改写链接口
type RewriteChain interface {
	Name() string
	Process(ctx context.Context, query string, metadata map[string]interface{}) (*RewriteResult, error)
	Priority() int
}

// RewriteResult 改写结果
type RewriteResult struct {
	OriginalQuery     string                 `json:"original_query"`
	RewrittenQuery    string                 `json:"rewritten_query"`
	ExtractedKeywords []KeywordInfo          `json:"extracted_keywords"`
	RemovedNoise      []string               `json:"removed_noise"`
	AddedContext      []string               `json:"added_context"`
	QualityScore      float64                `json:"quality_score"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// KeywordInfo 关键词信息
type KeywordInfo struct {
	Term     string  `json:"term"`
	Weight   float64 `json:"weight"`
	Category string  `json:"category"` // "technical", "domain", "general"
}

// NewQueryRewriter 创建查询改写器
func NewQueryRewriter(config *QueryRewriterConfig) *QueryRewriter {
	rewriter := &QueryRewriter{
		config:        config,
		enabled:       true,
		rewriteChains: make([]RewriteChain, 0),
	}

	// 🔥 初始化改写链 - 按优先级排序
	if config.KeywordExtraction.Enabled {
		rewriter.AddChain(&KeywordExtractionChain{config: &config.KeywordExtraction})
	}

	if config.NoiseReduction.Enabled {
		rewriter.AddChain(&NoiseReductionChain{config: &config.NoiseReduction})
	}

	if config.ContextEnrichment.Enabled {
		rewriter.AddChain(&ContextEnrichmentChain{config: &config.ContextEnrichment})
	}

	return rewriter
}

// AddChain 添加改写链
func (qr *QueryRewriter) AddChain(chain RewriteChain) {
	qr.rewriteChains = append(qr.rewriteChains, chain)
	// 按优先级排序
	for i := len(qr.rewriteChains) - 1; i > 0; i-- {
		if qr.rewriteChains[i].Priority() > qr.rewriteChains[i-1].Priority() {
			qr.rewriteChains[i], qr.rewriteChains[i-1] = qr.rewriteChains[i-1], qr.rewriteChains[i]
		}
	}
}

// ProcessQuery 处理查询 - 核心入口方法
func (qr *QueryRewriter) ProcessQuery(ctx context.Context, originalQuery string) (*RewriteResult, error) {
	if !qr.enabled {
		return &RewriteResult{
			OriginalQuery:  originalQuery,
			RewrittenQuery: originalQuery,
			QualityScore:   1.0,
		}, nil
	}

	startTime := time.Now()

	result := &RewriteResult{
		OriginalQuery:     originalQuery,
		RewrittenQuery:    originalQuery,
		ExtractedKeywords: make([]KeywordInfo, 0),
		RemovedNoise:      make([]string, 0),
		AddedContext:      make([]string, 0),
		QualityScore:      0.0,
		Metadata:          make(map[string]interface{}),
	}

	currentQuery := originalQuery

	// 🔥 多轮迭代改写
	for iteration := 0; iteration < qr.config.MaxIterations; iteration++ {
		iterationImproved := false

		// 按优先级执行改写链
		for _, chain := range qr.rewriteChains {
			chainResult, err := chain.Process(ctx, currentQuery, result.Metadata)
			if err != nil {
				continue // 链执行失败时继续下一个
			}

			// 合并结果
			if chainResult.RewrittenQuery != currentQuery {
				currentQuery = chainResult.RewrittenQuery
				iterationImproved = true

				// 合并关键词
				result.ExtractedKeywords = append(result.ExtractedKeywords, chainResult.ExtractedKeywords...)
				result.RemovedNoise = append(result.RemovedNoise, chainResult.RemovedNoise...)
				result.AddedContext = append(result.AddedContext, chainResult.AddedContext...)

				// 更新质量分数（取最高分）
				if chainResult.QualityScore > result.QualityScore {
					result.QualityScore = chainResult.QualityScore
				}
			}
		}

		// 🔥 质量评估 - 达到阈值则停止迭代
		if result.QualityScore >= qr.config.QualityThreshold {
			break
		}

		// 无改进则停止
		if !iterationImproved {
			break
		}
	}

	result.RewrittenQuery = currentQuery
	result.ProcessingTime = time.Since(startTime)

	// 🔥 最终质量检查
	if result.QualityScore == 0.0 {
		result.QualityScore = qr.calculateFinalQuality(originalQuery, currentQuery)
	}

	return result, nil
}

// 🔥 关键词提取链实现
type KeywordExtractionChain struct {
	config *KeywordExtractionConfig
}

func (kec *KeywordExtractionChain) Name() string  { return "keyword_extraction" }
func (kec *KeywordExtractionChain) Priority() int { return 100 }

func (kec *KeywordExtractionChain) Process(ctx context.Context, query string, metadata map[string]interface{}) (*RewriteResult, error) {
	keywords := make([]KeywordInfo, 0)

	// 🔥 技术术语提取
	technicalTerms := kec.extractTechnicalTerms(query)
	for _, term := range technicalTerms {
		keywords = append(keywords, KeywordInfo{
			Term:     term,
			Weight:   0.9,
			Category: "technical",
		})
	}

	// 🔥 命名实体提取
	entities := kec.extractNamedEntities(query)
	for _, entity := range entities {
		keywords = append(keywords, KeywordInfo{
			Term:     entity,
			Weight:   0.8,
			Category: "entity",
		})
	}

	// 🔥 基于密度的关键词
	densityKeywords := kec.extractByDensity(query)
	for _, kw := range densityKeywords {
		keywords = append(keywords, kw)
	}

	// 构建增强查询
	enhancedQuery := kec.buildEnhancedQuery(query, keywords)

	return &RewriteResult{
		OriginalQuery:     query,
		RewrittenQuery:    enhancedQuery,
		ExtractedKeywords: keywords,
		QualityScore:      kec.calculateKeywordQuality(keywords),
	}, nil
}

// 🔥 噪声去除链实现
type NoiseReductionChain struct {
	config *NoiseReductionConfig
}

func (nrc *NoiseReductionChain) Name() string  { return "noise_reduction" }
func (nrc *NoiseReductionChain) Priority() int { return 90 }

func (nrc *NoiseReductionChain) Process(ctx context.Context, query string, metadata map[string]interface{}) (*RewriteResult, error) {
	cleanedQuery := query
	removedNoise := make([]string, 0)

	// 🔥 移除停用词
	for _, stopWord := range nrc.config.StopWords {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(stopWord))
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(cleanedQuery) {
			cleanedQuery = re.ReplaceAllString(cleanedQuery, " ")
			removedNoise = append(removedNoise, stopWord)
		}
	}

	// 🔥 移除填充短语
	for _, filler := range nrc.config.FillerPhrases {
		if strings.Contains(strings.ToLower(cleanedQuery), strings.ToLower(filler)) {
			cleanedQuery = strings.ReplaceAll(cleanedQuery, filler, " ")
			removedNoise = append(removedNoise, filler)
		}
	}

	// 🔥 清理多余空格
	cleanedQuery = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedQuery, " ")
	cleanedQuery = strings.TrimSpace(cleanedQuery)

	// 检查最小长度
	if len(cleanedQuery) < nrc.config.MinLength {
		cleanedQuery = query // 恢复原查询
	}

	return &RewriteResult{
		OriginalQuery:  query,
		RewrittenQuery: cleanedQuery,
		RemovedNoise:   removedNoise,
		QualityScore:   nrc.calculateCleaningQuality(query, cleanedQuery),
	}, nil
}

// 🔥 上下文丰富链实现
type ContextEnrichmentChain struct {
	config *ContextEnrichmentConfig
}

func (cec *ContextEnrichmentChain) Name() string  { return "context_enrichment" }
func (cec *ContextEnrichmentChain) Priority() int { return 80 }

func (cec *ContextEnrichmentChain) Process(ctx context.Context, query string, metadata map[string]interface{}) (*RewriteResult, error) {
	enrichedQuery := query
	addedContext := make([]string, 0)

	// 🔥 添加领域术语
	domainTerms := cec.findRelevantDomainTerms(query)
	for _, term := range domainTerms {
		if !strings.Contains(strings.ToLower(enrichedQuery), strings.ToLower(term)) {
			enrichedQuery += " " + term
			addedContext = append(addedContext, term)
		}
	}

	// 🔥 同义词扩展
	if cec.config.SynonymExpansion {
		synonyms := cec.expandSynonyms(query)
		for _, synonym := range synonyms {
			enrichedQuery += " " + synonym
			addedContext = append(addedContext, synonym)
		}
	}

	// 🔥 相关概念补充
	if cec.config.RelatedConcepts {
		concepts := cec.addRelatedConcepts(query)
		for _, concept := range concepts {
			enrichedQuery += " " + concept
			addedContext = append(addedContext, concept)
		}
	}

	return &RewriteResult{
		OriginalQuery:  query,
		RewrittenQuery: strings.TrimSpace(enrichedQuery),
		AddedContext:   addedContext,
		QualityScore:   cec.calculateEnrichmentQuality(query, enrichedQuery),
	}, nil
}

// 🔥 辅助方法实现

func (kec *KeywordExtractionChain) extractTechnicalTerms(query string) []string {
	// 简化实现：基于常见技术术语模式
	technicalPatterns := []string{
		`\b[A-Z]{2,}\b`,     // 大写缩写 (API, HTTP, JSON)
		`\b\w+\(\)\b`,       // 函数调用模式
		`\b\w+\.\w+\b`,      // 属性访问模式
		`\b\w+Service\b`,    // 服务模式
		`\b\w+Controller\b`, // 控制器模式
	}

	terms := make([]string, 0)
	for _, pattern := range technicalPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(query, -1)
		terms = append(terms, matches...)
	}
	return terms
}

func (kec *KeywordExtractionChain) extractNamedEntities(query string) []string {
	// 简化实现：基于首字母大写模式
	re := regexp.MustCompile(`\b[A-Z][a-z]+\b`)
	return re.FindAllString(query, -1)
}

func (kec *KeywordExtractionChain) extractByDensity(query string) []KeywordInfo {
	words := strings.Fields(strings.ToLower(query))
	frequency := make(map[string]int)

	for _, word := range words {
		word = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(word, "")
		if len(word) > 2 { // 忽略过短的词
			frequency[word]++
		}
	}

	keywords := make([]KeywordInfo, 0)
	for word, freq := range frequency {
		if freq > 1 { // 出现多次的词
			keywords = append(keywords, KeywordInfo{
				Term:     word,
				Weight:   float64(freq) / float64(len(words)),
				Category: "density",
			})
		}
	}

	return keywords
}

func (kec *KeywordExtractionChain) buildEnhancedQuery(query string, keywords []KeywordInfo) string {
	// 根据关键词权重重新构建查询
	highValueKeywords := make([]string, 0)
	for _, kw := range keywords {
		if kw.Weight > 0.7 {
			highValueKeywords = append(highValueKeywords, kw.Term)
		}
	}

	if len(highValueKeywords) > 0 {
		return query + " " + strings.Join(highValueKeywords, " ")
	}
	return query
}

func (kec *KeywordExtractionChain) calculateKeywordQuality(keywords []KeywordInfo) float64 {
	if len(keywords) == 0 {
		return 0.5
	}

	totalWeight := 0.0
	for _, kw := range keywords {
		totalWeight += kw.Weight
	}

	avgWeight := totalWeight / float64(len(keywords))
	if avgWeight > 1.0 {
		return 1.0
	}
	return avgWeight
}

func (nrc *NoiseReductionChain) calculateCleaningQuality(original, cleaned string) float64 {
	originalLen := len(strings.Fields(original))
	cleanedLen := len(strings.Fields(cleaned))

	if originalLen == 0 {
		return 0.5
	}

	// 质量 = 保留的有用词汇比例
	preservation := float64(cleanedLen) / float64(originalLen)

	// 去噪效果评分
	if preservation > 0.7 && preservation < 1.0 {
		return 0.8 // 适度去噪
	} else if preservation == 1.0 {
		return 0.6 // 未去噪
	} else if preservation < 0.3 {
		return 0.3 // 过度去噪
	}

	return preservation
}

func (cec *ContextEnrichmentChain) findRelevantDomainTerms(query string) []string {
	// 简化实现：基于配置的领域术语匹配
	relevant := make([]string, 0)
	queryLower := strings.ToLower(query)

	for _, term := range cec.config.DomainTerms {
		// 如果查询与领域术语有关联，添加相关术语
		if strings.Contains(queryLower, strings.ToLower(term[:min(len(term), 3)])) {
			relevant = append(relevant, term)
		}
	}

	return relevant
}

func (cec *ContextEnrichmentChain) expandSynonyms(query string) []string {
	// 简化实现：常见同义词映射
	synonymMap := map[string][]string{
		"问题": {"issue", "bug", "故障"},
		"解决": {"fix", "resolve", "solve"},
		"优化": {"optimize", "improve", "enhance"},
		"配置": {"config", "configuration", "setting"},
	}

	synonyms := make([]string, 0)
	for word, syns := range synonymMap {
		if strings.Contains(strings.ToLower(query), word) {
			synonyms = append(synonyms, syns...)
		}
	}

	return synonyms
}

func (cec *ContextEnrichmentChain) addRelatedConcepts(query string) []string {
	// 简化实现：基于查询内容推断相关概念
	concepts := make([]string, 0)
	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "api") {
		concepts = append(concepts, "接口", "请求", "响应", "状态码")
	}
	if strings.Contains(queryLower, "数据库") {
		concepts = append(concepts, "SQL", "查询", "索引", "事务")
	}
	if strings.Contains(queryLower, "性能") {
		concepts = append(concepts, "延迟", "吞吐量", "并发", "缓存")
	}

	return concepts
}

func (cec *ContextEnrichmentChain) calculateEnrichmentQuality(original, enriched string) float64 {
	originalWords := len(strings.Fields(original))
	enrichedWords := len(strings.Fields(enriched))

	if originalWords == 0 {
		return 0.5
	}

	enrichmentRatio := float64(enrichedWords-originalWords) / float64(originalWords)

	// 适度丰富 (10%-50%) 获得最高分
	if enrichmentRatio >= 0.1 && enrichmentRatio <= 0.5 {
		return 0.9
	} else if enrichmentRatio > 0.5 {
		return 0.7 // 过度丰富
	} else {
		return 0.6 // 丰富不足
	}
}

func (qr *QueryRewriter) calculateFinalQuality(original, rewritten string) float64 {
	// 综合质量评估
	if original == rewritten {
		return 0.5 // 无改变
	}

	// 基于长度变化和复杂度评估
	originalComplexity := qr.calculateComplexity(original)
	rewrittenComplexity := qr.calculateComplexity(rewritten)

	if rewrittenComplexity > originalComplexity {
		return 0.8 // 复杂度提升
	} else if rewrittenComplexity == originalComplexity {
		return 0.6 // 复杂度持平
	} else {
		return 0.4 // 复杂度降低
	}
}

func (qr *QueryRewriter) calculateComplexity(query string) float64 {
	words := strings.Fields(query)
	uniqueWords := make(map[string]bool)

	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}

	// 复杂度 = 词汇多样性 + 长度因子
	diversity := float64(len(uniqueWords)) / float64(len(words))
	lengthFactor := float64(len(words)) / 10.0
	if lengthFactor > 1.0 {
		lengthFactor = 1.0
	}

	return (diversity + lengthFactor) / 2.0
}

// 获取配置
func (qr *QueryRewriter) GetConfig() *QueryRewriterConfig {
	return qr.config
}

// 启用/禁用
func (qr *QueryRewriter) SetEnabled(enabled bool) {
	qr.enabled = enabled
}

// 检查是否启用
func (qr *QueryRewriter) IsEnabled() bool {
	return qr.enabled
}

// 获取统计信息
func (qr *QueryRewriter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           qr.enabled,
		"chains_count":      len(qr.rewriteChains),
		"max_iterations":    qr.config.MaxIterations,
		"quality_threshold": qr.config.QualityThreshold,
	}
}

// 辅助函数已移至iterative_retriever.go中避免重复定义

type QueryRewriteLog struct {
	OriginalQuery       string           `json:"original_query"`
	FinalRewrittenQuery string           `json:"final_rewritten_query"`
	RewriteSteps        []RewriteStepLog `json:"rewrite_steps"`
	TotalIterations     int              `json:"total_iterations"`
	QualityImprovement  float64          `json:"quality_improvement"`
	ProcessingTime      time.Duration    `json:"processing_time"`
	Timestamp           time.Time        `json:"timestamp"`
}

type RewriteStepLog struct {
	Iteration     int                    `json:"iteration"`
	ChainName     string                 `json:"chain_name"`
	Priority      int                    `json:"priority"`
	InputQuery    string                 `json:"input_query"`
	OutputQuery   string                 `json:"output_query"`
	Changes       []string               `json:"changes"`
	QualityScore  float64                `json:"quality_score"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type QueryRewriteResult struct {
	OriginalQuery    string        `json:"original_query"`
	RewrittenQueries []string      `json:"rewritten_queries"`
	QualityScores    []float64     `json:"quality_scores"`
	FinalQuery       string        `json:"final_query"`
	TotalIterations  int           `json:"total_iterations"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

func (qr *QueryRewriter) RewriteQuery(originalQuery string) (*QueryRewriteResult, error) {
	startTime := time.Now()
	log := &QueryRewriteLog{
		OriginalQuery: originalQuery,
		Timestamp:     startTime,
		RewriteSteps:  make([]RewriteStepLog, 0),
	}

	result := &QueryRewriteResult{
		OriginalQuery:    originalQuery,
		RewrittenQueries: []string{originalQuery},
		QualityScores:    []float64{0.0},
		TotalIterations:  0,
		ProcessingTime:   0,
	}

	currentQuery := originalQuery
	initialQuality := qr.calculateInitialQuality(originalQuery)

	for iteration := 1; iteration <= qr.config.MaxIterations; iteration++ {
		iterationImproved := false

		// 按优先级执行改写链
		chains := qr.getSortedChains()
		for _, chain := range chains {
			stepStartTime := time.Now()

			// 调用改写链处理查询
			result, err := chain.Process(context.Background(), currentQuery, make(map[string]interface{}))
			if err != nil {
				continue
			}

			rewrittenQuery := result.RewrittenQuery

			// 记录改写步骤
			stepLog := RewriteStepLog{
				Iteration:     iteration,
				ChainName:     reflect.TypeOf(chain).Elem().Name(),
				Priority:      chain.Priority(),
				InputQuery:    currentQuery,
				OutputQuery:   rewrittenQuery,
				Changes:       qr.extractChanges(currentQuery, rewrittenQuery),
				ExecutionTime: time.Since(stepStartTime),
				Metadata:      make(map[string]interface{}),
			}

			// 计算质量分数
			if rewrittenQuery != currentQuery {
				stepLog.QualityScore = qr.calculateQueryQuality(rewrittenQuery)
				stepLog.Metadata["improvement"] = stepLog.QualityScore > qr.calculateQueryQuality(currentQuery)

				currentQuery = rewrittenQuery
				iterationImproved = true
			} else {
				stepLog.QualityScore = qr.calculateQueryQuality(currentQuery)
				stepLog.Metadata["improvement"] = false
			}

			log.RewriteSteps = append(log.RewriteSteps, stepLog)
		}

		// 检查质量阈值
		currentQuality := qr.calculateQueryQuality(currentQuery)
		if currentQuality >= qr.config.QualityThreshold {
			break
		}

		// 如果本轮没有改进，停止迭代
		if !iterationImproved {
			break
		}

		result.RewrittenQueries = append(result.RewrittenQueries, currentQuery)
		result.QualityScores = append(result.QualityScores, currentQuality)
	}

	// 完成日志记录
	log.FinalRewrittenQuery = currentQuery
	log.TotalIterations = len(result.RewrittenQueries) - 1
	log.ProcessingTime = time.Since(startTime)
	log.QualityImprovement = qr.calculateQueryQuality(currentQuery) - initialQuality

	// 输出详细日志
	qr.printQueryRewriteComparison(log)

	result.FinalQuery = currentQuery
	result.TotalIterations = log.TotalIterations
	result.ProcessingTime = log.ProcessingTime

	return result, nil
}

func (qr *QueryRewriter) printQueryRewriteComparison(log *QueryRewriteLog) {
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("🔍 QUERY REWRITE ANALYSIS - 查询改写优化分析")
	fmt.Println(strings.Repeat("=", 100))

	// 1. 原始查询信息
	fmt.Println("\n📝 1. ORIGINAL QUERY - 用户原始提问")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("原始查询: %s\n", log.OriginalQuery)
	fmt.Printf("查询长度: %d 字符\n", len(log.OriginalQuery))
	fmt.Printf("处理时间: %v\n", log.Timestamp.Format("2006-01-02 15:04:05"))

	// 分析原始查询特征
	originalFeatures := qr.analyzeQueryFeatures(log.OriginalQuery)
	fmt.Println("\n🔎 原始查询特征分析:")
	for feature, value := range originalFeatures {
		fmt.Printf("  - %s: %v\n", feature, value)
	}

	// 2. 改写过程详情
	fmt.Println("\n⚙️ 2. REWRITE PROCESS - 改写过程详情")
	fmt.Println(strings.Repeat("-", 80))

	if len(log.RewriteSteps) == 0 {
		fmt.Println("❌ 未执行任何改写步骤")
	} else {
		for i, step := range log.RewriteSteps {
			fmt.Printf("\n🔄 步骤 %d - %s (优先级:%d)\n", i+1, step.ChainName, step.Priority)
			fmt.Printf("  输入: %s\n", step.InputQuery)
			fmt.Printf("  输出: %s\n", step.OutputQuery)
			fmt.Printf("  质量分数: %.3f\n", step.QualityScore)
			fmt.Printf("  执行时间: %v\n", step.ExecutionTime)

			if len(step.Changes) > 0 {
				fmt.Println("  变化详情:")
				for _, change := range step.Changes {
					fmt.Printf("    • %s\n", change)
				}
			}

			if improvement, exists := step.Metadata["improvement"].(bool); exists {
				if improvement {
					fmt.Println("  ✅ 质量提升")
				} else {
					fmt.Println("  ⚪ 无明显改进")
				}
			}
		}
	}

	// 3. 最终改写结果
	fmt.Println("\n🎯 3. FINAL REWRITTEN QUERY - 最终改写结果")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("最终查询: %s\n", log.FinalRewrittenQuery)
	fmt.Printf("查询长度: %d 字符\n", len(log.FinalRewrittenQuery))

	// 分析最终查询特征
	finalFeatures := qr.analyzeQueryFeatures(log.FinalRewrittenQuery)
	fmt.Println("\n🔎 最终查询特征分析:")
	for feature, value := range finalFeatures {
		fmt.Printf("  - %s: %v\n", feature, value)
	}

	// 4. 对比分析
	fmt.Println("\n📊 4. COMPARISON ANALYSIS - 对比分析")
	fmt.Println(strings.Repeat("-", 80))

	// 长度对比
	lengthChange := len(log.FinalRewrittenQuery) - len(log.OriginalQuery)
	fmt.Printf("长度变化: %+d 字符", lengthChange)
	if lengthChange > 0 {
		fmt.Println(" (扩展)")
	} else if lengthChange < 0 {
		fmt.Println(" (压缩)")
	} else {
		fmt.Println(" (无变化)")
	}

	// 质量提升
	fmt.Printf("质量提升: %+.3f", log.QualityImprovement)
	if log.QualityImprovement > 0 {
		fmt.Println(" ✅ (改进)")
	} else if log.QualityImprovement < 0 {
		fmt.Println(" ❌ (退化)")
	} else {
		fmt.Println(" ⚪ (无变化)")
	}

	// 迭代统计
	fmt.Printf("迭代次数: %d\n", log.TotalIterations)
	fmt.Printf("总处理时间: %v\n", log.ProcessingTime)

	// 语义相似度
	similarity := qr.calculateSemanticSimilarity(log.OriginalQuery, log.FinalRewrittenQuery)
	fmt.Printf("语义相似度: %.3f", similarity)
	if similarity > 0.8 {
		fmt.Println(" ✅ (高度相似)")
	} else if similarity > 0.6 {
		fmt.Println(" ⚪ (中等相似)")
	} else {
		fmt.Println(" ⚠️ (低相似度)")
	}

	// 5. 改写效果总结
	fmt.Println("\n📋 5. REWRITE EFFECTIVENESS - 改写效果总结")
	fmt.Println(strings.Repeat("-", 80))

	effectiveness := qr.evaluateRewriteEffectiveness(log)
	fmt.Printf("整体评价: %s\n", effectiveness.Overall)
	fmt.Printf("语义保持: %s\n", effectiveness.SemanticPreservation)
	fmt.Printf("信息丰富: %s\n", effectiveness.InformationEnrichment)
	fmt.Printf("检索友好: %s\n", effectiveness.RetrievalFriendliness)

	if len(effectiveness.Recommendations) > 0 {
		fmt.Println("\n💡 优化建议:")
		for _, rec := range effectiveness.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
}

func (qr *QueryRewriter) extractChanges(original, rewritten string) []string {
	changes := make([]string, 0)

	// 简单的变化检测
	if len(rewritten) > len(original) {
		changes = append(changes, fmt.Sprintf("查询扩展: +%d字符", len(rewritten)-len(original)))
	} else if len(rewritten) < len(original) {
		changes = append(changes, fmt.Sprintf("查询压缩: -%d字符", len(original)-len(rewritten)))
	}

	// 检测新增的关键词
	originalWords := strings.Fields(strings.ToLower(original))
	rewrittenWords := strings.Fields(strings.ToLower(rewritten))

	originalSet := make(map[string]bool)
	for _, word := range originalWords {
		originalSet[word] = true
	}

	newWords := make([]string, 0)
	for _, word := range rewrittenWords {
		if !originalSet[word] && len(word) > 2 {
			newWords = append(newWords, word)
		}
	}

	if len(newWords) > 0 {
		changes = append(changes, fmt.Sprintf("新增关键词: %s", strings.Join(newWords, ", ")))
	}

	return changes
}

func (qr *QueryRewriter) analyzeQueryFeatures(query string) map[string]interface{} {
	features := make(map[string]interface{})

	features["字符数"] = len(query)
	features["词汇数"] = len(strings.Fields(query))
	features["包含技术术语"] = qr.containsTechnicalTerms(query)
	features["包含问号"] = strings.Contains(query, "?") || strings.Contains(query, "？")
	features["包含特殊符号"] = qr.containsSpecialChars(query)
	features["语言类型"] = qr.detectLanguage(query)

	return features
}

func (qr *QueryRewriter) calculateSemanticSimilarity(query1, query2 string) float64 {
	// 简单的语义相似度计算 (实际项目中应使用更sophisticated的方法)
	words1 := strings.Fields(strings.ToLower(query1))
	words2 := strings.Fields(strings.ToLower(query2))

	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	common := 0
	for _, word := range words2 {
		if set1[word] {
			common++
		}
	}

	total := len(set1)
	for _, word := range words2 {
		if !set1[word] {
			total++
		}
	}

	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

type RewriteEffectiveness struct {
	Overall               string   `json:"overall"`
	SemanticPreservation  string   `json:"semantic_preservation"`
	InformationEnrichment string   `json:"information_enrichment"`
	RetrievalFriendliness string   `json:"retrieval_friendliness"`
	Recommendations       []string `json:"recommendations"`
}

func (qr *QueryRewriter) evaluateRewriteEffectiveness(log *QueryRewriteLog) RewriteEffectiveness {
	effectiveness := RewriteEffectiveness{
		Recommendations: make([]string, 0),
	}

	// 整体评价
	if log.QualityImprovement > 0.1 {
		effectiveness.Overall = "优秀 ✅"
	} else if log.QualityImprovement > 0 {
		effectiveness.Overall = "良好 ⚪"
	} else {
		effectiveness.Overall = "需改进 ⚠️"
		effectiveness.Recommendations = append(effectiveness.Recommendations, "考虑调整改写策略参数")
	}

	// 语义保持评价
	similarity := qr.calculateSemanticSimilarity(log.OriginalQuery, log.FinalRewrittenQuery)
	if similarity > 0.8 {
		effectiveness.SemanticPreservation = "优秀 ✅"
	} else if similarity > 0.6 {
		effectiveness.SemanticPreservation = "良好 ⚪"
	} else {
		effectiveness.SemanticPreservation = "需注意 ⚠️"
		effectiveness.Recommendations = append(effectiveness.Recommendations, "改写可能偏离原始意图，需要调优")
	}

	// 信息丰富度评价
	lengthIncrease := len(log.FinalRewrittenQuery) - len(log.OriginalQuery)
	if lengthIncrease > 20 {
		effectiveness.InformationEnrichment = "丰富 ✅"
	} else if lengthIncrease > 0 {
		effectiveness.InformationEnrichment = "适中 ⚪"
	} else {
		effectiveness.InformationEnrichment = "简化 ⚪"
	}

	// 检索友好度评价
	if qr.containsTechnicalTerms(log.FinalRewrittenQuery) && len(strings.Fields(log.FinalRewrittenQuery)) > 3 {
		effectiveness.RetrievalFriendliness = "优秀 ✅"
	} else {
		effectiveness.RetrievalFriendliness = "一般 ⚪"
		effectiveness.Recommendations = append(effectiveness.Recommendations, "可以进一步丰富技术关键词")
	}

	return effectiveness
}

func (qr *QueryRewriter) containsTechnicalTerms(query string) bool {
	technicalTerms := []string{
		"api", "数据库", "算法", "框架", "架构", "python", "java", "go", "react", "vue",
		"docker", "kubernetes", "mysql", "redis", "nginx", "linux", "git", "http",
		"json", "xml", "rest", "grpc", "微服务", "分布式", "缓存", "消息队列",
	}

	queryLower := strings.ToLower(query)
	for _, term := range technicalTerms {
		if strings.Contains(queryLower, term) {
			return true
		}
	}
	return false
}

func (qr *QueryRewriter) containsSpecialChars(query string) bool {
	specialChars := []string{"@", "#", "$", "%", "&", "*", "(", ")", "[", "]", "{", "}"}
	for _, char := range specialChars {
		if strings.Contains(query, char) {
			return true
		}
	}
	return false
}

func (qr *QueryRewriter) detectLanguage(query string) string {
	// 简单的语言检测
	chineseCount := 0
	englishCount := 0

	for _, r := range query {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}
	}

	if chineseCount > englishCount {
		return "中文"
	} else if englishCount > chineseCount {
		return "英文"
	} else {
		return "混合"
	}
}

func (qr *QueryRewriter) calculateInitialQuality(query string) float64 {
	// 简单的初始质量计算
	score := 0.0

	// 长度分数
	if len(query) > 10 && len(query) < 200 {
		score += 0.3
	}

	// 技术术语分数
	if qr.containsTechnicalTerms(query) {
		score += 0.4
	}

	// 词汇丰富度分数
	words := strings.Fields(query)
	if len(words) > 3 {
		score += 0.3
	}

	return score
}

func (qr *QueryRewriter) calculateQueryQuality(query string) float64 {
	// 更详细的质量计算逻辑
	score := qr.calculateInitialQuality(query)

	// 可以添加更多质量评估维度
	// 如语法正确性、语义清晰度等

	return score
}

func (qr *QueryRewriter) getSortedChains() []RewriteChain {
	// 返回已配置的改写链，它们已经按优先级排序
	return qr.rewriteChains
}
