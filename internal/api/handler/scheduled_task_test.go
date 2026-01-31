package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScheduledTaskHandler(t *testing.T) (*ScheduledTaskHandler, *gin.Engine, *repository.SQLiteRepository) {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	taskManager := service.NewTaskManager(repo, repo, nil, nil, nil, logger)

	scheduledTaskManager := service.NewScheduledTaskManager(repo, repo, taskManager, logger)

	handler := NewScheduledTaskHandler(scheduledTaskManager)
	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, router, repo
}

func TestScheduledTaskHandler_CreateScheduledTask(t *testing.T) {
	handler, router, _ := setupScheduledTaskHandler(t)
	router.POST("/scheduled-tasks", handler.CreateScheduledTask)

	reqBody := models.CreateScheduledTaskRequest{
		Name:     "Test Scheduled Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images:    []string{"nginx:latest"},
			BatchSize: 10,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/scheduled-tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.ScheduledTask
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, reqBody.Name, response.Name)
	assert.Equal(t, reqBody.CronExpr, response.CronExpr)
}

func TestScheduledTaskHandler_CreateScheduledTask_InvalidRequest(t *testing.T) {
	handler, router, _ := setupScheduledTaskHandler(t)
	router.POST("/scheduled-tasks", handler.CreateScheduledTask)

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
	}{
		{
			name:       "Missing name",
			reqBody:    `{"cronExpr":"0 0 * * *","taskConfig":{"images":["nginx:latest"]}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing cronExpr",
			reqBody:    `{"name":"Test","taskConfig":{"images":["nginx:latest"]}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/scheduled-tasks", bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestScheduledTaskHandler_GetScheduledTask(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.GET("/scheduled-tasks/:id", handler.GetScheduledTask)

	task := &models.ScheduledTask{
		ID:       "task-get-test",
		Name:     "Get Test",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/scheduled-tasks/task-get-test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ScheduledTask
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, response.ID)
	assert.Equal(t, task.Name, response.Name)
}

func TestScheduledTaskHandler_GetScheduledTask_NotFound(t *testing.T) {
	handler, router, _ := setupScheduledTaskHandler(t)
	router.GET("/scheduled-tasks/:id", handler.GetScheduledTask)

	req, _ := http.NewRequest("GET", "/scheduled-tasks/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestScheduledTaskHandler_ListScheduledTasks(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.GET("/scheduled-tasks", handler.ListScheduledTasks)

	for i := 1; i <= 3; i++ {
		task := &models.ScheduledTask{
			ID:       "task-list-test-" + string(rune('0'+i)),
			Name:     "List Test " + string(rune('0'+i)),
			CronExpr: "0 0 * * *",
			Enabled:  true,
			TaskConfig: models.TaskConfig{
				Images: []string{"nginx:latest"},
			},
			CreatedBy: "test-user",
		}
		err := repo.CreateScheduledTask(context.Background(), task)
		require.NoError(t, err)
	}

	req, _ := http.NewRequest("GET", "/scheduled-tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ListScheduledTasksResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 3, response.Total)
	assert.Len(t, response.Tasks, 3)
}

func TestScheduledTaskHandler_DeleteScheduledTask(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.DELETE("/scheduled-tasks/:id", handler.DeleteScheduledTask)

	task := &models.ScheduledTask{
		ID:       "task-delete-test",
		Name:     "Delete Test",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	req, _ := http.NewRequest("DELETE", "/scheduled-tasks/task-delete-test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
}

func TestScheduledTaskHandler_EnableTask(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.PUT("/scheduled-tasks/:id/enable", handler.EnableTask)

	task := &models.ScheduledTask{
		ID:       "task-enable-test",
		Name:     "Enable Test",
		CronExpr: "0 0 * * *",
		Enabled:  false,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	_, err = repo.GetScheduledTask(context.Background(), "task-enable-test")
	require.NoError(t, err)

	req, _ := http.NewRequest("PUT", "/scheduled-tasks/task-enable-test/enable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	enabledTask, _ := repo.GetScheduledTask(context.Background(), "task-enable-test")
	assert.True(t, enabledTask.Enabled)
}

func TestScheduledTaskHandler_DisableTask(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.PUT("/scheduled-tasks/:id/disable", handler.DisableTask)

	task := &models.ScheduledTask{
		ID:       "task-disable-test",
		Name:     "Disable Test",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	req, _ := http.NewRequest("PUT", "/scheduled-tasks/task-disable-test/disable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	disabledTask, _ := repo.GetScheduledTask(context.Background(), "task-disable-test")
	assert.False(t, disabledTask.Enabled)
}

func TestScheduledTaskHandler_TriggerTask(t *testing.T) {
	handler, router, repo := setupScheduledTaskHandler(t)
	router.POST("/scheduled-tasks/:id/trigger", handler.TriggerTask)

	task := &models.ScheduledTask{
		ID:       "task-trigger-test",
		Name:     "Trigger Test",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		OverlapPolicy:  models.OverlapPolicySkip,
		TimeoutSeconds: 0,
		CreatedBy:      "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	req, _ := http.NewRequest("POST", "/scheduled-tasks/task-trigger-test/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func stringPtr(s string) *string {
	return &s
}
