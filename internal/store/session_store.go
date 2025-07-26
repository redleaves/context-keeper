package store

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/contextkeeper/service/internal/models"
)

// SessionStore 会话存储管理
type SessionStore struct {
	storePath string
	sessions  map[string]*models.Session
	histories map[string][]string // sessionID -> 最近历史记录
	mu        sync.RWMutex
}

// NewSessionStore 创建新的会话存储
func NewSessionStore(storePath string) (*SessionStore, error) {
	log.Printf("[会话存储] 初始化会话存储, 路径: %s", storePath)

	// 获取绝对路径
	absPath, err := filepath.Abs(storePath)
	if err != nil {
		log.Printf("[会话存储] 获取绝对路径失败: %v", err)
	} else {
		log.Printf("[会话存储] 存储绝对路径: %s", absPath)
	}

	// 确保存储目录存在
	sessionsPath := filepath.Join(storePath, "sessions")
	log.Printf("[会话存储] 会话目录: %s", sessionsPath)

	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		log.Printf("[会话存储] 错误: 创建会话存储目录失败: %v", err)
		return nil, fmt.Errorf("创建会话存储目录失败: %w", err)
	}

	// 同时创建历史记录目录
	historiesPath := filepath.Join(storePath, "histories")
	log.Printf("[会话存储] 历史记录目录: %s", historiesPath)

	if err := os.MkdirAll(historiesPath, 0755); err != nil {
		log.Printf("[会话存储] 错误: 创建历史记录目录失败: %v", err)
		return nil, fmt.Errorf("创建历史记录目录失败: %w", err)
	}

	store := &SessionStore{
		storePath: storePath,
		sessions:  make(map[string]*models.Session),
		histories: make(map[string][]string),
	}

	// 尝试加载现有会话
	if err := store.loadSessions(); err != nil {
		log.Printf("[会话存储] 警告: 加载会话失败: %v", err)
		return nil, fmt.Errorf("加载会话失败: %w", err)
	}

	log.Printf("[会话存储] 会话存储初始化完成, 已加载 %d 个会话", len(store.sessions))
	return store, nil
}

// GetSession 获取会话信息，如果不存在则创建
func (s *SessionStore) GetSession(sessionID string) (*models.Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if exists {
		return session, nil
	}

	// 会话不存在，创建新会话
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if session, exists = s.sessions[sessionID]; exists {
		return session, nil
	}

	session = models.NewSession(sessionID)
	s.sessions[sessionID] = session

	// 保存新会话
	if err := s.saveSession(session); err != nil {
		return nil, fmt.Errorf("保存新会话失败: %w", err)
	}

	return session, nil
}

// UpdateSession 更新会话信息并记录历史
func (s *SessionStore) UpdateSession(sessionID string, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[会话存储] 更新会话: 会话ID=%s, 内容长度=%d字节, 存储路径=%s",
		sessionID, len(content), s.storePath)

	// 获取或创建会话
	session, exists := s.sessions[sessionID]
	if !exists {
		log.Printf("[会话存储] 会话不存在，创建新会话: %s", sessionID)
		session = models.NewSession(sessionID)
		s.sessions[sessionID] = session
	}

	// 更新最后活动时间
	session.LastActive = time.Now()

	// 添加到历史记录
	history, exists := s.histories[sessionID]
	if !exists {
		history = []string{}
	}

	// 添加新内容到历史（保持最大长度限制）
	maxHistory := 20 // 最多保存20条历史记录
	history = append(history, content)
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	s.histories[sessionID] = history
	log.Printf("[会话存储] 更新后历史记录数: %d", len(history))

	// 保存会话和历史
	if err := s.saveSession(session); err != nil {
		log.Printf("[会话存储] 错误: 保存会话失败: %v", err)
		return fmt.Errorf("保存会话失败: %w", err)
	}

	if err := s.saveHistory(sessionID, history); err != nil {
		log.Printf("[会话存储] 错误: 保存历史记录失败: %v", err)
		return fmt.Errorf("保存历史记录失败: %w", err)
	}

	log.Printf("[会话存储] 成功更新会话: %s", sessionID)
	return nil
}

