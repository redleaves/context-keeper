package models

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
