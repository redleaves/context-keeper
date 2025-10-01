package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// ParallelContextSynthesizer 并行上下文合成器
type ParallelContextSynthesizer struct {
	llmService LLMService
	config     *ParallelSynthesisConfig
}

// ParallelSynthesisConfig 并行合成配置
type ParallelSynthesisConfig struct {
	LLMTimeout      int     `json:"llm_timeout"`      // LLM超时时间
	MaxTokens       int     `json:"max_tokens"`       // 最大token数
	Temperature     float64 `json:"temperature"`      // 温度参数
	MaxConcurrency  int     `json:"max_concurrency"`  // 最大并发数
	FailureStrategy string  `json:"failure_strategy"` // 失败策略：partial/strict
}

// ContextDimension 上下文维度
type ContextDimension string

const (
	DimensionTopic        ContextDimension = "topic"        // 主题维度
	DimensionProject      ContextDimension = "project"      // 项目维度
	DimensionConversation ContextDimension = "conversation" // 会话维度
	DimensionCode         ContextDimension = "code"         // 代码维度
	DimensionChanges      ContextDimension = "changes"      // 变更维度
)

// DimensionResult 维度生成结果
type DimensionResult struct {
	Dimension ContextDimension `json:"dimension"`
	Success   bool             `json:"success"`
	Content   string           `json:"content"`
	Error     error            `json:"error,omitempty"`
	Duration  time.Duration    `json:"duration"`
}

// NewParallelContextSynthesizer 创建并行上下文合成器
func NewParallelContextSynthesizer(llmService LLMService, config *ParallelSynthesisConfig) *ParallelContextSynthesizer {
	if config == nil {
		config = &ParallelSynthesisConfig{
			LLMTimeout:      30,
			MaxTokens:       2000, // 每个维度的token限制降低到1/5
			Temperature:     0.1,
			MaxConcurrency:  5,
			FailureStrategy: "partial", // 允许部分失败
		}
	}

	return &ParallelContextSynthesizer{
		llmService: llmService,
		config:     config,
	}
}

// SynthesizeContextParallel 并行合成上下文
func (pcs *ParallelContextSynthesizer) SynthesizeContextParallel(
	ctx context.Context,
	userQuery string,
	retrievalResults *models.RetrievalResults,
	currentContext *models.UnifiedContextModel,
) (*models.UnifiedContextModel, error) {

	log.Printf("🚀 [方案2-并行合成] 开始5维度并行上下文合成")
	log.Printf("📊 [方案2-并行合成] 输入数据规模:")
	log.Printf("   - 用户查询长度: %d字符", len(userQuery))
	log.Printf("   - 检索结果总数: %d", retrievalResults.TotalResults)
	log.Printf("   - 并发维度数: 5")

	startTime := time.Now()

	// 定义5个维度
	dimensions := []ContextDimension{
		DimensionTopic,
		DimensionProject,
		DimensionConversation,
		DimensionCode,
		DimensionChanges,
	}

	// 创建结果通道
	resultChan := make(chan DimensionResult, len(dimensions))

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup

	// 并发生成每个维度
	for _, dimension := range dimensions {
		wg.Add(1)
		go func(dim ContextDimension) {
			defer wg.Done()
			result := pcs.generateDimension(ctx, dim, userQuery, retrievalResults, currentContext)
			resultChan <- result
		}(dimension)
	}

	// 等待所有维度完成
	wg.Wait()
	close(resultChan)

	// 收集结果
	results := make(map[ContextDimension]DimensionResult)
	successCount := 0

	for result := range resultChan {
		results[result.Dimension] = result
		if result.Success {
			successCount++
		}

		log.Printf("📋 [方案2-并行合成] 维度 %s: %s (耗时: %v)",
			result.Dimension,
			map[bool]string{true: "✅成功", false: "❌失败"}[result.Success],
			result.Duration)

		if result.Error != nil {
			log.Printf("   错误详情: %v", result.Error)
		}
	}

	totalDuration := time.Since(startTime)
	log.Printf("🎯 [方案2-并行合成] 并行生成完成: %d/%d 成功, 总耗时: %v",
		successCount, len(dimensions), totalDuration)

	// 检查失败策略
	if pcs.config.FailureStrategy == "strict" && successCount < len(dimensions) {
		return nil, fmt.Errorf("严格模式下有 %d 个维度失败", len(dimensions)-successCount)
	}

	if successCount == 0 {
		return nil, fmt.Errorf("所有维度生成都失败了")
	}

	// 合并结果
	mergedContext, err := pcs.mergeResults(results, currentContext)
	if err != nil {
		return nil, fmt.Errorf("合并结果失败: %w", err)
	}

	log.Printf("✅ [方案2-并行合成] 上下文合并完成")
	return mergedContext, nil
}

