#!/usr/bin/env node

/**
 * Context-Keeper 本地存储同步工具
 * 处理服务器生成的本地存储指令，确保数据在本地正确存储
 */

const fs = require('fs').promises;
const path = require('path');
const os = require('os');

class LocalStorageSync {
    constructor() {
        this.homeDir = os.homedir();
        console.log('📁 本地存储同步工具启动');
        console.log('🏠 用户主目录:', this.homeDir);
    }

    /**
     * 展开路径，处理 ~/ 和系统路径
     */
    expandPath(pathTemplate) {
        if (pathTemplate.startsWith('~/Library/Application Support/')) {
            // macOS系统标准应用数据目录
            return path.join(this.homeDir, pathTemplate.slice(2));
        }
        if (pathTemplate.startsWith('~/')) {
            return path.join(this.homeDir, pathTemplate.slice(2));
        }
        return pathTemplate;
    }

    /**
     * 确保目录存在
     */
    async ensureDirectory(dirPath) {
        try {
            await fs.mkdir(dirPath, { recursive: true });
            return true;
        } catch (error) {
            console.error('❌ 创建目录失败:', dirPath, error.message);
            return false;
        }
    }

    /**
     * 检查文件是否存在
     */
    async fileExists(filePath) {
        try {
            await fs.access(filePath);
            return true;
        } catch {
            return false;
        }
    }

    /**
     * 写入JSON文件
     */
    async writeJSON(filePath, data) {
        try {
            const content = JSON.stringify(data, null, 2);
            await fs.writeFile(filePath, content, 'utf8');
            return true;
        } catch (error) {
            console.error('❌ 写入JSON文件失败:', filePath, error.message);
            return false;
        }
    }

    /**
     * 读取JSON文件
     */
    async readJSON(filePath) {
        try {
            const content = await fs.readFile(filePath, 'utf8');
            return JSON.parse(content);
        } catch (error) {
            console.error('❌ 读取JSON文件失败:', filePath, error.message);
            return null;
        }
    }

    /**
     * 处理短期记忆存储指令
     */
    async handleShortMemoryInstruction(instruction) {
        const { target, content, options } = instruction;
        const filePath = this.expandPath(target);

        console.log('💾 处理短期记忆存储:', filePath);

        // 确保目录存在
        const dir = path.dirname(filePath);
        if (!(await this.ensureDirectory(dir))) {
            return false;
        }

        let finalContent = content;

        // 如果需要合并，先读取现有数据
        if (options.merge && await this.fileExists(filePath)) {
            const existingData = await this.readJSON(filePath);
            if (Array.isArray(existingData) && Array.isArray(content)) {
                finalContent = [...existingData, ...content];
                
                // 应用最大长度限制
                if (options.maxAge && finalContent.length > 20) {
                    finalContent = finalContent.slice(-20);
                }
            }
        }

        // 写入文件
        const success = await this.writeJSON(filePath, finalContent);
        if (success) {
            console.log('✅ 短期记忆已保存:', filePath);
            console.log('📊 记录数量:', Array.isArray(finalContent) ? finalContent.length : 1);
        }

        return success;
    }

    /**
     * 处理用户配置存储指令
     */
    async handleUserConfigInstruction(instruction) {
        const { target, content, options } = instruction;
        const filePath = this.expandPath(target);

        console.log('⚙️  处理用户配置存储:', filePath);

        // 确保目录存在
        const dir = path.dirname(filePath);
        if (!(await this.ensureDirectory(dir))) {
            return false;
        }

        // 备份现有配置
        if (options.backup && await this.fileExists(filePath)) {
            const backupPath = `${filePath}.bak.${Date.now()}`;
            try {
                const existingData = await fs.readFile(filePath);
                await fs.writeFile(backupPath, existingData);
                console.log('💾 已备份现有配置:', backupPath);
            } catch (error) {
                console.error('⚠️  备份失败:', error.message);
            }
        }

        // 写入新配置
        const success = await this.writeJSON(filePath, content);
        if (success) {
            console.log('✅ 用户配置已保存:', content.userId || 'unknown');
        }

        return success;
    }

