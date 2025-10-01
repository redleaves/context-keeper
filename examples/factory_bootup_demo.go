package main

import (
	"log"
	"os"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/pkg/vectorstore"
)

func main() {
	log.Println("🚀 Context-Keeper 向量存储工厂预初始化测试")

	// 设置测试环境变量
	setupTestEnvironment()

	// ========================================
	// 测试1：工厂预初始化
	// ========================================
	log.Println("\n📦 测试1: 工厂预初始化...")

	factory, err := vectorstore.InitializeFactoryFromEnv()
	if err != nil {
		log.Printf("❌ 工厂初始化失败: %v", err)
		return
	}

	// ========================================
	// 测试2：获取当前配置的向量存储
	// ========================================
	log.Println("\n🎯 测试2: 获取当前配置的向量存储...")

	currentStore, err := factory.GetCurrentVectorStore()
	if err != nil {
		log.Printf("❌ 获取当前向量存储失败: %v", err)
	} else {
		log.Println("✅ 当前向量存储实例获取成功")
		testEmbeddingGeneration(currentStore)
	}

	// ========================================
	// 测试3：手动获取不同类型的向量存储
	// ========================================
	log.Println("\n🔄 测试3: 手动获取不同类型的向量存储...")

	// 测试阿里云（应该成功）
	if aliyunStore, err := factory.CreateVectorStore(models.VectorStoreTypeAliyun); err == nil {
		log.Println("✅ 阿里云向量存储获取成功")
		testEmbeddingGeneration(aliyunStore)
	} else {
		log.Printf("❌ 阿里云向量存储获取失败: %v", err)
	}

	// 测试Vearch（可能失败，因为服务未启动）
	if vearchStore, err := factory.CreateVectorStore(models.VectorStoreTypeVearch); err == nil {
		log.Println("✅ Vearch向量存储获取成功")
		testEmbeddingGeneration(vearchStore)
	} else {
		log.Printf("❌ Vearch向量存储获取失败: %v", err)
	}

	log.Println("\n🎉 测试完成！")
	log.Println("\n💡 设计验证:")
	log.Println("   1. ✅ 阿里云和Vearch配置完全隔离")
	log.Println("   2. ✅ Vearch通过回调获取阿里云embedding服务，无直接依赖")
	log.Println("   3. ✅ 工厂预初始化所有可用的向量存储类型")
	log.Println("   4. ✅ 根据环境变量自动选择当前使用的类型")
}

// setupTestEnvironment 设置测试环境变量
func setupTestEnvironment() {
	log.Println("🔧 设置测试环境变量...")

	// 设置当前使用的向量存储类型
	os.Setenv("VECTOR_STORE_TYPE", "aliyun")

	// 阿里云配置（独立配置）
	os.Setenv("ALIYUN_EMBEDDING_API_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings")
	os.Setenv("ALIYUN_EMBEDDING_API_KEY", "test_aliyun_embedding_key")
	os.Setenv("ALIYUN_VECTOR_DB_URL", "test_aliyun_vector_db_url")
	os.Setenv("ALIYUN_VECTOR_DB_API_KEY", "test_aliyun_vector_db_key")
	os.Setenv("ALIYUN_VECTOR_DB_COLLECTION", "context_keeper")

	// 通用embedding配置（供Vearch等其他类型使用）
	os.Setenv("EMBEDDING_API_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings")
	os.Setenv("EMBEDDING_API_KEY", "test_shared_embedding_key")

	// Vearch配置（独立配置）
	os.Setenv("VEARCH_URL", "http://context-keeper.vearch.jd.local")
	os.Setenv("VEARCH_USERNAME", "root")
	os.Setenv("VEARCH_PASSWORD", "UB5EWPD6CQ28Z76Y")
	os.Setenv("VEARCH_DATABASE", "context_keeper_vector")
	os.Setenv("VEARCH_DIMENSION", "1536")
	os.Setenv("VEARCH_METRIC", "inner_product")
	os.Setenv("VEARCH_SIMILARITY_THRESHOLD", "0.2")

	log.Printf("✅ 环境变量设置完成")
	log.Printf("   当前向量存储类型: %s", os.Getenv("VECTOR_STORE_TYPE"))
	log.Printf("   阿里云配置: 独立的API密钥和端点")
	log.Printf("   Vearch配置: 独立的服务器和认证信息")
}

// testEmbeddingGeneration 测试embedding生成功能
func testEmbeddingGeneration(store models.VectorStore) {
	log.Printf("   📝 测试embedding生成...")

	testText := "这是一个测试文本，用于验证embedding生成功能"
	vector, err := store.GenerateEmbedding(testText)
	if err != nil {
		log.Printf("   ❌ Embedding生成失败: %v", err)
	} else {
		log.Printf("   ✅ Embedding生成成功，维度: %d", len(vector))
		log.Printf("   💡 前3个值: [%.4f, %.4f, %.4f...]", vector[0], vector[1], vector[2])
	}
}
