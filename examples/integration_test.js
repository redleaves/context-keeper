/**
 * Context-Keeper MCP工具指令模式集成测试
 * 测试云端服务与本地存储的完整集成
 */

const { LocalStorageClient } = require('./local_storage_client.js');

class IntegrationTest {
    constructor() {
        this.serverUrl = 'http://localhost:8088';
        this.localClient = new LocalStorageClient();
        this.testSessionId = `integration-test-${Date.now()}`;
    }

    /**
     * 发送MCP工具请求到服务端
     */
    async callMCPTool(toolName, params) {
        const requestBody = {
            jsonrpc: "2.0",
            id: Date.now(),
            method: "tools/call",
            params: {
                name: toolName,
                arguments: params
            }
        };

        console.log(`🔧 调用MCP工具: ${toolName}`);
        console.log(`📤 请求参数:`, JSON.stringify(params, null, 2));

        try {
            const response = await fetch(`${this.serverUrl}/mcp`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestBody)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const mcpResult = await response.json();
            console.log(`📥 服务端MCP响应:`, JSON.stringify(mcpResult, null, 2));

            // 解析MCP标准格式的响应
            if (mcpResult.result && mcpResult.result.content && mcpResult.result.content[0]) {
                const textContent = mcpResult.result.content[0].text;
                try {
                    const parsedResult = JSON.parse(textContent);
                    console.log(`📋 解析后的结果:`, JSON.stringify(parsedResult, null, 2));
                    return { mcpResult, parsedResult };
                } catch (parseError) {
                    console.warn(`⚠️ 无法解析响应内容为JSON: ${parseError.message}`);
                    return { mcpResult, parsedResult: { text: textContent } };
                }
            }

            return { mcpResult, parsedResult: null };
        } catch (error) {
            console.error(`❌ MCP工具调用失败: ${error.message}`);
            throw error;
        }
    }

    /**
     * 处理服务端返回的本地存储指令
     */
    async processLocalInstruction(result) {
        const { parsedResult } = result;
        
        if (parsedResult && parsedResult.localInstruction) {
            const instruction = parsedResult.localInstruction;
            console.log(`🔄 处理本地存储指令: ${instruction.type}`);
            await this.localClient.executeLocalInstruction(instruction);
            return true;
        }
        return false;
    }

    /**
     * 测试1: 会话管理与本地存储
     */
    async testSessionManagement() {
        console.log('\n🧪 测试1: 会话管理与本地存储');
        
        const result = await this.callMCPTool('session_management', {
            action: 'get_or_create',
            userId: 'user_weixiaofeng', // 必需参数
            workspaceRoot: process.cwd(), // 必需参数，当前工作目录
            sessionId: this.testSessionId
        });

        // 处理本地存储指令
        await this.processLocalInstruction(result);

        // 验证会话是否创建成功
        if (result.parsedResult && result.parsedResult.sessionId) {
            console.log(`✅ 会话创建成功: ${result.parsedResult.sessionId}`);
            return true;
        } else {
            console.log(`❌ 会话创建失败`);
            return false;
        }
    }

    /**
     * 测试2: 长期记忆存储与本地指令
     */
    async testMemorizeContext() {
        console.log('\n🧪 测试2: 长期记忆存储与本地指令');

        const testContent = `集成测试记忆内容 - ${new Date().toISOString()}`;
        
        const result = await this.callMCPTool('memorize_context', {
            sessionId: this.testSessionId,
            content: testContent,
            priority: 'P1'
        });

        // 处理本地存储指令
        await this.processLocalInstruction(result);

        // 验证记忆是否存储成功
        if (result.parsedResult && result.parsedResult.memoryId) {
            console.log(`✅ 长期记忆存储成功: ${result.parsedResult.memoryId}`);
            return true;
        } else {
            console.log(`❌ 长期记忆存储失败`);
            return false;
        }
    }

