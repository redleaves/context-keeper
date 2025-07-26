package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath" // Added for reflect.ValueOf
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	// Added for models.Session

	"github.com/google/uuid"
)

const (
	// ConfigDirName 配置目录名称
	ConfigDirName = ".context-keeper"
	// ConfigFileName 用户配置文件名
	ConfigFileName = "user-config.json"
)

// UserConfig 用户配置信息
type UserConfig struct {
	UserID    string `json:"userId"`    // 用户唯一ID
	FirstUsed string `json:"firstUsed"` // 首次使用时间
}

// 全局缓存的用户ID，避免重复读取磁盘
var (
	cachedUserID   string
	cachedUserInfo *UserConfig
	userCacheMutex sync.Mutex
)

// 对话状态常量
const (
	DialogStateNone      = "none"      // 无状态
	DialogStateAsking    = "asking"    // 询问是否有用户凭证
	DialogStateNewUser   = "new_user"  // 新用户流程
	DialogStateExisting  = "existing"  // 已有用户输入用户ID
	DialogStateCompleted = "completed" // 配置完成
)

// DialogState 保存用户对话状态
type DialogState struct {
	State    string    // 当前状态
	UserID   string    // 用户ID (如果已分配)
	LastTime time.Time // 上次更新时间
}

// 存储会话状态，用于对话式初始化
var dialogStates = make(map[string]*DialogState) // sessionID -> DialogState

// InitUserCache 初始化用户缓存
// 应在程序启动时调用此方法，加载用户信息到内存
func InitUserCache() error {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()

	config, err := LoadUserConfig()
	if err != nil {
		return fmt.Errorf("初始化用户缓存失败: %w", err)
	}

	if config != nil && config.UserID != "" {
		cachedUserID = config.UserID
		cachedUserInfo = config
		log.Printf("用户缓存初始化成功: UserID=%s", cachedUserID)
		return nil
	}

	log.Printf("未找到有效的用户配置，需要通过对话初始化")
	return nil
}

// GetCachedUserID 获取缓存的用户ID
func GetCachedUserID() string {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()
	return cachedUserID
}

// SetCachedUserID 设置缓存的用户ID
func SetCachedUserID(userID string) {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()
	cachedUserID = userID
	log.Printf("更新缓存的用户ID: %s", userID)
}

// GetCachedUserInfo 获取缓存的用户信息
func GetCachedUserInfo() *UserConfig {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()
	return cachedUserInfo
}

// SetCachedUserInfo 设置缓存的用户信息
func SetCachedUserInfo(config *UserConfig) {
	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()
	cachedUserInfo = config
	if config != nil {
		cachedUserID = config.UserID
	}
	log.Printf("更新缓存的用户信息: %+v", config)
}

// GetUserID 获取用户ID的标准方法
// 按照优先级：1.内存缓存 2.磁盘配置 3.需要触发初始化
// 返回值：
// userID - 获取到的用户ID，如果没有则为空
// needInit - 是否需要触发用户初始化对话
// err - 如果发生错误则返回错误信息
func GetUserID() (string, bool, error) {
	// 1. 首先尝试从内存缓存获取
	userID := GetCachedUserID()
	if userID != "" {
		log.Printf("[用户ID获取] 从内存缓存获取用户ID: %s", userID)
		return userID, false, nil
	}

	// 2. 从磁盘配置文件获取
	config, err := LoadUserConfig()
	if err != nil {
		return "", true, fmt.Errorf("读取用户配置失败: %w", err)
	}

	// 3. 如果配置文件中有有效的用户ID，则更新缓存并返回
	if config != nil && config.UserID != "" {
		SetCachedUserInfo(config)
		log.Printf("[用户ID获取] 从磁盘配置获取用户ID: %s", config.UserID)
		return config.UserID, false, nil
	}

	// 4. 无法获取到用户ID，需要触发用户初始化对话
	log.Printf("[用户ID获取] 未找到有效的用户ID，需要初始化")
	return "", true, nil
}

