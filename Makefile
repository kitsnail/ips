.PHONY: help build run test clean fmt lint deps tidy
.PHONY: docker-build docker-run docker-stop docker-clean docker-compose-up docker-compose-down
.PHONY: k8s-deploy k8s-delete k8s-status k8s-logs k8s-port-forward k8s-restart

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME=apiserver
BINARY_DIR=bin
DOCKER_IMAGE=ips-apiserver
DOCKER_TAG=latest
K8S_NAMESPACE=ips-system
SERVER_PORT=8080

# 颜色输出
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

##@ 帮助

help: ## 显示帮助信息
	@echo '用法:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  ${GREEN}%-18s${RESET} %s\n", $$1, $$2 } /^##@/ { printf "\n${YELLOW}%s${RESET}\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ 开发

build: ## 构建二进制文件
	@echo "$(GREEN)Building API server...$(RESET)"
	@mkdir -p $(BINARY_DIR)
	@go build -ldflags="-w -s" -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/apiserver

run: build ## 构建并运行服务
	@echo "$(GREEN)Starting API server...$(RESET)"
	@./$(BINARY_DIR)/$(BINARY_NAME)

test: ## 运行测试
	@echo "$(GREEN)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## 运行测试并生成覆盖率报告
	@echo "$(GREEN)Generating coverage report...$(RESET)"
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

clean: ## 清理构建产物
	@echo "$(GREEN)Cleaning...$(RESET)"
	@rm -rf $(BINARY_DIR)/
	@rm -f coverage.txt coverage.html

fmt: ## 格式化代码
	@echo "$(GREEN)Formatting code...$(RESET)"
	@go fmt ./...

lint: ## 检查代码
	@echo "$(GREEN)Linting code...$(RESET)"
	@go vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, skipping...$(RESET)"; \
	fi

deps: ## 查看依赖
	@echo "$(GREEN)Listing dependencies...$(RESET)"
	@go list -m all

tidy: ## 整理依赖
	@echo "$(GREEN)Tidying dependencies...$(RESET)"
	@go mod tidy

##@ Docker

docker-build: ## 构建 Docker 镜像
	@echo "$(GREEN)Building Docker image...$(RESET)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-build-no-cache: ## 构建 Docker 镜像（不使用缓存）
	@echo "$(GREEN)Building Docker image (no cache)...$(RESET)"
	@docker build --no-cache -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## 运行 Docker 容器
	@echo "$(GREEN)Running Docker container...$(RESET)"
	@docker run -d \
		--name $(DOCKER_IMAGE) \
		-p $(SERVER_PORT):$(SERVER_PORT) \
		-v ~/.kube/config:/home/ips/.kube/config:ro \
		-e K8S_NAMESPACE=default \
		-e WORKER_IMAGE=busybox:latest \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Container started: http://localhost:$(SERVER_PORT)$(RESET)"

docker-stop: ## 停止 Docker 容器
	@echo "$(GREEN)Stopping Docker container...$(RESET)"
	@docker stop $(DOCKER_IMAGE) || true
	@docker rm $(DOCKER_IMAGE) || true

docker-logs: ## 查看 Docker 容器日志
	@docker logs -f $(DOCKER_IMAGE)

docker-clean: docker-stop ## 清理 Docker 资源
	@echo "$(GREEN)Cleaning Docker resources...$(RESET)"
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

docker-compose-up: ## 使用 Docker Compose 启动服务
	@echo "$(GREEN)Starting services with Docker Compose...$(RESET)"
	@docker-compose up -d
	@echo "$(GREEN)Services started: http://localhost:$(SERVER_PORT)$(RESET)"

docker-compose-down: ## 使用 Docker Compose 停止服务
	@echo "$(GREEN)Stopping services with Docker Compose...$(RESET)"
	@docker-compose down

docker-compose-logs: ## 查看 Docker Compose 日志
	@docker-compose logs -f

##@ Kubernetes

k8s-deploy: ## 部署到 Kubernetes
	@echo "$(GREEN)Deploying to Kubernetes...$(RESET)"
	@kubectl apply -f deploy/
	@echo "$(GREEN)Waiting for deployment to be ready...$(RESET)"
	@kubectl wait --for=condition=available --timeout=300s deployment/ips-apiserver -n $(K8S_NAMESPACE) || true
	@echo "$(GREEN)Deployment completed!$(RESET)"

k8s-deploy-kustomize: ## 使用 Kustomize 部署到 Kubernetes
	@echo "$(GREEN)Deploying to Kubernetes with Kustomize...$(RESET)"
	@kubectl apply -k deploy/
	@echo "$(GREEN)Deployment completed!$(RESET)"

k8s-delete: ## 从 Kubernetes 删除部署
	@echo "$(YELLOW)Deleting from Kubernetes...$(RESET)"
	@kubectl delete -f deploy/ || true

k8s-status: ## 查看 Kubernetes 部署状态
	@echo "$(GREEN)Checking deployment status...$(RESET)"
	@kubectl get all -n $(K8S_NAMESPACE)

k8s-pods: ## 查看 Pods
	@kubectl get pods -n $(K8S_NAMESPACE) -l app=ips

k8s-logs: ## 查看日志
	@kubectl logs -n $(K8S_NAMESPACE) -l app=ips -f --tail=100

k8s-describe: ## 查看 Pod 详情
	@kubectl describe pods -n $(K8S_NAMESPACE) -l app=ips

k8s-events: ## 查看事件
	@kubectl get events -n $(K8S_NAMESPACE) --sort-by='.lastTimestamp'

k8s-port-forward: ## 端口转发（本地访问）
	@echo "$(GREEN)Port forwarding to localhost:$(SERVER_PORT)...$(RESET)"
	@kubectl port-forward -n $(K8S_NAMESPACE) svc/ips-apiserver $(SERVER_PORT):8080

k8s-restart: ## 重启部署
	@echo "$(GREEN)Restarting deployment...$(RESET)"
	@kubectl rollout restart deployment/ips-apiserver -n $(K8S_NAMESPACE)

k8s-scale: ## 扩缩容 (usage: make k8s-scale REPLICAS=3)
	@echo "$(GREEN)Scaling deployment to $(REPLICAS) replicas...$(RESET)"
	@kubectl scale deployment/ips-apiserver -n $(K8S_NAMESPACE) --replicas=$(REPLICAS)

##@ 其他

version: ## 显示版本信息
	@echo "Go version: $(shell go version)"
	@echo "Docker version: $(shell docker --version 2>/dev/null || echo 'Not installed')"
	@echo "kubectl version: $(shell kubectl version --client --short 2>/dev/null || echo 'Not installed')"

api-test: ## 运行 API 测试脚本
	@echo "$(GREEN)Running API tests...$(RESET)"
	@./test-api.sh

.PHONY: all
all: clean fmt lint test build ## 执行完整的构建流程
