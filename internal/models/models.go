package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// API请求/响应模型 ---------------------------------

// StoreContextRequest 存储上下文请求
type StoreContextRequest struct {
	SessionID string                 `json:"sessionId"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Priority  string                 `json:"priority,omitempty"` // P0-P3
	// 添加bizType和userId字段，用于向量存储
	BizType int    `json:"bizType,omitempty"`
	UserID  string `json:"userId,omitempty"`
}

// RetrieveContextRequest 检索上下文请求
type RetrieveContextRequest struct {
	SessionID     string `json:"sessionId"`
	Query         string `json:"query"`
	Limit         int    `json:"limit,omitempty"`
	Strategy      string `json:"strategy,omitempty"`      // balanced, recent, relevant
	MemoryID      string `json:"memoryId,omitempty"`      // 新增：通过记忆ID精确检索
	BatchID       string `json:"batchId,omitempty"`       // 新增：通过批次ID检索
	SkipThreshold bool   `json:"skipThreshold,omitempty"` // 新增：是否跳过相似度阈值过滤
	IsBruteSearch int    `json:"isBruteSearch,omitempty"` // 新增：是否启用暴力搜索（用于索引未训练的情况）
}

// ContextResponse 上下文响应
type ContextResponse struct {
	SessionState      string `json:"session_state"`
	ShortTermMemory   string `json:"short_term_memory"`
	LongTermMemory    string `json:"long_term_memory"`
	RelevantKnowledge string `json:"relevant_knowledge"`
}

// SummarizeContextRequest 生成上下文摘要请求
type SummarizeContextRequest struct {
	SessionID string `json:"sessionId"`
	Format    string `json:"format,omitempty"` // text, json
}

// SummarizeToLongTermRequest 汇总内容到长期记忆请求
type SummarizeToLongTermRequest struct {
	SessionID          string   `json:"sessionId"`
	CustomDescription  string   `json:"customDescription,omitempty"`  // 用户自定义的描述
	Tags               []string `json:"tags,omitempty"`               // 可选标签
	IncludeRecentEdits bool     `json:"includeRecentEdits,omitempty"` // 是否包含最近编辑
}

// Message 对话消息模型
type Message struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Role        string                 `json:"role"` // user, assistant, system
	Content     string                 `json:"content"`
	ContentType string                 `json:"content_type"` // text, code, image等
	Timestamp   int64                  `json:"timestamp"`
	Vector      []float32              `json:"vector,omitempty"`
	Priority    string                 `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage 创建新的消息
func NewMessage(sessionID, role, content, contentType string, priority string, metadata map[string]interface{}) *Message {
	if contentType == "" {
		contentType = "text" // 默认内容类型
	}

	if priority == "" {
		priority = "P2" // 默认优先级
	}

	return &Message{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Role:        role,
		Content:     content,
		ContentType: contentType,
		Timestamp:   time.Now().Unix(),
		Priority:    priority,
		Metadata:    metadata,
	}
}

// StoreMessagesRequest 存储对话消息请求
type StoreMessagesRequest struct {
	SessionID         string `json:"sessionId"`
	SummarizeAndStore bool   `json:"summarizeAndStore,omitempty"` // 是否同时汇总并存储
	BatchID           string `json:"batchId,omitempty"`           // 批次ID
	Messages          []struct {
		Role        string                 `json:"role"`
		Content     string                 `json:"content"`
		ContentType string                 `json:"contentType,omitempty"`
		Priority    string                 `json:"priority,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	} `json:"messages"`
}

// StoreMessagesResponse 存储对话消息响应
type StoreMessagesResponse struct {
	MessageIDs []string `json:"messageIds,omitempty"`
	MemoryID   string   `json:"memoryId,omitempty"` // 如果summarizeAndStore为true，返回的记忆ID
	Status     string   `json:"status"`
}

// RetrieveConversationRequest 检索对话请求
type RetrieveConversationRequest struct {
	SessionID     string `json:"sessionId"`
	Query         string `json:"query,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Format        string `json:"format,omitempty"`        // chronological, relevant
	MessageID     string `json:"messageId,omitempty"`     // 新增：通过消息ID精确检索
	BatchID       string `json:"batchId,omitempty"`       // 新增：通过批次ID检索
	SkipThreshold bool   `json:"skipThreshold,omitempty"` // 新增：是否跳过相似度阈值过滤
}

// ConversationResponse 对话响应
type ConversationResponse struct {
	SessionID       string          `json:"sessionId"`
	Messages        []*Message      `json:"messages"`
	RelevantIndices []int           `json:"relevantIndices,omitempty"` // 特别相关的消息索引
	SessionInfo     *SessionSummary `json:"sessionInfo"`
}

// SessionSummary 会话摘要
type SessionSummary struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	LastActive    time.Time `json:"lastActive"`
	MessageCount  int       `json:"messageCount"`
	UserMessages  int       `json:"userMessages"`
	AgentMessages int       `json:"agentMessages"`
	Summary       string    `json:"summary,omitempty"`
}

// 内部数据模型 -------------------------------------

// Memory 记忆实体
type Memory struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Vector    []float32              `json:"vector,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Priority  string                 `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// 新增字段，用于单独存储业务类型和用户ID
	BizType int    `json:"bizType,omitempty"`
	UserID  string `json:"userId,omitempty"`
}

