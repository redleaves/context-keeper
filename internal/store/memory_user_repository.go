package store

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// MemoryUserRepository 内存版本的用户信息存储实现
// 适用于测试环境或小规模部署
type MemoryUserRepository struct {
	users map[string]*models.UserInfo
	mutex sync.RWMutex
}

// NewMemoryUserRepository 创建内存用户仓库实例
func NewMemoryUserRepository() models.UserRepository {
	return &MemoryUserRepository{
		users: make(map[string]*models.UserInfo),
	}
}

// CreateUser 创建新用户
func (repo *MemoryUserRepository) CreateUser(userInfo *models.UserInfo) error {
	log.Printf("🧠 [用户仓库-内存] ===== 开始创建用户: %s =====", userInfo.UserID)

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// 1. 检查用户是否已存在
	if _, exists := repo.users[userInfo.UserID]; exists {
		log.Printf("❌ [用户仓库-内存] 用户已存在: %s", userInfo.UserID)
		return fmt.Errorf("用户ID已存在: %s", userInfo.UserID)
	}

	// 2. 设置时间戳
	now := time.Now().Format(time.RFC3339)
	if userInfo.CreatedAt == "" {
		userInfo.CreatedAt = now
	}
	userInfo.UpdatedAt = now

	// 3. 深拷贝用户信息
	userCopy := &models.UserInfo{
		UserID:     userInfo.UserID,
		FirstUsed:  userInfo.FirstUsed,
		LastActive: userInfo.LastActive,
		CreatedAt:  userInfo.CreatedAt,
		UpdatedAt:  userInfo.UpdatedAt,
	}

	// 深拷贝DeviceInfo
	if userInfo.DeviceInfo != nil {
		userCopy.DeviceInfo = make(map[string]interface{})
		for k, v := range userInfo.DeviceInfo {
			userCopy.DeviceInfo[k] = v
		}
	}

	// 深拷贝Metadata
	if userInfo.Metadata != nil {
		userCopy.Metadata = make(map[string]interface{})
		for k, v := range userInfo.Metadata {
			userCopy.Metadata[k] = v
		}
	}

	// 4. 存储用户信息
	repo.users[userInfo.UserID] = userCopy

	log.Printf("✅ [用户仓库-内存] 用户创建成功: %s, 当前用户总数: %d", userInfo.UserID, len(repo.users))
	return nil
}

// UpdateUser 更新用户信息
func (repo *MemoryUserRepository) UpdateUser(userInfo *models.UserInfo) error {
	log.Printf("🧠 [用户仓库-内存] ===== 开始更新用户: %s =====", userInfo.UserID)

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// 1. 检查用户是否存在
	existingUser, exists := repo.users[userInfo.UserID]
	if !exists {
		log.Printf("❌ [用户仓库-内存] 用户不存在: %s", userInfo.UserID)
		return fmt.Errorf("用户不存在: %s", userInfo.UserID)
	}

	// 2. 保留原创建时间，更新其他信息
	userInfo.CreatedAt = existingUser.CreatedAt
	userInfo.UpdatedAt = time.Now().Format(time.RFC3339)

	// 3. 深拷贝更新的用户信息
	userCopy := &models.UserInfo{
		UserID:     userInfo.UserID,
		FirstUsed:  userInfo.FirstUsed,
		LastActive: userInfo.LastActive,
		CreatedAt:  userInfo.CreatedAt,
		UpdatedAt:  userInfo.UpdatedAt,
	}

	// 深拷贝DeviceInfo
	if userInfo.DeviceInfo != nil {
		userCopy.DeviceInfo = make(map[string]interface{})
		for k, v := range userInfo.DeviceInfo {
			userCopy.DeviceInfo[k] = v
		}
	}

	// 深拷贝Metadata
	if userInfo.Metadata != nil {
		userCopy.Metadata = make(map[string]interface{})
		for k, v := range userInfo.Metadata {
			userCopy.Metadata[k] = v
		}
	}

	// 4. 更新存储
	repo.users[userInfo.UserID] = userCopy

	log.Printf("✅ [用户仓库-内存] 用户更新成功: %s", userInfo.UserID)
	return nil
}

// GetUser 根据用户ID获取用户信息
func (repo *MemoryUserRepository) GetUser(userID string) (*models.UserInfo, error) {
	log.Printf("🔍 [用户仓库-内存] 开始查询用户: %s", userID)

	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	user, exists := repo.users[userID]
	if !exists {
		log.Printf("⚠️ [用户仓库-内存] 用户不存在: %s", userID)
		return nil, nil
	}

	// 返回深拷贝，避免外部修改
	userCopy := &models.UserInfo{
		UserID:     user.UserID,
		FirstUsed:  user.FirstUsed,
		LastActive: user.LastActive,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	// 深拷贝DeviceInfo
	if user.DeviceInfo != nil {
		userCopy.DeviceInfo = make(map[string]interface{})
		for k, v := range user.DeviceInfo {
			userCopy.DeviceInfo[k] = v
		}
	}

	// 深拷贝Metadata
	if user.Metadata != nil {
		userCopy.Metadata = make(map[string]interface{})
		for k, v := range user.Metadata {
			userCopy.Metadata[k] = v
		}
	}

	log.Printf("✅ [用户仓库-内存] 用户查询成功: %s", userID)
	return userCopy, nil
}

// CheckUserExists 检查用户是否存在
func (repo *MemoryUserRepository) CheckUserExists(userID string) (bool, error) {
	log.Printf("🔍 [用户仓库-内存] 检查用户存在性: %s", userID)

	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	_, exists := repo.users[userID]
	log.Printf("📊 [用户仓库-内存] 用户存在性检查结果: %s -> 存在: %t", userID, exists)
	return exists, nil
}

// InitRepository 初始化存储库
func (repo *MemoryUserRepository) InitRepository() error {
	log.Printf("🧠 [用户仓库-内存] 开始初始化内存用户存储库")

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// 内存版本无需特殊初始化，只清空现有数据
	repo.users = make(map[string]*models.UserInfo)

	log.Printf("✅ [用户仓库-内存] 内存用户存储库初始化成功")
	return nil
}

// GetUserCount 获取用户总数（额外功能，用于调试）
func (repo *MemoryUserRepository) GetUserCount() int {
	repo.mutex.RLock()
	defer repo.mutex.RUnlock()
	return len(repo.users)
}

// ListUsers 列出所有用户（额外功能，用于调试）
func (repo *MemoryUserRepository) ListUsers() []*models.UserInfo {
	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	users := make([]*models.UserInfo, 0, len(repo.users))
	for _, user := range repo.users {
		// 返回深拷贝
		userCopy := &models.UserInfo{
			UserID:     user.UserID,
			FirstUsed:  user.FirstUsed,
			LastActive: user.LastActive,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		}

		if user.DeviceInfo != nil {
			userCopy.DeviceInfo = make(map[string]interface{})
			for k, v := range user.DeviceInfo {
				userCopy.DeviceInfo[k] = v
			}
		}

		if user.Metadata != nil {
			userCopy.Metadata = make(map[string]interface{})
			for k, v := range user.Metadata {
				userCopy.Metadata[k] = v
			}
		}

		users = append(users, userCopy)
	}

	return users
}
