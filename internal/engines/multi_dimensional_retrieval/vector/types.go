package vector

import (
	"context"
	"time"
)

// VectorQuery 向量查询
type VectorQuery struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	QueryText   string                 `json:"query_text"`
	QueryVector []float64              `json:"query_vector"`
	TopK        int                    `json:"top_k"`
	MinScore    float64                `json:"min_score"`
	Filters     map[string]interface{} `json:"filters"`
	RequestID   string                 `json:"request_id"`
}

// VectorResult 向量查询结果
type VectorResult struct {
	Documents []VectorDocument `json:"documents"`
	Total     int              `json:"total"`
	QueryTime time.Duration    `json:"query_time"`
}

// VectorDocument 向量文档
type VectorDocument struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Vector   []float64              `json:"vector"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// VectorEngine 向量引擎接口
type VectorEngine interface {
	// Search 向量搜索
	Search(ctx context.Context, query *VectorQuery) (*VectorResult, error)

	// StoreDocument 存储文档
	StoreDocument(ctx context.Context, document *VectorDocument) (string, error)

	// IsEnabled 检查是否启用
	IsEnabled() bool

	// Close 关闭引擎
	Close() error
}

// VectorConfig 向量引擎配置
type VectorConfig struct {
	Provider   string        `json:"provider"`    // 向量数据库提供商
	Endpoint   string        `json:"endpoint"`    // 连接端点
	Database   string        `json:"database"`    // 数据库名
	Collection string        `json:"collection"`  // 集合名
	Dimension  int           `json:"dimension"`   // 向量维度
	IndexType  string        `json:"index_type"`  // 索引类型
	MetricType string        `json:"metric_type"` // 距离度量类型
	MaxResults int           `json:"max_results"` // 最大返回结果数
	Timeout    time.Duration `json:"timeout"`     // 超时时间
}
