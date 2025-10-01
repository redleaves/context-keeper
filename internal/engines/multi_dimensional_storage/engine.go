package multi_dimensional_storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// storageTask 存储任务
type storageTask struct {
	name   string
	result chan storageTaskResult
}

// storageTaskResult 存储任务结果
type storageTaskResult struct {
	success bool
	id      string
	err     error
}

// MultiDimensionalStorageEngine 多维度存储引擎
type MultiDimensionalStorageEngine struct {
	config           *MultiDimensionalStorageConfig
	llmAnalyzer      LLMAnalyzer
	timelineAdapter  TimelineStorageAdapter
	knowledgeAdapter KnowledgeStorageAdapter
	vectorAdapter    VectorStorageAdapter
	qualityValidator QualityValidator
	metrics          *StorageMetrics
	mu               sync.RWMutex
}

// NewMultiDimensionalStorageEngine 创建多维度存储引擎
func NewMultiDimensionalStorageEngine(
	config *MultiDimensionalStorageConfig,
	llmAnalyzer LLMAnalyzer,
	timelineAdapter TimelineStorageAdapter,
	knowledgeAdapter KnowledgeStorageAdapter,
	vectorAdapter VectorStorageAdapter,
) (*MultiDimensionalStorageEngine, error) {

	if config == nil {
		config = &MultiDimensionalStorageConfig{
			Enabled:          true,
			TimelineEnabled:  true,
			KnowledgeEnabled: true,
			VectorEnabled:    true,
			FallbackToLegacy: true,
			LLMProvider:      "deepseek",
			LLMModel:         "deepseek-chat",
			MaxRetries:       3,
			Timeout:          60 * time.Second,
		}
	}

	engine := &MultiDimensionalStorageEngine{
		config:           config,
		llmAnalyzer:      llmAnalyzer,
		timelineAdapter:  timelineAdapter,
		knowledgeAdapter: knowledgeAdapter,
		vectorAdapter:    vectorAdapter,
		qualityValidator: NewQualityValidator(),
		metrics: &StorageMetrics{
			TotalRequests:       0,
			SuccessfulStores:    0,
			FailedStores:        0,
			AverageProcessTime:  0,
			TimelineStoreCount:  0,
			KnowledgeStoreCount: 0,
			VectorStoreCount:    0,
			LLMAnalysisErrors:   0,
			StorageErrors:       0,
		},
	}

	log.Printf("✅ 多维度存储引擎初始化完成")
	log.Printf("   配置: 时间线=%v, 知识图谱=%v, 向量=%v",
		config.TimelineEnabled, config.KnowledgeEnabled, config.VectorEnabled)

	return engine, nil
}

