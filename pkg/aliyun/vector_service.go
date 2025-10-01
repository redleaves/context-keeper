package aliyun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// 日志颜色常量
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

// VectorService 阿里云向量服务客户端
type VectorService struct {
	EmbeddingAPIURL     string
	EmbeddingAPIKey     string
	VectorDBURL         string
	VectorDBAPIKey      string
	VectorDBCollection  string
	VectorDBDimension   int
	VectorDBMetric      string
	SimilarityThreshold float64
}

// NewVectorService 创建新的阿里云向量服务客户端
func NewVectorService(embeddingAPIURL, embeddingAPIKey, vectorDBURL, vectorDBAPIKey, collection string,
	dimension int, metric string, threshold float64) *VectorService {
	return &VectorService{
		EmbeddingAPIURL:     embeddingAPIURL,
		EmbeddingAPIKey:     embeddingAPIKey,
		VectorDBURL:         vectorDBURL,
		VectorDBAPIKey:      vectorDBAPIKey,
		VectorDBCollection:  collection,
		VectorDBDimension:   dimension,
		VectorDBMetric:      metric,
		SimilarityThreshold: threshold,
	}
}

// GenerateEmbedding 生成文本的向量表示
func (s *VectorService) GenerateEmbedding(text string) ([]float32, error) {
	log.Printf("\n[向量服务] 开始生成文本嵌入向量 ============================")
	log.Printf("[向量服务] 文本长度: %d 字符", len(text))

	// 构建请求体
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":           "text-embedding-v1",
		"input":           []string{text},
		"encoding_format": "float",
	})
	if err != nil {
		log.Printf("[向量服务] 错误: 序列化请求失败: %v", err)
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", s.EmbeddingAPIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[向量服务] 错误: 创建HTTP请求失败: %v", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.EmbeddingAPIKey)

	log.Printf("[向量服务] 发送嵌入API请求: %s", s.EmbeddingAPIURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[向量服务] 错误: API请求失败: %v", err)
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应数据
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[向量服务] 错误: 读取响应失败: %v", err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[向量服务] 错误: API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[向量服务] 错误: 解析响应失败: %v, 响应内容: %s", err, string(respBody))
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	// 检查返回的嵌入向量
	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		log.Printf("[向量服务] 错误: 未返回有效的嵌入向量")
		return nil, fmt.Errorf("未返回有效的嵌入向量")
	}

	// 输出向量的前几个元素，用于调试
	if len(result.Data[0].Embedding) > 5 {
		log.Printf("[向量服务] 成功生成向量，维度: %d, 前5个元素: %v",
			len(result.Data[0].Embedding), result.Data[0].Embedding[:5])
	}

	log.Printf("[向量服务] 成功完成文字转向量 ============================\n")

	return result.Data[0].Embedding, nil
}

// GenerateMultiDimensionalVectors 生成多维度向量（重新设计：基于LLM的一次性多维度数据抽取）
func (s *VectorService) GenerateMultiDimensionalVectors(content string, llmAPIKey string) (*models.MultiDimensionalVectors, error) {
	log.Printf("\n[多维度向量生成] 🔥 开始基于LLM的一次性多维度数据抽取 ============================")
	log.Printf("[多维度向量生成] 内容长度: %d 字符", len(content))
	log.Printf("[多维度向量生成] 内容预览: %s", content[:min(200, len(content))])

	// 🔥 核心：一次LLM调用，抽取符合不同存储引擎的形态数据
	analysisResult, err := s.analyzeLLMContent(content, llmAPIKey)
	if err != nil {
		log.Printf("[多维度向量生成] LLM分析失败: %v", err)
		return nil, fmt.Errorf("LLM分析失败: %w", err)
	}

	log.Printf("[多维度向量生成] 🎯 LLM一次性多维度分析完成:")
	if analysisResult.TimelineData != nil {
		log.Printf("  时间线故事: %s", analysisResult.TimelineData.StoryTitle)
		log.Printf("  关键事件数: %d", len(analysisResult.TimelineData.KeyEvents))
	}
	if analysisResult.KnowledgeGraphData != nil {
		log.Printf("  知识概念数: %d", len(analysisResult.KnowledgeGraphData.MainConcepts))
		log.Printf("  关系数: %d", len(analysisResult.KnowledgeGraphData.Relationships))
	}
	if analysisResult.VectorData != nil {
		log.Printf("  语义核心: %s", analysisResult.VectorData.SemanticCore[:min(50, len(analysisResult.VectorData.SemanticCore))])
		log.Printf("  搜索关键词: %v", analysisResult.VectorData.SearchKeywords)
	}

	// 🔥 第二步：基于分析结果生成专门的向量
	vectors := &models.MultiDimensionalVectors{}

	// 生成时间线向量（基于故事性摘要）
	if analysisResult.TimelineData != nil && analysisResult.TimelineData.StorySummary != "" {
		timelineVector, err := s.GenerateEmbedding(analysisResult.TimelineData.StorySummary)
		if err != nil {
			log.Printf("[多维度向量生成] 时间线向量生成失败: %v", err)
		} else {
			vectors.TimeVector = timelineVector
			log.Printf("[多维度向量生成] ✅ 时间线向量生成成功，维度: %d", len(timelineVector))
		}
	}

	// 生成知识图谱向量（基于概念和关系）
	if analysisResult.KnowledgeGraphData != nil {
		// 构建知识图谱的文本表示
		var kgText strings.Builder
		for _, concept := range analysisResult.KnowledgeGraphData.MainConcepts {
			kgText.WriteString(fmt.Sprintf("%s(%s) ", concept.Name, concept.Type))
		}
		for _, rel := range analysisResult.KnowledgeGraphData.Relationships {
			kgText.WriteString(fmt.Sprintf("%s-%s-%s ", rel.From, rel.Relation, rel.To))
		}

		if kgText.Len() > 0 {
			knowledgeVector, err := s.GenerateEmbedding(kgText.String())
			if err != nil {
				log.Printf("[多维度向量生成] 知识图谱向量生成失败: %v", err)
			} else {
				vectors.DomainVector = knowledgeVector
				log.Printf("[多维度向量生成] ✅ 知识图谱向量生成成功，维度: %d", len(knowledgeVector))
			}
		}
	}

	// 生成语义向量（基于精炼的语义核心）
	if analysisResult.VectorData != nil && analysisResult.VectorData.SemanticCore != "" {
		semanticVector, err := s.GenerateEmbedding(analysisResult.VectorData.SemanticCore)
		if err != nil {
			log.Printf("[多维度向量生成] 语义向量生成失败: %v", err)
		} else {
			vectors.SemanticVector = semanticVector
			log.Printf("[多维度向量生成] ✅ 语义向量生成成功，维度: %d", len(semanticVector))
		}
	}

	// 生成上下文向量（基于上下文信息）
	if analysisResult.VectorData != nil && analysisResult.VectorData.ContextInfo != "" {
		contextVector, err := s.GenerateEmbedding(analysisResult.VectorData.ContextInfo)
		if err != nil {
			log.Printf("[多维度向量生成] 上下文向量生成失败: %v", err)
		} else {
			vectors.ContextVector = contextVector
			log.Printf("[多维度向量生成] ✅ 上下文向量生成成功，维度: %d", len(contextVector))
		}
	}

	// 🔥 设置结构化分析结果
	if analysisResult.VectorData != nil {
		vectors.SemanticTags = analysisResult.VectorData.SemanticTags
		vectors.ContextSummary = analysisResult.VectorData.RelevanceContext
	}
	if analysisResult.MetaAnalysis != nil {
		vectors.TechStack = analysisResult.MetaAnalysis.TechStack
		vectors.EventType = analysisResult.MetaAnalysis.ContentType
		vectors.ImportanceScore = analysisResult.MetaAnalysis.BusinessValue
		vectors.RelevanceScore = analysisResult.MetaAnalysis.ReusePotential
		vectors.ProjectContext = analysisResult.MetaAnalysis.Priority
	}

	// 从知识图谱数据中提取概念实体
	if analysisResult.KnowledgeGraphData != nil {
		conceptNames := make([]string, len(analysisResult.KnowledgeGraphData.MainConcepts))
		for i, concept := range analysisResult.KnowledgeGraphData.MainConcepts {
			conceptNames[i] = concept.Name
		}
		vectors.ConceptEntities = conceptNames

		relatedConcepts := make([]string, len(analysisResult.KnowledgeGraphData.Relationships))
		for i, rel := range analysisResult.KnowledgeGraphData.Relationships {
			relatedConcepts[i] = fmt.Sprintf("%s-%s", rel.From, rel.To)
		}
		vectors.RelatedConcepts = relatedConcepts
	}

	log.Printf("[多维度向量生成] 🎉 多维度向量生成完成")
	log.Printf("  语义向量: %v", len(vectors.SemanticVector) > 0)
	log.Printf("  上下文向量: %v", len(vectors.ContextVector) > 0)
	log.Printf("  时间线向量: %v", len(vectors.TimeVector) > 0)
	log.Printf("  知识图谱向量: %v", len(vectors.DomainVector) > 0)
	log.Printf("==================================================== 多维度向量生成完成 ====================================================")

	return vectors, nil
}

// analyzeLLMContent 使用LLM分析内容，提取多维度信息（重新设计）
func (s *VectorService) analyzeLLMContent(content string, llmAPIKey string) (*models.MultiDimensionalAnalysisResult, error) {
	log.Printf("\n[LLM内容分析] 开始分析内容 ============================")

	// 构建专门的prompt，让LLM理解我们的意图
	prompt := s.buildMultiDimensionalAnalysisPrompt(content)

	log.Printf("[LLM内容分析] Prompt长度: %d 字符", len(prompt))

	// 🔍 详细打印Prompt内容
	log.Printf("🔍 [Prompt详情] ============================")
	log.Printf("📝 Prompt长度: %d 字符", len(prompt))
	log.Printf("📝 待分析内容长度: %d 字符", len(content))
	log.Printf("📝 完整Prompt内容:")
	log.Printf("%s", prompt)
	log.Printf("🔍 ==============================")

	log.Printf("[LLM内容分析] 发送LLM分析请求...")

	// 调用LLM API进行分析
	response, err := s.callLLMAPI(prompt, llmAPIKey)
	if err != nil {
		return nil, fmt.Errorf("LLM API调用失败: %w", err)
	}

	log.Printf("[LLM内容分析] LLM响应长度: %d 字符", len(response))
	log.Printf("[LLM内容分析] LLM响应内容: %s", response[:min(500, len(response))])

	// 解析LLM响应
	result, err := s.parseLLMAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	log.Printf("[LLM内容分析] 分析完成:")
	if result.TimelineData != nil {
		log.Printf("  时间线重要性: %d", result.TimelineData.ImportanceLevel)
	}
	if result.KnowledgeGraphData != nil {
		log.Printf("  概念数量: %d", len(result.KnowledgeGraphData.MainConcepts))
	}
	if result.MetaAnalysis != nil {
		log.Printf("  内容类型: %s", result.MetaAnalysis.ContentType)
		log.Printf("  业务价值: %.2f", result.MetaAnalysis.BusinessValue)
	}

	return result, nil
}

