# Consul 配置文件
# 用于 MockS3 微服务架构的服务发现和配置管理

# 数据中心配置
datacenter = "mocks3-dc1"
data_dir = "/consul/data"
log_level = "INFO"
node_name = "consul-server-1"

# 服务器配置
server = true
bootstrap_expect = 1

# 网络配置
bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"

# UI 配置
ui_config {
  enabled = true
}

# 连接配置
ports {
  grpc = 8502
  grpc_tls = 8503
}

# 性能配置
performance {
  raft_multiplier = 1
}

# 日志配置
log_rotate_duration = "24h"
log_rotate_max_files = 7

# 健康检查配置
check_update_interval = "5m"

# ACL 配置 (生产环境应启用)
acl = {
  enabled = false
  default_policy = "allow"
  enable_token_persistence = true
}

# 加密配置 (生产环境应启用)
encrypt_verify_incoming = false
encrypt_verify_outgoing = false

# 服务定义
services {
  name = "consul"
  tags = ["infrastructure", "service-discovery"]
  port = 8500
  check {
    http = "http://localhost:8500/v1/status/leader"
    interval = "10s"
  }
}

# KV 存储配置
# 预定义一些配置项
config_entries {
  bootstrap = [
    {
      kind = "service-defaults"
      name = "metadata-service"
      protocol = "http"
      connect {
        sidecar_service = {}
      }
    },
    {
      kind = "service-defaults"
      name = "storage-service"
      protocol = "http"
      connect {
        sidecar_service = {}
      }
    },
    {
      kind = "service-defaults"
      name = "queue-service"
      protocol = "http"
      connect {
        sidecar_service = {}
      }
    },
    {
      kind = "service-defaults"
      name = "third-party-service"
      protocol = "http"
      connect {
        sidecar_service = {}
      }
    },
    {
      kind = "service-defaults"
      name = "mock-error-service"
      protocol = "http"
      connect {
        sidecar_service = {}
      }
    }
  ]
}