package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/metrics"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
// 优先使用Watch机制，失败时降级到轮询
func (t *StatusTracker) TrackTask(ctx context.Context, taskID string) error {
	t.logger.WithField("taskId", taskID).Info("Starting task tracking")

	// 尝试使用Watch机制
	err := t.trackTaskWithWatch(ctx, taskID)
	if err != nil {
		t.logger.WithFields(logrus.Fields{
			"taskId": taskID,
			"error":  err,
		}).Warn("Watch mechanism failed, falling back to polling")
		// 降级到轮询
		return t.trackTaskWithPolling(ctx, taskID)
	}

	return nil
}

// trackTaskWithWatch 使用Watch机制跟踪任务
func (t *StatusTracker) trackTaskWithWatch(ctx context.Context, taskID string) error {
	// 创建Watch
	labelSelector := fmt.Sprintf("task-id=%s", taskID)
	watchInterface, err := t.jobCreator.GetK8sClient().Clientset.BatchV1().Jobs(t.jobCreator.GetK8sClient().Namespace).Watch(
		ctx,
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}
	defer watchInterface.Stop()

	t.logger.WithField("taskId", taskID).Info("Using Watch mechanism for task tracking")

	// 定期更新任务状态（每30秒或收到事件时）
	updateTicker := time.NewTicker(30 * time.Second)
	defer updateTicker.Stop()

	// 监听Job变化
	for {
		select {
		case event, ok := <-watchInterface.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			// 处理事件
			if event.Type == watch.Added || event.Type == watch.Modified || event.Type == watch.Deleted {
				t.logger.WithFields(logrus.Fields{
					"taskId":    taskID,
					"eventType": event.Type,
				}).Debug("Received Job event")

				// 获取任务并更新状态
				task, err := t.repo.Get(ctx, taskID)
				if err != nil {
					t.logger.WithFields(logrus.Fields{
						"taskId": taskID,
						"error":  err,
					}).Error("Failed to get task")
					continue
				}

				// 检查任务是否已结束
				if t.isTaskFinished(task) {
					t.logger.WithFields(logrus.Fields{
						"taskId": taskID,
						"status": task.Status,
					}).Info("Task tracking completed via Watch")
					return nil
				}

				// 更新任务状态
				if err := t.updateTaskStatus(ctx, task); err != nil {
					t.logger.WithFields(logrus.Fields{
						"taskId": taskID,
						"error":  err,
					}).Error("Failed to update task status")
				}

				// 再次检查是否完成
				task, _ = t.repo.Get(ctx, taskID)
				if task != nil && t.isTaskFinished(task) {
					t.logger.WithFields(logrus.Fields{
						"taskId": taskID,
						"status": task.Status,
					}).Info("Task tracking completed via Watch")
					return nil
				}
			}

		case <-updateTicker.C:
			// 定期更新（即使没有事件）
			task, err := t.repo.Get(ctx, taskID)
			if err != nil {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"error":  err,
				}).Error("Failed to get task during periodic update")
				continue
			}

			// 检查任务是否已结束
			if t.isTaskFinished(task) {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"status": task.Status,
				}).Info("Task tracking completed")
				return nil
			}

			// 更新任务状态
			if err := t.updateTaskStatus(ctx, task); err != nil {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"error":  err,
				}).Error("Failed to update task status during periodic update")
			}

		case <-ctx.Done():
			t.logger.WithField("taskId", taskID).Warn("Task tracking cancelled")
			return ctx.Err()
		}
	}
}

// trackTaskWithPolling 使用轮询方式跟踪任务（降级方案）
func (t *StatusTracker) trackTaskWithPolling(ctx context.Context, taskID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	t.logger.WithField("taskId", taskID).Info("Using polling for task tracking")

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
			if t.isTaskFinished(task) {
				t.logger.WithFields(logrus.Fields{
					"taskId": taskID,
					"status": task.Status,
				}).Info("Task tracking completed via polling")
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

// isTaskFinished 检查任务是否已结束
func (t *StatusTracker) isTaskFinished(task *models.Task) bool {
	return task.Status == models.TaskCompleted ||
		task.Status == models.TaskFailed ||
		task.Status == models.TaskCancelled
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

	// 初始化 NodeStatuses
	if task.NodeStatuses == nil {
		task.NodeStatuses = make(map[string]map[string]int)
	}

	// 获取 Pod 状态以解析详细信息
	for _, job := range jobs {
		nodeName := job.Labels["node"]
		// 列出该 Job 关联的 Pod
		podList, err := t.jobCreator.GetK8sClient().Clientset.CoreV1().Pods(t.jobCreator.GetK8sClient().Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
		if err != nil || len(podList.Items) == 0 {
			continue
		}

		pod := podList.Items[0]
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == "puller" && cs.State.Terminated != nil && cs.State.Terminated.Message != "" {
				// 解析 Termination Message
				var results map[string]int
				if err := json.Unmarshal([]byte(cs.State.Terminated.Message), &results); err == nil {
					task.NodeStatuses[nodeName] = results

					// 上报指标
					for img, status := range results {
						statusStr := "success"
						if status == 0 {
							statusStr = "failed"
						}
						metrics.ImagePrewarmStatus.WithLabelValues(nodeName, img, statusStr).Inc()
					}
				}
			}
		}
	}

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
