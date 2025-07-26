package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/agentic/config"
)

// DecisionResult 决策结果
type DecisionResult struct {
	ShouldRetrieve bool                   `json:"should_retrieve"`
	Confidence     float64                `json:"confidence"`
	Reason         string                 `json:"reason"`
	SuggestedQuery string                 `json:"suggested_query,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// DecisionContext 决策上下文
type DecisionContext struct {
	Query               string                 `json:"query"`
	SessionID           string                 `json:"session_id"`
	UserID              string                 `json:"user_id"`
	ConversationHistory []string               `json:"conversation_history,omitempty"`
	RecentQueries       []string               `json:"recent_queries,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// DecisionEngine 决策引擎接口
type DecisionEngine interface {
	Decide(ctx context.Context, query string, context map[string]interface{}) (*DecisionResult, error)
	UpdateConfig(config *config.AgenticConfig) error
}

// RuleBasedDecisionEngine 基于规则的决策引擎
type RuleBasedDecisionEngine struct {
	config *config.AgenticConfig
	rules  []DecisionRule
}

// DecisionRule 决策规则接口
type DecisionRule interface {
	Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error)
	Priority() int
	Name() string
}

// NewRuleBasedDecisionEngine 创建基于规则的决策引擎
func NewRuleBasedDecisionEngine(cfg *config.AgenticConfig) *RuleBasedDecisionEngine {
	engine := &RuleBasedDecisionEngine{
		config: cfg,
		rules:  make([]DecisionRule, 0),
	}

	// 注册默认规则
	engine.registerDefaultRules()

	return engine
}

// registerDefaultRules 注册默认决策规则
func (e *RuleBasedDecisionEngine) registerDefaultRules() {
	// 1. 简单问候和状态查询 - 无需检索
	e.rules = append(e.rules, &GreetingRule{})

	// 2. 元问题和系统问题 - 无需检索
	e.rules = append(e.rules, &MetaQuestionRule{})

	// 3. 重复问题检测 - 可能无需检索
	e.rules = append(e.rules, &RepetitiveQueryRule{})

	// 4. 编程相关问题 - 需要检索
	e.rules = append(e.rules, &ProgrammingQueryRule{})

	// 5. 知识查询 - 需要检索
	e.rules = append(e.rules, &KnowledgeQueryRule{})

	// 6. 默认规则 - 兜底
	e.rules = append(e.rules, &DefaultRule{})
}

// Decide 执行决策
func (e *RuleBasedDecisionEngine) Decide(ctx context.Context, query string, contextData map[string]interface{}) (*DecisionResult, error) {
	startTime := time.Now()

	// 构建决策上下文
	decisionCtx := &DecisionContext{
		Query:    strings.TrimSpace(query),
		Metadata: contextData,
	}

	// 从contextData中提取信息
	if sessionID, ok := contextData["sessionId"].(string); ok {
		decisionCtx.SessionID = sessionID
	}
	if userID, ok := contextData["userId"].(string); ok {
		decisionCtx.UserID = userID
	}

	// 按优先级评估规则
	for _, rule := range e.rules {
		result, err := rule.Evaluate(ctx, decisionCtx)
		if err != nil {
			continue // 跳过失败的规则
		}

		// 如果规则给出了明确决策，就使用这个结果
		if result != nil && result.Confidence > e.config.RetrievalDecision.ConfidenceThreshold {
			result.ProcessingTime = time.Since(startTime)
			result.Metadata = map[string]interface{}{
				"rule_name":     rule.Name(),
				"rule_priority": rule.Priority(),
			}
			return result, nil
		}
	}

	// 如果没有规则给出明确决策，返回默认结果
	return &DecisionResult{
		ShouldRetrieve: true, // 默认检索
		Confidence:     0.5,
		Reason:         "没有匹配的规则，使用默认策略",
		ProcessingTime: time.Since(startTime),
	}, nil
}

// UpdateConfig 更新配置
func (e *RuleBasedDecisionEngine) UpdateConfig(config *config.AgenticConfig) error {
	e.config = config
	return nil
}

// GreetingRule 问候规则
type GreetingRule struct{}

func (r *GreetingRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	greetings := []string{
		"你好", "hello", "hi", "hey", "早上好", "下午好", "晚上好",
		"how are you", "谢谢", "thank you", "thanks", "ok", "好的",
		"再见", "bye", "goodbye", "see you",
	}

	for _, greeting := range greetings {
		if strings.Contains(query, greeting) {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.9,
				Reason:         "检测到问候语，无需检索上下文",
			}, nil
		}
	}

	// 简单的是非问题
	if len(query) < 10 && (strings.Contains(query, "是") || strings.Contains(query, "不是") ||
		strings.Contains(query, "对") || strings.Contains(query, "错")) {
		return &DecisionResult{
			ShouldRetrieve: false,
			Confidence:     0.8,
			Reason:         "检测到简单回应，无需检索上下文",
		}, nil
	}

	return nil, nil // 不匹配此规则
}

