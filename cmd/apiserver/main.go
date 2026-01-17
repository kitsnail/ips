package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kitsnail/ips/internal/api"
	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化日志
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting Image Prewarm Service...")

	// 1. 初始化K8s客户端
	namespace := os.Getenv("K8S_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	k8sClient, err := k8s.NewClient(namespace)
	if err != nil {
		logger.Fatalf("Failed to create K8s client: %v", err)
	}
	logger.Info("K8s client initialized")

	// 2. 初始化存储层
	repo := repository.NewMemoryRepository()
	logger.Info("Repository initialized")

	// 3. 初始化服务组件
	workerImage := os.Getenv("WORKER_IMAGE")
	if workerImage == "" {
		workerImage = "registry.k8s.io/pause:3.10"
	}

	jobCreator := k8s.NewJobCreator(k8sClient, workerImage)
	nodeFilter := service.NewNodeFilter(k8sClient)
	batchScheduler := service.NewBatchScheduler(jobCreator, logger)
	statusTracker := service.NewStatusTracker(repo, jobCreator, logger)

	logger.Info("Service components initialized")

	// 4. 初始化任务管理器
	taskManager := service.NewTaskManager(
		repo,
		nodeFilter,
		batchScheduler,
		statusTracker,
		logger,
	)
	logger.Info("Task manager initialized")

	// 5. 设置路由
	router := api.SetupRouter(logger, taskManager)

	// 6. 创建HTTP服务器
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 启动服务器（在goroutine中）
	go func() {
		logger.Infof("Server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭，设置5秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}
