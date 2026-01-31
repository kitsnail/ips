package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/metrics"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

var (
	ErrCronExpressionInvalid = errors.New("invalid cron expression")
)

type ScheduledTaskManager struct {
	scheduledTaskRepo repository.ScheduledTaskRepository
	executionRepo     repository.ScheduledExecutionRepository
	taskManager       *TaskManager
	logger            *logrus.Logger

	cronScheduler  *cron.Cron
	mu             sync.RWMutex
	cronEntries    map[string]cron.EntryID
	executingTasks map[string]bool
	taskQueue      map[string][]string
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewScheduledTaskManager(
	scheduledTaskRepo repository.ScheduledTaskRepository,
	executionRepo repository.ScheduledExecutionRepository,
	taskManager *TaskManager,
	logger *logrus.Logger,
) *ScheduledTaskManager {
	return &ScheduledTaskManager{
		scheduledTaskRepo: scheduledTaskRepo,
		executionRepo:     executionRepo,
		taskManager:       taskManager,
		logger:            logger,
		cronScheduler:     cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor))),
		cronEntries:       make(map[string]cron.EntryID),
		executingTasks:    make(map[string]bool),
		taskQueue:         make(map[string][]string),
	}
}

func (m *ScheduledTaskManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cronScheduler != nil {
		tasks, err := m.scheduledTaskRepo.ListEnabledScheduledTasks(context.Background())
		if err != nil {
			return fmt.Errorf("failed to load scheduled tasks: %w", err)
		}

		for _, task := range tasks {
			if err := m.addTaskToScheduler(task); err != nil {
				m.logger.WithFields(logrus.Fields{
					"taskId":   task.ID,
					"cronExpr": task.CronExpr,
					"error":    err,
				}).Error("Failed to add scheduled task to scheduler")
				continue
			}
		}

		metrics.ActiveScheduledTasks.Set(float64(len(tasks)))
		m.cronScheduler.Start()
		m.logger.WithField("tasksLoaded", len(tasks)).Info("Scheduled task manager started")
	}

	return nil
}

func (m *ScheduledTaskManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cronScheduler != nil {
		ctx := m.cronScheduler.Stop()
		<-ctx.Done()
		m.logger.Info("Scheduled task manager stopped")
	}

	if m.cancel != nil {
		m.cancel()
	}
}

func (m *ScheduledTaskManager) AddTask(task *models.ScheduledTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !task.Enabled {
		return nil
	}

	return m.addTaskToScheduler(task)
}

func (m *ScheduledTaskManager) addTaskToScheduler(task *models.ScheduledTask) error {
	if _, exists := m.cronEntries[task.ID]; exists {
		return fmt.Errorf("task %s already scheduled", task.ID)
	}

	entryID, err := m.cronScheduler.AddFunc(task.CronExpr, func() {
		m.executeTask(task.ID)
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCronExpressionInvalid, err)
	}

	m.cronEntries[task.ID] = entryID

	nextRun := m.cronScheduler.Entry(entryID).Next
	task.NextExecutionAt = &nextRun

	if err := m.scheduledTaskRepo.UpdateScheduledTask(context.Background(), task); err != nil {
		m.logger.WithFields(logrus.Fields{
			"taskId":        task.ID,
			"nextExecution": nextRun,
			"error":         err,
		}).Error("Failed to update next execution time")
	}

	m.logger.WithFields(logrus.Fields{
		"taskId":        task.ID,
		"cronExpr":      task.CronExpr,
		"nextExecution": nextRun,
	}).Info("Scheduled task added to scheduler")

	return nil
}

func (m *ScheduledTaskManager) RemoveTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entryID, exists := m.cronEntries[taskID]
	if !exists {
		return nil
	}

	m.cronScheduler.Remove(entryID)
	delete(m.cronEntries, taskID)

	m.logger.WithField("taskId", taskID).Info("Scheduled task removed from scheduler")
	return nil
}