// GetUserIDFromMetadata 从元数据中获取用户ID
// 如果metadata中有userId字段，则返回该值
// 同时从metadata中删除该字段，以避免重复存储
func GetUserIDFromMetadata(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	// 检查metadata中是否有userId字段
	if userID, ok := metadata["userId"].(string); ok && userID != "" {
		// 从metadata中删除该字段，避免重复存储
		delete(metadata, "userId")
		log.Printf("[用户ID获取] 从元数据获取用户ID: %s", userID)
		return userID
	}

	return ""
}

// InitializeUserByDialog 初始化用户对话状态
func InitializeUserByDialog(sessionID string) (*DialogState, error) {
	log.Printf("[用户初始化] 开始初始化用户对话，sessionID=%s", sessionID)

	// 检查是否已有缓存的用户配置
	cachedUserID := GetCachedUserID()
	if cachedUserID != "" {
		log.Printf("[用户初始化] 发现缓存的用户ID: %s", cachedUserID)
		state := &DialogState{
			State:    DialogStateCompleted,
			UserID:   cachedUserID,
			LastTime: time.Now(),
		}
		dialogStates[sessionID] = state
		return state, nil
	}

	// 尝试从文件加载用户配置
	config, err := LoadUserConfig()
	if err == nil && config != nil && config.UserID != "" {
		log.Printf("[用户初始化] 从文件加载到用户配置: %s", config.UserID)

		// 更新缓存
		SetCachedUserInfo(config)

		state := &DialogState{
			State:    DialogStateCompleted,
			UserID:   config.UserID,
			LastTime: time.Now(),
		}
		dialogStates[sessionID] = state
		return state, nil
	}

	// 检查是否已有对话状态
	if state, exists := dialogStates[sessionID]; exists {
		log.Printf("[用户初始化] 找到现有对话状态: state=%s, userID=%s", state.State, state.UserID)
		return state, nil
	}

	// 没有用户配置，直接进入询问流程（去掉DialogStateAsking状态）
	log.Printf("[用户初始化] 未找到用户配置，直接进入询问流程")
	state := &DialogState{
		State:    DialogStateAsking, // 保持现有状态名，但语义改为直接询问
		LastTime: time.Now(),
	}
	dialogStates[sessionID] = state
	log.Printf("[用户初始化] 创建新的对话状态: state=%s, sessionID=%s", state.State, sessionID)

	return state, nil
}

