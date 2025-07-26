#!/bin/bash

# Context-Keeper Cursor集成安装脚本
# 版本: 2.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
CURSOR_CONFIG_DIR="$HOME/.cursor"
CONTEXT_KEEPER_DIR="$HOME/.context-keeper"
MCP_CONFIG_FILE="$CURSOR_CONFIG_DIR/mcp.json"

echo -e "${BLUE}🧠 Context-Keeper Cursor集成安装程序 v2.0.0${NC}"
echo ""

# 检查依赖
echo -e "${YELLOW}📋 检查系统依赖...${NC}"

# 检查Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ 未找到Node.js，请先安装Node.js (https://nodejs.org/)${NC}"
    exit 1
fi

NODE_VERSION=$(node --version)
echo -e "${GREEN}✅ Node.js已安装: $NODE_VERSION${NC}"

# 检查npm
if ! command -v npm &> /dev/null; then
    echo -e "${RED}❌ 未找到npm，请检查Node.js安装${NC}"
    exit 1
fi

NPM_VERSION=$(npm --version)
echo -e "${GREEN}✅ npm已安装: $NPM_VERSION${NC}"

# 检查Cursor
if [ ! -d "$CURSOR_CONFIG_DIR" ]; then
    echo -e "${YELLOW}⚠️  未找到Cursor配置目录，将创建目录结构${NC}"
    mkdir -p "$CURSOR_CONFIG_DIR"
fi

echo -e "${GREEN}✅ Cursor配置目录存在${NC}"

# 创建Context-Keeper目录
echo -e "${YELLOW}📁 创建Context-Keeper目录...${NC}"
mkdir -p "$CONTEXT_KEEPER_DIR"
mkdir -p "$CONTEXT_KEEPER_DIR/logs"
mkdir -p "$CONTEXT_KEEPER_DIR/config"
mkdir -p "$CONTEXT_KEEPER_DIR/extensions"

# 复制客户端文件
echo -e "${YELLOW}📦 安装客户端文件...${NC}"
cp mcp-client.js "$CONTEXT_KEEPER_DIR/"
cp cursor-extension.js "$CONTEXT_KEEPER_DIR/extensions/"
cp cursor-config-ui.html "$CONTEXT_KEEPER_DIR/config/"

# 安装WebSocket依赖
echo -e "${YELLOW}📦 安装WebSocket依赖...${NC}"
cd "$CONTEXT_KEEPER_DIR"
npm init -y > /dev/null 2>&1
npm install ws > /dev/null 2>&1
echo -e "${GREEN}✅ WebSocket依赖已安装${NC}"
cd - > /dev/null

# 创建包装器脚本
echo -e "${YELLOW}🔧 创建启动脚本...${NC}"
cat > "$CONTEXT_KEEPER_DIR/start-extension.js" << 'EOF'
#!/usr/bin/env node

/**
 * Context-Keeper Cursor扩展启动器
 */

const path = require('path');

// 设置环境变量
process.env.CONTEXT_KEEPER_HOME = __dirname;

// 启动扩展
require('./extensions/cursor-extension.js');

console.log('🧠 Context-Keeper Cursor扩展已启动');
EOF

chmod +x "$CONTEXT_KEEPER_DIR/start-extension.js"

# 配置MCP
echo -e "${YELLOW}⚙️  配置Cursor MCP...${NC}"

# 检查现有MCP配置
if [ -f "$MCP_CONFIG_FILE" ]; then
    echo -e "${YELLOW}📄 发现现有MCP配置，创建备份...${NC}"
    cp "$MCP_CONFIG_FILE" "$MCP_CONFIG_FILE.backup.$(date +%Y%m%d-%H%M%S)"
fi

# 安装MCP配置
cp cursor_mcp_config.json "$MCP_CONFIG_FILE"

echo -e "${GREEN}✅ MCP配置已安装${NC}"

# 创建默认配置
echo -e "${YELLOW}📝 创建默认配置文件...${NC}"

