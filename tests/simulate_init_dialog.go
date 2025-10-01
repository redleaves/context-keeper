package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// UserConfig 用户配置
type UserConfig struct {
	UserID string `json:"userId"` // 用户唯一标识
}

// DialogState 对话状态
type DialogState struct {
	State   string // 当前状态
	UserID  string // 用户ID (如果已分配)
	Message string // 消息
}

func main() {
	// 设置日志输出
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("开始模拟user_init_dialog功能")

	// 获取配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户主目录失败: %v", err)
	}
	configDir := filepath.Join(homeDir, ".context-keeper")
	configPath := filepath.Join(configDir, "user-config.json")

	// 检查配置是否存在
	configExists := false
	_, err = os.Stat(configPath)
	if err == nil {
		configExists = true
		log.Printf("发现现有配置文件: %s", configPath)
	} else if os.IsNotExist(err) {
		log.Printf("配置文件不存在: %s", configPath)
	} else {
		log.Fatalf("检查配置文件失败: %v", err)
	}

	// 备份配置（如果存在）
	if configExists {
		backupPath := configPath + ".bak"
		if err := os.Rename(configPath, backupPath); err != nil {
			log.Fatalf("备份配置文件失败: %v", err)
		}
		log.Printf("✅ 已备份配置文件到: %s", backupPath)

		// 恢复配置的函数
		defer func() {
			if err := os.Rename(backupPath, configPath); err != nil {
				log.Printf("❌ 恢复配置文件失败: %v", err)
			} else {
				log.Printf("✅ 已恢复原配置文件")
			}
		}()
	} else {
		// 确保目录存在
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Fatalf("创建配置目录失败: %v", err)
		}
	}

	// 模拟阶段1: 初始化对话，没有用户响应
	state := DialogState{
		State:   "asking",
		Message: "您好，这是您首次使用上下文记忆管理工具。请问您是否已在其他设备上使用过该工具？",
	}
	printState("初始对话状态", state)

	// 模拟阶段2: 用户回答"否"，创建新用户
	state = handleResponse(state, "否，我是新用户", configPath)
	printState("新用户响应", state)

	// 提取新创建的用户ID
	userID := state.UserID

	if state.State == "new_user" {
		log.Printf("✅ 成功创建新用户! 用户ID: %s", userID)
	} else {
		log.Fatalf("❌ 新用户创建失败")
	}

	// 模拟阶段3: 再次初始化对话，检查是否有配置
	state = initializeUserDialog()
	printState("验证配置状态", state)

	if state.State == "completed" {
		log.Printf("✅ 用户配置已完成，测试通过!")
	} else {
		log.Fatalf("❌ 配置验证失败, 状态: %s", state.State)
	}

	// 模拟阶段4: 清除配置，测试已有用户流程
	log.Println("====================== 测试已有用户流程 ======================")

	// 删除当前配置，模拟新环境
	if err := os.Remove(configPath); err != nil {
		log.Fatalf("删除配置文件失败: %v", err)
	}
	log.Printf("✅ 已删除配置文件，模拟新环境")

	// 模拟阶段5: 初始化对话，没有用户响应（新环境）
	state = DialogState{
		State:   "asking",
		Message: "您好，这是您首次使用上下文记忆管理工具。请问您是否已在其他设备上使用过该工具？",
	}
	printState("新环境初始对话状态", state)

	// 模拟阶段6: 用户回答"是"，进入输入用户ID流程
	state = handleResponse(state, "是的，我已经在其他设备使用过", configPath)
	printState("已有用户响应", state)

	if state.State == "existing" {
		log.Printf("✅ 状态已变更为等待输入用户ID")
	} else {
		log.Fatalf("❌ 状态未变更为等待输入用户ID")
	}

	// 模拟阶段7: 用户输入用户ID
	state = handleResponse(state, fmt.Sprintf("我的用户ID是: %s", userID), configPath)
	printState("用户ID响应", state)

	if state.State == "completed" && state.UserID == userID {
		log.Printf("✅ 成功验证用户ID! 用户ID: %s", state.UserID)
	} else {
		log.Fatalf("❌ 用户ID验证失败, 状态: %s", state.State)
	}

	log.Println("用户ID验证测试完成!")
}

