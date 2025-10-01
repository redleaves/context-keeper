#!/usr/bin/env node

/**
 * Context-Keeper 第二期完整功能验证测试
 * 验证MCP工具指令模式与第一期STDIO模式的完全兼容性
 * 
 * 测试内容：
 * 1. 会话管理
 * 2. 长期记忆存储 
 * 3. 短期记忆存储（本地指令）
 * 4. 代码文件关联（本地指令）
 * 5. 编辑记录（本地指令）
 * 6. 用户隔离验证
 * 7. 第一期数据格式兼容性
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class ContextKeeperVerificationTest {
    constructor() {
        this.baseURL = 'http://localhost:8088';
        this.testResults = [];
        this.sessionId = null;
        this.userId = null;
        this.baseDir = path.join(os.homedir(), '.context-keeper');
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

        console.log(`📤 发送请求: ${method}`);
        console.log('   参数:', JSON.stringify(params, null, 2));

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
        
        // 解析MCP Streamable HTTP响应格式
        let result = data.result;
        
        // 如果是Streamable HTTP格式，提取content中的text
        if (result && result.content && Array.isArray(result.content) && result.content.length > 0) {
            const textContent = result.content[0].text;
            try {
                result = JSON.parse(textContent);
            } catch (e) {
                result = textContent;
            }
        }
        
        // 如果result仍然是字符串，尝试解析为JSON
        if (typeof result === 'string') {
            try {
                result = JSON.parse(result);
            } catch (e) {
                // 如果不是JSON字符串，保持原样
            }
        }

        console.log(`📥 响应结果:`, JSON.stringify(result, null, 2));
        return result;
    }

    /**
     * 执行本地指令
     */
    async executeLocalInstruction(instruction) {
        if (!instruction) {
            console.log('⚠️  无本地指令需要执行');
            return { success: true };
        }

        console.log(`🔧 执行本地指令: ${instruction.type}`);
        console.log(`   目标路径: ${instruction.target}`);

        try {
            const targetPath = this.expandPath(instruction.target);
            
            // 确保目录存在
            await this.ensureDirectory(path.dirname(targetPath));

            // 根据指令类型执行相应操作
            switch (instruction.type) {
                case 'user_config':
                    await this.writeJSON(targetPath, instruction.content);
                    break;
                    
                case 'session_store':
                    await this.writeJSON(targetPath, instruction.content);
                    break;
                    
                case 'short_memory':
                    await this.handleShortMemory(instruction, targetPath);
                    break;
                    
                case 'code_context':
                    await this.handleCodeContext(instruction, targetPath);
                    break;
                    
                default:
                    console.log(`⚠️  未知指令类型: ${instruction.type}`);
                    return { success: false, error: `未知指令类型: ${instruction.type}` };
            }

            console.log(`✅ 本地指令执行成功: ${instruction.type}`);
            
            // 发送回调确认（模拟）
            console.log(`📤 发送回调确认: ${instruction.callbackId}`);
            
            return { 
                success: true, 
                timestamp: Date.now(),
                filePath: targetPath 
            };
            
        } catch (error) {
            console.error(`❌ 本地指令执行失败:`, error.message);
            return { 
                success: false, 
                error: error.message,
                timestamp: Date.now() 
            };
        }
    }

    /**
     * 处理短期记忆存储
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
        console.log(`   💾 短期记忆已保存: ${finalHistory.length}条记录`);
    }

    /**
     * 处理代码上下文存储
     */
    async handleCodeContext(instruction, targetPath) {
        const { content, options } = instruction;
        let finalContext = content;

        // 合并到现有代码上下文（第一期兼容）
        if (options.merge && await this.fileExists(targetPath)) {
            const existingContext = await this.readJSON(targetPath);
            finalContext = { ...(existingContext || {}), ...content };
        }

        await this.writeJSON(targetPath, finalContext);
        console.log(`   💾 代码上下文已保存: ${Object.keys(content).length}个文件`);
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
     * 测试1: 会话管理
     */
    async testSessionManagement() {
        const result = await this.sendMCPRequest('tools/call', {
            name: 'session_management',
            arguments: {
                action: 'get_or_create',
                userId: 'user_weixiaofeng', // 必需参数
                workspaceRoot: process.cwd() // 必需参数，当前工作目录
            }
        });

        if (!result.sessionId) {
            throw new Error('会话创建失败：未返回sessionId');
        }

        this.sessionId = result.sessionId;
        console.log(`   📋 会话ID: ${this.sessionId}`);
        
        return result;
    }

    /**
     * 测试2: 长期记忆存储
     */
    async testLongTermMemory() {
        const result = await this.sendMCPRequest('tools/call', {
            name: 'memorize_context',
            arguments: {
                sessionId: this.sessionId,
                content: '这是一个重要的架构决策：采用MCP工具指令模式实现云端+本地混合架构',
                priority: 'P1',
                metadata: {
                    type: 'architecture_decision',
                    phase: 'second_phase',
                    feature: 'mcp_local_instruction'
                }
            }
        });

        if (!result.memoryId) {
            throw new Error('长期记忆存储失败：未返回memoryId');
        }

        console.log(`   🧠 记忆ID: ${result.memoryId}`);
        return result;
    }

    /**
     * 测试3: 短期记忆存储（本地指令）
     */
    async testShortTermMemory() {
        const messages = [
            {
                role: 'user',
                content: '请解释MCP工具指令模式的核心优势',
                contentType: 'text',
                priority: 'P2'
            },
            {
                role: 'assistant', 
                content: 'MCP工具指令模式的核心优势是实现了云端计算与本地存储的完美结合，保持了第一期的所有本地存储优势',
                contentType: 'text',
                priority: 'P2'
            }
        ];

        const result = await this.sendMCPRequest('tools/call', {
            name: 'store_conversation',
            arguments: {
                sessionId: this.sessionId,
                messages: messages
            }
        });

        // 检查是否有本地指令
        if (result.localInstruction) {
            const instructionResult = await this.executeLocalInstruction(result.localInstruction);
            if (!instructionResult.success) {
                throw new Error(`本地指令执行失败: ${instructionResult.error}`);
            }
            
            // 验证文件是否创建且格式正确
            await this.verifyShortTermMemoryFile(instructionResult.filePath, messages);
            console.log(`   📁 本地文件已验证: ${instructionResult.filePath}`);
        } else {
            console.log('   ⚠️  未生成本地指令');
        }

        return result;
    }

    /**
     * 测试4: 代码文件关联（本地指令）
     */
    async testCodeAssociation() {
        const result = await this.sendMCPRequest('tools/call', {
            name: 'associate_file',
            arguments: {
                sessionId: this.sessionId,
                filePath: 'examples/complete_verification_test.js'
            }
        });

        // 检查是否有本地指令
        if (result.localInstruction) {
            const instructionResult = await this.executeLocalInstruction(result.localInstruction);
            if (!instructionResult.success) {
                throw new Error(`本地指令执行失败: ${instructionResult.error}`);
            }
            console.log(`   📁 代码上下文已保存: ${instructionResult.filePath}`);
        }

        return result;
    }

    /**
     * 测试5: 编辑记录（本地指令）
     */
    async testEditRecord() {
        const result = await this.sendMCPRequest('tools/call', {
            name: 'record_edit',
            arguments: {
                sessionId: this.sessionId,
                filePath: 'examples/complete_verification_test.js',
                diff: '+    // 新增：完整的端到端验证测试\n+    console.log("验证测试开始");'
            }
        });

        return result;
    }

    /**
     * 测试6: 用户隔离验证
     */
    async testUserIsolation() {
        // 检查用户配置文件
        const userConfigPath = path.join(this.baseDir, 'user-config.json');
        if (await this.fileExists(userConfigPath)) {
            const userConfig = await this.readJSON(userConfigPath);
            this.userId = userConfig.userId;
            console.log(`   👤 用户ID: ${this.userId}`);
            
            // 验证用户隔离的目录结构
            const userSessionsDir = path.join(this.baseDir, 'users', this.userId, 'sessions');
            const userHistoriesDir = path.join(this.baseDir, 'users', this.userId, 'histories');
            
            console.log(`   📂 用户会话目录: ${userSessionsDir}`);
            console.log(`   📂 用户历史目录: ${userHistoriesDir}`);
            
            // 检查目录是否存在
            const sessionsDirExists = await this.directoryExists(userSessionsDir);
            const historiesDirExists = await this.directoryExists(userHistoriesDir);
            
            if (!sessionsDirExists && !historiesDirExists) {
                console.log('   ⚠️  用户隔离目录尚未创建（正常，会在有数据时创建）');
            }
            
            return {
                userId: this.userId,
                userConfigExists: true,
                sessionsDirExists,
                historiesDirExists
            };
        } else {
            console.log('   ⚠️  用户配置文件不存在');
            return { userConfigExists: false };
        }
    }

    /**
     * 测试7: 第一期数据格式兼容性验证
     */
    async testFirstPhaseCompatibility() {
        // 查找短期记忆文件
        const historiesPattern = path.join(this.baseDir, '**', 'histories', '*.json');
        const historyFiles = await this.findFiles(this.baseDir, /histories.*\.json$/);
        
        if (historyFiles.length > 0) {
            const historyFile = historyFiles[0];
            console.log(`   📄 检查历史文件: ${historyFile}`);
            
            const historyData = await this.readJSON(historyFile);
            
            // 验证第一期格式：应该是字符串数组
            if (!Array.isArray(historyData)) {
                throw new Error('历史记录格式不正确：应该是数组');
            }
            
            if (historyData.length > 0 && typeof historyData[0] !== 'string') {
                throw new Error('历史记录格式不正确：数组元素应该是字符串');
            }
            
            // 验证时间戳格式
            if (historyData.length > 0) {
                const timestampPattern = /^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]/;
                if (!timestampPattern.test(historyData[0])) {
                    throw new Error('历史记录时间戳格式不正确');
                }
            }
            
            console.log(`   ✅ 第一期格式验证通过: ${historyData.length}条记录`);
            return {
                format: 'first_phase_compatible',
                recordCount: historyData.length,
                sampleRecord: historyData[0] || null
            };
        } else {
            console.log('   ⚠️  未找到历史记录文件');
            return { format: 'no_history_files' };
        }
    }

    /**
     * 验证短期记忆文件格式
     */
    async verifyShortTermMemoryFile(filePath, originalMessages) {
        const data = await this.readJSON(filePath);
        
        // 验证是否为数组格式（第一期兼容）
        if (!Array.isArray(data)) {
            throw new Error('短期记忆文件格式错误：应该是数组');
        }
        
        // 验证记录格式
        for (const record of data) {
            if (typeof record !== 'string') {
                throw new Error('短期记忆记录格式错误：应该是字符串');
            }
            
            // 验证时间戳格式
            const timestampPattern = /^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]/;
            if (!timestampPattern.test(record)) {
                throw new Error('短期记忆时间戳格式错误');
            }
        }
        
        console.log(`   ✅ 短期记忆格式验证通过`);
    }

    /**
     * 运行所有测试
     */
    async runAllTests() {
        console.log('🚀 Context-Keeper 第二期完整功能验证开始');
        console.log('=' .repeat(60));
        
        try {
            // 测试1: 会话管理
            await this.runTest('会话管理', () => this.testSessionManagement());
            
            // 测试2: 长期记忆存储
            await this.runTest('长期记忆存储', () => this.testLongTermMemory());
            
            // 测试3: 短期记忆存储（本地指令）
            await this.runTest('短期记忆存储（本地指令）', () => this.testShortTermMemory());
            
            // 测试4: 代码文件关联（本地指令）
            await this.runTest('代码文件关联（本地指令）', () => this.testCodeAssociation());
            
            // 测试5: 编辑记录（本地指令）
            await this.runTest('编辑记录', () => this.testEditRecord());
            
            // 测试6: 用户隔离验证
            await this.runTest('用户隔离验证', () => this.testUserIsolation());
            
            // 测试7: 第一期数据格式兼容性验证
            await this.runTest('第一期数据格式兼容性验证', () => this.testFirstPhaseCompatibility());
            
            this.printTestSummary();
            
        } catch (error) {
            console.error('\n💥 测试过程中发生错误:', error.message);
            this.printTestSummary();
            process.exit(1);
        }
    }

    /**
     * 打印测试摘要
     */
    printTestSummary() {
        console.log('\n📊 测试结果摘要');
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
            console.log('\n🎉 所有测试都通过了！第二期MCP工具指令模式与第一期完全兼容！');
        } else {
            console.log(`\n⚠️  有 ${failed} 个测试失败，需要进一步检查。`);
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

    async findFiles(dir, pattern) {
        const files = [];
        
        async function searchDir(currentDir) {
            try {
                const entries = await fs.readdir(currentDir, { withFileTypes: true });
                
                for (const entry of entries) {
                    const fullPath = path.join(currentDir, entry.name);
                    
                    if (entry.isDirectory()) {
                        await searchDir(fullPath);
                    } else if (entry.isFile() && pattern.test(fullPath)) {
                        files.push(fullPath);
                    }
                }
            } catch (error) {
                // 忽略权限错误等
            }
        }
        
        await searchDir(dir);
        return files;
    }
}

// 运行测试
async function main() {
    const test = new ContextKeeperVerificationTest();
    await test.runAllTests();
}

// 如果直接运行此文件
if (require.main === module) {
    main().catch(console.error);
}

module.exports = ContextKeeperVerificationTest; 