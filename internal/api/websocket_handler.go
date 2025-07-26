package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/contextkeeper/service/internal/services"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源的连接（生产环境中应该限制）
		return true
	},
}

// HandleWebSocket 处理WebSocket连接请求 - 支持工作空间级别连接隔离
func (h *Handler) HandleWebSocket(c *gin.Context) {
	log.Printf("🔗 [WebSocket连接] ===== 开始WebSocket连接处理 =====")

	// 🔥 修复：获取连接ID和工作空间信息
	userID := c.Query("userId")
	workspaceParam := c.Query("workspace") // 新增：直接从参数获取工作空间信息

	log.Printf("🔗 [WebSocket连接] 接收到的URL参数: userID=%s, workspace=%s", userID, workspaceParam)

	if userID == "" {
		log.Printf("🔗 [WebSocket连接] userID为空，尝试从系统获取用户ID")
		// 尝试从系统获取用户ID
		var err error
		userID, _, err = utils.GetUserID()
		if err != nil || userID == "" {
			log.Printf("🔗 [WebSocket连接] ❌ 获取用户ID失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "用户ID不能为空",
			})
			return
		}
		log.Printf("🔗 [WebSocket连接] ✅ 从系统获取用户ID成功: %s", userID)
	}

	// 🔥 重构：使用统一的工作空间标识方法
	workspaceHash := utils.GetWorkspaceIdentifier(workspaceParam)
	log.Printf("🔗 [WebSocket连接] 工作空间标识计算: '%s' -> '%s'", workspaceParam, workspaceHash)

	// 🔥 修复：始终生成工作空间级别的连接ID
	var connectionID string
	if workspaceHash == "" {
		log.Printf("🔗 [WebSocket连接] 工作空间哈希为空，生成随机标识")
		// 如果无法获取工作空间标识，生成一个随机的工作空间标识
		workspaceHash = utils.GenerateRandomString(8)
		log.Printf("🔗 [WebSocket连接] 生成随机工作空间标识: %s", workspaceHash)
	}
	// 始终包含工作空间标识，确保不同工作空间的连接被正确隔离
	connectionID = fmt.Sprintf("%s_ws_%s", userID, workspaceHash)

	log.Printf("🔗 [WebSocket连接] 连接ID生成: %s", connectionID)
	log.Printf("🔗 [WebSocket连接] 连接详情: userID=%s, workspace=%s, workspaceHash=%s, connectionID=%s",
		userID, workspaceParam, workspaceHash, connectionID)

	// 🔥 新增：获取用户会话存储并创建/获取活跃会话
	log.Printf("🔗 [WebSocket连接] 获取用户会话存储")
	userSessionStore, err := h.contextService.GetUserSessionStore(userID)
	if err != nil {
		log.Printf("🔗 [WebSocket连接] ❌ 获取用户会话存储失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取用户会话存储失败",
		})
		return
	}
	log.Printf("🔗 [WebSocket连接] ✅ 用户会话存储获取成功")

	// 🔥 修复：使用统一的会话获取逻辑，确保与MCP工具一致
	metadata := map[string]interface{}{
		"workspaceHash": workspaceHash,
		"connectionId":  connectionID,
		"source":        "websocket",
	}

	log.Printf("🔗 [WebSocket连接] 准备获取或创建会话: userID=%s, workspaceParam=%s, metadata=%+v", userID, workspaceParam, metadata)

	session, isNewSession, err := utils.GetWorkspaceSessionID(
		userSessionStore,
		userID,
		"",             // 不指定sessionID，让系统自动获取或创建
		workspaceParam, // 🔥 修复：直接传递工作空间路径参数
		metadata,
		h.config.SessionTimeout,
	)
	if err != nil {
		log.Printf("🔗 [WebSocket连接] ❌ 获取或创建会话失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取或创建会话失败",
		})
		return
	}

	if isNewSession {
		log.Printf("🔗 [WebSocket连接] 🆕 为连接创建新会话: %s (工作空间: %s)", session.ID, workspaceHash)
	} else {
		log.Printf("🔗 [WebSocket连接] 🔄 连接复用现有会话: %s (工作空间: %s)", session.ID, workspaceHash)
	}

	// 升级HTTP连接为WebSocket
	log.Printf("🔗 [WebSocket连接] 升级HTTP连接为WebSocket")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("🔗 [WebSocket连接] ❌ 升级连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "升级WebSocket连接失败",
		})
		return
	}
	log.Printf("🔗 [WebSocket连接] ✅ WebSocket连接升级成功")

	// 注册连接
	log.Printf("🔗 [WebSocket连接] 注册连接: %s", connectionID)
	services.GlobalWSManager.RegisterUser(connectionID, conn)

	// 🔥 修复：自动注册会话到连接的映射
	log.Printf("🔗 [WebSocket连接] 自动注册会话映射: %s → 连接: %s", session.ID, connectionID)
	success := services.GlobalWSManager.RegisterSession(session.ID, connectionID)
	if success {
		log.Printf("🔗 [WebSocket连接] ✅ 自动注册会话映射成功: %s → 连接: %s", session.ID, connectionID)
	} else {
		log.Printf("🔗 [WebSocket连接] ⚠️ 自动注册会话映射失败: %s → 连接: %s", session.ID, connectionID)
	}

	log.Printf("🔗 [WebSocket连接] ✅ 连接已建立: userID=%s, connectionID=%s, sessionID=%s, workspace=%s",
		userID, connectionID, session.ID, workspaceHash)
	log.Printf("🔗 [WebSocket连接] ===== WebSocket连接处理完成 =====")

}

