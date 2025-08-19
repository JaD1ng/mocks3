package repository

import (
	"context"
	"fmt"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"sync"
)

// StorageManager 存储管理器实现
type StorageManager struct {
	nodes             []interfaces.StorageNode
	thirdPartyService interfaces.ThirdPartyService
	mu                sync.RWMutex
}

// NewStorageManager 创建存储管理器
func NewStorageManager() *StorageManager {
	return &StorageManager{
		nodes: make([]interfaces.StorageNode, 0),
	}
}

// AddNode 添加存储节点
func (sm *StorageManager) AddNode(node interfaces.StorageNode) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.nodes = append(sm.nodes, node)
}

// WriteToAllNodes 写入所有存储节点
func (sm *StorageManager) WriteToAllNodes(ctx context.Context, object *models.Object) error {
	sm.mu.RLock()
	nodes := make([]interfaces.StorageNode, len(sm.nodes))
	copy(nodes, sm.nodes)
	sm.mu.RUnlock()

	if len(nodes) == 0 {
		return fmt.Errorf("no storage nodes available")
	}

	var lastErr error
	successCount := 0

	// 顺序写入每个节点
	for i, node := range nodes {
		// 为每个节点创建对象副本，避免并发修改
		objectCopy := *object
		if objectCopy.Headers == nil {
			objectCopy.Headers = make(map[string]string)
		}
		if objectCopy.Tags == nil {
			objectCopy.Tags = make(map[string]string)
		}

		err := node.Write(ctx, &objectCopy)
		if err != nil {
			lastErr = err
			fmt.Printf("Failed to write to node %s: %v\n", node.GetNodeID(), err)
			continue
		}

		successCount++
		fmt.Printf("Step %d: Successfully wrote to node %s\n", i+1, node.GetNodeID())

		// 更新原对象的元数据（使用第一个成功的节点的结果）
		if successCount == 1 {
			object.ID = objectCopy.ID
			object.MD5Hash = objectCopy.MD5Hash
			object.ETag = objectCopy.ETag
		}
	}

	// 如果至少有一个节点写入成功，则认为写入成功
	if successCount == 0 {
		return fmt.Errorf("failed to write to any storage node, last error: %v", lastErr)
	}

	if successCount < len(nodes) {
		fmt.Printf("Warning: Only %d out of %d nodes wrote successfully\n", successCount, len(nodes))
	}

	return nil
}

// ReadFromBestNode 从最佳节点读取（优先stg1）
func (sm *StorageManager) ReadFromBestNode(ctx context.Context, bucket, key string) (*models.Object, error) {
	sm.mu.RLock()
	nodes := make([]interfaces.StorageNode, len(sm.nodes))
	copy(nodes, sm.nodes)
	sm.mu.RUnlock()

	// 首先尝试从stg1读取
	for _, node := range nodes {
		if node.GetNodeID() == "stg1" {
			obj, err := node.Read(ctx, bucket, key)
			if err == nil {
				fmt.Printf("Successfully read from stg1: %s/%s\n", bucket, key)
				return obj, nil
			}
			fmt.Printf("Failed to read from stg1: %v\n", err)
			break
		}
	}

	// 如果stg1失败，尝试其他节点
	for _, node := range nodes {
		if node.GetNodeID() != "stg1" {
			obj, err := node.Read(ctx, bucket, key)
			if err == nil {
				fmt.Printf("Successfully read from node %s: %s/%s\n", node.GetNodeID(), bucket, key)
				return obj, nil
			}
			fmt.Printf("Failed to read from node %s: %v\n", node.GetNodeID(), err)
		}
	}

	// 如果所有节点都失败，尝试第三方服务
	if sm.thirdPartyService != nil {
		fmt.Printf("Attempting to fetch from third party service: %s/%s\n", bucket, key)
		obj, err := sm.thirdPartyService.GetObject(ctx, bucket, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get object from third party service: %w", err)
		}

		fmt.Printf("Successfully fetched from third party service: %s/%s\n", bucket, key)

		// 异步写入到所有节点（缓存第三方数据）
		go func() {
			if writeErr := sm.WriteToAllNodes(context.Background(), obj); writeErr != nil {
				fmt.Printf("Warning: failed to cache third party data: %v\n", writeErr)
			}
		}()

		return obj, nil
	}

	return nil, fmt.Errorf("failed to read file %s/%s from any storage node", bucket, key)
}

