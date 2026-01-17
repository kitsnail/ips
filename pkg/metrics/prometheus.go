package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TasksTotal 任务总数（按状态分类）
	TasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ips_tasks_total",
			Help: "Total number of image prewarming tasks by status",
		},
		[]string{"status"},
	)

	// TaskDuration 任务耗时直方图
	TaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ips_task_duration_seconds",
			Help:    "Duration of image prewarming tasks in seconds",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 1200, 1800, 3600}, // 10s, 30s, 1m, 2m, 5m, 10m, 20m, 30m, 1h
		},
		[]string{"status"},
	)

	// NodesProcessed 处理的节点数
	NodesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ips_nodes_processed_total",
			Help: "Total number of nodes processed for image prewarming",
		},
		[]string{"status"}, // success, failed
	)

	// ActiveTasks 当前活跃任务数
	ActiveTasks = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ips_active_tasks",
			Help: "Current number of active image prewarming tasks",
		},
	)

	// APIRequestDuration API 请求耗时
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ips_api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// APIRequestTotal API 请求总数
	APIRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ips_api_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// BatchExecutionDuration 批次执行耗时
	BatchExecutionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ips_batch_execution_duration_seconds",
			Help:    "Duration of batch execution in seconds",
			Buckets: []float64{5, 10, 30, 60, 120, 300, 600}, // 5s, 10s, 30s, 1m, 2m, 5m, 10m
		},
	)

	// JobCreationTotal Job 创建总数
	JobCreationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ips_job_creation_total",
			Help: "Total number of Kubernetes jobs created",
		},
		[]string{"status"}, // success, failed
	)

	// ImagesPulled 拉取的镜像数
	ImagesPulled = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ips_images_pulled_total",
			Help: "Total number of images pulled across all nodes",
		},
	)
)
