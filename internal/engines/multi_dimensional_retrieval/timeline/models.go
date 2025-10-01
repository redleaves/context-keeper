package timeline

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// TimelineEvent 时间线事件 - 使用统一模型
type TimelineEvent = models.TimelineEvent

// Entity 实体信息
type Entity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

// EntityArray 实体数组，实现database/sql的Scanner和Valuer接口
type EntityArray []Entity

// Scan 实现Scanner接口，用于从数据库读取
func (ea *EntityArray) Scan(value interface{}) error {
	if value == nil {
		*ea = EntityArray{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, ea)
	case string:
		return json.Unmarshal([]byte(v), ea)
	default:
		return fmt.Errorf("无法将 %T 转换为 EntityArray", value)
	}
}

// Value 实现Valuer接口，用于写入数据库
func (ea EntityArray) Value() (driver.Value, error) {
	if len(ea) == 0 {
		return nil, nil
	}
	return json.Marshal(ea)
}

// TimelineQuery 时间线查询
type TimelineQuery struct {
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	WorkspaceID string `json:"workspace_id"`

	// 时间查询
	TimeRanges []TimeRange `json:"time_ranges"`
	TimeWindow string      `json:"time_window"` // "1 hour", "1 day", "1 week"
	// 🆕 直接时间范围（用于时间回忆查询）
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`

	// 内容查询
	Keywords   []string `json:"keywords"`
	EventTypes []string `json:"event_types"`
	Intent     string   `json:"intent"`
	Categories []string `json:"categories"`

	// 全文搜索
	SearchText string `json:"search_text"`

	// 分页和排序
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	OrderBy string `json:"order_by"` // "timestamp", "relevance_score", "importance_score"

	// 过滤条件
	MinImportance float64 `json:"min_importance"`
	MinRelevance  float64 `json:"min_relevance"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Label     string    `json:"label"` // "recent", "yesterday", "last_week"
}

// TimelineResult 时间线检索结果
type TimelineResult struct {
	Events []TimelineEvent `json:"events"`
	Total  int             `json:"total"`

	// 聚合信息
	Aggregation *TimelineAggregation `json:"aggregation,omitempty"`
}

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
	AvgScore   float64   `json:"avg_score"`
}

// Summary 摘要信息
type Summary struct {
	TotalEvents   int      `json:"total_events"`
	TimeSpan      string   `json:"time_span"`
	TopKeywords   []string `json:"top_keywords"`
	TopEventTypes []string `json:"top_event_types"`
	TopCategories []string `json:"top_categories"`
	AvgImportance float64  `json:"avg_importance"`
	AvgRelevance  float64  `json:"avg_relevance"`
}

// CreateTimelineEventRequest 创建时间线事件请求
type CreateTimelineEventRequest struct {
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	WorkspaceID string `json:"workspace_id"`

	EventType string `json:"event_type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Summary   string `json:"summary,omitempty"`

	RelatedFiles    []string `json:"related_files,omitempty"`
	RelatedConcepts []string `json:"related_concepts,omitempty"`
	ParentEventID   string   `json:"parent_event_id,omitempty"`

	// LLM分析结果
	Intent     string   `json:"intent,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Entities   []Entity `json:"entities,omitempty"`
	Categories []string `json:"categories,omitempty"`

	ImportanceScore float64 `json:"importance_score,omitempty"`
	RelevanceScore  float64 `json:"relevance_score,omitempty"`
}

// UpdateTimelineEventRequest 更新时间线事件请求
type UpdateTimelineEventRequest struct {
	ID      string `json:"id"`
	Summary string `json:"summary,omitempty"`

	RelatedFiles    []string `json:"related_files,omitempty"`
	RelatedConcepts []string `json:"related_concepts,omitempty"`

	Intent     string   `json:"intent,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Entities   []Entity `json:"entities,omitempty"`
	Categories []string `json:"categories,omitempty"`

	ImportanceScore float64 `json:"importance_score,omitempty"`
	RelevanceScore  float64 `json:"relevance_score,omitempty"`
}

// EventType 事件类型常量
const (
	EventTypeCodeEdit       = "code_edit"
	EventTypeDiscussion     = "discussion"
	EventTypeDesign         = "design"
	EventTypeProblemSolve   = "problem_solve"
	EventTypeKnowledgeShare = "knowledge_share"
	EventTypeDecision       = "decision"
	EventTypeReview         = "review"
	EventTypeTest           = "test"
	EventTypeDeployment     = "deployment"
	EventTypeMeeting        = "meeting"
)

// GetEventTypeDescription 获取事件类型描述
func GetEventTypeDescription(eventType string) string {
	descriptions := map[string]string{
		EventTypeCodeEdit:       "代码编辑",
		EventTypeDiscussion:     "问题讨论",
		EventTypeDesign:         "方案设计",
		EventTypeProblemSolve:   "问题解决",
		EventTypeKnowledgeShare: "知识分享",
		EventTypeDecision:       "决策制定",
		EventTypeReview:         "代码审查",
		EventTypeTest:           "测试验证",
		EventTypeDeployment:     "部署发布",
		EventTypeMeeting:        "会议记录",
	}

	if desc, exists := descriptions[eventType]; exists {
		return desc
	}
	return "未知类型"
}

// 注意：Validate方法现在在unified_models.go中定义，这里不再重复定义

// Validate 验证查询参数
func (query *TimelineQuery) Validate() error {
	if query.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if query.Limit <= 0 {
		query.Limit = 50 // 默认限制
	}
	if query.Limit > 1000 {
		query.Limit = 1000 // 最大限制
	}
	if query.OrderBy == "" {
		query.OrderBy = "timestamp" // 默认按时间排序
	}
	return nil
}
