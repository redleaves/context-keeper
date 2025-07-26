package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamableHTTPHandler 专门处理MCP Streamable HTTP协议
type StreamableHTTPHandler struct {
	handler *Handler
}

// NewStreamableHTTPHandler 创建新的Streamable HTTP处理器
func NewStreamableHTTPHandler(handler *Handler) *StreamableHTTPHandler {
	return &StreamableHTTPHandler{
		handler: handler,
	}
}

// MCPRequest 表示MCP JSON-RPC请求
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id,omitempty"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse 表示MCP JSON-RPC响应
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError 表示MCP错误响应
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleStreamableHTTP 处理Streamable HTTP MCP请求
func (sh *StreamableHTTPHandler) HandleStreamableHTTP(c *gin.Context) {
	log.Printf("[Streamable HTTP] 收到请求: %s %s", c.Request.Method, c.Request.URL.Path)
	log.Printf("[Streamable HTTP] 请求头: %+v", c.Request.Header)

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// 处理OPTIONS预检请求
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	// 解析请求体
	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Streamable HTTP] JSON解析错误: %v", err)
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &MCPError{
				Code:    -32700,
				Message: "Parse error: " + err.Error(),
			},
		})
		return
	}

	// 验证JSONRPC版本
	if req.JSONRPC != "2.0" {
		log.Printf("[Streamable HTTP] 无效的JSONRPC版本: %s", req.JSONRPC)
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32600,
				Message: "Invalid Request: JSONRPC version must be 2.0",
			},
		})
		return
	}

	log.Printf("[Streamable HTTP] 处理方法: %s, ID: %v", req.Method, req.ID)

	// 使用defer来确保异常情况下也能返回合法的JSON响应
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Streamable HTTP] 发生恐慌: %v", r)
			c.JSON(http.StatusOK, MCPResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32603,
					Message: "Internal error: system panic",
					Data:    fmt.Sprintf("%v", r),
				},
			})
		}
	}()

	// 处理请求
	result, err := sh.processRequest(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Streamable HTTP] 处理错误: %v", err)

		// 根据错误类型确定错误代码
		var errorCode int
		errorMessage := err.Error()

		if strings.Contains(errorMessage, "方法不支持") {
			errorCode = -32601 // Method not found
		} else if strings.Contains(errorMessage, "缺少") || strings.Contains(errorMessage, "无效") {
			errorCode = -32602 // Invalid params
		} else {
			errorCode = -32603 // Internal error
		}

		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    errorCode,
				Message: errorMessage,
			},
		})
		return
	}

	// 返回成功响应
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	log.Printf("[Streamable HTTP] 响应结果: %+v", response)
	c.JSON(http.StatusOK, response)
}

// processRequest 处理具体的MCP请求
func (sh *StreamableHTTPHandler) processRequest(ctx context.Context, req MCPRequest) (interface{}, error) {
	switch req.Method {
	case "initialize":
		return sh.handleInitialize(ctx, req.Params)
	case "notifications/initialized":
		return sh.handleNotificationInitialized(ctx, req.Params)
	case "tools/list":
		return sh.handleToolsList(ctx, req.Params)
	case "tools/call":
		return sh.handleToolsCall(ctx, req.Params)
	default:
		log.Printf("[Streamable HTTP] 不支持的方法: %s", req.Method)
		return nil, fmt.Errorf("方法不支持: %s", req.Method)
	}
}

// handleInitialize 处理初始化请求
func (sh *StreamableHTTPHandler) handleInitialize(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	log.Printf("[Streamable HTTP] 处理初始化请求")

	// 返回服务器能力信息
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "context-keeper",
			"version": "1.0.0",
		},
	}, nil
}

// handleNotificationInitialized 处理初始化完成通知
func (sh *StreamableHTTPHandler) handleNotificationInitialized(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	log.Printf("[Streamable HTTP] 处理初始化完成通知")

	// 对于notification类型的请求，通常不需要返回响应内容
	// 但为了兼容性，我们返回一个简单的确认
	return map[string]interface{}{
		"status": "acknowledged",
	}, nil
}

