package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/contextkeeper/service/internal/agentic_beta"
	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/services"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/aliyun"
)

// 添加日志工具函数
// logToolCall 记录工具调用的详细日志
func logToolCall(name string, request map[string]interface{}, response interface{}, err error, duration time.Duration) {
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
	log.Printf("\n%s\n[工具调用: %s]\n%s", divider, name, divider)
	log.Printf("耗时: %v", duration)
	log.Printf("请求参数:\n%s", string(requestJSON))
	log.Printf("响应结果:\n%s", string(responseJSON))
	if err != nil {
		log.Printf("错误: %v", err)
	}
	log.Printf("%s\n[工具调用结束: %s]\n%s\n", divider, name, divider)
}

// initializeServices 初始化共享服务组件
// 🔥 修改：现在返回AgenticContextService以支持最新的智能功能
func initializeServices() (*agentic_beta.AgenticContextService, context.Context, context.CancelFunc) {
	// 加载环境变量和配置
	cfg := config.Load()
	log.Printf("加载配置: %s", cfg.String())

	// 验证关键配置
	embeddingAPIURL := getEnv("EMBEDDING_API_URL", cfg.EmbeddingAPIURL)
	embeddingAPIKey := getEnv("EMBEDDING_API_KEY", cfg.EmbeddingAPIKey)
	vectorDBURL := getEnv("VECTOR_DB_URL", cfg.VectorDBURL)
	vectorDBAPIKey := getEnv("VECTOR_DB_API_KEY", cfg.VectorDBAPIKey)

	// 检查是否在开发模式（HTTP模式允许演示运行）
	isHTTPMode := os.Getenv("HTTP_MODE") == "true" || os.Getenv("STREAMABLE_HTTP_MODE") == "true"

	if !isHTTPMode {
		// STDIO模式需要完整配置
		if embeddingAPIURL == "" {
			log.Fatalf("错误: EMBEDDING_API_URL 未设置")
		}
		if embeddingAPIKey == "" {
			log.Fatalf("错误: EMBEDDING_API_KEY 未设置")
		}
		if vectorDBURL == "" {
			log.Fatalf("错误: VECTOR_DB_URL 未设置")
		}
		if vectorDBAPIKey == "" {
			log.Fatalf("错误: VECTOR_DB_API_KEY 未设置")
		}
	} else {
		// HTTP模式警告但不退出
		if embeddingAPIURL == "" || embeddingAPIKey == "" || vectorDBURL == "" || vectorDBAPIKey == "" {
			log.Printf("警告: 缺少必需的API配置，部分功能可能不可用")
			log.Printf("请设置以下环境变量以获得完整功能:")
			if embeddingAPIURL == "" {
				log.Printf("  - EMBEDDING_API_URL")
			}
			if embeddingAPIKey == "" {
				log.Printf("  - EMBEDDING_API_KEY")
			}
			if vectorDBURL == "" {
				log.Printf("  - VECTOR_DB_URL")
			}
			if vectorDBAPIKey == "" {
				log.Printf("  - VECTOR_DB_API_KEY")
			}
		}
	}

	// 配置
	storagePath := getEnv("STORAGE_PATH", cfg.StoragePath)
	if storagePath == "" {
		log.Fatalf("错误: STORAGE_PATH 未设置")
	}

	// 检查存储路径是否为临时目录，如果是则替换为标准路径
	if strings.Contains(storagePath, "/tmp/") || strings.Contains(storagePath, "/temp/") ||
		strings.Contains(storagePath, "\\Temp\\") {
		log.Printf("警告: 存储路径位于临时目录: %s", storagePath)
		log.Printf("将使用操作系统标准应用数据目录代替")

		// 使用配置中的标准路径
		storagePath = cfg.StoragePath
		log.Printf("新的存储路径: %s", storagePath)
	}

	// 其他阿里云参数
	vectorDBCollection := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
	vectorDBDimension := getIntEnv("VECTOR_DB_DIMENSION", cfg.VectorDBDimension)
	vectorDBMetric := getEnv("VECTOR_DB_METRIC", cfg.VectorDBMetric)
	similarityThreshold := getFloatEnv("SIMILARITY_THRESHOLD", cfg.SimilarityThreshold)

	// 创建向量服务
	var vectorService *aliyun.VectorService
	if embeddingAPIURL != "" && embeddingAPIKey != "" && vectorDBURL != "" && vectorDBAPIKey != "" {
		vectorService = aliyun.NewVectorService(
			embeddingAPIURL,
			embeddingAPIKey,
			vectorDBURL,
			vectorDBAPIKey,
			vectorDBCollection,
			vectorDBDimension,
			vectorDBMetric,
			similarityThreshold,
		)

		// 确保向量集合存在
		log.Println("确保向量集合存在...")
		err := vectorService.EnsureCollection()
		if err != nil {
			if isHTTPMode {
				log.Printf("警告: 向量集合初始化失败: %v (HTTP模式继续运行)", err)
			} else {
				log.Fatalf("向量集合初始化失败: %v", err)
			}
		}
	} else {
		log.Printf("警告: 向量服务配置不完整，将使用模拟模式")
	}

	// 初始化会话存储
	log.Println("初始化会话存储...")
	ensureDirExists(storagePath)

	// 检查是否为HTTP模式（已在上面定义过了）

	var sessionStore *store.SessionStore
	var err error

	if isHTTPMode {
		log.Println("HTTP模式：初始化用户隔离的存储系统")
		// HTTP模式需要确保用户隔离，仍然需要SessionStore但存储路径结构不同
		sessionStore, err = store.NewSessionStore(storagePath)
		if err != nil {
			log.Fatalf("初始化会话存储失败: %v", err)
		}
	} else {
		log.Println("STDIO模式：使用直接SessionStore")
		sessionStore, err = store.NewSessionStore(storagePath)
		if err != nil {
			log.Fatalf("初始化会话存储失败: %v", err)
		}
	}

	// 初始化用户缓存
	log.Println("初始化用户缓存...")
	err = utils.InitUserCache()
	if err != nil {
		log.Printf("警告: 初始化用户缓存失败: %v, 将在首次对话时进行初始化", err)
	} else {
		userID := utils.GetCachedUserID()
		if userID != "" {
			log.Printf("已加载用户配置, ID: %s", userID)
		} else {
			log.Printf("未找到有效的用户配置，将在首次对话时进行初始化")
		}
	}

	// 🔥 修改：初始化Agentic智能上下文服务 - 直接基于ContextService
	log.Println("初始化Agentic智能上下文服务...")

	// 创建基础的ContextService
	originalContextService := services.NewContextService(vectorService, sessionStore, cfg)

	// 🔥 重构：直接基于ContextService创建完整的AgenticContextService
	// 集成智能查询优化、意图分析和决策中心等所有功能
	agenticContextService := agentic_beta.NewAgenticContextService(originalContextService)
	log.Printf("🚀 AgenticContextService v2.0 初始化完成，完整智能功能已启用")
	log.Printf("📋 统一服务架构:")
	log.Printf("  🏗️ ContextService (基础服务)")
	log.Printf("  🤖 AgenticContextService (完整智能解决方案)")
	log.Printf("    ├── 智能查询优化 (查询改写、噪声过滤、上下文丰富)")
	log.Printf("    ├── 意图分析器 (自动识别查询意图和领域)")
	log.Printf("    └── 决策中心 (基于意图制定处理策略)")

	// 创建会话清理的上下文
	cleanupCtx, cancelCleanup := context.WithCancel(context.Background())

	// 启动会话清理任务，使用配置文件中的时间设置
	log.Printf("启动会话清理任务: 超时=%v, 间隔=%v", cfg.SessionTimeout, cfg.CleanupInterval)
	agenticContextService.StartSessionCleanupTask(cleanupCtx, cfg.SessionTimeout, cfg.CleanupInterval)

	// 🔥 修改：返回完整的AgenticContextService，提供最完整的智能功能
	// AgenticContextService通过代理模式完全兼容ContextService的所有方法
	return agenticContextService, cleanupCtx, cancelCleanup
}

