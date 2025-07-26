package models

import "time"

// WebSocket回调结果
type CallbackResult struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// WebSocket消息类型
type WebSocketMessage struct {
	Type      string      `json:"type"`             // 消息类型：instruction, callback, heartbeat等
	Data      interface{} `json:"data"`             // 消息数据
	UserID    string      `json:"userId,omitempty"` // 用户ID
	Timestamp time.Time   `json:"timestamp"`        // 时间戳
}
