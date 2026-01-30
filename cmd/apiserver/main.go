package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kitsnail/ips/internal/api"
	"github.com/kitsnail/ips/internal/k8s"
	"github.com/kitsnail/ips/internal/puller"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/kitsnail/ips/pkg/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Check for version flag
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version.Info())
		return
	}

	// 检查是否是辅助拉取命令
	if len(os.Args) > 1 && os.Args[1] == "pull" {
		imagesStr := os.Getenv("IMAGES")
		socketPath := os.Getenv("CRI_SOCKET_PATH")
		if imagesStr == "" || socketPath == "" {
			fmt.Println("Missing IMAGES or CRI_SOCKET_PATH environment variables")
			os.Exit(1)
		}
		images := strings.Split(imagesStr, ",")
		puller.Run(images, socketPath)
		return
	}

	// 初始化日志
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetLevel(logrus.InfoLevel)

	logger.Infof("Starting Image Prewarm Service... %s", version.Info())

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

	// 2. 初始化存储层 (SQLite)
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "ips.db"
	}
	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		logger.Fatalf("Failed to initialize SQLite repository: %v", err)
	}
	logger.Infof("SQLite Repository initialized at %s", dbPath)

	// 初始化管理员用户
	ctx := context.Background()
	admin, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		logger.Errorf("Failed to check admin user: %v", err)
	} else if admin == nil {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		err = repo.CreateUser(ctx, &models.User{
			Username: "admin",
			Password: string(hashedPassword),
			Role:     models.RoleAdmin,
		})
		if err != nil {
			logger.Errorf("Failed to create default admin user: %v", err)
		} else {
			logger.Info("Default admin user created (admin/admin123)")
		}
	}

	// 3. 初始化服务组件
	workerImage := os.Getenv("WORKER_IMAGE")
	if workerImage == "" {
		workerImage = "registry.k8s.io/pause:3.10"
	}

	pullerImage := os.Getenv("PULLER_IMAGE")
	if pullerImage == "" {
		pullerImage = "registry.k8s.io/build-containers/crictl:v1.31.0"
	}

	criSocketPath := os.Getenv("CRI_SOCKET_PATH")
	if criSocketPath == "" {
		criSocketPath = "/run/containerd/containerd.sock"
	}

	jobCreator := k8s.NewJobCreator(k8sClient, workerImage, pullerImage, criSocketPath)
	nodeFilter := service.NewNodeFilter(k8sClient)
	batchScheduler := service.NewBatchScheduler(jobCreator, logger)
	statusTracker := service.NewStatusTracker(repo, jobCreator, logger)

	logger.Info("Service components initialized")

	// 4. 初始化认证服务
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ips-default-secret-change-me"
		logger.Warn("JWT_SECRET not set, using default secret")
	}
	authService := service.NewAuthService(repo, k8sClient.Clientset, jwtSecret)

	// 5. 初始化任务管理器
	taskManager := service.NewTaskManager(
		repo,
		repo,
		nodeFilter,
		batchScheduler,
		statusTracker,
		logger,
	)
	logger.Info("Task manager initialized")

	// 6. 设置路由
	router := api.SetupRouter(logger, taskManager, authService, repo, repo, repo)

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
