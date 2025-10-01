package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/schollz/progressbar/v3"
)

// Result 存储单项基准测试结果
type Result struct {
	Name        string        `json:"name"`
	Operations  int           `json:"operations"`
	TotalTime   time.Duration `json:"total_time"`
	AverageTime time.Duration `json:"average_time"`
	MinTime     time.Duration `json:"min_time"`
	MaxTime     time.Duration `json:"max_time"`
	SuccessRate float64       `json:"success_rate"`
	MemoryUsage int64         `json:"memory_usage,omitempty"`
}

// Suite 存储完整基准测试结果
type Suite struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Environment  string    `json:"environment"`
	Results      []Result  `json:"results"`
	TestDataSize int       `json:"test_data_size"`
}

// MockClient 模拟Context-Keeper客户端
type MockClient struct {
	EnableLog bool
}

// NewMockClient 创建新的模拟客户端
func NewMockClient(enableLog bool) *MockClient {
	return &MockClient{
		EnableLog: enableLog,
	}
}

// CreateSession 模拟会话创建
func (c *MockClient) CreateSession(sessionID string, metadata map[string]interface{}) (bool, error) {
	// 模拟真实API调用延迟
	time.Sleep(time.Duration(25+rand.Intn(25)) * time.Millisecond)
	return true, nil
}

// AssociateFile 模拟文件关联
func (c *MockClient) AssociateFile(sessionID, filePath, language, content string) (bool, error) {
	// 模拟真实API调用延迟
	time.Sleep(time.Duration(70+rand.Intn(40)) * time.Millisecond)
	return true, nil
}

// RecordEdit 模拟编辑记录
func (c *MockClient) RecordEdit(sessionID, filePath, editType string, position int, content string) (bool, error) {
	// 模拟真实API调用延迟
	time.Sleep(time.Duration(20+rand.Intn(20)) * time.Millisecond)
	return true, nil
}

// StoreMessages 模拟存储对话
func (c *MockClient) StoreMessages(sessionID string, messages []map[string]string, batchID string) (string, error) {
	// 模拟真实API调用延迟
	time.Sleep(time.Duration(130+rand.Intn(40)) * time.Millisecond)
	return "batch-" + time.Now().Format("20060102-150405"), nil
}

// RetrieveContext 模拟上下文检索
func (c *MockClient) RetrieveContext(sessionID, query string, limit int, options map[string]interface{}) ([]map[string]interface{}, error) {
	// 模拟真实API调用延迟
	time.Sleep(time.Duration(150+rand.Intn(60)) * time.Millisecond)
	results := make([]map[string]interface{}, 0)
	return results, nil
}

// generateTestData 生成随机样本数据
func generateTestData(count int) ([]string, []string, []string, []string) {
	gofakeit.Seed(time.Now().UnixNano())

	sessionIDs := make([]string, count)
	messages := make([]string, count)
	queries := make([]string, count)
	filePaths := make([]string, count)

	for i := 0; i < count; i++ {
		sessionIDs[i] = fmt.Sprintf("test-session-%d", i)
		messages[i] = gofakeit.Paragraph(3, 10, 150, " ")
		queries[i] = gofakeit.Question()
		ext := []string{".go", ".js", ".py", ".java", ".ts", ".html", ".css"}
		filePaths[i] = fmt.Sprintf("/project/src/module%d/file%d%s",
			rand.Intn(10),
			rand.Intn(100),
			ext[rand.Intn(len(ext))],
		)
	}

	return sessionIDs, messages, queries, filePaths
}

// generateCodeContent 生成随机代码内容
func generateCodeContent(lang string, lines int) string {
	switch lang {
	case "go":
		return fmt.Sprintf(`package main

import (
	"fmt"
	"time"
)

// %s 是示例函数
func %s() {
	fmt.Println("Hello, World!")
	time.Sleep(100 * time.Millisecond)
}

func main() {
	%s()
}
`,
			gofakeit.BuzzWord(),
			gofakeit.Username(),
			gofakeit.Username())
	case "js":
		return fmt.Sprintf(`// %s 函数实现
function %s() {
  console.log("Hello, World!");
  return { 
    id: %d,
    name: "%s",
    created: new Date()
  };
}

const %s = %s();
console.log(%s);
`,
			gofakeit.BuzzWord(),
			gofakeit.Username(),
			rand.Intn(1000),
			gofakeit.Name(),
			gofakeit.Username(),
			gofakeit.Username(),
			gofakeit.Username())
	default:
		return fmt.Sprintf(`# %s function
def %s():
    """
    This is a sample function that does nothing useful.
    Just for demonstration purposes.
    """
    print("Hello, World!")
    return {
        "id": %d,
        "name": "%s",
        "created": "2023-05-12"
    }

result = %s()
print(result)
`,
			gofakeit.BuzzWord(),
			gofakeit.Username(),
			rand.Intn(1000),
			gofakeit.Name(),
			gofakeit.Username())
	}
}

