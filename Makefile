.PHONY: help build run test clean fmt lint deps tidy
.PHONY: docker-build docker-run docker-stop docker-clean

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME=apiserver
BINARY_DIR=bin
DOCKER_IMAGE=ips-apiserver
K8S_NAMESPACE=ips-system
SERVER_PORT=8080

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

DOCKER_TAG ?= $(VERSION)

# Linker flags
LDFLAGS := -w -s \
	-X github.com/kitsnail/ips/pkg/version.Version=$(VERSION) \
	-X github.com/kitsnail/ips/pkg/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/kitsnail/ips/pkg/version.BuildTime=$(BUILD_TIME)

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

build: ## 构建二进制文件 (Linux amd64)
	@echo "$(GREEN)Building API server $(VERSION) for Linux amd64...$(RESET)"
	@mkdir -p $(BINARY_DIR)
	@GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -ldflags="$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/apiserver

build_darwin: ## 构建二进制文件 (Darwin arm64)
	@echo "$(GREEN)Building API server $(VERSION) for Darwin arm64...$(RESET)"
	@mkdir -p $(BINARY_DIR)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY_NAME)_darwin ./cmd/apiserver

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
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(RESET)"
	@docker build --platform linux/amd64 -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-build-dev: build ## 构建 Docker 镜像 (开发模式，使用宿主机二进制)
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG) (Dev mode)...$(RESET)"
	@docker build --platform linux/amd64 -f Dockerfile-dev -t $(DOCKER_IMAGE):dev .

docker-build-no-cache: ## 构建 Docker 镜像（不使用缓存）
	@echo "$(GREEN)Building Docker image (no cache)...$(RESET)"
	@docker build --no-cache -t $(DOCKER_IMAGE):$(DOCKER_TAG) .


##@ 其他

version: ## 显示版本信息
	@echo "App Version: $(VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go version:  $(shell go version)"
	@echo "Docker:      $(shell docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Target:      Linux amd64"

api-test: ## 运行 API 测试脚本
	@echo "$(GREEN)Running API tests...$(RESET)"
	@./test-api.sh

.PHONY: all
all: clean fmt lint test build ## 执行完整的构建流程
