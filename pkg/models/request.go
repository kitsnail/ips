package models

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Images        []string          `json:"images" binding:"required,min=1"`
	BatchSize     int               `json:"batchSize" binding:"required,min=1,max=100"`
	Priority      int               `json:"priority" binding:"omitempty,min=1,max=10"`                   // 优先级 1-10，默认 5
	NodeSelector  map[string]string `json:"nodeSelector,omitempty"`
	MaxRetries    int               `json:"maxRetries" binding:"omitempty,min=0,max=5"`                  // 最大重试次数，默认 0（不重试）
	RetryStrategy string            `json:"retryStrategy" binding:"omitempty,oneof=linear exponential"` // 重试策略，默认 linear
	RetryDelay    int               `json:"retryDelay" binding:"omitempty,min=1,max=300"`                // 重试延迟（秒），默认 30
	WebhookURL    string            `json:"webhookUrl" binding:"omitempty,url"`                          // Webhook 通知 URL
}

// ListTasksRequest 列表查询请求
type ListTasksRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int    `form:"offset" binding:"omitempty,min=0"`
}

// TaskFilter 任务过滤条件
type TaskFilter struct {
	Status *TaskStatus
	Limit  int
	Offset int
}
