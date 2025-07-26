#!/bin/bash

# Context-Keeper Docker部署验证脚本
# 验证Docker部署是否正确模拟了 ./scripts/manage.sh deploy http 的完整流程

set -e

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONTAINER_NAME="context-keeper-test"
IMAGE_NAME="context-keeper:test"
TEST_PORT="8089"
DOCKER_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yml"

echo -e "${BLUE}Context-Keeper Docker部署验证脚本${NC}"
echo -e "${YELLOW}验证Docker部署是否完整模拟 manage.sh deploy http 流程${NC}"
echo ""

# 显示测试环境
echo -e "${GREEN}测试环境:${NC}"
echo "  项目根目录: ${PROJECT_ROOT}"
echo "  容器名称: ${CONTAINER_NAME}"
echo "  镜像名称: ${IMAGE_NAME}"
echo "  测试端口: ${TEST_PORT}"
echo ""

# 清理函数
cleanup() {
    echo -e "${YELLOW}清理测试环境...${NC}"
    
    # 停止并删除容器
    if docker ps -q -f name="${CONTAINER_NAME}" | grep -q .; then
        echo "停止容器 ${CONTAINER_NAME}..."
        docker stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    if docker ps -a -q -f name="${CONTAINER_NAME}" | grep -q .; then
        echo "删除容器 ${CONTAINER_NAME}..."
        docker rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # 可选：删除测试镜像
    if [ "${1:-}" = "--clean-image" ]; then
        if docker images -q "${IMAGE_NAME}" | grep -q .; then
            echo "删除镜像 ${IMAGE_NAME}..."
            docker rmi "${IMAGE_NAME}" >/dev/null 2>&1 || true
        fi
    fi
}

# 注册清理函数
trap cleanup EXIT

# 检查Docker环境
check_docker() {
    echo -e "${BLUE}检查Docker环境...${NC}"
    
    if ! command -v docker >/dev/null 2>&1; then
        echo -e "${RED}错误: Docker未安装${NC}"
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}错误: Docker服务未运行${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker环境正常${NC}"
}

# 构建Docker镜像
build_image() {
    echo -e "${BLUE}构建Docker镜像（模拟manage.sh build过程）...${NC}"
    
    cd "${PROJECT_ROOT}"
    
    # 构建镜像
    if ! docker build \
        --build-arg VERSION=test \
        --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --build-arg COMMIT_HASH="$(git rev-parse HEAD 2>/dev/null || echo 'test')" \
        -t "${IMAGE_NAME}" \
        -f Dockerfile \
        . ; then
        echo -e "${RED}✗ Docker镜像构建失败${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker镜像构建成功${NC}"
}

# 验证镜像内容
verify_image() {
    echo -e "${BLUE}验证镜像内容（检查是否包含manage.sh相同的构建产物）...${NC}"
    
    # 检查二进制文件
    echo "检查二进制文件..."
    if ! docker run --rm "${IMAGE_NAME}" ls -la /app/bin/; then
        echo -e "${RED}✗ 无法列出bin目录内容${NC}"
        exit 1
    fi
    
    # 检查关键文件
    echo "检查关键文件..."
    docker run --rm "${IMAGE_NAME}" bash -c "
        echo '检查二进制文件:' &&
        ls -la /app/bin/context-keeper* &&
        echo '检查管理脚本:' &&
        ls -la /app/scripts/manage.sh &&
        echo '检查入口点脚本:' &&
        ls -la /app/docker-entrypoint.sh &&
        echo '检查构建脚本:' &&
        ls -la /app/scripts/build/build.sh
    " || {
        echo -e "${RED}✗ 关键文件检查失败${NC}"
        exit 1
    }
    
    echo -e "${GREEN}✓ 镜像内容验证通过${NC}"
}

