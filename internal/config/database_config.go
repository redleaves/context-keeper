package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// 时间线存储配置 (TimescaleDB/PostgreSQL)
	TimescaleDB TimescaleDBConfig `json:"timescaledb"`

	// 知识图谱存储配置 (Neo4j)
	Neo4j Neo4jConfig `json:"neo4j"`

	// 向量存储配置
	Vector VectorConfig `json:"vector"`
}

// TimescaleDBConfig TimescaleDB配置
type TimescaleDBConfig struct {
	Enabled     bool          `json:"enabled"`
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	Database    string        `json:"database"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	SSLMode     string        `json:"ssl_mode"`
	MaxConns    int           `json:"max_conns"`
	MaxIdleTime time.Duration `json:"max_idle_time"`
}

// Neo4jConfig Neo4j配置
type Neo4jConfig struct {
	Enabled                 bool          `json:"enabled"`
	URI                     string        `json:"uri"`
	Username                string        `json:"username"`
	Password                string        `json:"password"`
	Database                string        `json:"database"`
	MaxConnectionPoolSize   int           `json:"max_connection_pool_size"`
	ConnectionTimeout       time.Duration `json:"connection_timeout"`
	MaxTransactionRetryTime time.Duration `json:"max_transaction_retry_time"`
}

// VectorConfig 向量存储配置
type VectorConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	// 具体的向量存储配置由现有的向量存储配置文件管理
}

// LoadDatabaseConfig 加载数据库配置
func LoadDatabaseConfig() (*DatabaseConfig, error) {
	// 加载.env文件（必须存在）
	if err := godotenv.Load("config/.env"); err != nil {
		return nil, fmt.Errorf("❌ 配置文件 config/.env 不存在或加载失败: %w", err)
	}

	// 辅助函数：从环境变量获取必需的字符串值
	getRequiredEnv := func(key string) (string, error) {
		if value := os.Getenv(key); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("❌ 必需的环境变量 %s 未设置或为空", key)
	}

	// 辅助函数：从环境变量获取可选的字符串值
	getOptionalEnv := func(key string) string {
		return os.Getenv(key)
	}

	// 辅助函数：从环境变量获取布尔值
	getRequiredBoolEnv := func(key string) (bool, error) {
		value := os.Getenv(key)
		if value == "" {
			return false, fmt.Errorf("❌ 必需的环境变量 %s 未设置", key)
		}
		return getEnvAsBool(key, false), nil
	}

	// 辅助函数：从环境变量获取整数值
	getRequiredIntEnv := func(key string) (int, error) {
		value := os.Getenv(key)
		if value == "" {
			return 0, fmt.Errorf("❌ 必需的环境变量 %s 未设置", key)
		}
		return getEnvAsInt(key, 0), nil
	}

	// 辅助函数：从环境变量获取时间间隔值
	getRequiredDurationEnv := func(key string) (time.Duration, error) {
		value := os.Getenv(key)
		if value == "" {
			return 0, fmt.Errorf("❌ 必需的环境变量 %s 未设置", key)
		}
		return getEnvAsDuration(key, 0), nil
	}

	// 加载TimescaleDB配置
	timelineEnabled, err := getRequiredBoolEnv("TIMELINE_STORAGE_ENABLED")
	if err != nil {
		return nil, err
	}

	var timescaleConfig TimescaleDBConfig
	if timelineEnabled {
		host, err := getRequiredEnv("TIMESCALEDB_HOST")
		if err != nil {
			return nil, err
		}

		port, err := getRequiredIntEnv("TIMESCALEDB_PORT")
		if err != nil {
			return nil, err
		}

		database, err := getRequiredEnv("TIMESCALEDB_DATABASE")
		if err != nil {
			return nil, err
		}

		username, err := getRequiredEnv("TIMESCALEDB_USERNAME")
		if err != nil {
			return nil, err
		}

		sslMode, err := getRequiredEnv("TIMESCALEDB_SSL_MODE")
		if err != nil {
			return nil, err
		}

		maxConns, err := getRequiredIntEnv("TIMESCALEDB_MAX_CONNS")
		if err != nil {
			return nil, err
		}

		maxIdleTime, err := getRequiredDurationEnv("TIMESCALEDB_MAX_IDLE_TIME")
		if err != nil {
			return nil, err
		}

		timescaleConfig = TimescaleDBConfig{
			Enabled:     timelineEnabled,
			Host:        host,
			Port:        port,
			Database:    database,
			Username:    username,
			Password:    getOptionalEnv("TIMESCALEDB_PASSWORD"), // 密码可以为空
			SSLMode:     sslMode,
			MaxConns:    maxConns,
			MaxIdleTime: maxIdleTime,
		}
	} else {
		timescaleConfig = TimescaleDBConfig{Enabled: false}
	}

	// 加载Neo4j配置
	knowledgeEnabled, err := getRequiredBoolEnv("KNOWLEDGE_GRAPH_ENABLED")
	if err != nil {
		return nil, err
	}

	var neo4jConfig Neo4jConfig
	if knowledgeEnabled {
		uri, err := getRequiredEnv("NEO4J_URI")
		if err != nil {
			return nil, err
		}

		username, err := getRequiredEnv("NEO4J_USERNAME")
		if err != nil {
			return nil, err
		}

		password, err := getRequiredEnv("NEO4J_PASSWORD")
		if err != nil {
			return nil, err
		}

		database, err := getRequiredEnv("NEO4J_DATABASE")
		if err != nil {
			return nil, err
		}

		maxPoolSize, err := getRequiredIntEnv("NEO4J_MAX_CONNECTION_POOL_SIZE")
		if err != nil {
			return nil, err
		}

		connTimeout, err := getRequiredDurationEnv("NEO4J_CONNECTION_TIMEOUT")
		if err != nil {
			return nil, err
		}

		retryTime, err := getRequiredDurationEnv("NEO4J_MAX_TRANSACTION_RETRY_TIME")
		if err != nil {
			return nil, err
		}

		neo4jConfig = Neo4jConfig{
			Enabled:                 knowledgeEnabled,
			URI:                     uri,
			Username:                username,
			Password:                password,
			Database:                database,
			MaxConnectionPoolSize:   maxPoolSize,
			ConnectionTimeout:       connTimeout,
			MaxTransactionRetryTime: retryTime,
		}
	} else {
		neo4jConfig = Neo4jConfig{Enabled: false}
	}

	// 加载向量存储配置
	vectorEnabled, err := getRequiredBoolEnv("MULTI_DIM_VECTOR_ENABLED")
	if err != nil {
		return nil, err
	}

	vectorType, err := getRequiredEnv("VECTOR_STORE_TYPE")
	if err != nil {
		return nil, err
	}

	config := &DatabaseConfig{
		TimescaleDB: timescaleConfig,
		Neo4j:       neo4jConfig,
		Vector: VectorConfig{
			Enabled: vectorEnabled,
			Type:    vectorType,
		},
	}

	return config, nil
}

// 使用config包中已有的辅助函数

// GetTimescaleDBConnectionString 获取TimescaleDB连接字符串
func (c *TimescaleDBConfig) GetConnectionString() string {
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Database, c.SSLMode)

	if c.Password != "" {
		connStr += fmt.Sprintf(" password=%s", c.Password)
	}

	return connStr
}

// Validate 验证配置
func (c *DatabaseConfig) Validate() error {
	if c.TimescaleDB.Enabled {
		if c.TimescaleDB.Host == "" || c.TimescaleDB.Database == "" || c.TimescaleDB.Username == "" {
			return fmt.Errorf("TimescaleDB配置不完整: host=%s, database=%s, username=%s",
				c.TimescaleDB.Host, c.TimescaleDB.Database, c.TimescaleDB.Username)
		}
	}

	if c.Neo4j.Enabled {
		if c.Neo4j.URI == "" || c.Neo4j.Username == "" || c.Neo4j.Password == "" {
			return fmt.Errorf("Neo4j配置不完整: uri=%s, username=%s, password=%s",
				c.Neo4j.URI, c.Neo4j.Username, "***")
		}
	}

	return nil
}

// PrintConfig 打印配置信息（隐藏敏感信息）
func (c *DatabaseConfig) PrintConfig() {
	fmt.Printf("📊 [数据库配置] 配置加载完成:\n")

	if c.TimescaleDB.Enabled {
		fmt.Printf("  🕒 TimescaleDB: %s:%d/%s (用户: %s)\n",
			c.TimescaleDB.Host, c.TimescaleDB.Port, c.TimescaleDB.Database, c.TimescaleDB.Username)
	} else {
		fmt.Printf("  🕒 TimescaleDB: 已禁用\n")
	}

	if c.Neo4j.Enabled {
		fmt.Printf("  🕸️ Neo4j: %s/%s (用户: %s)\n",
			c.Neo4j.URI, c.Neo4j.Database, c.Neo4j.Username)
	} else {
		fmt.Printf("  🕸️ Neo4j: 已禁用\n")
	}

	if c.Vector.Enabled {
		fmt.Printf("  🔍 向量存储: %s (已启用)\n", c.Vector.Type)
	} else {
		fmt.Printf("  🔍 向量存储: 已禁用\n")
	}
}
