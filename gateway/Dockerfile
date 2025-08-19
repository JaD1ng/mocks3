# MockS3 Nginx Gateway Dockerfile
FROM nginx:1.25-alpine

# 安装必要的包，使用重试和镜像
RUN apk update --no-cache || \
    (echo "https://mirror.alpinelinux.org/alpine/v3.19/main" > /etc/apk/repositories && \
     echo "https://mirror.alpinelinux.org/alpine/v3.19/community" >> /etc/apk/repositories && \
     apk update --no-cache) && \
    apk add --no-cache \
    curl \
    openssl \
    openssl-dev \
    pcre-dev \
    zlib-dev \
    nginx-mod-http-lua

# 创建必要的目录
RUN mkdir -p /var/cache/nginx/levels \
    && mkdir -p /var/log/nginx \
    && mkdir -p /etc/nginx/conf.d \
    && mkdir -p /etc/nginx/ssl

# 复制配置文件
COPY nginx.conf /etc/nginx/nginx.conf

# 创建日志文件
RUN touch /var/log/nginx/access.log \
    && touch /var/log/nginx/error.log \
    && touch /var/log/nginx/s3_access.log \
    && touch /var/log/nginx/s3_error.log \
    && touch /var/log/nginx/s3_upload.log \
    && touch /var/log/nginx/s3_download.log \
    && touch /var/log/nginx/s3_delete.log \
    && touch /var/log/nginx/s3_list.log \
    && touch /var/log/nginx/admin_access.log \
    && touch /var/log/nginx/admin_error.log

# 设置权限
RUN chown -R nginx:nginx /var/cache/nginx \
    && chown -R nginx:nginx /var/log/nginx \
    && chmod 755 /var/cache/nginx \
    && chmod 644 /var/log/nginx/*

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 暴露端口
EXPOSE 8080 8081

# 启动命令
CMD ["nginx", "-g", "daemon off;"]