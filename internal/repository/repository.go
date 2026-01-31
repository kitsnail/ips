package repository

import (
	"context"
	"errors"
	"time"

	"github.com/kitsnail/ips/pkg/models"
)

var (
	// ErrTaskNotFound 任务不存在
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskAlreadyExists 任务已存在
	ErrTaskAlreadyExists = errors.New("task already exists")
	// ErrScheduledTaskNotFound 定时任务不存在
	ErrScheduledTaskNotFound = errors.New("scheduled task not found")
	// ErrCronExpressionInvalid Cron 表达式无效
	ErrCronExpressionInvalid = errors.New("invalid cron expression")
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
}

// APITokenRepository API Token 存储接口
type APITokenRepository interface {
	// CreateToken 创建 Token
	CreateToken(ctx context.Context, token *models.APIToken) error
	// GetToken 获取 Token
	GetToken(ctx context.Context, tokenStr string) (*models.APIToken, error)
	// ListTokens 列出用户的 Token
	ListTokens(ctx context.Context, userID int64) ([]*models.APIToken, error)
	// DeleteToken 删除 Token
	DeleteToken(ctx context.Context, id int64) error
}

// LibraryRepository 镜像库存储接口
type LibraryRepository interface {
	// SaveImage 保存镜像到库
	SaveImage(ctx context.Context, img *models.LibraryImage) error

	// ListImages 列出库中的镜像 (分页)
	ListImages(ctx context.Context, offset, limit int) ([]*models.LibraryImage, int, error)

	// DeleteImage 从库中删除镜像
	DeleteImage(ctx context.Context, id int64) error
}

// SecretRegistryRepository 私有仓库认证存储接口
type SecretRegistryRepository interface {
	// CreateSecret 创建仓库认证
	CreateSecret(ctx context.Context, secret *models.RegistrySecret) error

	// GetSecret 获取仓库认证
	GetSecret(ctx context.Context, id int64) (*models.RegistrySecret, error)

	// GetSecretByName 按名称获取仓库认证
	GetSecretByName(ctx context.Context, name string) (*models.RegistrySecret, error)

	// ListSecrets 列出所有仓库认证
	ListSecrets(ctx context.Context, offset, limit int) ([]*models.SecretListItem, int, error)

	// UpdateSecret 更新仓库认证
	UpdateSecret(ctx context.Context, secret *models.RegistrySecret) error

	// DeleteSecret 删除仓库认证
	DeleteSecret(ctx context.Context, id int64) error

	// GetSecretCredentials 获取认证凭据（包含密码）
	GetSecretCredentials(ctx context.Context, id int64) (*models.RegistrySecret, error)
}

// ScheduledTaskRepository 定时任务存储接口
type ScheduledTaskRepository interface {
	// CreateScheduledTask 创建定时任务
	CreateScheduledTask(ctx context.Context, task *models.ScheduledTask) error

	// GetScheduledTask 获取定时任务
	GetScheduledTask(ctx context.Context, id string) (*models.ScheduledTask, error)

	// ListScheduledTasks 列出定时任务
	ListScheduledTasks(ctx context.Context, offset, limit int) ([]*models.ScheduledTask, int, error)

	// ListEnabledScheduledTasks 列出所有启用的定时任务
	ListEnabledScheduledTasks(ctx context.Context) ([]*models.ScheduledTask, error)

	// UpdateScheduledTask 更新定时任务
	UpdateScheduledTask(ctx context.Context, task *models.ScheduledTask) error

	// DeleteScheduledTask 删除定时任务
	DeleteScheduledTask(ctx context.Context, id string) error
}

// ScheduledExecutionRepository 定时任务执行历史存储接口
type ScheduledExecutionRepository interface {
	// CreateExecution 创建执行记录
	CreateExecution(ctx context.Context, execution *models.ScheduledExecution) error

	// GetExecution 获取执行记录
	GetExecution(ctx context.Context, id int64) (*models.ScheduledExecution, error)

	// ListExecutions 列出执行历史
	ListExecutions(ctx context.Context, scheduledTaskID string, offset, limit int) ([]*models.ScheduledExecution, int, error)

	// ListRunningExecutions 获取正在运行的执行记录
	ListRunningExecutions(ctx context.Context, scheduledTaskID string) ([]*models.ScheduledExecution, error)

	// UpdateExecution 更新执行记录
	UpdateExecution(ctx context.Context, execution *models.ScheduledExecution) error

	// DeleteOldExecutions 删除旧的执行记录（90天前）
	DeleteOldExecutions(ctx context.Context, before time.Time) (int64, error)
}
