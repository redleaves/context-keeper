package multi_dimensional_retrieval

import (
	"context"

	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/knowledge"
	"github.com/contextkeeper/service/internal/engines/multi_dimensional_retrieval/timeline"
)

// TimelineEngineAdapter TimescaleDB引擎适配器
type TimelineEngineAdapter struct {
	engine *timeline.TimescaleDBEngine
}

// NewTimelineEngineAdapter 创建时间线引擎适配器
func NewTimelineEngineAdapter(engine *timeline.TimescaleDBEngine) TimelineEngine {
	return &TimelineEngineAdapter{
		engine: engine,
	}
}

// RetrieveEvents 检索时间线事件
func (adapter *TimelineEngineAdapter) RetrieveEvents(ctx context.Context, query *TimelineQuery) (*TimelineResult, error) {
	// 转换查询格式
	timelineQuery := &timeline.TimelineQuery{
		UserID:      query.UserID,
		SessionID:   query.SessionID,
		WorkspaceID: query.WorkspaceID,
		Keywords:    query.Keywords,
		EventTypes:  query.EventTypes,
		Limit:       query.Limit,
		Offset:      query.Offset,
	}

	// 转换时间范围
	if len(query.TimeRanges) > 0 {
		timelineQuery.TimeRanges = make([]timeline.TimeRange, len(query.TimeRanges))
		for i, tr := range query.TimeRanges {
			timelineQuery.TimeRanges[i] = timeline.TimeRange{
				StartTime: tr.StartTime,
				EndTime:   tr.EndTime,
				Label:     tr.Label,
			}
		}
	}

	// 执行查询
	result, err := adapter.engine.RetrieveEvents(ctx, timelineQuery)
	if err != nil {
		return nil, err
	}

	// 转换结果格式 - 直接返回统一模型，不需要转换
	events := result.Events

	return &TimelineResult{
		Events: events,
		Total:  result.Total,
	}, nil
}

// GetAggregation 获取时间聚合信息
func (adapter *TimelineEngineAdapter) GetAggregation(ctx context.Context, query *TimelineQuery) (*TimelineAggregation, error) {
	// TODO: 实现聚合查询
	// 目前返回空聚合结果
	return &TimelineAggregation{
		TimeBuckets: []TimeBucket{},
		Summary: &Summary{
			TotalEvents:   0,
			TimeSpan:      "unknown",
			TopKeywords:   []string{},
			TopEventTypes: []string{},
		},
	}, nil
}

// HealthCheck 健康检查
func (adapter *TimelineEngineAdapter) HealthCheck(ctx context.Context) error {
	return adapter.engine.HealthCheck(ctx)
}

// Close 关闭连接
func (adapter *TimelineEngineAdapter) Close() error {
	return adapter.engine.Close()
}

// KnowledgeEngineAdapter Neo4j引擎适配器
type KnowledgeEngineAdapter struct {
	engine *knowledge.Neo4jEngine
}

// NewKnowledgeEngineAdapter 创建知识图谱引擎适配器
func NewKnowledgeEngineAdapter(engine *knowledge.Neo4jEngine) KnowledgeEngine {
	return &KnowledgeEngineAdapter{
		engine: engine,
	}
}

// ExpandGraph 扩展知识图谱
func (adapter *KnowledgeEngineAdapter) ExpandGraph(ctx context.Context, query *KnowledgeQuery) (*KnowledgeResult, error) {
	// 转换查询格式
	knowledgeQuery := &knowledge.KnowledgeQuery{
		QueryType:     "expand",
		StartConcepts: query.StartNodes,
		MaxDepth:      query.MaxDepth,
		Limit:         query.MaxNodes,
		MinStrength:   query.MinWeight,
	}

	// 执行查询
	result, err := adapter.engine.ExpandKnowledge(ctx, knowledgeQuery)
	if err != nil {
		return nil, err
	}

	// 转换结果格式
	nodes := make([]KnowledgeNode, len(result.Nodes))
	for i, node := range result.Nodes {
		nodes[i] = KnowledgeNode{
			ID:         node.ID,
			Name:       node.Name,
			Type:       node.Category,
			Properties: node.Properties,
			Score:      node.Score,
			Depth:      0, // TODO: 计算深度
		}
	}

	relationships := make([]Relationship, len(result.Relationships))
	for i, rel := range result.Relationships {
		relationships[i] = Relationship{
			ID:         rel.ID,
			SourceID:   rel.StartNodeID,
			TargetID:   rel.EndNodeID,
			Type:       rel.Type,
			Weight:     rel.Strength,
			Properties: rel.Properties,
		}
	}

	return &KnowledgeResult{
		Nodes:         nodes,
		Relationships: relationships,
		Paths:         []Path{}, // TODO: 实现路径转换
		Total:         result.Total,
	}, nil
}