// HandleUserDialogResponse 处理用户对话响应（支持云端校验的版本）
func HandleUserDialogResponse(sessionID, response string) (*DialogState, error) {
	log.Printf("[用户初始化] 开始处理用户对话响应，sessionID=%s, response=%q", sessionID, response)

	// 获取对话状态
	state, exists := dialogStates[sessionID]
	if !exists {
		log.Printf("[用户初始化] 错误: 会话状态不存在，sessionID=%s", sessionID)
		return nil, fmt.Errorf("会话状态不存在")
	}

	log.Printf("[用户初始化] 当前对话状态: state=%s, userID=%s", state.State, state.UserID)

	// 检查是否是重置指令
	if containsResetCommand(response) {
		state.State = DialogStateAsking
		state.UserID = ""
		log.Printf("[用户初始化] 检测到重置指令，重置为询问状态")
		state.LastTime = time.Now()
		return state, nil
	}

	switch state.State {
	case DialogStateAsking:
		// 处理初始询问：是否已在其他设备使用过
		if containsNewUserRequest(response) {
			// 用户明确表示是新用户，进入创建用户ID流程
			state.State = DialogStateNewUser
			log.Printf("[用户初始化] 用户明确表示是新用户，进入创建用户ID流程")
		} else if containsAffirmative(response) {
			// 用户表示已有账户，进入输入用户ID流程
			state.State = DialogStateExisting
			log.Printf("[用户初始化] 用户表示已有账户，进入输入用户ID流程")
		} else {
			// 用户表示是新用户，进入创建用户ID流程
			state.State = DialogStateNewUser
			log.Printf("[用户初始化] 用户表示是新用户，进入创建用户ID流程")
		}

	case DialogStateNewUser:
		// 处理新用户的用户ID输入
		userID := extractUserID(response)
		if userID == "" {
			log.Printf("[用户初始化] 未能识别有效的用户ID: %s", response)
			return state, fmt.Errorf("请输入您想要的用户ID，格式如: user_abc12345 或直接输入 abc12345")
		}

		// 本地格式验证
		if !ValidateUserID(userID) {
			log.Printf("[用户初始化] 用户ID格式验证失败: %s", userID)
			return state, fmt.Errorf("用户ID格式无效，请使用格式: user_abc12345 或直接输入 abc12345")
		}

		log.Printf("[用户初始化] 新用户输入用户ID: %s，开始云端校验", userID)

		// 🔥 关键改进：调用云端API进行唯一性校验并存储
		err := CreateUserWithCloudValidation(userID)
		if err != nil {
			log.Printf("[用户初始化] 云端用户创建失败: %v", err)
			return state, fmt.Errorf("用户ID校验失败: %v，请尝试其他用户ID", err)
		}

		state.State = DialogStateCompleted
		state.UserID = userID
		log.Printf("[用户初始化] 云端用户创建成功，开始本地存储: %s", userID)

		// 云端成功后，执行本地存储
		config := &UserConfig{
			UserID:    state.UserID,
			FirstUsed: time.Now().Format(time.RFC3339),
		}

		if err := SaveUserConfig(config); err != nil {
			log.Printf("[用户初始化] 保存新用户配置失败: %v", err)
			return nil, err
		}

		// 更新缓存
		SetCachedUserInfo(config)
		log.Printf("[用户初始化] 新用户配置已保存并缓存")

	case DialogStateExisting:
		// 处理用户ID输入
		if containsCreateNewRequest(response) {
			// 用户改变主意要创建新账号，进入新用户ID输入流程
			state.State = DialogStateNewUser
			log.Printf("[用户初始化] 用户改变主意要创建新账号，请求输入新用户ID")
			return state, fmt.Errorf("好的，请输入您想要的新用户ID，格式如: user_abc12345 或直接输入 abc12345")
		}

		// 处理用户ID输入
		userID := extractUserID(response)
		if userID == "" {
			log.Printf("[用户初始化] 未能识别有效的用户ID: %s", response)
			return state, fmt.Errorf("无法识别用户ID，请重新输入。如果需要创建新账号，请回复'创建新账号'")
		}

		// 本地格式验证
		if !ValidateUserID(userID) {
			log.Printf("[用户初始化] 用户ID格式验证失败: %s", userID)
			return state, fmt.Errorf("用户ID格式无效，如果需要创建新账号，请回复'创建新账号'")
		}

		// 🔥 关键改进：调用云端API验证用户是否存在
		err := ValidateUserWithCloudAPI(userID)
		if err != nil {
			log.Printf("[用户初始化] 云端用户验证失败: %v", err)
			return state, fmt.Errorf("用户验证失败: %v。如果需要创建新账号，请回复'创建新账号'", err)
		}

		// 配置完成
		state.UserID = userID
		state.State = DialogStateCompleted
		log.Printf("[用户初始化] 云端用户验证成功，配置完成: %s", userID)

		// 保存用户配置
		config := &UserConfig{
			UserID:    userID,
			FirstUsed: time.Now().Format(time.RFC3339),
		}

		if err := SaveUserConfig(config); err != nil {
			log.Printf("[用户初始化] 保存用户配置失败: %v", err)
			return nil, err
		}

		SetCachedUserInfo(config)
		log.Printf("[用户初始化] 现有用户配置已保存并缓存")

	default:
		log.Printf("[用户初始化] 当前状态不需要处理用户响应: %s", state.State)
	}

	state.LastTime = time.Now()
	log.Printf("[用户初始化] 对话状态处理完成，最终状态: state=%s, userID=%s", state.State, state.UserID)
	return state, nil
}

