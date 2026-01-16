# Image Prewarm Service (IPS)

镜像预热服务 - 一个用于在 Kubernetes 集群中批量预热容器镜像的 RESTful API 服务。

## 功能特性

- ✅ RESTful API 接口，易于集成
- ✅ 批次调度，支持自定义批次大小
- ✅ 节点选择器，支持按标签过滤节点
- ✅ 实时进度跟踪
- ✅ 失败节点详情记录
- ✅ 任务生命周期管理（创建、查询、取消）
- ✅ 内存存储（MVP）

## 快速开始

### 前提条件

- Go 1.21+
- Kubernetes 集群访问权限
- kubectl 配置（本地测试）或 in-cluster 配置（生产环境）

### 编译

```bash
make build
```

### 运行

```bash
# 使用默认配置运行
make run

# 或直接运行二进制文件
./bin/apiserver

# 使用环境变量配置
SERVER_PORT=8080 K8S_NAMESPACE=default WORKER_IMAGE=busybox:latest ./bin/apiserver
```

### 测试

```bash
# 运行自动化测试脚本
./test-api.sh

# 或手动测试
curl http://localhost:8080/healthz
```

## API 文档

### 健康检查

```bash
# 健康检查
GET /healthz

# 就绪检查
GET /readyz
```

### 创建任务

```bash
POST /api/v1/tasks
Content-Type: application/json

{
  "images": ["nginx:latest", "redis:7"],
  "batchSize": 10,
  "nodeSelector": {
    "workload": "compute"  # 可选
  }
}
```

**响应示例：**

```json
{
  "taskId": "task-20260116-151234-a1b2c3d4",
  "status": "pending",
  "images": ["nginx:latest", "redis:7"],
  "batchSize": 10,
  "createdAt": "2026-01-16T15:12:34Z"
}
```

### 查询任务详情

```bash
GET /api/v1/tasks/:id
```

**响应示例：**

```json
{
  "taskId": "task-20260116-151234-a1b2c3d4",
  "status": "running",
  "images": ["nginx:latest", "redis:7"],
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
      "reason": "JobFailed",
      "message": "ImagePullBackOff",
      "timestamp": "2026-01-16T15:15:30Z"
    }
  ],
  "createdAt": "2026-01-16T15:12:34Z",
  "startedAt": "2026-01-16T15:12:35Z"
}
```

### 列出任务

```bash
# 列出所有任务
GET /api/v1/tasks

# 按状态过滤
GET /api/v1/tasks?status=running&limit=20&offset=0
```

**查询参数：**
- `status`: 任务状态（pending/running/completed/failed/cancelled）
- `limit`: 返回数量（默认 10，最大 100）
- `offset`: 偏移量（用于分页）

**响应示例：**

```json
{
  "tasks": [
    {
      "taskId": "task-20260116-151234-a1b2c3d4",
      "status": "running",
      "images": ["nginx:latest"],
      "progress": {
        "totalNodes": 50,
        "completedNodes": 25,
        "percentage": 50.0
      },
      "createdAt": "2026-01-16T15:12:34Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### 取消任务

```bash
DELETE /api/v1/tasks/:id
```

**响应示例：**

```json
{
  "taskId": "task-20260116-151234-a1b2c3d4",
  "status": "cancelled",
  "message": "Task cancelled successfully"
}
```

## 环境变量配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | HTTP 服务端口 | `8080` |
| `K8S_NAMESPACE` | Kubernetes 命名空间 | `default` |
| `WORKER_IMAGE` | Worker 镜像 | `busybox:latest` |

## 使用示例

### 使用 curl

```bash
# 创建预热任务
TASK_ID=$(curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 10
  }' | jq -r .taskId)

echo "Task created: $TASK_ID"

# 查询任务状态
curl http://localhost:8080/api/v1/tasks/$TASK_ID | jq .

# 取消任务
curl -X DELETE http://localhost:8080/api/v1/tasks/$TASK_ID | jq .
```

### 使用 Python

参见 [client/python/](client/python/) 目录。

## 架构设计

详细架构和开发计划请参考：
- [RESTful-API.md](RESTful-API.md) - API 设计文档
- [development-plan.md](development-plan.md) - 开发流程方案

## 项目结构

```
ips/
├── cmd/
│   └── apiserver/          # HTTP 服务入口
├── internal/
│   ├── api/                # HTTP 路由和中间件
│   │   ├── handler/        # API 处理器
│   │   └── middleware/     # 中间件
│   ├── service/            # 业务逻辑层
│   ├── repository/         # 存储层
│   └── k8s/                # K8s 客户端封装
├── pkg/
│   ├── models/             # 数据模型
│   └── metrics/            # Prometheus 指标
├── deploy/                 # K8s 部署配置
└── client/                 # 客户端 SDK
```

## 开发

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test

# 清理构建产物
make clean
```

## License

MIT
