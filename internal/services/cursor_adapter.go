package services

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/contextkeeper/service/internal/models"
)

// DecisionService 临时存根结构体，替代已删除的decision_service.go
type DecisionService struct{}

// GetDecisionsForSession 临时存根方法
func (d *DecisionService) GetDecisionsForSession(ctx context.Context, sessionID string) ([]models.DecisionSummary, error) {
	return []models.DecisionSummary{}, nil
}

// LinkDecisionToEdits 临时存根方法
func (d *DecisionService) LinkDecisionToEdits(ctx context.Context, req models.LinkDecisionRequest) (*models.LinkDecisionResponse, error) {
	return &models.LinkDecisionResponse{Status: "not_implemented"}, nil
}

// GetLinkedSessions 临时存根方法
func (d *DecisionService) GetLinkedSessions(ctx context.Context, sessionID string) ([]models.SessionReference, error) {
	return []models.SessionReference{}, nil
}

// CreateDecision 临时存根方法
func (d *DecisionService) CreateDecision(ctx context.Context, req models.CreateDecisionRequest) (*models.CreateDecisionResponse, error) {
	return &models.CreateDecisionResponse{Status: "not_implemented"}, nil
}

// CreateSessionLink 临时存根方法
func (d *DecisionService) CreateSessionLink(ctx context.Context, req models.SessionLinkRequest) (*models.SessionLinkResponse, error) {
	return &models.SessionLinkResponse{Status: "not_implemented"}, nil
}

// CursorAdapter 提供针对Cursor编辑器的特定适配功能
type CursorAdapter struct {
	contextService  *ContextService
	decisionService *DecisionService // 新增决策服务
}

// NewCursorAdapter 创建新的Cursor适配器
func NewCursorAdapter(contextService *ContextService, decisionService *DecisionService) *CursorAdapter {
	return &CursorAdapter{
		contextService:  contextService,
		decisionService: decisionService,
	}
}

// AssociateCodeFile 关联代码文件到会话
// 除了基本的文件关联外，还会提取代码特性并存储
func (c *CursorAdapter) AssociateCodeFile(ctx context.Context, req models.MCPCodeAssociationRequest) error {
	// 1. 调用基础的文件关联功能
	if err := c.contextService.AssociateCodeFile(ctx, req); err != nil {
		return fmt.Errorf("基础文件关联失败: %w", err)
	}

	// 2. 提取代码特性并存储为向量数据
	codeFeatures, err := c.extractCodeFeatures(req.Content, req.Language)
	if err != nil {
		// 即使特性提取失败，也不影响整体关联
		// 只记录错误但继续执行
		fmt.Printf("警告: 代码特性提取失败: %v\n", err)
	} else if codeFeatures != "" {
		// 存储提取的代码特性
		metadata := map[string]interface{}{
			"sessionId":   req.SessionID,
			"filePath":    req.FilePath,
			"language":    req.Language,
			"contentType": "code_features",
			"timestamp":   time.Now().Unix(),
		}

		// 调用向量存储服务存储代码特性
		_, err = c.contextService.StoreContext(ctx, models.StoreContextRequest{
			SessionID: req.SessionID,
			Content:   codeFeatures,
			Metadata:  metadata,
			Priority:  "P1", // 代码特性通常较重要
		})

		if err != nil {
			fmt.Printf("警告: 存储代码特性失败: %v\n", err)
		}
	}

	// 3. 为会话添加文件上下文摘要
	filename := filepath.Base(req.FilePath)
	extension := filepath.Ext(req.FilePath)
	language := req.Language
	if language == "" {
		language = getLanguageFromExtension(extension)
	}

	summary := fmt.Sprintf("文件 '%s' (类型: %s) 已关联到会话。", filename, language)

	// 存储文件关联的系统消息
	sysMsg := models.NewMessage(
		req.SessionID,
		"system",
		summary,
		"text",
		"P3",
		map[string]interface{}{
			"type":      "file_association",
			"filePath":  req.FilePath,
			"language":  language,
			"timestamp": time.Now().Unix(),
		},
	)

	// 更新会话消息
	messages := []*models.Message{sysMsg}
	if err := c.contextService.sessionStore.StoreMessages(req.SessionID, messages); err != nil {
		fmt.Printf("警告: 存储文件关联系统消息失败: %v\n", err)
	}

	return nil
}

// extractCodeFeatures 从代码内容中提取特征
// 例如函数定义、类定义、导入语句等
func (c *CursorAdapter) extractCodeFeatures(content, language string) (string, error) {
	if content == "" {
		return "", nil
	}

	var features []string
	lines := strings.Split(content, "\n")

	switch strings.ToLower(language) {
	case "javascript", "typescript", "jsx", "tsx":
		features = extractJSFeatures(lines)
	case "python":
		features = extractPythonFeatures(lines)
	case "go":
		features = extractGoFeatures(lines)
	case "java":
		features = extractJavaFeatures(lines)
	default:
		// 对于不支持的语言，使用通用提取方法
		features = extractGenericFeatures(lines)
	}

	return strings.Join(features, "\n"), nil
}

