package repository

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"mocks3/services/third-party/internal/config"
	"sync"
	"time"
)

// DataSourceRepository 数据源仓库
type DataSourceRepository struct {
	dataSources map[string]*models.DataSource
	mu          sync.RWMutex
}

// NewDataSourceRepository 创建数据源仓库
func NewDataSourceRepository(configs []config.DataSourceConfig) *DataSourceRepository {
	repo := &DataSourceRepository{
		dataSources: make(map[string]*models.DataSource),
	}
	
	// 初始化数据源
	for i, cfg := range configs {
		if cfg.Enabled {
			dataSource := &models.DataSource{
				ID:       fmt.Sprintf("ds-%d", i+1),
				Name:     cfg.Name,
				Type:     cfg.Type,
				Enabled:  cfg.Enabled,
				Priority: cfg.Priority,
				Config: map[string]string{
					"endpoint":    cfg.Endpoint,
					"access_key":  cfg.AccessKey,
					"secret_key":  cfg.SecretKey,
					"region":      cfg.Region,
					"bucket_name": cfg.BucketName,
					"timeout":     fmt.Sprintf("%d", cfg.Timeout),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			// 添加额外配置
			for k, v := range cfg.ExtraConfig {
				dataSource.Config[k] = v
			}
			
			repo.dataSources[dataSource.ID] = dataSource
		}
	}
	
	return repo
}

// GetAll 获取所有数据源
func (r *DataSourceRepository) GetAll(ctx context.Context) ([]models.DataSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var sources []models.DataSource
	for _, ds := range r.dataSources {
		if ds.Enabled {
			sources = append(sources, *ds)
		}
	}
	
	return sources, nil
}

// GetByID 根据ID获取数据源
func (r *DataSourceRepository) GetByID(ctx context.Context, id string) (*models.DataSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	ds, exists := r.dataSources[id]
	if !exists {
		return nil, fmt.Errorf("data source not found: %s", id)
	}
	
	if !ds.Enabled {
		return nil, fmt.Errorf("data source disabled: %s", id)
	}
	
	return ds, nil
}

// GetByPriority 按优先级获取数据源
func (r *DataSourceRepository) GetByPriority(ctx context.Context) ([]*models.DataSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var sources []*models.DataSource
	for _, ds := range r.dataSources {
		if ds.Enabled {
			sources = append(sources, ds)
		}
	}
	
	// 按优先级排序（优先级值越小越优先）
	for i := 0; i < len(sources)-1; i++ {
		for j := i + 1; j < len(sources); j++ {
			if sources[i].Priority > sources[j].Priority {
				sources[i], sources[j] = sources[j], sources[i]
			}
		}
	}
	
	return sources, nil
}

// Add 添加数据源
func (r *DataSourceRepository) Add(ctx context.Context, dataSource *models.DataSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if dataSource.ID == "" {
		dataSource.ID = fmt.Sprintf("ds-%d", time.Now().Unix())
	}
	
	dataSource.CreatedAt = time.Now()
	dataSource.UpdatedAt = time.Now()
	
	r.dataSources[dataSource.ID] = dataSource
	
	return nil
}

// Update 更新数据源
func (r *DataSourceRepository) Update(ctx context.Context, dataSource *models.DataSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.dataSources[dataSource.ID]; !exists {
		return fmt.Errorf("data source not found: %s", dataSource.ID)
	}
	
	dataSource.UpdatedAt = time.Now()
	r.dataSources[dataSource.ID] = dataSource
	
	return nil
}

// Delete 删除数据源
func (r *DataSourceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.dataSources[id]; !exists {
		return fmt.Errorf("data source not found: %s", id)
	}
	
	delete(r.dataSources, id)
	
	return nil
}

// Enable 启用数据源
func (r *DataSourceRepository) Enable(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	ds, exists := r.dataSources[id]
	if !exists {
		return fmt.Errorf("data source not found: %s", id)
	}
	
	ds.Enabled = true
	ds.UpdatedAt = time.Now()
	
	return nil
}

// Disable 禁用数据源
func (r *DataSourceRepository) Disable(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	ds, exists := r.dataSources[id]
	if !exists {
		return fmt.Errorf("data source not found: %s", id)
	}
	
	ds.Enabled = false
	ds.UpdatedAt = time.Now()
	
	return nil
}