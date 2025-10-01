package knowledge

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jEngine Neo4j知识图谱检索引擎
type Neo4jEngine struct {
	driver neo4j.DriverWithContext
	config *Neo4jConfig
}

// Neo4jConfig Neo4j配置
type Neo4jConfig struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`

	// 连接池配置
	MaxConnectionPoolSize   int           `json:"max_connection_pool_size"`
	ConnectionTimeout       time.Duration `json:"connection_timeout"`
	MaxTransactionRetryTime time.Duration `json:"max_transaction_retry_time"`
}

// NewNeo4jEngine 创建Neo4j引擎
func NewNeo4jEngine(config *Neo4jConfig) (*Neo4jEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("Neo4j配置不能为空，请使用统一配置管理器加载配置")
	}

	// 创建驱动
	driver, err := neo4j.NewDriverWithContext(
		config.URI,
		neo4j.BasicAuth(config.Username, config.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = config.MaxConnectionPoolSize
			c.ConnectionAcquisitionTimeout = config.ConnectionTimeout
			c.MaxTransactionRetryTime = config.MaxTransactionRetryTime
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建Neo4j驱动失败: %w", err)
	}

	engine := &Neo4jEngine{
		driver: driver,
		config: config,
	}

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.verifyConnection(ctx); err != nil {
		return nil, fmt.Errorf("Neo4j连接验证失败: %w", err)
	}

	// 初始化图谱结构
	if err := engine.initializeGraph(ctx); err != nil {
		return nil, fmt.Errorf("初始化图谱结构失败: %w", err)
	}

	log.Printf("✅ Neo4j引擎初始化成功 - 数据库: %s", config.Database)
	return engine, nil
}

// verifyConnection 验证连接
func (engine *Neo4jEngine) verifyConnection(ctx context.Context) error {
	return engine.driver.VerifyConnectivity(ctx)
}

// initializeGraph 初始化图谱结构
func (engine *Neo4jEngine) initializeGraph(ctx context.Context) error {
	session := engine.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: engine.config.Database,
	})
	defer session.Close(ctx)

	// 创建约束和索引
	constraints := []string{
		// 概念节点唯一性约束
		"CREATE CONSTRAINT concept_name_unique IF NOT EXISTS FOR (c:Concept) REQUIRE c.name IS UNIQUE",

		// 技术节点唯一性约束
		"CREATE CONSTRAINT technology_name_unique IF NOT EXISTS FOR (t:Technology) REQUIRE t.name IS UNIQUE",

		// 项目节点唯一性约束
		"CREATE CONSTRAINT project_name_unique IF NOT EXISTS FOR (p:Project) REQUIRE p.name IS UNIQUE",

		// 用户节点唯一性约束
		"CREATE CONSTRAINT user_id_unique IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE",
	}

	for _, constraint := range constraints {
		_, err := session.Run(ctx, constraint, nil)
		if err != nil {
			log.Printf("⚠️ 创建约束失败 (可能已存在): %v", err)
		}
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX concept_category_idx IF NOT EXISTS FOR (c:Concept) ON (c.category)",
		"CREATE INDEX technology_type_idx IF NOT EXISTS FOR (t:Technology) ON (t.type)",
		"CREATE INDEX project_domain_idx IF NOT EXISTS FOR (p:Project) ON (p.domain)",
		"CREATE FULLTEXT INDEX concept_search_idx IF NOT EXISTS FOR (c:Concept) ON EACH [c.name, c.description, c.keywords]",
		"CREATE FULLTEXT INDEX technology_search_idx IF NOT EXISTS FOR (t:Technology) ON EACH [t.name, t.description, t.keywords]",
	}

	for _, index := range indexes {
		_, err := session.Run(ctx, index, nil)
		if err != nil {
			log.Printf("⚠️ 创建索引失败 (可能已存在): %v", err)
		}
	}

	log.Printf("✅ Neo4j图谱结构初始化完成")
	return nil
}

// CreateConcept 创建概念节点
func (engine *Neo4jEngine) CreateConcept(ctx context.Context, concept *Concept) error {
	session := engine.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: engine.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MERGE (c:Concept {name: $name})
		SET c.description = $description,
		    c.category = $category,
		    c.keywords = $keywords,
		    c.importance = $importance,
		    c.created_at = datetime(),
		    c.updated_at = datetime()
		RETURN c.name as name`

	parameters := map[string]interface{}{
		"name":        concept.Name,
		"description": concept.Description,
		"category":    concept.Category,
		"keywords":    concept.Keywords,
		"importance":  concept.Importance,
	}

	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return fmt.Errorf("创建概念节点失败: %w", err)
	}

	if result.Next(ctx) {
		name, _ := result.Record().Get("name")
		log.Printf("✅ 创建概念节点: %s", name)
	}

	return result.Err()
}

