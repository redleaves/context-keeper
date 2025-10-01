package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// =============================================================================
// 统一时间线模型 - 单一数据源
// =============================================================================

// TimelineEvent 统一时间线事件模型
// 这是系统中唯一的TimelineEvent定义，所有其他包都应该使用这个定义
type TimelineEvent struct {
	// === 核心标识 ===
	ID          string `json:"id" db:"id"`                     // 事件唯一标识
	UserID      string `json:"user_id" db:"user_id"`           // 用户ID
	SessionID   string `json:"session_id" db:"session_id"`     // 会话ID
	WorkspaceID string `json:"workspace_id" db:"workspace_id"` // 工作空间ID

	// === 时间维度 ===
	Timestamp     time.Time      `json:"timestamp" db:"timestamp"`                     // 事件时间戳
	EventDuration *time.Duration `json:"event_duration,omitempty" db:"event_duration"` // 事件持续时间
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`                   // 创建时间
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`                   // 更新时间

	// === 事件内容 ===
	EventType string  `json:"event_type" db:"event_type"`     // 事件类型：commit, edit, build, deploy, discussion, decision, incident, run
	Title     string  `json:"title" db:"title"`               // 事件标题
	Content   string  `json:"content" db:"content"`           // 事件内容
	Summary   *string `json:"summary,omitempty" db:"summary"` // 事件摘要（可为空）

	// === 关联信息 ===
	RelatedFiles    pq.StringArray `json:"related_files,omitempty" db:"related_files"`       // 相关文件
	RelatedConcepts pq.StringArray `json:"related_concepts,omitempty" db:"related_concepts"` // 相关概念
	ParentEventID   *string        `json:"parent_event_id,omitempty" db:"parent_event_id"`   // 父事件ID

	// === LLM分析结果 ===
	Intent     *string        `json:"intent,omitempty" db:"intent"`         // 意图分析
	Keywords   pq.StringArray `json:"keywords,omitempty" db:"keywords"`     // 关键词
	Entities   EntityArray    `json:"entities,omitempty" db:"entities"`     // 实体信息
	Categories pq.StringArray `json:"categories,omitempty" db:"categories"` // 分类标签

	// === 质量指标 ===
	ImportanceScore float64 `json:"importance_score" db:"importance_score"` // 重要性评分 0-1
	RelevanceScore  float64 `json:"relevance_score" db:"relevance_score"`   // 相关性评分 0-1
}

// Entity 实体信息
type Entity struct {
	Text       string  `json:"text"`                // 实体文本（保持与现有代码兼容）
	Type       string  `json:"type"`                // 实体类型：person, organization, technology, concept
	Confidence float64 `json:"confidence"`          // 置信度 0-1
	StartPos   int     `json:"start_pos,omitempty"` // 在文本中的起始位置
	EndPos     int     `json:"end_pos,omitempty"`   // 在文本中的结束位置
}

// EntityArray 实体数组，实现数据库序列化
type EntityArray []Entity

// Value 实现 driver.Valuer 接口
func (ea EntityArray) Value() (driver.Value, error) {
	if ea == nil {
		return nil, nil
	}
	return json.Marshal(ea)
}

// Scan 实现 sql.Scanner 接口
func (ea *EntityArray) Scan(value interface{}) error {
	if value == nil {
		*ea = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("无法将 %T 转换为 EntityArray", value)
	}

	return json.Unmarshal(bytes, ea)
}

// =============================================================================
// 统一知识图谱模型 - 单一数据源
// =============================================================================

