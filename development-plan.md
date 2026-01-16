# 镜像预热服务 - 开发流程方案

## 目录
- [架构设计合理性分析](#架构设计合理性分析)
- [开发顺序规划](#开发顺序规划)
- [关键技术挑战与解决方案](#关键技术挑战与解决方案)
- [代码组织建议](#代码组织建议)
- [测试策略](#测试策略)
- [配置管理](#配置管理)
- [监控与可观测性](#监控与可观测性)
- [潜在优化点](#潜在优化点)
- [风险评估](#风险评估)
- [开发检查清单](#开发检查清单)

---

## 架构设计合理性分析

### ✅ 优点

1. **分层架构清晰**
   - API层 → 业务逻辑层 → 存储层 → K8s层
   - 各层职责明确，易于维护和测试

2. **RESTful API 设计符合标准**
   - 易于集成和理解
   - 支持多种客户端（curl、SDK、Web UI、CI/CD）

3. **灵活的存储方案**
   - MVP采用内存存储，快速启动
   - 生产环境可切换到Redis，扩展性好

4. **真正的服务化**
   - 用户无需了解Kubernetes
   - 降低使用门槛

### ⚠️ 需要注意的点

1. **并发控制**
   - 后台任务执行采用 goroutine 异步处理
   - 需要考虑并发任务的资源竞争
   - Repository需要线程安全

2. **可靠性保障**
   - Job 状态监控机制的选择（定期轮询 vs Watch机制）
   - 网络异常时的重试策略

3. **数据持久化**
   - 内存存储模式下，服务重启会丢失任务历史
   - 需要尽快规划持久化方案

---

## 开发顺序规划

### Phase 1: 基础框架 (第1天)

#### 1. 项目初始化
```bash
# 初始化Go模块
go mod init github.com/kitsnail/ips

# 创建目录结构
mkdir -p cmd/apiserver
mkdir -p internal/{api/{handler,middleware},service,repository,k8s}
mkdir -p pkg/{models,metrics}
mkdir -p deploy
mkdir -p client/{python,go}

# 引入依赖
go get github.com/gin-gonic/gin
go get k8s.io/client-go@latest
go get k8s.io/api@latest
go get github.com/sirupsen/logrus
go get github.com/prometheus/client_golang/prometheus
```

#### 2. 核心模型定义
- `pkg/models/task.go` - Task、Progress、Status等数据结构
- 常量定义（TaskStatus枚举）
- 数据模型的JSON序列化/反序列化

#### 3. K8s Client 封装
- `internal/k8s/client.go` - 初始化K8s客户端（支持in-cluster和kubeconfig）
- `internal/k8s/node.go` - 获取节点列表和节点信息
- 单元测试（使用fake clientset）

#### 4. HTTP Server 骨架
- `cmd/apiserver/main.go` - 程序入口
- `internal/api/router.go` - 路由注册
- 健康检查接口实现 (`/healthz`)
- 优雅关闭机制

---

### Phase 2: 核心业务逻辑 (第2天)

#### 5. 存储层实现
- `internal/repository/memory.go` - Repository接口定义
- 内存存储实现（使用sync.Map或sync.RWMutex保护）
- 基础CRUD方法：Create、Get、List、Update、Delete
- 单元测试

#### 6. Job创建器
- `internal/k8s/job_creator.go`
  - 生成Job YAML结构（使用client-go的类型化API）
  - 支持nodeSelector
  - 支持多镜像预热（在一个Job中拉取多个镜像）
  - 创建Job到集群
- 单元测试（使用fake clientset验证Job对象正确性）

#### 7. 节点过滤服务
- `internal/service/node_filter.go`
  - 根据nodeSelector过滤节点
  - 处理节点标签匹配逻辑
  - 排除不可调度的节点（cordoned、NotReady）
- 测试

#### 8. 批次调度器
- `internal/service/batch_scheduler.go`
  - 分批逻辑（按batchSize切片）
  - 顺序执行批次（上一批完成后再执行下一批）
  - 批次间等待机制
  - 支持并发批次（可选，高级特性）
- 测试

---

### Phase 3: 任务管理与API (第3天)

#### 9. 状态跟踪器
- `internal/service/status_tracker.go`
  - 定期轮询Job状态（MVP方案）
  - 或使用Watch机制（生产推荐）
  - 更新Task进度（completedNodes、failedNodes、percentage）
  - 记录失败节点详情（nodeName、image、reason、timestamp）
  - 计算预计完成时间
- 测试

#### 10. 任务管理器
- `internal/service/task_manager.go`
  - 任务生命周期管理（创建→运行→完成/失败/取消）
  - 协调各组件：filter → scheduler → tracker
  - 后台goroutine执行任务
  - 资源清理（完成后清理Job对象）
  - 上下文取消支持（用于任务取消）
- 集成测试

#### 11. API Handler实现
- `internal/api/handler/task.go`
  - POST `/api/v1/tasks` - 创建任务
  - GET `/api/v1/tasks/:id` - 查询任务详情
  - GET `/api/v1/tasks` - 列出任务（支持过滤和分页）
  - DELETE `/api/v1/tasks/:id` - 取消任务
- `internal/api/handler/health.go`
  - GET `/healthz` - 健康检查
  - GET `/readyz` - 就绪检查
- API测试（使用httptest）

#### 12. 中间件
- `internal/api/middleware/logging.go` - 请求日志（记录method、path、status、latency）
- `internal/api/middleware/recovery.go` - panic恢复
- `internal/api/middleware/cors.go` - CORS支持（可选）

---

### Phase 4: 完善与交付 (第4天)

#### 13. Prometheus指标
- `pkg/metrics/prometheus.go`
  - 任务计数器（按状态分类）
  - 任务耗时直方图
  - 节点处理计数器
  - 活跃任务数量
  - GET `/metrics` 端点
- 测试

#### 14. 部署配置
- `deploy/rbac.yaml` - ServiceAccount + Role（最小权限原则）
  - 权限：list/get nodes, create/delete/get/list/watch jobs
- `deploy/deployment.yaml` - API服务部署
- `deploy/service.yaml` - Service配置（ClusterIP）
- `deploy/ingress.yaml` - Ingress配置（可选）

#### 15. Dockerfile
- 多阶段构建（builder + runtime）
- 使用最小化基础镜像（alpine或distroless）
- 非root用户运行
- 健康检查支持

#### 16. 文档与示例
- `README.md` - 快速开始、部署指南
- API文档（可以使用Swagger/OpenAPI）
- 客户端示例
  - curl脚本
  - Python SDK示例
  - Go SDK示例

---

## 关键技术挑战与解决方案

### 挑战1: 任务状态同步

**问题**: API服务创建Job后，如何实时跟踪所有Job的执行状态？

**方案A - 定期轮询（简单，MVP推荐）**:
```go
func (t *StatusTracker) Track(ctx context.Context, taskID string) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 查询所有相关Job状态
            jobs, err := t.k8sClient.ListJobs(ctx, metav1.ListOptions{
                LabelSelector: fmt.Sprintf("task-id=%s", taskID),
            })
            if err != nil {
                log.Errorf("Failed to list jobs: %v", err)
                continue
            }

            // 更新Task进度
            t.updateTaskProgress(taskID, jobs)

        case <-ctx.Done():
            return
        }
    }
}
```

**方案B - Watch机制（高效，生产推荐）**:
```go
func (t *StatusTracker) WatchJobs(ctx context.Context, taskID string) {
    watcher, err := t.k8sClient.Watch(ctx, metav1.ListOptions{
        LabelSelector: fmt.Sprintf("task-id=%s", taskID),
    })
    if err != nil {
        log.Errorf("Failed to watch jobs: %v", err)
        return
    }
    defer watcher.Stop()

    for {
        select {
        case event := <-watcher.ResultChan():
            job, ok := event.Object.(*batchv1.Job)
            if !ok {
                continue
            }

            // 实时更新Task状态
            t.handleJobEvent(event.Type, job)

        case <-ctx.Done():
            return
        }
    }
}
```

**推荐**:
- MVP阶段使用方案A（简单可靠）
- 生产环境切换到方案B（降低API Server压力）

---

### 挑战2: 并发任务管理

**问题**: 多个任务同时执行时，如何避免资源竞争？

**解决方案**:

1. **Repository层使用锁保护**:
```go
type MemoryRepository struct {
    mu    sync.RWMutex
    tasks map[string]*models.Task
}

func (r *MemoryRepository) Get(id string) (*models.Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    task, ok := r.tasks[id]
    if !ok {
        return nil, ErrTaskNotFound
    }

    // 返回副本，避免外部修改
    return task.DeepCopy(), nil
}
```

2. **每个Task独立的goroutine和context**:
```go
func (m *TaskManager) CreateTask(req *CreateTaskRequest) (*Task, error) {
    task := &Task{
        ID:     generateTaskID(),
        Status: TaskPending,
        // ...
    }

    m.repo.Create(task)

    // 每个任务独立执行，互不干扰
    go func() {
        ctx, cancel := context.WithCancel(context.Background())
        m.taskContexts.Store(task.ID, cancel) // 保存cancel函数用于取消
        defer m.taskContexts.Delete(task.ID)

        m.executeTask(ctx, task)
    }()

    return task, nil
}
```

3. **K8s Job通过标签隔离**:
```go
func (j *JobCreator) CreateJob(taskID, nodeName string, images []string) error {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("prewarm-%s-%s", taskID, nodeName),
            Labels: map[string]string{
                "app":     "image-prewarm",
                "task-id": taskID,
                "node":    nodeName,
            },
        },
        // ...
    }

    _, err := j.clientset.BatchV1().Jobs(j.namespace).Create(ctx, job, metav1.CreateOptions{})
    return err
}
```

---

### 挑战3: 失败处理

**问题**: 某个节点的Job失败了怎么办？

**解决方案**:

1. **记录失败详情**:
```go
type FailedNode struct {
    NodeName  string    `json:"nodeName"`
    Image     string    `json:"image"`
    Reason    string    `json:"reason"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}

func (t *StatusTracker) handleFailedJob(job *batchv1.Job) {
    failedNode := FailedNode{
        NodeName:  job.Labels["node"],
        Image:     job.Labels["image"],
        Reason:    getJobFailureReason(job),
        Message:   getJobFailureMessage(job),
        Timestamp: time.Now(),
    }

    task.FailedNodes = append(task.FailedNodes, failedNode)
    t.repo.Update(task)
}
```

2. **继续执行其他批次**:
```go
func (s *BatchScheduler) ExecuteBatch(ctx context.Context, taskID string, nodes []string, images []string) error {
    var wg sync.WaitGroup

    for _, node := range nodes {
        wg.Add(1)
        go func(nodeName string) {
            defer wg.Done()

            // 即使某个Job失败，也不影响其他Job
            if err := s.jobCreator.CreateJob(taskID, nodeName, images); err != nil {
                log.Errorf("Failed to create job for node %s: %v", nodeName, err)
                // 记录失败，但继续
            }
        }(node)
    }

    wg.Wait()
    return nil
}
```

3. **最终状态判定**:
```go
func determineTaskStatus(completedNodes, failedNodes, totalNodes int) TaskStatus {
    successRate := float64(completedNodes) / float64(totalNodes)

    if completedNodes+failedNodes == totalNodes {
        // 所有节点都处理完毕
        if successRate >= 0.9 {
            return TaskCompleted // 成功率 >= 90%
        }
        return TaskFailed
    }

    return TaskRunning
}
```

---

### 挑战4: 资源清理

**问题**: 大量Job对象残留在集群中

**解决方案**:

1. **Job自动清理（推荐）**:
```go
func (j *JobCreator) CreateJob(taskID, nodeName string, images []string) error {
    ttl := int32(3600) // 1小时后自动清理

    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("prewarm-%s-%s", taskID, nodeName),
        },
        Spec: batchv1.JobSpec{
            TTLSecondsAfterFinished: &ttl, // Kubernetes 1.21+
            // ...
        },
    }

    _, err := j.clientset.BatchV1().Jobs(j.namespace).Create(ctx, job, metav1.CreateOptions{})
    return err
}
```

2. **任务完成后主动删除（可选）**:
```go
func (m *TaskManager) cleanupJobs(ctx context.Context, taskID string) error {
    deletePolicy := metav1.DeletePropagationBackground

    err := m.k8sClient.BatchV1().Jobs(m.namespace).DeleteCollection(ctx,
        metav1.DeleteOptions{
            PropagationPolicy: &deletePolicy,
        },
        metav1.ListOptions{
            LabelSelector: fmt.Sprintf("task-id=%s", taskID),
        },
    )

    return err
}
```

3. **定期清理孤儿Job（CronJob）**:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: prewarm-cleanup
spec:
  schedule: "0 */6 * * *"  # 每6小时执行一次
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: prewarm-cleaner
          containers:
          - name: cleanup
            image: bitnami/kubectl:latest
            command:
            - /bin/sh
            - -c
            - |
              # 清理24小时前完成的Job
              kubectl delete jobs -l app=image-prewarm --field-selector status.successful=1 \
                --field-selector metadata.creationTimestamp<$(date -d '24 hours ago' -u +%Y-%m-%dT%H:%M:%SZ)
          restartPolicy: OnFailure
```

---

## 代码组织建议

### 关键数据结构

#### pkg/models/task.go
```go
package models

import "time"

// Task 代表一个镜像预热任务
type Task struct {
    ID           string            `json:"taskId"`
    Status       TaskStatus        `json:"status"`
    Images       []string          `json:"images"`
    BatchSize    int               `json:"batchSize"`
    NodeSelector map[string]string `json:"nodeSelector,omitempty"`
    Progress     *Progress         `json:"progress,omitempty"`
    FailedNodes  []FailedNode      `json:"failedNodeDetails,omitempty"`
    CreatedAt    time.Time         `json:"createdAt"`
    StartedAt    *time.Time        `json:"startedAt,omitempty"`
    FinishedAt   *time.Time        `json:"finishedAt,omitempty"`
    EstimatedEnd *time.Time        `json:"estimatedCompletion,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskRunning   TaskStatus = "running"
    TaskCompleted TaskStatus = "completed"
    TaskFailed    TaskStatus = "failed"
    TaskCancelled TaskStatus = "cancelled"
)

// Progress 任务进度
type Progress struct {
    TotalNodes     int     `json:"totalNodes"`
    CompletedNodes int     `json:"completedNodes"`
    FailedNodes    int     `json:"failedNodes"`
    CurrentBatch   int     `json:"currentBatch"`
    TotalBatches   int     `json:"totalBatches"`
    Percentage     float64 `json:"percentage"`
}

// FailedNode 失败节点详情
type FailedNode struct {
    NodeName  string    `json:"nodeName"`
    Image     string    `json:"image"`
    Reason    string    `json:"reason"`
    Message   string    `json:"message,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

// DeepCopy 返回Task的深拷贝
func (t *Task) DeepCopy() *Task {
    // 实现深拷贝逻辑
    // ...
    return copy
}

// CalculateProgress 计算任务进度
func (t *Task) CalculateProgress() {
    if t.Progress == nil {
        return
    }

    total := t.Progress.TotalNodes
    if total == 0 {
        t.Progress.Percentage = 0
        return
    }

    completed := t.Progress.CompletedNodes
    t.Progress.Percentage = float64(completed) / float64(total) * 100
}
```

#### pkg/models/request.go
```go
package models

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
    Images       []string          `json:"images" binding:"required,min=1"`
    BatchSize    int               `json:"batchSize" binding:"required,min=1,max=100"`
    NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// ListTasksRequest 列表查询请求
type ListTasksRequest struct {
    Status string `form:"status"`
    Limit  int    `form:"limit" binding:"min=1,max=100"`
    Offset int    `form:"offset" binding:"min=0"`
}

// TaskFilter 任务过滤条件
type TaskFilter struct {
    Status *TaskStatus
    Limit  int
    Offset int
}
```

---

### 接口抽象

#### internal/repository/repository.go
```go
package repository

import (
    "context"
    "github.com/kitsnail/ips/pkg/models"
)

// TaskRepository 任务存储接口
type TaskRepository interface {
    // Create 创建任务
    Create(ctx context.Context, task *models.Task) error

    // Get 获取任务
    Get(ctx context.Context, id string) (*models.Task, error)

    // List 列出任务
    List(ctx context.Context, filter models.TaskFilter) ([]*models.Task, int, error)

    // Update 更新任务
    Update(ctx context.Context, task *models.Task) error

    // Delete 删除任务
    Delete(ctx context.Context, id string) error
}
```

#### internal/service/service.go
```go
package service

import (
    "context"
    "github.com/kitsnail/ips/pkg/models"
)

// TaskManager 任务管理服务接口
type TaskManager interface {
    // CreateTask 创建任务
    CreateTask(ctx context.Context, req *models.CreateTaskRequest) (*models.Task, error)

    // GetTask 获取任务
    GetTask(ctx context.Context, id string) (*models.Task, error)

    // ListTasks 列出任务
    ListTasks(ctx context.Context, filter models.TaskFilter) ([]*models.Task, int, error)

    // CancelTask 取消任务
    CancelTask(ctx context.Context, id string) error
}

// NodeFilter 节点过滤服务接口
type NodeFilter interface {
    // FilterNodes 根据选择器过滤节点
    FilterNodes(ctx context.Context, selector map[string]string) ([]string, error)
}

// BatchScheduler 批次调度服务接口
type BatchScheduler interface {
    // ScheduleBatches 分批调度任务
    ScheduleBatches(ctx context.Context, taskID string, nodes []string, images []string, batchSize int) error
}

// StatusTracker 状态跟踪服务接口
type StatusTracker interface {
    // TrackTask 跟踪任务状态
    TrackTask(ctx context.Context, taskID string) error
}
```

---

## 测试策略

### 单元测试

#### 1. Repository测试
```go
// internal/repository/memory_test.go
func TestMemoryRepository_Create(t *testing.T) {
    repo := NewMemoryRepository()

    task := &models.Task{
        ID:        "test-123",
        Status:    models.TaskPending,
        Images:    []string{"nginx:latest"},
        BatchSize: 10,
        CreatedAt: time.Now(),
    }

    err := repo.Create(context.Background(), task)
    assert.NoError(t, err)

    // 验证可以获取
    retrieved, err := repo.Get(context.Background(), "test-123")
    assert.NoError(t, err)
    assert.Equal(t, task.ID, retrieved.ID)
}
```

#### 2. NodeFilter测试
```go
// internal/service/node_filter_test.go
func TestNodeFilter_FilterNodes(t *testing.T) {
    // 使用fake clientset
    clientset := fake.NewSimpleClientset(
        &corev1.Node{
            ObjectMeta: metav1.ObjectMeta{
                Name: "node-1",
                Labels: map[string]string{
                    "workload": "compute",
                },
            },
            Status: corev1.NodeStatus{
                Conditions: []corev1.NodeCondition{
                    {Type: corev1.NodeReady, Status: corev1.ConditionTrue},
                },
            },
        },
        &corev1.Node{
            ObjectMeta: metav1.ObjectMeta{
                Name: "node-2",
                Labels: map[string]string{
                    "workload": "storage",
                },
            },
            Status: corev1.NodeStatus{
                Conditions: []corev1.NodeCondition{
                    {Type: corev1.NodeReady, Status: corev1.ConditionTrue},
                },
            },
        },
    )

    filter := NewNodeFilter(clientset)

    // 测试过滤
    nodes, err := filter.FilterNodes(context.Background(), map[string]string{
        "workload": "compute",
    })

    assert.NoError(t, err)
    assert.Equal(t, 1, len(nodes))
    assert.Equal(t, "node-1", nodes[0])
}
```

#### 3. BatchScheduler测试
```go
// internal/service/batch_scheduler_test.go
func TestBatchScheduler_SplitBatches(t *testing.T) {
    scheduler := &BatchScheduler{}

    nodes := []string{"node-1", "node-2", "node-3", "node-4", "node-5"}
    batchSize := 2

    batches := scheduler.splitBatches(nodes, batchSize)

    assert.Equal(t, 3, len(batches)) // [2, 2, 1]
    assert.Equal(t, 2, len(batches[0]))
    assert.Equal(t, 2, len(batches[1]))
    assert.Equal(t, 1, len(batches[2]))
}
```

#### 4. JobCreator测试
```go
// internal/k8s/job_creator_test.go
func TestJobCreator_CreateJob(t *testing.T) {
    clientset := fake.NewSimpleClientset()
    creator := NewJobCreator(clientset, "default")

    err := creator.CreateJob(context.Background(), "task-123", "node-1", []string{"nginx:latest", "redis:7"})
    assert.NoError(t, err)

    // 验证Job被创建
    jobs, err := clientset.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
    assert.NoError(t, err)
    assert.Equal(t, 1, len(jobs.Items))

    job := jobs.Items[0]
    assert.Equal(t, "task-123", job.Labels["task-id"])
    assert.Equal(t, "node-1", job.Labels["node"])
}
```

---

### 集成测试

#### API集成测试
```go
// internal/api/handler/task_test.go
func TestTaskHandler_CreateTask_E2E(t *testing.T) {
    // 设置测试环境
    repo := repository.NewMemoryRepository()
    k8sClient := fake.NewSimpleClientset(createTestNodes()...)
    taskManager := service.NewTaskManager(repo, k8sClient)
    handler := handler.NewTaskHandler(taskManager)

    // 创建测试服务器
    router := gin.Default()
    router.POST("/api/v1/tasks", handler.CreateTask)
    router.GET("/api/v1/tasks/:id", handler.GetTask)

    // 测试创建任务
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/tasks", strings.NewReader(`{
        "images": ["nginx:latest"],
        "batchSize": 10
    }`))
    req.Header.Set("Content-Type", "application/json")

    router.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)

    var response models.Task
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.NotEmpty(t, response.ID)
    assert.Equal(t, models.TaskPending, response.Status)

    // 测试查询任务
    time.Sleep(100 * time.Millisecond) // 等待异步执行

    w2 := httptest.NewRecorder()
    req2, _ := http.NewRequest("GET", "/api/v1/tasks/"+response.ID, nil)

    router.ServeHTTP(w2, req2)

    assert.Equal(t, 200, w2.Code)
}
```

---

### E2E测试（可选）

```go
// test/e2e/prewarm_test.go
// +build e2e

func TestPrewarmE2E(t *testing.T) {
    // 前提：需要有一个运行的Kubernetes集群（kind、minikube等）

    // 1. 部署API服务
    kubectl("apply", "-f", "../../deploy/")
    defer kubectl("delete", "-f", "../../deploy/")

    // 2. 等待服务就绪
    waitForDeployment("prewarm-api")

    // 3. 获取服务地址
    apiURL := getServiceURL("prewarm-api")

    // 4. 创建任务
    resp, err := http.Post(apiURL+"/api/v1/tasks", "application/json", strings.NewReader(`{
        "images": ["nginx:latest"],
        "batchSize": 5
    }`))
    require.NoError(t, err)
    require.Equal(t, 201, resp.StatusCode)

    var task models.Task
    json.NewDecoder(resp.Body).Decode(&task)

    // 5. 轮询直到完成
    timeout := time.After(5 * time.Minute)
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            t.Fatal("Task did not complete in time")
        case <-ticker.C:
            resp, _ := http.Get(apiURL + "/api/v1/tasks/" + task.ID)
            json.NewDecoder(resp.Body).Decode(&task)

            if task.Status == models.TaskCompleted {
                t.Logf("Task completed: %+v", task.Progress)
                return
            } else if task.Status == models.TaskFailed {
                t.Fatalf("Task failed: %+v", task)
            }

            t.Logf("Task progress: %.1f%%", task.Progress.Percentage)
        }
    }
}
```

---

## 配置管理

### 配置结构

```go
// internal/config/config.go
package config

import (
    "time"

    "github.com/kelseyhightower/envconfig"
)

// Config 应用配置
type Config struct {
    Server     ServerConfig
    Storage    StorageConfig
    Kubernetes KubernetesConfig
    Log        LogConfig
}

// ServerConfig HTTP服务配置
type ServerConfig struct {
    Port         int    `default:"8080" envconfig:"SERVER_PORT"`
    Mode         string `default:"release" envconfig:"GIN_MODE"` // debug, release, test
    ReadTimeout  int    `default:"30" envconfig:"SERVER_READ_TIMEOUT"`  // seconds
    WriteTimeout int    `default:"30" envconfig:"SERVER_WRITE_TIMEOUT"` // seconds
}

// StorageConfig 存储配置
type StorageConfig struct {
    Type     string `default:"memory" envconfig:"STORAGE_TYPE"` // memory, redis
    RedisURL string `envconfig:"REDIS_URL"`                     // redis://host:port
}

// KubernetesConfig K8s配置
type KubernetesConfig struct {
    Namespace       string `default:"default" envconfig:"K8S_NAMESPACE"`
    JobTTL          int    `default:"3600" envconfig:"JOB_TTL_SECONDS"`        // Job自动清理时间
    PollInterval    int    `default:"5" envconfig:"STATUS_POLL_INTERVAL"`      // 状态轮询间隔（秒）
    MaxConcurrent   int    `default:"10" envconfig:"MAX_CONCURRENT_TASKS"`     // 最大并发任务数
    WorkerImage     string `default:"busybox:latest" envconfig:"WORKER_IMAGE"` // Worker镜像
    WorkerPullImage string `default:"IfNotPresent" envconfig:"WORKER_IMAGE_PULL_POLICY"`
}

// LogConfig 日志配置
type LogConfig struct {
    Level  string `default:"info" envconfig:"LOG_LEVEL"`   // debug, info, warn, error
    Format string `default:"json" envconfig:"LOG_FORMAT"`  // json, text
}

// Load 加载配置（从环境变量）
func Load() (*Config, error) {
    var cfg Config

    if err := envconfig.Process("", &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", c.Server.Port)
    }

    if c.Storage.Type != "memory" && c.Storage.Type != "redis" {
        return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
    }

    if c.Storage.Type == "redis" && c.Storage.RedisURL == "" {
        return fmt.Errorf("redis URL is required when storage type is redis")
    }

    if c.Kubernetes.JobTTL < 60 {
        return fmt.Errorf("job TTL must be at least 60 seconds")
    }

    return nil
}
```

### 配置文件示例

```yaml
# config.yaml (可选，优先使用环境变量)
server:
  port: 8080
  mode: release
  read_timeout: 30
  write_timeout: 30

storage:
  type: memory
  redis_url: ""

kubernetes:
  namespace: default
  job_ttl_seconds: 3600
  poll_interval: 5
  max_concurrent_tasks: 10
  worker_image: busybox:latest
  worker_image_pull_policy: IfNotPresent

log:
  level: info
  format: json
```

### 环境变量示例

```bash
# .env
SERVER_PORT=8080
GIN_MODE=release

STORAGE_TYPE=memory
# REDIS_URL=redis://localhost:6379

K8S_NAMESPACE=default
JOB_TTL_SECONDS=3600
STATUS_POLL_INTERVAL=5
MAX_CONCURRENT_TASKS=10

LOG_LEVEL=info
LOG_FORMAT=json
```

---

## 监控与可观测性

### Prometheus指标

```go
// pkg/metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // TaskTotal 任务总数（按状态分类）
    TaskTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "prewarm_tasks_total",
            Help: "Total number of prewarm tasks",
        },
        []string{"status"}, // completed, failed, cancelled
    )

    // TaskDuration 任务耗时
    TaskDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "prewarm_task_duration_seconds",
            Help:    "Task duration in seconds",
            Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
        },
        []string{"status"},
    )

    // NodesProcessed 处理的节点数
    NodesProcessed = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "prewarm_nodes_processed_total",
            Help: "Total number of nodes processed",
        },
    )

    // ActiveTasks 当前活跃任务数
    ActiveTasks = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "prewarm_active_tasks",
            Help: "Number of currently active tasks",
        },
    )

    // ImagesPrewarmed 预热的镜像数量
    ImagesPrewarmed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "prewarm_images_total",
            Help: "Total number of images prewarmed",
        },
        []string{"image"}, // 按镜像名称分类
    )

    // APIRequestDuration API请求耗时
    APIRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "prewarm_api_request_duration_seconds",
            Help:    "API request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )
)

// RecordTaskStart 记录任务开始
func RecordTaskStart() {
    ActiveTasks.Inc()
}

// RecordTaskComplete 记录任务完成
func RecordTaskComplete(status string, duration float64) {
    TaskTotal.WithLabelValues(status).Inc()
    TaskDuration.WithLabelValues(status).Observe(duration)
    ActiveTasks.Dec()
}

// RecordNodeProcessed 记录节点处理
func RecordNodeProcessed(count int) {
    NodesProcessed.Add(float64(count))
}

// RecordImagePrewarmed 记录镜像预热
func RecordImagePrewarmed(image string) {
    ImagesPrewarmed.WithLabelValues(image).Inc()
}
```

### Prometheus中间件

```go
// internal/api/middleware/metrics.go
package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/kitsnail/ips/pkg/metrics"
)

// PrometheusMiddleware Prometheus指标中间件
func PrometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // 处理请求
        c.Next()

        // 记录指标
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())

        metrics.APIRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            status,
        ).Observe(duration)
    }
}
```

### 日志规范

```go
// pkg/logger/logger.go
package logger

import (
    "os"

    "github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// Init 初始化日志
func Init(level, format string) {
    Log = logrus.New()

    // 设置日志级别
    logLevel, err := logrus.ParseLevel(level)
    if err != nil {
        logLevel = logrus.InfoLevel
    }
    Log.SetLevel(logLevel)

    // 设置输出格式
    if format == "json" {
        Log.SetFormatter(&logrus.JSONFormatter{
            TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
        })
    } else {
        Log.SetFormatter(&logrus.TextFormatter{
            FullTimestamp:   true,
            TimestampFormat: "2006-01-02 15:04:05",
        })
    }

    Log.SetOutput(os.Stdout)
}

// WithTaskContext 创建带任务上下文的logger
func WithTaskContext(taskID string) *logrus.Entry {
    return Log.WithFields(logrus.Fields{
        "taskId": taskID,
    })
}

// WithRequestContext 创建带请求上下文的logger
func WithRequestContext(method, path, requestID string) *logrus.Entry {
    return Log.WithFields(logrus.Fields{
        "method":    method,
        "path":      path,
        "requestId": requestID,
    })
}
```

### 使用示例

```go
// 任务执行中记录日志
logger.WithTaskContext(task.ID).WithFields(logrus.Fields{
    "status":     task.Status,
    "nodeCount":  len(nodes),
    "batchSize":  task.BatchSize,
    "images":     task.Images,
}).Info("Task execution started")

// 记录错误
logger.WithTaskContext(task.ID).WithError(err).Error("Failed to create job")

// 记录指标
metrics.RecordTaskStart()
defer func() {
    duration := time.Since(startTime).Seconds()
    metrics.RecordTaskComplete(string(task.Status), duration)
}()
```

---

## 潜在优化点

### 短期优化 (MVP+)

#### 1. 任务优先级队列
```go
// 为任务添加优先级字段
type Task struct {
    // ...
    Priority int `json:"priority"` // 1-10, 10最高
}

// 使用堆实现优先级队列
type TaskQueue struct {
    mu    sync.Mutex
    tasks []*Task
}

func (q *TaskQueue) Push(task *Task) {
    q.mu.Lock()
    defer q.mu.Unlock()

    q.tasks = append(q.tasks, task)
    heap.Fix(q, len(q.tasks)-1)
}

func (q *TaskQueue) Pop() *Task {
    q.mu.Lock()
    defer q.mu.Unlock()

    if len(q.tasks) == 0 {
        return nil
    }

    return heap.Pop(q).(*Task)
}
```

#### 2. 任务重试机制
```go
type Task struct {
    // ...
    MaxRetries    int `json:"maxRetries"`
    CurrentRetry  int `json:"currentRetry"`
    RetryPolicy   string `json:"retryPolicy"` // exponential, linear
}

func (m *TaskManager) executeTaskWithRetry(ctx context.Context, task *Task) {
    for task.CurrentRetry <= task.MaxRetries {
        err := m.executeTask(ctx, task)
        if err == nil {
            return // 成功
        }

        task.CurrentRetry++
        if task.CurrentRetry > task.MaxRetries {
            task.Status = TaskFailed
            return
        }

        // 计算重试延迟
        delay := calculateRetryDelay(task.RetryPolicy, task.CurrentRetry)
        time.Sleep(delay)
    }
}
```

#### 3. Webhook通知
```go
type Task struct {
    // ...
    WebhookURL string `json:"webhookUrl,omitempty"`
}

func (m *TaskManager) notifyWebhook(task *Task) {
    if task.WebhookURL == "" {
        return
    }

    payload := map[string]interface{}{
        "taskId": task.ID,
        "status": task.Status,
        "progress": task.Progress,
    }

    go func() {
        body, _ := json.Marshal(payload)
        http.Post(task.WebhookURL, "application/json", bytes.NewBuffer(body))
    }()
}
```

#### 4. 并发任务数限制
```go
type TaskManager struct {
    // ...
    semaphore chan struct{} // 控制并发
}

func NewTaskManager(maxConcurrent int) *TaskManager {
    return &TaskManager{
        semaphore: make(chan struct{}, maxConcurrent),
    }
}

func (m *TaskManager) CreateTask(req *CreateTaskRequest) (*Task, error) {
    // 尝试获取执行槽
    select {
    case m.semaphore <- struct{}{}:
        // 获得槽位，可以执行
        go func() {
            defer func() { <-m.semaphore }() // 释放槽位
            m.executeTask(ctx, task)
        }()
    default:
        // 槽位已满，任务排队
        task.Status = TaskQueued
    }

    return task, nil
}
```

---

### 长期优化

#### 1. Watch机制替代轮询
```go
// internal/service/status_tracker.go
func (t *StatusTracker) WatchTaskJobs(ctx context.Context, taskID string) error {
    // 使用Informer机制
    informerFactory := informers.NewSharedInformerFactoryWithOptions(
        t.clientset,
        30*time.Second,
        informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
            opts.LabelSelector = fmt.Sprintf("task-id=%s", taskID)
        }),
    )

    jobInformer := informerFactory.Batch().V1().Jobs().Informer()

    jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        UpdateFunc: func(oldObj, newObj interface{}) {
            job := newObj.(*batchv1.Job)
            t.handleJobUpdate(job)
        },
    })

    informerFactory.Start(ctx.Done())
    informerFactory.WaitForCacheSync(ctx.Done())

    <-ctx.Done()
    return nil
}
```

#### 2. DaemonSet模式（更高效）
```go
// 不使用多个Job，而是使用DaemonSet
// DaemonSet会在所有节点上运行，更高效

func (j *JobCreator) CreateDaemonSet(taskID string, images []string) error {
    ds := &appsv1.DaemonSet{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("prewarm-%s", taskID),
            Labels: map[string]string{
                "app":     "image-prewarm",
                "task-id": taskID,
            },
        },
        Spec: appsv1.DaemonSetSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app":     "image-prewarm",
                    "task-id": taskID,
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":     "image-prewarm",
                        "task-id": taskID,
                    },
                },
                Spec: corev1.PodSpec{
                    InitContainers: createInitContainers(images),
                    Containers: []corev1.Container{
                        {
                            Name:    "pause",
                            Image:   "gcr.io/google_containers/pause:3.2",
                            Command: []string{"/pause"},
                        },
                    },
                    RestartPolicy: corev1.RestartPolicyAlways,
                },
            },
        },
    }

    _, err := j.clientset.AppsV1().DaemonSets(j.namespace).Create(ctx, ds, metav1.CreateOptions{})
    return err
}
```

#### 3. 镜像预热预测
```go
// 基于历史数据预测需要预热的镜像
type PrewarmPredictor struct {
    repo repository.TaskRepository
}

func (p *PrewarmPredictor) PredictImages() ([]string, error) {
    // 获取最近7天的任务
    tasks, _, err := p.repo.List(context.Background(), models.TaskFilter{
        // 过滤条件
    })
    if err != nil {
        return nil, err
    }

    // 统计镜像频率
    imageFreq := make(map[string]int)
    for _, task := range tasks {
        for _, image := range task.Images {
            imageFreq[image]++
        }
    }

    // 返回热门镜像
    var topImages []string
    for image, freq := range imageFreq {
        if freq >= 3 { // 出现3次以上
            topImages = append(topImages, image)
        }
    }

    return topImages, nil
}
```

#### 4. 多集群支持
```go
// 支持在多个集群中执行预热
type MultiClusterTaskManager struct {
    clusters map[string]*kubernetes.Clientset
}

func (m *MultiClusterTaskManager) CreateTask(clusterName string, req *CreateTaskRequest) (*Task, error) {
    clientset, ok := m.clusters[clusterName]
    if !ok {
        return nil, fmt.Errorf("cluster not found: %s", clusterName)
    }

    // 在指定集群中创建任务
    taskManager := service.NewTaskManager(repo, clientset)
    return taskManager.CreateTask(context.Background(), req)
}
```

---

## 风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| **K8s API限流导致Job创建失败** | 高 | 中 | 1. 添加重试机制+指数退避<br>2. 批量创建Job时添加延迟<br>3. 监控API请求失败率 |
| **单实例内存存储丢失数据** | 中 | 高 | 1. 尽快切换到Redis<br>2. 添加定期备份机制<br>3. 多实例部署+共享存储 |
| **大量并发任务耗尽集群资源** | 高 | 低 | 1. 添加并发任务数限制<br>2. Job资源配额限制<br>3. 优先级队列 |
| **Job状态更新延迟** | 低 | 中 | 1. 缩短轮询间隔<br>2. 生产环境使用Watch机制<br>3. 添加超时检测 |
| **镜像拉取超时** | 中 | 中 | 1. 设置合理的`activeDeadlineSeconds`<br>2. 添加重试机制<br>3. 使用镜像加速器 |
| **API服务单点故障** | 高 | 中 | 1. 多实例部署<br>2. 共享存储（Redis）<br>3. 健康检查+自动重启 |
| **RBAC权限不足** | 高 | 低 | 1. 遵循最小权限原则<br>2. 部署前验证权限<br>3. 详细的错误日志 |
| **Job对象泄漏** | 中 | 中 | 1. 使用TTL自动清理<br>2. 定期清理CronJob<br>3. 监控Job数量 |

---

## 开发检查清单

### 功能完整性

#### API功能
- [ ] POST `/api/v1/tasks` - 创建任务
- [ ] GET `/api/v1/tasks/:id` - 查询任务详情
- [ ] GET `/api/v1/tasks` - 列表任务（支持过滤和分页）
- [ ] DELETE `/api/v1/tasks/:id` - 取消任务
- [ ] GET `/healthz` - 健康检查
- [ ] GET `/readyz` - 就绪检查
- [ ] GET `/metrics` - Prometheus指标

#### 核心功能
- [ ] 节点标签选择器支持（nodeSelector）
- [ ] 批次大小配置（batchSize）
- [ ] 任务进度实时更新（percentage、completedNodes等）
- [ ] 失败节点详情记录（nodeName、reason、timestamp）
- [ ] 任务状态流转（pending→running→completed/failed/cancelled）
- [ ] 多镜像支持（一个任务预热多个镜像）
- [ ] Job自动清理（TTL机制）

---

### 非功能性需求

#### 可靠性
- [ ] 并发安全（goroutine + mutex）
- [ ] 错误处理完善（所有error都被处理）
- [ ] 上下文取消支持（context.Context）
- [ ] 重试机制（K8s API调用失败时）
- [ ] 超时控制（HTTP请求、Job执行）

#### 可观测性
- [ ] 结构化日志（使用logrus）
- [ ] 请求ID追踪
- [ ] Prometheus指标（任务数、耗时、节点数等）
- [ ] 错误日志详细（包含堆栈信息）

#### 性能
- [ ] 并发任务控制（semaphore）
- [ ] 批量操作（避免N+1查询）
- [ ] 连接池（Redis、K8s client）
- [ ] 响应压缩（gzip）

#### 安全性
- [ ] 输入参数校验（binding标签）
- [ ] RBAC最小权限
- [ ] 敏感信息脱敏（日志中不记录token等）
- [ ] Rate limiting（可选）
- [ ] 认证授权（可选，后期添加）

---

### 测试覆盖

#### 单元测试
- [ ] Repository CRUD测试
- [ ] NodeFilter 过滤逻辑测试
- [ ] BatchScheduler 分批逻辑测试
- [ ] JobCreator Job创建测试
- [ ] TaskManager 任务管理测试
- [ ] 测试覆盖率 > 70%

#### 集成测试
- [ ] API端到端测试
- [ ] 任务状态流转测试
- [ ] 并发任务测试
- [ ] 取消任务测试

#### E2E测试（可选）
- [ ] 真实K8s集群测试
- [ ] 完整工作流测试

---

### 部署就绪

#### Kubernetes配置
- [ ] RBAC配置完整（ServiceAccount + Role + RoleBinding）
- [ ] Deployment配置（资源限制、健康检查）
- [ ] Service配置（ClusterIP）
- [ ] Ingress配置（可选）
- [ ] ConfigMap/Secret管理（敏感配置）

#### 容器化
- [ ] Dockerfile优化（多阶段构建、最小镜像）
- [ ] 非root用户运行
- [ ] 健康检查端点
- [ ] 优雅关闭（SIGTERM处理）

#### 监控告警
- [ ] Prometheus指标暴露
- [ ] 关键指标告警规则（任务失败率、API错误率）
- [ ] 日志聚合（ELK、Loki等）

---

### 文档

#### 用户文档
- [ ] README.md（项目介绍、快速开始）
- [ ] API文档（所有端点的详细说明）
- [ ] 部署指南（K8s部署步骤）
- [ ] 使用示例（curl、Python、Go）
- [ ] 故障排查指南

#### 开发文档
- [ ] 架构设计文档
- [ ] 代码结构说明
- [ ] 贡献指南
- [ ] 变更日志（CHANGELOG.md）

---

## 总结

### 核心优势

1. **用户体验优先** - RESTful API比CRD更易用，降低使用门槛
2. **渐进式开发** - MVP → 完善 → 增强的清晰路线图
3. **技术栈合理** - Gin + client-go 是Go + K8s的标准组合
4. **扩展性好** - 存储层抽象支持替换，服务化架构易于扩展

### 建议的开发启动顺序

1. **先实现完整的垂直切片**（单个API端到端）
   - 创建任务API → Repository → K8s Job → 状态跟踪

2. **然后横向扩展其他API**
   - 查询任务API
   - 列表任务API
   - 取消任务API

3. **最后完善监控、测试、文档**
   - Prometheus指标
   - 单元测试 + 集成测试
   - 部署配置
   - 使用文档

### 预估工作量

文档估计的 **3-4天** 是合理的，前提是：

✅ 开发者熟悉Go和Kubernetes
✅ 使用现成的库（Gin、client-go）
✅ MVP阶段功能裁剪到位

### 下一步行动

1. **Day 1**: 搭建项目框架，实现K8s client和基础模型
2. **Day 2**: 实现核心业务逻辑（Job创建、批次调度）
3. **Day 3**: 完成API层和状态跟踪
4. **Day 4**: 测试、文档、部署配置

---

**准备好开始编码了吗？建议从 Phase 1 的项目初始化开始！**
