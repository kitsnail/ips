# Image Prewarm Service (IPS)

<div align="center">

![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?style=flat&logo=go)
![Kubernetes](https://img.shields.io/badge/kubernetes-v1.20+-326CE5?style=flat&logo=kubernetes)
![License](https://img.shields.io/badge/license-MIT-green)
[![Go Report Card](https://goreportcard.com/badge/github.com/kitsnail/ips)](https://goreportcard.com/report/github.com/kitsnail/ips)

**IPS (Image Prewarm Service) 是一个专为 Kubernetes 集群设计的高性能容器镜像预热服务。**
它通过 RESTful API 和可视化的 Web 界面，帮助用户在集群节点上批量、快速地预拉取镜像，从而显著减少应用启动时的镜像拉取延迟。

[功能特性](#-功能特性) • [快速开始](#-快速开始) • [文档](#-文档) • [架构](#-架构设计) • [贡献](#-参与贡献)

</div>

---

## 📖 简介

在 Kubernetes 集群中，Pod 的启动时间很大程度上取决于镜像拉取的速度。对于大镜像或网络环境不佳的场景，即时拉取会导致显著的启动延迟。IPS 旨在解决这一问题，它提供了一个中心化的控制台，允许管理员按需或定时将镜像分发到指定节点，确保业务容器能够“秒级”启动。

## ✨ 功能特性

### � 核心能力
- **批量预热**：支持一次性对成百上千个节点进行镜像预拉取。
- **智能调度**：
  - **并发控制**：内置信号量机制，防止大规模并发拉取耗尽节点网络带宽。
  - **批次处理**：支持自定义批次大小，平滑执行预热任务。
- **灵活筛选**：基于 Label Selector 的节点筛选机制，支持精细化的节点分组预热。

### 🖥️ 可视化管理 (Web UI)
- **实时看板**：直观展示任务进度、成功/失败节点数及详细状态。
- **任务管理**：支持任务的创建、查询、**批量删除**和**一键取消**。
- **用户友好**：
  - **交互优化**：采用现代化的 Toast 通知和确认模态框，拒绝原生弹窗。
  - **状态过滤**：支持按 Pending, Running, Completed 等状态快速筛选任务。
  - **移动端适配**：响应式布局，随时随地管理任务。

### 🛡️ 企业级特性
- **高可用架构**：支持多副本部署，配合 HPA 自动扩缩容。
- **安全性**：
  - JWT 身份认证。
  - 基于角色的访问控制 (RBAC)。
- **可观测性**：
  - 丰富的 Prometheus 指标（任务耗时、成功率、队列深度等）。
  - Webhook 通知集成（支持钉钉、Slack 等）。
- **多租户支持**：完善的用户管理和权限隔离。

## 🛠️ 快速开始

### 前提条件
- Kubernetes 1.20+ 集群
- `kubectl` 已配置并连接到集群
- Docker (用于构建镜像)

### 方式一：Kubernetes 部署 (推荐)

使用 Kustomize 一键部署到集群：

```bash
# 1. 部署所有组件 (API Server, Service, HPA, RBAC 等)
kubectl apply -k deploy/

# 2. 验证部署状态
kubectl get pods -n ips-system
```

### 方式二：本地开发运行

```bash
# 1. 编译二进制文件
make build


服务启动后，访问：
- **Web UI**: [http://localhost:8080/](http://localhost:8080/) 
# 默认用户名 admin，密码 admin123
- **API Health**: [http://localhost:8080/health](http://localhost:8080/health)

### 方式三：Docker 运行
docker run -d \
  --name ips-apiserver \
  -p 8080:8080 \
  -v ~/.kube/config:/home/ips/.kube/config:ro \
  ips-apiserver:latest
```

## 📦 项目结构

遵循标准的 Go 项目布局：

```
ips/
├── cmd/                # 主程序入口
├── internal/           # 私有应用代码
│   ├── api/            # API 路由与处理器
│   ├── service/        # 核心业务逻辑
│   ├── repository/     # 数据持久层
│   └── k8s/            # Kubernetes Client 封装
├── pkg/                # 公共库代码
├── deploy/             # Kubernetes 部署清单
├── web/                # 前端静态资源
└── scripts/            # 辅助脚本
```

## 🧩 架构设计

IPS 采用了典型的分层架构，确保了系统的高内聚低耦合：

- **接入层**：基于 Gin 框架的 RESTful API，提供统一的入口。
- **业务层**：TaskManager 负责任务的生命周期管理，BatchScheduler 负责任务的分批调度。
- **执行层**：通过 Kubernetes Job 或直接调用 CRI 接口（规划中）在节点上执行拉取操作。
- **存储层**：支持 SQLite（默认）及 MySQL 等多种存储后端。

详细设计文档请参阅：[docs/ARCHITECTURE.md](plan-arch.md)

## 📚 文档

- [API 接口文档](RESTful-API.md)
- [部署操作指南](deploy/DEPLOYMENT.md)
- [开发演进计划](development-plan.md)

## 🤝 参与贡献

欢迎提交 Pull Request 或 Issue！在提交代码前，请确保通过了所有测试和代码检查：

```bash
make lint
make test
```

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE) 发布。
