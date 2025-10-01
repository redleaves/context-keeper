package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// buildIntentAnalysisPrompt 构建意图分析Prompt
func (s *WideRecallService) buildIntentAnalysisPrompt(userQuery string) string {
	return fmt.Sprintf(`## 用户意图分析和查询拆解任务

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
}`, userQuery)
}

// buildContextSynthesisPrompt 构建上下文合成Prompt
func (s *WideRecallService) buildContextSynthesisPrompt(req *models.ContextSynthesisRequest) string {
	// 构建当前上下文信息
	currentContextStr := "无（首次构建）"
	if req.CurrentContext != nil && req.CurrentContext.CurrentTopic != nil {
		currentContextStr = fmt.Sprintf(`
**当前主题**: %s
**项目信息**: %s
**最近变更**: 存在变更记录
**代码焦点**: 存在代码上下文`,
			req.CurrentContext.CurrentTopic.MainTopic,
			getProjectInfo(req.CurrentContext.Project))
	}

	// 构建检索结果信息
	retrievalResultsStr := s.buildRetrievalResultsString(req.RetrievalResults)

	return fmt.Sprintf(`## 上下文评估与合成任务

你是一个专业的上下文合成专家，需要将宽召回检索结果与现有上下文进行智能评估和融合。

### 输入信息

#### 用户查询
**原始查询**: %s
**意图分析**: %s

#### 现有上下文
%s

#### 宽召回检索结果
%s

### 评估与合成任务

#### 1. 评估是否需要更新上下文
- 检索结果是否包含新的有价值信息？
- 用户意图是否发生变化？
- 现有上下文是否需要修正或补充？

#### 2. 信息质量评估
- 检索结果的可靠性和相关性
- 信息之间的一致性和冲突检测
- 信息的时效性和重要性

#### 3. 上下文合成策略
- 保留有效的现有信息
- 融合新的高质量信息
- 解决信息冲突
- 填补信息缺口

### 输出要求

{
  "evaluation_result": {
    "should_update": <true/false>,
    "update_confidence": <0-1的置信度>,
    "evaluation_reason": "<评估原因>",
    "semantic_changes": [
      {
        "dimension": "<topic/project/code/conversation>",
        "change_type": "<shift/expand/refine/contradict>",
        "change_description": "<变化描述>",
        "evidence": ["<证据1>", "<证据2>"]
      }
    ]
  },
  "synthesized_context": {
    "current_topic": {
      "main_topic": "<主要话题>",
      "topic_category": "<technical/project/business/learning/troubleshooting>",
      "user_intent": {
        "intent_type": "<query/command/analysis/review/planning/learning>",
        "intent_description": "<意图描述>",
        "priority": "<high/medium/low>"
      },
      "primary_pain_point": "<主要痛点>",
      "expected_outcome": "<期望结果>",
      "key_concepts": [
        {
          "concept_name": "<概念名称>",
          "definition": "<概念定义>",
          "importance": <0-1重要程度>,
          "source": "<来源>"
        }
      ],
      "confidence_level": <0-1置信度>
    },
    "project": {
      "project_name": "<项目名称>",
      "project_type": "<go/nodejs/python/java/rust/typescript/other>",
      "description": "<项目描述>",
      "primary_language": "<主要编程语言>",
      "current_phase": "<planning/development/testing/deployment/maintenance>",
      "confidence_level": <0-1置信度>
    },
    "recent_changes": {
      "changes_summary": "<变更总结>",
      "change_count": <变更数量>,
      "confidence_level": <0-1置信度>
    },
    "code": {
      "focused_components": [
        {
          "component_name": "<组件名称>",
          "focus_reason": "<关注原因>",
          "focus_level": <0-1关注程度>
        }
      ],
      "confidence_level": <0-1置信度>
    },
    "conversation": {
      "conversation_state": "<active/waiting/completed/paused>",
      "key_topics": ["<关键话题>"],
      "message_count": <消息数量>,
      "confidence_level": <0-1置信度>
    }
  },
  "synthesis_metadata": {
    "information_sources": {
      "timeline_contribution": <0-1贡献度>,
      "knowledge_contribution": <0-1贡献度>,
      "vector_contribution": <0-1贡献度>
    },
    "quality_assessment": {
      "overall_quality": <0-1质量评分>,
      "information_conflicts": ["<冲突描述>"],
      "information_gaps": ["<缺口描述>"]
    },
    "synthesis_notes": "<合成过程说明>"
  }
}`,
		req.UserQuery,
		s.buildIntentAnalysisString(req.IntentAnalysis),
		currentContextStr,
		retrievalResultsStr)
}

