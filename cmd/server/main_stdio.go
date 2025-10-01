//go:build stdio

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"github.com/contextkeeper/service/internal/config"
)

func main() {
	log.Println("启动 Context-Keeper STDIO MCP 服务器...")

	// 设置MCP模式环境变量
	os.Setenv("MCP_MODE", "true")

	// 确保日志目录存在
	logDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "context-keeper", "logs")
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		log.Printf("警告: 无法创建日志目录: %v", err)
	}

	// 设置日志同时输出到文件和标准输出，使用绝对路径
	logFilePath := filepath.Join(logDir, "context-keeper-debug.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("警告: 无法打开日志文件: %v，日志将仅输出到控制台", err)
	} else {
		// 创建多写入器，同时写入文件和标准错误（stderr）
		// 这样可以避免日志干扰MCP协议通信（MCP使用stdout）
		multiWriter := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(multiWriter)
		log.Printf("日志将同时输出到文件(%s)和标准错误输出", logFilePath)
	}

	// 设置日志输出
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("正在启动ContextKeeper MCP服务...")

	// 初始化共享组件（现在返回LLMDrivenContextService以支持LLM驱动智能功能）
	llmDrivenContextService, _, cancelCleanup := initializeServices()
	defer cancelCleanup()

	// 创建MCP服务器
	serverOptions := []server.ServerOption{}

	// 添加资源功能支持
	serverOptions = append(serverOptions, server.WithResourceCapabilities(true, true))

	// 根据调试模式添加日志
	cfg := config.Load()
	debug := getEnv("DEBUG", fmt.Sprintf("%t", cfg.Debug)) == "true"
	if debug {
		serverOptions = append(serverOptions, server.WithLogging())
	}

	// 使用mcp-go创建服务器
	s := server.NewMCPServer(
		"context-keeper",
		"1.0.0",
		serverOptions...,
	)

	// 注册所有MCP工具
	// 🔥 修改：传递LLMDrivenContextService给MCP工具注册
	registerMCPTools(s, llmDrivenContextService)

	// 启动MCP服务器（阻塞运行）
	log.Println("Context-Keeper STDIO MCP 服务器已启动，等待连接...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("MCP服务器启动失败: %v", err)
	}
}
