#!/usr/bin/env node

/**
 * 处理特定的本地指令
 */

const { LocalStorageSync } = require('./sync_local_storage.js');

async function processStoreConversationInstruction() {
    const sync = new LocalStorageSync();
    
    // 这是从前面store_conversation调用返回的本地指令
    const instruction = {
        "type": "short_memory",
        "target": "~/Library/Application Support/context-keeper/users/user_1ukhbs7v/histories/session-20250625-142557.json",
        "content": [
            "[2025-06-25 14:26:01] user: 测试路径统一功能",
            "[2025-06-25 14:26:01] assistant: 正在测试系统标准路径是否工作正常"
        ],
        "options": {
            "createDir": true,
            "merge": true,
            "maxAge": 604800,
            "cleanupOld": true
        },
        "callbackId": "short_memory_session-20250625-142557_1750832761",
        "priority": "normal"
    };

    console.log('📋 处理store_conversation生成的本地指令...');
    const success = await sync.processLocalInstruction(instruction);
    
    if (success) {
        console.log('✅ 本地指令处理成功！');
    } else {
        console.log('❌ 本地指令处理失败！');
    }
    
    return success;
}

// 运行处理
processStoreConversationInstruction().then(success => {
    process.exit(success ? 0 : 1);
}).catch(error => {
    console.error('❌ 处理失败:', error);
    process.exit(1);
}); 