// handleToolsList 处理工具列表请求
func (sh *StreamableHTTPHandler) handleToolsList(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	log.Printf("[Streamable HTTP] 处理工具列表请求")

	// 获取工具定义
	tools := sh.getToolsDefinition()

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall 处理工具调用请求
func (sh *StreamableHTTPHandler) handleToolsCall(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	toolName, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少工具名称")
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	log.Printf("[Streamable HTTP] 调用工具: %s, 参数: %+v", toolName, arguments)

	// 记录调用开始时间
	startTime := time.Now()

	// 调用工具
	result, err := sh.handler.dispatchToolCall(toolName, arguments)

	// 记录调用耗时
	duration := time.Since(startTime)

	// 记录工具调用日志
	sh.logToolCall(toolName, arguments, result, err, duration)

	if err != nil {
		return nil, err
	}

	// 正确序列化结果为JSON字符串
	var resultText string
	if jsonBytes, err := json.Marshal(result); err != nil {
		// 如果JSON序列化失败，使用字符串表示
		resultText = fmt.Sprintf("%v", result)
		log.Printf("[Streamable HTTP] JSON序列化失败，使用字符串表示: %v", err)
	} else {
		resultText = string(jsonBytes)
		log.Printf("[Streamable HTTP] JSON序列化成功，长度: %d", len(jsonBytes))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resultText,
			},
		},
	}, nil
}

// getToolsDefinition 获取工具定义
func (sh *StreamableHTTPHandler) getToolsDefinition() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "associate_file",
			"description": "关联代码文件到当前编程会话",
			"inputSchema": map[string]interface{}{
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
			"inputSchema": map[string]interface{}{
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
			"inputSchema": map[string]interface{}{
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
			"inputSchema": map[string]interface{}{
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
			"name":        "session_management",
			"description": "创建或获取会话信息",
			"inputSchema": map[string]interface{}{
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
			"name":        "store_conversation",
			"description": "存储并总结当前对话内容到短期记忆",
			"inputSchema": map[string]interface{}{
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
			"name":        "retrieve_memory",
			"description": "基于memoryId或batchId检索历史对话",
			"inputSchema": map[string]interface{}{
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
			"name":        "memorize_context",
			"description": "将重要内容汇总并存储到长期记忆",
			"inputSchema": map[string]interface{}{
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
			"name":        "retrieve_todos",
			"description": "获取我的待办事项列表",
			"inputSchema": map[string]interface{}{
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
			"name":        "user_init_dialog",
			"description": "用户初始化对话处理",
			"inputSchema": map[string]interface{}{
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

// logToolCall 记录工具调用的详细日志
func (sh *StreamableHTTPHandler) logToolCall(name string, request map[string]interface{}, response interface{}, err error, duration time.Duration) {
	// 将请求参数转为漂亮的JSON格式
	requestJSON, jsonErr := json.MarshalIndent(request, "", "  ")
	if jsonErr != nil {
		requestJSON = []byte(fmt.Sprintf("无法序列化请求: %v", jsonErr))
	}

	// 将响应内容转为漂亮的JSON格式
	var responseJSON []byte
	if err != nil {
		responseJSON = []byte(fmt.Sprintf("错误: %v", err))
	} else {
		var jsonErr error
		switch v := response.(type) {
		case string:
			// 尝试解析字符串为JSON对象以美化输出
			var jsonObj interface{}
			if unmarshalErr := json.Unmarshal([]byte(v), &jsonObj); unmarshalErr == nil {
				responseJSON, jsonErr = json.MarshalIndent(jsonObj, "", "  ")
			} else {
				responseJSON = []byte(v)
			}
		default:
			responseJSON, jsonErr = json.MarshalIndent(v, "", "  ")
			if jsonErr != nil {
				responseJSON = []byte(fmt.Sprintf("无法序列化响应: %v", jsonErr))
			}
		}
	}

	// 记录详细日志
	divider := "====================================================="
	log.Printf("\n%s\n[Streamable HTTP 工具调用: %s]\n%s", divider, name, divider)
	log.Printf("耗时: %v", duration)
	log.Printf("请求参数:\n%s", string(requestJSON))
	log.Printf("响应结果:\n%s", string(responseJSON))
	if err != nil {
		log.Printf("错误: %v", err)
	}
	log.Printf("%s\n[工具调用结束: %s]\n%s\n", divider, name, divider)
}

// RegisterStreamableHTTPRoutes 注册Streamable HTTP路由
func (sh *StreamableHTTPHandler) RegisterStreamableHTTPRoutes(router *gin.Engine) {
	// MCP Streamable HTTP端点
	router.POST("/mcp", sh.HandleStreamableHTTP)

	// 也可以支持GET请求用于能力查询
	router.GET("/mcp/capabilities", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"transport":       "streamable-http",
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "context-keeper",
				"version": "1.0.0",
			},
		})
	})
	log.Printf("[Streamable HTTP] 注册MCP路由: POST /mcp")
}

// formatResultAsText 将结果格式化为可读的文本格式
func formatResultAsText(result interface{}) string {
	if result == nil {
		return "操作成功完成"
	}

	// 尝试格式化为JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("结果: %v", result)
	}

	return string(jsonBytes)
}
