package config

import (
	"encoding/json"
	"fmt"
	"log"
)

// AgenticControlRequest Agentic控制请求
type AgenticControlRequest struct {
	Action  string `json:"action"`  // enable, disable, status, config
	Feature string `json:"feature"` // 特性名称
}

// AgenticControlResponse Agentic控制响应
type AgenticControlResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// MCPAgenticTool Agentic功能的MCP工具
type MCPAgenticTool struct {
	flagManager *FeatureFlagManager
}

// NewMCPAgenticTool 创建Agentic MCP工具
func NewMCPAgenticTool(flagManager *FeatureFlagManager) *MCPAgenticTool {
	return &MCPAgenticTool{
		flagManager: flagManager,
	}
}

// HandleAgenticControl 处理Agentic控制请求
func (t *MCPAgenticTool) HandleAgenticControl(params map[string]interface{}) (interface{}, error) {
	// 解析请求参数
	var req AgenticControlRequest
	paramBytes, _ := json.Marshal(params)
	if err := json.Unmarshal(paramBytes, &req); err != nil {
		return &AgenticControlResponse{
			Success: false,
			Message: fmt.Sprintf("参数解析失败: %v", err),
		}, nil
	}

	log.Printf("🎛️ 处理Agentic控制请求: action=%s, feature=%s", req.Action, req.Feature)

	switch req.Action {
	case "enable":
		return t.enableFeature(req.Feature)
	case "disable":
		return t.disableFeature(req.Feature)
	case "status":
		return t.getStatus()
	case "config":
		return t.getConfig()
	case "list":
		return t.listFeatures()
	default:
		return &AgenticControlResponse{
			Success: false,
			Message: fmt.Sprintf("未知操作: %s", req.Action),
		}, nil
	}
}

// enableFeature 启用特性
func (t *MCPAgenticTool) enableFeature(feature string) (*AgenticControlResponse, error) {
	if feature == "" {
		return &AgenticControlResponse{
			Success: false,
			Message: "特性名称不能为空",
		}, nil
	}

	err := t.flagManager.EnableFeature(feature)
	if err != nil {
		return &AgenticControlResponse{
			Success: false,
			Message: fmt.Sprintf("启用特性失败: %v", err),
		}, nil
	}

	log.Printf("✅ 已启用特性: %s", feature)
	return &AgenticControlResponse{
		Success: true,
		Message: fmt.Sprintf("成功启用特性: %s", feature),
		Data: map[string]interface{}{
			"feature": feature,
			"enabled": true,
		},
	}, nil
}

// disableFeature 禁用特性
func (t *MCPAgenticTool) disableFeature(feature string) (*AgenticControlResponse, error) {
	if feature == "" {
		return &AgenticControlResponse{
			Success: false,
			Message: "特性名称不能为空",
		}, nil
	}

	err := t.flagManager.DisableFeature(feature)
	if err != nil {
		return &AgenticControlResponse{
			Success: false,
			Message: fmt.Sprintf("禁用特性失败: %v", err),
		}, nil
	}

	log.Printf("⏸️ 已禁用特性: %s", feature)
	return &AgenticControlResponse{
		Success: true,
		Message: fmt.Sprintf("成功禁用特性: %s", feature),
		Data: map[string]interface{}{
			"feature": feature,
			"enabled": false,
		},
	}, nil
}

// getStatus 获取状态
func (t *MCPAgenticTool) getStatus() (*AgenticControlResponse, error) {
	flags := t.flagManager.ListFlags()
	config := t.flagManager.GetConfig()

	status := map[string]interface{}{
		"flags":  flags,
		"config": config,
		"summary": map[string]interface{}{
			"total_flags":    len(flags),
			"enabled_flags":  t.countEnabledFlags(flags),
			"agentic_active": t.isAgenticActive(flags),
		},
	}

	log.Printf("📊 返回Agentic状态信息")
	return &AgenticControlResponse{
		Success: true,
		Message: "获取状态成功",
		Data:    status,
	}, nil
}

// getConfig 获取配置
func (t *MCPAgenticTool) getConfig() (*AgenticControlResponse, error) {
	config := t.flagManager.GetConfig()

	log.Printf("⚙️ 返回Agentic配置信息")
	return &AgenticControlResponse{
		Success: true,
		Message: "获取配置成功",
		Data: map[string]interface{}{
			"config": config,
		},
	}, nil
}

// listFeatures 列出所有特性
func (t *MCPAgenticTool) listFeatures() (*AgenticControlResponse, error) {
	flags := t.flagManager.ListFlags()

	features := make([]map[string]interface{}, 0, len(flags))
	for name, flag := range flags {
		features = append(features, map[string]interface{}{
			"name":        name,
			"enabled":     flag.Enabled,
			"description": flag.Description,
		})
	}

	log.Printf("📋 返回特性列表，共%d个特性", len(features))
	return &AgenticControlResponse{
		Success: true,
		Message: fmt.Sprintf("获取特性列表成功，共%d个特性", len(features)),
		Data: map[string]interface{}{
			"features": features,
			"count":    len(features),
		},
	}, nil
}

// countEnabledFlags 计算启用的特性数量
func (t *MCPAgenticTool) countEnabledFlags(flags map[string]*FeatureFlag) int {
	count := 0
	for _, flag := range flags {
		if flag.Enabled {
			count++
		}
	}
	return count
}

// isAgenticActive 检查Agentic功能是否活跃
func (t *MCPAgenticTool) isAgenticActive(flags map[string]*FeatureFlag) bool {
	primaryFeatures := []string{"retrieval_decision", "query_optimization", "quality_evaluation"}

	for _, feature := range primaryFeatures {
		if flag, exists := flags[feature]; exists && flag.Enabled {
			return true
		}
	}
	return false
}

// GetToolDefinition 获取MCP工具定义
func (t *MCPAgenticTool) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "agentic_control",
		"description": "控制Context-Keeper的Agentic RAG功能",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "操作类型",
					"enum":        []string{"enable", "disable", "status", "config", "list"},
				},
				"feature": map[string]interface{}{
					"type":        "string",
					"description": "特性名称（enable/disable操作时必需）",
					"enum":        []string{"retrieval_decision", "query_optimization", "quality_evaluation", "multi_hop_reasoning"},
				},
			},
			"required": []string{"action"},
		},
	}
}