cat > "$CONTEXT_KEEPER_DIR/config/default-config.json" << EOF
{
  "serverConnection": {
    "serverURL": "http://localhost:8088",
    "timeout": 15000
  },
  "userSettings": {
    "userId": "",
    "baseDir": "$CONTEXT_KEEPER_DIR"
  },
  "automationFeatures": {
    "autoCapture": true,
    "autoAssociate": true,
    "autoRecord": true,
    "captureInterval": 30
  },
  "retryConfig": {
    "maxRetries": 3,
    "retryDelay": 1000,
    "backoffMultiplier": 2
  },
  "logging": {
    "enabled": true,
    "level": "info",
    "logToFile": false,
    "logFile": "$CONTEXT_KEEPER_DIR/logs/cursor-extension.log"
  }
}
EOF

# 创建快速启动脚本
echo -e "${YELLOW}🚀 创建快速启动脚本...${NC}"

cat > "$CONTEXT_KEEPER_DIR/quick-start.sh" << 'EOF'
#!/bin/bash

echo "🧠 Context-Keeper 快速启动"
echo ""

# 检查服务器状态
echo "🔍 检查Context-Keeper服务器状态..."
if curl -s http://localhost:8088/health > /dev/null; then
    echo "✅ 服务器运行正常"
else
    echo "❌ 服务器未运行，请先启动Context-Keeper服务器"
    echo "   启动命令: ./scripts/manage.sh start http --port 8088"
    exit 1
fi

# 测试MCP连接
echo "🔗 测试MCP连接..."
node -e "
const Client = require('$CONTEXT_KEEPER_DIR/mcp-client.js');
const client = new Client();
client.healthCheck().then(result => {
    if (result.success) {
        console.log('✅ MCP连接正常');
    } else {
        console.log('❌ MCP连接失败:', result.message);
        process.exit(1);
    }
}).catch(err => {
    console.log('❌ 连接测试失败:', err.message);
    process.exit(1);
});
"

echo ""
echo "🎉 Context-Keeper已准备就绪！"
echo ""
echo "📖 接下来的步骤:"
echo "1. 重启Cursor以加载新的MCP配置"
echo "2. 在Cursor中开始使用Context-Keeper功能"
echo "3. 查看配置界面: file://$PWD/config/cursor-config-ui.html"
EOF

chmod +x "$CONTEXT_KEEPER_DIR/quick-start.sh"

# 创建卸载脚本
cat > "$CONTEXT_KEEPER_DIR/uninstall.sh" << 'EOF'
#!/bin/bash

echo "🗑️  卸载Context-Keeper Cursor集成"
echo ""

read -p "确定要卸载吗？这将删除所有配置和数据 (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # 恢复MCP配置备份
    if [ -f "$HOME/.cursor/mcp.json.backup" ]; then
        mv "$HOME/.cursor/mcp.json.backup" "$HOME/.cursor/mcp.json"
        echo "✅ 已恢复MCP配置备份"
    fi
    
    # 删除Context-Keeper目录
    rm -rf "$HOME/.context-keeper"
    echo "✅ 已删除Context-Keeper目录"
    
    echo "🎉 卸载完成"
else
    echo "❌ 取消卸载"
fi
EOF

chmod +x "$CONTEXT_KEEPER_DIR/uninstall.sh"

# 完成安装
echo ""
echo -e "${GREEN}🎉 安装完成！${NC}"
echo ""
echo -e "${BLUE}📋 安装摘要：${NC}"
echo -e "  • Context-Keeper目录: ${CONTEXT_KEEPER_DIR}"
echo -e "  • MCP配置文件: ${MCP_CONFIG_FILE}"
echo -e "  • 配置界面: file://${CONTEXT_KEEPER_DIR}/config/cursor-config-ui.html"
echo -e "  • 日志目录: ${CONTEXT_KEEPER_DIR}/logs"
echo ""
echo -e "${YELLOW}📖 下一步操作：${NC}"
echo -e "  1. ${YELLOW}重启Cursor以加载新的MCP配置${NC}"
echo -e "  2. ${YELLOW}确保Context-Keeper服务器正在运行${NC}"
echo -e "  3. ${YELLOW}运行快速启动检查: ${CONTEXT_KEEPER_DIR}/quick-start.sh${NC}"
echo -e "  4. ${YELLOW}查看详细文档: README.md${NC}"
echo ""
echo -e "${GREEN}✨ 享受智能编程体验！${NC}" 