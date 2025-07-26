package aliyun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
