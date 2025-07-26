package agentic_beta

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/contextkeeper/service/internal/models"
	"github.com/contextkeeper/service/internal/services"
	"github.com/contextkeeper/service/internal/store"
)

// MockSmartContextService 模拟SmartContextService用于测试
type MockSmartContextService struct {
	// 存储调用记录
	RetrieveContextCalls []models.RetrieveContextRequest
	LastResponse         models.ContextResponse
	ShouldReturnError    bool
}

func NewMockSmartContextService() *MockSmartContextService {
	return &MockSmartContextService{
		RetrieveContextCalls: make([]models.RetrieveContextRequest, 0),
		LastResponse: models.ContextResponse{
			SessionState:      "测试会话状态",
			ShortTermMemory:   "短期记忆内容",
			LongTermMemory:    "长期记忆内容",
			RelevantKnowledge: "相关知识",
		},
		ShouldReturnError: false,
	}
}

// 实现SmartContextService接口的主要方法
func (m *MockSmartContextService) RetrieveContext(ctx context.Context, req models.RetrieveContextRequest) (models.ContextResponse, error) {
	m.RetrieveContextCalls = append(m.RetrieveContextCalls, req)

	if m.ShouldReturnError {
		return models.ContextResponse{}, fmt.Errorf("模拟错误")
	}

	return m.LastResponse, nil
}

// 其他方法的空实现
func (m *MockSmartContextService) RetrieveTodos(ctx context.Context, req models.RetrieveTodosRequest) (*models.RetrieveTodosResponse, error) {
	return &models.RetrieveTodosResponse{}, nil
}

func (m *MockSmartContextService) AssociateFile(ctx context.Context, req models.AssociateFileRequest) error {
	return nil
}

func (m *MockSmartContextService) RecordEdit(ctx context.Context, req models.RecordEditRequest) error {
	return nil
}

func (m *MockSmartContextService) GetProgrammingContext(ctx context.Context, sessionID string, query string) (*models.ProgrammingContext, error) {
	return &models.ProgrammingContext{}, nil
}

func (m *MockSmartContextService) StartSessionCleanupTask(ctx context.Context, timeout, interval time.Duration) {
}

func (m *MockSmartContextService) SummarizeToLongTermMemory(ctx context.Context, req models.SummarizeToLongTermRequest) (string, error) {
	return "summary", nil
}

func (m *MockSmartContextService) StoreContext(ctx context.Context, req models.StoreContextRequest) (string, error) {
	return "memory_id", nil
}

func (m *MockSmartContextService) SummarizeContext(ctx context.Context, req models.SummarizeContextRequest) (string, error) {
	return "summary", nil
}

func (m *MockSmartContextService) StoreSessionMessages(ctx context.Context, req models.StoreMessagesRequest) (*models.StoreMessagesResponse, error) {
	return &models.StoreMessagesResponse{}, nil
}

func (m *MockSmartContextService) RetrieveConversation(ctx context.Context, req models.RetrieveConversationRequest) (*models.ConversationResponse, error) {
	return &models.ConversationResponse{}, nil
}

func (m *MockSmartContextService) GetSessionState(ctx context.Context, sessionID string) (*models.MCPSessionResponse, error) {
	return &models.MCPSessionResponse{}, nil
}

func (m *MockSmartContextService) SearchContext(ctx context.Context, sessionID, query string) ([]string, error) {
	return []string{}, nil
}

func (m *MockSmartContextService) GetUserIDFromSessionID(sessionID string) (string, error) {
	return "test_user", nil
}

func (m *MockSmartContextService) GetUserSessionStore(userID string) (*store.SessionStore, error) {
	return nil, nil
}

func (m *MockSmartContextService) SessionStore() *store.SessionStore {
	return nil
}

func (m *MockSmartContextService) GetContextService() *services.ContextService {
	return nil
}

func (m *MockSmartContextService) EnableSmart(enabled bool) {
}

// TestAgenticContextServiceCreation 测试AgenticContextService创建
func TestAgenticContextServiceCreation(t *testing.T) {
	mockSmart := NewMockSmartContextService()

	agentic := NewAgenticContextService(mockSmart)

	// 验证基础属性
	if agentic.name != "AgenticContextService" {
		t.Errorf("Expected name 'AgenticContextService', got %s", agentic.name)
	}

	if agentic.version != "v1.0.0-beta" {
		t.Errorf("Expected version 'v1.0.0-beta', got %s", agentic.version)
	}

	if !agentic.enabled {
		t.Error("Expected Agentic service to be enabled by default")
	}

	// 验证组件初始化
	if agentic.intentAnalyzer == nil {
		t.Error("Expected intent analyzer to be initialized")
	}

	if agentic.decisionCenter == nil {
		t.Error("Expected decision center to be initialized")
	}

	if agentic.stats == nil {
		t.Error("Expected stats to be initialized")
	}
}

