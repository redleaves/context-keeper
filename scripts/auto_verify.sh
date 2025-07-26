#!/bin/bash

# 🔄 Context-Keeper 自动验证脚本
# 用于服务端代码修改后的完整验证流程

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 输出函数
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查函数
check_command() {
    if command -v $1 >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 验证结果统计
PASSED_TESTS=0
FAILED_TESTS=0
TOTAL_TESTS=0

test_result() {
    local test_name="$1"
    local result="$2"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$result" = "pass" ]; then
        success "$test_name: 通过"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        error "$test_name: 失败"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

echo "🚀 Context-Keeper 自动验证流程开始..."
echo "============================================"

# 第一阶段：预检查
info "第一阶段：预检查"
echo "--------------------------------------------"

# 检查必要命令
info "检查必要命令..."
if ! check_command "go"; then
    error "Go 未安装或不在 PATH 中"
    exit 1
fi

if ! check_command "curl"; then
    error "curl 未安装或不在 PATH 中"
    exit 1
fi

if ! check_command "node"; then
    warning "Node.js 未安装，将跳过 JavaScript 测试"
fi

success "预检查通过"

# 第二阶段：服务重启
info "第二阶段：服务重启"
echo "--------------------------------------------"

# 停止现有服务
info "停止现有服务..."
./scripts/manage.sh stop >/dev/null 2>&1 || true
pkill -f context-keeper >/dev/null 2>&1 || true
sleep 2
success "服务已停止"

# 编译新版本
info "编译新版本..."
if go build -o bin/context-keeper-http cmd/server/main.go cmd/server/main_http.go; then
    success "HTTP服务编译成功"
else
    error "HTTP服务编译失败"
    exit 1
fi

if go build -o bin/context-keeper-websocket cmd/server/main.go cmd/server/main_websocket.go; then
    success "WebSocket服务编译成功"
else
    error "WebSocket服务编译失败"
    exit 1
fi

# 启动服务
info "启动服务..."
if ./scripts/manage.sh deploy http --port 8088 >/dev/null 2>&1; then
    success "HTTP服务启动成功"
else
    error "HTTP服务启动失败"
    exit 1
fi

# 启动 WebSocket 服务
info "启动WebSocket服务..."
WEBSOCKET_SERVER_PORT=7890 nohup ./bin/context-keeper-websocket > logs/context-keeper-websocket.log 2>&1 & echo $! > logs/context-keeper-websocket.pid
sleep 3

# 检查进程状态
info "检查进程状态..."
PROCESS_COUNT=$(ps aux | grep context-keeper | grep -v grep | wc -l)
if [ "$PROCESS_COUNT" -ge 2 ]; then
    success "服务进程启动正常 ($PROCESS_COUNT 个进程)"
else
    warning "服务进程数量异常 ($PROCESS_COUNT 个进程)"
fi

# 第三阶段：基础验证
info "第三阶段：基础验证"
echo "--------------------------------------------"

# HTTP服务健康检查
info "HTTP服务健康检查..."
if curl -s http://localhost:8088/health | grep -q "healthy"; then
    test_result "HTTP服务健康检查" "pass"
else
    test_result "HTTP服务健康检查" "fail"
fi

# WebSocket服务健康检查
info "WebSocket服务健康检查..."
if curl -s http://localhost:7890/health | grep -q "healthy"; then
    test_result "WebSocket服务健康检查" "pass"
else
    test_result "WebSocket服务健康检查" "fail"
fi

# 端口监听验证
info "端口监听验证..."
if netstat -an | grep LISTEN | grep -E '8088|7890' | wc -l | grep -q '2'; then
    test_result "端口监听验证" "pass"
else
    test_result "端口监听验证" "fail"
fi

# 第四阶段：功能回归验证
info "第四阶段：功能回归验证"
echo "--------------------------------------------"

# Session管理验证
info "Session管理验证..."
SESSION_RESPONSE=$(curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"session_management","arguments":{"action":"get_or_create"}}}')

if echo "$SESSION_RESPONSE" | grep -q "session-"; then
    test_result "Session管理验证" "pass"
    # 提取sessionId用于后续测试（使用更可靠的模式）
    SESSION_ID=$(echo "$SESSION_RESPONSE" | grep -o 'session-[0-9]*-[0-9]*' | head -1)
    info "创建的SessionID: $SESSION_ID"
else
    test_result "Session管理验证" "fail"
    SESSION_ID="test-session-fallback"
fi

# WebSocket状态检查
info "WebSocket状态检查..."
if curl -s http://localhost:8088/ws/status | grep -q "success"; then
    test_result "WebSocket状态检查" "pass"
else
    test_result "WebSocket状态检查" "fail"
fi

# Session注册端点验证
info "Session注册端点验证..."
REGISTER_RESPONSE=$(curl -s -X POST "http://localhost:8088/api/ws/register-session" \
  -H "Content-Type: application/json" \
  -d "{\"sessionId\":\"$SESSION_ID\",\"connectionId\":\"test-connection-$(date +%s)\"}")

if echo "$REGISTER_RESPONSE" | grep -q "success\|注册成功"; then
    test_result "Session注册端点验证" "pass"
else
    test_result "Session注册端点验证" "fail"
fi

# MCP工具基础验证
info "MCP工具基础验证..."
TOOLS_RESPONSE=$(curl -s -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}')

if echo "$TOOLS_RESPONSE" | grep -q "session_management"; then
    test_result "MCP工具基础验证" "pass"
else
    test_result "MCP工具基础验证" "fail"
fi

# 第五阶段：JavaScript测试验证（如果Node.js可用）
if check_command "node"; then
    info "第五阶段：JavaScript测试验证"
    echo "--------------------------------------------"
    
    # Session状态检查
    if [ -f "session_status_check.js" ]; then
        info "运行Session状态检查..."
        # macOS兼容的超时处理
        if node session_status_check.js >/dev/null 2>&1; then
            test_result "Session状态检查脚本" "pass"
        else
            test_result "Session状态检查脚本" "fail"
        fi
    fi
    
    # WebSocket测试客户端（简短测试）
    if [ -f "websocket_test_client.js" ]; then
        info "运行WebSocket客户端测试..."
        # 检查WebSocket客户端是否能连接成功
        WS_TEST_OUTPUT=$(node websocket_test_client.js 2>&1 | grep -E "连接成功|WebSocket连接已建立" | head -1)
        if [ -n "$WS_TEST_OUTPUT" ]; then
            test_result "WebSocket客户端测试" "pass"
        else
            test_result "WebSocket客户端测试" "fail"
        fi
    fi
fi

# 验证结果汇总
echo ""
echo "============================================"
info "验证结果汇总"
echo "============================================"

echo "📊 测试统计："
echo "   总计: $TOTAL_TESTS"
echo "   通过: $PASSED_TESTS"
echo "   失败: $FAILED_TESTS"

if [ "$FAILED_TESTS" -eq 0 ]; then
    echo ""
    success "🎉 所有验证通过！系统状态良好"
    echo ""
    info "服务状态摘要："
    echo "   - HTTP服务: 运行中 (端口: 8088)"
    echo "   - WebSocket服务: 运行中 (端口: 7890)"
    echo "   - MCP协议: 正常工作"
    echo "   - Session管理: 功能正常"
    echo ""
    success "✨ 可以继续开发新功能！"
    exit 0
else
    echo ""
    error "⚠️  发现 $FAILED_TESTS 个问题需要修复"
    echo ""
    warning "修复建议："
    echo "   1. 检查失败的测试项目"
    echo "   2. 查看服务日志: logs/context-keeper-*.log"
    echo "   3. 确认端口未被占用"
    echo "   4. 重新编译并测试"
    echo ""
    warning "🔧 请修复问题后重新运行验证"
    exit 1
fi 