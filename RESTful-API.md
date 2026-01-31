# 镜像预热服务 - RESTful API 方案（最终确定版）

## 方案演进史

通过三轮深度思考，我们的方案持续优化：

```
第一轮思考：CRD + Controller (1500行)
↓ 反思：过度设计，太复杂
第二轮思考：Job + ConfigMap (370行)
↓ 反思：ConfigMap 不必要，步骤多
第三轮思考：Job + 命令行参数 (320行)
↓ 反思：用户还是要懂 kubectl
第四轮思考：RESTful API 服务 (500行) ✅
✓ 最终结论：真正的服务化，用户体验最好
```

---

## 为什么选择 RESTful API？

### 用户的关键洞察

> "我希望用户调用 RESTful API 的方式，这样兼容性、扩展性更好。比如需要预热 10 个镜像，就调用预热服务的 API 下发任务。这样可以兼容很多客户端，比如 curl、kubectl、web，或者其它能调用 API 的工具，也方便实现集成自动化。"

**这个洞察非常正确！**

### 之前方案的根本问题

所有之前的方案（脚本、Job+ConfigMap、Job+命令行参数）都有一个共同的致命缺陷：

❌ **用户必须直接操作 Kubernetes**

这导致：
1. 用户需要 kubectl 权限
2. 用户需要理解 Kubernetes 资源
3. 用户需要编写 YAML
4. 难以集成到其他系统
5. 无法提供 Web UI
6. 客户端单一（只有 kubectl）

### RESTful API 的核心优势

✅ **用户无需了解 Kubernetes**
- 只需要知道 API 地址
- 发送 HTTP 请求即可
- 无需 kubectl 权限

✅ **客户端无关**
- curl（命令行）
- Python/Go SDK（编程）
- Web UI（浏览器）
- CI/CD 工具（Jenkins、GitLab）
- 任何能发 HTTP 请求的工具

✅ **易于集成**
- 标准的 RESTful 接口
- JSON 格式数据
- 可以被任何系统调用

✅ **服务化能力**
- 任务历史记录
- 状态查询
- 生命周期管理
- 审计日志

---

## 架构设计

### 整体架构图

```
┌─────────────────────────────────────────────────────┐
│                  客户端层（多样化）                  │
│                                                      │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────┐ │
│  │  curl   │  │ Web UI  │  │ Python  │  │ CI/CD │ │
│  │ (命令行) │  │ (浏览器) │  │  SDK   │  │(自动化)│ │
│  └─────────┘  └─────────┘  └─────────┘  └───────┘ │
└─────────────────────────────────────────────────────┘
                    ↓ HTTP/HTTPS (统一接口)
┌─────────────────────────────────────────────────────┐
│            镜像预热 API 服务 (Go + Gin)              │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  HTTP 路由层                                 │  │
│  │                                              │  │
│  │  POST   /api/v1/tasks      创建预热任务     │  │
│  │  GET    /api/v1/tasks      列出所有任务     │  │
│  │  GET    /api/v1/tasks/:id  查询任务详情     │  │
│  │  DELETE /api/v1/tasks/:id  取消任务         │  │
│  │  GET    /healthz           健康检查          │  │
│  │  GET    /metrics           Prometheus指标    │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  业务逻辑层                                  │  │
│  │                                              │  │
│  │  ┌────────────┐    ┌──────────────┐        │  │
│  │  │ 任务管理器  │    │ 节点过滤器    │        │  │
│  │  │TaskManager │    │ NodeFilter   │        │  │
│  │  └────────────┘    └──────────────┘        │  │
│  │                                              │  │
│  │  ┌────────────┐    ┌──────────────┐        │  │
│  │  │ 批次调度器  │    │ 状态跟踪器    │        │  │
│  │  │BatchSchedu │    │StatusTracker │        │  │
│  │  └────────────┘    └──────────────┘        │  │
│  │                                              │  │
│  │  ┌──────────────────────────┐               │  │
│  │  │  Job 创建器 (JobCreator)  │               │  │
│  │  └──────────────────────────┘               │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  存储层（可选 Redis，MVP 用内存）            │  │
│  │  - 任务状态                                   │  │
│  │  - 任务历史                                   │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                    ↓ Kubernetes Client API
┌─────────────────────────────────────────────────────┐
│              Kubernetes 集群                         │
│                                                      │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │ Worker  │  │ Worker  │  │ Worker  │            │
│  │ Job #1  │  │ Job #2  │  │ Job #3  │  ...       │
│  │ (Node1) │  │ (Node2) │  │ (Node3) │            │
│  └─────────┘  └─────────┘  └─────────┘            │
└─────────────────────────────────────────────────────┘
```

### 工作流程

```
1. 用户发送 HTTP POST 请求（创建任务）
   ↓
2. API 服务接收请求，生成任务 ID
   ↓
3. API 返回任务 ID 给用户
   ↓
4. API 服务后台开始执行：
   - 获取节点列表
   - 分批创建 Worker Job
   - 监控 Job 状态
   - 更新任务进度
   ↓
5. 用户通过 HTTP GET 查询任务状态
   ↓
6. 任务完成后，API 清理资源
```

---

## API 接口设计

### 1. 创建预热任务

