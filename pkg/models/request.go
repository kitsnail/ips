package models

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Images       []string          `json:"images" binding:"required,min=1"`
	BatchSize    int               `json:"batchSize" binding:"required,min=1,max=100"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
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
