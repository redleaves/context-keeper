package services

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/contextkeeper/service/internal/config"
	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/store"
	"github.com/contextkeeper/service/pkg/aliyun"
	"github.com/contextkeeper/service/pkg/vectorstore"
)

// UserRepositoryType 用户存储类型
type UserRepositoryType string

const (
	// UserRepositoryTypeAliyun 阿里云向量存储
	UserRepositoryTypeAliyun UserRepositoryType = "aliyun"
	// UserRepositoryTypeVearch Vearch向量存储
	UserRepositoryTypeVearch UserRepositoryType = "vearch"
	// UserRepositoryTypeMemory 内存存储
	UserRepositoryTypeMemory UserRepositoryType = "memory"
	// UserRepositoryTypeMySQL MySQL数据库存储（预留）
	UserRepositoryTypeMySQL UserRepositoryType = "mysql"
	// UserRepositoryTypeTencent 腾讯云向量存储（预留）
	UserRepositoryTypeTencent UserRepositoryType = "tencent"
)

// UserRepositoryFactory 用户存储工厂
type UserRepositoryFactory struct {
	repositoryType UserRepositoryType
	vectorService  *aliyun.VectorService
	vearchClient   vectorstore.VearchClient
}

// NewUserRepositoryFactory 创建用户存储工厂
func NewUserRepositoryFactory(repositoryType UserRepositoryType, vectorService *aliyun.VectorService) *UserRepositoryFactory {
	return &UserRepositoryFactory{
		repositoryType: repositoryType,
		vectorService:  vectorService,
	}
}

// NewUserRepositoryFactoryWithVearch 创建支持Vearch的用户存储工厂
func NewUserRepositoryFactoryWithVearch(repositoryType UserRepositoryType, vectorService *aliyun.VectorService, vearchClient vectorstore.VearchClient) *UserRepositoryFactory {
	return &UserRepositoryFactory{
		repositoryType: repositoryType,
		vectorService:  vectorService,
		vearchClient:   vearchClient,
	}
}

// CreateUserRepository 创建用户存储仓库实例
func (factory *UserRepositoryFactory) CreateUserRepository() (models.UserRepository, error) {
	log.Printf("🏭 [用户仓库工厂] 创建用户存储实例，类型: %s", factory.repositoryType)

	switch factory.repositoryType {
	case UserRepositoryTypeAliyun:
		if factory.vectorService == nil {
			return nil, fmt.Errorf("阿里云向量服务未配置")
		}
		log.Printf("✅ [用户仓库工厂] 创建阿里云向量存储用户仓库")
		return aliyun.NewAliyunVectorUserRepository(factory.vectorService), nil

	case UserRepositoryTypeVearch:
		if factory.vearchClient == nil {
			return nil, fmt.Errorf("Vearch客户端未配置")
		}
		log.Printf("✅ [用户仓库工厂] 创建Vearch向量存储用户仓库")
		return vectorstore.NewVearchUserRepository(factory.vearchClient), nil

	case UserRepositoryTypeMemory:
		log.Printf("✅ [用户仓库工厂] 创建内存存储用户仓库")
		return store.NewMemoryUserRepository(), nil

	case UserRepositoryTypeMySQL:
		log.Printf("⚠️ [用户仓库工厂] MySQL存储尚未实现，回退到内存存储")
		return store.NewMemoryUserRepository(), nil

	case UserRepositoryTypeTencent:
		log.Printf("⚠️ [用户仓库工厂] 腾讯云向量存储尚未实现，回退到内存存储")
		return store.NewMemoryUserRepository(), nil

	default:
		log.Printf("❌ [用户仓库工厂] 未知的存储类型: %s，回退到内存存储", factory.repositoryType)
		return store.NewMemoryUserRepository(), nil
	}
}

