#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}开始测试Context-Keeper完整工作流程...${NC}"

# 基础URL设置
BASE_URL="http://localhost:8088"
MCP_BASE_URL="$BASE_URL/api/mcp/context-keeper"

# 生成唯一会话ID
SESSION_ID="test-session-$(date +%s)"
echo -e "${YELLOW}使用会话ID: $SESSION_ID${NC}"

# 测试健康检查
echo -e "\n${YELLOW}测试1：健康检查...${NC}"
HEALTH_RESPONSE=$(curl -s "$BASE_URL/health")
echo "响应: $HEALTH_RESPONSE"
if [[ "$HEALTH_RESPONSE" == *"healthy"* ]]; then
  echo -e "${GREEN}健康检查通过！${NC}"
else
  echo -e "${RED}健康检查失败！${NC}"
  exit 1
fi

# 测试关联文件
echo -e "\n${YELLOW}测试2：关联JavaScript文件...${NC}"
ASSOCIATE_RESPONSE=$(curl -s -X POST "$MCP_BASE_URL/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/test/server.js",
    "content": "const express = require(\"express\");\nconst app = express();\nconst port = 3000;\n\napp.use(express.json());\n\napp.get(\"/\", (req, res) => {\n  res.send(\"Hello World!\");\n});\n\napp.listen(port, () => {\n  console.log(`Server running at http://localhost:${port}`);\n});"
  }')
echo "响应: $ASSOCIATE_RESPONSE"
if [[ "$ASSOCIATE_RESPONSE" == *"success"* ]]; then
  echo -e "${GREEN}文件关联成功！${NC}"
else
  echo -e "${RED}文件关联失败！${NC}"
fi

# 测试记录编辑
echo -e "\n${YELLOW}测试3：记录编辑操作...${NC}"
EDIT_RESPONSE=$(curl -s -X POST "$MCP_BASE_URL/recordEdit" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/test/server.js",
    "type": "insert",
    "position": 182,
    "content": "\n\napp.get(\"/api/users\", (req, res) => {\n  res.json([{id: 1, name: \"John\"}, {id: 2, name: \"Jane\"}]);\n});\n"
  }')
echo "响应: $EDIT_RESPONSE"
if [[ "$EDIT_RESPONSE" == *"success"* ]]; then
  echo -e "${GREEN}编辑操作记录成功！${NC}"
else
  echo -e "${RED}编辑操作记录失败！${NC}"
fi

# 测试存储上下文
echo -e "\n${YELLOW}测试4：存储代码上下文...${NC}"
STORE_RESPONSE=$(curl -s -X POST "$MCP_BASE_URL/storeContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "content": "在Express应用中，我想添加一个用于用户认证的路由。",
    "type": "programming_question",
    "priority": "high"
  }')
echo "响应: $STORE_RESPONSE"
if [[ "$STORE_RESPONSE" == *"memoryId"* ]]; then
  echo -e "${GREEN}上下文存储成功！${NC}"
  # 提取memoryId以便后续使用
  MEMORY_ID=$(echo $STORE_RESPONSE | grep -o '"memoryId":"[^"]*' | sed 's/"memoryId":"//')
  echo "存储ID: $MEMORY_ID"
else
  echo -e "${RED}上下文存储失败！${NC}"
fi

# 测试检索上下文（带查询）
echo -e "\n${YELLOW}测试5：检索代码上下文（带查询）...${NC}"
RETRIEVE_RESPONSE=$(curl -s -X POST "$MCP_BASE_URL/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "query": "如何在Express中实现用户认证？"
  }')
echo "响应摘要: $(echo "$RETRIEVE_RESPONSE" | head -c 300)..."
if [[ "$RETRIEVE_RESPONSE" == *"session_state"* ]]; then
  echo -e "${GREEN}上下文检索成功！${NC}"
else
  echo -e "${RED}上下文检索失败！${NC}"
fi

# 测试获取会话状态
echo -e "\n${YELLOW}测试6：获取会话状态...${NC}"
SESSION_STATE_RESPONSE=$(curl -s "$MCP_BASE_URL/sessionState?sessionId=$SESSION_ID")
echo "响应: $SESSION_STATE_RESPONSE"
if [[ "$SESSION_STATE_RESPONSE" == *"sessionId"* ]]; then
  echo -e "${GREEN}会话状态获取成功！${NC}"
else
  echo -e "${RED}会话状态获取失败！${NC}"
fi

# 测试关联另一个文件
echo -e "\n${YELLOW}测试7：关联另一个文件...${NC}"
ASSOCIATE_RESPONSE2=$(curl -s -X POST "$MCP_BASE_URL/associateFile" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "filePath": "/projects/test/auth.js",
    "content": "const jwt = require(\"jsonwebtoken\");\nconst bcrypt = require(\"bcrypt\");\n\nconst SECRET_KEY = \"your-secret-key\";\n\nfunction generateToken(user) {\n  return jwt.sign({ id: user.id, username: user.username }, SECRET_KEY, {\n    expiresIn: \"2h\",\n  });\n}\n\nfunction verifyPassword(password, hashedPassword) {\n  return bcrypt.compareSync(password, hashedPassword);\n}\n\nmodule.exports = { generateToken, verifyPassword };"
  }')
echo "响应: $ASSOCIATE_RESPONSE2"
if [[ "$ASSOCIATE_RESPONSE2" == *"success"* ]]; then
  echo -e "${GREEN}第二个文件关联成功！${NC}"
else
  echo -e "${RED}第二个文件关联失败！${NC}"
fi

# 测试基于新关联文件的上下文检索
echo -e "\n${YELLOW}测试8：基于新关联文件的上下文检索...${NC}"
CONTEXT_RESPONSE=$(curl -s -X POST "$MCP_BASE_URL/retrieveContext" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "'"$SESSION_ID"'",
    "query": "如何使用JWT进行用户认证？"
  }')
echo "响应摘要: $(echo "$CONTEXT_RESPONSE" | head -c 300)..."
if [[ "$CONTEXT_RESPONSE" == *"jwt"* ]]; then
  echo -e "${GREEN}基于新文件的上下文检索成功！${NC}"
else
  echo -e "${RED}基于新文件的上下文检索失败！${NC}"
fi

echo -e "\n${GREEN}测试完成！${NC}"
echo -e "${YELLOW}会话ID: $SESSION_ID 可用于在Cursor中进行进一步测试${NC}" 