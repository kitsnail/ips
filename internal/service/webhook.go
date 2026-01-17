package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
)

// WebhookNotifier Webhook 通知器
type WebhookNotifier struct {
	client  *http.Client
	logger  *logrus.Logger
	timeout time.Duration
}

// NewWebhookNotifier 创建 Webhook 通知器
func NewWebhookNotifier(logger *logrus.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:  logger,
		timeout: 10 * time.Second,
	}
}

// WebhookPayload Webhook 通知的 payload
type WebhookPayload struct {
	Event     string        `json:"event"`     // 事件类型: task.completed, task.failed, task.cancelled
	Task      *models.Task  `json:"task"`      // 任务信息
	Timestamp time.Time     `json:"timestamp"` // 通知时间
	Message   string        `json:"message"`   // 消息描述
}

// Notify 发送 Webhook 通知
func (w *WebhookNotifier) Notify(ctx context.Context, task *models.Task, event, message string) error {
	if task.WebhookURL == "" {
		// 没有配置 Webhook，跳过
		return nil
	}

	payload := WebhookPayload{
		Event:     event,
		Task:      task,
		Timestamp: time.Now(),
		Message:   message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// 异步发送，带重试
	go w.sendWithRetry(task.WebhookURL, jsonData, 3)

	return nil
}

// sendWithRetry 发送 Webhook 请求，支持重试
func (w *WebhookNotifier) sendWithRetry(url string, payload []byte, maxRetries int) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := w.send(url, payload)
		if err == nil {
			w.logger.WithFields(logrus.Fields{
				"url":     url,
				"attempt": attempt,
			}).Info("Webhook notification sent successfully")
			return
		}

		w.logger.WithFields(logrus.Fields{
			"url":     url,
			"attempt": attempt,
			"error":   err,
		}).Warn("Failed to send webhook notification")

		if attempt < maxRetries {
			// 指数退避
			delay := time.Duration(attempt) * 2 * time.Second
			time.Sleep(delay)
		}
	}

	w.logger.WithFields(logrus.Fields{
		"url":        url,
		"maxRetries": maxRetries,
	}).Error("Webhook notification failed after all retries")
}

// send 发送 HTTP POST 请求
func (w *WebhookNotifier) send(url string, payload []byte) error {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		url,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "IPS-Webhook/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// NotifyTaskCompleted 通知任务完成
func (w *WebhookNotifier) NotifyTaskCompleted(ctx context.Context, task *models.Task) error {
	return w.Notify(ctx, task, "task.completed", "Task completed successfully")
}

// NotifyTaskFailed 通知任务失败
func (w *WebhookNotifier) NotifyTaskFailed(ctx context.Context, task *models.Task) error {
	return w.Notify(ctx, task, "task.failed", "Task failed after all retries")
}

// NotifyTaskCancelled 通知任务取消
func (w *WebhookNotifier) NotifyTaskCancelled(ctx context.Context, task *models.Task) error {
	return w.Notify(ctx, task, "task.cancelled", "Task was cancelled")
}
