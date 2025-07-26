#!/bin/bash

# Context-Keeper 日志查看脚本
# 用于查看运行中服务的实时日志

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 获取工作目录
WORKSPACE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

# 默认配置
LOG_LINES=50
FOLLOW=false
LOG_TYPE="all"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --follow|-f)
      FOLLOW=true
      shift
      ;;
    --lines|-n)
      LOG_LINES="$2"
      shift 2
      ;;
    --stdio)
      LOG_TYPE="stdio"
      shift
      ;;
    --http)
      LOG_TYPE="http"
      shift
      ;;
    --streamable)
      LOG_TYPE="streamable-http"
      shift
      ;;
    --help|-h)
      echo "Context-Keeper 日志查看脚本"
      echo ""
      echo "用法: ./logs.sh [选项]"
      echo ""
      echo "选项:"
      echo "  --follow, -f         跟踪实时日志输出"
      echo "  --lines N, -n N      显示最后N行日志（默认: 50）"
      echo "  --stdio              仅显示STDIO模式日志"
      echo "  --http               仅显示HTTP模式日志"
      echo "  --streamable         仅显示Streamable HTTP模式日志"
      echo "  --help, -h           显示此帮助信息"
      echo ""
      echo "示例:"
      echo "  ./logs.sh --follow                    # 跟踪所有日志"
      echo "  ./logs.sh --streamable --follow       # 跟踪Streamable HTTP日志"
      echo "  ./logs.sh --lines 100                 # 显示最后100行日志"
      exit 0
      ;;
    *)
      echo "未知参数: $1"
      echo "使用 --help 查看帮助"
      exit 1
      ;;
  esac
done

# 切换到工作目录
cd "$WORKSPACE_DIR" || { echo -e "${RED}无法进入工作目录${NC}"; exit 1; }

# 检查日志目录
if [[ ! -d "logs" ]]; then
  echo -e "${RED}日志目录不存在: logs/${NC}"
  echo -e "${YELLOW}请先启动Context-Keeper服务${NC}"
  exit 1
fi

# 根据日志类型选择日志文件
LOG_FILES=()
case $LOG_TYPE in
  "stdio")
    if [[ -f "logs/service.log" ]]; then
      LOG_FILES+=("logs/service.log")
    fi
    if [[ -f "logs/context-keeper.log" ]]; then
      LOG_FILES+=("logs/context-keeper.log")
    fi
    ;;
  "http")
    if [[ -f "logs/service.log" ]]; then
      LOG_FILES+=("logs/service.log")
    fi
    if [[ -f "logs/context-keeper-http.log" ]]; then
      LOG_FILES+=("logs/context-keeper-http.log")
    fi
    ;;
  "streamable-http")
    if [[ -f "logs/streamable-http.log" ]]; then
      LOG_FILES+=("logs/streamable-http.log")
    fi
    ;;
  "all")
    # 自动检测可用的日志文件
    for log_file in "logs/service.log" "logs/streamable-http.log" "logs/context-keeper.log" "logs/context-keeper-http.log"; do
      if [[ -f "$log_file" ]]; then
        LOG_FILES+=("$log_file")
      fi
    done
    ;;
esac

# 检查是否找到日志文件
if [[ ${#LOG_FILES[@]} -eq 0 ]]; then
  echo -e "${RED}未找到任何日志文件${NC}"
  echo -e "${YELLOW}可用的日志文件类型:${NC}"
  echo -e "${YELLOW}  --stdio      STDIO模式日志${NC}"
  echo -e "${YELLOW}  --http       HTTP模式日志${NC}"
  echo -e "${YELLOW}  --streamable Streamable HTTP模式日志${NC}"
  echo ""
  echo -e "${YELLOW}请确保服务已启动并生成了日志文件${NC}"
  exit 1
fi

# 显示日志信息
echo -e "${BLUE}=== Context-Keeper 日志查看器 ===${NC}"
echo -e "${YELLOW}工作目录: ${WORKSPACE_DIR}${NC}"
echo -e "${YELLOW}日志文件: ${LOG_FILES[*]}${NC}"

if [[ $FOLLOW == true ]]; then
  echo -e "${YELLOW}模式: 实时跟踪${NC}"
  echo -e "${YELLOW}按 Ctrl+C 停止跟踪${NC}"
else
  echo -e "${YELLOW}模式: 显示最后 ${LOG_LINES} 行${NC}"
fi

echo -e "${BLUE}=== 日志内容 ===${NC}"
echo ""

# 显示日志
if [[ $FOLLOW == true ]]; then
  # 实时跟踪日志
  if [[ ${#LOG_FILES[@]} -eq 1 ]]; then
    tail -f "${LOG_FILES[0]}"
  else
    # 多个日志文件，使用 tail -f 跟踪所有文件
    tail -f "${LOG_FILES[@]}"
  fi
else
  # 显示指定行数的日志
  for log_file in "${LOG_FILES[@]}"; do
    if [[ ${#LOG_FILES[@]} -gt 1 ]]; then
      echo -e "${BLUE}==> ${log_file} <==${NC}"
    fi
    tail -n "$LOG_LINES" "$log_file"
    if [[ ${#LOG_FILES[@]} -gt 1 ]]; then
      echo ""
    fi
  done
fi 