# 构建阶段
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json frontend/ ./

RUN npm ci

COPY frontend/ .

RUN npm run build

# 运行阶段
FROM alpine:latest

# 安装 ca-certificates，用于 HTTPS 请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 ips && \
    adduser -D -u 1000 -G ips ips

WORKDIR /app

# 从 frontend-builder 阶段复制构建产物
COPY --from=frontend-builder /app/frontend/dist /app/web/dist

# 从宿主机复制二进制文件 (必须预先 build)
COPY bin/apiserver /app/apiserver
# 从 scripts 目录复制 crictl
COPY scripts/crictl /usr/local/bin/crictl

# 复制其他静态文件（如果有）
COPY web/static /app/web/static

# 修改文件权限
RUN chown -R ips:ips /app

# 切换到非 root 用户
USER ips

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
ENTRYPOINT ["/app/apiserver"]
