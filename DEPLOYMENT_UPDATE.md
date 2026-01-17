# 快速更新部署指南

## 方法一：使用镜像仓库（推荐）

如果你的镜像仓库 `192.168.3.81` 可用：

```bash
# 1. 构建并标记镜像
docker build -t ips:latest .
docker tag ips:latest 192.168.3.81/library/ips-apiserver:latest

# 2. 推送镜像
docker push 192.168.3.81/library/ips-apiserver:latest

# 3. 重启部署
kubectl rollout restart deployment/ips-apiserver -n default
kubectl rollout status deployment/ips-apiserver -n default
```

## 方法二：导出镜像手动加载

如果镜像仓库不可用，使用此方法：

```bash
# 1. 构建并标记镜像
docker build -t ips:latest .
docker tag ips:latest 192.168.3.81/library/ips-apiserver:latest

# 2. 导出镜像
docker save 192.168.3.81/library/ips-apiserver:latest -o ips-apiserver.tar

# 3. 将镜像文件复制到所有 Kubernetes 节点
# 方式 A: 使用 scp
for node in node1 node2 node3; do
    scp ips-apiserver.tar $node:/tmp/
    ssh $node "docker load -i /tmp/ips-apiserver.tar"
done

# 方式 B: 手动复制
# 将 ips-apiserver.tar 复制到所有节点，然后在每个节点上运行：
# docker load -i /tmp/ips-apiserver.tar

# 4. 重启部署
kubectl rollout restart deployment/ips-apiserver -n default
kubectl rollout status deployment/ips-apiserver -n default
```

## 方法三：使用 kind/minikube 本地部署

如果你使用 kind 或 minikube 本地测试：

```bash
# kind
docker build -t ips:latest .
kind load docker-image ips:latest

# minikube
docker build -t ips:latest .
minikube image load ips:latest
```

## 验证部署

```bash
# 检查 Pod 状态
kubectl get pods -l app=ips-apiserver

# 查看日志
kubectl logs -f deployment/ips-apiserver

# 获取服务地址
kubectl get svc ips-apiserver

# 测试 Web UI
EXTERNAL_IP=$(kubectl get svc ips-apiserver -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl http://$EXTERNAL_IP:8080/health
echo "Web UI: http://$EXTERNAL_IP:8080/web/"
```

## 快速修复当前部署

对于你当前的部署 (IP: 192.168.3.106)：

```bash
# 1. 构建新镜像
docker build -t ips:latest .
docker tag ips:latest 192.168.3.81/library/ips-apiserver:latest

# 2. 导出镜像
docker save 192.168.3.81/library/ips-apiserver:latest -o ips-apiserver.tar

# 3. 找出 Pod 所在的节点
kubectl get pods -l app=ips-apiserver -o wide

# 4. 将镜像加载到这些节点（需要节点访问权限）
# 然后重启 deployment
kubectl rollout restart deployment/ips-apiserver -n default
```

## 注意事项

1. 确保 `web/static/` 目录包含以下文件：
   - index.html
   - app.js

2. Dockerfile 已更新，包含 Web UI 文件复制：
   ```dockerfile
   COPY --from=builder /build/web /app/web
   ```

3. 如果 Pod 一直使用旧镜像，检查 ImagePullPolicy：
   ```bash
   kubectl set image deployment/ips-apiserver ips-apiserver=192.168.3.81/library/ips-apiserver:latest-$(date +%s)
   ```
