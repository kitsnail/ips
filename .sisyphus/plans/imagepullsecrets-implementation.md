# imagePullSecrets Support Implementation

## TL;DR

> **Quick Summary**: Add Kubernetes imagePullSecrets support to enable private registry authentication (Harbor, Docker Hub private, etc.) by creating ephemeral dockerconfigjson Secrets and attaching them to Jobs.
>
> **Deliverables**:
> - API request fields for registry credentials
> - Secret creation/deletion methods in JobCreator
> - ImagePullSecrets field in Job PodSpec
> - RBAC updates for Secret permissions
> - Test coverage for new functionality
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Models → JobCreator → TaskManager → Tests

---

## Context

### Original Request
Implement private registry authentication support for the Image Prewarm Service (IPS) using Kubernetes native imagePullSecrets approach. The service currently creates Jobs to prewarm container images but lacks support for pulling from private registries.

### Current Implementation Analysis

**Architecture:**
- **API Layer**: Gin-based handlers with validation tags (`internal/api/handler/task.go`)
- **Service Layer**: TaskManager orchestrates lifecycle, BatchScheduler executes batches, JobCreator creates K8s Jobs
- **K8s Layer**: Uses client-go v0.29.0, Jobs with initContainers to pull images via crictl
- **Models**: CreateTaskRequest with validation, Task model for persistence
- **RBAC**: ClusterRole with permissions for nodes, jobs, pods (no Secret permissions)

**Current Job Creation Pattern:**
```go
Job naming: prewarm-{taskID}-{nodeName}
TTL: 900 seconds
Labels: app=image-prewarm, task-id=xxx, node=xxx
NodeSelector: kubernetes.io/hostname: {nodeName}
Tolerations: All taints
```

**Batch Execution Flow:**
```
API → TaskManager.CreateTask() → TaskManager.executeTask()
  → BatchScheduler.ExecuteBatches()
    → JobCreator.CreateJob() for each node
    → StatusTracker.TrackTask()
```

### Design Decisions

**Credential Structure**: Single registry per task (Registry, Username, Password fields)
- Simpler implementation, sufficient for most use cases
- Extensible to multiple registries later if needed

**Secret Lifecycle**: Per-task Secret
- Create Secret before first Job creation
- Delete Secret after all Jobs complete
- Secret naming: `image-pull-secret-{taskID}`

**Credential Persistence**: Ephemeral only
- Credentials NOT stored in Task model (security)
- Passed through request → execution
- Lost on service restart (acceptable for ephemeral Secret approach)

**Error Handling**: Fail task early
- Secret creation failure = task fails immediately
- Clear error messages for debugging

---

## Work Objectives

### Core Objective
Enable IPS to pull container images from private registries by creating ephemeral Kubernetes Secrets with registry credentials and attaching them to Jobs.

### Concrete Deliverables
- Extended `CreateTaskRequest` with credential fields (Registry, Username, Password)
- `JobCreator.CreateSecret()` method for dockerconfigjson Secrets
- Modified `JobCreator.CreateJob()` to add ImagePullSecrets to PodSpec
- `JobCreator.DeleteSecret()` method for cleanup
- Updated `BatchScheduler` and `TaskManager` to handle Secret lifecycle
- Updated RBAC permissions for Secret operations
- Test coverage for Secret creation, Job creation with secrets, and credential validation

### Definition of Done
- [ ] Create task with credentials → Secret created successfully
- [ ] Jobs reference Secret in imagePullSecrets field
- [ ] Images pull from private registry successfully
- [ ] Secret deleted after task completion
- [ ] All tests pass (unit + handler tests)
- [ ] RBAC allows Secret operations

### Must Have
- Credential validation in API handler (non-empty username/password)
- Secret type: `kubernetes.io/dockerconfigjson`
- Base64-encoded .dockerconfigjson format
- Secret cleanup on task completion/cancellation
- RBAC permissions for Secret get/create/delete
- Log sanitization (never log passwords)

### Must NOT Have (Guardrails)
- Credentials stored persistently in Task model or repository
- Multiple registries per task (out of scope for MVP)
- Secret TTL-based cleanup (must be explicit delete)
- Hardcoded credential values in tests
- Credential exposure in API responses (exclude from JSON)

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (Go tests with fake client)
- **User wants tests**: Tests after implementation
- **Framework**: Go testing with fake.NewSimpleClientset()