// buildMultiDimensionalAnalysisPrompt 构建多维度分析的prompt（核心设计）
func (s *VectorService) buildMultiDimensionalAnalysisPrompt(content string) string {
	// 🔥 这是整个架构的核心：prompt设计决定了数据质量和查询效率
	prompt := `你是一个专业的记忆存储分析专家，需要将用户的内容分解为适合不同存储引擎的数据形态。

## 任务目标
从用户内容中抽取出符合以下三种记忆存储引擎的形态数据：
1. **时间线故事性存储** - 适合TimescaleDB，记录事件发展过程
2. **知识图谱存储** - 适合Neo4j，记录概念关系和实体连接
3. **向量知识库存储** - 适合向量数据库，记录语义和上下文信息

## 输出格式（严格JSON）
{
  "timeline_data": {
    "story_title": "简洁的故事标题（10-20字）",
    "story_summary": "故事性描述，突出时间发展脉络（50-80字）",
    "key_events": ["事件1", "事件2", "事件3"],
    "time_sequence": "时间序列特征描述",
    "outcome": "最终结果或当前状态",
    "lessons_learned": "经验教训或收获",
    "importance_level": 8
  },
  "knowledge_graph_data": {
    "main_concepts": [
      {"name": "概念名", "type": "技术|业务|工具|方法", "importance": 0.9},
      {"name": "概念名", "type": "技术|业务|工具|方法", "importance": 0.8}
    ],
    "relationships": [
      {"from": "概念A", "to": "概念B", "relation": "解决|导致|包含|依赖|优化", "strength": 0.9},
      {"from": "概念C", "to": "概念D", "relation": "解决|导致|包含|依赖|优化", "strength": 0.8}
    ],
    "domain": "技术领域分类",
    "complexity_level": "简单|中等|复杂"
  },
  "vector_data": {
    "semantic_core": "去噪后的核心语义内容（30-50字）",
    "context_info": "上下文背景信息（30-50字）",
    "search_keywords": ["搜索关键词1", "搜索关键词2", "搜索关键词3"],
    "semantic_tags": ["语义标签1", "语义标签2", "语义标签3"],
    "relevance_context": "相关性上下文描述"
  },
  "meta_analysis": {
    "content_type": "问题解决|技术学习|经验分享|决策记录|讨论交流",
    "priority": "P1|P2|P3",
    "tech_stack": ["技术栈1", "技术栈2"],
    "business_value": 0.8,
    "reuse_potential": 0.9
  }
}

## 分析要求
1. **时间线数据**：突出故事性和发展脉络，适合按时间检索
2. **知识图谱数据**：明确概念和关系，适合关联查询
3. **向量数据**：精炼语义核心，去除噪声，适合相似性搜索
4. **所有评分**：基于实际价值，范围0-1或1-10
5. **严格JSON格式**：不要添加任何解释文字

## 待分析内容
` + content + `

请开始分析：`

	return prompt
}

// callLLMAPI 调用LLM API
func (s *VectorService) callLLMAPI(prompt string, apiKey string) (string, error) {
	// 构建请求体
	reqBody, err := json.Marshal(map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.1, // 低温度确保结果稳定
		"max_tokens":  2000,
	})
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[LLM API调用] 发送请求到: %s", req.URL.String())
	log.Printf("[LLM API调用] 请求体大小: %d 字节", len(reqBody))

	// 🔍 详细打印请求参数
	log.Printf("🔍 [LLM请求详情] ============================")
	log.Printf("📤 请求URL: %s", req.URL.String())
	log.Printf("📤 请求方法: %s", req.Method)
	log.Printf("📤 请求头: Content-Type=%s", req.Header.Get("Content-Type"))
	log.Printf("📤 请求头: Authorization=%s", req.Header.Get("Authorization")[:20]+"...")
	log.Printf("📤 请求体内容:")
	log.Printf("%s", string(reqBody))
	log.Printf("🔍 ==============================")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("[LLM API调用] 响应状态码: %d", resp.StatusCode)
	log.Printf("[LLM API调用] 响应体大小: %d 字节", len(respBody))

	// 🔍 详细打印响应内容
	log.Printf("🔍 [LLM响应详情] ============================")
	log.Printf("📥 响应状态码: %d", resp.StatusCode)
	log.Printf("📥 响应头: Content-Type=%s", resp.Header.Get("Content-Type"))
	log.Printf("📥 响应体大小: %d 字节", len(respBody))
	log.Printf("📥 响应体内容:")
	log.Printf("%s", string(respBody))
	log.Printf("🔍 ==============================")

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ [LLM API调用] 错误响应: %s", string(respBody))
		log.Printf("❌ [LLM API调用] 状态码: %d", resp.StatusCode)
		log.Printf("❌ [LLM API调用] 完整错误信息: %s", string(respBody))
		return "", fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM未返回有效响应")
	}

	content := result.Choices[0].Message.Content
	log.Printf("[LLM API调用] 成功获取LLM响应，内容长度: %d", len(content))

	// 🔍 详细打印LLM返回的内容
	log.Printf("🔍 [LLM返回内容] ============================")
	log.Printf("✅ LLM响应成功")
	log.Printf("📝 返回内容长度: %d 字符", len(content))
	log.Printf("📝 返回内容:")
	log.Printf("%s", content)
	log.Printf("🔍 ==============================")

	return content, nil
}

// parseLLMAnalysisResponse 解析LLM分析响应（重新设计）
func (s *VectorService) parseLLMAnalysisResponse(response string) (*models.MultiDimensionalAnalysisResult, error) {
	log.Printf("[LLM响应解析] 开始解析响应...")

	// 清理响应内容，提取JSON部分
	jsonContent := s.extractJSONFromResponse(response)

	log.Printf("[LLM响应解析] 提取的JSON长度: %d", len(jsonContent))
	log.Printf("[LLM响应解析] JSON内容: %s", jsonContent[:min(300, len(jsonContent))])

	// 解析JSON为新的多维度分析结果
	var result models.MultiDimensionalAnalysisResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		log.Printf("[LLM响应解析] JSON解析失败: %v", err)
		log.Printf("[LLM响应解析] 原始响应: %s", response)
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 验证和清理结果
	s.validateAndCleanMultiDimensionalResult(&result)

	log.Printf("[LLM响应解析] 解析成功:")
	if result.TimelineData != nil {
		log.Printf("  时间线标题: %s", result.TimelineData.StoryTitle)
	}
	if result.KnowledgeGraphData != nil {
		log.Printf("  概念数量: %d", len(result.KnowledgeGraphData.MainConcepts))
	}
	if result.VectorData != nil {
		log.Printf("  语义核心: %s", result.VectorData.SemanticCore[:min(30, len(result.VectorData.SemanticCore))])
	}

	return &result, nil
}

// extractJSONFromResponse 从响应中提取JSON内容
func (s *VectorService) extractJSONFromResponse(response string) string {
	// 查找JSON开始和结束位置
	start := strings.Index(response, "{")
	if start == -1 {
		log.Printf("[JSON提取] 未找到JSON开始标记")
		return response
	}

	// 从后往前查找最后一个}
	end := strings.LastIndex(response, "}")
	if end == -1 || end <= start {
		log.Printf("[JSON提取] 未找到有效的JSON结束标记")
		return response
	}

	jsonContent := response[start : end+1]
	log.Printf("[JSON提取] 提取JSON成功，长度: %d", len(jsonContent))

	return jsonContent
}

// validateAndCleanAnalysisResult 验证和清理分析结果
func (s *VectorService) validateAndCleanAnalysisResult(result *models.LLMAnalysisResult) {
	// 设置默认值
	if result.ImportanceScore < 0 || result.ImportanceScore > 1 {
		result.ImportanceScore = 0.5
	}
	if result.RelevanceScore < 0 || result.RelevanceScore > 1 {
		result.RelevanceScore = 0.5
	}

	// 清理空字符串
	if result.SemanticSummary == "" {
		result.SemanticSummary = "内容摘要"
	}
	if result.ContextSummary == "" {
		result.ContextSummary = "上下文信息"
	}
	if result.EventType == "" {
		result.EventType = "其他"
	}

	// 确保数组不为nil
	if result.Keywords == nil {
		result.Keywords = []string{}
	}
	if result.ConceptEntities == nil {
		result.ConceptEntities = []string{}
	}
	if result.RelatedConcepts == nil {
		result.RelatedConcepts = []string{}
	}
	if result.TechStack == nil {
		result.TechStack = []string{}
	}

	log.Printf("[结果验证] 验证和清理完成")
}

