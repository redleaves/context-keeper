package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/contextkeeper/service/internal/agentic_beta/config"
)

// OptimizedDecisionEngine 优化后的决策引擎
type OptimizedDecisionEngine struct {
	config *config.AgenticConfig
	rules  []DecisionRule
}

// NewOptimizedDecisionEngine 创建优化后的决策引擎
func NewOptimizedDecisionEngine(cfg *config.AgenticConfig) *OptimizedDecisionEngine {
	engine := &OptimizedDecisionEngine{
		config: cfg,
		rules:  make([]DecisionRule, 0),
	}

	// 注册优化后的规则（按优先级排序）
	engine.registerOptimizedRules()

	return engine
}

// registerOptimizedRules 注册优化后的规则
func (e *OptimizedDecisionEngine) registerOptimizedRules() {
	// 1. 状态确认规则 - 优先级100
	e.rules = append(e.rules, &StatusConfirmationRule{})

	// 2. 情感表达规则 - 优先级95
	e.rules = append(e.rules, &EmotionalFeedbackRule{})

	// 3. 系统元查询规则 - 优先级90
	e.rules = append(e.rules, &OptimizedMetaQueryRule{})

	// 4. 简单测试规则 - 优先级85
	e.rules = append(e.rules, &SimpleTestRule{})

	// 5. 重复请求规则 - 优先级80
	e.rules = append(e.rules, &RepeatRequestRule{})

	// 6. 超短查询规则 - 优先级75
	e.rules = append(e.rules, &UltraShortQueryRule{})

	// 7. 默认检索规则 - 优先级1（兜底）
	e.rules = append(e.rules, &DefaultRetrievalRule{})
}

// StatusConfirmationRule 状态确认规则
type StatusConfirmationRule struct{}

func (r *StatusConfirmationRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.TrimSpace(strings.ToLower(decisionCtx.Query))

	// 确认类词汇（高频无意义）
	confirmations := []string{
		// 中文确认
		"好的", "好", "ok", "行", "可以", "没问题", "明白", "了解", "知道了", "收到",
		// 英文确认
		"yes", "sure", "alright", "got it", "i see", "understood", "noted",
		// 简单回应
		"嗯", "哦", "啊", "是的", "对", "right", "yeah", "yep",
	}

	// 精确匹配或者非常相似的短语
	for _, confirmation := range confirmations {
		if query == confirmation ||
			strings.Contains(query, confirmation) && len(query) <= len(confirmation)+3 {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.95,
				Reason:         fmt.Sprintf("检测到状态确认: '%s'，无需检索", confirmation),
			}, nil
		}
	}

	return nil, nil
}

func (r *StatusConfirmationRule) Priority() int { return 100 }
func (r *StatusConfirmationRule) Name() string  { return "StatusConfirmationRule" }

// EmotionalFeedbackRule 情感表达规则
type EmotionalFeedbackRule struct{}

func (r *EmotionalFeedbackRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.TrimSpace(strings.ToLower(decisionCtx.Query))

	// 情感表达词汇
	emotions := []string{
		// 感谢类
		"谢谢", "谢了", "感谢", "thanks", "thank you", "thx",
		// 赞美类
		"很好", "不错", "棒", "赞", "牛", "厉害", "awesome", "great", "nice", "perfect", "excellent",
		// 惊叹类
		"哇", "wow", "amazing", "incredible",
		// 表情类
		"👍", "😊", "😄", "❤️", ":)", ":-)", "^_^",
	}

	for _, emotion := range emotions {
		if query == emotion ||
			(strings.Contains(query, emotion) && len(query) <= len(emotion)+5) {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.9,
				Reason:         fmt.Sprintf("检测到情感表达: '%s'，无需检索", emotion),
			}, nil
		}
	}

	return nil, nil
}

func (r *EmotionalFeedbackRule) Priority() int { return 95 }
func (r *EmotionalFeedbackRule) Name() string  { return "EmotionalFeedbackRule" }

// OptimizedMetaQueryRule 优化后的元查询规则
type OptimizedMetaQueryRule struct{}

func (r *OptimizedMetaQueryRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	// 精确的系统查询（不需要检索历史）
	metaQueries := []string{
		// 功能查询
		"你能做什么", "有什么功能", "能帮我做什么", "what can you do", "what are your capabilities",
		// 系统信息
		"你是谁", "你是什么", "什么是context-keeper", "who are you", "what is context-keeper",
		// 使用帮助
		"怎么用", "使用方法", "使用说明", "how to use", "how does it work", "help",
		// 状态查询
		"还在吗", "工作正常吗", "能听到吗", "are you there", "are you working", "status",
	}

	for _, metaQuery := range metaQueries {
		if strings.Contains(query, metaQuery) {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.88,
				Reason:         fmt.Sprintf("检测到系统元查询: '%s'，使用标准回复", metaQuery),
			}, nil
		}
	}

	return nil, nil
}

