package models

import "time"

// OverlapPolicy 定义定时任务重叠时的处理策略
type OverlapPolicy string

const (
	// OverlapPolicySkip 跳过本次执行（默认，防止资源耗尽）
	OverlapPolicySkip OverlapPolicy = "skip"
	// OverlapPolicyAllow 允许并行执行
	OverlapPolicyAllow OverlapPolicy = "allow"
	// OverlapPolicyQueue 等待上次完成后执行（暂未实现）
	OverlapPolicyQueue OverlapPolicy = "queue"
)

// ScheduledTaskExecutionStatus 定时任务执行状态
type ScheduledTaskExecutionStatus string

const (
	ScheduledExecutionSuccess ScheduledTaskExecutionStatus = "success"
	ScheduledExecutionFailed  ScheduledTaskExecutionStatus = "failed"
	ScheduledExecutionSkipped ScheduledTaskExecutionStatus = "skipped"
	ScheduledExecutionTimeout ScheduledTaskExecutionStatus = "timeout"
)

// TaskConfig 定时任务执行时的任务配置（复用 CreateTaskRequest）
type TaskConfig struct {
	Images        []string          `json:"images"`
	BatchSize     int               `json:"batchSize"`
	Priority      int               `json:"priority"`
	NodeSelector  map[string]string `json:"nodeSelector,omitempty"`
	MaxRetries    int               `json:"maxRetries"`
	RetryStrategy string            `json:"retryStrategy"`
	RetryDelay    int               `json:"retryDelay"`
	WebhookURL    string            `json:"webhookUrl,omitempty"`
	SecretID      int64             `json:"secretId,omitempty"`
}

// ScheduledTask 定时任务模型
type ScheduledTask struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	CronExpr        string        `json:"cronExpr"` // Crontab 表达式（5字段标准格式）
	Enabled         bool          `json:"enabled"`
	TaskConfig      TaskConfig    `json:"taskConfig"`
	OverlapPolicy   OverlapPolicy `json:"overlapPolicy"`  // 默认: skip
	TimeoutSeconds  int           `json:"timeoutSeconds"` // 0 表示无限制
	LastExecutionAt *time.Time    `json:"lastExecutionAt,omitempty"`
	NextExecutionAt *time.Time    `json:"nextExecutionAt,omitempty"`
	CreatedBy       string        `json:"createdBy"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// ScheduledExecution 定时任务执行记录
type ScheduledExecution struct {
	ID              int64                        `json:"id"`
	ScheduledTaskID string                       `json:"scheduledTaskId"`
	TaskID          string                       `json:"taskId"` // 关联的实际 Task ID
	Status          ScheduledTaskExecutionStatus `json:"status"` // success | failed | skipped | timeout
	StartedAt       time.Time                    `json:"startedAt"`
	FinishedAt      *time.Time                   `json:"finishedAt,omitempty"`
	DurationSeconds float64                      `json:"durationSeconds"`
	ErrorMessage    string                       `json:"errorMessage,omitempty"`
	TriggeredAt     time.Time                    `json:"triggeredAt"` // 计划执行时间（实际执行可能因排队而延迟）
}

// CreateScheduledTaskRequest 创建定时任务请求
type CreateScheduledTaskRequest struct {
	Name           string        `json:"name" binding:"required"`
	Description    string        `json:"description"`
	CronExpr       string        `json:"cronExpr" binding:"required"`
	Enabled        bool          `json:"enabled"`
	TaskConfig     TaskConfig    `json:"taskConfig" binding:"required"`
	OverlapPolicy  OverlapPolicy `json:"overlapPolicy"`
	TimeoutSeconds int           `json:"timeoutSeconds"`
}

// UpdateScheduledTaskRequest 更新定时任务请求
type UpdateScheduledTaskRequest struct {
	Name           *string        `json:"name,omitempty"`
	Description    *string        `json:"description,omitempty"`
	CronExpr       *string        `json:"cronExpr,omitempty"`
	Enabled        *bool          `json:"enabled,omitempty"`
	TaskConfig     *TaskConfig    `json:"taskConfig,omitempty"`
	OverlapPolicy  *OverlapPolicy `json:"overlapPolicy,omitempty"`
	TimeoutSeconds *int           `json:"timeoutSeconds,omitempty"`
}

// ScheduledTaskResponse 定时任务响应
type ScheduledTaskResponse struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	CronExpr        string        `json:"cronExpr"`
	Enabled         bool          `json:"enabled"`
	TaskConfig      TaskConfig    `json:"taskConfig"`
	OverlapPolicy   OverlapPolicy `json:"overlapPolicy"`
	TimeoutSeconds  int           `json:"timeoutSeconds"`
	LastExecutionAt *time.Time    `json:"lastExecutionAt,omitempty"`
	NextExecutionAt *time.Time    `json:"nextExecutionAt,omitempty"`
	CreatedBy       string        `json:"createdBy"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// ListScheduledTasksRequest 列出定时任务请求
type ListScheduledTasksRequest struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

// ListScheduledTasksResponse 列出定时任务响应
type ListScheduledTasksResponse struct {
	Tasks  []*ScheduledTask `json:"tasks"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// ListScheduledExecutionsRequest 列出执行历史请求
type ListScheduledExecutionsRequest struct {
	ScheduledTaskID string `form:"scheduledTaskId"`
	Limit           int    `form:"limit"`
	Offset          int    `form:"offset"`
}

// ListScheduledExecutionsResponse 列出执行历史响应
type ListScheduledExecutionsResponse struct {
	Executions []*ScheduledExecution `json:"executions"`
	Total      int                   `json:"total"`
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
}
