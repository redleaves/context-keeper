#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
  echo -e "${GREEN}[Context-Keeper打包]${NC} $1"
}

print_error() {
  echo -e "${RED}[错误]${NC} $1"
}

print_step() {
  echo -e "${BLUE}[步骤]${NC} $1"
}

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# 创建临时目录
TEMP_DIR="$PROJECT_ROOT/tmp/installer-build"
mkdir -p "$TEMP_DIR"

# 版本信息
VERSION=$(date +"%Y%m%d")
if [ -f "$PROJECT_ROOT/VERSION" ]; then
  VERSION=$(cat "$PROJECT_ROOT/VERSION")
fi

# 创建安装包目录结构
print_step "创建安装包目录结构..."
PACKAGE_DIR="$TEMP_DIR/context-keeper-$VERSION"
mkdir -p "$PACKAGE_DIR/bin"
mkdir -p "$PACKAGE_DIR/config"

# 复制安装脚本和说明文档
print_step "复制安装脚本和文档..."
cp "$SCRIPT_DIR/install-package/install.sh" "$PACKAGE_DIR/"
cp "$SCRIPT_DIR/install-package/install.bat" "$PACKAGE_DIR/"
cp "$SCRIPT_DIR/install-package/README.md" "$PACKAGE_DIR/"
chmod +x "$PACKAGE_DIR/install.sh"

# 复制配置文件
print_step "复制配置文件..."
cp -r "$SCRIPT_DIR/install-package/config/"* "$PACKAGE_DIR/config/"

# 检测操作系统并复制合适的二进制文件
OS="$(uname -s)"
ARCH="$(uname -m)"
BIN_DIR="$PROJECT_ROOT/bin"

print_step "检测操作系统: $OS ($ARCH)"

# 复制当前系统的二进制文件
if [ "$OS" = "Darwin" ]; then
  if [ "$ARCH" = "arm64" ]; then
    print_step "复制 macOS ARM64 二进制文件..."
    cp "$BIN_DIR/context-keeper" "$PACKAGE_DIR/bin/context-keeper"
    # 创建Windows和Linux的空文件，以便压缩包内有占位符
    touch "$PACKAGE_DIR/bin/context-keeper.exe"
    touch "$PACKAGE_DIR/bin/context-keeper-linux-amd64"
  else
    print_step "复制 macOS x86_64 二进制文件..."
    cp "$BIN_DIR/context-keeper" "$PACKAGE_DIR/bin/context-keeper"
    # 创建Windows和Linux的空文件，以便压缩包内有占位符
    touch "$PACKAGE_DIR/bin/context-keeper.exe"
    touch "$PACKAGE_DIR/bin/context-keeper-linux-amd64"
  fi
elif [ "$OS" = "Linux" ]; then
  print_step "复制 Linux 二进制文件..."
  cp "$BIN_DIR/context-keeper" "$PACKAGE_DIR/bin/context-keeper-linux-amd64"
  # 创建macOS和Windows的空文件，以便压缩包内有占位符
  touch "$PACKAGE_DIR/bin/context-keeper"
  touch "$PACKAGE_DIR/bin/context-keeper.exe"
else
  print_error "不支持的操作系统: $OS"
  exit 1
fi

# 添加版本文件
echo "$VERSION" > "$PACKAGE_DIR/VERSION"

# 创建压缩包
print_step "创建安装包..."
OUTPUT_DIR="$PROJECT_ROOT/dist"
mkdir -p "$OUTPUT_DIR"

ZIP_NAME="context-keeper-$VERSION.zip"
ZIP_PATH="$OUTPUT_DIR/$ZIP_NAME"

cd "$TEMP_DIR"
zip -r "$ZIP_PATH" "context-keeper-$VERSION"

if [ $? -eq 0 ]; then
  print_message "安装包创建成功: $ZIP_PATH"
  # 清理临时文件
  rm -rf "$TEMP_DIR"
else
  print_error "创建安装包失败"
  exit 1
fi

print_message "打包完成！" 