func (r *OptimizedMetaQueryRule) Priority() int { return 90 }
func (r *OptimizedMetaQueryRule) Name() string  { return "OptimizedMetaQueryRule" }

// SimpleTestRule 简单测试规则
type SimpleTestRule struct{}

func (r *SimpleTestRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.TrimSpace(strings.ToLower(decisionCtx.Query))

	// 明显的测试查询
	testPatterns := []string{
		"测试", "test", "试试", "试一下", "testing",
		"hello world", "world", "你好世界",
		"1", "2", "3", "a", "b", "c",
		"ping", "echo", "check",
	}

	for _, pattern := range testPatterns {
		if query == pattern {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.85,
				Reason:         fmt.Sprintf("检测到简单测试: '%s'，无需检索", pattern),
			}, nil
		}
	}

	return nil, nil
}

func (r *SimpleTestRule) Priority() int { return 85 }
func (r *SimpleTestRule) Name() string  { return "SimpleTestRule" }

// RepeatRequestRule 重复请求规则
type RepeatRequestRule struct{}

func (r *RepeatRequestRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.ToLower(decisionCtx.Query)

	// 要求重复的表达
	repeatKeywords := []string{
		"再说一遍", "重复一下", "没听清", "刚才说什么", "再来一次",
		"repeat", "say again", "what did you say", "pardon", "come again",
		"again", "once more",
	}

	for _, keyword := range repeatKeywords {
		if strings.Contains(query, keyword) {
			return &DecisionResult{
				ShouldRetrieve: false,
				Confidence:     0.8,
				Reason:         fmt.Sprintf("检测到重复请求: '%s'，返回上一次回复", keyword),
			}, nil
		}
	}

	return nil, nil
}

func (r *RepeatRequestRule) Priority() int { return 80 }
func (r *RepeatRequestRule) Name() string  { return "RepeatRequestRule" }

// UltraShortQueryRule 超短查询规则
type UltraShortQueryRule struct{}

func (r *UltraShortQueryRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	query := strings.TrimSpace(decisionCtx.Query)

	// 超短查询（1-2个字符）通常没有意义
	if len(query) <= 2 {
		return &DecisionResult{
			ShouldRetrieve: false,
			Confidence:     0.75,
			Reason:         "查询过短（≤2字符），可能是误触发",
		}, nil
	}

	// 3-4个字符的查询，检查是否为有意义的技术词汇
	if len(query) <= 4 {
		meaningfulShort := []string{
			"api", "bug", "git", "sql", "css", "js", "go", "py",
			"docker", "k8s", "aws", "gcp", "tcp", "http", "rest",
		}

		queryLower := strings.ToLower(query)
		for _, meaningful := range meaningfulShort {
			if queryLower == meaningful {
				// 这是有意义的技术词汇，需要检索
				return nil, nil
			}
		}

		return &DecisionResult{
			ShouldRetrieve: false,
			Confidence:     0.7,
			Reason:         "查询过短（≤4字符）且非技术词汇",
		}, nil
	}

	return nil, nil
}

func (r *UltraShortQueryRule) Priority() int { return 75 }
func (r *UltraShortQueryRule) Name() string  { return "UltraShortQueryRule" }

// DefaultRetrievalRule 默认检索规则（兜底）
type DefaultRetrievalRule struct{}

func (r *DefaultRetrievalRule) Evaluate(ctx context.Context, decisionCtx *DecisionContext) (*DecisionResult, error) {
	// 所有其他查询都需要检索
	return &DecisionResult{
		ShouldRetrieve: true,
		Confidence:     0.6,
		Reason:         "默认策略：进行上下文检索",
	}, nil
}

func (r *DefaultRetrievalRule) Priority() int { return 1 }
func (r *DefaultRetrievalRule) Name() string  { return "DefaultRetrievalRule" }

// Decide 执行决策（与原来的接口保持一致）
func (e *OptimizedDecisionEngine) Decide(ctx context.Context, query string, contextData map[string]interface{}) (*DecisionResult, error) {
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
func (e *OptimizedDecisionEngine) UpdateConfig(config *config.AgenticConfig) error {
	e.config = config
	return nil
}
