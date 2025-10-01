//go:build websocket

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
	"github.com/contextkeeper/service/internal/services"
	"github.com/contextkeeper/service/internal/utils"
	"github.com/contextkeeper/service/pkg/aliyun"
)

func main() {
	log.Println("启动 Context-Keeper WebSocket HTTP 服务器...")

	// 设置WebSocket HTTP模式环境变量
	os.Setenv("HTTP_MODE", "true")
	os.Setenv("WEBSOCKET_HTTP_MODE", "true")

	// WebSocket模式：日志直接输出到标准输出（不再需要文件日志）
	// 这样云端部署时可以通过 docker logs 直接查看业务日志
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("✅ WebSocket模式：日志输出到标准输出（便于云端查看）")

	// 初始化TraceID系统
	utils.InitTraceIDSystem()

	// 初始化共享组件
	contextService, _, cancelCleanup := initializeServices()
	defer cancelCleanup()

	// 初始化WebSocket管理器
	if services.GlobalWSManager == nil {
		log.Println("初始化WebSocket管理器...")
		// GlobalWSManager已经在websocket_manager.go中初始化了，这里只需要确认
	}

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

	// 获取向量服务
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

	// 创建用户存储仓库（使用工厂模式）
	userRepository, err := services.CreateUserRepositoryWithAutoDetection(vectorService)
	if err != nil {
		log.Fatalf("创建用户存储仓库失败: %v", err)
	}

	// 初始化用户存储仓库
	if err := userRepository.InitRepository(); err != nil {
		log.Printf("警告：初始化用户存储仓库失败: %v", err)
	} else {
		log.Println("用户存储仓库初始化成功")
	}

	// 创建API处理器
	handler := api.NewHandler(contextService, vectorService, userRepository, cfg)

	// 注册WebSocket路由
	handler.RegisterWebSocketRoutes(router)

	// 创建并注册Streamable HTTP处理器（支持MCP协议）
	streamableHandler := api.NewStreamableHTTPHandler(handler)
	streamableHandler.RegisterStreamableHTTPRoutes(router)

	// 添加基础健康检查和服务信息路由
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "context-keeper",
			"version":     "1.0.0",
			"mode":        "websocket-http",
			"protocol":    "WebSocket + MCP Streamable HTTP",
			"status":      "running",
			"timestamp":   time.Now().Format(time.RFC3339),
			"description": "Context Keeper WebSocket + MCP HTTP 服务器",
			"endpoints": gin.H{
				"mcp":       "/mcp",
				"websocket": "/ws",
				"status":    "/ws/status",
			},
		})
	})

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"mode":   "websocket-http",
			"websocket": gin.H{
				"enabled":     true,
				"connections": len(services.GlobalWSManager.GetOnlineUsers()),
			},
		})
	})

	// 获取端口配置 - 优先使用WEBSOCKET_SERVER_PORT，兼容PORT
	port := getEnv("WEBSOCKET_SERVER_PORT", getEnv("HTTP_SERVER_PORT", getEnv("PORT", cfg.WebSocketServerPort)))
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

		// 关闭WebSocket连接
		if services.GlobalWSManager != nil {
			// 关闭所有在线用户的连接
			onlineUsers := services.GlobalWSManager.GetOnlineUsers()
			for _, userID := range onlineUsers {
				services.GlobalWSManager.UnregisterUser(userID)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("服务器关闭时出错: %v", err)
		}
		log.Println("服务器已关闭")
	}()

	// 启动HTTP服务器
	log.Printf("Context-Keeper WebSocket HTTP 服务器启动在 %s", addr)
	log.Printf("服务信息: http://%s/", addr)
	log.Printf("健康检查: http://%s/health", addr)
	log.Printf("MCP协议端点: http://%s/mcp", addr)
	log.Printf("WebSocket端点: ws://%s/ws", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP服务器启动失败: %v", err)
	}
}
