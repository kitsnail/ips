package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware 请求日志中间件
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(startTime)

		// 记录日志
		entry := logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
		})

		if c.Writer.Status() >= 500 {
			entry.Error("Server error")
		} else if c.Writer.Status() >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request completed")
		}
	}
}
