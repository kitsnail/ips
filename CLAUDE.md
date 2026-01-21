# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Image Prewarm Service (IPS) is a RESTful API service built in Go that pre-warms container images across Kubernetes clusters. It creates Kubernetes Jobs on target nodes to pull images, with features including batch scheduling, priority queues, retry mechanisms, webhook notifications, and Prometheus metrics.

**Core technology stack:**
- Go 1.23+
- Gin web framework
- Kubernetes client-go (v0.29.0)
- Prometheus metrics
- Logrus for structured logging

## Architecture

### Component Structure

The service follows a layered architecture:

1. **API Layer** (`internal/api/`)
   - HTTP handlers for task operations (create, get, list, cancel)
   - Middleware: logging, recovery, Prometheus metrics
   - Router setup with Gin

2. **Service Layer** (`internal/service/`)
   - **TaskManager**: Orchestrates task lifecycle with priority queue and concurrency control (max 3 concurrent tasks by default)
   - **BatchScheduler**: Splits nodes into batches and creates Kubernetes Jobs sequentially
   - **StatusTracker**: Uses Kubernetes Watch API to track Job status in real-time
   - **NodeFilter**: Filters nodes based on selector labels and readiness
   - **WebhookNotifier**: Sends HTTP notifications for task completion/failure/cancellation
   - **Retry strategies**: Linear and exponential backoff

3. **Repository Layer** (`internal/repository/`)
   - In-memory storage implementation (thread-safe with sync.RWMutex)
   - Interface-based design for future persistence options

4. **Kubernetes Layer** (`internal/k8s/`)
   - **Client**: Wrapper around kubernetes clientset (auto-detects in-cluster vs kubeconfig)
   - **JobCreator**: Creates Jobs with initContainers to pull target images
   - **Node operations**: List and filter nodes

5. **Models** (`pkg/models/`)
   - Task: Core data model with status, progress, priority, retry config, webhook URL
   - Progress: Tracks node completion percentage and batch progress
   - Request/Response DTOs

### Task Execution Flow

1. API receives CreateTask request → TaskManager validates and saves task with priority
2. Task queued in priority queue (1-10, higher = more urgent)
3. TaskManager acquires concurrency slot (semaphore-based)
4. NodeFilter gets matching nodes → BatchScheduler splits into batches
5. For each batch: JobCreator creates Kubernetes Jobs with initContainers pulling images
6. StatusTracker watches Job events and updates task progress
7. On failure: retry logic kicks in (linear/exponential backoff) if retries remain
8. On completion/failure/cancellation: WebhookNotifier sends notification if configured

### Kubernetes Job Design

Jobs use **initContainers** to pull target images:
- Each image gets its own initContainer with `ImagePullPolicy: Always`
- Main container uses worker image (default: `registry.k8s.io/pause:3.10`)
- NodeSelector pins Job to specific node
- Tolerates all taints to reach any node
- TTL: 900 seconds (auto-cleanup after completion)

## Development Commands

### Build and Run
```bash
make build               # Build binary to bin/apiserver
make run                 # Build and start server (port 8080)
make clean               # Remove build artifacts
```

### Testing
```bash
make test                # Run all tests with race detector and coverage
make test-coverage       # Generate HTML coverage report
./test-api.sh            # End-to-end API integration tests
```

### Code Quality
```bash
make fmt                 # Format code with go fmt
make lint                # Run go vet and golangci-lint
make tidy                # Tidy go.mod dependencies
```

### Docker
```bash
make docker-build        # Build Docker image
make docker-run          # Run containerized service (mounts ~/.kube/config)
make docker-stop         # Stop and remove container
docker-compose up -d     # Start with Docker Compose
```

### Kubernetes Deployment
```bash
make k8s-deploy          # Deploy to ips-system namespace
make k8s-status          # View deployment status
make k8s-logs            # Stream logs from all pods
make k8s-port-forward    # Forward port 8080 to localhost
make k8s-restart         # Rolling restart
make k8s-delete          # Remove all resources
```

**Important**: The service requires RBAC permissions to:
- List nodes
- Create/watch/delete Jobs in the configured namespace
- Get Job events

Deployment config is in `deploy/` with namespace, RBAC, ConfigMap, Deployment, Service, and HPA.

## Configuration

Environment variables:
- `SERVER_PORT`: HTTP port (default: 8080)
- `K8S_NAMESPACE`: Namespace for Jobs (default: default)
- `WORKER_IMAGE`: Worker container image (default: registry.k8s.io/pause:3.10)

## Testing

### Unit Tests
Most service components have `*_test.go` files:
- `memory_test.go`: Repository operations
- `batch_scheduler_test.go`: Batch splitting logic
- `priority_queue_test.go`: Priority sorting
- `task_test.go`: Handler tests with httptest

Run specific test:
```bash
go test -v -run TestFunctionName ./internal/service/
```

### Integration Tests
`test-api.sh` covers full workflow:
1. Health check
2. Create task
3. Poll status until completion
4. List/filter tasks
5. Cancel task

## Key Considerations

### Concurrency Control
- TaskManager uses semaphore to limit concurrent tasks (default: 3)
- Prevents API server overload from too many simultaneous Job creations
- Queued tasks wait for slot availability

### Retry Mechanism
- Tasks can retry on failure (max 5 retries)
- Linear strategy: fixed delay between retries
- Exponential strategy: delay doubles each retry
- Retries check if task was cancelled before re-execution

### Status Tracking
- StatusTracker uses Kubernetes Watch API for real-time updates
- Tracks Job conditions: Complete/Failed
- Records failed node details with timestamp and reason
- Watch timeout: 5 minutes

### Metrics
Service exposes Prometheus metrics on `/metrics`:
- `ips_tasks_total`: Counter by status
- `ips_active_tasks`: Gauge of running tasks
- `ips_task_duration_seconds`: Histogram by status
- `ips_nodes_processed_total`: Counter by result (success/failed)
- `ips_batch_execution_duration_seconds`: Histogram of batch timing
- `ips_job_creation_total`: Counter by result
- `ips_images_pulled_total`: Total images successfully pulled

### Web UI
Static web interface served at `/` and `/web/`:
- Built with vanilla HTML/CSS/JavaScript
- Polls `/api/v1/tasks` every 5 seconds
- Displays task list, create form, status filtering
- Located in `web/` directory

## Common Development Patterns

### Adding a New API Endpoint
1. Define handler in `internal/api/handler/`
2. Add route in `internal/api/router.go` (use middleware chain)
3. Update TaskManager method if needed
4. Add test in `*_test.go`

### Adding a New Metric
1. Define in `pkg/metrics/prometheus.go`
2. Register in `init()` function
3. Instrument code with `.Inc()`, `.Observe()`, etc.

### Extending Storage
Implement `repository.TaskRepository` interface for persistence (e.g., Redis, PostgreSQL).

### Custom Retry Strategy
Implement `service.RetryStrategy` interface and register in `GetRetryStrategy()`.

## Troubleshooting

**Jobs not created**: Check RBAC permissions in `deploy/rbac.yaml`

**Image pull fails**: Verify image exists and node has registry access. Check Job pod events:
```bash
kubectl describe pod <pod-name> -n <namespace>
```

**Tasks stuck pending**: Check concurrency limit (TaskManager.maxConcurrency)

**Watch timeouts**: StatusTracker watch has 5-minute timeout. Long-running tasks may need adjustment.