### Test Coverage Strategy

**Unit Tests (fake client):**
- Secret creation with valid credentials
- Secret creation error handling
- Job creation with imagePullSecrets reference
- Secret deletion cleanup
- Credential validation in API handler

**Handler Tests (existing pattern):**
- Create task with credentials → success
- Create task with invalid credentials → 400 error
- Create task with missing credentials → 400 error

**Integration Validation (manual verification):**
- Deploy updated RBAC
- Create task with Harbor credentials
- Verify Job has imagePullSecrets
- Verify ImagePull succeeds

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately):
├── Task 1: Add credential fields to models
├── Task 2: Update RBAC permissions
└── Task 3: Add Secret creation/deletion methods to JobCreator

Wave 2 (After Wave 1):
├── Task 4: Modify JobCreator.CreateJob() to use ImagePullSecrets
├── Task 5: Update BatchScheduler to pass credentials
└── Task 6: Update TaskManager for Secret lifecycle

Wave 3 (After Wave 2):
├── Task 7: Add credential validation to API handler
├── Task 8: Write unit tests for JobCreator methods
└── Task 9: Write handler tests for credential validation

Wave 4 (After Wave 3):
└── Task 10: Manual verification and integration testing
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 1 | None | 4, 5 | 2, 3 |
| 2 | None | 10 | 1, 3 |
| 3 | None | 4, 5 | 1, 2 |
| 4 | 1, 3 | 5, 6 | None |
| 5 | 1, 4 | 6 | None |
| 6 | 4, 5 | 7 | None |
| 7 | 6 | 8, 9 | None |
| 8 | 3, 7 | 10 | 9 |
| 9 | 1, 7 | 10 | 8 |
| 10 | 2, 8, 9 | None | None (final) |

Critical Path: Task 1 → Task 3 → Task 4 → Task 5 → Task 6 → Task 7 → Task 8 → Task 10
Parallel Speedup: ~30% faster than sequential

---

## TODOs

