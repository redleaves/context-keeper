#!/bin/bash

# 清理会话并验证新的会话创建逻辑

echo "🧹 开始清理会话..."

# 获取当前工作目录
CURRENT_DIR=$(pwd)
echo "当前工作目录: $CURRENT_DIR"

# 查找数据目录
DATA_DIR="./data"
HOME_DATA_DIR="$HOME/.context-keeper"
MAC_DATA_DIR="$HOME/Library/Application Support/context-keeper"

if [ -d "$DATA_DIR" ]; then
    echo "找到本地数据目录: $DATA_DIR"
    SESSIONS_DIR="$DATA_DIR/sessions"
elif [ -d "$HOME_DATA_DIR" ]; then
    echo "找到主目录数据目录: $HOME_DATA_DIR"
    SESSIONS_DIR="$HOME_DATA_DIR/sessions"
elif [ -d "$MAC_DATA_DIR" ]; then
    echo "找到Mac应用数据目录: $MAC_DATA_DIR"
    SESSIONS_DIR="$MAC_DATA_DIR/sessions"
else
    echo "❌ 无法找到会话数据目录"
    exit 1
fi

# 检查会话目录是否存在
if [ ! -d "$SESSIONS_DIR" ]; then
    echo "❌ 会话目录不存在: $SESSIONS_DIR"
    exit 1
fi

echo "📂 会话目录: $SESSIONS_DIR"
echo "当前会话文件列表:"
ls -la "$SESSIONS_DIR"

# 查找并删除特定会话
SESSION_TO_DELETE="session-20250703-142210.json"
if [ -f "$SESSIONS_DIR/$SESSION_TO_DELETE" ]; then
    echo "🗑️ 删除会话文件: $SESSION_TO_DELETE"
    rm "$SESSIONS_DIR/$SESSION_TO_DELETE"
    echo "✅ 会话文件已删除"
else
    echo "⚠️ 会话文件不存在: $SESSION_TO_DELETE"
fi

# 停止现有服务
echo "⏹️ 停止现有服务..."
./scripts/manage.sh stop

# 编译服务
echo "🔨 编译服务..."
go build -o bin/context-keeper-http cmd/server/main.go cmd/server/main_http.go
go build -o bin/context-keeper-websocket cmd/server/main.go cmd/server/main_websocket.go

# 启动服务
echo "🚀 启动服务..."
./scripts/manage.sh deploy http --port 8088
WEBSOCKET_SERVER_PORT=7890 nohup ./bin/context-keeper-websocket > logs/context-keeper-websocket.log 2>&1 & echo $! > logs/context-keeper-websocket.pid

# 等待服务就绪
echo "⏱️ 等待服务就绪..."
sleep 3

# 验证服务状态
echo "🔍 验证服务状态..."
curl -s http://localhost:8088/health
echo
curl -s http://localhost:7890/health
echo

echo "🔄 验证会话创建..."
curl -X POST http://localhost:8088/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "id":1,
    "method":"tools/call",
    "params":{
      "name":"session_management",
      "arguments":{
        "action":"create",
        "metadata":{
          "workspaceRoot":"/Users/weixiaofeng12/coding/context-keeper",
          "workspaceHash":"e3160bbc"
        }
      }
    }
  }'
echo

echo "✅ 清理和验证完成"
echo "现在可以重新测试工作空间隔离功能" 