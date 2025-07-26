package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/contextkeeper/service/pkg/aliyun"
	"github.com/gin-gonic/gin"
)

// BatchEmbeddingHandler 批量embedding处理器
type BatchEmbeddingHandler struct {
	batchService *aliyun.BatchEmbeddingService
}

// NewBatchEmbeddingHandler 创建新的批量embedding处理器
func NewBatchEmbeddingHandler(batchService *aliyun.BatchEmbeddingService) *BatchEmbeddingHandler {
	return &BatchEmbeddingHandler{
		batchService: batchService,
	}
}

// SubmitBatchEmbeddingRequest 提交批量embedding请求结构
type SubmitBatchEmbeddingRequest struct {
	FileURL     string                 `json:"file_url" binding:"required"` // 文件URL（指向包含文本的JSON文件）
	UserData    map[string]interface{} `json:"user_data,omitempty"`         // 用户自定义数据
	CallbackURL string                 `json:"callback_url,omitempty"`      // 回调URL（TODO功能）
	TextType    string                 `json:"text_type,omitempty"`         // 文本类型：query或document
}

// SubmitBatchEmbeddingResponse 提交批量embedding响应结构
type SubmitBatchEmbeddingResponse struct {
	Status   string `json:"status"`    // success或error
	TaskID   string `json:"task_id"`   // 任务ID
	Message  string `json:"message"`   // 响应消息
	QueuedAt int64  `json:"queued_at"` // 入队时间戳
}

// QueryTaskStatusRequest 查询任务状态请求结构
type QueryTaskStatusRequest struct {
	TaskID string `json:"task_id" binding:"required"` // 任务ID
}

// QueryTaskStatusResponse 查询任务状态响应结构
type QueryTaskStatusResponse struct {
	Status      string                 `json:"status"`                 // success或error
	TaskID      string                 `json:"task_id"`                // 任务ID
	TaskStatus  string                 `json:"task_status"`            // PENDING, RUNNING, COMPLETED, FAILED
	Message     string                 `json:"message"`                // 响应消息
	Embeddings  [][]float32            `json:"embeddings,omitempty"`   // embedding结果（仅当COMPLETED时）
	Texts       []string               `json:"texts,omitempty"`        // 原始文本（仅当COMPLETED时）
	UserData    map[string]interface{} `json:"user_data,omitempty"`    // 用户数据
	ProcessTime string                 `json:"process_time,omitempty"` // 处理时间
	Error       string                 `json:"error,omitempty"`        // 错误信息（仅当FAILED时）
}

// QueueStatusResponse 队列状态响应结构
type QueueStatusResponse struct {
	Status        string `json:"status"`         // success
	QueueCapacity int    `json:"queue_capacity"` // 队列容量
	QueueLength   int    `json:"queue_length"`   // 当前队列长度
	WorkerRunning bool   `json:"worker_running"` // worker是否运行中
	Available     bool   `json:"available"`      // 服务是否可用
}

// RegisterBatchEmbeddingRoutes 注册批量embedding路由
func (h *BatchEmbeddingHandler) RegisterBatchEmbeddingRoutes(router *gin.Engine) {
	// 批量embedding接口组
	batch := router.Group("/api/batch-embedding")
	{
		// 提交批量embedding任务
		batch.POST("/submit", h.HandleSubmitBatchEmbedding)

		// 查询任务状态
		batch.POST("/status", h.HandleQueryTaskStatus)
		batch.GET("/status/:task_id", h.HandleQueryTaskStatusGET)

		// 获取队列状态
		batch.GET("/queue-status", h.HandleGetQueueStatus)

		// 健康检查
		batch.GET("/health", h.HandleBatchEmbeddingHealth)
	}

	log.Println("批量Embedding路由已注册:")
	log.Println("  POST /api/batch-embedding/submit - 提交批量embedding任务")
	log.Println("  POST /api/batch-embedding/status - 查询任务状态")
	log.Println("  GET  /api/batch-embedding/status/:task_id - 查询任务状态(GET方式)")
	log.Println("  GET  /api/batch-embedding/queue-status - 获取队列状态")
	log.Println("  GET  /api/batch-embedding/health - 批量embedding健康检查")
}