    /**
     * 测试3: 短期记忆存储与本地文件
     */
    async testStoreConversation() {
        console.log('\n🧪 测试3: 短期记忆存储与本地文件');

        const testMessages = [
            {
                role: 'user',
                content: '这是集成测试消息',
                timestamp: Math.floor(Date.now() / 1000)
            },
            {
                role: 'assistant', 
                content: '这是测试回复',
                timestamp: Math.floor(Date.now() / 1000)
            }
        ];

        const result = await this.callMCPTool('store_conversation', {
            sessionId: this.testSessionId,
            messages: testMessages
        });

        // 处理本地存储指令
        const hasLocalInstruction = await this.processLocalInstruction(result);

        // 验证对话是否存储成功
        if (result.parsedResult && result.parsedResult.status === 'success') {
            console.log(`✅ 短期记忆存储成功${hasLocalInstruction ? ' (包含本地指令)' : ''}`);
            return true;
        } else {
            console.log(`❌ 短期记忆存储失败`);
            return false;
        }
    }

    /**
     * 测试4: 代码文件关联与本地指令
     */
    async testAssociateFile() {
        console.log('\n🧪 测试4: 代码文件关联与本地指令');

        const result = await this.callMCPTool('associate_file', {
            sessionId: this.testSessionId,
            filePath: 'examples/integration_test.js'
        });

        // 处理本地存储指令
        await this.processLocalInstruction(result);

        // 验证文件关联是否成功
        if (result.parsedResult && result.parsedResult.status === 'success') {
            console.log(`✅ 文件关联成功`);
            return true;
        } else {
            console.log(`❌ 文件关联失败`);
            return false;
        }
    }

    /**
     * 测试5: 编辑记录与本地存储
     */
    async testRecordEdit() {
        console.log('\n🧪 测试5: 编辑记录与本地存储');

        const result = await this.callMCPTool('record_edit', {
            sessionId: this.testSessionId,
            filePath: 'examples/integration_test.js',
            diff: '+ 添加集成测试代码\n- 删除旧代码'
        });

        // 处理本地存储指令
        await this.processLocalInstruction(result);

        // 验证编辑记录是否成功
        if (result.parsedResult && result.parsedResult.status === 'success') {
            console.log(`✅ 编辑记录成功`);
            return true;
        } else {
            console.log(`❌ 编辑记录失败`);
            return false;
        }
    }

    /**
     * 运行完整的集成测试套件
     */
    async runFullTest() {
        console.log('🚀 开始Context-Keeper MCP工具指令模式集成测试');
        console.log(`📍 服务端地址: ${this.serverUrl}`);
        console.log(`🆔 测试会话ID: ${this.testSessionId}`);

        const tests = [
            { name: '会话管理', method: this.testSessionManagement },
            { name: '长期记忆存储', method: this.testMemorizeContext },
            { name: '短期记忆存储', method: this.testStoreConversation },
            { name: '代码文件关联', method: this.testAssociateFile },
            { name: '编辑记录', method: this.testRecordEdit }
        ];

        let passedTests = 0;
        let totalTests = tests.length;

        for (const test of tests) {
            try {
                const success = await test.method.call(this);
                if (success) {
                    passedTests++;
                }
                // 测试间隔
                await new Promise(resolve => setTimeout(resolve, 1000));
            } catch (error) {
                console.error(`❌ 测试"${test.name}"出现异常: ${error.message}`);
            }
        }

        console.log('\n📊 测试结果汇总:');
        console.log(`✅ 通过: ${passedTests}/${totalTests}`);
        console.log(`❌ 失败: ${totalTests - passedTests}/${totalTests}`);
        console.log(`📈 成功率: ${((passedTests / totalTests) * 100).toFixed(1)}%`);

        if (passedTests === totalTests) {
            console.log('\n🎉 所有测试通过！MCP工具指令模式集成成功！');
        } else {
            console.log('\n⚠️ 部分测试失败，需要进一步调试');
        }

        return passedTests === totalTests;
    }
}

// 如果直接运行此文件，执行集成测试
if (require.main === module) {
    const test = new IntegrationTest();
    test.runFullTest().catch(console.error);
}

module.exports = { IntegrationTest }; 