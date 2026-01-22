package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/metrics"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// TaskManager 任务管理器
type TaskManager struct {
	repo            repository.TaskRepository
	nodeFilter      *NodeFilter
	batchScheduler  *BatchScheduler
	statusTracker   *StatusTracker
	webhookNotifier *WebhookNotifier
	logger          *logrus.Logger
	concurrencySem  *semaphore.Weighted // 并发控制信号量
	maxConcurrency  int64               // 最大并发任务数

	// 用于存储任务的取消函数
	taskContexts sync.Map // map[string]context.CancelFunc
}

// NewTaskManager 创建任务管理器
func NewTaskManager(
	repo repository.TaskRepository,
	nodeFilter *NodeFilter,
	batchScheduler *BatchScheduler,
	statusTracker *StatusTracker,
	logger *logrus.Logger,
) *TaskManager {
	// 默认最大并发任务数为 3
	maxConcurrency := int64(3)
	// 可以通过环境变量配置
	// if envMax := os.Getenv("MAX_CONCURRENT_TASKS"); envMax != "" {
	//     if max, err := strconv.ParseInt(envMax, 10, 64); err == nil && max > 0 {
	//         maxConcurrency = max
	//     }
	// }

	return &TaskManager{
		repo:            repo,
		nodeFilter:      nodeFilter,
		batchScheduler:  batchScheduler,
		statusTracker:   statusTracker,
		webhookNotifier: NewWebhookNotifier(logger),
		logger:          logger,
		concurrencySem:  semaphore.NewWeighted(maxConcurrency),
		maxConcurrency:  maxConcurrency,
	}
}

// CreateTask 创建任务
func (m *TaskManager) CreateTask(ctx context.Context, req *models.CreateTaskRequest) (*models.Task, error) {
	// 校验镜像数量
	if len(req.Images) > 50 {
		return nil, fmt.Errorf("too many images: max 50 images allowed per task")
	}

	// 生成任务ID
	taskID := models.GenerateTaskID()

	// 设置默认优先级
	priority := req.Priority
	if priority == 0 {
		priority = 5 // 默认优先级为 5
	}

	// 设置重试配置默认值
	retryStrategy := req.RetryStrategy
	if retryStrategy == "" {
		retryStrategy = "linear"
	}

	retryDelay := req.RetryDelay
	if retryDelay == 0 {
		retryDelay = 30 // 默认30秒
	}

	// 创建任务对象
	task := &models.Task{
		ID:            taskID,
		Status:        models.TaskPending,
		Priority:      priority,
		Images:        req.Images,
		BatchSize:     req.BatchSize,
		NodeSelector:  req.NodeSelector,
		MaxRetries:    req.MaxRetries,
		RetryCount:    0,
		RetryStrategy: retryStrategy,
		RetryDelay:    retryDelay,
		WebhookURL:    req.WebhookURL,
		CreatedAt:     time.Now(),
	}

	// 保存任务
	err := m.repo.CreateTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 记录指标
	metrics.TasksTotal.WithLabelValues(string(models.TaskPending)).Inc()
	metrics.ActiveTasks.Inc()

	m.logger.WithFields(logrus.Fields{
		"taskId":        task.ID,
		"priority":      task.Priority,
		"images":        task.Images,
		"batchSize":     task.BatchSize,
		"maxRetries":    task.MaxRetries,
		"retryStrategy": task.RetryStrategy,
	}).Info("Task created")

	// 在后台执行任务
	go func() {
		// Create context and store it immediately so the task can be cancelled while pending
		ctx, cancel := context.WithCancel(context.Background())
		m.taskContexts.Store(task.ID, cancel)

		// Ensure cleanup happens when the goroutine exits
		defer m.taskContexts.Delete(task.ID)
		defer cancel()

		// 等待获取并发槽位
		// Use the cancellable context so we can abort if the task is cancelled while waiting
		if err := m.concurrencySem.Acquire(ctx, 1); err != nil {
			m.logger.WithFields(logrus.Fields{
				"taskId": task.ID,
				"error":  err,
			}).Warn("Failed to acquire concurrency slot (task cancelled or timeout)")
			return
		}
		defer m.concurrencySem.Release(1)

		m.logger.WithFields(logrus.Fields{
			"taskId":         task.ID,
			"maxConcurrency": m.maxConcurrency,
		}).Info("Task acquired execution slot")

		// Check strictly if context is already cancelled (redundant with Acquire but safe)
		if ctx.Err() != nil {
			return
		}

		if err := m.executeTask(ctx, task); err != nil {
			m.logger.WithFields(logrus.Fields{
				"taskId": task.ID,
				"error":  err,
			}).Error("Task execution failed")
		}
	}()

	return task, nil
}