// CreateTechnology 创建技术节点
func (engine *Neo4jEngine) CreateTechnology(ctx context.Context, tech *Technology) error {
	session := engine.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: engine.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MERGE (t:Technology {name: $name})
		SET t.description = $description,
		    t.type = $type,
		    t.version = $version,
		    t.keywords = $keywords,
		    t.popularity = $popularity,
		    t.created_at = datetime(),
		    t.updated_at = datetime()
		RETURN t.name as name`

	parameters := map[string]interface{}{
		"name":        tech.Name,
		"description": tech.Description,
		"type":        tech.Type,
		"version":     tech.Version,
		"keywords":    tech.Keywords,
		"popularity":  tech.Popularity,
	}

	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return fmt.Errorf("创建技术节点失败: %w", err)
	}

	if result.Next(ctx) {
		name, _ := result.Record().Get("name")
		log.Printf("✅ 创建技术节点: %s", name)
	}

	return result.Err()
}

// CreateRelationship 创建关系
func (engine *Neo4jEngine) CreateRelationship(ctx context.Context, rel *Relationship) error {
	session := engine.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: engine.config.Database,
	})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (from {name: $from_name})
		MATCH (to {name: $to_name})
		MERGE (from)-[r:%s]->(to)
		SET r.strength = $strength,
		    r.description = $description,
		    r.created_at = datetime(),
		    r.updated_at = datetime()
		RETURN type(r) as relationship_type`, rel.Type)

	parameters := map[string]interface{}{
		"from_name":   rel.FromName,
		"to_name":     rel.ToName,
		"strength":    rel.Strength,
		"description": rel.Description,
	}

	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return fmt.Errorf("创建关系失败: %w", err)
	}

	if result.Next(ctx) {
		relType, _ := result.Record().Get("relationship_type")
		log.Printf("✅ 创建关系: %s -[%s]-> %s", rel.FromName, relType, rel.ToName)
	}

	return result.Err()
}

// ExpandKnowledge 知识图谱扩展检索
func (engine *Neo4jEngine) ExpandKnowledge(ctx context.Context, query *KnowledgeQuery) (*KnowledgeResult, error) {
	session := engine.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: engine.config.Database,
	})
	defer session.Close(ctx)

	startTime := time.Now()

	// 构建Cypher查询
	cypherQuery, parameters := engine.buildKnowledgeQuery(query)

	log.Printf("🔍 执行知识图谱查询: %s", cypherQuery)

	// 执行查询
	result, err := session.Run(ctx, cypherQuery, parameters)
	if err != nil {
		return nil, fmt.Errorf("执行知识图谱查询失败: %w", err)
	}

	// 解析结果
	nodes := []KnowledgeNode{}
	relationships := []KnowledgeRelationship{}

	for result.Next(ctx) {
		record := result.Record()

		// 解析节点
		if nodeValue, found := record.Get("node"); found {
			if node, ok := nodeValue.(neo4j.Node); ok {
				knowledgeNode := engine.parseNode(node)
				nodes = append(nodes, knowledgeNode)
			}
		}

		// 解析关系
		if relValue, found := record.Get("relationship"); found {
			if rel, ok := relValue.(neo4j.Relationship); ok {
				knowledgeRel := engine.parseRelationship(rel)
				relationships = append(relationships, knowledgeRel)
			}
		}
	}

	if err = result.Err(); err != nil {
		return nil, fmt.Errorf("解析查询结果失败: %w", err)
	}

	duration := time.Since(startTime)

	return &KnowledgeResult{
		Nodes:         nodes,
		Relationships: relationships,
		Total:         len(nodes),
		Duration:      duration,
		Query:         query,
	}, nil
}

