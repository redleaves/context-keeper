package vectorstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultVearchClient Vearch客户端的默认实现
type DefaultVearchClient struct {
	config     *VearchConfig
	httpClient *http.Client
	baseURL    string
	connected  bool
	apiManager *VearchAPIManager // 统一API管理器
}

// NewDefaultVearchClient 创建新的Vearch客户端
func NewDefaultVearchClient(config *VearchConfig) VearchClient {
	if len(config.Endpoints) == 0 {
		log.Printf("[Vearch客户端] ❌ 错误：未提供Vearch服务器地址")
		panic("Vearch配置错误：必须提供VEARCH_URL环境变量")
	}

	// 确保URL有正确的协议前缀
	baseURL := config.Endpoints[0]
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return &DefaultVearchClient{
		config:     config,
		baseURL:    baseURL,
		apiManager: NewVearchAPIManager(baseURL), // 初始化API管理器
		httpClient: &http.Client{
			Timeout: time.Duration(config.RequestTimeoutSeconds) * time.Second,
			// 禁用keep-alive连接池来避免连接问题
			Transport: &http.Transport{
				DisableKeepAlives:     false, // 保持keep-alive但添加更多控制
				MaxIdleConns:          10,
				MaxIdleConnsPerHost:   5,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

// Connect 连接到Vearch集群
func (c *DefaultVearchClient) Connect() error {
	log.Printf("[Vearch客户端] 连接集群: %s", c.baseURL)

	// 测试连接
	if err := c.Ping(); err != nil {
		return fmt.Errorf("连接测试失败: %v", err)
	}

	c.connected = true
	return nil
}

// Close 关闭连接
func (c *DefaultVearchClient) Close() error {
	c.connected = false
	return nil
}

// Ping 测试连接
func (c *DefaultVearchClient) Ping() error {
	url := c.apiManager.GetClusterInfo()
	return c.makeRequest("GET", url, nil, nil)
}

// CreateDatabase 创建数据库（✅ 严格按官方文档）
func (c *DefaultVearchClient) CreateDatabase(name string) error {
	log.Printf("[Vearch客户端] 创建数据库: %s", name)

	// ✅ 使用API管理器获取URL
	url := c.apiManager.CreateDatabase(name)

	// 官方文档显示POST请求不需要payload，直接使用URL中的db_name
	return c.makeRequest("POST", url, nil, nil)
}

// ListDatabases 列出数据库（✅ 严格按官方文档）
func (c *DefaultVearchClient) ListDatabases() ([]string, error) {
	// ✅ 使用API管理器获取URL
	url := c.apiManager.ListDatabases()

	var response struct {
		Code int `json:"code"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := c.makeRequest("GET", url, nil, &response); err != nil {
		return nil, err
	}

	// 提取数据库名称列表
	var dbNames []string
	for _, db := range response.Data {
		dbNames = append(dbNames, db.Name)
	}

	return dbNames, nil
}

// DatabaseExists 检查数据库是否存在（修正：按官方文档规范）
func (c *DefaultVearchClient) DatabaseExists(name string) (bool, error) {
	databases, err := c.ListDatabases()
	if err != nil {
		return false, err
	}

	for _, db := range databases {
		if db == name {
			return true, nil
		}
	}

	return false, nil
}

// CreateSpace 创建空间（✅ 修正：按官方文档规范）
func (c *DefaultVearchClient) CreateSpace(database, name string, config *SpaceConfig) error {
	log.Printf("[Vearch客户端] 创建空间: db=%s, space=%s", database, name)

	// ✅ 使用API管理器获取URL
	url := c.apiManager.CreateSpace(database)

	// ✅ 根据官方文档的正确payload格式
	payload := map[string]interface{}{
		"name":          name,
		"partition_num": config.PartitionNum,
		"replica_num":   config.ReplicaNum,
		"fields":        config.Properties, // 使用fields而不是properties
	}

	// 解析Vearch API响应，检查是否真正创建成功
	var result map[string]interface{}
	if err := c.makeRequest("POST", url, payload, &result); err != nil {
		return err
	}

	// 检查Vearch API响应中的错误码
	if code, ok := result["code"].(float64); ok {
		if code != 0 && code != 200 { // 创建失败
			return fmt.Errorf("Vearch创建空间失败: code=%v, msg=%v", result["code"], result["msg"])
		}
	}

	log.Printf("[Vearch客户端] 空间创建API调用成功: %s", name)
	return nil
}

// ListSpaces 列出空间（✅ 修正：按官方文档规范）
func (c *DefaultVearchClient) ListSpaces(database string) ([]string, error) {
	// ✅ 使用API管理器获取URL
	url := c.apiManager.ListSpaces(database)

	var response map[string]interface{}
	if err := c.makeRequest("GET", url, nil, &response); err != nil {
		return nil, err
	}

	// 解析返回的空间列表（具体格式需要根据实际API响应调整）
	var spaces []string
	if data, ok := response["data"].([]interface{}); ok {
		for _, item := range data {
			if spaceInfo, ok := item.(map[string]interface{}); ok {
				if spaceName, ok := spaceInfo["name"].(string); ok {
					spaces = append(spaces, spaceName)
				}
			}
		}
	}

	return spaces, nil
}

// SpaceExists 检查空间是否存在（✅ 修正：按官方文档规范）
func (c *DefaultVearchClient) SpaceExists(database, name string) (bool, error) {
	// ✅ 使用API管理器获取URL
	url := c.apiManager.GetSpace(database, name)

	var response map[string]interface{}
	err := c.makeRequest("GET", url, nil, &response)

	// 如果404错误，说明空间不存在
	if err != nil && strings.Contains(err.Error(), "状态码: 404") {
		return false, nil
	}

	// 其他错误
	if err != nil {
		return false, err
	}

	// 如果能获取到数据，说明空间存在
	return true, nil
}

// DropSpace 删除空间（✅ 修正：按官方文档规范）
func (c *DefaultVearchClient) DropSpace(database, name string) error {
	log.Printf("[Vearch客户端] 删除空间: db=%s, space=%s", database, name)

	// ✅ 使用API管理器获取URL
	url := c.apiManager.DeleteSpace(database, name)

	return c.makeRequest("DELETE", url, nil, nil)
}

// Insert 插入文档（✅ 正确：使用API Manager和实际工作的格式）
func (c *DefaultVearchClient) Insert(database, space string, docs []map[string]interface{}) error {
	log.Printf("[Vearch客户端] 插入文档: db=%s, space=%s, count=%d", database, space, len(docs))

	// ✅ 使用API Manager获取插入API路径
	url := c.apiManager.InsertDocument(database, space)

	// ✅ 构造实际工作的插入请求格式（需要在payload中传db_name和space_name）
	payload := map[string]interface{}{
		"db_name":    database,
		"space_name": space,
		"documents":  docs,
	}

	log.Printf("[Vearch客户端] 使用插入API: %s", url)
	return c.makeRequest("POST", url, payload, nil)
}

// Search 搜索文档（✅ 修正：严格按照官方文档格式）
func (c *DefaultVearchClient) Search(database, space string, query *VearchSearchRequest) (*VearchSearchResponse, error) {
	log.Printf("[Vearch客户端] 搜索文档: db=%s, space=%s, limit=%d", database, space, query.Limit)

	// 检查向量数据
	if len(query.Vectors) == 0 || len(query.Vectors[0].Feature) == 0 {
		return nil, fmt.Errorf("向量数据为空，无法执行搜索")
	}

	// ✅ 使用API Manager获取搜索API路径
	url := c.apiManager.SearchDocuments(database, space)

	// ✅ 直接使用VearchSearchRequest结构，因为它现在符合官方文档格式
	// 确保db_name和space_name字段正确设置
	query.DbName = database
	query.SpaceName = space

	// 设置默认的index_params（如果没有设置）
	if query.IndexParams == nil {
		query.IndexParams = map[string]interface{}{
			"metric_type": "InnerProduct", // ✅ 官方文档：使用index_params
		}
	}

	log.Printf("[Vearch客户端] 使用搜索API: %s", url)
	log.Printf("[Vearch客户端] 请求格式: 严格按照官方文档")

	var response VearchSearchResponse
	err := c.makeRequest("POST", url, query, &response)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %v", err)
	}

	return &response, nil
}

// Delete 删除文档（✅ 正确：使用API Manager和实际工作的格式）
func (c *DefaultVearchClient) Delete(database, space string, ids []string) error {
	log.Printf("[Vearch客户端] 删除文档: db=%s, space=%s, count=%d", database, space, len(ids))

	// ✅ 使用API Manager获取删除API路径
	url := c.apiManager.DeleteDocuments(database, space)

	// ✅ 构造实际工作的删除请求格式（需要在payload中传db_name和space_name）
	payload := map[string]interface{}{
		"db_name":      database,
		"space_name":   space,
		"document_ids": ids,
	}

	// ✅ 使用POST方法（document/delete API使用POST而不是DELETE）
	err := c.makeRequest("POST", url, payload, nil)
	if err != nil {
		return fmt.Errorf("删除文档失败: %v", err)
	}

	log.Printf("[Vearch客户端] 成功删除%d个文档", len(ids))
	return nil
}

// BulkIndex 批量索引向量（修正：按官方文档规范）
func (c *DefaultVearchClient) BulkIndex(database, space string, vectors []VearchBulkVector) error {
	log.Printf("[Vearch客户端] 批量索引: db=%s, space=%s, count=%d", database, space, len(vectors))

	// 转换为官方文档格式的文档
	docs := make([]map[string]interface{}, len(vectors))
	for i, vector := range vectors {
		doc := map[string]interface{}{
			"_id": vector.ID,
		}

		// 添加向量字段
		if len(vector.Vector) > 0 {
			doc["vector"] = vector.Vector
		}

		// 添加其他字段
		for k, v := range vector.Fields {
			doc[k] = v
		}

		docs[i] = doc
	}

	// 使用Insert方法，按官方文档格式
	return c.Insert(database, space, docs)
}

// makeRequest 发送HTTP请求的通用方法（带重试逻辑）
func (c *DefaultVearchClient) makeRequest(method, url string, payload interface{}, result interface{}) error {
	// 502错误重试配置
	maxRetries := 3
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.doRequest(method, url, payload, result, attempt)

		// 如果成功或者非502错误，直接返回
		if err == nil {
			return nil
		}

		// 检查是否是502错误且还有重试次数
		if attempt < maxRetries && isRetryableError(err) {
			delay := time.Duration(attempt+1) * baseDelay
			log.Printf("[HTTP请求] ⚠️ 收到502错误，%v后重试 (尝试 %d/%d): %v",
				delay, attempt+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		// 最终失败
		return err
	}

	return fmt.Errorf("重试失败")
}

// doRequest 执行单次HTTP请求
func (c *DefaultVearchClient) doRequest(method, url string, payload interface{}, result interface{}, attempt int) error {
	var body io.Reader
	var requestBody []byte

	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化请求数据失败: %v", err)
		}
		requestBody = jsonData
		body = bytes.NewBuffer(jsonData)
	}

	// 🔍 打印请求详情
	log.Printf("[HTTP请求] ==== 开始请求 ====")
	log.Printf("[HTTP请求] 方法: %s", method)
	log.Printf("[HTTP请求] URL: %s", url)
	log.Printf("[HTTP请求] 认证: %s:%s", c.config.Username, "***")
	if requestBody != nil {
		log.Printf("[HTTP请求] 请求体: %s", string(requestBody))
	} else {
		log.Printf("[HTTP请求] 请求体: 无")
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Printf("[HTTP请求] ❌ 创建请求失败: %v", err)
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头 - 添加更多标准HTTP头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "context-keeper-vearch-client/1.0")
	req.Header.Set("Connection", "keep-alive")

	// 如果有请求体，设置Content-Length
	if requestBody != nil {
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(requestBody)))
	}

	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[HTTP请求] ❌ 请求失败: %v", err)
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[HTTP请求] ❌ 读取响应失败: %v", err)
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 🔍 打印响应详情
	log.Printf("[HTTP响应] 状态码: %d", resp.StatusCode)
	log.Printf("[HTTP响应] 响应头: %v", resp.Header)
	log.Printf("[HTTP响应] 响应体: %s", string(respBody))
	log.Printf("[HTTP响应] ==== 请求完成 ====")

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析结果
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败: %v", err)
		}
	}

	return nil
}

// CreateVearchStoreFromConfig 从配置创建Vearch存储
// 已废弃：请使用VectorStoreFactory来创建实例
func CreateVearchStoreFromConfig(config *VearchConfig) *VearchStore {
	client := NewDefaultVearchClient(config)
	// 这里传入nil作为embedding服务，将使用默认实现
	return NewVearchStore(client, config, nil)
}

// CreateVearchStoreFromEnv 从环境变量创建Vearch存储
func CreateVearchStoreFromEnv() (*VearchStore, error) {
	vearchURL := getEnvOrDefault("VEARCH_URL", "")
	if vearchURL == "" {
		return nil, fmt.Errorf("VEARCH_URL环境变量未设置，请提供Vearch服务器地址")
	}

	config := &VearchConfig{
		Endpoints:             []string{vearchURL},
		Username:              getEnvOrDefault("VEARCH_USERNAME", ""),
		Password:              getEnvOrDefault("VEARCH_PASSWORD", ""),
		Database:              getEnvOrDefault("VEARCH_DATABASE", "context_keeper_vector"),
		EmbeddingModel:        getEnvOrDefault("VEARCH_EMBEDDING_MODEL", "text-embedding-ada-002"),
		EmbeddingEndpoint:     getEnvOrDefault("VEARCH_EMBEDDING_ENDPOINT", ""),
		EmbeddingAPIKey:       getEnvOrDefault("VEARCH_EMBEDDING_API_KEY", ""),
		Dimension:             getEnvIntOrDefault("VEARCH_DIMENSION", 1536),
		DefaultTopK:           getEnvIntOrDefault("VEARCH_DEFAULT_TOP_K", 10),
		SimilarityThreshold:   getEnvFloatOrDefault("VEARCH_SIMILARITY_THRESHOLD", 0.7),
		SearchTimeoutSeconds:  getEnvIntOrDefault("VEARCH_SEARCH_TIMEOUT", 30),
		ConnectionPoolSize:    getEnvIntOrDefault("VEARCH_CONNECTION_POOL_SIZE", 10),
		RequestTimeoutSeconds: getEnvIntOrDefault("VEARCH_REQUEST_TIMEOUT", 30),
	}

	return CreateVearchStoreFromConfig(config), nil
}

// 辅助函数用于从环境变量获取配置
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// isRetryableError 检查错误是否可重试（502/503/504等网关错误）
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 检查是否是502、503、504等可重试的HTTP错误
	return strings.Contains(errStr, "状态码: 502") ||
		strings.Contains(errStr, "状态码: 503") ||
		strings.Contains(errStr, "状态码: 504") ||
		strings.Contains(errStr, "Bad Gateway") ||
		strings.Contains(errStr, "Service Unavailable") ||
		strings.Contains(errStr, "Gateway Timeout")
}
