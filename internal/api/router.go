package api

import (
	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/api/handler"
	"github.com/kitsnail/ips/internal/api/middleware"
	"github.com/kitsnail/ips/internal/service"
	"github.com/sirupsen/logrus"
)

// SetupRouter 设置路由
func SetupRouter(logger *logrus.Logger, taskManager *service.TaskManager) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 全局中间件
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.LoggingMiddleware(logger))

	// 健康检查处理器
	healthHandler := handler.NewHealthHandler()

	// 健康检查端点（不需要认证）
	router.GET("/healthz", healthHandler.HealthCheck)
	router.GET("/readyz", healthHandler.ReadyCheck)

	// 任务处理器
	taskHandler := handler.NewTaskHandler(taskManager)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.DELETE("/tasks/:id", taskHandler.CancelTask)
	}

	return router
}
