package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/gin-gonic/gin"
)

// SSE连接统计
var (
	connectionCounter uint64       // 连接总计数器
	activeConnections int          // 当前活跃连接数
	totalConnections  int          // 历史连接总数
	connMutex         sync.RWMutex // 连接统计锁
)

// HandleSSE 处理SSE连接请求
func (h *Handler) HandleSSE(c *gin.Context) {
	// 检查Accept头，确保客户端期望接收事件流
	acceptHeader := c.GetHeader("Accept")
	if acceptHeader != "" && acceptHeader != "text/event-stream" {
		c.String(http.StatusNotAcceptable, "必须接受text/event-stream内容类型")
		return
	}

	// 记录所有请求头，帮助调试
	log.Printf("SSE连接请求头:")
	for key, values := range c.Request.Header {
		log.Printf("  %s: %s", key, strings.Join(values, ", "))
	}

	// 设置SSE相关的HTTP头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no") // 防止Nginx缓冲

	// 获取客户端信息，用于日志和调试
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	lastEventID := c.Request.Header.Get("Last-Event-ID")

	// 分配连接ID并更新连接计数
	connID := atomic.AddUint64(&connectionCounter, 1)

	connMutex.Lock()
	activeConnections++
	totalConnections++
	connMutex.Unlock()

	// 记录连接建立
	log.Printf("[conn-%d] 新SSE连接已建立: IP=%s, UA=%s, LastEventID=%s",
		connID, clientIP, userAgent, lastEventID)

	// 确保连接关闭时清理资源
	defer func() {
		connMutex.Lock()
		activeConnections--
		connMutex.Unlock()
		log.Printf("[conn-%d] 连接已关闭", connID)
	}()

	// 创建上下文，允许主循环在连接关闭时停止
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 写入连接成功事件 - 这是SSE特定的事件，不是JSON-RPC消息
	err := writeSSE(c.Writer, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	if err != nil {
		log.Printf("[conn-%d] 发送连接成功事件失败: %v", connID, err)
		return
	}

	// 创建请求通道，用于接收RPC请求
	requestChan := make(chan map[string]interface{}, 10)

	// 注册请求通道，以便RPC请求能转发到这个SSE连接
	log.Printf("[conn-%d] 注册SSE请求通道...", connID)
	RegisterSSERequestChannel(connID, requestChan)
	defer UnregisterSSERequestChannel(connID)

	// 创建消息通道，用于在主循环之外发送消息
	messageChan := make(chan string, 10)

	// 启动HTTP转发器，监听RPC请求并转发到requestChan
	go func() {
		// 等待RPC请求
		log.Printf("[conn-%d] 等待RPC请求转发...", connID)
	}()

	// 注册SSE连接
	log.Printf("[conn-%d] 已注册SSE连接", connID)
	log.Printf("已注册SSE连接: %d", connID)

	// 设置心跳间隔
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// 标记是否已初始化并发送manifest
	initialized := false

	// 主循环处理
	for {
		select {
		case <-ctx.Done():
			// 连接被取消
			log.Printf("[conn-%d] 连接被取消", connID)
			return
		case request := <-requestChan:
			// 处理来自RPC端点的请求
			method, ok := request["method"].(string)
			if !ok {
				log.Printf("[conn-%d] 无效的RPC请求: 缺少method字段", connID)
				continue
			}

			// 获取请求ID，用于响应
			id, _ := request["id"].(string)
			log.Printf("[conn-%d] 收到RPC请求: method=%s, id=%s", connID, method, id)

			// 处理initialize请求
			if method == "initialize" && !initialized {
				log.Printf("[conn-%d] 开始处理initialize请求", connID)
				// 发送initialize响应
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]interface{}{
						"capabilities": map[string]interface{}{
							"tools": map[string]interface{}{
								"listChanged": true,
							},
						},
						"protocolVersion": "mcp/v1",
						"serverInfo": map[string]interface{}{
							"name":    "context-keeper",
							"version": "1.0.0",
						},
					},
				}

				responseJSON, err := json.Marshal(response)
				if err != nil {
					log.Printf("[conn-%d] 序列化initialize响应失败: %v", connID, err)
					continue
				}

				err = writeSSE(c.Writer, "data: "+string(responseJSON)+"\n\n")
				if err != nil {
					log.Printf("[conn-%d] 发送initialize响应失败: %v", connID, err)
					return
				}
				log.Printf("[conn-%d] 已发送initialize响应", connID)

				// 获取所有已注册工具
				log.Printf("[conn-%d] 获取已注册工具列表", connID)
				toolsDefinitions := h.getRegisteredTools()
				log.Printf("[conn-%d] 找到 %d 个已注册工具", connID, len(toolsDefinitions))

				// 在初始化响应后发送manifest通知 (没有id字段，这是通知不是响应)
				manifest := map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "manifest",
					"params": map[string]interface{}{
						"$schema":        "http://json-schema.org/draft-07/schema#",
						"name":           "context-keeper",
						"display_name":   "Context Keeper",
						"description":    "编程上下文保持服务，用于增强大模型的代码理解能力",
						"id":             "context-keeper",
						"protocol":       "mcp/v1",
						"schema_version": "mcp/v1",
						"capabilities":   []string{"tools"},
						"tools":          toolsDefinitions,
					},
				}

				manifestJSON, err := json.Marshal(manifest)
				if err != nil {
					log.Printf("[conn-%d] 序列化manifest失败: %v", connID, err)
				} else {
					log.Printf("[conn-%d] 准备发送manifest通知", connID)
					err = writeSSE(c.Writer, "data: "+string(manifestJSON)+"\n\n")
					if err != nil {
						log.Printf("[conn-%d] 发送manifest失败: %v", connID, err)
						return
					}
					log.Printf("[conn-%d] 已发送manifest", connID)
				}

				// 提取工具名称列表用于通知
				var toolNames []string
				for _, tool := range toolsDefinitions {
					if name, ok := tool["name"].(string); ok {
						toolNames = append(toolNames, name)
					}
				}
				log.Printf("[conn-%d] 工具名称列表: %v", connID, toolNames)

				// 发送工具列表变更通知
				toolsChangedNotification := map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "tools/list/changed",
					"params": map[string]interface{}{
						"tools": toolNames,
					},
				}

				toolsChangedJSON, err := json.Marshal(toolsChangedNotification)
				if err != nil {
					log.Printf("[conn-%d] 序列化工具列表通知失败: %v", connID, err)
				} else {
					log.Printf("[conn-%d] 准备发送工具列表变更通知", connID)
					err = writeSSE(c.Writer, "data: "+string(toolsChangedJSON)+"\n\n")
					if err != nil {
						log.Printf("[conn-%d] 发送工具列表通知失败: %v", connID, err)
						return
					}
					log.Printf("[conn-%d] 已发送自动工具列表通知", connID)
				}

				// 标记为已初始化
				initialized = true
				log.Printf("[conn-%d] 初始化完成", connID)
			} else if method == "tools/list" {
				// 处理工具列表请求
				toolsDefinitions := h.getRegisteredTools()

				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]interface{}{
						"tools": toolsDefinitions,
					},
				}

				responseJSON, err := json.Marshal(response)
				if err != nil {
					log.Printf("[conn-%d] 序列化工具列表响应失败: %v", connID, err)
					continue
				}

				err = writeSSE(c.Writer, "data: "+string(responseJSON)+"\n\n")
				if err != nil {
					log.Printf("[conn-%d] 发送工具列表响应失败: %v", connID, err)
					return
				}
				log.Printf("[conn-%d] 已发送工具列表响应", connID)
			} else if method == "tools/call" {
				// 处理工具调用请求
				params, ok := request["params"].(map[string]interface{})
				if !ok || params == nil {
					log.Printf("[conn-%d] 工具调用缺少有效参数", connID)

					errorResponse := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      id,
						"error": map[string]interface{}{
							"code":    -32602,
							"message": "无效的参数",
						},
					}

					errorJSON, _ := json.Marshal(errorResponse)
					err = writeSSE(c.Writer, "data: "+string(errorJSON)+"\n\n")
					continue
				}

				// 提取工具名称和参数
				toolName, ok := params["name"].(string)
				if !ok || toolName == "" {
					log.Printf("[conn-%d] 工具调用缺少工具名称", connID)

					errorResponse := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      id,
						"error": map[string]interface{}{
							"code":    -32602,
							"message": "缺少工具名称",
						},
					}

					errorJSON, _ := json.Marshal(errorResponse)
					err = writeSSE(c.Writer, "data: "+string(errorJSON)+"\n\n")
					continue
				}

				arguments, _ := params["arguments"].(map[string]interface{})

				// 构建MCP请求对象
				mcpRequest := models.MCPRequest{
					JSONRPC: "2.0",
					ID:      id,
					Method:  "tools/call",
					Params: models.MCPParams{
						Name:      toolName,
						Arguments: arguments,
					},
				}

				// 调用工具处理器
				result, err := h.processMCPToolRequest(c.Request.Context(), mcpRequest)

				// 构建响应
				var responseObj map[string]interface{}
				if err != nil {
					responseObj = map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      id,
						"error": map[string]interface{}{
							"code":    -32603,
							"message": err.Error(),
						},
					}
				} else {
					responseObj = map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      id,
						"result":  result,
					}
				}

				// 序列化并发送响应
				responseJSON, err := json.Marshal(responseObj)
				if err != nil {
					log.Printf("[conn-%d] 序列化工具调用响应失败: %v", connID, err)
					continue
				}

				err = writeSSE(c.Writer, "data: "+string(responseJSON)+"\n\n")
				if err != nil {
					log.Printf("[conn-%d] 发送工具调用响应失败: %v", connID, err)
					return
				}
				log.Printf("[conn-%d] 已发送工具调用响应: %s", connID, toolName)
			} else {
				// 处理其他请求
				log.Printf("[conn-%d] 收到未处理的RPC请求: %s", connID, method)
			}
		case msg := <-messageChan:
			// 处理来自其他goroutine的消息
			err := writeSSE(c.Writer, msg)
			if err != nil {
				log.Printf("[conn-%d] 发送消息失败: %v", connID, err)
				return
			}
		case <-ticker.C:
			// 发送心跳
			err := writeSSE(c.Writer, ": ping\n\n")
			if err != nil {
				log.Printf("[conn-%d] 发送心跳失败: %v", connID, err)
				return
			}
		}
	}
}

