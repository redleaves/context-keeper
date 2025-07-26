#!/bin/bash

# =============================================================================
# 🔧 ONNX Runtime 环境配置脚本
# 用于解决 fastembed-go 的 ONNX Runtime 依赖问题
# =============================================================================

set -e

echo "🚀 开始配置 ONNX Runtime 环境..."

# 检测操作系统
OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
    Darwin*)    PLATFORM="osx";;
    Linux*)     PLATFORM="linux";;
    *)          echo "❌ 不支持的操作系统: ${OS}"; exit 1;;
esac

case "${ARCH}" in
    x86_64)     ARCH_TYPE="x64";;
    arm64)      ARCH_TYPE="arm64";;
    aarch64)    ARCH_TYPE="arm64";;
    *)          echo "❌ 不支持的架构: ${ARCH}"; exit 1;;
esac

echo "📋 检测到系统: ${PLATFORM}-${ARCH_TYPE}"

# ONNX Runtime 版本
ONNX_VERSION="1.16.3"
ONNX_DIR="onnxruntime"
ONNX_FULL_NAME="onnxruntime-${PLATFORM}-${ARCH_TYPE}-${ONNX_VERSION}"

# 下载URL
if [ "${PLATFORM}" = "osx" ]; then
    DOWNLOAD_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${ONNX_FULL_NAME}.tgz"
    LIB_NAME="libonnxruntime.dylib"
else
    DOWNLOAD_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${ONNX_FULL_NAME}.tgz"
    LIB_NAME="libonnxruntime.so"
fi

echo "📥 下载 ONNX Runtime ${ONNX_VERSION}..."
echo "URL: ${DOWNLOAD_URL}"

# 创建目录
mkdir -p ${ONNX_DIR}
cd ${ONNX_DIR}

# 下载并解压
if [ ! -f "${ONNX_FULL_NAME}.tgz" ]; then
    curl -L -o "${ONNX_FULL_NAME}.tgz" "${DOWNLOAD_URL}"
    echo "✅ 下载完成"
else
    echo "📁 文件已存在，跳过下载"
fi

# 解压
if [ ! -d "${ONNX_FULL_NAME}" ]; then
    tar -xzf "${ONNX_FULL_NAME}.tgz"
    echo "✅ 解压完成"
else
    echo "📁 目录已存在，跳过解压"
fi

# 设置环境变量
LIB_PATH="$(pwd)/${ONNX_FULL_NAME}/lib"
ABS_LIB_PATH="$(cd "${LIB_PATH}" && pwd)"

echo "📍 ONNX Runtime 库路径: ${ABS_LIB_PATH}"

# 验证库文件
if [ -f "${ABS_LIB_PATH}/${LIB_NAME}" ]; then
    echo "✅ ONNX Runtime 库文件存在: ${LIB_NAME}"
    ls -la "${ABS_LIB_PATH}/${LIB_NAME}"
else
    echo "❌ ONNX Runtime 库文件不存在: ${ABS_LIB_PATH}/${LIB_NAME}"
    echo "📋 目录内容:"
    ls -la "${ABS_LIB_PATH}/"
    exit 1
fi

# 创建环境配置文件
cd ..
ENV_FILE=".env.onnx"
cat > "${ENV_FILE}" << EOF
# ONNX Runtime 环境配置
export ONNX_RUNTIME_PATH="${ABS_LIB_PATH}"
export LD_LIBRARY_PATH="${ABS_LIB_PATH}:\$LD_LIBRARY_PATH"
export DYLD_LIBRARY_PATH="${ABS_LIB_PATH}:\$DYLD_LIBRARY_PATH"

# Go CGO 环境
export CGO_CFLAGS="-I${ABS_LIB_PATH}/../include"
export CGO_LDFLAGS="-L${ABS_LIB_PATH} -lonnxruntime"

echo "✅ ONNX Runtime 环境已加载"
echo "📍 库路径: ${ABS_LIB_PATH}"
EOF

echo "✅ 环境配置文件已创建: ${ENV_FILE}"

# 创建测试脚本
cat > "test_onnx_setup.sh" << 'EOF'
#!/bin/bash
echo "🧪 测试 ONNX Runtime 环境..."

# 加载环境
source .env.onnx

# 检查环境变量
echo "📋 环境变量检查:"
echo "ONNX_RUNTIME_PATH: ${ONNX_RUNTIME_PATH}"
echo "LD_LIBRARY_PATH: ${LD_LIBRARY_PATH}"
echo "DYLD_LIBRARY_PATH: ${DYLD_LIBRARY_PATH}"

# 检查库文件
if [ -f "${ONNX_RUNTIME_PATH}/libonnxruntime.so" ]; then
    echo "✅ Linux 库文件存在"
elif [ -f "${ONNX_RUNTIME_PATH}/libonnxruntime.dylib" ]; then
    echo "✅ macOS 库文件存在"
else
    echo "❌ 库文件不存在"
    exit 1
fi

# 测试 fastembed-go
echo "🧪 测试 fastembed-go 编译..."
cd test_fastembed
if go build -o test_fastembed main.go; then
    echo "✅ fastembed-go 编译成功"
    if ./test_fastembed; then
        echo "✅ fastembed-go 运行成功"
    else
        echo "⚠️ fastembed-go 运行失败，但编译成功"
    fi
else
    echo "❌ fastembed-go 编译失败"
fi
EOF

chmod +x test_onnx_setup.sh

echo ""
echo "🎉 ONNX Runtime 环境配置完成！"
echo ""
echo "📋 下一步操作:"
echo "1. 加载环境: source .env.onnx"
echo "2. 测试环境: ./test_onnx_setup.sh"
echo "3. 运行您的程序"
echo ""
echo "🔧 永久配置 (可选):"
echo "将以下内容添加到 ~/.bashrc 或 ~/.zshrc:"
echo "source $(pwd)/.env.onnx"
echo ""
echo "✅ 安装完成！" 