// GetSessionState 获取会话状态信息
func (s *SessionStore) GetSessionState(sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return "", fmt.Errorf("会话不存在: %s", sessionID)
	}

	created := session.CreatedAt.Format("2006-01-02 15:04:05")
	lastActive := session.LastActive.Format("2006-01-02 15:04:05")

	return fmt.Sprintf("会话ID: %s\n创建时间: %s\n最后活动: %s\n状态: %s",
		session.ID, created, lastActive, session.Status), nil
}

// GetRecentHistory 获取最近的历史记录
func (s *SessionStore) GetRecentHistory(sessionID string, count int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.histories[sessionID]
	if !exists {
		if _, sessionExists := s.sessions[sessionID]; !sessionExists {
			return nil, fmt.Errorf("会话不存在: %s", sessionID)
		}
		return []string{}, nil
	}

	if count <= 0 || count > len(history) {
		count = len(history)
	}

	// 返回最近的count条记录
	result := make([]string, count)
	start := len(history) - count
	copy(result, history[start:])

	return result, nil
}

// UpdateSessionSummary 更新会话摘要
func (s *SessionStore) UpdateSessionSummary(sessionID string, summary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 更新摘要
	session.Summary = summary
	session.LastActive = time.Now()

	// 保存会话
	if err := s.saveSession(session); err != nil {
		return fmt.Errorf("保存会话失败: %w", err)
	}

	return nil
}

// StoreMessages 存储消息到会话
func (s *SessionStore) StoreMessages(sessionID string, messages []*models.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[会话存储] 存储消息: 会话ID=%s, 消息数=%d, 存储路径=%s",
		sessionID, len(messages), s.storePath)

	// 获取或创建会话
	session, exists := s.sessions[sessionID]
	if !exists {
		log.Printf("[会话存储] 会话不存在，创建新会话: %s", sessionID)
		session = models.NewSession(sessionID)
		s.sessions[sessionID] = session
	}

	// 更新最后活动时间
	session.LastActive = time.Now()

	// 添加消息
	if session.Messages == nil {
		session.Messages = make([]*models.Message, 0)
	}

	// 添加新消息
	session.Messages = append(session.Messages, messages...)
	log.Printf("[会话存储] 添加消息后，会话总消息数: %d", len(session.Messages))

	// 保存会话
	if err := s.saveSession(session); err != nil {
		log.Printf("[会话存储] 错误: 保存会话失败: %v", err)
		return fmt.Errorf("保存会话失败: %w", err)
	}

	log.Printf("[会话存储] 成功存储消息到会话: %s", sessionID)
	return nil
}

// GetMessages 获取会话中的消息
func (s *SessionStore) GetMessages(sessionID string, limit int) ([]*models.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", sessionID)
	}

	if session.Messages == nil || len(session.Messages) == 0 {
		return []*models.Message{}, nil
	}

	// 如果limit小于等于0或大于消息数量，返回所有消息
	if limit <= 0 || limit > len(session.Messages) {
		return session.Messages, nil
	}

	// 返回最近的limit条消息
	startIdx := len(session.Messages) - limit
	return session.Messages[startIdx:], nil
}

// AssociateFile 关联文件到会话
func (s *SessionStore) AssociateFile(sessionID, filePath, language string, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建会话
	session, exists := s.sessions[sessionID]
	if !exists {
		session = models.NewSession(sessionID)
		s.sessions[sessionID] = session
	}

	// 确保CodeContext已初始化
	if session.CodeContext == nil {
		session.CodeContext = make(map[string]*models.CodeFile)
	}

	// 创建或更新文件信息
	codeFile := &models.CodeFile{
		Path:     filePath,
		Language: language,
		LastEdit: time.Now().Unix(),
	}

	// 如果提供了内容，可以后续添加内容摘要功能
	if content != "" {
		// 这里可以添加代码摘要生成逻辑
		codeFile.Summary = fmt.Sprintf("文件长度: %d字节", len(content))
	}

	// 存储文件信息
	session.CodeContext[filePath] = codeFile
	session.LastActive = time.Now()

	// 保存会话
	if err := s.saveSession(session); err != nil {
		return fmt.Errorf("保存会话失败: %w", err)
	}

	return nil
}

