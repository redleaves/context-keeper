package main

import (
	"log"
	"os"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/pkg/vectorstore"
)

func main() {
	log.Println("🚀 向量存储工厂模式演示")

	// 创建工厂实例
	factory := vectorstore.NewVectorStoreFactory()

	// 配置阿里云向量存储
	aliyunConfig := &models.VectorStoreConfig{
		Provider: string(models.VectorStoreTypeAliyun),
		EmbeddingConfig: &models.EmbeddingConfig{
			Model:       "text-embedding-ada-002",
			APIEndpoint: os.Getenv("EMBEDDING_API_URL"),
			APIKey:      os.Getenv("EMBEDDING_API_KEY"),
			Dimension:   1536,
		},
		DatabaseConfig: &models.DatabaseConfig{
			Endpoint:   os.Getenv("VECTOR_DB_URL"),
			APIKey:     os.Getenv("VECTOR_DB_API_KEY"),
			Collection: "context_keeper",
			Metric:     "inner_product",
		},
		SimilarityThreshold: 0.3,
	}

	// 配置Vearch向量存储
	vearchConfig := &models.VectorStoreConfig{
		Provider: string(models.VectorStoreTypeVearch),
		EmbeddingConfig: &models.EmbeddingConfig{
			Model:       "text-embedding-ada-002",
			APIEndpoint: os.Getenv("EMBEDDING_API_URL"),
			APIKey:      os.Getenv("EMBEDDING_API_KEY"),
			Dimension:   1536,
		},
		DatabaseConfig: &models.DatabaseConfig{
			Endpoint:   os.Getenv("VEARCH_URL"),
			Collection: os.Getenv("VEARCH_DATABASE"),
			Metric:     "inner_product",
			ExtraParams: map[string]interface{}{
				"username": os.Getenv("VEARCH_USERNAME"),
				"password": os.Getenv("VEARCH_PASSWORD"),
			},
		},
		SimilarityThreshold: 0.2,
	}

	// 注册配置
	factory.RegisterConfig(models.VectorStoreTypeAliyun, aliyunConfig)
	factory.RegisterConfig(models.VectorStoreTypeVearch, vearchConfig)

	// 预初始化所有实例
	log.Println("📦 开始预初始化所有向量存储实例...")
	if err := factory.InitializeAllInstances(); err != nil {
		log.Printf("❌ 初始化失败: %v", err)
		// 继续演示，可能只是某些服务不可用
	}

	// 演示使用预初始化的实例
	log.Println("\n🔍 演示使用预初始化的实例:")

	// 使用阿里云存储
	log.Println("📋 获取阿里云向量存储实例...")
	aliyunStore, err := factory.CreateVectorStore(models.VectorStoreTypeAliyun)
	if err != nil {
		log.Printf("❌ 创建阿里云存储失败: %v", err)
	} else {
		log.Println("✅ 阿里云存储实例获取成功")
		// 演示embedding生成
		if vector, err := aliyunStore.GenerateEmbedding("测试文本embedding"); err != nil {
			log.Printf("❌ 阿里云embedding生成失败: %v", err)
		} else {
			log.Printf("✅ 阿里云embedding生成成功，维度: %d", len(vector))
		}
	}

	// 使用Vearch存储
	log.Println("\n📋 获取Vearch向量存储实例...")
	vearchStore, err := factory.CreateVectorStore(models.VectorStoreTypeVearch)
	if err != nil {
		log.Printf("❌ 创建Vearch存储失败: %v", err)
	} else {
		log.Println("✅ Vearch存储实例获取成功")
		// 演示embedding生成（复用阿里云服务）
		if vector, err := vearchStore.GenerateEmbedding("测试文本embedding"); err != nil {
			log.Printf("❌ Vearch embedding生成失败: %v", err)
		} else {
			log.Printf("✅ Vearch embedding生成成功，维度: %d (复用阿里云服务)", len(vector))
		}
	}

	log.Println("\n🎉 向量存储工厂模式演示完成！")
	log.Println("\n💡 核心优势:")
	log.Println("   1. ♻️  复用阿里云embedding服务，避免重复代码")
	log.Println("   2. 🚀 预初始化所有实例，提升性能")
	log.Println("   3. 🔧 工厂模式统一管理，便于扩展")
	log.Println("   4. 🎯 Vearch与阿里云概念对齐，Space替代Collection")
}