- [ ] 1. Add Credential Fields to CreateTaskRequest

  **What to do**:
  - Add `Registry string` field (required if credentials provided)
  - Add `Username string` field (required if Registry provided)
  - Add `Password string` field (required if Registry provided)
  - Add Gin validation tags: `binding:"required_with=Registry,omitempty"`
  - Update JSON tags to ensure credentials are NOT exposed in responses
  - Consider using `json:"-"` for Password field or custom marshaler

  **Must NOT do**:
  - Store credentials in Task model
  - Log credentials in any handler/service
  - Return credentials in API responses

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple model field additions, straightforward Go struct changes
  - **Skills**: None required - basic Go struct modifications
  - **Skills Evaluated but Omitted**:
    - `git-master`: Not needed for initial code changes, will commit later

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Tasks 4, 5 (BatchScheduler, JobCreator changes depend on request structure)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `pkg/models/request.go:CreateTaskRequest` - Existing field structure with Gin validation tags
  - `pkg/models/request.go:11-12` - Priority field with default value and validation

  **Data Contract References**:
  - `internal/api/handler/task.go:30-38` - Handler binding pattern with ShouldBindJSON
  - `internal/api/handler/task.go:32` - Error handling for binding failures

  **Test References**:
  - `internal/api/handler/task_test.go:73-76` - CreateTaskRequest test data pattern
  - `internal/api/handler/task_test.go:69-99` - Test setup and assertion patterns

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: CreateTaskRequest with valid credentials → binding succeeds
  - [ ] Test: CreateTaskRequest with Registry but no Username → binding fails
  - [ ] Test: CreateTaskRequest with Registry but no Password → binding fails
  - [ ] `go test ./pkg/models/` → PASS (3 new tests)

  **Automated Verification (Go test):**
  ```bash
  # Test validation rules work correctly
  go test -v -run TestCreateTaskRequest_Credentials ./pkg/models/
  # Assert: PASS with validation errors for missing fields
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing validation behavior
  - [ ] Code diff showing added fields

  **Commit**: YES (groups with Task 3)
  - Message: `feat(models): add registry credential fields to CreateTaskRequest`
  - Files: `pkg/models/request.go`

---

- [ ] 2. Update RBAC Permissions for Secret Operations

  **What to do**:
  - Add Secret resource permissions to ClusterRole in `deploy/rbac.yaml`
  - Required verbs: `get`, `create`, `delete`
  - Add to existing rules section
  - Maintain existing permissions (nodes, jobs, pods)

  **Must NOT do**:
  - Grant Secret list/watch permissions (not needed)
  - Modify existing permissions for other resources
  - Add Secret permissions to wrong resource group (should be core/v1)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple YAML addition, Kubernetes RBAC configuration
  - **Skills**: None required - YAML RBAC updates
  - **Skills Evaluated but Omitted**:
    - `kubernetes-specialist`: Overkill for simple RBAC permission addition

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 10 (manual verification requires RBAC applied)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Configuration References**:
  - `deploy/rbac.yaml:11-37` - Current ClusterRole rules structure
  - `deploy/rbac.yaml:24-27` - Jobs permissions pattern (add similar for Secrets)

  **Kubernetes Documentation**:
  - Official K8s RBAC: Secret resource in core/v1 API group
  - Required verbs for Secret lifecycle: get (read), create, delete

  **Acceptance Criteria**:

  **Automated Verification (kubectl dry-run):**
  ```bash
  # Verify YAML is valid
  kubectl apply --dry-run=client -f deploy/rbac.yaml
  # Assert: No errors, configuration valid

  # Verify Secret permissions present
  kubectl get clusterrole ips-apiserver -o yaml | grep -A5 'secrets'
  # Assert: Shows verbs ["get", "create", "delete"]
  ```

  **Evidence to Capture**:
  - [ ] kubectl dry-run output showing valid RBAC
  - [ ] YAML diff showing added Secret permissions

  **Commit**: YES (group with Task 10 after verification)
  - Message: `feat(rbac): add Secret permissions for imagePullSecrets support`
  - Files: `deploy/rbac.yaml`
  - Pre-commit: `kubectl apply --dry-run=client -f deploy/rbac.yaml`

---

- [ ] 3. Add Secret Creation and Deletion Methods to JobCreator

  **What to do**:
  - Create `CreateSecret(ctx, taskID, registry, username, password)` method
  - Construct `.dockerconfigjson` data structure with base64 encoding
  - Create Secret with type `kubernetes.io/dockerconfigjson`
  - Secret name: `image-pull-secret-{taskID}`
  - Add labels: `app=image-prewarm`, `task-id={taskID}`
  - Create `DeleteSecret(ctx, secretName)` method
  - Use client-go: `client.Clientset.CoreV1().Secrets(ns).Create()`

  **Must NOT do**:
  - Log credentials in method implementation
  - Create Secret without labels (tracking required)
  - Use wrong Secret type (must be kubernetes.io/dockerconfigjson)
  - Forget base64 encoding of .dockerconfigjson

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Client-go Secret creation follows existing patterns
  - **Skills**: None required - standard Kubernetes client-go usage
  - **Skills Evaluated but Omitted**:
    - `librarian`: Not needed, Kubernetes Secret API is standard

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Task 4 (JobCreator.CreateJob needs Secret creation logic)
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `internal/k8s/job_creator.go:45-133` - Job creation pattern (Create method structure)
  - `internal/k8s/job_creator.go:52-60` - Job metadata labels pattern
  - `internal/k8s/job_creator.go:127-130` - Clientset API call pattern with error handling

  **Kubernetes API References**:
  - corev1.Secret type and fields (Type, Data, ObjectMeta)
  - `.dockerconfigjson` format: `{"auths":{"registry":{"username":"xxx","password":"xxx","auth":"base64(user:pass)"}}}`
  - Secret creation API: `clientset.CoreV1().Secrets(ns).Create(ctx, secret, opts)`

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: CreateSecret with valid credentials → Secret created, returns secretName
  - [ ] Test: CreateSecret returns Secret name in correct format
  - [ ] Test: DeleteSecret with valid name → Secret deleted, no error
  - [ ] Test: DeleteSecret with invalid name → error handled gracefully
  - [ ] `go test ./internal/k8s/ -v -run TestSecret` → PASS (4 tests)

  **Automated Verification (Go test with fake client):**
  ```bash
  # Unit test Secret creation
  go test -v ./internal/k8s/ -run TestJobCreator_CreateSecret
  # Assert: PASS, Secret created with correct type and labels

  # Unit test Secret deletion
  go test -v ./internal/k8s/ -run TestJobCreator_DeleteSecret
  # Assert: PASS, Secret deleted successfully
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing Secret creation/deletion
  - [ ] Code showing .dockerconfigjson construction with base64 encoding

  **Commit**: YES (group with Task 1)
  - Message: `feat(k8s): add Secret creation/deletion methods to JobCreator`
  - Files: `internal/k8s/job_creator.go`
  - Pre-commit: `go test ./internal/k8s/`

