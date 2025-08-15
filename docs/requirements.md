# 技术栈
- 容器化: Docker
- 服务通信: Http
- 服务发现: Consul
- API网关: Nginx
- 数据库: PostgreSQL + Redis缓存metadata
- 配置管理: Consul KV
- 消息队列: Redis streams（和缓存都是Redis，但要分成两个实例）
- 监控: Prometheus
- 日志: ElasticSearch
- 链路: ElasticSearch
- 观测：Opentelemetry观测监控、日志、链路

# 服务分类
- API网关
- S3 API服务（业务编排）
- 元数据管理服务（使用读写分离）
- 分布式存储服务
- 异步任务处理服务
- 错误注入服务
- Admin API服务

注：配置管理功能通过 Consul KV 提供，无需独立服务