// UpdateCodeFileRelations 更新代码文件与相关讨论的关联
func (s *SessionStore) UpdateCodeFileRelations(sessionID, filePath string, discussions []models.DiscussionRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取会话
	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 确保CodeContext已初始化
	if session.CodeContext == nil {
		session.CodeContext = make(map[string]*models.CodeFile)
	}

	// 获取或创建代码文件
	codeFile, exists := session.CodeContext[filePath]
	if !exists {
		codeFile = &models.CodeFile{
			Path:     filePath,
			LastEdit: time.Now().Unix(),
		}
		session.CodeContext[filePath] = codeFile
	}

	// 转换为内部格式的CodeFile，以兼容现有代码
	codeFileInfo := &models.CodeFileInfo{
		Path:     codeFile.Path,
		Language: codeFile.Language,
		LastEdit: codeFile.LastEdit,
		Summary:  codeFile.Summary,
	}

	// 更新相关讨论
	if len(discussions) > 0 {
		codeFileInfo.RelatedDiscussions = discussions

		// 更新重要性评分（基于关联讨论数量）
		codeFileInfo.Importance = float64(len(discussions)) * 0.2
		if codeFileInfo.Importance > 1.0 {
			codeFileInfo.Importance = 1.0
		}
	}

	// 将更新后的信息同步回CodeContext
	// 由于目前CodeFile结构不包含RelatedDiscussions字段，
	// 我们使用metadata来存储这些额外信息
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}

	// 创建或获取代码文件与讨论的关联映射
	codeToDiscussions, ok := session.Metadata["code_discussions"].(map[string]interface{})
	if !ok {
		codeToDiscussions = make(map[string]interface{})
	}

	// 序列化关联讨论列表
	discussionsData, err := json.Marshal(discussions)
	if err != nil {
		return fmt.Errorf("序列化关联讨论失败: %w", err)
	}

	// 存储到元数据
	codeToDiscussions[filePath] = string(discussionsData)
	session.Metadata["code_discussions"] = codeToDiscussions

	// 更新最后活动时间
	session.LastActive = time.Now()

	// 保存会话
	if err := s.saveSession(session); err != nil {
		return fmt.Errorf("保存会话失败: %w", err)
	}

	log.Printf("[会话存储] 更新文件关联: 会话ID=%s, 文件=%s, 关联讨论数=%d",
		sessionID, filePath, len(discussions))
	return nil
}

// GetCodeFileRelations 获取代码文件的关联讨论
func (s *SessionStore) GetCodeFileRelations(sessionID, filePath string) ([]models.DiscussionRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 获取会话
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 检查元数据中是否存在关联讨论
	if session.Metadata == nil {
		return []models.DiscussionRef{}, nil
	}

	codeToDiscussions, ok := session.Metadata["code_discussions"].(map[string]interface{})
	if !ok {
		return []models.DiscussionRef{}, nil
	}

	discussionsData, ok := codeToDiscussions[filePath].(string)
	if !ok {
		return []models.DiscussionRef{}, nil
	}

	// 反序列化讨论列表
	var discussions []models.DiscussionRef
	if err := json.Unmarshal([]byte(discussionsData), &discussions); err != nil {
		return nil, fmt.Errorf("解析关联讨论失败: %w", err)
	}

	return discussions, nil
}

// RecordEditAction 记录编辑操作
func (s *SessionStore) RecordEditAction(sessionID, filePath, editType string, position int, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建会话
	session, exists := s.sessions[sessionID]
	if !exists {
		session = models.NewSession(sessionID)
		s.sessions[sessionID] = session
	}

	// 确保EditHistory已初始化
	if session.EditHistory == nil {
		session.EditHistory = make([]*models.EditAction, 0)
	}

	// 创建编辑动作
	action := &models.EditAction{
		Timestamp: time.Now().Unix(),
		FilePath:  filePath,
		Type:      editType,
		Position:  position,
		Content:   content,
	}

	// 添加编辑动作
	session.EditHistory = append(session.EditHistory, action)
	session.LastActive = time.Now()

	// 更新关联文件的最后编辑时间
	if session.CodeContext != nil {
		if file, ok := session.CodeContext[filePath]; ok {
			file.LastEdit = time.Now().Unix()
		}
	}

	// 保存会话
	if err := s.saveSession(session); err != nil {
		return fmt.Errorf("保存会话失败: %w", err)
	}

	return nil
}