// 处理用户响应
func handleResponse(state DialogState, response string, configPath string) DialogState {
	switch state.State {
	case "asking":
		// 处理是否有用户ID的响应
		if containsAffirmative(response) {
			// 用户表示已在其他设备使用过
			state.State = "existing"
			state.Message = "请输入您的用户ID"
		} else {
			// 用户是新用户
			state.State = "new_user"
			state.UserID = generateUserID()
			state.Message = "已为您创建新用户账号"

			// 保存新用户配置
			err := saveUserConfig(configPath, &UserConfig{
				UserID: state.UserID,
			})
			if err != nil {
				log.Fatalf("保存用户配置失败: %v", err)
			}
		}
	case "existing":
		// 处理用户ID响应
		userID := extractUserID(response)
		if userID == "" {
			state.Message = "无法识别用户ID，请重新输入"
			return state
		}

		// 验证用户ID (简单实现，支持8-20个字符的用户名部分)
		if !strings.HasPrefix(userID, "user_") || len(userID) < 13 || len(userID) > 25 {
			state.Message = "用户ID格式无效，请重新输入"
			return state
		}

		// 更新状态
		state.UserID = userID
		state.State = "completed"
		state.Message = "用户配置已完成"

		// 保存用户配置
		err := saveUserConfig(configPath, &UserConfig{
			UserID: userID,
		})
		if err != nil {
			log.Fatalf("保存用户配置失败: %v", err)
		}
	}

	return state
}

// 初始化用户对话
func initializeUserDialog() DialogState {
	// 读取用户配置
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户主目录失败: %v", err)
	}
	configPath := filepath.Join(homeDir, ".context-keeper", "user-config.json")

	config, err := readUserConfig(configPath)
	if err == nil && config.UserID != "" {
		// 已有用户配置，直接返回完成状态
		return DialogState{
			State:   "completed",
			UserID:  config.UserID,
			Message: "用户配置已完成",
		}
	}

	// 无配置，返回初始询问状态
	return DialogState{
		State:   "asking",
		Message: "您好，这是您首次使用上下文记忆管理工具。请问您是否已在其他设备上使用过该工具？",
	}
}

// 读取用户配置
func readUserConfig(path string) (*UserConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，返回空配置
			return &UserConfig{}, nil
		}
		return nil, fmt.Errorf("读取用户配置失败: %w", err)
	}

	var config UserConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析用户配置失败: %w", err)
	}

	return &config, nil
}

// 保存用户配置
func saveUserConfig(path string, config *UserConfig) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入用户配置失败: %w", err)
	}

	log.Printf("已保存用户配置: UserID=%s", config.UserID)
	return nil
}

// 检测肯定回答
func containsAffirmative(response string) bool {
	return contains(response, []string{"是", "有", "对", "yes", "true", "正确"})
}

// 提取用户ID
func extractUserID(response string) string {
	// 匹配user_xxxxxxxx格式
	words := strings.Fields(response)
	for _, word := range words {
		word = strings.Trim(word, ",:;.\"'()[]{}") // 移除常见标点符号
		if strings.HasPrefix(word, "user_") && len(word) == 13 {
			return word
		}
	}
	return ""
}

// 生成用户ID
func generateUserID() string {
	return fmt.Sprintf("user_%s", randomString(8))
}

// 生成随机字符串
func randomString(length int, upperCase ...bool) string {
	const lowerChars = "abcdefghijklmnopqrstuvwxyz0123456789"
	const upperChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除容易混淆的字符

	chars := lowerChars
	if len(upperCase) > 0 && upperCase[0] {
		chars = upperChars
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}

// 辅助函数：检查字符串是否包含列表中的某个字符串
func contains(s string, substrs []string) bool {
	for _, sub := range substrs {
		if sub != "" && strings.Contains(strings.ToLower(s), strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// 辅助函数：打印状态
func printState(label string, state DialogState) {
	fmt.Printf("\n===== %s =====\n", label)
	fmt.Printf("状态: %s\n", state.State)
	fmt.Printf("消息: %s\n", state.Message)
	if state.UserID != "" {
		fmt.Printf("用户ID: %s\n", state.UserID)
	}
	fmt.Println("=====================")
}
