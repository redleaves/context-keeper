package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/contextkeeper/service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
)

// 上下文键名保持统一：session_id、user_id、workspacePath、workspaceHash
const (
	CtxKeySessionID     = "session_id" // 🆕 新增：会话ID
	CtxKeyUserID        = "user_id"
	CtxKeyWorkspacePath = "workspacePath"
	CtxKeyWorkspaceHash = "workspaceHash"
)

// InjectSessionContext 基于 sessionId 注入 sessionId、userId、workspacePath、workspaceHash 到上下文
// 仅依赖 ContextService，便于在各工具/路由的处理入口统一调用
func InjectSessionContext(ctx context.Context, cs *services.ContextService, sessionID string) context.Context {
	if cs == nil || sessionID == "" {
		return ctx
	}

	// 获取用户ID
	userID, err := cs.GetUserIDFromSessionID(sessionID)
	if err != nil {
		log.Printf("[CtxInject] 未能通过sessionID获取用户ID: %v", err)
		return ctx
	}

	// 读取会话元数据以获得工作空间信息
	var workspacePath, workspaceHash string
	// 🔥 修复：使用全局SessionStore，与session_management保持一致
	if sess, err := cs.SessionStore().GetSession(sessionID); err == nil && sess != nil && sess.Metadata != nil {
		if wp, ok := sess.Metadata["workspacePath"].(string); ok {
			workspacePath = wp
		}
		if wh, ok := sess.Metadata["workspaceHash"].(string); ok {
			workspaceHash = wh
		}
		log.Printf("[CtxInject] 成功从全局SessionStore获取会话元数据")
	} else {
		log.Printf("[CtxInject] 从全局SessionStore获取会话元数据失败: %v", err)
	}

	// 注入上下文
	ctx = context.WithValue(ctx, CtxKeySessionID, sessionID) // 🆕 注入会话ID
	ctx = context.WithValue(ctx, CtxKeyUserID, userID)
	if workspacePath != "" {
		ctx = context.WithValue(ctx, CtxKeyWorkspacePath, workspacePath)
	}
	if workspaceHash != "" {
		ctx = context.WithValue(ctx, CtxKeyWorkspaceHash, workspaceHash)
	}

	log.Printf("[CtxInject] 注入完成: sessionId=%s, userId=%s, workspacePath=%s, workspaceHash=%s",
		sessionID, userID, workspacePath, workspaceHash)
	return ctx
}

// SessionContextConfig 可配置拦截：按HTTP路由前缀、HTTP方法、或MCP工具名生效
type SessionContextConfig struct {
	// HTTP 路由前缀白名单，如 ["/mcp", "/api/context", "/api/ws/"]
	HTTPPathPrefixes []string
	// HTTP 方法白名单，如 ["POST", "GET"]，空表示不限制
	HTTPMethods []string
	// MCP 工具名白名单，如 ["retrieve_context", "get_context", "store_conversation"]
	MCPTools []string
}

