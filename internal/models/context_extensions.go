package models

import (
	"time"
)

// RecentChangesContext 最近变更上下文
type RecentChangesContext struct {
	// === 时间范围 ===
	TimeRange TimeRange `json:"time_range"` // 变更时间范围

	// === Git变更信息 ===
	RecentCommits  []CommitInfo     `json:"recent_commits"`  // 最近提交
	ModifiedFiles  []FileChangeInfo `json:"modified_files"`  // 修改文件
	BranchActivity []BranchActivity `json:"branch_activity"` // 分支活动

	// === 功能变更 ===
	NewFeatures    []FeatureChange `json:"new_features"`    // 新增功能
	FeatureUpdates []FeatureUpdate `json:"feature_updates"` // 功能更新
	BugFixes       []BugFixInfo    `json:"bug_fixes"`       // 问题修复

	// === 任务进度 ===
	CompletedTasks []TaskInfo `json:"completed_tasks"` // 完成任务
	OngoingTasks   []TaskInfo `json:"ongoing_tasks"`   // 进行中任务
	BlockedTasks   []TaskInfo `json:"blocked_tasks"`   // 阻塞任务

	// === 性能和质量变更 ===
	PerformanceChanges []PerformanceChange `json:"performance_changes"` // 性能变化
	QualityMetrics     []QualityMetric     `json:"quality_metrics"`     // 质量指标

	// === 配置和环境变更 ===
	ConfigChanges     []ConfigChange     `json:"config_changes"`     // 配置变更
	DependencyUpdates []DependencyUpdate `json:"dependency_updates"` // 依赖更新

	// === 元数据 ===
	LastUpdated      time.Time        `json:"last_updated"`      // 最后更新
	ChangesSummary   string           `json:"changes_summary"`   // 变更摘要
	ImpactAssessment ImpactAssessment `json:"impact_assessment"` // 影响评估
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"` // 开始时间
	EndTime   time.Time `json:"end_time"`   // 结束时间
}

// CommitInfo 提交信息
type CommitInfo struct {
	CommitHash    string       `json:"commit_hash"`    // 提交哈希
	Author        string       `json:"author"`         // 作者
	Timestamp     time.Time    `json:"timestamp"`      // 提交时间
	Message       string       `json:"message"`        // 提交消息
	FilesChanged  []string     `json:"files_changed"`  // 变更文件
	LinesAdded    int          `json:"lines_added"`    // 新增行数
	LinesDeleted  int          `json:"lines_deleted"`  // 删除行数
	CommitType    CommitType   `json:"commit_type"`    // 提交类型
	RelatedIssues []string     `json:"related_issues"` // 相关问题
	Impact        CommitImpact `json:"impact"`         // 提交影响
}

// CommitType 提交类型
type CommitType string

const (
	CommitTypeFeature  CommitType = "feature"
	CommitTypeBugfix   CommitType = "bugfix"
	CommitTypeRefactor CommitType = "refactor"
	CommitTypeDocs     CommitType = "docs"
	CommitTypeTest     CommitType = "test"
	CommitTypeChore    CommitType = "chore"
)

// CommitImpact 提交影响
type CommitImpact struct {
	Scope       string  `json:"scope"`       // 影响范围
	Severity    string  `json:"severity"`    // 严重程度
	Confidence  float64 `json:"confidence"`  // 置信度
	Description string  `json:"description"` // 影响描述
}

// FileChangeInfo 文件变更信息
type FileChangeInfo struct {
	FilePath          string         `json:"file_path"`          // 文件路径
	ChangeType        FileChangeType `json:"change_type"`        // 变更类型
	Language          string         `json:"language"`           // 编程语言
	LinesChanged      int            `json:"lines_changed"`      // 变更行数
	ChangeDescription string         `json:"change_description"` // 变更描述
	Importance        float64        `json:"importance"`         // 重要程度
	RelatedFeatures   []string       `json:"related_features"`   // 相关功能
	LastModified      time.Time      `json:"last_modified"`      // 最后修改
}

// FileChangeType 文件变更类型
type FileChangeType string

const (
	FileChangeTypeAdded    FileChangeType = "added"
	FileChangeTypeModified FileChangeType = "modified"
	FileChangeTypeDeleted  FileChangeType = "deleted"
	FileChangeTypeRenamed  FileChangeType = "renamed"
)

// BranchActivity 分支活动
type BranchActivity struct {
	BranchName   string    `json:"branch_name"`   // 分支名称
	ActivityType string    `json:"activity_type"` // 活动类型：create/merge/delete
	Timestamp    time.Time `json:"timestamp"`     // 时间戳
	Author       string    `json:"author"`        // 作者
	Description  string    `json:"description"`   // 描述
}

