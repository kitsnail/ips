# Image Prewarm Service (IPS)

镜像预热服务 - 一个用于在 Kubernetes 集群中批量预热容器镜像的 RESTful API 服务。

## 功能特性

### 核心功能
- ✅ RESTful API 接口，易于集成
- ✅ **Web UI 管理界面**（可视化任务管理）
- ✅ 批次调度，支持自定义批次大小
- ✅ 节点选择器，支持按标签过滤节点
- ✅ 实时进度跟踪（基于 Kubernetes Watch 机制）
- ✅ 失败节点详情记录
- ✅ 任务生命周期管理（创建、查询、取消）

### 高级特性
- ✅ **任务优先级队列**（1-10 级，支持紧急任务优先执行）
- ✅ **自动重试机制**（支持线性和指数退避策略）
- ✅ **Webhook 通知**（任务完成/失败/取消自动通知）
- ✅ **并发控制**（防止资源耗尽，默认最大 3 个并发任务）
- ✅ **Prometheus 监控指标**（任务统计、耗时、节点处理等 9 种指标）

### 部署与运维
- ✅ 内存存储（轻量级，适合短期任务）
- ✅ Docker 容器化支持
- ✅ Kubernetes 完整部署配置
- ✅ 水平自动扩缩容（HPA）
- ✅ 健康检查和优雅关闭
- ✅ 完善的单元测试和集成测试

## 快速开始

### 方式一：本地开发

#### 前提条件

- Go 1.23+
- Kubernetes 集群访问权限
- kubectl 配置（本地测试）或 in-cluster 配置（生产环境）

#### 编译

```bash
make build
```

#### 运行

```bash
# 使用默认配置运行
make run

# 或直接运行二进制文件
./bin/apiserver

# 使用环境变量配置
SERVER_PORT=8080 K8S_NAMESPACE=default WORKER_IMAGE=busybox:latest ./bin/apiserver
```

#### 测试

```bash
# 运行自动化测试脚本
./test-api.sh

# 或手动测试
curl http://localhost:8080/health
```

### 使用 Web UI

服务启动后，可以通过浏览器访问 Web 管理界面：

```
http://localhost:8080/
或
http://localhost:8080/web/
```

Web UI 功能：
- 📊 **任务列表视图** - 实时查看所有任务状态和进度
- ➕ **创建任务** - 可视化表单创建新的镜像预热任务
- 🔍 **任务详情** - 查看任务完整信息、进度和失败节点详情
- 🎯 **状态筛选** - 按任务状态快速筛选
- ❌ **取消任务** - 一键取消运行中的任务
- 🔄 **自动刷新** - 每 5 秒自动更新任务状态

### 使用 API

查看 [API 文档](#api-文档) 了解如何通过 RESTful API 管理任务。

### 方式二：Docker 部署

#### 使用 Docker

```bash
# 构建镜像
make docker-build

# 运行容器
docker run -d \
  --name ips-apiserver \
  -p 8080:8080 \
  -v ~/.kube/config:/home/ips/.kube/config:ro \
  -e K8S_NAMESPACE=default \
  -e WORKER_IMAGE=busybox:latest \
  ips-apiserver:latest
```

#### 使用 Docker Compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 方式三：Kubernetes 部署

```bash
# 快速部署
make k8s-deploy

# 或手动部署
kubectl apply -f deploy/

# 查看部署状态
kubectl get all -n ips-system

# 查看日志
kubectl logs -l app=ips -n ips-system -f
```

详细的部署指南请参考 [deploy/DEPLOYMENT.md](deploy/DEPLOYMENT.md)。

## API 文档

### 健康检查

```bash
# 健康检查
GET /health

# 就绪检查 (同 /health)
GET /readyz
```

### 创建任务

```bash
POST /api/v1/tasks
Content-Type: application/json

{
  "images": ["nginx:latest", "redis:7"],
  "batchSize": 10,
  "priority": 5,              # 可选，优先级 1-10，默认 5
  "maxRetries": 3,            # 可选，最大重试次数 0-5，默认 0
  "retryStrategy": "linear",  # 可选，重试策略: linear 或 exponential，默认 linear
  "retryDelay": 30,           # 可选，重试延迟（秒），默认 30
  "webhookUrl": "https://example.com/webhook",  # 可选，任务完成通知
  "nodeSelector": {           # 可选，节点选择器
    "workload": "compute"
  }
}
```

**响应示例：**

```json
{
  "taskId": "task-20260116-151234-a1b2c3d4",
  "status": "pending",
  "priority": 5,
  "images": ["nginx:latest", "redis:7"],
  "batchSize": 10,
  "maxRetries": 3,
  "retryCount": 0,
  "retryStrategy": "linear",
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
│   └── models/             # 数据模型
├── deploy/                 # K8s 部署配置
│   ├── namespace.yaml      # 命名空间
│   ├── rbac.yaml           # RBAC 权限
│   ├── configmap.yaml      # 配置
│   ├── deployment.yaml     # 部署配置
│   ├── service.yaml        # 服务
│   ├── ingress.yaml        # Ingress
│   ├── hpa.yaml            # 水平自动扩缩容
│   ├── pdb.yaml            # Pod 中断预算
│   ├── resource-quota.yaml # 资源配额
│   ├── kustomization.yaml  # Kustomize 配置
│   └── DEPLOYMENT.md       # 部署指南
├── client/                 # 客户端 SDK
│   └── python/             # Python 客户端
├── Dockerfile              # Docker 镜像构建
├── docker-compose.yml      # Docker Compose 配置
├── Makefile                # 构建和部署命令
└── README.md               # 项目文档
```

## 开发

### 常用命令

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test

# 清理构建产物
make clean

# 构建二进制文件
make build

# 本地运行
make run
```

### Docker 相关

```bash
# 构建 Docker 镜像
make docker-build

# 运行 Docker 容器
make docker-run

# 停止 Docker 容器
make docker-stop

# 使用 Docker Compose
make docker-compose-up
make docker-compose-down
```

- push
```bash
skopeo copy \
  --override-os linux \
  --override-arch arm64 \
  --dest-tls-verify=false \
  --dest-creds admin:Harbor12345 \
  docker-daemon:ips-apiserver:latest \
  docker://cr01.home.lan/library/ips-apiserver:latest
```

### Kubernetes 相关

```bash
# 部署到 Kubernetes
make k8s-deploy

# 查看部署状态
make k8s-status

# 查看日志
make k8s-logs

# 删除部署
make k8s-delete

# 端口转发（本地访问）
make k8s-port-forward
```

### 完整命令列表

运行 `make help` 查看所有可用命令。

## 文档

- [API 设计文档](RESTful-API.md)
- [部署指南](deploy/DEPLOYMENT.md)
- [开发计划](development-plan.md)
- [架构设计](plan-arch.md)

## License

MIT
