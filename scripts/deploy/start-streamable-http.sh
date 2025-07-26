#!/bin/bash

# Context-Keeper Streamable HTTP MCP 服务启动脚本
# 专门用于启动和配置Streamable HTTP模式

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 获取工作目录
WORKSPACE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
echo -e "${GREEN}工作目录: ${WORKSPACE_DIR}${NC}"

# 默认配置
HTTP_PORT=8088
BACKGROUND=false
AUTO_CONFIG_MCP=true
SHOW_LOGS=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --port)
      HTTP_PORT="$2"
      shift 2
      ;;
    --background)
      BACKGROUND=true
      shift
      ;;
    --no-auto-config)
      AUTO_CONFIG_MCP=false
      shift
      ;;
    --show-logs)
      SHOW_LOGS=true
      shift
      ;;
    --help|-h)
      echo "Context-Keeper Streamable HTTP MCP 服务启动脚本"
      echo ""
      echo "用法: ./start-streamable-http.sh [选项]"
      echo ""
      echo "选项:"
      echo "  --port PORT          指定HTTP服务器端口（默认: 8088）"
      echo "  --background         在后台运行服务"
      echo "  --no-auto-config     不自动配置MCP客户端"
      echo "  --show-logs          启动后自动显示实时日志"
      echo "  --help, -h           显示此帮助信息"
      echo ""
      echo "MCP端点:"
      echo "  主端点: http://localhost:PORT/mcp"
      echo "  能力查询: http://localhost:PORT/mcp/capabilities"
      echo ""
      echo "Cursor配置:"
      echo '  {'
      echo '    "mcpServers": {'
      echo '      "context-keeper": {'
      echo '        "url": "http://localhost:PORT/mcp"'
      echo '      }'
      echo '    }'
      echo '  }'
      exit 0
      ;;
    *)
      echo "未知参数: $1"
      echo "使用 --help 查看帮助"
      exit 1
      ;;
  esac
done

echo -e "${BLUE}🚀 启动 Context-Keeper Streamable HTTP MCP 服务${NC}"
echo -e "${BLUE}端口: ${HTTP_PORT}${NC}"

# 检查端口占用
echo -e "${YELLOW}检查端口 ${HTTP_PORT} 是否被占用...${NC}"
PORT_PROCESS=$(lsof -i:$HTTP_PORT -t 2>/dev/null)

if [[ -n "$PORT_PROCESS" ]]; then
  echo -e "${YELLOW}发现占用端口 ${HTTP_PORT} 的进程，PID: $PORT_PROCESS，正在终止...${NC}"
  kill -9 $PORT_PROCESS
  sleep 1
  echo -e "${GREEN}进程已终止${NC}"
else
  echo -e "${GREEN}端口 ${HTTP_PORT} 可用${NC}"
fi

# 检查运行中的context-keeper进程
RUNNING_PROCESS=$(pgrep -f context-keeper)
if [[ -n "$RUNNING_PROCESS" ]]; then
  echo -e "${YELLOW}发现运行中的 context-keeper 进程，PID: $RUNNING_PROCESS，正在终止...${NC}"
  kill -9 $RUNNING_PROCESS
  sleep 1
  echo -e "${GREEN}进程已终止${NC}"
fi

# 切换到工作目录
cd "$WORKSPACE_DIR" || { echo -e "${RED}无法进入工作目录${NC}"; exit 1; }

# 检查HTTP模式二进制文件
BINARY="./bin/context-keeper-http"
if [[ ! -f "$BINARY" ]]; then
  echo -e "${YELLOW}HTTP模式二进制文件不存在，正在编译...${NC}"
  ./scripts/build/build.sh --http
  
  if [[ $? -ne 0 ]]; then
    echo -e "${RED}编译失败，请检查代码和错误信息${NC}"
    exit 1
  fi
  
  echo -e "${GREEN}编译成功${NC}"
fi

# 设置环境变量
export HTTP_SERVER_PORT="$HTTP_PORT"
export STREAMABLE_HTTP_MODE="true"

# 自动配置MCP客户端
if [[ "$AUTO_CONFIG_MCP" == true ]]; then
  MCP_CONFIG_FILE="$HOME/.cursor/mcp.json"
  echo -e "${YELLOW}配置MCP客户端...${NC}"
  
  if command -v jq >/dev/null 2>&1; then
    # 创建或更新配置文件
    if [[ -f "$MCP_CONFIG_FILE" ]]; then
      # 更新现有配置
      TMP_FILE=$(mktemp)
      jq --arg url "http://localhost:$HTTP_PORT/mcp" '
        .mcpServers."context-keeper" = {
          "url": $url
        }
      ' "$MCP_CONFIG_FILE" > "$TMP_FILE" && mv "$TMP_FILE" "$MCP_CONFIG_FILE"
      echo -e "${GREEN}已更新现有MCP配置${NC}"
    else
      # 创建新配置
      mkdir -p "$(dirname "$MCP_CONFIG_FILE")"
      jq -n --arg url "http://localhost:$HTTP_PORT/mcp" '
        {
          "mcpServers": {
            "context-keeper": {
              "url": $url
            }
          }
        }
      ' > "$MCP_CONFIG_FILE"
      echo -e "${GREEN}已创建新的MCP配置文件${NC}"
    fi
  else
    echo -e "${YELLOW}未安装jq，请手动配置MCP客户端${NC}"
  fi