// 辅助函数：提取用户ID
func extractUserID(response string) string {
	// 支持两种格式：1. 完整的用户ID (user_xxxxxxxx) 2. 简短格式 (xxxxxxxx)
	response = strings.TrimSpace(response)

	// 尝试匹配完整的用户ID格式
	reUserId := regexp.MustCompile(`user_[a-z0-9]{8}`)
	if match := reUserId.FindString(response); match != "" {
		log.Printf("[用户初始化] 提取到完整用户ID: %s", match)
		return match
	}

	// 尝试匹配简短格式，自动添加前缀
	reShort := regexp.MustCompile(`[a-z0-9]{8}`)
	if match := reShort.FindString(response); match != "" {
		userID := "user_" + match
		log.Printf("[用户初始化] 从简短格式提取并转换为用户ID: %s", userID)
		return userID
	}

	return ""
}

// ValidateUserID 验证用户ID
func ValidateUserID(userID string) bool {
	log.Printf("🔍 [用户ID验证] === 开始验证用户ID ===")
	log.Printf("🔍 [用户ID验证] 输入userID: '%s' (长度: %d)", userID, len(userID))

	// 验证用户ID格式
	if !strings.HasPrefix(userID, "user_") {
		log.Printf("🔍 [用户ID验证] 验证失败: 不以'user_'开头")
		return false
	}
	log.Printf("🔍 [用户ID验证] 步骤1 - 前缀验证通过")

	// 🔥 修复：放宽长度限制，允许8-20个字符的后缀
	if len(userID) < 8 || len(userID) > 30 { // "user_" + 8到30个字符
		log.Printf("🔍 [用户ID验证] 验证失败: 长度无效 (实际: %d, 期望: 13-35)", len(userID))
		return false
	}
	log.Printf("🔍 [用户ID验证] 步骤2 - 长度验证通过")

	// 验证字符组成
	suffix := userID[5:] // 获取"user_"后面的部分
	log.Printf("🔍 [用户ID验证] 步骤3 - 用户名部分: '%s' (长度: %d)", suffix, len(suffix))

	pattern := `^[a-z0-9_]{8,30}$`
	re := regexp.MustCompile(pattern) // 🔥 修复：允许下划线和更长的用户名
	matched := re.MatchString(suffix)
	log.Printf("🔍 [用户ID验证] 步骤3 - 正则匹配 '%s': %t", pattern, matched)

	if !matched {
		log.Printf("🔍 [用户ID验证] 验证失败: 用户名部分格式无效")
		return false
	}

	log.Printf("🔍 [用户ID验证] === 用户ID验证通过 ===")
	return matched
}

// GenerateUserID 生成用户ID
func GenerateUserID() string {
	// 生成随机的用户ID
	return fmt.Sprintf("user_%s", randomString(8))
}

// 辅助函数：检测肯定回答（排除新用户表述）
func containsAffirmative(response string) bool {
	response = strings.ToLower(response)

	// 先检查是否包含新用户相关的表述，如果是就不认为是肯定回答
	if containsNewUserRequest(response) {
		return false
	}

	return strings.Contains(response, "有") ||
		strings.Contains(response, "对") ||
		strings.Contains(response, "yes") ||
		strings.Contains(response, "true") ||
		strings.Contains(response, "正确") ||
		(strings.Contains(response, "是") && !strings.Contains(response, "新用户"))
}

// 辅助函数：检测重置命令
func containsResetCommand(response string) bool {
	response = strings.ToLower(response)
	return strings.Contains(response, "重置") ||
		strings.Contains(response, "重新开始") ||
		strings.Contains(response, "restart") ||
		strings.Contains(response, "reset")
}

// 辅助函数：检测创建新账号请求
func containsCreateNewRequest(response string) bool {
	response = strings.ToLower(response)
	return strings.Contains(response, "创建新") ||
		strings.Contains(response, "新账号") ||
		strings.Contains(response, "帮我创建") ||
		strings.Contains(response, "创建一个新的") ||
		strings.Contains(response, "new account") ||
		strings.Contains(response, "create new")
}

// 辅助函数：检测新用户请求
func containsNewUserRequest(response string) bool {
	response = strings.ToLower(response)
	return strings.Contains(response, "新用户") ||
		strings.Contains(response, "新账号") ||
		strings.Contains(response, "帮我创建") ||
		strings.Contains(response, "创建一个新的") ||
		strings.Contains(response, "new account") ||
		strings.Contains(response, "create new")
}

