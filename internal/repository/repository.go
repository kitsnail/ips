package repository

import (
	"context"
	"errors"

	"github.com/kitsnail/ips/pkg/models"
)

var (
	// ErrTaskNotFound 任务不存在
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskAlreadyExists 任务已存在
	ErrTaskAlreadyExists = errors.New("task already exists")
)

// TaskRepository 任务存储接口
type TaskRepository interface {
	// Create 创建任务
	Create(ctx context.Context, task *models.Task) error

	// Get 获取任务
	Get(ctx context.Context, id string) (*models.Task, error)

	// List 列出任务
	List(ctx context.Context, filter models.TaskFilter) ([]*models.Task, int, error)

	// Update 更新任务
	Update(ctx context.Context, task *models.Task) error

	// Delete 删除任务
	Delete(ctx context.Context, id string) error
}
