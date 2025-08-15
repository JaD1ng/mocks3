FROM golang:1.24-alpine AS builder

# 安装必要工具
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制全局go.mod和go.sum，利用Docker层缓存
COPY go.mod go.sum ./

# 下载所有依赖
RUN go mod download

# 复制shared模块
COPY ./shared ./shared

# 复制服务源码
COPY ./services/metadata ./services/metadata

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./services/metadata/cmd/main.go

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