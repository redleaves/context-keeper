#!/bin/bash

# Context-Keeper代码上下文管理功能测试脚本启动器

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
  echo -e "${GREEN}[Context-Keeper测试]${NC} $1"
}

print_error() {
  echo -e "${RED}[错误]${NC} $1"
}

print_step() {
  echo -e "${BLUE}[步骤]${NC} $1"
}

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# 检查Context-Keeper服务是否运行
print_step "检查Context-Keeper服务状态..."
curl -s http://localhost:8088/api/health > /dev/null
if [ $? -ne 0 ]; then
    print_error "Context-Keeper服务似乎没有运行。请先启动服务!"
    exit 1
else
    print_message "Context-Keeper服务正在运行"
fi

# 执行Python测试脚本
print_step "启动测试脚本..."
python3 "$SCRIPT_DIR/test_context_keeper.py"

exit_code=$?
if [ $exit_code -eq 0 ]; then
    print_message "测试完成"
else
    print_error "测试失败，退出码: $exit_code"
fi 