// CleanupInactiveSessions 清理不活跃的会话
func (s *SessionStore) CleanupInactiveSessions(timeout time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清理过期会话
	var cleanedCount int
	now := time.Now()

	for id, session := range s.sessions {
		// 检查上次活动时间
		if now.Sub(session.LastActive) > timeout {
			// 设置会话为已归档
			session.Status = "archived"

			// 保存更新的状态
			if err := s.saveSession(session); err != nil {
				log.Printf("保存归档会话状态失败: %v", err)
				continue
			}

			// 从内存中移除
			delete(s.sessions, id)
			delete(s.histories, id)
			cleanedCount++
		}
	}

	return cleanedCount
}

// CleanupShortTermMemory 清理短期记忆，只保留最近指定天数的数据
func (s *SessionStore) CleanupShortTermMemory(days int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if days <= 0 {
		days = 2 // 默认保留最近2天
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -days)
	var cleanedCount int

	// 遍历会话
	for _, session := range s.sessions {
		// 过滤消息
		if session.Messages != nil && len(session.Messages) > 0 {
			var recentMessages []*models.Message
			for _, msg := range session.Messages {
				msgTime := time.Unix(msg.Timestamp, 0)
				if msgTime.After(cutoffTime) {
					recentMessages = append(recentMessages, msg)
				}
			}

			// 如果有消息被过滤掉
			if len(recentMessages) < len(session.Messages) {
				cleanedCount += len(session.Messages) - len(recentMessages)
				session.Messages = recentMessages
				// 保存更新的会话
				if err := s.saveSession(session); err != nil {
					log.Printf("保存清理后的会话失败: %v", err)
				}
			}
		}
	}

	log.Printf("短期记忆清理完成: 清理了%d条超过%d天的消息", cleanedCount, days)
	return cleanedCount
}

// loadSessions 从文件加载会话
func (s *SessionStore) loadSessions() error {
	log.Printf("[会话存储] 开始从文件加载会话, 路径: %s", s.storePath)

	sessionsPath := filepath.Join(s.storePath, "sessions")
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		log.Printf("[会话存储] 错误: 读取会话目录失败: %v", err)
		if os.IsNotExist(err) {
			return nil // 目录不存在，属于正常情况
		}
		return fmt.Errorf("读取会话目录失败: %w", err)
	}

	loadedCount := 0
	filteredCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue // 跳过子目录
		}

		// 检查是否是会话文件
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".json") {
			continue
		}

		// 提取会话ID
		sessionID := strings.TrimSuffix(filename, ".json")

		// 读取会话文件
		filePath := filepath.Join(sessionsPath, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取会话文件失败: %w", err)
		}

		// 解析JSON
		var session models.Session
		if err := json.Unmarshal(data, &session); err != nil {
			return fmt.Errorf("解析会话JSON失败: %w", err)
		}

		// 过滤掉archived状态的会话
		if session.Status == models.SessionStatusArchived {
			filteredCount++
			continue // 跳过已归档会话
		}

		// 存储会话
		s.sessions[sessionID] = &session
		loadedCount++

		// 加载历史记录
		if history, err := s.loadHistory(sessionID); err == nil {
			s.histories[sessionID] = history
		}
	}

	log.Printf("[会话存储] 会话加载完成: 已加载%d个活跃会话, 过滤掉%d个归档会话",
		loadedCount, filteredCount)
	return nil
}

