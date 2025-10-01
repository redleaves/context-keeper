package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/llm"
	"github.com/contextkeeper/service/internal/models"
)

// TestRealDeepSeekAPI 真实测试DeepSeek API的能力
func TestRealDeepSeekAPI(t *testing.T) {
	// 跳过测试，除非明确要求运行真实API测试
	if os.Getenv("RUN_REAL_API_TEST") != "true" {
		t.Skip("跳过真实API测试，设置 RUN_REAL_API_TEST=true 来运行")
	}

	log.Printf("🚀 [真实API测试] 开始测试真实的DeepSeek API")

	// 从环境变量获取API密钥
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = "sk-31206448be1f4e6980ca7450cc8a21cb" // 从配置文件中的密钥
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 测试DeepSeek-V3模型
	t.Run("DeepSeek-V3模型测试", func(t *testing.T) {
		testDeepSeekModel(t, ctx, apiKey, "deepseek-chat", "DeepSeek-V3")
	})

	// 测试DeepSeek-R1模型
	t.Run("DeepSeek-R1模型测试", func(t *testing.T) {
		testDeepSeekModel(t, ctx, apiKey, "deepseek-reasoner", "DeepSeek-R1")
	})
}

func testDeepSeekModel(t *testing.T, ctx context.Context, apiKey, model, modelName string) {
	log.Printf("🤖 [%s测试] 开始测试 %s 模型", modelName, model)

	// 创建LLM配置
	config := &llm.LLMConfig{
		Provider:   llm.DeepSeek,
		APIKey:     apiKey,
		Model:      model,
		MaxRetries: 3,
		Timeout:    120 * time.Second,
		RateLimit:  60,
	}

	// 创建DeepSeek客户端
	client, err := llm.NewLLMClient(config)
	if err != nil {
		t.Fatalf("创建%s客户端失败: %v", modelName, err)
	}

	// 构建复杂的UnifiedContextModel生成prompt
	prompt := buildComplexContextPrompt()

	log.Printf("📤 [%s测试] 发送请求详情:", modelName)
	log.Printf("   🔗 API端点: DeepSeek API")
	log.Printf("   🤖 模型: %s", model)
	log.Printf("   📝 Prompt长度: %d字符", len(prompt))
	log.Printf("   🎯 目标: 生成完整的UnifiedContextModel JSON (138个字段)")
	log.Printf("   ⚙️  参数: MaxTokens=8000, Temperature=0.1")

	// 显示prompt内容的前1000字符
	promptPreview := prompt
	if len(promptPreview) > 1000 {
		promptPreview = promptPreview[:1000] + "..."
	}
	log.Printf("📄 [%s测试] Prompt内容预览:\n%s", modelName, promptPreview)

	// 构建请求
	request := &llm.LLMRequest{
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   8000,
		Temperature: 0.1,
		TopP:        0.9,
		Stream:      false,
	}

	startTime := time.Now()

	// 调用真实的DeepSeek API
	log.Printf("⏳ [%s测试] 正在调用真实的DeepSeek API...", modelName)
	response, err := client.GenerateResponse(ctx, request)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ [%s测试] API调用失败: %v", modelName, err)
		log.Printf("   ⏱️  耗时: %v", duration)
		t.Errorf("%s API调用失败: %v", modelName, err)
		return
	}

	log.Printf("📥 [%s测试] 收到API响应:", modelName)
	log.Printf("   ✅ 调用成功")
	log.Printf("   📊 响应长度: %d字符", len(response.Content))
	log.Printf("   🔢 Token使用: Prompt=%d, Completion=%d, Total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	log.Printf("   ⏱️  总耗时: %v", duration)
	log.Printf("   🚀 生成速度: %.1f tokens/秒",
		float64(response.Usage.CompletionTokens)/duration.Seconds())

	// 显示完整的响应内容
	log.Printf("📄 [%s测试] 完整响应内容:", modelName)
	log.Printf("=== 响应开始 ===")
	log.Printf("%s", response.Content)
	log.Printf("=== 响应结束 ===")

	// 尝试解析为JSON
	log.Printf("🔍 [%s测试] 开始JSON解析验证", modelName)

	// 清理响应内容
	cleanContent := strings.TrimSpace(response.Content)
	if strings.HasPrefix(cleanContent, "```json") {
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
	}
	if strings.HasSuffix(cleanContent, "```") {
		cleanContent = strings.TrimSuffix(cleanContent, "```")
	}
	cleanContent = strings.TrimSpace(cleanContent)

	log.Printf("🧹 [%s测试] 清理后内容长度: %d字符", modelName, len(cleanContent))

	// 首先验证是否为有效JSON
	var genericJSON map[string]interface{}
	if err := json.Unmarshal([]byte(cleanContent), &genericJSON); err != nil {
		log.Printf("❌ [%s测试] JSON格式无效: %v", modelName, err)
		log.Printf("🔍 [%s测试] 响应不是有效的JSON格式", modelName)
		t.Errorf("%s生成的响应不是有效JSON: %v", modelName, err)
		return
	}

	log.Printf("✅ [%s测试] JSON格式有效", modelName)
	log.Printf("📊 [%s测试] JSON结构分析:", modelName)
	log.Printf("   🔑 顶层字段数: %d", len(genericJSON))
	log.Printf("   🏷️  顶层字段: %v", getMapKeys(genericJSON))

	// 尝试解析为UnifiedContextModel
	var unifiedContext models.UnifiedContextModel
	if err := json.Unmarshal([]byte(cleanContent), &unifiedContext); err != nil {
		log.Printf("❌ [%s测试] 无法解析为UnifiedContextModel: %v", modelName, err)
		log.Printf("🔍 [%s测试] 结构不匹配UnifiedContextModel", modelName)

		// 分析具体哪些字段不匹配
		analyzeJSONStructure(genericJSON, modelName)
		t.Errorf("%s生成的JSON无法解析为UnifiedContextModel: %v", modelName, err)
		return
	}

	log.Printf("🎉 [%s测试] 成功解析为UnifiedContextModel!", modelName)

	// 详细分析生成的字段
	analyzeUnifiedContextModel(&unifiedContext, modelName)

	// 计算字段完整性
	completeness := calculateFieldCompleteness(&unifiedContext)
	log.Printf("📊 [%s测试] 字段完整性: %.1f%% (%d/138)", modelName, completeness, int(completeness*138/100))

	// 最终结论
	log.Printf("🎯 [%s测试] 最终结论:", modelName)
	if completeness > 80 {
		log.Printf("   ✅ %s能够生成高质量的UnifiedContextModel", modelName)
		log.Printf("   ✅ 字段完整性优秀: %.1f%%", completeness)
		log.Printf("   ⚠️  生成时间: %v", duration)
	} else if completeness > 50 {
		log.Printf("   ⚠️  %s能够生成基本的UnifiedContextModel", modelName)
		log.Printf("   ⚠️  字段完整性中等: %.1f%%", completeness)
		log.Printf("   📋 建议: 需要优化prompt或简化结构")
	} else {
		log.Printf("   ❌ %s无法生成完整的UnifiedContextModel", modelName)
		log.Printf("   ❌ 字段完整性较低: %.1f%%", completeness)
		log.Printf("   📋 建议: 需要拆解或简化结构")
	}
}

