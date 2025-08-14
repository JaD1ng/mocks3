FROM golang:1.24-alpine AS builder

# 安装必要工具
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制shared模块
COPY ./shared ./shared

# 复制 go.mod 和 go.sum
COPY ./services/metadata/go.mod ./services/metadata/go.sum ./

# 下载依赖
RUN go mod download

# 复制源码
COPY ./services/metadata .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

# 运行阶段
FROM alpine:latest

# 安装证书
RUN apk --no-cache add ca-certificates

# 创建工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 暴露端口
EXPOSE 8081

# 运行服务
CMD ["./main"]