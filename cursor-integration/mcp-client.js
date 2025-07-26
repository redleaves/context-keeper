#!/usr/bin/env node

/**
 * Context-Keeper MCP客户端 - Cursor集成版本
 * 专为Cursor IDE优化的智能上下文管理客户端
 * 
 * @version 2.0.0
 * @author Context-Keeper Team
 * @license MIT
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class ContextKeeperMCPClient {
    constructor(config = {}) {
        // 默认配置 - 包含客户端基本设置，但存储路径由服务端指令决定
        this.defaultConfig = {
            serverURL: 'http://localhost:8088',
            userId: null, // 通过服务端获取
            // 客户端临时工作目录（仅用于临时文件，实际存储路径由服务端决定）
            tempDir: path.join(os.tmpdir(), 'context-keeper-temp'),
            timeout: 15000,
            retryConfig: {
                maxRetries: 3,
                retryDelay: 1000,
                backoffMultiplier: 2
            },
            logging: {
                enabled: true,
                level: 'info'
            },
            features: {
                autoBackup: true,
                compressionEnabled: false,
                encryptionEnabled: false
            }
        };
        
        // 合并配置
        this.config = this.mergeConfig(this.defaultConfig, config);
        this.initialized = false;
        
        // 钩子系统初始化
        this.hooks = {
            // 响应钩子 - 在MCP响应处理完成后触发
            'response.processed': [],
            'response.localInstructionExecuted': [],
            'response.error': [],
            
            // 请求钩子 - 在发送MCP请求前后触发
            'request.beforeSend': [],
            'request.afterSend': [],
            
            // 本地指令钩子 - 在执行本地指令时触发
            'localInstruction.beforeExecute': [],
            'localInstruction.afterExecute': [],
            'localInstruction.error': [],
            
            // 特定类型指令钩子
            'shortMemory.stored': [],
            'sessionData.saved': [],
            'codeContext.updated': [],
            'userConfig.changed': [],
            
            // 扩展插件钩子 - 允许扩展插件注册自定义钩子
            'extension.custom': []
        };
        
        this.log('[Context-Keeper] MCP客户端初始化 - 钩子系统已启用');
        
        // 自动注册内置钩子
        this.registerBuiltinHooks();
    }

    /**
     * 深度合并配置对象
     */
    mergeConfig(defaultConfig, userConfig) {
        const merged = { ...defaultConfig };
        
        for (const key in userConfig) {
            if (userConfig[key] && typeof userConfig[key] === 'object' && !Array.isArray(userConfig[key])) {
                merged[key] = { ...(merged[key] || {}), ...userConfig[key] };
            } else {
                merged[key] = userConfig[key];
            }
        }
        
        return merged;
    }

    /**
     * 日志输出（支持不同级别）
     */
    log(message, data = null, level = 'info') {
        if (!this.config.logging.enabled) return;
        
        const logLevels = { debug: 0, info: 1, warn: 2, error: 3 };
        const currentLevel = logLevels[this.config.logging.level] || 1;
        const messageLevel = logLevels[level] || 1;
        
        if (messageLevel >= currentLevel) {
            const timestamp = new Date().toISOString();
            const prefix = `[${timestamp}] [${level.toUpperCase()}]`;
            
            if (data) {
                console.log(`${prefix} ${message}`, data);
            } else {
                console.log(`${prefix} ${message}`);
            }
        }
    }

    /**
     * 动态更新配置
     */
    updateConfig(newConfig) {
        this.config = this.mergeConfig(this.config, newConfig);
        this.log('配置已更新', newConfig, 'info');
    }

    /**
     * 获取当前配置
     */
    getConfig() {
        return JSON.parse(JSON.stringify(this.config));
    }

    /**
     * 设置用户ID
     */
    /**
     * 客户端不再管理用户ID和用户初始化
     * 这些职责完全由服务端负责
     * 客户端只负责执行服务端返回的本地指令
     */

    /**
     * 保存用户配置 - 不再直接保存，通过服务端本地指令处理
     */
    async saveUserConfig() {
        // 用户配置的保存现在通过服务端的本地指令来完成
        // 客户端不再直接操作本地文件系统
        this.log('用户配置将通过服务端本地指令保存');
        return true;
    }

    /**
     * 发送MCP请求（增强版 - 支持钩子）
     */
    async sendMCPRequest(method, params = {}) {
        // 客户端不再主动获取用户ID，由服务端处理用户管理
        const requestId = Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        let requestData;
        
        // 根据方法类型构造请求
        if (this.isMCPTool(method)) {
            requestData = {
                jsonrpc: '2.0',
                id: requestId,
                method: 'tools/call',
                params: {
                    name: method,
                    arguments: params
                }
            };
        } else {
            requestData = {
                jsonrpc: '2.0',
                id: requestId,
                method: method,
                params: params
            };
        }

        // 触发请求前钩子
        await this.triggerHook('request.beforeSend', {
            method,
            params,
            requestId,
            requestData
        });

        this.log(`发送MCP请求: ${method}`, { requestId, params }, 'debug');

        let lastError;
        let processedResult = null;
        const { maxRetries, retryDelay, backoffMultiplier } = this.config.retryConfig;

        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

                const response = await fetch(`${this.config.serverURL}/mcp`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'User-Agent': 'Context-Keeper-Cursor/2.0.0',
                        'X-Client-Version': '2.0.0',
                        'X-Request-ID': requestId
                    },
                    body: JSON.stringify(requestData),
                    signal: controller.signal
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const result = await response.json();
                
                if (result.error) {
                    throw new Error(`MCP错误 [${result.error.code}]: ${result.error.message}`);
                }

                this.log(`MCP请求成功: ${method}`, { requestId, duration: Date.now() - parseInt(requestId.split('-')[0]) }, 'debug');

                // 处理本地指令
                processedResult = await this.processResponseWithLocalInstructions(result.result);
                
                // 触发请求完成钩子
                await this.triggerHook('request.afterSend', {
                    method,
                    params,
                    requestId,
                    result: processedResult,
                    success: true,
                    duration: Date.now() - parseInt(requestId.split('-')[0])
                });
                
                return processedResult;

            } catch (error) {
                lastError = error;
                this.log(`MCP请求失败 (${attempt}/${maxRetries}): ${method}`, error.message, 'warn');
                
                if (attempt < maxRetries) {
                    const delay = retryDelay * Math.pow(backoffMultiplier, attempt - 1);
                    this.log(`${delay}ms后重试...`, null, 'debug');
                    await new Promise(resolve => setTimeout(resolve, delay));
                }
            }
        }

        // 触发请求失败钩子
        await this.triggerHook('request.afterSend', {
            method,
            params,
            requestId,
            success: false,
            error: lastError.message,
            attempts: maxRetries
        });

        // 触发响应错误钩子
        await this.triggerHook('response.error', {
            method,
            params,
            requestId,
            error: lastError.message,
            attempts: maxRetries
        });

        this.log(`MCP请求最终失败: ${method}`, lastError.message, 'error');
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
                    this.log(`检测到本地指令: ${parsedResponse.localInstruction.type}`, null, 'debug');
                    
                    // 触发本地指令执行前钩子
                    await this.triggerHook('localInstruction.beforeExecute', {
                        instruction: parsedResponse.localInstruction,
                        response: parsedResponse
                    });
                    
                    const instructionResult = await this.executeLocalInstruction(parsedResponse.localInstruction);
                    
                    // 将执行结果合并到响应中
                    parsedResponse.localInstructionResult = instructionResult;
                    
                    // 触发本地指令执行后钩子
                    await this.triggerHook('localInstruction.afterExecute', {
                        instruction: parsedResponse.localInstruction,
                        result: instructionResult,
                        response: parsedResponse
                    });
                    
                    // 触发本地指令成功钩子
                    if (instructionResult.success) {
                        await this.triggerHook('response.localInstructionExecuted', {
                            instruction: parsedResponse.localInstruction,
                            result: instructionResult,
                            response: parsedResponse
                        });
                    } else {
                        // 触发本地指令错误钩子
                        await this.triggerHook('localInstruction.error', {
                            instruction: parsedResponse.localInstruction,
                            error: instructionResult.error,
                            response: parsedResponse
                        });
                    }
                    
                    // 发送回调确认
                    if (instructionResult.success && parsedResponse.localInstruction.callbackId) {
                        await this.sendCallback(parsedResponse.localInstruction.callbackId, instructionResult);
                    }
                }
                
                // 触发响应处理完成钩子
                await this.triggerHook('response.processed', {
                    response: parsedResponse,
                    hasLocalInstruction: !!parsedResponse.localInstruction
                });
                
                return parsedResponse;
            } catch (e) {
                this.log('响应解析失败，使用原始响应', e.message, 'debug');
                
                // 触发响应错误钩子
                await this.triggerHook('response.error', {
                    error: e.message,
                    originalResponse: response
                });
                
                return response;
            }
        }

        return response;
    }

    /**
     * 执行本地指令（增强版）
     */
    async executeLocalInstruction(instruction) {
        if (!instruction) {
            return { success: true, message: '无指令需要执行' };
        }

        this.log(`执行本地指令: ${instruction.type}`, { target: instruction.target }, 'debug');

        try {
            const targetPath = this.expandPath(instruction.target);
            
            // 确保目录存在
            await this.ensureDirectory(path.dirname(targetPath));

            // 备份现有文件（如果启用）
            if (this.config.features.autoBackup && await this.fileExists(targetPath)) {
                await this.createBackup(targetPath);
            }

            // 根据指令类型执行相应操作
            let result;
            switch (instruction.type) {
                case 'user_config':
                    result = await this.handleUserConfig(instruction, targetPath);
                    break;
                
                case 'session_store':
                    result = await this.handleSessionStore(instruction, targetPath);
                    break;
                
                case 'short_memory':
                    result = await this.handleShortMemory(instruction, targetPath);
                    break;
                
                case 'code_context':
                    result = await this.handleCodeContext(instruction, targetPath);
                    break;
                
                case 'preferences':
                    result = await this.handlePreferences(instruction, targetPath);
                    break;
                
                case 'cache_update':
                    result = await this.handleCacheUpdate(instruction, targetPath);
                    break;
                
                default:
                    throw new Error(`未知指令类型: ${instruction.type}`);
            }

            this.log(`本地指令执行成功: ${instruction.type}`, result, 'debug');
            return result;

        } catch (error) {
            this.log(`本地指令执行失败: ${instruction.type}`, error.message, 'error');
            return { 
                success: false, 
                error: error.message,
                type: instruction.type,
                timestamp: Date.now()
            };
        }
    }

    /**
     * 创建备份文件
     */
    async createBackup(filePath) {
        try {
            const backupDir = path.join(path.dirname(filePath), '.backups');
            await this.ensureDirectory(backupDir);
            
            const fileName = path.basename(filePath);
            const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
            const backupPath = path.join(backupDir, `${fileName}.${timestamp}.backup`);
            
            await this.copyFile(filePath, backupPath);
            this.log(`备份文件已创建: ${backupPath}`, null, 'debug');
            
            // 清理旧备份（保留最近10个）
            await this.cleanupOldBackups(backupDir, fileName, 10);
            
        } catch (error) {
            this.log('创建备份失败', error.message, 'warn');
        }
    }

    /**
     * 清理旧备份文件
     */
    async cleanupOldBackups(backupDir, fileName, keepCount) {
        try {
            const files = await fs.readdir(backupDir);
            const backupFiles = files
                .filter(file => file.startsWith(fileName) && file.endsWith('.backup'))
                .map(file => ({
                    name: file,
                    path: path.join(backupDir, file),
                    stat: null
                }));

            // 获取文件统计信息
            for (const file of backupFiles) {
                try {
                    file.stat = await fs.stat(file.path);
                } catch (error) {
                    // 忽略无法访问的文件
                }
            }

            // 按修改时间排序，保留最新的
            backupFiles
                .filter(file => file.stat)
                .sort((a, b) => b.stat.mtime.getTime() - a.stat.mtime.getTime())
                .slice(keepCount)
                .forEach(async file => {
                    try {
                        await fs.unlink(file.path);
                        this.log(`已清理旧备份: ${file.name}`, null, 'debug');
                    } catch (error) {
                        // 忽略删除失败
                    }
                });

        } catch (error) {
            this.log('清理旧备份失败', error.message, 'warn');
        }
    }

    /**
     * 处理用户配置指令
     */
    async handleUserConfig(instruction, targetPath) {
        const options = instruction.options || {};
        
        // 写入配置
        await this.writeJSON(targetPath, instruction.content);
        
        const result = { 
            success: true, 
            filePath: targetPath,
            type: 'user_config',
            timestamp: Date.now(),
            size: JSON.stringify(instruction.content).length
        };
        
        // 触发用户配置变更钩子
        await this.triggerHook('userConfig.changed', result);
        
        return result;
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
        } else {
            await this.writeJSON(targetPath, instruction.content);
        }

        // 清理旧文件
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(targetPath), options.maxAge);
        }

        const result = { 
            success: true, 
            filePath: targetPath,
            type: 'session_store',
            timestamp: Date.now(),
            sessionId: instruction.content?.sessionId || 'unknown'
        };
        
        // 触发会话数据保存钩子
        await this.triggerHook('sessionData.saved', result);
        
        return result;
    }

    /**
     * 处理短期记忆指令（优化版）
     */
    async handleShortMemory(instruction, targetPath) {
        const content = instruction.content;
        const options = instruction.options || {};
        let finalData;

        // 处理合并选项
        if (options.merge && await this.fileExists(targetPath)) {
            const existingContent = await fs.readFile(targetPath, 'utf8');
            const existingData = JSON.parse(existingContent);
            
            // 合并新内容
            const mergedData = [...existingData, ...content];
            
            // 数量限制和去重
            const maxRecords = options.maxRecords || 100;
            const uniqueData = this.deduplicateMemories(mergedData);
            finalData = uniqueData.slice(-maxRecords);
            
            await this.writeJSON(targetPath, finalData);
            this.log(`短期记忆已合并: ${finalData.length}条记录`, null, 'debug');
        } else {
            finalData = content;
            await this.writeJSON(targetPath, content);
            this.log(`短期记忆已保存: ${content.length}条记录`, null, 'debug');
        }

        // 清理旧文件
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(targetPath), options.maxAge);
        }

        const result = { 
            success: true, 
            filePath: targetPath,
            type: 'short_memory',
            recordCount: Array.isArray(content) ? content.length : 0,
            totalRecords: Array.isArray(finalData) ? finalData.length : 0,
            size: JSON.stringify(finalData).length,
            timestamp: Date.now()
        };
        
        // 触发短期记忆存储钩子
        await this.triggerHook('shortMemory.stored', result);
        
        return result;
    }

    /**
     * 记忆去重
     */
    deduplicateMemories(memories) {
        const seen = new Set();
        return memories.filter(memory => {
            const key = typeof memory === 'string' ? memory : JSON.stringify(memory);
            if (seen.has(key)) {
                return false;
            }
            seen.add(key);
            return true;
        });
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
        } else {
            await this.writeJSON(targetPath, instruction.content);
        }

        const result = { 
            success: true, 
            filePath: targetPath,
            type: 'code_context',
            timestamp: Date.now()
        };
        
        // 触发代码上下文更新钩子
        await this.triggerHook('codeContext.updated', result);
        
        return result;
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
        } else {
            await this.writeJSON(targetPath, instruction.content);
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
            this.log(`发送回调确认: ${callbackId}`, null, 'debug');
            
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
            this.log('回调确认已生成', callbackData, 'debug');
            
            return true;
        } catch (error) {
            this.log('发送回调失败', error.message, 'error');
            return false;
        }
    }

    /**
     * 工具方法
     */
    expandPath(pathTemplate) {
        // 直接使用服务端返回的路径，不进行客户端的路径处理
        // 服务端负责路径的标准化和用户ID替换
        let expandedPath = pathTemplate;
        
        // 只处理 ~ 符号的扩展，其他路径处理由服务端完成
        if (expandedPath.startsWith('~/')) {
            expandedPath = path.join(os.homedir(), expandedPath.slice(2));
        }
        
        return path.resolve(expandedPath);
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
        const jsonString = JSON.stringify(data, null, 2);
        await fs.writeFile(filePath, jsonString, 'utf8');
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
                try {
                    const stats = await fs.stat(filePath);
                    const ageInSeconds = (now - stats.mtime.getTime()) / 1000;
                    
                    if (ageInSeconds > maxAge) {
                        await fs.unlink(filePath);
                        this.log(`已清理旧文件: ${filePath}`, null, 'debug');
                    }
                } catch (error) {
                    // 忽略无法访问的文件
                }
            }
        } catch (error) {
            this.log('清理旧文件失败', error.message, 'warn');
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

    /**
     * 健康检查
     */
    async healthCheck() {
        try {
            const response = await fetch(`${this.config.serverURL}/health`, {
                method: 'GET',
                timeout: 5000
            });
            
            return {
                success: response.ok,
                status: response.status,
                message: response.ok ? 'Server is healthy' : 'Server is unhealthy'
            };
        } catch (error) {
            return {
                success: false,
                message: `Health check failed: ${error.message}`
            };
        }
    }

    /**
     * 注册钩子 - 允许扩展插件注册回调函数
     * @param {string} hookName 钩子名称
     * @param {function} callback 回调函数
     * @param {object} options 选项（优先级、条件等）
     */
    addHook(hookName, callback, options = {}) {
        if (!this.hooks[hookName]) {
            this.hooks[hookName] = [];
        }
        
        const hookEntry = {
            callback,
            priority: options.priority || 10, // 默认优先级
            condition: options.condition, // 可选的执行条件
            once: options.once || false, // 是否只执行一次
            name: options.name || `hook_${Date.now()}`, // 钩子名称（用于移除）
            executed: false
        };
        
        this.hooks[hookName].push(hookEntry);
        
        // 按优先级排序（数字越小优先级越高）
        this.hooks[hookName].sort((a, b) => a.priority - b.priority);
        
        this.log(`钩子已注册: ${hookName} (优先级: ${hookEntry.priority})`, null, 'debug');
        
        return hookEntry.name; // 返回钩子ID，用于后续移除
    }

    /**
     * 移除钩子
     * @param {string} hookName 钩子名称
     * @param {string} hookId 钩子ID（注册时返回的名称）
     */
    removeHook(hookName, hookId) {
        if (!this.hooks[hookName]) return false;
        
        const index = this.hooks[hookName].findIndex(hook => hook.name === hookId);
        if (index !== -1) {
            this.hooks[hookName].splice(index, 1);
            this.log(`钩子已移除: ${hookName}/${hookId}`, null, 'debug');
            return true;
        }
        
        return false;
    }

    /**
     * 触发钩子 - 执行所有注册的回调函数
     * @param {string} hookName 钩子名称
     * @param {object} data 传递给钩子的数据
     * @param {object} context 上下文信息
     */
    async triggerHook(hookName, data = {}, context = {}) {
        if (!this.hooks[hookName] || this.hooks[hookName].length === 0) {
            return;
        }
        
        this.log(`触发钩子: ${hookName}`, { hookCount: this.hooks[hookName].length }, 'debug');
        
        const results = [];
        
        for (const hook of this.hooks[hookName]) {
            // 检查是否只执行一次且已执行
            if (hook.once && hook.executed) {
                continue;
            }
            
            // 检查执行条件
            if (hook.condition && typeof hook.condition === 'function') {
                try {
                    if (!hook.condition(data, context)) {
                        continue;
                    }
                } catch (error) {
                    this.log(`钩子条件检查失败: ${hookName}/${hook.name}`, error.message, 'warn');
                    continue;
                }
            }
            
            try {
                const startTime = Date.now();
                const result = await hook.callback(data, context);
                const duration = Date.now() - startTime;
                
                hook.executed = true;
                
                results.push({
                    hookName: hook.name,
                    success: true,
                    result,
                    duration
                });
                
                this.log(`钩子执行成功: ${hookName}/${hook.name} (${duration}ms)`, null, 'debug');
                
            } catch (error) {
                results.push({
                    hookName: hook.name,
                    success: false,
                    error: error.message
                });
                
                this.log(`钩子执行失败: ${hookName}/${hook.name}`, error.message, 'error');
                
                // 触发错误钩子
                if (hookName !== 'hook.error') {
                    await this.triggerHook('hook.error', {
                        originalHook: hookName,
                        hookName: hook.name,
                        error: error.message,
                        data
                    }, context);
                }
            }
        }
        
        return results;
    }

    /**
     * 注册内置钩子 - 为常用功能提供便捷方法
     */
    registerBuiltinHooks() {
        // 短期记忆存储完成后的钩子
        this.addHook('shortMemory.stored', async (data, context) => {
            this.log('短期记忆已存储', { filePath: data.filePath, size: data.size }, 'info');
            
            // 可以在这里添加后续处理，比如：
            // - 更新UI状态
            // - 发送通知
            // - 同步到云端
            // - 更新统计信息
        }, { priority: 1, name: 'builtin_shortMemory_logger' });
        
        // 会话数据保存完成后的钩子
        this.addHook('sessionData.saved', async (data, context) => {
            this.log('会话数据已保存', { sessionId: data.sessionId, filePath: data.filePath }, 'info');
        }, { priority: 1, name: 'builtin_sessionData_logger' });
        
        // 代码上下文更新完成后的钩子
        this.addHook('codeContext.updated', async (data, context) => {
            this.log('代码上下文已更新', { filePath: data.filePath }, 'info');
        }, { priority: 1, name: 'builtin_codeContext_logger' });
        
        // 用户配置变更后的钩子
        this.addHook('userConfig.changed', async (data, context) => {
            this.log('用户配置已更新', { configPath: data.filePath }, 'info');
        }, { priority: 1, name: 'builtin_userConfig_logger' });
        
        this.log('内置钩子已注册', null, 'debug');
    }

    /**
     * 获取钩子状态 - 用于调试和监控
     */
    getHookStatus() {
        const status = {};
        
        for (const [hookName, hooks] of Object.entries(this.hooks)) {
            status[hookName] = {
                count: hooks.length,
                hooks: hooks.map(hook => ({
                    name: hook.name,
                    priority: hook.priority,
                    executed: hook.executed,
                    once: hook.once
                }))
            };
        }
        
        return status;
    }
}

// 导出客户端类
module.exports = ContextKeeperMCPClient; 