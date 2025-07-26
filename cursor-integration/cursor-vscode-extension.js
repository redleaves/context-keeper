/**
 * Context-Keeper Cursor/VSCode 扩展
 * 提供内置的配置界面、状态管理、实时监控和WebSocket集成
 */

const vscode = require('vscode');
const fs = require('fs').promises;
const path = require('path');
const os = require('os');
const WebSocket = require('ws');

// 引入MCP客户端
const ContextKeeperMCPClient = require('./mcp-client.js');

class ContextKeeperExtension {
    constructor(context) {
        this.context = context;
        this.client = null;
        this.statusBarItem = null;
        this.outputChannel = null;
        this.configPath = path.join(os.homedir(), '.context-keeper', 'config', 'default-config.json');
        this.isActive = false;
        
        // 添加格式化时间的辅助方法
        this.formatLocalTime = () => {
            const now = new Date();
            return now.toLocaleString('zh-CN', { 
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
            }).replace(/\//g, '-');
        };
        
        // 添加日志辅助方法
        this.log = (message) => {
            if (this.outputChannel) {
                // 使用本地时间而非UTC时间
                const localTimeStr = this.formatLocalTime();
                this.outputChannel.appendLine(`[${localTimeStr}] ${message}`);
            }
        };
        
        // WebSocket集成 - 新增功能
        this.websocket = null;
        this.wsConnectionState = 'disconnected';
        this.userIdCheckInterval = null;
        this.pendingCallbacks = new Map();
        this.currentSessionId = null;
        this.config = null;
        // 🔥 新增：存储连接ID以便重连时使用
        this.currentConnectionId = null;
        this.currentUserId = null;
        this.currentWorkspaceRoot = null;
        
        this.init();
    }

    async init() {
        // 使用本地时间格式
        const localTimeStr = this.formatLocalTime();
        console.log(`[${localTimeStr}] 🧠 Context-Keeper扩展正在初始化...`);
        
        // 创建输出通道
        this.outputChannel = vscode.window.createOutputChannel('Context-Keeper');
        this.outputChannel.appendLine(`[${new Date().toISOString()}] Context-Keeper扩展已启动`);

        // 创建状态栏项
        this.createStatusBarItem();
        
        // 注册命令
        this.registerCommands();
        
        // 加载配置
        await this.loadFullConfig();
        
        // 🔥 添加配置变更监听
        this.setupConfigurationWatcher();
        
        // 初始化MCP客户端
        await this.initializeClient();
        
        // 启动WebSocket集成
        await this.initializeWebSocketIntegration();
        
        // 设置文件监听
        this.setupFileWatchers();
        
        // 自动启动功能
        await this.autoStart();
        
        // 使用本地时间格式
        const endTimeStr = this.formatLocalTime();
        console.log(`[${endTimeStr}] ✅ Context-Keeper扩展初始化完成`);
    }

    // WebSocket集成初始化
    async initializeWebSocketIntegration() {
        this.log('🔌 初始化WebSocket集成...');
        
        // 🔥 修复：确保清理旧的连接
        this.stopWebSocketServices();
        
        // 启动用户ID检查和自动连接
        this.startUserIdCheck();
        
        this.log('✅ WebSocket集成已启动');
    }

    // 🔥 新方法：设置配置变更监听器
    setupConfigurationWatcher() {
        // 监听配置变更
        const configWatcher = vscode.workspace.onDidChangeConfiguration(async (event) => {
            if (event.affectsConfiguration('context-keeper')) {
                this.log('[配置] 检测到配置变更，重新加载...');
                
                // 重新加载配置
                const oldConfig = this.config;
                this.config = await this.loadConfig();
                
                // 检查关键配置是否变更
                if (oldConfig?.serverConnection?.serverURL !== this.config.serverConnection.serverURL) {
                    this.log('[配置] 服务器URL变更，重新初始化客户端...');
                    await this.initializeClient();
                }
                
                if (oldConfig?.serverConnection?.websocketURL !== this.config.serverConnection.websocketURL) {
                    this.log('[配置] WebSocket URL变更，重新连接...');
                    this.stopUserIdCheck();
                    if (this.config.webSocket.autoConnect) {
                        this.startUserIdCheck();
                    }
                }
                
                if (oldConfig?.webSocket?.autoConnect !== this.config.webSocket.autoConnect) {
                    this.log(`[配置] WebSocket自动连接设置变更: ${this.config.webSocket.autoConnect}`);
                    if (this.config.webSocket.autoConnect) {
                        this.startUserIdCheck();
                    } else {
                        this.stopUserIdCheck();
                    }
                }
                
                if (oldConfig?.ui?.showStatusBar !== this.config.ui.showStatusBar) {
                    this.log(`[配置] 状态栏显示设置变更: ${this.config.ui.showStatusBar}`);
                    if (this.config.ui.showStatusBar) {
                        this.statusBarItem.show();
                    } else {
                        this.statusBarItem.hide();
                    }
                }
                
                // 显示通知（如果启用）
                if (this.config.ui.showNotifications) {
                    vscode.window.showInformationMessage('✅ Context Keeper配置已更新');
                }
            }
        });
        
        this.context.subscriptions.push(configWatcher);
    }

    // 启动连接检查循环
    startUserIdCheck() {
        // 🔥 修复：先停止已有的检查循环，避免多重定时器
        this.stopUserIdCheck();
        
        this.log('[WebSocket] 启动连接检查循环...');
        
        // 立即检查一次
        this.checkUserIdAndConnect();
        
        // 🔥 修复：确保只有一个定时器运行
        this.userIdCheckInterval = setInterval(() => {
            // 🔥 新增：如果已经连接，跳过检查以避免干扰
            if (this.wsConnectionState === 'connected') {
                return;
            }
            this.checkUserIdAndConnect();
        }, 5000); // 🔥 增加间隔时间，减少检查频率
    }

    // 🔥 新方法：停止用户ID检查循环
    stopUserIdCheck() {
        if (this.userIdCheckInterval) {
            this.log('[WebSocket] 停止连接检查循环...');
            clearInterval(this.userIdCheckInterval);
            this.userIdCheckInterval = null;
        }
    }

    // 🔥 新增：彻底停止WebSocket相关服务
    stopWebSocketServices() {
        this.stopUserIdCheck();
        this.stopHeartbeat();
        
        // 关闭WebSocket连接
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.log('[WebSocket] 关闭连接...');
            this.websocket.close(1000, 'Extension cleanup'); // 正常关闭
            this.wsConnectionState = 'disconnected';
            this.updateStatusBar('已断开', 'gray');
        }
    }

    // 🔥 检查WebSocket服务状态（修复：支持HTTPS协议）
    async checkWebSocketServiceHealth() {
        try {
            const serverURL = this.config?.serverConnection?.serverURL || 'http://localhost:8088';
            const url = new URL('/health', serverURL);
            
            // 🔧 修复：根据协议选择正确的模块
            const isHttps = url.protocol === 'https:';
            const httpModule = isHttps ? require('https') : require('http');
            
            this.log(`[健康检查] 🔍 检查服务状态: ${url.href} (${isHttps ? 'HTTPS' : 'HTTP'})`);
            
            return new Promise((resolve) => {
                const timeout = setTimeout(() => {
                    resolve(false);
                }, 5000);
                
                const req = httpModule.get(url, (res) => {
                    clearTimeout(timeout);
                    
                    if (res.statusCode === 200) {
                        let data = '';
                        res.on('data', chunk => data += chunk);
                        res.on('end', () => {
                            try {
                                const health = JSON.parse(data);
                                this.log(`[健康检查] ✅ 服务健康状态: ${JSON.stringify(health)}`);
                                resolve(health.websocket && health.websocket.connections >= 0);
                            } catch (err) {
                                this.log(`[健康检查] ❌ 解析健康检查响应失败: ${err.message}`);
                                resolve(false);
                            }
                        });
                    } else {
                        this.log(`[健康检查] ❌ 服务返回状态码: ${res.statusCode}`);
                        resolve(false);
                    }
                });
                
                req.on('error', (error) => {
                    clearTimeout(timeout);
                    this.log(`[健康检查] ❌ 请求错误: ${error.message}`);
                    resolve(false);
                });
                
                req.setTimeout(5000, () => {
                    req.destroy();
                    resolve(false);
                });
            });
        } catch (error) {
            this.log(`[健康检查] ❌ 服务检查失败: ${error.message}`);
            return false;
        }
    }