func (m *ScheduledTaskManager) EnableTask(taskID string) error {
	task, err := m.scheduledTaskRepo.GetScheduledTask(context.Background(), taskID)
	if err != nil {
		return err
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	if err := m.scheduledTaskRepo.UpdateScheduledTask(context.Background(), task); err != nil {
		return err
	}

	return m.AddTask(task)
}

func (m *ScheduledTaskManager) DisableTask(taskID string) error {
	task, err := m.scheduledTaskRepo.GetScheduledTask(context.Background(), taskID)
	if err != nil {
		return err
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	if err := m.scheduledTaskRepo.UpdateScheduledTask(context.Background(), task); err != nil {
		return err
	}

	return m.RemoveTask(taskID)
}

func (m *ScheduledTaskManager) TriggerTask(taskID string) (string, error) {
	task, err := m.scheduledTaskRepo.GetScheduledTask(context.Background(), taskID)
	if err != nil {
		return "", err
	}

	actualTaskID, err := m.executeTask(task.ID)
	if err != nil {
		return "", err
	}

	return actualTaskID, nil
}

func (m *ScheduledTaskManager) GetScheduledTask(ctx context.Context, taskID string) (*models.ScheduledTask, error) {
	return m.scheduledTaskRepo.GetScheduledTask(ctx, taskID)
}

func (m *ScheduledTaskManager) CreateScheduledTask(ctx context.Context, task *models.ScheduledTask) error {
	return m.scheduledTaskRepo.CreateScheduledTask(ctx, task)
}

func (m *ScheduledTaskManager) ListScheduledTasks(ctx context.Context, offset, limit int) ([]*models.ScheduledTask, int, error) {
	return m.scheduledTaskRepo.ListScheduledTasks(ctx, offset, limit)
}

func (m *ScheduledTaskManager) DeleteScheduledTask(ctx context.Context, taskID string) error {
	return m.scheduledTaskRepo.DeleteScheduledTask(ctx, taskID)
}

func (m *ScheduledTaskManager) isPreviousTaskRunning(scheduledTaskID string) bool {
	executions, err := m.executionRepo.ListRunningExecutions(context.Background(), scheduledTaskID)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"scheduledTaskId": scheduledTaskID,
			"error":           err,
		}).Error("Failed to check running executions")
		return false
	}

	m.logger.WithFields(logrus.Fields{
		"scheduledTaskId":   scheduledTaskID,
		"runningExecutions": len(executions),
	}).Debug("Checking if previous task is running")

	if len(executions) > 0 {
		return true
	}
	return false
}

