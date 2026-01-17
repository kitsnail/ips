#!/bin/bash

# IPS 快速更新部署脚本
# 用于在镜像仓库不可用时直接更新 Kubernetes 部署

set -e

echo "==================================="
echo "  IPS 快速更新部署"
echo "==================================="

# 1. 构建 Docker 镜像
echo "📦 步骤 1/4: 构建 Docker 镜像..."
docker build -t ips:latest .
echo "✅ 镜像构建完成"

# 2. 标记镜像
echo "🏷️  步骤 2/4: 标记镜像..."
docker tag ips:latest 192.168.3.81/library/ips-apiserver:latest
echo "✅ 镜像标记完成"

# 3. 推送镜像（如果镜像仓库可用）
echo "⬆️  步骤 3/4: 推送镜像..."
if docker push 192.168.3.81/library/ips-apiserver:latest 2>/dev/null; then
    echo "✅ 镜像推送成功"
else
    echo "⚠️  镜像推送失败，尝试直接加载到节点..."

    # 获取所有节点
    NODES=$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}')

    # 保存镜像为 tar 文件
    echo "💾 导出镜像..."
    docker save 192.168.3.81/library/ips-apiserver:latest -o /tmp/ips-apiserver.tar

    # 将镜像加载到每个节点
    for NODE in $NODES; do
        echo "📥 加载镜像到节点: $NODE"
        # 这里需要根据实际环境调整加载方式
        # 例如使用 scp + docker load，或者其他节点访问方式
        echo "   请手动将 /tmp/ips-apiserver.tar 加载到节点 $NODE"
    done

    echo ""
    echo "⚠️  手动操作提示："
    echo "   1. 镜像已导出到: /tmp/ips-apiserver.tar"
    echo "   2. 需要将此文件复制到所有节点"
    echo "   3. 在每个节点上运行: docker load -i ips-apiserver.tar"
    echo ""
    read -p "按 Enter 继续部署（确保镜像已加载到所有节点）..."
fi

# 4. 重启部署
echo "🔄 步骤 4/4: 重启 Kubernetes 部署..."
kubectl rollout restart deployment/ips-apiserver -n default
kubectl rollout status deployment/ips-apiserver -n default --timeout=120s

echo ""
echo "==================================="
echo "  ✅ 部署更新完成！"
echo "==================================="
echo ""
echo "🌐 服务访问信息:"
echo "   LoadBalancer: $(kubectl get svc ips-apiserver -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):8080"
echo "   Web UI: http://$(kubectl get svc ips-apiserver -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):8080/web/"
echo ""
echo "📊 查看状态:"
echo "   kubectl get pods -l app=ips-apiserver"
echo "   kubectl logs -f deployment/ips-apiserver"
echo ""
