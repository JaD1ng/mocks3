package nodes

import (
	"context"
	"fmt"
	"github.com/mocks3/shared/logger"
)

// StorageNode 存储节点接口
type StorageNode interface {
	Write(obj *FileObject) error
	Read(key string) (*FileObject, error)
	Delete(key string) error
	GetNodeID() string
	GetStatus() *NodeStatus
	GetCapacity() int64
	GetUsed() int64
}

// ThirdPartyService 第三方服务接口
type ThirdPartyService interface {
	GetObject(key string) (*FileObject, error)
	GetName() string
}

// StorageManager 存储管理器
type StorageManager struct {
	nodes             []StorageNode
	thirdPartyService ThirdPartyService
	logger            logger.Logger
}

// NodeStatus 节点状态
type NodeStatus struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Capacity int64  `json:"capacity"`
	Used     int64  `json:"used"`
	Health   string `json:"health"`
}

// NewStorageManager 创建存储管理器
func NewStorageManager() *StorageManager {
	return &StorageManager{
		nodes:  make([]StorageNode, 0),
		logger: logger.DefaultLogger,
	}
}

// NewStorageManagerWithLogger 创建带自定义logger的存储管理器
func NewStorageManagerWithLogger(l logger.Logger) *StorageManager {
	return &StorageManager{
		nodes:  make([]StorageNode, 0),
		logger: l,
	}
}

// AddNode 添加存储节点
func (sm *StorageManager) AddNode(node StorageNode) {
	sm.nodes = append(sm.nodes, node)
}

// SetThirdPartyService 设置第三方服务
func (sm *StorageManager) SetThirdPartyService(service ThirdPartyService) {
	sm.thirdPartyService = service
}

// WriteToAllNodes 写入所有存储节点
func (sm *StorageManager) WriteToAllNodes(obj *FileObject) error {
	if len(sm.nodes) == 0 {
		return fmt.Errorf("no storage nodes available")
	}

	var lastErr error
	successCount := 0

	// 尝试写入每个节点
	for i, node := range sm.nodes {
		err := node.Write(obj)
		if err != nil {
			lastErr = err
			sm.logger.Error(context.Background(), "Failed to write to node", err, map[string]any{
				"node_id": node.GetNodeID(),
			})
			continue
		}
		successCount++
		sm.logger.Info(context.Background(), "Successfully wrote to node", map[string]any{
			"step":    i + 1,
			"node_id": node.GetNodeID(),
		})
	}

	// 根据业务需求，这里要求所有节点都成功
	if successCount != len(sm.nodes) {
		return fmt.Errorf("write failed: only %d out of %d nodes succeeded, last error: %v",
			successCount, len(sm.nodes), lastErr)
	}

	sm.logger.Info(context.Background(), "Successfully wrote to all storage nodes", map[string]any{
		"node_count": len(sm.nodes),
	})
	return nil
}

// ReadFromStg1OrThirdParty 优先从 stg1 读取，失败则从第三方获取
func (sm *StorageManager) ReadFromStg1OrThirdParty(key string) (*FileObject, error) {
	// 首先尝试从 stg1 读取
	if len(sm.nodes) > 0 {
		stg1Node := sm.nodes[0] // 假设第一个节点是 stg1
		if stg1Node.GetNodeID() == "stg1" {
			obj, err := stg1Node.Read(key)
			if err == nil {
				sm.logger.Info(context.Background(), "Successfully read from stg1", map[string]any{
					"key": key,
				})
				return obj, nil
			}
			sm.logger.Error(context.Background(), "Failed to read from stg1", err, map[string]any{
				"key": key,
			})
		}
	}

	// 如果 stg1 读取失败，尝试从第三方服务获取
	if sm.thirdPartyService != nil {
		sm.logger.Info(context.Background(), "Attempting to fetch from third party service", map[string]any{
			"key": key,
		})
		obj, err := sm.thirdPartyService.GetObject(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get object from third party service: %v", err)
		}

		sm.logger.Info(context.Background(), "Successfully fetched from third party service", map[string]any{
			"key": key,
		})

		// 异步写入本地存储节点
		go func() {
			err := sm.WriteToAllNodes(obj)
			if err != nil {
				sm.logger.Error(context.Background(), "Failed to write fetched object to storage nodes", err, nil)
			} else {
				sm.logger.Info(context.Background(), "Successfully cached fetched object to storage nodes", map[string]any{
					"key": key,
				})
			}
		}()

		return obj, nil
	}

	return nil, fmt.Errorf("failed to read file %s from stg1 and no third party service configured", key)
}

