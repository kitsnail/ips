package api

import (
	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/api/handler"
	"github.com/kitsnail/ips/internal/api/middleware"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// SetupRouter 设置路由
func SetupRouter(logger *logrus.Logger, taskManager *service.TaskManager, scheduledTaskManager *service.ScheduledTaskManager, authService *service.AuthService, userRepo repository.UserRepository, libraryRepo repository.LibraryRepository, secretRepo repository.SecretRegistryRepository) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 全局中间件
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.PrometheusMiddleware())

	// 静态文件服务 (Web UI)
	router.Static("/web", "./web/static")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/web/")
	})

	// 健康检查处理器
	healthHandler := handler.NewHealthHandler()

	// 健康检查端点（不需要认证）
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/healthz", healthHandler.HealthCheck)
	router.GET("/readyz", healthHandler.ReadyCheck)

	// Prometheus 指标端点
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 任务处理器
	taskHandler := handler.NewTaskHandler(taskManager)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	libraryHandler := handler.NewLibraryHandler(libraryRepo)
	secretHandler := handler.NewSecretHandler(secretRepo)
	scheduledTaskHandler := handler.NewScheduledTaskHandler(scheduledTaskManager)

	// 登录接口 (公开)
	router.POST("/api/v1/login", authHandler.Login)

	// API v1 路由组 (受保护)
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(authService))
	{
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.DELETE("/tasks/:id", taskHandler.DeleteTask)

		// 镜像库
		v1.GET("/library", libraryHandler.ListImages)
		v1.POST("/library", libraryHandler.SaveImage)
		v1.DELETE("/library/:id", libraryHandler.DeleteImage)

		// 私有仓库认证
		v1.GET("/secrets", secretHandler.ListSecrets)
		v1.POST("/secrets", secretHandler.CreateSecret)
		v1.GET("/secrets/:id", secretHandler.GetSecret)
		v1.PUT("/secrets/:id", secretHandler.UpdateSecret)
		v1.DELETE("/secrets/:id", secretHandler.DeleteSecret)

		// 修改密码 (所有登录用户都可调用，Handler 内部做权限校验)
		v1.PUT("/users/:id", userHandler.UpdateUser)

		// 用户管理 (仅限管理员)
		users := v1.Group("/users")
		users.Use(middleware.AdminOnly())
		{
			users.GET("", userHandler.ListUsers)
			users.POST("", userHandler.CreateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}

		// 定时任务管理 (仅限管理员)
		scheduledTasks := v1.Group("/scheduled-tasks")
		scheduledTasks.Use(middleware.AdminOnly())
		{
			scheduledTasks.POST("", scheduledTaskHandler.CreateScheduledTask)
			scheduledTasks.GET("", scheduledTaskHandler.ListScheduledTasks)
			scheduledTasks.GET("/:id", scheduledTaskHandler.GetScheduledTask)
			scheduledTasks.PUT("/:id", scheduledTaskHandler.UpdateScheduledTask)
			scheduledTasks.DELETE("/:id", scheduledTaskHandler.DeleteScheduledTask)
			scheduledTasks.PUT("/:id/enable", scheduledTaskHandler.EnableTask)
			scheduledTasks.PUT("/:id/disable", scheduledTaskHandler.DisableTask)
			scheduledTasks.POST("/:id/trigger", scheduledTaskHandler.TriggerTask)
			scheduledTasks.GET("/:id/executions", scheduledTaskHandler.ListExecutions)
			scheduledTasks.GET("/:id/executions/:executionId", scheduledTaskHandler.GetExecution)
		}

		return router
	}

	return router
}