// GetRepositoryTypeFromConfig 从配置中获取存储类型
func GetRepositoryTypeFromConfig() UserRepositoryType {
	// 1. 优先从配置文件读取（通过config包的Load方法）
	cfg := config.Load()
	if cfg.UserRepositoryType != "" {
		configType := strings.ToLower(strings.TrimSpace(cfg.UserRepositoryType))
		log.Printf("📋 [用户仓库工厂] 从配置文件读取存储类型: %s", configType)

		switch configType {
		case "aliyun":
			return UserRepositoryTypeAliyun
		case "vearch":
			return UserRepositoryTypeVearch
		case "memory":
			return UserRepositoryTypeMemory
		case "mysql":
			return UserRepositoryTypeMySQL
		case "tencent":
			return UserRepositoryTypeTencent
		default:
			log.Printf("⚠️ [用户仓库工厂] 配置文件中的存储类型 '%s' 不被支持，尝试环境变量", configType)
		}
	}

	// 2. 兜底从环境变量读取存储类型配置
	repoType := os.Getenv("USER_REPOSITORY_TYPE")
	if repoType == "" {
		repoType = "aliyun" // 默认使用阿里云向量存储
	}

	repoType = strings.ToLower(strings.TrimSpace(repoType))
	log.Printf("📋 [用户仓库工厂] 从环境变量读取存储类型: %s", repoType)

	switch repoType {
	case "aliyun":
		return UserRepositoryTypeAliyun
	case "vearch":
		return UserRepositoryTypeVearch
	case "memory":
		return UserRepositoryTypeMemory
	case "mysql":
		return UserRepositoryTypeMySQL
	case "tencent":
		return UserRepositoryTypeTencent
	default:
		log.Printf("⚠️ [用户仓库工厂] 配置的存储类型 '%s' 不被支持，使用默认: aliyun", repoType)
		return UserRepositoryTypeAliyun
	}
}

// GetAvailableRepositoryTypes 获取所有支持的存储类型
func GetAvailableRepositoryTypes() []UserRepositoryType {
	return []UserRepositoryType{
		UserRepositoryTypeAliyun,
		UserRepositoryTypeVearch,
		UserRepositoryTypeMemory,
		UserRepositoryTypeMySQL,
		UserRepositoryTypeTencent,
	}
}

// ValidateRepositoryType 验证存储类型是否有效
func ValidateRepositoryType(repoType UserRepositoryType) bool {
	availableTypes := GetAvailableRepositoryTypes()
	for _, availableType := range availableTypes {
		if repoType == availableType {
			return true
		}
	}
	return false
}

// GetRepositoryTypeDescription 获取存储类型的描述
func GetRepositoryTypeDescription(repoType UserRepositoryType) string {
	switch repoType {
	case UserRepositoryTypeAliyun:
		return "阿里云向量存储 - 生产环境推荐，支持海量数据和高并发"
	case UserRepositoryTypeVearch:
		return "Vearch向量存储 - 开源向量数据库，支持混合检索"
	case UserRepositoryTypeMemory:
		return "内存存储 - 测试环境使用，高性能但数据不持久化"
	case UserRepositoryTypeMySQL:
		return "MySQL数据库存储 - 传统关系型数据库（待实现）"
	case UserRepositoryTypeTencent:
		return "腾讯云向量存储 - 多云部署选择（待实现）"
	default:
		return "未知存储类型"
	}
}

