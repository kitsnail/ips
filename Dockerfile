# 多阶段构建
# 阶段1: 构建
FROM golang:1.23-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /build/bin/apiserver ./cmd/apiserver

# 阶段2: 运行
FROM alpine:latest

# 安装 ca-certificates，用于 HTTPS 请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 ips && \
    adduser -D -u 1000 -G ips ips

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/bin/apiserver /app/apiserver

# 复制 Web UI 静态文件
COPY --from=builder /build/web /app/web

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