---

- [ ] 4. Modify JobCreator.CreateJob() to Support ImagePullSecrets

  **What to do**:
  - Add optional `secretName string` parameter to `CreateJob()`
  - If `secretName` is not empty, add `ImagePullSecrets` to PodSpec
  - ImagePullSecrets reference: `corev1.LocalObjectReference{Name: secretName}`
  - Add to `job.Spec.Template.Spec.ImagePullSecrets` array
  - Maintain backward compatibility: if `secretName` is empty, work as before

  **Must NOT do**:
  - Break existing functionality (must work without credentials)
  - Add ImagePullSecrets without checking if secretName is provided
  - Modify Job naming or other metadata
  - Change existing container configuration

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Existing Job creation with optional parameter addition
  - **Skills**: None required - straightforward Go API modification
  - **Skills Evaluated but Omitted**:
    - None - simple parameter addition

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (Wave 2)
  - **Blocks**: Task 5 (BatchScheduler depends on updated signature)
  - **Blocked By**: Task 1 (needs credential fields), Task 3 (needs Secret creation logic)

  **References**:

  **Pattern References**:
  - `internal/k8s/job_creator.go:45-133` - Current CreateJob implementation
  - `internal/k8s/job_creator.go:72-121` - PodSpec structure to modify
  - `internal/k8s/job_creator.go:64-65` - Spec-level fields (add ImagePullSecrets after TTLSecondsAfterFinished)

  **Kubernetes API References**:
  - `corev1.PodSpec.ImagePullSecrets []corev1.LocalObjectReference`
  - `corev1.LocalObjectReference{Name: "secret-name"}`

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: CreateJob with secretName → Job has ImagePullSecrets field
  - [ ] Test: CreateJob without secretName → Job created without ImagePullSecrets
  - [ ] Test: CreateJob backward compatibility → existing tests still pass
  - [ ] `go test ./internal/k8s/ -v` → PASS (existing + 2 new tests)

  **Automated Verification (Go test):**
  ```bash
  # Test Job creation with ImagePullSecrets
  go test -v ./internal/k8s/ -run TestJobCreator_CreateJob_WithSecret
  # Assert: PASS, Job.Spec.Template.Spec.ImagePullSecrets contains secret

  # Test backward compatibility
  go test -v ./internal/k8s/ -run TestJobCreator_CreateJob_NoSecret
  # Assert: PASS, Job created without ImagePullSecrets field
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing ImagePullSecrets in Job spec
  - [ ] Code diff showing parameter and PodSpec modification

  **Commit**: YES (group with Task 5)
  - Message: `feat(k8s): add ImagePullSecrets support to JobCreator.CreateJob`
  - Files: `internal/k8s/job_creator.go`
  - Pre-commit: `go test ./internal/k8s/`

---

- [ ] 5. Update BatchScheduler to Pass Credentials to JobCreator

  **What to do**:
  - Add optional `secretName string` parameter to `ExecuteBatches()`
  - Pass `secretName` to `jobCreator.CreateJob()` calls
  - Maintain backward compatibility: if empty, pass empty string
  - Update method signature in `internal/service/batch_scheduler.go`

  **Must NOT do**:
  - Create or delete Secrets in BatchScheduler (TaskManager handles lifecycle)
  - Modify batch splitting or execution logic
  - Break existing ExecuteBatches calls without credentials

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple parameter pass-through, no business logic changes
  - **Skills**: None required - parameter addition and propagation
  - **Skills Evaluated but Omitted**:
    - None - straightforward change

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (Wave 2)
  - **Blocks**: Task 6 (TaskManager depends on updated signature)
  - **Blocked By**: Task 4 (JobCreator signature updated)

  **References**:

  **Pattern References**:
  - `internal/service/batch_scheduler.go:33-40` - ExecuteBatches method signature
  - `internal/service/batch_scheduler.go:64-65` - jobCreator.CreateJob() call pattern

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: ExecuteBatches with secretName → JobCreator called with secret
  - [ ] Test: ExecuteBatches backward compatibility → works without secretName
  - [ ] `go test ./internal/service/ -v -run TestBatchScheduler` → PASS (existing + 2 new tests)

  **Automated Verification (Go test):**
  ```bash
  # Test with secretName parameter
  go test -v ./internal/service/ -run TestBatchScheduler_ExecuteBatches_WithSecret
  # Assert: PASS, secretName passed to JobCreator

  # Test backward compatibility
  go test -v ./internal/service/ -run TestBatchScheduler_ExecuteBatches_NoSecret
  # Assert: PASS, batch execution works without credentials
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing secretName propagation
  - [ ] Code diff showing parameter addition

  **Commit**: YES (group with Task 4)
  - Message: `feat(service): add secretName parameter to BatchScheduler.ExecuteBatches`
  - Files: `internal/service/batch_scheduler.go`
  - Pre-commit: `go test ./internal/service/`