// NewMemory 创建新的记忆实体
func NewMemory(sessionID, content string, priority string, metadata map[string]interface{}) *Memory {
	if priority == "" {
		priority = "P2" // 默认优先级
	}

	return &Memory{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Content:   content,
		Timestamp: time.Now().Unix(),
		Priority:  priority,
		Metadata:  metadata,
	}
}

// GenerateMemoryID 生成记忆ID
// 对于不提供memoryID的情况，使用自定义格式: memory-{date}-{time}
// 对于需要拆分的情况，可以使用 {memoryID}-{index} 的格式
func GenerateMemoryID(memoryID string) string {
	if memoryID != "" {
		return memoryID // 如果提供了memoryID，直接使用作为batchID
	}

	// 使用时间戳格式生成新ID
	now := time.Now()
	return fmt.Sprintf("memory-%s-%s",
		now.Format("20060102"),
		now.Format("150405"))
}

// Session 会话实体
type Session struct {
	ID         string                 `json:"id"`
	CreatedAt  time.Time              `json:"created_at"`
	LastActive time.Time              `json:"last_active"`
	Summary    string                 `json:"summary,omitempty"`
	Status     string                 `json:"status"` // active, archived
	Messages   []*Message             `json:"messages,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	// 新增会话管理字段
	ProjectInfo *ProjectInfo         `json:"project_info,omitempty"`
	EditHistory []*EditAction        `json:"edit_history,omitempty"`
	CodeContext map[string]*CodeFile `json:"code_context,omitempty"`
}

// MCP协议支持 ------------------------------------

// ProjectInfo 项目信息
type ProjectInfo struct {
	Name        string               `json:"name"`
	Language    string               `json:"language,omitempty"`
	Description string               `json:"description,omitempty"`
	Files       map[string]*CodeFile `json:"files,omitempty"`
}

// CodeFile 代码文件信息
type CodeFile struct {
	Path     string `json:"path"`
	Language string `json:"language,omitempty"`
	LastEdit int64  `json:"last_edit"`
	Summary  string `json:"summary,omitempty"`
}

// EditAction 编辑动作
type EditAction struct {
	ID          string                 `json:"id"` // 新增编辑ID
	Timestamp   int64                  `json:"timestamp"`
	FilePath    string                 `json:"file_path"`
	Type        string                 `json:"type"` // insert, delete, modify
	Position    int                    `json:"position"`
	Content     string                 `json:"content,omitempty"`
	DecisionIDs []string               `json:"decision_ids,omitempty"` // 关联的决策ID列表
	Tags        []string               `json:"tags,omitempty"`         // 编辑标签
	Metadata    map[string]interface{} `json:"metadata,omitempty"`     // 额外元数据
}

// MCPSessionRequest 会话状态请求
type MCPSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// MCPSessionResponse 会话状态响应
type MCPSessionResponse struct {
	SessionID    string    `json:"sessionId"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActive   time.Time `json:"lastActive"`
	Status       string    `json:"status"`
	MessageCount int       `json:"messageCount"`
}