**请求**：
```http
POST /api/v1/tasks
Content-Type: application/json

{
  "images": [
    "nginx:latest",
    "redis:7",
    "postgres:14"
  ],
  "batchSize": 10,
  "nodeSelector": {
    "workload": "compute"  // 可选：节点标签选择器
  }
}
```

**成功响应（201 Created）**：
```json
{
  "taskId": "task-20240116-abc123",
  "status": "pending",
  "images": ["nginx:latest", "redis:7", "postgres:14"],
  "batchSize": 10,
  "createdAt": "2024-01-16T10:00:00Z",
  "message": "Task created successfully"
}
```

**错误响应（400 Bad Request）**：
```json
{
  "error": "Invalid request",
  "details": "images field is required"
}
```

---

### 2. 查询任务详情

**请求**：
```http
GET /api/v1/tasks/task-20240116-abc123
```

**成功响应（200 OK）**：
```json
{
  "taskId": "task-20240116-abc123",
  "status": "running",
  "images": ["nginx:latest", "redis:7", "postgres:14"],
  "batchSize": 10,
  "progress": {
    "totalNodes": 50,
    "completedNodes": 25,
    "failedNodes": 2,
    "currentBatch": 3,
    "totalBatches": 5,
    "percentage": 50.0
  },
  "failedNodeDetails": [
    {
      "nodeName": "node-5",
      "image": "redis:7",
      "reason": "ImagePullBackOff",
      "timestamp": "2024-01-16T10:05:30Z"
    }
  ],
  "createdAt": "2024-01-16T10:00:00Z",
  "startedAt": "2024-01-16T10:00:05Z",
  "estimatedCompletion": "2024-01-16T10:15:00Z"
}
```

**错误响应（404 Not Found）**：
```json
{
  "error": "Task not found",
  "taskId": "task-20240116-abc123"
}
```

---

### 3. 列出所有任务

**请求**：
```http
GET /api/v1/tasks?status=running&limit=20&offset=0
```

**查询参数**：
- `status`：过滤状态（pending/running/completed/failed）
- `limit`：返回数量（默认 10）
- `offset`：偏移量（分页）

**成功响应（200 OK）**：
```json
{
  "tasks": [
    {
      "taskId": "task-20240116-abc123",
      "status": "running",
      "images": ["nginx:latest", "redis:7"],
      "progress": {
        "totalNodes": 50,
        "completedNodes": 25,
        "percentage": 50.0
      },
      "createdAt": "2024-01-16T10:00:00Z"
    },
    {
      "taskId": "task-20240116-def456",
      "status": "completed",
      "images": ["mysql:8"],
      "progress": {
        "totalNodes": 30,
        "completedNodes": 30,
        "percentage": 100.0
      },
      "createdAt": "2024-01-16T09:00:00Z",
      "finishedAt": "2024-01-16T09:15:00Z"
    }
  ],
  "total": 2,
  "limit": 20,
  "offset": 0
}
```

---

### 4. 取消任务

**请求**：
```http
DELETE /api/v1/tasks/task-20240116-abc123
```

**成功响应（200 OK）**：
```json
{
  "taskId": "task-20240116-abc123",
  "status": "cancelled",
  "message": "Task cancelled successfully. All running jobs will be terminated."
}
```

**错误响应（404 Not Found）**：
```json
{
  "error": "Task not found"
}
```

**错误响应（409 Conflict）**：
```json
{
  "error": "Task already completed",
  "status": "completed"
}
```

---

### 5. 定时任务管理

#### 5.1 创建定时任务

**请求**：
```http
POST /api/v1/scheduled-tasks
```

**请求体**：
```json
{
  "name": "Daily Image Prewarm",
  "description": "Prewarm nginx and redis every day at midnight",
  "cronExpr": "0 0 * * *",
  "enabled": true,
  "taskConfig": {
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 10,
    "priority": 1,
    "nodeSelector": {"prewarm": "true"},
    "maxRetries": 3,
    "retryStrategy": "linear",
    "retryDelay": 60,
    "webhookUrl": "https://hooks.example.com/notify"
  },
  "overlapPolicy": "skip",
  "timeoutSeconds": 3600
}
```

**成功响应（201 Created）**：
```json
{
  "id": "scheduled-task-20240116-abc123",
  "name": "Daily Image Prewarm",
  "description": "Prewarm nginx and redis every day at midnight",
  "cronExpr": "0 0 * * *",
  "enabled": true,
  "taskConfig": {
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 10,
    "priority": 1,
    "nodeSelector": {"prewarm": "true"},
    "maxRetries": 3,
    "retryStrategy": "linear",
    "retryDelay": 60,
    "webhookUrl": "https://hooks.example.com/notify"
  },
  "overlapPolicy": "skip",
  "timeoutSeconds": 3600,
  "lastExecutionAt": null,
  "nextExecutionAt": "2024-01-17T00:00:00Z",
  "createdBy": "admin",
  "createdAt": "2024-01-16T10:00:00Z",
  "updatedAt": "2024-01-16T10:00:00Z"
}
```

**字段说明**：
- `cronExpr`: Cron 表达式（5字段标准格式：分 时 日 月 周）
- `overlapPolicy`: 重叠策略
  - `skip`: 跳过本次执行（默认）
  - `allow`: 允许并行执行
  - `queue`: 等待上次完成后执行（暂未实现）
- `timeoutSeconds`: 超时时间（0表示无限制）

#### 5.2 获取定时任务列表

