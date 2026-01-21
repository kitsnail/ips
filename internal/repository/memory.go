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

// CreateTask 创建任务
func (r *MemoryRepository) CreateTask(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}

	r.tasks[task.ID] = task
	return nil
}

// GetTask 获取任务
func (r *MemoryRepository) GetTask(ctx context.Context, id string) (*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return task, nil
}

// ListTasks 列出任务
func (r *MemoryRepository) ListTasks(ctx context.Context) ([]*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allTasks []*models.Task
	for _, task := range r.tasks {
		allTasks = append(allTasks, task)
	}

	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})

	return allTasks, nil
}

// UpdateTask 更新任务
func (r *MemoryRepository) UpdateTask(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}

	r.tasks[task.ID] = task
	return nil
}

// DeleteTask 删除任务
func (r *MemoryRepository) DeleteTask(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}

// Dummy User methods to satisfy interface

func (r *MemoryRepository) CreateUser(ctx context.Context, user *models.User) error { return nil }
func (r *MemoryRepository) GetUser(ctx context.Context, id int64) (*models.User, error) {
	return nil, nil
}
func (r *MemoryRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (r *MemoryRepository) ListUsers(ctx context.Context) ([]*models.User, error)         { return nil, nil }
func (r *MemoryRepository) UpdateUser(ctx context.Context, user *models.User) error       { return nil }
func (r *MemoryRepository) DeleteUser(ctx context.Context, id int64) error                { return nil }
func (r *MemoryRepository) CreateToken(ctx context.Context, token *models.APIToken) error { return nil }
func (r *MemoryRepository) GetToken(ctx context.Context, tokenStr string) (*models.APIToken, error) {
	return nil, nil
}
func (r *MemoryRepository) ListTokens(ctx context.Context, userID int64) ([]*models.APIToken, error) {
	return nil, nil
}
func (r *MemoryRepository) DeleteToken(ctx context.Context, id int64) error { return nil }
