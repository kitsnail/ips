# Project Context

## Purpose

IPS (Image Prewarm Service) is a high-performance container image prewarming service designed for Kubernetes clusters. It provides a centralized RESTful API and visual web interface for administrators to batch and quickly pre-pull images across cluster nodes, significantly reducing application startup latency caused by image pull delays.

**Key Goals:**
- Reduce Pod startup latency by pre-warming container images on cluster nodes
- Provide intuitive web UI for task management and monitoring
- Support intelligent scheduling with concurrency control and batch processing
- Enable fine-grained node selection via Kubernetes Label Selectors
- Deliver enterprise-grade features: high availability, security (JWT + RBAC), observability (Prometheus metrics), and multi-tenant support

## Tech Stack

### Backend
- **Language**: Go 1.23
- **Web Framework**: Gin (v1.11.0) for RESTful API
- **Kubernetes Integration**: k8s.io/client-go (v0.29.0) for cluster interaction
- **Authentication**: golang-jwt/jwt/v5 (v5.3.0)
- **Database**: modernc.org/sqlite (v1.30.1) - embedded SQLite (MySQL support available)
- **Scheduling**: robfig/cron/v3 (v3.0.1) for scheduled tasks
- **Logging**: logrus (v1.9.4)
- **Metrics**: Prometheus client_golang (v1.19.1)
- **Concurrency**: golang.org/x/sync (v0.16.0) for semaphore-based concurrency control

### Frontend
- **Framework**: Vue 3 (v3.5.24) with Composition API and `<script setup>`
- **Language**: TypeScript (~5.9.3)
- **Build Tool**: Vite (v7.2.4)
- **UI Components**: Element Plus (v2.13.2) with icons
- **State Management**: Pinia (v3.0.4)
- **Routing**: Vue Router (v5.0.1)
- **HTTP Client**: Axios (v1.13.4)

### DevOps & Deployment
- **Container Runtime**: Docker
- **Orchestration**: Kubernetes 1.20+
- **Deployment Tool**: Kustomize
- **Platform Targets**: Linux amd64 (production), Darwin arm64 (local dev)

## Project Conventions

### Code Style

#### Go Backend
- **Standard Project Layout**: Follows standard Go directory structure
  - `cmd/` - Application entry points
  - `internal/` - Private application code (api, service, repository, k8s, puller)
  - `pkg/` - Public library code (models, metrics, version)
  - `deploy/` - Kubernetes manifests
  - `web/` - Frontend static assets
- **Naming Conventions**:
  - Interfaces: Simple, descriptive names (e.g., `Repository`, `TaskManager`)
  - Implementations: Add concrete type prefix (e.g., `SQLiteRepository`, `MemoryRepository`)
  - Package names: Lowercase, single word, no underscores (e.g., `service`, `repository`)
- **Formatting**: Use `gofmt` (enforced via `make fmt`)
- **Linting**: Use `golangci-lint` and `go vet` (enforced via `make lint`)
- **Testing**:
  - All test files end with `_test.go`
  - Use `go test -v -race` for race detection
  - Coverage targets via `go test -coverprofile=coverage.txt -covermode=atomic`
- **Error Handling**: Use explicit error returns, never suppress errors

#### Frontend (TypeScript + Vue 3)
- **Vue Style**: Use Composition API with `<script setup>` syntax
- **Type Safety**: Strict TypeScript configuration, no `any` types
- **Component Organization**: Single File Components (`.vue`)
- **State Management**: Pinia stores for shared state
- **Naming**:
  - Components: PascalCase (e.g., `TaskList.vue`, `TaskDetail.vue`)
  - Functions/Variables: camelCase
  - Constants: UPPER_SNAKE_CASE
- **Styling**: Use Element Plus components with custom CSS as needed

### Architecture Patterns

**Four-Layer Architecture**:
1. **Access Layer** (`internal/api/`): Gin-based RESTful API handlers, middleware (auth, logging, recovery, prometheus)
2. **Business Layer** (`internal/service/`): Core business logic (TaskManager, BatchScheduler, ScheduledTaskManager)
3. **Execution Layer** (`internal/k8s/`, `internal/puller/`): Kubernetes Job creation, node filtering, image pull execution
4. **Storage Layer** (`internal/repository/`): Data persistence (SQLiteRepository, MemoryRepository)

**Key Patterns**:
- **Repository Pattern**: Abstraction for data access (Repository interface with multiple implementations)
- **Service Pattern**: Business logic encapsulation (TaskManager, AuthService, WebhookService)
- **Middleware Pattern**: Gin middleware for cross-cutting concerns (auth, logging, recovery)
- **Factory Pattern**: JobCreator for Kubernetes resource creation
- **Strategy Pattern**: RetryStrategy for configurable retry logic