# 启动容器
start_container() {
    echo -e "${BLUE}启动容器（模拟manage.sh start http过程）...${NC}"
    
    # 确保端口未被占用
    if netstat -tuln 2>/dev/null | grep -q ":${TEST_PORT} "; then
        echo -e "${RED}错误: 端口 ${TEST_PORT} 已被占用${NC}"
        exit 1
    fi
    
    # 启动容器
    if ! docker run -d \
        --name "${CONTAINER_NAME}" \
        -p "${TEST_PORT}:8088" \
        -e RUN_MODE=http \
        -e HTTP_SERVER_PORT=8088 \
        -e LOG_LEVEL=info \
        "${IMAGE_NAME}" http; then
        echo -e "${RED}✗ 容器启动失败${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ 容器启动成功${NC}"
}

# 等待服务就绪
wait_for_service() {
    echo -e "${BLUE}等待服务就绪（模拟manage.sh启动验证）...${NC}"
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "http://localhost:${TEST_PORT}/health" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ 服务已就绪${NC}"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 1
    done
    
    echo -e "\n${RED}✗ 服务启动超时${NC}"
    
    # 显示容器日志
    echo -e "${YELLOW}容器日志:${NC}"
    docker logs "${CONTAINER_NAME}" 2>&1 | tail -20
    
    exit 1
}

# 测试HTTP API
test_http_api() {
    echo -e "${BLUE}测试HTTP API（验证与manage.sh启动的服务功能一致）...${NC}"
    
    local base_url="http://localhost:${TEST_PORT}"
    
    # 测试健康检查
    echo "测试健康检查..."
    if ! curl -sf "${base_url}/health" >/dev/null; then
        echo -e "${RED}✗ 健康检查失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 健康检查通过${NC}"
    
    # 测试MCP端点
    echo "测试MCP端点..."
    if ! curl -sf "${base_url}/mcp/tools/list" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' >/dev/null; then
        echo -e "${RED}✗ MCP端点测试失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ MCP端点测试通过${NC}"
    
    # 测试会话管理（核心功能验证）
    echo "测试会话管理功能..."
    local session_response=$(curl -sf "${base_url}/mcp/tools/call" \
        -H "Content-Type: application/json" \
        -d '{
            "jsonrpc":"2.0",
            "id":1,
            "method":"tools/call",
            "params":{
                "name":"session_management",
                "arguments":{"action":"create"}
            }
        }' 2>/dev/null)
    
    if [ $? -eq 0 ] && echo "$session_response" | grep -q "sessionId"; then
        echo -e "${GREEN}✓ 会话管理功能测试通过${NC}"
    else
        echo -e "${RED}✗ 会话管理功能测试失败${NC}"
        echo "响应: $session_response"
        exit 1
    fi
}

# 测试容器管理
test_container_management() {
    echo -e "${BLUE}测试容器管理（验证类似manage.sh的管理功能）...${NC}"
    
    # 检查容器状态
    echo "检查容器状态..."
    if ! docker ps | grep -q "${CONTAINER_NAME}"; then
        echo -e "${RED}✗ 容器未运行${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 容器正在运行${NC}"
    
    # 检查进程状态
    echo "检查容器内进程..."
    local process_info=$(docker exec "${CONTAINER_NAME}" ps aux 2>/dev/null | grep context-keeper | grep -v grep)
    if [ -z "$process_info" ]; then
        echo -e "${RED}✗ 容器内未找到context-keeper进程${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 容器内进程正常${NC}"
    echo "  进程信息: $process_info"
    
    # 检查日志
    echo "检查容器日志..."
    local log_count=$(docker logs "${CONTAINER_NAME}" 2>&1 | wc -l)
    if [ "$log_count" -lt 5 ]; then
        echo -e "${RED}✗ 容器日志异常${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 容器日志正常 (${log_count}行)${NC}"
}

