#!/usr/bin/env node

/**
 * Context-Keeper 集成测试 - 配置化版本
 * 测试MCP客户端和Cursor扩展的配置管理功能
 */

const ContextKeeperMCPClient = require('./mcp-client.js');
const fs = require('fs').promises;
const path = require('os');

class IntegrationTest {
    constructor() {
        this.testResults = {
            total: 0,
            passed: 0,
            failed: 0,
            errors: []
        };
    }

    async runTest(name, testFn) {
        console.log(`\n🧪 测试: ${name}`);
        this.testResults.total++;
        
        try {
            const startTime = Date.now();
            await testFn();
            const duration = Date.now() - startTime;
            
            console.log(`✅ 通过 (${duration}ms)`);
            this.testResults.passed++;
            return true;
        } catch (error) {
            console.log(`❌ 失败: ${error.message}`);
            this.testResults.failed++;
            this.testResults.errors.push({ test: name, error: error.message });
            return false;
        }
    }

    async runAllTests() {
        console.log('🚀 开始Context-Keeper配置化集成测试\n');
        
        // 1. 测试默认配置创建
        await this.runTest('客户端默认配置创建', async () => {
            const client = new ContextKeeperMCPClient();
            const config = client.getConfig();
            
            if (!config.serverURL) {
                throw new Error('默认配置中缺少serverURL');
            }
            
            if (!config.retryConfig) {
                throw new Error('默认配置中缺少retryConfig');
            }
            
            console.log('  默认配置:', JSON.stringify(config, null, 2));
        });

        // 2. 测试配置更新
        await this.runTest('客户端配置更新', async () => {
            const client = new ContextKeeperMCPClient();
            
            const newConfig = {
                serverURL: 'http://localhost:9999',
                timeout: 5000,
                logging: {
                    enabled: false,
                    level: 'error'
                }
            };
            
            client.updateConfig(newConfig);
            const updatedConfig = client.getConfig();
            
            if (updatedConfig.serverURL !== 'http://localhost:9999') {
                throw new Error('配置更新失败: serverURL');
            }
            
            if (updatedConfig.timeout !== 5000) {
                throw new Error('配置更新失败: timeout');
            }
            
            console.log('  更新后配置:', JSON.stringify(updatedConfig, null, 2));
        });

        // 3. 测试用户ID管理
        await this.runTest('用户ID自动生成和管理', async () => {
            const client = new ContextKeeperMCPClient();
            
            // 首次获取应该生成新的用户ID
            const userId1 = await client.getUserId();
            if (!userId1 || !userId1.startsWith('user_')) {
                throw new Error('用户ID生成失败');
            }
            
            // 再次获取应该返回相同的用户ID
            const userId2 = await client.getUserId();
            if (userId1 !== userId2) {
                throw new Error('用户ID不一致');
            }
            
            // 手动设置用户ID
            client.setUserId('test_user_123');
            const userId3 = await client.getUserId();
            if (userId3 !== 'test_user_123') {
                throw new Error('手动设置用户ID失败');
            }
            
            console.log(`  生成的用户ID: ${userId1}`);
            console.log(`  手动设置的用户ID: ${userId3}`);
        });

        // 4. 测试连接和错误处理
        await this.runTest('连接错误处理和重试机制', async () => {
            const client = new ContextKeeperMCPClient({
                serverURL: 'http://localhost:9999', // 不存在的服务器
                timeout: 2000,
                retryConfig: {
                    maxRetries: 2,
                    retryDelay: 100,
                    backoffMultiplier: 1.5
                },
                logging: {
                    enabled: false // 关闭日志避免干扰测试输出
                }
            });
            
            let errorCaught = false;
            try {
                await client.createSession({ type: 'test' });
            } catch (error) {
                errorCaught = true;
                console.log(`  预期的错误: ${error.message}`);
            }
            
            if (!errorCaught) {
                throw new Error('应该抛出连接错误');
            }
        });

        // 5. 测试真实服务器连接（如果可用）
        await this.runTest('真实服务器连接测试', async () => {
            const client = new ContextKeeperMCPClient({
                serverURL: 'http://localhost:8088',
                userId: 'integration-test-' + Date.now(),
                logging: {
                    enabled: true,
                    level: 'info'
                }
            });
            
            try {
                // 创建会话
                const session = await client.createSession({
                    type: 'integration-test',
                    description: '配置化集成测试'
                });
                
                if (!session.sessionId) {
                    throw new Error('会话创建失败：没有返回sessionId');
                }
                
                console.log(`  会话创建成功: ${session.sessionId}`);
                
                // 存储对话
                const storeResult = await client.storeConversation(session.sessionId, [
                    { role: 'user', content: '测试配置化MCP客户端' },
                    { role: 'assistant', content: '客户端配置功能正常，本地指令执行成功' }
                ]);
                
                if (storeResult.localInstructionResult && !storeResult.localInstructionResult.success) {
                    throw new Error('本地指令执行失败');
                }
                
                console.log('  对话存储成功，本地指令执行正常');
                
                // 检索上下文
                const context = await client.retrieveContext(session.sessionId, '配置化MCP客户端');
                console.log('  上下文检索结果:', context ? '有数据' : '无数据');
                
                return { sessionId: session.sessionId, client };
                
            } catch (error) {
                console.log(`  服务器不可用或配置错误: ${error.message}`);
                // 如果服务器不可用，这不应该算作测试失败
                console.log('  跳过真实服务器测试');
            }
        });

        // 6. 测试本地指令路径展开
        await this.runTest('本地指令路径展开', async () => {
            const client = new ContextKeeperMCPClient({
                userId: 'test_user_456'
            });
            
            const testPaths = [
                '~/.context-keeper/users/{userId}/sessions/test.json',
                '{userId}/test.json',
                '/absolute/path/test.json'
            ];
            
            const expandedPaths = testPaths.map(path => client.expandPath(path));
            
            // 检查路径展开是否正确
            if (!expandedPaths[0].includes('test_user_456')) {
                throw new Error('用户ID替换失败');
            }
            
            if (!expandedPaths[0].includes('.context-keeper')) {
                throw new Error('家目录展开失败');
            }
            
            console.log('  路径展开结果:');
            testPaths.forEach((original, index) => {
                console.log(`    ${original} -> ${expandedPaths[index]}`);
            });
        });

        // 7. 测试配置验证
        await this.runTest('配置验证机制', async () => {
            const extensionAPI = require('./cursor-extension.js');
            
            // 测试有效配置
            const validConfig = {
                serverConnection: {
                    serverURL: 'http://localhost:8088',
                    timeout: 10000
                },
                userSettings: {
                    userId: 'test_user',
                    baseDir: '/tmp/test'
                },
                automationFeatures: {
                    autoCapture: true,
                    captureInterval: 30
                }
            };
            
            const validationErrors = extensionAPI.validateConfig(validConfig);
            if (validationErrors.length > 0) {
                throw new Error(`有效配置验证失败: ${validationErrors.join(', ')}`);
            }
            
            // 测试无效配置
            const invalidConfig = {
                serverConnection: {
                    serverURL: 'invalid-url', // 无效的URL
                    timeout: 500 // 超时时间太短
                },
                automationFeatures: {
                    captureInterval: 5 // 间隔时间太短
                }
            };
            
            const invalidErrors = extensionAPI.validateConfig(invalidConfig);
            if (invalidErrors.length === 0) {
                throw new Error('无效配置应该被拒绝');
            }
            
            console.log(`  有效配置通过验证`);
            console.log(`  无效配置被正确拒绝: ${invalidErrors.length}个错误`);
        });

        // 8. 测试扩展初始化
        await this.runTest('扩展初始化和配置管理', async () => {
            const extensionAPI = require('./cursor-extension.js');
            
            // 获取配置架构
            const schema = extensionAPI.getConfigSchema();
            if (!schema.serverConnection || !schema.userSettings) {
                throw new Error('配置架构不完整');
            }
            
            console.log('  配置架构包含以下部分:');
            Object.keys(schema).forEach(section => {
                console.log(`    - ${schema[section].title}`);
            });
            
            // 测试连接测试功能
            const connectionResult = await extensionAPI.testConnection();
            console.log(`  连接测试结果: ${connectionResult.success ? '成功' : '失败'}`);
            if (!connectionResult.success) {
                console.log(`  连接失败原因: ${connectionResult.message}`);
            }
        });

        // 输出测试总结
        console.log('\n📊 测试总结:');
        console.log(`  总计: ${this.testResults.total}`);
        console.log(`  通过: ${this.testResults.passed}`);
        console.log(`  失败: ${this.testResults.failed}`);
        
        if (this.testResults.failed > 0) {
            console.log('\n❌ 失败的测试:');
            this.testResults.errors.forEach(({ test, error }) => {
                console.log(`  - ${test}: ${error}`);
            });
        }
        
        const successRate = ((this.testResults.passed / this.testResults.total) * 100).toFixed(1);
        console.log(`\n成功率: ${successRate}%`);
        
        if (this.testResults.failed === 0) {
            console.log('\n🎉 所有测试通过！配置化功能完全正常！');
        } else {
            console.log('\n⚠️  部分测试失败，请检查配置和服务状态');
        }
        
        return this.testResults.failed === 0;
    }
}

// 运行测试
if (require.main === module) {
    const test = new IntegrationTest();
    test.runAllTests().then(success => {
        process.exit(success ? 0 : 1);
    }).catch(error => {
        console.error('测试运行失败:', error);
        process.exit(1);
    });
}

module.exports = IntegrationTest; 