// MCPCodeAssociationRequest 代码关联请求
type MCPCodeAssociationRequest struct {
	SessionID string `json:"sessionId"`
	FilePath  string `json:"filePath"`
	Language  string `json:"language"`
	Content   string `json:"content,omitempty"`
}

// MCPEditRecordRequest 编辑记录请求
type MCPEditRecordRequest struct {
	SessionID string `json:"sessionId"`
	FilePath  string `json:"filePath"`
	Type      string `json:"type"`
	Position  int    `json:"position"`
	Content   string `json:"content"`
}

// MCPResponse 通用MCP响应
type MCPResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// NewSession 创建新的会话
func NewSession(id string) *Session {
	now := time.Now()
	if id == "" {
		id = uuid.New().String()
	}

	return &Session{
		ID:          id,
		CreatedAt:   now,
		LastActive:  now,
		Status:      "active",
		Messages:    []*Message{},
		Metadata:    map[string]interface{}{},
		CodeContext: make(map[string]*CodeFile),
	}
}

// SearchResult 搜索结果
type SearchResult struct {
	ID     string                 `json:"id"`
	Score  float64                `json:"score"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// Metadata 元数据常量
const (
	MetadataTypeKey         = "type"
	MetadataTypeUser        = "user_message"
	MetadataTypeAssistant   = "assistant_message"
	MetadataTypeSystem      = "system_message"
	MetadataTypeCode        = "code"
	MetadataTypeRequirement = "requirement"
	MetadataTypeDecision    = "decision"
)

// Priority 优先级常量
const (
	PriorityP0 = "P0" // 关键信息，永久保留
	PriorityP1 = "P1" // 重要信息，长期保留
	PriorityP2 = "P2" // 一般信息，中期保留
	PriorityP3 = "P3" // 临时信息，短期保留
)

// Role 角色常量
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// 编辑类型常量
const (
	EditTypeInsert = "insert"
	EditTypeDelete = "delete"
	EditTypeModify = "modify"
)

// 会话状态常量
const (
	SessionStatusActive   = "active"
	SessionStatusInactive = "inactive"
	SessionStatusArchived = "archived"
)

// 新增决策与编辑关联相关的结构

// EditDecisionLink 编辑与决策关联
type EditDecisionLink struct {
	ID         string                 `json:"id"`
	EditID     string                 `json:"edit_id"`
	DecisionID string                 `json:"decision_id"`
	Timestamp  int64                  `json:"timestamp"`
	Strength   float64                `json:"strength"` // 关联强度 0.0-1.0
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// DecisionRecord 设计决策记录
type DecisionRecord struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"` // 架构、算法、接口等
	Timestamp    int64                  `json:"timestamp"`
	Vector       []float32              `json:"vector,omitempty"`
	RelatedEdits []string               `json:"related_edits,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Priority     string                 `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewDecisionRecord 创建新的决策记录
func NewDecisionRecord(sessionID, title, description, category string, priority string, metadata map[string]interface{}) *DecisionRecord {
	if priority == "" {
		priority = "P1" // 设计决策默认为高优先级
	}

	return &DecisionRecord{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		Title:        title,
		Description:  description,
		Category:     category,
		Timestamp:    time.Now().Unix(),
		Priority:     priority,
		Metadata:     metadata,
		RelatedEdits: []string{},
		Tags:         []string{},
	}
}

// SessionLink 会话间关联
type SessionLink struct {
	ID           string `json:"id"`
	SourceID     string `json:"source_id"`    // 源会话ID
	TargetID     string `json:"target_id"`    // 目标会话ID
	Relationship string `json:"relationship"` // 关系类型: continuation, reference, related
	Timestamp    int64  `json:"timestamp"`
	Description  string `json:"description,omitempty"`
}

// 关系类型常量
const (
	RelationshipContinuation = "continuation" // 会话的延续
	RelationshipReference    = "reference"    // 会话的引用
	RelationshipRelated      = "related"      // 相关会话
)

// LinkDecisionRequest 关联决策与编辑请求
type LinkDecisionRequest struct {
	SessionID   string   `json:"sessionId"`
	DecisionID  string   `json:"decisionId"`
	EditIDs     []string `json:"editIds"`
	Strength    float64  `json:"strength,omitempty"` // 关联强度 0.0-1.0，默认为1.0
	Description string   `json:"description,omitempty"`
}

// LinkDecisionResponse 关联决策响应
type LinkDecisionResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	LinkIDs []string `json:"linkIds,omitempty"`
}