// FeatureChange 功能变更
type FeatureChange struct {
	FeatureName      string            `json:"feature_name"`      // 功能名称
	ChangeType       FeatureChangeType `json:"change_type"`       // 变更类型
	Description      string            `json:"description"`       // 变更描述
	RelatedCommits   []string          `json:"related_commits"`   // 相关提交
	RelatedFiles     []string          `json:"related_files"`     // 相关文件
	CompletionStatus float64           `json:"completion_status"` // 完成状态
	Priority         Priority          `json:"priority"`          // 优先级
	Timestamp        time.Time         `json:"timestamp"`         // 变更时间
}

// FeatureChangeType 功能变更类型
type FeatureChangeType string

const (
	FeatureChangeTypeAdded    FeatureChangeType = "added"
	FeatureChangeTypeModified FeatureChangeType = "modified"
	FeatureChangeTypeRemoved  FeatureChangeType = "removed"
	FeatureChangeTypeEnhanced FeatureChangeType = "enhanced"
)

// FeatureUpdate 功能更新
type FeatureUpdate struct {
	FeatureName   string    `json:"feature_name"`   // 功能名称
	UpdateType    string    `json:"update_type"`    // 更新类型
	Description   string    `json:"description"`    // 更新描述
	Version       string    `json:"version"`        // 版本
	Timestamp     time.Time `json:"timestamp"`      // 更新时间
	RelatedIssues []string  `json:"related_issues"` // 相关问题
}

// BugFixInfo 问题修复信息
type BugFixInfo struct {
	BugID          string    `json:"bug_id"`          // 问题ID
	Title          string    `json:"title"`           // 问题标题
	Description    string    `json:"description"`     // 问题描述
	Severity       string    `json:"severity"`        // 严重程度
	Status         string    `json:"status"`          // 状态
	FixedBy        string    `json:"fixed_by"`        // 修复者
	FixedAt        time.Time `json:"fixed_at"`        // 修复时间
	RelatedCommits []string  `json:"related_commits"` // 相关提交
	TestCases      []string  `json:"test_cases"`      // 测试用例
}

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID          string     `json:"task_id"`          // 任务ID
	TaskName        string     `json:"task_name"`        // 任务名称
	TaskType        TaskType   `json:"task_type"`        // 任务类型
	Description     string     `json:"description"`      // 任务描述
	Status          TaskStatus `json:"status"`           // 任务状态
	Priority        Priority   `json:"priority"`         // 优先级
	AssignedTo      string     `json:"assigned_to"`      // 分配给
	EstimatedHours  float64    `json:"estimated_hours"`  // 预估工时
	ActualHours     float64    `json:"actual_hours"`     // 实际工时
	CompletionRate  float64    `json:"completion_rate"`  // 完成率
	RelatedFeatures []string   `json:"related_features"` // 相关功能
	Dependencies    []string   `json:"dependencies"`     // 依赖任务
	CreatedAt       time.Time  `json:"created_at"`       // 创建时间
	UpdatedAt       time.Time  `json:"updated_at"`       // 更新时间
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeDevelopment   TaskType = "development"
	TaskTypeTesting       TaskType = "testing"
	TaskTypeDocumentation TaskType = "documentation"
	TaskTypeResearch      TaskType = "research"
	TaskTypeBugfix        TaskType = "bugfix"
	TaskTypeRefactor      TaskType = "refactor"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// PerformanceChange 性能变化
type PerformanceChange struct {
	MetricName    string    `json:"metric_name"`    // 指标名称
	OldValue      float64   `json:"old_value"`      // 旧值
	NewValue      float64   `json:"new_value"`      // 新值
	ChangePercent float64   `json:"change_percent"` // 变化百分比
	Unit          string    `json:"unit"`           // 单位
	Timestamp     time.Time `json:"timestamp"`      // 时间戳
	Context       string    `json:"context"`        // 上下文
}

// QualityMetric 质量指标
type QualityMetric struct {
	MetricName  string    `json:"metric_name"` // 指标名称
	Value       float64   `json:"value"`       // 值
	Threshold   float64   `json:"threshold"`   // 阈值
	Status      string    `json:"status"`      // 状态：pass/fail/warning
	Timestamp   time.Time `json:"timestamp"`   // 时间戳
	Description string    `json:"description"` // 描述
}

