package llm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFile 加载.env文件
func LoadEnvFile(envPath string) error {
	// 如果没有指定路径，尝试查找.env文件
	if envPath == "" {
		var err error
		envPath, err = findEnvFile()
		if err != nil {
			return err
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("config/.env文件不存在: %s", envPath)
	}

	// 读取文件
	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("打开.env文件失败: %w", err)
	}
	defer file.Close()

	// 逐行解析
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("config/.env文件第%d行格式错误: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除引号
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// 设置环境变量（只有当前环境变量不存在时才设置）
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取config/.env文件失败: %w", err)
	}

	return nil
}

// findEnvFile 查找.env文件
func findEnvFile() (string, error) {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 向上查找config/.env文件
	for {
		envPath := filepath.Join(wd, "config", ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", fmt.Errorf("未找到config/.env文件")
}

// InitializeWithEnv 使用环境变量初始化
func InitializeWithEnv() error {
	// 1. 加载config/.env文件
	if err := LoadEnvFile(""); err != nil {
		// config/.env文件不存在不是致命错误，可能环境变量已经设置
		fmt.Printf("Warning: %v\n", err)
	}

	// 2. 查找配置文件
	configPath, err := FindConfigFile()
	if err != nil {
		return fmt.Errorf("查找配置文件失败: %w", err)
	}

	// 3. 初始化配置
	return InitializeFromConfig(configPath)
}
