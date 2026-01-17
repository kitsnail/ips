package service

import (
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryStrategy 重试策略
type RetryStrategy interface {
	// CalculateDelay 计算重试延迟
	CalculateDelay(retryCount int, baseDelay int) time.Duration
}

// LinearRetryStrategy 线性重试策略
// 每次重试等待时间固定
type LinearRetryStrategy struct {
	logger *logrus.Logger
}

// NewLinearRetryStrategy 创建线性重试策略
func NewLinearRetryStrategy(logger *logrus.Logger) *LinearRetryStrategy {
	return &LinearRetryStrategy{
		logger: logger,
	}
}

// CalculateDelay 计算延迟（固定延迟）
func (s *LinearRetryStrategy) CalculateDelay(retryCount int, baseDelay int) time.Duration {
	return time.Duration(baseDelay) * time.Second
}

// ExponentialRetryStrategy 指数退避重试策略
// 延迟时间随重试次数指数增长
type ExponentialRetryStrategy struct {
	logger *logrus.Logger
}

// NewExponentialRetryStrategy 创建指数退避重试策略
func NewExponentialRetryStrategy(logger *logrus.Logger) *ExponentialRetryStrategy {
	return &ExponentialRetryStrategy{
		logger: logger,
	}
}

// CalculateDelay 计算延迟（指数增长）
// delay = baseDelay * 2^(retryCount-1)
func (s *ExponentialRetryStrategy) CalculateDelay(retryCount int, baseDelay int) time.Duration {
	if retryCount <= 0 {
		return time.Duration(baseDelay) * time.Second
	}

	// 指数退避: baseDelay * 2^(retryCount-1)
	multiplier := math.Pow(2, float64(retryCount-1))
	delay := float64(baseDelay) * multiplier

	// 设置最大延迟时间（10分钟）
	maxDelay := 600.0
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Duration(delay) * time.Second
}

// GetRetryStrategy 根据策略名称获取重试策略
func GetRetryStrategy(strategyName string, logger *logrus.Logger) RetryStrategy {
	switch strategyName {
	case "exponential":
		return NewExponentialRetryStrategy(logger)
	case "linear":
		fallthrough
	default:
		return NewLinearRetryStrategy(logger)
	}
}
