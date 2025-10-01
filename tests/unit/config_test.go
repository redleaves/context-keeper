package unit

import (
	"testing"

	"github.com/contextkeeper/service/internal/config"
)

func TestConfigLoad(t *testing.T) {
	// 加载配置
	cfg := config.Load()

	// 测试配置有效性
	if cfg == nil {
		t.Fatal("配置加载失败，返回了nil")
	}

	// 测试配置项是否有默认值
	if cfg.ServiceName == "" {
		t.Error("ServiceName应该有默认值")
	}

	if cfg.Port <= 0 {
		t.Error("Port应该大于0")
	}

	// 测试存储路径配置
	if cfg.StoragePath == "" {
		t.Error("StoragePath不应为空")
	}
}
