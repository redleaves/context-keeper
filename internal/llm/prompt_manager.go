package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// =============================================================================
// Prompt工程设计
// =============================================================================

// PromptManager Prompt管理器
type PromptManager struct {
	templates map[string]*PromptTemplate
	config    *PromptConfig
}

// PromptTemplate Prompt模板
type PromptTemplate struct {
	Name         string          `json:"name"`
	SystemPrompt string          `json:"system_prompt"`
	UserTemplate string          `json:"user_template"`
	Variables    []string        `json:"variables"`
	OutputFormat string          `json:"output_format"`
	Examples     []PromptExample `json:"examples"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Tags         []string        `json:"tags"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// PromptExample Prompt示例
type PromptExample struct {
	Input     map[string]interface{} `json:"input"`
	Output    string                 `json:"output"`
	Reasoning []string               `json:"reasoning"`
}

// PromptConfig Prompt配置
type PromptConfig struct {
	DefaultLanguage string            `json:"default_language"`
	MaxTokens       int               `json:"max_tokens"`
	Temperature     float64           `json:"temperature"`
	CustomVars      map[string]string `json:"custom_vars"`
}

// PromptContext Prompt上下文
type PromptContext struct {
	SessionHistory   []Message              `json:"session_history"`
	WorkspaceContext *WorkspaceContext      `json:"workspace_context"`
	UserProfile      *UserProfile           `json:"user_profile"`
	AnalysisType     string                 `json:"analysis_type"`
	OriginalQuery    string                 `json:"original_query"`
	ThreeElements    *ThreeElementsModel    `json:"three_elements"`
	RetrievalContext *RetrievalContext      `json:"retrieval_context"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Message 消息结构
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// WorkspaceContext 工作空间上下文
type WorkspaceContext struct {
	ProjectType string                 `json:"project_type"`
	TechStack   []string               `json:"tech_stack"`
	ProjectName string                 `json:"project_name"`
	Environment string                 `json:"environment"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// UserProfile 用户画像
type UserProfile struct {
	TechStack       []string               `json:"tech_stack"`
	ExperienceLevel string                 `json:"experience_level"`
	Role            string                 `json:"role"`
	Domain          string                 `json:"domain"`
	Preferences     map[string]interface{} `json:"preferences"`
}

// ThreeElementsModel 三要素模型（简化版）
type ThreeElementsModel struct {
	User      UserElement      `json:"user"`
	Situation SituationElement `json:"situation"`
	Problem   ProblemElement   `json:"problem"`
}

type UserElement struct {
	TechStack       []string `json:"tech_stack"`
	ExperienceLevel string   `json:"experience_level"`
	Role            string   `json:"role"`
	Domain          string   `json:"domain"`
}

type SituationElement struct {
	ProjectType     string            `json:"project_type"`
	TechEnvironment map[string]string `json:"tech_environment"`
	BusinessContext string            `json:"business_context"`
}

type ProblemElement struct {
	ExplicitProblem string   `json:"explicit_problem"`
	ImplicitNeeds   []string `json:"implicit_needs"`
	Intent          string   `json:"intent"`
}

// RetrievalContext 检索上下文
type RetrievalContext struct {
	Strategy  string                 `json:"strategy"`
	TopK      int                    `json:"top_k"`
	Threshold float64                `json:"threshold"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Prompt 构建的Prompt
type Prompt struct {
	SystemPrompt string                 `json:"system_prompt"`
	Content      string                 `json:"content"`
	Format       string                 `json:"format"`
	Examples     []PromptExample        `json:"examples"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewPromptManager 创建Prompt管理器
func NewPromptManager(config *PromptConfig) *PromptManager {
	if config == nil {
		config = &PromptConfig{
			DefaultLanguage: "zh-CN",
			MaxTokens:       4096,
			Temperature:     0.7,
			CustomVars:      make(map[string]string),
		}
	}

	pm := &PromptManager{
		templates: make(map[string]*PromptTemplate),
		config:    config,
	}

	// 初始化默认模板
	pm.initializeDefaultTemplates()

	return pm
}

// initializeDefaultTemplates 初始化默认模板
func (pm *PromptManager) initializeDefaultTemplates() {
	// 三要素分析模板
	pm.RegisterTemplate(&PromptTemplate{
		Name: "three_elements_analysis",
		SystemPrompt: `你是一个专业的用户行为分析专家。你需要基于用户的历史对话和工作空间信息，分析出三个关键要素：

1. 用户要素（User Element）：
   - 技术栈：用户熟悉的编程语言、框架、工具
   - 经验水平：初级(junior)、中级(mid)、高级(senior)、专家(expert)
   - 角色定位：前端工程师、后端工程师、全栈工程师、架构师等
   - 领域专长：微服务、大数据、AI/ML、移动开发等

2. 情景要素（Situation Element）：
   - 项目类型：Web应用、移动应用、微服务、数据平台等
   - 技术环境：开发环境、生产环境、测试环境
   - 业务背景：电商、金融、教育、游戏等

3. 问题要素（Problem Element）：
   - 明确问题：用户直接表达的问题
   - 隐含需求：用户可能需要但没有明确表达的需求
   - 查询意图：学习概念、解决bug、性能优化、架构设计等

请严格按照JSON格式返回分析结果。`,
		UserTemplate: `请分析以下用户信息：

=== 历史对话 ===
{{range .SessionHistory}}
用户: {{.Content}}
{{end}}

{{if .WorkspaceContext}}
=== 工作空间信息 ===
项目类型: {{.WorkspaceContext.ProjectType}}
技术栈: {{join .WorkspaceContext.TechStack ", "}}
{{end}}

请基于以上信息进行三要素分析：`,
		Variables:    []string{"SessionHistory", "WorkspaceContext"},
		OutputFormat: "json",
		Version:      "v1.0.0",
		Description:  "分析用户、情景、问题三要素",
		Tags:         []string{"analysis", "user_profiling"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	// 查询改写模板
	pm.RegisterTemplate(&PromptTemplate{
		Name: "query_rewrite",
		SystemPrompt: `你是一个专业的查询优化专家。基于用户的技术背景、项目情境和问题意图，对查询进行智能改写和扩展。

改写原则：
1. 保持原意不变
2. 根据用户技术栈添加相关技术术语
3. 根据项目情境添加场景相关词汇
4. 避免添加无关噪声

返回JSON格式：
{
  "rewritten_query": "改写后的查询",
  "expansions": ["扩展词1", "扩展词2"],
  "quality_score": 0.85,
  "reasoning": ["改写理由1", "改写理由2"]
}`,
		UserTemplate: `原始查询: {{.OriginalQuery}}

{{if .ThreeElements}}
用户技术栈: {{join .ThreeElements.User.TechStack ", "}}
用户经验水平: {{.ThreeElements.User.ExperienceLevel}}
用户角色: {{.ThreeElements.User.Role}}
项目类型: {{.ThreeElements.Situation.ProjectType}}
业务背景: {{.ThreeElements.Situation.BusinessContext}}
查询意图: {{.ThreeElements.Problem.Intent}}
{{end}}

请基于以上信息对查询进行智能改写和扩展：`,
		Variables:    []string{"OriginalQuery", "ThreeElements"},
		OutputFormat: "json",
		Version:      "v1.0.0",
		Description:  "智能查询改写和扩展",
		Tags:         []string{"query_rewrite", "optimization"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
}

// RegisterTemplate 注册模板
func (pm *PromptManager) RegisterTemplate(template *PromptTemplate) {
	template.UpdatedAt = time.Now()
	pm.templates[template.Name] = template
}

// GetTemplate 获取模板
func (pm *PromptManager) GetTemplate(name string) (*PromptTemplate, bool) {
	template, exists := pm.templates[name]
	return template, exists
}

// BuildPrompt 构建Prompt
func (pm *PromptManager) BuildPrompt(templateName string, context *PromptContext) (*Prompt, error) {
	template, exists := pm.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// 构建用户Prompt
	userPrompt, err := pm.renderTemplate(template.UserTemplate, context)
	if err != nil {
		return nil, fmt.Errorf("render user template failed: %w", err)
	}

	return &Prompt{
		SystemPrompt: template.SystemPrompt,
		Content:      userPrompt,
		Format:       template.OutputFormat,
		Examples:     template.Examples,
		Metadata: map[string]interface{}{
			"template_name":    template.Name,
			"template_version": template.Version,
			"generated_at":     time.Now(),
		},
	}, nil
}

// renderTemplate 渲染模板
func (pm *PromptManager) renderTemplate(templateStr string, context *PromptContext) (string, error) {
	// 创建模板函数
	funcMap := template.FuncMap{
		"join": strings.Join,
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
	}

	// 解析模板
	tmpl, err := template.New("prompt").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	// 渲染模板
	var buf strings.Builder
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

	return buf.String(), nil
}

// ListTemplates 列出所有模板
func (pm *PromptManager) ListTemplates() []string {
	names := make([]string, 0, len(pm.templates))
	for name := range pm.templates {
		names = append(names, name)
	}
	return names
}

// Hash 计算Prompt哈希值（用于缓存）
func (p *Prompt) Hash() string {
	content := p.SystemPrompt + p.Content + p.Format
	// 简化的哈希实现
	hash := 0
	for _, c := range content {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("%x", hash)
}