**请求**：
```http
GET /api/v1/scheduled-tasks?offset=0&limit=10
```

**成功响应（200 OK）**：
```json
{
  "tasks": [
    {
      "id": "scheduled-task-20240116-abc123",
      "name": "Daily Image Prewarm",
      "cronExpr": "0 0 * * *",
      "enabled": true,
      "nextExecutionAt": "2024-01-17T00:00:00Z"
    }
  ],
  "total": 5,
  "limit": 10,
  "offset": 0
}
```

#### 5.3 获取单个定时任务

**请求**：
```http
GET /api/v1/scheduled-tasks/:id
```

**成功响应（200 OK）**：
```json
{
  "id": "scheduled-task-20240116-abc123",
  "name": "Daily Image Prewarm",
  "cronExpr": "0 0 * * *",
  "enabled": true,
  "taskConfig": {...},
  "overlapPolicy": "skip",
  "timeoutSeconds": 3600,
  "lastExecutionAt": "2024-01-16T00:00:00Z",
  "nextExecutionAt": "2024-01-17T00:00:00Z",
  "createdBy": "admin",
  "createdAt": "2024-01-16T10:00:00Z",
  "updatedAt": "2024-01-16T10:00:00Z"
}
```

#### 5.4 更新定时任务

**请求**：
```http
PUT /api/v1/scheduled-tasks/:id
```

**请求体**（所有字段可选）**：
```json
{
  "name": "Updated Task Name",
  "cronExpr": "0 6 * * *",
  "enabled": true,
  "taskConfig": {
    "images": ["nginx:latest"]
  },
  "overlapPolicy": "allow",
  "timeoutSeconds": 1800
}
```

**成功响应（200 OK）**：
返回更新后的定时任务对象（同 GET 响应）

#### 5.5 删除定时任务

**请求**：
```http
DELETE /api/v1/scheduled-tasks/:id
```

**成功响应（200 OK）**：
```json
{
  "taskId": "scheduled-task-20240116-abc123",
  "status": "success",
  "message": "Scheduled task deleted successfully"
}
```

#### 5.6 启用定时任务

**请求**：
```http
PUT /api/v1/scheduled-tasks/:id/enable
```

**成功响应（200 OK）**：
```json
{
  "taskId": "scheduled-task-20240116-abc123",
  "status": "success",
  "message": "Scheduled task enabled successfully"
}
```

#### 5.7 禁用定时任务

**请求**：
```http
PUT /api/v1/scheduled-tasks/:id/disable
```

**成功响应（200 OK）**：
```json
{
  "taskId": "scheduled-task-20240116-abc123",
  "status": "success",
  "message": "Scheduled task disabled successfully"
}
```

#### 5.8 手动触发定时任务

**请求**：
```http
POST /api/v1/scheduled-tasks/:id/trigger
```

**成功响应（200 OK）**：
```json
{
  "taskId": "task-20240116-xyz789",
  "status": "success",
  "message": "Scheduled task triggered successfully"
}
```

#### 5.9 查询执行历史

**请求**：
```http
GET /api/v1/scheduled-tasks/:id/executions?offset=0&limit=10
```

**成功响应（200 OK）**：
```json
{
  "executions": [
    {
      "id": 1,
      "scheduledTaskId": "scheduled-task-20240116-abc123",
      "taskId": "task-20240116-xyz789",
      "status": "success",
      "startedAt": "2024-01-16T00:00:00Z",
      "finishedAt": "2024-01-16T00:10:00Z",
      "durationSeconds": 600.0,
      "errorMessage": "",
      "triggeredAt": "2024-01-16T00:00:00Z"
    }
  ],
  "total": 50,
  "limit": 10,
  "offset": 0
}
```

**执行状态**：
- `success`: 执行成功
- `failed`: 执行失败
- `skipped`: 跳过执行（重叠策略为 skip，上次任务仍在运行）
- `timeout`: 执行超时

#### 5.10 获取单次执行详情

**请求**：
```http
GET /api/v1/scheduled-tasks/:id/executions/:executionId
```

**成功响应（200 OK）**：
```json
{
  "id": 1,
  "scheduledTaskId": "scheduled-task-20240116-abc123",
  "taskId": "task-20240116-xyz789",
  "status": "success",
  "startedAt": "2024-01-16T00:00:00Z",
  "finishedAt": "2024-01-16T00:10:00Z",
  "durationSeconds": 600.0,
  "errorMessage": "",
  "triggeredAt": "2024-01-16T00:00:00Z"
}
```

---

### 6. 健康检查

**请求**：
```http
GET /healthz
```

**响应**：
```json
{
  "status": "healthy",
  "kubernetes": "connected",
  "timestamp": "2024-01-16T10:00:00Z"
}
```

---

### 6. Prometheus 指标

**请求**：
```http
GET /metrics
```

**响应**（Prometheus 格式）：
```
# HELP prewarm_tasks_total Total prewarm tasks
# TYPE prewarm_tasks_total counter
prewarm_tasks_total{status="completed"} 42
prewarm_tasks_total{status="failed"} 3

# HELP prewarm_task_duration_seconds Task duration
# TYPE prewarm_task_duration_seconds histogram
prewarm_task_duration_seconds_bucket{le="60"} 5
prewarm_task_duration_seconds_bucket{le="300"} 30
...
```