// executeTask 执行任务
func (m *TaskManager) executeTask(ctx context.Context, task *models.Task) error {
	m.logger.WithField("taskId", task.ID).Info("Starting task execution")

	// 记录任务开始时间
	startTime := time.Now()

	// 1. 获取符合条件的节点
	nodes, err := m.nodeFilter.FilterNodes(ctx, task.NodeSelector)
	if err != nil {
		return m.markTaskFailed(ctx, task, fmt.Errorf("failed to filter nodes: %w", err), startTime)
	}

	if len(nodes) == 0 {
		return m.markTaskFailed(ctx, task, fmt.Errorf("no ready nodes found"), startTime)
	}

	m.logger.WithFields(logrus.Fields{
		"taskId":    task.ID,
		"nodeCount": len(nodes),
	}).Info("Nodes filtered")

	// 2. 初始化进度
	totalBatches, err := m.batchScheduler.CalculateBatches(len(nodes), task.BatchSize)
	if err != nil {
		return m.markTaskFailed(ctx, task, err, startTime)
	}

	task.Progress = &models.Progress{
		TotalNodes:     len(nodes),
		CompletedNodes: 0,
		FailedNodes:    0,
		CurrentBatch:   0,
		TotalBatches:   totalBatches,
		Percentage:     0,
	}

	// Check if context is cancelled before starting
	if ctx.Err() != nil {
		return ctx.Err()
	}

	task.Status = models.TaskRunning
	now := time.Now()
	task.StartedAt = &now

	if err := m.repo.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 记录任务状态变更指标
	metrics.TasksTotal.WithLabelValues(string(models.TaskRunning)).Inc()

	// 3. 执行批次调度 (创建所有 Job)
	err = m.batchScheduler.ExecuteBatches(
		ctx,
		task.ID,
		nodes,
		task.Images,
		task.BatchSize,
		func(batchNum, succeeded, failed int) {
			m.logger.WithFields(logrus.Fields{
				"taskId":    task.ID,
				"batchNum":  batchNum,
				"succeeded": succeeded,
				"failed":    failed,
			}).Info("Batch submitted")
			// 更新批次数进度
			task.Progress.CurrentBatch = batchNum
			m.repo.UpdateTask(ctx, task)
		},
	)

	if err != nil {
		return m.markTaskFailed(ctx, task, fmt.Errorf("batch execution failed: %w", err), startTime)
	}

	// 4. 同步等待状态跟踪器完成 (它会观察 Job 状态并上报最终结果)
	err = m.statusTracker.TrackTask(ctx, task.ID)
	if err != nil {
		return m.markTaskFailed(ctx, task, fmt.Errorf("status tracking failed: %w", err), startTime)
	}

	m.logger.WithField("taskId", task.ID).Info("Task execution context finished")
	return nil
}

