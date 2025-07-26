#!/bin/bash

# 构建脚本
# 用于编译context-keeper服务

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 项目根目录
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# 切换到项目根目录
cd "${PROJECT_ROOT}"
echo "切换到项目根目录: ${PROJECT_ROOT}"

# 设置Go环境变量
export GO111MODULE=on
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)

# 构建信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# 编译参数
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}"

echo "开始构建 context-keeper..."
echo "版本: ${VERSION}"
echo "构建时间: ${BUILD_TIME}"
echo "提交哈希: ${COMMIT_HASH}"

# 创建bin目录
mkdir -p ./bin

# 处理命令行参数
BUILD_TYPE="all"
while [[ $# -gt 0 ]]; do
  case $1 in
    --stdio)
      BUILD_TYPE="stdio"
      shift
      ;;
    --http)
      BUILD_TYPE="http"
      shift
      ;;
    --all)
      BUILD_TYPE="all"
      shift
      ;;
    --help)
      echo "用法: ./build.sh [选项]"
      echo "选项:"
      echo "  --stdio    仅编译STDIO版本（用于MCP通信）"
      echo "  --http     仅编译HTTP/SSE版本（用于网络通信）"
      echo "  --all      编译所有版本（默认）"
      echo "  --help     显示此帮助信息"
      exit 0
      ;;
    *)
      echo "未知参数: $1"
      echo "使用 --help 查看帮助"
      exit 1
      ;;
  esac
done

# 编译函数
build_binary() {
    local mode=$1
    local output=$2
    echo "正在编译 ${mode} 模式的二进制文件..."
    go build -ldflags "${LDFLAGS}" -tags "${mode}" -o "${output}" ./cmd/server
    if [ $? -eq 0 ]; then
        echo "构建成功! 可执行文件位于: ${output}"
        chmod +x "${output}"
    else
        echo "构建失败!"
        exit 1
    fi
}

# 根据构建类型执行编译
if [ "$BUILD_TYPE" = "all" ] || [ "$BUILD_TYPE" = "stdio" ]; then
    build_binary "stdio" "./bin/context-keeper"
fi

if [ "$BUILD_TYPE" = "all" ] || [ "$BUILD_TYPE" = "http" ]; then
    build_binary "http" "./bin/context-keeper-http"
fi

echo "完成!" 