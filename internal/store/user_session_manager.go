package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UserSessionManager 用户会话管理器
type UserSessionManager struct {
	baseStorePath string
	userStores    map[string]*SessionStore // 用户ID到SessionStore的映射
	mu            sync.RWMutex
}

// NewUserSessionManager 创建新的用户会话管理器
func NewUserSessionManager(baseStorePath string) *UserSessionManager {
	log.Printf("[用户会话管理] 初始化用户会话管理器, 基础路径: %s", baseStorePath)

	// 确保基础目录存在
	usersPath := filepath.Join(baseStorePath, "users")
	if err := os.MkdirAll(usersPath, 0755); err != nil {
		log.Printf("[用户会话管理] 警告: 创建用户目录失败: %v", err)
	}

	return &UserSessionManager{
		baseStorePath: baseStorePath,
		userStores:    make(map[string]*SessionStore),
	}
}

// GetUserSessionStore 获取特定用户的会话存储
func (m *UserSessionManager) GetUserSessionStore(userID string) (*SessionStore, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	// 先检查是否已有此用户的存储
	m.mu.RLock()
	store, exists := m.userStores[userID]
	m.mu.RUnlock()

	if exists {
		return store, nil
	}

	// 创建用户专属存储目录路径
	userStorePath := filepath.Join(m.baseStorePath, "users", userID)

	// 创建新的SessionStore
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if store, exists = m.userStores[userID]; exists {
		return store, nil
	}

	// 确保用户目录存在
	if err := os.MkdirAll(userStorePath, 0755); err != nil {
		return nil, fmt.Errorf("创建用户存储目录失败: %w", err)
	}

	// 创建并初始化SessionStore
	var err error
	store, err = NewSessionStore(userStorePath)
	if err != nil {
		return nil, fmt.Errorf("创建用户会话存储失败: %w", err)
	}

	// 保存到映射
	m.userStores[userID] = store

	log.Printf("[用户会话管理] 已创建用户%s的会话存储", userID)
	return store, nil
}

// CleanupInactiveUserStores 清理不活跃的用户存储
func (m *UserSessionManager) CleanupInactiveUserStores(inactiveThreshold time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var cleanedCount int

	for userID, store := range m.userStores {
		// 检查此用户的最后活跃时间
		lastActive := store.GetLastActiveTime()
		if now.Sub(lastActive) > inactiveThreshold {
			delete(m.userStores, userID)
			cleanedCount++
			log.Printf("[用户会话管理] 已清理不活跃用户%s的会话存储", userID)
		}
	}

	return cleanedCount
}

// CleanupAllShortTermMemory 清理所有用户的短期记忆
func (m *UserSessionManager) CleanupAllShortTermMemory(maxAgeDays int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalCleaned := 0
	for userID, store := range m.userStores {
		cleaned := store.CleanupShortTermMemory(maxAgeDays)
		totalCleaned += cleaned
		log.Printf("[用户会话管理] 已清理用户%s的%d个过期会话", userID, cleaned)
	}

	return totalCleaned
}