---

## 使用场景详解

### 场景 1：开发人员手动触发（curl）

```bash
# 1. 创建预热任务
curl -X POST http://prewarm-api.example.com/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["nginx:latest", "redis:7", "postgres:14"],
    "batchSize": 10
  }'

# 响应：
# {
#   "taskId": "task-abc123",
#   "status": "pending",
#   "createdAt": "2024-01-16T10:00:00Z"
# }

# 2. 查询任务状态
curl http://prewarm-api.example.com/api/v1/tasks/task-abc123

# 响应：
# {
#   "status": "running",
#   "progress": {
#     "totalNodes": 50,
#     "completedNodes": 25,
#     "percentage": 50.0
#   }
# }

# 3. 等待完成（循环查询）
while true; do
  status=$(curl -s http://prewarm-api/api/v1/tasks/task-abc123 | jq -r .status)
  progress=$(curl -s http://prewarm-api/api/v1/tasks/task-abc123 | jq -r .progress.percentage)
  echo "状态: $status, 进度: $progress%"

  if [ "$status" = "completed" ]; then
    echo "✅ 任务完成"
    break
  elif [ "$status" = "failed" ]; then
    echo "❌ 任务失败"
    exit 1
  fi

  sleep 5
done
```

**优势**：
- 简单直观
- 无需 Kubernetes 知识
- 易于调试

---

### 场景 2：Python 脚本集成

```python
#!/usr/bin/env python3
import requests
import time

class ImagePrewarmClient:
    """镜像预热服务客户端"""

    def __init__(self, base_url):
        self.base_url = base_url.rstrip('/')

    def create_task(self, images, batch_size=10, node_selector=None):
        """创建预热任务"""
        payload = {
            "images": images,
            "batchSize": batch_size
        }
        if node_selector:
            payload["nodeSelector"] = node_selector

        resp = requests.post(
            f"{self.base_url}/api/v1/tasks",
            json=payload
        )
        resp.raise_for_status()
        return resp.json()

    def get_task(self, task_id):
        """查询任务状态"""
        resp = requests.get(f"{self.base_url}/api/v1/tasks/{task_id}")
        resp.raise_for_status()
        return resp.json()

    def list_tasks(self, status=None, limit=10):
        """列出任务"""
        params = {"limit": limit}
        if status:
            params["status"] = status

        resp = requests.get(f"{self.base_url}/api/v1/tasks", params=params)
        resp.raise_for_status()
        return resp.json()

    def cancel_task(self, task_id):
        """取消任务"""
        resp = requests.delete(f"{self.base_url}/api/v1/tasks/{task_id}")
        resp.raise_for_status()
        return resp.json()

    def wait_for_completion(self, task_id, timeout=600, poll_interval=5):
        """等待任务完成"""
        start_time = time.time()

        while time.time() - start_time < timeout:
            task = self.get_task(task_id)
            status = task["status"]

            if status == "completed":
                return task
            elif status == "failed":
                raise Exception(f"Task failed: {task}")
            elif status == "cancelled":
                raise Exception(f"Task was cancelled")

            # 打印进度
            progress = task.get("progress", {})
            percentage = progress.get("percentage", 0)
            print(f"进度: {percentage:.1f}% ({progress.get('completedNodes', 0)}/{progress.get('totalNodes', 0)})")

            time.sleep(poll_interval)

        raise TimeoutError(f"Task did not complete within {timeout} seconds")


# 使用示例
if __name__ == "__main__":
    client = ImagePrewarmClient("http://prewarm-api.example.com")

    # 创建任务
    print("创建预热任务...")
    task = client.create_task(
        images=["nginx:latest", "redis:7", "postgres:14"],
        batch_size=10
    )
    task_id = task["taskId"]
    print(f"任务已创建: {task_id}")

    # 等待完成
    print("等待任务完成...")
    result = client.wait_for_completion(task_id)

    # 打印结果
    progress = result["progress"]
    print(f"\n✅ 任务完成!")
    print(f"总节点: {progress['totalNodes']}")
    print(f"成功: {progress['completedNodes']}")
    print(f"失败: {progress['failedNodes']}")
    print(f"成功率: {progress['percentage']:.1f}%")
```

**使用**：
```bash
python prewarm_client.py
```

---

### 场景 3：Jenkins Pipeline 集成

```groovy
// Jenkinsfile
pipeline {
    agent any

    environment {
        PREWARM_API = 'http://prewarm-api.example.com'
        APP_VERSION = "${BUILD_NUMBER}"
    }

    stages {
        stage('Build') {
            steps {
                sh 'docker build -t myapp:${APP_VERSION} .'
                sh 'docker push myapp:${APP_VERSION}'
            }
        }

        stage('Deploy') {
            steps {
                sh 'kubectl set image deployment/myapp app=myapp:${APP_VERSION}'
            }
        }

        stage('Prewarm Images') {
            steps {
                script {
                    // 自动预热刚部署的镜像
                    def images = [
                        "myapp:${APP_VERSION}",
                        "nginx:latest",
                        "redis:7"
                    ]

                    // 调用预热 API
                    def response = httpRequest(
                        url: "${PREWARM_API}/api/v1/tasks",
                        httpMode: 'POST',
                        contentType: 'APPLICATION_JSON',
                        requestBody: groovy.json.JsonOutput.toJson([
                            images: images,
                            batchSize: 20
                        ])
                    )

                    def task = readJSON(text: response.content)
                    def taskId = task.taskId
                    echo "✅ 预热任务已创建: ${taskId}"

                    // 等待完成（可选：可以不等待，异步执行）
                    timeout(time: 10, unit: 'MINUTES') {
                        waitUntil {
                            def statusResp = httpRequest(
                                url: "${PREWARM_API}/api/v1/tasks/${taskId}"
                            )
                            def taskStatus = readJSON(text: statusResp.content)

                            echo "进度: ${taskStatus.progress.percentage}%"

                            return taskStatus.status in ['completed', 'failed']
                        }
                    }

                    echo "✅ 镜像预热完成"
                }
            }
        }
    }
}
```