// 获取配置目录路径
func getConfigDir() string {
	// 尝试获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("警告: 无法获取用户主目录: %v", err)
		homeDir = "."
	}

	appName := "context-keeper"
	var configDir string

	// 根据操作系统选择标准应用数据目录
	switch runtime.GOOS {
	case "darwin": // macOS
		// ~/Library/Application Support/context-keeper/
		configDir = filepath.Join(homeDir, "Library", "Application Support", appName)

	case "windows":
		// 尝试使用APPDATA环境变量
		appData := os.Getenv("APPDATA")
		if appData != "" {
			configDir = filepath.Join(appData, appName)
		} else {
			// 回退到用户目录下的标准位置
			configDir = filepath.Join(homeDir, "AppData", "Roaming", appName)
		}

	default: // Linux和其他UNIX系统
		// ~/.local/share/context-keeper/
		configDir = filepath.Join(homeDir, ".local", "share", appName)

		// 检查XDG_DATA_HOME环境变量
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			configDir = filepath.Join(xdgDataHome, appName)
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("警告: 创建配置目录失败: %v，将使用用户主目录", err)
		return filepath.Join(homeDir, ConfigDirName)
	}

	return configDir
}

// 获取配置文件路径
func getConfigFilePath() string {
	return filepath.Join(getConfigDir(), ConfigFileName)
}

// LoadUserConfig 加载用户配置
func LoadUserConfig() (*UserConfig, error) {
	configPath := getConfigFilePath()
	log.Printf("[用户配置] 开始加载用户配置，路径: %s", configPath)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("[用户配置] 配置文件不存在: %s", configPath)
		return nil, nil // 不存在则返回nil，而不是错误
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[用户配置] 读取配置文件失败: %v", err)
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	log.Printf("[用户配置] 成功读取配置文件，大小: %d 字节", len(data))

	// 解析JSON
	var config UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[用户配置] 解析配置JSON失败: %v", err)
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	log.Printf("[用户配置] 成功加载用户配置: userID=%s, firstUsed=%s",
		config.UserID, config.FirstUsed)
	return &config, nil
}

// SaveUserConfig 保存用户配置
func SaveUserConfig(config *UserConfig) error {
	if config == nil {
		log.Printf("[用户配置] 错误: 尝试保存空配置")
		return fmt.Errorf("无法保存空配置")
	}

	configDir := getConfigDir()
	log.Printf("[用户配置] 开始保存用户配置: userID=%s, 目录=%s", config.UserID, configDir)

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("[用户配置] 创建配置目录失败: %v", err)
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	log.Printf("[用户配置] 确保配置目录存在: %s", configDir)

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("[用户配置] 序列化配置失败: %v", err)
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	log.Printf("[用户配置] 配置已序列化为JSON，大小: %d 字节", len(data))

	// 写入文件
	configPath := getConfigFilePath()
	log.Printf("[用户配置] 准备写入配置文件: %s", configPath)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Printf("[用户配置] 写入配置文件失败: %v", err)
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	log.Printf("[用户配置] 成功保存用户配置到: %s", configPath)
	return nil
}

// GetOrCreateUserID 获取或创建用户ID
func GetOrCreateUserID() (string, error) {
	// 尝试加载现有配置
	config, err := LoadUserConfig()
	if err != nil {
		return "", fmt.Errorf("加载用户配置失败: %w", err)
	}

	// 如果配置存在且有userID，直接返回
	if config != nil && config.UserID != "" {
		return config.UserID, nil
	}

	// 否则，创建新的用户ID并保存
	userID := uuid.New().String()

	// 创建新配置
	newConfig := &UserConfig{
		UserID:    userID,
		FirstUsed: time.Now().Format(time.RFC3339),
	}

	// 保存配置
	if err := SaveUserConfig(newConfig); err != nil {
		return "", fmt.Errorf("保存用户配置失败: %w", err)
	}

	return userID, nil
}

// randomString 生成指定长度的随机字符串
func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = chars[randomInt(0, len(chars)-1)]
	}

	return string(result)
}