fi

# 创建日志目录
mkdir -p logs

# 显示配置信息
echo -e "${BLUE}=== 服务配置 ===${NC}"
echo -e "${YELLOW}模式: Streamable HTTP MCP${NC}"
echo -e "${YELLOW}端口: ${HTTP_PORT}${NC}"
echo -e "${YELLOW}MCP端点: http://localhost:${HTTP_PORT}/mcp${NC}"
echo -e "${YELLOW}能力查询: http://localhost:${HTTP_PORT}/mcp/capabilities${NC}"
echo -e "${YELLOW}服务信息: http://localhost:${HTTP_PORT}/${NC}"

# 启动服务
if [[ "$BACKGROUND" == true ]]; then
  echo -e "${GREEN}以后台模式启动服务...${NC}"
  nohup "$BINARY" > logs/streamable-http.log 2>&1 &
  PID=$!
  
  # 等待服务启动
  echo -e "${YELLOW}等待服务启动...${NC}"
  sleep 3
  
  # 检查服务状态
  if ps -p $PID > /dev/null; then
    echo -e "${GREEN}✅ 服务启动成功！PID: $PID${NC}"
    echo -e "${YELLOW}日志文件: logs/streamable-http.log${NC}"
    
    # 测试服务连通性
    if curl -s "http://localhost:$HTTP_PORT/health" >/dev/null; then
      echo -e "${GREEN}✅ 服务健康检查通过${NC}"
    else
      echo -e "${YELLOW}⚠️  服务健康检查未通过，请检查日志${NC}"
    fi
    
    echo -e "${BLUE}=== 下一步操作 ===${NC}"
    echo -e "${YELLOW}1. 重启Cursor以加载新的MCP配置${NC}"
    echo -e "${YELLOW}2. 在Cursor中应该能看到context-keeper工具${NC}"
    echo -e "${YELLOW}3. 查看日志: tail -f logs/streamable-http.log${NC}"
    echo -e "${YELLOW}4. 停止服务: kill $PID${NC}"
    
    # 询问是否显示实时日志
    if [[ "$SHOW_LOGS" == true ]]; then
      echo -e "${BLUE}=== 显示实时日志 ===${NC}"
      echo -e "${YELLOW}按 Ctrl+C 可停止日志显示（服务将继续在后台运行）${NC}"
      sleep 1
      tail -f logs/streamable-http.log
    else
      echo ""
      echo -e "${BLUE}是否要查看实时日志？${NC}"
      echo -e "${YELLOW}输入 'y' 查看实时日志，或按任意键退出${NC}"
      read -t 10 -n 1 response
      echo ""
      
      if [[ "$response" == "y" || "$response" == "Y" ]]; then
        echo -e "${BLUE}=== 显示实时日志 ===${NC}"
        echo -e "${YELLOW}按 Ctrl+C 可停止日志显示（服务将继续在后台运行）${NC}"
        sleep 1
        tail -f logs/streamable-http.log
      else
        echo -e "${GREEN}服务已在后台运行，可使用以下命令查看日志：${NC}"
        echo -e "${YELLOW}tail -f logs/streamable-http.log${NC}"
      fi
    fi
    
  else
    echo -e "${RED}❌ 服务启动失败，查看日志了解详情:${NC}"
    cat logs/streamable-http.log
    exit 1
  fi
else
  # 前台启动
  echo -e "${GREEN}以前台模式启动服务...${NC}"
  echo -e "${YELLOW}按 Ctrl+C 可停止服务${NC}"
  echo -e "${BLUE}=== 服务启动中 ===${NC}"
  echo ""
  echo -e "${BLUE}=== 服务启动提示 ===${NC}"
  echo -e "${YELLOW}1. 重启Cursor以加载新的MCP配置${NC}"
  echo -e "${YELLOW}2. MCP端点: http://localhost:${HTTP_PORT}/mcp${NC}"
  echo -e "${YELLOW}3. 能力查询: http://localhost:${HTTP_PORT}/mcp/capabilities${NC}"
  echo -e "${YELLOW}4. 服务信息: http://localhost:${HTTP_PORT}/${NC}"
  echo ""
  echo -e "${BLUE}=== 服务日志 ===${NC}"
  
  "$BINARY"
fi 