//go:build http

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/contextkeeper/service/internal/api"
	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/services"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/aliyun"
	"github.com/contextkeeper/service/pkg/vectorstore"
)

func main() {
	log.Println("启动 Context-Keeper Streamable HTTP MCP 服务器...")

	// 设置Streamable HTTP模式环境变量
	os.Setenv("HTTP_MODE", "true")
	os.Setenv("STREAMABLE_HTTP_MODE", "true")

	// HTTP模式：日志直接输出到标准输出（不再需要文件日志）
	// 这样云端部署时可以通过 docker logs 直接查看业务日志
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("✅ HTTP模式：日志输出到标准输出（便于云端查看）")

	// 初始化TraceID系统
	utils.InitTraceIDSystem()

	// 初始化共享组件（🔥 修改：现在返回LLMDrivenContextService以支持LLM驱动智能功能）
	llmDrivenContextService, _, cancelCleanup := initializeServices()
	defer cancelCleanup()

	// 加载配置
	cfg := config.Load()

	// 设置Gin模式
	if getEnv("GIN_MODE", cfg.GinMode) == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin路由器
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	// 🔥 【新增】添加TraceID中间件
	router.Use(utils.TraceIDMiddleware())

	// 配置CORS
	config_cors := cors.DefaultConfig()
	config_cors.AllowAllOrigins = true
	config_cors.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config_cors.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "Cache-Control", "X-Requested-With", "Last-Event-ID", "X-Trace-ID"}
	config_cors.AllowCredentials = true
	config_cors.ExposeHeaders = []string{"Content-Length", "X-Trace-ID"}
	config_cors.MaxAge = 12 * time.Hour
	router.Use(cors.New(config_cors))

	// 🔥 【新逻辑】初始化向量存储工厂
	log.Println("🏭 [向量存储工厂] 开始初始化向量存储工厂...")
	factory, err := vectorstore.InitializeFactoryFromEnv()
	if err != nil {
		log.Printf("❌ [向量存储工厂] 工厂初始化失败: %v", err)
		log.Printf("⚠️ [向量存储工厂] 将回退到传统阿里云VectorService")

		// 回退到传统方式
		embeddingAPIURL := getEnv("EMBEDDING_API_URL", cfg.EmbeddingAPIURL)
		embeddingAPIKey := getEnv("EMBEDDING_API_KEY", cfg.EmbeddingAPIKey)
		vectorDBURL := getEnv("VECTOR_DB_URL", cfg.VectorDBURL)
		vectorDBAPIKey := getEnv("VECTOR_DB_API_KEY", cfg.VectorDBAPIKey)
		vectorDBCollection := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
		vectorDBDimension := getIntEnv("VECTOR_DB_DIMENSION", cfg.VectorDBDimension)
		vectorDBMetric := getEnv("VECTOR_DB_METRIC", cfg.VectorDBMetric)
		similarityThreshold := getFloatEnv("SIMILARITY_THRESHOLD", cfg.SimilarityThreshold)

		vectorService := aliyun.NewVectorService(
			embeddingAPIURL,
			embeddingAPIKey,
			vectorDBURL,
			vectorDBAPIKey,
			vectorDBCollection,
			vectorDBDimension,
			vectorDBMetric,
			similarityThreshold,
		)

		// 🔍 调试日志：检查向量服务是否创建成功
		if vectorService == nil {
			log.Printf("❌ [调试] 向量服务创建失败：vectorService为nil")
		} else {
			log.Printf("✅ [调试] 传统向量服务创建成功：vectorService非nil")
		}

		// 🔍 调试日志：检查环境变量配置
		log.Printf("🔍 [调试] USER_REPOSITORY_TYPE环境变量: '%s'", os.Getenv("USER_REPOSITORY_TYPE"))
		log.Printf("🔍 [调试] 向量服务配置 - URL: %s, APIKey: %s, Collection: %s",
			vectorDBURL,
			"***"+vectorDBAPIKey[len(vectorDBAPIKey)-4:], // 只显示最后4位
			vectorDBCollection)

		// 创建用户存储仓库（使用工厂模式）
		log.Printf("🏭 [调试] 开始创建用户存储仓库...")
		userRepository, err := services.CreateUserRepositoryWithAutoDetection(vectorService)
		if err != nil {
			log.Fatalf("创建用户存储仓库失败: %v", err)
		}
		log.Printf("✅ [调试] 用户存储仓库创建成功")

		// 初始化用户存储仓库
		if err := userRepository.InitRepository(); err != nil {
			log.Printf("警告：初始化用户存储仓库失败: %v", err)
		} else {
			log.Println("用户存储仓库初始化成功")
		}

		// 创建API处理器
		handler := api.NewHandler(llmDrivenContextService, vectorService, userRepository, cfg)

		// 注册路由并启动服务器
		setupRoutesAndStartServer(router, handler, cfg)
		return
	}

	// 🔥 【新逻辑】工厂初始化成功，获取当前配置的向量存储
	log.Println("✅ [向量存储工厂] 工厂初始化成功，获取当前向量存储实例...")

	currentVectorStore, err := factory.GetCurrentVectorStore()
	if err != nil {
		log.Printf("⚠️ [向量存储工厂] 获取当前向量存储失败: %v (HTTP模式继续运行)", err)
		// 使用默认的向量服务，不中断服务启动
		currentVectorStore = nil
	}

	// 检查向量存储类型
	vectorStoreType := getEnv("VECTOR_STORE_TYPE", cfg.VectorStoreType)
	log.Printf("✅ [向量存储工厂] 成功加载向量存储类型: %s", vectorStoreType)

	// 初始化向量数据库和表空间
	if currentVectorStore != nil {
		log.Printf("🔧 [向量存储工厂] 开始初始化向量数据库和表空间...")
		collectionName := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
		if err := currentVectorStore.EnsureCollection(collectionName); err != nil {
			log.Printf("⚠️ [向量存储工厂] 向量集合初始化失败: %v (HTTP模式继续运行)", err)
		} else {
			log.Printf("✅ [向量存储工厂] 向量集合初始化成功")
		}
	} else {
		log.Printf("⚠️ [向量存储工厂] 向量存储不可用，跳过集合初始化")
	}

	// 🔥 【重要】根据USER_REPOSITORY_TYPE创建对应的客户端
	userRepositoryType := getEnv("USER_REPOSITORY_TYPE", cfg.UserRepositoryType)
	log.Printf("🔍 [用户存储仓库] 检测到用户存储类型: %s", userRepositoryType)

	var userRepository models.UserRepository
	var compatibilityVectorService *aliyun.VectorService

	switch userRepositoryType {
	case "vearch":
		log.Printf("🔧 [用户存储仓库] 使用Vearch存储，从工厂获取VearchClient...")
		// 从向量存储工厂获取VearchClient
		vearchClient, err := factory.GetVearchClient()
		if err != nil {
			log.Printf("⚠️ [用户存储仓库] 获取VearchClient失败: %v，使用内存存储", err)
			// 降级到内存存储
			userRepository = store.NewMemoryUserRepository()
		} else {
			log.Printf("✅ [用户存储仓库] 成功获取VearchClient")

			// 创建Vearch用户存储仓库
			userRepository, err = services.CreateUserRepositoryWithAutoDetection(vearchClient)
			if err != nil {
				log.Printf("⚠️ [用户存储仓库] 创建Vearch用户存储仓库失败: %v，使用内存存储", err)
				userRepository = store.NewMemoryUserRepository()
			} else {
				log.Printf("✅ [用户存储仓库] Vearch用户存储仓库创建成功")

				// 为了兼容性，仍然需要创建阿里云VectorService用于API Handler
				embeddingAPIURL := getEnv("EMBEDDING_API_URL", cfg.EmbeddingAPIURL)
				embeddingAPIKey := getEnv("EMBEDDING_API_KEY", cfg.EmbeddingAPIKey)
				vectorDBURL := getEnv("VECTOR_DB_URL", cfg.VectorDBURL)
				vectorDBAPIKey := getEnv("VECTOR_DB_API_KEY", cfg.VectorDBAPIKey)
				vectorDBCollection := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
				vectorDBDimension := getIntEnv("VECTOR_DB_DIMENSION", cfg.VectorDBDimension)
				vectorDBMetric := getEnv("VECTOR_DB_METRIC", cfg.VectorDBMetric)
				similarityThreshold := getFloatEnv("SIMILARITY_THRESHOLD", cfg.SimilarityThreshold)

				compatibilityVectorService = aliyun.NewVectorService(
					embeddingAPIURL,
					embeddingAPIKey,
					vectorDBURL,
					vectorDBAPIKey,
					vectorDBCollection,
					vectorDBDimension,
					vectorDBMetric,
					similarityThreshold,
				)
				log.Printf("✅ [用户存储仓库] 创建阿里云实例用于API兼容性")
			}
		}

	case "aliyun":
		log.Printf("🔧 [用户存储仓库] 使用阿里云存储...")
		// 创建传统阿里云VectorService用于用户存储仓库
		embeddingAPIURL := getEnv("EMBEDDING_API_URL", cfg.EmbeddingAPIURL)
		embeddingAPIKey := getEnv("EMBEDDING_API_KEY", cfg.EmbeddingAPIKey)
		vectorDBURL := getEnv("VECTOR_DB_URL", cfg.VectorDBURL)
		vectorDBAPIKey := getEnv("VECTOR_DB_API_KEY", cfg.VectorDBAPIKey)
		vectorDBCollection := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
		vectorDBDimension := getIntEnv("VECTOR_DB_DIMENSION", cfg.VectorDBDimension)
		vectorDBMetric := getEnv("VECTOR_DB_METRIC", cfg.VectorDBMetric)
		similarityThreshold := getFloatEnv("SIMILARITY_THRESHOLD", cfg.SimilarityThreshold)

		aliyunVectorService := aliyun.NewVectorService(
			embeddingAPIURL,
			embeddingAPIKey,
			vectorDBURL,
			vectorDBAPIKey,
			vectorDBCollection,
			vectorDBDimension,
			vectorDBMetric,
			similarityThreshold,
		)
		log.Printf("✅ [用户存储仓库] 创建阿里云实例用于用户存储")

		// 创建用户存储仓库（使用阿里云实例）
		userRepository, err = services.CreateUserRepositoryWithAutoDetection(aliyunVectorService)
		if err != nil {
			log.Fatalf("❌ [用户存储仓库] 创建阿里云用户存储仓库失败: %v", err)
		}
		log.Printf("✅ [用户存储仓库] 阿里云用户存储仓库创建成功")

		// 兼容性VectorService就是阿里云本身
		compatibilityVectorService = aliyunVectorService

	default:
		log.Printf("🔧 [用户存储仓库] 使用默认存储类型: %s", userRepositoryType)
		// 其他类型（memory, mysql, tencent等）不需要特殊的客户端
		userRepository, err = services.CreateUserRepositoryWithAutoDetection(nil)
		if err != nil {
			log.Fatalf("❌ [用户存储仓库] 创建%s用户存储仓库失败: %v", userRepositoryType, err)
		}
		log.Printf("✅ [用户存储仓库] %s用户存储仓库创建成功", userRepositoryType)

		// 创建阿里云VectorService用于API兼容性
		embeddingAPIURL := getEnv("EMBEDDING_API_URL", cfg.EmbeddingAPIURL)
		embeddingAPIKey := getEnv("EMBEDDING_API_KEY", cfg.EmbeddingAPIKey)
		vectorDBURL := getEnv("VECTOR_DB_URL", cfg.VectorDBURL)
		vectorDBAPIKey := getEnv("VECTOR_DB_API_KEY", cfg.VectorDBAPIKey)
		vectorDBCollection := getEnv("VECTOR_DB_COLLECTION", cfg.VectorDBCollection)
		vectorDBDimension := getIntEnv("VECTOR_DB_DIMENSION", cfg.VectorDBDimension)
		vectorDBMetric := getEnv("VECTOR_DB_METRIC", cfg.VectorDBMetric)
		similarityThreshold := getFloatEnv("SIMILARITY_THRESHOLD", cfg.SimilarityThreshold)

		compatibilityVectorService = aliyun.NewVectorService(
			embeddingAPIURL,
			embeddingAPIKey,
			vectorDBURL,
			vectorDBAPIKey,
			vectorDBCollection,
			vectorDBDimension,
			vectorDBMetric,
			similarityThreshold,
		)
		log.Printf("✅ [用户存储仓库] 创建阿里云实例用于API兼容性")
	}

	// 初始化用户存储仓库
	if err := userRepository.InitRepository(); err != nil {
		log.Printf("警告：初始化用户存储仓库失败: %v", err)
	} else {
		log.Println("用户存储仓库初始化成功")
	}

	// 🔥 【重要】修改LLMDrivenContextService以使用新的向量存储工厂
	log.Printf("🔧 [向量存储工厂] 更新LLMDrivenContextService以使用新的向量存储...")
	llmDrivenContextService.GetContextService().SetVectorStore(currentVectorStore)
	log.Printf("✅ [向量存储工厂] LLMDrivenContextService更新完成")

	// 🔥 重要：向量存储设置完成后，重新进行延迟赋值
	log.Printf("🔧 [延迟赋值] 重新设置MultiDimensionalRetriever的向量引擎...")
	llmDrivenContextService.ReinitializeVectorEngine()
	log.Printf("✅ [延迟赋值] 向量引擎重新设置完成")

	// 创建API处理器（使用兼容性VectorService）
	handler := api.NewHandler(llmDrivenContextService, compatibilityVectorService, userRepository, cfg)

	// 注册路由并启动服务器
	setupRoutesAndStartServer(router, handler, cfg)
}

