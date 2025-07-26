package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/utils"
)

// 对话状态常量
const (
	DialogStateNone      = "none"      // 无状态
	DialogStateAsking    = "asking"    // 询问是否有用户凭证
	DialogStateNewUser   = "new_user"  // 新用户流程
	DialogStateExisting  = "existing"  // 已有用户输入访问码
	DialogStateCompleted = "completed" // 配置完成
)

// DialogState 保存用户对话状态
type DialogState struct {
	State    string    // 当前状态
	UserID   string    // 用户ID (如果已分配)
	LastTime time.Time // 上次更新时间
}

var dialogStates = make(map[string]*DialogState) // sessionID -> DialogState

// UserService 用户服务，处理用户配置和凭证
type UserService struct {
	configPath string
}

// NewUserService 创建新的用户服务
func NewUserService(storagePath string) *UserService {
	return &UserService{
		configPath: filepath.Join(storagePath, "user-config.json"),
	}
}

// InitializeUserByDialog 通过对话方式初始化用户
func (s *UserService) InitializeUserByDialog(sessionID string) (*DialogState, error) {
	// 检查用户ID是否已经加载到全局变量
	cachedUserId := utils.GetCachedUserID()

	// 如果全局已有用户ID，说明启动时已成功加载，直接返回完成状态
	if cachedUserId != "" {
		log.Printf("[用户服务] 检测到已加载的用户ID: %s，跳过对话初始化", cachedUserId)
		state := &DialogState{
			State:    DialogStateCompleted,
			UserID:   cachedUserId,
			LastTime: time.Now(),
		}
		dialogStates[sessionID] = state
		return state, nil
	}

	// 如果没有缓存的用户ID，检查本地配置文件
	config, err := s.ReadUserConfig()
	if err == nil && config.UserID != "" {
		// 发现有效配置，缓存ID并返回完成状态
		utils.SetCachedUserID(config.UserID)
		state := &DialogState{
			State:    DialogStateCompleted,
			UserID:   config.UserID,
			LastTime: time.Now(),
		}
		dialogStates[sessionID] = state
		log.Printf("[用户服务] 从配置文件加载用户ID: %s", config.UserID)
		return state, nil
	}

	// 检查会话是否已有状态（继续之前的对话）
	if state, exists := dialogStates[sessionID]; exists {
		return state, nil
	}

	// 没有用户配置，初始化对话状态
	log.Printf("[用户服务] 未找到用户配置，初始化对话流程")
	state := &DialogState{
		State:    DialogStateAsking,
		LastTime: time.Now(),
	}
	dialogStates[sessionID] = state
	return state, nil
}

// HandleUserDialogResponse 处理用户对话响应
func (s *UserService) HandleUserDialogResponse(sessionID, response string) (*DialogState, error) {
	// 获取对话状态
	state, exists := dialogStates[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话状态不存在")
	}

	log.Printf("[用户服务] 处理对话响应: 状态=%s, 响应=%s", state.State, response)

	switch state.State {
	case DialogStateAsking:
		// 处理是否有用户ID的响应
		if containsAffirmative(response) {
			// 用户表示已在其他设备使用过
			state.State = DialogStateExisting
		} else {
			// 用户是新用户
			state.State = DialogStateNewUser
			// 生成新的用户ID
			state.UserID = GenerateUserID()

			// 保存新用户配置
			err := s.SaveUserConfig(&models.UserConfig{
				UserID: state.UserID,
			})
			if err != nil {
				log.Printf("[用户服务] 保存新用户配置出错: %v", err)
				return nil, err
			}
		}
	case DialogStateExisting:
		// 处理用户ID响应
		inputUserID := extractUserID(response)
		if inputUserID == "" {
			return state, fmt.Errorf("无法识别用户ID，请重新输入")
		}

		// 验证用户ID
		validUserID, err := s.ValidateUserID(inputUserID)
		if err != nil {
			return state, fmt.Errorf("用户ID无效: %v", err)
		}

		// 更新状态
		state.UserID = validUserID
		state.State = DialogStateCompleted

		// 保存用户配置
		err = s.SaveUserConfig(&models.UserConfig{
			UserID: validUserID,
		})
		if err != nil {
			log.Printf("[用户服务] 保存用户配置出错: %v", err)
			return nil, err
		}
	}

	state.LastTime = time.Now()
	return state, nil
}

// GetDialogState 获取指定会话的对话状态
func (s *UserService) GetDialogState(sessionID string) (*DialogState, bool) {
	state, exists := dialogStates[sessionID]
	return state, exists
}

// 辅助函数：检测肯定回答
func containsAffirmative(response string) bool {
	response = strings.ToLower(response)
	return strings.Contains(response, "是") ||
		strings.Contains(response, "有") ||
		strings.Contains(response, "对") ||
		strings.Contains(response, "yes") ||
		strings.Contains(response, "true") ||
		strings.Contains(response, "正确")
}

// 辅助函数：提取用户ID
func extractUserID(response string) string {
	// 匹配用户ID模式 user_xxxxxxxx
	re := regexp.MustCompile(`user_[a-z0-9]{8}`)
	match := re.FindString(response)
	return match
}

// ReadUserConfig 读取用户配置
func (s *UserService) ReadUserConfig() (*models.UserConfig, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，返回空配置
			return &models.UserConfig{}, nil
		}
		return nil, fmt.Errorf("读取用户配置失败: %w", err)
	}

	var config models.UserConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析用户配置失败: %w", err)
	}

	return &config, nil
}

// SaveUserConfig 保存用户配置
func (s *UserService) SaveUserConfig(config *models.UserConfig) error {
	// 确保目录存在
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入用户配置失败: %w", err)
	}

	log.Printf("[用户服务] 已保存用户配置: UserID=%s", config.UserID)
	return nil
}

// ValidateUserID 验证用户ID
func (s *UserService) ValidateUserID(userID string) (string, error) {
	// 验证用户ID格式 (支持8-20个字符的用户名部分)
	if !strings.HasPrefix(userID, "user_") || len(userID) < 13 || len(userID) > 25 {
		return "", fmt.Errorf("用户ID格式无效")
	}

	// 简单实现：直接返回输入的用户ID
	// 实际应用中应该连接到某个服务验证用户ID
	return userID, nil
}

// GenerateUserID 生成用户ID
func GenerateUserID() string {
	// 生成随机的用户ID
	return fmt.Sprintf("user_%s", randomString(8))
}

// randomString 生成指定长度的随机字符串
func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

// GetCachedUserID 获取缓存的用户ID
func (s *UserService) GetCachedUserID() string {
	return utils.GetCachedUserID()
}

// SetCachedUserID 设置缓存的用户ID
func (s *UserService) SetCachedUserID(userID string) {
	utils.SetCachedUserID(userID)
}
