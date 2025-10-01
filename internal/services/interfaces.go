package services

import (
	"context"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// 临时接口定义，用于宽召回服务开发

// === LLM服务接口 ===

// LLMService LLM服务接口
type LLMService interface {
	// 宽召回服务使用的方法
	GenerateResponse(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// 统一上下文管理器使用的方法（兼容现有代码）
	AnalyzeUserIntent(userQuery string) (*models.IntentAnalysisResult, error)
	SynthesizeAndEvaluateContext(
		userQuery string,
		currentContext *models.UnifiedContextModel,
		retrievalResults *models.ParallelRetrievalResult,
		intentAnalysis *models.IntentAnalysisResult,
	) (*models.ContextSynthesisResult, error)
}

// GenerateRequest LLM生成请求
type GenerateRequest struct {
	Prompt      string  `json:"prompt"`      // 提示词
	MaxTokens   int     `json:"max_tokens"`  // 最大token数
	Temperature float64 `json:"temperature"` // 温度参数
	Format      string  `json:"format"`      // 输出格式
}

// GenerateResponse LLM生成响应
type GenerateResponse struct {
	Content string `json:"content"` // 生成内容
	Usage   Usage  `json:"usage"`   // 使用统计
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 提示词token数
	CompletionTokens int `json:"completion_tokens"` // 完成token数
	TotalTokens      int `json:"total_tokens"`      // 总token数
}

// === 存储服务接口 ===

// TimelineStore 时间线存储接口
type TimelineStore interface {
	SearchEvents(ctx context.Context, req *TimelineSearchRequest) ([]*TimelineSearchResult, error)
}

// TimelineSearchRequest 时间线搜索请求
type TimelineSearchRequest struct {
	UserID      string   `json:"user_id"`      // 用户ID
	WorkspaceID string   `json:"workspace_id"` // 工作空间ID
	Query       string   `json:"query"`        // 查询内容
	TimeRange   string   `json:"time_range"`   // 时间范围
	EventTypes  []string `json:"event_types"`  // 事件类型
	MaxResults  int      `json:"max_results"`  // 最大结果数
}

// TimelineSearchResult 时间线搜索结果
type TimelineSearchResult struct {
	EventID         string                 `json:"event_id"`         // 事件ID
	EventType       string                 `json:"event_type"`       // 事件类型
	Title           string                 `json:"title"`            // 标题
	Content         string                 `json:"content"`          // 内容
	Timestamp       time.Time              `json:"timestamp"`        // 时间戳
	Source          string                 `json:"source"`           // 来源
	ImportanceScore float64                `json:"importance_score"` // 重要性评分
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	Tags            []string               `json:"tags"`             // 标签
	Metadata        map[string]interface{} `json:"metadata"`         // 元数据
}

// KnowledgeStore 知识图谱存储接口
type KnowledgeStore interface {
	SearchConcepts(ctx context.Context, req *KnowledgeSearchRequest) ([]*KnowledgeSearchResult, error)
}

// KnowledgeSearchRequest 知识图谱搜索请求
type KnowledgeSearchRequest struct {
	UserID        string   `json:"user_id"`        // 用户ID
	WorkspaceID   string   `json:"workspace_id"`   // 工作空间ID
	Query         string   `json:"query"`          // 查询内容
	ConceptTypes  []string `json:"concept_types"`  // 概念类型
	RelationTypes []string `json:"relation_types"` // 关系类型
	MaxResults    int      `json:"max_results"`    // 最大结果数
}

// KnowledgeSearchResult 知识图谱搜索结果
type KnowledgeSearchResult struct {
	ConceptID       string                 `json:"concept_id"`       // 概念ID
	ConceptName     string                 `json:"concept_name"`     // 概念名称
	ConceptType     string                 `json:"concept_type"`     // 概念类型
	Description     string                 `json:"description"`      // 描述
	RelatedConcepts []RelatedConcept       `json:"related_concepts"` // 相关概念
	Properties      map[string]interface{} `json:"properties"`       // 属性
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	ConfidenceScore float64                `json:"confidence_score"` // 置信度评分
	Source          string                 `json:"source"`           // 来源
	LastUpdated     time.Time              `json:"last_updated"`     // 最后更新时间
}

// RelatedConcept 相关概念
type RelatedConcept struct {
	ConceptName    string  `json:"concept_name"`    // 概念名称
	RelationType   string  `json:"relation_type"`   // 关系类型
	RelationWeight float64 `json:"relation_weight"` // 关系权重
}

// VectorStore 向量存储接口
type VectorStore interface {
	SearchSimilar(ctx context.Context, req *VectorSearchRequest) ([]*VectorSearchResult, error)
}

// VectorSearchRequest 向量搜索请求
type VectorSearchRequest struct {
	UserID              string  `json:"user_id"`              // 用户ID
	WorkspaceID         string  `json:"workspace_id"`         // 工作空间ID
	Query               string  `json:"query"`                // 查询内容
	SimilarityThreshold float64 `json:"similarity_threshold"` // 相似度阈值
	MaxResults          int     `json:"max_results"`          // 最大结果数
}

// VectorSearchResult 向量搜索结果
type VectorSearchResult struct {
	DocumentID      string                 `json:"document_id"`      // 文档ID
	Content         string                 `json:"content"`          // 内容
	ContentType     string                 `json:"content_type"`     // 内容类型
	Source          string                 `json:"source"`           // 来源
	Similarity      float64                `json:"similarity"`       // 相似度
	RelevanceScore  float64                `json:"relevance_score"`  // 相关性评分
	Timestamp       time.Time              `json:"timestamp"`        // 时间戳
	Tags            []string               `json:"tags"`             // 标签
	Metadata        map[string]interface{} `json:"metadata"`         // 元数据
	MatchedSegments []MatchedSegment       `json:"matched_segments"` // 匹配片段
}

// MatchedSegment 匹配片段
type MatchedSegment struct {
	SegmentText string  `json:"segment_text"` // 片段文本
	StartPos    int     `json:"start_pos"`    // 开始位置
	EndPos      int     `json:"end_pos"`      // 结束位置
	Similarity  float64 `json:"similarity"`   // 相似度
}
