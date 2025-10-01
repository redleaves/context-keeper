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

// AnalysisStrategyFactory 策略工厂
type AnalysisStrategyFactory struct {
	llmClient llm.LLMClient
	config    *SemanticAnalysisConfig
}

// NewAnalysisStrategyFactory 创建策略工厂
func NewAnalysisStrategyFactory(llmClient llm.LLMClient, config *SemanticAnalysisConfig) *AnalysisStrategyFactory {
	return &AnalysisStrategyFactory{
		llmClient: llmClient,
		config:    config,
	}
}

// GetStrategy 获取策略实例
func (f *AnalysisStrategyFactory) GetStrategy(strategyType string) SemanticAnalysisStrategy {
	switch strategyType {
	case "lightweight":
		return &LightweightAnalysisStrategy{
			llmClient: f.llmClient,
			config:    f.config,
		}
	case "deepIntent":
		return &DeepIntentAnalysisStrategy{
			llmClient: f.llmClient,
			config:    f.config,
		}
	default:
		// 默认使用轻量策略
		return &LightweightAnalysisStrategy{
			llmClient: f.llmClient,
			config:    f.config,
		}
	}
}

// LightweightAnalysisStrategy 轻量拆解策略（宽召回）
type LightweightAnalysisStrategy struct {
	llmClient llm.LLMClient
	config    *SemanticAnalysisConfig
}

// GetStrategyName 获取策略名称
func (s *LightweightAnalysisStrategy) GetStrategyName() string {
	return "lightweight"
}

// AnalyzeQuery 轻量分析查询
func (s *LightweightAnalysisStrategy) AnalyzeQuery(ctx context.Context, query string, contextInfo *ContextInfo) (*SemanticAnalysisResult, error) {
	log.Printf("🔍 [轻量策略] 开始分析查询: %s", query)
	log.Printf("📋 [轻量策略] 上下文信息: %+v", contextInfo)

	prompt := s.buildLightweightPrompt(query)
	log.Printf("📝 [轻量策略] 完整Prompt内容:\n%s", prompt)

	// 调用LLM
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
		Format:      "json",
		Model:       s.config.Model,
		Metadata: map[string]interface{}{
			"strategy": "lightweight",
			"task":     "dimension_extraction",
		},
	}

	llmResponse, err := s.llmClient.Complete(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("轻量策略LLM调用失败: %w", err)
	}

	log.Printf("✅ [轻量策略] LLM调用完成，Token使用: %d", llmResponse.TokensUsed)
	log.Printf("📄 [轻量策略] LLM原始响应内容: %s", llmResponse.Content)

	// 解析响应
	result, err := parseLLMResponse(llmResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("轻量策略响应解析失败: %w", err)
	}

	result.TokenUsage = llmResponse.TokensUsed
	return result, nil
}