// KnowledgeNode 统一知识图谱节点模型
// 这是系统中唯一的KnowledgeNode定义，所有其他包都应该使用这个定义
type KnowledgeNode struct {
	// === 核心标识 ===
	ID     string   `json:"id" db:"id"`         // 节点唯一标识
	Labels []string `json:"labels" db:"labels"` // 节点标签
	Name   string   `json:"name" db:"name"`     // 节点名称
	Type   string   `json:"type" db:"type"`     // 节点类型：concept, technology, pattern, solution, problem, file, function, class

	// === 内容信息 ===
	Description string                 `json:"description,omitempty" db:"description"` // 节点描述
	Content     string                 `json:"content,omitempty" db:"content"`         // 节点内容
	Category    string                 `json:"category,omitempty" db:"category"`       // 节点分类
	Keywords    pq.StringArray         `json:"keywords,omitempty" db:"keywords"`       // 关键词
	Properties  map[string]interface{} `json:"properties,omitempty" db:"properties"`   // 扩展属性

	// === 质量指标 ===
	Score        float64 `json:"score,omitempty" db:"score"`               // 相关性评分 0-1
	Importance   float64 `json:"importance,omitempty" db:"importance"`     // 重要性评分 0-1
	Confidence   float64 `json:"confidence,omitempty" db:"confidence"`     // 置信度 0-1
	Completeness float64 `json:"completeness,omitempty" db:"completeness"` // 完整性评分 0-1
	Freshness    float64 `json:"freshness,omitempty" db:"freshness"`       // 新鲜度评分 0-1

	// === 关联信息 ===
	UserID      string `json:"user_id,omitempty" db:"user_id"`           // 创建用户ID
	WorkspaceID string `json:"workspace_id,omitempty" db:"workspace_id"` // 工作空间ID

	// === 时间戳 ===
	CreatedAt time.Time `json:"created_at" db:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // 更新时间
}

// KnowledgeEdge 统一知识图谱边模型
type KnowledgeEdge struct {
	// === 核心标识 ===
	ID       string `json:"id" db:"id"`               // 边唯一标识
	SourceID string `json:"source_id" db:"source_id"` // 源节点ID
	TargetID string `json:"target_id" db:"target_id"` // 目标节点ID

	// === 关系信息 ===
	Type        string                 `json:"type" db:"type"`                         // 关系类型：depends_on, related_to, implements, references, contains
	Weight      float64                `json:"weight,omitempty" db:"weight"`           // 关系权重 0-1
	Strength    float64                `json:"strength,omitempty" db:"strength"`       // 关系强度 0-1
	Description string                 `json:"description,omitempty" db:"description"` // 关系描述
	Properties  map[string]interface{} `json:"properties,omitempty" db:"properties"`   // 扩展属性

	// === 关联信息 ===
	UserID      string `json:"user_id,omitempty" db:"user_id"`           // 创建用户ID
	WorkspaceID string `json:"workspace_id,omitempty" db:"workspace_id"` // 工作空间ID

	// === 时间戳 ===
	CreatedAt time.Time `json:"created_at" db:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // 更新时间
}

// =============================================================================
// 统一向量匹配模型 - 单一数据源
// =============================================================================

// VectorMatch 统一向量匹配结果模型
type VectorMatch struct {
	// === 核心标识 ===
	ID       string  `json:"id"`                 // 匹配项ID
	Score    float64 `json:"score"`              // 相似度评分 0-1
	Distance float64 `json:"distance,omitempty"` // 向量距离

	// === 内容信息 ===
	Title    string                 `json:"title,omitempty"`    // 标题
	Content  string                 `json:"content,omitempty"`  // 内容
	Summary  string                 `json:"summary,omitempty"`  // 摘要
	Keywords []string               `json:"keywords,omitempty"` // 关键词
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 元数据

	// === 关联信息 ===
	UserID      string `json:"user_id,omitempty"`      // 用户ID
	WorkspaceID string `json:"workspace_id,omitempty"` // 工作空间ID
	SourceType  string `json:"source_type,omitempty"`  // 来源类型：timeline, knowledge, vector

	// === 时间戳 ===
	Timestamp time.Time `json:"timestamp,omitempty"` // 时间戳
}

// =============================================================================
// 模型验证方法
// =============================================================================

// Validate 验证时间线事件
func (event *TimelineEvent) Validate() error {
	if event.ID == "" {
		return fmt.Errorf("事件ID不能为空")
	}
	if event.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if event.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	if event.WorkspaceID == "" {
		return fmt.Errorf("工作空间ID不能为空")
	}
	if event.EventType == "" {
		return fmt.Errorf("事件类型不能为空")
	}
	if event.Title == "" {
		return fmt.Errorf("标题不能为空")
	}
	if event.Content == "" {
		return fmt.Errorf("内容不能为空")
	}
	if event.ImportanceScore < 0 || event.ImportanceScore > 1 {
		return fmt.Errorf("重要性评分必须在0-1之间")
	}
	if event.RelevanceScore < 0 || event.RelevanceScore > 1 {
		return fmt.Errorf("相关性评分必须在0-1之间")
	}
	return nil
}

// Validate 验证知识节点
func (node *KnowledgeNode) Validate() error {
	if node.ID == "" {
		return fmt.Errorf("节点ID不能为空")
	}
	if node.Name == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if node.Type == "" {
		return fmt.Errorf("节点类型不能为空")
	}
	if node.Score < 0 || node.Score > 1 {
		return fmt.Errorf("相关性评分必须在0-1之间")
	}
	if node.Importance < 0 || node.Importance > 1 {
		return fmt.Errorf("重要性评分必须在0-1之间")
	}
	if node.Confidence < 0 || node.Confidence > 1 {
		return fmt.Errorf("置信度必须在0-1之间")
	}
	return nil
}

// Validate 验证知识边
func (edge *KnowledgeEdge) Validate() error {
	if edge.ID == "" {
		return fmt.Errorf("边ID不能为空")
	}
	if edge.SourceID == "" {
		return fmt.Errorf("源节点ID不能为空")
	}
	if edge.TargetID == "" {
		return fmt.Errorf("目标节点ID不能为空")
	}
	if edge.Type == "" {
		return fmt.Errorf("关系类型不能为空")
	}
	if edge.Weight < 0 || edge.Weight > 1 {
		return fmt.Errorf("关系权重必须在0-1之间")
	}
	if edge.Strength < 0 || edge.Strength > 1 {
		return fmt.Errorf("关系强度必须在0-1之间")
	}
	return nil
}
