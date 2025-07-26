package vectorstore

import (
	"fmt"
	"strings"
)

// VearchAPIManager 统一管理所有Vearch API的URL操作
// 根据官方文档 https://vearch.readthedocs.io/zh-cn/latest/use_op/op_db.html
// 和常见REST API模式构建完整的API路径管理器
type VearchAPIManager struct {
	baseURL string
}

// NewVearchAPIManager 创建API管理器
func NewVearchAPIManager(baseURL string) *VearchAPIManager {
	// 确保URL有正确的协议前缀
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	// 移除末尾的斜杠
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &VearchAPIManager{
		baseURL: baseURL,
	}
}

// ========== 集群操作 API ==========

// GetClusterInfo 获取集群信息
// GET /
func (api *VearchAPIManager) GetClusterInfo() string {
	return api.baseURL
}

// GetClusterHealth 获取集群健康状态
// GET /cluster/health
func (api *VearchAPIManager) GetClusterHealth() string {
	return fmt.Sprintf("%s/cluster/health", api.baseURL)
}

// GetClusterStats 获取集群统计信息
// GET /cluster/stats
func (api *VearchAPIManager) GetClusterStats() string {
	return fmt.Sprintf("%s/cluster/stats", api.baseURL)
}

// ========== 数据库操作 API ==========
// 根据官方文档 https://vearch.readthedocs.io/zh-cn/latest/use_op/op_db.html

// ListDatabases 查看集群中所有库
// GET /dbs
func (api *VearchAPIManager) ListDatabases() string {
	return fmt.Sprintf("%s/dbs", api.baseURL)
}

// CreateDatabase 创建库
// POST /dbs/$db_name
func (api *VearchAPIManager) CreateDatabase(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s", api.baseURL, dbName)
}

// GetDatabase 查看指定库
// GET /dbs/$db_name
func (api *VearchAPIManager) GetDatabase(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s", api.baseURL, dbName)
}

// DeleteDatabase 删除库
// DELETE /dbs/$db_name
func (api *VearchAPIManager) DeleteDatabase(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s", api.baseURL, dbName)
}

// ========== 表空间操作 API ==========

// ListSpaces 查看指定库下所有表空间
// GET /dbs/$db_name/spaces
func (api *VearchAPIManager) ListSpaces(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces", api.baseURL, dbName)
}

// CreateSpace 创建表空间
// POST /dbs/$db_name/spaces
func (api *VearchAPIManager) CreateSpace(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces", api.baseURL, dbName)
}

// GetSpace 查看指定表空间
// GET /dbs/$db_name/spaces/$space_name
func (api *VearchAPIManager) GetSpace(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s", api.baseURL, dbName, spaceName)
}

// UpdateSpace 更新表空间配置
// PUT /dbs/$db_name/spaces/$space_name
func (api *VearchAPIManager) UpdateSpace(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s", api.baseURL, dbName, spaceName)
}

// DeleteSpace 删除表空间
// DELETE /dbs/$db_name/spaces/$space_name
func (api *VearchAPIManager) DeleteSpace(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s", api.baseURL, dbName, spaceName)
}

// GetSpaceStats 获取表空间统计信息
// GET /dbs/$db_name/spaces/$space_name/_stats
func (api *VearchAPIManager) GetSpaceStats(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_stats", api.baseURL, dbName, spaceName)
}

// ========== 文档操作 API ==========

// InsertDocument 插入文档
// POST /document/upsert (Vearch实际支持的插入API)
func (api *VearchAPIManager) InsertDocument(dbName, spaceName string) string {
	// 注意：此API使用通用路径，db_name和space_name需要在请求payload中指定
	return fmt.Sprintf("%s/document/upsert", api.baseURL)
}

// GetDocument 获取单个文档
// GET /dbs/$db_name/spaces/$space_name/documents/$doc_id
func (api *VearchAPIManager) GetDocument(dbName, spaceName, docID string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/documents/%s", api.baseURL, dbName, spaceName, docID)
}

// UpdateDocument 更新文档
// PUT /dbs/$db_name/spaces/$space_name/documents/$doc_id
func (api *VearchAPIManager) UpdateDocument(dbName, spaceName, docID string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/documents/%s", api.baseURL, dbName, spaceName, docID)
}

// DeleteDocument 删除文档
// DELETE /dbs/$db_name/spaces/$space_name/documents/$doc_id
func (api *VearchAPIManager) DeleteDocument(dbName, spaceName, docID string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/documents/%s", api.baseURL, dbName, spaceName, docID)
}

