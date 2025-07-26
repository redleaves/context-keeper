package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// LocalInstructionService 本地存储指令服务
type LocalInstructionService struct{}

// NewLocalInstructionService 创建本地存储指令服务
func NewLocalInstructionService() *LocalInstructionService {
	return &LocalInstructionService{}
}

// replaceUserIDInPath 在路径中替换用户ID占位符
func (s *LocalInstructionService) replaceUserIDInPath(pathTemplate, userID string) string {
	return strings.ReplaceAll(pathTemplate, "{userId}", userID)
}

// GenerateUserInitInstruction 生成用户初始化指令
func (s *LocalInstructionService) GenerateUserInitInstruction(userID, accessCode string) *models.LocalInstruction {
	userConfig := &models.LocalUserConfig{
		UserID:    userID,
		FirstUsed: time.Now().Format(time.RFC3339),
	}

	targetPath := models.LocalPathUserConfig

	return &models.LocalInstruction{
		Type:    models.LocalInstructionUserConfig,
		Target:  targetPath,
		Content: userConfig,
		Options: models.LocalOperationOptions{
			CreateDir: true,
			Backup:    true,
		},
		CallbackID: fmt.Sprintf("user_init_%s", userID),
		Priority:   "high",
	}
}

// GenerateUserConfigUpdateInstruction 生成用户配置更新指令
func (s *LocalInstructionService) GenerateUserConfigUpdateInstruction(userID string) *models.LocalInstruction {
	userConfig := &models.LocalUserConfig{
		UserID:    userID,
		FirstUsed: time.Now().Format(time.RFC3339),
	}

	targetPath := models.LocalPathUserConfig

	return &models.LocalInstruction{
		Type:    models.LocalInstructionUserConfig,
		Target:  targetPath,
		Content: userConfig,
		Options: models.LocalOperationOptions{
			CreateDir: true,
			Backup:    true,
		},
		CallbackID: fmt.Sprintf("user_config_%s_%d", userID, time.Now().Unix()),
		Priority:   "high",
	}
}

// GenerateSessionStoreInstruction 生成会话存储指令
func (s *LocalInstructionService) GenerateSessionStoreInstruction(session *models.Session, userID string) *models.LocalInstruction {
	sessionData := &models.LocalSessionData{
		ID:          session.ID,
		CreatedAt:   session.CreatedAt,
		LastActive:  session.LastActive,
		Status:      session.Status,
		Messages:    session.Messages,
		Summary:     session.Summary,
		Metadata:    session.Metadata,
		CodeContext: session.CodeContext,
		EditHistory: session.EditHistory,
	}

	// 使用用户隔离的路径
	sessionsPath := s.replaceUserIDInPath(models.LocalPathSessions, userID)
	targetPath := fmt.Sprintf("%s%s.json", sessionsPath, session.ID)

	return &models.LocalInstruction{
		Type:    models.LocalInstructionSessionStore,
		Target:  targetPath,
		Content: sessionData,
		Options: models.LocalOperationOptions{
			CreateDir:  true,
			CleanupOld: true,
			MaxAge:     30 * 24 * 3600, // 30天
		},
		CallbackID: fmt.Sprintf("session_%s_%d", session.ID, time.Now().Unix()),
		Priority:   "normal",
	}
}

// GenerateShortMemoryStoreInstruction 生成短期记忆存储指令
func (s *LocalInstructionService) GenerateShortMemoryStoreInstruction(sessionID string, messages []*models.Message, userID string) *models.LocalInstruction {
	// 将消息转换为简化的历史记录格式 (兼容第一期)
	historyData := make(models.LocalHistoryData, 0, len(messages))
	for _, msg := range messages {
		historyContent := fmt.Sprintf("[%s] %s: %s",
			time.Unix(msg.Timestamp, 0).Format("2006-01-02 15:04:05"),
			msg.Role,
			msg.Content)
		historyData = append(historyData, historyContent)
	}

	// 使用用户隔离的路径
	historiesPath := s.replaceUserIDInPath(models.LocalPathHistories, userID)
	targetPath := fmt.Sprintf("%s%s.json", historiesPath, sessionID)

	return &models.LocalInstruction{
		Type:    models.LocalInstructionShortMemory,
		Target:  targetPath,
		Content: historyData,
		Options: models.LocalOperationOptions{
			CreateDir:  true,
			Merge:      true, // 合并到现有历史记录
			CleanupOld: true,
			MaxAge:     7 * 24 * 3600, // 7天
		},
		CallbackID: fmt.Sprintf("short_memory_%s_%d", sessionID, time.Now().Unix()),
		Priority:   "normal",
	}
}