**Concurrency Model**:
- Semaphore-based concurrency control in BatchScheduler
- Priority queue for task scheduling
- Goroutine-based async execution with proper synchronization using `golang.org/x/sync`

### Testing Strategy

#### Backend Testing
- **Unit Tests**: Cover all service and repository methods
- **Integration Tests**: Test API handlers with mock repositories
- **Race Detection**: Always run tests with `-race` flag
- **Coverage**: Generate coverage reports (`make test-coverage`)
- **Test Organization**: Place test files alongside implementation (e.g., `task_manager.go` + `task_manager_test.go`)

#### Frontend Testing
- **Component Testing**: Test Vue components in isolation
- **E2E Testing**: (Planned) Use Playwright for end-to-end testing
- **Type Checking**: Use `vue-tsc` for compile-time type validation

### Git Workflow

- **Version Management**:
  - Use `git describe --tags --always --dirty` for versioning
  - Embed version info at build time via linker flags
  - Format: `[tag]-[commits]-g[commit]-dirty` (if uncommitted changes)
- **Branching**: (Recommended)
  - `main` - Production-ready code
  - `develop` - Integration branch for features
  - `feature/*` - Feature branches
  - `bugfix/*` - Bug fix branches
- **Commit Messages**: (Recommended) Use Conventional Commits format:
  - `feat: add scheduled task management`
  - `fix: resolve memory leak in task scheduler`
  - `refactor: simplify node filter logic`
- **CI/CD**:
  - Run tests on every commit: `make lint test`
  - Build Docker image on merge to main
  - Deploy to Kubernetes via Kustomize

## Domain Context

**Kubernetes Image Pulling Problem:**
In Kubernetes clusters, Pod startup time is heavily dependent on image pull speed. Large images or poor network conditions cause significant startup delays. IPS solves this by allowing administrators to pre-distribute images to specified nodes on-demand or on schedule.

**Core Concepts:**
- **Task**: A request to pull a set of container images on selected nodes
- **Node Selector**: Kubernetes label-based filtering (e.g., `node-role.kubernetes.io/worker`)
- **Batch Scheduler**: Controls concurrent pulls per task to prevent network overload
- **Scheduled Task**: Recurring task execution via cron expressions
- **Priority Queue**: Task ordering based on priority (e.g., urgent vs background)
- **Webhook Notification**: Integration with external systems (DingTalk, Slack) for task status updates

**User Roles**:
- **Admin**: Full access to all features (user management, task creation, configuration)
- **Operator**: Can create and manage tasks, but not users or system settings
- **Viewer**: Read-only access to tasks and status

## Important Constraints

### Technical Constraints
- **Kubernetes Version**: Requires Kubernetes 1.20+ for client-go compatibility
- **Storage**: Default SQLite database (embedded, no external dependencies), MySQL support available
- **Image Pull Methods**: Currently uses Kubernetes Job API; direct CRI interface planned
- **Concurrency**: Semaphore-based limit to prevent network bandwidth exhaustion
- **Node Access**: Requires service account with sufficient RBAC permissions to create Jobs on target nodes

### Performance Constraints
- **Batch Size**: Configurable (default: 10 nodes per batch)
- **Concurrent Pulls**: Configurable semaphore limit (default: 5 per batch)
- **Task Timeout**: Default 30 minutes per task (configurable)
- **Retry Strategy**: Exponential backoff with configurable max attempts (default: 3)

### Security Constraints
- **Authentication**: JWT tokens with configurable expiration (default: 24 hours)
- **RBAC**: Role-based access control with Admin/Operator/Viewer roles
- **Secret Management**: Passwords and API keys hashed using bcrypt
- **Kubeconfig**: Read-only access to kubeconfig for cluster communication

### Operational Constraints
- **Deployment**: Supports multi-replica deployment with HPA
- **Persistence**: SQLite file persists on pod recreation (requires PVC)
- **Observability**: Prometheus metrics exposed on `/metrics` endpoint
- **Logging**: Structured JSON logs via logrus for easier parsing

## External Dependencies

### Kubernetes Cluster
- **Purpose**: Target cluster for image prewarming operations
- **Required**: Kubeconfig with service account permissions
- **Permissions Needed**:
  - `create`, `list`, `watch`, `delete` Jobs
  - `list`, `watch` Nodes
  - `get` Node labels

### Webhook Services (Optional)
- **DingTalk**: For task status notifications
- **Slack**: For task status notifications
- **Custom**: Generic webhook support for any HTTP endpoint

### Database (Optional)
- **SQLite**: Embedded, no external setup required (default)
- **MySQL**: External database support (configure via environment variables)
  - Host, Port, Database, User, Password required

### Container Registry
- **Supported**: Any Docker-compatible registry (Docker Hub, Harbor, GCR, ECR, etc.)
- **Authentication**: Image pull secrets must be pre-configured in Kubernetes namespace