// registerMCPTools 注册所有MCP工具到服务器
func registerMCPTools(s *server.MCPServer, contextService *services.ContextService) {
	// 注册工具：关联文件
	associateFileTool := mcp.NewTool("associate_file",
		mcp.WithDescription("关联代码文件到当前编程会话"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("filePath",
			mcp.Required(),
			mcp.Description("文件路径"),
		),
	)
	s.AddTool(associateFileTool, associateFileHandler(contextService))

	// 注册工具：记录编辑
	recordEditTool := mcp.NewTool("record_edit",
		mcp.WithDescription("记录代码编辑操作"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("filePath",
			mcp.Required(),
			mcp.Description("文件路径"),
		),
		mcp.WithString("diff",
			mcp.Required(),
			mcp.Description("编辑差异内容"),
		),
	)
	s.AddTool(recordEditTool, recordEditHandler(contextService))

	// 注册工具：检索上下文
	retrieveContextTool := mcp.NewTool("retrieve_context",
		mcp.WithDescription("基于查询检索相关编程上下文"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("查询内容"),
		),
	)
	s.AddTool(retrieveContextTool, retrieveContextHandler(contextService))

	// 注册工具：编程上下文
	programmingContextTool := mcp.NewTool("programming_context",
		mcp.WithDescription("获取编程特征和上下文摘要"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("query",
			mcp.Description("可选查询参数"),
		),
	)
	s.AddTool(programmingContextTool, programmingContextHandler(contextService))

	// 注册工具：会话管理
	sessionManagementTool := mcp.NewTool("session_management",
		mcp.WithDescription("创建或获取会话信息"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("操作类型: get_or_create"),
		),
		mcp.WithString("userId",
			mcp.Required(),
			mcp.Description("用户ID，必需参数。客户端必须从配置文件获取：macOS: ~/Library/Application Support/context-keeper/user-config.json, Windows: ~/AppData/Roaming/context-keeper/user-config.json, Linux: ~/.local/share/context-keeper/user-config.json"),
		),
		mcp.WithString("workspaceRoot",
			mcp.Required(),
			mcp.Description("工作空间根路径，必需参数，用于会话隔离，确保不同工作空间的session完全独立"),
		),
		mcp.WithObject("metadata",
			mcp.Description("会话元数据，可选"),
		),
	)
	s.AddTool(sessionManagementTool, sessionManagementHandler(contextService))

	// 注册工具：存储对话
	storeConversationTool := mcp.NewTool("store_conversation",
		mcp.WithDescription("存储并总结当前对话内容到短期记忆"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithArray("messages",
			mcp.Required(),
			mcp.Description("对话消息列表"),
		),
		mcp.WithString("batchId",
			mcp.Description("批次ID，可选，不提供则自动生成"),
		),
	)
	s.AddTool(storeConversationTool, storeConversationHandler(contextService))

	// 注册工具：检索记忆
	retrieveMemoryTool := mcp.NewTool("retrieve_memory",
		mcp.WithDescription("基于memoryId或batchId检索历史对话"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("memoryId",
			mcp.Description("记忆ID"),
		),
		mcp.WithString("batchId",
			mcp.Description("批次ID"),
		),
		mcp.WithString("format",
			mcp.Description("返回格式: full, summary"),
		),
	)
	s.AddTool(retrieveMemoryTool, retrieveMemoryHandler(contextService))

	// 注册工具：记忆化上下文
	memorizeContextTool := mcp.NewTool("memorize_context",
		mcp.WithDescription("将重要内容汇总并存储到长期记忆"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("要记忆的内容"),
		),
		mcp.WithString("priority",
			mcp.Description("优先级，可选: P1(高), P2(中), P3(低)，默认P2"),
		),
		mcp.WithObject("metadata",
			mcp.Description("记忆相关的元数据，可选"),
		),
	)
	s.AddTool(memorizeContextTool, memorizeContextHandler(contextService))

	// 注册工具：检索待办事项
	retrieveTodosTool := mcp.NewTool("retrieve_todos",
		mcp.WithDescription("获取我的待办事项列表"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("status",
			mcp.Description("筛选状态: all, pending, completed"),
		),
		mcp.WithString("limit",
			mcp.Description("返回结果数量限制"),
		),
	)
	s.AddTool(retrieveTodosTool, retrieveTodosHandler(contextService))

	// 注册工具：用户初始化对话
	userInitDialogTool := mcp.NewTool("user_init_dialog",
		mcp.WithDescription("用户初始化对话处理"),
		mcp.WithString("sessionId",
			mcp.Required(),
			mcp.Description("当前会话ID"),
		),
		mcp.WithString("userResponse",
			mcp.Description("用户对初始化提示的响应"),
		),
	)
	s.AddTool(userInitDialogTool, userInitDialogHandler())
}

// 工具处理函数

// associateFileHandler 处理文件关联请求
func associateFileHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("associate_file", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		filePath, ok := request.Params.Arguments["filePath"].(string)
		if !ok || filePath == "" {
			errMsg := "错误: filePath必须是非空字符串"
			log.Println(errMsg)
			logToolCall("associate_file", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("关联文件: sessionID=%s, filePath=%s", sessionID, filePath)

		err := contextService.AssociateFile(ctx, models.AssociateFileRequest{
			SessionID: sessionID,
			FilePath:  filePath,
		})
		if err != nil {
			errMsg := fmt.Sprintf("关联文件失败: %v", err)
			log.Println(errMsg)
			logToolCall("associate_file", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		successMsg := fmt.Sprintf("成功关联文件: %s", filePath)
		log.Println(successMsg)
		logToolCall("associate_file", request.Params.Arguments, successMsg, nil, time.Since(startTime))
		return mcp.NewToolResultText(successMsg), nil
	}
}

// recordEditHandler 处理编辑记录请求
func recordEditHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("record_edit", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		filePath, ok := request.Params.Arguments["filePath"].(string)
		if !ok || filePath == "" {
			errMsg := "错误: filePath必须是非空字符串"
			log.Println(errMsg)
			logToolCall("record_edit", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		diff, ok := request.Params.Arguments["diff"].(string)
		if !ok {
			errMsg := "错误: diff必须是字符串"
			log.Println(errMsg)
			logToolCall("record_edit", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("记录编辑: sessionID=%s, filePath=%s, diff长度=%d", sessionID, filePath, len(diff))

		err := contextService.RecordEdit(ctx, models.RecordEditRequest{
			SessionID: sessionID,
			FilePath:  filePath,
			Diff:      diff,
		})
		if err != nil {
			errMsg := fmt.Sprintf("记录编辑失败: %v", err)
			log.Println(errMsg)
			logToolCall("record_edit", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		successMsg := "成功记录编辑操作"
		log.Println(successMsg)
		logToolCall("record_edit", request.Params.Arguments, successMsg, nil, time.Since(startTime))
		return mcp.NewToolResultText(successMsg), nil
	}
}

// retrieveContextHandler 处理上下文检索请求
func retrieveContextHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("retrieve_context", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		query, ok := request.Params.Arguments["query"].(string)
		if !ok {
			errMsg := "错误: query必须是字符串"
			log.Println(errMsg)
			logToolCall("retrieve_context", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 检查是否启用暴力搜索
		isBruteSearch := 0 // 默认值为0
		if bruteSearchVal, ok := request.Params.Arguments["isBruteSearch"]; ok {
			if bruteSearchFloat, ok := bruteSearchVal.(float64); ok {
				isBruteSearch = int(bruteSearchFloat)
			} else if bruteSearchInt, ok := bruteSearchVal.(int); ok {
				isBruteSearch = bruteSearchInt
			}
		}

		log.Printf("检索上下文: sessionID=%s, query=%s, isBruteSearch=%d", sessionID, query, isBruteSearch)

		result, err := contextService.RetrieveContext(ctx, models.RetrieveContextRequest{
			SessionID:     sessionID,
			Query:         query,
			IsBruteSearch: isBruteSearch, // 传递暴力搜索参数
		})
		if err != nil {
			errMsg := fmt.Sprintf("检索上下文失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 使用json.Marshal正确序列化结果
		jsonData, err := json.Marshal(result)
		if err != nil {
			errMsg := fmt.Sprintf("序列化结果失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("检索上下文成功: 结果长度=%d字节", len(jsonData))
		logToolCall("retrieve_context", request.Params.Arguments, string(jsonData), nil, time.Since(startTime))
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// programmingContextHandler 处理编程上下文摘要请求
func programmingContextHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("programming_context", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 处理可选参数
		var query string
		if queryVal, ok := request.Params.Arguments["query"]; ok && queryVal != nil {
			query, ok = queryVal.(string)
			if !ok {
				query = ""
				log.Println("警告: query参数类型不是字符串，已设为空字符串")
			}
		}

		log.Printf("获取编程上下文: sessionID=%s, query=%s", sessionID, query)

		// 使用GetProgrammingContext方法获取编程上下文
		result, err := contextService.GetProgrammingContext(ctx, sessionID, query)
		if err != nil {
			errMsg := fmt.Sprintf("获取编程上下文失败: %v", err)
			log.Println(errMsg)
			logToolCall("programming_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 使用json.Marshal正确序列化结果
		jsonData, err := json.Marshal(result)
		if err != nil {
			errMsg := fmt.Sprintf("序列化结果失败: %v", err)
			log.Println(errMsg)
			logToolCall("programming_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("获取编程上下文成功: 结果长度=%d字节", len(jsonData))
		logToolCall("programming_context", request.Params.Arguments, string(jsonData), nil, time.Since(startTime))
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// sessionManagementHandler 处理会话管理请求
func sessionManagementHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		action, ok := request.Params.Arguments["action"].(string)
		if !ok || action == "" {
			errMsg := "错误: action必须是非空字符串"
			log.Println(errMsg)
			logToolCall("session_management", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		sessionID, _ := request.Params.Arguments["sessionId"].(string)

		// 获取用户ID参数
		userID, _ := request.Params.Arguments["userId"].(string)
		if userID == "" {
			// 尝试从上下文获取
			userID = utils.GetCachedUserID()
			log.Printf("🔍 [会话管理] 从缓存获取userID: %s", userID)
		}

		// 获取元数据
		metadataRaw, hasMetadata := request.Params.Arguments["metadata"]
		var metadata map[string]interface{}
		if hasMetadata {
			metadata, _ = metadataRaw.(map[string]interface{})
			log.Printf("🔍 [会话管理] 解析元数据成功，键数量: %d", len(metadata))
			for key, value := range metadata {
				log.Printf("🔍 [会话管理] 元数据 %s: %+v (类型: %T)", key, value, value)
			}
		} else {
			log.Printf("🔍 [会话管理] 未提供元数据")
		}

		log.Printf("会话管理: action=%s, sessionID=%s, userID=%s", action, sessionID, userID)

		// 🔐 强制使用用户隔离的会话存储，避免数据泄露
		var sessionStore *store.SessionStore
		var err error

		// 🔐 严格按照一期stdio协议：获取用户ID并检查是否需要初始化
		if userID == "" {
			var needUserInit bool
			var err error
			userID, needUserInit, err = utils.GetUserID()
			if err != nil {
				log.Printf("[会话管理] 获取用户ID失败: %v", err)
				// 错误情况下记录但继续处理
			}

			// 严格按照一期逻辑：如果需要用户初始化，拒绝操作并返回初始化提示
			if needUserInit || userID == "" {
				log.Printf("[会话管理] 用户未初始化，拒绝操作")
				result := map[string]interface{}{
					"needUserInit": true,
					"initPrompt":   "需要进行用户初始化才能将记忆与您的个人账户关联。请完成用户初始化流程。",
					"status":       "error",
					"message":      "操作被拒绝：请先完成用户初始化",
				}
				jsonData, _ := json.Marshal(result)
				responseStr := string(jsonData)
				log.Println("[会话管理] 返回用户初始化需求: " + responseStr)
				logToolCall("session_management", request.Params.Arguments, responseStr, nil, time.Since(startTime))
				return mcp.NewToolResultText(responseStr), nil
			}
		}

		// 使用用户专属会话存储
		sessionStore, err = contextService.GetUserSessionStore(userID)
		if err != nil {
			errMsg := fmt.Sprintf("获取用户会话存储失败: %v", err)
			log.Println(errMsg)
			logToolCall("session_management", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		switch action {
		case "get_or_create":
			log.Printf("🔍 [会话管理] === 处理get_or_create操作 ===")

			// 🔥 新增：获取或创建会话，基于用户ID和工作空间哈希
			// 获取工作空间哈希参数
			workspaceHash, _ := request.Params.Arguments["workspaceHash"].(string)
			log.Printf("🔍 [会话管理] 步骤1 - 从参数获取workspaceHash: '%s'", workspaceHash)

			// 🔥 修复：从顶级参数中获取工作空间路径
			workspacePath, _ := request.Params.Arguments["workspaceRoot"].(string)
			log.Printf("🔍 [会话管理] 步骤2 - 从元数据获取工作空间路径: '%s'", workspacePath)

			log.Printf("🔍 [会话管理] 步骤3 - 最终workspaceHash: '%s'", workspaceHash)
			log.Printf("🔍 [会话管理] 步骤4 - 准备调用GetWorkspaceSessionID，参数: userID=%s, sessionID=%s, workspaceHash=%s", userID, sessionID, workspaceHash)

			// 使用统一的工具函数获取会话
			sessionTimeout := 30 * time.Minute // 30分钟会话超时
			session, isNewSession, err := utils.GetWorkspaceSessionID(sessionStore, userID, sessionID, workspacePath, metadata, sessionTimeout)
			if err != nil {
				errMsg := fmt.Sprintf("获取或创建会话失败: %v", err)
				log.Printf("🔍 [会话管理] 步骤5 - GetWorkspaceSessionID失败: %v", err)
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, err, time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			log.Printf("🔍 [会话管理] 步骤5 - GetWorkspaceSessionID成功: sessionID=%s, isNew=%t", session.ID, isNewSession)

			// 检查会话的工作空间哈希
			sessionWorkspaceHash := ""
			if session.Metadata != nil {
				if hash, ok := session.Metadata["workspaceHash"].(string); ok {
					sessionWorkspaceHash = hash
				}
			}
			log.Printf("🔍 [会话管理] 步骤6 - 会话实际workspaceHash: '%s'", sessionWorkspaceHash)

			result := map[string]interface{}{
				"sessionId":     session.ID,
				"created":       session.CreatedAt,
				"status":        "active",
				"isNewSession":  isNewSession,
				"lastActive":    session.LastActive,
				"userID":        userID,
				"workspaceHash": workspaceHash,
			}

			jsonData, _ := json.Marshal(result)
			successMsg := string(jsonData)
			log.Println("[会话管理] 获取或创建会话成功: " + successMsg)
			logToolCall("session_management", request.Params.Arguments, successMsg, nil, time.Since(startTime))
			return mcp.NewToolResultText(successMsg), nil

		case "get":
			if sessionID == "" {
				errMsg := "错误: 获取会话时sessionId不能为空"
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			session, err := sessionStore.GetSession(sessionID)
			if err != nil {
				errMsg := fmt.Sprintf("获取会话失败: %v", err)
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, err, time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			// 构建会话信息响应
			sessionInfo := map[string]interface{}{
				"sessionId":   session.ID,
				"created":     session.CreatedAt,
				"lastActive":  session.LastActive,
				"status":      session.Status,
				"metadata":    session.Metadata,
				"summary":     session.Summary,
				"codeContext": make(map[string]interface{}),
			}

			// 添加代码文件信息
			if session.CodeContext != nil {
				for path, file := range session.CodeContext {
					sessionInfo["codeContext"].(map[string]interface{})[path] = map[string]interface{}{
						"language": file.Language,
						"lastEdit": file.LastEdit,
						"summary":  file.Summary,
					}
				}
			}

			// 获取关联的记忆统计
			countStats, _ := contextService.CountSessionMemories(ctx, sessionID)
			if countStats != nil {
				sessionInfo["memories"] = countStats
			}

			jsonData, _ := json.Marshal(sessionInfo)
			responseStr := string(jsonData)
			logToolCall("session_management", request.Params.Arguments, responseStr, nil, time.Since(startTime))
			return mcp.NewToolResultText(responseStr), nil

		case "update":
			if sessionID == "" {
				errMsg := "错误: 更新会话时sessionId不能为空"
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			session, err := sessionStore.GetSession(sessionID)
			if err != nil {
				errMsg := fmt.Sprintf("获取会话失败: %v", err)
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, err, time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			// 如果有用户ID，添加到元数据
			if userID != "" {
				if metadata == nil {
					metadata = make(map[string]interface{})
				}
				metadata["userId"] = userID
			}

			// 更新元数据
			if metadata != nil {
				if session.Metadata == nil {
					session.Metadata = metadata
				} else {
					for k, v := range metadata {
						session.Metadata[k] = v
					}
				}
			}

			// 更新最后活动时间
			session.LastActive = time.Now()

			// 保存会话
			if err := sessionStore.SaveSession(session); err != nil {
				errMsg := fmt.Sprintf("更新会话失败: %v", err)
				log.Println(errMsg)
				logToolCall("session_management", request.Params.Arguments, errMsg, err, time.Since(startTime))
				return mcp.NewToolResultText(errMsg), nil
			}

			responseStr := fmt.Sprintf("{\"status\":\"success\",\"sessionId\":\"%s\"}", sessionID)
			logToolCall("session_management", request.Params.Arguments, responseStr, nil, time.Since(startTime))
			return mcp.NewToolResultText(responseStr), nil

		case "list":
			// 获取会话列表
			var sessions []*models.Session

			// 默认只获取活跃会话
			onlyActive := true
			if onlyActiveVal, ok := request.Params.Arguments["onlyActive"].(bool); ok {
				onlyActive = onlyActiveVal
			}

			sessions = sessionStore.GetSessionList()

			// 构建响应列表
			responseList := make([]map[string]interface{}, 0)
			for _, session := range sessions {
				// 如果需要过滤活跃状态
				if onlyActive && session.Status != models.SessionStatusActive {
					continue
				}

				sessionInfo := map[string]interface{}{
					"sessionId":  session.ID,
					"created":    session.CreatedAt,
					"lastActive": session.LastActive,
					"status":     session.Status,
					"summary":    session.Summary,
				}
				responseList = append(responseList, sessionInfo)
			}

			// 按最后活跃时间排序
			sort.Slice(responseList, func(i, j int) bool {
				iTime, _ := responseList[i]["lastActive"].(time.Time)
				jTime, _ := responseList[j]["lastActive"].(time.Time)
				return iTime.After(jTime)
			})

			jsonData, _ := json.Marshal(responseList)
			logToolCall("session_management", request.Params.Arguments, string(jsonData), nil, time.Since(startTime))
			return mcp.NewToolResultText(string(jsonData)), nil

		default:
			errMsg := fmt.Sprintf("错误: 不支持的操作类型: %s", action)
			log.Println(errMsg)
			logToolCall("session_management", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}
	}
}

// storeConversationHandler 处理对话存储请求
func storeConversationHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("store_conversation", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		messagesRaw, ok := request.Params.Arguments["messages"]
		if !ok {
			errMsg := "错误: messages参数必须提供"
			log.Println(errMsg)
			logToolCall("store_conversation", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		messagesArray, ok := messagesRaw.([]interface{})
		if !ok {
			errMsg := "错误: messages必须是数组"
			log.Println(errMsg)
			logToolCall("store_conversation", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 可选的批次ID
		batchID, _ := request.Params.Arguments["batchId"].(string)

		// 如果未提供batchID，生成一个新的memoryId作为batchId
		// memoryId格式为UUID，如果有需要拆分，可以按"memoryId-1", "memoryId-2"等格式拆分
		if batchID == "" {
			memoryID := "" // 不提供memoryID，让GenerateMemoryID自动生成
			batchID = models.GenerateMemoryID(memoryID)
			log.Printf("[对话存储] 生成新的batchId: %s", batchID)
		}

		log.Printf("存储对话: sessionID=%s, 消息数量=%d, batchID=%s",
			sessionID, len(messagesArray), batchID)

		// 构建消息列表
		var messages []*models.Message
		for _, msgRaw := range messagesArray {
			msgMap, ok := msgRaw.(map[string]interface{})
			if !ok {
				continue
			}

			role, _ := msgMap["role"].(string)
			content, _ := msgMap["content"].(string)

			if role == "" || content == "" {
				continue
			}

			// 创建元数据，将batchId作为主要标识
			metadata := map[string]interface{}{
				"batchId":   batchID,
				"timestamp": time.Now().Unix(),
				"type":      "conversation_message",
			}

			// 创建消息对象，元数据中包含batchId，用于向量存储时作为ID
			message := models.NewMessage(
				sessionID,
				role,
				content,
				"text",
				"P2", // 使用默认优先级
				metadata,
			)

			messages = append(messages, message)
		}

		if len(messages) == 0 {
			errMsg := "错误: 没有有效的消息可存储"
			log.Println(errMsg)
			logToolCall("store_conversation", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 构建消息请求
		msgReqs := make([]struct {
			Role        string                 `json:"role"`
			Content     string                 `json:"content"`
			ContentType string                 `json:"contentType,omitempty"`
			Priority    string                 `json:"priority,omitempty"`
			Metadata    map[string]interface{} `json:"metadata,omitempty"`
		}, len(messages))

		for i, msg := range messages {
			msgReqs[i] = struct {
				Role        string                 `json:"role"`
				Content     string                 `json:"content"`
				ContentType string                 `json:"contentType,omitempty"`
				Priority    string                 `json:"priority,omitempty"`
				Metadata    map[string]interface{} `json:"metadata,omitempty"`
			}{
				Role:        msg.Role,
				Content:     msg.Content,
				ContentType: msg.ContentType,
				Priority:    msg.Priority,
				Metadata:    msg.Metadata,
			}
		}

		// 存储消息到短期记忆
		resp, err := contextService.StoreSessionMessages(ctx, models.StoreMessagesRequest{
			SessionID: sessionID,
			BatchID:   batchID,
			Messages:  msgReqs,
		})

		if err != nil {
			errMsg := fmt.Sprintf("存储对话到短期记忆失败: %v", err)
			log.Println(errMsg)
			logToolCall("store_conversation", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 生成对话摘要
		summary, err := contextService.SummarizeContext(ctx, models.SummarizeContextRequest{
			SessionID: sessionID,
			Format:    "text",
		})

		// 构建响应
		result := map[string]interface{}{
			"status":     "success",
			"batchId":    batchID,
			"messageIds": resp.MessageIDs,
			"summary":    summary,
		}

		// 获取用户ID用于WebSocket推送
		userID, _, err := utils.GetUserID()
		if err == nil && userID != "" {
			// 构建本地指令
			localInstruction := map[string]interface{}{
				"type":    "short_memory",
				"target":  fmt.Sprintf("~/Library/Application Support/context-keeper/users/%s/histories/%s.json", userID, sessionID),
				"content": msgReqs,
				"options": map[string]interface{}{
					"createDir":  true,
					"merge":      true,
					"maxAge":     604800, // 7天
					"cleanupOld": true,
				},
				"callbackId": fmt.Sprintf("short_memory_%s_%d", sessionID, time.Now().UnixNano()),
				"priority":   "normal",
			}

			// 尝试通过WebSocket推送本地指令
			// 注意：这里我们需要导入WebSocket管理器
			// 推送失败不影响MCP响应的正常返回
			result["localInstruction"] = localInstruction

			log.Printf("[WebSocket] 准备推送本地指令到用户: %s", userID)
			// TODO: 这里需要调用WebSocket推送逻辑

			// 尝试通过WebSocket推送指令
			if services.GlobalWSManager != nil {
				instruction := models.LocalInstruction{
					Type:    models.LocalInstructionType(localInstruction["type"].(string)),
					Target:  localInstruction["target"].(string),
					Content: localInstruction["content"],
					Options: models.LocalOperationOptions{
						CreateDir:  true,
						Merge:      true,
						MaxAge:     604800,
						CleanupOld: true,
					},
					CallbackID: localInstruction["callbackId"].(string),
					Priority:   localInstruction["priority"].(string),
				}

				// 🔥 精确推送：优先使用基于sessionId的精确推送
				var callbackChan chan models.CallbackResult
				if sessionChan, sessionErr := services.GlobalWSManager.PushInstructionToSession(sessionID, instruction); sessionErr == nil {
					callbackChan = sessionChan
					log.Printf("[WebSocket] 本地指令已精确推送到会话 %s: %s", sessionID, instruction.CallbackID)
				} else {
					log.Printf("[WebSocket] 精确推送失败 (会话 %s 未注册)，回退到用户级别推送: %v", sessionID, sessionErr)
					// 回退到传统的用户级别推送
					if fallbackChan, fallbackErr := services.GlobalWSManager.PushInstruction(userID, instruction); fallbackErr == nil {
						callbackChan = fallbackChan
						log.Printf("[WebSocket] 回退推送成功: %s", instruction.CallbackID)
					} else {
						log.Printf("[WebSocket] 回退推送也失败: %v", fallbackErr)
					}
				}

				if callbackChan != nil {
					log.Printf("[WebSocket] 本地指令已推送: %s", instruction.CallbackID)

					// 异步等待回调结果（不阻塞MCP响应）
					go func() {
						select {
						case callbackResult := <-callbackChan:
							log.Printf("[WebSocket] 本地指令执行完成: %s - %s", instruction.CallbackID, callbackResult.Message)
						case <-time.After(30 * time.Second):
							log.Printf("[WebSocket] 本地指令执行超时: %s", instruction.CallbackID)
						}
					}()
				} else {
					log.Printf("[WebSocket] 推送指令失败: %v, 用户可能未连接WebSocket", err)
				}
			}
		}

		jsonData, _ := json.Marshal(result)
		responseStr := string(jsonData)
		logToolCall("store_conversation", request.Params.Arguments, responseStr, nil, time.Since(startTime))
		return mcp.NewToolResultText(responseStr), nil
	}
}

// retrieveMemoryHandler 处理记忆检索请求
func retrieveMemoryHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("retrieve_memory", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		memoryID, _ := request.Params.Arguments["memoryId"].(string)
		batchID, _ := request.Params.Arguments["batchId"].(string)

		// 检查是否有id参数，如果有则优先使用id作为batchId
		if id, ok := request.Params.Arguments["id"].(string); ok && id != "" {
			batchID = id
			log.Printf("发现id参数，将其用作batchId: %s", id)
		}

		format, _ := request.Params.Arguments["format"].(string)

		if memoryID == "" && batchID == "" {
			errMsg := "错误: 必须至少提供memoryId或batchId之一"
			log.Println(errMsg)
			logToolCall("retrieve_memory", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("检索记忆: sessionID=%s, memoryID=%s, batchID=%s, format=%s",
			sessionID, memoryID, batchID, format)

		// 创建检索请求
		req := models.RetrieveContextRequest{
			SessionID:     sessionID,
			MemoryID:      memoryID,
			BatchID:       batchID,
			SkipThreshold: true, // 对精确ID检索跳过相似度过滤
		}

		// 执行检索
		result, err := contextService.RetrieveContext(ctx, req)
		if err != nil {
			errMsg := fmt.Sprintf("检索记忆失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_memory", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 根据格式选择返回方式
		if format == "summary" {
			// 返回简洁摘要
			summary := map[string]interface{}{
				"sessionId":     sessionID,
				"sessionState":  result.SessionState,
				"shortSummary":  getSummaryFromResult(result.ShortTermMemory),
				"memoryCount":   countMemories(result),
				"relevantCount": countRelevantMemories(result),
			}

			jsonData, _ := json.Marshal(summary)
			responseStr := string(jsonData)
			logToolCall("retrieve_memory", request.Params.Arguments, responseStr, nil, time.Since(startTime))
			return mcp.NewToolResultText(responseStr), nil
		}

		// 返回完整结果
		jsonData, err := json.Marshal(result)
		if err != nil {
			errMsg := fmt.Sprintf("序列化结果失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_memory", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("检索记忆成功: 结果长度=%d字节", len(jsonData))
		responseStr := string(jsonData)
		logToolCall("retrieve_memory", request.Params.Arguments, responseStr, nil, time.Since(startTime))
		return mcp.NewToolResultText(responseStr), nil
	}
}

// memorizeContextHandler 处理长期记忆存储请求
func memorizeContextHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("memorize_context", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		content, ok := request.Params.Arguments["content"].(string)
		if !ok || content == "" {
			errMsg := "错误: content必须是非空字符串"
			log.Println(errMsg)
			logToolCall("memorize_context", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 可选参数
		priority, _ := request.Params.Arguments["priority"].(string)
		if priority == "" {
			priority = "P2" // 默认中等优先级
		}

		// 处理元数据
		metadata := make(map[string]interface{})
		if metadataRaw, ok := request.Params.Arguments["metadata"]; ok {
			if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
				for k, v := range metadataMap {
					metadata[k] = v
				}
			}
		}

		// 获取用户ID
		var userID string
		var needUserInit bool

		// 1. 首先从元数据中获取userId
		userID = utils.GetUserIDFromMetadata(metadata)
		if userID != "" {
			log.Printf("[记忆上下文] 从元数据获取到用户ID: %s", userID)
		} else {
			// 2. 如果元数据中没有，使用标准方法获取
			var err error
			userID, needUserInit, err = utils.GetUserID()
			if err != nil {
				log.Printf("[记忆上下文] 获取用户ID失败: %v", err)
			}

			if userID != "" {
				log.Printf("[记忆上下文] 使用缓存/配置获取到用户ID: %s", userID)
			} else {
				log.Printf("[记忆上下文] 警告: 未能获取有效的用户ID，记忆可能无法被正确检索")
			}
		}

		// 设置基本元数据
		metadata["timestamp"] = time.Now().Unix()
		metadata["stored_at"] = time.Now().Format(time.RFC3339)
		metadata["manual_store"] = true // 标记为手动存储

		// 检查是否为待办事项
		bizType := 0 // 默认为常规记忆

		// 优化待办事项检测逻辑
		// 1. 检查是否有显式标记为待办项
		if metadata != nil && metadata["type"] == "todo" {
			log.Printf("[记忆上下文] 元数据中显式标记为待办事项")
			metadata["type"] = "todo"
			bizType = models.BizTypeTodo
			log.Printf("[记忆上下文] 设置bizType=%d (BizTypeTodo)", models.BizTypeTodo)
		} else {
			// 2. 使用扩展的正则表达式检查内容格式
			todoRegex := regexp.MustCompile(`(?i)^(- \[ \]|TODO:|待办:|提醒:|task:)`)
			// 3. 检查内容中是否包含待办关键词
			todoKeywordsRegex := regexp.MustCompile(`(?i)(待办事项|todo item|task list|待完成|to-do|to do)`)

			if todoRegex.MatchString(content) || todoKeywordsRegex.MatchString(content) {
				log.Printf("[记忆上下文] 检测到待办事项: %s", content)
				metadata["type"] = "todo" // 确保type字段为todo
				bizType = models.BizTypeTodo
				log.Printf("[记忆上下文] 设置bizType=%d (BizTypeTodo)", models.BizTypeTodo)
			} else {
				// 不是待办事项，设置为长期记忆
				metadata["type"] = "long_term_memory"
				log.Printf("[记忆上下文] 内容不匹配待办事项模式，设置为普通长期记忆")
			}
		}

		log.Printf("[记忆上下文] 存储记忆: sessionID=%s, userID=%s, 类型=%s, 优先级=%s",
			sessionID, userID, metadata["type"], priority)

		// 创建存储上下文请求
		storeRequest := models.StoreContextRequest{
			SessionID: sessionID,
			UserID:    userID,
			Content:   content,
			Priority:  priority,
			Metadata:  metadata,
			BizType:   bizType,
		}

		log.Printf("存储长期记忆: sessionID=%s, 内容长度=%d, 优先级=%s, 类型=%s",
			sessionID, len(content), priority, metadata["type"])

		// 调用长期记忆存储
		memoryID, err := contextService.StoreContext(ctx, storeRequest)

		if err != nil {
			errMsg := fmt.Sprintf("存储长期记忆失败: %v", err)
			log.Println(errMsg)
			logToolCall("memorize_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 构建响应
		response := map[string]interface{}{
			"memoryId":     memoryID,
			"success":      true,
			"message":      "成功将内容存储到长期记忆",
			"type":         metadata["type"],
			"needUserInit": needUserInit,
		}

		if userID != "" {
			response["userId"] = userID
		}

		// 如果需要用户初始化，添加提示信息
		if needUserInit {
			response["initPrompt"] = "需要进行用户初始化才能将记忆与您的个人账户关联。请完成用户初始化流程。"
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			errMsg := fmt.Sprintf("序列化响应失败: %v", err)
			log.Println(errMsg)
			logToolCall("memorize_context", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("[记忆上下文] 成功存储记忆: memoryID=%s, 类型=%s", memoryID, metadata["type"])
		logToolCall("memorize_context", request.Params.Arguments, string(jsonData), nil, time.Since(startTime))
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// 辅助函数

// getSummaryFromResult 从结果中提取摘要信息
func getSummaryFromResult(memory string) string {
	// 这里可以编写更复杂的逻辑来提取或生成简洁摘要
	// 简单实现：取前100个字符
	if len(memory) > 100 {
		return memory[:100] + "..."
	}
	return memory
}

// countMemories 计算结果中的记忆数量
func countMemories(result models.ContextResponse) int {
	// 简单实现：计算短期和长期记忆的条目数
	count := 0

	if result.ShortTermMemory != "" {
		count += countStringLines(result.ShortTermMemory)
	}

	if result.LongTermMemory != "" {
		count += countStringLines(result.LongTermMemory)
	}

	return count
}

// countRelevantMemories 计算相关记忆数量
func countRelevantMemories(result models.ContextResponse) int {
	if result.LongTermMemory == "" {
		return 0
	}
	return countStringLines(result.LongTermMemory)
}

// countStringLines 计算字符串中的行数
func countStringLines(s string) int {
	if s == "" {
		return 0
	}

	lineCount := 0
	for _, ch := range s {
		if ch == '\n' {
			lineCount++
		}
	}
	return lineCount + 1
}

// 帮助函数

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := TryParseInt(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := TryParseFloat(value); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func TryParseInt(value string) (int, error) {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}

func TryParseFloat(value string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(value, "%f", &result)
	return result, err
}

func ensureDirExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("创建目录失败: %v", err)
		}
	}
}

// 新增处理函数: 检索待办事项
func retrieveTodosHandler(contextService *services.ContextService) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("retrieve_todos", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 获取可选参数
		status, _ := request.Params.Arguments["status"].(string)
		if status == "" {
			status = "all" // 默认查询所有状态
		}

		limitStr, _ := request.Params.Arguments["limit"].(string)
		limit := 20 // 默认限制
		if limitStr != "" {
			limitVal, err := strconv.Atoi(limitStr)
			if err == nil && limitVal > 0 {
				limit = limitVal
			}
		}

		// 获取用户ID
		var userID string
		var needUserInit bool

		// 1. 首先从请求参数中查找userId
		if requestUserID, ok := request.Params.Arguments["userId"].(string); ok && requestUserID != "" {
			userID = requestUserID
			log.Printf("[检索待办] 从请求参数获取用户ID: %s", userID)
		} else {
			// 2. 使用标准方法获取
			var err error
			userID, needUserInit, err = utils.GetUserID()
			if err != nil {
				log.Printf("[检索待办] 获取用户ID失败: %v", err)
			}

			if userID != "" {
				log.Printf("[检索待办] 使用缓存/配置获取用户ID: %s", userID)
			} else {
				log.Printf("[检索待办] 警告: 未能获取有效用户ID，待办检索可能失败")
			}
		}

		log.Printf("[检索待办] 执行检索: sessionID=%s, userID=%s, status=%s, limit=%d",
			sessionID, userID, status, limit)

		// 调用服务执行检索
		todosResp, err := contextService.RetrieveTodos(ctx, models.RetrieveTodosRequest{
			SessionID: sessionID,
			UserID:    userID,
			Status:    status,
			Limit:     limit,
		})

		if err != nil {
			errMsg := fmt.Sprintf("检索待办事项失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_todos", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		// 构建自定义响应，包含needUserInit字段
		response := map[string]interface{}{
			"items":        todosResp.Items,
			"total":        todosResp.Total,
			"status":       todosResp.Status,
			"userId":       todosResp.UserID,
			"needUserInit": needUserInit,
		}

		// 如果需要用户初始化，添加描述信息
		if needUserInit {
			response["description"] = "需要进行用户初始化才能将待办事项与您的个人账户关联。请完成用户初始化流程。"
		}

		// 转换为JSON字符串响应
		jsonData, err := json.Marshal(response)
		if err != nil {
			errMsg := fmt.Sprintf("序列化结果失败: %v", err)
			log.Println(errMsg)
			logToolCall("retrieve_todos", request.Params.Arguments, errMsg, err, time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		log.Printf("[检索待办] 检索成功: 找到%d个待办事项", len(todosResp.Items))
		logToolCall("retrieve_todos", request.Params.Arguments, string(jsonData), nil, time.Since(startTime))
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// userInitDialogHandler 处理用户初始化对话请求
func userInitDialogHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()

		// 详细日志：开始处理用户初始化对话
		log.Printf("[用户初始化对话] 开始处理请求，参数: %+v", request.Params.Arguments)

		// 验证参数
		sessionID, ok := request.Params.Arguments["sessionId"].(string)
		if !ok || sessionID == "" {
			errMsg := "错误: sessionId必须是非空字符串"
			log.Println(errMsg)
			logToolCall("user_init_dialog", request.Params.Arguments, errMsg, fmt.Errorf(errMsg), time.Since(startTime))
			return mcp.NewToolResultText(errMsg), nil
		}

		userResponse, _ := request.Params.Arguments["userResponse"].(string)
		log.Printf("[用户初始化对话] 处理会话ID=%s, 用户响应=%q", sessionID, userResponse)

		// 如果有用户响应，则处理响应
		var state *utils.DialogState
		var err error

		// 使用defer捕获和记录任何可能的panic
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[用户初始化对话] 发生panic: %v", r)
				// 记录堆栈信息
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				log.Printf("[用户初始化对话] 堆栈: %s", buf[:n])
			}
		}()

		// 首先检查会话状态是否已经存在
		dialogExists := false
		// 这里不直接访问dialogStates，而是通过尝试初始化来检查
		tmpState, _ := utils.InitializeUserByDialog(sessionID)
		if tmpState != nil {
			dialogExists = true
			log.Printf("[用户初始化对话] 检测到会话状态已存在: state=%s", tmpState.State)
		}

		if userResponse != "" {
			log.Printf("[用户初始化对话] 处理用户响应: %q", userResponse)

			// 如果有响应但没有会话状态，可能是第一次调用，先确保初始化
			if !dialogExists {
				log.Printf("[用户初始化对话] 警告: 收到用户响应但会话状态不存在，先初始化状态")
				tmpState, err = utils.InitializeUserByDialog(sessionID)
				if err != nil {
					log.Printf("[用户初始化对话] 初始化对话状态失败: %v", err)
					logToolCall("user_init_dialog", request.Params.Arguments, err.Error(), err, time.Since(startTime))
					return mcp.NewToolResultText(fmt.Sprintf("处理用户配置对话出错: 无法初始化会话状态: %v", err)), nil
				}
			}

			// 添加详细的错误处理
			state, err = utils.HandleUserDialogResponse(sessionID, userResponse)
			if err != nil {
				log.Printf("[用户初始化对话] 处理用户响应失败: %v", err)
			}
		} else {
			log.Printf("[用户初始化对话] 初始化或获取当前对话状态")
			// 初始化或获取当前对话状态
			state, err = utils.InitializeUserByDialog(sessionID)
			if err != nil {
				log.Printf("[用户初始化对话] 初始化对话状态失败: %v", err)
			}
		}

		if err != nil {
			log.Printf("[用户初始化对话] 错误: %v", err)
			logToolCall("user_init_dialog", request.Params.Arguments, err.Error(), err, time.Since(startTime))
			return mcp.NewToolResultText(fmt.Sprintf("处理用户配置对话出错: %v", err)), nil
		}

		log.Printf("[用户初始化对话] 获取到对话状态: state=%s, userID=%s", state.State, state.UserID)

		// 如果用户配置完成，更新全局缓存
		if state.State == utils.DialogStateCompleted && state.UserID != "" {
			log.Printf("[用户初始化对话] 用户配置完成，确保全局缓存已更新，UserID: %s", state.UserID)
			// 确保用户ID被缓存
			utils.SetCachedUserID(state.UserID)
		}

		// 构建响应
		result := map[string]interface{}{
			"state": state.State,
		}

		// 根据状态添加相应字段
		switch state.State {
		case utils.DialogStateNewUser:
			result["userId"] = state.UserID
			result["message"] = "已为您创建新用户账号"
			result["welcomeMessage"] = "欢迎使用上下文记忆管理工具！您的数据将与您的用户ID关联。请妥善保管您的用户ID，当您在其他设备使用时需要输入它。"
			log.Printf("[用户初始化对话] 新用户状态: userID=%s", state.UserID)
		case utils.DialogStateExisting:
			result["message"] = "请输入您的用户ID以继续"
			result["prompt"] = "用户ID格式为'user_'开头加8位字母数字，您可以直接粘贴完整ID"
			result["helpText"] = "如果您没有用户ID或想创建新账号，请回复'创建新账号'。如需重置流程，请回复'重置'"
			log.Printf("[用户初始化对话] 已有用户状态，等待输入用户ID")
		case utils.DialogStateCompleted:
			result["userId"] = state.UserID
			result["message"] = "用户配置已完成"
			result["isFirstTime"] = (userResponse != "") // 标记是否是首次配置完成
			log.Printf("[用户初始化对话] 配置完成状态: userID=%s, isFirstTime=%v", state.UserID, userResponse != "")
		default:
			result["message"] = "欢迎使用上下文记忆管理工具。为了在多设备间同步您的数据，我们需要创建一个用户ID。请问您是否已在其他设备上使用过该工具？"
			result["prompt"] = "回答'是'或'否'"
			result["helpText"] = "如果您以前使用过，我们将引导您输入用户ID；如果没有，我们将为您创建新账号"
			log.Printf("[用户初始化对话] 初始询问状态")
		}

		// 记录要返回的结果对象
		log.Printf("[用户初始化对话] 准备返回结果: %+v", result)

		// 记录工具调用日志
		logToolCall("user_init_dialog", request.Params.Arguments, result, nil, time.Since(startTime))

		// 序列化JSON结果，但不要在外层再包装成字符串
		jsonData, err := json.Marshal(result)
		if err != nil {
			log.Printf("[用户初始化对话] 错误: 无法序列化结果: %v", err)
			return mcp.NewToolResultText(fmt.Sprintf("处理用户配置对话出错: %v", err)), nil
		}

		// 使用原始JSON字符串返回，不要添加额外的引号
		log.Printf("[用户初始化对话] 完成处理，耗时: %v", time.Since(startTime))
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
