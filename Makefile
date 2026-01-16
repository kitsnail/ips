.PHONY: build run test clean

# 构建二进制文件
build:
	@echo "Building API server..."
	@go build -o bin/apiserver ./cmd/apiserver

# 运行服务
run: build
	@echo "Starting API server..."
	@./bin/apiserver

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf bin/

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 检查代码
lint:
	@echo "Linting code..."
	@go vet ./...

# 查看依赖
deps:
	@echo "Listing dependencies..."
	@go list -m all

# 更新依赖
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
