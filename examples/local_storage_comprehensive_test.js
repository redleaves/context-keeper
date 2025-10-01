#!/usr/bin/env node

/**
 * Context-Keeper 本地存储逻辑全面验证测试
 * 专门验证本地存储的完整逻辑，包括存储、查询、合并、清理等
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class LocalStorageComprehensiveTest {
    constructor() {
        this.baseURL = 'http://localhost:8088';
        this.testResults = [];
        this.sessionId = null;
        this.userId = null;
        this.baseDir = path.join(os.homedir(), '.context-keeper');
        this.testSessionIds = []; // 用于清理测试数据
    }

    /**
     * 发送MCP请求
     */
    async sendMCPRequest(method, params = {}) {
        const requestData = {
            jsonrpc: '2.0',
            id: Date.now(),
            method: method,
            params: params
        };

        const response = await fetch(`${this.baseURL}/mcp`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestData)
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const data = await response.json();
        let result = data.result;
        
        // 解析MCP Streamable HTTP响应格式
        if (result && result.content && Array.isArray(result.content) && result.content.length > 0) {
            const textContent = result.content[0].text;
            try {
                result = JSON.parse(textContent);
            } catch (e) {
                result = textContent;
            }
        }
        
        if (typeof result === 'string') {
            try {
                result = JSON.parse(result);
            } catch (e) {
                // 保持原样
            }
        }

        return result;
    }

    /**
     * 执行本地指令并返回结果
     */
    async executeLocalInstruction(instruction) {
        if (!instruction) return { success: true };

        const targetPath = this.expandPath(instruction.target);
        await this.ensureDirectory(path.dirname(targetPath));

        switch (instruction.type) {
            case 'short_memory':
                await this.handleShortMemory(instruction, targetPath);
                break;
            default:
                throw new Error(`未支持的指令类型: ${instruction.type}`);
        }

        return { 
            success: true, 
            filePath: targetPath,
            content: instruction.content
        };
    }

    /**
     * 处理短期记忆存储（完整的第一期兼容逻辑）
     */
    async handleShortMemory(instruction, targetPath) {
        const { content, options } = instruction;
        let finalHistory = content;

        // 合并到现有历史记录（第一期兼容）
        if (options.merge && await this.fileExists(targetPath)) {
            const existingHistory = await this.readJSON(targetPath);
            finalHistory = [...(existingHistory || []), ...content];
            
            // 保持最大长度限制（第一期兼容：最多20条）
            const maxHistory = 20;
            if (finalHistory.length > maxHistory) {
                finalHistory = finalHistory.slice(-maxHistory);
            }
        }

        await this.writeJSON(targetPath, finalHistory);
        console.log(`   💾 短期记忆已保存: ${finalHistory.length}条记录到 ${targetPath}`);
    }

    /**
     * 运行单个测试
     */
    async runTest(testName, testFunc) {
        console.log(`\n🧪 测试: ${testName}`);
        console.log('='.repeat(50));
        
        try {
            const startTime = Date.now();
            const result = await testFunc();
            const duration = Date.now() - startTime;
            
            this.testResults.push({
                name: testName,
                status: 'PASS',
                duration: duration,
                result: result
            });
            
            console.log(`✅ 测试通过: ${testName} (${duration}ms)`);
            return result;
            
        } catch (error) {
            this.testResults.push({
                name: testName,
                status: 'FAIL',
                error: error.message,
                stack: error.stack
            });
            
            console.error(`❌ 测试失败: ${testName}`);
            console.error(`   错误: ${error.message}`);
            throw error;
        }
    }

    /**
     * 测试1: 创建测试会话
     */
    async testCreateSession() {
        const result = await this.sendMCPRequest('tools/call', {
            name: 'session_management',
            arguments: { action: 'create' }
        });

        this.sessionId = result.sessionId;
        this.testSessionIds.push(this.sessionId);
        console.log(`   📋 测试会话ID: ${this.sessionId}`);
        
        return result;
    }

    /**
     * 测试2: 存储多条短期记忆
     */
    async testStoreMultipleMemories() {
        const memories = [
            {
                messages: [
                    { role: 'user', content: '第一条测试消息', contentType: 'text', priority: 'P2' },
                    { role: 'assistant', content: '第一条回复消息', contentType: 'text', priority: 'P2' }
                ]
            },
            {
                messages: [
                    { role: 'user', content: '第二条测试消息', contentType: 'text', priority: 'P2' },
                    { role: 'assistant', content: '第二条回复消息', contentType: 'text', priority: 'P2' }
                ]
            },
            {
                messages: [
                    { role: 'user', content: '第三条测试消息', contentType: 'text', priority: 'P2' },
                    { role: 'assistant', content: '第三条回复消息', contentType: 'text', priority: 'P2' }
                ]
            }
        ];

        const results = [];
        for (let i = 0; i < memories.length; i++) {
            console.log(`   📝 存储第${i + 1}批消息...`);
            
            const result = await this.sendMCPRequest('tools/call', {
                name: 'store_conversation',
                arguments: {
                    sessionId: this.sessionId,
                    messages: memories[i].messages
                }
            });

            if (result.localInstruction) {
                await this.executeLocalInstruction(result.localInstruction);
            }

            results.push(result);
            
            // 等待一秒确保时间戳不同
            await new Promise(resolve => setTimeout(resolve, 1000));
        }

        console.log(`   ✅ 已存储${memories.length}批消息`);
        return results;
    }

    /**
     * 测试3: 验证合并逻辑
     */
    async testMergeLogic() {
        // 获取存储的文件路径
        const historyPath = await this.getHistoryFilePath();
        
        if (!historyPath || !await this.fileExists(historyPath)) {
            throw new Error('历史记录文件不存在');
        }

        const historyData = await this.readJSON(historyPath);
        
        // 验证合并逻辑
        if (!Array.isArray(historyData)) {
            throw new Error('历史记录格式错误：应该是数组');
        }

        // 应该有6条记录（3批 × 2条消息）
        if (historyData.length !== 6) {
            throw new Error(`历史记录数量错误：期望6条，实际${historyData.length}条`);
        }

        // 验证时间戳顺序（应该是按时间递增的）
        const timestamps = historyData.map(record => {
            const match = record.match(/^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]/);
            return match ? new Date(match[1]) : null;
        }).filter(Boolean);

        for (let i = 1; i < timestamps.length; i++) {
            if (timestamps[i] < timestamps[i-1]) {
                throw new Error('时间戳顺序错误：记录应该按时间递增排序');
            }
        }

        console.log(`   ✅ 合并逻辑验证通过: ${historyData.length}条记录，时间戳顺序正确`);
        console.log(`   📄 示例记录: ${historyData[0]}`);
        
        return {
            recordCount: historyData.length,
            timestampsValid: true,
            sampleRecord: historyData[0]
        };
    }

    /**
     * 测试4: 查询场景验证
     */
    async testQueryScenarios() {
        // 测试通过retrieve_context工具查询短期记忆
        const result = await this.sendMCPRequest('tools/call', {
            name: 'retrieve_context',
            arguments: {
                sessionId: this.sessionId,
                query: '测试消息',
                limit: 10
            }
        });

        console.log(`   🔍 查询结果:`, JSON.stringify(result, null, 2));

        // 验证查询结果
        if (!result.short_term_memory) {
            throw new Error('查询结果中缺少短期记忆数据');
        }

        // 检查是否包含我们存储的测试消息
        const shortTermMemory = result.short_term_memory;
        const containsTestMessage = shortTermMemory.includes('测试消息');
        
        if (!containsTestMessage) {
            console.log(`   ⚠️  短期记忆内容: ${shortTermMemory.substring(0, 200)}...`);
            throw new Error('查询结果中未找到测试消息');
        }

        console.log(`   ✅ 查询验证通过: 成功检索到测试消息`);
        
        return {
            querySuccess: true,
            shortTermMemoryLength: shortTermMemory.length,
            containsTestMessage: true
        };
    }

    /**
     * 测试5: 20条记录限制验证
     */
    async testRecordLimit() {
        console.log(`   📝 测试20条记录限制...`);
        
        // 存储25条记录，验证是否只保留最新的20条
        for (let i = 0; i < 25; i++) {
            const result = await this.sendMCPRequest('tools/call', {
                name: 'store_conversation',
                arguments: {
                    sessionId: this.sessionId,
                    messages: [
                        { 
                            role: 'user', 
                            content: `限制测试消息${i + 1}`, 
                            contentType: 'text', 
                            priority: 'P2' 
                        }
                    ]
                }
            });

            if (result.localInstruction) {
                await this.executeLocalInstruction(result.localInstruction);
            }

            // 每5条检查一次
            if ((i + 1) % 5 === 0) {
                console.log(`     已存储${i + 1}条记录...`);
            }
        }

        // 检查最终记录数量
        const historyPath = await this.getHistoryFilePath();
        const historyData = await this.readJSON(historyPath);
        
        if (historyData.length > 20) {
            throw new Error(`记录数量超过限制：期望最多20条，实际${historyData.length}条`);
        }

        // 验证是否保留了最新的记录
        const lastRecord = historyData[historyData.length - 1];
        if (!lastRecord.includes('限制测试消息25')) {
            throw new Error('未保留最新的记录');
        }

        console.log(`   ✅ 记录限制验证通过: 最终保留${historyData.length}条记录`);
        console.log(`   📄 最后一条记录: ${lastRecord}`);
        
        return {
            finalRecordCount: historyData.length,
            limitRespected: historyData.length <= 20,
            lastRecordCorrect: true
        };
    }

    /**
     * 测试6: 用户隔离验证
     */
    async testUserIsolation() {
        // 创建第二个会话
        const session2Result = await this.sendMCPRequest('tools/call', {
            name: 'session_management',
            arguments: { action: 'create' }
        });

        const session2Id = session2Result.sessionId;
        this.testSessionIds.push(session2Id);

        // 在第二个会话中存储不同的消息
        const result = await this.sendMCPRequest('tools/call', {
            name: 'store_conversation',
            arguments: {
                sessionId: session2Id,
                messages: [
                    { role: 'user', content: '第二个会话的消息', contentType: 'text', priority: 'P2' }
                ]
            }
        });

        if (result.localInstruction) {
            await this.executeLocalInstruction(result.localInstruction);
        }

        // 验证两个会话的文件是分开的
        const history1Path = await this.getHistoryFilePath(this.sessionId);
        const history2Path = await this.getHistoryFilePath(session2Id);

        if (history1Path === history2Path) {
            throw new Error('不同会话使用了相同的历史文件路径');
        }

        // 验证文件内容不同
        const history1Data = await this.readJSON(history1Path);
        const history2Data = await this.readJSON(history2Path);

        const history1Content = JSON.stringify(history1Data);
        const history2Content = JSON.stringify(history2Data);

        if (history1Content === history2Content) {
            throw new Error('不同会话的历史内容相同，用户隔离失效');
        }

        console.log(`   ✅ 用户隔离验证通过`);
        console.log(`   📁 会话1文件: ${history1Path}`);
        console.log(`   📁 会话2文件: ${history2Path}`);
        console.log(`   📊 会话1记录数: ${history1Data.length}`);
        console.log(`   📊 会话2记录数: ${history2Data.length}`);

        return {
            session1Path: history1Path,
            session2Path: history2Path,
            session1Records: history1Data.length,
            session2Records: history2Data.length,
            isolationWorking: true
        };
    }

    /**
     * 测试7: 数据一致性验证
     */
    async testDataConsistency() {
        // 验证本地存储的数据与通过API查询的数据是否一致
        const apiResult = await this.sendMCPRequest('tools/call', {
            name: 'retrieve_context',
            arguments: {
                sessionId: this.sessionId,
                query: '限制测试消息',
                limit: 5
            }
        });

        const localHistoryPath = await this.getHistoryFilePath();
        const localData = await this.readJSON(localHistoryPath);

        // 从本地数据中提取最近的限制测试消息
        const localTestMessages = localData.filter(record => 
            record.includes('限制测试消息')
        ).slice(-5);

        console.log(`   📊 本地历史记录数: ${localData.length}`);
        console.log(`   📊 本地测试消息数: ${localTestMessages.length}`);
        console.log(`   📊 API查询结果长度: ${apiResult.short_term_memory.length}`);

        // 验证API返回的短期记忆包含本地的测试消息
        let consistencyCount = 0;
        for (const localMsg of localTestMessages) {
            if (apiResult.short_term_memory.includes(localMsg.split('] ')[1])) {
                consistencyCount++;
            }
        }

        const consistencyRate = consistencyCount / localTestMessages.length;
        console.log(`   ✅ 数据一致性: ${(consistencyRate * 100).toFixed(1)}% (${consistencyCount}/${localTestMessages.length})`);

        if (consistencyRate < 0.8) {
            throw new Error(`数据一致性过低: ${(consistencyRate * 100).toFixed(1)}%`);
        }

        return {
            localRecords: localData.length,
            localTestMessages: localTestMessages.length,
            consistencyRate: consistencyRate,
            consistencyPassed: consistencyRate >= 0.8
        };
    }

    /**
     * 获取历史文件路径
     */
    async getHistoryFilePath(sessionId = null) {
        sessionId = sessionId || this.sessionId;
        
        // 检查用户配置获取userId
        const userConfigPath = path.join(this.baseDir, 'user-config.json');
        if (await this.fileExists(userConfigPath)) {
            const userConfig = await this.readJSON(userConfigPath);
            this.userId = userConfig.userId;
        }

        if (!this.userId) {
            // 尝试从已存在的文件中推断userId
            const usersDir = path.join(this.baseDir, 'users');
            if (await this.directoryExists(usersDir)) {
                const userDirs = await fs.readdir(usersDir);
                if (userDirs.length > 0) {
                    this.userId = userDirs[0];
                }
            }
        }

        if (!this.userId) {
            throw new Error('无法确定用户ID');
        }

        return path.join(this.baseDir, 'users', this.userId, 'histories', `${sessionId}.json`);
    }

    /**
     * 清理测试数据
     */
    async cleanupTestData() {
        console.log('\n🧹 清理测试数据...');
        
        for (const sessionId of this.testSessionIds) {
            try {
                const historyPath = await this.getHistoryFilePath(sessionId);
                if (await this.fileExists(historyPath)) {
                    await fs.unlink(historyPath);
                    console.log(`   🗑️  已删除: ${historyPath}`);
                }
            } catch (error) {
                console.log(`   ⚠️  清理失败: ${sessionId} - ${error.message}`);
            }
        }
    }

    /**
     * 运行所有测试
     */
    async runAllTests() {
        console.log('🚀 Context-Keeper 本地存储逻辑全面验证开始');
        console.log('=' .repeat(60));
        
        try {
            await this.runTest('创建测试会话', () => this.testCreateSession());
            await this.runTest('存储多条短期记忆', () => this.testStoreMultipleMemories());
            await this.runTest('验证合并逻辑', () => this.testMergeLogic());
            await this.runTest('查询场景验证', () => this.testQueryScenarios());
            await this.runTest('20条记录限制验证', () => this.testRecordLimit());
            await this.runTest('用户隔离验证', () => this.testUserIsolation());
            await this.runTest('数据一致性验证', () => this.testDataConsistency());
            
            this.printTestSummary();
            
        } catch (error) {
            console.error('\n💥 测试过程中发生错误:', error.message);
            this.printTestSummary();
        } finally {
            // 清理测试数据
            await this.cleanupTestData();
        }
    }

    /**
     * 打印测试摘要
     */
    printTestSummary() {
        console.log('\n📊 本地存储验证结果摘要');
        console.log('=' .repeat(60));
        
        const passed = this.testResults.filter(t => t.status === 'PASS').length;
        const failed = this.testResults.filter(t => t.status === 'FAIL').length;
        const total = this.testResults.length;
        
        console.log(`总测试数: ${total}`);
        console.log(`通过: ${passed}`);
        console.log(`失败: ${failed}`);
        console.log(`成功率: ${((passed / total) * 100).toFixed(1)}%`);
        
        console.log('\n详细结果:');
        this.testResults.forEach((test, index) => {
            const status = test.status === 'PASS' ? '✅' : '❌';
            const duration = test.duration ? ` (${test.duration}ms)` : '';
            console.log(`${index + 1}. ${status} ${test.name}${duration}`);
            if (test.status === 'FAIL') {
                console.log(`   错误: ${test.error}`);
            }
        });
        
        if (failed === 0) {
            console.log('\n🎉 本地存储逻辑验证全部通过！');
            console.log('✅ 存储逻辑正确');
            console.log('✅ 查询功能正常');
            console.log('✅ 合并机制有效');
            console.log('✅ 数据格式兼容');
            console.log('✅ 用户隔离工作');
            console.log('✅ 记录限制生效');
            console.log('✅ 数据一致性保证');
        } else {
            console.log(`\n⚠️  有 ${failed} 个测试失败，本地存储逻辑需要进一步检查。`);
        }
    }

    // 工具方法
    expandPath(targetPath) {
        if (targetPath.startsWith('~/')) {
            return path.join(os.homedir(), targetPath.slice(2));
        }
        if (targetPath.startsWith('~/.context-keeper/')) {
            return path.join(this.baseDir, targetPath.slice(18));
        }
        return targetPath;
    }

    async ensureDirectory(dirPath) {
        try {
            await fs.mkdir(dirPath, { recursive: true });
        } catch (error) {
            if (error.code !== 'EEXIST') {
                throw error;
            }
        }
    }

    async fileExists(filePath) {
        try {
            await fs.access(filePath);
            return true;
        } catch {
            return false;
        }
    }

    async directoryExists(dirPath) {
        try {
            const stats = await fs.stat(dirPath);
            return stats.isDirectory();
        } catch {
            return false;
        }
    }

    async readJSON(filePath) {
        const data = await fs.readFile(filePath, 'utf-8');
        return JSON.parse(data);
    }

    async writeJSON(filePath, data) {
        const jsonString = JSON.stringify(data, null, 2);
        await fs.writeFile(filePath, jsonString, 'utf-8');
    }
}

// 运行测试
async function main() {
    const test = new LocalStorageComprehensiveTest();
    await test.runAllTests();
}

if (require.main === module) {
    main().catch(console.error);
}

module.exports = LocalStorageComprehensiveTest; 