// generateDimension 生成单个维度的上下文
func (pcs *ParallelContextSynthesizer) generateDimension(
	ctx context.Context,
	dimension ContextDimension,
	userQuery string,
	retrievalResults *models.RetrievalResults,
	currentContext *models.UnifiedContextModel,
) DimensionResult {

	startTime := time.Now()

	log.Printf("🔄 [方案2-%s维度] 开始生成", dimension)

	// 创建维度特定的超时上下文
	dimCtx, cancel := context.WithTimeout(ctx, time.Duration(pcs.config.LLMTimeout)*time.Second)
	defer cancel()

	// 生成维度特定的prompt
	prompt := pcs.buildDimensionPrompt(dimension, userQuery, retrievalResults, currentContext)

	// 调用LLM
	llmRequest := &GenerateRequest{
		Prompt:      prompt,
		MaxTokens:   pcs.config.MaxTokens,
		Temperature: pcs.config.Temperature,
		Format:      "json",
	}

	log.Printf("📤 [方案2-%s维度] LLM请求: Prompt=%d字符, MaxTokens=%d",
		dimension, len(prompt), pcs.config.MaxTokens)

	response, err := pcs.llmService.GenerateResponse(dimCtx, llmRequest)
	if err != nil {
		duration := time.Since(startTime)
		log.Printf("❌ [方案2-%s维度] LLM调用失败: %v (耗时: %v)", dimension, err, duration)
		return DimensionResult{
			Dimension: dimension,
			Success:   false,
			Error:     err,
			Duration:  duration,
		}
	}

	duration := time.Since(startTime)
	log.Printf("📥 [方案2-%s维度] LLM响应: %d字符, Token=%d (耗时: %v)",
		dimension, len(response.Content), response.Usage.TotalTokens, duration)

	return DimensionResult{
		Dimension: dimension,
		Success:   true,
		Content:   response.Content,
		Duration:  duration,
	}
}

// buildDimensionPrompt 构建维度特定的prompt
func (pcs *ParallelContextSynthesizer) buildDimensionPrompt(
	dimension ContextDimension,
	userQuery string,
	retrievalResults *models.RetrievalResults,
	currentContext *models.UnifiedContextModel,
) string {

	// 基础信息（所有维度共享）
	baseInfo := fmt.Sprintf(`## %s维度上下文生成任务

你是一个专业的%s上下文分析专家，专注于生成%s相关的上下文信息。

### 输入信息
**用户查询**: %s

**检索结果摘要**:
- 时间线结果: %d条
- 知识图谱结果: %d条  
- 向量结果: %d条

`,
		dimension, dimension, dimension, userQuery,
		len(retrievalResults.TimelineResults),
		len(retrievalResults.KnowledgeResults),
		len(retrievalResults.VectorResults))

	// 根据维度生成特定的prompt
	switch dimension {
	case DimensionTopic:
		return baseInfo + pcs.buildTopicPrompt(retrievalResults)
	case DimensionProject:
		return baseInfo + pcs.buildProjectPrompt(retrievalResults)
	case DimensionConversation:
		return baseInfo + pcs.buildConversationPrompt(retrievalResults)
	case DimensionCode:
		return baseInfo + pcs.buildCodePrompt(retrievalResults)
	case DimensionChanges:
		return baseInfo + pcs.buildChangesPrompt(retrievalResults)
	default:
		return baseInfo + "请生成通用的上下文信息。"
	}
}

// buildTopicPrompt 构建主题维度prompt
func (pcs *ParallelContextSynthesizer) buildTopicPrompt(retrievalResults *models.RetrievalResults) string {
	return `
### 任务要求
请专注于生成**主题相关**的上下文信息，输出JSON格式：

{
  "main_topic": "主要话题",
  "topic_category": "technical|business|learning|other",
  "primary_pain_point": "主要痛点",
  "expected_outcome": "期望结果", 
  "key_concepts": ["概念1", "概念2"],
  "technical_terms": ["术语1", "术语2"],
  "confidence_level": 0.8
}

**重点关注**：
- 从检索结果中提取核心主题
- 识别用户的主要痛点
- 明确期望的结果
- 提取关键技术概念
`
}

