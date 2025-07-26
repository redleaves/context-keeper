package utils

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateWorkspaceHash 根据工作空间路径生成哈希值
func GenerateWorkspaceHash(workspacePath string) string {
	if workspacePath == "" {
		return "default"
	}
	// 使用标准化路径生成一致的哈希
	cleanPath := filepath.Clean(workspacePath)
	hasher := sha256.New()
	hasher.Write([]byte(cleanPath))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return hash[:16] // 取前16个字符作为工作空间标识
}

// 🔥 新增：GetWorkspaceIdentifier 统一的工作空间标识获取方法
func GetWorkspaceIdentifier(workspacePath string) string {
	// 优先级1: 使用传入的工作空间路径
	if workspacePath != "" && workspacePath != "unknown" {
		return GenerateWorkspaceHash(workspacePath)
	}

	// 优先级2: 从环境变量获取工作空间路径
	if envWorkspace := os.Getenv("WORKSPACE_ROOT"); envWorkspace != "" {
		return GenerateWorkspaceHash(envWorkspace)
	}

	// 优先级3: 从当前工作目录获取
	if cwd, err := os.Getwd(); err == nil {
		return GenerateWorkspaceHash(cwd)
	}

	// 回退: 生成随机标识而不是使用固定的"default"值
	// 这确保即使在无法确定工作空间的情况下，每个连接也有唯一的工作空间标识
	return GenerateRandomString(8)
}
