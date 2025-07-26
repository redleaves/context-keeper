package aliyun

import (
	"fmt"
	"log"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// AliyunVectorUserRepository 阿里云向量存储的用户信息存储实现
type AliyunVectorUserRepository struct {
	vectorService  *VectorService
	collectionName string
}

// NewAliyunVectorUserRepository 创建阿里云向量存储用户仓库实例
func NewAliyunVectorUserRepository(vectorService *VectorService) models.UserRepository {
	return &AliyunVectorUserRepository{
		vectorService:  vectorService,
		collectionName: UserCollectionName,
	}
}

// CreateUser 创建新用户
func (repo *AliyunVectorUserRepository) CreateUser(userInfo *models.UserInfo) error {
	log.Printf("🔥 [用户仓库-阿里云] ===== 开始创建用户: %s =====", userInfo.UserID)

	// 1. 检查用户是否已存在
	exists, err := repo.CheckUserExists(userInfo.UserID)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 检查用户存在性失败: %v", err)
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}
	if exists {
		log.Printf("❌ [用户仓库-阿里云] 用户已存在: %s", userInfo.UserID)
		return fmt.Errorf("用户ID已存在: %s", userInfo.UserID)
	}

	// 2. 设置时间戳
	now := time.Now().Format(time.RFC3339)
	if userInfo.CreatedAt == "" {
		userInfo.CreatedAt = now
	}
	userInfo.UpdatedAt = now

	// 3. 调用向量服务存储
	err = repo.vectorService.StoreUserInfo(userInfo)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 存储用户信息失败: %v", err)
		return fmt.Errorf("存储用户信息失败: %w", err)
	}

	log.Printf("✅ [用户仓库-阿里云] 用户创建成功: %s", userInfo.UserID)
	return nil
}

// UpdateUser 更新用户信息
func (repo *AliyunVectorUserRepository) UpdateUser(userInfo *models.UserInfo) error {
	log.Printf("🔥 [用户仓库-阿里云] ===== 开始更新用户: %s =====", userInfo.UserID)

	// 1. 检查用户是否存在
	exists, err := repo.CheckUserExists(userInfo.UserID)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 检查用户存在性失败: %v", err)
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}
	if !exists {
		log.Printf("❌ [用户仓库-阿里云] 用户不存在: %s", userInfo.UserID)
		return fmt.Errorf("用户不存在: %s", userInfo.UserID)
	}

	// 2. 获取现有用户信息（保留创建时间）
	existingUser, err := repo.GetUser(userInfo.UserID)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 获取现有用户信息失败: %v", err)
		return fmt.Errorf("获取现有用户信息失败: %w", err)
	}
	if existingUser != nil {
		userInfo.CreatedAt = existingUser.CreatedAt // 保留原创建时间
	}

	// 3. 设置更新时间
	userInfo.UpdatedAt = time.Now().Format(time.RFC3339)

	// 4. 调用向量服务更新（实际是重新存储）
	err = repo.vectorService.StoreUserInfo(userInfo)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 更新用户信息失败: %v", err)
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	log.Printf("✅ [用户仓库-阿里云] 用户更新成功: %s", userInfo.UserID)
	return nil
}

// GetUser 根据用户ID获取用户信息
func (repo *AliyunVectorUserRepository) GetUser(userID string) (*models.UserInfo, error) {
	log.Printf("🔍 [用户仓库-阿里云] 开始查询用户: %s", userID)

	userInfo, err := repo.vectorService.GetUserInfo(userID)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 查询用户失败: %v", err)
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	if userInfo == nil {
		log.Printf("⚠️ [用户仓库-阿里云] 用户不存在: %s", userID)
		return nil, nil
	}

	log.Printf("✅ [用户仓库-阿里云] 用户查询成功: %s", userID)
	return userInfo, nil
}

// CheckUserExists 检查用户是否存在
func (repo *AliyunVectorUserRepository) CheckUserExists(userID string) (bool, error) {
	log.Printf("🔍 [用户仓库-阿里云] 检查用户存在性: %s", userID)

	// 直接使用GetUser方法来检查用户是否存在
	userInfo, err := repo.vectorService.GetUserInfo(userID)
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 检查用户存在性失败: %v", err)
		return false, fmt.Errorf("检查用户存在性失败: %w", err)
	}

	exists := userInfo != nil
	log.Printf("📊 [用户仓库-阿里云] 用户存在性检查结果: %s -> 存在: %t", userID, exists)
	return exists, nil
}

// InitRepository 初始化存储库
func (repo *AliyunVectorUserRepository) InitRepository() error {
	log.Printf("🔧 [用户仓库-阿里云] 开始初始化用户存储库")

	err := repo.vectorService.InitUserCollection()
	if err != nil {
		log.Printf("❌ [用户仓库-阿里云] 初始化用户集合失败: %v", err)
		return fmt.Errorf("初始化用户集合失败: %w", err)
	}

	log.Printf("✅ [用户仓库-阿里云] 用户存储库初始化成功")
	return nil
}
