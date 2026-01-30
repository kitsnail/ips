# IPS 部署指南

本文档提供 IPS (Image Prewarm Service) 的 Kubernetes 部署指南。

## 目录

- [快速部署](#快速部署)
- [配置说明](#配置说明)
- [验证部署](#验证部署)
- [故障排查](#故障排查)
- [卸载](#卸载)

---

## 快速部署

### 前置要求

- Kubernetes 1.24+
- kubectl 命令行工具
- 集群管理员权限（用于创建 RBAC 资源）

### 一键部署

```bash
# 1. 应用所有配置
kubectl apply -f deploy/

# 2. 等待 Pod 就绪
kubectl wait --for=condition=ready pod -l app=ips -n ips --timeout=300s

# 3. 查看部署状态
kubectl get all -n ips
```

### 获取访问地址

```bash
# 获取 Service 外部 IP
kubectl get svc ips-apiserver -n ips

# 访问健康检查
curl http://<EXTERNAL-IP>:8080/health
```

---

## 配置说明

### 环境变量

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_PORT` | 服务监听端口 | `8080` |
| `K8S_NAMESPACE` | 创建预热 Job 的命名空间 | `ips` |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |

### ConfigMap 配置

编辑 `deploy/configmap.yaml` 修改配置：

```yaml
data:
  SERVER_PORT: "8080"
  K8S_NAMESPACE: "ips"
  LOG_LEVEL: "info"
```

修改后重新应用：

```bash
kubectl apply -f deploy/configmap.yaml
kubectl rollout restart deployment/ips-apiserver -n ips
```

### Service 配置

项目使用 **LoadBalancer** 类型 Service，通过 MetalLB 自动分配外部 IP。

**文件**: `deploy/service.yaml`

**特点**：
- 自动分配外部 IP（通过 MetalLB）
- 直接从集群外部访问
- 配置了会话亲和性（Session Affinity），确保多副本部署时同一客户端请求路由到同一个 Pod

---

### 资源配置

默认资源配置（`deploy/deployment.yaml`）：

- **Requests**: CPU 500m, Memory 512Mi
- **Limits**: CPU 1000m, Memory 2Gi

根据实际负载调整资源配置。

### RBAC 权限

ServiceAccount 拥有以下权限（`deploy/rbac.yaml`）：

- **nodes**: get, list, watch
- **jobs**: get, list, watch, create, delete
- **pods**: get, list, watch
- **secrets**: get, create, delete（用于私有镜像仓库认证）

**注意**: 如果需要使用私有镜像仓库认证，Secrets 权限已包含在 RBAC 中。

---

## 验证部署

### 检查 Pod 状态

```bash
# 查看 Pod
kubectl get pods -n ips

# 查看 Pod 详情
kubectl describe pod -l app=ips -n ips

# 查看日志
kubectl logs -l app=ips -n ips -f
```

### 健康检查

```bash
# 在集群内测试
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://ips-apiserver.ips:8080/health

# 使用端口转发测试
kubectl port-forward -n ips svc/ips-apiserver 8080:8080

# 在另一个终端访问
curl http://localhost:8080/health
```

---

## 测试 API

### 创建公共镜像预热任务

```bash
# 创建预热任务（公共镜像仓库）
curl -X POST http://<EXTERNAL-IP>:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 10
  }'

# 查询任务列表
curl http://<EXTERNAL-IP>:8080/api/v1/tasks
```

### 创建私有镜像仓库预热任务

IPS 支持私有镜像仓库认证，通过创建临时的 Kubernetes Secret 实现。

**使用方式**:

```bash
# 创建预热任务（私有镜像仓库）
curl -X POST http://<EXTERNAL-IP>:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["harbor.example.com/myapp:v1.0.0"],
    "batchSize": 10,
    "registry": "harbor.example.com",
    "username": "your-username",
    "password": "your-password"
  }'
```

**参数说明**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `registry` | string | 是 | 镜像仓库地址（如 `harbor.example.com`） |
| `username` | string | 是 | 镜像仓库用户名 |
| `password` | string | 是 | 镜像仓库密码 |
| `images` | []string | 是 | 要预热的镜像列表（需包含 registry 前缀） |

**重要提示**:

1. **完整性要求**: 提供了 `registry` 时，必须同时提供 `username` 和 `password`
2. **Secret 管理**: IPS 会为每个任务自动创建临时 Secret（格式：`image-pull-secret-{taskID}`）
3. **自动清理**: 任务完成后，Secret 会自动删除，无论成功或失败
4. **镜像格式**: 私有镜像必须包含完整仓库地址，如 `harbor.example.com/myapp:v1.0.0`

**示例场景**:

```bash
# Harbor 私有仓库
curl -X POST http://<EXTERNAL-IP>:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["harbor.company.com/project/app:2.1.0"],
    "batchSize": 20,
    "registry": "harbor.company.com",
    "username": "robot$deploy",
    "password": "your-token-here"
  }'

# AWS ECR（需要提前获取临时凭证）
curl -X POST http://<EXTERNAL-IP>:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["123456789.dkr.ecr.us-west-2.amazonaws.com/myapp:latest"],
    "batchSize": 15,
    "registry": "123456789.dkr.ecr.us-west-2.amazonaws.com",
    "username": "AWS",
    "password": "AWS_TOKEN_HERE"
  }'

# Docker Hub 私有仓库
curl -X POST http://<EXTERNAL-IP>:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["docker.io/mycompany/production:latest"],
    "batchSize": 10,
    "registry": "https://index.docker.io/v1/",
    "username": "myusername",
    "password": "mypassword"
  }'
```

**错误处理**:

如果认证信息不完整，API 会返回 400 错误：

```json
{
  "error": "Private registry credentials incomplete",
  "details": "When registry is provided, both username and password are required"
}
```

---

## 故障排查

### 查看事件

```bash
kubectl get events -n ips --sort-by='.lastTimestamp'
```

### 检查 RBAC 权限

```bash
# 验证 ServiceAccount
kubectl get sa -n ips

# 验证 ClusterRole
kubectl get clusterrole ips-apiserver

# 验证 ClusterRoleBinding
kubectl get clusterrolebinding ips-apiserver
```

### Pod 无法启动

1. 检查镜像是否存在：

```bash
kubectl describe pod -l app=ips -n ips | grep -A 5 Events
```

2. 检查节点资源：

```bash
kubectl top nodes
```

### 无法创建 Job

1. 验证 RBAC 权限是否正确配置
2. 检查目标命名空间是否存在
3. 查看 API Server 日志

```bash
kubectl logs -l app=ips -n ips
```

---

## 卸载

```bash
# 删除所有资源
kubectl delete -f deploy/

# 或使用 kustomize
kubectl delete -k deploy/

# 删除命名空间（会删除命名空间内所有资源）
kubectl delete namespace ips
```

---

## 参考资料

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
