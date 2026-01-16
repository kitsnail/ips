package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
)

// TaskManager 任务管理器
type TaskManager struct {
	repo           repository.TaskRepository
	nodeFilter     *NodeFilter
	batchScheduler *BatchScheduler
	statusTracker  *StatusTracker
	logger         *logrus.Logger

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
	return &TaskManager{
		repo:           repo,
		nodeFilter:     nodeFilter,
		batchScheduler: batchScheduler,
		statusTracker:  statusTracker,
		logger:         logger,
	}
}

// CreateTask 创建任务
func (m *TaskManager) CreateTask(ctx context.Context, req *models.CreateTaskRequest) (*models.Task, error) {
	// 生成任务ID
	taskID := models.GenerateTaskID()

	// 创建任务对象
	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskPending,
		Images:       req.Images,
		BatchSize:    req.BatchSize,
		NodeSelector: req.NodeSelector,
		CreatedAt:    time.Now(),
	}

	// 保存任务
	err := m.repo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"taskId":    task.ID,
		"images":    task.Images,
		"batchSize": task.BatchSize,
	}).Info("Task created")

	// 在后台执行任务
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.taskContexts.Store(task.ID, cancel)
		defer m.taskContexts.Delete(task.ID)

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

	// 1. 获取符合条件的节点
	nodes, err := m.nodeFilter.FilterNodes(ctx, task.NodeSelector)
	if err != nil {
		return m.markTaskFailed(ctx, task, fmt.Errorf("failed to filter nodes: %w", err))
	}

	if len(nodes) == 0 {
		return m.markTaskFailed(ctx, task, fmt.Errorf("no ready nodes found"))
	}

	m.logger.WithFields(logrus.Fields{
		"taskId":    task.ID,
		"nodeCount": len(nodes),
	}).Info("Nodes filtered")

	// 2. 初始化进度
	totalBatches, err := m.batchScheduler.CalculateBatches(len(nodes), task.BatchSize)
	if err != nil {
		return m.markTaskFailed(ctx, task, err)
	}

	task.Progress = &models.Progress{
		TotalNodes:     len(nodes),
		CompletedNodes: 0,
		FailedNodes:    0,
		CurrentBatch:   0,
		TotalBatches:   totalBatches,
		Percentage:     0,
	}

	task.Status = models.TaskRunning
	now := time.Now()
	task.StartedAt = &now

	if err := m.repo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 3. 启动状态跟踪器
	go func() {
		if err := m.statusTracker.TrackTask(ctx, task.ID); err != nil {
			m.logger.WithFields(logrus.Fields{
				"taskId": task.ID,
				"error":  err,
			}).Error("Status tracking failed")
		}
	}()

	// 4. 执行批次调度
	err = m.batchScheduler.ExecuteBatches(
		ctx,
		task.ID,
		nodes,
		task.Images,
		task.BatchSize,
		func(batchNum, succeeded, failed int) {
			// 批次完成回调
			m.logger.WithFields(logrus.Fields{
				"taskId":    task.ID,
				"batchNum":  batchNum,
				"succeeded": succeeded,
				"failed":    failed,
			}).Info("Batch completed")

			// 更新当前批次
			task, err := m.repo.Get(ctx, task.ID)
			if err == nil && task.Progress != nil {
				task.Progress.CurrentBatch = batchNum
				m.repo.Update(ctx, task)
			}
		},
	)

	if err != nil {
		return m.markTaskFailed(ctx, task, fmt.Errorf("batch execution failed: %w", err))
	}

	// 标记任务完成
	updatedTask, getErr := m.repo.Get(ctx, task.ID)
	if getErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": task.ID,
			"error":  getErr,
		}).Error("Failed to get task for completion")
		return getErr
	}

	updatedTask.Status = models.TaskCompleted
	completedAt := time.Now()
	updatedTask.FinishedAt = &completedAt

	if updateErr := m.repo.Update(ctx, updatedTask); updateErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": updatedTask.ID,
			"error":  updateErr,
		}).Error("Failed to update task status to completed")
		return updateErr
	}

	m.logger.WithField("taskId", updatedTask.ID).Info("Task execution completed")
	return nil
}

// markTaskFailed 标记任务失败
func (m *TaskManager) markTaskFailed(ctx context.Context, task *models.Task, err error) error {
	task.Status = models.TaskFailed
	now := time.Now()
	task.FinishedAt = &now

	if updateErr := m.repo.Update(ctx, task); updateErr != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId": task.ID,
			"error":  updateErr,
		}).Error("Failed to update task status to failed")
	}

	return err
}

// GetTask 获取任务
func (m *TaskManager) GetTask(ctx context.Context, id string) (*models.Task, error) {
	return m.repo.Get(ctx, id)
}

// ListTasks 列出任务
func (m *TaskManager) ListTasks(ctx context.Context, filter models.TaskFilter) ([]*models.Task, int, error) {
	return m.repo.List(ctx, filter)
}

// CancelTask 取消任务
func (m *TaskManager) CancelTask(ctx context.Context, id string) error {
	// 获取任务
	task, err := m.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status == models.TaskCompleted || task.Status == models.TaskFailed || task.Status == models.TaskCancelled {
		return fmt.Errorf("task already finished with status: %s", task.Status)
	}

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

	err = m.repo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	m.logger.WithField("taskId", id).Info("Task cancelled")
	return nil
}