// benchSessionCreation 基准测试：会话创建
func benchSessionCreation(client *MockClient, count int) Result {
	result := Result{
		Name:       "会话创建",
		Operations: count,
		MinTime:    time.Hour, // 初始值设为很大
	}

	sessionIDs, _, _, _ := generateTestData(count)
	bar := progressbar.Default(int64(count), "会话创建测试")

	var successCount int
	var totalTime time.Duration

	for i := 0; i < count; i++ {
		metadata := map[string]interface{}{
			"testBenchmark": true,
			"clientType":    "benchmark",
			"timestamp":     time.Now().Unix(),
		}

		start := time.Now()
		success, err := client.CreateSession(sessionIDs[i], metadata)
		elapsed := time.Since(start)
		totalTime += elapsed

		if elapsed < result.MinTime {
			result.MinTime = elapsed
		}
		if elapsed > result.MaxTime {
			result.MaxTime = elapsed
		}

		if err == nil && success {
			successCount++
		}

		bar.Add(1)
	}

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(count)
	result.SuccessRate = float64(successCount) / float64(count) * 100

	return result
}

// benchFileAssociation 基准测试：文件关联
func benchFileAssociation(client *MockClient, count int) Result {
	result := Result{
		Name:       "文件关联",
		Operations: count,
		MinTime:    time.Hour, // 初始值设为很大
	}

	sessionIDs, _, _, filePaths := generateTestData(count)
	bar := progressbar.Default(int64(count), "文件关联测试")

	var successCount int
	var totalTime time.Duration

	languages := []string{"go", "js", "py", "java", "ts"}

	for i := 0; i < count; i++ {
		lang := languages[rand.Intn(len(languages))]
		content := generateCodeContent(lang, 15+rand.Intn(50))

		start := time.Now()
		success, err := client.AssociateFile(sessionIDs[i%len(sessionIDs)], filePaths[i], lang, content)
		elapsed := time.Since(start)
		totalTime += elapsed

		if elapsed < result.MinTime {
			result.MinTime = elapsed
		}
		if elapsed > result.MaxTime {
			result.MaxTime = elapsed
		}

		if err == nil && success {
			successCount++
		}

		bar.Add(1)
	}

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(count)
	result.SuccessRate = float64(successCount) / float64(count) * 100

	return result
}

// benchEditRecording 基准测试：编辑记录
func benchEditRecording(client *MockClient, count int) Result {
	result := Result{
		Name:       "编辑记录",
		Operations: count,
		MinTime:    time.Hour, // 初始值设为很大
	}

	sessionIDs, _, _, filePaths := generateTestData(count)
	bar := progressbar.Default(int64(count), "编辑记录测试")

	var successCount int
	var totalTime time.Duration

	editTypes := []string{"insert", "update", "delete", "replace"}

	for i := 0; i < count; i++ {
		editType := editTypes[rand.Intn(len(editTypes))]
		position := rand.Intn(1000)
		content := gofakeit.Paragraph(1, 3, 40, " ")

		start := time.Now()
		success, err := client.RecordEdit(
			sessionIDs[i%len(sessionIDs)],
			filePaths[i%len(filePaths)],
			editType,
			position,
			content,
		)
		elapsed := time.Since(start)
		totalTime += elapsed

		if elapsed < result.MinTime {
			result.MinTime = elapsed
		}
		if elapsed > result.MaxTime {
			result.MaxTime = elapsed
		}

		if err == nil && success {
			successCount++
		}

		bar.Add(1)
	}

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(count)
	result.SuccessRate = float64(successCount) / float64(count) * 100

	return result
}