// DeleteDocuments 删除文档
// POST /document/delete (Vearch实际支持的删除API)
func (api *VearchAPIManager) DeleteDocuments(dbName, spaceName string) string {
	// 注意：此API使用通用路径，db_name和space_name需要在请求payload中指定
	return fmt.Sprintf("%s/document/delete", api.baseURL)
}

// BulkOperation 批量操作
// POST /dbs/$db_name/spaces/$space_name/_bulk
func (api *VearchAPIManager) BulkOperation(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_bulk", api.baseURL, dbName, spaceName)
}

// ========== 搜索操作 API ==========

// SearchDocuments 搜索文档
// POST /document/search (Vearch实际支持的搜索API)
func (api *VearchAPIManager) SearchDocuments(dbName, spaceName string) string {
	// 注意：此API使用通用路径，db_name和space_name需要在请求payload中指定
	return fmt.Sprintf("%s/document/search", api.baseURL)
}

// QueryBySQL SQL查询
// POST /dbs/$db_name/spaces/$space_name/_query
func (api *VearchAPIManager) QueryBySQL(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_query", api.baseURL, dbName, spaceName)
}

// MultiSearch 多空间搜索
// POST /dbs/$db_name/_msearch
func (api *VearchAPIManager) MultiSearch(dbName string) string {
	return fmt.Sprintf("%s/dbs/%s/_msearch", api.baseURL, dbName)
}

// ========== 索引操作 API ==========

// RebuildIndex 重建索引
// POST /dbs/$db_name/spaces/$space_name/_rebuild
func (api *VearchAPIManager) RebuildIndex(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_rebuild", api.baseURL, dbName, spaceName)
}

// FlushIndex 刷新索引
// POST /dbs/$db_name/spaces/$space_name/_flush
func (api *VearchAPIManager) FlushIndex(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_flush", api.baseURL, dbName, spaceName)
}

// OptimizeIndex 优化索引
// POST /dbs/$db_name/spaces/$space_name/_optimize
func (api *VearchAPIManager) OptimizeIndex(dbName, spaceName string) string {
	return fmt.Sprintf("%s/dbs/%s/spaces/%s/_optimize", api.baseURL, dbName, spaceName)
}

// ========== 备用API路径（兼容不同版本） ==========

// LegacyInsertDocument 备用插入API
// POST /document/upsert
func (api *VearchAPIManager) LegacyInsertDocument() string {
	return fmt.Sprintf("%s/document/upsert", api.baseURL)
}

// LegacySearchDocument 备用搜索API
// POST /document/search
func (api *VearchAPIManager) LegacySearchDocument() string {
	return fmt.Sprintf("%s/document/search", api.baseURL)
}

// LegacyDeleteDocument 备用删除API
// POST /document/delete
func (api *VearchAPIManager) LegacyDeleteDocument() string {
	return fmt.Sprintf("%s/document/delete", api.baseURL)
}

// ========== 辅助方法 ==========

// GetBaseURL 获取基础URL
func (api *VearchAPIManager) GetBaseURL() string {
	return api.baseURL
}

// IsValidURL 检查URL是否有效
func (api *VearchAPIManager) IsValidURL() bool {
	return api.baseURL != "" && (strings.HasPrefix(api.baseURL, "http://") || strings.HasPrefix(api.baseURL, "https://"))
}

// ========== API操作类型枚举 ==========

// APIOperation API操作类型
type APIOperation string

const (
	// 集群操作
	OpClusterInfo   APIOperation = "cluster_info"
	OpClusterHealth APIOperation = "cluster_health"
	OpClusterStats  APIOperation = "cluster_stats"

	// 数据库操作
	OpListDatabases  APIOperation = "list_databases"
	OpCreateDatabase APIOperation = "create_database"
	OpGetDatabase    APIOperation = "get_database"
	OpDeleteDatabase APIOperation = "delete_database"

	// 表空间操作
	OpListSpaces    APIOperation = "list_spaces"
	OpCreateSpace   APIOperation = "create_space"
	OpGetSpace      APIOperation = "get_space"
	OpUpdateSpace   APIOperation = "update_space"
	OpDeleteSpace   APIOperation = "delete_space"
	OpGetSpaceStats APIOperation = "get_space_stats"

	// 文档操作
	OpInsertDocument  APIOperation = "insert_document"
	OpGetDocument     APIOperation = "get_document"
	OpUpdateDocument  APIOperation = "update_document"
	OpDeleteDocument  APIOperation = "delete_document"
	OpDeleteDocuments APIOperation = "delete_documents"
	OpBulkOperation   APIOperation = "bulk_operation"

	// 搜索操作
	OpSearchDocuments APIOperation = "search_documents"
	OpQueryBySQL      APIOperation = "query_by_sql"
	OpMultiSearch     APIOperation = "multi_search"

	// 索引操作
	OpRebuildIndex  APIOperation = "rebuild_index"
	OpFlushIndex    APIOperation = "flush_index"
	OpOptimizeIndex APIOperation = "optimize_index"
)

