package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/kitsnail/ips/pkg/models"
)

// TaskHandler 任务处理器
type TaskHandler struct {
	taskManager *service.TaskManager
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskManager *service.TaskManager) *TaskHandler {
	return &TaskHandler{
		taskManager: taskManager,
	}
}

// CreateTask 创建任务
// @Summary 创建镜像预热任务
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 创建任务
	task, err := h.taskManager.CreateTask(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Router /api/v1/tasks/:id [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.taskManager.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":  "Task not found",
				"taskId": taskID,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks 列出任务
// ListTasks 列出任务
// @Summary 列出所有任务
// @Router /api/v1/tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
	// 解析查询参数
	var req models.ListTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// TODO: Implement Status filtering in repository layer if needed
	// For now, we only support pagination

	// 查询任务列表
	tasks, total, err := h.taskManager.ListTasks(c.Request.Context(), req.Offset, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list tasks",
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

// DeleteTask 删除或取消任务
// @Summary 删除或取消任务
// @Router /api/v1/tasks/:id [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")

	action, err := h.taskManager.DeleteTask(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete/cancel task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"status":  "success",
		"action":  action,
		"message": fmt.Sprintf("Task %s successfully", action),
	})
}

// parseIntParam 解析整数参数
func parseIntParam(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}