func (m *ScheduledTaskManager) executeTask(scheduledTaskID string) (string, error) {
	task, err := m.scheduledTaskRepo.GetScheduledTask(context.Background(), scheduledTaskID)
	if err != nil {
		return "", err
	}

	if !task.Enabled {
		return "", fmt.Errorf("task %s is disabled", scheduledTaskID)
	}

	triggeredAt := time.Now()

	execution := &models.ScheduledExecution{
		ScheduledTaskID: scheduledTaskID,
		TriggeredAt:     triggeredAt,
		Status:          models.ScheduledExecutionSuccess,
		StartedAt:       triggeredAt,
	}

	if task.OverlapPolicy == models.OverlapPolicySkip {
		if m.isPreviousTaskRunning(scheduledTaskID) {
			execution.Status = models.ScheduledExecutionSkipped
			execution.ErrorMessage = "Previous task still running, skipped this execution"
			finishedAt := triggeredAt
			execution.FinishedAt = &finishedAt

			if err := m.executionRepo.CreateExecution(context.Background(), execution); err != nil {
				m.logger.WithFields(logrus.Fields{
					"scheduledTaskId": scheduledTaskID,
					"error":           err,
				}).Error("Failed to create skipped execution record")
			}

			m.logger.WithField("scheduledTaskId", scheduledTaskID).Info("Scheduled task execution skipped (previous task still running)")
			metrics.ScheduledTaskExecutionsTotal.WithLabelValues("skipped").Inc()

			m.mu.Lock()
			delete(m.executingTasks, scheduledTaskID)
			m.mu.Unlock()
			return "", fmt.Errorf("task execution skipped: previous task still running")
		}
	}

	if task.OverlapPolicy == models.OverlapPolicyQueue {
		if m.isPreviousTaskRunning(scheduledTaskID) {
			execution.Status = models.ScheduledExecutionSkipped
			execution.ErrorMessage = "Previous task still running, queued this execution"
			finishedAt := triggeredAt
			execution.FinishedAt = &finishedAt

			if err := m.executionRepo.CreateExecution(context.Background(), execution); err != nil {
				m.logger.WithFields(logrus.Fields{
					"scheduledTaskId": scheduledTaskID,
					"error":           err,
				}).Error("Failed to create queued execution record")
			}

			m.mu.Lock()
			m.taskQueue[scheduledTaskID] = append(m.taskQueue[scheduledTaskID], "queued-"+time.Now().Format("20060102150405.999"))
			m.mu.Unlock()

			m.logger.WithField("scheduledTaskId", scheduledTaskID).Info("Scheduled task execution queued (previous task still running)")
			metrics.ScheduledTaskExecutionsTotal.WithLabelValues("queued").Inc()

			m.mu.Lock()
			delete(m.executingTasks, scheduledTaskID)
			m.mu.Unlock()
			return "", fmt.Errorf("task execution queued: previous task still running")
		}
	}

	if task.OverlapPolicy == models.OverlapPolicyQueue {
		if m.isPreviousTaskRunning(scheduledTaskID) {
			execution.Status = models.ScheduledExecutionSkipped
			execution.ErrorMessage = "Previous task still running, queued this execution"
			finishedAt := triggeredAt
			execution.FinishedAt = &finishedAt

			if err := m.executionRepo.CreateExecution(context.Background(), execution); err != nil {
				m.logger.WithFields(logrus.Fields{
					"scheduledTaskId": scheduledTaskID,
					"error":           err,
				}).Error("Failed to create queued execution record")
			}

			m.mu.Lock()
			m.taskQueue[scheduledTaskID] = append(m.taskQueue[scheduledTaskID], "queued-"+time.Now().Format("20060102150405.999"))
			m.mu.Unlock()

			m.logger.WithField("scheduledTaskId", scheduledTaskID).Info("Scheduled task execution queued (previous task still running)")
			metrics.ScheduledTaskExecutionsTotal.WithLabelValues("queued").Inc()

			return "", fmt.Errorf("task execution queued: previous task still running")
		}
	}

	m.mu.Lock()
	m.executingTasks[scheduledTaskID] = true
	m.mu.Unlock()

	ctx := context.Background()
	if task.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(task.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	createReq := &models.CreateTaskRequest{
		Images:        task.TaskConfig.Images,
		BatchSize:     task.TaskConfig.BatchSize,
		Priority:      task.TaskConfig.Priority,
		NodeSelector:  task.TaskConfig.NodeSelector,
		MaxRetries:    task.TaskConfig.MaxRetries,
		RetryStrategy: task.TaskConfig.RetryStrategy,
		RetryDelay:    task.TaskConfig.RetryDelay,
		WebhookURL:    task.TaskConfig.WebhookURL,
		SecretID:      task.TaskConfig.SecretID,
	}

	actualTask, err := m.taskManager.CreateTask(ctx, createReq)
	if err != nil {
		execution.Status = models.ScheduledExecutionFailed
		execution.ErrorMessage = fmt.Sprintf("Failed to create task: %v", err)
		finishedAt := time.Now()
		execution.FinishedAt = &finishedAt
		execution.DurationSeconds = finishedAt.Sub(triggeredAt).Seconds()

		m.executionRepo.CreateExecution(context.Background(), execution)
		metrics.ScheduledTaskExecutionsTotal.WithLabelValues("failed").Inc()

		m.mu.Lock()
		delete(m.executingTasks, scheduledTaskID)
		m.mu.Unlock()

		return "", err
	}

	execution.TaskID = actualTask.ID

	if err := m.executionRepo.CreateExecution(context.Background(), execution); err != nil {
		m.logger.WithFields(logrus.Fields{
			"scheduledTaskId": scheduledTaskID,
			"taskId":          actualTask.ID,
			"error":           err,
		}).Error("Failed to create execution record")
	}

	go m.monitorExecution(scheduledTaskID, actualTask.ID, execution, task.TimeoutSeconds)

	task.LastExecutionAt = &triggeredAt
	if err := m.scheduledTaskRepo.UpdateScheduledTask(context.Background(), task); err != nil {
		m.logger.WithFields(logrus.Fields{
			"scheduledTaskId": scheduledTaskID,
			"error":           err,
		}).Error("Failed to update last execution time")
	}

	m.logger.WithFields(logrus.Fields{
		"scheduledTaskId": scheduledTaskID,
		"taskId":          actualTask.ID,
		"triggeredAt":     triggeredAt,
	}).Info("Scheduled task execution started")

	return actualTask.ID, nil
}

func (m *ScheduledTaskManager) monitorExecution(scheduledTaskID, taskID string, execution *models.ScheduledExecution, timeoutSeconds int) {
	ctx := context.Background()
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	for {
		select {
		case <-ctx.Done():
			if timeoutSeconds > 0 && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				execution.Status = models.ScheduledExecutionTimeout
				execution.ErrorMessage = "Task execution timed out"
				finishedAt := time.Now()
				execution.FinishedAt = &finishedAt
				execution.DurationSeconds = finishedAt.Sub(execution.StartedAt).Seconds()

				m.executionRepo.UpdateExecution(context.Background(), execution)
				metrics.ScheduledTaskExecutionsTotal.WithLabelValues("timeout").Inc()

				m.logger.WithFields(logrus.Fields{
					"scheduledTaskId": scheduledTaskID,
					"taskId":          taskID,
				}).Warn("Scheduled task execution timed out")

				m.taskManager.DeleteTask(context.Background(), taskID)
			}

			m.mu.Lock()
			delete(m.executingTasks, scheduledTaskID)
			m.mu.Unlock()

			m.mu.Lock()
			if queue, exists := m.taskQueue[scheduledTaskID]; exists && len(queue) > 0 {
				nextTaskID := queue[0]
				m.taskQueue[scheduledTaskID] = queue[1:]

				m.logger.WithFields(logrus.Fields{
					"scheduledTaskId": scheduledTaskID,
					"queuedTaskId":    nextTaskID,
				}).Info("Processing queued scheduled task execution")

				go m.executeTask(scheduledTaskID)
			}
			m.mu.Unlock()

			return
		}
	}
}

func (m *ScheduledTaskManager) ListExecutions(ctx context.Context, scheduledTaskID string, offset, limit int) ([]*models.ScheduledExecution, int, error) {
	return m.executionRepo.ListExecutions(ctx, scheduledTaskID, offset, limit)
}

func (m *ScheduledTaskManager) GetExecution(ctx context.Context, id int64) (*models.ScheduledExecution, error) {
	return m.executionRepo.GetExecution(ctx, id)
}

// CleanupOldExecutions 清理90天前的执行历史
func (m *ScheduledTaskManager) CleanupOldExecutions(ctx context.Context) (int64, error) {
	before := time.Now().Add(-90 * 24 * time.Hour)
	deleted, err := m.executionRepo.DeleteOldExecutions(ctx, before)
	if err != nil {
		return 0, err
	}
	if deleted > 0 {
		m.logger.WithFields(logrus.Fields{
			"deleted": deleted,
			"before":  before,
		}).Info("Cleaned up old scheduled task executions")
	}
	return deleted, nil
}