// markTaskFailed 标记任务失败或触发重试
func (m *TaskManager) markTaskFailed(ctx context.Context, task *models.Task, err error, startTime time.Time) error {
	// If the context is cancelled, it means the task was manually cancelled.
	// We should NOT overwrite the Cancelled status with Failed or Pending.
	if ctx.Err() != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": task.ID,
			"error":  err,
		}).Info("Task context cancelled, skipping failure handling")
		return ctx.Err()
	}

	// 检查是否可以重试
	if task.RetryCount < task.MaxRetries {
		// 增加重试计数
		task.RetryCount++

		// 获取重试策略
		retryStrategy := GetRetryStrategy(task.RetryStrategy, m.logger)
		delay := retryStrategy.CalculateDelay(task.RetryCount, task.RetryDelay)

		m.logger.WithFields(logrus.Fields{
			"taskId":     task.ID,
			"retryCount": task.RetryCount,
			"maxRetries": task.MaxRetries,
			"delay":      delay.Seconds(),
			"error":      err,
		}).Info("Task failed, scheduling retry")

		// 更新任务状态为 pending 以准备重试
		task.Status = models.TaskPending
		task.FinishedAt = nil // 清除完成时间

		if updateErr := m.repo.UpdateTask(ctx, task); updateErr != nil {
			m.logger.WithFields(logrus.Fields{
				"taskId": task.ID,
				"error":  updateErr,
			}).Error("Failed to update task for retry")
		}

		// 等待重试延迟后重新执行任务
		go func() {
			time.Sleep(delay)

			m.logger.WithFields(logrus.Fields{
				"taskId":     task.ID,
				"retryCount": task.RetryCount,
			}).Info("Retrying task")

			// 重新获取任务确保获取最新状态
			latestTask, getErr := m.repo.GetTask(context.Background(), task.ID)
			if getErr != nil {
				m.logger.WithFields(logrus.Fields{
					"taskId": task.ID,
					"error":  getErr,
				}).Error("Failed to get task for retry")
				return
			}

			// 检查任务是否被取消
			if latestTask.Status == models.TaskCancelled {
				m.logger.WithField("taskId", task.ID).Info("Task was cancelled, skipping retry")
				return
			}

			// 重新执行任务
			ctx, cancel := context.WithCancel(context.Background())
			m.taskContexts.Store(task.ID, cancel)
			defer m.taskContexts.Delete(task.ID)

			if execErr := m.executeTask(ctx, latestTask); execErr != nil {
				m.logger.WithFields(logrus.Fields{
					"taskId": task.ID,
					"error":  execErr,
				}).Error("Task retry failed")
			}
		}()

		return err
	}

	// 达到最大重试次数，标记为失败
	m.logger.WithFields(logrus.Fields{
		"taskId":     task.ID,
		"retryCount": task.RetryCount,
		"error":      err,
	}).Error("Task failed after max retries")

	task.Status = models.TaskFailed
	now := time.Now()
	task.FinishedAt = &now

	if updateErr := m.repo.UpdateTask(ctx, task); updateErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": task.ID,
			"error":  updateErr,
		}).Error("Failed to update task status to failed")
	}

	// 记录任务失败指标
	duration := time.Since(startTime).Seconds()
	metrics.TasksTotal.WithLabelValues(string(models.TaskFailed)).Inc()
	metrics.TaskDuration.WithLabelValues(string(models.TaskFailed)).Observe(duration)
	metrics.ActiveTasks.Dec()

	// 发送 Webhook 通知
	if webhookErr := m.webhookNotifier.NotifyTaskFailed(ctx, task); webhookErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": task.ID,
			"error":  webhookErr,
		}).Warn("Failed to send webhook notification for failed task")
	}

	return err
}

// GetTask 获取任务
func (m *TaskManager) GetTask(ctx context.Context, id string) (*models.Task, error) {
	return m.repo.GetTask(ctx, id)
}

// ListTasks 列出任务
func (m *TaskManager) ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int, error) {
	tasks, total, err := m.repo.ListTasks(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// DeleteTask 删除或取消任务
// 如果任务正在运行，则取消任务
// 如果任务已结束，则删除任务记录
func (m *TaskManager) DeleteTask(ctx context.Context, id string) (string, error) {
	// 获取任务
	task, err := m.repo.GetTask(ctx, id)
	if err != nil {
		return "", err
	}

	// 检查任务状态
	// 如果是终止状态，直接删除记录
	if task.Status == models.TaskCompleted || task.Status == models.TaskFailed || task.Status == models.TaskCancelled {
		if err := m.repo.DeleteTask(ctx, id); err != nil {
			return "", fmt.Errorf("failed to delete task record: %w", err)
		}
		m.logger.WithField("taskId", id).Info("Task record deleted")
		return "deleted", nil
	}

	// 如果是运行状态，执行取消逻辑
	// 取消任务的上下文
	if cancelFunc, ok := m.taskContexts.Load(id); ok {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			cancel()
		}
	}

	// 更新任务状态
	task.Status = models.TaskCancelled
	now := time.Now()
	task.FinishedAt = &now

	err = m.repo.UpdateTask(ctx, task)
	if err != nil {
		return "", fmt.Errorf("failed to update task status to cancelled: %w", err)
	}

	// 记录任务取消指标
	if task.StartedAt != nil {
		duration := time.Since(*task.StartedAt).Seconds()
		metrics.TaskDuration.WithLabelValues(string(models.TaskCancelled)).Observe(duration)
	}
	metrics.TasksTotal.WithLabelValues(string(models.TaskCancelled)).Inc()
	metrics.ActiveTasks.Dec()

	// 发送 Webhook 通知
	if webhookErr := m.webhookNotifier.NotifyTaskCancelled(ctx, task); webhookErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": id,
			"error":  webhookErr,
		}).Warn("Failed to send webhook notification for cancelled task")
	}

	m.logger.WithField("taskId", id).Info("Task cancelled")
	return "cancelled", nil
}