---

- [ ] 6. Update TaskManager for Secret Lifecycle Management

  **What to do**:
  - In `executeTask()`: Create Secret before calling BatchScheduler
  - Call `jobCreator.CreateSecret()` with credentials from request
  - Pass `secretName` to `BatchScheduler.ExecuteBatches()`
  - After `StatusTracker.TrackTask()` completes: Delete Secret
  - Handle task cancellation: Delete Secret in cancellation path
  - Handle Secret creation failure: Mark task as failed with clear error message
  - Credentials should NOT be stored in Task model (ephemeral only)

  **Must NOT do**:
  - Store credentials in Task struct or repository
  - Log credentials in TaskManager
  - Forget to delete Secret on failure/cancellation
  - Modify existing retry logic (Secret is created fresh on each attempt)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Adding Secret lifecycle orchestration to existing flow
  - **Skills**: None required - orchestration logic in existing method
  - **Skills Evaluated but Omitted**:
    - None - logical addition to existing flow

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (Wave 2)
  - **Blocks**: Task 7 (API handler depends on TaskManager signature)
  - **Blocked By**: Task 4, 5 (JobCreator and BatchScheduler updated)

  **References**:

  **Pattern References**:
  - `internal/service/task_manager.go:61-162` - CreateTask method (validate, create, enqueue)
  - `internal/service/task_manager.go:164-249` - executeTask method (main execution flow)
  - `internal/service/task_manager.go:172-179` - Node filtering pattern (add Secret creation after)
  - `internal/service/task_manager.go:218-235` - BatchScheduler.ExecuteBatches call (pass secretName)
  - `internal/service/task_manager.go:242-245` - StatusTracker call (add Secret deletion after)
  - `internal/service/task_manager.go:381-437` - DeleteTask method (add Secret cleanup if task cancelled)

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: CreateTask with credentials → Secret created, Jobs use secret
  - [ ] Test: Secret creation failure → task marked failed with error
  - [ ] Test: Task completion → Secret deleted
  - [ ] Test: Task cancellation → Secret deleted
  - [ ] Test: CreateTask without credentials → works as before
  - [ ] `go test ./internal/service/ -v -run TestTaskManager` → PASS (existing + 5 new tests)

  **Automated Verification (Go test):**
  ```bash
  # Test Secret lifecycle in task execution
  go test -v ./internal/service/ -run TestTaskManager_SecretLifecycle
  # Assert: PASS, Secret created before execution, deleted after

  # Test Secret creation failure handling
  go test -v ./internal/service/ -run TestTaskManager_SecretCreationFailure
  # Assert: PASS, task marked failed with clear error message
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing Secret lifecycle
  - [ ] Code showing Secret create/delete orchestration
  - [ ] Logs showing Secret operations (without credentials)

  **Commit**: YES (group with Task 7)
  - Message: `feat(service): add Secret lifecycle management to TaskManager`
  - Files: `internal/service/task_manager.go`
  - Pre-commit: `go test ./internal/service/`

---

- [ ] 7. Add Credential Validation to API Handler

  **What to do**:
  - In `TaskHandler.CreateTask()`: Validate credential consistency
  - If Registry is provided, Username and Password must be provided
  - If Username is provided, Registry must be provided
  - Return 400 Bad Request with clear error messages for invalid credentials
  - Ensure credentials are NOT included in API response JSON
  - Password field should use `json:"password"` but exclude from response

  **Must NOT do**:
  - Log credentials in handler
  - Return credentials in response
  - Allow partial credential configuration (Registry without password)
  - Modify other handler methods

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Validation logic addition, existing error handling pattern
  - **Skills**: None required - validation and error response pattern
  - **Skills Evaluated but Omitted**:
    - None - straightforward validation addition

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (Wave 3)
  - **Blocks**: Tasks 8, 9 (tests depend on handler implementation)
  - **Blocked By**: Task 6 (TaskManager signature updated)

  **References**:

  **Pattern References**:
  - `internal/api/handler/task.go:29-51` - CreateTask handler with binding and error handling
  - `internal/api/handler/task.go:32-38` - Validation error response pattern
  - `internal/api/handler/task.go:40-48` - Task creation error handling pattern

  **Acceptance Criteria**:

  **If TDD (tests enabled):**
  - [ ] Test: CreateTask with valid credentials → 201 Created
  - [ ] Test: CreateTask with Registry but no Password → 400 Bad Request
  - [ ] Test: CreateTask with Username but no Registry → 400 Bad Request
  - [ ] Test: CreateTask without credentials → 201 Created (backward compatible)
  - [ ] Test: Response does NOT include password field
  - [ ] `go test ./internal/api/handler/ -v -run TestTaskHandler_CreateTask_Credentials` → PASS (5 new tests)

  **Automated Verification (Go test):**
  ```bash
  # Test credential validation
  go test -v ./internal/api/handler/ -run TestTaskHandler_CreateTask_Credentials
  # Assert: PASS, validation errors returned correctly

  # Test backward compatibility (no credentials)
  go test -v ./internal/api/handler/ -run TestTaskHandler_CreateTask_NoCredentials
  # Assert: PASS, task created without credentials
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing validation behavior
  - [ ] HTTP response examples showing error messages
  - [ ] JSON response showing password field excluded

  **Commit**: YES (group with Task 6)
  - Message: `feat(handler): add credential validation to CreateTask endpoint`
  - Files: `internal/api/handler/task.go`
  - Pre-commit: `go test ./internal/api/handler/`

