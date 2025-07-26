package api

import (
	"log"
	"net/http"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/services"
	"github.com/gin-gonic/gin"
)

// CursorHandler Cursor相关API处理函数
type CursorHandler struct {
	cursorAdapter   *services.CursorAdapter
	contextService  *services.ContextService
	decisionService *services.DecisionService // 新增决策服务
}

// NewCursorHandler 创建新的Cursor处理器
func NewCursorHandler(cursorAdapter *services.CursorAdapter, contextService *services.ContextService, decisionService *services.DecisionService) *CursorHandler {
	return &CursorHandler{
		cursorAdapter:   cursorAdapter,
		contextService:  contextService,
		decisionService: decisionService,
	}
}

// RegisterRoutes 注册Cursor特定的路由
func (h *CursorHandler) RegisterRoutes(router *gin.Engine) {
	cursor := router.Group("/api/cursor")
	{
		// 专门针对Cursor的上下文检索接口
		cursor.POST("/retrieveContext", h.handleRetrieveContext)

		// 专门针对Cursor的文件关联接口
		cursor.POST("/associateFile", h.handleAssociateFile)

		// 专门针对Cursor的编辑记录接口
		cursor.POST("/recordEdit", h.handleRecordEdit)

		// 获取编程上下文
		cursor.POST("/programmingContext", h.handleProgrammingContext)

		// 创建设计决策
		cursor.POST("/createDecision", h.handleCreateDecision)

		// 关联决策与编辑
		cursor.POST("/linkDecisionToEdits", h.handleLinkDecisionToEdits)

		// 获取会话的设计决策
		cursor.POST("/getDecisions", h.handleGetDecisions)

		// 创建会话关联
		cursor.POST("/createSessionLink", h.handleCreateSessionLink)

		// 获取关联会话
		cursor.POST("/getLinkedSessions", h.handleGetLinkedSessions)
	}
}

// handleRetrieveContext 处理Cursor的上下文检索请求
func (h *CursorHandler) handleRetrieveContext(c *gin.Context) {
	var req models.RetrieveContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}

	// 使用Cursor适配器获取上下文
	resp, err := h.cursorAdapter.GetContextForCursor(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 检索上下文失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "检索上下文失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleAssociateFile 处理Cursor的文件关联请求
func (h *CursorHandler) handleAssociateFile(c *gin.Context) {
	var req models.MCPCodeAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}
	if req.FilePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "文件路径不能为空",
		})
		return
	}

	// 使用Cursor适配器关联文件
	err := h.cursorAdapter.AssociateCodeFile(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 关联文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "关联文件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MCPResponse{
		Status:  "success",
		Message: "文件关联成功",
	})
}

// handleRecordEdit 处理Cursor的编辑记录请求
func (h *CursorHandler) handleRecordEdit(c *gin.Context) {
	var req models.MCPEditRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}
	if req.FilePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "文件路径不能为空",
		})
		return
	}

	// 使用Cursor适配器记录编辑
	err := h.cursorAdapter.RecordEditAction(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 记录编辑失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "记录编辑失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MCPResponse{
		Status:  "success",
		Message: "编辑记录成功",
	})
}

// handleProgrammingContext 处理获取编程上下文请求
func (h *CursorHandler) handleProgrammingContext(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
		Query     string `json:"query,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}

	// 提取编程上下文
	progContext, err := h.cursorAdapter.ExtractProgrammingContext(c.Request.Context(), req.SessionID, req.Query)
	if err != nil {
		log.Printf("[Cursor API] 提取编程上下文失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "提取编程上下文失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, progContext)
}

// handleCreateDecision 处理创建设计决策请求
func (h *CursorHandler) handleCreateDecision(c *gin.Context) {
	var req models.CreateDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "决策标题不能为空",
		})
		return
	}
	if req.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "决策描述不能为空",
		})
		return
	}

	// 创建设计决策
	resp, err := h.decisionService.CreateDecision(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 创建设计决策失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建设计决策失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleLinkDecisionToEdits 处理关联决策与编辑请求
func (h *CursorHandler) handleLinkDecisionToEdits(c *gin.Context) {
	var req models.LinkDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}
	if req.DecisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "决策ID不能为空",
		})
		return
	}
	if req.EditIDs == nil || len(req.EditIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "编辑ID列表不能为空",
		})
		return
	}

	// 关联决策与编辑
	resp, err := h.decisionService.LinkDecisionToEdits(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 关联决策与编辑失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "关联决策与编辑失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleGetDecisions 处理获取设计决策请求
func (h *CursorHandler) handleGetDecisions(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}

	// 获取设计决策
	decisions, err := h.decisionService.GetDecisionsForSession(c.Request.Context(), req.SessionID)
	if err != nil {
		log.Printf("[Cursor API] 获取设计决策失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取设计决策失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"decisions": decisions,
	})
}

// handleCreateSessionLink 处理创建会话关联请求
func (h *CursorHandler) handleCreateSessionLink(c *gin.Context) {
	var req models.SessionLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SourceID == "" || req.TargetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "源会话ID和目标会话ID不能为空",
		})
		return
	}
	if req.Relationship == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "关系类型不能为空",
		})
		return
	}

	// 创建会话关联
	resp, err := h.decisionService.CreateSessionLink(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Cursor API] 创建会话关联失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建会话关联失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleGetLinkedSessions 处理获取关联会话请求
func (h *CursorHandler) handleGetLinkedSessions(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 验证必要字段
	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}

	// 获取关联会话
	sessions, err := h.decisionService.GetLinkedSessions(c.Request.Context(), req.SessionID)
	if err != nil {
		log.Printf("[Cursor API] 获取关联会话失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取关联会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"sessions": sessions,
	})
}