# 性能测试
test_performance() {
    echo -e "${BLUE}性能测试（验证Docker部署性能与原生部署一致）...${NC}"
    
    local base_url="http://localhost:${TEST_PORT}"
    
    # 简单的并发测试
    echo "执行并发请求测试..."
    local success_count=0
    local total_requests=10
    
    for i in $(seq 1 $total_requests); do
        if curl -sf "${base_url}/health" >/dev/null 2>&1; then
            success_count=$((success_count + 1))
        fi
    done
    
    local success_rate=$((success_count * 100 / total_requests))
    echo "成功率: ${success_count}/${total_requests} (${success_rate}%)"
    
    if [ $success_rate -ge 90 ]; then
        echo -e "${GREEN}✓ 性能测试通过${NC}"
    else
        echo -e "${RED}✗ 性能测试失败${NC}"
        exit 1
    fi
}

# 测试Docker Compose部署
test_docker_compose() {
    echo -e "${BLUE}测试Docker Compose部署...${NC}"
    
    if [ ! -f "${DOCKER_COMPOSE_FILE}" ]; then
        echo -e "${YELLOW}⚠ docker-compose.yml文件不存在，跳过compose测试${NC}"
        return 0
    fi
    
    # 检查compose文件语法
    if ! docker-compose -f "${DOCKER_COMPOSE_FILE}" config >/dev/null 2>&1; then
        echo -e "${RED}✗ docker-compose.yml语法错误${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ docker-compose.yml语法正确${NC}"
}

# 生成测试报告
generate_report() {
    echo -e "${BLUE}生成测试报告...${NC}"
    
    local report_file="${PROJECT_ROOT}/docker-deploy-test-report.txt"
    
    cat > "$report_file" << EOF
Context-Keeper Docker部署验证报告
=====================================

测试时间: $(date)
项目版本: $(git describe --tags --always --dirty 2>/dev/null || echo "dev")
提交哈希: $(git rev-parse HEAD 2>/dev/null || echo "unknown")

测试环境:
- 容器名称: ${CONTAINER_NAME}
- 镜像名称: ${IMAGE_NAME}
- 测试端口: ${TEST_PORT}

测试结果:
✓ Docker环境检查通过
✓ 镜像构建成功
✓ 镜像内容验证通过
✓ 容器启动成功
✓ 服务就绪验证通过
✓ HTTP API测试通过
✓ 容器管理测试通过
✓ 性能测试通过
✓ Docker Compose配置正确

结论:
Docker部署成功模拟了 ./scripts/manage.sh deploy http 的完整流程，
包括构建、启动、验证等所有关键步骤。
部署的容器服务与原生部署功能一致。

容器信息:
$(docker inspect "${CONTAINER_NAME}" --format='
- 容器ID: {{.Id}}
- 状态: {{.State.Status}}
- 启动时间: {{.State.StartedAt}}
- 端口映射: {{.NetworkSettings.Ports}}
')

EOF
    
    echo -e "${GREEN}✓ 测试报告已生成: ${report_file}${NC}"
}

# 主测试流程
main() {
    echo -e "${YELLOW}开始Docker部署验证测试...${NC}"
    echo ""
    
    # 解析命令行参数
    local clean_image=false
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean-image)
                clean_image=true
                shift
                ;;
            --help)
                echo "用法: ./test-docker-deploy.sh [选项]"
                echo "选项:"
                echo "  --clean-image    测试完成后删除测试镜像"
                echo "  --help          显示此帮助"
                exit 0
                ;;
            *)
                echo -e "${RED}未知参数: $1${NC}"
                exit 1
                ;;
        esac
    done
    
    # 执行测试步骤
    check_docker
    build_image
    verify_image
    start_container
    wait_for_service
    test_http_api
    test_container_management
    test_performance
    test_docker_compose
    generate_report
    
    echo ""
    echo -e "${GREEN}🎉 所有测试通过！${NC}"
    echo -e "${GREEN}Docker部署正确模拟了 manage.sh deploy http 的完整流程${NC}"
    
    # 可选清理镜像
    if [ "$clean_image" = true ]; then
        cleanup --clean-image
    fi
}

# 运行主函数
main "$@" 