package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BatchEmbeddingService 批量embedding服务
// 基于阿里云text-embedding-async-v1模型实现异步批处理
type BatchEmbeddingService struct {
	APIEndpoint string // 批量embedding API端点
	APIKey      string // API密钥
	TaskQueue   *TaskQueue
	Worker      *AsyncWorker
	client      *http.Client
}

// BatchEmbeddingRequest 批量embedding请求
type BatchEmbeddingRequest struct {
	Model string `json:"model"` // text-embedding-async-v1
	Input struct {
		URL string `json:"url"` // 文件URL，指向包含文本的JSON文件
	} `json:"input"`
	Parameters struct {
		TextType string `json:"text_type,omitempty"` // query或document
	} `json:"parameters,omitempty"`
}

// BatchEmbeddingResponse 批量embedding响应
type BatchEmbeddingResponse struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// BatchTaskStatus 批量任务状态
type BatchTaskStatus struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"` // PENDING, RUNNING, COMPLETED, FAILED
		Result     struct {
			Embeddings []struct {
				TextIndex int       `json:"text_index"`
				Embedding []float32 `json:"embedding"`
			} `json:"embeddings"`
		} `json:"result,omitempty"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}

// BatchTask 批量任务
type BatchTask struct {
	TaskID     string                 `json:"task_id"`
	Texts      []string               `json:"texts"`
	SubmitTime time.Time              `json:"submit_time"`
	Status     string                 `json:"status"`      // PENDING, RUNNING, COMPLETED, FAILED
	RetryCount int                    `json:"retry_count"` // 重试次数
	Callback   func(TaskResult) error `json:"-"`           // 回调函数
	UserData   map[string]interface{} `json:"user_data"`   // 用户自定义数据
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Embeddings  [][]float32            `json:"embeddings,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Texts       []string               `json:"texts"`
	UserData    map[string]interface{} `json:"user_data,omitempty"`
	ProcessTime time.Duration          `json:"process_time"`
}

// TaskQueue 任务队列
type TaskQueue struct {
	tasks chan *BatchTask
	mutex sync.RWMutex
	size  int
}

// AsyncWorker 异步worker
type AsyncWorker struct {
	service      *BatchEmbeddingService
	pollInterval time.Duration
	maxRetries   int
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	runningMutex sync.RWMutex
}

