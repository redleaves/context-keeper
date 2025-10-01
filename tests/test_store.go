package main

import (
	"fmt"
	"log"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/google/uuid"
)

// Message 消息结构
type Message struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Role        string                 `json:"role"`
	Content     string                 `json:"content"`
	ContentType string                 `json:"content_type"`
	Timestamp   int64                  `json:"timestamp"`
	Priority    string                 `json:"priority"`
	Vector      []float32              `json:"vector,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func main() {
	// 由于导入路径问题，我们创建一个测试用脚本，可以单独编译运行

	// 4. 存储我们的多轮对话 - 模拟数据
	sessionID := "test-session-" + time.Now().Format("20060102-150405")

	// 使用统一的ID生成函数
	batchID := models.GenerateMemoryID("")
	fmt.Printf("使用sessionID: %s, batchID: %s\n", sessionID, batchID)

	// 构建我们的对话内容
	messages := []*Message{
		{
			ID:          uuid.New().String(),
			SessionID:   sessionID,
			Role:        "user",
			Content:     "请检查一下我们最近对向量存储ID的修改是否正确",
			ContentType: "text",
			Timestamp:   time.Now().Unix(),
			Priority:    "P2",
			Metadata: map[string]interface{}{
				"batchId": batchID,
				"type":    "conversation_message",
			},
		},
		{
			ID:          uuid.New().String(),
			SessionID:   sessionID,
			Role:        "assistant",
			Content:     "我已经检查了向量存储的修改，现在我们使用batchId作为向量存储ID而不是message.ID，这样可以更好地组织和检索相关内容。",
			ContentType: "text",
			Timestamp:   time.Now().Unix() + 1,
			Priority:    "P2",
			Metadata: map[string]interface{}{
				"batchId": batchID,
				"type":    "conversation_message",
			},
		},
		{
			ID:          uuid.New().String(),
			SessionID:   sessionID,
			Role:        "user",
			Content:     "还需要添加格式化的时间戳字段",
			ContentType: "text",
			Timestamp:   time.Now().Unix() + 2,
			Priority:    "P2",
			Metadata: map[string]interface{}{
				"batchId": batchID,
				"type":    "conversation_message",
			},
		},
		{
			ID:          uuid.New().String(),
			SessionID:   sessionID,
			Role:        "assistant",
			Content:     "我已经添加了formatted_time字段，它将时间戳格式化为'2006-01-02 15:04:05'格式，方便查看和检索。",
			ContentType: "text",
			Timestamp:   time.Now().Unix() + 3,
			Priority:    "P2",
			Metadata: map[string]interface{}{
				"batchId": batchID,
				"type":    "conversation_message",
			},
		},
	}

	// 打印出模拟的消息内容
	for i, msg := range messages {
		formattedTime := time.Unix(msg.Timestamp, 0).Format("2006-01-02 15:04:05")
		log.Printf("消息 #%d:\n ID=%s\n 角色=%s\n 内容=%s\n 时间戳=%s\n batchId=%s\n",
			i+1, msg.ID, msg.Role, msg.Content, formattedTime, msg.Metadata["batchId"])
	}

	fmt.Println("\n这是一个模拟测试，显示了如何使用batchId作为向量存储ID的实现。")
	fmt.Println("在实际应用中，这些消息会被存储到向量数据库，并可通过batchId进行检索。")
	fmt.Println("测试完成!")
}