// GenerateCodeContextStoreInstruction 生成代码上下文存储指令
func (s *LocalInstructionService) GenerateCodeContextStoreInstruction(sessionID string, codeContext map[string]*models.CodeFile, userID string) *models.LocalInstruction {
	// 使用用户隔离的路径
	codeContextPath := s.replaceUserIDInPath(models.LocalPathCodeContext, userID)
	targetPath := fmt.Sprintf("%s%s.json", codeContextPath, sessionID)

	return &models.LocalInstruction{
		Type:    models.LocalInstructionCodeContext,
		Target:  targetPath,
		Content: models.LocalCodeContextData(codeContext),
		Options: models.LocalOperationOptions{
			CreateDir: true,
			Merge:     true, // 合并到现有代码上下文
		},
		CallbackID: fmt.Sprintf("code_context_%s_%d", sessionID, time.Now().Unix()),
		Priority:   "normal",
	}
}

// GeneratePreferencesStoreInstruction 生成偏好设置存储指令
func (s *LocalInstructionService) GeneratePreferencesStoreInstruction(preferences *models.LocalPreferencesData, userID string) *models.LocalInstruction {
	targetPath := models.LocalPathPreferences

	return &models.LocalInstruction{
		Type:    models.LocalInstructionPreferences,
		Target:  targetPath,
		Content: preferences,
		Options: models.LocalOperationOptions{
			CreateDir: true,
			Merge:     true, // 合并到现有偏好设置
		},
		CallbackID: fmt.Sprintf("preferences_%s_%d", userID, time.Now().Unix()),
		Priority:   "normal",
	}
}

// GenerateCacheUpdateInstruction 生成缓存更新指令
func (s *LocalInstructionService) GenerateCacheUpdateInstruction(userID string, sessionStates map[string]interface{}) *models.LocalInstruction {
	cacheData := &models.LocalCacheData{
		UserID:        userID,
		SessionStates: sessionStates,
		LastUpdated:   time.Now().Unix(),
	}

	// 使用用户隔离的路径
	cachePath := s.replaceUserIDInPath(models.LocalPathCache, userID)
	targetPath := fmt.Sprintf("%scache.json", cachePath)

	return &models.LocalInstruction{
		Type:    models.LocalInstructionCacheUpdate,
		Target:  targetPath,
		Content: cacheData,
		Options: models.LocalOperationOptions{
			CreateDir: true,
		},
		CallbackID: fmt.Sprintf("cache_%s_%d", userID, time.Now().Unix()),
		Priority:   "low",
	}
}

// ShouldGenerateLocalInstruction 判断是否应该生成本地存储指令
// 这个函数可以根据用户偏好、会话状态等条件来决定是否生成本地指令
func (s *LocalInstructionService) ShouldGenerateLocalInstruction(instructionType models.LocalInstructionType, context map[string]interface{}) bool {
	// 默认策略：对于用户配置和会话存储总是生成指令
	switch instructionType {
	case models.LocalInstructionUserConfig, models.LocalInstructionUserInit:
		return true
	case models.LocalInstructionSessionStore:
		// 只有当会话有消息时才存储
		if messageCount, ok := context["messageCount"].(int); ok {
			return messageCount > 0
		}
		return true
	case models.LocalInstructionShortMemory:
		// 只有当有新消息时才存储
		if hasNewMessages, ok := context["hasNewMessages"].(bool); ok {
			return hasNewMessages
		}
		return false
	case models.LocalInstructionCodeContext:
		// 只有当有代码关联时才存储
		if hasCodeContext, ok := context["hasCodeContext"].(bool); ok {
			return hasCodeContext
		}
		return false
	default:
		return true
	}
}

// GetCallbackInstructionType 根据回调ID获取指令类型
func (s *LocalInstructionService) GetCallbackInstructionType(callbackID string) models.LocalInstructionType {
	// 从回调ID中解析指令类型
	if len(callbackID) == 0 {
		return ""
	}

	switch {
	case contains(callbackID, "user_init"):
		return models.LocalInstructionUserInit
	case contains(callbackID, "user_config"):
		return models.LocalInstructionUserConfig
	case contains(callbackID, "session"):
		return models.LocalInstructionSessionStore
	case contains(callbackID, "short_memory"):
		return models.LocalInstructionShortMemory
	case contains(callbackID, "code_context"):
		return models.LocalInstructionCodeContext
	case contains(callbackID, "preferences"):
		return models.LocalInstructionPreferences
	case contains(callbackID, "cache"):
		return models.LocalInstructionCacheUpdate
	default:
		return ""
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) >= len(substr) && findSubstring(s, substr))
}

// findSubstring 在字符串中查找子串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