---

### 场景 4：GitLab CI 集成

```yaml
# .gitlab-ci.yml
stages:
  - build
  - deploy
  - prewarm

build:
  stage: build
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA

deploy:
  stage: deploy
  script:
    - kubectl set image deployment/myapp app=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA

prewarm:
  stage: prewarm
  script:
    - |
      TASK_ID=$(curl -X POST http://prewarm-api/api/v1/tasks \
        -H "Content-Type: application/json" \
        -d "{\"images\": [\"$CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA\"], \"batchSize\": 20}" \
        | jq -r .taskId)

      echo "预热任务: $TASK_ID"

      # 等待完成
      while true; do
        STATUS=$(curl -s http://prewarm-api/api/v1/tasks/$TASK_ID | jq -r .status)
        PROGRESS=$(curl -s http://prewarm-api/api/v1/tasks/$TASK_ID | jq -r .progress.percentage)

        echo "状态: $STATUS, 进度: $PROGRESS%"

        if [ "$STATUS" = "completed" ]; then
          echo "✅ 预热完成"
          break
        elif [ "$STATUS" = "failed" ]; then
          echo "❌ 预热失败"
          exit 1
        fi

        sleep 5
      done
```

---

### 场景 5：Web UI 管理界面

**功能需求**：
- 创建预热任务
- 查看任务列表
- 查看任务详情（实时进度）
- 取消正在运行的任务
- 查看历史记录

**技术栈**：
- 前端：React / Vue
- UI 框架：Ant Design / Element UI
- 图表：ECharts（进度可视化）

**界面示例**：

```
┌─────────────────────────────────────────────────────┐
│  镜像预热服务管理平台                                │
├─────────────────────────────────────────────────────┤
│                                                      │
│  [创建新任务]                                        │
│                                                      │
│  镜像列表：                                          │
│  ┌────────────────────────────────────────────┐    │
│  │ nginx:latest                               │    │
│  │ redis:7                                    │    │
│  │ postgres:14                                │    │
│  └────────────────────────────────────────────┘    │
│                                                      │
│  批次大小：[10] ▼                                    │
│                                                      │
│  节点选择器（可选）：                                │
│  标签: [workload=compute]                           │
│                                                      │
│  [开始预热]  [重置]                                  │
│                                                      │
├─────────────────────────────────────────────────────┤
│  当前任务                                            │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │ task-abc123  运行中  50%  [取消]            │  │
│  │ ████████████░░░░░░░░░░░░  25/50 节点       │  │
│  │ 开始时间: 10:00:05  预计完成: 10:15:00      │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │ task-def456  已完成  100%  [详情]          │  │
│  │ ████████████████████████  30/30 节点       │  │
│  │ 耗时: 15分钟  成功率: 100%                  │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
└─────────────────────────────────────────────────────┘
```

---

## 客户端工具对比

### 对比矩阵

| 客户端 | 适用场景 | 学习成本 | 自动化能力 | 示例 |
|--------|---------|---------|-----------|------|
| **curl** | 快速测试<br>手动操作 | 低 | 可脚本化 | `curl -X POST ...` |
| **Python SDK** | 自动化脚本<br>批量操作 | 低 | 强 | `client.create_task(...)` |
| **Go SDK** | Go 项目集成 | 低 | 强 | `client.CreateTask(...)` |
| **Web UI** | 运维管理<br>可视化 | 无 | 中 | 浏览器点击操作 |
| **kubectl plugin** | K8s 用户习惯 | 低 | 中 | `kubectl prewarm ...` |
| **Jenkins** | CI/CD 集成 | 中 | 强 | Pipeline 步骤 |
| **GitLab CI** | CI/CD 集成 | 中 | 强 | .gitlab-ci.yml |

---

## 项目结构