func (r *GreetingRule) Priority() int { return 100 }
func (r *GreetingRule) Name() string  { return "GreetingRule" }

// MetaQuestionRule 元问题规则
type MetaQuestionRule struct{}

func (r *MetaQuestionRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	metaKeywords := []string{
		"你是谁", "你叫什么", "什么是context", "你能做什么", "帮助",
		"how to use", "what is", "who are you", "what can you do",
		"使用方法", "功能介绍", "系统介绍",
	}

	for _, keyword := range metaKeywords {
		if strings.Contains(query, keyword) {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.85,
				Reason:         "检测到系统元问题，使用内置回答",
			}, nil
		}
	}

	return nil, nil
}

func (r *MetaQuestionRule) Priority() int { return 90 }
func (r *MetaQuestionRule) Name() string  { return "MetaQuestionRule" }

// ProgrammingQueryRule 编程查询规则
type ProgrammingQueryRule struct{}

func (r *ProgrammingQueryRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	programmingKeywords := []string{
		"代码", "函数", "方法", "类", "接口", "变量", "算法", "数据结构",
		"debug", "bug", "错误", "异常", "编译", "运行", "测试",
		"code", "function", "method", "class", "interface", "variable",
		"api", "sdk", "framework", "library", "package", "module",
		"go", "python", "java", "javascript", "react", "vue", "node",
		"数据库", "sql", "mysql", "redis", "mongodb",
		"docker", "kubernetes", "git", "github",
	}

	for _, keyword := range programmingKeywords {
		if strings.Contains(query, keyword) {
			return &DecisionResult{
				ShouldRetrieve: true,
				Confidence:     0.9,
				Reason:         fmt.Sprintf("检测到编程相关关键词: %s", keyword),
				SuggestedQuery: decisionCtx.Query, // 保持原查询
			}, nil
		}
	}

	// 检测代码片段
	if strings.Contains(query, "{") || strings.Contains(query, "}") ||
		strings.Contains(query, "()") || strings.Contains(query, "[]") {
		return &DecisionResult{
			ShouldRetrieve: true,
			Confidence:     0.8,
			Reason:         "检测到代码片段，需要上下文支持",
		}, nil
	}

	return nil, nil
}

func (r *ProgrammingQueryRule) Priority() int { return 80 }
func (r *ProgrammingQueryRule) Name() string  { return "ProgrammingQueryRule" }

// RepetitiveQueryRule 重复查询规则
type RepetitiveQueryRule struct{}

func (r *RepetitiveQueryRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	// 这里可以集成session历史查询，检测重复性
	// 暂时返回nil，表示不匹配此规则
	return nil, nil
}

func (r *RepetitiveQueryRule) Priority() int { return 70 }
func (r *RepetitiveQueryRule) Name() string  { return "RepetitiveQueryRule" }

// KnowledgeQueryRule 知识查询规则
type KnowledgeQueryRule struct{}

func (r *KnowledgeQueryRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	// 疑问词
	questionWords := []string{
		"什么", "怎么", "如何", "为什么", "哪里", "什么时候", "谁",
		"what", "how", "why", "where", "when", "who", "which",
		"是否", "能否", "可以", "应该",
	}

	for _, word := range questionWords {
		if strings.Contains(query, word) {
			return &DecisionResult{
				ShouldRetrieve: true,
				Confidence:     0.75,
				Reason:         fmt.Sprintf("检测到疑问词: %s，可能需要知识支持", word),
			}, nil
		}
	}

	// 长查询通常需要上下文
	if len(decisionCtx.Query) > 20 {
		return &DecisionResult{
			ShouldRetrieve: true,
			Confidence:     0.7,
			Reason:         "查询较长，可能需要上下文支持",
		}, nil
	}

	return nil, nil
}

func (r *KnowledgeQueryRule) Priority() int { return 60 }
func (r *KnowledgeQueryRule) Name() string  { return "KnowledgeQueryRule" }

// DefaultRule 默认规则
type DefaultRule struct{}

func (r *DefaultRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	// 默认策略：短查询不检索，长查询检索
	if len(decisionCtx.Query) < 5 {
		return &DecisionResult{
			ShouldRetrieve: false,
			Confidence:     0.6,
			Reason:         "查询过短，使用直接回答",
		}, nil
	}

	return &DecisionResult{
		ShouldRetrieve: true,
		Confidence:     0.6,
		Reason:         "默认策略：进行上下文检索",
	}, nil
}

func (r *DefaultRule) Priority() int { return 1 }
func (r *DefaultRule) Name() string  { return "DefaultRule" }