    /**
     * 处理会话存储指令
     */
    async handleSessionStoreInstruction(instruction) {
        const { target, content, options } = instruction;
        const filePath = this.expandPath(target);

        console.log('📝 处理会话存储:', filePath);

        // 确保目录存在
        const dir = path.dirname(filePath);
        if (!(await this.ensureDirectory(dir))) {
            return false;
        }

        // 写入会话数据
        const success = await this.writeJSON(filePath, content);
        if (success) {
            console.log('✅ 会话已保存:', content.id || 'unknown');
        }

        return success;
    }

    /**
     * 处理本地指令
     */
    async processLocalInstruction(instruction) {
        if (!instruction || !instruction.type) {
            console.error('❌ 无效的本地指令');
            return false;
        }

        console.log(`\n🔄 处理指令类型: ${instruction.type}`);
        
        switch (instruction.type) {
            case 'short_memory':
                return await this.handleShortMemoryInstruction(instruction);
            
            case 'user_config':
                return await this.handleUserConfigInstruction(instruction);
                
            case 'session_store':
                return await this.handleSessionStoreInstruction(instruction);
                
            default:
                console.log(`⚠️  未处理的指令类型: ${instruction.type}`);
                return false;
        }
    }

    /**
     * 从MCP工具响应中提取并处理本地指令
     */
    async processResponseWithLocalInstruction(response) {
        if (!response || !response.localInstruction) {
            console.log('ℹ️  响应中没有本地指令');
            return true;
        }

        console.log('\n📋 发现本地指令，开始处理...');
        return await this.processLocalInstruction(response.localInstruction);
    }

    /**
     * 测试功能
     */
    async testStorageSync() {
        console.log('\n🧪 开始测试本地存储同步...');

        // 测试短期记忆指令
        const testInstruction = {
            type: 'short_memory',
            target: '~/Library/Application Support/context-keeper/users/user_1ukhbs7v/histories/test-session.json',
            content: [
                '[2025-06-25 14:30:00] user: 测试消息1',
                '[2025-06-25 14:30:01] assistant: 测试回复1'
            ],
            options: {
                createDir: true,
                merge: true,
                maxAge: 604800,
                cleanupOld: true
            }
        };

        const success = await this.processLocalInstruction(testInstruction);
        
        if (success) {
            console.log('✅ 测试成功！');
            
            // 验证文件是否存在
            const filePath = this.expandPath(testInstruction.target);
            if (await this.fileExists(filePath)) {
                console.log('✅ 文件创建成功:', filePath);
                
                // 读取验证
                const data = await this.readJSON(filePath);
                if (data && Array.isArray(data)) {
                    console.log('✅ 数据验证成功，记录数:', data.length);
                    data.forEach((record, index) => {
                        console.log(`  ${index + 1}: ${record}`);
                    });
                }
            }
        } else {
            console.log('❌ 测试失败！');
        }

        return success;
    }
}

// 如果直接运行脚本，执行测试
if (require.main === module) {
    const sync = new LocalStorageSync();
    
    // 检查命令行参数
    const args = process.argv.slice(2);
    
    if (args.includes('--test')) {
        sync.testStorageSync().then(() => {
            console.log('\n🏁 测试完成');
            process.exit(0);
        }).catch(error => {
            console.error('❌ 测试失败:', error);
            process.exit(1);
        });
    } else {
        console.log('📖 使用方法:');
        console.log('  node scripts/sync_local_storage.js --test');
        console.log('');
        console.log('💡 或在代码中使用:');
        console.log('  const { LocalStorageSync } = require("./scripts/sync_local_storage.js");');
        console.log('  const sync = new LocalStorageSync();');
        console.log('  await sync.processResponseWithLocalInstruction(response);');
    }
}

module.exports = { LocalStorageSync }; 