// ConfigChange 配置变更
type ConfigChange struct {
	ConfigFile string    `json:"config_file"` // 配置文件
	ChangeType string    `json:"change_type"` // 变更类型
	OldValue   string    `json:"old_value"`   // 旧值
	NewValue   string    `json:"new_value"`   // 新值
	Reason     string    `json:"reason"`      // 变更原因
	Timestamp  time.Time `json:"timestamp"`   // 时间戳
	ChangedBy  string    `json:"changed_by"`  // 变更者
}

// DependencyUpdate 依赖更新
type DependencyUpdate struct {
	DependencyName string    `json:"dependency_name"` // 依赖名称
	OldVersion     string    `json:"old_version"`     // 旧版本
	NewVersion     string    `json:"new_version"`     // 新版本
	UpdateType     string    `json:"update_type"`     // 更新类型：major/minor/patch
	Reason         string    `json:"reason"`          // 更新原因
	Timestamp      time.Time `json:"timestamp"`       // 时间戳
	SecurityFix    bool      `json:"security_fix"`    // 是否安全修复
	BreakingChange bool      `json:"breaking_change"` // 是否破坏性变更
}

// ImpactAssessment 影响评估
type ImpactAssessment struct {
	OverallImpact   string            `json:"overall_impact"`  // 总体影响
	AffectedAreas   []string          `json:"affected_areas"`  // 影响区域
	RiskLevel       string            `json:"risk_level"`      // 风险级别
	Recommendations []string          `json:"recommendations"` // 建议
	Confidence      float64           `json:"confidence"`      // 置信度
	Details         map[string]string `json:"details"`         // 详细信息
}

// CodeContext 代码上下文
type CodeContext struct {
	// === 会话关联 ===
	SessionID string `json:"session_id"` // 会话ID（新增）

	// === 当前活跃代码 ===
	ActiveFiles       []ActiveFileInfo  `json:"active_files"`       // 当前活跃文件
	RecentEdits       []ContextEditInfo `json:"recent_edits"`       // 最近编辑
	FocusedComponents []string          `json:"focused_components"` // 关注组件

	// === 代码结构信息 ===
	KeyFunctions       []FunctionInfo  `json:"key_functions"`       // 关键函数
	ImportantTypes     []TypeInfo      `json:"important_types"`     // 重要类型
	CriticalInterfaces []InterfaceInfo `json:"critical_interfaces"` // 关键接口

	// === 代码关系 ===
	DependencyGraph DependencyGraph  `json:"dependency_graph"` // 依赖图
	CallGraph       CallGraph        `json:"call_graph"`       // 调用图
	ModuleRelations []ModuleRelation `json:"module_relations"` // 模块关系

	// === 代码质量 ===
	QualityMetrics CodeQualityMetrics `json:"quality_metrics"` // 质量指标
	TestCoverage   TestCoverageInfo   `json:"test_coverage"`   // 测试覆盖
	CodeSmells     []CodeSmell        `json:"code_smells"`     // 代码异味

	// === 开发模式 ===
	DevelopmentPatterns   []DevelopmentPattern `json:"development_patterns"`   // 开发模式
	ArchitecturalPatterns []ArchPattern        `json:"architectural_patterns"` // 架构模式

	// === 元数据 ===
	LastAnalyzed    time.Time     `json:"last_analyzed"`    // 最后分析
	AnalysisDepth   AnalysisDepth `json:"analysis_depth"`   // 分析深度
	ConfidenceLevel float64       `json:"confidence_level"` // 置信度
}

// ActiveFileInfo 活跃文件信息
type ActiveFileInfo struct {
	FilePath        string    `json:"file_path"`        // 文件路径
	Language        string    `json:"language"`         // 编程语言
	FileType        FileType  `json:"file_type"`        // 文件类型
	LastAccessed    time.Time `json:"last_accessed"`    // 最后访问
	AccessFrequency int       `json:"access_frequency"` // 访问频率
	EditFrequency   int       `json:"edit_frequency"`   // 编辑频率
	Importance      float64   `json:"importance"`       // 重要程度

	// 文件内容摘要
	FileSummary  string   `json:"file_summary"`  // 文件摘要
	KeyFunctions []string `json:"key_functions"` // 关键函数
	MainPurpose  string   `json:"main_purpose"`  // 主要用途

	// 关联信息
	RelatedFiles []string `json:"related_files"` // 相关文件
	Dependencies []string `json:"dependencies"`  // 依赖文件
	Dependents   []string `json:"dependents"`    // 被依赖文件
}

// FileType 文件类型
type FileType string

