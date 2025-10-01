package vector

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// MockVectorEngine 模拟向量引擎实现
type MockVectorEngine struct {
	config    *VectorConfig
	documents map[string]*VectorDocument
	mu        sync.RWMutex
	enabled   bool
}

// NewMockVectorEngine 创建模拟向量引擎
func NewMockVectorEngine(config *VectorConfig) (*MockVectorEngine, error) {
	if config == nil {
		config = &VectorConfig{
			Provider:   "mock",
			Database:   "context_keeper_vector",
			Collection: "documents",
			Dimension:  1536, // OpenAI embedding维度
			IndexType:  "HNSW",
			MetricType: "cosine",
			MaxResults: 100,
			Timeout:    30 * time.Second,
		}
	}

	engine := &MockVectorEngine{
		config:    config,
		documents: make(map[string]*VectorDocument),
		enabled:   true,
	}

	// 初始化一些模拟数据
	engine.initMockData()

	log.Printf("✅ 模拟向量引擎初始化完成 - 维度: %d, 集合: %s",
		config.Dimension, config.Collection)

	return engine, nil
}

// StoreDocument 存储文档
func (e *MockVectorEngine) StoreDocument(ctx context.Context, document *VectorDocument) (string, error) {
	if !e.enabled {
		return "", fmt.Errorf("向量引擎未启用")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 生成向量（如果没有提供）
	if len(document.Vector) == 0 {
		document.Vector = e.generateMockVector(document.Content)
	}

	// 生成ID（如果没有提供）
	if document.ID == "" {
		document.ID = fmt.Sprintf("doc_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
	}

	// 添加时间戳
	if document.Metadata == nil {
		document.Metadata = make(map[string]interface{})
	}
	document.Metadata["stored_at"] = time.Now().Unix()
	document.Metadata["engine"] = "mock_vector"

	// 存储文档
	e.documents[document.ID] = document

	log.Printf("✅ 向量文档存储成功 - ID: %s, 内容长度: %d",
		document.ID, len(document.Content))

	return document.ID, nil
}

// Search 向量搜索
func (e *MockVectorEngine) Search(ctx context.Context, query *VectorQuery) (*VectorResult, error) {
	if !e.enabled {
		return nil, fmt.Errorf("向量引擎未启用")
	}

	startTime := time.Now()

	e.mu.RLock()
	defer e.mu.RUnlock()

	log.Printf("🔍 开始向量搜索 - 查询: %s, TopK: %d",
		query.QueryText[:min(50, len(query.QueryText))], query.TopK)

	// 生成查询向量（如果没有提供）
	queryVector := query.QueryVector
	if len(queryVector) == 0 {
		queryVector = e.generateMockVector(query.QueryText)
	}

	// 计算相似度并排序
	var candidates []VectorDocument

	for _, doc := range e.documents {
		// 应用过滤器
		if !e.matchesFilters(doc, query.Filters) {
			continue
		}

		// 计算相似度
		similarity := e.calculateSimilarity(queryVector, doc.Vector)

		// 检查最小分数阈值
		if similarity < query.MinScore {
			continue
		}

		// 创建结果文档
		resultDoc := *doc
		resultDoc.Score = similarity
		candidates = append(candidates, resultDoc)
	}

	// 按相似度排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// 限制返回数量
	topK := query.TopK
	if topK <= 0 || topK > e.config.MaxResults {
		topK = e.config.MaxResults
	}

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	queryTime := time.Since(startTime)

	log.Printf("✅ 向量搜索完成 - 找到: %d个结果, 耗时: %v",
		len(candidates), queryTime)

	return &VectorResult{
		Documents: candidates,
		Total:     len(candidates),
		QueryTime: queryTime,
	}, nil
}

// IsEnabled 检查是否启用
func (e *MockVectorEngine) IsEnabled() bool {
	return e.enabled
}

// Close 关闭引擎
func (e *MockVectorEngine) Close() error {
	e.enabled = false
	log.Printf("🔒 模拟向量引擎关闭")
	return nil
}

// initMockData 初始化模拟数据
func (e *MockVectorEngine) initMockData() {
	mockDocs := []struct {
		content  string
		metadata map[string]interface{}
	}{
		{
			content: "Redis是一个开源的内存数据结构存储，用作数据库、缓存和消息代理",
			metadata: map[string]interface{}{
				"user_id":    "test_user_001",
				"session_id": "test_session_001",
				"topic":      "Redis",
				"type":       "技术概念",
			},
		},
		{
			content: "TimescaleDB是一个基于PostgreSQL的时间序列数据库，专为处理时间序列数据而优化",
			metadata: map[string]interface{}{
				"user_id":    "test_user_001",
				"session_id": "test_session_002",
				"topic":      "TimescaleDB",
				"type":       "数据库技术",
			},
		},
		{
			content: "Neo4j是一个图数据库管理系统，使用Cypher查询语言来操作图数据",
			metadata: map[string]interface{}{
				"user_id":    "test_user_001",
				"session_id": "test_session_003",
				"topic":      "Neo4j",
				"type":       "图数据库",
			},
		},
	}

	for i, doc := range mockDocs {
		document := &VectorDocument{
			ID:       fmt.Sprintf("mock_doc_%d", i+1),
			Content:  doc.content,
			Vector:   e.generateMockVector(doc.content),
			Metadata: doc.metadata,
		}

		e.documents[document.ID] = document
	}

	log.Printf("📊 初始化了 %d 个模拟向量文档", len(mockDocs))
}

// generateMockVector 生成模拟向量
func (e *MockVectorEngine) generateMockVector(content string) []float64 {
	// 基于内容生成确定性的模拟向量
	vector := make([]float64, e.config.Dimension)

	// 使用内容的哈希作为种子
	seed := int64(0)
	for _, char := range content {
		seed += int64(char)
	}

	rng := rand.New(rand.NewSource(seed))

	// 生成正态分布的向量
	for i := range vector {
		vector[i] = rng.NormFloat64()
	}

	// 归一化向量
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	for i := range vector {
		vector[i] /= norm
	}

	return vector
}

// calculateSimilarity 计算余弦相似度
func (e *MockVectorEngine) calculateSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := range vec1 {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0.0 || norm2 == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// matchesFilters 检查文档是否匹配过滤器
func (e *MockVectorEngine) matchesFilters(doc *VectorDocument, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, expectedValue := range filters {
		if actualValue, exists := doc.Metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