```
ips/
├── cmd/
│   └── apiserver/
│       └── main.go                 # HTTP 服务入口
│
├── internal/
│   ├── api/
│   │   ├── handler/
│   │   │   ├── task.go            # 任务相关 API handler
│   │   │   └── health.go          # 健康检查 handler
│   │   ├── middleware/
│   │   │   ├── auth.go            # 认证中间件
│   │   │   └── logging.go         # 日志中间件
│   │   └── router.go               # 路由配置
│   │
│   ├── service/
│   │   ├── task_manager.go        # 任务管理服务
│   │   ├── node_filter.go         # 节点过滤服务
│   │   ├── batch_scheduler.go     # 批次调度服务
│   │   └── status_tracker.go      # 状态跟踪服务
│   │
│   ├── repository/
│   │   ├── memory.go              # 内存存储实现
│   │   └── redis.go               # Redis 存储实现（可选）
│   │
│   └── k8s/
│       ├── client.go              # Kubernetes 客户端
│       ├── job_creator.go         # Job 创建器
│       └── job_watcher.go         # Job 状态监控
│
├── pkg/
│   ├── models/
│   │   └── task.go                # 任务数据模型
│   └── metrics/
│       └── prometheus.go          # Prometheus 指标
│
├── deploy/
│   ├── rbac.yaml                  # RBAC 权限
│   ├── deployment.yaml            # API 服务部署
│   ├── service.yaml               # Service 配置
│   └── ingress.yaml               # Ingress 配置
│
├── client/
│   ├── python/
│   │   └── prewarm_client.py     # Python SDK
│   ├── go/
│   │   └── client.go              # Go SDK
│   └── kubectl-prewarm            # kubectl 插件
│
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

**代码量估算**：
- API 层：~150 行
- Service 层：~200 行
- Repository 层：~50 行
- K8s 交互层：~100 行
- **总计：~500 行 Go 代码**

---

## 核心组件职责

### 1. HTTP API Handler

**职责**：
- 接收 HTTP 请求
- 参数校验和绑定
- 调用 Service 层
- 返回 JSON 响应
- 错误处理

**不负责**：
- ❌ 业务逻辑
- ❌ Kubernetes 操作
- ❌ 数据存储

---

### 2. Task Manager Service

**职责**：
- 任务生命周期管理
- 调用节点过滤器获取节点
- 调用批次调度器执行任务
- 更新任务状态
- 清理完成的任务

**不负责**：
- ❌ HTTP 请求处理
- ❌ 直接操作 Kubernetes
- ❌ 数据持久化细节

---

### 3. Batch Scheduler

**职责**：
- 将节点列表分批
- 为每批次创建 Worker Job
- 等待批次完成
- 记录进度

**执行模式**：
- 同步执行（阻塞直到完成）
- 或异步执行（后台 goroutine）

---

### 4. Status Tracker

**职责**：
- 监听 Kubernetes Job 状态变化
- 更新任务进度
- 记录失败节点
- 计算预计完成时间

**实现方式**：
- 定期轮询（简单）
- 或 Watch 机制（高效）

---

### 5. Repository

**职责**：
- 存储任务数据
- 查询任务数据
- 更新任务状态

**接口设计**：
```go
type TaskRepository interface {
    Create(task *Task) error
    Get(id string) (*Task, error)
    List(filter TaskFilter) ([]*Task, error)
    Update(task *Task) error
    Delete(id string) error
}
```

**实现**：
- MVP：内存实现（sync.Map）
- 生产：Redis 实现

---

## 部署架构

### MVP 部署（单实例 + 内存存储）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prewarm-api
spec:
  replicas: 1  # MVP 单实例
  selector:
    matchLabels:
      app: prewarm-api
  template:
    metadata:
      labels:
        app: prewarm-api
    spec:
      serviceAccountName: prewarm-api
      containers:
      - name: api
        image: prewarm-api:v1.0
        ports:
        - containerPort: 8080
        env:
        - name: STORAGE_TYPE
          value: "memory"  # 内存存储
        - name: LOG_LEVEL
          value: "info"
```

### 生产部署（多实例 + Redis）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prewarm-api
spec:
  replicas: 2  # 高可用
  selector:
    matchLabels:
      app: prewarm-api
  template:
    metadata:
      labels:
        app: prewarm-api
    spec:
      serviceAccountName: prewarm-api
      containers:
      - name: api
        image: prewarm-api:v1.0
        ports:
        - containerPort: 8080
        env:
        - name: STORAGE_TYPE
          value: "redis"
        - name: REDIS_URL
          value: "redis://prewarm-redis:6379"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

---

## 任务执行模式

### 模式选择

#### 模式 1：同步执行（简单）

```go
func (h *TaskHandler) CreateTask(c *gin.Context) {
    // 1. 创建任务
    task := &Task{ID: generateID(), Status: TaskPending}
    h.repo.Create(task)

    // 2. 返回任务 ID（立即返回）
    c.JSON(201, task)

    // 3. 后台执行（goroutine）
    go h.taskManager.Execute(task)  // 异步执行
}
```

**优势**：
- API 立即返回，不阻塞
- 用户通过轮询查询进度
- 简单易实现

**劣势**：
- 需要轮询（稍微增加 API 调用）

---

#### 模式 2：WebSocket 实时推送（高级）

```go
// WebSocket 端点
GET /api/v1/tasks/:id/watch

// 客户端连接后，服务器实时推送进度
{
  "type": "progress",
  "data": {
    "completedNodes": 25,
    "totalNodes": 50,
    "percentage": 50.0
  }
}
```

**优势**：
- 实时进度推送
- 无需轮询
- 用户体验更好

**劣势**：
- 实现复杂度增加
- 需要维护 WebSocket 连接

**建议**：MVP 用模式 1，后续可选模式 2

---

## 复杂度对比

### 最终方案对比