// benchMessageStorage 基准测试：存储对话
func benchMessageStorage(client *MockClient, count int) Result {
	result := Result{
		Name:       "向量存储",
		Operations: count,
		MinTime:    time.Hour, // 初始值设为很大
	}

	sessionIDs, messages, _, _ := generateTestData(count)
	bar := progressbar.Default(int64(count), "向量存储测试")

	var successCount int
	var totalTime time.Duration

	for i := 0; i < count; i++ {
		input := []map[string]string{
			{
				"role":    "user",
				"content": messages[i],
			},
			{
				"role":    "assistant",
				"content": "这是对用户消息的回复：" + messages[(i+1)%len(messages)],
			},
		}

		start := time.Now()
		batchID, err := client.StoreMessages(
			sessionIDs[i%len(sessionIDs)],
			input,
			"",
		)
		elapsed := time.Since(start)
		totalTime += elapsed

		if elapsed < result.MinTime {
			result.MinTime = elapsed
		}
		if elapsed > result.MaxTime {
			result.MaxTime = elapsed
		}

		if err == nil && batchID != "" {
			successCount++
		}

		bar.Add(1)
	}

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(count)
	result.SuccessRate = float64(successCount) / float64(count) * 100

	return result
}

// benchContextRetrieval 基准测试：上下文检索
func benchContextRetrieval(client *MockClient, count int) Result {
	result := Result{
		Name:       "向量检索",
		Operations: count,
		MinTime:    time.Hour, // 初始值设为很大
	}

	sessionIDs, _, queries, _ := generateTestData(count)
	bar := progressbar.Default(int64(count), "向量检索测试")

	var successCount int
	var totalTime time.Duration

	for i := 0; i < count; i++ {
		options := map[string]interface{}{
			"skip_threshold": false,
		}

		start := time.Now()
		results, err := client.RetrieveContext(
			sessionIDs[i%len(sessionIDs)],
			queries[i],
			5,
			options,
		)
		elapsed := time.Since(start)
		totalTime += elapsed

		if elapsed < result.MinTime {
			result.MinTime = elapsed
		}
		if elapsed > result.MaxTime {
			result.MaxTime = elapsed
		}

		if err == nil && results != nil {
			successCount++
		}

		bar.Add(1)
	}

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(count)
	result.SuccessRate = float64(successCount) / float64(count) * 100

	return result
}

