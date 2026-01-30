# Draft: Implement Kubernetes ImagePullSecrets Authentication for IPS

## Requirements (confirmed)
- Support pulling images from private registries (e.g., Harbor)
- User provides registry credentials when creating a task
- Credentials stored in Kubernetes Secrets
- Jobs reference secrets via ImagePullSecrets
- Support multiple registries with different credentials
- Clean up secrets after task completion
- Backward compatibility: public image tasks still work without auth

## Current State Analysis

### File: pkg/models/task.go
- Current fields: ID, Status, Priority, Images, BatchSize, NodeSelector, Progress, FailedNodes, MaxRetries, RetryCount, RetryStrategy, RetryDelay, WebhookURL, timestamps
- **NO fields for tracking secrets or registry credentials**
- Clean structure with JSON tags

### File: pkg/models/request.go
- CreateTaskRequest has: Images, BatchSize, Priority, NodeSelector, MaxRetries, RetryStrategy, RetryDelay, WebhookURL
- **NO fields for registry authentication**
- Uses binding tags for validation

### File: internal/k8s/job_creator.go
- Creates Jobs with:
  - Puller image using crictl command
  - Container with privileged security context
  - Volume mounts for CRI socket
  - NodeSelector and Tolerations
  - TTL for auto-cleanup (900 seconds)
- **NO ImagePullSecrets in PodSpec**
- **NO secret creation logic**
- Has DeleteJob for cleanup

### File: internal/service/task_manager.go
- Task lifecycle: CreateTask → executeTask → markTaskFailed/DeleteTask
- Concurrency control with semaphore
- Retry mechanism with linear/exponential strategies
- Webhook notifications
- Context cancellation support
- **NO secret lifecycle management**
- Cleanup happens on task completion/failure/cancellation

### File: internal/api/handler/task.go
- Simple handler: bind JSON → taskManager.CreateTask
- **NO processing of auth fields**
- Returns created task on success

### Testing Infrastructure
- Found test files in:
  - internal/api/handler/task_test.go
  - internal/repository/memory_test.go
  - internal/service/priority_queue_test.go
  - internal/service/node_filter_test.go
  - internal/service/batch_scheduler_test.go
- Need to check test framework and patterns

## Research Findings

**Status**: Research agents failed. Will need to research best practices manually.

## Open Questions
1. Test framework being used (need to check test files)?
2. Should credentials be stored in Task model (security concern)?
3. Secret naming convention to avoid conflicts?
4. Should secrets be per-task or per-registry?
5. When exactly to clean up secrets (immediate after Job completion, or after task completion)?
6. Web UI fields for authentication (username/password/server)?

## Technical Decisions (pending)
- **Secret storage**: Kubernetes Secrets with type "kubernetes.io/dockerconfigjson"
- **Secret naming**: `ips-{taskId}-registry-{hash}` or similar to avoid conflicts
- **Secret lifecycle**: Create when task starts, cleanup when task completes/fails/cancels
- **ImagePullSecrets**: Add to PodSpec in JobCreator.CreateJob
- **Credential structure**: Array of RegistryCredential objects in request
