// API Response Types
export interface ApiResponse<T> {
  data?: T
  error?: string
  details?: string
}

// User Types
export type UserRole = 'admin' | 'viewer'

export interface User {
  id: number
  username: string
  role: UserRole
  createdAt: string
  updatedAt: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface CreateUserRequest {
  username: string
  password: string
  role: UserRole
}

export interface UpdateUserRequest {
  role: UserRole
}

export interface UpdatePasswordRequest {
  newPassword: string
}

// Task Types
export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface Progress {
  totalNodes: number
  completedNodes: number
  failedNodes: number
  currentBatch: number
  totalBatches: number
  percentage: number
}

export interface FailedNode {
  nodeName: string
  image: string
  reason: string
  message?: string
  timestamp: string
}

export interface Task {
  taskId: string
  status: TaskStatus
  priority: number
  images: string[]
  batchSize: number
  nodeSelector?: Record<string, string>
  progress?: Progress
  failedNodeDetails?: FailedNode[]
  maxRetries: number
  retryCount: number
  retryStrategy: 'linear' | 'exponential'
  retryDelay?: number
  webhookUrl?: string
  secretName?: string
  secretId?: number
  registry?: string
  username?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
  estimatedCompletion?: string
  errorMessage?: string
  nodeStatuses?: Record<string, Record<string, number>>
}

export interface CreateTaskRequest {
  images: string[]
  batchSize: number
  priority: number
  nodeSelector?: Record<string, string>
  maxRetries: number
  retryStrategy: 'linear' | 'exponential'
  retryDelay?: number
  webhookUrl?: string
  secretId?: number
  registry?: string
  username?: string
  password?: string
}

export interface ListTasksRequest {
  limit?: number
  offset?: number
  status?: TaskStatus
}

export interface ListTasksResponse {
  tasks: Task[]
  total: number
  limit: number
  offset: number
}

// Scheduled Task Types
export type OverlapPolicy = 'skip' | 'allow' | 'queue'

export interface TaskConfig {
  images: string[]
  batchSize: number
  priority: number
  nodeSelector?: Record<string, string>
  maxRetries: number
  retryStrategy: string
  retryDelay: number
  webhookUrl?: string
  secretId?: number
  registry?: string
  username?: string
  password?: string
}

export type ScheduledExecutionStatus = 'success' | 'failed' | 'skipped' | 'timeout'

export interface ScheduledTask {
  id: string
  name: string
  description: string
  cronExpr: string
  enabled: boolean
  taskConfig: TaskConfig
  overlapPolicy: OverlapPolicy
  timeoutSeconds: number
  lastExecutionAt?: string
  nextExecutionAt?: string
  createdBy: string
  createdAt: string
  updatedAt: string
}

export interface ScheduledExecution {
  id: number
  scheduledTaskId: string
  taskId: string
  status: ScheduledExecutionStatus
  startedAt: string
  finishedAt?: string
  durationSeconds: number
  errorMessage?: string
  triggeredAt: string
}

export interface CreateScheduledTaskRequest {
  name: string
  description: string
  cronExpr: string
  enabled: boolean
  taskConfig: TaskConfig
  overlapPolicy: OverlapPolicy
  timeoutSeconds: number
}

export interface UpdateScheduledTaskRequest {
  name?: string
  description?: string
  cronExpr?: string
  enabled?: boolean
  taskConfig?: TaskConfig
  overlapPolicy?: OverlapPolicy
  timeoutSeconds?: number
}

export interface ListScheduledTasksRequest {
  limit?: number
  offset?: number
  enabled?: boolean
}

export interface ListScheduledTasksResponse {
  tasks: ScheduledTask[]
  total: number
  limit: number
  offset: number
}

export interface ListScheduledExecutionsRequest {
  scheduledTaskId?: string
  limit?: number
  offset?: number
}

export interface ListScheduledExecutionsResponse {
  executions: ScheduledExecution[]
  total: number
  limit: number
  offset: number
}

// Library Types
export interface LibraryImage {
  id: number
  name: string
  image: string
  createdAt: string
}

export interface SaveImageRequest {
  name: string
  image: string
}

export interface ListImagesRequest {
  limit?: number
  offset?: number
}

export interface ListImagesResponse {
  images: LibraryImage[]
  total: number
  limit: number
  offset: number
}

// Secret Types
export interface Secret {
  id: number
  name: string
  registry: string
  username: string
  createdAt: string
  updatedAt: string
}

export interface CreateSecretRequest {
  name: string
  registry: string
  username: string
  password: string
}

export interface UpdateSecretRequest {
  name?: string
  registry?: string
  username?: string
  password?: string
}

export interface ListSecretsRequest {
  page?: number
  pageSize?: number
}

export interface ListSecretsResponse {
  secrets: Secret[]
  total: number
  page: number
  pageSize: number
}

// Pagination Types
export interface PaginationState {
  page: number
  pageSize: number
  total: number
}

export interface FilterState {
  status?: string
  search: string
  enabled?: string
}

// Log Types
export type LogLevel = 'info' | 'warning' | 'error' | 'success'

export interface LogEntry {
  id: number
  timestamp: string
  level: LogLevel
  message: string
  details?: string
}

// Stats Types
export interface NodeStats {
  total: number
  ready: number
  coverage: number
}

export interface StatsResponse {
  nodes: NodeStats
}
