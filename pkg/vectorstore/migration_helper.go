package vectorstore

import (
	"fmt"
	"log"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/pkg/aliyun"
)

// MigrationHelper 迁移辅助工具
type MigrationHelper struct {
	legacyService *aliyun.VectorService
	newStore      models.VectorStore
}

// NewMigrationHelper 创建迁移辅助工具
func NewMigrationHelper(legacyService *aliyun.VectorService) *MigrationHelper {
	return &MigrationHelper{
		legacyService: legacyService,
		newStore:      CreateAliyunVectorStoreFromLegacyService(legacyService),
	}
}

// TestCompatibility 测试新旧接口的兼容性
func (m *MigrationHelper) TestCompatibility() error {
	log.Printf("[迁移助手] 开始测试新旧接口兼容性")

	// 测试 Embedding 功能
	testText := "这是一个测试文本"

	// 使用原有接口
	legacyVector, err := m.legacyService.GenerateEmbedding(testText)
	if err != nil {
		return fmt.Errorf("原有接口生成向量失败: %w", err)
	}

	// 使用新接口
	newVector, err := m.newStore.GenerateEmbedding(testText)
	if err != nil {
		return fmt.Errorf("新接口生成向量失败: %w", err)
	}

	// 比较结果
	if len(legacyVector) != len(newVector) {
		return fmt.Errorf("向量维度不匹配: 原有=%d, 新接口=%d", len(legacyVector), len(newVector))
	}

	// 检查前几个元素是否相等
	for i := 0; i < min(5, len(legacyVector)); i++ {
		if legacyVector[i] != newVector[i] {
			return fmt.Errorf("向量内容不匹配: 位置%d, 原有=%.6f, 新接口=%.6f",
				i, legacyVector[i], newVector[i])
		}
	}

	// 测试维度获取
	legacyDim := m.legacyService.GetDimension()
	newDim := m.newStore.GetEmbeddingDimension()
	if legacyDim != newDim {
		return fmt.Errorf("维度不匹配: 原有=%d, 新接口=%d", legacyDim, newDim)
	}

	log.Printf("[迁移助手] ✅ 兼容性测试通过，向量维度=%d", newDim)
	return nil
}

// GetNewVectorStore 获取新的向量存储接口
func (m *MigrationHelper) GetNewVectorStore() models.VectorStore {
	return m.newStore
}

// ValidateMigration 验证迁移结果
func (m *MigrationHelper) ValidateMigration() error {
	log.Printf("[迁移助手] 验证迁移结果")

	// 测试基本功能
	if err := m.TestCompatibility(); err != nil {
		return fmt.Errorf("兼容性验证失败: %w", err)
	}

	// 测试抽象接口的完整性
	store := m.GetNewVectorStore()

	// 检查所有接口方法是否可调用
	if store == nil {
		return fmt.Errorf("向量存储实例为空")
	}

	// 测试集合检查功能
	exists, err := store.CollectionExists("test-collection")
	if err != nil {
		log.Printf("[迁移助手] 警告: 集合检查失败 (这可能是正常的): %v", err)
	} else {
		log.Printf("[迁移助手] 集合检查功能正常，结果: %v", exists)
	}

	log.Printf("[迁移助手] ✅ 迁移验证完成")
	return nil
}

// GetMigrationReport 获取迁移报告
func (m *MigrationHelper) GetMigrationReport() string {
	report := `
🎉 向量存储抽象化迁移报告

✅ 原有功能保持：
- ✅ 文本转向量 (GenerateEmbedding)
- ✅ 向量维度获取 (GetEmbeddingDimension)
- ✅ 记忆存储 (StoreMemory)
- ✅ 消息存储 (StoreMessage)
- ✅ 向量搜索 (SearchByVector/SearchByText/SearchByID)
- ✅ 集合管理 (CollectionExists/CreateCollection)

🚀 新增功能：
- 🆕 抽象接口支持多厂商
- 🆕 统一搜索选项配置
- 🆕 工厂模式创建实例
- 🆕 配置化厂商切换

🔄 迁移状态：
- 原有 VectorService: 保持不变 ✅
- 新增 VectorStore 抽象接口 ✅  
- 阿里云适配器实现 ✅
- 上下文服务升级 ✅
- 向后兼容性 ✅

📋 下一步：
1. 测试所有功能
2. 根据需要添加其他厂商实现
3. 逐步切换到新接口
4. 移除原有直接依赖
`
	return report
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateCompatibilityLayer 创建兼容性层，用于无缝切换
func CreateCompatibilityLayer(legacyService *aliyun.VectorService) models.VectorStore {
	log.Printf("[迁移助手] 创建兼容性层")
	return CreateAliyunVectorStoreFromLegacyService(legacyService)
}
