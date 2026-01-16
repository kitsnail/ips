package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kitsnail/ips/internal/k8s"
	"github.com/sirupsen/logrus"
)

// BatchScheduler 批次调度器
type BatchScheduler struct {
	jobCreator *k8s.JobCreator
	logger     *logrus.Logger
}

// NewBatchScheduler 创建批次调度器
func NewBatchScheduler(jobCreator *k8s.JobCreator, logger *logrus.Logger) *BatchScheduler {
	return &BatchScheduler{
		jobCreator: jobCreator,
		logger:     logger,
	}
}

// ExecuteBatches 分批执行任务
// taskID: 任务ID
// nodes: 目标节点列表
// images: 要预热的镜像列表
// batchSize: 每批次的节点数
// onBatchComplete: 每批次完成后的回调函数（批次号, 成功数, 失败数）
func (s *BatchScheduler) ExecuteBatches(
	ctx context.Context,
	taskID string,
	nodes []string,
	images []string,
	batchSize int,
	onBatchComplete func(batchNum, succeeded, failed int),
) error {
	// 分批
	batches := s.splitBatches(nodes, batchSize)

	s.logger.WithFields(logrus.Fields{
		"taskId":       taskID,
		"totalNodes":   len(nodes),
		"totalBatches": len(batches),
		"batchSize":    batchSize,
	}).Info("Starting batch execution")

	// 顺序执行每个批次
	for i, batch := range batches {
		batchNum := i + 1

		s.logger.WithFields(logrus.Fields{
			"taskId":    taskID,
			"batchNum":  batchNum,
			"batchSize": len(batch),
		}).Info("Executing batch")

		// 为批次中的每个节点创建Job
		succeeded, failed := s.executeBatch(ctx, taskID, batch, images)

		s.logger.WithFields(logrus.Fields{
			"taskId":    taskID,
			"batchNum":  batchNum,
			"succeeded": succeeded,
			"failed":    failed,
		}).Info("Batch completed")

		// 回调通知批次完成
		if onBatchComplete != nil {
			onBatchComplete(batchNum, succeeded, failed)
		}

		// 如果上下文被取消，停止执行
		select {
		case <-ctx.Done():
			s.logger.WithField("taskId", taskID).Warn("Batch execution cancelled")
			return ctx.Err()
		default:
			// 继续下一批
		}
	}

	return nil
}

// executeBatch 执行单个批次
func (s *BatchScheduler) executeBatch(ctx context.Context, taskID string, nodes []string, images []string) (succeeded, failed int) {
	// 为批次中的每个节点创建Job
	for _, nodeName := range nodes {
		err := s.jobCreator.CreateJob(ctx, taskID, nodeName, images)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"taskId":   taskID,
				"nodeName": nodeName,
				"error":    err,
			}).Error("Failed to create job")
			failed++
		} else {
			succeeded++
		}
	}

	// 等待一小段时间，避免创建Job过快导致API Server压力过大
	if len(nodes) > 10 {
		time.Sleep(2 * time.Second)
	}

	return succeeded, failed
}

// splitBatches 将节点列表分批
func (s *BatchScheduler) splitBatches(nodes []string, batchSize int) [][]string {
	if batchSize <= 0 {
		batchSize = 10 // 默认批次大小
	}

	var batches [][]string

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batches = append(batches, nodes[i:end])
	}

	return batches
}

// CalculateBatches 计算批次信息
func (s *BatchScheduler) CalculateBatches(totalNodes, batchSize int) (totalBatches int, err error) {
	if batchSize <= 0 {
		return 0, fmt.Errorf("batch size must be greater than 0")
	}

	if totalNodes <= 0 {
		return 0, fmt.Errorf("total nodes must be greater than 0")
	}

	totalBatches = (totalNodes + batchSize - 1) / batchSize
	return totalBatches, nil
}