// ReadFromAnyNode 从任意可用节点读取
func (sm *StorageManager) ReadFromAnyNode(key string) (*FileObject, error) {
	for _, node := range sm.nodes {
		obj, err := node.Read(key)
		if err == nil {
			sm.logger.Info(context.Background(), "Successfully read from node", map[string]any{
				"node_id": node.GetNodeID(),
				"key":     key,
			})
			return obj, nil
		}
		sm.logger.Error(context.Background(), "Failed to read from node", err, map[string]any{
			"node_id": node.GetNodeID(),
			"key":     key,
		})
	}

	return nil, fmt.Errorf("failed to read file %s from any storage node", key)
}

// DeleteFromAllNodes 从所有节点删除
func (sm *StorageManager) DeleteFromAllNodes(key string) error {
	var lastErr error
	successCount := 0

	for _, node := range sm.nodes {
		err := node.Delete(key)
		if err != nil {
			lastErr = err
			sm.logger.Error(context.Background(), "Failed to delete from node", err, map[string]any{
				"node_id": node.GetNodeID(),
				"key":     key,
			})
			continue
		}
		successCount++
		sm.logger.Info(context.Background(), "Successfully deleted from node", map[string]any{
			"node_id": node.GetNodeID(),
			"key":     key,
		})
	}

	// 只要有一个节点删除成功就认为删除成功
	if successCount == 0 {
		return fmt.Errorf("failed to delete file %s from any storage node, last error: %v", key, lastErr)
	}

	if successCount < len(sm.nodes) {
		sm.logger.Warn(context.Background(), "Only partial nodes deleted successfully", map[string]any{
			"success_count": successCount,
			"total_nodes":   len(sm.nodes),
		})
	}

	return nil
}

// GetNodes 获取所有存储节点信息
func (sm *StorageManager) GetNodes() []*NodeStatus {
	var nodeStatuses []*NodeStatus

	for _, node := range sm.nodes {
		status := node.GetStatus()
		nodeStatuses = append(nodeStatuses, status)
	}

	return nodeStatuses
}

// GetNodeStatus 获取指定节点状态
func (sm *StorageManager) GetNodeStatus(nodeID string) (*NodeStatus, error) {
	for _, node := range sm.nodes {
		if node.GetNodeID() == nodeID {
			return node.GetStatus(), nil
		}
	}
	return nil, fmt.Errorf("node %s not found", nodeID)
}

// GetStats 获取存储统计信息
func (sm *StorageManager) GetStats() map[string]any {
	stats := make(map[string]any)

	var totalCapacity int64 = 0
	var totalUsed int64 = 0
	var healthyNodes int = 0

	nodeStats := make([]map[string]any, 0)

	for _, node := range sm.nodes {
		status := node.GetStatus()

		totalCapacity += status.Capacity
		totalUsed += status.Used

		if status.Health == "healthy" {
			healthyNodes++
		}

		nodeStats = append(nodeStats, map[string]any{
			"id":       status.ID,
			"status":   status.Status,
			"capacity": status.Capacity,
			"used":     status.Used,
			"health":   status.Health,
		})
	}

	stats["total_nodes"] = len(sm.nodes)
	stats["healthy_nodes"] = healthyNodes
	stats["total_capacity"] = totalCapacity
	stats["total_used"] = totalUsed
	stats["usage_percent"] = float64(totalUsed) / float64(totalCapacity) * 100
	stats["nodes"] = nodeStats

	if sm.thirdPartyService != nil {
		stats["third_party_service"] = sm.thirdPartyService.GetName()
	}

	return stats
}
