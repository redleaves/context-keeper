#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # 无颜色

# 获取工作目录
WORKSPACE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
echo -e "${GREEN}工作目录: ${WORKSPACE_DIR}${NC}"

# 默认值
RUN_MODE="stdio"
BACKGROUND=false
HTTP_PORT=8088

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --http)
      RUN_MODE="http"
      shift
      ;;
    --stdio)
      RUN_MODE="stdio"
      shift
      ;;
    --port)
      HTTP_PORT="$2"
      shift 2
      ;;
    --background)
      BACKGROUND=true
      shift
      ;;
    --help|-h)
      echo "用法: ./start.sh [选项]"
      echo "选项:"
      echo "  --stdio             使用STDIO模式启动（默认，适用于MCP通信）"
      echo "  --http              使用HTTP模式启动（适用于网络通信）"
      echo "  --port PORT         指定HTTP服务器端口（默认: 8088）"
      echo "  --background        在后台运行服务（不阻塞终端）"
      echo "  --help, -h          显示此帮助信息"
      exit 0
      ;;
    *)
      echo "未知参数: $1"
      echo "使用 --help 查看帮助"
      exit 1
      ;;
  esac
done

echo -e "${GREEN}正在启动Context-keeper服务...${NC}"
echo -e "${GREEN}运行模式: ${RUN_MODE}${NC}"

# 如果是HTTP模式，检查指定端口是否被占用
if [[ "$RUN_MODE" == "http" ]]; then
  echo -e "${YELLOW}检查是否有占用 ${HTTP_PORT} 端口的进程...${NC}"
  PORT_PROCESS=$(lsof -i:$HTTP_PORT -t 2>/dev/null)
  
  if [[ -n "$PORT_PROCESS" ]]; then
    echo -e "${YELLOW}发现占用 ${HTTP_PORT} 端口的进程，PID: $PORT_PROCESS，正在终止...${NC}"
    kill -9 $PORT_PROCESS
    sleep 1
    echo -e "${GREEN}进程已终止${NC}"
  else
    echo -e "${GREEN}未发现占用 ${HTTP_PORT} 端口的进程${NC}"
  fi
fi

# 检查是否有运行中的context-keeper进程
RUNNING_PROCESS=$(pgrep -f context-keeper)
if [[ -n "$RUNNING_PROCESS" ]]; then
  echo -e "${YELLOW}发现运行中的 context-keeper 进程，PID: $RUNNING_PROCESS，正在终止...${NC}"
  kill -9 $RUNNING_PROCESS
  sleep 1
  echo -e "${GREEN}进程已终止${NC}"
else
  echo -e "${GREEN}未发现运行中的 context-keeper 进程${NC}"
fi

# 切换到工作目录
cd "$WORKSPACE_DIR" || { echo -e "${RED}无法进入工作目录${NC}"; exit 1; }

# 根据运行模式选择二进制文件
if [[ "$RUN_MODE" == "http" ]]; then
  BINARY="context-keeper-http"
else
  BINARY="context-keeper-stdio"
fi

echo -e "${YELLOW}当前选择的二进制文件: ${BINARY}${NC}"

# 检查二进制文件是否存在，如果不存在则构建
if [[ -f "$BINARY" ]]; then
  echo -e "${GREEN}使用现有的 ${BINARY} 二进制文件${NC}"
else
  echo -e "${YELLOW}二进制文件 ${BINARY} 不存在，正在编译...${NC}"
  if [[ "$RUN_MODE" == "http" ]]; then
    echo -e "${YELLOW}使用HTTP模式编译...${NC}"
    go build -o "$BINARY" ./cmd/server
  else
    echo -e "${YELLOW}使用STDIO模式编译...${NC}"
    go build -o "$BINARY" ./cmd/server
  fi
  
  if [[ $? -ne 0 ]]; then
    echo -e "${RED}编译失败，请检查代码和错误信息${NC}"
    exit 1
  fi
  
  echo -e "${GREEN}编译成功: ${BINARY}${NC}"
fi

# 设置环境变量
export RUN_MODE="$RUN_MODE"
if [[ "$RUN_MODE" == "http" ]]; then
  export HTTP_SERVER_PORT="$HTTP_PORT"
fi

# 打印环境变量以便调试
echo -e "${YELLOW}当前环境变量:${NC}"
echo -e "${YELLOW}RUN_MODE=${RUN_MODE}${NC}"
if [[ "$RUN_MODE" == "http" ]]; then
  echo -e "${YELLOW}HTTP_SERVER_PORT=${HTTP_PORT}${NC}"
fi

# 将环境变量写入一个临时文件，方便检查
echo "RUN_MODE=${RUN_MODE}" > ./logs/env_vars.log
if [[ "$RUN_MODE" == "http" ]]; then
  echo "HTTP_SERVER_PORT=${HTTP_PORT}" >> ./logs/env_vars.log