// NewBatchEmbeddingService 创建新的批量embedding服务
func NewBatchEmbeddingService(apiEndpoint, apiKey string, queueSize int) *BatchEmbeddingService {
	service := &BatchEmbeddingService{
		APIEndpoint: apiEndpoint,
		APIKey:      apiKey,
		TaskQueue:   NewTaskQueue(queueSize),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 创建异步worker
	service.Worker = NewAsyncWorker(service, 5*time.Second, 3)

	log.Printf("[批量Embedding] 服务初始化完成, API端点: %s, 队列大小: %d", apiEndpoint, queueSize)
	return service
}

// NewTaskQueue 创建新的任务队列
func NewTaskQueue(size int) *TaskQueue {
	return &TaskQueue{
		tasks: make(chan *BatchTask, size),
		size:  size,
	}
}

// createTextFile 创建包含文本的临时JSON文件
func (s *BatchEmbeddingService) createTextFile(texts []string) (string, error) {
	// 根据阿里云文档，文件格式应该是JSON数组
	// 创建包含所有文本的JSON结构
	textData := map[string]interface{}{
		"inputs": texts,
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(textData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化文本数据失败: %w", err)
	}

	// 创建临时文件
	tempDir := "./data/temp"
	os.MkdirAll(tempDir, 0755)

	fileName := fmt.Sprintf("batch_texts_%d.json", time.Now().UnixNano())
	filePath := filepath.Join(tempDir, fileName)

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 返回文件的HTTP URL
	// 注意：实际项目中，这里应该返回一个公网可访问的URL
	// 这里我们先返回本地路径，需要配置文件服务器
	fileURL := fmt.Sprintf("http://localhost:8088/temp/%s", fileName)
	log.Printf("[批量Embedding] 创建文本文件: %s", fileURL)

	return fileURL, nil
}

// NewAsyncWorker 创建新的异步worker
func NewAsyncWorker(service *BatchEmbeddingService, pollInterval time.Duration, maxRetries int) *AsyncWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncWorker{
		service:      service,
		pollInterval: pollInterval,
		maxRetries:   maxRetries,
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
	}
}

// SubmitBatchTask 提交批量embedding任务
func (s *BatchEmbeddingService) SubmitBatchTask(fileURL string, callback func(TaskResult) error, userData map[string]interface{}) (string, error) {
	if fileURL == "" {
		return "", fmt.Errorf("文件URL不能为空")
	}

	// 简单验证URL格式
	if !strings.HasPrefix(fileURL, "http://") && !strings.HasPrefix(fileURL, "https://") {
		return "", fmt.Errorf("文件URL必须是有效的HTTP/HTTPS地址")
	}

	log.Printf("[批量Embedding] 提交批量任务, 文件URL: %s", fileURL)

	// 构建请求
	req := BatchEmbeddingRequest{
		Model: "text-embedding-async-v1",
	}
	req.Input.URL = fileURL
	req.Parameters.TextType = "query" // 使用query类型，与成功示例保持一致

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", s.APIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable") // 🔥 关键: 启用异步模式

	log.Printf("[批量Embedding] 发送批量embedding请求: %s", s.APIEndpoint)
	log.Printf("[批量Embedding] 请求头: Content-Type=%s, Authorization=Bearer %s..., X-DashScope-Async=%s",
		httpReq.Header.Get("Content-Type"),
		s.APIKey[:10]+"***",
		httpReq.Header.Get("X-DashScope-Async"))
	log.Printf("[批量Embedding] 请求体: %s", string(reqBody))

	// 发送请求
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[批量Embedding] API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var batchResp BatchEmbeddingResponse
	if err := json.Unmarshal(respBody, &batchResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	taskID := batchResp.Output.TaskID
	if taskID == "" {
		return "", fmt.Errorf("API未返回有效的task_id")
	}

	log.Printf("[批量Embedding] 任务提交成功, task_id: %s, 状态: %s", taskID, batchResp.Output.TaskStatus)

	// 创建批量任务对象
	task := &BatchTask{
		TaskID:     taskID,
		Texts:      []string{}, // 现在使用文件URL，不存储具体文本
		SubmitTime: time.Now(),
		Status:     batchResp.Output.TaskStatus,
		RetryCount: 0,
		Callback:   callback,
		UserData:   userData,
	}

	// 将文件URL存储在UserData中
	if task.UserData == nil {
		task.UserData = make(map[string]interface{})
	}
	task.UserData["source_file_url"] = fileURL

	// 将任务加入队列
	select {
	case s.TaskQueue.tasks <- task:
		log.Printf("[批量Embedding] 任务已加入队列: %s", taskID)
	default:
		return "", fmt.Errorf("任务队列已满，无法加入新任务")
	}

	return taskID, nil
}

// QueryTaskStatus 查询任务状态
func (s *BatchEmbeddingService) QueryTaskStatus(taskID string) (*BatchTaskStatus, error) {
	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/tasks/%s", taskID)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	log.Printf("[批量Embedding] 查询任务状态: %s", taskID)

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("[批量Embedding] 查询任务状态失败: %d, 响应: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("查询任务状态失败: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var status BatchTaskStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	log.Printf("[批量Embedding] 任务状态查询成功: %s, 状态: %s", taskID, status.Output.TaskStatus)

	return &status, nil
}

// StartWorker 启动异步worker
func (s *BatchEmbeddingService) StartWorker() error {
	return s.Worker.Start()
}

// StopWorker 停止异步worker
func (s *BatchEmbeddingService) StopWorker() error {
	return s.Worker.Stop()
}

// Start 启动异步worker
func (w *AsyncWorker) Start() error {
	w.runningMutex.Lock()
	defer w.runningMutex.Unlock()

	if w.running {
		return fmt.Errorf("worker已在运行中")
	}

	w.running = true
	log.Printf("[异步Worker] 启动worker, 轮询间隔: %v, 最大重试次数: %d", w.pollInterval, w.maxRetries)

	// 启动worker goroutine
	go w.run()

	return nil
}

// Stop 停止异步worker
func (w *AsyncWorker) Stop() error {
	w.runningMutex.Lock()
	defer w.runningMutex.Unlock()

	if !w.running {
		return nil
	}

	log.Printf("[异步Worker] 停止worker")
	w.cancel()
	w.running = false

	return nil
}

// run worker主循环
func (w *AsyncWorker) run() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	log.Printf("[异步Worker] Worker开始运行")

	for {
		select {
		case <-w.ctx.Done():
			log.Printf("[异步Worker] Worker收到停止信号，退出")
			return

		case <-ticker.C:
			// 检查队列中的任务
			w.processQueuedTasks()

		case task := <-w.service.TaskQueue.tasks:
			// 处理新加入的任务
			if task != nil {
				w.processTask(task)
			}
		}
	}
}

// processQueuedTasks 处理队列中的任务
func (w *AsyncWorker) processQueuedTasks() {
	// 非阻塞地检查队列
	select {
	case task := <-w.service.TaskQueue.tasks:
		if task != nil {
			w.processTask(task)
		}
	default:
		// 队列为空，继续轮询
	}
}

// processTask 处理单个任务
func (w *AsyncWorker) processTask(task *BatchTask) {
	log.Printf("[异步Worker] 开始处理任务: %s, 状态: %s", task.TaskID, task.Status)

	startTime := time.Now()

	// 查询任务状态
	status, err := w.service.QueryTaskStatus(task.TaskID)
	if err != nil {
		log.Printf("[异步Worker] 查询任务状态失败: %s, 错误: %v", task.TaskID, err)
		w.handleTaskError(task, err, startTime)
		return
	}

	// 更新任务状态
	task.Status = status.Output.TaskStatus

	switch status.Output.TaskStatus {
	case "COMPLETED":
		// 任务完成，处理结果
		w.handleTaskCompleted(task, status, startTime)

	case "FAILED":
		// 任务失败
		w.handleTaskFailed(task, "任务执行失败", startTime)

	case "PENDING", "RUNNING":
		// 任务仍在处理中，重新加入队列
		log.Printf("[异步Worker] 任务仍在处理中: %s, 状态: %s, 重新加入队列", task.TaskID, status.Output.TaskStatus)
		select {
		case w.service.TaskQueue.tasks <- task:
			// 成功重新加入队列
		default:
			log.Printf("[异步Worker] 队列已满，任务 %s 无法重新加入队列", task.TaskID)
		}

	default:
		log.Printf("[异步Worker] 未知任务状态: %s, 任务: %s", status.Output.TaskStatus, task.TaskID)
		w.handleTaskError(task, fmt.Errorf("未知任务状态: %s", status.Output.TaskStatus), startTime)
	}
}

// handleTaskCompleted 处理任务完成
func (w *AsyncWorker) handleTaskCompleted(task *BatchTask, status *BatchTaskStatus, startTime time.Time) {
	log.Printf("[异步Worker] 任务完成: %s, embedding数量: %d", task.TaskID, len(status.Output.Result.Embeddings))

	// 提取embedding结果
	embeddings := make([][]float32, len(task.Texts))
	for _, emb := range status.Output.Result.Embeddings {
		if emb.TextIndex >= 0 && emb.TextIndex < len(embeddings) {
			embeddings[emb.TextIndex] = emb.Embedding
		}
	}

	// 构建任务结果
	result := TaskResult{
		TaskID:      task.TaskID,
		Status:      "COMPLETED",
		Embeddings:  embeddings,
		Texts:       task.Texts,
		UserData:    task.UserData,
		ProcessTime: time.Since(startTime),
	}

	// 调用回调函数
	if task.Callback != nil {
		if err := task.Callback(result); err != nil {
			log.Printf("[异步Worker] 回调函数执行失败: %s, 错误: %v", task.TaskID, err)
		} else {
			log.Printf("[异步Worker] 任务回调执行成功: %s", task.TaskID)
		}
	}
}

// handleTaskFailed 处理任务失败
func (w *AsyncWorker) handleTaskFailed(task *BatchTask, errorMsg string, startTime time.Time) {
	log.Printf("[异步Worker] 任务失败: %s, 错误: %s", task.TaskID, errorMsg)

	// 构建任务结果
	result := TaskResult{
		TaskID:      task.TaskID,
		Status:      "FAILED",
		Error:       errorMsg,
		Texts:       task.Texts,
		UserData:    task.UserData,
		ProcessTime: time.Since(startTime),
	}

	// 调用回调函数
	if task.Callback != nil {
		if err := task.Callback(result); err != nil {
			log.Printf("[异步Worker] 失败任务回调执行失败: %s, 错误: %v", task.TaskID, err)
		}
	}
}

// handleTaskError 处理任务错误（重试逻辑）
func (w *AsyncWorker) handleTaskError(task *BatchTask, err error, startTime time.Time) {
	task.RetryCount++

	if task.RetryCount <= w.maxRetries {
		// 指数退避策略：2^retryCount 秒 (2, 4, 8, 16, 32...)
		retryDelay := time.Duration(1<<uint(task.RetryCount)) * time.Second
		log.Printf("[异步Worker] 任务错误，尝试重试: %s, 重试次数: %d/%d, 延迟: %v",
			task.TaskID, task.RetryCount, w.maxRetries, retryDelay)

		// 指数退避延迟后重新加入队列
		time.AfterFunc(retryDelay, func() {
			select {
			case w.service.TaskQueue.tasks <- task:
				log.Printf("[异步Worker] 任务重新加入队列成功: %s (延迟 %v)", task.TaskID, retryDelay)
			default:
				log.Printf("[异步Worker] 队列已满，任务 %s 重试失败", task.TaskID)
			}
		})
	} else {
		log.Printf("[异步Worker] 任务重试次数已达上限，标记为失败: %s", task.TaskID)
		w.handleTaskFailed(task, fmt.Sprintf("重试次数达到上限: %v", err), startTime)
	}
}

// GetQueueStatus 获取队列状态
func (s *BatchEmbeddingService) GetQueueStatus() map[string]interface{} {
	return map[string]interface{}{
		"queue_capacity": cap(s.TaskQueue.tasks),
		"queue_length":   len(s.TaskQueue.tasks),
		"worker_running": s.Worker.running,
	}
}
