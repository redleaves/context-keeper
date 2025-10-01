package multi_dimensional_retrieval

import (
	"fmt"
	"time"
)

// MultiDimensionalRetrievalConfig 多维度检索配置
type MultiDimensionalRetrievalConfig struct {
	// 总开关
	Enabled bool `yaml:"enabled" json:"enabled"`

	// 各维度开关
	TimelineEnabled  bool `yaml:"timeline_enabled" json:"timeline_enabled"`
	KnowledgeEnabled bool `yaml:"knowledge_enabled" json:"knowledge_enabled"`
	VectorEnabled    bool `yaml:"vector_enabled" json:"vector_enabled"`

	// 检索策略
	Strategy RetrievalStrategy `yaml:"strategy" json:"strategy"`

	// 性能配置
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// 存储引擎配置
	StorageEngines StorageEnginesConfig `yaml:"storage_engines" json:"storage_engines"`
}

// RetrievalStrategy 检索策略配置
type RetrievalStrategy struct {
	// 维度权重
	TimelineWeight  float64 `yaml:"timeline_weight" json:"timeline_weight"`
	KnowledgeWeight float64 `yaml:"knowledge_weight" json:"knowledge_weight"`
	VectorWeight    float64 `yaml:"vector_weight" json:"vector_weight"`

	// 并行策略
	EnableParallel bool          `yaml:"enable_parallel" json:"enable_parallel"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`

	// 结果融合
	FusionMethod string  `yaml:"fusion_method" json:"fusion_method"` // "weighted", "rank_fusion", "score_fusion"
	MaxResults   int     `yaml:"max_results" json:"max_results"`
	MinRelevance float64 `yaml:"min_relevance" json:"min_relevance"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// 并发控制
	MaxConcurrentQueries int           `yaml:"max_concurrent_queries" json:"max_concurrent_queries"`
	QueryTimeout         time.Duration `yaml:"query_timeout" json:"query_timeout"`

	// 缓存配置
	EnableCache bool          `yaml:"enable_cache" json:"enable_cache"`
	CacheTTL    time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	CacheSize   int           `yaml:"cache_size" json:"cache_size"`

	// 限流配置
	RateLimit int `yaml:"rate_limit" json:"rate_limit"` // requests per minute
}

// StorageEnginesConfig 存储引擎配置
type StorageEnginesConfig struct {
	// TimescaleDB配置
	TimescaleDB TimescaleDBConfig `yaml:"timescaledb" json:"timescaledb"`

	// Neo4j配置
	Neo4j Neo4jConfig `yaml:"neo4j" json:"neo4j"`

	// 向量存储配置（复用现有配置）
	Vector VectorConfig `yaml:"vector" json:"vector"`
}

// TimescaleDBConfig TimescaleDB配置
type TimescaleDBConfig struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	Host        string        `yaml:"host" json:"host"`
	Port        int           `yaml:"port" json:"port"`
	Database    string        `yaml:"database" json:"database"`
	Username    string        `yaml:"username" json:"username"`
	Password    string        `yaml:"password" json:"password"`
	SSLMode     string        `yaml:"ssl_mode" json:"ssl_mode"`
	MaxConns    int           `yaml:"max_conns" json:"max_conns"`
	MaxIdleTime time.Duration `yaml:"max_idle_time" json:"max_idle_time"`
}

// Neo4jConfig Neo4j配置
type Neo4jConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	URI      string `yaml:"uri" json:"uri"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`

	// 连接池配置
	MaxConnectionPoolSize int           `yaml:"max_connection_pool_size" json:"max_connection_pool_size"`
	ConnectionTimeout     time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
}

// VectorConfig 向量存储配置（复用现有）
type VectorConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// 这里复用现有的向量存储配置
	// 不修改现有配置结构
}

// DefaultConfig 默认配置
func DefaultConfig() *MultiDimensionalRetrievalConfig {
	return &MultiDimensionalRetrievalConfig{
		Enabled: false, // 默认关闭，需要手动开启

		TimelineEnabled:  false, // 分步启用
		KnowledgeEnabled: false,
		VectorEnabled:    true, // 向量检索默认启用（复用现有）

		Strategy: RetrievalStrategy{
			TimelineWeight:  0.3,
			KnowledgeWeight: 0.3,
			VectorWeight:    0.4,
			EnableParallel:  true,
			Timeout:         30 * time.Second,
			FusionMethod:    "weighted",
			MaxResults:      50,
			MinRelevance:    0.5,
		},

		Performance: PerformanceConfig{
			MaxConcurrentQueries: 10,
			QueryTimeout:         10 * time.Second,
			EnableCache:          true,
			CacheTTL:             5 * time.Minute,
			CacheSize:            1000,
			RateLimit:            100,
		},

		StorageEngines: StorageEnginesConfig{
			TimescaleDB: TimescaleDBConfig{
				Enabled:     false, // 默认关闭
				Host:        "localhost",
				Port:        5432,
				Database:    "context_keeper_timeline",
				Username:    "postgres",
				Password:    "",
				SSLMode:     "disable",
				MaxConns:    10,
				MaxIdleTime: 5 * time.Minute,
			},

			Neo4j: Neo4jConfig{
				Enabled:               false, // 默认关闭
				URI:                   "bolt://localhost:7687",
				Username:              "neo4j",
				Password:              "",
				Database:              "neo4j",
				MaxConnectionPoolSize: 10,
				ConnectionTimeout:     5 * time.Second,
			},

			Vector: VectorConfig{
				Enabled: true, // 复用现有向量存储
			},
		},
	}
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*MultiDimensionalRetrievalConfig, error) {
	// 如果配置文件不存在，返回默认配置
	config := DefaultConfig()

	// TODO: 从配置文件加载配置
	// 这里可以扩展配置文件加载逻辑

	return config, nil
}

// IsEnabled 检查是否启用多维度检索
func (c *MultiDimensionalRetrievalConfig) IsEnabled() bool {
	return c.Enabled
}

// GetEnabledEngines 获取启用的存储引擎
func (c *MultiDimensionalRetrievalConfig) GetEnabledEngines() []string {
	engines := []string{}

	if c.TimelineEnabled && c.StorageEngines.TimescaleDB.Enabled {
		engines = append(engines, "timeline")
	}

	if c.KnowledgeEnabled && c.StorageEngines.Neo4j.Enabled {
		engines = append(engines, "knowledge")
	}

	if c.VectorEnabled && c.StorageEngines.Vector.Enabled {
		engines = append(engines, "vector")
	}

	return engines
}

// Validate 验证配置
func (c *MultiDimensionalRetrievalConfig) Validate() error {
	// 验证权重总和
	totalWeight := c.Strategy.TimelineWeight + c.Strategy.KnowledgeWeight + c.Strategy.VectorWeight
	if totalWeight <= 0 {
		return fmt.Errorf("策略权重总和必须大于0")
	}

	// 验证至少启用一个存储引擎
	enabledEngines := c.GetEnabledEngines()
	if len(enabledEngines) == 0 {
		return fmt.Errorf("至少需要启用一个存储引擎")
	}

	return nil
}