// CreateDecisionRequest 创建设计决策请求
type CreateDecisionRequest struct {
	SessionID    string                 `json:"sessionId"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category,omitempty"`
	RelatedEdits []string               `json:"relatedEdits,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CreateDecisionResponse 创建设计决策响应
type CreateDecisionResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	DecisionID string `json:"decisionId"`
}

// SessionLinkRequest 会话关联请求
type SessionLinkRequest struct {
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	Relationship string `json:"relationship"`
	Description  string `json:"description,omitempty"`
}

// SessionLinkResponse 会话关联响应
type SessionLinkResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	LinkID  string `json:"linkId"`
}

// TodoItem 待办事项
type TodoItem struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Status      string                 `json:"status"` // pending, completed
	Priority    string                 `json:"priority"`
	CreatedAt   int64                  `json:"createdAt"`
	CompletedAt int64                  `json:"completedAt,omitempty"`
	UserID      string                 `json:"userId,omitempty"` // 非必须字段
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RetrieveTodosRequest 检索待办事项请求
type RetrieveTodosRequest struct {
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId,omitempty"` // 非必须参数
	Status    string `json:"status,omitempty"` // all, pending, completed
	Limit     int    `json:"limit,omitempty"`
}

// RetrieveTodosResponse 检索待办事项响应
type RetrieveTodosResponse struct {
	Items       []*TodoItem `json:"items"`
	Total       int         `json:"total"`
	Status      string      `json:"status"`
	UserID      string      `json:"userId,omitempty"` // 只有在有userId时返回
	Description string      `json:"description,omitempty"`
}

// UserConfig 用户配置
type UserConfig struct {
	UserID string `json:"userId"` // 用户唯一标识
}

// 业务类型常量
const (
	BizTypeTodo = 1 // 待办事项
)

// 本地存储指令模型 (第二期增强) -----------------------

// LocalInstructionType 本地存储指令类型
type LocalInstructionType string

const (
	LocalInstructionUserInit     LocalInstructionType = "user_init"     // 用户初始化
	LocalInstructionUserConfig   LocalInstructionType = "user_config"   // 用户配置更新
	LocalInstructionSessionStore LocalInstructionType = "session_store" // 会话存储
	LocalInstructionShortMemory  LocalInstructionType = "short_memory"  // 短期记忆存储
	LocalInstructionCodeContext  LocalInstructionType = "code_context"  // 代码上下文存储
	LocalInstructionPreferences  LocalInstructionType = "preferences"   // 个人设置存储
	LocalInstructionCacheUpdate  LocalInstructionType = "cache_update"  // 缓存更新
)

// LocalOperationOptions 本地操作选项
type LocalOperationOptions struct {
	CreateDir  bool `json:"createDir,omitempty"`  // 是否创建目录
	Backup     bool `json:"backup,omitempty"`     // 是否备份
	Merge      bool `json:"merge,omitempty"`      // 是否合并
	MaxAge     int  `json:"maxAge,omitempty"`     // 最大保留时间(秒)
	CleanupOld bool `json:"cleanupOld,omitempty"` // 是否清理旧数据
}

// LocalInstruction 本地存储指令
type LocalInstruction struct {
	Type       LocalInstructionType  `json:"type"`               // 指令类型
	Target     string                `json:"target"`             // 目标路径
	Content    interface{}           `json:"content"`            // 操作内容
	Options    LocalOperationOptions `json:"options,omitempty"`  // 操作选项
	CallbackID string                `json:"callbackId"`         // 回调ID
	Priority   string                `json:"priority,omitempty"` // 优先级 (low/normal/high)
}