// GetRelatedConcepts 获取相关概念
func (adapter *KnowledgeEngineAdapter) GetRelatedConcepts(ctx context.Context, concepts []string) ([]string, error) {
	if len(concepts) == 0 {
		return []string{}, nil
	}

	// 使用扩展查询获取相关概念
	query := &knowledge.KnowledgeQuery{
		QueryType:     "expand",
		StartConcepts: concepts,
		MaxDepth:      2,
		Limit:         20,
		MinStrength:   0.5,
	}

	result, err := adapter.engine.ExpandKnowledge(ctx, query)
	if err != nil {
		return nil, err
	}

	// 提取概念名称
	relatedConcepts := make([]string, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		// 排除起始概念
		isStartConcept := false
		for _, startConcept := range concepts {
			if node.Name == startConcept {
				isStartConcept = true
				break
			}
		}
		if !isStartConcept {
			relatedConcepts = append(relatedConcepts, node.Name)
		}
	}

	return relatedConcepts, nil
}

// HealthCheck 健康检查
func (adapter *KnowledgeEngineAdapter) HealthCheck(ctx context.Context) error {
	return adapter.engine.HealthCheck(ctx)
}

// Close 关闭连接
func (adapter *KnowledgeEngineAdapter) Close() error {
	return adapter.engine.Close(context.Background())
}

// MockVectorEngine 模拟向量引擎（用于测试）
type MockVectorEngine struct{}

// NewMockVectorEngine 创建模拟向量引擎
func NewMockVectorEngine() VectorEngine {
	return &MockVectorEngine{}
}

// SearchMultiDimensional 多维度向量检索
func (engine *MockVectorEngine) SearchMultiDimensional(ctx context.Context, query *VectorQuery) (*VectorResult, error) {
	// 返回模拟结果
	documents := []VectorDocument{
		{
			ID:         "vector_doc_1",
			Content:    "模拟向量检索结果",
			Score:      0.85,
			Similarity: 0.85,
			Dimensions: map[string]float64{
				"semantic": 0.9,
				"context":  0.8,
			},
			Metadata: map[string]interface{}{
				"source": "mock_vector_engine",
				"type":   "multi_dimensional",
			},
		},
	}

	return &VectorResult{
		Documents: documents,
		Total:     len(documents),
	}, nil
}

// SearchLegacy 传统向量检索
func (engine *MockVectorEngine) SearchLegacy(ctx context.Context, query *LegacyVectorQuery) (*VectorResult, error) {
	// 返回模拟结果
	documents := []VectorDocument{
		{
			ID:         "legacy_vector_doc_1",
			Content:    "传统向量检索结果",
			Score:      0.80,
			Similarity: 0.80,
			Dimensions: map[string]float64{
				"legacy": 0.8,
			},
			Metadata: map[string]interface{}{
				"source": "mock_vector_engine",
				"type":   "legacy",
			},
		},
	}

	return &VectorResult{
		Documents: documents,
		Total:     len(documents),
	}, nil
}

// HealthCheck 健康检查
func (engine *MockVectorEngine) HealthCheck(ctx context.Context) error {
	return nil
}

// Close 关闭连接
func (engine *MockVectorEngine) Close() error {
	return nil
}
