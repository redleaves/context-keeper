#!/usr/bin/env node

/**
 * Context-Keeper扩展安装验证脚本
 */

const ContextKeeperMCPClient = require('./mcp-client.js');

async function verifyInstallation() {
    console.log('🔍 开始验证Context-Keeper扩展安装...\n');
    
    try {
        // 1. 验证MCP客户端
        console.log('📡 测试1: MCP客户端连接');
        const client = new ContextKeeperMCPClient({
            serverURL: 'http://localhost:8088'
        });
        
        const healthCheck = await client.healthCheck();
        if (healthCheck.success) {
            console.log('✅ MCP客户端连接成功');
        } else {
            console.log('❌ MCP客户端连接失败:', healthCheck.message);
            return false;
        }
        
        // 2. 创建测试会话
        console.log('\n🆕 测试2: 创建测试会话');
        const sessionResult = await client.createSession({ type: 'installation-verification' });
        
        if (sessionResult.success) {
            console.log('✅ 会话创建成功:', sessionResult.data.sessionId);
        } else {
            console.log('❌ 会话创建失败:', sessionResult.error);
            return false;
        }
        
        // 3. 测试存储功能
        console.log('\n💾 测试3: 存储对话');
        const storeResult = await client.storeConversation(
            sessionResult.data.sessionId,
            [
                { role: 'user', content: '安装验证测试消息' },
                { role: 'assistant', content: '扩展安装验证成功！' }
            ]
        );
        
        if (storeResult.success) {
            console.log('✅ 对话存储成功');
        } else {
            console.log('❌ 对话存储失败:', storeResult.error);
            return false;
        }
        
        // 4. 测试检索功能
        console.log('\n🔍 测试4: 上下文检索');
        const retrieveResult = await client.retrieveContext(
            sessionResult.data.sessionId,
            '安装验证'
        );
        
        if (retrieveResult.success) {
            console.log('✅ 上下文检索成功');
        } else {
            console.log('❌ 上下文检索失败:', retrieveResult.error);
            return false;
        }
        
        console.log('\n🎉 所有验证测试通过！扩展已准备就绪！');
        
        console.log('\n📋 下一步操作指南:');
        console.log('1. 在Cursor中按 Cmd+Shift+P 打开命令面板');
        console.log('2. 搜索 "Context-Keeper" 查看可用命令');
        console.log('3. 查看右下角状态栏的 "🧠 CK:" 状态指示器');
        console.log('4. 右键点击状态栏图标查看状态面板');
        
        return true;
        
    } catch (error) {
        console.log('❌ 验证过程出错:', error.message);
        return false;
    }
}

// 运行验证
verifyInstallation().then(success => {
    process.exit(success ? 0 : 1);
}).catch(error => {
    console.error('验证脚本执行失败:', error);
    process.exit(1);
}); 