// HandleSubmitBatchEmbedding 处理提交批量embedding任务请求
func (h *BatchEmbeddingHandler) HandleSubmitBatchEmbedding(c *gin.Context) {
	log.Printf("[批量Embedding API] ===== 开始处理批量embedding提交请求 =====")

	var req SubmitBatchEmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[批量Embedding API] ❌ 请求参数错误: %v", err)
		c.JSON(http.StatusBadRequest, SubmitBatchEmbeddingResponse{
			Status:  "error",
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证文件URL
	if req.FileURL == "" {
		log.Printf("[批量Embedding API] ❌ 文件URL为空")
		c.JSON(http.StatusBadRequest, SubmitBatchEmbeddingResponse{
			Status:  "error",
			Message: "文件URL不能为空",
		})
		return
	}

	// 简单验证URL格式
	if !strings.HasPrefix(req.FileURL, "http://") && !strings.HasPrefix(req.FileURL, "https://") {
		log.Printf("[批量Embedding API] ❌ 文件URL格式无效: %s", req.FileURL)
		c.JSON(http.StatusBadRequest, SubmitBatchEmbeddingResponse{
			Status:  "error",
			Message: "文件URL必须是有效的HTTP/HTTPS地址",
		})
		return
	}

	log.Printf("[批量Embedding API] 📋 请求验证通过，文件URL: %s", req.FileURL)

	// 检查批量服务是否可用
	if h.batchService == nil {
		log.Printf("[批量Embedding API] ❌ 批量embedding服务未初始化")
		c.JSON(http.StatusServiceUnavailable, SubmitBatchEmbeddingResponse{
			Status:  "error",
			Message: "批量embedding服务不可用",
		})
		return
	}

	// 设置默认用户数据
	if req.UserData == nil {
		req.UserData = make(map[string]interface{})
	}
	req.UserData["submit_time"] = time.Now().Unix()
	req.UserData["file_url"] = req.FileURL
	req.UserData["client_ip"] = c.ClientIP()
	if req.TextType != "" {
		req.UserData["text_type"] = req.TextType
	}

	// 创建回调函数
	callback := func(result aliyun.TaskResult) error {
		log.Printf("[批量Embedding回调] 任务完成回调: %s, 状态: %s", result.TaskID, result.Status)
		if result.Status == "COMPLETED" {
			log.Printf("[批量Embedding回调] ✅ 任务成功完成，embedding数量: %d, 耗时: %v",
				len(result.Embeddings), result.ProcessTime)
		} else {
			log.Printf("[批量Embedding回调] ❌ 任务失败: %s", result.Error)
		}

		// TODO: 在这里可以添加更多的业务逻辑
		// 比如存储结果到数据库、发送通知等

		return nil
	}

	// 提交批量任务
	taskID, err := h.batchService.SubmitBatchTask(req.FileURL, callback, req.UserData)
	if err != nil {
		log.Printf("[批量Embedding API] ❌ 提交批量任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, SubmitBatchEmbeddingResponse{
			Status:  "error",
			Message: "提交批量任务失败: " + err.Error(),
		})
		return
	}

	log.Printf("[批量Embedding API] ✅ 批量任务提交成功: %s", taskID)

	c.JSON(http.StatusOK, SubmitBatchEmbeddingResponse{
		Status:   "success",
		TaskID:   taskID,
		Message:  "批量embedding任务提交成功",
		QueuedAt: time.Now().Unix(),
	})

	log.Printf("[批量Embedding API] ===== 批量embedding提交请求处理完成 =====")
}

// HandleQueryTaskStatus 处理查询任务状态请求（POST方式）
func (h *BatchEmbeddingHandler) HandleQueryTaskStatus(c *gin.Context) {
	log.Printf("[批量Embedding API] ===== 开始处理任务状态查询请求 =====")

	var req QueryTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[批量Embedding API] ❌ 请求参数错误: %v", err)
		c.JSON(http.StatusBadRequest, QueryTaskStatusResponse{
			Status:  "error",
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	h.queryTaskStatusInternal(c, req.TaskID)
}

// HandleQueryTaskStatusGET 处理查询任务状态请求（GET方式）
func (h *BatchEmbeddingHandler) HandleQueryTaskStatusGET(c *gin.Context) {
	log.Printf("[批量Embedding API] ===== 开始处理任务状态查询请求(GET) =====")

	taskID := c.Param("task_id")
	if taskID == "" {
		log.Printf("[批量Embedding API] ❌ 缺少task_id参数")
		c.JSON(http.StatusBadRequest, QueryTaskStatusResponse{
			Status:  "error",
			Message: "缺少task_id参数",
		})
		return
	}

	h.queryTaskStatusInternal(c, taskID)
}

// queryTaskStatusInternal 查询任务状态的内部实现
func (h *BatchEmbeddingHandler) queryTaskStatusInternal(c *gin.Context, taskID string) {
	log.Printf("[批量Embedding API] 📋 查询任务状态: %s", taskID)

	// 检查批量服务是否可用
	if h.batchService == nil {
		log.Printf("[批量Embedding API] ❌ 批量embedding服务未初始化")
		c.JSON(http.StatusServiceUnavailable, QueryTaskStatusResponse{
			Status:  "error",
			Message: "批量embedding服务不可用",
		})
		return
	}

	// 查询任务状态
	status, err := h.batchService.QueryTaskStatus(taskID)
	if err != nil {
		log.Printf("[批量Embedding API] ❌ 查询任务状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, QueryTaskStatusResponse{
			Status:  "error",
			TaskID:  taskID,
			Message: "查询任务状态失败: " + err.Error(),
		})
		return
	}

	log.Printf("[批量Embedding API] ✅ 任务状态查询成功: %s, 状态: %s", taskID, status.Output.TaskStatus)

	// 构建响应
	response := QueryTaskStatusResponse{
		Status:     "success",
		TaskID:     taskID,
		TaskStatus: status.Output.TaskStatus,
		Message:    "任务状态查询成功",
	}

	// 如果任务完成，添加embedding结果 (阿里云返回SUCCEEDED状态)
	if (status.Output.TaskStatus == "COMPLETED" || status.Output.TaskStatus == "SUCCEEDED") && len(status.Output.Result.Embeddings) > 0 {
		embeddings := make([][]float32, len(status.Output.Result.Embeddings))
		texts := make([]string, len(status.Output.Result.Embeddings))

		for _, emb := range status.Output.Result.Embeddings {
			if emb.TextIndex >= 0 && emb.TextIndex < len(embeddings) {
				embeddings[emb.TextIndex] = emb.Embedding
				// 注意：这里无法直接获取原始文本，需要从其他地方获取
				texts[emb.TextIndex] = fmt.Sprintf("text_%d", emb.TextIndex)
			}
		}

		response.Embeddings = embeddings
		response.Texts = texts
		response.ProcessTime = "已完成"

		log.Printf("[批量Embedding API] 📊 任务已完成，embedding数量: %d", len(embeddings))
	} else if status.Output.TaskStatus == "FAILED" {
		response.Error = "任务执行失败"
		log.Printf("[批量Embedding API] ❌ 任务执行失败: %s", taskID)
	}

	c.JSON(http.StatusOK, response)

	log.Printf("[批量Embedding API] ===== 任务状态查询请求处理完成 =====")
}

// HandleGetQueueStatus 处理获取队列状态请求
func (h *BatchEmbeddingHandler) HandleGetQueueStatus(c *gin.Context) {
	log.Printf("[批量Embedding API] 查询队列状态")

	// 检查批量服务是否可用
	if h.batchService == nil {
		c.JSON(http.StatusServiceUnavailable, QueueStatusResponse{
			Status:    "error",
			Available: false,
		})
		return
	}

	// 获取队列状态
	queueStatus := h.batchService.GetQueueStatus()

	response := QueueStatusResponse{
		Status:        "success",
		QueueCapacity: queueStatus["queue_capacity"].(int),
		QueueLength:   queueStatus["queue_length"].(int),
		WorkerRunning: queueStatus["worker_running"].(bool),
		Available:     true,
	}

	log.Printf("[批量Embedding API] 队列状态 - 容量: %d, 当前长度: %d, Worker运行: %t",
		response.QueueCapacity, response.QueueLength, response.WorkerRunning)

	c.JSON(http.StatusOK, response)
}

// HandleBatchEmbeddingHealth 处理批量embedding健康检查
func (h *BatchEmbeddingHandler) HandleBatchEmbeddingHealth(c *gin.Context) {
	if h.batchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "batch-embedding",
			"message": "批量embedding服务未初始化",
		})
		return
	}

	queueStatus := h.batchService.GetQueueStatus()

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"service":        "batch-embedding",
		"queue_capacity": queueStatus["queue_capacity"],
		"queue_length":   queueStatus["queue_length"],
		"worker_running": queueStatus["worker_running"],
		"api_endpoint":   "阿里云批量embedding API",
		"model":          "text-embedding-async-v1",
	})
}