// buildComplexContextPrompt 构建复杂的上下文生成prompt
func buildComplexContextPrompt() string {
	return `你是一个专业的上下文建模专家。请根据以下用户查询生成一个完整的UnifiedContextModel JSON结构。

用户查询：我需要设计一个高并发的微服务架构，包括缓存层、数据库分片、消息队列和监控系统，请帮我分析技术选型和架构设计。

请生成一个完整的UnifiedContextModel JSON，必须包含以下所有字段：

1. 基础字段：
   - session_id: 会话ID
   - user_id: 用户ID  
   - workspace_id: 工作空间ID
   - created_at: 创建时间 (ISO 8601格式)
   - updated_at: 更新时间 (ISO 8601格式)

2. current_topic (TopicContext对象)：
   - main_topic: 主要话题
   - topic_category: 话题分类 ("technical"|"business"|"learning"|"troubleshooting")
   - user_intent: 用户意图对象
   - primary_pain_point: 主要痛点
   - secondary_pain_points: 次要痛点数组
   - expected_outcome: 期望结果
   - key_concepts: 关键概念数组
   - technical_terms: 技术术语数组
   - business_terms: 业务术语数组
   - topic_evolution: 话题演进数组
   - related_topics: 相关话题数组
   - topic_start_time: 话题开始时间
   - last_updated: 最后更新时间
   - update_count: 更新次数
   - confidence_level: 置信度

3. project (ProjectContext对象)：
   - project_name: 项目名称
   - project_path: 项目路径
   - project_type: 项目类型
   - description: 项目描述
   - primary_language: 主要编程语言
   - current_phase: 当前阶段
   - confidence_level: 置信度

4. code (CodeContext对象)：
   - session_id: 会话ID
   - active_files: 活跃文件数组
   - recent_edits: 最近编辑数组
   - focused_components: 关注组件数组
   - key_functions: 关键函数数组
   - important_types: 重要类型数组

5. recent_changes (RecentChangesContext对象)：
   - time_range: 时间范围对象
   - recent_commits: 最近提交数组
   - modified_files: 修改文件数组
   - branch_activity: 分支活动数组
   - new_features: 新功能数组
   - feature_updates: 功能更新数组
   - bug_fixes: 错误修复数组
   - completed_tasks: 已完成任务数组
   - ongoing_tasks: 进行中任务数组
   - blocked_tasks: 阻塞任务数组

请确保：
- 所有字段都有合理的值，不要使用null
- 时间字段使用ISO 8601格式
- 数组字段至少包含1-3个元素
- 嵌套对象完整
- JSON格式严格正确
- 内容与微服务架构设计相关

直接输出JSON，不要包含任何解释文字：`
}