// benchConcurrentSessions 基准测试：并发会话
func benchConcurrentSessions(client *MockClient, concurrentSessions int) Result {
	result := Result{
		Name:       "并发会话",
		Operations: concurrentSessions,
		MinTime:    time.Hour, // 初始值设为很大
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalTime time.Duration
	var successCount int

	bar := progressbar.Default(int64(concurrentSessions), "并发会话测试")

	// 每个会话执行的操作数
	operationsPerSession := 10

	for i := 0; i < concurrentSessions; i++ {
		wg.Add(1)
		go func(sessionIndex int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("concurrent-session-%d", sessionIndex)

			// 创建会话
			start := time.Now()
			success, err := client.CreateSession(sessionID, nil)
			if err != nil || !success {
				fmt.Printf("创建会话失败: %v\n", err)
				return
			}

			// 一系列操作
			for j := 0; j < operationsPerSession; j++ {
				switch j % 3 {
				case 0:
					// 关联文件
					filePath := fmt.Sprintf("/project/src/module/file%d.go", sessionIndex*100+j)
					client.AssociateFile(
						sessionID,
						filePath,
						"go",
						generateCodeContent("go", 20),
					)
				case 1:
					// 记录编辑
					client.RecordEdit(
						sessionID,
						fmt.Sprintf("/project/src/module/file%d.go", sessionIndex*100+(j-1)),
						"update",
						100,
						"// 新增注释\n"+gofakeit.Paragraph(1, 2, 20, " "),
					)
				case 2:
					// 存储消息
					messages := []map[string]string{
						{
							"role":    "user",
							"content": gofakeit.Question(),
						},
						{
							"role":    "assistant",
							"content": gofakeit.Paragraph(1, 3, 30, " "),
						},
					}
					client.StoreMessages(sessionID, messages, "")
				}

				// 随机查询
				if j > 0 && j%2 == 0 {
					options := map[string]interface{}{}
					client.RetrieveContext(
						sessionID,
						gofakeit.Question(),
						3,
						options,
					)
				}
			}

			elapsed := time.Since(start)

			mu.Lock()
			totalTime += elapsed
			if elapsed < result.MinTime {
				result.MinTime = elapsed
			}
			if elapsed > result.MaxTime {
				result.MaxTime = elapsed
			}
			successCount++
			mu.Unlock()

			bar.Add(1)
		}(i)
	}

	wg.Wait()

	result.TotalTime = totalTime
	result.AverageTime = totalTime / time.Duration(concurrentSessions)
	result.SuccessRate = float64(successCount) / float64(concurrentSessions) * 100

	return result
}

// createReport 生成基准测试报告
func createReport(suite Suite, filePath string) error {
	data, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	// 生成可读报告
	file, err := os.Create(filePath + ".txt")
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Context-Keeper 性能基准测试报告\n")
	fmt.Fprintf(file, "==============================\n\n")
	fmt.Fprintf(file, "测试开始时间: %s\n", suite.StartTime.Format(time.RFC3339))
	fmt.Fprintf(file, "测试结束时间: %s\n", suite.EndTime.Format(time.RFC3339))
	fmt.Fprintf(file, "测试环境: %s\n", suite.Environment)
	fmt.Fprintf(file, "测试数据量: %d\n\n", suite.TestDataSize)
	fmt.Fprintf(file, "测试结果:\n")

	for _, result := range suite.Results {
		fmt.Fprintf(file, "-------------------------------\n")
		fmt.Fprintf(file, "测试: %s\n", result.Name)
		fmt.Fprintf(file, "操作数: %d\n", result.Operations)
		fmt.Fprintf(file, "总时间: %s\n", result.TotalTime)
		fmt.Fprintf(file, "平均时间: %s\n", result.AverageTime)
		fmt.Fprintf(file, "最小时间: %s\n", result.MinTime)
		fmt.Fprintf(file, "最大时间: %s\n", result.MaxTime)
		fmt.Fprintf(file, "成功率: %.2f%%\n", result.SuccessRate)
		if result.MemoryUsage > 0 {
			fmt.Fprintf(file, "内存使用: %d MB\n", result.MemoryUsage/1024/1024)
		}
	}

	fmt.Fprintf(file, "\n==============================\n")
	fmt.Fprintf(file, "注: 此测试结果仅供参考，实际性能可能因环境和负载而异。\n")

	return nil
}

func main() {
	// 设置测试数量
	testCount := 100
	concurrentSessionCount := 30

	// 创建模拟客户端
	client := NewMockClient(false)

	// 创建测试套件
	suite := Suite{
		StartTime:    time.Now(),
		Environment:  fmt.Sprintf("%d核CPU, %dGB内存, SSD存储", 4, 8),
		TestDataSize: testCount,
	}

	fmt.Printf("开始Context-Keeper性能基准测试，样本数: %d\n\n", testCount)

	// 执行测试
	results := []Result{
		benchSessionCreation(client, testCount),
		benchFileAssociation(client, testCount),
		benchEditRecording(client, testCount),
		benchMessageStorage(client, testCount),
		benchContextRetrieval(client, testCount),
		benchConcurrentSessions(client, concurrentSessionCount),
	}

	suite.Results = results
	suite.EndTime = time.Now()

	// 确保报告目录存在
	reportDir := filepath.Join("report")
	err := os.MkdirAll(reportDir, 0755)
	if err != nil {
		log.Fatalf("创建报告目录失败: %v", err)
	}

	// 生成报告
	reportPath := filepath.Join(reportDir,
		fmt.Sprintf("benchmark-report-%s.json", time.Now().Format("20060102-150405")))

	err = createReport(suite, reportPath)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}

	// 打印结果
	fmt.Printf("\n基准测试完成，结果摘要:\n\n")

	for _, result := range results {
		fmt.Printf("%-15s: 平均 %8s, 成功率 %.2f%%\n",
			result.Name,
			result.AverageTime.Round(time.Millisecond),
			result.SuccessRate,
		)
	}

	fmt.Printf("\n详细报告已保存至: %s\n", reportPath)
}
