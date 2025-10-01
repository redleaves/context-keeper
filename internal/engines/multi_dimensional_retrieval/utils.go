package multi_dimensional_retrieval

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SimpleCache 简单内存缓存实现
type SimpleCache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	maxSize int
	ttl     time.Duration
}

type cacheItem struct {
	value     interface{}
	createdAt time.Time
}

// NewCache 创建缓存
func NewCache(maxSize int, ttl time.Duration) Cache {
	cache := &SimpleCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// Get 获取缓存项
func (c *SimpleCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(item.createdAt) > c.ttl {
		delete(c.items, key)
		return nil
	}

	return item.value
}

// Set 设置缓存项
func (c *SimpleCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果缓存已满，删除最旧的项
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &cacheItem{
		value:     value,
		createdAt: time.Now(),
	}
}

// Delete 删除缓存项
func (c *SimpleCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear 清空缓存
func (c *SimpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Size 获取缓存大小
func (c *SimpleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictOldest 删除最旧的缓存项
func (c *SimpleCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.createdAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanup 清理过期缓存项
func (c *SimpleCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.Sub(item.createdAt) > c.ttl {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// SimpleRateLimiter 简单限流器实现
type SimpleRateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerMinute int) RateLimiter {
	// 转换为每秒请求数
	rps := float64(requestsPerMinute) / 60.0
	return &SimpleRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), requestsPerMinute),
	}
}

// Wait 等待获取令牌
func (r *SimpleRateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}

// Allow 检查是否允许请求
func (r *SimpleRateLimiter) Allow() bool {
	return r.limiter.Allow()
}

// SimpleMetrics 简单指标实现
type SimpleMetrics struct {
	mu           sync.RWMutex
	queryStats   *QueryStats
	engineStats  map[string]*EngineStats
	queryHistory []QueryRecord
}

type QueryRecord struct {
	Timestamp   time.Time
	Duration    time.Duration
	ResultCount int
	Engines     []string
	Success     bool
}

// NewMetrics 创建指标收集器
func NewMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		queryStats: &QueryStats{
			LastUpdated: time.Now(),
		},
		engineStats:  make(map[string]*EngineStats),
		queryHistory: make([]QueryRecord, 0),
	}
}

// RecordQuery 记录查询
func (m *SimpleMetrics) RecordQuery(duration time.Duration, resultCount int, engines []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 记录查询历史
	record := QueryRecord{
		Timestamp:   time.Now(),
		Duration:    duration,
		ResultCount: resultCount,
		Engines:     engines,
		Success:     true,
	}

	m.queryHistory = append(m.queryHistory, record)

	// 保持最近1000条记录
	if len(m.queryHistory) > 1000 {
		m.queryHistory = m.queryHistory[len(m.queryHistory)-1000:]
	}

	// 更新统计信息
	m.updateQueryStats()
	m.updateEngineStats(engines, duration)
}

// updateQueryStats 更新查询统计
func (m *SimpleMetrics) updateQueryStats() {
	if len(m.queryHistory) == 0 {
		return
	}

	totalDuration := time.Duration(0)
	successCount := 0

	for _, record := range m.queryHistory {
		totalDuration += record.Duration
		if record.Success {
			successCount++
		}
	}

	m.queryStats.TotalQueries = int64(len(m.queryHistory))
	m.queryStats.AverageLatency = totalDuration / time.Duration(len(m.queryHistory))
	m.queryStats.SuccessRate = float64(successCount) / float64(len(m.queryHistory))
	m.queryStats.LastUpdated = time.Now()
}

// updateEngineStats 更新引擎统计
func (m *SimpleMetrics) updateEngineStats(engines []string, duration time.Duration) {
	for _, engine := range engines {
		stats, exists := m.engineStats[engine]
		if !exists {
			stats = &EngineStats{
				Name:            engine,
				HealthStatus:    "unknown",
				LastHealthCheck: time.Now(),
			}
			m.engineStats[engine] = stats
		}

		stats.QueriesHandled++
		// 简单的移动平均
		if stats.AverageLatency == 0 {
			stats.AverageLatency = duration
		} else {
			stats.AverageLatency = (stats.AverageLatency + duration) / 2
		}
	}
}

// GetQueryStats 获取查询统计
func (m *SimpleMetrics) GetQueryStats() *QueryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	stats := *m.queryStats
	return &stats
}

// GetEngineStats 获取引擎统计
func (m *SimpleMetrics) GetEngineStats() map[string]*EngineStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	result := make(map[string]*EngineStats)
	for name, stats := range m.engineStats {
		statsCopy := *stats
		result[name] = &statsCopy
	}

	return result
}

// Reset 重置指标
func (m *SimpleMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queryStats = &QueryStats{
		LastUpdated: time.Now(),
	}
	m.engineStats = make(map[string]*EngineStats)
	m.queryHistory = make([]QueryRecord, 0)
}

// AdaptSemanticAnalysisToQueries 将语义分析结果适配到各存储引擎查询
func AdaptSemanticAnalysisToQueries(analysis interface{}, userContext *UserContext) (*AdaptedQueries, error) {
	// TODO: 实现语义分析结果到查询的适配逻辑
	return &AdaptedQueries{}, nil
}

// UserContext 用户上下文
type UserContext struct {
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	WorkspaceID string `json:"workspace_id"`
}

// AdaptedQueries 适配后的查询
type AdaptedQueries struct {
	TimelineQuery  *TimelineQuery  `json:"timeline_query"`
	KnowledgeQuery *KnowledgeQuery `json:"knowledge_query"`
	VectorQuery    *VectorQuery    `json:"vector_query"`
}