// ShouldInterceptHTTP 判断是否拦截当前HTTP请求
func (c *SessionContextConfig) ShouldInterceptHTTP(r *http.Request) bool {
	if c == nil {
		return false
	}
	path := r.URL.Path
	method := r.Method
	// 方法限制
	if len(c.HTTPMethods) > 0 {
		ok := false
		for _, m := range c.HTTPMethods {
			if strings.EqualFold(m, method) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	// 路由前缀限制
	if len(c.HTTPPathPrefixes) > 0 {
		for _, p := range c.HTTPPathPrefixes {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
		return false
	}
	return false
}

// ShouldInterceptMCP 判断是否拦截指定MCP工具
func (c *SessionContextConfig) ShouldInterceptMCP(tool string) bool {
	if c == nil || tool == "" {
		return false
	}
	for _, t := range c.MCPTools {
		if t == tool {
			return true
		}
	}
	return false
}

// ========== 集成辅助 ==========

// NewSessionContextMiddleware 生成Gin中间件：可配置拦截指定HTTP路由/方法，将session上下文注入到request.Context
// 使用方式（在路由注册处）：
//
//	router.Use(api.NewSessionContextMiddleware(cfg, handler.GetContextService()))
func NewSessionContextMiddleware(cfg *SessionContextConfig, cs *services.ContextService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 未配置则直接透传
			if cfg == nil || !cfg.ShouldInterceptHTTP(r) {
				next.ServeHTTP(w, r)
				return
			}

			// 提取sessionId（query 或 form）
			sessionID := r.URL.Query().Get("sessionId")
			if sessionID == "" && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
				_ = r.ParseForm()
				if v := r.Form.Get("sessionId"); v != "" {
					sessionID = v
				}
			}

			if sessionID != "" {
				ctx := InjectSessionContext(r.Context(), cs, sessionID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WrapMCPToolWithInjection 包装MCP工具处理函数：按配置拦截指定工具，自动注入session上下文
// 使用方式（在工具注册时）：
//
//	s.AddTool(tool, api.WrapMCPToolWithInjection("retrieve_context", baseCS, cfg, retrieveContextHandler(baseCS)))
func WrapMCPToolWithInjection(toolName string, cs *services.ContextService, cfg *SessionContextConfig,
	handler func(ctx context.Context, request interface{}) (interface{}, error)) func(ctx context.Context, request interface{}) (interface{}, error) {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if cfg != nil && cfg.ShouldInterceptMCP(toolName) {
			// 尝试从请求对象中提取sessionId（MCP通用结构：Map或有Params.Arguments）
			// 为避免强依赖具体类型，这里做通用的反射式提取
			var sid string
			switch req := request.(type) {
			case map[string]interface{}:
				if params, ok := req["params"].(map[string]interface{}); ok {
					if args, ok := params["arguments"].(map[string]interface{}); ok {
						if v, ok := args["sessionId"].(string); ok {
							sid = v
						}
					}
				}
			default:
				// 其他类型由具体handler内部保障
			}
			if sid != "" {
				ctx = InjectSessionContext(ctx, cs, sid)
			}
		}
		return handler(ctx, request)
	}
}

// WrapMCPToolWithInjectionV2 针对 mcp-go 的强类型包装
func WrapMCPToolWithInjectionV2(toolName string, cs *services.ContextService, cfg *SessionContextConfig,
	handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if cfg != nil && cfg.ShouldInterceptMCP(toolName) {
			if sid, ok := request.Params.Arguments["sessionId"].(string); ok && sid != "" {
				log.Printf("[CtxInject] 命中MCP拦截: tool=%s, sessionId=%s", toolName, sid)
				ctx = InjectSessionContext(ctx, cs, sid)
			}
		}
		return handler(ctx, request)
	}
}

// BuildSessionContextConfigFromEnv 从环境变量构建拦截配置
// INTERCEPT_HTTP_PATH_PREFIXES=/mcp,/api/context
// INTERCEPT_HTTP_METHODS=POST,GET
// INTERCEPT_MCP_TOOLS=retrieve_context,get_context
func BuildSessionContextConfigFromEnv() *SessionContextConfig {
	parseCSV := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return &SessionContextConfig{
		HTTPPathPrefixes: parseCSV(getenv("INTERCEPT_HTTP_PATH_PREFIXES")),
		HTTPMethods:      parseCSV(getenv("INTERCEPT_HTTP_METHODS")),
		MCPTools:         parseCSV(getenv("INTERCEPT_MCP_TOOLS")),
	}
}

// NewGinSessionContextMiddleware Gin 版本中间件
func NewGinSessionContextMiddleware(cfg *SessionContextConfig, cs *services.ContextService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg != nil && cfg.ShouldInterceptHTTP(c.Request) {
			sessionID := c.Query("sessionId")
			if sessionID == "" && (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch) {
				_ = c.Request.ParseForm()
				if v := c.Request.FormValue("sessionId"); v != "" {
					sessionID = v
				}
			}
			if sessionID != "" {
				ctx := InjectSessionContext(c.Request.Context(), cs, sessionID)
				c.Request = c.Request.WithContext(ctx)
			}
		}
		c.Next()
	}
}

// getenv 包装，兼容测试
func getenv(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
