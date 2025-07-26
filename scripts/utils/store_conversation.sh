#!/bin/bash

# 设置服务基础URL和会话ID
BASE_URL="http://localhost:8088/api/mcp/context-keeper"
SESSION_ID="cursor-claude-real-conversation-$(date +%s)"
echo "使用会话ID: $SESSION_ID"

# 第一批对话（1-5轮）
echo -e "\n存储第1-5轮对话..."
curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "user",
        "content": "请为handlers.go文件添加消息处理相关代码，实现消息存储和检索的路由及处理函数。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "1",
          "turn": "1"
        }
      },
      {
        "role": "assistant",
        "content": "我将为handlers.go文件添加消息处理相关代码，实现两个主要功能：消息存储和对话检索。我会添加RegisterRoutes方法中的新路由，以及实现handleStoreMessages和handleRetrieveConversation两个处理函数。这些函数将验证请求格式和会话ID，并调用服务层进行相应操作。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "1",
          "turn": "1"
        }
      },
      {
        "role": "user",
        "content": "具体实现中需要验证哪些内容？如何处理错误情况？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "1",
          "turn": "2"
        }
      },
      {
        "role": "assistant",
        "content": "在具体实现中，我们需要验证：1) 请求体格式是否正确；2) 会话ID是否提供；3) 消息列表是否为空等。对于错误情况，会返回相应的HTTP状态码和错误信息，例如400表示请求格式错误，500表示服务器内部错误。每个错误都会有详细说明，方便客户端理解和处理。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "1",
          "turn": "2"
        }
      },
      {
        "role": "user",
        "content": "handleRetrieveConversation函数如何处理查询参数？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "1",
          "turn": "3"
        }
      }
    ]
  }' | jq || echo "无法解析响应"

sleep 1

# 第二批对话（6-10轮）
echo -e "\n存储第6-10轮对话..."
curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "assistant",
        "content": "handleRetrieveConversation函数会处理多种查询参数：1) 必需的sessionID用于定位会话；2) 可选的query参数用于语义搜索相关消息；3) 可选的limit参数限制返回结果数量，如果未提供则使用默认值10；4) 可选的format参数决定结果排序方式，可以是按时间顺序或相关性排序。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "2",
          "turn": "3"
        }
      },
      {
        "role": "user",
        "content": "在代码实现中，如何处理向量搜索和生成？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "2",
          "turn": "4"
        }
      },
      {
        "role": "assistant",
        "content": "在代码实现中，向量搜索和生成主要通过contextService和vectorService来处理。存储消息时，会为每条消息生成向量表示（通过调用vectorService.GenerateEmbedding），然后存储到向量数据库。检索时，先将查询文本转换为向量，再使用向量相似度搜索来找到相关消息。系统会记录生成和搜索的时间消耗，便于性能监控。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "2",
          "turn": "4"
        }
      },
      {
        "role": "user",
        "content": "RetrieveConversation和RetrieveContext有什么区别？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "2",
          "turn": "5"
        }
      },
      {
        "role": "assistant",
        "content": "RetrieveConversation和RetrieveContext有明显区别：RetrieveConversation专注于检索对话消息，返回完整的消息对象列表，包括角色、时间戳等信息，适合显示对话历史。而RetrieveContext更关注提取语义相关的上下文内容，返回多个类别的上下文（会话状态、短期记忆、长期记忆等），适合为大模型提供上下文支持。前者按对话组织，后者按上下文类型组织。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "2",
          "turn": "5"
        }
      }
    ]
  }' | jq || echo "无法解析响应"

sleep 1

# 第三批对话（11-15轮）
echo -e "\n存储第11-15轮对话..."
curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "user",
        "content": "我们如何测试新实现的API功能？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "3",
          "turn": "6"
        }
      },
      {
        "role": "assistant",
        "content": "我们可以创建一个测试脚本来验证API功能。脚本应包括：1) 存储不同角色(用户/助手)的消息；2) 检索所有对话记录；3) 基于关键词检索特定对话；4) 验证返回的消息格式和内容是否正确。我们可以使用curl命令发送HTTP请求，用jq解析JSON响应。这样可以测试API的各项功能和错误处理能力。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "3",
          "turn": "6"
        }
      },
      {
        "role": "user",
        "content": "系统在处理大量消息时会有性能问题吗？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "3",
          "turn": "7"
        }
      },
      {
        "role": "assistant",
        "content": "处理大量消息时确实可能面临性能挑战：1) 向量生成耗时较长；2) 向量搜索随数据量增加而变慢；3) 大型会话的内存开销增加。可能的优化方法包括：批量处理请求、实现向量缓存、优化查询策略、定期归档老旧会话、按优先级管理内存等。系统已实现了性能日志记录，方便发现瓶颈。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "3",
          "turn": "7"
        }
      },
      {
        "role": "user",
        "content": "我建议测试一下这个功能，把我们的对话存储到向量数据库中，然后验证检索是否正常工作。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "3",
          "turn": "8"
        }
      }
    ]
  }' | jq || echo "无法解析响应"

sleep 1

# 第四批对话（16-20轮）
echo -e "\n存储第16-20轮对话..."
curl -s -X POST "${BASE_URL}/storeMessages" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "messages": [
      {
        "role": "assistant",
        "content": "这是个好主意！我们可以将当前的实际对话作为测试数据，存储到向量数据库并测试检索功能。这样不仅能验证API的功能正确性，还能测试系统在真实场景下的表现。我会创建一个测试脚本来模拟我们的对话，包括多轮交互，并使用不同的查询方式来检索内容。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "4",
          "turn": "8"
        }
      },
      {
        "role": "user",
        "content": "考虑到网络请求和向量数据库可能有大小限制，我们应该如何处理大量对话？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "4",
          "turn": "9"
        }
      },
      {
        "role": "assistant",
        "content": "处理大量对话时，应该采用分批存储策略：1) 将长对话按一定数量（如5轮）分批；2) 使用统一的会话ID关联这些批次；3) 添加批次信息到元数据中便于管理；4) 可以实现自动分页机制，当检测到消息数量过多时自动拆分请求。这种方式既能避免请求过大导致失败，又能保持对话的连贯性和完整性。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "4",
          "turn": "9"
        }
      },
      {
        "role": "user",
        "content": "这20轮对话是否足够测试我们的系统功能？",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "4",
          "turn": "10"
        }
      },
      {
        "role": "assistant",
        "content": "20轮对话基本足够测试核心功能，它涵盖了：1) 不同角色的消息存储；2) 多批次存储与关联；3) 基于关键词的检索；4) 完整会话检索。但全面测试还应包括：边界情况处理（空消息、超大消息）、错误恢复能力、性能压力测试等。这次测试可作为基础功能验证，后续可增加更多专项测试。",
        "contentType": "text",
        "priority": "P1",
        "metadata": {
          "source": "真实对话",
          "batch": "4",
          "turn": "10"
        }
      }
    ]
  }' | jq || echo "无法解析响应"

sleep 1

# 检索所有对话
echo -e "\n检索所有存储的对话..."
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "limit": 20
  }' | jq || echo "无法解析响应"

sleep 1

# 按关键词检索对话
echo -e "\n按关键词'向量搜索'检索对话..."
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "query": "向量搜索 数据库",
    "limit": 5
  }' | jq || echo "无法解析响应"

echo -e "\n按关键词'API测试'检索对话..."
curl -s -X POST "${BASE_URL}/retrieveConversation" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionID": "'"${SESSION_ID}"'",
    "query": "测试脚本 功能验证",
    "limit": 5
  }' | jq || echo "无法解析响应"

echo -e "\n测试完成!" 