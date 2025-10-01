/**
 * Context-Keeper 本地存储客户端示例
 * 基于第一期本地存储思路实现
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class LocalStorageClient {
    constructor() {
        // 获取操作系统标准应用数据目录 (兼容第一期逻辑)
        this.baseDir = this.getStandardDataDirectory();
        this.userConfigPath = path.join(this.baseDir, 'user-config.json');
        // 注意：sessions和histories应该通过expandPath动态获取用户隔离路径
        // 这些路径变量主要用于向后兼容，实际使用时会通过instruction.target获取具体路径
        this.historiesDir = path.join(this.baseDir, 'histories');
        this.codeContextDir = path.join(this.baseDir, 'code_context');
        this.shortMemoryDir = path.join(this.baseDir, 'short_memory');
        this.cacheDir = path.join(this.baseDir, 'cache');
        this.preferencesPath = path.join(this.baseDir, 'preferences.json');

        console.log('📁 Context-Keeper 本地存储路径:', this.baseDir);
    }

    /**
     * 获取操作系统标准应用数据目录 (第一期兼容)
     */
    getStandardDataDirectory() {
        const appName = 'context-keeper';
        const homeDir = os.homedir();

        switch (process.platform) {
            case 'darwin': // macOS
                return path.join(homeDir, 'Library', 'Application Support', appName);
            case 'win32': // Windows
                const appData = process.env.APPDATA;
                return appData ? 
                    path.join(appData, appName) : 
                    path.join(homeDir, 'AppData', 'Roaming', appName);
            default: // Linux and others
                const xdgDataHome = process.env.XDG_DATA_HOME;
                return xdgDataHome ? 
                    path.join(xdgDataHome, appName) : 
                    path.join(homeDir, '.local', 'share', appName);
        }
    }

    /**
     * 执行本地存储指令
     */
    async executeLocalInstruction(instruction) {
        try {
            console.log(`🔧 执行本地指令: ${instruction.type} -> ${instruction.target}`);
            
            switch (instruction.type) {
                case 'user_config':
                    await this.handleUserConfig(instruction);
                    break;
                case 'session_store':
                    await this.handleSessionStore(instruction);
                    break;
                case 'short_memory':
                    await this.handleShortMemory(instruction);
                    break;
                case 'code_context':
                    await this.handleCodeContext(instruction);
                    break;
                case 'preferences':
                    await this.handlePreferences(instruction);
                    break;
                case 'cache_update':
                    await this.handleCacheUpdate(instruction);
                    break;
                default:
                    throw new Error(`未知指令类型: ${instruction.type}`);
            }

            // 发送成功回调
            await this.sendCallback(instruction.callbackId, {
                success: true,
                type: instruction.type,
                timestamp: Date.now()
            });

            console.log(`✅ 本地指令执行成功: ${instruction.callbackId}`);

        } catch (error) {
            console.error(`❌ 本地指令执行失败: ${error.message}`);
            
            // 发送失败回调
            await this.sendCallback(instruction.callbackId, {
                success: false,
                error: error.message,
                type: instruction.type,
                timestamp: Date.now()
            });
        }
    }

    /**
     * 处理用户配置存储 (第一期兼容)
     */
    async handleUserConfig(instruction) {
        const { content, options } = instruction;
        
        // 确保目录存在
        if (options.createDir) {
            await this.ensureDirectory(path.dirname(this.userConfigPath));
        }

        // 备份现有配置
        if (options.backup && await this.fileExists(this.userConfigPath)) {
            const backupPath = `${this.userConfigPath}.bak.${Date.now()}`;
            await this.copyFile(this.userConfigPath, backupPath);
            console.log(`💾 备份用户配置: ${backupPath}`);
        }

        // 写入用户配置
        await this.writeJSON(this.userConfigPath, content);
        console.log(`✅ 用户配置已保存: ${content.userId}`);
    }

    /**
     * 处理会话存储 (第一期兼容)
     */
    async handleSessionStore(instruction) {
        const { content, options } = instruction;
        const sessionPath = this.expandPath(instruction.target);

        // 确保目录存在
        await this.ensureDirectory(path.dirname(sessionPath));

        // 合并现有会话数据
        if (options.merge && await this.fileExists(sessionPath)) {
            const existingSession = await this.readJSON(sessionPath);
            content.messages = [...(existingSession.messages || []), ...(content.messages || [])];
            content.codeContext = { ...(existingSession.codeContext || {}), ...(content.codeContext || {}) };
            content.editHistory = [...(existingSession.editHistory || []), ...(content.editHistory || [])];
        }

        // 清理旧数据
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(sessionPath), options.maxAge);
        }

        // 写入会话数据
        await this.writeJSON(sessionPath, content);
        console.log(`✅ 会话已保存: ${content.id}`);
    }

    /**
     * 处理短期记忆存储 (第一期兼容)
     */
    async handleShortMemory(instruction) {
        const { content, options } = instruction;
        const historyPath = this.expandPath(instruction.target);

        // 确保目录存在
        await this.ensureDirectory(path.dirname(historyPath));

        let finalHistory = content;

        // 合并到现有历史记录
        if (options.merge && await this.fileExists(historyPath)) {
            const existingHistory = await this.readJSON(historyPath);
            finalHistory = [...(existingHistory || []), ...content];
            
            // 保持最大长度限制 (第一期兼容: 最多20条)
            const maxHistory = 20;
            if (finalHistory.length > maxHistory) {
                finalHistory = finalHistory.slice(-maxHistory);
            }
        }

        // 清理旧数据
        if (options.cleanupOld && options.maxAge) {
            await this.cleanupOldFiles(path.dirname(historyPath), options.maxAge);
        }

        await this.writeJSON(historyPath, finalHistory);
        console.log(`✅ 短期记忆已保存: ${finalHistory.length}条记录`);
    }

    /**
     * 处理代码上下文存储 (第一期兼容)
     */
    async handleCodeContext(instruction) {
        const { content, options } = instruction;
        const contextPath = this.expandPath(instruction.target);

        // 确保目录存在
        await this.ensureDirectory(path.dirname(contextPath));

        let finalContext = content;

        // 合并到现有代码上下文
        if (options.merge && await this.fileExists(contextPath)) {
            const existingContext = await this.readJSON(contextPath);
            finalContext = { ...(existingContext || {}), ...content };
        }

        await this.writeJSON(contextPath, finalContext);
        console.log(`✅ 代码上下文已保存: ${Object.keys(content).length}个文件`);
    }

    /**
     * 处理偏好设置存储
     */
    async handlePreferences(instruction) {
        const { content, options } = instruction;

        // 确保目录存在
        await this.ensureDirectory(path.dirname(this.preferencesPath));

        let finalPrefs = content;

        // 合并用户偏好设置
        if (options.merge && await this.fileExists(this.preferencesPath)) {
            const existingPrefs = await this.readJSON(this.preferencesPath);
            finalPrefs = { ...existingPrefs, ...content };
        }

        await this.writeJSON(this.preferencesPath, finalPrefs);
        console.log(`✅ 用户偏好已保存`);
    }

    /**
     * 处理缓存更新
     */
    async handleCacheUpdate(instruction) {
        const { content } = instruction;
        const cachePath = this.expandPath(instruction.target);

        // 确保目录存在
        await this.ensureDirectory(path.dirname(cachePath));

        await this.writeJSON(cachePath, content);
        console.log(`✅ 缓存已更新: ${Object.keys(content.sessionStates || {}).length}个会话状态`);
    }

    /**
     * 发送回调到服务端
     */
    async sendCallback(callbackId, result) {
        const callbackData = {
            callbackId,
            success: result.success,
            data: result.success ? result : undefined,
            error: result.success ? undefined : result.error,
            timestamp: result.timestamp
        };

        // 这里可以实际发送HTTP请求到服务端
        console.log(`📤 发送回调: ${callbackId} (${result.success ? '成功' : '失败'})`);
        
        // 示例：实际的HTTP回调请求
        // try {
        //     const response = await fetch('http://localhost:8088/api/mcp/tools/local_operation_callback', {
        //         method: 'POST',
        //         headers: { 'Content-Type': 'application/json' },
        //         body: JSON.stringify(callbackData)
        //     });
        //     console.log('回调发送成功:', await response.json());
        // } catch (error) {
        //     console.error('回调发送失败:', error.message);
        // }
    }

    // 工具方法

    expandPath(targetPath) {
        if (targetPath.startsWith('~/')) {
            return path.join(os.homedir(), targetPath.slice(2));
        }
        if (targetPath.startsWith('~/.context-keeper/')) {
            // 支持用户隔离的路径结构 (第一期兼容)
            // 例如: ~/.context-keeper/users/{userId}/sessions/ -> /Users/xxx/.context-keeper/users/user123/sessions/
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

    async copyFile(src, dest) {
        await fs.copyFile(src, dest);
    }

    async readJSON(filePath) {
        const data = await fs.readFile(filePath, 'utf-8');
        return JSON.parse(data);
    }

    async writeJSON(filePath, data) {
        const jsonString = JSON.stringify(data, null, 2);
        await fs.writeFile(filePath, jsonString, 'utf-8');
    }

    async cleanupOldFiles(dirPath, maxAge) {
        try {
            const files = await fs.readdir(dirPath);
            const now = Date.now();
            
            for (const file of files) {
                const filePath = path.join(dirPath, file);
                const stats = await fs.stat(filePath);
                const ageInSeconds = (now - stats.mtime.getTime()) / 1000;
                
                if (ageInSeconds > maxAge) {
                    await fs.unlink(filePath);
                    console.log(`🗑️ 清理过期文件: ${file}`);
                }
            }
        } catch (error) {
            console.warn(`清理文件时出错: ${error.message}`);
        }
    }
}

// 示例使用
async function demo() {
    const client = new LocalStorageClient();

    // 模拟接收到的本地存储指令
    const userConfigInstruction = {
        type: 'user_config',
        target: '~/.context-keeper/user-config.json',
        content: {
            userId: 'user_1703123456',
            firstUsed: new Date().toISOString()
        },
        options: {
            createDir: true,
            backup: true
        },
        callbackId: 'user_init_user_1703123456',
        priority: 'high'
    };

    const sessionInstruction = {
        type: 'session_store',
        target: '~/.context-keeper/sessions/session-test.json',
        content: {
            id: 'session-test',
            createdAt: new Date().toISOString(),
            lastActive: new Date().toISOString(),
            status: 'active',
            messages: [
                {
                    role: 'user',
                    content: '你好，Context-Keeper！',
                    timestamp: Math.floor(Date.now() / 1000)
                }
            ],
            codeContext: {},
            editHistory: []
        },
        options: {
            createDir: true,
            merge: true,
            cleanupOld: true,
            maxAge: 30 * 24 * 3600 // 30天
        },
        callbackId: 'session_session-test_1703123456',
        priority: 'normal'
    };

    // 执行本地存储指令
    await client.executeLocalInstruction(userConfigInstruction);
    await client.executeLocalInstruction(sessionInstruction);
}

// 如果直接运行此文件，执行演示
if (require.main === module) {
    demo().catch(console.error);
}

module.exports = { LocalStorageClient }; 