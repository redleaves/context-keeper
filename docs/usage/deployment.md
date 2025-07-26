# Context-Keeper 部署指南

本文档提供了在不同环境中部署 Context-Keeper 服务的详细说明。

## 目录

- [本地开发环境部署](#本地开发环境部署)
- [生产环境部署](#生产环境部署)
  - [直接部署](#直接部署)
  - [Docker部署](#docker部署)
  - [云服务部署](#云服务部署)
- [高可用部署](#高可用部署)
- [安全配置](#安全配置)
- [性能优化](#性能优化)

## 本地开发环境部署

### 前提条件

- Go 1.20+
- Git
- 向量数据库账号（推荐使用阿里云向量数据库、Pinecone或Milvus）
- 文本嵌入API（如OpenAI Embeddings API或阿里云文本嵌入服务）

### 步骤

1. **克隆代码仓库**

   ```bash
   git clone https://github.com/your-org/context-keeper.git
   cd context-keeper
   ```

2. **安装依赖**

   ```bash
   go mod tidy
   ```

3. **准备配置文件**

   ```bash
   cp config-template.json config.json
   ```

   修改 `config.json` 填入您的API密钥和服务端点：

   ```json
   {
     "vector_db": {
       "url": "YOUR_VECTOR_DB_ENDPOINT",
       "api_key": "YOUR_API_KEY"
     },
     "embedding": {
       "api_url": "YOUR_EMBEDDING_API_ENDPOINT",
       "api_key": "YOUR_API_KEY"
     }
   }
   ```

4. **运行服务**

   ```bash
   go run cmd/server/main.go --config config.json
   ```

   或者编译后运行：

   ```bash
   go build -o context-keeper ./cmd/server/
   ./context-keeper --config config.json
   ```

5. **验证服务**

   ```bash
   curl http://localhost:8081/health
   # 应返回: {"status":"healthy"}
   ```

## 生产环境部署

### 直接部署

1. **编译服务**

   ```bash
   # 在开发机上编译
   GOOS=linux GOARCH=amd64 go build -o context-keeper ./cmd/server/
   
   # 或在目标服务器上编译
   go build -o context-keeper ./cmd/server/
   ```

2. **创建服务用户**

   ```bash
   sudo useradd -r -s /bin/false context-keeper
   ```

3. **准备目录结构**

   ```bash
   sudo mkdir -p /opt/context-keeper/bin
   sudo mkdir -p /opt/context-keeper/config
   sudo mkdir -p /opt/context-keeper/data
   sudo mkdir -p /opt/context-keeper/logs
   ```

4. **复制文件**

   ```bash
   sudo cp context-keeper /opt/context-keeper/bin/
   sudo cp config.json /opt/context-keeper/config/
   sudo chown -R context-keeper:context-keeper /opt/context-keeper
   ```

5. **创建Systemd服务**

   创建文件 `/etc/systemd/system/context-keeper.service`：

   ```ini
   [Unit]
   Description=Context-Keeper - Programming Context Management Service
   After=network.target
   
   [Service]
   Type=simple
   User=context-keeper
   Group=context-keeper
   WorkingDirectory=/opt/context-keeper
   ExecStart=/opt/context-keeper/bin/context-keeper --config /opt/context-keeper/config/config.json
   Restart=on-failure
   RestartSec=5s
   
   [Install]
   WantedBy=multi-user.target
   ```

6. **启动服务**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable context-keeper
   sudo systemctl start context-keeper
   sudo systemctl status context-keeper
   ```

### Docker部署

1. **创建Dockerfile**

   ```Dockerfile
   FROM golang:1.20-alpine AS builder
   
   WORKDIR /app
   COPY . .
   RUN go mod download
   RUN CGO_ENABLED=0 GOOS=linux go build -o context-keeper ./cmd/server/
   
   FROM alpine:latest
   
   WORKDIR /app
   COPY --from=builder /app/context-keeper .
   COPY config-template.json /app/config/config.json
   
   # 创建数据和日志目录
   RUN mkdir -p /app/data /app/logs
   
   # 设置环境变量
   ENV CONFIG_PATH=/app/config/config.json
   
   EXPOSE 8081
   
   CMD ["./context-keeper", "--config", "/app/config/config.json"]
   ```

2. **构建镜像**

   ```bash
   docker build -t context-keeper:latest .
   ```

3. **准备配置文件**

   创建 `config.json` 并填入您的API密钥和端点。

4. **运行容器**

   ```bash
   docker run -d --name context-keeper \
     -p 8081:8081 \
     -v $(pwd)/config.json:/app/config/config.json \
     -v $(pwd)/data:/app/data \
     -v $(pwd)/logs:/app/logs \
     context-keeper:latest
   ```

5. **使用Docker Compose**

   创建 `docker-compose.yml`：

   ```yaml
   version: '3'
   
   services:
     context-keeper:
       image: context-keeper:latest
       ports:
         - "8081:8081"
       volumes:
         - ./config.json:/app/config/config.json
         - ./data:/app/data
         - ./logs:/app/logs
       restart: unless-stopped
   ```

   启动服务：

   ```bash
   docker-compose up -d
   ```

### 云服务部署

#### 阿里云ECS

1. 按照[直接部署](#直接部署)或[Docker部署](#docker部署)步骤操作
2. 配置安全组开放8081端口
3. 可选：配置SLB实现负载均衡

#### AWS EC2

1. 按照[直接部署](#直接部署)或[Docker部署](#docker部署)步骤操作
2. 配置安全组开放8081端口
3. 可选：配置ELB实现负载均衡

#### 云原生部署 (Kubernetes)

1. **创建Kubernetes配置文件**

   `context-keeper-deployment.yaml`:

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: context-keeper
     labels:
       app: context-keeper
   spec:
     replicas: 2
     selector:
       matchLabels:
         app: context-keeper
     template:
       metadata:
         labels:
           app: context-keeper
       spec:
         containers:
         - name: context-keeper
           image: context-keeper:latest
           ports:
           - containerPort: 8081
           volumeMounts:
           - name: config-volume
             mountPath: /app/config
           - name: data-volume
             mountPath: /app/data
           resources:
             limits:
               cpu: "1"
               memory: "1Gi"
             requests:
               cpu: "0.5"
               memory: "512Mi"
         volumes:
         - name: config-volume
           configMap:
             name: context-keeper-config
         - name: data-volume
           persistentVolumeClaim:
             claimName: context-keeper-pvc
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: context-keeper
   spec:
     selector:
       app: context-keeper
     ports:
     - port: 8081
       targetPort: 8081
     type: ClusterIP
   ```

2. **创建ConfigMap**

   ```bash
   kubectl create configmap context-keeper-config --from-file=config.json
   ```

3. **创建PersistentVolumeClaim**

   ```yaml
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: context-keeper-pvc
   spec:
     accessModes:
       - ReadWriteOnce
     resources:
       requests:
         storage: 10Gi
   ```

4. **部署应用**

   ```bash
   kubectl apply -f context-keeper-deployment.yaml
   ```

## 高可用部署

### 多实例部署

部署多个实例并使用负载均衡：

1. **修改配置，使用共享存储**
   - 确保不同实例间的数据同步
   - 可以使用对象存储或网络文件系统

2. **负载均衡**
   - Nginx负载均衡
   - 云服务负载均衡服务

### 数据持久化策略

- **本地数据**：使用持久化存储卷
- **向量数据库**：使用托管服务，确保备份
- **会话数据**：使用Redis管理会话状态

## 安全配置

### API认证

修改 `config.json` 添加认证：

```json
{
  "auth": {
    "enabled": true,
    "type": "api_key",
    "api_keys": ["your-secret-api-key"]
  }
}
```

### HTTPS配置

1. **获取证书**（如使用Let's Encrypt）
2. **配置HTTPS**：

   ```json
   {
     "https": {
       "enabled": true,
       "cert_file": "/path/to/cert.pem",
       "key_file": "/path/to/key.pem"
     }
   }
   ```

### 网络安全

1. **防火墙配置**：只开放必要端口
2. **反向代理**：使用Nginx作为安全层
3. **限流**：配置请求速率限制

## 性能优化

### 资源分配

- **内存分配**：根据工作负载调整Go的GOMAXPROCS和内存分配
- **CPU使用**：对于高负载场景，配置足够的CPU资源

### 缓存策略

在 `config.json` 中配置缓存：

```json
{
  "cache": {
    "enabled": true,
    "type": "memory",  // 或 "redis"
    "ttl": 3600,       // 缓存有效期（秒）
    "size": 1000       // 内存缓存项目数量上限
  }
}
```

### 监控和调优

1. **指标收集**：集成Prometheus监控
2. **日志分析**：使用ELK或类似工具分析日志
3. **性能分析**：使用pprof对Go应用进行性能分析

---

如遇到部署问题，请参考[故障排除指南](troubleshooting.md)或提交Issue。 