// GetWebSocketStatus 获取WebSocket连接状态 - 显示多工作空间连接统计
func (h *Handler) GetWebSocketStatus(c *gin.Context) {
	onlineUsers := services.GlobalWSManager.GetOnlineUsers()
	connectionStats := services.GlobalWSManager.GetConnectionStats()

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"onlineUsers":      onlineUsers,
		"onlineCount":      len(onlineUsers),
		"totalConnections": connectionStats["total_connections"],
		"userConnections":  connectionStats["user_connections"],
		"mode":             "workspace-isolated",
	})
}

// 🔥 新增：HandleSessionRegister 处理会话注册请求
func (h *Handler) HandleSessionRegister(c *gin.Context) {
	log.Printf("🔗 [会话注册] ===== 开始处理会话注册请求 =====")

	var req struct {
		SessionID    string `json:"sessionId" binding:"required"`
		ConnectionID string `json:"connectionId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("🔗 [会话注册] ❌ 参数错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	log.Printf("🔗 [会话注册] 接收到的参数: sessionID=%s, connectionID=%s", req.SessionID, req.ConnectionID)

	// 验证连接ID格式
	userID := services.GlobalWSManager.ExtractUserIDFromConnectionID(req.ConnectionID)
	log.Printf("🔗 [会话注册] 从连接ID中提取用户ID: %s", userID)
	log.Printf("🔗 [会话注册] 处理注册请求: sessionID=%s, connectionID=%s (用户: %s)",
		req.SessionID, req.ConnectionID, userID)

	// 🔥 修复：检查连接ID是否包含工作空间标识
	var actualConnectionID string
	log.Printf("🔗 [会话注册] 检查连接ID格式: %s", req.ConnectionID)
	if !strings.Contains(req.ConnectionID, "_ws_") {
		log.Printf("🔗 [会话注册] ⚠️ 连接ID格式不正确，查找用户活跃连接")
		// 如果连接ID不包含工作空间标识，查找该用户的实际连接ID
		connections := services.GlobalWSManager.GetUserConnections(userID)
		log.Printf("🔗 [会话注册] 用户 %s 的活跃连接: %v", userID, connections)
		if len(connections) > 0 {
			actualConnectionID = connections[0]
			log.Printf("🔗 [会话注册] ⚠️ 连接ID格式不正确，已修正: %s → %s", req.ConnectionID, actualConnectionID)
		} else {
			log.Printf("🔗 [会话注册] ❌ 用户 %s 没有活跃连接", userID)
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "用户没有活跃连接",
			})
			return
		}
	} else {
		actualConnectionID = req.ConnectionID
		log.Printf("🔗 [会话注册] ✅ 连接ID格式正确: %s", actualConnectionID)
	}

	// 注册会话到连接的映射
	log.Printf("🔗 [会话注册] 注册会话映射: %s → 连接: %s", req.SessionID, actualConnectionID)
	success := services.GlobalWSManager.RegisterSession(req.SessionID, actualConnectionID)

	// 🔥 修复：检查注册是否成功
	if !success {
		log.Printf("🔗 [会话注册] ❌ 会话注册失败: %s → 连接: %s (连接不存在)",
			req.SessionID, actualConnectionID)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "连接不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "会话注册成功",
		"sessionId":    req.SessionID,
		"connectionId": actualConnectionID,
		"userId":       userID,
	})

	log.Printf("🔗 [会话注册] ✅ 会话注册成功: %s → 连接: %s",
		req.SessionID, actualConnectionID)
	log.Printf("🔗 [会话注册] ===== 会话注册处理完成 =====")

}

// GetWSDebugStatus 获取WebSocket连接状态详情（调试用）
func (h *Handler) GetWSDebugStatus(c *gin.Context) {
	// 获取在线用户
	onlineUsers := services.GlobalWSManager.GetOnlineUsers()

	// 构建详细信息
	debugInfo := map[string]interface{}{
		"status":       "success",
		"online_users": onlineUsers,
		"connections":  make(map[string]interface{}),
	}

	// 遍历每个用户，获取其连接详情
	for _, userID := range onlineUsers {
		connections := services.GlobalWSManager.GetUserConnections(userID)
		userInfo := map[string]interface{}{
			"connection_count": len(connections),
			"connection_ids":   connections,
		}
		debugInfo["connections"].(map[string]interface{})[userID] = userInfo
	}

	c.JSON(200, debugInfo)
}