    // 🔥 检查用户ID并尝试连接WebSocket（直接集成优化版本）
    async checkUserIdAndConnect() {
        try {
            // 🔥 修复：更详细的状态检查和日志
            if (this.wsConnectionState === 'connected') {
                // this.log('[WebSocket] 跳过检查：已连接');
                return;
            }
            
            if (this.wsConnectionState === 'connecting') {
                this.log('[WebSocket] 跳过检查：正在连接中');
                return;
            }
            
            // 优化：先检查用户ID，再检查服务状态（减少不必要的网络请求）
            const userId = await this.getUserIdFromDisk();
            
            if (!userId) {
                if (this.wsConnectionState !== 'waiting_init') {
                    this.log('[WebSocket] ⚠️ 用户未初始化，等待MCP客户端初始化...');
                    this.wsConnectionState = 'waiting_init';
                    this.updateStatusBar('等待初始化', 'orange');
                }
                return;
            }
            
            // 检查服务健康状态（仅在有用户ID时）
            const isServiceHealthy = await this.checkWebSocketServiceHealth();
            if (!isServiceHealthy) {
                if (this.wsConnectionState !== 'service_unavailable') {
                    this.log('[WebSocket] ⚠️ 服务不可用，检查服务器是否启动...');
                    this.wsConnectionState = 'service_unavailable';
                    this.updateStatusBar('服务不可用', 'red');
                }
                return;
            }
            
            // 🔥 修复：再次检查状态，防止异步操作期间状态变化
            if (this.wsConnectionState === 'connected' || this.wsConnectionState === 'connecting') {
                this.log('[WebSocket] 跳过连接：状态已变化');
                return;
            }
            
            // 尝试建立连接
            this.log(`[WebSocket] ✅ 发现用户ID: ${userId}`);
            this.log('[WebSocket] 🚀 建立WebSocket连接...');
            this.updateStatusBar('连接中...', 'yellow');
            await this.connectWebSocket(userId);
            
        } catch (error) {
            this.log(`[WebSocket] ❌ 连接检查失败: ${error.message}`);
            this.wsConnectionState = 'error';
            this.updateStatusBar('连接错误', 'red');
        }
    }

    // 从本地磁盘获取用户ID
    async getUserIdFromDisk() {
        try {
            const baseDir = path.join(os.homedir(), 'Library', 'Application Support', 'context-keeper');
            
            // 检查全局配置文件
            const globalConfigPath = path.join(baseDir, 'user-config.json');
            try {
                const globalConfig = JSON.parse(await fs.readFile(globalConfigPath, 'utf8'));
                if (globalConfig.userId) {
                    return globalConfig.userId;
                }
            } catch (err) {
                // 全局配置文件不存在，继续其他方法
            }
            
            // 扫描users目录，查找活跃用户
            const usersDir = path.join(baseDir, 'users');
            try {
                const userDirs = await fs.readdir(usersDir);
                
                for (const userDir of userDirs) {
                    if (userDir.startsWith('user_')) {
                        const userConfigPath = path.join(usersDir, userDir, 'user-config.json');
                        try {
                            const userConfig = JSON.parse(await fs.readFile(userConfigPath, 'utf8'));
                            if (userConfig.userId && userConfig.active !== false) {
                                return userConfig.userId;
                            }
                        } catch (err) {
                            continue;
                        }
                    }
                }
            } catch (err) {
                // users目录不存在
            }
            
            return null;
            
        } catch (error) {
            console.error('[UserID] 从磁盘获取用户ID时出错:', error);
            return null;
        }
    }