// DeleteFromAllNodes 从所有节点删除
func (sm *StorageManager) DeleteFromAllNodes(ctx context.Context, bucket, key string) error {
	sm.mu.RLock()
	nodes := make([]interfaces.StorageNode, len(sm.nodes))
	copy(nodes, sm.nodes)
	sm.mu.RUnlock()

	var errors []error
	successCount := 0

	for _, node := range nodes {
		if err := node.Delete(ctx, bucket, key); err != nil {
			errors = append(errors, fmt.Errorf("node %s: %w", node.GetNodeID(), err))
			fmt.Printf("Warning: failed to delete from node %s: %v\n", node.GetNodeID(), err)
		} else {
			successCount++
			fmt.Printf("Successfully deleted from node %s: %s/%s\n", node.GetNodeID(), bucket, key)
		}
	}

	// 如果所有节点都失败，返回错误
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("failed to delete from all nodes: %v", errors)
	}

	return nil
}

// GetHealthyNodes 获取健康的节点
func (sm *StorageManager) GetHealthyNodes() []interfaces.StorageNode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var healthyNodes []interfaces.StorageNode
	for _, node := range sm.nodes {
		if node.IsHealthy(context.Background()) {
			healthyNodes = append(healthyNodes, node)
		}
	}

	return healthyNodes
}

// GetAllNodes 获取所有节点
func (sm *StorageManager) GetAllNodes() []interfaces.StorageNode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	nodes := make([]interfaces.StorageNode, len(sm.nodes))
	copy(nodes, sm.nodes)
	return nodes
}

// GetNodeByID 根据ID获取节点
func (sm *StorageManager) GetNodeByID(nodeID string) interfaces.StorageNode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, node := range sm.nodes {
		if node.GetNodeID() == nodeID {
			return node
		}
	}
	return nil
}

// GetNodeIDs 获取所有节点ID
func (sm *StorageManager) GetNodeIDs() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]string, len(sm.nodes))
	for i, node := range sm.nodes {
		ids[i] = node.GetNodeID()
	}
	return ids
}

// SetThirdPartyService 设置第三方服务
func (sm *StorageManager) SetThirdPartyService(service interfaces.ThirdPartyService) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.thirdPartyService = service
}

// GetThirdPartyService 获取第三方服务
func (sm *StorageManager) GetThirdPartyService() interfaces.ThirdPartyService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.thirdPartyService
}

// ListObjects 列出对象（从第一个健康节点）
func (sm *StorageManager) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.ObjectInfo, error) {
	healthyNodes := sm.GetHealthyNodes()
	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy storage nodes available")
	}

	// 使用第一个健康节点进行列表操作
	firstNode := healthyNodes[0]

	// 类型断言检查节点是否支持列表操作
	if lister, ok := firstNode.(*FileStorageNode); ok {
		return lister.ListObjects(ctx, bucket, prefix, limit)
	}

	return nil, fmt.Errorf("storage node does not support list operations")
}

// GetStats 获取所有节点的统计信息
func (sm *StorageManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	sm.mu.RLock()
	nodes := make([]interfaces.StorageNode, len(sm.nodes))
	copy(nodes, sm.nodes)
	sm.mu.RUnlock()

	stats := make(map[string]interface{})
	nodeStats := make([]map[string]interface{}, 0, len(nodes))

	var totalSize int64
	var totalFiles int64
	healthyCount := 0

	for _, node := range nodes {
		if fileNode, ok := node.(*FileStorageNode); ok {
			nodeStat, err := fileNode.GetStats(ctx)
			if err != nil {
				nodeStat = map[string]interface{}{
					"node_id": node.GetNodeID(),
					"error":   err.Error(),
					"healthy": false,
				}
			} else {
				if size, ok := nodeStat["total_size"].(int64); ok {
					totalSize += size
				}
				if files, ok := nodeStat["total_files"].(int64); ok {
					totalFiles += files
				}
				if healthy, ok := nodeStat["healthy"].(bool); ok && healthy {
					healthyCount++
				}
			}
			nodeStats = append(nodeStats, nodeStat)
		}
	}

	stats["total_nodes"] = len(nodes)
	stats["healthy_nodes"] = healthyCount
	stats["total_size"] = totalSize
	stats["total_files"] = totalFiles
	stats["nodes"] = nodeStats

	return stats, nil
}