---

- [ ] 8. Write Unit Tests for JobCreator Methods

  **What to do**:
  - Write `TestJobCreator_CreateSecret` - verify Secret creation with correct type, labels, data
  - Write `TestJobCreator_DeleteSecret` - verify Secret deletion
  - Write `TestJobCreator_CreateJob_WithSecret` - verify Job has ImagePullSecrets
  - Write `TestJobCreator_CreateJob_NoSecret` - verify backward compatibility
  - Use fake.NewSimpleClientset() for testing
  - Follow existing test patterns in codebase

  **Must NOT do**:
  - Use actual K8s cluster for unit tests
  - Hardcode real credentials in tests
  - Skip testing error cases
  - Skip testing backward compatibility

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: Test writing following existing patterns
  - **Skills**: None required - Go testing patterns
  - **Skills Evaluated but Omitted**:
    - None - straightforward test implementation

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 9)
  - **Blocks**: Task 10 (verification needs passing tests)
  - **Blocked By**: Task 3, 4 (JobCreator methods implemented)

  **References**:

  **Test Pattern References**:
  - `internal/api/handler/task_test.go:22-67` - Test setup with fake client pattern
  - `internal/service/batch_scheduler_test.go:7-61` - Unit test pattern for scheduler
  - `internal/api/handler/task_test.go:37-40` - fakeClientset initialization

  **Acceptance Criteria**:

  **Automated Verification (Go test):**
  ```bash
  # Run all JobCreator tests
  go test -v ./internal/k8s/ -run TestJobCreator
  # Assert: All PASS (existing + 4 new tests)

  # Run all tests in package
  go test ./internal/k8s/
  # Assert: 0 failures, coverage > 80%
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing all tests pass
  - [ ] Test coverage report

  **Commit**: YES (group with Task 9)
  - Message: `test(k8s): add unit tests for Secret and JobCreator methods`
  - Files: `internal/k8s/job_creator_test.go` (create if not exists)
  - Pre-commit: `go test ./internal/k8s/`

---

- [ ] 9. Write Handler Tests for Credential Validation

  **What to do**:
  - Write `TestTaskHandler_CreateTask_WithCredentials` - verify success with valid credentials
  - Write `TestTaskHandler_CreateTask_MissingPassword` - verify 400 error
  - Write `TestTaskHandler_CreateTask_MissingUsername` - verify 400 error
  - Write `TestTaskHandler_CreateTask_NoCredentials` - verify backward compatibility
  - Verify password not in response JSON
  - Use httptest and gin.TestMode pattern

  **Must NOT do**:
  - Skip testing validation edge cases
  - Use real HTTP server (use httptest)
  - Skip testing backward compatibility

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
    - Reason: Handler test writing following existing patterns
  - **Skills**: None required - Gin handler testing patterns
  - **Skills Evaluated but Omitted**:
    - None - test implementation following existing patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 8)
  - **Blocks**: Task 10 (verification needs passing tests)
  - **Blocked By**: Task 7 (handler implementation complete)

  **References**:

  **Test Pattern References**:
  - `internal/api/handler/task_test.go:69-99` - CreateTask test pattern
  - `internal/api/handler/task_test.go:73-76` - Request body construction pattern
  - `internal/api/handler/task_test.go:101-115` - Invalid request test pattern
  - `internal/api/handler/task_test.go:62-64` - Gin test mode setup

  **Acceptance Criteria**:

  **Automated Verification (Go test):**
  ```bash
  # Run handler tests
  go test -v ./internal/api/handler/ -run TestTaskHandler_CreateTask_Credentials
  # Assert: All PASS (5 new tests)

  # Run all handler tests
  go test ./internal/api/handler/
  # Assert: 0 failures, coverage > 70%
  ```

  **Evidence to Capture**:
  - [ ] Go test output showing validation tests pass
  - [ ] HTTP response examples from tests

  **Commit**: YES (group with Task 8)
  - Message: `test(handler): add credential validation tests to task handler`
  - Files: `internal/api/handler/task_test.go`
  - Pre-commit: `go test ./internal/api/handler/`

---

- [ ] 10. Manual Verification and Integration Testing

  **What to do**:
  - Apply updated RBAC to Kubernetes cluster: `kubectl apply -f deploy/rbac.yaml`
  - Start IPS service: `make run` or deploy to cluster
  - Create test task with Harbor credentials (use test registry)
  - Verify Secret created: `kubectl get secret image-pull-secret-{taskID} -n {namespace} -o yaml`
  - Verify Job has imagePullSecrets: `kubectl get job prewarm-{taskID}-{node} -o yaml | grep imagePullSecrets -A2`
  - Verify ImagePull succeeds: `kubectl describe pod <job-pod> | grep -A10 Events`
  - Verify Secret deleted after task completion: `kubectl get secret image-pull-secret-{taskID}` (should be not found)
  - Test cancellation: cancel task, verify Secret deleted
  - Test invalid credentials: create task with bad password, verify task fails

  **Must NOT do**:
  - Use production credentials for testing
  - Skip verifying Secret cleanup
  - Skip testing cancellation scenario

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Manual verification with kubectl commands
  - **Skills**: [`dev-browser`]
    - `dev-browser`: For navigating web UI and testing form submission if needed
  - **Skills Evaluated but Omitted**:
    - `playwright`: Not needed, verification via kubectl and API

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Final (Wave 4)
  - **Blocks**: None (final task)
  - **Blocked By**: Tasks 2, 8, 9 (RBAC, tests must pass)

  **References**:

  **Verification References**:
  - `deploy/rbac.yaml` - RBAC to apply
  - `README.md` - Deployment and running instructions
  - Existing Secret verification patterns in K8s documentation

  **Acceptance Criteria**:

  **Automated Verification (kubectl commands):**
  ```bash
  # 1. Apply RBAC
  kubectl apply -f deploy/rbac.yaml
  # Assert: clusterrole/clusterrolebinding configured

  # 2. Verify Secret permissions
  kubectl auth can-i create secrets --as=system:serviceaccount:ips-system:ips-apiserver
  # Assert: yes

  # 3. Create task with credentials via API
  curl -X POST http://localhost:8080/api/v1/tasks \
    -H "Content-Type: application/json" \
    -d '{"images":["harbor.example.com/test/image:latest"],"batchSize":1,"registry":"harbor.example.com","username":"testuser","password":"testpass"}'
  # Assert: 201 Created, returns taskId

  # 4. Verify Secret created
  kubectl get secret image-pull-secret-{taskId} -n default -o yaml | grep kubernetes.io/dockerconfigjson
  # Assert: Secret exists with correct type

  # 5. Verify Job has imagePullSecrets
  kubectl get job prewarm-{taskId}-{nodeName} -n default -o yaml | grep imagePullSecrets -A2
  # Assert: Shows imagePullSecrets: - name: image-pull-secret-{taskId}

  # 6. Wait for task completion
  sleep 60  # Wait for job to complete
  kubectl get task {taskId}  # Via API
  # Assert: status: "completed"

  # 7. Verify Secret deleted
  kubectl get secret image-pull-secret-{taskId} -n default
  # Assert: Error: NotFound (secret deleted)

  # 8. Test cancellation
  curl -X DELETE http://localhost:8080/api/v1/tasks/{taskId}
  # Verify Secret deleted immediately
  ```

  **Evidence to Capture**:
  - [ ] kubectl output showing Secret creation
  - [ ] kubectl output showing Job with imagePullSecrets
  - [ ] kubectl describe output showing successful ImagePull
  - [ ] kubectl output showing Secret deletion
  - [ ] Screenshot of web UI task creation form with credential fields (if applicable)

  **Commit**: YES (final commit with RBAC)
  - Message: `feat(rbac): apply Secret permissions and verify imagePullSecrets support`
  - Files: `deploy/rbac.yaml` (applied to cluster)
  - Verification commands above

---

## Commit Strategy

| After Task | Message | Files | Verification |
|------------|---------|-------|--------------|
| 1, 3 | `feat(models, k8s): add credential fields and Secret methods` | `pkg/models/request.go`, `internal/k8s/job_creator.go` | `go test ./pkg/models/ && go test ./internal/k8s/` |
| 4, 5 | `feat(k8s, service): add ImagePullSecrets support to Job execution` | `internal/k8s/job_creator.go`, `internal/service/batch_scheduler.go` | `go test ./internal/k8s/ && go test ./internal/service/` |
| 6, 7 | `feat(service, handler): add Secret lifecycle and credential validation` | `internal/service/task_manager.go`, `internal/api/handler/task.go` | `go test ./internal/service/ && go test ./internal/api/handler/` |
| 8, 9 | `test: add unit tests for Secret and credential validation` | `internal/k8s/job_creator_test.go`, `internal/api/handler/task_test.go` | `go test ./...` |
| 10 | `feat(rbac): apply Secret permissions and verify integration` | `deploy/rbac.yaml` | Manual verification commands |

---

## Success Criteria

### Verification Commands

```bash
# 1. All unit tests pass
go test ./... -v
# Expected: PASS, 0 failures