fi

# 如果是HTTP模式，并且命令要求以HTTP/SSE模式启动，则更新MCP配置文件
if [[ "$RUN_MODE" == "http" ]]; then
  MCP_CONFIG_FILE="$HOME/.cursor/mcp.json"
  if [[ -f "$MCP_CONFIG_FILE" ]]; then
    echo -e "${YELLOW}正在更新MCP配置文件以使用Streamable HTTP模式...${NC}"
    # 创建临时文件
    TMP_FILE=$(mktemp)
    # 使用 jq 来更新 JSON 配置，将 command 模式改为 url 模式
    if command -v jq >/dev/null 2>&1; then
      # 使用jq更新配置
      jq --arg url "http://localhost:$HTTP_PORT/mcp" '
        .mcpServers."context-keeper" = {
          "url": $url
        }
      ' "$MCP_CONFIG_FILE" > "$TMP_FILE"
      
      if [[ $? -eq 0 ]]; then
      mv "$TMP_FILE" "$MCP_CONFIG_FILE"
        echo -e "${GREEN}MCP配置已更新为Streamable HTTP模式，端点: http://localhost:$HTTP_PORT/mcp${NC}"
    else
        echo -e "${RED}使用jq更新MCP配置失败${NC}"
        rm -f "$TMP_FILE"
      fi
    else
      # 如果没有jq，提供手动配置提示
      echo -e "${YELLOW}未安装jq，请手动更新MCP配置文件：${NC}"
      echo -e "${YELLOW}将以下配置添加到 $MCP_CONFIG_FILE:${NC}"
      echo -e "${YELLOW}{${NC}"
      echo -e "${YELLOW}  \"mcpServers\": {${NC}"
      echo -e "${YELLOW}    \"context-keeper\": {${NC}"
      echo -e "${YELLOW}      \"url\": \"http://localhost:$HTTP_PORT/mcp\"${NC}"
      echo -e "${YELLOW}    }${NC}"
      echo -e "${YELLOW}  }${NC}"
      echo -e "${YELLOW}}${NC}"
    fi
  else
    echo -e "${YELLOW}MCP配置文件不存在，创建新配置文件: ${MCP_CONFIG_FILE}${NC}"
    mkdir -p "$(dirname "$MCP_CONFIG_FILE")"
    cat > "$MCP_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "context-keeper": {
      "url": "http://localhost:$HTTP_PORT/mcp"
    }
  }
}
EOF
    echo -e "${GREEN}已创建新的MCP配置文件，使用Streamable HTTP模式${NC}"
  fi
  
  echo -e "${YELLOW}📌 重要提示:${NC}"
  echo -e "${YELLOW}1. 服务启动后，请重启Cursor以加载新的MCP配置${NC}"
  echo -e "${YELLOW}2. Streamable HTTP端点: http://localhost:$HTTP_PORT/mcp${NC}"
  echo -e "${YELLOW}3. 能力查询端点: http://localhost:$HTTP_PORT/mcp/capabilities${NC}"
fi

# 创建日志目录
mkdir -p logs

# 根据运行模式和后台设置启动服务
if [[ "$BACKGROUND" == true ]]; then
  echo -e "${GREEN}以后台模式启动 ${BINARY} 服务...${NC}"
  
  echo -e "${YELLOW}使用明确的环境变量启动服务...${NC}"
  nohup "./$BINARY" > logs/service.log 2>&1 &
  
  PID=$!
  
  # 等待服务启动
  echo -e "${YELLOW}等待服务启动...${NC}"
  sleep 2
  
  # 检查进程是否还在运行
  if ps -p $PID > /dev/null; then
    echo -e "${GREEN}服务启动成功！PID: $PID${NC}"
    echo -e "${YELLOW}可以通过查看 logs/service.log 文件了解服务运行状态${NC}"
    
    if [[ "$RUN_MODE" == "http" ]]; then
      echo -e "${YELLOW}服务地址: http://localhost:$HTTP_PORT${NC}"
      # 显示最近的日志内容
      echo -e "${YELLOW}最近的日志内容:${NC}"
      tail -n 15 logs/service.log
    fi
  else
    echo -e "${RED}服务启动失败，查看日志了解详情:${NC}"
    cat logs/service.log
    exit 1
  fi
else
  # 前台启动
  echo -e "${GREEN}以前台模式启动 ${BINARY} 服务...${NC}"
  
  if [[ "$RUN_MODE" == "http" ]]; then
    echo -e "${YELLOW}HTTP服务将在 http://localhost:$HTTP_PORT 上运行${NC}"
    echo -e "${YELLOW}按 Ctrl+C 可停止服务${NC}"
  fi
  
  echo -e "${YELLOW}使用明确的环境变量启动服务...${NC}"
  "./$BINARY"
fi 