package models

import "time"

// ... existing code ...

type SearchContextRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Query     string `json:"query" binding:"required"`
}

type AssociateFileRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	FilePath  string `json:"filePath" binding:"required"`
}

type RecordEditRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	FilePath  string `json:"filePath" binding:"required"`
	Diff      string `json:"diff" binding:"required"`
}

// TimelineSearchRequest 时间线搜索请求
type TimelineSearchRequest struct {
	Query       string   `json:"query"`        // 搜索查询
	Limit       int      `json:"limit"`        // 结果限制
	KeyConcepts []string `json:"key_concepts"` // LLM分析的关键概念
	UserID      string   `json:"user_id"`      // 用户ID
	WorkspaceID string   `json:"workspace_id"` // 工作空间ID
	// 🆕 时间范围查询字段
	StartTime *time.Time `json:"start_time,omitempty"` // 开始时间
	EndTime   *time.Time `json:"end_time,omitempty"`   // 结束时间
}
