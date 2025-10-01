import * as vscode from 'vscode';

interface ContextKeeperConfig {
    serverUrl: string;
    websocketUrl: string;
    autoCapture: boolean;
    captureInterval: number;
    websocketAutoConnect: boolean;
    websocketReconnectAttempts: number;
}

class ContextKeeperExtension {
    private context: vscode.ExtensionContext;
    private config: ContextKeeperConfig;
    private currentSessionId: string | null = null;
    private captureTimer: NodeJS.Timeout | null = null;
    private statusBarItem: vscode.StatusBarItem;
    
    constructor(context: vscode.ExtensionContext) {
        this.context = context;
        this.config = this.loadConfig();
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.text = "$(brain) Context Keeper";
        this.statusBarItem.command = 'contextKeeper.showDashboard';
        this.statusBarItem.show();
    }

    private loadConfig(): ContextKeeperConfig {
        const config = vscode.workspace.getConfiguration('contextKeeper');
        const serverUrl = config.get('serverURL', 'http://localhost:8088');
        const websocketUrl = config.get('websocketURL', '') || serverUrl.replace('http', 'ws') + '/ws';
        
        return {
            serverUrl,
            websocketUrl,
            autoCapture: config.get('autoCapture', true),
            captureInterval: config.get('captureInterval', 30),
            websocketAutoConnect: config.get('websocketAutoConnect', true),
            websocketReconnectAttempts: config.get('websocketReconnectAttempts', 5)
        };
    }

    async activate() {
        // 创建会话
        await this.ensureSession();
        
        // 注册所有钩子和命令
        this.registerCommands();
        this.registerEventListeners();
        this.registerLanguageFeatures();
        
        // 开始自动捕获
        if (this.config.autoCapture) {
            this.startAutoCapture();
        }

        console.log('Context Keeper 扩展已激活');
    }

    private registerCommands() {
        // 存储重要记忆
        const storeMemoryCommand = vscode.commands.registerCommand('contextKeeper.storeMemory', async () => {
            const editor = vscode.window.activeTextEditor;
            if (editor && editor.selection) {
                const selectedText = editor.document.getText(editor.selection);
                if (selectedText.trim()) {
                    await this.storeImportantMemory(selectedText);
                    vscode.window.showInformationMessage('重要记忆已存储！');
                }
            }
        });

        // 查询上下文
        const queryContextCommand = vscode.commands.registerCommand('contextKeeper.queryContext', async () => {
            const query = await vscode.window.showInputBox({
                prompt: '输入查询内容...',
                placeHolder: '例如：如何实现用户认证？'
            });
            
            if (query) {
                const result = await this.queryContext(query);
                this.showContextResult(result);
            }
        });

        // 显示控制面板
        const showDashboardCommand = vscode.commands.registerCommand('contextKeeper.showDashboard', () => {
            this.showDashboard();
        });

        // 配置设置
        const configureCommand = vscode.commands.registerCommand('contextKeeper.configureSettings', () => {
            vscode.commands.executeCommand('workbench.action.openSettings', 'contextKeeper');
        });

        this.context.subscriptions.push(
            storeMemoryCommand,
            queryContextCommand, 
            showDashboardCommand,
            configureCommand
        );
    }

    private registerEventListeners() {
        // 文件保存时自动记录
        const onSave = vscode.workspace.onDidSaveTextDocument(async (document) => {
            if (document.uri.scheme === 'file') {
                await this.recordFileEdit(document);
                this.updateStatusBar('📝 已保存');
            }
        });

        // 文件打开时自动关联
        const onOpen = vscode.workspace.onDidOpenTextDocument(async (document) => {
            if (document.uri.scheme === 'file') {
                await this.associateFile(document.uri.fsPath);
                this.updateStatusBar('🔗 已关联');
            }
        });

        // 编辑器切换时更新上下文
        const onEditorChange = vscode.window.onDidChangeActiveTextEditor(async (editor) => {
            if (editor) {
                await this.captureCurrentContext();
                this.updateStatusBar('🧠 已更新');
            }
        });

        // 工作区文件夹变化
        const onWorkspaceChange = vscode.workspace.onDidChangeWorkspaceFolders(async () => {
            await this.ensureSession(); // 重新创建会话
        });

        // 配置变化
        const onConfigChange = vscode.workspace.onDidChangeConfiguration((event) => {
            if (event.affectsConfiguration('contextKeeper')) {
                this.config = this.loadConfig();
                this.restartAutoCapture();
            }
        });

        this.context.subscriptions.push(
            onSave,
            onOpen, 
            onEditorChange,
            onWorkspaceChange,
            onConfigChange
        );
    }

