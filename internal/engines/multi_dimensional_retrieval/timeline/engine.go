package timeline

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// TimescaleDBEngine TimescaleDB时间线检索引擎
type TimescaleDBEngine struct {
	db     *sql.DB
	config *TimescaleDBConfig
}

// TimescaleDBConfig TimescaleDB配置
type TimescaleDBConfig struct {
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	Database    string        `json:"database"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	SSLMode     string        `json:"ssl_mode"`
	MaxConns    int           `json:"max_conns"`
	MaxIdleTime time.Duration `json:"max_idle_time"`
}

// NewTimescaleDBEngine 创建TimescaleDB引擎
func NewTimescaleDBEngine(config *TimescaleDBConfig) (*TimescaleDBEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("TimescaleDB配置不能为空，请使用统一配置管理器加载配置")
	}

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Database, config.SSLMode)

	if config.Password != "" {
		connStr += fmt.Sprintf(" password=%s", config.Password)
	}

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("连接TimescaleDB失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(config.MaxConns)
	db.SetMaxIdleConns(config.MaxConns / 2)
	db.SetConnMaxIdleTime(config.MaxIdleTime)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("TimescaleDB连接测试失败: %w", err)
	}

	engine := &TimescaleDBEngine{
		db:     db,
		config: config,
	}

	// 初始化数据库结构
	if err := engine.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("初始化数据库结构失败: %w", err)
	}

	log.Printf("✅ TimescaleDB引擎初始化成功 - 数据库: %s", config.Database)
	return engine, nil
}

// initializeDatabase 初始化数据库结构
func (engine *TimescaleDBEngine) initializeDatabase() error {
	ctx := context.Background()

	// 创建数据库（如果不存在）
	if err := engine.createDatabaseIfNotExists(); err != nil {
		return err
	}

	// 创建扩展
	if err := engine.createExtensions(ctx); err != nil {
		return err
	}

	// 创建表结构
	if err := engine.createTables(ctx); err != nil {
		return err
	}

	// 创建hypertable
	if err := engine.createHypertable(ctx); err != nil {
		return err
	}

	// 创建索引
	if err := engine.createIndexes(ctx); err != nil {
		return err
	}

	return nil
}

// createDatabaseIfNotExists 创建数据库（如果不存在）
func (engine *TimescaleDBEngine) createDatabaseIfNotExists() error {
	// 连接到postgres数据库来创建目标数据库
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=postgres sslmode=%s",
		engine.config.Host, engine.config.Port, engine.config.Username, engine.config.SSLMode)

	if engine.config.Password != "" {
		connStr += fmt.Sprintf(" password=%s", engine.config.Password)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	// 检查数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
	err = db.QueryRow(query, engine.config.Database).Scan(&exists)
	if err != nil {
		return err
	}

	// 如果数据库不存在，创建它
	if !exists {
		createQuery := fmt.Sprintf("CREATE DATABASE %s", engine.config.Database)
		_, err = db.Exec(createQuery)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
		log.Printf("✅ 创建数据库: %s", engine.config.Database)
	}

	return nil
}

// createExtensions 创建扩展
func (engine *TimescaleDBEngine) createExtensions(ctx context.Context) error {
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE",
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
	}

	for _, ext := range extensions {
		if _, err := engine.db.ExecContext(ctx, ext); err != nil {
			log.Printf("⚠️ 创建扩展失败 (可能已存在): %v", err)
			// 不返回错误，因为扩展可能已经存在
		}
	}

	return nil
}

// createTables 创建表结构
func (engine *TimescaleDBEngine) createTables(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS timeline_events (
		id UUID DEFAULT gen_random_uuid(),
		user_id VARCHAR(255) NOT NULL,
		session_id VARCHAR(255) NOT NULL,
		workspace_id VARCHAR(255) NOT NULL,

		-- 时间维度（TimescaleDB的核心）
		timestamp TIMESTAMPTZ NOT NULL,
		event_duration INTERVAL,
		
		-- 事件内容
		event_type VARCHAR(100) NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		summary TEXT,
		
		-- 关联信息
		related_files TEXT[],
		related_concepts TEXT[],
		parent_event_id UUID, -- 逻辑外键，不使用数据库约束
		
		-- LLM分析结果
		intent VARCHAR(100),
		keywords TEXT[],
		entities JSONB,
		categories TEXT[],
		
		-- 质量指标
		importance_score FLOAT DEFAULT 0.5,
		relevance_score FLOAT DEFAULT 0.5,
		
		-- 索引字段（使用触发器更新）
		content_tsvector TSVECTOR,
		
		-- 创建和更新时间
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),

		-- TimescaleDB要求主键包含分区键
		PRIMARY KEY (id, timestamp)
	)`

	_, err := engine.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	log.Printf("✅ 创建表: timeline_events")
	return nil
}