| 方案 | 代码量 | 开发时间 | 易用性 | 集成性 | 扩展性 | 推荐度 |
|------|--------|---------|--------|--------|--------|--------|
| CRD+Controller | 1500行 | 2-3周 | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ 过度 |
| Job+ConfigMap | 370行 | 2-3天 | ⭐⭐⭐ | ⭐ | ⭐⭐ | ❌ 复杂 |
| Job+命令行参数 | 320行 | 2-3天 | ⭐⭐⭐⭐ | ⭐ | ⭐⭐ | ❌ 局限 |
| **RESTful API** | **500行** | **3-4天** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **⭐⭐⭐⭐⭐** | **✅ 最佳** |

### 核心指标对比

**易用性评分**：
- CRD：需要理解 Operator、CRD、kubectl（2/5）
- Job+ConfigMap：需要理解 ConfigMap、Job、kubectl（3/5）
- Job+参数：需要理解 Job、kubectl（4/5）
- **RESTful API：只需要 HTTP 客户端（5/5）** ✅

**集成性评分**：
- CRD：仅 kubectl 和 K8s API（4/5）
- Job+ConfigMap：仅 kubectl（1/5）
- Job+参数：仅 kubectl（1/5）
- **RESTful API：curl、SDK、Web、CI/CD 全支持（5/5）** ✅

**扩展性评分**：
- CRD：最灵活，但复杂（5/5）
- Job+ConfigMap：难以扩展（2/5）
- Job+参数：难以扩展（2/5）
- **RESTful API：易于添加功能（5/5）** ✅

---

## 实施路线图

### 阶段 1：MVP（3 天）

**目标**：实现核心 API，验证可行性

**功能**：
- ✅ POST /api/v1/tasks - 创建任务
- ✅ GET /api/v1/tasks/:id - 查询任务
- ✅ 内存存储
- ✅ 基本的批次调度
- ✅ 健康检查

**不包含**：
- ❌ Redis 存储
- ❌ 认证授权
- ❌ 任务取消
- ❌ 高级过滤

---

### 阶段 2：完善（1-2 天）

**功能**：
- ✅ GET /api/v1/tasks - 列出任务
- ✅ DELETE /api/v1/tasks/:id - 取消任务
- ✅ Prometheus 指标
- ✅ 详细日志
- ✅ 错误处理

---

### 阶段 3：增强（可选，1-2 天）

**功能**：
- ✅ Redis 存储（持久化）
- ✅ Token 认证
- ✅ Python SDK
- ✅ kubectl 插件
- ✅ Web UI（简单版）

---

### 阶段 4：生产化（可选，1-2 天）

**功能**：
- ✅ 多实例部署
- ✅ 速率限制
- ✅ 审计日志
- ✅ 告警集成
- ✅ 完整文档

---

## 与 CRD 方案的终极对比

### CRD 方案的优势

✅ Kubernetes 原生（kubectl get prewarmtasks）
✅ 声明式 API
✅ 生态集成（ArgoCD、Flux 等可直接管理）

### CRD 方案的劣势

❌ 复杂度高（1500 行代码）
❌ 开发周期长（2-3 周）
❌ 学习成本高（需要理解 Operator 模式）
❌ 客户端单一（只能 kubectl）
❌ 难以提供 Web UI

### RESTful API 方案的优势

✅ 复杂度适中（500 行代码）
✅ 开发周期短（3-4 天）
✅ 学习成本低（HTTP API）
✅ **客户端多样化（curl、SDK、Web UI、CI/CD）** ⭐
✅ **易于集成** ⭐
✅ **用户体验最好** ⭐

### RESTful API 方案的劣势

⚠️ 不是 Kubernetes 原生 API（但这不重要，用户不需要知道）
⚠️ 需要部署 API 服务（但这是必要的）

---

## 使用体验对比

### CRD 方案使用体验

```yaml
# 1. 用户需要写 YAML
apiVersion: prewarm.io/v1alpha1
kind: PrewarmTask
metadata:
  name: my-prewarm
spec:
  images:
    - name: nginx:latest
    - name: redis:7
  batchSize: 10

# 2. 用户需要 kubectl
kubectl apply -f prewarm.yaml

# 3. 用户查看状态
kubectl get prewarmtasks
kubectl describe prewarmtask my-prewarm
```

**学习成本**：需要理解 CRD、kubectl、YAML

---

### RESTful API 使用体验

```bash
# 1. 简单的 HTTP 请求
curl -X POST http://prewarm-api/api/v1/tasks \
  -d '{"images": ["nginx:latest", "redis:7"], "batchSize": 10}'

# 响应：{"taskId": "task-abc123"}

# 2. 查询状态
curl http://prewarm-api/api/v1/tasks/task-abc123

# 响应：{"status": "running", "progress": {"percentage": 50}}
```

**学习成本**：只需要会 curl（几乎所有开发者都会）

---

## 集成能力对比

### CRD 方案集成

```python
# Python 集成（需要 kubernetes 库）
from kubernetes import client, config

config.load_incluster_config()
api = client.CustomObjectsApi()

# 创建任务（复杂）
body = {
    "apiVersion": "prewarm.io/v1alpha1",
    "kind": "PrewarmTask",
    "metadata": {"name": "my-task"},
    "spec": {
        "images": [{"name": "nginx:latest"}],
        "batchSize": 10
    }
}
api.create_namespaced_custom_object(
    group="prewarm.io",
    version="v1alpha1",
    namespace="default",
    plural="prewarmtasks",
    body=body
)
```

**问题**：
- 需要 kubernetes Python 库（重量级）
- 需要理解 CRD 的结构
- 需要 K8s 集群访问权限
- 代码冗长