    private registerLanguageFeatures() {
        // 智能代码补全 - 基于历史上下文
        const completionProvider = vscode.languages.registerCompletionItemProvider(
            { scheme: 'file' },
            {
                async provideCompletionItems(document, position) {
                    // 查询相关的历史代码模式
                    const context = document.getText(new vscode.Range(
                        new vscode.Position(Math.max(0, position.line - 5), 0),
                        position
                    ));
                    
                    // 这里可以调用Context Keeper API获取智能建议
                    return [];
                }
            },
            '.'
        );

        // 悬停提示 - 显示相关记忆
        const hoverProvider = vscode.languages.registerHoverProvider(
            { scheme: 'file' },
            {
                async provideHover(document, position) {
                    const word = document.getWordRangeAtPosition(position);
                    if (word) {
                        const text = document.getText(word);
                        // 查询相关记忆
                        const memories = await this.searchMemories(text);
                        if (memories.length > 0) {
                            const contents = memories.map(m => `💡 ${m.content}`);
                            return new vscode.Hover(contents);
                        }
                    }
                }
            }
        );

        // 代码操作 - 基于上下文的建议
        const codeActionProvider = vscode.languages.registerCodeActionsProvider(
            { scheme: 'file' },
            {
                async provideCodeActions(document, range) {
                    const actions = [];
                    
                    // 添加"存储为重要记忆"操作
                    const storeAction = new vscode.CodeAction(
                        '存储为重要记忆',
                        vscode.CodeActionKind.QuickFix
                    );
                    storeAction.command = {
                        command: 'contextKeeper.storeMemory',
                        title: '存储记忆'
                    };
                    actions.push(storeAction);
                    
                    return actions;
                }
            }
        );

        this.context.subscriptions.push(
            completionProvider,
            hoverProvider,
            codeActionProvider
        );
    }

    private async ensureSession(): Promise<void> {
        try {
            const response = await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'session_management',
                params: {
                    action: 'create',
                    metadata: {
                        source: 'vscode-extension',
                        workspace: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
                    }
                }
            });
            