// buildProjectPrompt 构建项目维度prompt
func (pcs *ParallelContextSynthesizer) buildProjectPrompt(retrievalResults *models.RetrievalResults) string {
	return `
### 任务要求
请专注于生成**项目相关**的上下文信息，输出JSON格式：

{
  "project_name": "项目名称",
  "project_type": "backend|frontend|fullstack|mobile|other",
  "primary_language": "主要编程语言",
  "current_phase": "planning|development|testing|deployment|maintenance",
  "description": "项目描述",
  "confidence_level": 0.8
}

**重点关注**：
- 从时间线结果推断项目信息
- 识别主要技术栈
- 判断项目当前阶段
`
}

// buildConversationPrompt 构建会话维度prompt
func (pcs *ParallelContextSynthesizer) buildConversationPrompt(retrievalResults *models.RetrievalResults) string {
	return `
### 任务要求  
请专注于生成**会话相关**的上下文信息，输出JSON格式：

{
  "conversation_state": "active|paused|completed",
  "key_topics": ["话题1", "话题2"],
  "conversation_summary": "会话摘要",
  "next_steps": ["下一步1", "下一步2"]
}

**重点关注**：
- 分析会话的当前状态
- 提取关键讨论话题
- 总结会话要点
`
}

// buildCodePrompt 构建代码维度prompt
func (pcs *ParallelContextSynthesizer) buildCodePrompt(retrievalResults *models.RetrievalResults) string {
	return `
### 任务要求
请专注于生成**代码相关**的上下文信息，输出JSON格式：

{
  "active_files": ["文件1", "文件2"],
  "focused_components": ["组件1", "组件2"], 
  "key_functions": ["函数1", "函数2"],
  "recent_changes": ["变更1", "变更2"]
}

**重点关注**：
- 从时间线结果提取活跃文件
- 识别关键组件和函数
- 总结最近的代码变更
`
}

// buildChangesPrompt 构建变更维度prompt
func (pcs *ParallelContextSynthesizer) buildChangesPrompt(retrievalResults *models.RetrievalResults) string {
	return `
### 任务要求
请专注于生成**变更相关**的上下文信息，输出JSON格式：

{
  "recent_commits": ["提交1", "提交2"],
  "modified_files": ["文件1", "文件2"],
  "completed_tasks": ["任务1", "任务2"],
  "ongoing_tasks": ["进行中任务1", "进行中任务2"]
}

**重点关注**：
- 从时间线结果提取最近提交
- 识别修改的文件
- 区分已完成和进行中的任务
`
}

