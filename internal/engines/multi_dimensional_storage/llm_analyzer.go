package multi_dimensional_storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/contextkeeper/service/internal/llm"
)

// DeepSeekLLMAnalyzer DeepSeek LLM分析器实现
type DeepSeekLLMAnalyzer struct {
	client   llm.LLMClient
	template *template.Template
	config   *MultiDimensionalStorageConfig
}

// NewDeepSeekLLMAnalyzer 创建DeepSeek LLM分析器
func NewDeepSeekLLMAnalyzer(apiKey string, config *MultiDimensionalStorageConfig) (*DeepSeekLLMAnalyzer, error) {
	llmConfig := &llm.LLMConfig{
		APIKey:  apiKey,
		BaseURL: "https://api.deepseek.com",
		Model:   config.LLMModel,
	}

	client, err := llm.NewDeepSeekClient(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("创建DeepSeek客户端失败: %w", err)
	}

	// 解析Prompt模板
	tmpl, err := template.New("storage_analysis").Parse(MULTI_DIMENSIONAL_STORAGE_PROMPT)
	if err != nil {
		return nil, fmt.Errorf("解析Prompt模板失败: %w", err)
	}

	analyzer := &DeepSeekLLMAnalyzer{
		client:   client,
		template: tmpl,
		config:   config,
	}

	log.Printf("✅ DeepSeek LLM分析器初始化完成")
	return analyzer, nil
}

// Analyze 分析请求，返回结构化数据
func (a *DeepSeekLLMAnalyzer) Analyze(request *StorageRequest) (*LLMAnalysisResult, error) {
	startTime := time.Now()

	// 1. 构建Prompt
	prompt, err := a.buildPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("构建Prompt失败: %w", err)
	}

	log.Printf("🔍 开始LLM分析 - 用户: %s, 会话: %s", request.UserID, request.SessionID)
	log.Printf("📝 查询内容: %s", request.Query[:min(100, len(request.Query))])

	// 2. 调用LLM
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Timeout)
	defer cancel()

	response, err := a.client.Complete(ctx, &llm.LLMRequest{
		Prompt:      prompt,
		Model:       a.config.LLMModel,
		Temperature: 0.1, // 低温度确保结果稳定
		MaxTokens:   4000,
		Format:      "json",
	})

	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 3. 解析响应
	result, err := a.parseResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	analysisTime := time.Since(startTime)
	log.Printf("✅ LLM分析完成 - 耗时: %v", analysisTime)
	log.Printf("📊 分析结果: 时间线优先级=%.2f, 知识图谱优先级=%.2f, 向量优先级=%.2f",
		result.StorageRecommendation.TimelinePriority,
		result.StorageRecommendation.KnowledgePriority,
		result.StorageRecommendation.VectorPriority)

	return result, nil
}

// IsAvailable 检查LLM是否可用
func (a *DeepSeekLLMAnalyzer) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 简单的健康检查
	_, err := a.client.Complete(ctx, &llm.LLMRequest{
		Prompt:    "Hello",
		Model:     a.config.LLMModel,
		MaxTokens: 10,
	})

	return err == nil
}

// buildPrompt 构建Prompt
func (a *DeepSeekLLMAnalyzer) buildPrompt(request *StorageRequest) (string, error) {
	// 准备模板变量
	vars := map[string]interface{}{
		"UserID":      request.UserID,
		"SessionID":   request.SessionID,
		"WorkspaceID": request.WorkspaceID,
		"Timestamp":   request.Timestamp.Format("2006-01-02 15:04:05"),
		"Query":       request.Query,
		"Context":     request.Context,
	}

	// 渲染模板
	var buf bytes.Buffer
	if err := a.template.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// parseResponse 解析LLM响应
func (a *DeepSeekLLMAnalyzer) parseResponse(content string) (*LLMAnalysisResult, error) {
	// 清理响应内容，提取JSON部分
	content = strings.TrimSpace(content)

	// 查找JSON开始和结束位置
	startIdx := strings.Index(content, "{")
	endIdx := strings.LastIndex(content, "}")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("响应中未找到有效的JSON格式")
	}

	jsonContent := content[startIdx : endIdx+1]

	// 解析JSON
	var result LLMAnalysisResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		log.Printf("⚠️ JSON解析失败，原始内容: %s", jsonContent)
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 验证必要字段
	if result.StorageRecommendation == nil {
		result.StorageRecommendation = &StorageRecommendation{
			TimelinePriority:  0.5,
			KnowledgePriority: 0.5,
			VectorPriority:    0.5,
			Reasoning:         "默认推荐",
		}
	}

	return &result, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MULTI_DIMENSIONAL_STORAGE_PROMPT 多维度存储分析Prompt
const MULTI_DIMENSIONAL_STORAGE_PROMPT = `
你是一个专业的记忆分析专家，需要从用户的查询/命令和上下文中抽取出适合不同存储引擎的结构化数据。

## 上下文信息
用户ID: {{.UserID}}
会话ID: {{.SessionID}}
工作空间: {{.WorkspaceID}}
时间戳: {{.Timestamp}}

## 原始输入
查询/命令: {{.Query}}
当前上下文: {{.Context}}

## 任务要求
请分析上述内容，抽取出以下三种类型的结构化数据：

### 1. 时间线故事性数据 (Timeline Data)
适合存储到TimescaleDB的事件数据，包括：
- 事件标题 (简洁描述)
- 事件内容 (详细描述)
- 事件类型 (问题解决/学习记录/决策过程/技术讨论等)
- 关键词列表 (便于搜索)
- 重要性评分 (1-10)
- 相关技术栈
- 项目关联

### 2. 知识图谱数据 (Knowledge Graph Data)
适合存储到Neo4j的概念和关系数据，包括：
- 核心概念/实体 (技术概念、工具、方法等)
- 概念之间的关系 (依赖、相似、对比、包含等)
- 概念属性 (类型、重要性、复杂度等)
- 关系强度 (0.0-1.0)

### 3. 向量知识库数据 (Vector Data)
适合存储到向量数据库的语义数据，包括：
- 核心语义内容 (去除噪音的纯净内容)
- 语义标签 (主题分类)
- 上下文摘要 (便于理解)
- 关联度评分

## 输出格式
请严格按照以下JSON格式输出：

{
  "timeline_data": {
    "title": "事件标题",
    "content": "详细内容",
    "event_type": "事件类型",
    "keywords": ["关键词1", "关键词2"],
    "importance_score": 8,
    "tech_stack": ["技术1", "技术2"],
    "project_context": "项目上下文"
  },
  "knowledge_graph_data": {
    "concepts": [
      {
        "name": "概念名称",
        "type": "概念类型",
        "properties": {"属性1": "值1"},
        "importance": 0.8
      }
    ],
    "relationships": [
      {
        "source": "源概念",
        "target": "目标概念",
        "type": "关系类型",
        "strength": 0.7,
        "description": "关系描述"
      }
    ]
  },
  "vector_data": {
    "content": "纯净语义内容",
    "semantic_tags": ["标签1", "标签2"],
    "context_summary": "上下文摘要",
    "relevance_score": 0.9
  },
  "storage_recommendation": {
    "timeline_priority": 0.8,
    "knowledge_priority": 0.6,
    "vector_priority": 0.9,
    "reasoning": "推荐理由"
  }
}

## 注意事项
1. 如果某类数据不适合，可以设置为null或空对象
2. 优先保证数据质量，而不是数量
3. 关键词要具体且有搜索价值
4. 概念抽取要准确，避免过度泛化
5. 向量数据要去除噪音，保留核心语义
`
