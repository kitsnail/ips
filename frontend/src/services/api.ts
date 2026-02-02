import axios, { AxiosError } from 'axios'
import type {
  User,
  LoginRequest,
  LoginResponse,
  CreateUserRequest,
  UpdateUserRequest,
  UpdatePasswordRequest,
  Task,
  CreateTaskRequest,
  ListTasksRequest,
  ListTasksResponse,
  ScheduledTask,
  CreateScheduledTaskRequest,
  UpdateScheduledTaskRequest,
  ListScheduledTasksRequest,
  ListScheduledTasksResponse,
  ScheduledExecution,
  ListScheduledExecutionsRequest,
  ListScheduledExecutionsResponse,
  SaveImageRequest,
  ListImagesResponse,
  Secret,
  CreateSecretRequest,
  UpdateSecretRequest,
  ListSecretsResponse,
  StatsResponse,
} from '@/types/api'

const API_BASE = '/api/v1'

const apiClient = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('ips_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('ips_token')
      localStorage.removeItem('ips_user')
      window.location.href = '/web/'
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const response = await axios.post<LoginResponse>(`${API_BASE}/login`, data)
    localStorage.setItem('ips_token', response.data.token)
    localStorage.setItem('ips_user', JSON.stringify(response.data.user))
    return response.data
  },

  logout: () => {
    localStorage.removeItem('ips_token')
    localStorage.removeItem('ips_user')
  },

  updatePassword: async (userId: number, data: UpdatePasswordRequest): Promise<void> => {
    await apiClient.put(`/users/${userId}`, data)
  },
}

export const taskApi = {
  list: async (params: ListTasksRequest): Promise<ListTasksResponse> => {
    const response = await apiClient.get<ListTasksResponse>('/tasks', { params })
    return response.data
  },

  get: async (id: string): Promise<Task> => {
    const response = await apiClient.get<Task>(`/tasks/${id}`)
    return response.data
  },

  create: async (data: CreateTaskRequest): Promise<Task> => {
    const response = await apiClient.post<Task>('/tasks', data)
    return response.data
  },

  delete: async (id: string): Promise<void> => {
    try {
      const response = await apiClient.delete(`/tasks/${id}`)
      // API返回 { "taskId": "...", "status": "success", "action": "...", "message": "..." }
      // 我们只需要确保请求成功，不需要返回数据
      console.log('Task delete response:', response.data)
    } catch (error) {
      console.error('Task delete error:', error)
      throw error
    }
  },
}

export const scheduledTaskApi = {
  list: async (params: ListScheduledTasksRequest): Promise<ListScheduledTasksResponse> => {
    const response = await apiClient.get<ListScheduledTasksResponse>('/scheduled-tasks', { params })
    return response.data
  },

  get: async (id: string): Promise<ScheduledTask> => {
    const response = await apiClient.get<ScheduledTask>(`/scheduled-tasks/${id}`)
    return response.data
  },

  create: async (data: CreateScheduledTaskRequest): Promise<ScheduledTask> => {
    const response = await apiClient.post<ScheduledTask>('/scheduled-tasks', data)
    return response.data
  },

  update: async (id: string, data: UpdateScheduledTaskRequest): Promise<ScheduledTask> => {
    const response = await apiClient.put<ScheduledTask>(`/scheduled-tasks/${id}`, data)
    return response.data
  },

  delete: async (id: string): Promise<void> => {
    try {
      const response = await apiClient.delete(`/scheduled-tasks/${id}`)
      console.log('Scheduled task delete response:', response.data)
    } catch (error) {
      console.error('Scheduled task delete error:', error)
      throw error
    }
  },

  enable: async (id: string): Promise<void> => {
    await apiClient.put(`/scheduled-tasks/${id}/enable`)
  },

  disable: async (id: string): Promise<void> => {
    await apiClient.put(`/scheduled-tasks/${id}/disable`)
  },

  trigger: async (id: string): Promise<void> => {
    await apiClient.post(`/scheduled-tasks/${id}/trigger`)
  },

  listExecutions: async (params: ListScheduledExecutionsRequest): Promise<ListScheduledExecutionsResponse> => {
    const response = await apiClient.get<ListScheduledExecutionsResponse>(`/scheduled-tasks/${params.scheduledTaskId}/executions`, { params })
    return response.data
  },

  getExecution: async (scheduledTaskId: string, executionId: number): Promise<ScheduledExecution> => {
    const response = await apiClient.get<ScheduledExecution>(`/scheduled-tasks/${scheduledTaskId}/executions/${executionId}`)
    return response.data
  },
}

export const libraryApi = {
  list: async (params?: { limit?: number; offset?: number }): Promise<ListImagesResponse> => {
    const response = await apiClient.get<ListImagesResponse>('/library', { params })
    return response.data
  },

  create: async (data: SaveImageRequest): Promise<void> => {
    await apiClient.post('/library', data)
  },

  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/library/${id}`)
  },
}

export const secretApi = {
  list: async (params?: { page?: number; pageSize?: number }): Promise<ListSecretsResponse> => {
    const response = await apiClient.get<ListSecretsResponse>('/secrets', { params })
    return response.data
  },

  get: async (id: number): Promise<Secret> => {
    const response = await apiClient.get<Secret>(`/secrets/${id}`)
    return response.data
  },

  create: async (data: CreateSecretRequest): Promise<Secret> => {
    const response = await apiClient.post<Secret>('/secrets', data)
    return response.data
  },

  update: async (id: number, data: UpdateSecretRequest): Promise<Secret> => {
    const response = await apiClient.put<Secret>(`/secrets/${id}`, data)
    return response.data
  },

  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/secrets/${id}`)
  },
}

export const userApi = {
  list: async (): Promise<User[]> => {
    const response = await apiClient.get<User[]>('/users')
    return response.data
  },

  create: async (data: CreateUserRequest): Promise<User> => {
    const response = await apiClient.post<User>('/users', data)
    return response.data
  },

  update: async (id: number, data: UpdateUserRequest): Promise<User> => {
    const response = await apiClient.put<User>(`/users/${id}`, data)
    return response.data
  },

  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/users/${id}`)
  },
}

export const statsApi = {
  getStats: async (): Promise<StatsResponse> => {
    const response = await apiClient.get<StatsResponse>('/stats')
    return response.data
  },
}

export const healthApi = {
  check: async (): Promise<string> => {
    const response = await axios.get<string>('/health')
    return response.data
  },
}
