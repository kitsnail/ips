package models

import "time"

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// Task 代表一个镜像预热任务
type Task struct {
	ID            string            `json:"taskId"`
	Status        TaskStatus        `json:"status"`
	Priority      int               `json:"priority"` // 优先级 1-10，数字越大优先级越高
	Images        []string          `json:"images"`
	BatchSize     int               `json:"batchSize"`
	NodeSelector  map[string]string `json:"nodeSelector,omitempty"`
	Progress      *Progress         `json:"progress,omitempty"`
	FailedNodes   []FailedNode      `json:"failedNodeDetails,omitempty"`
	MaxRetries    int               `json:"maxRetries"`           // 最大重试次数
	RetryCount    int               `json:"retryCount"`           // 当前重试次数
	RetryStrategy string            `json:"retryStrategy"`        // 重试策略: "linear" 或 "exponential"
	RetryDelay    int               `json:"retryDelay,omitempty"` // 重试延迟（秒）
	WebhookURL    string            `json:"webhookUrl,omitempty"` // Webhook 通知 URL
	CreatedAt     time.Time         `json:"createdAt"`
	StartedAt     *time.Time        `json:"startedAt,omitempty"`
	FinishedAt    *time.Time        `json:"finishedAt,omitempty"`
	EstimatedEnd  *time.Time        `json:"estimatedCompletion,omitempty"`
}

// Progress 任务进度
type Progress struct {
	TotalNodes     int     `json:"totalNodes"`
	CompletedNodes int     `json:"completedNodes"`
	FailedNodes    int     `json:"failedNodes"`
	CurrentBatch   int     `json:"currentBatch"`
	TotalBatches   int     `json:"totalBatches"`
	Percentage     float64 `json:"percentage"`
}

// FailedNode 失败节点详情
type FailedNode struct {
	NodeName  string    `json:"nodeName"`
	Image     string    `json:"image"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// CalculateProgress 计算任务进度
func (t *Task) CalculateProgress() {
	if t.Progress == nil {
		return
	}

	total := t.Progress.TotalNodes
	if total == 0 {
		t.Progress.Percentage = 0
		return
	}

	completed := t.Progress.CompletedNodes
	t.Progress.Percentage = float64(completed) / float64(total) * 100
}