// GetOperationURL 根据操作类型获取URL
func (api *VearchAPIManager) GetOperationURL(operation APIOperation, params ...string) string {
	switch operation {
	// 集群操作
	case OpClusterInfo:
		return api.GetClusterInfo()
	case OpClusterHealth:
		return api.GetClusterHealth()
	case OpClusterStats:
		return api.GetClusterStats()

	// 数据库操作
	case OpListDatabases:
		return api.ListDatabases()
	case OpCreateDatabase:
		if len(params) >= 1 {
			return api.CreateDatabase(params[0])
		}
	case OpGetDatabase:
		if len(params) >= 1 {
			return api.GetDatabase(params[0])
		}
	case OpDeleteDatabase:
		if len(params) >= 1 {
			return api.DeleteDatabase(params[0])
		}

	// 表空间操作
	case OpListSpaces:
		if len(params) >= 1 {
			return api.ListSpaces(params[0])
		}
	case OpCreateSpace:
		if len(params) >= 1 {
			return api.CreateSpace(params[0])
		}
	case OpGetSpace:
		if len(params) >= 2 {
			return api.GetSpace(params[0], params[1])
		}
	case OpUpdateSpace:
		if len(params) >= 2 {
			return api.UpdateSpace(params[0], params[1])
		}
	case OpDeleteSpace:
		if len(params) >= 2 {
			return api.DeleteSpace(params[0], params[1])
		}
	case OpGetSpaceStats:
		if len(params) >= 2 {
			return api.GetSpaceStats(params[0], params[1])
		}

	// 文档操作
	case OpInsertDocument:
		if len(params) >= 2 {
			return api.InsertDocument(params[0], params[1])
		}
	case OpGetDocument:
		if len(params) >= 3 {
			return api.GetDocument(params[0], params[1], params[2])
		}
	case OpUpdateDocument:
		if len(params) >= 3 {
			return api.UpdateDocument(params[0], params[1], params[2])
		}
	case OpDeleteDocument:
		if len(params) >= 3 {
			return api.DeleteDocument(params[0], params[1], params[2])
		}
	case OpDeleteDocuments:
		if len(params) >= 2 {
			return api.DeleteDocuments(params[0], params[1])
		}
	case OpBulkOperation:
		if len(params) >= 2 {
			return api.BulkOperation(params[0], params[1])
		}

	// 搜索操作
	case OpSearchDocuments:
		if len(params) >= 2 {
			return api.SearchDocuments(params[0], params[1])
		}
	case OpQueryBySQL:
		if len(params) >= 2 {
			return api.QueryBySQL(params[0], params[1])
		}
	case OpMultiSearch:
		if len(params) >= 1 {
			return api.MultiSearch(params[0])
		}

	// 索引操作
	case OpRebuildIndex:
		if len(params) >= 2 {
			return api.RebuildIndex(params[0], params[1])
		}
	case OpFlushIndex:
		if len(params) >= 2 {
			return api.FlushIndex(params[0], params[1])
		}
	case OpOptimizeIndex:
		if len(params) >= 2 {
			return api.OptimizeIndex(params[0], params[1])
		}
	}

	return ""
}

// HTTPMethod HTTP方法枚举
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
)

// GetOperationHTTPMethod 获取操作对应的HTTP方法
func GetOperationHTTPMethod(operation APIOperation) HTTPMethod {
	switch operation {
	// GET 操作
	case OpClusterInfo, OpClusterHealth, OpClusterStats,
		OpListDatabases, OpGetDatabase,
		OpListSpaces, OpGetSpace, OpGetSpaceStats,
		OpGetDocument:
		return GET

	// POST 操作
	case OpCreateDatabase, OpCreateSpace, OpInsertDocument, OpBulkOperation,
		OpSearchDocuments, OpQueryBySQL, OpMultiSearch,
		OpRebuildIndex, OpFlushIndex, OpOptimizeIndex:
		return POST

	// PUT 操作
	case OpUpdateSpace, OpUpdateDocument:
		return PUT

	// DELETE 操作
	case OpDeleteDatabase, OpDeleteSpace, OpDeleteDocument, OpDeleteDocuments:
		return DELETE

	default:
		return GET
	}
}