// extractJSFeatures 提取JavaScript/TypeScript代码特性
func extractJSFeatures(lines []string) []string {
	var features []string
	var imports []string
	var functions []string
	var classes []string
	var variables []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 提取导入语句
		if strings.HasPrefix(trimmedLine, "import ") || strings.HasPrefix(trimmedLine, "require(") {
			imports = append(imports, trimmedLine)
		}

		// 提取函数定义
		if strings.HasPrefix(trimmedLine, "function ") ||
			strings.Contains(trimmedLine, " function(") ||
			strings.Contains(trimmedLine, " => {") {
			functions = append(functions, trimmedLine)
		}

		// 提取类定义
		if strings.HasPrefix(trimmedLine, "class ") {
			classes = append(classes, trimmedLine)
		}

		// 提取关键变量定义
		if strings.HasPrefix(trimmedLine, "const ") ||
			strings.HasPrefix(trimmedLine, "let ") ||
			strings.HasPrefix(trimmedLine, "var ") {
			variables = append(variables, trimmedLine)
		}
	}

	// 汇总特性
	if len(imports) > 0 {
		features = append(features, "导入语句:")
		for _, imp := range imports[:min(5, len(imports))] {
			features = append(features, "  "+imp)
		}
	}

	if len(classes) > 0 {
		features = append(features, "类定义:")
		for _, cls := range classes {
			features = append(features, "  "+cls)
		}
	}

	if len(functions) > 0 {
		features = append(features, "函数定义:")
		for _, fn := range functions[:min(10, len(functions))] {
			features = append(features, "  "+fn)
		}
	}

	if len(variables) > 0 {
		features = append(features, "关键变量:")
		for _, v := range variables[:min(5, len(variables))] {
			features = append(features, "  "+v)
		}
	}

	return features
}

// extractPythonFeatures 提取Python代码特性
func extractPythonFeatures(lines []string) []string {
	var features []string
	var imports []string
	var functions []string
	var classes []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 提取导入语句
		if strings.HasPrefix(trimmedLine, "import ") || strings.HasPrefix(trimmedLine, "from ") {
			imports = append(imports, trimmedLine)
		}

		// 提取函数定义
		if strings.HasPrefix(trimmedLine, "def ") {
			functions = append(functions, trimmedLine)
		}

		// 提取类定义
		if strings.HasPrefix(trimmedLine, "class ") {
			classes = append(classes, trimmedLine)
		}
	}

	// 汇总特性
	if len(imports) > 0 {
		features = append(features, "导入语句:")
		for _, imp := range imports[:min(5, len(imports))] {
			features = append(features, "  "+imp)
		}
	}

	if len(classes) > 0 {
		features = append(features, "类定义:")
		for _, cls := range classes {
			features = append(features, "  "+cls)
		}
	}

	if len(functions) > 0 {
		features = append(features, "函数定义:")
		for _, fn := range functions[:min(10, len(functions))] {
			features = append(features, "  "+fn)
		}
	}

	return features
}

// extractGoFeatures 提取Go代码特性
func extractGoFeatures(lines []string) []string {
	var features []string
	var imports []string
	var functions []string
	var structs []string
	var interfaces []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 提取导入语句
		if strings.HasPrefix(trimmedLine, "import ") {
			imports = append(imports, trimmedLine)
		}

		// 提取函数定义
		if strings.HasPrefix(trimmedLine, "func ") {
			functions = append(functions, trimmedLine)
		}

		// 提取结构体定义
		if strings.HasPrefix(trimmedLine, "type ") && strings.Contains(trimmedLine, " struct ") {
			structs = append(structs, trimmedLine)
		}

		// 提取接口定义
		if strings.HasPrefix(trimmedLine, "type ") && strings.Contains(trimmedLine, " interface ") {
			interfaces = append(interfaces, trimmedLine)
		}
	}

	// 汇总特性
	if len(imports) > 0 {
		features = append(features, "导入语句:")
		for _, imp := range imports[:min(5, len(imports))] {
			features = append(features, "  "+imp)
		}
	}

	if len(structs) > 0 {
		features = append(features, "结构体定义:")
		for _, s := range structs {
			features = append(features, "  "+s)
		}
	}

	if len(interfaces) > 0 {
		features = append(features, "接口定义:")
		for _, i := range interfaces {
			features = append(features, "  "+i)
		}
	}

	if len(functions) > 0 {
		features = append(features, "函数定义:")
		for _, fn := range functions[:min(10, len(functions))] {
			features = append(features, "  "+fn)
		}
	}

	return features
}