# 2. RBAC permissions valid
kubectl apply --dry-run=client -f deploy/rbac.yaml
# Expected: No errors

# 3. Secret creation works
kubectl auth can-i create secrets --as=system:serviceaccount:ips-system:ips-apiserver
# Expected: yes

# 4. End-to-end test with credentials
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"images":["nginx:latest"],"batchSize":1,"registry":"docker.io","username":"test","password":"test"}' \
  | jq '.taskId'
# Expected: Returns taskId

# 5. Verify Secret created
TASK_ID=<from_above>
kubectl get secret image-pull-secret-$TASK_ID -n default
# Expected: Secret exists

# 6. Verify Job uses Secret
kubectl get job prewarm-$TASK_ID-* -n default -o yaml | grep imagePullSecrets -A2
# Expected: Shows imagePullSecrets reference

# 7. Backward compatibility (no credentials)
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"images":["nginx:latest"],"batchSize":1}'
# Expected: 201 Created, works without credentials
```

### Final Checklist

- [ ] CreateTaskRequest has Registry, Username, Password fields with validation
- [ ] RBAC includes Secret get/create/delete permissions
- [ ] JobCreator has CreateSecret and DeleteSecret methods
- [ ] JobCreator.CreateJob accepts optional secretName parameter
- [ ] Job with credentials has ImagePullSecrets in PodSpec
- [ ] TaskManager creates Secret before execution
- [ ] TaskManager deletes Secret after completion
- [ ] TaskManager deletes Secret on cancellation
- [ ] Secret creation failure marks task as failed
- [ ] Credentials NOT stored in Task model
- [ ] Credentials NOT logged anywhere
- [ ] Credentials NOT returned in API responses
- [ ] All unit tests pass (existing + new)
- [ ] Manual verification with real K8s cluster succeeds
- [ ] Backward compatibility maintained (tasks without credentials work)