    // 建立WebSocket连接
    async connectWebSocket(userId) {
        try {
            this.log(`[WebSocket] 🚀 开始建立WebSocket连接，用户ID: ${userId}`);
            
            // 先停止现有连接
            if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
                this.log('[WebSocket] 🔄 关闭现有连接以建立新连接');
                this.websocket.close(1000, 'New connection attempt');
                this.websocket = null;
            }
            
            this.wsConnectionState = 'connecting';
            
            // 🔥 重构：获取工作空间信息，计算工作空间哈希
            const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri?.fsPath;
            
            // 🚨 修复：如果没有工作空间，不建立连接，避免创建unknown工作空间
            if (!workspaceRoot) {
                this.log('[WebSocket] ⚠️ 没有活跃工作空间，跳过WebSocket连接');
                this.wsConnectionState = 'no_workspace';
                this.updateStatusBar('无工作空间', 'orange');
                return;
            }
            
                // 🔥 修复：使用与服务端完全一致的SHA256哈希算法生成工作空间标识
    const workspaceHash = this.generateWorkspaceHash(workspaceRoot);
    const connectionId = `${userId}_ws_${workspaceHash}`;
            
            // 🔥 存储连接信息以便重连时使用
            this.currentConnectionId = connectionId;
            this.currentUserId = userId;
            this.currentWorkspaceRoot = workspaceRoot;
            
            // 优先使用用户配置的WebSocket地址
            const userWebSocketURL = this.config?.serverConnection?.websocketURL;
            let wsURL;
            
            if (userWebSocketURL && userWebSocketURL.trim()) {
                wsURL = userWebSocketURL.trim();
                this.log(`[WebSocket] 🎯 使用配置的WebSocket地址: ${wsURL}`);
            } else {
                // 默认根据serverURL自动生成
                const serverURL = this.config?.serverURL || 'http://localhost:8088';
                // 🔧 修复：正确处理https到wss的转换
                wsURL = serverURL.replace(/^https?/, serverURL.startsWith('https') ? 'wss' : 'ws') + '/ws';
                this.log(`[WebSocket] 🔧 自动生成WebSocket地址: ${wsURL}`);
            }
            
            // 🔥 重构：构建包含工作空间信息的连接URL，移除哈希处理
            const fullURL = `${wsURL}?userId=${encodeURIComponent(userId)}&workspace=${encodeURIComponent(workspaceRoot)}`;
            
            this.log(`[WebSocket] 📁 工作空间: ${workspaceRoot}`);
            this.log(`[WebSocket] 🔑 连接ID: ${connectionId}`);
            this.log(`[WebSocket] 🌐 连接到: ${fullURL}`);
            
            this.websocket = new WebSocket(fullURL);
            
            this.websocket.onopen = async () => {
                this.log('[WebSocket] 🎉 连接建立成功！');
                this.wsConnectionState = 'connected';
                this.updateStatusBar('已连接', 'lightgreen');
                this.startHeartbeat();
                
                // 🔥 修复：连接成功后，停止连接检查循环，避免干扰
                this.stopUserIdCheck();
                
                // 🔥 新增：注册当前活跃会话到WebSocket连接
                await this.registerActiveSession();
                
                // 显示成功通知
                vscode.window.showInformationMessage('Context-Keeper WebSocket连接成功');
            };
            
            this.websocket.onmessage = (event) => {
                this.handleWebSocketMessage(event);
            };
            
            this.websocket.onclose = (event) => {
                this.log(`[WebSocket] 🔌 连接关闭: ${event.code} - ${event.reason}`);
                this.wsConnectionState = 'disconnected';
                this.updateStatusBar('连接断开', 'red');
                this.stopHeartbeat();
                
                // 🔥 修复：改进重连逻辑，避免与定时器冲突
                if (event.code !== 1000) { // 非正常关闭才重连
                    this.log('[WebSocket] 🔄 5秒后自动重连...');
                    // 🔥 修复：使用单次重连，而不是启动定时器
                    setTimeout(() => {
                        if (this.wsConnectionState === 'disconnected' && this.currentUserId) {
                            this.log('[WebSocket] 🚀 开始自动重连...');
                            // 🔥 修复：使用存储的用户ID重连，确保工作空间一致性
                            this.connectWebSocket(this.currentUserId);
                        }
                    }, 5000);
                } else {
                    this.log('[WebSocket] ✅ 连接正常关闭，不进行重连');
                }
            };
            
            this.websocket.onerror = (error) => {
                this.log(`[WebSocket] ❌ 连接错误: ${error.message || '未知错误'}`);
                this.wsConnectionState = 'error';
                this.updateStatusBar('连接错误', 'red');
            };
            
        } catch (error) {
            this.log(`[WebSocket] ❌ 连接失败: ${error.message}`);
            this.wsConnectionState = 'error';
            this.updateStatusBar('连接失败', 'red');
        }
    }

    // 🔥 新增：注册当前活跃会话到WebSocket连接
    async registerActiveSession() {
        try {
            // 获取或创建当前活跃的sessionId
            const sessionId = await this.getOrCreateActiveSession();
            
            if (!sessionId) {
                this.log('[会话注册] ⚠️ 无法获取活跃会话ID，跳过注册');
                return;
            }
            
            // 添加日志，显示会话ID的格式和服务器时间
            this.log(`[会话注册] 📅 getOrCreateActiveSession: ${sessionId}`);
            if (sessionId.startsWith('session-')) {
                const parts = sessionId.split('-');
                if (parts.length >= 3) {
                    const dateStr = parts[1]; // 20250703
                    const timeStr = parts[2]; // 142210
                    if (dateStr.length === 8 && timeStr.length >= 6) {
                        const year = dateStr.substring(0, 4);
                        const month = dateStr.substring(4, 6);
                        const day = dateStr.substring(6, 8);
                        const hour = timeStr.substring(0, 2);
                        const minute = timeStr.substring(2, 4);
                        const second = timeStr.substring(4, 6);
                        this.log(`[会话注册] 📅 服务器时间解析: ${year}-${month}-${day} ${hour}:${minute}:${second}`);
                        this.log(`[会话注册] 📅 当前本地时间: ${this.formatLocalTime()}`);
                    }
                }
            }
            
            // 向服务端注册会话映射
            const registerUrl = `${this.config?.serverURL || 'http://localhost:8088'}/api/ws/register-session`;
            const registerData = {
                sessionId: sessionId,
                connectionId: this.currentConnectionId // 已修复为包含工作空间标识的连接ID
            };
            
            this.log(`[会话注册] 📋 注册会话: ${sessionId} → 连接: ${this.currentConnectionId}`);
            
            const response = await fetch(registerUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(registerData)
            });
            
            if (response.ok) {
                const result = await response.json();
                this.log(`[会话注册] ✅ 会话注册成功: ${sessionId}`);
                
                // 存储服务端返回的实际连接ID，以防服务端做了修正
                if (result.connectionId && result.connectionId !== this.currentConnectionId) {
                    this.log(`[会话注册] ℹ️ 服务端修正了连接ID: ${this.currentConnectionId} → ${result.connectionId}`);
                    this.currentConnectionId = result.connectionId;
                }
                
                // 存储当前活跃的会话ID，供MCP工具调用使用
                this.currentSessionId = sessionId;
            } else {
                const errorText = await response.text();
                this.log(`[会话注册] ❌ 会话注册失败: ${response.status} - ${errorText}`);
            }
            
        } catch (error) {
            this.log(`[会话注册] ❌ 会话注册异常: ${error.message}`);
        }
    }
    
    // 🔥 新增：获取或创建当前活跃会话
    async getOrCreateActiveSession() {
        try {
            // 1. 获取当前工作空间和用户信息
            const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri?.fsPath;
            const userId = this.currentUserId || await this.getUserIdFromDisk();
            
            // 🚨 修复：如果没有工作空间，不创建会话
            if (!workspaceRoot) {
                this.log(`[会话管理] ⚠️ 没有活跃工作空间，无法创建会话`);
                return null;
            }
            
            if (!userId) {
                this.log(`[会话管理] ❌ 无法获取用户ID，会话创建失败`);
                return null;
            }
            
            // 2. 向服务端请求基于用户ID和工作空间的会话
            // 🔥 修复：添加必需的workspaceRoot参数
            const getSessionUrl = `${this.config?.serverURL || 'http://localhost:8088'}/mcp`;
            const getSessionData = {
                jsonrpc: '2.0',
                id: Date.now(),
                method: 'tools/call',
                params: {
                    name: 'session_management',
                    arguments: {
                        action: 'get_or_create',
                        userId: userId,
                        workspaceRoot: workspaceRoot,  // 🔥 修复：添加必需的workspaceRoot参数
                        metadata: {
                            vscodeVersion: vscode.version,
                            extensionVersion: '1.0.0',
                            clientTimestamp: Date.now(),
                            clientTime: this.formatLocalTime()
                        }
                    }
                }
            };
            
            this.log(`[会话管理] 🔍 请求会话，用户ID: ${userId}, 工作空间: ${workspaceRoot}`);
            
            const response = await fetch(getSessionUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(getSessionData)
            });
            
            if (response.ok) {
                const result = await response.json();
                if (result.result && result.result.content && result.result.content[0]) {
                    const contentText = result.result.content[0].text;
                    const parsedContent = JSON.parse(contentText);

                    if (parsedContent.sessionId) {
                        const sessionId = parsedContent.sessionId;

                        // 记录会话ID但不缓存，每次都从服务端获取
                        this.currentSessionId = sessionId;

                        this.log(`[会话管理] ✅ 会话获取成功: ${sessionId}`);
                        this.log(`[会话管理] 📁 工作空间: ${workspaceRoot}`);
                        return sessionId;
                    }
                }
            }

            this.log('[会话管理] ❌ 获取会话失败');
            return null;
            
        } catch (error) {
            this.log(`[会话管理] ❌ 会话管理异常: ${error.message}`);
            return null;
        }
    }

    // 🔥 新增：处理WebSocket消息（从cursor-extension.js移植）
    handleWebSocketMessage(event) {
        try {
            const message = JSON.parse(event.data);
            this.log(`[WebSocket] 📨 收到消息: ${message.type}`);
            
            // 🔥 添加调试：显示完整消息内容
            if (message.type === 'instruction') {
                this.log(`[调试] 完整消息: ${JSON.stringify(message, null, 2)}`);
            }
            
            switch (message.type) {
                case 'instruction':
                    this.executeWebSocketInstruction(message.data);
                    break;
                case 'callback_result':
                    this.handleCallbackResult(message);
                    break;
                case 'ping':
                    this.websocket.send(JSON.stringify({ type: 'pong' }));
                    break;
                default:
                    this.log(`[WebSocket] ⚠️ 未知消息类型: ${message.type}`);
            }
        } catch (error) {
            this.log(`[WebSocket] ❌ 处理消息失败: ${error.message}`);
        }
    }

    // 🔥 新增：执行WebSocket指令（从cursor-extension.js移植）
    async executeWebSocketInstruction(instruction) {
        try {
            this.log(`[指令执行] 🎯 执行指令: ${instruction.type}`);
            
            let result;
            switch (instruction.type) {
                case 'short_memory':
                    // 🔥 修复：使用正确的字段名 target 而不是 targetPath
                    this.log(`[调试] 指令详情: target=${instruction.target}, content类型=${typeof instruction.content}`);
                    if (!instruction.target) {
                        result = { success: false, error: '缺少target路径参数' };
                    } else {
                        result = await this.handleShortMemoryDirect(instruction.target, instruction.content, instruction.options);
                    }
                    break;
                case 'local_instruction':
                    result = await this.executeLocalInstructionDirect(instruction);
                    break;
                case 'user_config':
                case 'session_store':
                case 'code_context':
                case 'preferences':
                case 'cache_update':
                    // 🔥 新增：处理其他类型的本地指令，统一使用 target 字段
                    result = await this.executeLocalInstructionDirect({
                        target: instruction.target,
                        content: instruction.content,
                        options: instruction.options
                    });
                    break;
                default:
                    result = { success: false, error: `未知指令类型: ${instruction.type}` };
            }
            
            // 🔥 修复：使用正确的字段名 callbackId
            if (instruction.callbackId) {
                this.sendCallbackResult(instruction.callbackId, result);
            }
            
            this.log(`[指令执行] ✅ 指令执行完成: ${JSON.stringify(result)}`);
            
        } catch (error) {
            this.log(`[指令执行] ❌ 指令执行失败: ${error.message}`);
            
            // 🔥 修复：使用正确的字段名 callbackId
            if (instruction.callbackId) {
                this.sendCallbackResult(instruction.callbackId, {
                    success: false,
                    error: error.message
                });
            }
        }
    }

    // 🔥 新增：执行本地指令（从cursor-extension.js移植）
    async executeLocalInstructionDirect(instruction) {
        try {
            // 🔥 修复：使用正确的字段名 target 而不是 targetPath
            const { target, content } = instruction;
            
            // 展开路径模板
            const expandedPath = this.expandPath(target);
            
            this.outputChannel.appendLine(`[本地指令] 📂 目标路径: ${expandedPath}`);
            
            // 确保目录存在
            const dir = path.dirname(expandedPath);
            await fs.mkdir(dir, { recursive: true });
            
            // 🔥 修复：处理不同类型的内容格式
            let finalContent;
            if (Array.isArray(content)) {
                // 如果是数组（如短期记忆的历史记录），转换为JSON字符串
                finalContent = JSON.stringify(content, null, 2);
            } else if (typeof content === 'object') {
                // 如果是对象，转换为JSON字符串
                finalContent = JSON.stringify(content, null, 2);
            } else {
                // 如果是字符串，直接使用
                finalContent = content;
            }
            
            // 写入内容
            await fs.writeFile(expandedPath, finalContent, 'utf8');
            
            this.outputChannel.appendLine(`[本地指令] 💾 文件已写入: ${expandedPath}`);
            
            return {
                success: true,
                message: `文件已成功写入: ${expandedPath}`,
                targetPath: expandedPath
            };
            
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    // 🔥 新增：处理短期记忆（从cursor-extension.js移植）
    async handleShortMemoryDirect(targetPath, content, options = {}) {
        try {
            const expandedPath = this.expandPath(targetPath);
            
            this.outputChannel.appendLine(`[短期记忆] 📝 存储到: ${expandedPath}`);
            
            // 确保目录存在
            const dir = path.dirname(expandedPath);
            await fs.mkdir(dir, { recursive: true });
            
            // 🔥 修复：按照第一期标准处理JSON数组格式
            let finalHistory = Array.isArray(content) ? content : [content];

            // 合并到现有历史记录（第一期兼容）
            if (options.merge) {
                try {
                    const existingData = await fs.readFile(expandedPath, 'utf8');
                    const existingHistory = JSON.parse(existingData);
                    if (Array.isArray(existingHistory)) {
                        finalHistory = [...existingHistory, ...finalHistory];
                        
                        // 保持最大长度限制（第一期兼容：最多20条）
                        const maxHistory = 20;
                        if (finalHistory.length > maxHistory) {
                            finalHistory = finalHistory.slice(-maxHistory);
                        }
                    }
                } catch (error) {
                    // 如果读取或解析失败，使用新的历史记录
                    this.outputChannel.appendLine(`[短期记忆] ℹ️ 无现有历史记录或格式错误，创建新记录`);
                }
            }

            // 🔥 修复：以JSON数组格式写入文件
            const jsonContent = JSON.stringify(finalHistory, null, 2);
            await fs.writeFile(expandedPath, jsonContent, 'utf8');
            
            this.outputChannel.appendLine(`[短期记忆] ✅ 记忆已存储: ${expandedPath} (${finalHistory.length}条记录)`);
            
            return {
                success: true,
                message: `短期记忆已存储到: ${expandedPath}`,
                targetPath: expandedPath,
                recordCount: finalHistory.length
            };
            
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    // 🔥 新增：展开路径模板（从cursor-extension.js移植）
    expandPath(pathTemplate) {
        // 🔥 添加防护：检查参数是否有效
        if (!pathTemplate || typeof pathTemplate !== 'string') {
            throw new Error(`无效的路径模板: ${pathTemplate}`);
        }
        
        this.outputChannel.appendLine(`[路径展开] 🔧 原始路径: ${pathTemplate}`);
        
        let expandedPath = pathTemplate
            // 🔥 关键修复：处理 ~ 开头的路径
            .replace(/^~/, os.homedir())
            .replace(/\$\{HOME\}/g, os.homedir())
            .replace(/\$\{USER\}/g, os.userInfo().username)
            .replace(/\$\{DATE\}/g, new Date().toISOString().split('T')[0])
            .replace(/\$\{TIMESTAMP\}/g, new Date().toISOString());
            
        this.outputChannel.appendLine(`[路径展开] ✅ 展开后路径: ${expandedPath}`);
        return expandedPath;
    }

    // 🔥 新增：发送回调结果（从cursor-extension.js移植）
    sendCallbackResult(callbackId, result) {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            // 🔥 修复：使用服务端期望的回调消息格式
            const message = {
                type: 'callback',
                callbackId,
                success: result.success,
                message: result.message || result.error || '',
                data: result.data || result,
                timestamp: Date.now()
            };
            this.websocket.send(JSON.stringify(message));
            this.log(`[WebSocket] 📤 发送回调结果: ${callbackId} - ${result.success ? '成功' : '失败'}`);
        }
    }

    // 🔥 新增：启动心跳（从cursor-extension.js移植）
    startHeartbeat() {
        // ✅ 新增：WebSocket协议级别心跳监控
        // Node.js WebSocket会自动回复服务端的ping帧，无需手动处理
        // 只需要监控连接状态，确保连接健康
        this.heartbeatInterval = setInterval(() => {
            if (this.websocket) {
                const state = this.websocket.readyState;
                if (state === WebSocket.CONNECTING) {
                    this.log('[心跳] WebSocket连接中...');
                } else if (state === WebSocket.OPEN) {
                    this.log('[心跳] WebSocket连接正常');
                } else if (state === WebSocket.CLOSING || state === WebSocket.CLOSED) {
                    this.log('[心跳] WebSocket连接已关闭，准备重连...');
                    this.checkUserIdAndConnect();
                }
            }
        }, 30000); // 30秒检查一次连接状态
    }

    // 🔥 新增：停止心跳（从cursor-extension.js移植）
    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    createStatusBarItem() {
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.text = '$(brain) Context-Keeper';
        this.statusBarItem.tooltip = 'Context-Keeper状态';
        this.statusBarItem.command = 'context-keeper.openSettings';
        
        // 根据配置决定是否显示
        if (this.config?.ui?.showStatusBar !== false) {
        this.statusBarItem.show();
        }
        
        this.context.subscriptions.push(this.statusBarItem);
    }

    updateStatusBar(status, color = 'white') {
        if (!this.statusBarItem) return;
        
        const colorMap = {
            'lightgreen': '$(check)',
            'red': '$(error)',
            'yellow': '$(loading~spin)',
            'orange': '$(warning)',
            'gray': '$(circle-slash)'
        };
        
        const icon = colorMap[color] || '$(brain)';
        this.statusBarItem.text = `${icon} ${status}`;
            this.statusBarItem.tooltip = `Context-Keeper: ${status}`;
        
        // 根据配置决定是否显示
        if (this.config?.ui?.showStatusBar !== false) {
            this.statusBarItem.show();
        }
    }

    registerCommands() {
        const commands = [
            // 主要命令
            vscode.commands.registerCommand('context-keeper.showStatus', () => this.showStatusPanel()),
            vscode.commands.registerCommand('context-keeper.openSettings', () => this.openSettingsPanel()),
            vscode.commands.registerCommand('context-keeper.showLogs', () => this.showLogsPanel()),
            vscode.commands.registerCommand('context-keeper.testConnection', () => this.testConnection()),
            
            // 配置命令 - 使用VSCode内置设置
            vscode.commands.registerCommand('context-keeper.configureSettings', () => this.openVSCodeSettings()),
            vscode.commands.registerCommand('context-keeper.resetConfig', () => this.resetConfig()),
            vscode.commands.registerCommand('context-keeper.exportConfig', () => this.exportConfig()),
            vscode.commands.registerCommand('context-keeper.importConfig', () => this.importConfig()),
            
            // 数据管理
            vscode.commands.registerCommand('context-keeper.clearData', () => this.clearUserData()),
            vscode.commands.registerCommand('context-keeper.backupData', () => this.backupUserData()),
            vscode.commands.registerCommand('context-keeper.showUserData', () => this.showUserDataPanel()),
            
            // 服务管理
            vscode.commands.registerCommand('context-keeper.start', () => this.startService()),
            vscode.commands.registerCommand('context-keeper.stop', () => this.stopService()),
            vscode.commands.registerCommand('context-keeper.restart', () => this.restartService()),
            
            // 用户配置管理
            vscode.commands.registerCommand('context-keeper.saveUserConfig', (userId) => this.handleSaveUserConfig(userId)),
            vscode.commands.registerCommand('context-keeper.editUserConfig', () => this.handleEditUserConfig()),
            vscode.commands.registerCommand('context-keeper.resetUserConfig', () => this.handleResetUserConfig()),
        ];

        commands.forEach(command => {
            this.context.subscriptions.push(command);
        });
    }

    // 🔥 新方法：直接打开VSCode设置页面到Context Keeper部分
    async openVSCodeSettings() {
        // 打开设置并过滤到Context Keeper相关设置
        await vscode.commands.executeCommand('workbench.action.openSettings', '@ext:context-keeper');
        
        vscode.window.showInformationMessage(
            '💡 在这里可以配置所有Context Keeper设置！',
            '查看服务器设置',
            '查看自动化功能'
        ).then(selection => {
            if (selection === '查看服务器设置') {
                vscode.commands.executeCommand('workbench.action.openSettings', 'context-keeper.serverURL');
            } else if (selection === '查看自动化功能') {
                vscode.commands.executeCommand('workbench.action.openSettings', 'context-keeper.autoCapture');
        }
        });
    }

    // 🔥 重构：使用VSCode配置API读取配置
    async loadConfig() {
        const config = vscode.workspace.getConfiguration('context-keeper');
        
        return {
                serverConnection: {
            serverURL: config.get('serverURL', 'http://localhost:8088'),
            websocketURL: config.get('websocketURL', ''),
            timeout: config.get('timeout', 15000)
                },
                userSettings: {
                userId: config.get('userId', ''),
                accessCode: config.get('accessCode', ''),
                    baseDir: path.join(os.homedir(), '.context-keeper')
                },
                automationFeatures: {
                autoCapture: config.get('autoCapture', true),
                autoAssociate: config.get('autoAssociate', true),
                autoRecord: config.get('autoRecord', true),
                captureInterval: config.get('captureInterval', 30)
                },
                logging: {
                enabled: config.get('logging.enabled', true),
                level: config.get('logging.level', 'info')
            },
            webSocket: {
                autoConnect: config.get('webSocket.autoConnect', true),
                reconnectAttempts: config.get('webSocket.reconnectAttempts', 5)
            },
            ui: {
                showStatusBar: config.get('ui.showStatusBar', true),
                showNotifications: config.get('ui.showNotifications', true)
            },
            memory: {
                maxShortTermEntries: config.get('memory.maxShortTermEntries', 100),
                autoCleanup: config.get('memory.autoCleanup', true)
            }
        };
    }

    // 🔥 新方法：保存配置到VSCode设置
    async saveConfigToVSCode(configSection, key, value) {
        const config = vscode.workspace.getConfiguration('context-keeper');
        await config.update(key, value, vscode.ConfigurationTarget.Global);
        
        this.outputChannel.appendLine(`✅ 配置已保存: ${configSection}.${key} = ${value}`);
        
        // 重新加载配置
        this.config = await this.loadConfig();
        
        // 重新初始化相关服务
        if (configSection === 'serverConnection') {
            await this.initializeClient();
        }
        
        if (configSection === 'webSocket' && key === 'autoConnect') {
            if (value) {
                this.startUserIdCheck();
            } else {
                this.stopUserIdCheck();
            }
        }
    }

    // 🔥 简化的设置面板：移除浏览器依赖
    async openSettingsPanel() {
        // 直接打开VSCode设置，而不是外部HTML
        const action = await vscode.window.showQuickPick([
            {
                label: '$(gear) 打开设置页面',
                description: '在VSCode设置中配置Context Keeper',
                action: 'settings'
            },
            {
                label: '$(dashboard) 查看状态面板',
                description: '显示连接状态和数据统计',
                action: 'status'
            },
            {
                label: '$(test-view) 测试连接',
                description: '测试与服务器的连接',
                action: 'test'
            },
            {
                label: '$(database) 管理数据',
                description: '备份、清理或导出数据',
                action: 'data'
            }
        ], {
            placeHolder: '选择要执行的操作',
            title: 'Context Keeper 管理'
        });

        if (!action) return;

        switch (action.action) {
            case 'settings':
                await this.openVSCodeSettings();
                break;
            case 'status':
                await this.showStatusPanel();
                break;
            case 'test':
                await this.testConnection();
                break;
            case 'data':
                await this.showDataManagementPanel();
                break;
        }
    }

    // 🔥 新方法：数据管理面板
    async showDataManagementPanel() {
        const action = await vscode.window.showQuickPick([
            {
                label: '$(export) 备份数据',
                description: '备份用户数据到文件',
                action: 'backup'
            },
            {
                label: '$(import) 导入配置',
                description: '从文件导入配置',
                action: 'import'
            },
            {
                label: '$(export) 导出配置',
                description: '导出当前配置到文件',
                action: 'export'
            },
            {
                label: '$(trash) 清理数据',
                description: '清理用户数据（谨慎操作）',
                action: 'clear'
            },
            {
                label: '$(refresh) 重置配置',
                description: '重置所有配置为默认值',
                action: 'reset'
            }
        ], {
            placeHolder: '选择数据管理操作',
            title: 'Context Keeper 数据管理'
        });

        if (!action) return;

        switch (action.action) {
            case 'backup':
                await this.backupUserData();
                break;
            case 'import':
                await this.importConfig();
                break;
            case 'export':
                await this.exportConfig();
                break;
            case 'clear':
                await this.clearUserData();
                break;
            case 'reset':
                await this.resetConfig();
                break;
        }
    }

    async initializeClient() {
        try {
            const config = await this.loadConfig();
            this.client = new ContextKeeperMCPClient(config);
            
            // 测试连接
            const healthCheck = await this.client.healthCheck();
            if (healthCheck.success) {
                this.isActive = true;
                this.log('✅ MCP客户端连接成功');
            } else {
                this.log(`❌ MCP客户端连接失败: ${healthCheck.message}`);
            }
        } catch (error) {
            this.log(`❌ 客户端初始化失败: ${error.message}`);
        }
    }

    // 🔥 简化：使用VSCode配置API代替文件配置
    async loadFullConfig() {
        try {
            // 直接使用loadConfig方法，它已经使用VSCode配置API
            this.config = await this.loadConfig();
            this.outputChannel.appendLine('✅ 配置加载完成');
        } catch (error) {
            this.outputChannel.appendLine(`❌ 配置加载失败: ${error.message}`);
            // 使用默认配置
            this.config = await this.loadConfig();
        }
    }

    async saveConfig(config) {
        try {
            const configDir = path.dirname(this.configPath);
            await fs.mkdir(configDir, { recursive: true });
            await fs.writeFile(this.configPath, JSON.stringify(config, null, 2));
            
            // 🔥 更新内部配置
            this.config = config;
            
            // 重新初始化客户端
            await this.initializeClient();
            
            vscode.window.showInformationMessage('配置已保存并生效');
        } catch (error) {
            vscode.window.showErrorMessage(`保存配置失败: ${error.message}`);
        }
    }

    async showStatusPanel() {
        const panel = vscode.window.createWebviewPanel(
            'context-keeper-status',
            'Context-Keeper 状态',
            vscode.ViewColumn.One,
            {
                enableScripts: true,
                retainContextWhenHidden: true
            }
        );

        panel.webview.html = await this.getStatusPanelHTML();
        
        // 处理来自webview的消息
        panel.webview.onDidReceiveMessage(async (message) => {
            switch (message.command) {
                case 'refresh':
                    panel.webview.html = await this.getStatusPanelHTML();
                    break;
                case 'testConnection':
                    await this.testConnection();
                    panel.webview.html = await this.getStatusPanelHTML();
                    break;
                case 'openSettings':
                    await this.openSettingsPanel();
                    break;
                case 'saveUserConfig':
                    await this.handleSaveUserConfig(message.userId);
                    panel.webview.html = await this.getStatusPanelHTML(); // 刷新界面
                    break;
                case 'editUserConfig':
                    await this.handleEditUserConfig();
                    panel.webview.html = await this.getStatusPanelHTML(); // 刷新界面
                    break;
                case 'resetUserConfig':
                    await this.handleResetUserConfig();
                    panel.webview.html = await this.getStatusPanelHTML(); // 刷新界面
                    break;
                default:
                    this.outputChannel.appendLine(`未知消息: ${message.command}`);
            }
        });
    }

    async getStatusPanelHTML() {
        const config = await this.loadConfig();
        const healthCheck = this.client ? await this.client.healthCheck() : { success: false, message: '客户端未初始化' };
        
        // 获取用户数据统计
        const userStats = await this.getUserDataStats();
        
        // 获取用户配置状态
        const userConfigStatus = await this.getUserConfigStatus();
        
        return `
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <title>Context-Keeper 状态</title>
            <style>
                body { 
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                    padding: 20px;
                    background-color: var(--vscode-editor-background);
                    color: var(--vscode-editor-foreground);
                }
                .status-card {
                    background: var(--vscode-editor-inactiveSelectionBackground);
                    border: 1px solid var(--vscode-panel-border);
                    border-radius: 8px;
                    padding: 16px;
                    margin: 10px 0;
                }
                .status-indicator {
                    display: inline-block;
                    width: 12px;
                    height: 12px;
                    border-radius: 50%;
                    margin-right: 8px;
                }
                .status-connected { background-color: #4CAF50; }
                .status-disconnected { background-color: #f44336; }
                .status-warning { background-color: #ff9800; }
                .btn {
                    background: var(--vscode-button-background);
                    color: var(--vscode-button-foreground);
                    border: none;
                    padding: 8px 16px;
                    border-radius: 4px;
                    margin: 4px;
                    cursor: pointer;
                }
                .btn:hover {
                    background: var(--vscode-button-hoverBackground);
                }
                .btn-primary {
                    background: var(--vscode-button-background);
                    color: var(--vscode-button-foreground);
                }
                .btn-secondary {
                    background: var(--vscode-button-secondaryBackground);
                    color: var(--vscode-button-secondaryForeground);
                }
                .stats-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                    gap: 10px;
                    margin: 10px 0;
                }
                .stat-item {
                    text-align: center;
                    padding: 10px;
                    background: var(--vscode-editorWidget-background);
                    border-radius: 4px;
                }
                .stat-number {
                    font-size: 24px;
                    font-weight: bold;
                    color: var(--vscode-textLink-foreground);
                }
                .user-config-form {
                    margin: 15px 0;
                }
                .form-group {
                    margin: 10px 0;
                }
                .form-group label {
                    display: block;
                    margin-bottom: 5px;
                    font-weight: bold;
                }
                .form-group input {
                    width: 100%;
                    padding: 8px;
                    border: 1px solid var(--vscode-input-border);
                    border-radius: 4px;
                    background: var(--vscode-input-background);
                    color: var(--vscode-input-foreground);
                }
                .alert {
                    padding: 10px;
                    margin: 10px 0;
                    border-radius: 4px;
                    border: 1px solid;
                }
                .alert-warning {
                    background: var(--vscode-inputValidation-warningBackground);
                    border-color: var(--vscode-inputValidation-warningBorder);
                    color: var(--vscode-inputValidation-warningForeground);
                }
                .alert-success {
                    background: var(--vscode-inputValidation-infoBackground);
                    border-color: var(--vscode-inputValidation-infoBorder);
                    color: var(--vscode-inputValidation-infoForeground);
                }
                .hidden {
                    display: none;
                }
            </style>
        </head>
        <body>
            <h1>🧠 Context-Keeper 状态面板</h1>
            
            <!-- 用户配置区域 -->
            <div class="status-card">
                <h3>
                    <span class="status-indicator ${userConfigStatus.isConfigured ? 'status-connected' : 'status-warning'}"></span>
                    👤 用户配置
                </h3>
                
                ${userConfigStatus.isConfigured ? `
                    <p><strong>用户ID:</strong> ${userConfigStatus.userId}</p>
                    <p><strong>配置时间:</strong> ${userConfigStatus.firstUsed ? new Date(userConfigStatus.firstUsed).toLocaleString() : '未知'}</p>
                    <p><strong>配置文件:</strong> ${userConfigStatus.configPath || '未知'}</p>
                    <button class="btn btn-secondary" onclick="sendMessage('editUserConfig')">编辑配置</button>
                    <button class="btn btn-secondary" onclick="sendMessage('resetUserConfig')">重置配置</button>
                ` : `
                    <div class="alert alert-warning">
                        <strong>⚠️ 用户信息未配置</strong><br>
                        请配置用户信息以使用 Context-Keeper 的完整功能。
                    </div>
                    
                    <div class="user-config-form">
                        <div class="form-group">
                            <label for="userId">用户ID</label>
                            <input type="text" id="userId" placeholder="格式: user_xxxxxxxx (至少8位字符)">
                            <small style="color: var(--vscode-descriptionForeground); margin-top: 5px; display: block;">
                                格式要求：user_ + 至少8位字母数字字符 (例如: user_abc12345)
                            </small>
                        </div>
                        
                        <!-- 提示信息区域 -->
                        <div id="userIdMessage" class="alert hidden" style="margin: 10px 0;"></div>
                        
                        <div style="margin-top: 10px;">
                            <button class="btn btn-primary" onclick="saveUserConfig()">保存配置</button>
                            <button class="btn btn-secondary" onclick="generateNewUserId()">生成新ID</button>
                        </div>
                    </div>
                `}
            </div>
            
            <!-- 连接状态区域 -->
            <div class="status-card">
                <h3>
                    <span class="status-indicator ${healthCheck.success ? 'status-connected' : 'status-disconnected'}"></span>
                    连接状态: ${healthCheck.success ? '已连接' : '未连接'}
                </h3>
                <p>服务器: ${config.serverConnection.serverURL}</p>
                <p>用户ID: ${config.userSettings.userId || '未设置'}</p>
                
                <button class="btn" onclick="sendMessage('testConnection')">测试连接</button>
                <button class="btn" onclick="sendMessage('openSettings')">打开设置</button>
            </div>

            <!-- 数据统计区域 -->
            <div class="status-card">
                <h3>📊 数据统计</h3>
                <div class="stats-grid">
                    <div class="stat-item">
                        <div class="stat-number">${userStats.sessionCount}</div>
                        <div class="stat-label">会话数量</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-number">${userStats.historyCount}</div>
                        <div class="stat-label">历史记录</div>
                    </div>
                </div>
            </div>

            <script>
                const vscode = acquireVsCodeApi();
                
                function sendMessage(command, data = {}) {
                    vscode.postMessage({ command: command, ...data });
                }
                
                function generateNewUserId() {
                    // 生成恰好8位字符的随机ID
                    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
                    let result = '';
                    for (let i = 0; i < 8; i++) {
                        result += chars.charAt(Math.floor(Math.random() * chars.length));
                    }
                    const userId = 'user_' + result;
                    document.getElementById('userId').value = userId;
                }
                
                function showMessage(message, type) {
                    type = type || 'warning';
                    var messageDiv = document.getElementById('userIdMessage');
                    messageDiv.textContent = message;
                    messageDiv.className = 'alert alert-' + type;
                    messageDiv.classList.remove('hidden');
                    
                    if (type === 'success') {
                        setTimeout(function() {
                            messageDiv.classList.add('hidden');
                        }, 3000);
                    }
                }
                
                function hideMessage() {
                    var messageDiv = document.getElementById('userIdMessage');
                    messageDiv.classList.add('hidden');
                }
                
                function validateUserId(userId) {
                    if (!userId) {
                        return { valid: false, message: '请输入用户ID或点击"生成新ID"按钮' };
                    }
                    
                    if (userId.indexOf('user_') !== 0) {
                        return { valid: false, message: '用户ID必须以 "user_" 开头' };
                    }
                    
                    var suffix = userId.substring(5);
                    if (suffix.length < 8) {
                        return { valid: false, message: '用户ID至少需要8位字符，当前只有' + suffix.length + '位' };
                    }
                    
                    var userIdRegex = /^user_[a-zA-Z0-9]{8,}$/;
                    if (!userIdRegex.test(userId)) {
                        return { valid: false, message: '用户ID只能包含字母和数字字符' };
                    }
                    
                    return { valid: true, message: '用户ID格式正确' };
                }
                
                function saveUserConfig() {
                    var userId = document.getElementById('userId').value.trim();
                    var validation = validateUserId(userId);
                    
                    if (!validation.valid) {
                        showMessage(validation.message, 'warning');
                        return;
                    }
                    
                    hideMessage();
                    sendMessage('saveUserConfig', { userId: userId });
                }
                
                // 添加输入框实时验证
                document.addEventListener('DOMContentLoaded', function() {
                    var userIdInput = document.getElementById('userId');
                    if (userIdInput) {
                        userIdInput.addEventListener('input', function() {
                            var userId = this.value.trim();
                            if (userId.length > 0) {
                                var validation = validateUserId(userId);
                                if (validation.valid) {
                                    showMessage('✅ ' + validation.message, 'success');
                                } else if (userId.length > 5) {
                                    showMessage(validation.message, 'warning');
                                } else {
                                    hideMessage();
                                }
                            } else {
                                hideMessage();
                            }
                        });
                    }
                });
            </script>
        </body>
        </html>
        `;
    }

    async getUserDataStats() {
        try {
            const userDir = path.join(os.homedir(), '.context-keeper', 'users');
            const users = await fs.readdir(userDir).catch(() => []);
            
            let sessionCount = 0;
            let historyCount = 0;
            
            for (const userId of users) {
                const sessionsDir = path.join(userDir, userId, 'sessions');
                const historiesDir = path.join(userDir, userId, 'histories');
                
                const sessions = await fs.readdir(sessionsDir).catch(() => []);
                const histories = await fs.readdir(historiesDir).catch(() => []);
                
                sessionCount += sessions.length;
                historyCount += histories.length;
            }
            
            return {
                userCount: users.length,
                sessionCount,
                historyCount
            };
        } catch (error) {
            return {
                userCount: 0,
                sessionCount: 0,
                historyCount: 0
            };
        }
    }

    async getUserConfigStatus() {
        try {
            // 🔧 修复：使用与saveUserConfigToDisk相同的路径
            const userConfigDir = path.join(os.homedir(), 'Library', 'Application Support', 'context-keeper');
            const configPath = path.join(userConfigDir, 'user-config.json');
            
            // 检查配置文件是否存在
            const configExists = await fs.access(configPath).then(() => true).catch(() => false);
            
            if (!configExists) {
                return {
                    isConfigured: false,
                    userId: null,
                    configPath: null,
                    firstUsed: null
                };
            }
            
            // 读取配置文件
            const configContent = await fs.readFile(configPath, 'utf8');
            const config = JSON.parse(configContent);
            
            return {
                isConfigured: true,
                userId: config.userId,
                configPath: configPath,
                firstUsed: config.firstUsed
            };
        } catch (error) {
            this.outputChannel.appendLine(`[用户配置] 获取状态失败: ${error.message}`);
            return {
                isConfigured: false,
                userId: null,
                configPath: null,
                firstUsed: null
            };
        }
    }

    async saveUserConfigToDisk(userId) {
        try {
            // 🔧 修复：使用与getUserConfigStatus相同的路径
            const userConfigDir = path.join(os.homedir(), 'Library', 'Application Support', 'context-keeper');
            const configPath = path.join(userConfigDir, 'user-config.json');
            
            // 确保目录存在
            await fs.mkdir(userConfigDir, { recursive: true });
            
            // 创建用户配置
            const userConfig = {
                userId: userId,
                firstUsed: new Date().toISOString(),
                version: "1.0.0"
            };
            
            // 写入配置文件
            await fs.writeFile(configPath, JSON.stringify(userConfig, null, 2), 'utf8');
            
            this.outputChannel.appendLine(`[用户配置] 已保存用户配置: ${userId}`);
            this.outputChannel.appendLine(`[用户配置] 配置文件路径: ${configPath}`);
            
            return {
                success: true,
                message: '用户配置已保存成功',
                configPath: configPath
            };
        } catch (error) {
            this.outputChannel.appendLine(`[用户配置] 保存失败: ${error.message}`);
            return {
                success: false,
                message: `保存配置失败: ${error.message}`
            };
        }
    }

    async handleSaveUserConfig(userId) {
        try {
            // 验证用户ID格式
            if (!userId || !userId.trim()) {
                vscode.window.showErrorMessage('请输入有效的用户ID');
                return;
            }
            
            userId = userId.trim();
            
            // 验证用户ID格式
            const userIdRegex = /^user_[a-zA-Z0-9]{8,}$/;
            if (!userIdRegex.test(userId)) {
                vscode.window.showErrorMessage('用户ID格式不正确，应为: user_xxxxxxxx (至少8位字符)');
                return;
            }
            
            // 🔥 新增：先请求服务端进行唯一性校验和存储
            const existingConfig = await this.getUserConfigStatus();
            const isNewUser = !existingConfig.isConfigured || existingConfig.userId !== userId;
            
            // 构造用户信息
            const userInfo = {
                userId: userId,
                firstUsed: new Date().toISOString(),
                lastActive: new Date().toISOString(),
                deviceInfo: {
                    platform: process.platform,
                    nodeVersion: process.version,
                    vscodeVersion: vscode.version
                },
                metadata: {
                    version: '2.0.0',
                    source: 'vscode-extension'
                }
            };
            
            // 尝试调用服务端API
            try {
                this.outputChannel.appendLine(`[用户配置] 正在向服务端${isNewUser ? '新增' : '更新'}用户: ${userId}`);
                
                let response;
                if (isNewUser) {
                    // 新增用户：调用 POST /api/users
                    response = await fetch('http://localhost:8088/api/users', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(userInfo)
                    });
                } else {
                    // 更新用户：调用 PUT /api/users/:userId
                    response = await fetch(`http://localhost:8088/api/users/${userId}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            firstUsed: userInfo.firstUsed,
                            lastActive: userInfo.lastActive,
                            deviceInfo: userInfo.deviceInfo,
                            metadata: userInfo.metadata
                        })
                    });
                }
                
                const result = await response.json();
                
                if (!response.ok) {
                    if (response.status === 409) {
                        // 用户ID已存在
                        vscode.window.showErrorMessage(`用户ID "${userId}" 已存在，请更换其他用户ID`);
                        this.outputChannel.appendLine(`[用户配置] 用户ID重复: ${userId}`);
                        return;
                    } else {
                        throw new Error(result.message || '服务端处理失败');
                    }
                }
                
                this.outputChannel.appendLine(`[用户配置] 服务端验证成功: ${result.message}`);
                vscode.window.showInformationMessage(`🌐 服务端验证成功: ${result.message}`);
                
            } catch (serverError) {
                this.outputChannel.appendLine(`[用户配置] 服务端请求失败，将仅保存到本地: ${serverError.message}`);
                vscode.window.showWarningMessage(`服务端连接失败，仅保存到本地。错误: ${serverError.message}`);
            }
            
            // 🔥 无论服务端是否成功，都继续执行原有的本地存储逻辑
            const result = await this.saveUserConfigToDisk(userId);
            
            if (result.success) {
                vscode.window.showInformationMessage(`✅ 用户配置已保存: ${userId}`);
                this.outputChannel.appendLine(`[用户配置] 用户配置保存成功: ${userId}`);
                
                // 尝试重新连接WebSocket（如果已启用）
                if (this.config.webSocket?.autoConnect) {
                    await this.checkUserIdAndConnect();
                }
            } else {
                vscode.window.showErrorMessage(`❌ 保存失败: ${result.message}`);
            }
        } catch (error) {
            vscode.window.showErrorMessage(`保存用户配置失败: ${error.message}`);
            this.outputChannel.appendLine(`[用户配置] 保存异常: ${error.message}`);
        }
    }

    async handleEditUserConfig() {
        try {
            const userConfigStatus = await this.getUserConfigStatus();
            
            if (!userConfigStatus.isConfigured) {
                vscode.window.showWarningMessage('尚未配置用户信息，请先配置用户ID');
                return;
            }
            
            const newUserId = await vscode.window.showInputBox({
                prompt: '请输入新的用户ID',
                value: userConfigStatus.userId,
                validateInput: (value) => {
                    if (!value || !value.trim()) {
                        return '用户ID不能为空';
                    }
                    const userIdRegex = /^user_[a-zA-Z0-9]{8,}$/;
                    if (!userIdRegex.test(value.trim())) {
                        return '用户ID格式不正确，应为: user_xxxxxxxx (至少8位字符)';
                    }
                    return null;
                }
            });
            
            if (newUserId && newUserId.trim() !== userConfigStatus.userId) {
                await this.handleSaveUserConfig(newUserId.trim());
            }
        } catch (error) {
            vscode.window.showErrorMessage(`编辑用户配置失败: ${error.message}`);
        }
    }

    async handleResetUserConfig() {
        try {
            const confirm = await vscode.window.showWarningMessage(
                '确定要重置用户配置吗？这将清除当前的用户ID设置。',
                { modal: true },
                '确定',
                '取消'
            );
            
            if (confirm === '确定') {
                // 🔧 修复：使用与getUserConfigStatus相同的路径
                const userConfigDir = path.join(os.homedir(), 'Library', 'Application Support', 'context-keeper');
                const configPath = path.join(userConfigDir, 'user-config.json');
                
                // 删除配置文件
                await fs.unlink(configPath).catch(() => {
                    // 忽略文件不存在的错误
                });
                
                this.outputChannel.appendLine(`[用户配置] 已重置用户配置`);
                vscode.window.showInformationMessage('✅ 用户配置已重置');
                
                // 断开WebSocket连接
                if (this.ws) {
                    this.ws.close();
                    this.ws = null;
                }
                
                // 清除相关状态
                this.currentUserId = null;
                this.currentConnectionId = null;
                this.currentSessionId = null;
                
                // 更新状态栏
                this.updateStatusBar('未配置', 'gray');
            }
        } catch (error) {
            vscode.window.showErrorMessage(`重置用户配置失败: ${error.message}`);
        }
    }

    async testConnection() {
        if (!this.client) {
            vscode.window.showWarningMessage('MCP客户端未初始化');
            return;
        }
        
        const result = await this.client.healthCheck();
        
        if (result.success) {
            vscode.window.showInformationMessage('✅ 连接测试成功');
            this.updateStatusBar('已连接', 'lightgreen');
        } else {
            vscode.window.showErrorMessage(`❌ 连接测试失败: ${result.message}`);
            this.updateStatusBar('连接失败', 'red');
        }
    }

    async resetConfig() {
        const result = await vscode.window.showWarningMessage(
            '确定要重置配置为默认值吗？',
            '确定',
            '取消'
        );
        
        if (result === '确定') {
            try {
                await fs.unlink(this.configPath);
                await this.initializeClient();
                vscode.window.showInformationMessage('配置已重置');
            } catch (error) {
                vscode.window.showErrorMessage(`重置失败: ${error.message}`);
            }
        }
    }

    // 🔥 新增：处理WebSocket回调结果
    handleCallbackResult(message) {
        const { callbackId, result } = message;
        const callback = this.pendingCallbacks.get(callbackId);
        
        if (callback) {
            callback(result);
            this.pendingCallbacks.delete(callbackId);
            this.outputChannel.appendLine(`[WebSocket] ✅ 回调结果已处理: ${callbackId}`);
        } else {
            this.outputChannel.appendLine(`[WebSocket] ⚠️ 未找到回调: ${callbackId}`);
        }
    }

    // 🔥 新增：显示日志面板
    async showLogsPanel() {
        this.outputChannel.show();
    }

    // 🔥 新增：各种占位符方法（后续可扩展）
    async exportConfig() {
        try {
            const config = await this.loadConfig();
            const configString = JSON.stringify(config, null, 2);
            
            const result = await vscode.window.showSaveDialog({
                defaultUri: vscode.Uri.file('context-keeper-config.json'),
                filters: {
                    'JSON Files': ['json'],
                    'All Files': ['*']
                }
            });
            
            if (result) {
                await fs.writeFile(result.fsPath, configString);
                vscode.window.showInformationMessage('配置已导出');
            }
        } catch (error) {
            vscode.window.showErrorMessage(`导出失败: ${error.message}`);
        }
    }

    async importConfig() {
        try {
            const result = await vscode.window.showOpenDialog({
                canSelectFiles: true,
                canSelectFolders: false,
                canSelectMany: false,
                filters: {
                    'JSON Files': ['json'],
                    'All Files': ['*']
                }
            });
            
            if (result && result[0]) {
                const configContent = await fs.readFile(result[0].fsPath, 'utf-8');
                const config = JSON.parse(configContent);
                
                await this.saveConfig(config);
                vscode.window.showInformationMessage('配置已导入');
            }
        } catch (error) {
            vscode.window.showErrorMessage(`导入失败: ${error.message}`);
        }
    }

    async clearUserData() {
        const result = await vscode.window.showWarningMessage(
            '确定要清除所有用户数据吗？此操作不可恢复！',
            '确定',
            '取消'
        );
        
        if (result === '确定') {
            try {
                const userDir = path.join(os.homedir(), '.context-keeper', 'users');
                await fs.rmdir(userDir, { recursive: true }).catch(() => {});
                vscode.window.showInformationMessage('用户数据已清除');
            } catch (error) {
                vscode.window.showErrorMessage(`清除失败: ${error.message}`);
            }
        }
    }

    async backupUserData() {
        try {
            const result = await vscode.window.showSaveDialog({
                defaultUri: vscode.Uri.file('context-keeper-backup.json'),
                filters: {
                    'JSON Files': ['json'],
                    'All Files': ['*']
                }
            });
            
            if (result) {
                const stats = await this.getUserDataStats();
                const backupData = {
                    timestamp: new Date().toISOString(),
                    stats,
                    note: '数据备份'
                };
                
                await fs.writeFile(result.fsPath, JSON.stringify(backupData, null, 2));
                vscode.window.showInformationMessage('数据已备份');
            }
        } catch (error) {
            vscode.window.showErrorMessage(`备份失败: ${error.message}`);
        }
    }

    async showUserDataPanel() {
        const stats = await this.getUserDataStats();
        
        const panel = vscode.window.createWebviewPanel(
            'context-keeper-userdata',
            'Context-Keeper 用户数据',
            vscode.ViewColumn.One,
            { enableScripts: true }
        );

        panel.webview.html = `
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <title>用户数据</title>
            <style>
                body { 
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                    padding: 20px;
                    background-color: var(--vscode-editor-background);
                    color: var(--vscode-editor-foreground);
                }
                .stat-item { margin: 10px 0; }
                .stat-value { font-weight: bold; color: var(--vscode-textLink-foreground); }
            </style>
        </head>
        <body>
            <h1>📊 用户数据统计</h1>
            <div class="stat-item">用户数量: <span class="stat-value">${stats.userCount}</span></div>
            <div class="stat-item">会话数量: <span class="stat-value">${stats.sessionCount}</span></div>
            <div class="stat-item">历史记录: <span class="stat-value">${stats.historyCount}</span></div>
        </body>
        </html>
        `;
    }

    async startService() {
        // 重新初始化WebSocket连接
        await this.initializeWebSocketIntegration();
        vscode.window.showInformationMessage('Context-Keeper服务已启动');
    }

    async stopService() {
        // 🔥 修复：使用新的统一停止方法
        this.stopWebSocketServices();
        this.updateStatusBar('已停止', 'gray');
        vscode.window.showInformationMessage('Context-Keeper服务已停止');
    }

    async restartService() {
        // 🔥 修复：重启时先彻底停止所有服务
        this.stopWebSocketServices();
        await new Promise(resolve => setTimeout(resolve, 1000)); // 等待1秒确保清理完成
        await this.initializeWebSocketIntegration();
        vscode.window.showInformationMessage('Context-Keeper服务已重启');
    }

    // 🔥 从extension.js移植：进程清理功能
    cleanup() {
        this.outputChannel.appendLine('[扩展清理] 🧹 正在清理资源...');
        
        // 🔥 修复：使用统一的WebSocket清理方法
        this.stopWebSocketServices();
        
        // 清理VSCode资源
        if (this.statusBarItem) {
            this.statusBarItem.dispose();
        }
        if (this.outputChannel) {
            this.outputChannel.appendLine('[扩展清理] ✅ 资源清理完成');
            this.outputChannel.dispose();
        }
        
        this.outputChannel.appendLine('[扩展清理] 👋 扩展已安全关闭');
    }

    setupFileWatchers() {
        const fileWatcher = vscode.workspace.onDidSaveTextDocument(async (document) => {
            if (this.client && this.isActive) {
                this.outputChannel.appendLine(`文件已保存: ${document.fileName}`);
                
                // 🔥 新增：自动文件关联
                if (this.config?.automationFeatures?.autoAssociate) {
                    // 这里可以添加自动文件关联逻辑
                    this.outputChannel.appendLine(`[自动关联] 检测到文件变更: ${document.fileName}`);
                }
            }
        });
        
        this.context.subscriptions.push(fileWatcher);
    }

    async autoStart() {
        this.outputChannel.appendLine('✅ 自动功能已启用');
        this.outputChannel.appendLine(`🔌 WebSocket状态: ${this.wsConnectionState}`);
        this.outputChannel.appendLine(`🔧 MCP客户端状态: ${this.isActive ? '已连接' : '未连接'}`);
    }

    dispose() {
        // 🔥 修复：使用统一的WebSocket清理方法
        this.stopWebSocketServices();
        
        if (this.statusBarItem) {
            this.statusBarItem.dispose();
        }
        if (this.outputChannel) {
            this.outputChannel.dispose();
        }
    }


    
    // 🔥 修复：删除错误的MD5哈希函数，统一使用generateWorkspaceHash
    // 这个函数已被generateWorkspaceHash替代，使用SHA256与服务端保持一致

    // 🔥 新增：与服务端完全一致的工作空间哈希生成方法
    generateWorkspaceHash(workspacePath) {
        if (!workspacePath || workspacePath === "") {
            return "default";
        }
        
        // 🔥 关键：使用Node.js crypto模块生成SHA256哈希，与Go服务端保持一致
        const crypto = require('crypto');
        // 标准化路径处理（对应Go的filepath.Clean）
        const path = require('path');
        const cleanPath = path.resolve(workspacePath);
        
        // 生成SHA256哈希并取前16个字符，与服务端GenerateWorkspaceHash逻辑一致
        const hash = crypto.createHash('sha256').update(cleanPath).digest('hex');
        return hash.substring(0, 16);
    }
}

// 扩展激活函数
function activate(context) {
    console.log('Context-Keeper扩展正在激活...');
    const extension = new ContextKeeperExtension(context);
    return extension;
}

// 扩展停用函数
function deactivate() {
    console.log('Context-Keeper扩展正在停用...');
}

module.exports = {
    activate,
    deactivate,
    ContextKeeperExtension
}; 