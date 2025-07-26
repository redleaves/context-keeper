#!/bin/bash

# Context-Keeper 安装脚本
# 适用于 macOS 和 Linux 系统

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
  echo -e "${GREEN}[Context-Keeper安装]${NC} $1"
}

print_error() {
  echo -e "${RED}[错误]${NC} $1"
}

print_step() {
  echo -e "${BLUE}[步骤]${NC} $1"
}

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# 安装目录设置
DEFAULT_INSTALL_DIR="$HOME/.context-keeper"
DATA_DIR="$DEFAULT_INSTALL_DIR/data"
CONFIG_DIR="$DEFAULT_INSTALL_DIR/config"
BIN_DIR="$DEFAULT_INSTALL_DIR/bin"

# 检测操作系统
OS="$(uname -s)"
case "${OS}" in
    Darwin*)    OS="macos";;
    Linux*)     OS="linux";;
    *)          OS="unknown";;
esac

# 检测架构
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64*)    ARCH="amd64";;
    arm64*)     ARCH="arm64";;
    *)          ARCH="unknown";;
esac

print_message "欢迎安装 Context-Keeper！"
print_message "检测到系统: ${OS} (${ARCH})"

# 询问安装目录
read -p "请输入安装目录 [默认: $DEFAULT_INSTALL_DIR]: " INSTALL_DIR
INSTALL_DIR=${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}

# 创建目录
print_step "创建必要的目录..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/bin"
mkdir -p "$INSTALL_DIR/config"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/logs"

# 复制文件
print_step "复制程序文件..."
cp "$SCRIPT_DIR/bin/context-keeper" "$INSTALL_DIR/bin/"
chmod +x "$INSTALL_DIR/bin/context-keeper"

# 复制配置文件
if [ -d "$SCRIPT_DIR/config" ]; then
  print_step "复制配置文件..."
  cp -r "$SCRIPT_DIR/config/"* "$INSTALL_DIR/config/"
fi

# 创建启动脚本
print_step "创建启动脚本..."
cat > "$INSTALL_DIR/start-context-keeper.sh" << EOF
#!/bin/bash
$INSTALL_DIR/bin/context-keeper
EOF
chmod +x "$INSTALL_DIR/start-context-keeper.sh"

# 验证安装
print_step "验证安装..."
echo "运行版本检查:"
"$INSTALL_DIR/bin/context-keeper" --version || {
  print_error "无法执行 context-keeper。请确保您有执行权限和所需的系统依赖。"
  exit 1
}

# 完成安装
print_message "安装完成！Context-Keeper 已安装到 $INSTALL_DIR"
print_message "请在 Cursor 编辑器中配置 MCP 服务器路径为:"
echo "$INSTALL_DIR/bin/context-keeper"

# 添加使用提示
cat << EOF

===== 使用方法 =====
1. 打开 Cursor 编辑器设置 (Cmd+, 或 Ctrl+,)
2. 搜索 "MCP" 或 "Model Context Protocol"
3. 在 MCP 服务器配置部分添加以下路径:
   $INSTALL_DIR/bin/context-keeper
4. 保存设置并重启 Cursor

查看更多使用说明，请参考 README.md 文件。
EOF

print_message "感谢使用 Context-Keeper！" 