const (
	FileTypeSource        FileType = "source"
	FileTypeTest          FileType = "test"
	FileTypeConfig        FileType = "config"
	FileTypeDocumentation FileType = "documentation"
	FileTypeScript        FileType = "script"
)

// ContextEditInfo 上下文编辑信息（避免与programming_context.go中的EditInfo冲突）
type ContextEditInfo struct {
	FilePath    string    `json:"file_path"`   // 文件路径
	EditType    string    `json:"edit_type"`   // 编辑类型
	LineStart   int       `json:"line_start"`  // 开始行
	LineEnd     int       `json:"line_end"`    // 结束行
	Content     string    `json:"content"`     // 编辑内容
	Timestamp   time.Time `json:"timestamp"`   // 编辑时间
	Author      string    `json:"author"`      // 编辑者
	Description string    `json:"description"` // 编辑描述
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	FunctionName string       `json:"function_name"` // 函数名称
	FilePath     string       `json:"file_path"`     // 所在文件
	FunctionType FunctionType `json:"function_type"` // 函数类型
	Signature    string       `json:"signature"`     // 函数签名
	Description  string       `json:"description"`   // 函数描述

	// 函数特征
	LineCount  int     `json:"line_count"` // 代码行数
	Complexity int     `json:"complexity"` // 复杂度
	CallCount  int     `json:"call_count"` // 调用次数
	Importance float64 `json:"importance"` // 重要程度

	// 关联信息
	CalledBy     []string `json:"called_by"`     // 被调用者
	Calls        []string `json:"calls"`         // 调用函数
	RelatedTypes []string `json:"related_types"` // 相关类型
	TestCoverage float64  `json:"test_coverage"` // 测试覆盖率

	// 元数据
	LastModified time.Time `json:"last_modified"` // 最后修改
	Author       string    `json:"author"`        // 作者
}

// FunctionType 函数类型
type FunctionType string

const (
	FunctionTypePublic      FunctionType = "public"
	FunctionTypePrivate     FunctionType = "private"
	FunctionTypeMethod      FunctionType = "method"
	FunctionTypeConstructor FunctionType = "constructor"
	FunctionTypeUtility     FunctionType = "utility"
)

// TypeInfo 类型信息
type TypeInfo struct {
	TypeName     string   `json:"type_name"`     // 类型名称
	TypeCategory string   `json:"type_category"` // 类型分类
	FilePath     string   `json:"file_path"`     // 所在文件
	Description  string   `json:"description"`   // 类型描述
	Fields       []string `json:"fields"`        // 字段列表
	Methods      []string `json:"methods"`       // 方法列表
	Interfaces   []string `json:"interfaces"`    // 实现的接口
	Importance   float64  `json:"importance"`    // 重要程度
}

// DependencyGraph 依赖图
type DependencyGraph struct {
	Nodes                []DependencyNode `json:"nodes"`                 // 依赖节点
	Edges                []DependencyEdge `json:"edges"`                 // 依赖边
	CriticalPaths        [][]string       `json:"critical_paths"`        // 关键路径
	CircularDependencies [][]string       `json:"circular_dependencies"` // 循环依赖
}

// DependencyNode 依赖节点
type DependencyNode struct {
	NodeID     string  `json:"node_id"`    // 节点ID
	NodeType   string  `json:"node_type"`  // 节点类型
	Name       string  `json:"name"`       // 名称
	FilePath   string  `json:"file_path"`  // 文件路径
	Importance float64 `json:"importance"` // 重要程度
}

// DependencyEdge 依赖边
type DependencyEdge struct {
	FromNode    string  `json:"from_node"`   // 源节点
	ToNode      string  `json:"to_node"`     // 目标节点
	EdgeType    string  `json:"edge_type"`   // 边类型
	Strength    float64 `json:"strength"`    // 依赖强度
	Description string  `json:"description"` // 描述
}

// CallGraph 调用图
type CallGraph struct {
	Functions []CallGraphNode `json:"functions"` // 函数节点
	Calls     []CallEdge      `json:"calls"`     // 调用边
}

// CallGraphNode 调用图节点
type CallGraphNode struct {
	FunctionName string  `json:"function_name"` // 函数名称
	FilePath     string  `json:"file_path"`     // 文件路径
	CallCount    int     `json:"call_count"`    // 调用次数
	Importance   float64 `json:"importance"`    // 重要程度
}

// CallEdge 调用边
type CallEdge struct {
	Caller    string `json:"caller"`     // 调用者
	Callee    string `json:"callee"`     // 被调用者
	CallCount int    `json:"call_count"` // 调用次数
}

