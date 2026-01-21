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
func SetupRouter(logger *logrus.Logger, taskManager *service.TaskManager, authService *service.AuthService, userRepo repository.UserRepository) *gin.Engine {
	// 设置Gin模式
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

	// 登录接口 (公开)
	router.POST("/api/v1/login", authHandler.Login)

	// API v1 路由组 (受保护)
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(authService))
	{
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.DELETE("/tasks/:id", taskHandler.CancelTask)

		// 用户管理 (仅限管理员)
		users := v1.Group("/users")
		users.Use(middleware.AdminOnly())
		{
			users.GET("", userHandler.ListUsers)
			users.POST("", userHandler.CreateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	return router
}
