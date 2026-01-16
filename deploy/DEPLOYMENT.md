# 部署指南

本文档提供 IPS (Image Prewarm Service) 的 Docker 和 Kubernetes 部署指南。

## 目录

- [Docker 部署](#docker-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [配置说明](#配置说明)
- [验证部署](#验证部署)

---

## Docker 部署

### 前置要求

- Docker 20.10+
- Docker Compose 1.29+ (可选)
- 访问 Kubernetes 集群的 kubeconfig

### 构建镜像

```bash
# 在项目根目录执行
docker build -t ips-apiserver:latest .
```

### 使用 Docker 运行

```bash
docker run -d \
  --name ips-apiserver \
  -p 8080:8080 \
  -v ~/.kube/config:/home/ips/.kube/config:ro \
  -e K8S_NAMESPACE=default \
  -e WORKER_IMAGE=busybox:latest \
  ips-apiserver:latest
```

### 使用 Docker Compose 运行

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

---

## Kubernetes 部署

### 前置要求

- Kubernetes 1.24+
- kubectl 命令行工具
- 集群管理员权限（用于创建 RBAC 资源）

### 快速部署

#### 方法 1: 使用 kubectl 直接部署

```bash
# 1. 应用所有配置
kubectl apply -f deploy/

# 2. 等待 Pod 就绪
kubectl wait --for=condition=ready pod -l app=ips -n ips-system --timeout=300s

# 3. 查看部署状态
kubectl get all -n ips-system
```

#### 方法 2: 使用 Kustomize 部署

```bash
# 1. 预览部署资源
kubectl kustomize deploy/

# 2. 应用配置
kubectl apply -k deploy/

# 3. 查看部署状态
kubectl get all -n ips-system
```

### 构建和推送镜像（生产环境）

```bash
# 1. 构建镜像
docker build -t your-registry/ips-apiserver:v1.0.0 .

# 2. 推送到镜像仓库
docker push your-registry/ips-apiserver:v1.0.0

# 3. 更新 Deployment 配置
# 编辑 deploy/deployment.yaml，修改镜像地址
image: your-registry/ips-apiserver:v1.0.0

# 或者使用 kustomize 设置镜像
cd deploy/
kustomize edit set image ips-apiserver=your-registry/ips-apiserver:v1.0.0
kubectl apply -k .
```

### 分步部署

如果需要逐个资源部署，可以按以下顺序：

```bash
# 1. 创建命名空间
kubectl apply -f deploy/namespace.yaml

# 2. 创建 RBAC 资源
kubectl apply -f deploy/rbac.yaml

# 3. 创建 ConfigMap
kubectl apply -f deploy/configmap.yaml

# 4. 创建资源配额（可选）
kubectl apply -f deploy/resource-quota.yaml

# 5. 创建 Deployment
kubectl apply -f deploy/deployment.yaml

# 6. 创建 Service
kubectl apply -f deploy/service.yaml

# 7. 创建 Ingress（可选）
kubectl apply -f deploy/ingress.yaml

# 8. 创建 HPA（可选）
kubectl apply -f deploy/hpa.yaml

# 9. 创建 PDB（可选）
kubectl apply -f deploy/pdb.yaml
```

---

## 配置说明

### 环境变量

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_PORT` | 服务监听端口 | `8080` |
| `K8S_NAMESPACE` | 创建预热 Job 的命名空间 | `default` |
| `WORKER_IMAGE` | 预热 Job 使用的镜像 | `busybox:latest` |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |

### Service 类型选择

项目提供两种 Service 配置方式：

#### 1. LoadBalancer (默认，使用 MetalLB)

**文件**: `deploy/service.yaml`

```yaml
spec:
  type: LoadBalancer
  ports:
    - port: 8080
      targetPort: http
```

**特点**：
- 自动分配外部 IP（通过 MetalLB）
- 直接从集群外部访问
- 不需要 Ingress Controller
- 适合内网环境或简单部署

**访问方式**：
```bash
# 获取外部 IP
kubectl get svc ips-apiserver -n ips-system

# 直接访问
curl http://<EXTERNAL-IP>:8080/health
```

**重要提示 - 多副本部署注意事项**：

当前版本使用**内存存储**，在多副本部署时需要配置会话亲和性（Session Affinity），确保同一客户端的请求总是路由到同一个 Pod。

Service 配置已包含会话亲和性：
```yaml
sessionAffinity: ClientIP
sessionAffinityConfig:
  clientIP:
    timeoutSeconds: 10800  # 3 小时
```

如果遇到任务查询不一致的问题（创建成功但查询失败），可以：
1. 检查 Service 的会话亲和性配置是否生效
2. 临时减少副本数到 1：`kubectl scale deployment/ips-apiserver -n ips-system --replicas=1`
3. 长期方案：实现 Redis 或数据库存储替代内存存储

**配置 IP 地址池（可选）**：

编辑 `deploy/service.yaml`，取消注释以下行：
```yaml
annotations:
  metallb.universe.tf/address-pool: default
  # metallb.universe.tf/loadBalancerIPs: 192.168.3.106  # 指定固定 IP
```

#### 2. ClusterIP (配合 Ingress 使用)

**文件**: `deploy/service-clusterip.yaml`

```yaml
spec:
  type: ClusterIP
  ports:
    - port: 8080
      targetPort: http
```

**特点**：
- 仅集群内部可访问
- 需要配合 Ingress Controller 使用
- 支持域名访问和 HTTPS
- 适合需要高级路由功能的场景

**使用方式**：
```bash
# 删除 LoadBalancer Service
kubectl delete -f deploy/service.yaml

# 应用 ClusterIP Service
kubectl apply -f deploy/service-clusterip.yaml

# 配置并应用 Ingress
kubectl apply -f deploy/ingress.yaml
```

**访问方式**：
```bash
# 通过 Ingress 域名访问
curl http://ips.example.com/health

# 或通过端口转发测试
kubectl port-forward svc/ips-apiserver -n ips-system 8080:8080
curl http://localhost:8080/health
```

### ConfigMap 配置

编辑 `deploy/configmap.yaml` 修改配置：

```yaml
data:
  SERVER_PORT: "8080"
  K8S_NAMESPACE: "default"
  WORKER_IMAGE: "busybox:latest"
  LOG_LEVEL: "info"
```

修改后重新应用：

```bash
kubectl apply -f deploy/configmap.yaml
kubectl rollout restart deployment/ips-apiserver -n ips-system
```

### Ingress 配置

如果使用 Ingress 暴露服务，需要：

1. 修改 `deploy/ingress.yaml` 中的域名：

```yaml
spec:
  rules:
    - host: ips.example.com  # 修改为实际域名
```

2. 如果使用 HTTPS，配置 TLS：

```yaml
spec:
  tls:
    - hosts:
        - ips.example.com
      secretName: ips-tls
```

3. 应用配置：

```bash
kubectl apply -f deploy/ingress.yaml
```

### 资源配置

默认资源配置：

- **Requests**: CPU 100m, Memory 128Mi
- **Limits**: CPU 500m, Memory 512Mi

根据实际负载调整 `deploy/deployment.yaml` 中的资源配置。

### HPA 配置

水平自动扩缩容配置：

- **最小副本数**: 2
- **最大副本数**: 10
- **CPU 目标**: 70%
- **内存目标**: 80%

根据需要调整 `deploy/hpa.yaml`。

---

## 验证部署

### 检查 Pod 状态

```bash
# 查看 Pod
kubectl get pods -n ips-system

# 查看 Pod 详情
kubectl describe pod -l app=ips -n ips-system

# 查看日志
kubectl logs -l app=ips -n ips-system -f
```

### 健康检查

```bash
# 在集群内测试
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://ips-apiserver.ips-system:8080/health

# 使用端口转发测试
kubectl port-forward -n ips-system svc/ips-apiserver 8080:8080

# 在另一个终端访问
curl http://localhost:8080/health
```

### 测试 API

```bash
# 创建预热任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "images": ["nginx:latest", "redis:7"],
    "nodeSelector": {
      "labels": {"env": "prod"}
    }
  }'

# 查询任务列表
curl http://localhost:8080/api/v1/tasks

# 查询特定任务
curl http://localhost:8080/api/v1/tasks/{task-id}
```

---

## 故障排查

### 查看事件

```bash
kubectl get events -n ips-system --sort-by='.lastTimestamp'
```

### 检查 RBAC 权限

```bash
# 验证 ServiceAccount
kubectl get sa -n ips-system

# 验证 ClusterRole
kubectl get clusterrole ips-apiserver

# 验证 ClusterRoleBinding
kubectl get clusterrolebinding ips-apiserver
```

### Pod 无法启动

1. 检查镜像是否存在：

```bash
kubectl describe pod -l app=ips -n ips-system | grep -A 5 Events
```

2. 检查资源配额：

```bash
kubectl describe resourcequota -n ips-system
```

3. 检查节点资源：

```bash
kubectl top nodes
```

### 无法创建 Job

1. 验证 RBAC 权限是否正确配置
2. 检查目标命名空间是否存在
3. 查看 API Server 日志

```bash
kubectl logs -l app=ips -n ips-system
```

---

## 升级和回滚

### 升级

```bash
# 1. 构建新版本镜像
docker build -t your-registry/ips-apiserver:v1.1.0 .
docker push your-registry/ips-apiserver:v1.1.0

# 2. 更新 Deployment
kubectl set image deployment/ips-apiserver \
  apiserver=your-registry/ips-apiserver:v1.1.0 \
  -n ips-system

# 3. 查看滚动更新状态
kubectl rollout status deployment/ips-apiserver -n ips-system
```

### 回滚

```bash
# 查看历史版本
kubectl rollout history deployment/ips-apiserver -n ips-system

# 回滚到上一个版本
kubectl rollout undo deployment/ips-apiserver -n ips-system

# 回滚到特定版本
kubectl rollout undo deployment/ips-apiserver --to-revision=2 -n ips-system
```

---

## 卸载

### 删除 Kubernetes 资源

```bash
# 删除所有资源
kubectl delete -f deploy/

# 或使用 kustomize
kubectl delete -k deploy/

# 删除命名空间（会删除命名空间内所有资源）
kubectl delete namespace ips-system
```

### 删除 Docker 容器

```bash
# Docker Compose
docker-compose down -v

# Docker
docker stop ips-apiserver
docker rm ips-apiserver
```

---

## 生产环境建议

1. **安全性**
   - 使用私有镜像仓库
   - 配置 imagePullSecrets
   - 启用 Pod Security Standards
   - 定期更新镜像和依赖

2. **高可用**
   - 部署多个副本 (≥2)
   - 配置 Pod Anti-Affinity
   - 使用 PodDisruptionBudget

3. **监控和日志**
   - 集成 Prometheus 进行监控
   - 配置日志收集（如 ELK、Loki）
   - 设置告警规则

4. **资源管理**
   - 根据实际负载调整资源配额
   - 配置 HPA 进行自动扩缩容
   - 使用 ResourceQuota 和 LimitRange

5. **备份和恢复**
   - 定期备份配置文件
   - 记录部署版本
   - 制定灾难恢复计划

---

## 参考资料

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Docker 官方文档](https://docs.docker.com/)
- [Kustomize 文档](https://kustomize.io/)