// setupRoutesAndStartServer 注册路由并启动服务器
func setupRoutesAndStartServer(router *gin.Engine, handler *api.Handler, cfg *config.Config) {
	// 注册WebSocket路由
	handler.RegisterWebSocketRoutes(router)

	// 🔥 新增：注册Session管理接口 - 独立于MCP协议的管理端点
	handler.RegisterManagementRoutes(router)

	// 🔥 新增：注册批量embedding路由 - 直接在这里调用，不通过RegisterRoutes
	if handler.GetBatchEmbeddingHandler() != nil {
		handler.GetBatchEmbeddingHandler().RegisterBatchEmbeddingRoutes(router)
		log.Println("✅ 批量Embedding路由注册成功")
	} else {
		log.Println("⚠️ 批量Embedding服务未初始化，跳过路由注册")
	}

	// 🔥 新增：注册静态文件服务器 - 为批量embedding提供临时文件访问
	router.Static("/temp", "./data/temp")
	log.Println("✅ 静态文件服务器注册成功: /temp -> ./data/temp")

	// 创建并注册Streamable HTTP处理器 - 这是HTTP模式的主要协议
	streamableHandler := api.NewStreamableHTTPHandler(handler)
	streamableHandler.RegisterStreamableHTTPRoutes(router)

	// 添加基础健康检查和服务信息路由
	router.GET("/", func(c *gin.Context) {
		// 获取当前向量存储类型
		vectorStoreType := getEnv("VECTOR_STORE_TYPE", cfg.VectorStoreType)

		c.JSON(http.StatusOK, gin.H{
			"service":     "context-keeper",
			"version":     "1.0.0",
			"mode":        "streamable-http",
			"protocol":    "MCP Streamable HTTP",
			"status":      "running",
			"timestamp":   time.Now().Format(time.RFC3339),
			"description": "Context Keeper Streamable HTTP MCP 服务器",
			"vectorStore": gin.H{
				"type":        vectorStoreType,
				"factory":     "enabled",
				"description": fmt.Sprintf("当前使用 %s 向量存储", vectorStoreType),
			},
			"endpoints": gin.H{
				"mcp":          "/mcp",
				"capabilities": "/mcp/capabilities",
			},
		})
	})

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		// 获取当前向量存储类型
		vectorStoreType := getEnv("VECTOR_STORE_TYPE", cfg.VectorStoreType)

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"mode":   "streamable-http",
			"vectorStore": gin.H{
				"type":    vectorStoreType,
				"factory": "enabled",
			},
			"websocket": gin.H{
				"enabled":     true,
				"connections": len(services.GlobalWSManager.GetOnlineUsers()),
			},
		})
	})

	// 获取端口配置 - 优先使用HTTP_SERVER_PORT，兼容PORT
	port := getEnv("HTTP_SERVER_PORT", getEnv("PORT", cfg.HTTPServerPort))
	host := getEnv("HOST", cfg.Host)
	addr := fmt.Sprintf("%s:%s", host, port)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  2 * time.Minute, // 增加到2分钟，支持长时间LLM调用
		WriteTimeout: 2 * time.Minute, // 增加到2分钟，支持长时间响应
		IdleTimeout:  5 * time.Minute, // 增加空闲超时
	}

	// 优雅关闭处理
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("正在关闭服务器...")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("服务器关闭时出错: %v", err)
		}
		log.Println("服务器已关闭")
	}()

	// 启动HTTP服务器
	log.Printf("Context-Keeper Streamable HTTP MCP 服务器启动在 %s", addr)
	log.Printf("服务信息: http://%s/", addr)
	log.Printf("健康检查: http://%s/health", addr)
	log.Printf("MCP协议端点: http://%s/mcp", addr)
	log.Printf("能力查询端点: http://%s/mcp/capabilities", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP服务器启动失败: %v", err)
	}
}
