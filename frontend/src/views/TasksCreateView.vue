<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { taskApi, libraryApi, secretApi } from '@/services/api'
import type { CreateTaskRequest, LibraryImage, Secret, Task } from '@/types/api'

const router = useRouter()

const form = ref<CreateTaskRequest>({
  images: [],
  batchSize: 10,
  priority: 5,
  maxRetries: 0,
  retryStrategy: 'linear',
  retryDelay: 30,
  nodeSelector: {},
})

const enableRegistry = ref(false)
const authMode = ref<'manual' | 'select'>('manual')
const loading = ref(false)
const selectedImages = ref<string[]>([])

const libraryImages = ref<LibraryImage[]>([])
const secrets = ref<Secret[]>([])
const libraryLoading = ref(false)
const showLibrarySelector = ref(false)
const showManualInput = ref(false)
const manualImageInput = ref('')
const searchText = ref('')
const sortField = ref<'name' | 'createdAt'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')

// Form validation
const formErrors = ref<Record<string, string>>({})
const formTouched = ref<Record<string, boolean>>({})

const validateForm = (): boolean => {
  const errors: Record<string, string> = {}

  // Validate images
  if (selectedImages.value.length === 0) {
    errors.images = '请至少添加一个镜像'
  } else if (selectedImages.value.length > 50) {
    errors.images = '镜像数量不能超过50个'
  } else {
    errors.images = ''
  }

  // Validate batchSize
  if (form.value.batchSize < 1 || form.value.batchSize > 100) {
    errors.batchSize = '批次大小必须在1-100之间'
  } else {
    errors.batchSize = ''
  }

  // Validate priority
  if (form.value.priority < 1 || form.value.priority > 10) {
    errors.priority = '优先级必须在1-10之间'
  } else {
    errors.priority = ''
  }

  // Validate maxRetries
  if (form.value.maxRetries < 0 || form.value.maxRetries > 5) {
    errors.maxRetries = '最大重试次数必须在0-5之间'
  } else {
    errors.maxRetries = ''
  }

  // Validate retryDelay
  if (!form.value.retryDelay || form.value.retryDelay < 5 || form.value.retryDelay > 300) {
    errors.retryDelay = '重试延迟必须在5-300秒之间'
  } else {
    errors.retryDelay = ''
  }

  // Validate retryStrategy
  if (!['linear', 'exponential'].includes(form.value.retryStrategy)) {
    errors.retryStrategy = '请选择有效的重试策略'
  } else {
    errors.retryStrategy = ''
  }

  // Validate private registry
  if (enableRegistry.value) {
    if (authMode.value === 'manual') {
      if (!form.value.registry || form.value.registry.trim() === '') {
        errors.registry = '请输入仓库地址'
      } else {
        errors.registry = ''
      }

      if (!form.value.username || form.value.username.trim() === '') {
        errors.username = '请输入用户名'
      } else {
        errors.username = ''
      }

      if (!form.value.password || form.value.password.trim() === '') {
        errors.password = '请输入密码'
      } else if (form.value.password.length < 6) {
        errors.password = '密码至少6个字符'
      } else {
        errors.password = ''
      }
    } else if (authMode.value === 'select') {
      if (!form.value.secretId) {
        errors.secretId = '请选择认证信息'
      } else {
        errors.secretId = ''
      }
    }
  } else {
    errors.registry = ''
    errors.username = ''
    errors.password = ''
    errors.secretId = ''
  }

  formErrors.value = errors
  return Object.values(errors).every(error => error === '')
}

const markFieldTouched = (field: string) => {
  formTouched.value[field] = true
}

// Reference panel data
const runningTasks = ref<Task[]>([])
const recentFailedTasks = ref<Task[]>([])
const popularImages = ref<string[]>([])
const statsLoading = ref(false)
const activeTab = ref<'stats' | 'images' | 'history'>('stats')

