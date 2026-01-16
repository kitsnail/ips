package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/kitsnail/ips/pkg/models"
)

// MemoryRepository 内存存储实现
type MemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]*models.Task
}

// NewMemoryRepository 创建内存存储
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tasks: make(map[string]*models.Task),
	}
}

// Create 创建任务
func (r *MemoryRepository) Create(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}

	r.tasks[task.ID] = task
	return nil
}

// Get 获取任务
func (r *MemoryRepository) Get(ctx context.Context, id string) (*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return task, nil
}

// List 列出任务
func (r *MemoryRepository) List(ctx context.Context, filter models.TaskFilter) ([]*models.Task, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 收集所有任务
	var allTasks []*models.Task
	for _, task := range r.tasks {
		// 状态过滤
		if filter.Status != nil && task.Status != *filter.Status {
			continue
		}
		allTasks = append(allTasks, task)
	}

	// 按创建时间倒序排序（最新的在前面）
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})

	total := len(allTasks)

	// 应用分页
	if filter.Limit > 0 {
		start := filter.Offset
		if start > total {
			return []*models.Task{}, total, nil
		}

		end := start + filter.Limit
		if end > total {
			end = total
		}

		allTasks = allTasks[start:end]
	}

	return allTasks, total, nil
}

// Update 更新任务
func (r *MemoryRepository) Update(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}

	r.tasks[task.ID] = task
	return nil
}

// Delete 删除任务
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}
