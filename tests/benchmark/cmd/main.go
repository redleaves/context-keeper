package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"context-keeper/tests/benchmark"
)

func main() {
	// 设置测试数量
	testCount := 100
	concurrentSessionCount := 30

	// 创建模拟客户端
	client := benchmark.NewMockClient(false)

	// 创建测试套件
	suite := benchmark.Suite{
		StartTime:    time.Now(),
		Environment:  fmt.Sprintf("%d核CPU, %dGB内存, SSD存储", 4, 8),
		TestDataSize: testCount,
	}

	fmt.Printf("开始Context-Keeper性能基准测试，样本数: %d\n\n", testCount)

	// 执行测试
	results := []benchmark.Result{
		benchmark.BenchSessionCreation(client, testCount),
		benchmark.BenchFileAssociation(client, testCount),
		benchmark.BenchEditRecording(client, testCount),
		benchmark.BenchMessageStorage(client, testCount),
		benchmark.BenchContextRetrieval(client, testCount),
		benchmark.BenchConcurrentSessions(client, concurrentSessionCount),
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

	err = benchmark.CreateReport(suite, reportPath)
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