// validateAndCleanMultiDimensionalResult 验证和清理多维度分析结果
func (s *VectorService) validateAndCleanMultiDimensionalResult(result *models.MultiDimensionalAnalysisResult) {
	// 验证时间线数据
	if result.TimelineData != nil {
		if result.TimelineData.StoryTitle == "" {
			result.TimelineData.StoryTitle = "未命名事件"
		}
		if result.TimelineData.ImportanceLevel < 1 || result.TimelineData.ImportanceLevel > 10 {
			result.TimelineData.ImportanceLevel = 5
		}
		if result.TimelineData.KeyEvents == nil {
			result.TimelineData.KeyEvents = []string{}
		}
	}

	// 验证知识图谱数据
	if result.KnowledgeGraphData != nil {
		if result.KnowledgeGraphData.MainConcepts == nil {
			result.KnowledgeGraphData.MainConcepts = []models.Concept{}
		}
		if result.KnowledgeGraphData.Relationships == nil {
			result.KnowledgeGraphData.Relationships = []models.Relationship{}
		}
		if result.KnowledgeGraphData.Domain == "" {
			result.KnowledgeGraphData.Domain = "通用"
		}
	}

	// 验证向量数据
	if result.VectorData != nil {
		if result.VectorData.SemanticCore == "" {
			result.VectorData.SemanticCore = "内容摘要"
		}
		if result.VectorData.SearchKeywords == nil {
			result.VectorData.SearchKeywords = []string{}
		}
		if result.VectorData.SemanticTags == nil {
			result.VectorData.SemanticTags = []string{}
		}
	}

	// 验证元分析数据
	if result.MetaAnalysis != nil {
		if result.MetaAnalysis.ContentType == "" {
			result.MetaAnalysis.ContentType = "其他"
		}
		if result.MetaAnalysis.Priority == "" {
			result.MetaAnalysis.Priority = "P2"
		}
		if result.MetaAnalysis.BusinessValue < 0 || result.MetaAnalysis.BusinessValue > 1 {
			result.MetaAnalysis.BusinessValue = 0.5
		}
		if result.MetaAnalysis.ReusePotential < 0 || result.MetaAnalysis.ReusePotential > 1 {
			result.MetaAnalysis.ReusePotential = 0.5
		}
		if result.MetaAnalysis.TechStack == nil {
			result.MetaAnalysis.TechStack = []string{}
		}
	}

	log.Printf("[多维度结果验证] 验证和清理完成")
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StoreVectors 存储向量到Aliyun向量数据库
func (s *VectorService) StoreVectors(memory *models.Memory) error {
	log.Printf("\n[向量存储] 开始存储向量 ============================")
	log.Printf("[向量存储] 记忆ID: %s, 会话ID: %s, 内容长度: %d, 向量维度: %d",
		memory.ID, memory.SessionID, len(memory.Content), len(memory.Vector))

	// 记录bizType和userId信息
	log.Printf("[向量存储] 待存储记录类型信息 - bizType: %d, userId: %s", memory.BizType, memory.UserID)

	// 检查向量是否已生成
	if memory.Vector == nil || len(memory.Vector) == 0 {
		log.Printf("错误: 存储前必须先生成向量")
		return fmt.Errorf("存储前必须先生成向量")
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(memory.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 将metadata转换为JSON字符串
	metadataStr := "{}"
	var storageId string = memory.ID // 默认使用memory.ID作为存储ID

	if memory.Metadata != nil {
		// 如果元数据中有batchId，则使用batchId作为存储ID
		if batchId, ok := memory.Metadata["batchId"].(string); ok && batchId != "" {
			storageId = batchId
			log.Printf("[向量存储] 使用batchId作为存储ID: %s", storageId)
		}

		if metadataBytes, err := json.Marshal(memory.Metadata); err == nil {
			metadataStr = string(metadataBytes)
			log.Printf("[向量存储] 元数据: %s", metadataStr)
		} else {
			log.Printf("[向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建文档
	doc := map[string]interface{}{
		"id":     storageId, // 使用storageId(batchId或memoryId)作为向量存储的主键
		"vector": memory.Vector,
		"fields": map[string]interface{}{
			"session_id":     memory.SessionID,
			"content":        memory.Content,
			"timestamp":      memory.Timestamp,
			"formatted_time": formattedTime,
			"priority":       memory.Priority,
			"metadata":       metadataStr, // 使用字符串格式的元数据
			"memory_id":      memory.ID,   // 保留原始memory_id
			// 在fields中也添加业务类型和用户ID字段
			"bizType": memory.BizType, // 业务类型
			"userId":  memory.UserID,  // 用户ID
		},
	}

	// 构建插入请求
	insertReq := map[string]interface{}{
		"docs": []map[string]interface{}{doc},
	}

	// 序列化请求
	reqBody, err := json.Marshal(insertReq)
	if err != nil {
		log.Printf("[向量存储] 错误: 序列化插入请求失败: %v", err)
		return fmt.Errorf("序列化插入请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs", s.VectorDBURL, s.VectorDBCollection)
	log.Printf("[向量存储] 发送存储请求: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[向量存储] 错误: 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	log.Printf("[向量存储] 发送存储请求: %s", url)

	// 发送请求
	startTime := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[向量存储] 错误: API请求失败: %v", err)
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[向量存储] 错误: 读取响应失败: %v", err)
		return fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("[向量存储] 响应时间: %v, 状态码: %d", time.Since(startTime), resp.StatusCode)
	log.Printf("[向量存储] 响应内容: %s", string(respBody))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[向量存储] 错误: API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[向量存储] 错误: 解析响应失败: %v", err)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		log.Printf("[向量存储] 错误: API返回错误: %d, %s", result.Code, result.Message)
		return fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	log.Printf("[向量存储] 成功存储向量ID: %s, 会话: %s", memory.ID, memory.SessionID)
	log.Printf("[向量存储] 成功完成向量存储 ============================\n")
	return nil
}

// SearchVectors 在向量数据库中搜索相似向量
func (s *VectorService) SearchVectors(vector []float32, sessionID string, topK int) ([]models.SearchResult, error) {
	if topK <= 0 {
		topK = 5 // 默认返回5个结果
	}

	// 构建过滤条件（可选，只搜索特定会话的记忆）
	var filter string
	if sessionID != "" {
		filter = fmt.Sprintf("session_id = '%s'", sessionID)
	}

	// 构建请求体
	searchReq := map[string]interface{}{
		"vector":         vector,
		"topk":           topK,
		"include_vector": false,
	}

	// 如果有过滤条件，添加到请求中
	if filter != "" {
		searchReq["filter"] = filter
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录请求信息 - 添加颜色
	log.Printf("%s[向量搜索-请求] 会话ID=%s, topK=%d, 向量维度=%d%s",
		colorCyan, sessionID, topK, len(vector), colorReset)

	// 记录请求体 - 添加颜色
	log.Printf("%s[向量搜索-请求体] %s%s", colorCyan, string(reqBody), colorReset)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录原始响应 - 添加颜色
	log.Printf("%s[向量搜索-响应体] %s%s", colorCyan, string(respBody), colorReset)

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":    item.Fields["content"],
					"session_id": item.Fields["session_id"],
					"priority":   item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[向量搜索] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[向量搜索] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[向量搜索] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 构造返回结果 - 修改过滤逻辑（余弦距离，值越小越相似）
	var searchResults []models.SearchResult
	var mostSimilarItem *models.SearchResult
	var smallestScore float64 = 999.0 // 初始化为一个很大的值

	log.Printf("[向量搜索] 开始评估数据，相似度阈值: %.4f (小于等于此值视为相关)", s.SimilarityThreshold)

	for _, item := range result.Output {
		// 应用相似度阈值过滤（余弦距离：越小越相似）
		if item.Score <= s.SimilarityThreshold {
			newResult := models.SearchResult{
				ID:     item.Id,
				Score:  item.Score,
				Fields: item.Fields,
			}
			searchResults = append(searchResults, newResult)

			log.Printf("[向量搜索] 符合条件的数据项: ID=%s, 相似度=%.4f (小于等于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)

			// 跟踪最相似的结果（得分最小）
			if item.Score < smallestScore {
				smallestScore = item.Score
				mostSimilarItem = &models.SearchResult{
					ID:     item.Id,
					Score:  item.Score,
					Fields: item.Fields,
				}
			}
		} else {
			log.Printf("[向量搜索] 过滤掉的数据项: ID=%s, 相似度=%.4f (大于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)
		}
	}

	// 输出最相似结果信息
	if mostSimilarItem != nil {
		content, _ := mostSimilarItem.Fields["content"].(string)
		log.Printf("[向量搜索] 最相似数据项: ID=%s, 相似度=%.4f, 内容=%s",
			mostSimilarItem.ID, mostSimilarItem.Score, content)

		// 输出完整的最佳匹配记录
		bestMatchJSON, _ := json.MarshalIndent(mostSimilarItem, "", "  ")
		log.Printf("[向量搜索-最终选择] 得分最低的记录完整数据:\n%s", string(bestMatchJSON))
	} else {
		log.Printf("[向量搜索] 未找到符合阈值的相关数据")
	}

	log.Printf("[向量检索] 查询结果: 找到 %d 条记录, 过滤后保留 %d 条",
		len(result.Output), len(searchResults))
	log.Printf("==================================================== 向量搜索完成 ====================================================")
	return searchResults, nil
}

// StoreMessage 存储消息到向量数据库
func (s *VectorService) StoreMessage(message *models.Message) error {
	// 确保已生成向量
	if len(message.Vector) == 0 {
		return fmt.Errorf("存储前必须先生成向量")
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(message.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 将metadata转换为JSON字符串
	metadataStr := "{}"
	var storageId string = message.ID // 默认使用message.ID作为存储ID

	if message.Metadata != nil {
		// 如果元数据中有batchId，则使用batchId作为存储ID
		if batchId, ok := message.Metadata["batchId"].(string); ok && batchId != "" {
			storageId = batchId
			log.Printf("[向量存储] 使用batchId作为消息存储ID: %s", storageId)
		}

		if metadataBytes, err := json.Marshal(message.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建文档
	doc := map[string]interface{}{
		"id":     storageId, // 使用storageId(batchId或messageId)作为向量存储的主键
		"vector": message.Vector,
		"fields": map[string]interface{}{
			"session_id":     message.SessionID,
			"role":           message.Role,
			"content":        message.Content,
			"content_type":   message.ContentType,
			"timestamp":      message.Timestamp,
			"formatted_time": formattedTime,
			"priority":       message.Priority,
			"metadata":       metadataStr,
			"message_id":     message.ID, // 保留原始message_id
		},
	}

	// 构建插入请求
	insertReq := map[string]interface{}{
		"docs": []map[string]interface{}{doc},
	}

	// 序列化请求
	reqBody, err := json.Marshal(insertReq)
	if err != nil {
		return fmt.Errorf("序列化插入请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		return fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	log.Printf("[向量存储] 成功存储消息ID: %s, 会话: %s, 角色: %s", message.ID, message.SessionID, message.Role)
	log.Printf("==================================================== 存储消息完成 ====================================================")
	return nil
}

// StoreEnhancedMemory 存储增强的多维度记忆（新增方法）
func (s *VectorService) StoreEnhancedMemory(memory *models.EnhancedMemory) error {
	log.Printf("\n[增强向量存储] 开始存储增强记忆 ============================")
	log.Printf("[增强向量存储] 记忆ID: %s, 会话ID: %s, 内容长度: %d",
		memory.Memory.ID, memory.Memory.SessionID, len(memory.Memory.Content))

	// 🔥 关键改进：生成真实的多维度向量
	log.Printf("[增强向量存储] 开始生成多维度向量...")

	// 如果多维度向量为空，使用LLM分析生成
	if len(memory.SemanticVector) == 0 && len(memory.ContextVector) == 0 {
		// TODO: 从环境变量或配置中获取LLM API Key
		llmAPIKey := os.Getenv("DEEPSEEK_API_KEY")
		if llmAPIKey == "" {
			log.Printf("[增强向量存储] 警告: 未设置DEEPSEEK_API_KEY，跳过多维度向量生成")
		} else {
			multiVectors, err := s.GenerateMultiDimensionalVectors(memory.Memory.Content, llmAPIKey)
			if err != nil {
				log.Printf("[增强向量存储] 多维度向量生成失败: %v", err)
				// 不返回错误，继续使用基础向量存储
			} else {
				// 将生成的多维度向量设置到memory中
				memory.SemanticVector = multiVectors.SemanticVector
				memory.ContextVector = multiVectors.ContextVector
				memory.TimeVector = multiVectors.TimeVector
				memory.DomainVector = multiVectors.DomainVector
				memory.SemanticTags = multiVectors.SemanticTags
				memory.ConceptEntities = multiVectors.ConceptEntities
				memory.RelatedConcepts = multiVectors.RelatedConcepts
				memory.ImportanceScore = multiVectors.ImportanceScore
				memory.RelevanceScore = multiVectors.RelevanceScore
				memory.ContextSummary = multiVectors.ContextSummary
				memory.TechStack = multiVectors.TechStack
				memory.ProjectContext = multiVectors.ProjectContext
				memory.EventType = multiVectors.EventType

				log.Printf("[增强向量存储] 多维度向量生成成功:")
				log.Printf("  语义向量: %d维", len(memory.SemanticVector))
				log.Printf("  上下文向量: %d维", len(memory.ContextVector))
				log.Printf("  时间向量: %d维", len(memory.TimeVector))
				log.Printf("  领域向量: %d维", len(memory.DomainVector))
			}
		}
	}

	// 确保基础向量已生成
	if memory.Memory.Vector == nil || len(memory.Memory.Vector) == 0 {
		log.Printf("[增强向量存储] 生成基础向量...")
		baseVector, err := s.GenerateEmbedding(memory.Memory.Content)
		if err != nil {
			return fmt.Errorf("生成基础向量失败: %w", err)
		}
		memory.Memory.Vector = baseVector
		log.Printf("[增强向量存储] 基础向量生成成功: %d维", len(baseVector))
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(memory.Memory.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 处理元数据
	metadataStr := "{}"
	var storageId string = memory.Memory.ID

	if memory.Memory.Metadata != nil {
		if batchId, ok := memory.Memory.Metadata["batchId"].(string); ok && batchId != "" {
			storageId = batchId
			log.Printf("[增强向量存储] 使用batchId作为存储ID: %s", storageId)
		}

		if metadataBytes, err := json.Marshal(memory.Memory.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[增强向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建增强文档（包含所有现有字段 + 新增多维度字段）
	fields := map[string]interface{}{
		// 现有字段（完全兼容）
		"session_id":     memory.Memory.SessionID,
		"content":        memory.Memory.Content,
		"timestamp":      memory.Memory.Timestamp,
		"formatted_time": formattedTime,
		"priority":       memory.Memory.Priority,
		"metadata":       metadataStr,
		"memory_id":      memory.Memory.ID,
		"bizType":        memory.Memory.BizType,
		"userId":         memory.Memory.UserID,

		// 新增多维度字段
		"semantic_tags":    memory.SemanticTags,
		"concept_entities": memory.ConceptEntities,
		"related_concepts": memory.RelatedConcepts,
		"importance_score": memory.ImportanceScore,
		"relevance_score":  memory.RelevanceScore,
		"context_summary":  memory.ContextSummary,
		"tech_stack":       memory.TechStack,
		"project_context":  memory.ProjectContext,
		"event_type":       memory.EventType,
	}

	// 添加多维度向量字段（如果存在）
	if len(memory.SemanticVector) > 0 {
		fields["semantic_vector"] = memory.SemanticVector
	}
	if len(memory.ContextVector) > 0 {
		fields["context_vector"] = memory.ContextVector
	}
	if len(memory.TimeVector) > 0 {
		fields["time_vector"] = memory.TimeVector
	}
	if len(memory.DomainVector) > 0 {
		fields["domain_vector"] = memory.DomainVector
	}

	// 添加多维度元数据
	if memory.MultiDimMetadata != nil {
		if multiDimBytes, err := json.Marshal(memory.MultiDimMetadata); err == nil {
			fields["multi_dim_metadata"] = string(multiDimBytes)
		}
	}

	// 构建文档
	doc := map[string]interface{}{
		"id":     storageId,
		"vector": memory.Memory.Vector, // 使用基础向量作为主向量
		"fields": fields,
	}

	// 构建插入请求
	insertReq := map[string]interface{}{
		"docs": []map[string]interface{}{doc},
	}

	// 序列化请求
	reqBody, err := json.Marshal(insertReq)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 序列化插入请求失败: %v", err)
		return fmt.Errorf("序列化插入请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs", s.VectorDBURL, s.VectorDBCollection)
	log.Printf("[增强向量存储] 发送存储请求: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[增强向量存储] 错误: 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 发送HTTP请求失败: %v", err)
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 读取响应失败: %v", err)
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("[增强向量存储] 错误: HTTP状态码 %d, 响应: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("向量存储失败: HTTP %d, %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[增强向量存储] 错误: 解析响应失败: %v", err)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		return fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	log.Printf("[增强向量存储] 增强记忆存储成功: ID=%s", memory.Memory.ID)

	// 🔥 TODO: 集成多维度存储引擎
	// 这里应该调用多维度存储引擎，将数据存储到TimescaleDB和Neo4j
	// 但目前多维度存储引擎未完全集成，需要后续实现
	log.Printf("[增强向量存储] ⚠️ 多维度存储引擎集成待实现")
	log.Printf("[增强向量存储] 当前仅存储到向量数据库，TimescaleDB和Neo4j存储待集成")

	return nil
}

// StoreEnhancedMessage 存储增强的多维度消息（新增方法）
func (s *VectorService) StoreEnhancedMessage(message *models.EnhancedMessage) error {
	log.Printf("\n[增强向量存储] 开始存储增强消息 ============================")
	log.Printf("[增强向量存储] 消息ID: %s, 会话ID: %s, 角色: %s",
		message.Message.ID, message.Message.SessionID, message.Message.Role)

	// 首先确保基础向量已生成
	if message.Message.Vector == nil || len(message.Message.Vector) == 0 {
		log.Printf("错误: 存储前必须先生成基础向量")
		return fmt.Errorf("存储前必须先生成基础向量")
	}

	// 生成格式化的时间戳
	formattedTime := time.Unix(message.Message.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 处理元数据
	metadataStr := "{}"
	if message.Message.Metadata != nil {
		if metadataBytes, err := json.Marshal(message.Message.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("[增强向量存储] 警告: 无法序列化元数据: %v", err)
		}
	}

	// 构建增强文档（包含所有现有字段 + 新增多维度字段）
	fields := map[string]interface{}{
		// 现有字段（完全兼容）
		"session_id":     message.Message.SessionID,
		"content":        message.Message.Content,
		"timestamp":      message.Message.Timestamp,
		"formatted_time": formattedTime,
		"role":           message.Message.Role,
		"metadata":       metadataStr,
		"message_id":     message.Message.ID,
		"userId":         "", // Message模型中没有UserID字段

		// 新增多维度字段
		"semantic_tags":    message.SemanticTags,
		"concept_entities": message.ConceptEntities,
		"related_concepts": message.RelatedConcepts,
		"importance_score": message.ImportanceScore,
		"relevance_score":  message.RelevanceScore,
		"context_summary":  message.ContextSummary,
		"tech_stack":       message.TechStack,
		"project_context":  message.ProjectContext,
		"event_type":       message.EventType,
	}

	// 添加多维度向量字段（如果存在）
	if len(message.SemanticVector) > 0 {
		fields["semantic_vector"] = message.SemanticVector
	}
	if len(message.ContextVector) > 0 {
		fields["context_vector"] = message.ContextVector
	}
	if len(message.TimeVector) > 0 {
		fields["time_vector"] = message.TimeVector
	}
	if len(message.DomainVector) > 0 {
		fields["domain_vector"] = message.DomainVector
	}

	// 添加多维度元数据
	if message.MultiDimMetadata != nil {
		if multiDimBytes, err := json.Marshal(message.MultiDimMetadata); err == nil {
			fields["multi_dim_metadata"] = string(multiDimBytes)
		}
	}

	// 构建文档
	doc := map[string]interface{}{
		"id":     message.Message.ID,
		"vector": message.Message.Vector, // 使用基础向量作为主向量
		"fields": fields,
	}

	// 构建插入请求
	insertReq := map[string]interface{}{
		"docs": []map[string]interface{}{doc},
	}

	// 序列化请求
	reqBody, err := json.Marshal(insertReq)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 序列化插入请求失败: %v", err)
		return fmt.Errorf("序列化插入请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs", s.VectorDBURL, s.VectorDBCollection)
	log.Printf("[增强向量存储] 发送存储请求: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[增强向量存储] 错误: 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 发送HTTP请求失败: %v", err)
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[增强向量存储] 错误: 读取响应失败: %v", err)
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("[增强向量存储] 错误: HTTP状态码 %d, 响应: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("向量存储失败: HTTP %d, %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[增强向量存储] 错误: 解析响应失败: %v", err)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		return fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	log.Printf("[增强向量存储] 增强消息存储成功: ID=%s", message.Message.ID)
	return nil
}

// SearchMessages 在向量数据库中搜索相似消息
func (s *VectorService) SearchMessages(vector []float32, sessionID string, topK int) ([]models.SearchResult, error) {
	if topK <= 0 {
		topK = 5 // 默认返回5个结果
	}

	// 构建过滤条件（可选，只搜索特定会话的记忆）
	var filter string
	if sessionID != "" {
		filter = fmt.Sprintf("session_id = '%s'", sessionID)
	}

	// 构建请求体
	searchReq := map[string]interface{}{
		"vector":         vector,
		"topk":           topK,
		"include_vector": false,
	}

	// 如果有过滤条件，添加到请求中
	if filter != "" {
		searchReq["filter"] = filter
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录请求信息 - 添加颜色
	log.Printf("%s[消息搜索-请求] 会话ID=%s, topK=%d, 向量维度=%d%s",
		colorCyan, sessionID, topK, len(vector), colorReset)

	// 记录请求体 - 添加颜色
	log.Printf("%s[消息搜索-请求体] %s%s", colorCyan, string(reqBody), colorReset)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录原始响应 - 添加颜色
	log.Printf("%s[消息搜索-响应体] %s%s", colorCyan, string(respBody), colorReset)

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			role := "unknown"
			if r, ok := item.Fields["role"].(string); ok {
				role = r
			}

			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":      item.Fields["content"],
					"role":         role,
					"session_id":   item.Fields["session_id"],
					"content_type": item.Fields["content_type"],
					"priority":     item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[向量搜索] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[向量搜索] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[向量搜索] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 构造返回结果
	var searchResults []models.SearchResult
	var mostSimilarItem *models.SearchResult
	var smallestScore float64 = 999.0 // 初始化为一个很大的值

	log.Printf("[向量搜索] 开始评估数据，相似度阈值: %.4f (小于等于此值视为相关)", s.SimilarityThreshold)

	for _, item := range result.Output {
		// 应用相似度阈值过滤（余弦距离：越小越相似）
		if item.Score <= s.SimilarityThreshold {
			newResult := models.SearchResult{
				ID:     item.Id,
				Score:  item.Score,
				Fields: item.Fields,
			}
			searchResults = append(searchResults, newResult)

			role := "unknown"
			if r, ok := item.Fields["role"].(string); ok {
				role = r
			}

			log.Printf("[向量搜索] 符合条件的消息: ID=%s, 角色=%s, 相似度=%.4f (小于等于阈值 %.4f)",
				item.Id, role, item.Score, s.SimilarityThreshold)

			// 跟踪最相似的结果（得分最小）
			if item.Score < smallestScore {
				smallestScore = item.Score
				mostSimilarItem = &models.SearchResult{
					ID:     item.Id,
					Score:  item.Score,
					Fields: item.Fields,
				}
			}
		} else {
			role := "unknown"
			if r, ok := item.Fields["role"].(string); ok {
				role = r
			}

			log.Printf("[向量搜索] 过滤掉的消息: ID=%s, 角色=%s, 相似度=%.4f (大于阈值 %.4f)",
				item.Id, role, item.Score, s.SimilarityThreshold)
		}
	}

	// 输出最相似结果信息
	if mostSimilarItem != nil {
		content, _ := mostSimilarItem.Fields["content"].(string)
		role, _ := mostSimilarItem.Fields["role"].(string)
		log.Printf("[向量搜索] 最相似消息: ID=%s, 角色=%s, 相似度=%.4f, 内容=%s",
			mostSimilarItem.ID, role, mostSimilarItem.Score, content)

		// 输出完整的最佳匹配记录
		bestMatchJSON, _ := json.MarshalIndent(mostSimilarItem, "", "  ")
		log.Printf("[消息搜索-最终选择] 得分最低的记录完整数据:\n%s", string(bestMatchJSON))
	} else {
		log.Printf("[向量搜索] 未找到符合阈值的相关消息")
	}

	log.Printf("[向量检索] 查询结果: 找到 %d 条记录, 过滤后保留 %d 条, 请求ID: %s",
		len(result.Output), len(searchResults), result.RequestId)
	log.Printf("==================================================== 消息搜索完成 ====================================================")
	return searchResults, nil
}

// EnsureCollection 确保向量集合存在
func (s *VectorService) EnsureCollection() error {
	// 首先检查集合是否存在
	exists, err := s.CheckCollectionExists(s.VectorDBCollection)
	if err != nil {
		return fmt.Errorf("检查集合是否存在时出错: %w", err)
	}

	if exists {
		log.Printf("[向量服务] 集合 %s 已存在", s.VectorDBCollection)
		return nil
	}

	// 集合不存在，创建新集合
	return s.CreateCollection(s.VectorDBCollection, s.VectorDBDimension, s.VectorDBMetric)
}

// CheckCollectionExists 检查集合是否存在
func (s *VectorService) CheckCollectionExists(name string) (bool, error) {
	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s", s.VectorDBURL, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 如果返回404，表示集合不存在
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("解析响应失败: %w", err)
	}

	// 判断集合是否存在
	if result.Code == 0 {
		return true, nil
	}

	// 其他错误
	if result.Message == "Collection not exist" ||
		result.Message == "Collection not exists" ||
		result.Message == "Collection doesn't exist" {
		return false, nil
	}

	return false, fmt.Errorf("检查集合是否存在失败: %d, %s", result.Code, result.Message)
}

// ListCollections 列出所有集合
func (s *VectorService) ListCollections() ([]map[string]interface{}, error) {
	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections", s.VectorDBURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 为了调试，记录完整响应
	log.Printf("[向量服务] 列出集合响应: %s", string(respBody))

	// 尝试解析为带有字符串输出的结构
	var result struct {
		Code      int      `json:"code"`
		Message   string   `json:"message"`
		RequestId string   `json:"request_id"`
		Output    []string `json:"output"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应: %s", err, string(respBody))
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 将字符串集合名称转换为映射结构
	var collections []map[string]interface{}
	for _, name := range result.Output {
		collections = append(collections, map[string]interface{}{
			"name": name,
		})
	}

	return collections, nil
}

// CreateCollection 创建新集合
func (s *VectorService) CreateCollection(name string, dimension int, metric string) error {
	log.Printf("[向量服务] 开始创建集合 %s...", name)

	// 构建创建集合请求
	createReq := map[string]interface{}{
		"name":      name,
		"dimension": dimension,
		"metric":    metric,
		"fields_schema": map[string]string{
			"session_id":   "STRING",
			"content":      "STRING",
			"role":         "STRING",
			"content_type": "STRING",
			"timestamp":    "INT",
			"priority":     "STRING",
			"metadata":     "STRING",
		},
	}

	// 序列化请求
	reqBody, err := json.Marshal(createReq)
	if err != nil {
		return fmt.Errorf("序列化创建集合请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections", s.VectorDBURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应
	if result.Code != 0 {
		// 特殊情况：如果集合已存在，视为成功
		if resp.StatusCode == http.StatusBadRequest &&
			(result.Message == "Collection already exist" ||
				result.Message == "Collection already exists") {
			log.Printf("[向量服务] 集合 %s 已存在，直接使用", name)
			return nil
		}
		return fmt.Errorf("创建集合失败: %d, %s", result.Code, result.Message)
	}

	log.Printf("[向量服务] 集合 %s 创建成功!", name)
	return nil
}

// DeleteCollection 删除集合
func (s *VectorService) DeleteCollection(name string) error {
	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s", s.VectorDBURL, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		// 特殊情况：如果集合不存在，也视为成功
		if result.Message == "Collection not exist" ||
			result.Message == "Collection not exists" ||
			result.Message == "Collection doesn't exist" {
			log.Printf("[向量服务] 集合 %s 不存在，无需删除", name)
			return nil
		}
		return fmt.Errorf("删除集合失败: %d, %s", result.Code, result.Message)
	}

	log.Printf("[向量服务] 集合 %s 删除成功!", name)
	return nil
}

// GetDimension 获取向量维度
func (s *VectorService) GetDimension() int {
	return s.VectorDBDimension
}

// GetMetric 获取向量相似度度量方式
func (s *VectorService) GetMetric() string {
	return s.VectorDBMetric
}

// AddSearchByIDDirect 添加一个直接通过ID获取记录的函数，绕过向量查询API
func (s *VectorService) SearchByIDDirect(id string) ([]models.SearchResult, error) {
	// 查询单个记录的API - 尝试使用RESTful格式
	url := fmt.Sprintf("%s/v1/collections/%s/docs/%s", s.VectorDBURL, s.VectorDBCollection, id)
	// 添加颜色
	log.Printf("%s[ID直接搜索-请求] 请求URL: %s%s", colorCyan, url, colorReset)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[ID直接搜索] 创建HTTP请求失败: %v", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ID直接搜索] 发送请求失败: %v", err)
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ID直接搜索] 读取响应失败: %v", err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录原始响应 - 添加颜色
	log.Printf("%s[ID直接搜索-响应] 状态码=%d, 响应体=%s%s", colorCyan, resp.StatusCode, string(respBody), colorReset)

	// 检查状态码 - 404表示未找到
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[ID直接搜索] 未找到ID=%s的记录，状态码: %d", id, resp.StatusCode)
		return []models.SearchResult{}, nil
	}

	// 检查其他错误状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[ID直接搜索] API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return []models.SearchResult{}, nil
	}

	// 解析响应 - 根据阿里云API文档调整
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    struct {
			Id     string                 `json:"id"`
			Vector []float32              `json:"vector,omitempty"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		log.Printf("[ID直接搜索] API返回错误: %d, %s", result.Code, result.Message)
		return []models.SearchResult{}, nil
	}

	// 美化JSON输出
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		log.Printf("[ID直接搜索-响应] 美化格式输出:\n%s", string(prettyJSON))
	}

	// 构造返回结果
	searchResult := models.SearchResult{
		ID:     result.Output.Id,
		Score:  0, // 直接获取不计算相似度
		Fields: result.Output.Fields,
	}

	log.Printf("[ID直接搜索] 找到记录, ID=%s", id)
	log.Printf("==================================================== 直接ID搜索完成 ====================================================")
	return []models.SearchResult{searchResult}, nil
}

// SearchByID 通过ID搜索记录
func (s *VectorService) SearchByID(id string, fieldName string) ([]models.SearchResult, error) {
	if fieldName == "" {
		fieldName = "id" // 默认按ID字段检索
	}

	// 定义请求体
	searchReq := map[string]interface{}{
		"topk":           200, // 增加返回上限
		"include_vector": false,
	}

	// 根据字段类型构建不同的请求
	if fieldName == "id" {
		// 当查询主ID时，使用id参数（符合阿里云API规范）
		log.Printf("[ID搜索] 使用主键ID查询: %s", id)
		searchReq["id"] = id
	} else if strings.Contains(fieldName, "batchId") {
		// 对于metadata中的批次ID字段，也使用id参数进行主键检索而不是filter
		log.Printf("[ID搜索] 使用批次ID作为主键查询: %s", id)
		searchReq["id"] = id
	} else {
		// 其他字段直接匹配filter
		filter := fmt.Sprintf("%s = '%s'", fieldName, id)
		log.Printf("[ID搜索] 使用字段匹配，过滤条件: %s", filter)
		searchReq["filter"] = filter
	}

	// 序列化请求
	reqBodyBytes, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	reqBodyStr := string(reqBodyBytes)
	// 添加颜色
	log.Printf("%s[ID搜索-请求体] %s%s", colorCyan, reqBodyStr, colorReset)

	// 记录请求信息 - 添加颜色
	log.Printf("%s[ID搜索-请求] 字段=%s, ID值=%s, 请求URL=%s/v1/collections/%s/query%s",
		colorCyan, fieldName, id, s.VectorDBURL, s.VectorDBCollection, colorReset)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 打印完整响应以便调试 - 添加颜色
	log.Printf("%s[ID搜索-响应体] %s%s", colorCyan, string(respBody), colorReset)

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 美化JSON输出 - 添加为检索服务响应美化格式输出
	if len(result.Output) > 0 {
		// 创建一个格式化的JSON输出
		prettyJSON, err := json.MarshalIndent(result, "", "  ")
		if err == nil {
			log.Printf("[ID搜索-响应] 美化格式输出:\n%s", string(prettyJSON))
		}
	}

	// 在这里添加一个最终选择记录的输出逻辑
	var bestMatch *struct {
		Id     string                 `json:"id"`
		Score  float64                `json:"score"`
		Fields map[string]interface{} `json:"fields"`
	}
	var hasBestMatch bool
	var smallestScore float64 = 999.0 // 初始化为一个足够大的值

	// 记录相似度阈值用于筛选
	log.Printf("[ID搜索] 开始评估数据，相似度阈值: %.4f (小于等于此值视为相关)", s.SimilarityThreshold)

	// 先筛选符合阈值的记录，然后从中找出得分最低的
	for i, item := range result.Output {
		// 应用相似度阈值过滤（与其他搜索函数一致）
		if item.Score <= s.SimilarityThreshold {
			log.Printf("[ID搜索] 符合条件的数据项: ID=%s, 相似度=%.4f (小于等于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)

			// 初始化最佳匹配或更新为更相似（分数更低）的匹配
			if !hasBestMatch || item.Score < smallestScore {
				// 直接存储数组中的元素的索引，而不是指针
				bestMatch = &result.Output[i]
				smallestScore = item.Score
				hasBestMatch = true
			}
		} else {
			log.Printf("[ID搜索] 过滤掉的数据项: ID=%s, 相似度=%.4f (大于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)
		}
	}

	// 输出最相似结果信息
	if hasBestMatch {
		// 输出完整的最佳匹配记录
		bestMatchJSON, _ := json.MarshalIndent(bestMatch, "", "  ")
		log.Printf("[ID搜索-最终选择] 得分最低的记录完整数据:\n%s", string(bestMatchJSON))

		// 同时添加简洁日志
		content, _ := bestMatch.Fields["content"].(string)
		contentPreview := content
		if len(contentPreview) > 50 {
			contentPreview = contentPreview[:50] + "..."
		}
		log.Printf("[ID搜索-最终选择] ID=%s, 相似度=%.4f, 内容预览=%s",
			bestMatch.Id, bestMatch.Score, contentPreview)
	} else {
		log.Printf("[ID搜索] 未找到符合阈值的相关数据")
	}

	// 构造返回结果 - 修改为只返回符合相似度阈值的结果
	var searchResults []models.SearchResult

	// 修改返回逻辑：如果找到了符合条件的最佳匹配，只返回它
	// 如果没有符合条件的结果，返回空结果集
	if hasBestMatch {
		searchResults = append(searchResults, models.SearchResult{
			ID:     bestMatch.Id,
			Score:  bestMatch.Score,
			Fields: bestMatch.Fields,
		})
		log.Printf("[ID搜索] 筛选后返回 1 条符合阈值的记录，ID=%s, 相似度=%.4f",
			bestMatch.Id, bestMatch.Score)
	} else {
		log.Printf("[ID搜索] 筛选后没有符合阈值的结果，返回空结果集")
	}

	log.Printf("[ID搜索] 找到 %d 条原始记录，筛选后保留 %d 条，ID=%s, 字段=%s",
		len(result.Output), len(searchResults), id, fieldName)
	log.Printf("==================================================== ID搜索完成 ====================================================")
	return searchResults, nil
}

// SearchBySessionID 通过会话ID搜索记录
func (s *VectorService) SearchBySessionID(sessionID string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 50 // 默认返回50条记录
	}

	// 构建过滤条件 - 精确匹配sessionID
	filter := fmt.Sprintf("session_id = '%s'", sessionID)

	// 构建请求体
	searchReq := map[string]interface{}{
		"filter":         filter,
		"topk":           limit,
		"include_vector": false,
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录请求信息
	log.Printf("[会话搜索] 请求信息: 会话ID=%s, 限制=%d", sessionID, limit)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			role := "unknown"
			if r, ok := item.Fields["role"].(string); ok {
				role = r
			}

			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":      item.Fields["content"],
					"role":         role,
					"session_id":   item.Fields["session_id"],
					"content_type": item.Fields["content_type"],
					"priority":     item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[会话搜索] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[会话搜索] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[会话搜索] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 构造返回结果
	var searchResults []models.SearchResult
	for _, item := range result.Output {
		searchResults = append(searchResults, models.SearchResult{
			ID:     item.Id,
			Score:  item.Score,
			Fields: item.Fields,
		})
	}

	log.Printf("[会话搜索] 找到 %d 条记录，会话ID=%s", len(result.Output), sessionID)
	log.Printf("==================================================== 会话ID搜索完成 ====================================================")
	return searchResults, nil
}

// SearchByFilter 通过自定义过滤条件搜索记录
func (s *VectorService) SearchByFilter(filter string, limit int) ([]models.SearchResult, error) {
	log.Printf("\n[过滤搜索] ======================= 开始执行过滤搜索 =======================")
	log.Printf("[过滤搜索] 执行过滤条件搜索, 过滤条件: %s, 限制数量: %d", filter, limit)

	if limit <= 0 {
		limit = 50 // 默认返回50条记录
	}

	// 构建请求体
	searchReq := map[string]interface{}{
		"filter":         filter,
		"topk":           limit,
		"include_vector": false,
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录详细的请求信息
	log.Printf("[过滤搜索] 完整请求体: %s", string(reqBody))
	log.Printf("[过滤搜索] 请求URL: %s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)
	log.Printf("[过滤搜索] 请求头: Content-Type=application/json, API密钥长度=%d", len(s.VectorDBAPIKey))

	// 发送请求
	startTime := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	requestDuration := time.Since(startTime)
	if err != nil {
		log.Printf("[过滤搜索] 请求失败: %v, 耗时: %v", err, requestDuration)
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[过滤搜索] 请求已发送，HTTP状态: %d, 耗时: %v", resp.StatusCode, requestDuration)

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录完整的原始响应
	log.Printf("[过滤搜索] 原始响应体: %s", string(respBody))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[过滤搜索] 错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[过滤搜索] 响应解析失败: %v, 原始响应: %s", err, string(respBody))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			role := "unknown"
			if r, ok := item.Fields["role"].(string); ok {
				role = r
			}

			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":      item.Fields["content"],
					"role":         role,
					"session_id":   item.Fields["session_id"],
					"content_type": item.Fields["content_type"],
					"priority":     item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[过滤搜索] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[过滤搜索] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[过滤搜索] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 构造返回结果
	var searchResults []models.SearchResult
	for _, item := range result.Output {
		searchResults = append(searchResults, models.SearchResult{
			ID:     item.Id,
			Score:  item.Score,
			Fields: item.Fields,
		})
	}

	log.Printf("[过滤搜索] 找到 %d 条记录，过滤条件=%s", len(result.Output), filter)
	log.Printf("==================================================== 过滤搜索完成 ====================================================")
	return searchResults, nil
}

// SearchByKeywordsFilter 通过关键词过滤条件搜索记录
func (s *VectorService) SearchByKeywordsFilter(field string, value string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 50 // 默认返回50条记录
	}

	// 构建过滤条件 - 使用标准格式
	filter := fmt.Sprintf("%s = \"%s\"", field, value)
	log.Printf("[关键词过滤] 使用条件: %s", filter)

	// 构建请求体
	searchReq := map[string]interface{}{
		"filter":         filter,
		"topk":           limit,
		"include_vector": false,
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录请求信息
	log.Printf("[关键词过滤] 请求信息: 过滤字段=%s, 值=%s, 限制=%d", field, value, limit)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录响应信息
	log.Printf("[关键词过滤-响应体] %s", string(respBody))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":    item.Fields["content"],
					"session_id": item.Fields["session_id"],
					"priority":   item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[关键词过滤] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[关键词过滤] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[关键词过滤] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 构造返回结果 - 应用相似度阈值过滤
	var searchResults []models.SearchResult
	var mostSimilarItem *models.SearchResult
	var smallestScore float64 = 999.0 // 初始化为一个很大的值

	log.Printf("[关键词过滤] 开始评估数据，相似度阈值: %.4f (小于等于此值视为相关)", s.SimilarityThreshold)

	for _, item := range result.Output {
		// 应用相似度阈值过滤（余弦距离：越小越相似）
		if item.Score <= s.SimilarityThreshold {
			newResult := models.SearchResult{
				ID:     item.Id,
				Score:  item.Score,
				Fields: item.Fields,
			}
			searchResults = append(searchResults, newResult)

			log.Printf("[关键词过滤] 符合条件的数据项: ID=%s, 相似度=%.4f (小于等于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)

			// 跟踪最相似的结果（得分最小）
			if item.Score < smallestScore {
				smallestScore = item.Score
				mostSimilarItem = &models.SearchResult{
					ID:     item.Id,
					Score:  item.Score,
					Fields: item.Fields,
				}
			}
		} else {
			log.Printf("[关键词过滤] 过滤掉的数据项: ID=%s, 相似度=%.4f (大于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)
		}
	}

	// 输出最相似结果信息
	if mostSimilarItem != nil {
		content, _ := mostSimilarItem.Fields["content"].(string)
		contentPreview := content
		if len(contentPreview) > 50 {
			contentPreview = contentPreview[:50] + "..."
		}
		log.Printf("[关键词过滤] 最相似数据项: ID=%s, 相似度=%.4f, 内容预览=%s",
			mostSimilarItem.ID, mostSimilarItem.Score, contentPreview)

		// 输出完整的最佳匹配记录
		bestMatchJSON, _ := json.MarshalIndent(mostSimilarItem, "", "  ")
		log.Printf("[关键词过滤-最终选择] 得分最低的记录完整数据:\n%s", string(bestMatchJSON))
	} else {
		log.Printf("[关键词过滤] 未找到符合阈值的相关数据")
	}

	log.Printf("[关键词过滤] 找到 %d 条原始记录，筛选后保留 %d 条，字段=%s, 值=%s",
		len(result.Output), len(searchResults), field, value)
	log.Printf("==================================================== 关键词过滤搜索完成 ====================================================")
	return searchResults, nil
}

// SearchVectorsAdvanced 增强现有的 SearchVectors 函数，支持高级参数
func (s *VectorService) SearchVectorsAdvanced(vector []float32, sessionID string, topK int, options map[string]interface{}) ([]models.SearchResult, error) {
	if topK <= 0 {
		topK = 5 // 默认返回5个结果
	}

	// 构建过滤条件（可选，只搜索特定会话的记忆）
	var filter string
	if sessionID != "" {
		filter = fmt.Sprintf("session_id = '%s'", sessionID)
	}

	// 如果options中提供了filter，优先使用options中的filter
	if optFilter, ok := options["filter"].(string); ok && optFilter != "" {
		filter = optFilter
	}

	// 构建请求体
	searchReq := map[string]interface{}{
		"vector":         vector,
		"topk":           topK,
		"include_vector": false,
	}

	// 如果有过滤条件，添加到请求中
	if filter != "" {
		searchReq["filter"] = filter
	}

	// 添加向量搜索参数
	if vectorParams, ok := options["vector_param"].(map[string]interface{}); ok {
		searchReq["vector_param"] = vectorParams
	} else {
		// 如果未提供向量参数，但需要设置更宽松的相似度阈值，添加默认参数
		if _, wideSimilarity := options["wide_similarity"]; wideSimilarity {
			searchReq["vector_param"] = map[string]interface{}{
				"radius": s.SimilarityThreshold * 1.5, // 放宽相似度阈值
				"ef":     100,                         // 增加搜索效率
			}
		}
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	// 记录请求信息 - 添加颜色
	log.Printf("%s[高级向量搜索-请求] 会话ID=%s, topK=%d, 向量维度=%d%s",
		colorCyan, sessionID, topK, len(vector), colorReset)

	// 记录请求体摘要 - 避免输出完整向量数据
	reqSummary := fmt.Sprintf("{\"topk\":%d,\"include_vector\":%v,\"filter\":\"%s\",\"vector\":\"[%d维向量数据已省略]\"}",
		topK, false, fmt.Sprintf("userId=\"%s\"", sessionID), len(vector))
	log.Printf("%s[高级向量搜索-请求体摘要] %s%s", colorCyan, reqSummary, colorReset)

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", s.VectorDBURL, s.VectorDBCollection)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录原始响应 - 添加颜色
	log.Printf("%s[高级向量搜索-响应体] %s%s", colorCyan, string(respBody), colorReset)

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			Id     string                 `json:"id"`
			Score  float64                `json:"score"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 美化JSON输出
	var prettyJSON bytes.Buffer
	if len(result.Output) > 0 {
		// 创建一个简化版的结果用于日志记录
		simplifiedOutput := make([]map[string]interface{}, 0, len(result.Output))
		for _, item := range result.Output {
			simplifiedOutput = append(simplifiedOutput, map[string]interface{}{
				"id":    item.Id,
				"score": item.Score,
				"fields": map[string]interface{}{
					"content":    item.Fields["content"],
					"session_id": item.Fields["session_id"],
					"priority":   item.Fields["priority"],
				},
			})
		}

		// 构建简化版结果
		simplified := map[string]interface{}{
			"code":      result.Code,
			"message":   result.Message,
			"requestId": result.RequestId,
			"output":    simplifiedOutput,
		}

		// 格式化为美观的JSON
		encoder := json.NewEncoder(&prettyJSON)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(simplified); err == nil {
			log.Printf("[高级向量搜索] 响应体 (美化格式):\n%s", prettyJSON.String())
		} else {
			log.Printf("[高级向量搜索] 响应解析失败: %v", err)
		}
	} else {
		log.Printf("[高级向量搜索] 未找到匹配结果")
	}

	// 检查API结果码
	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	// 是否跳过阈值过滤
	skipFilter := false
	if skip, ok := options["skip_threshold_filter"].(bool); ok {
		skipFilter = skip
	}

	// 构造返回结果
	var searchResults []models.SearchResult
	var mostSimilarItem *models.SearchResult
	var smallestScore float64 = 999.0 // 初始化为一个很大的值

	log.Printf("[高级向量搜索] 开始评估数据，相似度阈值: %.4f (小于等于此值视为相关)", s.SimilarityThreshold)

	for _, item := range result.Output {
		// 应用相似度阈值过滤（余弦距离：越小越相似）
		if skipFilter || item.Score <= s.SimilarityThreshold {
			newResult := models.SearchResult{
				ID:     item.Id,
				Score:  item.Score,
				Fields: item.Fields,
			}
			searchResults = append(searchResults, newResult)

			log.Printf("[高级向量搜索] 符合条件的数据项: ID=%s, 相似度=%.4f",
				item.Id, item.Score)

			// 跟踪最相似的结果（得分最小）
			if item.Score < smallestScore {
				smallestScore = item.Score
				mostSimilarItem = &models.SearchResult{
					ID:     item.Id,
					Score:  item.Score,
					Fields: item.Fields,
				}
			}
		} else {
			log.Printf("[高级向量搜索] 过滤掉的数据项: ID=%s, 相似度=%.4f (大于阈值 %.4f)",
				item.Id, item.Score, s.SimilarityThreshold)
		}
	}

	// 输出最相似结果信息
	if mostSimilarItem != nil {
		content, _ := mostSimilarItem.Fields["content"].(string)
		contentPreview := content
		if len(contentPreview) > 50 {
			contentPreview = contentPreview[:50] + "..."
		}
		log.Printf("[高级向量搜索] 最相似数据项: ID=%s, 相似度=%.4f, 内容预览=%s",
			mostSimilarItem.ID, mostSimilarItem.Score, contentPreview)

		// 输出完整的最佳匹配记录
		bestMatchJSON, _ := json.MarshalIndent(mostSimilarItem, "", "  ")
		log.Printf("[高级向量搜索-最终选择] 得分最低的记录完整数据:\n%s", string(bestMatchJSON))
	} else {
		log.Printf("[高级向量搜索] 未找到符合阈值的相关数据")
	}

	log.Printf("[高级向量检索] 查询结果: 找到 %d 条记录, 过滤后保留 %d 条",
		len(result.Output), len(searchResults))
	log.Printf("==================================================== 高级向量搜索完成 ====================================================")
	return searchResults, nil
}

// CountSessionMemories 统计指定会话的记忆数量
func (s *VectorService) CountSessionMemories(sessionID string) (int, error) {
	log.Printf("\n[向量搜索] 开始统计会话记忆 ============================")
	log.Printf("[向量搜索] 会话ID: %s", sessionID)

	// 构建过滤查询请求体
	filter := fmt.Sprintf(`fields.session_id = "%s"`, sessionID)
	requestBody := map[string]interface{}{
		"filter": filter,
		"limit":  1, // 只需要计数，不需要实际数据
	}

	// 序列化请求
	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return 0, fmt.Errorf("序列化统计请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs/count", s.VectorDBURL, s.VectorDBCollection)
	log.Printf("[向量搜索] 发送记忆计数请求: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", s.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Output  struct {
			Count int `json:"count"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API结果码
	if result.Code != 0 {
		return 0, fmt.Errorf("API返回错误: %d, %s", result.Code, result.Message)
	}

	return result.Output.Count, nil
}

// UserInfo类型现在定义在models包中

const (
	UserCollectionName = "context_keeper_users" // 用户信息集合名称
)

// CheckUserIDUniqueness 检查用户ID唯一性
func (vs *VectorService) CheckUserIDUniqueness(userID string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("用户ID不能为空")
	}

	log.Printf("[向量服务] 开始检查用户ID唯一性: %s", userID)

	// 确保用户集合已初始化
	if err := vs.InitUserCollection(); err != nil {
		log.Printf("[向量服务] 初始化用户集合失败: %v", err)
		return false, fmt.Errorf("初始化用户集合失败: %w", err)
	}

	// 构造查询请求
	searchRequest := map[string]interface{}{
		"filter":        fmt.Sprintf(`fields.userId = "%s"`, userID),
		"limit":         1,
		"output_fields": []string{"fields.userId"},
	}

	// 序列化请求
	reqBody, err := json.Marshal(searchRequest)
	if err != nil {
		log.Printf("[向量服务] 序列化查询请求失败: %v", err)
		return false, fmt.Errorf("序列化查询请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", vs.VectorDBURL, UserCollectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[向量服务] 创建HTTP请求失败: %v", err)
		return false, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", vs.VectorDBAPIKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[向量服务] 用户ID唯一性检查请求失败: %v", err)
		return false, fmt.Errorf("用户ID唯一性检查失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[向量服务] 读取响应失败: %v", err)
		return false, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[向量服务] API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		// 如果是404错误（集合不存在），认为用户ID是唯一的
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[向量服务] 用户集合不存在，用户ID唯一: %s", userID)
			return true, nil
		}
		return false, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[向量服务] 解析用户ID检查响应失败: %v", err)
		return false, fmt.Errorf("解析用户ID检查响应失败: %w", err)
	}

	// 检查是否找到匹配的用户ID
	found := len(result.Data) > 0
	if found {
		// 进一步精确验证userId字段
		for _, item := range result.Data {
			if foundUserID, ok := item["userId"].(string); ok && foundUserID == userID {
				log.Printf("[向量服务] 用户ID已存在: %s", userID)
				return false, nil // 用户ID已存在，不唯一
			}
		}
	}

	log.Printf("[向量服务] 用户ID唯一，可以使用: %s", userID)
	return true, nil // 用户ID唯一，可以使用
}

// StoreUserInfo 存储用户信息到向量数据库
func (vs *VectorService) StoreUserInfo(userInfo *models.UserInfo) error {
	if userInfo.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	log.Printf("🔥 [向量服务-用户存储] ===== 开始存储用户信息: %s =====", userInfo.UserID)
	log.Printf("📝 [向量服务-用户存储] 用户信息详情: UserID=%s, FirstUsed=%s, LastActive=%s",
		userInfo.UserID, userInfo.FirstUsed, userInfo.LastActive)
	log.Printf("📝 [向量服务-用户存储] 设备信息: %+v", userInfo.DeviceInfo)
	log.Printf("📝 [向量服务-用户存储] 目标集合: %s", UserCollectionName)

	// 设置时间戳
	now := time.Now().Format(time.RFC3339)
	if userInfo.CreatedAt == "" {
		userInfo.CreatedAt = now
		log.Printf("📅 [向量服务-用户存储] 设置创建时间: %s", userInfo.CreatedAt)
	}
	userInfo.UpdatedAt = now
	log.Printf("📅 [向量服务-用户存储] 设置更新时间: %s", userInfo.UpdatedAt)

	// 生成文本向量
	vectorText := fmt.Sprintf("user %s %s", userInfo.UserID, userInfo.FirstUsed)
	log.Printf("🔧 [向量服务-用户存储] 生成向量文本: %s", vectorText)

	vector, err := vs.GenerateEmbedding(vectorText)
	if err != nil {
		log.Printf("❌ [向量服务-用户存储] 生成用户信息向量失败: %v", err)
		return fmt.Errorf("生成用户信息向量失败: %w", err)
	}
	log.Printf("✅ [向量服务-用户存储] 向量生成成功，维度: %d", len(vector))

	// 生成唯一的文档ID
	documentID := fmt.Sprintf("user_%s_%d", userInfo.UserID, time.Now().Unix())
	log.Printf("🔑 [向量服务-用户存储] 生成文档ID: %s", documentID)

	// 序列化复杂字段为JSON字符串，确保向量数据库兼容性
	var deviceInfoStr, metadataStr string
	if userInfo.DeviceInfo != nil {
		if deviceInfoBytes, err := json.Marshal(userInfo.DeviceInfo); err == nil {
			deviceInfoStr = string(deviceInfoBytes)
		} else {
			log.Printf("⚠️ [向量服务-用户存储] 序列化设备信息失败: %v", err)
			deviceInfoStr = "{}"
		}
	} else {
		deviceInfoStr = "{}"
	}

	if userInfo.Metadata != nil {
		if metadataBytes, err := json.Marshal(userInfo.Metadata); err == nil {
			metadataStr = string(metadataBytes)
		} else {
			log.Printf("⚠️ [向量服务-用户存储] 序列化元数据失败: %v", err)
			metadataStr = "{}"
		}
	} else {
		metadataStr = "{}"
	}

	log.Printf("📦 [向量服务-用户存储] 序列化设备信息: %s", deviceInfoStr)
	log.Printf("📦 [向量服务-用户存储] 序列化元数据: %s", metadataStr)

	// 构建文档 - 使用字符串字段确保兼容性
	doc := map[string]interface{}{
		"id":     documentID,
		"vector": vector,
		"fields": map[string]interface{}{
			"userId":     userInfo.UserID,
			"firstUsed":  userInfo.FirstUsed,
			"lastActive": userInfo.LastActive,
			"deviceInfo": deviceInfoStr, // 序列化为JSON字符串
			"metadata":   metadataStr,   // 序列化为JSON字符串
			"createdAt":  userInfo.CreatedAt,
			"updatedAt":  userInfo.UpdatedAt,
		},
	}
	log.Printf("📦 [向量服务-用户存储] 构建文档完成，字段数: %d", len(doc["fields"].(map[string]interface{})))

	// 构建插入请求
	insertReq := map[string]interface{}{
		"docs": []map[string]interface{}{doc},
	}

	// 序列化请求
	reqBody, err := json.Marshal(insertReq)
	if err != nil {
		log.Printf("❌ [向量服务-用户存储] 序列化插入请求失败: %v", err)
		return fmt.Errorf("序列化插入请求失败: %w", err)
	}
	log.Printf("📝 [向量服务-用户存储] 请求体大小: %d bytes", len(reqBody))

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/docs", vs.VectorDBURL, UserCollectionName)
	log.Printf("🌐 [向量服务-用户存储] 请求URL: %s", url)
	log.Printf("🌐 [向量服务-用户存储] 向量数据库URL: %s", vs.VectorDBURL)
	log.Printf("🌐 [向量服务-用户存储] 用户集合名称: %s", UserCollectionName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("❌ [向量服务-用户存储] 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", vs.VectorDBAPIKey)
	log.Printf("🔑 [向量服务-用户存储] 设置dashvector-auth-token头，API Key长度: %d", len(vs.VectorDBAPIKey))

	// 发送请求
	log.Printf("🚀 [向量服务-用户存储] 开始发送HTTP请求...")
	client := &http.Client{Timeout: 30 * time.Second}
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ [向量服务-用户存储] 存储用户信息请求失败: %v", err)
		return fmt.Errorf("存储用户信息失败: %w", err)
	}
	defer resp.Body.Close()
	requestDuration := time.Since(startTime)
	log.Printf("⏱️ [向量服务-用户存储] 请求耗时: %v", requestDuration)

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ [向量服务-用户存储] 读取响应失败: %v", err)
		return fmt.Errorf("读取响应失败: %w", err)
	}
	log.Printf("📨 [向量服务-用户存储] 响应状态码: %d", resp.StatusCode)
	log.Printf("📨 [向量服务-用户存储] 响应体长度: %d bytes", len(respBody))
	log.Printf("📨 [向量服务-用户存储] 响应体内容: %s", string(respBody))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ [向量服务-用户存储] 存储用户信息失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("存储用户信息失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应检查业务状态码
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("❌ [向量服务-用户存储] 解析响应失败: %v", err)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查业务状态码
	if result.Code != 0 {
		log.Printf("❌ [向量服务-用户存储] API返回业务错误: %d, %s", result.Code, result.Message)
		return fmt.Errorf("API返回业务错误: %d, %s", result.Code, result.Message)
	}

	log.Printf("✅ [向量服务-用户存储] 用户信息存储成功: %s", userInfo.UserID)
	log.Printf("🔥 [向量服务-用户存储] ===== 用户信息存储完成: %s =====", userInfo.UserID)
	return nil
}

// GetUserInfo 获取用户信息
func (vs *VectorService) GetUserInfo(userID string) (*models.UserInfo, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	log.Printf("🔥 [向量服务-用户查询] ===== 开始查询用户信息: %s =====", userID)
	log.Printf("📝 [向量服务-用户查询] 查询目标集合: %s", UserCollectionName)

	// 方案1：先尝试使用文档列表查询 (不使用过滤器)
	listRequest := map[string]interface{}{
		"limit":         100, // 获取更多文档以便查找
		"output_fields": []string{"userId", "firstUsed", "lastActive", "deviceInfo", "metadata", "createdAt", "updatedAt"},
	}
	log.Printf("📝 [向量服务-用户查询] 使用列表查询模式，不使用过滤器")

	// 序列化请求
	reqBody, err := json.Marshal(listRequest)
	if err != nil {
		log.Printf("❌ [向量服务-用户查询] 序列化查询请求失败: %v", err)
		return nil, fmt.Errorf("序列化查询请求失败: %w", err)
	}
	log.Printf("📝 [向量服务-用户查询] 请求体: %s", string(reqBody))

	// 创建HTTP请求
	url := fmt.Sprintf("%s/v1/collections/%s/query", vs.VectorDBURL, UserCollectionName)
	log.Printf("🌐 [向量服务-用户查询] 查询URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("❌ [向量服务-用户查询] 创建HTTP请求失败: %v", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("dashvector-auth-token", vs.VectorDBAPIKey)
	log.Printf("🔑 [向量服务-用户查询] 设置dashvector-auth-token头，API Key长度: %d", len(vs.VectorDBAPIKey))

	// 发送请求
	log.Printf("🚀 [向量服务-用户查询] 开始发送查询请求...")
	client := &http.Client{Timeout: 10 * time.Second}
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ [向量服务-用户查询] 查询用户信息请求失败: %v", err)
		return nil, fmt.Errorf("查询用户信息失败: %w", err)
	}
	defer resp.Body.Close()
	requestDuration := time.Since(startTime)
	log.Printf("⏱️ [向量服务-用户查询] 查询耗时: %v", requestDuration)

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ [向量服务-用户查询] 读取响应失败: %v", err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	log.Printf("📨 [向量服务-用户查询] 响应状态码: %d", resp.StatusCode)
	log.Printf("📨 [向量服务-用户查询] 响应体长度: %d bytes", len(respBody))
	log.Printf("📨 [向量服务-用户查询] 响应体内容: %s", string(respBody))

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ [向量服务-用户查询] 查询用户信息失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("查询用户信息失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应检查业务状态码
	var result struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		RequestId string `json:"request_id"`
		Output    []struct {
			ID     string                 `json:"id"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"output"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("❌ [向量服务-用户查询] 解析用户信息查询响应失败: %v", err)
		return nil, fmt.Errorf("解析用户信息查询响应失败: %w", err)
	}

	// 检查业务状态码
	if result.Code != 0 {
		log.Printf("❌ [向量服务-用户查询] API返回业务错误: %d, %s", result.Code, result.Message)
		if result.Code == -2976 {
			log.Printf("⚠️ [向量服务-用户查询] 认证失败，请检查API Key配置")
		}
		return nil, fmt.Errorf("API返回业务错误: %d, %s", result.Code, result.Message)
	}

	// 在结果中查找匹配的用户ID
	log.Printf("📊 [向量服务-用户查询] 获取到 %d 条文档，开始查找匹配用户", len(result.Output))
	var matchedItem *struct {
		ID     string                 `json:"id"`
		Fields map[string]interface{} `json:"fields"`
	}

	for i, item := range result.Output {
		log.Printf("📄 [向量服务-用户查询] 检查文档 %d: ID=%s", i+1, item.ID)
		log.Printf("📄 [向量服务-用户查询] 字段数据: %+v", item.Fields)

		// 检查字段中的userId
		if fieldsUserID := getStringFromFields(item.Fields, "userId"); fieldsUserID == userID {
			log.Printf("✅ [向量服务-用户查询] 找到匹配用户: ID=%s, 文档ID=%s", fieldsUserID, item.ID)
			matchedItem = &item
			break
		}

		// 同时检查文档ID是否匹配模式 user_{userId}_*
		expectedPrefix := fmt.Sprintf("user_%s_", userID)
		if strings.HasPrefix(item.ID, expectedPrefix) {
			log.Printf("✅ [向量服务-用户查询] 通过文档ID模式找到匹配: %s", item.ID)
			matchedItem = &item
			break
		}
	}

	// 检查是否找到用户
	if matchedItem == nil {
		log.Printf("⚠️ [向量服务-用户查询] 在 %d 条记录中未找到用户: %s", len(result.Output), userID)
		return nil, nil
	}
	log.Printf("✅ [向量服务-用户查询] 成功找到用户文档: %s", matchedItem.ID)

	// 解析用户信息，处理序列化字段
	userInfo := &models.UserInfo{
		UserID:     getStringFromFields(matchedItem.Fields, "userId"),
		FirstUsed:  getStringFromFields(matchedItem.Fields, "firstUsed"),
		LastActive: getStringFromFields(matchedItem.Fields, "lastActive"),
		CreatedAt:  getStringFromFields(matchedItem.Fields, "createdAt"),
		UpdatedAt:  getStringFromFields(matchedItem.Fields, "updatedAt"),
	}

	// 反序列化复杂字段
	deviceInfoStr := getStringFromFields(matchedItem.Fields, "deviceInfo")
	if deviceInfoStr != "" && deviceInfoStr != "{}" {
		var deviceInfo map[string]interface{}
		if err := json.Unmarshal([]byte(deviceInfoStr), &deviceInfo); err == nil {
			userInfo.DeviceInfo = deviceInfo
			log.Printf("📝 [向量服务-用户查询] 解析设备信息: %+v", deviceInfo)
		} else {
			log.Printf("⚠️ [向量服务-用户查询] 反序列化设备信息失败: %v", err)
			userInfo.DeviceInfo = make(map[string]interface{})
		}
	} else {
		userInfo.DeviceInfo = make(map[string]interface{})
	}

	metadataStr := getStringFromFields(matchedItem.Fields, "metadata")
	if metadataStr != "" && metadataStr != "{}" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			userInfo.Metadata = metadata
			log.Printf("📝 [向量服务-用户查询] 解析元数据: %+v", metadata)
		} else {
			log.Printf("⚠️ [向量服务-用户查询] 反序列化元数据失败: %v", err)
			userInfo.Metadata = make(map[string]interface{})
		}
	} else {
		userInfo.Metadata = make(map[string]interface{})
	}

	log.Printf("✅ [向量服务-用户查询] 用户信息查询成功: %s, 数据: %+v", userID, userInfo)
	log.Printf("🔥 [向量服务-用户查询] ===== 用户信息查询完成: %s =====", userID)
	return userInfo, nil
}

// getStringFromFields 安全地从fields map中获取字符串值
func getStringFromFields(fields map[string]interface{}, key string) string {
	if v, ok := fields[key].(string); ok {
		return v
	}
	return ""
}

// InitUserCollection 初始化用户信息集合
func (vs *VectorService) InitUserCollection() error {
	log.Printf("[向量服务] 开始初始化用户信息集合: %s", UserCollectionName)

	// 先检查集合是否已存在
	exists, err := vs.CheckCollectionExists(UserCollectionName)
	if err != nil {
		log.Printf("[向量服务] 检查用户集合是否存在失败: %v", err)
		return fmt.Errorf("检查用户集合是否存在失败: %w", err)
	}

	if exists {
		log.Printf("[向量服务] 用户信息集合已存在: %s", UserCollectionName)
		return nil
	}

	// 创建新集合
	err = vs.CreateCollection(UserCollectionName, vs.VectorDBDimension, vs.VectorDBMetric)
	if err != nil {
		log.Printf("[向量服务] 创建用户信息集合失败: %v", err)
		return fmt.Errorf("创建用户信息集合失败: %w", err)
	}

	log.Printf("[向量服务] 用户信息集合初始化成功: %s", UserCollectionName)
	return nil
}