// extractJavaFeatures 提取Java代码特性
func extractJavaFeatures(lines []string) []string {
	var features []string
	var imports []string
	var methods []string
	var classes []string
	var interfaces []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 提取导入语句
		if strings.HasPrefix(trimmedLine, "import ") {
			imports = append(imports, trimmedLine)
		}

		// 提取类定义
		if strings.HasPrefix(trimmedLine, "public class ") ||
			strings.HasPrefix(trimmedLine, "private class ") ||
			strings.HasPrefix(trimmedLine, "protected class ") ||
			strings.HasPrefix(trimmedLine, "class ") {
			classes = append(classes, trimmedLine)
		}

		// 提取接口定义
		if strings.HasPrefix(trimmedLine, "public interface ") ||
			strings.HasPrefix(trimmedLine, "private interface ") ||
			strings.HasPrefix(trimmedLine, "protected interface ") ||
			strings.HasPrefix(trimmedLine, "interface ") {
			interfaces = append(interfaces, trimmedLine)
		}

		// 提取方法定义
		if (strings.Contains(trimmedLine, "public ") ||
			strings.Contains(trimmedLine, "private ") ||
			strings.Contains(trimmedLine, "protected ")) &&
			strings.Contains(trimmedLine, "(") &&
			!strings.HasPrefix(trimmedLine, "//") {
			methods = append(methods, trimmedLine)
		}
	}

	// 汇总特性
	if len(imports) > 0 {
		features = append(features, "导入语句:")
		for _, imp := range imports[:min(5, len(imports))] {
			features = append(features, "  "+imp)
		}
	}

	if len(classes) > 0 {
		features = append(features, "类定义:")
		for _, cls := range classes {
			features = append(features, "  "+cls)
		}
	}

	if len(interfaces) > 0 {
		features = append(features, "接口定义:")
		for _, i := range interfaces {
			features = append(features, "  "+i)
		}
	}

	if len(methods) > 0 {
		features = append(features, "方法定义:")
		for _, m := range methods[:min(10, len(methods))] {
			features = append(features, "  "+m)
		}
	}

	return features
}

// extractGenericFeatures 提取通用代码特性
func extractGenericFeatures(lines []string) []string {
	var features []string

	// 提取文件的摘要信息
	if len(lines) > 0 {
		var summary []string
		// 查找可能的注释
		for i := 0; i < min(20, len(lines)); i++ {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "//") ||
				strings.HasPrefix(line, "#") ||
				strings.HasPrefix(line, "/*") ||
				strings.HasPrefix(line, "*") ||
				strings.HasPrefix(line, "'''") ||
				strings.HasPrefix(line, "\"\"\"") {
				summary = append(summary, line)
			}
		}

		if len(summary) > 0 {
			features = append(features, "文件注释:")
			for _, s := range summary[:min(5, len(summary))] {
				features = append(features, "  "+s)
			}
		}
	}

	// 提取关键行
	var keyLines []string
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 10 && !strings.HasPrefix(trimmedLine, "//") && !strings.HasPrefix(trimmedLine, "#") {
			keyLines = append(keyLines, trimmedLine)
		}
	}

	if len(keyLines) > 0 {
		features = append(features, "关键代码行:")
		step := max(1, len(keyLines)/10)
		for i := 0; i < len(keyLines) && i < 50; i += step {
			features = append(features, "  "+keyLines[i])
		}
	}

	return features
}