// LocalCallbackRequest 本地操作回调请求
type LocalCallbackRequest struct {
	CallbackID string                 `json:"callbackId"`
	Success    bool                   `json:"success"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
}

// 扩展现有响应结构支持本地指令
type EnhancedResponse struct {
	// 原有响应字段
	Result  interface{} `json:"result,omitempty"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`

	// 新增本地操作指令字段 (可选)
	LocalInstruction *LocalInstruction `json:"localInstruction,omitempty"`
}

// 本地存储相关的标准路径常量 (统一使用macOS系统标准应用数据目录)
const (
	LocalPathUserConfig  = "~/Library/Application Support/context-keeper/user-config.json"
	LocalPathPreferences = "~/Library/Application Support/context-keeper/preferences.json"
	LocalPathSessions    = "~/Library/Application Support/context-keeper/users/{userId}/sessions/"
	LocalPathHistories   = "~/Library/Application Support/context-keeper/users/{userId}/histories/"
	LocalPathCodeContext = "~/Library/Application Support/context-keeper/users/{userId}/code_context/"
	LocalPathShortMemory = "~/Library/Application Support/context-keeper/users/{userId}/short_memory/"
	LocalPathCache       = "~/Library/Application Support/context-keeper/users/{userId}/cache/"
)

// 第一期兼容的本地存储数据结构

// LocalUserConfig 本地用户配置 (兼容第一期)
type LocalUserConfig struct {
	UserID    string `json:"userId"`    // 用户ID
	FirstUsed string `json:"firstUsed"` // 首次使用时间
}

// LocalSessionData 本地会话数据 (兼容第一期)
type LocalSessionData struct {
	ID          string                 `json:"id"`
	CreatedAt   time.Time              `json:"createdAt"`
	LastActive  time.Time              `json:"lastActive"`
	Status      string                 `json:"status"`
	Messages    []*Message             `json:"messages,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CodeContext map[string]*CodeFile   `json:"codeContext,omitempty"`
	EditHistory []*EditAction          `json:"editHistory,omitempty"`
}

// LocalHistoryData 本地历史记录数据 (兼容第一期)
type LocalHistoryData []string

// LocalCodeContextData 本地代码上下文数据 (兼容第一期)
type LocalCodeContextData map[string]*CodeFile

// LocalPreferencesData 本地偏好设置数据
type LocalPreferencesData struct {
	ShortTermRetention     int     `json:"shortTermRetention,omitempty"`     // 短期记忆保留天数
	AutoSummarizeThreshold int     `json:"autoSummarizeThreshold,omitempty"` // 自动摘要阈值
	SimilarityThreshold    float64 `json:"similarityThreshold,omitempty"`    // 相似度阈值
	EnableCrossSessions    bool    `json:"enableCrossSessions,omitempty"`    // 启用跨会话记忆
	DefaultMemoryPriority  string  `json:"defaultMemoryPriority,omitempty"`  // 默认记忆优先级
}

// LocalCacheData 本地缓存数据
type LocalCacheData struct {
	UserID        string                 `json:"userId,omitempty"`
	SessionStates map[string]interface{} `json:"sessionStates,omitempty"`
	LastUpdated   int64                  `json:"lastUpdated"`
}

// UserInfo 用户信息结构体
type UserInfo struct {
	UserID     string                 `json:"userId"`     // 用户唯一ID
	FirstUsed  string                 `json:"firstUsed"`  // 首次使用时间
	LastActive string                 `json:"lastActive"` // 最后活跃时间
	DeviceInfo map[string]interface{} `json:"deviceInfo"` // 设备信息
	Metadata   map[string]interface{} `json:"metadata"`   // 其他元数据
	CreatedAt  string                 `json:"createdAt"`  // 创建时间
	UpdatedAt  string                 `json:"updatedAt"`  // 更新时间
}

// UserRepository 用户信息存储接口
// 支持多种存储介质实现：阿里云向量存储、腾讯云向量存储、MySQL等
type UserRepository interface {
	// CreateUser 创建新用户
	// 返回错误如果用户已存在或创建失败
	CreateUser(userInfo *UserInfo) error

	// UpdateUser 更新用户信息
	// 返回错误如果用户不存在或更新失败
	UpdateUser(userInfo *UserInfo) error

	// GetUser 根据用户ID获取用户信息
	// 返回nil, nil如果用户不存在
	// 返回userInfo, nil如果找到用户
	// 返回nil, error如果查询失败
	GetUser(userID string) (*UserInfo, error)

	// CheckUserExists 检查用户是否存在
	// 返回true如果用户存在，false如果不存在
	// 返回error如果检查失败
	CheckUserExists(userID string) (bool, error)

	// InitRepository 初始化存储库（如创建表、集合等）
	InitRepository() error
}
