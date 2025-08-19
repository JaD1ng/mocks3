# 构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go模块文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o storage-service ./services/storage/cmd/server/

# 运行阶段
FROM alpine:latest

# 安装ca证书和时区数据，使用重试和镜像
RUN apk update --no-cache || \
    (echo "https://mirror.alpinelinux.org/alpine/v3.19/main" > /etc/apk/repositories && \
     echo "https://mirror.alpinelinux.org/alpine/v3.19/community" >> /etc/apk/repositories && \
     apk update --no-cache) && \
    apk add --no-cache ca-certificates tzdata

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/storage-service .

# 创建数据目录
RUN mkdir -p /data/storage

# 设置可执行权限
RUN chmod +x ./storage-service

# 暴露端口
EXPOSE 8082

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# 运行应用
CMD ["./storage-service"]