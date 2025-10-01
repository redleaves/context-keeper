#!/usr/bin/env node

/**
 * Context-Keeper 查询逻辑调试测试
 * 专门测试查询功能，看看是否调用了本地存储逻辑
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class QueryDebugTest {
    constructor() {
        this.baseURL = 'http://localhost:8088';
        this.userId = 'user_1703123456';
        this.sessionId = null;
        this.baseDir = path.join(os.homedir(), '.context-keeper');
    }

    async sendMCPRequest(method, params = {}) {
        let requestData;
        
        if (method === 'session_management' || method === 'store_conversation' || method === 'retrieve_context') {
            // 对于工具调用，使用 tools/call 格式
            requestData = {
                jsonrpc: '2.0',
                id: Date.now(),
                method: 'tools/call',
                params: {
                    name: method,
                    arguments: params
                }
            };
        } else {
            // 其他方法保持原有格式
            requestData = {
                jsonrpc: '2.0',
                id: Date.now(),
                method: method,
                params: params
            };
        }

        console.log(`🔍 发送请求: ${method}`);
        console.log(`📝 参数:`, JSON.stringify(params, null, 2));

        const response = await fetch(`${this.baseURL}/mcp`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status} ${response.statusText}`);
        }

        const result = await response.json();
        console.log(`📋 响应:`, JSON.stringify(result, null, 2));
        
        if (result.error) {
            throw new Error(`MCP错误: ${result.error.message}`);
        }

        // 处理返回结果中的本地指令
        const mcpResult = result.result;
        if (mcpResult && mcpResult.content && mcpResult.content[0] && mcpResult.content[0].text) {
            try {
                const parsedResponse = JSON.parse(mcpResult.content[0].text);
                if (parsedResponse.localInstruction) {
                    console.log(`🔧 检测到本地指令: ${parsedResponse.localInstruction.type}`);
                    await this.executeLocalInstruction(parsedResponse.localInstruction);
                }
                return parsedResponse;
            } catch (e) {
                // 如果不是JSON，直接返回
                return mcpResult;
            }
        }

        return result.result;
    }

    async executeLocalInstruction(instruction) {
        if (!instruction) return { success: true };

        console.log(`🔧 执行本地指令: ${instruction.type} -> ${instruction.target}`);
        
        try {
            const targetPath = this.expandPath(instruction.target);
            
            // 确保目录存在
            await this.ensureDirectory(path.dirname(targetPath));

            // 根据指令类型执行相应操作
            switch (instruction.type) {
                case 'short_memory':
                    await this.handleShortMemory(instruction, targetPath);
                    break;
                default:
                    console.log(`⚠️  未知指令类型: ${instruction.type}`);
                    return { success: false, error: `未知指令类型: ${instruction.type}` };
            }

            console.log(`✅ 本地指令执行成功: ${instruction.type}`);
            return { success: true, filePath: targetPath };
            
        } catch (error) {
            console.error(`❌ 本地指令执行失败:`, error.message);
            return { success: false, error: error.message };
        }
    }

    async handleShortMemory(instruction, targetPath) {
        const content = instruction.content;
        const options = instruction.options || {};

        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            // 读取现有内容
            const existingContent = await fs.readFile(targetPath, 'utf8');
            const existingData = JSON.parse(existingContent);
            
            // 合并新内容
            const mergedData = [...existingData, ...content];
            await fs.writeFile(targetPath, JSON.stringify(mergedData, null, 2));
            console.log(`💾 短期记忆已合并: ${mergedData.length}条记录到 ${targetPath}`);
        } else {
            // 直接写入
            await fs.writeFile(targetPath, JSON.stringify(content, null, 2));
            console.log(`💾 短期记忆已保存: ${content.length}条记录到 ${targetPath}`);
        }
    }

    expandPath(pathTemplate) {
        return pathTemplate.replace('~', os.homedir());
    }

    async ensureDirectory(dirPath) {
        await fs.mkdir(dirPath, { recursive: true });
    }

    async fileExists(filePath) {
        try {
            await fs.access(filePath);
            return true;
        } catch {
            return false;
        }
    }

    async createTestData() {
        console.log('\n📝 创建测试数据...');

        // 1. 创建会话
        const sessionResponse = await this.sendMCPRequest('session_management', {
            action: 'create'
        });
        
        // 解析会话ID（从返回的结果中提取）
        if (sessionResponse && sessionResponse.sessionId) {
            this.sessionId = sessionResponse.sessionId;
        } else if (sessionResponse && sessionResponse.content && sessionResponse.content[0] && sessionResponse.content[0].text) {
            const textResult = JSON.parse(sessionResponse.content[0].text);
            this.sessionId = textResult.sessionId;
        }
        console.log(`✅ 会话创建成功: ${this.sessionId}`);

        // 2. 存储一些短期记忆
        const messages = [
            {role: 'user', content: '我想学习Go语言编程'},
            {role: 'assistant', content: '好的，我来帮你学习Go语言。Go是一种简洁、高效的编程语言。'}
        ];

        await this.sendMCPRequest('store_conversation', {
            sessionId: this.sessionId,
            messages: messages
        });

        console.log('✅ 测试数据创建完成');

        // 3. 验证本地文件存在
        const historyPath = path.join(this.baseDir, 'users', this.userId, 'histories', `${this.sessionId}.json`);
        console.log(`🔍 检查本地文件: ${historyPath}`);
        
        try {
            const content = await fs.readFile(historyPath, 'utf8');
            const history = JSON.parse(content);
            console.log(`✅ 本地文件存在，包含 ${history.length} 条记录`);
            console.log(`📄 内容预览: ${history[0]}`);
        } catch (error) {
            console.log(`❌ 本地文件读取失败: ${error.message}`);
        }
    }

    async testQuery() {
        console.log('\n🔍 测试查询功能...');

        try {
            const response = await this.sendMCPRequest('retrieve_context', {
                sessionId: this.sessionId,
                query: '学习Go语言'
            });

            console.log('\n📊 查询结果分析:');
            console.log(`🔹 短期记忆: ${response.shortTermMemory || '无'}`);
            console.log(`🔹 长期记忆: ${response.longTermMemory || '无'}`);
            
            // 检查是否包含预期内容
            const hasShortTermMemory = response.shortTermMemory && !response.shortTermMemory.includes('无相关内容');
            const hasExpectedContent = response.shortTermMemory && response.shortTermMemory.includes('Go语言');

            console.log(`\n✅ 短期记忆状态: ${hasShortTermMemory ? '有数据' : '无数据'}`);
            console.log(`✅ 包含预期内容: ${hasExpectedContent ? '是' : '否'}`);

            return hasShortTermMemory && hasExpectedContent;

        } catch (error) {
            console.log(`❌ 查询失败: ${error.message}`);
            return false;
        }
    }

    async cleanup() {
        if (this.sessionId) {
            console.log('\n🧹 清理测试数据...');
            const historyPath = path.join(this.baseDir, 'users', this.userId, 'histories', `${this.sessionId}.json`);
            try {
                await fs.unlink(historyPath);
                console.log('✅ 测试文件已删除');
            } catch (error) {
                console.log(`⚠️  删除测试文件失败: ${error.message}`);
            }
        }
    }

    async run() {
        console.log('🚀 Context-Keeper 查询逻辑调试测试\n');

        try {
            await this.createTestData();
            
            // 等待一秒确保数据已保存
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            const success = await this.testQuery();
            
            console.log('\n📊 测试结果:');
            console.log(`🎯 查询功能: ${success ? '✅ 正常' : '❌ 异常'}`);
            
            if (!success) {
                console.log('\n🔧 调试建议:');
                console.log('1. 检查服务日志中是否有 "成功从本地读取短期记忆" 信息');
                console.log('2. 检查 getLocalShortTermMemory 方法是否被调用');
                console.log('3. 验证用户ID缓存是否正确设置');
            }

        } catch (error) {
            console.log(`💥 测试失败: ${error.message}`);
        } finally {
            await this.cleanup();
        }
    }
}

// 运行测试
if (require.main === module) {
    const test = new QueryDebugTest();
    test.run().catch(console.error);
}

module.exports = QueryDebugTest; 