---

### RESTful API 集成

```python
# Python 集成（只需要 requests 库）
import requests

# 创建任务（简单）
resp = requests.post(
    "http://prewarm-api/api/v1/tasks",
    json={"images": ["nginx:latest"], "batchSize": 10}
)
task = resp.json()
```

**优势**：
- 只需要标准 HTTP 库
- 代码简洁
- 无需 K8s 访问权限
- 易于理解

---

## 总结

### 方案最终确定

**推荐方案**：RESTful API 服务

### 核心理由

✅ **用户体验最好**
```
用户视角：我不需要知道底层是 Kubernetes
我只需要：调用 API → 得到结果
```

✅ **集成能力最强**
```
支持所有 HTTP 客户端：
- curl（命令行快速测试）
- Python SDK（自动化脚本）
- Web UI（运维人员友好）
- CI/CD（Jenkins、GitLab 等）
- 其他服务（微服务调用）
```

✅ **复杂度适中**
```
代码量：500 行（比 CRD 少 1000 行）
开发时间：3-4 天（比 CRD 少 2 周）
维护成本：低（标准 HTTP 服务）
```

✅ **扩展性好**
```
未来可以轻松添加：
- 更多 API 端点
- Web UI
- SDK（各种语言）
- Webhook 通知
- 更复杂的调度策略
```

### 关键价值

> **将底层 Kubernetes 复杂性封装为简单的 HTTP API**

这就是真正的"服务化"：
- 用户只需要知道"调用什么 API"
- 不需要知道"底层怎么实现"
- 这才是符合现代软件架构的设计

### 适用场景

✅ **完美适合**：
- 单团队或多团队使用
- 需要 CI/CD 集成
- 需要提供 Web UI
- 需要给非技术人员使用
- 需要自动化编排
- 需要历史记录和审计

### KISS 原则的最终实践

> "The best code is no code at all. The second best is simple code."

我们选择的是：
- ❌ 不追求"Kubernetes Native"（那是实现细节）
- ❌ 不追求"架构完美"（CRD 虽然优雅，但过于复杂）
- ✅ 追求"用户体验"（用户最容易使用的方案）
- ✅ 追求"实用主义"（解决问题，而不是炫技）

---

## 开发计划

### 第 1 天：搭建框架
- HTTP 服务器（Gin）
- 基本路由
- 任务模型定义
- 内存存储

### 第 2 天：核心逻辑
- 任务管理器
- 节点过滤
- 批次调度
- Job 创建

### 第 3 天：完善功能
- 状态跟踪
- 错误处理
- 健康检查
- 日志记录

### 第 4 天：测试和文档
- 单元测试
- 集成测试
- API 文档
- 使用示例

---

## 快速参考

### 典型使用流程

```bash
# 1. 创建任务
TASK_ID=$(curl -X POST http://prewarm-api/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"images": ["nginx:latest", "redis:7"], "batchSize": 10}' \
  | jq -r .taskId)

echo "任务创建: $TASK_ID"

# 2. 等待完成
while true; do
  STATUS=$(curl -s http://prewarm-api/api/v1/tasks/$TASK_ID | jq -r .status)
  PROGRESS=$(curl -s http://prewarm-api/api/v1/tasks/$TASK_ID | jq -r .progress.percentage)

  echo "[$STATUS] 进度: $PROGRESS%"

  [ "$STATUS" = "completed" ] && break
  sleep 5
done

echo "✅ 预热完成"
```

### API 速查

| 端点 | 方法 | 用途 | 示例 |
|------|------|------|------|
| `/api/v1/tasks` | POST | 创建任务 | `curl -X POST ... -d '{...}'` |
| `/api/v1/tasks` | GET | 列出任务 | `curl http://.../tasks` |
| `/api/v1/tasks/:id` | GET | 查询详情 | `curl http://.../tasks/task-123` |
| `/api/v1/tasks/:id` | DELETE | 取消任务 | `curl -X DELETE http://.../tasks/task-123` |
| `/healthz` | GET | 健康检查 | `curl http://.../healthz` |
| `/metrics` | GET | Prometheus | `curl http://.../metrics` |

---

## 写在最后

### 方案演进的启示

经过 4 轮深度思考，我们发现：

1. **第一轮**：追求"Kubernetes Native"（CRD）→ 发现过度设计
2. **第二轮**：简化为 Job + ConfigMap → 发现 ConfigMap 不必要
3. **第三轮**：去掉 ConfigMap → 发现用户还是要懂 kubectl
4. **第四轮**：RESTful API → 找到了最佳方案

### 关键教训

> **技术选择应该以用户体验为中心，而不是以技术优雅为中心**

- CRD 在技术上很优雅，但用户体验不好
- RESTful API 在技术上更传统，但用户体验最好

**用户不关心你用什么技术实现，用户只关心是否好用。**

### 最终结论

**RESTful API 服务是镜像预热场景的最佳方案**

理由：
- ✅ 用户体验最好（无需 K8s 知识）
- ✅ 集成能力最强（支持所有客户端）
- ✅ 复杂度适中（500 行，3-4 天）
- ✅ 扩展性优秀（易于添加功能）
- ✅ 真正的服务化（符合现代架构）

**这就是 KISS 原则与用户体验完美结合的范例！**