const loadRunningTasks = async () => {
  try {
    statsLoading.value = true
    const response = await taskApi.list({ limit: 10, status: 'running' as any })
    runningTasks.value = response.tasks
  } catch (error) {
    console.error('Failed to load running tasks:', error)
  } finally {
    statsLoading.value = false
  }
}

const loadRecentFailed = async () => {
  try {
    statsLoading.value = true
    const response = await taskApi.list({ limit: 5, status: 'failed' as any })
    recentFailedTasks.value = response.tasks.slice(0, 3)
  } catch (error) {
    console.error('Failed to load failed tasks:', error)
  } finally {
    statsLoading.value = false
  }
}

const loadPopularImages = async () => {
  try {
    const response = await libraryApi.list({ limit: 10 })
    popularImages.value = response.images.slice(0, 5).map(img => img.image)
  } catch (error) {
    console.error('Failed to load popular images:', error)
  }
}

const loadReferenceData = () => {
  loadRunningTasks()
  loadRecentFailed()
  loadPopularImages()
}

const addPopularImage = (imageUrl: string) => {
  if (!selectedImages.value.includes(imageUrl)) {
    selectedImages.value.push(imageUrl)
  }
}

const getShortImageName = (imageUrl: string): string => {
  const parts = imageUrl.split('/')
  const lastPart = parts[parts.length - 1]
  return lastPart || imageUrl
}

const filteredLibraryImages = computed(() => {
  let filtered = libraryImages.value

  if (searchText.value) {
    const searchLower = searchText.value.toLowerCase()
    filtered = filtered.filter(img =>
      img.name.toLowerCase().includes(searchLower) ||
      img.image.toLowerCase().includes(searchLower)
    )
  }

  if (sortField.value === 'name') {
    filtered = [...filtered].sort((a, b) => {
      const comparison = a.name.localeCompare(b.name)
      return sortOrder.value === 'asc' ? comparison : -comparison
    })
  } else {
    filtered = [...filtered].sort((a, b) => {
      const comparison = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      return sortOrder.value === 'asc' ? comparison : -comparison
    })
  }

  return filtered
})

onMounted(() => {
  loadLibraryImages()
  loadSecrets()
  loadReferenceData()
})

const loadLibraryImages = async () => {
  try {
    libraryLoading.value = true
    const response = await libraryApi.list({ limit: 100 })
    libraryImages.value = response.images
  } catch (error) {
    ElMessage.error('加载镜像库失败')
  } finally {
    libraryLoading.value = false
  }
}

const clearSearch = () => {
  searchText.value = ''
}

const loadSecrets = async () => {
  try {
    const response = await secretApi.list({ pageSize: 100 })
    secrets.value = response.secrets
  } catch (error) {
    ElMessage.error('加载认证信息失败')
  }
}

const addImage = (imageUrl: string) => {
  if (!selectedImages.value.includes(imageUrl)) {
    selectedImages.value.push(imageUrl)
  }
}

const removeImage = (imageUrl: string) => {
  const index = selectedImages.value.indexOf(imageUrl)
  if (index > -1) {
    selectedImages.value.splice(index, 1)
  }
}

const addManualImages = () => {
  const images = manualImageInput.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)

  images.forEach(img => {
    if (!selectedImages.value.includes(img)) {
      selectedImages.value.push(img)
    }
  })

  manualImageInput.value = ''
  showManualInput.value = false
}

const submit = async () => {
  if (!validateForm()) {
    ElMessage.error('表单填写有误，请检查')
    return
  }

  try {
    form.value.images = selectedImages.value
    loading.value = true
    await taskApi.create(form.value)
    ElMessage.success('任务创建成功')
    router.push('/tasks')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '创建任务失败')
  } finally {
    loading.value = false
  }
}

const cancel = () => {
  router.back()
}
</script>

