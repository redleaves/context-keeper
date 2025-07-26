# Context-Keeper 云端部署 Dockerfile
# 支持 HTTP 和 STDIO 模式，优化用于生产环境
# 完全模拟 ./scripts/manage.sh deploy http 的部署流程
# 🔥 新增：解决云端日志查看问题，将业务日志重定向到标准输出

# ================================
# 第一阶段：构建阶段
# ================================
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    bash \
    file \
    && update-ca-certificates

# 设置 Go 环境变量（模拟build.sh的设置）
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download && go mod verify

# 复制源代码和构建脚本
COPY . .

# 构建信息（模拟build.sh的构建信息设置）
ARG VERSION=docker
ARG BUILD_TIME
ARG COMMIT_HASH

# 🔥 关键修复：使用项目的build.sh脚本进行编译，完全模拟manage.sh deploy过程
RUN chmod +x ./scripts/build/build.sh && \
    # 设置构建信息环境变量（模拟build.sh）
    export VERSION=${VERSION} && \
    export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") && \
    export COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "docker-build") && \
    # 调用项目的构建脚本，确保与manage.sh deploy完全一致
    ./scripts/build/build.sh --all

# 验证构建产物（模拟manage.sh的验证过程）
RUN ls -la ./bin/ && \
    file ./bin/context-keeper && \
    file ./bin/context-keeper-http

# ================================
# 第二阶段：运行时镜像
# ================================
FROM alpine:3.19

# 🔥 新增：安装日志处理工具
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    jq \
    bash \
    procps \
    coreutils \
    util-linux \
    && rm -rf /var/cache/apk/*

# 创建非特权用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件（使用build.sh生成的文件）
COPY --from=builder /app/bin/context-keeper /app/bin/context-keeper
COPY --from=builder /app/bin/context-keeper-http /app/bin/context-keeper-http

# 复制管理脚本和入口点脚本（支持完整的管理功能）
COPY scripts/manage.sh /app/scripts/manage.sh
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY scripts/build/ /app/scripts/build/

# 🔥 新增：复制日志监控脚本
COPY scripts/docker-log-monitor.sh /app/log-monitor.sh

# 🔥 修改：创建必要的目录，包括统一的日志目录
RUN mkdir -p /app/data /app/data/logs /app/config /app/bin && \
    # 复制二进制文件到标准位置（模拟manage.sh的文件布局）
    cp /app/bin/context-keeper /app/context-keeper-stdio && \
    cp /app/bin/context-keeper-http /app/context-keeper-http && \
    # 设置所有权
    chown -R appuser:appgroup /app

# 设置执行权限（模拟manage.sh的权限设置）
RUN chmod +x /app/bin/context-keeper /app/bin/context-keeper-http \
    /app/context-keeper-stdio /app/context-keeper-http \
    /app/scripts/manage.sh /app/docker-entrypoint.sh \
    /app/log-monitor.sh

# 🔥 修改：设置环境变量，包括日志路径重定向
ENV RUN_MODE=http \
    HTTP_SERVER_PORT=8088 \
    STORAGE_PATH=/app/data \
    LOG_LEVEL=info \
    TZ=Asia/Shanghai \
    # 模拟manage.sh的项目环境
    PROJECT_ROOT=/app \
    PID_DIR=/app/logs \
    # 🔥 新增：重定向日志到统一目录
    CONTEXT_KEEPER_LOG_DIR=/app/data/logs \
    CONTEXT_KEEPER_LOG_TO_STDOUT=true

# 健康检查（模拟manage.sh status检查）
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD if [ "$RUN_MODE" = "http" ]; then \
            curl -f http://localhost:${HTTP_SERVER_PORT}/health || exit 1; \
        else \
            pgrep -f context-keeper > /dev/null || exit 1; \
        fi

# 暴露端口
EXPOSE 8088

# 切换到非特权用户
USER appuser

# 🔥 关键修复：设置入口点支持完整的manage.sh功能
ENTRYPOINT ["/app/docker-entrypoint.sh"]

# 默认命令：模拟 ./scripts/manage.sh deploy http
CMD ["http"] 