// createHypertable 创建hypertable
func (engine *TimescaleDBEngine) createHypertable(ctx context.Context) error {
	// 检查是否已经是hypertable
	var isHypertable bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'timeline_events'
		)`

	err := engine.db.QueryRowContext(ctx, checkQuery).Scan(&isHypertable)
	if err != nil {
		log.Printf("⚠️ 检查hypertable状态失败: %v", err)
		return nil // 不阻止初始化过程
	}

	if !isHypertable {
		createHypertableSQL := `SELECT create_hypertable('timeline_events', 'timestamp')`
		_, err = engine.db.ExecContext(ctx, createHypertableSQL)
		if err != nil {
			log.Printf("⚠️ 创建hypertable失败: %v", err)
			return nil // 不阻止初始化过程
		}
		log.Printf("✅ 创建hypertable: timeline_events")
	}

	return nil
}

// createIndexes 创建索引
func (engine *TimescaleDBEngine) createIndexes(ctx context.Context) error {
	// 创建触发器函数来更新tsvector
	triggerFunction := `
		CREATE OR REPLACE FUNCTION update_content_tsvector() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsvector := to_tsvector('english',
				COALESCE(NEW.title, '') || ' ' ||
				COALESCE(NEW.content, '') || ' ' ||
				COALESCE(array_to_string(NEW.keywords, ' '), '')
			);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;`

	if _, err := engine.db.ExecContext(ctx, triggerFunction); err != nil {
		log.Printf("⚠️ 创建触发器函数失败: %v", err)
	}

	// 创建触发器
	trigger := `
		DROP TRIGGER IF EXISTS tsvector_update_trigger ON timeline_events;
		CREATE TRIGGER tsvector_update_trigger
		BEFORE INSERT OR UPDATE ON timeline_events
		FOR EACH ROW EXECUTE FUNCTION update_content_tsvector();`

	if _, err := engine.db.ExecContext(ctx, trigger); err != nil {
		log.Printf("⚠️ 创建触发器失败: %v", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_timeline_user_time ON timeline_events (user_id, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_workspace_time ON timeline_events (workspace_id, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_session ON timeline_events (session_id, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_content_search ON timeline_events USING GIN (content_tsvector)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_keywords ON timeline_events USING GIN (keywords)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_entities ON timeline_events USING GIN (entities)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_event_type ON timeline_events (event_type)",
		"CREATE INDEX IF NOT EXISTS idx_timeline_intent ON timeline_events (intent)",
	}

	for _, indexSQL := range indexes {
		if _, err := engine.db.ExecContext(ctx, indexSQL); err != nil {
			log.Printf("⚠️ 创建索引失败: %v", err)
			// 继续创建其他索引
		}
	}

	log.Printf("✅ 创建索引和触发器完成")
	return nil
}

// HealthCheck 健康检查
func (engine *TimescaleDBEngine) HealthCheck(ctx context.Context) error {
	return engine.db.PingContext(ctx)
}

// RetrieveEvents 检索时间线事件
func (engine *TimescaleDBEngine) RetrieveEvents(ctx context.Context, query *TimelineQuery) (*TimelineResult, error) {
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("查询参数验证失败: %w", err)
	}

	// 构建SQL查询
	sqlQuery, args := engine.buildRetrievalQuery(query)

	log.Printf("🔍 执行时间线查询: %s", sqlQuery)

	// 打印SQL参数（按占位符顺序）
	log.Printf("📋 SQL参数详情:")
	for i, arg := range args {
		log.Printf("  $%d: %v", i+1, arg)
	}

	// 执行查询
	rows, err := engine.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	// 解析结果
	events := []TimelineEvent{}
	for rows.Next() {
		var event TimelineEvent
		err := rows.Scan(
			&event.ID, &event.UserID, &event.SessionID, &event.WorkspaceID,
			&event.Timestamp, &event.EventDuration,
			&event.EventType, &event.Title, &event.Content, &event.Summary,
			&event.RelatedFiles, &event.RelatedConcepts, &event.ParentEventID,
			&event.Intent, &event.Keywords, &event.Entities, &event.Categories,
			&event.ImportanceScore, &event.RelevanceScore,
			&event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("解析查询结果失败: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("查询结果遍历失败: %w", err)
	}

	// 获取总数
	total, err := engine.getEventCount(ctx, query)
	if err != nil {
		log.Printf("⚠️ 获取总数失败: %v", err)
		total = len(events)
	}

	return &TimelineResult{
		Events: events,
		Total:  total,
	}, nil
}

// buildRetrievalQuery 构建检索查询
func (engine *TimescaleDBEngine) buildRetrievalQuery(query *TimelineQuery) (string, []interface{}) {
	baseSQL := `
		SELECT
			id, user_id, session_id, workspace_id,
			timestamp, event_duration,
			event_type, title, content, summary,
			related_files, related_concepts, parent_event_id,
			intent, keywords, entities, categories,
			importance_score, relevance_score,
			created_at, updated_at
		FROM timeline_events
		WHERE 1=1`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// 用户过滤
	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
	args = append(args, query.UserID)
	argIndex++

	// 工作空间过滤
	if query.WorkspaceID != "" {
		conditions = append(conditions, fmt.Sprintf("workspace_id = $%d", argIndex))
		args = append(args, query.WorkspaceID)
		argIndex++
	}

	// 会话过滤
	if query.SessionID != "" {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIndex))
		args = append(args, query.SessionID)
		argIndex++
	}

	// 🆕 直接时间范围过滤（用于时间回忆查询，优先级最高）
	if !query.StartTime.IsZero() && !query.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d AND timestamp <= $%d", argIndex, argIndex+1))
		args = append(args, query.StartTime, query.EndTime)
		argIndex += 2
		log.Printf("🕒 [时间回忆] 使用直接时间范围过滤: %s - %s",
			query.StartTime.Format("2006-01-02 15:04:05"),
			query.EndTime.Format("2006-01-02 15:04:05"))
	} else {
		// 时间范围过滤（原有逻辑）
		if len(query.TimeRanges) > 0 {
			timeConditions := []string{}
			for _, tr := range query.TimeRanges {
				timeConditions = append(timeConditions,
					fmt.Sprintf("(timestamp >= $%d AND timestamp <= $%d)", argIndex, argIndex+1))
				args = append(args, tr.StartTime, tr.EndTime)
				argIndex += 2
			}
			if len(timeConditions) > 0 {
				conditions = append(conditions, "("+fmt.Sprintf("%s", timeConditions[0])+")")
			}
		}

		// 时间窗口过滤
		if query.TimeWindow != "" {
			conditions = append(conditions, fmt.Sprintf("timestamp >= NOW() - INTERVAL '%s'", query.TimeWindow))
		}
	}

	// 事件类型过滤
	if len(query.EventTypes) > 0 {
		conditions = append(conditions, fmt.Sprintf("event_type = ANY($%d)", argIndex))
		args = append(args, pq.Array(query.EventTypes))
		argIndex++
	}

	// 意图过滤
	if query.Intent != "" {
		conditions = append(conditions, fmt.Sprintf("intent = $%d", argIndex))
		args = append(args, query.Intent)
		argIndex++
	}

	// 全文搜索 - 使用混合策略：tsvector + keywords + title/content ILIKE
	if query.SearchText != "" {
		// 构建搜索条件数组
		var searchConditions []string

		// 1. tsvector全文搜索
		tsvectorCondition := fmt.Sprintf("content_tsvector @@ plainto_tsquery('chinese_zh', $%d)", argIndex)
		args = append(args, query.SearchText)
		argIndex++
		searchConditions = append(searchConditions, tsvectorCondition)

		// 2. 基于LLM关键词的多维度搜索
		if len(query.Keywords) > 0 {
			// 限制关键词数量，避免性能问题（最多5个关键词）
			effectiveKeywords := query.Keywords
			if len(effectiveKeywords) > 5 {
				effectiveKeywords = effectiveKeywords[:5]
			}

			// 构建LIKE模式数组
			var likePatterns []string
			for _, keyword := range effectiveKeywords {
				likePatterns = append(likePatterns, "%"+keyword+"%")
			}

			// 2a. keywords字段搜索
			keywordsCondition := fmt.Sprintf("keywords::text ILIKE ANY($%d)", argIndex)
			args = append(args, pq.Array(likePatterns))
			argIndex++
			searchConditions = append(searchConditions, keywordsCondition)

			// 2b. title和content字段搜索
			titleContentCondition := fmt.Sprintf("(title ILIKE ANY($%d) OR content ILIKE ANY($%d))",
				argIndex, argIndex+1)
			args = append(args, pq.Array(likePatterns), pq.Array(likePatterns))
			argIndex += 2
			searchConditions = append(searchConditions, titleContentCondition)

			log.Printf("🔍 [时间线查询] 构建多维度搜索：tsvector + keywords + title/content，关键词数量: %d", len(effectiveKeywords))
		} else {
			log.Printf("⚠️ [时间线查询] 仅使用tsvector搜索，因为关键词列表为空")
		}

		// 组合所有搜索条件为OR关系
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(searchConditions, " OR ")))
	}

	// 质量过滤
	if query.MinImportance > 0 {
		conditions = append(conditions, fmt.Sprintf("importance_score >= $%d", argIndex))
		args = append(args, query.MinImportance)
		argIndex++
	}

	if query.MinRelevance > 0 {
		conditions = append(conditions, fmt.Sprintf("relevance_score >= $%d", argIndex))
		args = append(args, query.MinRelevance)
		argIndex++
	}

	// 组合条件
	if len(conditions) > 0 {
		baseSQL += " AND " + fmt.Sprintf("%s", conditions[0])
		for i := 1; i < len(conditions); i++ {
			baseSQL += " AND " + conditions[i]
		}
	}

	// 排序
	switch query.OrderBy {
	case "relevance_score":
		baseSQL += " ORDER BY relevance_score DESC, timestamp DESC"
	case "importance_score":
		baseSQL += " ORDER BY importance_score DESC, timestamp DESC"
	default:
		baseSQL += " ORDER BY timestamp DESC"
	}

	// 分页
	baseSQL += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, query.Limit, query.Offset)

	return baseSQL, args
}

// getEventCount 获取事件总数
func (engine *TimescaleDBEngine) getEventCount(ctx context.Context, query *TimelineQuery) (int, error) {
	countSQL := "SELECT COUNT(*) FROM timeline_events WHERE user_id = $1"
	var total int
	err := engine.db.QueryRowContext(ctx, countSQL, query.UserID).Scan(&total)
	return total, err
}

// CreateEvent 创建时间线事件
func (engine *TimescaleDBEngine) CreateEvent(ctx context.Context, req *CreateTimelineEventRequest) (*TimelineEvent, error) {
	insertSQL := `
		INSERT INTO timeline_events (
			user_id, session_id, workspace_id,
			timestamp, event_type, title, content, summary,
			related_files, related_concepts, parent_event_id,
			intent, keywords, entities, categories,
			importance_score, relevance_score
		) VALUES (
			$1, $2, $3, NOW(), $4, $5, $6, $7,
			$8, $9, NULLIF($10, '')::uuid, $11, $12, $13, $14, $15, $16
		) RETURNING id, timestamp, created_at, updated_at`

	var event TimelineEvent
	var parentEventID *string
	if req.ParentEventID != "" {
		parentEventID = &req.ParentEventID
	}

	err := engine.db.QueryRowContext(ctx, insertSQL,
		req.UserID, req.SessionID, req.WorkspaceID,
		req.EventType, req.Title, req.Content, req.Summary,
		pq.Array(req.RelatedFiles), pq.Array(req.RelatedConcepts), parentEventID,
		req.Intent, pq.Array(req.Keywords), EntityArray(req.Entities), pq.Array(req.Categories),
		req.ImportanceScore, req.RelevanceScore,
	).Scan(&event.ID, &event.Timestamp, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("创建时间线事件失败: %w", err)
	}

	// 填充返回的事件信息
	event.UserID = req.UserID
	event.SessionID = req.SessionID
	event.WorkspaceID = req.WorkspaceID
	event.EventType = req.EventType
	event.Title = req.Title
	event.Content = req.Content
	event.Summary = &req.Summary

	log.Printf("✅ 创建时间线事件: %s - %s", event.ID, event.Title)
	return &event, nil
}

// Close 关闭连接
func (engine *TimescaleDBEngine) Close() error {
	if engine.db != nil {
		return engine.db.Close()
	}
	return nil
}

// StoreEvent 存储时间线事件
func (engine *TimescaleDBEngine) StoreEvent(ctx context.Context, event *TimelineEvent) (string, error) {
	// 验证事件
	if err := event.Validate(); err != nil {
		return "", fmt.Errorf("事件验证失败: %w", err)
	}

	// 生成ID（如果没有提供）
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// 设置时间戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event.UpdatedAt = time.Now()

	// 构建插入SQL
	insertSQL := `
		INSERT INTO timeline_events (
			id, user_id, session_id, workspace_id, timestamp, event_duration,
			event_type, title, content, summary, related_files, related_concepts,
			parent_event_id, intent, keywords, entities, categories,
			importance_score, relevance_score, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	// 执行插入
	_, err := engine.db.ExecContext(ctx, insertSQL,
		event.ID, event.UserID, event.SessionID, event.WorkspaceID,
		event.Timestamp, event.EventDuration, event.EventType, event.Title,
		event.Content, event.Summary, pq.Array(event.RelatedFiles),
		pq.Array(event.RelatedConcepts), event.ParentEventID, event.Intent,
		pq.Array(event.Keywords), event.Entities, pq.Array(event.Categories),
		event.ImportanceScore, event.RelevanceScore, event.CreatedAt, event.UpdatedAt,
	)

	if err != nil {
		return "", fmt.Errorf("插入时间线事件失败: %w", err)
	}

	log.Printf("✅ 时间线事件存储成功 - ID: %s, 标题: %s", event.ID, event.Title)
	return event.ID, nil
}
