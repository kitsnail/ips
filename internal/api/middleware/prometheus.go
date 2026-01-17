package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/pkg/metrics"
)

// PrometheusMiddleware Prometheus 监控中间件
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start).Seconds()
		method := c.Request.Method
		endpoint := c.FullPath()
		statusCode := strconv.Itoa(c.Writer.Status())

		metrics.APIRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)
		metrics.APIRequestTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	}
}