// getMapKeys 获取map的所有键
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// analyzeJSONStructure 分析JSON结构
func analyzeJSONStructure(data map[string]interface{}, modelName string) {
	log.Printf("🔍 [%s测试] JSON结构详细分析:", modelName)
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			log.Printf("   📁 %s: 对象 (包含%d个字段)", key, len(v))
		case []interface{}:
			log.Printf("   📋 %s: 数组 (包含%d个元素)", key, len(v))
		case string:
			log.Printf("   📝 %s: 字符串 (%d字符)", key, len(v))
		case float64:
			log.Printf("   🔢 %s: 数字 (%.3f)", key, v)
		case bool:
			log.Printf("   ✅ %s: 布尔值 (%t)", key, v)
		default:
			log.Printf("   ❓ %s: 其他类型 (%T)", key, v)
		}
	}
}

// analyzeUnifiedContextModel 分析UnifiedContextModel
func analyzeUnifiedContextModel(ctx *models.UnifiedContextModel, modelName string) {
	log.Printf("🎯 [%s测试] UnifiedContextModel详细分析:", modelName)
	log.Printf("   📋 基础信息:")
	log.Printf("      SessionID: %s", ctx.SessionID)
	log.Printf("      UserID: %s", ctx.UserID)
	log.Printf("      WorkspaceID: %s", ctx.WorkspaceID)

	if ctx.CurrentTopic != nil {
		log.Printf("   📋 主题上下文: ✅")
		log.Printf("      主题: %s", ctx.CurrentTopic.MainTopic)
		log.Printf("      痛点: %s", ctx.CurrentTopic.PrimaryPainPoint)
		log.Printf("      关键概念数: %d", len(ctx.CurrentTopic.KeyConcepts))
	} else {
		log.Printf("   📋 主题上下文: ❌ nil")
	}

	if ctx.Project != nil {
		log.Printf("   📋 项目上下文: ✅")
		log.Printf("      项目名: %s", ctx.Project.ProjectName)
		log.Printf("      项目类型: %s", string(ctx.Project.ProjectType))
	} else {
		log.Printf("   📋 项目上下文: ❌ nil")
	}

	if ctx.Code != nil {
		log.Printf("   📋 代码上下文: ✅")
		log.Printf("      活跃文件数: %d", len(ctx.Code.ActiveFiles))
	} else {
		log.Printf("   📋 代码上下文: ❌ nil")
	}

	if ctx.RecentChanges != nil {
		log.Printf("   📋 变更上下文: ✅")
		log.Printf("      最近提交数: %d", len(ctx.RecentChanges.RecentCommits))
	} else {
		log.Printf("   📋 变更上下文: ❌ nil")
	}
}

// calculateFieldCompleteness 计算字段完整性
func calculateFieldCompleteness(ctx *models.UnifiedContextModel) float64 {
	totalFields := 138.0 // 预估的总字段数
	completedFields := 0.0

	// 基础字段 (5个)
	if ctx.SessionID != "" {
		completedFields++
	}
	if ctx.UserID != "" {
		completedFields++
	}
	if ctx.WorkspaceID != "" {
		completedFields++
	}
	if !ctx.CreatedAt.IsZero() {
		completedFields++
	}
	if !ctx.UpdatedAt.IsZero() {
		completedFields++
	}

	// 主题上下文 (约50个字段)
	if ctx.CurrentTopic != nil {
		completedFields += 50
	}

	// 项目上下文 (约20个字段)
	if ctx.Project != nil {
		completedFields += 20
	}

	// 代码上下文 (约30个字段)
	if ctx.Code != nil {
		completedFields += 30
	}

	// 变更上下文 (约30个字段)
	if ctx.RecentChanges != nil {
		completedFields += 30
	}

	// 会话上下文 (约3个字段)
	if ctx.Conversation != nil {
		completedFields += 3
	}

	return (completedFields / totalFields) * 100
}
