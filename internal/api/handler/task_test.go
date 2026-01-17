package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func setupTestHandler() (*TaskHandler, *gin.Engine) {
	// 创建 fake K8s clientset
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node-1",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
		Namespace: "default",
	}

	// 创建依赖
	repo := repository.NewMemoryRepository()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 测试时减少日志输出

	jobCreator := k8s.NewJobCreator(k8sClient, "busybox:latest")
	nodeFilter := service.NewNodeFilter(k8sClient)
	batchScheduler := service.NewBatchScheduler(jobCreator, logger)
	statusTracker := service.NewStatusTracker(repo, jobCreator, logger)

	taskManager := service.NewTaskManager(
		repo,
		nodeFilter,
		batchScheduler,
		statusTracker,
		logger,
	)

	handler := NewTaskHandler(taskManager)

	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, router
}

func TestTaskHandler_CreateTask(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)

	reqBody := models.CreateTaskRequest{
		Images:    []string{"nginx:latest"},
		BatchSize: 10,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	if task.ID == "" {
		t.Error("Expected task ID, got empty string")
	}

	if task.Status != models.TaskPending {
		t.Errorf("Expected status %s, got %s", models.TaskPending, task.Status)
	}
}

func TestTaskHandler_CreateTask_InvalidRequest(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)

	// 发送无效的 JSON
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTaskHandler_GetTask(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)
	router.GET("/api/v1/tasks/:id", handler.GetTask)

	// 先创建一个任务
	reqBody := models.CreateTaskRequest{
		Images:    []string{"redis:7"},
		BatchSize: 5,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdTask models.Task
	json.Unmarshal(w.Body.Bytes(), &createdTask)

	// 获取任务详情
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+createdTask.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var task models.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	if task.ID != createdTask.ID {
		t.Errorf("Expected task ID %s, got %s", createdTask.ID, task.ID)
	}
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/api/v1/tasks/:id", handler.GetTask)

	req := httptest.NewRequest("GET", "/api/v1/tasks/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestTaskHandler_ListTasks(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)
	router.GET("/api/v1/tasks", handler.ListTasks)

	// 创建多个任务
	for i := 0; i < 3; i++ {
		reqBody := models.CreateTaskRequest{
			Images:    []string{"nginx:latest"},
			BatchSize: 10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 等待任务创建完成
	time.Sleep(100 * time.Millisecond)

	// 列出任务
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		Tasks  []models.Task `json:"tasks"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 3 {
		t.Errorf("Expected total 3, got %d", resp.Total)
	}

	if len(resp.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(resp.Tasks))
	}
}

func TestTaskHandler_ListTasks_WithFilter(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/api/v1/tasks", handler.ListTasks)

	// 列出任务，带分页参数
	req := httptest.NewRequest("GET", "/api/v1/tasks?limit=5&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		Tasks  []models.Task `json:"tasks"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Limit != 5 {
		t.Errorf("Expected limit 5, got %d", resp.Limit)
	}

	if resp.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", resp.Offset)
	}
}

func TestTaskHandler_CancelTask(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)
	router.DELETE("/api/v1/tasks/:id", handler.CancelTask)

	// 创建任务
	reqBody := models.CreateTaskRequest{
		Images:    []string{"mysql:8"},
		BatchSize: 10,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdTask models.Task
	json.Unmarshal(w.Body.Bytes(), &createdTask)

	// 取消任务
	req = httptest.NewRequest("DELETE", "/api/v1/tasks/"+createdTask.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %v", resp["status"])
	}
}

func TestTaskHandler_CancelTask_NotFound(t *testing.T) {
	handler, router := setupTestHandler()
	router.DELETE("/api/v1/tasks/:id", handler.CancelTask)

	req := httptest.NewRequest("DELETE", "/api/v1/tasks/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