// getRegisteredTools 获取所有已注册的工具定义
func (h *Handler) getRegisteredTools() []map[string]interface{} {
	// 返回与handleMCPToolsList中一致的工具定义
	return []map[string]interface{}{
		{
			"name":        "associate_file",
			"description": "关联代码文件到当前编程会话",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"filePath": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
				},
				"required": []string{"sessionId", "filePath"},
			},
		},
		{
			"name":        "record_edit",
			"description": "记录代码编辑操作",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"filePath": map[string]interface{}{
						"type":        "string",
						"description": "文件路径",
					},
					"diff": map[string]interface{}{
						"type":        "string",
						"description": "编辑差异内容",
					},
				},
				"required": []string{"sessionId", "filePath", "diff"},
			},
		},
		{
			"name":        "retrieve_context",
			"description": "基于查询检索相关编程上下文",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "查询内容",
					},
				},
				"required": []string{"sessionId", "query"},
			},
		},
		{
			"name":        "programming_context",
			"description": "获取编程特征和上下文摘要",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "可选查询参数",
					},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			"name":        "store_conversation",
			"description": "存储并总结当前对话内容到短期记忆",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"messages": map[string]interface{}{
						"type":        "array",
						"description": "对话消息列表",
					},
					"batchId": map[string]interface{}{
						"type":        "string",
						"description": "批次ID，可选，不提供则自动生成",
					},
				},
				"required": []string{"sessionId", "messages"},
			},
		},
		{
			"name":        "memorize_context",
			"description": "将重要内容汇总并存储到长期记忆",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "要记忆的内容",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "优先级，可选: P1(高), P2(中), P3(低)，默认P2",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "记忆相关的元数据，可选",
					},
				},
				"required": []string{"sessionId", "content"},
			},
		},
		{
			"name":        "retrieve_memory",
			"description": "基于memoryId或batchId检索历史对话",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"memoryId": map[string]interface{}{
						"type":        "string",
						"description": "记忆ID",
					},
					"batchId": map[string]interface{}{
						"type":        "string",
						"description": "批次ID",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "返回格式: full, summary",
					},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			"name":        "retrieve_todos",
			"description": "获取我的待办事项列表",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "筛选状态: all, pending, completed",
					},
					"limit": map[string]interface{}{
						"type":        "string",
						"description": "返回结果数量限制",
					},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			"name":        "session_management",
			"description": "创建或获取会话信息",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "操作类型: get_or_create",
					},
					"userId": map[string]interface{}{
						"type":        "string",
						"description": "用户ID，必需参数。客户端必须从配置文件获取：macOS: ~/Library/Application Support/context-keeper/user-config.json, Windows: ~/AppData/Roaming/context-keeper/user-config.json, Linux: ~/.local/share/context-keeper/user-config.json",
					},
					"workspaceRoot": map[string]interface{}{
						"type":        "string",
						"description": "工作空间根路径，必需参数，用于会话隔离，确保不同工作空间的session完全独立",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "会话元数据，可选",
					},
				},
				"required": []string{"action", "userId", "workspaceRoot"},
			},
		},
		{
			"name":        "user_init_dialog",
			"description": "用户初始化对话处理",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "当前会话ID",
					},
					"userResponse": map[string]interface{}{
						"type":        "string",
						"description": "用户对初始化提示的响应",
					},
				},
				"required": []string{"sessionId"},
			},
		},
	}
}

// processMCPToolRequest 处理MCP工具请求
func (h *Handler) processMCPToolRequest(ctx context.Context, request models.MCPRequest) (interface{}, error) {
	log.Printf("处理MCP工具请求: %s, ID: %s", request.Method, request.ID)

	if request.Method != "tools/call" {
		return nil, fmt.Errorf("不支持的方法: %s", request.Method)
	}

	// 提取工具名称和参数
	toolName := request.Params.Name
	arguments := request.Params.Arguments

	if toolName == "" {
		return nil, fmt.Errorf("缺少工具名称")
	}

	log.Printf("调用工具: %s, 参数: %+v", toolName, arguments)

	// 直接复用handlers.go中的dispatchToolCall函数处理业务逻辑
	// 这确保SSE和STDIO模式使用完全相同的业务实现
	return h.dispatchToolCall(toolName, arguments)
}

// generateRandomID 生成随机ID
func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// writeSSE 发送SSE格式数据
func writeSSE(w http.ResponseWriter, data string) error {
	_, err := fmt.Fprint(w, data)
	if err != nil {
		return err
	}
	w.(http.Flusher).Flush()
	return nil
}