// ModuleRelation 模块关系
type ModuleRelation struct {
	SourceModule string  `json:"source_module"` // 源模块
	TargetModule string  `json:"target_module"` // 目标模块
	RelationType string  `json:"relation_type"` // 关系类型
	Strength     float64 `json:"strength"`      // 关系强度
	Description  string  `json:"description"`   // 描述
}

// CodeQualityMetrics 代码质量指标
type CodeQualityMetrics struct {
	OverallScore    float64 `json:"overall_score"`   // 总体评分
	Maintainability float64 `json:"maintainability"` // 可维护性
	Readability     float64 `json:"readability"`     // 可读性
	Testability     float64 `json:"testability"`     // 可测试性
	Performance     float64 `json:"performance"`     // 性能指标
	Security        float64 `json:"security"`        // 安全性

	// 具体指标
	CyclomaticComplexity float64           `json:"cyclomatic_complexity"` // 圈复杂度
	TechnicalDebt        TechnicalDebtInfo `json:"technical_debt"`        // 技术债务
	CodeDuplication      float64           `json:"code_duplication"`      // 代码重复率
	TestCoverageRate     float64           `json:"test_coverage_rate"`    // 测试覆盖率
}

// TechnicalDebtInfo 技术债务信息
type TechnicalDebtInfo struct {
	TotalDebt       float64            `json:"total_debt"`      // 总债务
	DebtByType      map[string]float64 `json:"debt_by_type"`    // 按类型分类的债务
	DebtTrend       string             `json:"debt_trend"`      // 债务趋势
	Priority        Priority           `json:"priority"`        // 优先级
	Recommendations []string           `json:"recommendations"` // 建议
}

// TestCoverageInfo 测试覆盖信息
type TestCoverageInfo struct {
	OverallCoverage  float64            `json:"overall_coverage"`  // 总体覆盖率
	LineCoverage     float64            `json:"line_coverage"`     // 行覆盖率
	BranchCoverage   float64            `json:"branch_coverage"`   // 分支覆盖率
	FunctionCoverage float64            `json:"function_coverage"` // 函数覆盖率
	CoverageByFile   map[string]float64 `json:"coverage_by_file"`  // 按文件分类的覆盖率
	UncoveredLines   []UncoveredLine    `json:"uncovered_lines"`   // 未覆盖行
	TestFiles        []string           `json:"test_files"`        // 测试文件
}

// UncoveredLine 未覆盖行
type UncoveredLine struct {
	FilePath   string `json:"file_path"`   // 文件路径
	LineNumber int    `json:"line_number"` // 行号
	Content    string `json:"content"`     // 行内容
	Reason     string `json:"reason"`      // 未覆盖原因
}

// CodeSmell 代码异味
type CodeSmell struct {
	SmellType   string    `json:"smell_type"`  // 异味类型
	FilePath    string    `json:"file_path"`   // 文件路径
	LineStart   int       `json:"line_start"`  // 开始行
	LineEnd     int       `json:"line_end"`    // 结束行
	Description string    `json:"description"` // 描述
	Severity    string    `json:"severity"`    // 严重程度
	Suggestion  string    `json:"suggestion"`  // 建议
	DetectedAt  time.Time `json:"detected_at"` // 检测时间
}

// DevelopmentPattern 开发模式
type DevelopmentPattern struct {
	PatternName string   `json:"pattern_name"` // 模式名称
	PatternType string   `json:"pattern_type"` // 模式类型
	Description string   `json:"description"`  // 描述
	Examples    []string `json:"examples"`     // 示例
	Benefits    []string `json:"benefits"`     // 优点
	Drawbacks   []string `json:"drawbacks"`    // 缺点
	Usage       float64  `json:"usage"`        // 使用频率
}

// ArchPattern 架构模式
type ArchPattern struct {
	PatternName   string   `json:"pattern_name"`  // 模式名称
	Category      string   `json:"category"`      // 分类
	Description   string   `json:"description"`   // 描述
	Components    []string `json:"components"`    // 组件
	Relationships []string `json:"relationships"` // 关系
	Applicability string   `json:"applicability"` // 适用性
	Consequences  []string `json:"consequences"`  // 后果
}

// AnalysisDepth 分析深度
type AnalysisDepth string

const (
	AnalysisDepthBasic    AnalysisDepth = "basic"
	AnalysisDepthDetailed AnalysisDepth = "detailed"
	AnalysisDepthDeep     AnalysisDepth = "deep"
)
