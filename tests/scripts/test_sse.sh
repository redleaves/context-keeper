#!/bin/bash

echo "测试Context-Keeper SSE端点..."
curl -N http://localhost:8088/mcp/sse

echo -e "\n\n测试健康检查端点..."
curl http://localhost:8088/health 