// parseLLMResponse 解析LLM响应（精细化版本）
func parseLLMResponse(content string) (*SemanticAnalysisResult, error) {
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

	// 解析精细化的JSON结构
	var rawResult struct {
		TimelineRecall *struct {
			StartTime string `json:"start_time"`
			EndTime   string `json:"end_time"`
		} `json:"timeline_recall"` // 🆕 新增时间回忆字段
		IntentAnalysis struct {
			CoreIntent      string   `json:"core_intent"`
			IntentType      string   `json:"intent_type"`
			IntentCategory  string   `json:"intent_category"`
			KeyConcepts     []string `json:"key_concepts"`
			TimeScope       string   `json:"time_scope"`
			UrgencyLevel    string   `json:"urgency_level"`
			ExpectedOutcome string   `json:"expected_outcome"`
		} `json:"intent_analysis"`
		KeyExtraction struct {
			ProjectKeywords   []string `json:"project_keywords"`
			TechnicalKeywords []string `json:"technical_keywords"`
			BusinessKeywords  []string `json:"business_keywords"`
			TimeKeywords      []string `json:"time_keywords"`
			ActionKeywords    []string `json:"action_keywords"`
		} `json:"key_extraction"`
		RetrievalStrategy struct {
			TimelineQueries []struct {
				QueryText  string   `json:"query_text"`
				TimeRange  string   `json:"time_range"`
				EventTypes []string `json:"event_types"`
				Priority   int      `json:"priority"`
			} `json:"timeline_queries"`
			KnowledgeQueries []struct {
				QueryText     string   `json:"query_text"`
				ConceptTypes  []string `json:"concept_types"`
				RelationTypes []string `json:"relation_types"`
				Priority      int      `json:"priority"`
			} `json:"knowledge_queries"`
			VectorQueries []struct {
				QueryText           string  `json:"query_text"`
				SemanticFocus       string  `json:"semantic_focus"`
				SimilarityThreshold float64 `json:"similarity_threshold"`
				Priority            int     `json:"priority"`
			} `json:"vector_queries"`
		} `json:"retrieval_strategy"`
		ConfidenceLevel float64 `json:"confidence_level"`
	}

	if err := json.Unmarshal([]byte(content), &rawResult); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w, 内容: %s", err, content)
	}

	// 🔥 优先判断时间回忆查询 - 必须有有效的时间范围才走专用逻辑
	if rawResult.TimelineRecall != nil &&
		rawResult.TimelineRecall.StartTime != "" &&
		rawResult.TimelineRecall.EndTime != "" {
		log.Printf("🕒 [轻量策略] 识别到时间回忆查询: %s 到 %s",
			rawResult.TimelineRecall.StartTime, rawResult.TimelineRecall.EndTime)

		// 🔥 时间回忆查询走专用逻辑，直接返回简化结果
		return &SemanticAnalysisResult{
			Intent:     models.IntentType("time_recall"), // 专用意图类型
			Confidence: rawResult.ConfidenceLevel,
			Categories: []string{"时间回忆"},
			Keywords:   []string{}, // 时间回忆不需要关键词
			Entities:   []models.Entity{},
			Queries:    &models.MultiDimensionalQuery{}, // 空查询，走专用路径
			SmartAnalysis: &models.SmartAnalysisResult{
				TimelineRecall: &models.TimelineRecall{
					StartTime: rawResult.TimelineRecall.StartTime,
					EndTime:   rawResult.TimelineRecall.EndTime,
				},
			},
		}, nil
	}

	// 🔥 非时间回忆查询走原有逻辑（保持向后兼容）
	result := &SemanticAnalysisResult{
		Intent:     models.IntentType(rawResult.IntentAnalysis.IntentType),
		Confidence: rawResult.ConfidenceLevel,
		Categories: []string{rawResult.IntentAnalysis.IntentCategory}, // 简化为单个分类
		Keywords:   append(rawResult.KeyExtraction.TechnicalKeywords, rawResult.KeyExtraction.ProjectKeywords...),
		Entities:   []models.Entity{}, // 暂时为空，后续可以从关键词生成
		Queries: &models.MultiDimensionalQuery{
			ContextQueries:   extractQueryTexts(rawResult.RetrievalStrategy.TimelineQueries),
			TimelineQueries:  extractQueryTexts(rawResult.RetrievalStrategy.TimelineQueries),
			KnowledgeQueries: extractKnowledgeQueryTexts(rawResult.RetrievalStrategy.KnowledgeQueries),
			VectorQueries:    extractVectorQueryTexts(rawResult.RetrievalStrategy.VectorQueries),
		},
	}

	return result, nil
}

// 辅助函数：提取时间线查询文本
func extractQueryTexts(queries []struct {
	QueryText  string   `json:"query_text"`
	TimeRange  string   `json:"time_range"`
	EventTypes []string `json:"event_types"`
	Priority   int      `json:"priority"`
}) []string {
	texts := make([]string, len(queries))
	for i, q := range queries {
		texts[i] = q.QueryText
	}
	return texts
}

