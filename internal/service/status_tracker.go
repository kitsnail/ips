package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
)

// StatusTracker 状态跟踪器
type StatusTracker struct {
	repo       repository.TaskRepository
	jobCreator *k8s.JobCreator
	logger     *logrus.Logger
}

// NewStatusTracker 创建状态跟踪器
func NewStatusTracker(repo repository.TaskRepository, jobCreator *k8s.JobCreator, logger *logrus.Logger) *StatusTracker {
	return &StatusTracker{
		repo:       repo,
		jobCreator: jobCreator,
		logger:     logger,
	}
}

// TrackTask 跟踪任务状态
// 使用轮询方式定期检查Job状态
func (t *StatusTracker) TrackTask(ctx context.Context, taskID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	t.logger.WithField("taskId", taskID).Info("Starting task tracking")

	for {
		select {
		case <-ticker.C:
			// 获取任务
			task, err := t.repo.Get(ctx, taskID)
			if err != nil {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"error":  err,
				}).Error("Failed to get task")
				continue
			}

			// 如果任务已经完成/失败/取消，停止跟踪
			if task.Status == models.TaskCompleted ||
				task.Status == models.TaskFailed ||
				task.Status == models.TaskCancelled {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"status": task.Status,
				}).Info("Task tracking completed")
				return nil
			}

			// 更新任务状态
			err = t.updateTaskStatus(ctx, task)
			if err != nil {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"error":  err,
				}).Error("Failed to update task status")
			}

		case <-ctx.Done():
			t.logger.WithField("taskId", taskID).Warn("Task tracking cancelled")
			return ctx.Err()
		}
	}
}

// updateTaskStatus 更新任务状态
func (t *StatusTracker) updateTaskStatus(ctx context.Context, task *models.Task) error {
	// 获取任务相关的所有Job
	jobs, err := t.jobCreator.ListJobsByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	if len(jobs) == 0 {
		// 如果还没有Job，任务状态保持为pending或running
		return nil
	}

	// 统计Job状态
	var completed, failed, running int
	var failedNodes []models.FailedNode

	for _, job := range jobs {
		nodeName := job.Labels["node"]

		if job.Status.Succeeded > 0 {
			completed++
		} else if job.Status.Failed > 0 {
			failed++
			// 记录失败详情
			failedNode := models.FailedNode{
				NodeName:  nodeName,
				Reason:    "JobFailed",
				Message:   getJobFailureMessage(&job),
				Timestamp: time.Now(),
			}
			failedNodes = append(failedNodes, failedNode)
		} else {
			running++
		}
	}

	// 更新任务进度
	if task.Progress == nil {
		task.Progress = &models.Progress{}
	}

	task.Progress.CompletedNodes = completed
	task.Progress.FailedNodes = failed
	task.FailedNodes = failedNodes

	// 计算百分比
	task.CalculateProgress()

	// 判断任务最终状态
	totalProcessed := completed + failed
	if totalProcessed == task.Progress.TotalNodes {
		// 所有节点都处理完毕
		now := time.Now()
		task.FinishedAt = &now

		// 根据成功率判定最终状态
		successRate := float64(completed) / float64(task.Progress.TotalNodes)
		if successRate >= 0.9 {
			task.Status = models.TaskCompleted
		} else {
			task.Status = models.TaskFailed
		}

		t.logger.WithFields(logrus.Fields{
			"taskId":      task.ID,
			"status":      task.Status,
			"completed":   completed,
			"failed":      failed,
			"successRate": successRate,
		}).Info("Task finished")
	} else if running > 0 {
		// 还有Job在运行
		if task.Status == models.TaskPending {
			task.Status = models.TaskRunning
			now := time.Now()
			task.StartedAt = &now
		}
	}

	// 保存任务状态
	return t.repo.Update(ctx, task)
}

// getJobFailureMessage 获取Job失败原因
func getJobFailureMessage(job *batchv1.Job) string {
	if len(job.Status.Conditions) > 0 {
		lastCondition := job.Status.Conditions[len(job.Status.Conditions)-1]
		return lastCondition.Message
	}
	return "Job failed without detailed message"
}
