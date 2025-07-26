package models

// MCP协议相关的数据结构定义

// MCPRequest 表示MCP协议请求
type MCPRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      string    `json:"id"`
	Method  string    `json:"method"`
	Params  MCPParams `json:"params"`
}

// MCPParams MCP协议请求参数
type MCPParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPRPCResponse MCP协议RPC响应
type MCPRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError MCP错误响应
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPToolCallRequest 工具调用请求
type MCPToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolCallResponse 工具调用响应
type MCPToolCallResponse struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent 内容块
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
