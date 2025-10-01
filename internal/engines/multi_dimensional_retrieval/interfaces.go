package multi_dimensional_retrieval

import (
	"context"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// TimelineEngine 时间线检索引擎接口
type TimelineEngine interface {
	// 检索时间线事件
	RetrieveEvents(ctx context.Context, query *TimelineQuery) (*TimelineResult, error)

	// 获取时间聚合信息
	GetAggregation(ctx context.Context, query *TimelineQuery) (*TimelineAggregation, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 关闭连接
	Close() error
}

// KnowledgeEngine 知识图谱检索引擎接口
type KnowledgeEngine interface {
	// 扩展知识图谱
	ExpandGraph(ctx context.Context, query *KnowledgeQuery) (*KnowledgeResult, error)

	// 获取相关概念
	GetRelatedConcepts(ctx context.Context, concepts []string) ([]string, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 关闭连接
	Close() error
}

// VectorEngine 向量检索引擎接口
type VectorEngine interface {
	// 多维度向量检索
	SearchMultiDimensional(ctx context.Context, query *VectorQuery) (*VectorResult, error)

	// 传统向量检索（兼容现有逻辑）
	SearchLegacy(ctx context.Context, query *LegacyVectorQuery) (*VectorResult, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 关闭连接
	Close() error
}

// Cache 缓存接口
type Cache interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Delete(key string)
	Clear()
	Size() int
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Wait(ctx context.Context) error
	Allow() bool
}

// Metrics 性能指标接口
type Metrics interface {
	RecordQuery(duration time.Duration, resultCount int, engines []string)
	GetQueryStats() *QueryStats
	GetEngineStats() map[string]*EngineStats
	Reset()
}

// TimelineQuery 时间线查询
type TimelineQuery struct {
	UserID      string      `json:"user_id"`
	SessionID   string      `json:"session_id"`
	WorkspaceID string      `json:"workspace_id"`
	Keywords    []string    `json:"keywords"`
	TimeRanges  []TimeRange `json:"time_ranges"`
	EventTypes  []string    `json:"event_types"`
	Limit       int         `json:"limit"`
	Offset      int         `json:"offset"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Label     string    `json:"label"` // "recent", "yesterday", "last_week"
}

// TimelineResult 时间线检索结果
type TimelineResult struct {
	Events []models.TimelineEvent `json:"events"`
	Total  int                    `json:"total"`
}

// 注意：TimelineEvent现在使用internal/models/unified_models.go中的统一定义
// 这里不再重复定义，请导入并使用统一模型

// TimelineAggregation 时间线聚合结果
type TimelineAggregation struct {
	TimeBuckets []TimeBucket `json:"time_buckets"`
	Summary     *Summary     `json:"summary"`
}

// TimeBucket 时间桶
type TimeBucket struct {
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	EventCount int       `json:"event_count"`
	Keywords   []string  `json:"keywords"`
	EventTypes []string  `json:"event_types"`
}

// Summary 摘要信息
type Summary struct {
	TotalEvents   int      `json:"total_events"`
	TimeSpan      string   `json:"time_span"`
	TopKeywords   []string `json:"top_keywords"`
	TopEventTypes []string `json:"top_event_types"`
}

// KnowledgeQuery 知识图谱查询
type KnowledgeQuery struct {
	StartNodes    []string `json:"start_nodes"`
	MaxDepth      int      `json:"max_depth"`
	MaxNodes      int      `json:"max_nodes"`
	MinWeight     float64  `json:"min_weight"`
	RelationTypes []string `json:"relation_types"`
	NodeTypes     []string `json:"node_types"`
}

// KnowledgeResult 知识图谱检索结果
type KnowledgeResult struct {
	Nodes         []KnowledgeNode `json:"nodes"`
	Relationships []Relationship  `json:"relationships"`
	Paths         []Path          `json:"paths"`
	Total         int             `json:"total"`
}

// KnowledgeNode 知识节点
type KnowledgeNode struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Score      float64                `json:"score"`
	Depth      int                    `json:"depth"`
}

// Relationship 关系
type Relationship struct {
	ID         string                 `json:"id"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Type       string                 `json:"type"`
	Weight     float64                `json:"weight"`
	Properties map[string]interface{} `json:"properties"`
}

// Path 路径
type Path struct {
	Nodes         []string `json:"nodes"`
	Relationships []string `json:"relationships"`
	Weight        float64  `json:"weight"`
	Length        int      `json:"length"`
}

// VectorQuery 向量查询
type VectorQuery struct {
	QueryText     string                 `json:"query_text"`
	QueryVectors  map[string][]float32   `json:"query_vectors"` // 多维度向量
	Dimensions    []string               `json:"dimensions"`    // 启用的维度
	Weights       map[string]float64     `json:"weights"`       // 维度权重
	TopK          int                    `json:"top_k"`
	MinSimilarity float64                `json:"min_similarity"`
	Filters       map[string]interface{} `json:"filters"`
}

// LegacyVectorQuery 传统向量查询（兼容现有逻辑）
type LegacyVectorQuery struct {
	QueryText     string  `json:"query_text"`
	TopK          int     `json:"top_k"`
	MinSimilarity float64 `json:"min_similarity"`
	// 其他现有字段...
}

// VectorResult 向量检索结果
type VectorResult struct {
	Documents []VectorDocument `json:"documents"`
	Total     int              `json:"total"`
}

// VectorDocument 向量文档
type VectorDocument struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
	Similarity float64                `json:"similarity"`
	Dimensions map[string]float64     `json:"dimensions"` // 各维度得分
	Metadata   map[string]interface{} `json:"metadata"`
}

// QueryStats 查询统计
type QueryStats struct {
	TotalQueries   int64         `json:"total_queries"`
	AverageLatency time.Duration `json:"average_latency"`
	SuccessRate    float64       `json:"success_rate"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
	LastUpdated    time.Time     `json:"last_updated"`
}

// EngineStats 引擎统计
type EngineStats struct {
	Name            string        `json:"name"`
	QueriesHandled  int64         `json:"queries_handled"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	HealthStatus    string        `json:"health_status"`
	LastHealthCheck time.Time     `json:"last_health_check"`
}