// 生成随机整数
func randomInt(min, max int) int {
	// 使用crypto/rand代替os.Urandom
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		log.Printf("警告: 生成随机数失败: %v", err)
		return min
	}
	return min + int(n.Int64())
}

// CreateUserWithCloudValidation 通过云端API创建用户（强制依赖）
func CreateUserWithCloudValidation(userID string) error {
	log.Printf("[云端用户创建] 开始为用户ID创建云端账户: %s", userID)

	// 构建请求数据
	requestData := map[string]interface{}{
		"userId":     userID,
		"firstUsed":  time.Now().Format(time.RFC3339),
		"lastActive": time.Now().Format(time.RFC3339),
		"deviceInfo": map[string]interface{}{
			"platform": "cursor-extension",
			"version":  "1.0.0",
		},
		"metadata": map[string]interface{}{
			"createdVia": "dialog",
			"source":     "context-keeper-extension",
		},
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Printf("[云端用户创建] 序列化请求数据失败: %v", err)
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 发送POST请求到云端API
	apiURL := getCloudAPIURL() + "/api/users"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[云端用户创建] 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[云端用户创建] HTTP请求失败: %v", err)
		return fmt.Errorf("云端服务不可用，请检查网络连接或稍后重试: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		log.Printf("[云端用户创建] 解析响应失败: %v", err)
		return fmt.Errorf("解析云端响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode == http.StatusConflict {
		// 用户ID已存在，需要重新生成
		log.Printf("[云端用户创建] 用户ID已存在: %s", userID)
		return fmt.Errorf("用户ID已被使用，请重新生成")
	}

	if resp.StatusCode != http.StatusCreated {
		log.Printf("[云端用户创建] 云端API返回错误状态码: %d, 响应: %+v", resp.StatusCode, responseData)
		if message, ok := responseData["message"].(string); ok {
			return fmt.Errorf("云端用户创建失败: %s", message)
		}
		return fmt.Errorf("云端用户创建失败，状态码: %d", resp.StatusCode)
	}

	// 验证响应内容
	if success, ok := responseData["success"].(bool); !ok || !success {
		log.Printf("[云端用户创建] 云端响应指示创建失败: %+v", responseData)
		return fmt.Errorf("云端用户创建失败")
	}

	log.Printf("[云端用户创建] 用户在云端创建成功: %s", userID)
	return nil
}

// ValidateUserWithCloudAPI 通过云端API验证用户是否存在（强制依赖）
func ValidateUserWithCloudAPI(userID string) error {
	log.Printf("[云端用户验证] 开始验证用户是否存在: %s", userID)

	// 发送GET请求到云端API
	apiURL := getCloudAPIURL() + "/api/users/" + userID
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("[云端用户验证] 创建HTTP请求失败: %v", err)
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[云端用户验证] HTTP请求失败: %v", err)
		return fmt.Errorf("云端服务不可用，请检查网络连接或稍后重试: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		log.Printf("[云端用户验证] 解析响应失败: %v", err)
		return fmt.Errorf("解析云端响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[云端用户验证] 用户不存在: %s", userID)
		return fmt.Errorf("用户ID不存在，请检查输入是否正确")
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[云端用户验证] 云端API返回错误状态码: %d, 响应: %+v", resp.StatusCode, responseData)
		if message, ok := responseData["message"].(string); ok {
			return fmt.Errorf("用户验证失败: %s", message)
		}
		return fmt.Errorf("用户验证失败，状态码: %d", resp.StatusCode)
	}

	// 验证响应内容
	if success, ok := responseData["success"].(bool); !ok || !success {
		log.Printf("[云端用户验证] 云端响应指示验证失败: %+v", responseData)
		return fmt.Errorf("用户验证失败")
	}

	log.Printf("[云端用户验证] 用户验证成功: %s", userID)
	return nil
}

// getCloudAPIURL 获取云端API的URL
func getCloudAPIURL() string {
	// 优先从环境变量获取
	if url := os.Getenv("CONTEXT_KEEPER_API_URL"); url != "" {
		return strings.TrimSuffix(url, "/")
	}

	// 默认使用localhost（开发环境）
	return "http://localhost:8088"
}