// getLanguageFromExtension 根据文件扩展名判断编程语言
func getLanguageFromExtension(ext string) string {
	langMap := map[string]string{
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".py":    "python",
		".go":    "go",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".php":   "php",
		".rb":    "ruby",
		".swift": "swift",
		".kt":    "kotlin",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return "unknown"
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RecordEditAction 记录编辑行为并增强上下文感知
func (c *CursorAdapter) RecordEditAction(ctx context.Context, req models.MCPEditRecordRequest) error {
	// 1. 生成编辑ID
	editID := uuid.New().String()

	// 2. 调用基础的编辑记录功能（增强版）
	if err := c.enhancedRecordEditAction(ctx, req, editID); err != nil {
		return fmt.Errorf("基础编辑记录失败: %w", err)
	}

	// 3. 生成编辑描述并存储为系统消息
	editDescription := c.generateEditDescription(req)

	// 存储编辑记录的系统消息
	sysMsg := models.NewMessage(
		req.SessionID,
		"system",
		editDescription,
		"text",
		"P3",
		map[string]interface{}{
			"type":      "edit_record",
			"filePath":  req.FilePath,
			"editType":  req.Type,
			"position":  req.Position,
			"editId":    editID, // 新增编辑ID
			"timestamp": time.Now().Unix(),
		},
	)

	// 更新会话消息
	messages := []*models.Message{sysMsg}
	if err := c.contextService.sessionStore.StoreMessages(req.SessionID, messages); err != nil {
		fmt.Printf("警告: 存储编辑记录系统消息失败: %v\n", err)
	}

	// 4. 为重要编辑存储向量记录
	if req.Type == "modify" || req.Type == "insert" {
		// 只有修改和插入操作需要存储内容
		if len(req.Content) > 20 { // 只存储有意义的编辑
			metadata := map[string]interface{}{
				"sessionId":   req.SessionID,
				"filePath":    req.FilePath,
				"editType":    req.Type,
				"position":    req.Position,
				"editId":      editID, // 新增编辑ID
				"contentType": "edit_record",
				"timestamp":   time.Now().Unix(),
			}

			// 存储编辑记录内容
			_, err := c.contextService.StoreContext(ctx, models.StoreContextRequest{
				SessionID: req.SessionID,
				Content:   editDescription + "\n\n编辑内容: " + req.Content,
				Metadata:  metadata,
				Priority:  "P2", // 编辑记录通常较重要但不如代码特性重要
			})

			if err != nil {
				fmt.Printf("警告: 存储编辑记录内容失败: %v\n", err)
			}
		}
	}

	// 5. 分析编辑内容判断是否与设计决策相关
	if isSignificantEdit(req.Type, req.Content) {
		// 检查是否存在最近的设计决策
		decisions, err := c.decisionService.GetDecisionsForSession(ctx, req.SessionID)
		if err == nil && len(decisions) > 0 {
			// 找到最近的决策（5分钟内）
			now := time.Now().Unix()
			for _, decision := range decisions {
				// 判断决策是否是最近5分钟内创建的
				if now-decision.Timestamp < 300 { // 5分钟 = 300秒
					// 自动关联最近的设计决策
					linkReq := models.LinkDecisionRequest{
						SessionID:   req.SessionID,
						DecisionID:  decision.ID,
						EditIDs:     []string{editID},
						Strength:    0.8, // 自动关联的强度稍低
						Description: "系统自动关联的近期编辑",
					}
					_, linkErr := c.decisionService.LinkDecisionToEdits(ctx, linkReq)
					if linkErr != nil {
						fmt.Printf("警告: 自动关联编辑到决策失败: %v\n", linkErr)
					}
					break
				}
			}
		}
	}

	return nil
}

// enhancedRecordEditAction 增强版编辑记录功能
func (c *CursorAdapter) enhancedRecordEditAction(ctx context.Context, req models.MCPEditRecordRequest, editID string) error {
	// 1. 获取会话信息
	session, err := c.contextService.sessionStore.GetSession(req.SessionID)
	if err != nil {
		return fmt.Errorf("获取会话失败: %w", err)
	}

	// 2. 创建增强版编辑记录
	editAction := models.EditAction{
		ID:        editID,
		Timestamp: time.Now().Unix(),
		FilePath:  req.FilePath,
		Type:      req.Type,
		Position:  req.Position,
		Content:   req.Content,
		Tags:      []string{}, // 初始化空标签列表
		Metadata: map[string]interface{}{
			"language":   getLanguageFromPath(req.FilePath),
			"createTime": time.Now().Format(time.RFC3339),
		},
	}

	// 3. 分析编辑内容，添加自动标签
	editAction.Tags = c.generateEditTags(req)

	// 4. 更新会话的编辑历史
	if session.EditHistory == nil {
		session.EditHistory = make([]*models.EditAction, 0)
	}
	session.EditHistory = append(session.EditHistory, &editAction)

	// 5. 更新关联代码文件的最后编辑时间
	if session.CodeContext == nil {
		session.CodeContext = make(map[string]*models.CodeFile)
	}

	codeFile, exists := session.CodeContext[req.FilePath]
	if !exists {
		// 如果文件还未关联，自动关联
		language := getLanguageFromPath(req.FilePath)
		codeFile = &models.CodeFile{
			Path:     req.FilePath,
			Language: language,
			LastEdit: time.Now().Unix(),
		}
		session.CodeContext[req.FilePath] = codeFile
	} else {
		// 更新最后编辑时间
		codeFile.LastEdit = time.Now().Unix()
	}

	// 6. 更新会话
	return c.contextService.sessionStore.UpdateSession(session.ID, "")
}

// generateEditTags 生成编辑标签
func (c *CursorAdapter) generateEditTags(req models.MCPEditRecordRequest) []string {
	var tags []string

	// 1. 添加基本标签
	tags = append(tags, req.Type) // 添加编辑类型作为标签

	// 2. 根据文件类型添加标签
	ext := filepath.Ext(req.FilePath)
	if ext != "" {
		tags = append(tags, ext[1:]) // 去掉点号
	}

	// 3. 根据内容大小添加标签
	contentLen := len(req.Content)
	if contentLen > 500 {
		tags = append(tags, "large-edit")
	} else if contentLen > 100 {
		tags = append(tags, "medium-edit")
	} else if contentLen > 0 {
		tags = append(tags, "small-edit")
	}

	// 4. 尝试检测内容特征
	content := strings.ToLower(req.Content)

	// 检测是否包含特定关键词
	if strings.Contains(content, "fix") || strings.Contains(content, "bug") || strings.Contains(content, "修复") {
		tags = append(tags, "bugfix")
	}

	if strings.Contains(content, "feat") || strings.Contains(content, "功能") || strings.Contains(content, "feature") {
		tags = append(tags, "feature")
	}

	if strings.Contains(content, "refactor") || strings.Contains(content, "重构") {
		tags = append(tags, "refactor")
	}

	if strings.Contains(content, "test") || strings.Contains(content, "测试") {
		tags = append(tags, "test")
	}

	if strings.Contains(content, "doc") || strings.Contains(content, "文档") {
		tags = append(tags, "docs")
	}

	return tags
}

// isSignificantEdit 判断编辑是否重要
func isSignificantEdit(editType string, content string) bool {
	// 删除操作通常不算重要编辑（除非删除了大量内容）
	if editType == "delete" && len(content) < 100 {
		return false
	}

	// 内容太短也不算重要
	if len(content) < 50 {
		return false
	}

	// 其他情况认为是重要编辑
	return true
}

// getLanguageFromPath 根据文件路径判断编程语言
func getLanguageFromPath(path string) string {
	ext := filepath.Ext(path)
	return getLanguageFromExtension(ext)
}

// generateEditDescription 生成编辑描述
func (c *CursorAdapter) generateEditDescription(req models.MCPEditRecordRequest) string {
	filename := filepath.Base(req.FilePath)
	var action string

	switch req.Type {
	case "insert":
		action = "插入"
	case "delete":
		action = "删除"
	case "modify":
		action = "修改"
	default:
		action = req.Type
	}

	description := fmt.Sprintf("文件 '%s' 在位置 %d 进行了%s操作", filename, req.Position, action)

	if req.Content != "" {
		if len(req.Content) > 50 {
			description += fmt.Sprintf("，内容摘要: %s...", req.Content[:50])
		} else {
			description += fmt.Sprintf("，内容: %s", req.Content)
		}
	}

	return description
}

// ExtractProgrammingContext 提取编程上下文
// 基于会话状态和相关代码文件
func (c *CursorAdapter) ExtractProgrammingContext(ctx context.Context, sessionID string, query string) (*models.ProgrammingContext, error) {
	// 1. 获取会话信息
	session, err := c.contextService.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 2. 初始化上下文结构
	progContext := &models.ProgrammingContext{
		SessionID:         sessionID,
		AssociatedFiles:   make([]models.CodeFileInfo, 0),
		RecentEdits:       make([]models.EditInfo, 0),
		RelevantSnippets:  make([]models.CodeSnippet, 0),
		DesignDecisions:   make([]models.DecisionSummary, 0),
		LinkedSessions:    make([]models.SessionReference, 0),
		RelatedContexts:   make([]models.ContextReference, 0),
		ExtractedFeatures: make([]string, 0),
		Statistics:        models.ProgrammingStatistics{},
	}

	// 3. 收集关联文件信息
	if session.CodeContext != nil {
		for path, file := range session.CodeContext {
			fileInfo := models.CodeFileInfo{
				Path:     path,
				Language: file.Language,
				LastEdit: file.LastEdit,
				Summary:  file.Summary,
			}
			progContext.AssociatedFiles = append(progContext.AssociatedFiles, fileInfo)
		}
	}

	// 4. 收集最近编辑记录
	if session.EditHistory != nil {
		// 只获取最近的10个编辑
		editCount := len(session.EditHistory)
		startIdx := 0
		if editCount > 10 {
			startIdx = editCount - 10
		}

		for i := startIdx; i < editCount; i++ {
			edit := session.EditHistory[i]
			editInfo := models.EditInfo{
				ID:               edit.ID,
				Timestamp:        edit.Timestamp,
				FilePath:         edit.FilePath,
				Type:             edit.Type,
				Position:         edit.Position,
				Content:          edit.Content,
				RelatedDecisions: edit.DecisionIDs,
				Tags:             edit.Tags,
			}
			progContext.RecentEdits = append(progContext.RecentEdits, editInfo)
		}
	}

	// 5. 获取设计决策
	decisions, err := c.decisionService.GetDecisionsForSession(ctx, sessionID)
	if err == nil && len(decisions) > 0 {
		// 只保留最近的5个决策
		maxDecisions := 5
		if len(decisions) > maxDecisions {
			// 按时间戳排序，保留最新的
			sort.Slice(decisions, func(i, j int) bool {
				return decisions[i].Timestamp > decisions[j].Timestamp
			})
			decisions = decisions[:maxDecisions]
		}
		progContext.DesignDecisions = decisions
	}

	// 6. 获取关联会话
	linkedSessions, err := c.decisionService.GetLinkedSessions(ctx, sessionID)
	if err == nil {
		progContext.LinkedSessions = linkedSessions
	}

	// 7. 如果有查询，搜索相关代码片段
	if query != "" {
		// 准备过滤条件，只搜索代码特性
		filters := map[string]interface{}{
			"sessionId":   sessionID,
			"contentType": "code_features",
		}

		// 使用文本搜索功能
		results, err := c.contextService.vectorService.SearchWithTextAndFilters(ctx, query, 5, filters, false)

		if err == nil && len(results) > 0 {
			for _, result := range results {
				if content, ok := result.Fields["content"].(string); ok {
					filePath := ""
					if path, ok := result.Fields["filePath"].(string); ok {
						filePath = path
					}

					snippet := models.CodeSnippet{
						Content:  content,
						FilePath: filePath,
						Score:    result.Score,
						Context:  c.getSnippetContext(filePath, content),
					}
					progContext.RelevantSnippets = append(progContext.RelevantSnippets, snippet)
				}
			}
		}

		// 8. 搜索相关的对话上下文
		contextFilters := map[string]interface{}{
			"sessionId":   sessionID,
			"contentType": "text",
		}
		contextResults, err := c.contextService.vectorService.SearchWithTextAndFilters(ctx, query, 3, contextFilters, false)
		if err == nil && len(contextResults) > 0 {
			for _, result := range contextResults {
				if content, ok := result.Fields["content"].(string); ok {
					reference := models.ContextReference{
						Type:           "conversation",
						Content:        content,
						SourceID:       result.ID,
						Timestamp:      int64(result.Fields["timestamp"].(float64)),
						RelevanceScore: result.Score,
					}
					progContext.RelatedContexts = append(progContext.RelatedContexts, reference)
				}
			}
		}
	}

	// 9. 提取特性统计
	progContext.ExtractedFeatures = c.extractSessionFeatures(session)

	// 10. 生成编程统计信息
	progContext.Statistics = c.generateProgrammingStatistics(session)

	return progContext, nil
}

// getSnippetContext 获取代码片段的上下文
func (c *CursorAdapter) getSnippetContext(filePath, content string) string {
	// 简单实现：返回文件名作为上下文
	if filePath != "" {
		return fmt.Sprintf("来自文件: %s", filepath.Base(filePath))
	}
	return ""
}

// generateProgrammingStatistics 生成编程统计信息
func (c *CursorAdapter) generateProgrammingStatistics(session *models.Session) models.ProgrammingStatistics {
	stats := models.ProgrammingStatistics{
		LanguageUsage:       make(map[string]int),
		EditsByFile:         make(map[string]int),
		ActivityByDay:       make(map[string]int),
		DecisionsByCategory: make(map[string]int),
	}

	// 统计文件数量
	if session.CodeContext != nil {
		stats.TotalFiles = len(session.CodeContext)

		// 统计语言使用情况
		for _, file := range session.CodeContext {
			if file.Language != "" {
				stats.LanguageUsage[file.Language]++
			}
		}
	}

	// 统计编辑数量和分布
	if session.EditHistory != nil {
		stats.TotalEdits = len(session.EditHistory)

		for _, edit := range session.EditHistory {
			// 按文件统计编辑
			stats.EditsByFile[edit.FilePath]++

			// 按日期统计活动
			day := time.Unix(edit.Timestamp, 0).Format("2006-01-02")
			stats.ActivityByDay[day]++
		}
	}

	return stats
}

// extractSessionFeatures 从会话中提取编程特性
func (c *CursorAdapter) extractSessionFeatures(session *models.Session) []string {
	var features []string

	// 1. 统计使用的编程语言
	langCount := make(map[string]int)
	if session.CodeContext != nil {
		for _, file := range session.CodeContext {
			if file.Language != "" {
				langCount[file.Language]++
			}
		}
	}

	if len(langCount) > 0 {
		features = append(features, "使用的编程语言:")
		for lang, count := range langCount {
			features = append(features, fmt.Sprintf("  %s: %d个文件", lang, count))
		}
	}

	// 2. 统计编辑类型
	if session.EditHistory != nil && len(session.EditHistory) > 0 {
		editCount := len(session.EditHistory)
		insertCount := 0
		deleteCount := 0
		modifyCount := 0

		for _, edit := range session.EditHistory {
			switch edit.Type {
			case "insert":
				insertCount++
			case "delete":
				deleteCount++
			case "modify":
				modifyCount++
			}
		}

		features = append(features, "编辑操作统计:")
		features = append(features, fmt.Sprintf("  总编辑数: %d", editCount))
		if insertCount > 0 {
			features = append(features, fmt.Sprintf("  插入操作: %d", insertCount))
		}
		if deleteCount > 0 {
			features = append(features, fmt.Sprintf("  删除操作: %d", deleteCount))
		}
		if modifyCount > 0 {
			features = append(features, fmt.Sprintf("  修改操作: %d", modifyCount))
		}
	}

	// 3. 统计活跃文件
	if session.EditHistory != nil && len(session.EditHistory) > 0 {
		fileCount := make(map[string]int)
		for _, edit := range session.EditHistory {
			fileCount[edit.FilePath]++
		}

		if len(fileCount) > 0 {
			features = append(features, "活跃文件:")
			type fileActivity struct {
				path  string
				count int
			}

			// 转换为切片以便排序
			activeFiles := make([]fileActivity, 0, len(fileCount))
			for path, count := range fileCount {
				activeFiles = append(activeFiles, fileActivity{path, count})
			}

			// 按活跃度排序
			sort.Slice(activeFiles, func(i, j int) bool {
				return activeFiles[i].count > activeFiles[j].count
			})

			// 取前5个最活跃的文件
			for i := 0; i < min(5, len(activeFiles)); i++ {
				file := activeFiles[i]
				features = append(features, fmt.Sprintf("  %s: %d次编辑", filepath.Base(file.path), file.count))
			}
		}
	}

	return features
}

// FormatProgrammingContext 将编程上下文格式化为字符串
// 用于在MCP响应中呈现
func (c *CursorAdapter) FormatProgrammingContext(ctx *models.ProgrammingContext) string {
	if ctx == nil {
		return ""
	}

	var sb strings.Builder

	// 添加会话标识
	sb.WriteString(fmt.Sprintf("会话ID: %s\n\n", ctx.SessionID))

	// 添加特性统计
	if len(ctx.ExtractedFeatures) > 0 {
		sb.WriteString("【会话特性】\n")
		for _, feature := range ctx.ExtractedFeatures {
			sb.WriteString(feature + "\n")
		}
		sb.WriteString("\n")
	}

	// 添加设计决策信息（新增）
	if len(ctx.DesignDecisions) > 0 {
		sb.WriteString("【设计决策】\n")
		for i, decision := range ctx.DesignDecisions {
			timeStr := time.Unix(decision.Timestamp, 0).Format("2006-01-02 15:04:05")
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, timeStr, decision.Title))
			if decision.Description != "" {
				sb.WriteString("   " + decision.Description + "\n")
			}
			if decision.Category != "" {
				sb.WriteString(fmt.Sprintf("   类别: %s\n", decision.Category))
			}
			if len(decision.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("   标签: %s\n", strings.Join(decision.Tags, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// 添加关联会话（新增）
	if len(ctx.LinkedSessions) > 0 {
		sb.WriteString("【关联会话】\n")
		for i, session := range ctx.LinkedSessions {
			relationship := getRelationshipDisplayName(session.Relationship)
			timeStr := time.Unix(session.Timestamp, 0).Format("2006-01-02 15:04:05")
			sb.WriteString(fmt.Sprintf("%d. [%s] 会话 %s (%s关系)\n",
				i+1, timeStr, session.SessionID, relationship))
			if session.Description != "" {
				sb.WriteString("   说明: " + session.Description + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// 添加关联文件
	if len(ctx.AssociatedFiles) > 0 {
		sb.WriteString("【关联文件】\n")
		for i, file := range ctx.AssociatedFiles {
			sb.WriteString(fmt.Sprintf("%d. %s (语言: %s)\n", i+1, file.Path, file.Language))
			if file.Summary != "" {
				sb.WriteString("   摘要: " + file.Summary + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// 添加最近编辑
	if len(ctx.RecentEdits) > 0 {
		sb.WriteString("【最近编辑】\n")
		for i, edit := range ctx.RecentEdits {
			timeStr := time.Unix(edit.Timestamp, 0).Format("2006-01-02 15:04:05")
			filename := filepath.Base(edit.FilePath)
			sb.WriteString(fmt.Sprintf("%d. [%s] %s 在位置 %d 进行了%s操作\n",
				i+1, timeStr, filename, edit.Position, edit.Type))

			// 添加编辑标签（新增）
			if len(edit.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("   标签: %s\n", strings.Join(edit.Tags, ", ")))
			}

			// 添加关联决策（新增）
			if len(edit.RelatedDecisions) > 0 {
				sb.WriteString(fmt.Sprintf("   关联决策: %d个\n", len(edit.RelatedDecisions)))
			}
		}
		sb.WriteString("\n")
	}

	// 添加相关代码片段
	if len(ctx.RelevantSnippets) > 0 {
		sb.WriteString("【相关代码片段】\n")
		for i, snippet := range ctx.RelevantSnippets {
			filename := ""
			if snippet.FilePath != "" {
				filename = " (" + filepath.Base(snippet.FilePath) + ")"
			}
			sb.WriteString(fmt.Sprintf("%d. 相关度: %.4f%s\n", i+1, 1.0-snippet.Score, filename))
			if snippet.Context != "" {
				sb.WriteString("   " + snippet.Context + "\n")
			}
			sb.WriteString("```\n" + snippet.Content + "\n```\n\n")
		}
	}

	// 添加相关上下文引用（新增）
	if len(ctx.RelatedContexts) > 0 {
		sb.WriteString("【相关上下文】\n")
		for i, ref := range ctx.RelatedContexts {
			timeStr := time.Unix(ref.Timestamp, 0).Format("2006-01-02 15:04:05")
			sb.WriteString(fmt.Sprintf("%d. [%s] %s类型 (相关度: %.4f)\n",
				i+1, timeStr, getContextTypeName(ref.Type), 1.0-ref.RelevanceScore))
			sb.WriteString("   " + ref.Content + "\n\n")
		}
	}

	// 添加统计信息（新增）
	sb.WriteString("【编程统计】\n")
	sb.WriteString(fmt.Sprintf("总文件数: %d\n", ctx.Statistics.TotalFiles))
	sb.WriteString(fmt.Sprintf("总编辑数: %d\n", ctx.Statistics.TotalEdits))

	if len(ctx.Statistics.LanguageUsage) > 0 {
		sb.WriteString("语言分布: ")
		langParts := make([]string, 0)
		for lang, count := range ctx.Statistics.LanguageUsage {
			langParts = append(langParts, fmt.Sprintf("%s(%d)", lang, count))
		}
		sb.WriteString(strings.Join(langParts, ", ") + "\n")
	}

	return sb.String()
}

// getContextTypeName 获取上下文类型的中文名称
func getContextTypeName(typeName string) string {
	switch typeName {
	case "conversation":
		return "对话"
	case "code":
		return "代码"
	case "decision":
		return "决策"
	default:
		return typeName
	}
}

// getRelationshipDisplayName 获取关系类型的中文名称
func getRelationshipDisplayName(relationship string) string {
	switch relationship {
	case models.RelationshipContinuation:
		return "延续"
	case models.RelationshipReference:
		return "引用"
	case models.RelationshipRelated:
		return "相关"
	default:
		return relationship
	}
}

// FormatMCPResponse 格式化MCP协议响应
// 将会话状态、短期记忆、长期记忆和编程上下文组合为MCP格式响应
func (c *CursorAdapter) FormatMCPResponse(ctx context.Context, sessionID string, query string) (*models.ContextResponse, error) {
	// 1. 获取会话状态
	sessionState, err := c.contextService.sessionStore.GetSessionState(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话状态失败: %w", err)
	}

	// 2. 获取短期记忆
	shortTermMemory, err := c.contextService.GetShortTermMemory(ctx, sessionID, 10)
	if err != nil {
		shortTermMemory = "【最近对话】\n暂无最近对话记录"
	}

	// 3. 获取长期记忆（如果有查询）
	var longTermMemory string
	if query != "" {
		// 构建检索请求
		retrieveReq := models.RetrieveContextRequest{
			SessionID: sessionID,
			Query:     query,
			Limit:     2000,
		}

		// 执行检索
		contextResp, err := c.contextService.RetrieveContext(ctx, retrieveReq)
		if err != nil {
			longTermMemory = "【相关历史】\n检索历史记忆时发生错误"
		} else {
			longTermMemory = contextResp.LongTermMemory
		}
	} else {
		longTermMemory = "【相关历史】\n未提供查询内容，无法检索相关记忆"
	}

	// 4. 获取编程上下文
	progContext, err := c.ExtractProgrammingContext(ctx, sessionID, query)
	var relevantKnowledge string
	if err != nil {
		relevantKnowledge = "【编程上下文】\n提取编程上下文时发生错误"
	} else {
		relevantKnowledge = "【编程上下文】\n" + c.FormatProgrammingContext(progContext)
	}

	// 5. 组装MCP响应
	response := &models.ContextResponse{
		SessionState:      sessionState,
		ShortTermMemory:   shortTermMemory,
		LongTermMemory:    longTermMemory,
		RelevantKnowledge: relevantKnowledge,
	}

	return response, nil
}

// GetContextForCursor 获取适合Cursor的上下文
// 综合会话状态、对话记忆和编程特定上下文
func (c *CursorAdapter) GetContextForCursor(ctx context.Context, req models.RetrieveContextRequest) (*models.ContextResponse, error) {
	// 首先尝试获取标准上下文
	standardResp, err := c.contextService.RetrieveContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("检索标准上下文失败: %w", err)
	}

	// 再获取编程相关上下文
	progContext, err := c.ExtractProgrammingContext(ctx, req.SessionID, req.Query)
	if err != nil {
		// 如果编程上下文获取失败，仍然返回标准上下文
		return &standardResp, nil
	}

	// 组合上下文
	programmingInfo := c.FormatProgrammingContext(progContext)
	if programmingInfo != "" {
		// 添加编程上下文到相关知识部分
		if standardResp.RelevantKnowledge == "" {
			standardResp.RelevantKnowledge = "【编程上下文】\n" + programmingInfo
		} else {
			standardResp.RelevantKnowledge += "\n\n【编程上下文】\n" + programmingInfo
		}
	}

	return &standardResp, nil
}