// saveSession 保存会话到文件
func (s *SessionStore) saveSession(session *models.Session) error {
	sessionsPath := filepath.Join(s.storePath, "sessions")
	filePath := filepath.Join(sessionsPath, session.ID+".json")

	// 添加日志记录
	log.Printf("[会话存储] 尝试保存会话到文件: %s", filePath)
	absPath, _ := filepath.Abs(filePath)
	log.Printf("[会话存储] 文件绝对路径: %s", absPath)

	// 序列化会话为JSON
	data, err := json.Marshal(session)
	if err != nil {
		log.Printf("[会话存储] 错误: 序列化会话失败: %v", err)
		return fmt.Errorf("序列化会话失败: %w", err)
	}

	// 如果目录不存在，则创建目录
	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		log.Printf("[会话存储] 错误: 创建目录失败: %s, 错误: %v", sessionsPath, err)
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("[会话存储] 错误: 写入会话文件失败: %v", err)
		return fmt.Errorf("写入会话文件失败: %w", err)
	}

	log.Printf("[会话存储] 成功保存会话到文件: %s", filePath)
	return nil
}

// saveHistory 保存历史记录到文件
func (s *SessionStore) saveHistory(sessionID string, history []string) error {
	historyPath := filepath.Join(s.storePath, "histories")

	// 添加日志记录
	log.Printf("[会话存储] 尝试保存历史记录: 会话ID=%s, 历史记录数=%d", sessionID, len(history))
	log.Printf("[会话存储] 历史记录目录: %s", historyPath)
	absPath, _ := filepath.Abs(historyPath)
	log.Printf("[会话存储] 目录绝对路径: %s", absPath)

	if err := os.MkdirAll(historyPath, 0755); err != nil {
		log.Printf("[会话存储] 错误: 创建历史记录目录失败: %v", err)
		return fmt.Errorf("创建历史记录目录失败: %w", err)
	}

	filePath := filepath.Join(historyPath, sessionID+".json")
	log.Printf("[会话存储] 历史记录文件路径: %s", filePath)

	// 序列化历史记录为JSON
	data, err := json.Marshal(history)
	if err != nil {
		log.Printf("[会话存储] 错误: 序列化历史记录失败: %v", err)
		return fmt.Errorf("序列化历史记录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("[会话存储] 错误: 写入历史记录文件失败: %v", err)
		return fmt.Errorf("写入历史记录文件失败: %w", err)
	}

	log.Printf("[会话存储] 成功保存历史记录到文件: %s", filePath)
	return nil
}

// loadHistory 从文件加载历史记录
func (s *SessionStore) loadHistory(sessionID string) ([]string, error) {
	historyPath := filepath.Join(s.storePath, "histories")
	filePath := filepath.Join(historyPath, sessionID+".json")

	// 读取历史记录文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // 文件不存在，返回空历史记录
		}
		return nil, fmt.Errorf("读取历史记录文件失败: %w", err)
	}

	// 解析JSON
	var history []string
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("解析历史记录JSON失败: %w", err)
	}

	return history, nil
}

// GetSessionCount 获取会话数量
func (s *SessionStore) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// GetActiveSessionCount 获取活跃会话数量
func (s *SessionStore) GetActiveSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, session := range s.sessions {
		if session.Status == models.SessionStatusActive {
			count++
		}
	}
	return count
}

// GetSessionList 获取会话列表
func (s *SessionStore) GetSessionList() []*models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		result = append(result, session)
	}
	return result
}

// SaveSession 保存会话到存储
func (s *SessionStore) SaveSession(session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[会话存储] 开始保存会话: ID=%s, 消息数=%d, 存储路径=%s",
		session.ID, len(session.Messages), s.storePath)

	// 更新会话映射
	s.sessions[session.ID] = session

	// 保存到文件
	sessionsPath := filepath.Join(s.storePath, "sessions")
	filePath := filepath.Join(sessionsPath, session.ID+".json")

	log.Printf("[会话存储] 会话文件路径: %s", filePath)
	absPath, _ := filepath.Abs(filePath)
	log.Printf("[会话存储] 会话文件绝对路径: %s", absPath)

	// 确保目录存在
	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		log.Printf("[会话存储] 错误: 创建会话目录失败: %s, 错误: %v", sessionsPath, err)
		return fmt.Errorf("创建会话目录失败: %w", err)
	}

	// 序列化会话
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		log.Printf("[会话存储] 错误: 序列化会话失败: %v", err)
		return fmt.Errorf("序列化会话失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("[会话存储] 错误: 写入会话文件失败: %v", err)
		return fmt.Errorf("写入会话文件失败: %w", err)
	}

	log.Printf("[会话存储] 成功保存会话到文件: %s, 大小=%d字节", filePath, len(data))
	return nil
}

