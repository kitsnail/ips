package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/service"
	"github.com/kitsnail/ips/pkg/models"
)

type ScheduledTaskHandler struct {
	scheduledTaskManager *service.ScheduledTaskManager
}

func NewScheduledTaskHandler(scheduledTaskManager *service.ScheduledTaskManager) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{
		scheduledTaskManager: scheduledTaskManager,
	}
}

func (h *ScheduledTaskHandler) CreateScheduledTask(c *gin.Context) {
	var req models.CreateScheduledTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if req.Enabled == false {
		req.Enabled = true
	}

	if req.OverlapPolicy == "" {
		req.OverlapPolicy = models.OverlapPolicySkip
	}

	taskID := models.GenerateTaskID()
	task := &models.ScheduledTask{
		ID:             taskID,
		Name:           req.Name,
		Description:    req.Description,
		CronExpr:       req.CronExpr,
		Enabled:        req.Enabled,
		TaskConfig:     req.TaskConfig,
		OverlapPolicy:  req.OverlapPolicy,
		TimeoutSeconds: req.TimeoutSeconds,
		CreatedBy:      c.GetString("username"),
	}

	if err := h.scheduledTaskManager.CreateScheduledTask(context.Background(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create scheduled task in database",
			"details": err.Error(),
		})
		return
	}

	if task.Enabled {
		if err := h.scheduledTaskManager.AddTask(task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to add scheduled task to scheduler",
				"details": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusCreated, task)
}

func (h *ScheduledTaskHandler) GetScheduledTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.scheduledTaskManager.GetScheduledTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Scheduled task not found",
			"taskId": taskID,
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *ScheduledTaskHandler) ListScheduledTasks(c *gin.Context) {
	var req models.ListScheduledTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	tasks, total, err := h.scheduledTaskManager.ListScheduledTasks(c.Request.Context(), req.Offset, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list scheduled tasks",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

func (h *ScheduledTaskHandler) UpdateScheduledTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.scheduledTaskManager.GetScheduledTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Scheduled task not found",
			"taskId": taskID,
		})
		return
	}

	var req models.UpdateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.CronExpr != nil {
		task.CronExpr = *req.CronExpr
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}
	if req.TaskConfig != nil {
		task.TaskConfig = *req.TaskConfig
	}
	if req.OverlapPolicy != nil {
		task.OverlapPolicy = *req.OverlapPolicy
	}
	if req.TimeoutSeconds != nil {
		task.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := h.scheduledTaskManager.RemoveTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update scheduled task",
			"details": err.Error(),
		})
		return
	}

	if task.Enabled {
		if err := h.scheduledTaskManager.AddTask(task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to add scheduled task to scheduler",
				"details": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, task)
}

func (h *ScheduledTaskHandler) DeleteScheduledTask(c *gin.Context) {
	taskID := c.Param("id")

	if err := h.scheduledTaskManager.RemoveTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete scheduled task",
			"details": err.Error(),
		})
		return
	}

	if err := h.scheduledTaskManager.DeleteScheduledTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete scheduled task from database",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"status":  "success",
		"message": "Scheduled task deleted successfully",
	})
}

func (h *ScheduledTaskHandler) EnableTask(c *gin.Context) {
	taskID := c.Param("id")

	if err := h.scheduledTaskManager.EnableTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enable scheduled task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"status":  "success",
		"message": "Scheduled task enabled successfully",
	})
}

func (h *ScheduledTaskHandler) DisableTask(c *gin.Context) {
	taskID := c.Param("id")

	if err := h.scheduledTaskManager.DisableTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to disable scheduled task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"status":  "success",
		"message": "Scheduled task disabled successfully",
	})
}

func (h *ScheduledTaskHandler) TriggerTask(c *gin.Context) {
	taskID := c.Param("id")

	taskID, err := h.scheduledTaskManager.TriggerTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to trigger scheduled task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"status":  "success",
		"message": "Scheduled task triggered successfully",
	})
}

func (h *ScheduledTaskHandler) ListExecutions(c *gin.Context) {
	taskID := c.Param("id")

	var req models.ListScheduledExecutionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	executions, total, err := h.scheduledTaskManager.ListExecutions(c.Request.Context(), taskID, req.Offset, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list executions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      total,
		"limit":      req.Limit,
		"offset":     req.Offset,
	})
}

func (h *ScheduledTaskHandler) GetExecution(c *gin.Context) {
	idStr := c.Param("executionId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid execution ID",
			"details": err.Error(),
		})
		return
	}

	execution, err := h.scheduledTaskManager.GetExecution(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":       "Execution not found",
			"executionId": id,
		})
		return
	}

	c.JSON(http.StatusOK, execution)
}
