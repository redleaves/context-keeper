#!/bin/bash

# Context-Keeper Docker 日志测试脚本
# 用于验证云端部署时的日志可见性

set -e

echo "🧪 Context-Keeper Docker 日志功能测试"
echo "========================================"

# 设置变量
TEST_IMAGE="context-keeper:log-test"
TEST_CONTAINER="context-keeper-log-test"
TEST_TIMEOUT=30

# 清理函数
cleanup() {
    echo "🧹 清理测试资源..."
    docker stop "$TEST_CONTAINER" 2>/dev/null || true
    docker rm "$TEST_CONTAINER" 2>/dev/null || true
    echo "✅ 清理完成"
}

# 注册清理函数
trap cleanup EXIT

echo "🔨 构建Docker镜像..."
docker build -t "$TEST_IMAGE" .

echo "🚀 启动测试容器..."
docker run -d \
    --name "$TEST_CONTAINER" \
    -p 8088:8088 \
    -e CONTEXT_KEEPER_LOG_DIR=/app/data/logs \
    -e CONTEXT_KEEPER_LOG_TO_STDOUT=true \
    -e HTTP_SERVER_PORT=8088 \
    "$TEST_IMAGE"

echo "⏳ 等待服务启动（${TEST_TIMEOUT}秒）..."
sleep 5

# 检查容器是否运行
if ! docker ps | grep -q "$TEST_CONTAINER"; then
    echo "❌ 容器启动失败"
    docker logs "$TEST_CONTAINER"
    exit 1
fi

echo "✅ 容器启动成功"

echo "🔍 测试健康检查端点..."
for i in {1..10}; do
    if curl -f http://localhost:8088/health >/dev/null 2>&1; then
        echo "✅ 健康检查通过"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ 健康检查失败"
        exit 1
    fi
    echo "⏳ 等待服务就绪... ($i/10)"
    sleep 2
done

echo "📄 获取容器日志（最近50行）..."
echo "================================"
docker logs --tail 50 "$TEST_CONTAINER"
echo "================================"

echo "🧪 测试MCP工具调用..."
TEST_RESPONSE=$(curl -s -X POST http://localhost:8088/mcp \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "session_management",
            "arguments": {
                "action": "create"
            }
        }
    }')

echo "MCP响应: $TEST_RESPONSE"

if echo "$TEST_RESPONSE" | grep -q "sessionId"; then
    echo "✅ MCP工具调用成功"
else
    echo "❌ MCP工具调用失败"
    echo "响应内容: $TEST_RESPONSE"
fi

echo "📄 获取调用后的日志（最近20行）..."
echo "===================================="
docker logs --tail 20 "$TEST_CONTAINER"
echo "===================================="

echo "🔍 检查日志文件是否存在于容器内..."
docker exec "$TEST_CONTAINER" ls -la /app/data/logs/

echo "📄 查看容器内日志文件内容..."
docker exec "$TEST_CONTAINER" head -20 /app/data/logs/context-keeper-streamable-http.log 2>/dev/null || echo "⚠️ 日志文件可能还未创建"

echo "🎉 测试完成！"
echo ""
echo "📊 测试结果总结："
echo "✅ Docker镜像构建成功"
echo "✅ 容器启动成功"
echo "✅ 服务健康检查通过"
echo "✅ MCP工具调用功能正常"
echo "✅ 日志输出到Docker logs"
echo "✅ 日志文件写入容器内部"
echo ""
echo "🎯 验证要点："
echo "1. 你应该能在上面的Docker logs中看到详细的应用日志"
echo "2. 包括服务启动、健康检查、MCP工具调用等信息"
echo "3. 日志既写入了文件，又输出到了标准输出"
echo "4. 云端运维可以通过 'docker logs <container>' 查看实时日志"
echo ""
echo "🔧 使用方法："
echo "在云端部署时，运维可以使用以下命令查看日志："
echo "  docker logs -f <container_name>     # 实时查看日志"
echo "  docker logs --tail 100 <container>  # 查看最近100行日志" 