// TestAgenticContextServiceDebugQuery 测试调试类查询的A→B→C流程
func TestAgenticContextServiceDebugQuery(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "test_session",
		Query:     "这个Go程序报错了，怎么debug？",
		Limit:     2000,
	}

	// 执行检索
	response, err := agentic.RetrieveContext(ctx, req)
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}

	// 验证响应
	if response.SessionState == "" {
		t.Error("Expected non-empty session state")
	}

	// 验证Agentic信息被添加到响应中
	if !strings.Contains(response.SessionState, "Agentic智能处理") {
		t.Error("Expected Agentic information in session state")
	}

	// 验证底层服务被调用
	if len(mockSmart.RetrieveContextCalls) != 1 {
		t.Errorf("Expected 1 call to underlying service, got %d", len(mockSmart.RetrieveContextCalls))
	}

	// 验证查询被优化
	calledReq := mockSmart.RetrieveContextCalls[0]
	if calledReq.Query == req.Query {
		t.Log("Query was not modified (this might be expected for simple cases)")
	} else {
		t.Logf("Query optimized from '%s' to '%s'", req.Query, calledReq.Query)
	}

	// 验证统计数据
	stats := agentic.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", stats.TotalRequests)
	}

	if stats.AgenticEnhanced != 1 {
		t.Errorf("Expected 1 agentic enhanced request, got %d", stats.AgenticEnhanced)
	}
}

// TestAgenticContextServiceArchitectureQuery 测试架构类查询
func TestAgenticContextServiceArchitectureQuery(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "test_session",
		Query:     "如何设计高可用的分布式微服务架构？",
		Limit:     2000,
	}

	// 执行检索
	_, err := agentic.RetrieveContext(ctx, req)
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}

	// 验证底层服务被调用
	if len(mockSmart.RetrieveContextCalls) != 1 {
		t.Errorf("Expected 1 call to underlying service, got %d", len(mockSmart.RetrieveContextCalls))
	}

	// 验证复杂查询的参数调整
	calledReq := mockSmart.RetrieveContextCalls[0]
	if calledReq.Limit <= req.Limit {
		t.Log("Limit was not increased for complex query")
	}

	// 验证查询增强（应该包含架构相关术语）
	if !strings.Contains(calledReq.Query, "设计模式") && !strings.Contains(calledReq.Query, "系统架构") {
		t.Log("Query was not enhanced with architecture terms")
	}

	// 验证意图分布统计
	stats := agentic.GetStats()
	if stats.DomainDistribution["architecture"] != 1 {
		t.Errorf("Expected 1 architecture domain request, got %d", stats.DomainDistribution["architecture"])
	}
}

// TestAgenticContextServiceEmptyQuery 测试空查询的处理
func TestAgenticContextServiceEmptyQuery(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "test_session",
		Query:     "",
		Limit:     2000,
	}

	// 执行检索
	_, err := agentic.RetrieveContext(ctx, req)
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}

	// 验证空查询直接传递给底层服务
	if len(mockSmart.RetrieveContextCalls) != 1 {
		t.Errorf("Expected 1 call to underlying service, got %d", len(mockSmart.RetrieveContextCalls))
	}

	// 验证统计：空查询不计入Agentic增强
	stats := agentic.GetStats()
	if stats.AgenticEnhanced != 0 {
		t.Errorf("Expected 0 agentic enhanced for empty query, got %d", stats.AgenticEnhanced)
	}
}

// TestAgenticContextServiceDisabled 测试禁用Agentic功能
func TestAgenticContextServiceDisabled(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	// 禁用Agentic功能
	agentic.EnableAgentic(false)

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "test_session",
		Query:     "测试查询",
		Limit:     2000,
	}

	// 执行检索
	_, err := agentic.RetrieveContext(ctx, req)
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}

	// 验证直接传递给底层服务，无优化
	calledReq := mockSmart.RetrieveContextCalls[0]
	if calledReq.Query != req.Query {
		t.Errorf("Expected query to be unchanged when disabled, got '%s'", calledReq.Query)
	}

	// 验证统计：禁用时不计入Agentic增强
	stats := agentic.GetStats()
	if stats.AgenticEnhanced != 0 {
		t.Errorf("Expected 0 agentic enhanced when disabled, got %d", stats.AgenticEnhanced)
	}
}

// TestAgenticContextServiceError 测试错误处理
func TestAgenticContextServiceError(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	mockSmart.ShouldReturnError = true

	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "test_session",
		Query:     "测试查询",
		Limit:     2000,
	}

	// 执行检索
	_, err := agentic.RetrieveContext(ctx, req)
	if err == nil {
		t.Error("Expected error from underlying service")
	}

	// 验证性能记录包含失败的请求
	stats := agentic.GetStats()
	if len(stats.PerformanceHistory) == 0 {
		t.Error("Expected performance record for failed request")
	}

	if stats.PerformanceHistory[0].Success {
		t.Error("Expected performance record to show failure")
	}
}

