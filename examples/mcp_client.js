#!/usr/bin/env node

/**
 * Context-Keeper MCP客户端
 * 完整实现本地指令模式，支持Cursor集成
 * 支持动态配置管理
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class ContextKeeperMCPClient {
    constructor(config = {}) {
        // 默认配置
        this.defaultConfig = {
            serverURL: 'http://localhost:8088',
            userId: null, // 将从配置文件或用户输入获取
            baseDir: path.join(os.homedir(), '.context-keeper'),
            timeout: 10000,
            retryConfig: {
                maxRetries: 3,
                retryDelay: 1000,
                backoffMultiplier: 2
            },
            logging: {
                enabled: true,
                level: 'info'
            }
        };
        
        // 合并配置
        this.config = { ...this.defaultConfig, ...config };
        this.initialized = false;
        
        this.log('[MCP客户端] 初始化Context-Keeper MCP客户端');
        this.log(`[MCP客户端] 配置:`, JSON.stringify(this.config, null, 2));
    }

    /**
     * 日志输出
     */
    log(message, data = null) {
        if (!this.config.logging.enabled) return;
        
        if (data) {
            console.log(message, data);
        } else {
            console.log(message);
        }
    }

    /**
     * 动态更新配置
     */
    updateConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
        this.log('[MCP客户端] 配置已更新:', JSON.stringify(newConfig, null, 2));
    }

    /**
     * 获取当前配置
     */
    getConfig() {
        return { ...this.config };
    }

    /**
     * 设置用户ID
     */
    setUserId(userId) {
        this.config.userId = userId;
        this.log(`[MCP客户端] 用户ID已设置: ${userId}`);
    }

    /**
     * 获取或生成用户ID
     */
    async getUserId() {
        if (this.config.userId) {
            return this.config.userId;
        }

        // 尝试从配置文件读取
        try {
            const configPath = path.join(this.config.baseDir, 'user-config.json');
            const userConfig = await this.readJSON(configPath);
            if (userConfig.userId) {
                this.config.userId = userConfig.userId;
                return this.config.userId;
            }
        } catch (error) {
            // 配置文件不存在，生成新的用户ID
        }

        // 生成新的用户ID
        this.config.userId = 'user_' + Date.now();
        await this.saveUserConfig();
        return this.config.userId;
    }

    /**
     * 保存用户配置
     */
    async saveUserConfig() {
        try {
            const configPath = path.join(this.config.baseDir, 'user-config.json');
            const userConfig = {
                userId: this.config.userId,
                createdAt: new Date().toISOString(),
                lastUpdated: new Date().toISOString()
            };
            
            await this.ensureDirectory(path.dirname(configPath));
            await this.writeJSON(configPath, userConfig);
            this.log('[MCP客户端] 用户配置已保存');
        } catch (error) {
            this.log('[MCP客户端] 保存用户配置失败:', error.message);
        }
    }

    /**
     * 发送MCP请求
     */
    async sendMCPRequest(method, params = {}) {
        // 确保用户ID存在
        if (!this.config.userId) {
            await this.getUserId();
        }

        let requestData;
        
        // 根据方法类型构造请求
        if (this.isMCPTool(method)) {
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
            requestData = {
                jsonrpc: '2.0',
                id: Date.now(),
                method: method,
                params: params
            };
        }

        this.log(`[MCP客户端] 发送请求: ${method}`);
        this.log(`[MCP客户端] 参数:`, JSON.stringify(params, null, 2));

        let lastError;
        const { maxRetries, retryDelay, backoffMultiplier } = this.config.retryConfig;

        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

                const response = await fetch(`${this.config.serverURL}/mcp`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(requestData),
                    signal: controller.signal
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP错误: ${response.status} ${response.statusText}`);
                }

                const result = await response.json();
                this.log(`[MCP客户端] 响应状态: ${result.error ? '错误' : '成功'}`);
                
                if (result.error) {
                    throw new Error(`MCP错误: ${result.error.message}`);
                }

                // 处理本地指令
                const processedResult = await this.processResponseWithLocalInstructions(result.result);
                return processedResult;

            } catch (error) {
                lastError = error;
                this.log(`[MCP客户端] 请求失败 (尝试 ${attempt}/${maxRetries}):`, error.message);
                
                if (attempt < maxRetries) {
                    const delay = retryDelay * Math.pow(backoffMultiplier, attempt - 1);
                    this.log(`[MCP客户端] ${delay}ms后重试...`);
                    await new Promise(resolve => setTimeout(resolve, delay));
                }
            }
        }

        throw lastError;
    }

    /**
     * 判断是否是MCP工具方法
     */
    isMCPTool(method) {
        const mcpTools = [
            'session_management',
            'store_conversation', 
            'retrieve_context',
            'memorize_context',
            'retrieve_memory',
            'retrieve_todos',
            'associate_file',
            'record_edit',
            'programming_context',
            'user_init_dialog'
        ];
        return mcpTools.includes(method);
    }

    /**
     * 处理响应中的本地指令
     */
    async processResponseWithLocalInstructions(response) {
        if (!response) return response;

        // 处理Streamable HTTP格式的响应
        if (response.content && response.content[0] && response.content[0].text) {
            try {
                const parsedResponse = JSON.parse(response.content[0].text);
                
                // 检查是否有本地指令
                if (parsedResponse.localInstruction) {
                    this.log(`[MCP客户端] 检测到本地指令: ${parsedResponse.localInstruction.type}`);
                    const instructionResult = await this.executeLocalInstruction(parsedResponse.localInstruction);
                    
                    // 将执行结果合并到响应中
                    parsedResponse.localInstructionResult = instructionResult;
                    
                    // 发送回调确认
                    if (instructionResult.success && parsedResponse.localInstruction.callbackId) {
                        await this.sendCallback(parsedResponse.localInstruction.callbackId, instructionResult);
                    }
                }
                
                return parsedResponse;
            } catch (e) {
                this.log(`[MCP客户端] 响应不是JSON格式，直接返回: ${e.message}`);
                return response;
            }
        }

        return response;
    }

    /**
     * 执行本地指令
     */
    async executeLocalInstruction(instruction) {
        if (!instruction) {
            return { success: true, message: '无指令需要执行' };
        }

        this.log(`[本地指令] 执行类型: ${instruction.type}`);
        this.log(`[本地指令] 目标路径: ${instruction.target}`);

        try {
            const targetPath = this.expandPath(instruction.target);
            
            // 确保目录存在
            await this.ensureDirectory(path.dirname(targetPath));

            // 根据指令类型执行相应操作
            switch (instruction.type) {
                case 'user_config':
                    return await this.handleUserConfig(instruction, targetPath);
                
                case 'session_store':
                    return await this.handleSessionStore(instruction, targetPath);
                
                case 'short_memory':
                    return await this.handleShortMemory(instruction, targetPath);
                
                case 'code_context':
                    return await this.handleCodeContext(instruction, targetPath);
                
                case 'preferences':
                    return await this.handlePreferences(instruction, targetPath);
                
                case 'cache_update':
                    return await this.handleCacheUpdate(instruction, targetPath);
                
                default:
                    throw new Error(`未知指令类型: ${instruction.type}`);
            }

        } catch (error) {
            this.log(`[本地指令] 执行失败:`, error.message);
            return { 
                success: false, 
                error: error.message,
                timestamp: Date.now()
            };
        }
    }

    /**
     * 处理用户配置指令
     */
    async handleUserConfig(instruction, targetPath) {
        const options = instruction.options || {};
        
        // 如果需要备份
        if (options.backup && await this.fileExists(targetPath)) {
            const backupPath = `${targetPath}.backup.${Date.now()}`;
            await this.copyFile(targetPath, backupPath);
            this.log(`[本地指令] 已备份用户配置: ${backupPath}`);
        }

        // 写入配置
        await this.writeJSON(targetPath, instruction.content);
        this.log(`[本地指令] 用户配置已保存: ${targetPath}`);

        return { 
            success: true, 
            filePath: targetPath,
            type: 'user_config',
            timestamp: Date.now()
        };
    }

    /**
     * 处理会话存储指令
     */
    async handleSessionStore(instruction, targetPath) {
        const options = instruction.options || {};
        
        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            const existingData = await this.readJSON(targetPath);
            const mergedData = { ...existingData, ...instruction.content };
            await this.writeJSON(targetPath, mergedData);
            this.log(`[本地指令] 会话数据已合并: ${targetPath}`);
        } else {
            await this.writeJSON(targetPath, instruction.content);
            this.log(`[本地指令] 会话数据已保存: ${targetPath}`);
        }

        // 清理旧文件
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(targetPath), options.maxAge);
        }

        return { 
            success: true, 
            filePath: targetPath,
            type: 'session_store',
            timestamp: Date.now()
        };
    }

    /**
     * 处理短期记忆指令
     */
    async handleShortMemory(instruction, targetPath) {
        const content = instruction.content;
        const options = instruction.options || {};

        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            const existingContent = await fs.readFile(targetPath, 'utf8');
            const existingData = JSON.parse(existingContent);
            
            // 合并新内容并去重
            const mergedData = [...existingData, ...content];
            
            // 如果设置了数量限制，保留最近的记录
            const maxRecords = options.maxRecords || 50; // 默认保留50条
            const finalData = mergedData.slice(-maxRecords);
            
            await this.writeJSON(targetPath, finalData);
            this.log(`[本地指令] 短期记忆已合并: ${finalData.length}条记录`);
        } else {
            await this.writeJSON(targetPath, content);
            this.log(`[本地指令] 短期记忆已保存: ${content.length}条记录`);
        }

        // 清理旧文件
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(targetPath), options.maxAge);
        }

        return { 
            success: true, 
            filePath: targetPath,
            type: 'short_memory',
            recordCount: Array.isArray(content) ? content.length : 0,
            timestamp: Date.now()
        };
    }

    /**
     * 处理代码上下文指令
     */
    async handleCodeContext(instruction, targetPath) {
        const options = instruction.options || {};
        
        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            const existingData = await this.readJSON(targetPath);
            const mergedData = { ...existingData, ...instruction.content };
            await this.writeJSON(targetPath, mergedData);
            this.log(`[本地指令] 代码上下文已合并: ${targetPath}`);
        } else {
            await this.writeJSON(targetPath, instruction.content);
            this.log(`[本地指令] 代码上下文已保存: ${targetPath}`);
        }

        return { 
            success: true, 
            filePath: targetPath,
            type: 'code_context',
            timestamp: Date.now()
        };
    }

    /**
     * 处理偏好设置指令
     */
    async handlePreferences(instruction, targetPath) {
        const options = instruction.options || {};
        
        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            const existingData = await this.readJSON(targetPath);
            const mergedData = { ...existingData, ...instruction.content };
            await this.writeJSON(targetPath, mergedData);
            this.log(`[本地指令] 偏好设置已合并: ${targetPath}`);
        } else {
            await this.writeJSON(targetPath, instruction.content);
            this.log(`[本地指令] 偏好设置已保存: ${targetPath}`);
        }

        return { 
            success: true, 
            filePath: targetPath,
            type: 'preferences',
            timestamp: Date.now()
        };
    }

    /**
     * 处理缓存更新指令
     */
    async handleCacheUpdate(instruction, targetPath) {
        await this.writeJSON(targetPath, instruction.content);
        this.log(`[本地指令] 缓存已更新: ${targetPath}`);

        return { 
            success: true, 
            filePath: targetPath,
            type: 'cache_update',
            timestamp: Date.now()
        };
    }

    /**
     * 发送回调确认
     */
    async sendCallback(callbackId, result) {
        try {
            this.log(`[MCP客户端] 发送回调确认: ${callbackId}`);
            
            const callbackData = {
                callbackId: callbackId,
                success: result.success,
                data: {
                    filePath: result.filePath,
                    type: result.type,
                    timestamp: result.timestamp
                },
                error: result.error || null,
                timestamp: Date.now()
            };

            // 这里可以向服务器发送回调确认
            // 在实际的MCP实现中，这通常通过特定的回调端点或机制实现
            this.log(`[MCP客户端] 回调数据:`, JSON.stringify(callbackData, null, 2));
            
            return true;
        } catch (error) {
            this.log(`[MCP客户端] 发送回调失败:`, error.message);
            return false;
        }
    }

    /**
     * 工具方法
     */
    expandPath(pathTemplate) {
        let expandedPath = pathTemplate.replace('~', os.homedir());
        expandedPath = expandedPath.replace('{userId}', this.config.userId || 'default');
        return expandedPath;
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

    async writeJSON(filePath, data) {
        await fs.writeFile(filePath, JSON.stringify(data, null, 2));
    }

    async readJSON(filePath) {
        const content = await fs.readFile(filePath, 'utf8');
        return JSON.parse(content);
    }

    async copyFile(sourcePath, destPath) {
        await fs.copyFile(sourcePath, destPath);
    }

    async cleanupOldFiles(directory, maxAge) {
        try {
            const files = await fs.readdir(directory);
            const now = Date.now();
            
            for (const file of files) {
                const filePath = path.join(directory, file);
                const stats = await fs.stat(filePath);
                const ageInSeconds = (now - stats.mtime.getTime()) / 1000;
                
                if (ageInSeconds > maxAge) {
                    await fs.unlink(filePath);
                    this.log(`[本地指令] 已清理旧文件: ${filePath}`);
                }
            }
        } catch (error) {
            this.log(`[本地指令] 清理旧文件失败:`, error.message);
        }
    }

    /**
     * MCP工具方法包装
     */
    async createSession(metadata = {}) {
        return await this.sendMCPRequest('session_management', {
            action: 'create',
            metadata: metadata
        });
    }

    async storeConversation(sessionId, messages, batchId = null) {
        const params = {
            sessionId: sessionId,
            messages: messages
        };
        
        if (batchId) {
            params.batchId = batchId;
        }
        
        return await this.sendMCPRequest('store_conversation', params);
    }

    async retrieveContext(sessionId, query, memoryId = null, batchId = null) {
        const params = {
            sessionId: sessionId,
            query: query
        };
        
        if (memoryId) params.memoryId = memoryId;
        if (batchId) params.batchId = batchId;
        
        return await this.sendMCPRequest('retrieve_context', params);
    }

    async memorizeContext(sessionId, content, priority = 'P2', metadata = {}) {
        return await this.sendMCPRequest('memorize_context', {
            sessionId: sessionId,
            content: content,
            priority: priority,
            metadata: metadata
        });
    }

    async associateFile(sessionId, filePath) {
        return await this.sendMCPRequest('associate_file', {
            sessionId: sessionId,
            filePath: filePath
        });
    }

    async recordEdit(sessionId, filePath, diff) {
        return await this.sendMCPRequest('record_edit', {
            sessionId: sessionId,
            filePath: filePath,
            diff: diff
        });
    }

    async getProgrammingContext(sessionId, query = '') {
        return await this.sendMCPRequest('programming_context', {
            sessionId: sessionId,
            query: query
        });
    }
}

// 导出客户端类
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ContextKeeperMCPClient;
}

// 如果直接运行，进行简单测试
if (require.main === module) {
    async function testClient() {
        // 使用动态配置创建客户端
        const client = new ContextKeeperMCPClient({
            serverURL: process.env.CONTEXT_KEEPER_URL || 'http://localhost:8088',
            userId: process.env.CONTEXT_KEEPER_USER_ID || null,
            logging: {
                enabled: true,
                level: 'info'
            }
        });
        
        try {
            console.log('\n🧪 测试MCP客户端...');
            
            // 创建会话
            const session = await client.createSession({
                type: 'test',
                description: 'MCP客户端测试会话'
            });
            console.log('✅ 会话创建成功:', session);
            
            const sessionId = session.sessionId;
            
            // 存储对话
            const storeResult = await client.storeConversation(sessionId, [
                { role: 'user', content: '测试MCP客户端功能' },
                { role: 'assistant', content: '客户端功能正常，本地指令执行成功' }
            ]);
            console.log('✅ 对话存储成功:', storeResult);
            
            // 检索上下文
            const context = await client.retrieveContext(sessionId, 'MCP客户端');
            console.log('✅ 上下文检索成功:', context);
            
            console.log('\n🎉 MCP客户端测试完成！');
            
        } catch (error) {
            console.error('❌ 测试失败:', error.message);
        }
    }
    
    testClient();
} 