// buildRetrievalResultsString 构建检索结果字符串
func (s *WideRecallService) buildRetrievalResultsString(results *models.RetrievalResults) string {
	var builder strings.Builder

	// 时间线检索结果
	builder.WriteString(fmt.Sprintf("**时间线检索结果** (%d条):\n", results.TimelineCount))
	for i, result := range results.TimelineResults {
		if i >= 5 { // 只显示前5条
			builder.WriteString("...\n")
			break
		}
		builder.WriteString(fmt.Sprintf("- [%s] %s: %s\n  内容: %s\n  重要性: %.2f\n",
			result.Timestamp.Format("2006-01-02 15:04"),
			result.EventType,
			result.Title,
			truncateStringWR(result.Content, 100),
			result.ImportanceScore))
	}

	// 知识图谱检索结果
	builder.WriteString(fmt.Sprintf("\n**知识图谱检索结果** (%d条):\n", results.KnowledgeCount))
	for i, result := range results.KnowledgeResults {
		if i >= 5 { // 只显示前5条
			builder.WriteString("...\n")
			break
		}
		relatedConcepts := make([]string, len(result.RelatedConcepts))
		for j, related := range result.RelatedConcepts {
			relatedConcepts[j] = related.ConceptName
		}
		builder.WriteString(fmt.Sprintf("- 概念: %s (%s)\n  描述: %s\n  相关概念: %s\n",
			result.ConceptName,
			result.ConceptType,
			truncateStringWR(result.Description, 100),
			strings.Join(relatedConcepts, ", ")))
	}

	// 向量检索结果
	builder.WriteString(fmt.Sprintf("\n**向量检索结果** (%d条):\n", results.VectorCount))
	for i, result := range results.VectorResults {
		if i >= 5 { // 只显示前5条
			builder.WriteString("...\n")
			break
		}
		builder.WriteString(fmt.Sprintf("- 相似度: %.2f\n  内容: %s\n  来源: %s\n",
			result.Similarity,
			truncateStringWR(result.Content, 100),
			result.Source))
	}

	return builder.String()
}

// buildIntentAnalysisString 构建意图分析字符串
func (s *WideRecallService) buildIntentAnalysisString(intentAnalysis *models.WideRecallIntentAnalysis) string {
	if intentAnalysis == nil {
		return "无意图分析结果"
	}

	return fmt.Sprintf("核心意图: %s, 意图类型: %s, 紧急程度: %s",
		intentAnalysis.IntentAnalysis.CoreIntent,
		string(intentAnalysis.IntentAnalysis.IntentType),
		string(intentAnalysis.IntentAnalysis.UrgencyLevel))
}

// getProjectInfo 获取项目信息字符串
func getProjectInfo(project *models.ProjectContext) string {
	if project == nil {
		return "无项目信息"
	}
	return fmt.Sprintf("%s (%s) - %s", project.ProjectName, string(project.ProjectType), project.Description)
}

// truncateStringWR 截断字符串（宽召回专用）
func truncateStringWR(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseIntentAnalysisResponse 解析意图分析响应
func (s *WideRecallService) parseIntentAnalysisResponse(response string) (*models.WideRecallIntentAnalysis, error) {
	var result models.WideRecallIntentAnalysis

	// 清理响应内容，移除可能的markdown标记
	cleanResponse := strings.TrimSpace(response)
	if strings.HasPrefix(cleanResponse, "```json") {
		cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
	}
	if strings.HasSuffix(cleanResponse, "```") {
		cleanResponse = strings.TrimSuffix(cleanResponse, "```")
	}
	cleanResponse = strings.TrimSpace(cleanResponse)

	err := json.Unmarshal([]byte(cleanResponse), &result)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 设置分析时间
	result.AnalysisTime = time.Now()

	return &result, nil
}

// parseContextSynthesisResponse 解析上下文合成响应
func (s *WideRecallService) parseContextSynthesisResponse(response string) (*ContextSynthesisResult, error) {
	var result ContextSynthesisResult

	// 清理响应内容，移除可能的markdown标记
	cleanResponse := strings.TrimSpace(response)
	if strings.HasPrefix(cleanResponse, "```json") {
		cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
	}
	if strings.HasSuffix(cleanResponse, "```") {
		cleanResponse = strings.TrimSuffix(cleanResponse, "```")
	}
	cleanResponse = strings.TrimSpace(cleanResponse)

	log.Printf("🔍 [上下文合成] 准备解析JSON，长度: %d", len(cleanResponse))
	log.Printf("🔍 [上下文合成] JSON内容前500字符: %s", cleanResponse[:min(500, len(cleanResponse))])

	err := json.Unmarshal([]byte(cleanResponse), &result)
	if err != nil {
		log.Printf("❌ [上下文合成] JSON解析失败，错误: %v", err)
		log.Printf("❌ [上下文合成] 完整响应内容: %s", cleanResponse)
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	log.Printf("✅ [上下文合成] JSON解析成功")
	log.Printf("🔍 [上下文合成] EvaluationResult是否为nil: %t", result.EvaluationResult == nil)
	log.Printf("🔍 [上下文合成] SynthesizedContext是否为nil: %t", result.SynthesizedContext == nil)

	return &result, nil
}
