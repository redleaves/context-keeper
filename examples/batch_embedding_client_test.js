/**
 * 批量Embedding API客户端测试示例
 * 演示如何使用context-keeper的批量embedding功能
 * 
 * 功能：
 * 1. 提交批量文本embedding任务
 * 2. 轮询查询任务状态
 * 3. 获取队列状态
 * 4. 处理任务结果
 */

const axios = require('axios');

class BatchEmbeddingClient {
    constructor(baseURL = 'http://localhost:8088') {
        this.baseURL = baseURL;
        this.axiosInstance = axios.create({
            baseURL: baseURL,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    /**
     * 提交批量embedding任务
     * @param {string[]} texts - 文本数组
     * @param {object} userData - 用户自定义数据
     * @param {string} textType - 文本类型：query或document
     * @returns {Promise<object>} 任务提交结果
     */
    async submitBatchEmbedding(texts, userData = {}, textType = 'document') {
        try {
            console.log(`🚀 [批量Embedding客户端] 提交批量任务，文本数量: ${texts.length}`);
            
            const response = await this.axiosInstance.post('/api/batch-embedding/submit', {
                texts: texts,
                user_data: userData,
                text_type: textType
            });

            console.log(`✅ [批量Embedding客户端] 任务提交成功: ${response.data.task_id}`);
            return response.data;
        } catch (error) {
            console.error(`❌ [批量Embedding客户端] 任务提交失败:`, error.response?.data || error.message);
            throw error;
        }
    }

    /**
     * 查询任务状态（POST方式）
     * @param {string} taskId - 任务ID
     * @returns {Promise<object>} 任务状态
     */
    async queryTaskStatus(taskId) {
        try {
            const response = await this.axiosInstance.post('/api/batch-embedding/status', {
                task_id: taskId
            });

            return response.data;
        } catch (error) {
            console.error(`❌ [批量Embedding客户端] 查询任务状态失败:`, error.response?.data || error.message);
            throw error;
        }
    }

    /**
     * 查询任务状态（GET方式）
     * @param {string} taskId - 任务ID
     * @returns {Promise<object>} 任务状态
     */
    async queryTaskStatusGET(taskId) {
        try {
            const response = await this.axiosInstance.get(`/api/batch-embedding/status/${taskId}`);
            return response.data;
        } catch (error) {
            console.error(`❌ [批量Embedding客户端] 查询任务状态失败:`, error.response?.data || error.message);
            throw error;
        }
    }

    /**
     * 获取队列状态
     * @returns {Promise<object>} 队列状态
     */
    async getQueueStatus() {
        try {
            const response = await this.axiosInstance.get('/api/batch-embedding/queue-status');
            return response.data;
        } catch (error) {
            console.error(`❌ [批量Embedding客户端] 获取队列状态失败:`, error.response?.data || error.message);
            throw error;
        }
    }

    /**
     * 检查服务健康状态
     * @returns {Promise<object>} 健康状态
     */
    async checkHealth() {
        try {
            const response = await this.axiosInstance.get('/api/batch-embedding/health');
            return response.data;
        } catch (error) {
            console.error(`❌ [批量Embedding客户端] 健康检查失败:`, error.response?.data || error.message);
            throw error;
        }
    }

    /**
     * 轮询等待任务完成
     * @param {string} taskId - 任务ID
     * @param {number} pollInterval - 轮询间隔（毫秒）
     * @param {number} maxAttempts - 最大尝试次数
     * @returns {Promise<object>} 最终任务结果
     */
    async waitForTaskCompletion(taskId, pollInterval = 5000, maxAttempts = 120) {
        console.log(`⏳ [批量Embedding客户端] 开始轮询任务: ${taskId}`);
        
        for (let attempt = 1; attempt <= maxAttempts; attempt++) {
            try {
                const result = await this.queryTaskStatus(taskId);
                
                console.log(`🔍 [批量Embedding客户端] 第${attempt}次查询 - 状态: ${result.task_status}`);
                
                if (result.task_status === 'COMPLETED') {
                    console.log(`🎉 [批量Embedding客户端] 任务完成! embedding数量: ${result.embeddings?.length || 0}`);
                    return result;
                } else if (result.task_status === 'FAILED') {
                    console.error(`💥 [批量Embedding客户端] 任务失败: ${result.error}`);
                    throw new Error(`任务执行失败: ${result.error}`);
                } else if (result.task_status === 'PENDING' || result.task_status === 'RUNNING') {
                    console.log(`⏱️ [批量Embedding客户端] 任务处理中，等待${pollInterval/1000}秒后重试...`);
                    await new Promise(resolve => setTimeout(resolve, pollInterval));
                    continue;
                } else {
                    console.warn(`⚠️ [批量Embedding客户端] 未知任务状态: ${result.task_status}`);
                }
            } catch (error) {
                console.error(`❌ [批量Embedding客户端] 第${attempt}次查询失败:`, error.message);
                if (attempt === maxAttempts) {
                    throw new Error(`轮询超时，最大尝试次数已达: ${maxAttempts}`);
                }
                await new Promise(resolve => setTimeout(resolve, pollInterval));
            }
        }
        
        throw new Error(`轮询超时，任务可能仍在处理中: ${taskId}`);
    }
}

/**
 * 运行完整的批量embedding测试
 */
async function runBatchEmbeddingTest() {
    console.log('\n🔥 ===== 批量Embedding API测试开始 =====\n');
    
    const client = new BatchEmbeddingClient();
    
    try {
        // 1. 检查服务健康状态
        console.log('📋 1. 检查服务健康状态...');
        const health = await client.checkHealth();
        console.log('✅ 服务状态:', health);
        
        // 2. 获取队列状态
        console.log('\n📋 2. 获取队列状态...');
        const queueStatus = await client.getQueueStatus();
        console.log('📊 队列状态:', queueStatus);
        
        // 3. 准备测试文本
        const testTexts = [
            "人工智能是计算机科学的一个分支",
            "机器学习是人工智能的重要组成部分",
            "深度学习通过神经网络模拟人脑的学习过程",
            "自然语言处理让计算机能够理解和生成人类语言",
            "计算机视觉让机器能够理解和分析图像",
            "强化学习通过奖励和惩罚机制训练智能体",
            "数据挖掘从大量数据中发现有价值的模式",
            "算法是解决问题的一系列明确指令",
            "云计算提供按需的计算资源服务",
            "大数据技术处理海量、多样化的数据集"
        ];
        
        // 4. 提交批量embedding任务
        console.log('\n📋 3. 提交批量embedding任务...');
        const submitResult = await client.submitBatchEmbedding(testTexts, {
            test_name: 'AI相关文本embedding测试',
            created_by: 'batch_embedding_test_client',
            timestamp: Date.now()
        }, 'document');
        
        console.log('🎯 任务提交结果:', submitResult);
        const taskId = submitResult.task_id;
        
        // 5. 轮询等待任务完成
        console.log('\n📋 4. 等待任务完成...');
        const finalResult = await client.waitForTaskCompletion(taskId, 3000, 60); // 3秒轮询，最多60次
        
        // 6. 分析结果
        console.log('\n📋 5. 分析结果...');
        if (finalResult.embeddings && finalResult.embeddings.length > 0) {
            console.log(`🎉 成功获取 ${finalResult.embeddings.length} 个embedding向量`);
            
            // 展示第一个embedding的信息
            const firstEmbedding = finalResult.embeddings[0];
            if (firstEmbedding && firstEmbedding.length > 0) {
                console.log(`📊 向量维度: ${firstEmbedding.length}`);
                console.log(`📊 第一个向量前5个元素: [${firstEmbedding.slice(0, 5).map(x => x.toFixed(4)).join(', ')}...]`);
            }
            
            // 计算向量之间的相似度（示例）
            if (finalResult.embeddings.length >= 2) {
                const similarity = calculateCosineSimilarity(finalResult.embeddings[0], finalResult.embeddings[1]);
                console.log(`📊 前两个文本的余弦相似度: ${similarity.toFixed(4)}`);
            }
        }
        
        // 7. 最终队列状态
        console.log('\n📋 6. 最终队列状态...');
        const finalQueueStatus = await client.getQueueStatus();
        console.log('📊 最终队列状态:', finalQueueStatus);
        
        console.log('\n🎉 ===== 批量Embedding API测试完成 =====\n');
        
    } catch (error) {
        console.error('\n💥 ===== 批量Embedding API测试失败 =====');
        console.error('错误详情:', error.message);
        if (error.response?.data) {
            console.error('响应数据:', error.response.data);
        }
        process.exit(1);
    }
}

/**
 * 计算两个向量的余弦相似度
 * @param {number[]} vecA - 向量A
 * @param {number[]} vecB - 向量B
 * @returns {number} 余弦相似度
 */
function calculateCosineSimilarity(vecA, vecB) {
    if (!vecA || !vecB || vecA.length !== vecB.length) {
        return 0;
    }
    
    let dotProduct = 0;
    let normA = 0;
    let normB = 0;
    
    for (let i = 0; i < vecA.length; i++) {
        dotProduct += vecA[i] * vecB[i];
        normA += vecA[i] * vecA[i];
        normB += vecB[i] * vecB[i];
    }
    
    normA = Math.sqrt(normA);
    normB = Math.sqrt(normB);
    
    if (normA === 0 || normB === 0) {
        return 0;
    }
    
    return dotProduct / (normA * normB);
}

/**
 * 运行简单的提交测试（不等待完成）
 */
async function runSimpleSubmitTest() {
    console.log('\n🔥 ===== 简单提交测试开始 =====\n');
    
    const client = new BatchEmbeddingClient();
    
    try {
        const testTexts = [
            "这是一个测试文本",
            "另一个测试文本",
            "第三个测试文本"
        ];
        
        const result = await client.submitBatchEmbedding(testTexts, {
            test_type: 'simple_submit_test'
        });
        
        console.log('✅ 任务提交成功:', result);
        console.log(`📋 任务ID: ${result.task_id}`);
        console.log('💡 可以使用以下命令查询任务状态:');
        console.log(`curl -X GET "http://localhost:8088/api/batch-embedding/status/${result.task_id}"`);
        
    } catch (error) {
        console.error('❌ 简单提交测试失败:', error.message);
    }
}

// 根据命令行参数决定运行哪个测试
const testType = process.argv[2] || 'full';

if (testType === 'simple') {
    runSimpleSubmitTest();
} else if (testType === 'full') {
    runBatchEmbeddingTest();
} else {
    console.log('用法:');
    console.log('  node batch_embedding_client_test.js full   # 运行完整测试（默认）');
    console.log('  node batch_embedding_client_test.js simple # 运行简单提交测试');
}

module.exports = {
    BatchEmbeddingClient,
    runBatchEmbeddingTest,
    runSimpleSubmitTest
}; 