            this.currentSessionId = response.data.sessionId;
            this.updateStatusBar('✅ 已连接');
        } catch (error) {
            console.error('创建会话失败:', error);
            this.updateStatusBar('❌ 连接失败');
        }
    }

    private async storeImportantMemory(content: string): Promise<void> {
        if (!this.currentSessionId) return;
        
        try {
            await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'memorize_context',
                params: {
                    sessionId: this.currentSessionId,
                    content,
                    priority: 'P1',
                    metadata: {
                        source: 'vscode-selection',
                        timestamp: Date.now()
                    }
                }
            });
        } catch (error) {
            console.error('存储记忆失败:', error);
        }
    }

    private async associateFile(filePath: string): Promise<void> {
        if (!this.currentSessionId) return;
        
        try {
            await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'associate_file',
                params: {
                    sessionId: this.currentSessionId,
                    filePath
                }
            });
        } catch (error) {
            console.error('文件关联失败:', error);
        }
    }

    private async recordFileEdit(document: vscode.TextDocument): Promise<void> {
        if (!this.currentSessionId) return;
        
        try {
            // 这里可以计算文件差异
            const content = document.getText();
            
            await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'record_edit',
                params: {
                    sessionId: this.currentSessionId,
                    filePath: document.uri.fsPath,
                    diff: `Modified: ${document.fileName}`, // 简化的差异信息
                    metadata: {
                        language: document.languageId,
                        lineCount: document.lineCount
                    }
                }
            });
        } catch (error) {
            console.error('记录编辑失败:', error);
        }
    }

    private async captureCurrentContext(): Promise<void> {
        if (!this.currentSessionId) return;
        
        try {
            await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'programming_context',
                params: {
                    sessionId: this.currentSessionId,
                    query: 'vscode-context-capture'
                }
            });
        } catch (error) {
            console.error('捕获上下文失败:', error);
        }
    }

    private async queryContext(query: string): Promise<any> {
        if (!this.currentSessionId) return null;
        
        try {
            const response = await axios.post(`${this.config.serverUrl}/mcp`, {
                method: 'retrieve_context',
                params: {
                    sessionId: this.currentSessionId,
                    query
                }
            });
            return response.data;
        } catch (error) {
            console.error('查询上下文失败:', error);
            return null;
        }
    }

    private async searchMemories(query: string): Promise<any[]> {
        // 搜索相关记忆的简化实现
        return [];
    }

    private showContextResult(result: any): void {
        if (result) {
            const panel = vscode.window.createWebviewPanel(
                'contextResult',
                'Context Query Result',
                vscode.ViewColumn.Beside,
                { enableScripts: true }
            );
            
            panel.webview.html = this.getWebviewContent(result);
        }
    }

    private showDashboard(): void {
        const panel = vscode.window.createWebviewPanel(
            'contextDashboard',
            'Context Keeper Dashboard',
            vscode.ViewColumn.One,
            { enableScripts: true }
        );
        
        panel.webview.html = this.getDashboardContent();
        
        // 处理来自webview的消息
        panel.webview.onDidReceiveMessage(async (message) => {
            switch (message.command) {
                case 'queryContext':
                    const result = await this.queryContext(message.query);
                    panel.webview.postMessage({ type: 'queryResult', data: result });
                    break;
            }
        });
    }

    private getWebviewContent(result: any): string {
        return `<!DOCTYPE html>
        <html>
        <head>
            <title>Context Result</title>
        </head>
        <body>
            <h1>查询结果</h1>
            <pre>${JSON.stringify(result, null, 2)}</pre>
        </body>
        </html>`;
    }

    private getDashboardContent(): string {
        return `<!DOCTYPE html>
        <html>
        <head>
            <title>Context Keeper Dashboard</title>
            <style>
                body { font-family: Arial, sans-serif; padding: 20px; }
                .query-box { margin: 20px 0; }
                input[type="text"] { width: 300px; padding: 8px; }
                button { padding: 8px 16px; margin-left: 8px; }
            </style>
        </head>
        <body>
            <h1>🧠 Context Keeper Dashboard</h1>
            
            <div class="query-box">
                <h3>查询编程上下文</h3>
                <input type="text" id="queryInput" placeholder="输入你的问题..." />
                <button onclick="queryContext()">查询</button>
            </div>
            
            <div id="results"></div>
            
            <script>
                const vscode = acquireVsCodeApi();
                
                function queryContext() {
                    const query = document.getElementById('queryInput').value;
                    if (query.trim()) {
                        vscode.postMessage({
                            command: 'queryContext',
                            query: query
                        });
                    }
                }
                
                window.addEventListener('message', event => {
                    const message = event.data;
                    if (message.type === 'queryResult') {
                        document.getElementById('results').innerHTML = 
                            '<h3>结果:</h3><pre>' + JSON.stringify(message.data, null, 2) + '</pre>';
                    }
                });
            </script>
        </body>
        </html>`;
    }

    private startAutoCapture(): void {
        if (this.captureTimer) {
            clearInterval(this.captureTimer);
        }
        
        this.captureTimer = setInterval(() => {
            this.captureCurrentContext();
        }, this.config.captureInterval * 1000);
    }

    private restartAutoCapture(): void {
        if (this.config.autoCapture) {
            this.startAutoCapture();
        } else if (this.captureTimer) {
            clearInterval(this.captureTimer);
            this.captureTimer = null;
        }
    }

    private updateStatusBar(text: string): void {
        this.statusBarItem.text = `$(brain) ${text}`;
        // 3秒后恢复默认文本
        setTimeout(() => {
            this.statusBarItem.text = "$(brain) Context Keeper";
        }, 3000);
    }

    deactivate(): void {
        if (this.captureTimer) {
            clearInterval(this.captureTimer);
        }
        this.statusBarItem.dispose();
    }
}

let extension: ContextKeeperExtension;

export function activate(context: vscode.ExtensionContext) {
    extension = new ContextKeeperExtension(context);
    extension.activate();
}

export function deactivate() {
    if (extension) {
        extension.deactivate();
    }
} 