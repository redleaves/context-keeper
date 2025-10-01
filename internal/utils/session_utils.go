package utils

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
)

// GetWorkspaceSessionID 获取带工作区标识的会话ID
// 重构：强制要求工作空间路径，所有session都必须基于用户+工作空间隔离
func GetWorkspaceSessionID(
	sessionStore *store.SessionStore,
	userID string,
	sessionID string,
	workspacePath string, // 🔥 必需参数：工作空间路径
	metadata map[string]interface{},
	sessionTimeout time.Duration,
) (*models.Session, bool, error) {
	log.Printf("🔍 [会话工具] === 开始GetWorkspaceSessionID ===")
	log.Printf("🔍 [会话工具] 输入参数: userID=%s, sessionID=%s, workspacePath=%s, sessionTimeout=%v",
		userID, sessionID, workspacePath, sessionTimeout)

	// 🔥 强制验证：工作空间路径不能为空
	if workspacePath == "" {
		log.Printf("🔍 [会话工具] ❌ 错误：工作空间路径不能为空")
		return nil, false, fmt.Errorf("工作空间路径不能为空，session必须基于用户+工作空间隔离")
	}

	var session *models.Session
	var isNewSession bool
	var err error

	// 如果指定了sessionID，直接获取该会话
	if sessionID != "" {
		log.Printf("🔍 [会话工具] 步骤1 - 指定了sessionID，直接获取: %s", sessionID)
		session, err = sessionStore.GetSession(sessionID)
		if err != nil {
			log.Printf("🔍 [会话工具] 步骤1 - 获取指定会话失败: %v", err)
			return nil, false, fmt.Errorf("获取指定会话失败: %v", err)
		}

		// 🔥 验证会话的工作空间是否匹配
		sessionWorkspacePath := ""
		if session.Metadata != nil {
			if wp, ok := session.Metadata["workspacePath"].(string); ok {
				sessionWorkspacePath = wp
			}
		}

		expectedWorkspaceHash := GenerateWorkspaceHash(workspacePath)
		sessionWorkspaceHash := GenerateWorkspaceHash(sessionWorkspacePath)

		if sessionWorkspaceHash != expectedWorkspaceHash {
			log.Printf("🔍 [会话工具] ❌ 会话工作空间不匹配: 期望=%s, 实际=%s", expectedWorkspaceHash, sessionWorkspaceHash)
			return nil, false, fmt.Errorf("会话工作空间不匹配，无法跨工作空间访问session")
		}

		isNewSession = false
		log.Printf("🔍 [会话工具] 步骤1 - 获取指定会话成功且工作空间匹配: %s", sessionID)
		log.Printf("🔍 [会话工具] === GetWorkspaceSessionID完成(指定会话) ===")
		return session, isNewSession, nil
	}

	// 生成工作空间哈希
	workspaceHash := GenerateWorkspaceHash(workspacePath)
	log.Printf("🔍 [会话工具] 步骤2 - 工作空间: '%s' -> 哈希: '%s'", workspacePath, workspaceHash)

	// 🔥 强制使用工作空间会话模式
	log.Printf("🔍 [会话工具] 步骤3 - 使用工作空间会话模式")
	session, isNewSession, err = sessionStore.GetOrCreateActiveSessionWithWorkspace(userID, workspaceHash, sessionTimeout)
	if err != nil {
		log.Printf("🔍 [会话工具] 步骤3 - GetOrCreateActiveSessionWithWorkspace失败: %v", err)
		return nil, false, fmt.Errorf("获取或创建工作空间会话失败: %v", err)
	}

	log.Printf("🔍 [会话工具] 步骤3 - GetOrCreateActiveSessionWithWorkspace成功: sessionID=%s, isNew=%t", session.ID, isNewSession)

	if isNewSession {
		log.Printf("🔍 [会话工具] 创建新工作空间会话: %s (工作空间: %s)", session.ID, workspaceHash)
	} else {
		log.Printf("🔍 [会话工具] 复用工作空间会话: %s (工作空间: %s)", session.ID, workspaceHash)
	}

	// 确保会话元数据包含工作空间信息
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}
	session.Metadata["workspaceHash"] = workspaceHash
	session.Metadata["workspacePath"] = workspacePath

	// 🔥 修复：关键问题 - 必须将userId存储到metadata中
	// 这是其他MCP工具能够从会话中获取用户ID的关键
	session.Metadata["userId"] = userID
	log.Printf("🔍 [会话工具] 🔥 关键修复 - 已将userId存储到会话metadata: %s", userID)

	// 更新会话活跃时间
	log.Printf("🔍 [会话工具] 步骤4 - 更新会话活跃时间")
	session.LastActive = time.Now()

	// 如果提供了额外元数据，合并到会话中
	if metadata != nil && len(metadata) > 0 {
		log.Printf("🔍 [会话工具] 步骤5 - 合并额外元数据，数量: %d", len(metadata))
		for k, v := range metadata {
			// 不允许覆盖工作空间相关的元数据
			if k != "workspaceHash" && k != "workspacePath" {
				log.Printf("🔍 [会话工具] 步骤5 - 设置元数据 %s: %+v", k, v)
				session.Metadata[k] = v
			}
		}
	}

	// 保存会话
	if err := sessionStore.SaveSession(session); err != nil {
		log.Printf("🔍 [会话工具] 步骤6 - 保存会话失败: %v", err)
	} else {
		log.Printf("🔍 [会话工具] 步骤6 - 保存会话成功")
	}

	log.Printf("🔍 [会话工具] === GetWorkspaceSessionID完成 ===")
	return session, isNewSession, nil
}

// ExtractWorkspaceNameFromPath 从完整路径提取工作空间名称
// 🔥 这是所有服务共用的工具函数，避免重复定义
func ExtractWorkspaceNameFromPath(workspacePath string) string {
	if workspacePath == "" {
		return ""
	}

	// 🔥 从完整路径中提取最后一级目录名作为工作空间名
	if strings.Contains(workspacePath, "/") {
		parts := strings.Split(workspacePath, "/")
		workspaceName := parts[len(parts)-1]
		if workspaceName != "" {
			log.Printf("🔧 [工作空间名提取] 从路径 %s 提取工作空间名: %s", workspacePath, workspaceName)
			return workspaceName
		}
	}

	// 如果路径不包含/，直接返回原路径
	log.Printf("🔧 [工作空间名提取] 路径不包含分隔符，直接使用: %s", workspacePath)
	return workspacePath
}
