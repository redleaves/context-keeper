package aliyun

import (
	"context"
	"fmt"
	"log"

	"github.com/contextkeeper/service/internal/models"
)

// SearchWithText 使用文本查询搜索向量数据库
// 将自动生成文本的向量表示，并执行向量搜索
func (s *VectorService) SearchWithText(ctx context.Context, query string, limit int, skipThreshold bool) ([]models.SearchResult, error) {
	// 1. 生成查询的向量表示
	vector, err := s.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	log.Printf("[文本搜索] 已将查询文本'%s'转换为向量表示 (维度: %d)", query, len(vector))

	// 2. 使用向量执行高级搜索
	options := map[string]interface{}{}
	if skipThreshold {
		options["skip_threshold"] = true
	}

	// 执行高级向量搜索
	return s.SearchVectorsAdvanced(vector, "", limit, options)
}

// SearchWithTextAndFilters 使用文本查询和过滤条件搜索向量数据库
func (s *VectorService) SearchWithTextAndFilters(ctx context.Context, query string, limit int, filters map[string]interface{}, skipThreshold bool) ([]models.SearchResult, error) {
	// 1. 生成查询的向量表示
	vector, err := s.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 2. 构建过滤条件字符串
	var filterStr string
	if filters != nil && len(filters) > 0 {
		// 构建过滤器
		filterParts := make([]string, 0, len(filters))
		for key, value := range filters {
			// 根据值类型构建不同的过滤条件
			switch v := value.(type) {
			case string:
				// 如果是批次ID，使用包含匹配（针对metadata字段）
				if key == "batchId" {
					filterParts = append(filterParts, fmt.Sprintf("metadata LIKE '%%\"%s\":%%'", v))
				} else {
					filterParts = append(filterParts, fmt.Sprintf("%s = '%s'", key, v))
				}
			case int, int64, float32, float64:
				filterParts = append(filterParts, fmt.Sprintf("%s = %v", key, v))
			case bool:
				filterParts = append(filterParts, fmt.Sprintf("%s = %t", key, v))
			default:
				// 对于复杂类型，跳过
				log.Printf("[文本搜索] 跳过不支持的过滤器类型: %s", key)
			}
		}

		// 组合过滤条件
		if len(filterParts) > 0 {
			filterStr = filterParts[0]
			for i := 1; i < len(filterParts); i++ {
				filterStr += " AND " + filterParts[i]
			}
		}
	}

	// 记录搜索信息
	log.Printf("[文本过滤搜索] 查询: '%s', 过滤条件: '%s', 限制: %d, 跳过阈值: %t",
		query, filterStr, limit, skipThreshold)

	// 3. 准备搜索选项
	options := map[string]interface{}{}
	if skipThreshold {
		options["skip_threshold"] = true
	}
	if filterStr != "" {
		options["filter"] = filterStr
	}

	// 4. 执行高级向量搜索
	return s.SearchVectorsAdvanced(vector, "", limit, options)
}