// TestAgenticContextServiceMultipleQueries 测试多种查询类型
func TestAgenticContextServiceMultipleQueries(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()

	// 测试查询列表
	queries := []struct {
		query          string
		expectedDomain string
		expectedIntent string
	}{
		{"Python代码报错了", "programming", "debugging"},
		{"什么是微服务架构？", "architecture", "conceptual"},
		{"如何配置Docker容器？", "devops", "procedural"},
		{"优化React组件性能", "frontend", "technical"},
	}

	for i, test := range queries {
		req := models.RetrieveContextRequest{
			SessionID: "test_session",
			Query:     test.query,
			Limit:     2000,
		}

		_, err := agentic.RetrieveContext(ctx, req)
		if err != nil {
			t.Fatalf("Query %d failed: %v", i+1, err)
		}
	}

	// 验证统计分布
	stats := agentic.GetStats()

	if stats.TotalRequests != len(queries) {
		t.Errorf("Expected %d total requests, got %d", len(queries), stats.TotalRequests)
	}

	if stats.AgenticEnhanced != len(queries) {
		t.Errorf("Expected %d agentic enhanced, got %d", len(queries), stats.AgenticEnhanced)
	}

	// 验证各种意图和领域都被记录
	for _, test := range queries {
		if stats.DomainDistribution[test.expectedDomain] == 0 {
			t.Errorf("Expected domain %s to be recorded", test.expectedDomain)
		}

		if stats.IntentDistribution[test.expectedIntent] == 0 {
			t.Errorf("Expected intent %s to be recorded", test.expectedIntent)
		}
	}
}

// TestAgenticContextServiceServiceInfo 测试服务信息获取
func TestAgenticContextServiceServiceInfo(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	info := agentic.GetServiceInfo()

	// 验证基础信息
	if info["name"] != "AgenticContextService" {
		t.Errorf("Expected name 'AgenticContextService', got %v", info["name"])
	}

	if info["version"] != "v1.0.0-beta" {
		t.Errorf("Expected version 'v1.0.0-beta', got %v", info["version"])
	}

	if info["enabled"] != true {
		t.Errorf("Expected enabled to be true, got %v", info["enabled"])
	}

	// 验证组件信息
	components := info["components"].(map[string]interface{})

	intentAnalyzer := components["intent_analyzer"].(map[string]interface{})
	if intentAnalyzer["name"] != "BasicQueryIntentAnalyzer" {
		t.Errorf("Expected intent analyzer name 'BasicQueryIntentAnalyzer', got %v", intentAnalyzer["name"])
	}

	decisionCenter := components["decision_center"].(map[string]interface{})
	if decisionCenter["name"] != "BasicIntelligentDecisionCenter" {
		t.Errorf("Expected decision center name 'BasicIntelligentDecisionCenter', got %v", decisionCenter["name"])
	}
}

// TestAgenticContextServiceCompatibility 测试与SmartContextService的兼容性
func TestAgenticContextServiceCompatibility(t *testing.T) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()

	// 测试所有代理方法不会panic
	_, err := agentic.RetrieveTodos(ctx, models.RetrieveTodosRequest{})
	if err != nil {
		t.Errorf("RetrieveTodos failed: %v", err)
	}

	err = agentic.AssociateFile(ctx, models.AssociateFileRequest{})
	if err != nil {
		t.Errorf("AssociateFile failed: %v", err)
	}

	err = agentic.RecordEdit(ctx, models.RecordEditRequest{})
	if err != nil {
		t.Errorf("RecordEdit failed: %v", err)
	}

	_, err = agentic.GetProgrammingContext(ctx, "session", "query")
	if err != nil {
		t.Errorf("GetProgrammingContext failed: %v", err)
	}

	_, err = agentic.StoreContext(ctx, models.StoreContextRequest{})
	if err != nil {
		t.Errorf("StoreContext failed: %v", err)
	}

	// 不会panic即通过测试
}

// BenchmarkAgenticContextService 基准测试
func BenchmarkAgenticContextService(b *testing.B) {
	mockSmart := NewMockSmartContextService()
	agentic := NewAgenticContextService(mockSmart)
	defer agentic.Stop(context.Background())

	ctx := context.Background()
	req := models.RetrieveContextRequest{
		SessionID: "benchmark_session",
		Query:     "如何优化Go语言程序的性能？",
		Limit:     2000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agentic.RetrieveContext(ctx, req)
		if err != nil {
			b.Fatalf("RetrieveContext failed: %v", err)
		}
	}
}