// GetLastActiveTime 获取此存储中最近的活跃时间
func (s *SessionStore) GetLastActiveTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastActive := time.Time{} // 零值时间

	for _, session := range s.sessions {
		if session.LastActive.After(lastActive) {
			lastActive = session.LastActive
		}
	}

	// 如果没有会话，返回当前时间
	if lastActive.IsZero() {
		return time.Now()
	}

	return lastActive
}

// GetStorePath 获取会话存储路径
func (s *SessionStore) GetStorePath() string {
	return s.storePath
}

// GetOrCreateActiveSession 获取或创建活跃会话 - 修复工作空间隔离问题
func (s *SessionStore) GetOrCreateActiveSession(userID string, sessionTimeout time.Duration) (*models.Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var activeSession *models.Session
	var sessionID string

	// 1. 检查是否有未过期的活跃会话
	for id, session := range s.sessions {
		if session.Status == models.SessionStatusActive {
			// 检查会话是否还在有效期内
			if now.Sub(session.LastActive) <= sessionTimeout {
				log.Printf("[会话管理] 找到活跃会话: %s, 最后活动: %v", id, session.LastActive)
				// 更新最后活动时间
				session.LastActive = now
				if err := s.saveSession(session); err != nil {
					log.Printf("[会话管理] 警告: 更新会话活动时间失败: %v", err)
				}
				return session, false, nil // 返回现有会话，false表示不是新创建的
			} else {
				log.Printf("[会话管理] 会话已过期: %s, 最后活动: %v, 超时时间: %v",
					id, session.LastActive, sessionTimeout)
			}
		}
	}

	// 2. 没有找到活跃会话，创建新会话
	// 添加用户哈希避免不同用户冲突
	userHash := fmt.Sprintf("%x", md5.Sum([]byte(userID)))[:8]
	sessionID = fmt.Sprintf("session-%s-%s-%s",
		now.Format("20060102"),
		now.Format("150405"),
		userHash)

	// 确保会话ID唯一
	for s.sessions[sessionID] != nil {
		time.Sleep(time.Millisecond) // 等待1毫秒确保时间戳不同
		now = time.Now()
		// 重试时也使用相同的用户哈希逻辑
		userHash := fmt.Sprintf("%x", md5.Sum([]byte(userID)))[:8]
		sessionID = fmt.Sprintf("session-%s-%s-%s",
			now.Format("20060102"),
			now.Format("150405"),
			userHash)
	}

	activeSession = models.NewSession(sessionID)

	// 添加用户ID到元数据
	if userID != "" {
		if activeSession.Metadata == nil {
			activeSession.Metadata = make(map[string]interface{})
		}
		activeSession.Metadata["userId"] = userID
	}

	s.sessions[sessionID] = activeSession

	// 保存新会话
	if err := s.saveSession(activeSession); err != nil {
		delete(s.sessions, sessionID) // 回滚
		return nil, false, fmt.Errorf("保存新会话失败: %w", err)
	}

	log.Printf("[会话管理] 创建新活跃会话: %s, 用户ID: %s", sessionID, userID)
	return activeSession, true, nil
}