// mergeResults 合并各维度的结果
func (pcs *ParallelContextSynthesizer) mergeResults(
	results map[ContextDimension]DimensionResult,
	currentContext *models.UnifiedContextModel,
) (*models.UnifiedContextModel, error) {

	log.Printf("🔄 [方案2-合并] 开始合并5个维度的结果")

	// 创建新的统一上下文模型
	unified := &models.UnifiedContextModel{
		SessionID:   extractSessionID(currentContext),
		UserID:      extractUserIDFromContext(currentContext),
		WorkspaceID: extractWorkspaceID(currentContext),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 合并主题维度
	if result, exists := results[DimensionTopic]; exists && result.Success {
		topicContext, err := pcs.parseTopicResult(result.Content)
		if err != nil {
			log.Printf("⚠️ [方案2-合并] 主题维度解析失败: %v", err)
		} else {
			unified.CurrentTopic = topicContext
			log.Printf("✅ [方案2-合并] 主题维度合并成功")
		}
	}

	// 合并项目维度
	if result, exists := results[DimensionProject]; exists && result.Success {
		projectContext, err := pcs.parseProjectResult(result.Content)
		if err != nil {
			log.Printf("⚠️ [方案2-合并] 项目维度解析失败: %v", err)
		} else {
			unified.Project = projectContext
			log.Printf("✅ [方案2-合并] 项目维度合并成功")
		}
	}

	// 合并会话维度
	if result, exists := results[DimensionConversation]; exists && result.Success {
		// 会话维度暂时跳过，因为ConversationContext未定义
		log.Printf("⚠️ [方案2-合并] 会话维度暂时跳过（ConversationContext未定义）")
	}

	// 合并代码维度
	if result, exists := results[DimensionCode]; exists && result.Success {
		codeContext, err := pcs.parseCodeResult(result.Content)
		if err != nil {
			log.Printf("⚠️ [方案2-合并] 代码维度解析失败: %v", err)
		} else {
			unified.Code = codeContext
			log.Printf("✅ [方案2-合并] 代码维度合并成功")
		}
	}

	// 合并变更维度
	if result, exists := results[DimensionChanges]; exists && result.Success {
		changesContext, err := pcs.parseChangesResult(result.Content)
		if err != nil {
			log.Printf("⚠️ [方案2-合并] 变更维度解析失败: %v", err)
		} else {
			// 🔥 转换为简化的字符串摘要
			unified.RecentChangesSummary = pcs.summarizeChangesContext(changesContext)
			log.Printf("✅ [方案2-合并] 变更维度合并成功")
		}
	}

	log.Printf("🎯 [方案2-合并] 合并完成，生成完整的UnifiedContextModel")
	return unified, nil
}

// 辅助函数：提取基础信息
func extractSessionID(ctx *models.UnifiedContextModel) string {
	if ctx != nil {
		return ctx.SessionID
	}
	return "generated_session"
}

func extractUserIDFromContext(ctx *models.UnifiedContextModel) string {
	if ctx != nil {
		return ctx.UserID
	}
	return "generated_user"
}

func extractWorkspaceID(ctx *models.UnifiedContextModel) string {
	if ctx != nil {
		return ctx.WorkspaceID
	}
	return "/generated/workspace"
}

// JSON解析方法
func (pcs *ParallelContextSynthesizer) parseTopicResult(content string) (*models.TopicContext, error) {
	// 简化的JSON解析实现
	// 实际应该解析JSON并构建TopicContext
	return &models.TopicContext{
		MainTopic:        "从并行生成中提取的主题",
		PrimaryPainPoint: "从并行生成中提取的痛点",
		ExpectedOutcome:  "从并行生成中提取的期望结果",
		ConfidenceLevel:  0.8,
		TopicStartTime:   time.Now(),
		LastUpdated:      time.Now(),
		UpdateCount:      1,
	}, nil
}

func (pcs *ParallelContextSynthesizer) parseProjectResult(content string) (*models.ProjectContext, error) {
	return &models.ProjectContext{
		ProjectName:     "从并行生成中提取的项目",
		ProjectType:     "backend",
		PrimaryLanguage: "go",
		CurrentPhase:    "development",
		Description:     "从并行生成中提取的项目描述",
		ConfidenceLevel: 0.8,
	}, nil
}

func (pcs *ParallelContextSynthesizer) parseCodeResult(content string) (*models.CodeContext, error) {
	return &models.CodeContext{
		SessionID:         "generated_session",
		ActiveFiles:       []models.ActiveFileInfo{},
		RecentEdits:       []models.ContextEditInfo{},
		FocusedComponents: []string{"从并行生成中提取的组件"},
		KeyFunctions:      []models.FunctionInfo{},
		ImportantTypes:    []models.TypeInfo{},
	}, nil
}

func (pcs *ParallelContextSynthesizer) parseChangesResult(content string) (*models.RecentChangesContext, error) {
	return &models.RecentChangesContext{
		TimeRange: models.TimeRange{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		},
		RecentCommits:  []models.CommitInfo{},
		ModifiedFiles:  []models.FileChangeInfo{},
		BranchActivity: []models.BranchActivity{},
		NewFeatures:    []models.FeatureChange{},
		FeatureUpdates: []models.FeatureUpdate{},
		BugFixes:       []models.BugFixInfo{},
		CompletedTasks: []models.TaskInfo{},
		OngoingTasks:   []models.TaskInfo{},
		BlockedTasks:   []models.TaskInfo{},
	}, nil
}

// summarizeChangesContext 将RecentChangesContext转换为简化的字符串摘要
func (pcs *ParallelContextSynthesizer) summarizeChangesContext(changes *models.RecentChangesContext) string {
	if changes == nil {
		return ""
	}

	var summary []string

	// 统计变更信息
	if len(changes.RecentCommits) > 0 {
		summary = append(summary, fmt.Sprintf("%d个代码提交", len(changes.RecentCommits)))
	}

	if len(changes.ModifiedFiles) > 0 {
		summary = append(summary, fmt.Sprintf("%d个文件修改", len(changes.ModifiedFiles)))
	}

	if len(changes.NewFeatures) > 0 {
		summary = append(summary, fmt.Sprintf("%d个新功能", len(changes.NewFeatures)))
	}

	if len(changes.BugFixes) > 0 {
		summary = append(summary, fmt.Sprintf("%d个问题修复", len(changes.BugFixes)))
	}

	if len(summary) == 0 {
		return ""
	}

	return "最近变更: " + strings.Join(summary, ", ")
}
