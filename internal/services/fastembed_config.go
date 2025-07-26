package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anush008/fastembed-go"
)

// 🔧 统一的FastEmbed配置管理
// 解决硬编码参数散落各处的问题，确保前后一致性

// FastEmbedModelConfig 模型配置结构
type FastEmbedModelConfig struct {
	// 🎯 核心模型配置
	ModelType   fastembed.EmbeddingModel `json:"model_type"`  // 实际使用的枚举
	ModelName   string                   `json:"model_name"`  // 显示用的名称
	Description string                   `json:"description"` // 模型描述

	// 📊 模型参数
	MaxLength  int    `json:"max_length"` // 最大文本长度
	Dimension  int    `json:"dimension"`  // 向量维度
	Parameters string `json:"parameters"` // 参数量说明

	// 🚀 性能配置
	CacheDir  string `json:"cache_dir"`  // 缓存目录
	BatchSize int    `json:"batch_size"` // 批处理大小

	// 🔍 适用场景
	Languages []string `json:"languages"` // 支持语言
	UseCase   string   `json:"use_case"`  // 使用场景
}

// GetDefaultFastEmbedConfig 获取默认配置
// 🔥 所有FastEmbed参数的唯一真实来源（Single Source of Truth）
func GetDefaultFastEmbedConfig() *FastEmbedModelConfig {
	// 动态获取缓存目录
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "fastembed")

	return &FastEmbedModelConfig{
		// 🎯 核心配置 - 确保枚举和名称一致！
		ModelType:   fastembed.BGEBaseEN,
		ModelName:   "BAAI/bge-base-en-v1.5",
		Description: "BGE-Base-EN high-precision multilingual model",

		// 📊 技术参数
		MaxLength:  512,
		Dimension:  768,
		Parameters: "109M",

		// 🚀 运行时配置
		CacheDir:  cacheDir,
		BatchSize: 2,

		// 🔍 能力描述
		Languages: []string{"en", "zh", "multilingual"},
		UseCase:   "high_precision_semantic_similarity",
	}
}

// GetAlternativeConfigs 获取备用模型配置
// 🔄 支持动态切换不同模型
func GetAlternativeConfigs() map[string]*FastEmbedModelConfig {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "fastembed")

	return map[string]*FastEmbedModelConfig{
		"bge_base_en": {
			ModelType:   fastembed.BGEBaseEN,
			ModelName:   "BAAI/bge-base-en-v1.5",
			Description: "BGE-Base-EN high-precision model (Current)",
			MaxLength:   512,
			Dimension:   768,
			Parameters:  "109M",
			CacheDir:    cacheDir,
			BatchSize:   2,
			Languages:   []string{"en", "zh", "multilingual"},
			UseCase:     "high_precision",
		},
		"bge_small_en": {
			ModelType:   fastembed.BGESmallEN,
			ModelName:   "BAAI/bge-small-en-v1.5",
			Description: "BGE-Small-EN 平衡性能英文模型",
			MaxLength:   512,
			Dimension:   384,
			Parameters:  "33.4M",
			CacheDir:    cacheDir,
			BatchSize:   4,
			Languages:   []string{"en", "english"},
			UseCase:     "english_balanced",
		},
		"bge_small_zh": {
			ModelType:   fastembed.BGESmallZH,
			ModelName:   "BAAI/bge-small-zh-v1.5",
			Description: "BGE-Small-ZH 中文优化模型",
			MaxLength:   512,
			Dimension:   384,
			Parameters:  "33.4M",
			CacheDir:    cacheDir,
			BatchSize:   4,
			Languages:   []string{"zh", "chinese"},
			UseCase:     "chinese_optimized",
		},
		"all_minilm": {
			ModelType:   fastembed.AllMiniLML6V2,
			ModelName:   "sentence-transformers/all-MiniLM-L6-v2",
			Description: "All-MiniLM-L6-v2 轻量级英文模型",
			MaxLength:   256,
			Dimension:   384,
			Parameters:  "22.7M",
			CacheDir:    cacheDir,
			BatchSize:   8,
			Languages:   []string{"en", "english"},
			UseCase:     "lightweight_fast",
		},
	}
}

// ValidateConfig 验证配置一致性
// 🛡️ 确保枚举值和字符串名称匹配
func (config *FastEmbedModelConfig) ValidateConfig() error {
	// 验证模型类型和名称的一致性
	expectedNames := map[fastembed.EmbeddingModel]string{
		fastembed.BGEBaseEN:     "BAAI/bge-base-en-v1.5",
		fastembed.BGESmallEN:    "BAAI/bge-small-en-v1.5",
		fastembed.AllMiniLML6V2: "sentence-transformers/all-MiniLM-L6-v2",
	}

	if expectedName, exists := expectedNames[config.ModelType]; exists {
		if config.ModelName != expectedName {
			return fmt.Errorf("模型配置不一致: ModelType=%v 对应的名称应该是 '%s', 但实际是 '%s'",
				config.ModelType, expectedName, config.ModelName)
		}
	}

	// 验证必要参数
	if config.MaxLength <= 0 {
		return fmt.Errorf("MaxLength 必须大于 0")
	}

	if config.CacheDir == "" {
		return fmt.Errorf("CacheDir 不能为空")
	}

	return nil
}

// ToInitOptions 转换为FastEmbed初始化选项
// 🔄 统一的参数转换，避免重复硬编码
func (config *FastEmbedModelConfig) ToInitOptions() *fastembed.InitOptions {
	return &fastembed.InitOptions{
		Model:     config.ModelType,
		CacheDir:  config.CacheDir,
		MaxLength: config.MaxLength,
	}
}

// GetDisplayInfo 获取显示信息
// 📊 统一的模型信息展示
func (config *FastEmbedModelConfig) GetDisplayInfo() map[string]interface{} {
	return map[string]interface{}{
		"model_name":  config.ModelName,
		"description": config.Description,
		"max_length":  config.MaxLength,
		"dimension":   config.Dimension,
		"parameters":  config.Parameters,
		"languages":   config.Languages,
		"use_case":    config.UseCase,
		"cache_dir":   config.CacheDir,
	}
}

// 🎯 全局默认配置实例
var DefaultFastEmbedConfig = GetDefaultFastEmbedConfig()