// 🔥 新增：GetOrCreateActiveSessionWithWorkspace 获取或创建带工作空间隔离的活跃会话
func (s *SessionStore) GetOrCreateActiveSessionWithWorkspace(userID string, workspaceHash string, sessionTimeout time.Duration) (*models.Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var activeSession *models.Session
	var sessionID string

	log.Printf("🔄 [会话存储] ===== 开始GetOrCreateActiveSessionWithWorkspace =====")
	log.Printf("🔄 [会话存储] 输入参数: userID=%s, workspaceHash=%s, sessionTimeout=%v", userID, workspaceHash, sessionTimeout)
	log.Printf("🔄 [会话存储] 当前时间: %s", now.Format("2006-01-02 15:04:05"))
	log.Printf("🔄 [会话存储] 当前存储中的会话数量: %d", len(s.sessions))

	// 🔥 工作空间隔离：如果提供了工作空间哈希，按工作空间查找活跃会话
	if workspaceHash != "" {
		log.Printf("🔄 [会话存储] 步骤1: 查找工作空间 '%s' 的活跃会话", workspaceHash)

		// 1. 检查是否有当前工作空间的未过期活跃会话
		var candidateSessions []string
		for id, session := range s.sessions {
			log.Printf("🔄 [会话存储] 步骤1.1: 检查会话 %s (状态: %s, 最后活动: %s)",
				id, session.Status, session.LastActive.Format("2006-01-02 15:04:05"))

			if session.Status == models.SessionStatusActive {
				candidateSessions = append(candidateSessions, id)
				log.Printf("🔄 [会话存储] 步骤1.2: 会话 %s 状态为活跃", id)

				// 检查是否属于同一工作空间
				sessionWorkspace := ""
				if session.Metadata != nil {
					if ws, ok := session.Metadata["workspaceHash"].(string); ok {
						sessionWorkspace = ws
					}
				}

				log.Printf("🔄 [会话存储] 步骤1.3: 会话 %s 工作空间哈希: '%s', 目标工作空间哈希: '%s'",
					id, sessionWorkspace, workspaceHash)

				if sessionWorkspace == workspaceHash {
					log.Printf("🔄 [会话存储] 步骤1.4: 会话 %s 工作空间匹配", id)

					// 🔥 关键修复：还必须检查用户ID是否匹配
					sessionUserID := ""
					if session.Metadata != nil {
						if uid, ok := session.Metadata["userId"].(string); ok {
							sessionUserID = uid
						}
					}

					log.Printf("🔄 [会话存储] 步骤1.4.1: 会话用户ID检查: 会话用户='%s', 请求用户='%s'", sessionUserID, userID)

					if sessionUserID != userID {
						log.Printf("🔄 [会话存储] 🚫 会话 %s 用户ID不匹配，跳过 (会话用户: %s, 请求用户: %s)", id, sessionUserID, userID)
						continue // 🔥 用户ID不匹配，继续查找下一个会话
					}

					log.Printf("🔄 [会话存储] ✅ 会话 %s 用户ID匹配", id)

					// 检查会话是否还在有效期内
					timeSinceLastActive := now.Sub(session.LastActive)
					log.Printf("🔄 [会话存储] 步骤1.5: 会话 %s 距离最后活动: %v, 超时阈值: %v",
						id, timeSinceLastActive, sessionTimeout)

					if timeSinceLastActive <= sessionTimeout {
						log.Printf("🔄 [会话存储] ✅ 找到工作空间 %s 用户 %s 的活跃会话: %s", workspaceHash, userID, id)
						log.Printf("🔄 [会话存储] 步骤1.6: 更新会话最后活动时间: %s -> %s",
							session.LastActive.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))

						// 更新最后活动时间
						session.LastActive = now
						if err := s.saveSession(session); err != nil {
							log.Printf("🔄 [会话存储] ⚠️ 更新会话活动时间失败: %v", err)
						} else {
							log.Printf("🔄 [会话存储] ✅ 会话活动时间更新成功")
						}

						log.Printf("🔄 [会话存储] ===== GetOrCreateActiveSessionWithWorkspace完成(复用) =====")
						return session, false, nil // 返回现有会话，false表示不是新创建的
					} else {
						log.Printf("🔄 [会话存储] ⏰ 工作空间 %s 用户 %s 的会话已过期: %s, 最后活动: %v, 超时时间: %v",
							workspaceHash, userID, id, session.LastActive, sessionTimeout)
					}
				} else {
					log.Printf("🔄 [会话存储] 🔀 会话 %s 工作空间不匹配", id)
				}
			} else {
				log.Printf("🔄 [会话存储] 💤 会话 %s 状态非活跃: %s", id, session.Status)
			}
		}

		log.Printf("🔄 [会话存储] 步骤1总结: 找到 %d 个活跃会话，但没有匹配的工作空间会话", len(candidateSessions))
		if len(candidateSessions) > 0 {
			log.Printf("🔄 [会话存储] 活跃会话列表: %v", candidateSessions)
		}

		// 2. 没有找到工作空间的活跃会话，创建新的工作空间会话
		log.Printf("🔄 [会话存储] 步骤2: 创建新的工作空间会话")

		// 添加用户哈希避免不同用户冲突
		userHash := fmt.Sprintf("%x", md5.Sum([]byte(userID)))[:8]
		sessionID = fmt.Sprintf("session-%s-%s-%s-ws_%s",
			now.Format("20060102"),
			now.Format("150405"),
			userHash,
			workspaceHash)

		log.Printf("🔄 [会话存储] 步骤2.1: 生成初始会话ID: %s", sessionID)

		// 确保会话ID唯一
		originalSessionID := sessionID
		retryCount := 0
		for s.sessions[sessionID] != nil {
			retryCount++
			log.Printf("🔄 [会话存储] 步骤2.2: 会话ID冲突，重试第 %d 次", retryCount)
			time.Sleep(time.Millisecond) // 等待1毫秒确保时间戳不同
			now = time.Now()
			// 重试时也使用相同的用户哈希逻辑
			userHash := fmt.Sprintf("%x", md5.Sum([]byte(userID)))[:8]
			sessionID = fmt.Sprintf("session-%s-%s-%s-ws_%s",
				now.Format("20060102"),
				now.Format("150405"),
				userHash,
				workspaceHash)
			log.Printf("🔄 [会话存储] 步骤2.3: 新的会话ID: %s", sessionID)
		}

		if retryCount > 0 {
			log.Printf("🔄 [会话存储] 步骤2.4: 会话ID冲突解决，最终ID: %s (原始: %s, 重试: %d次)",
				sessionID, originalSessionID, retryCount)
		}

		log.Printf("🔄 [会话存储] 步骤2.5: 创建新会话对象")
		activeSession = models.NewSession(sessionID)

		// 添加用户ID和工作空间标识到元数据
		if activeSession.Metadata == nil {
			activeSession.Metadata = make(map[string]interface{})
		}
		activeSession.Metadata["userId"] = userID
		activeSession.Metadata["workspaceHash"] = workspaceHash

		log.Printf("🔄 [会话存储] 步骤2.6: 设置会话元数据: userId=%s, workspaceHash=%s", userID, workspaceHash)
		log.Printf("🔄 [会话存储] 🆕 创建新的工作空间会话: %s, 用户ID: %s, 工作空间: %s", sessionID, userID, workspaceHash)
	} else {
		log.Printf("🔄 [会话存储] ⚠️ 工作空间哈希为空，回退到原有逻辑")
		// 回退到原有的逻辑（向后兼容）
		return s.GetOrCreateActiveSession(userID, sessionTimeout)
	}

	log.Printf("🔄 [会话存储] 步骤3: 将新会话添加到存储中")
	s.sessions[sessionID] = activeSession

	// 保存新会话
	log.Printf("🔄 [会话存储] 步骤4: 保存新会话到文件")
	if err := s.saveSession(activeSession); err != nil {
		log.Printf("🔄 [会话存储] ❌ 保存新会话失败: %v", err)
		log.Printf("🔄 [会话存储] 步骤4.1: 回滚会话创建")
		delete(s.sessions, sessionID) // 回滚
		log.Printf("🔄 [会话存储] ===== GetOrCreateActiveSessionWithWorkspace失败 =====")
		return nil, false, fmt.Errorf("保存新会话失败: %w", err)
	}

	log.Printf("🔄 [会话存储] ✅ 新会话保存成功")
	log.Printf("🔄 [会话存储] 最终会话信息: ID=%s, 创建时间=%s, 最后活动=%s",
		activeSession.ID, activeSession.CreatedAt.Format("2006-01-02 15:04:05"),
		activeSession.LastActive.Format("2006-01-02 15:04:05"))
	log.Printf("🔄 [会话存储] 存储状态: 总会话数=%d", len(s.sessions))
	log.Printf("🔄 [会话存储] ===== GetOrCreateActiveSessionWithWorkspace完成(新建) =====")

	return activeSession, true, nil
}
