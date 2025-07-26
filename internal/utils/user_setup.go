package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// 用户引导界面样式常量
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
)

// 验证用户ID的方法
// 在实际生产环境中，这应该是一个与服务端API通信的函数
func verifyUserID(userID string) (string, error) {
	// 这里应该是通过API调用服务端验证用户ID
	// 验证格式：user_xxxxxxxx (支持8-20个字符的用户名部分)
	if !strings.HasPrefix(userID, "user_") || len(userID) < 13 || len(userID) > 25 {
		return "", fmt.Errorf("无效的用户ID格式")
	}

	// 实际应从服务端验证ID的有效性，这里简单返回输入的ID
	return userID, nil
}

// InitializeUserInteractive 交互式初始化用户
// 返回用户ID和是否是新用户的标志
func InitializeUserInteractive() (string, bool, error) {
	// 检查是否在MCP模式下运行
	if isMCPMode() {
		// 在MCP模式下，将通过对话方式初始化，先返回空值
		return "", false, nil
	}

	// 尝试加载现有配置
	config, err := LoadUserConfig()
	if err != nil {
		return "", false, fmt.Errorf("加载用户配置失败: %w", err)
	}

	// 如果已有配置，直接返回userId
	if config != nil && config.UserID != "" {
		return config.UserID, false, nil
	}

	// 交互式引导
	fmt.Println()
	fmt.Println(ColorCyan + "欢迎使用 Context Keeper 记忆功能！" + ColorReset)
	fmt.Println(ColorCyan + "-------------------------------------" + ColorReset)
	fmt.Println("我们需要创建一个用户ID，用于识别您的个人数据。")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(ColorYellow + "您是否在其他设备使用过 Context Keeper？(y/n): " + ColorReset)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	var userId string
	var isNewUser bool

	if answer == "y" || answer == "yes" {
		// 用户在其他设备使用过，提示输入用户ID
		fmt.Print(ColorYellow + "请输入您的用户ID(格式如 user_xxxxxxxx): " + ColorReset)
		inputUserID, _ := reader.ReadString('\n')
		inputUserID = strings.TrimSpace(inputUserID)

		// 验证用户ID
		verifiedUserId, err := verifyUserID(inputUserID)
		if err != nil {
			fmt.Println(ColorRed + "❌ 无效的用户ID，将为您创建新用户ID。" + ColorReset)
			userId, err = createNewUserConfig()
			if err != nil {
				return "", false, err
			}
			isNewUser = true
		} else {
			// 保存配置
			newConfig := &UserConfig{
				UserID:    verifiedUserId,
				FirstUsed: time.Now().Format(time.RFC3339),
			}

			if err := SaveUserConfig(newConfig); err != nil {
				return "", false, fmt.Errorf("保存用户配置失败: %w", err)
			}

			fmt.Println(ColorGreen + "✅ 用户ID验证成功！您的数据已准备就绪。" + ColorReset)
			userId = verifiedUserId
			isNewUser = false
		}
	} else {
		// 用户首次使用，创建新用户ID
		userId, err = createNewUserConfig()
		if err != nil {
			return "", false, err
		}
		isNewUser = true
	}

	return userId, isNewUser, nil
}

// createNewUserConfig 创建新的用户配置
func createNewUserConfig() (string, error) {
	// 生成唯一ID
	userId := GenerateUserID()

	// 创建配置
	newConfig := &UserConfig{
		UserID:    userId,
		FirstUsed: time.Now().Format(time.RFC3339),
	}

	// 保存到本地文件
	if err := SaveUserConfig(newConfig); err != nil {
		return "", fmt.Errorf("保存用户配置失败: %w", err)
	}

	// 向用户展示信息
	fmt.Println()
	fmt.Println(ColorGreen + "🎉 您的 Context Keeper 用户ID已创建！" + ColorReset)
	fmt.Println(ColorCyan + "-------------------------------------" + ColorReset)
	fmt.Printf(ColorYellow+"您的用户ID是: %s\n"+ColorReset, userId)
	fmt.Println()
	fmt.Println(ColorYellow + "⚠️ 重要提示:" + ColorReset)
	fmt.Println("1. 请妥善保管此用户ID，在您更换设备时需要输入它")
	fmt.Printf("2. 用户ID存储在您的主目录下: %s\n", getConfigFilePath())
	fmt.Println("3. 如果您忘记用户ID，可以随时查看此文件")
	fmt.Println(ColorCyan + "-------------------------------------" + ColorReset)
	fmt.Println()

	return userId, nil
}

// FindUserCredential 查找用户凭证
func FindUserCredential() string {
	config, err := LoadUserConfig()
	if err != nil {
		fmt.Printf(ColorRed+"❌ 获取用户ID失败: %v\n"+ColorReset, err)
		return ""
	}

	if config == nil {
		fmt.Println(ColorRed + "❌ 未找到用户配置文件。您可能尚未设置用户ID。" + ColorReset)
		return ""
	}

	fmt.Println()
	fmt.Println(ColorCyan + "您的用户信息:" + ColorReset)
	fmt.Println(ColorCyan + "-------------------------------------" + ColorReset)
	fmt.Printf(ColorYellow+"用户ID: %s\n"+ColorReset, config.UserID)
	fmt.Printf("首次使用时间: %s\n", config.FirstUsed)
	fmt.Println(ColorCyan + "-------------------------------------" + ColorReset)
	fmt.Println()

	return config.UserID
}

// isMCPMode 检测是否在MCP模式下运行
func isMCPMode() bool {
	// 检查环境变量
	if os.Getenv("MCP_MODE") == "true" {
		return true
	}

	// 检查命令行参数
	for _, arg := range os.Args {
		if arg == "--mcp" || arg == "-mcp" {
			return true
		}
	}

	// 默认判断：如果标准输入被重定向，则可能是MCP模式
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return true
	}

	return false
}