// buildKnowledgeQuery 构建知识图谱查询
func (engine *Neo4jEngine) buildKnowledgeQuery(query *KnowledgeQuery) (string, map[string]interface{}) {
	var cypherQuery string
	parameters := make(map[string]interface{})

	switch query.QueryType {
	case "expand":
		// 扩展查询：从给定概念开始，扩展相关概念
		cypherQuery = `
			MATCH (start {name: $start_concept})
			MATCH (start)-[r]-(related)
			WHERE r.strength >= $min_strength
			RETURN DISTINCT related as node, r as relationship
			ORDER BY r.strength DESC
			LIMIT $limit`

		parameters["start_concept"] = query.StartConcepts[0]
		parameters["min_strength"] = query.MinStrength
		parameters["limit"] = query.Limit

	case "path":
		// 路径查询：查找两个概念之间的路径
		cypherQuery = `
			MATCH path = shortestPath((start {name: $start_concept})-[*..4]-(end {name: $end_concept}))
			UNWIND nodes(path) as node
			UNWIND relationships(path) as relationship
			RETURN DISTINCT node, relationship
			LIMIT $limit`

		parameters["start_concept"] = query.StartConcepts[0]
		parameters["end_concept"] = query.EndConcepts[0]
		parameters["limit"] = query.Limit

	case "similarity":
		// 相似性查询：查找相似的概念
		cypherQuery = `
			MATCH (concept:Concept)
			WHERE concept.category IN $categories
			AND any(keyword IN $keywords WHERE keyword IN concept.keywords)
			RETURN concept as node, null as relationship
			ORDER BY concept.importance DESC
			LIMIT $limit`

		parameters["categories"] = query.Categories
		parameters["keywords"] = query.Keywords
		parameters["limit"] = query.Limit

	default:
		// 默认全文搜索
		cypherQuery = `
			CALL db.index.fulltext.queryNodes('concept_search_idx', $search_text)
			YIELD node, score
			WHERE score >= $min_score
			RETURN node, null as relationship, score
			ORDER BY score DESC
			LIMIT $limit`

		parameters["search_text"] = query.SearchText
		parameters["min_score"] = query.MinScore
		parameters["limit"] = query.Limit
	}

	return cypherQuery, parameters
}

// parseNode 解析节点
func (engine *Neo4jEngine) parseNode(node neo4j.Node) KnowledgeNode {
	props := node.Props

	return KnowledgeNode{
		ID:          node.ElementId,
		Labels:      node.Labels,
		Name:        getStringProp(props, "name"),
		Description: getStringProp(props, "description"),
		Category:    getStringProp(props, "category"),
		Keywords:    getStringArrayProp(props, "keywords"),
		Properties:  props,
	}
}

// parseRelationship 解析关系
func (engine *Neo4jEngine) parseRelationship(rel neo4j.Relationship) KnowledgeRelationship {
	props := rel.Props

	return KnowledgeRelationship{
		ID:          rel.ElementId,
		Type:        rel.Type,
		StartNodeID: rel.StartElementId,
		EndNodeID:   rel.EndElementId,
		Strength:    getFloatProp(props, "strength"),
		Description: getStringProp(props, "description"),
		Properties:  props,
	}
}

// 辅助函数
func getStringProp(props map[string]interface{}, key string) string {
	if val, ok := props[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringArrayProp(props map[string]interface{}, key string) []string {
	if val, ok := props[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, len(arr))
			for i, v := range arr {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return []string{}
}

func getFloatProp(props map[string]interface{}, key string) float64 {
	if val, ok := props[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
		if i, ok := val.(int64); ok {
			return float64(i)
		}
	}
	return 0.0
}

// HealthCheck 健康检查
func (engine *Neo4jEngine) HealthCheck(ctx context.Context) error {
	return engine.driver.VerifyConnectivity(ctx)
}

// Close 关闭连接
func (engine *Neo4jEngine) Close(ctx context.Context) error {
	return engine.driver.Close(ctx)
}
