.PHONY: benchmark setup clean report

# 默认目标
all: setup benchmark

# 安装依赖
setup:
	@echo "正在安装依赖..."
	go get github.com/brianvoe/gofakeit/v6
	go get github.com/schollz/progressbar/v3
	@echo "依赖安装完成"

# 运行基准测试
benchmark:
	@echo "开始运行基准测试..."
	@mkdir -p report
	go run benchmark.go
	@echo "基准测试完成"

# 查看最新报告
report:
	@echo "显示最新基准测试报告..."
	@ls -t report/*.txt | head -1 | xargs cat

# 清理测试数据
clean:
	@echo "清理基准测试数据..."
	rm -rf ../../data/benchmark_test
	@echo "清理完成" 