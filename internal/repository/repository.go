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
	// CreateTask 创建任务
	CreateTask(ctx context.Context, task *models.Task) error

	// GetTask 获取任务
	GetTask(ctx context.Context, id string) (*models.Task, error)

	// ListTasks 列出任务
	ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int, error)

	// UpdateTask 更新任务
	UpdateTask(ctx context.Context, task *models.Task) error

	// DeleteTask 删除任务
	DeleteTask(ctx context.Context, id string) error
}

// UserRepository 用户存储接口
type UserRepository interface {
	// CreateUser 创建用户
	CreateUser(ctx context.Context, user *models.User) error
	// GetUser 获取用户
	GetUser(ctx context.Context, id int64) (*models.User, error)
	// GetByUsername 按用户名获取用户
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	// ListUsers 列出用户
	ListUsers(ctx context.Context) ([]*models.User, error)
	// UpdateUser 更新用户
	UpdateUser(ctx context.Context, user *models.User) error
	// DeleteUser 删除用户
	DeleteUser(ctx context.Context, id int64) error

	// APIToken 相关
	CreateToken(ctx context.Context, token *models.APIToken) error
	GetToken(ctx context.Context, tokenStr string) (*models.APIToken, error)
	ListTokens(ctx context.Context, userID int64) ([]*models.APIToken, error)
	DeleteToken(ctx context.Context, id int64) error
}