// CreateUserRepositoryWithAutoDetection 根据环境自动检测并创建最适合的用户存储
// 支持传入不同类型的客户端：aliyun.VectorService, vectorstore.VearchClient, 或 nil
func CreateUserRepositoryWithAutoDetection(client interface{}) (models.UserRepository, error) {
	log.Printf("🤖 [用户仓库工厂] 开始自动检测最适合的用户存储类型")

	// 🔍 调试日志：检查传入的客户端类型
	if client == nil {
		log.Printf("❌ [用户仓库工厂] 传入的客户端为nil")
	} else {
		switch client.(type) {
		case *aliyun.VectorService:
			log.Printf("✅ [用户仓库工厂] 传入的是阿里云向量服务")
		case vectorstore.VearchClient:
			log.Printf("✅ [用户仓库工厂] 传入的是Vearch客户端")
		default:
			log.Printf("⚠️ [用户仓库工厂] 传入的客户端类型未知: %T", client)
		}
	}

	// 1. 优先从配置文件读取
	configType := GetRepositoryTypeFromConfig()
	log.Printf("📋 [用户仓库工厂] 配置指定的存储类型: %s", configType)

	// 2. 根据配置类型和传入的客户端类型创建相应的存储
	switch configType {
	case UserRepositoryTypeAliyun:
		log.Printf("🔍 [用户仓库工厂] 配置要求使用阿里云存储，检查向量服务可用性...")
		if vectorService, ok := client.(*aliyun.VectorService); ok && vectorService != nil {
			log.Printf("✅ [用户仓库工厂] 阿里云向量服务可用，使用阿里云存储")
			factory := NewUserRepositoryFactory(UserRepositoryTypeAliyun, vectorService)
			return factory.CreateUserRepository()
		} else {
			log.Printf("⚠️ [用户仓库工厂] 阿里云向量服务不可用，回退到内存存储")
			factory := NewUserRepositoryFactory(UserRepositoryTypeMemory, nil)
			return factory.CreateUserRepository()
		}

	case UserRepositoryTypeVearch:
		log.Printf("🔍 [用户仓库工厂] 配置要求使用Vearch存储，检查Vearch客户端可用性...")
		if vearchClient, ok := client.(vectorstore.VearchClient); ok && vearchClient != nil {
			log.Printf("✅ [用户仓库工厂] Vearch客户端可用，使用Vearch存储")
			factory := NewUserRepositoryFactoryWithVearch(UserRepositoryTypeVearch, nil, vearchClient)
			return factory.CreateUserRepository()
		} else {
			log.Printf("⚠️ [用户仓库工厂] Vearch客户端不可用，回退到内存存储")
			factory := NewUserRepositoryFactory(UserRepositoryTypeMemory, nil)
			return factory.CreateUserRepository()
		}

	default:
		log.Printf("🔧 [用户仓库工厂] 使用配置的存储类型: %s", configType)
		// 对于其他类型（memory, mysql, tencent），不需要特殊的客户端
		var vectorService *aliyun.VectorService
		if vs, ok := client.(*aliyun.VectorService); ok {
			vectorService = vs
		}
		factory := NewUserRepositoryFactory(configType, vectorService)
		return factory.CreateUserRepository()
	}
}

// TestRepositoryConnection 测试存储连接是否正常
func TestRepositoryConnection(repo models.UserRepository) error {
	log.Printf("🔧 [用户仓库工厂] 开始测试存储连接")

	// 1. 初始化存储库
	if err := repo.InitRepository(); err != nil {
		return fmt.Errorf("初始化存储库失败: %w", err)
	}

	// 2. 测试用户创建和查询
	testUserID := "factory_test_user_123"
	testUser := &models.UserInfo{
		UserID:     testUserID,
		FirstUsed:  "2025-01-06T00:00:00Z",
		LastActive: "2025-01-06T00:00:00Z",
		DeviceInfo: map[string]interface{}{
			"platform": "test",
			"version":  "1.0.0",
		},
		Metadata: map[string]interface{}{
			"test":   true,
			"source": "factory_test",
		},
	}

	// 3. 创建测试用户
	if err := repo.CreateUser(testUser); err != nil {
		return fmt.Errorf("创建测试用户失败: %w", err)
	}

	// 4. 查询测试用户
	retrievedUser, err := repo.GetUser(testUserID)
	if err != nil {
		return fmt.Errorf("查询测试用户失败: %w", err)
	}

	if retrievedUser == nil {
		return fmt.Errorf("测试用户未找到")
	}

	if retrievedUser.UserID != testUserID {
		return fmt.Errorf("测试用户ID不匹配: 期望 %s, 实际 %s", testUserID, retrievedUser.UserID)
	}

	log.Printf("✅ [用户仓库工厂] 存储连接测试成功")
	return nil
}