<template>
  <div class="task-create">
    <div class="page-header">
      <h1>创建镜像预热任务</h1>
      <div class="header-actions">
        <el-button @click="cancel">取消</el-button>
        <el-button type="primary" @click="submit" :loading="loading">
          创建任务
        </el-button>
      </div>
    </div>

    <div class="content-wrapper">
      <!-- Left: Form Section -->
      <div class="form-section">
        <el-form label-width="120px">
        <!-- 镜像配置 -->
        <div class="form-section">
          <div class="section-title">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            镜像配置
          </div>

          <div class="selected-images">
            <div v-if="selectedImages.length === 0" class="empty-state">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                  d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              <p>暂无已选镜像</p>
            </div>
            <div v-else class="image-list">
              <div v-for="image in selectedImages" :key="image" class="image-tag">
                <span class="image-url">{{ image }}</span>
                <button class="remove-btn" @click="removeImage(image)">
                  <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path
                      d="M6 18L18 6M6 6l12 12"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                </button>
              </div>
            </div>
            <div v-if="formErrors.images" class="error-message">{{ formErrors.images }}</div>
            <div class="add-buttons">
              <el-button @click="showLibrarySelector = true">
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                从镜像库选择
              </el-button>
              <el-button @click="showManualInput = true">
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M12 4v16m8-8H4"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                手动输入
              </el-button>
            </div>
          </div>
        </div>

        <!-- 任务参数 -->
        <div class="form-section">
          <div class="section-title">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 6V4m0 2a2 2 0 100 4 0 2 2 0 000-4zm0 16v-2m0 2a2 2 0 100 4 0 2 2 0 000-4zm8-8h-2m2 0a2 2 0 100 4 0 2 2 0 000-4zM6 12H4m2 0a2 2 0 100 4 0 2 2 0 000-4z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            任务参数
          </div>
          <el-row :gutter="24">
            <el-col :span="8">
              <el-form-item label="批次大小" :error="!!formErrors.batchSize">
                <el-input-number v-model="form.batchSize" :min="1" :max="100" style="width: 100%" @focus="markFieldTouched('batchSize')" />
                <div v-if="formErrors.batchSize" class="error-hint">{{ formErrors.batchSize }}</div>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="优先级" :error="!!formErrors.priority">
                <el-input-number v-model="form.priority" :min="1" :max="10" style="width: 100%" @focus="markFieldTouched('priority')" />
                <div v-if="formErrors.priority" class="error-hint">{{ formErrors.priority }}</div>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="最大重试" :error="!!formErrors.maxRetries">
                <el-input-number v-model="form.maxRetries" :min="0" :max="5" style="width: 100%" @focus="markFieldTouched('maxRetries')" />
                <div v-if="formErrors.maxRetries" class="error-hint">{{ formErrors.maxRetries }}</div>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="24">
            <el-col :span="12">
              <el-form-item label="重试策略" :error="!!formErrors.retryStrategy">
                <el-select v-model="form.retryStrategy" style="width: 100%" @focus="markFieldTouched('retryStrategy')">
                  <el-option label="线性" value="linear" />
                  <el-option label="指数退避" value="exponential" />
                </el-select>
                <div v-if="formErrors.retryStrategy" class="error-hint">{{ formErrors.retryStrategy }}</div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="重试延迟(秒)" :error="!!formErrors.retryDelay">
                <el-input-number v-model="form.retryDelay" :min="5" :max="300" style="width: 100%" @focus="markFieldTouched('retryDelay')" />
                <div v-if="formErrors.retryDelay" class="error-hint">{{ formErrors.retryDelay }}</div>
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <!-- 私有仓库 -->
        <div class="form-section">
          <div class="section-title">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            私有仓库
            <el-switch v-model="enableRegistry" />
          </div>

          <template v-if="enableRegistry">
            <el-form-item label="认证方式">
              <el-radio-group v-model="authMode">
                <el-radio value="manual">手动输入</el-radio>
                <el-radio value="select">选择已保存</el-radio>
              </el-radio-group>
            </el-form-item>

            <template v-if="authMode === 'manual'">
              <el-form-item label="仓库地址" required :error="!!formErrors.registry">
                <el-input v-model="form.registry" placeholder="harbor.example.com" @focus="markFieldTouched('registry')" />
                <div v-if="formErrors.registry" class="error-hint">{{ formErrors.registry }}</div>
              </el-form-item>
              <el-form-item label="用户名" required :error="!!formErrors.username">
                <el-input v-model="form.username" @focus="markFieldTouched('username')" />
                <div v-if="formErrors.username" class="error-hint">{{ formErrors.username }}</div>
              </el-form-item>
              <el-form-item label="密码" required :error="!!formErrors.password">
                <el-input v-model="form.password" type="password" show-password @focus="markFieldTouched('password')" />
                <div v-if="formErrors.password" class="error-hint">{{ formErrors.password }}</div>
              </el-form-item>
            </template>

            <template v-if="authMode === 'select'">
              <el-form-item label="选择认证" required :error="!!formErrors.secretId">
                <el-select v-model="form.secretId" placeholder="请选择认证" style="width: 100%" @focus="markFieldTouched('secretId')">
                  <el-option
                    v-for="secret in secrets"
                    :key="secret.id"
                    :label="secret.name"
                    :value="secret.id"
                  >
                    <span style="float: left">{{ secret.name }}</span>
                    <span style="float: right; color: #8492a6; font-size: 12px">
                      {{ secret.registry }}
                    </span>
                  </el-option>
                </el-select>
                <div v-if="formErrors.secretId" class="error-hint">{{ formErrors.secretId }}</div>
              </el-form-item>
            </template>
          </template>
        </div>
      </el-form>
      </div>

      <!-- Right: Reference Panel -->
      <div class="reference-panel">
        <div class="panel-tabs">
          <button
            class="tab-button"
            :class="{ active: activeTab === 'stats' }"
            @click="activeTab = 'stats'"
          >
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M3 13h8V3H3v10zm0 0l3-3M3 16l3 3"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            运行统计
          </button>
          <button
            class="tab-button"
            :class="{ active: activeTab === 'images' }"
            @click="activeTab = 'images'"
          >
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M4.318 6.318a4.5 4.5 0 000-9 0 4.5 4.5 0 009 0zM21.5 9a4.5 4.5 0 00-9 0 4.5 4.5 0 009 0zM4.5 9.5a4.5 4.5 0 013 0 4.5 4.5 0 000-9 0zM21.5 12.5a4.5 4.5 0 01-1.38 8.62 4.5 4.5 0 01-8.62 1.38zM4.5 19.5a4.5 4.5 0 01-2.62 3.38 4.5 4.5 0 01-3.38-2.62zM21.5 16.5a4.5 4.5 0 000-9 0 4.5 4.5 0 000 9 0z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            常用镜像
          </button>
        </div>

        <!-- Stats Tab Content -->
        <div v-if="activeTab === 'stats'" class="panel-content">
          <!-- Running Tasks -->
          <div class="stat-card primary">
            <div class="stat-header">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0 9 9 0 01-8-2 9-9 0 00-2 9 9 0 002-8-2v-2a2 2 0 00-2 2 0 000-4zm0 0l3-3M3 16l3 3"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              <span>运行中任务</span>
            </div>
            <div class="stat-value">{{ runningTasks.length }}</div>
            <div v-if="runningTasks.length === 0" class="stat-trend neutral">暂无运行中任务</div>
            <div v-else class="stat-trend">
              <router-link to="/tasks">查看全部</router-link>
            </div>
          </div>

          <!-- Recent Failed Tasks -->
          <div v-if="recentFailedTasks.length > 0" class="failed-tasks-section">
            <div class="section-title-small">最近失败任务</div>
            <div class="failed-list">
              <div
                v-for="task in recentFailedTasks"
                :key="task.taskId"
                class="failed-item"
                @click="router.push(`/tasks/${task.taskId}`)"
              >
                <div class="failed-task-id">{{ task.taskId }}</div>
                <div class="failed-task-error">{{ task.errorMessage || '任务执行失败' }}</div>
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M9 5l7 7-7-7"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
            </div>
          </div>

          <!-- Tips -->
          <div class="tips-section">
            <div class="section-title-small">提示</div>
            <ul class="tips-list">
              <li>批次大小建议设置为10-20,避免单批次节点过多</li>
              <li>优先级越高,任务执行越早</li>
              <li>私有仓库认证信息已保存在"仓库认证"页面</li>
            </ul>
          </div>
        </div>

        <!-- Images Tab Content -->
        <div v-if="activeTab === 'images'" class="panel-content">
          <div class="section-title-small">常用镜像推荐</div>
          <div class="popular-images-list">
            <div
              v-for="image in popularImages"
              :key="image"
              class="popular-image-item"
              :class="{ selected: selectedImages.includes(image) }"
              @click="addPopularImage(image)"
            >
              <div class="popular-image-name">{{ getShortImageName(image) }}</div>
              <div class="popular-image-url">{{ image }}</div>
              <div v-if="selectedImages.includes(image)" class="check-icon">
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M5 13l4 4L19 7"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
            </div>
          </div>

          <div v-if="popularImages.length === 0" class="empty-hint">
            暂无常用镜像数据
          </div>

          <div class="tips-section">
            <div class="section-title-small">镜像使用建议</div>
            <ul class="tips-list">
              <li>推荐使用:latest或指定tag版本</li>
              <li>建议将常用镜像保存到镜像库</li>
              <li>私有仓库镜像需要配置认证信息</li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <!-- Library Selector Dialog -->
    <el-dialog
      v-model="showLibrarySelector"
      title="从镜像库选择"
      width="900px"
      @open="loadLibraryImages"
    >
      <div class="library-header">
        <el-input
          v-model="searchText"
          placeholder="搜索镜像..."
          :prefix-icon="'Search'"
          class="search-input"
          clearable
        />
        <div class="sort-controls">
          <el-select v-model="sortField" size="small" class="sort-select">
            <el-option label="按名称" value="name" />
            <el-option label="按添加时间" value="createdAt" />
          </el-select>
          <el-button
            size="small"
            @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
          >
            <svg v-if="sortOrder === 'asc'" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M18 15l-6-6 6-6M6 6l6 6 6-6"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M6 9l6-6 6 6M6 15l6-6 6-6"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </el-button>
        </div>
      </div>
      <div v-if="libraryLoading" class="loading-text">加载中...</div>
      <div v-else class="library-grid">
        <div
          v-for="img in filteredLibraryImages"
          :key="img.id"
          class="library-item"
          :class="{ selected: selectedImages.includes(img.image) }"
          @click="addImage(img.image)"
        >
          <div class="library-item-name">{{ img.name }}</div>
          <div class="library-item-image">{{ img.image }}</div>
          <div v-if="selectedImages.includes(img.image)" class="check-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M5 13l4 4L19 7"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showLibrarySelector = false; clearSearch()">关闭</el-button>
        <el-button type="primary" @click="showLibrarySelector = false">
          已选择 {{ selectedImages.length }} 个镜像
        </el-button>
      </template>
    </el-dialog>

    <!-- Manual Input Dialog -->
    <el-dialog v-model="showManualInput" title="手动输入镜像" width="600px">
      <el-input
        v-model="manualImageInput"
        type="textarea"
        :rows="8"
        placeholder="每行输入一个镜像地址，例如：&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7"
      />
      <template #footer>
        <el-button @click="showManualInput = false">取消</el-button>
        <el-button type="primary" @click="addManualImages">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.task-create {
  max-width: 1600px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.content-wrapper {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

.form-section {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  flex: 1;
}

.form-section {
  margin-bottom: 32px;
}

.form-section:last-child {
  margin-bottom: 0;
}

.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  margin-bottom: 24px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
  gap: 12px;
}

.section-title svg {
  width: 20px;
  height: 20px;
  color: #0891b2;
}

.selected-images {
  min-height: 120px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  border: 2px dashed #e2e8f0;
  border-radius: 8px;
  color: #94a3b8;
}

.empty-state svg {
  width: 48px;
  height: 48px;
  margin-bottom: 16px;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
}

.image-tag {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #0f172a;
  max-width: 400px;
}

.image-url {
  font-family: monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remove-btn {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 4px;
  color: #94a3b8;
  transition: all 0.2s;
  flex-shrink: 0;
}

.remove-btn:hover {
  background: #ef4444;
  color: white;
}

.remove-btn svg {
  width: 14px;
  height: 14px;
}

.add-buttons {
  display: flex;
  gap: 12px;
}

.error-message {
  color: #ef4444;
  font-size: 13px;
  margin-top: 12px;
  padding: 12px;
  background: #fef2f2;
  border-radius: 6px;
  border-left: 3px solid #ef4444;
}

.error-hint {
  color: #ef4444;
  font-size: 12px;
  margin-top: 4px;
}

/* Responsive */
@media (max-width: 1440px) {
  .task-create {
    max-width: 100%;
  }

  .content-wrapper {
    flex-direction: column;
  }

  .reference-panel {
    width: 100%;
    border-top: 1px solid #e2e8f0;
    padding-top: 24px;
  }

  .form-section {
    border-radius: 12px 12px 0 0;
  }
}

@media (max-width: 1024px) {
  .reference-panel {
    display: none;
  }
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h1 {
    color: #f8fafc;
  }

  .form-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .section-title {
    color: #f8fafc;
    border-bottom-color: #334155;
  }

  .empty-state {
    border-color: #334155;
    color: #64748b;
  }

  .image-tag {
    background: #334155;
    border-color: #475569;
    color: #f8fafc;
  }

  .library-item {
    background: #0f172a;
    border-color: #334155;
  }

  .library-item:hover {
    background: #1e293b;
    border-color: #22d3ee;
  }

  .library-item.selected {
    background: rgba(34, 197, 94, 0.2);
    border-color: #22c55e;
  }

  .library-item-name {
    color: #f8fafc;
  }

  .library-item-image {
    color: #94a3b8;
  }

  .loading-text {
    color: #94a3b8;
  }

  /* Reference Panel Dark Mode */
  .reference-panel {
    background: #1e293b;
    border-color: #334155;
  }

  .panel-tabs {
    border-bottom-color: #334155;
  }

  .tab-button {
    color: #94a3b8;
  }

  .tab-button:hover {
    background: #334155;
    color: #f8fafc;
  }

  .tab-button.active {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .section-title-small {
    color: #cbd5e1;
  }

  .failed-item {
    background: rgba(239, 68, 68, 0.15);
    border-left-color: #f87171;
  }

  .failed-task-id {
    color: #f8fafc;
  }

  .failed-task-error {
    color: #94a3b8;
  }

  .tips-section {
    border-top-color: #334155;
  }

  .tips-list li {
    color: #94a3b8;
  }

  .tips-list li::before {
    color: #22d3ee;
  }

  .popular-image-item {
    background: #0f172a;
    border-color: #334155;
  }

  .popular-image-item:hover {
    background: #1e293b;
    border-color: #22d3ee;
  }

  .popular-image-item.selected {
    background: rgba(34, 197, 94, 0.2);
    border-color: #22c55e;
  }

  .popular-image-name {
    color: #f8fafc;
  }

  .popular-image-url {
    color: #94a3b8;
  }

  .empty-hint {
    color: #64748b;
  }

  /* Responsive */
  @media (max-width: 1440px) {
    .form-section {
      border-radius: 12px 12px 0 0;
    }

    .reference-panel {
      border-top-color: #334155;
    }
  }

  @media (max-width: 1024px) {
    .reference-panel {
      display: none;
    }
  }
}
</style>
