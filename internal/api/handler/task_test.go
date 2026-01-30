package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func setupTestHandler() (*TaskHandler, *gin.Engine) {
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

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	jobCreator := k8s.NewJobCreator(k8sClient, "busybox:latest", "crictl:v1.31.0", "/run/containerd/containerd.sock")
	nodeFilter := service.NewNodeFilter(k8sClient)
	batchScheduler := service.NewBatchScheduler(jobCreator, logger)
	statusTracker := service.NewStatusTracker(repository.NewMemoryRepository(), jobCreator, logger)
	taskManager := service.NewTaskManager(repository.NewMemoryRepository(), repository.NewMemoryRepository(), nodeFilter, batchScheduler, statusTracker, logger)

	handler := NewTaskHandler(taskManager)
	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, router
}

func TestTaskHandler_CreateTask_PrivateRegistry_MissingFields(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/api/v1/tasks", handler.CreateTask)

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
	}{
		{
			name:       "提供了 registry 但缺少 username",
			reqBody:    `{"images":["nginx:latest"],"registry":"harbor.example.com","batchSize":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "提供了 registry 但缺少 password",
			reqBody:    `{"images":["nginx:latest"],"registry":"harbor.example.com","username":"testuser","batchSize":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "提供了 username 但缺少 registry 和 password",
			reqBody:    `{"images":["nginx:latest"],"username":"testuser","batchSize":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "提供了 password 但缺少 registry 和 username",
			reqBody:    `{"images":["nginx:latest"],"password":"testpass","batchSize":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "提供了完整的私有仓库凭证",
			reqBody:    `{"images":["nginx:latest"],"registry":"harbor.example.com","username":"testuser","password":"testpass","batchSize":10}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(tt.reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != tt.wantStatus {
			t.Errorf("Test %s: Expected status %d, got %d", tt.name, tt.wantStatus, w.Code)
		}
	}
}