// 辅助函数：提取知识图谱查询文本
func extractKnowledgeQueryTexts(queries []struct {
	QueryText     string   `json:"query_text"`
	ConceptTypes  []string `json:"concept_types"`
	RelationTypes []string `json:"relation_types"`
	Priority      int      `json:"priority"`
}) []string {
	texts := make([]string, len(queries))
	for i, q := range queries {
		texts[i] = q.QueryText
	}
	return texts
}

// 辅助函数：提取向量查询文本
func extractVectorQueryTexts(queries []struct {
	QueryText           string  `json:"query_text"`
	SemanticFocus       string  `json:"semantic_focus"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	Priority            int     `json:"priority"`
}) []string {
	texts := make([]string, len(queries))
	for i, q := range queries {
		texts[i] = q.QueryText
	}
	return texts
}

// buildLightweightPrompt 构建轻量策略的Prompt（包含时间回忆查询识别功能）
func (s *LightweightAnalysisStrategy) buildLightweightPrompt(query string) string {
	// 获取当前日期作为时间解析的基准
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`## 用户意图分析和查询拆解任务

**📅 当前日期：%s** （时间解析的基准日期）

你是一个专业的意图分析专家，需要分析用户的查询意图，并生成多维度检索策略。

### 用户输入
**原始查询**: %s
**查询类型**: query
**上下文信息**: 无

### 分析目标
1. **核心意图识别**: 用户真正想要什么？
2. **关键信息提取**: 时间、项目、技术、业务等关键词
3. **检索策略生成**: 为三个维度生成精确的检索查询

### 输出要求
请严格按照以下JSON格式输出：

{
  "intent_analysis": {
    "core_intent": "<用户的核心意图>",
    "intent_type": "<query/command/analysis/review/planning/learning>",
    "intent_category": "<technical/project/business/troubleshooting>",
    "key_concepts": ["<关键概念1>", "<关键概念2>"],
    "time_scope": "<时间范围: recent/today/yesterday/week/month/all>",
    "urgency_level": "<high/medium/low>",
    "expected_outcome": "<用户期望的结果>"
  },
  "key_extraction": {
    "project_keywords": ["<项目相关关键词>"],
    "technical_keywords": ["<技术相关关键词>"],
    "business_keywords": ["<业务相关关键词>"],
    "time_keywords": ["<时间相关关键词>"],
    "action_keywords": ["<动作相关关键词>"]
  },
  "timeline_recall": {
    "start_time": "<YYYY-MM-DD HH:mm:ss格式，仅当识别为时间回忆查询时填充>",
    "end_time": "<YYYY-MM-DD HH:mm:ss格式，仅当识别为时间回忆查询时填充>"
  },
  "retrieval_strategy": {
    "timeline_queries": [
      {
        "query_text": "<时间线检索查询>",
        "time_range": "<时间范围>",
        "event_types": ["<事件类型>"],
        "priority": <1-5优先级>
      }
    ],
    "knowledge_queries": [
      {
        "query_text": "<知识图谱检索查询>",
        "concept_types": ["<概念类型>"],
        "relation_types": ["<关系类型>"],
        "priority": <1-5优先级>
      }
    ],
    "vector_queries": [
      {
        "query_text": "<向量检索查询>",
        "semantic_focus": "<语义焦点>",
        "similarity_threshold": <相似度阈值>,
        "priority": <1-5优先级>
      }
    ]
  },
  "confidence_level": <0-1的置信度>
}

## 🕒 时间回忆查询识别（重要！新增）

**🎯 时间回忆查询识别规则**：
当用户查询符合以下特征时，**必须使用专用格式**：

**识别触发条件**：
1. **时间范围词汇** + **回忆动词**：
   - "回忆一下昨天..."、"想起上周..."、"总结今天..."、"回顾最近..."
   
2. **时间点** + **完成状态查询**：
   - "昨天完成了什么"、"今天做了哪些事"、"上午干了什么"、"这周的进展"
   
3. **具体日期** + **工作状态**：
   - 用户明确指定某个日期或时间段的工作总结/进展/完成情况
   - 示例格式（不是固定日期）："X月X日的工作总结"、"YYYY年MM月DD日的开发情况"

**🔥 时间回忆查询专用输出格式（与标准格式互斥）**：
{
  "timeline_recall": {
    "start_time": "YYYY-MM-DD HH:mm:ss",
    "end_time": "YYYY-MM-DD HH:mm:ss"
  },
  "intent_analysis": null,
  "key_extraction": null,
  "retrieval_strategy": null,
  "confidence_level": 0.95
}

**🔥 时间解析重要说明**：
- 如果识别为时间回忆类需求，**必须使用上述专用格式**
- timeline_recall 与其他分析字段完全互斥，其他字段必须设为null
- 时间格式统一使用 YYYY-MM-DD HH:mm:ss

**⏰ 时间词汇解析规则（关键！）**：
请根据用户查询中的时间词汇，动态计算出正确的时间范围：

1. **相对时间词汇**（基于用户提问的实际当前时间动态计算）：
   - "今天" → 当前日期的00:00:00到23:59:59
   - "今天上午" → 当前日期的00:00:00到12:00:00
   - "今天下午" → 当前日期的12:00:00到23:00:00
   - "昨天" → 当前日期前一天的00:00:00到23:59:59
   - "上周" → 上一周的周一到周日

2. **明确日期**（直接解析用户指定的具体日期）：
   - 解析用户明确提到的年月日，转换为对应的时间范围

**⚠️ 重要：绝对不要使用任何固定日期！必须根据用户实际查询时的当前时间动态计算相对时间！**

请只返回JSON，不要包含其他文本
`, currentDate, query)
}

// DeepIntentAnalysisStrategy 深度意图策略（精召回）
type DeepIntentAnalysisStrategy struct {
	llmClient llm.LLMClient
	config    *SemanticAnalysisConfig
}

// GetStrategyName 获取策略名称
func (s *DeepIntentAnalysisStrategy) GetStrategyName() string {
	return "deepIntent"
}

// AnalyzeQuery 深度分析查询
func (s *DeepIntentAnalysisStrategy) AnalyzeQuery(ctx context.Context, query string, contextInfo *ContextInfo) (*SemanticAnalysisResult, error) {
	log.Printf("🎯 [深度策略] 开始分析查询: %s", query)
	log.Printf("📋 [深度策略] 详细上下文信息:")
	if contextInfo != nil {
		log.Printf("   🔄 最近对话: %s", contextInfo.RecentConversation)
		log.Printf("   🎯 会话主题: %s", contextInfo.SessionTopic)
		log.Printf("   📁 当前项目: %s", contextInfo.CurrentProject)
		log.Printf("   🏗️ 工作空间: %s", contextInfo.WorkspaceContext)
		log.Printf("   📚 相关历史: %s", contextInfo.RelevantHistory)
		log.Printf("   ⚙️ 技术栈: %v", contextInfo.TechStack)
		log.Printf("   📝 当前任务: %s", contextInfo.CurrentTask)
	} else {
		log.Printf("   ❌ 上下文信息为空")
	}

	prompt := s.buildDeepIntentPrompt(query, contextInfo)
	log.Printf("📝 [深度策略] 完整Prompt内容:\n%s", prompt)

	// 调用LLM
	llmRequest := &llm.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
		Format:      "json",
		Model:       s.config.Model,
		Metadata: map[string]interface{}{
			"strategy": "deepIntent",
			"task":     "intent_analysis",
		},
	}

	llmResponse, err := s.llmClient.Complete(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("深度策略LLM调用失败: %w", err)
	}

	log.Printf("✅ [深度策略] LLM调用完成，Token使用: %d", llmResponse.TokensUsed)
	log.Printf("📄 [深度策略] LLM原始响应内容: %s", llmResponse.Content)

	// 解析响应
	result, err := parseLLMResponse(llmResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("深度策略响应解析失败: %w", err)
	}

	result.TokenUsage = llmResponse.TokensUsed
	return result, nil
}

// buildDeepIntentPrompt 构建深度意图策略的Prompt
func (s *DeepIntentAnalysisStrategy) buildDeepIntentPrompt(query string, contextInfo *ContextInfo) string {
	contextSection := ""
	if contextInfo != nil {
		contextSection = fmt.Sprintf(`
## 上下文信息
**最近对话**: %s
**会话主题**: %s
**项目背景**: %s

请充分利用上下文信息，结合用户的历史需求和当前项目背景，精准理解用户的真实意图。`,
			contextInfo.ShortTermMemory,
			contextInfo.LongTermMemory,
			contextInfo.SessionState)
	}

	return fmt.Sprintf(`你是一个资深的智能上下文分析专家，擅长结合上下文信息进行精准的意图理解和检索策略制定。

## 任务目标
基于用户查询和丰富的上下文信息，进行深度意图分析，生成高精度的检索策略，确保检索结果的相关性和准确性。

## 用户查询
"%s"

%s

## 分析要求
请结合上下文信息进行深度分析，返回JSON格式结果：

### 1. 精准意图识别 (intent)
基于上下文选择最精确的意图类型：
- information_query: 信息查询
- knowledge_lookup: 知识查询
- problem_solving: 问题解决
- troubleshooting: 故障排查
- code_generation: 代码生成
- content_creation: 内容创建
- code_analysis: 代码分析
- concept_explanation: 概念解释
- how_to_guide: 操作指导
- best_practices: 最佳实践
- clarification: 澄清确认
- follow_up: 后续讨论

### 2. 高置信度评估 (confidence)
基于上下文信息的丰富程度和匹配度，给出0-1之间的置信度

### 3. 精确领域分类 (categories)
结合上下文确定的精确技术领域或业务领域

### 4. 核心关键词提取 (keywords)
基于上下文筛选出最相关的核心关键词，避免冗余

### 5. 上下文实体确认 (entities)
结合上下文信息确认的高相关性实体，格式：{"text": "实体名", "type": "实体类型", "confidence": 置信度}

### 6. 精准检索策略 (queries)
基于上下文生成高针对性的检索查询：
- context_queries: 结合上下文的项目和任务相关查询
- timeline_queries: 基于时间上下文的查询
- knowledge_queries: 上下文相关的知识实体查询
- vector_queries: 结合上下文的精准语义查询

## 返回格式示例
{
  "intent_analysis": {
    "core_intent": "<用户的核心意图>",
    "intent_type": "<query/command/analysis/review/planning/learning>",
    "intent_category": "<technical/project/business/troubleshooting>",
    "key_concepts": ["<关键概念1>", "<关键概念2>"],
    "time_scope": "<时间范围: recent/today/yesterday/week/month/all>",
    "urgency_level": "<high/medium/low>",
    "expected_outcome": "<用户期望的结果>"
  },
  "key_extraction": {
    "project_keywords": ["<项目相关关键词>"],
    "technical_keywords": ["<技术相关关键词>"],
    "business_keywords": ["<业务相关关键词>"],
    "time_keywords": ["<时间相关关键词>"],
    "action_keywords": ["<动作相关关键词>"]
  },
  "retrieval_strategy": {
    "timeline_queries": [
      {
        "query_text": "<时间线检索查询>",
        "time_range": "<时间范围>",
        "event_types": ["<事件类型>"],
        "priority": <1-5优先级>
      }
    ],
    "knowledge_queries": [
      {
        "query_text": "<知识图谱检索查询>",
        "concept_types": ["<概念类型>"],
        "relation_types": ["<关系类型>"],
        "priority": <1-5优先级>
      }
    ],
    "vector_queries": [
      {
        "query_text": "<向量检索查询>",
        "semantic_focus": "<语义焦点>",
        "similarity_threshold": <相似度阈值>,
        "priority": <1-5优先级>
      }
    ]
  },
  "confidence_level": <0-1的置信度>
}

请只返回JSON，不要包含其他文本：`, query, contextSection)
}