// Store 存储数据
func (e *MultiDimensionalStorageEngine) Store(request *StorageRequest, analysisResult *LLMAnalysisResult) (*StorageResult, error) {
	startTime := time.Now()

	// 更新指标
	e.mu.Lock()
	e.metrics.TotalRequests++
	e.mu.Unlock()

	log.Printf("🚀 开始多维度存储 - 用户: %s, 会话: %s", request.UserID, request.SessionID)

	result := &StorageResult{
		Success:         true,
		TimelineStored:  false,
		KnowledgeStored: false,
		VectorStored:    false,
		StoredIDs:       make(map[string]string),
		Errors:          []string{},
		ProcessingTime:  0,
		LLMAnalysisTime: 0,
		StorageTime:     0,
		Metadata:        make(map[string]interface{}),
	}

	// 1. LLM分析（如果没有提供分析结果）
	if analysisResult == nil {
		log.Printf("🔍 开始LLM分析...")
		llmStartTime := time.Now()

		var err error
		analysisResult, err = e.llmAnalyzer.Analyze(request)
		if err != nil {
			e.mu.Lock()
			e.metrics.LLMAnalysisErrors++
			e.metrics.FailedStores++
			e.mu.Unlock()

			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("LLM分析失败: %v", err))

			if e.config.FallbackToLegacy {
				log.Printf("⚠️ LLM分析失败，回退到传统存储逻辑")
				return result, nil // 让调用方处理回退逻辑
			}

			return result, err
		}

		result.LLMAnalysisTime = time.Since(llmStartTime)
		log.Printf("✅ LLM分析完成 - 耗时: %v", result.LLMAnalysisTime)
	}

	// 2. 并行存储到不同引擎
	storageStartTime := time.Now()

	// 使用channel收集存储结果
	var tasks []storageTask

	// 2.1 时间线存储
	if e.config.TimelineEnabled && analysisResult.TimelineData != nil &&
		analysisResult.StorageRecommendation.TimelinePriority > 0.3 {

		task := storageTask{
			name:   "timeline",
			result: make(chan storageTaskResult, 1),
		}
		tasks = append(tasks, task)

		go func() {
			defer close(task.result)

			// 质量验证
			valid, errors := e.qualityValidator.ValidateTimelineData(analysisResult.TimelineData)
			if !valid {
				task.result <- storageTaskResult{
					success: false,
					err:     fmt.Errorf("时间线数据质量验证失败: %v", errors),
				}
				return
			}

			// 存储
			id, err := e.timelineAdapter.StoreTimelineData(request.UserID, request.SessionID, analysisResult.TimelineData)
			task.result <- storageTaskResult{
				success: err == nil,
				id:      id,
				err:     err,
			}
		}()
	}

	// 2.2 知识图谱存储
	if e.config.KnowledgeEnabled && analysisResult.KnowledgeGraphData != nil &&
		analysisResult.StorageRecommendation.KnowledgePriority > 0.3 {

		task := storageTask{
			name:   "knowledge",
			result: make(chan storageTaskResult, 1),
		}
		tasks = append(tasks, task)

		go func() {
			defer close(task.result)

			// 质量验证
			valid, errors := e.qualityValidator.ValidateKnowledgeData(analysisResult.KnowledgeGraphData)
			if !valid {
				task.result <- storageTaskResult{
					success: false,
					err:     fmt.Errorf("知识图谱数据质量验证失败: %v", errors),
				}
				return
			}

			// 存储
			id, err := e.knowledgeAdapter.StoreKnowledgeData(request.UserID, request.SessionID, analysisResult.KnowledgeGraphData)
			task.result <- storageTaskResult{
				success: err == nil,
				id:      id,
				err:     err,
			}
		}()
	}

	// 2.3 向量存储
	if e.config.VectorEnabled && analysisResult.VectorData != nil &&
		analysisResult.StorageRecommendation.VectorPriority > 0.3 {

		task := storageTask{
			name:   "vector",
			result: make(chan storageTaskResult, 1),
		}
		tasks = append(tasks, task)

		go func() {
			defer close(task.result)

			// 质量验证
			valid, errors := e.qualityValidator.ValidateVectorData(analysisResult.VectorData)
			if !valid {
				task.result <- storageTaskResult{
					success: false,
					err:     fmt.Errorf("向量数据质量验证失败: %v", errors),
				}
				return
			}

			// 存储
			id, err := e.vectorAdapter.StoreVectorData(request.UserID, request.SessionID, analysisResult.VectorData)
			task.result <- storageTaskResult{
				success: err == nil,
				id:      id,
				err:     err,
			}
		}()
	}

	// 3. 收集存储结果
	ctx, cancel := context.WithTimeout(context.Background(), e.config.Timeout)
	defer cancel()

	for _, task := range tasks {
		select {
		case taskResult := <-task.result:
			if taskResult.success {
				result.StoredIDs[task.name] = taskResult.id

				switch task.name {
				case "timeline":
					result.TimelineStored = true
					e.mu.Lock()
					e.metrics.TimelineStoreCount++
					e.mu.Unlock()
				case "knowledge":
					result.KnowledgeStored = true
					e.mu.Lock()
					e.metrics.KnowledgeStoreCount++
					e.mu.Unlock()
				case "vector":
					result.VectorStored = true
					e.mu.Lock()
					e.metrics.VectorStoreCount++
					e.mu.Unlock()
				}

				log.Printf("✅ %s存储成功 - ID: %s", task.name, taskResult.id)
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s存储失败: %v", task.name, taskResult.err))
				log.Printf("❌ %s存储失败: %v", task.name, taskResult.err)
			}

		case <-ctx.Done():
			result.Errors = append(result.Errors, fmt.Sprintf("%s存储超时", task.name))
			log.Printf("⚠️ %s存储超时", task.name)
		}
	}

	result.StorageTime = time.Since(storageStartTime)
	result.ProcessingTime = time.Since(startTime)

	// 4. 更新指标
	e.mu.Lock()
	if len(result.Errors) == 0 {
		e.metrics.SuccessfulStores++
	} else {
		e.metrics.FailedStores++
		e.metrics.StorageErrors++
	}

	// 更新平均处理时间
	totalTime := e.metrics.AverageProcessTime * time.Duration(e.metrics.TotalRequests-1)
	e.metrics.AverageProcessTime = (totalTime + result.ProcessingTime) / time.Duration(e.metrics.TotalRequests)
	e.mu.Unlock()

	// 5. 记录结果
	successCount := 0
	if result.TimelineStored {
		successCount++
	}
	if result.KnowledgeStored {
		successCount++
	}
	if result.VectorStored {
		successCount++
	}

	log.Printf("🎉 多维度存储完成 - 成功: %d/%d, 总耗时: %v",
		successCount, len(tasks), result.ProcessingTime)

	if len(result.Errors) > 0 {
		log.Printf("⚠️ 存储错误: %v", result.Errors)
	}

	return result, nil
}

// IsEnabled 检查是否启用
func (e *MultiDimensionalStorageEngine) IsEnabled() bool {
	return e.config.Enabled
}

// GetConfig 获取配置
func (e *MultiDimensionalStorageEngine) GetConfig() *MultiDimensionalStorageConfig {
	return e.config
}

// GetMetrics 获取指标
func (e *MultiDimensionalStorageEngine) GetMetrics() *StorageMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 返回指标的副本
	return &StorageMetrics{
		TotalRequests:       e.metrics.TotalRequests,
		SuccessfulStores:    e.metrics.SuccessfulStores,
		FailedStores:        e.metrics.FailedStores,
		AverageProcessTime:  e.metrics.AverageProcessTime,
		TimelineStoreCount:  e.metrics.TimelineStoreCount,
		KnowledgeStoreCount: e.metrics.KnowledgeStoreCount,
		VectorStoreCount:    e.metrics.VectorStoreCount,
		LLMAnalysisErrors:   e.metrics.LLMAnalysisErrors,
		StorageErrors:       e.metrics.StorageErrors,
	}
}

// Close 关闭引擎
func (e *MultiDimensionalStorageEngine) Close() error {
	log.Printf("🔒 多维度存储引擎关闭")
	return nil
}
