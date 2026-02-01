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


</script>

<template>
  <div class="max-w-[1600px] mx-auto p-6 space-y-8">
    <!-- Page Header -->
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">创建镜像预热任务</h1>
      <div class="flex gap-3">

        <el-button type="primary" @click="submit" :loading="loading">
          创建任务
        </el-button>
      </div>
    </div>

    <div class="flex flex-col xl:flex-row gap-6 items-start">
      <!-- Left: Form Section -->
      <div class="flex-1 w-full space-y-6">
        <!-- 镜像配置 -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-8 shadow-sm border border-slate-100 dark:border-slate-700/50">
          <div class="flex items-center justify-between text-base font-semibold text-slate-900 dark:text-slate-100 mb-6 pb-3 border-b border-slate-200 dark:border-slate-700 gap-3">
            <div class="flex items-center gap-3">
              <svg class="w-5 h-5 text-cyan-500" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              镜像配置
            </div>
          </div>

          <div class="min-h-[120px]">
            <div v-if="selectedImages.length === 0" class="flex flex-col items-center justify-center p-10 border-2 border-dashed border-slate-200 dark:border-slate-700 rounded-lg text-slate-400 dark:text-slate-500 mb-6">
              <svg class="w-12 h-12 mb-4 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <p class="text-sm">暂无已选镜像</p>
            </div>
            
            <div v-else class="flex flex-wrap gap-3 mb-6">
              <div v-for="image in selectedImages" :key="image" class="flex items-center gap-2 px-3 py-2 bg-slate-100 dark:bg-slate-700/50 border border-slate-200 dark:border-slate-700 rounded-lg max-w-md group transition-colors">
                <span class="font-mono text-xs text-slate-700 dark:text-slate-300 truncate">{{ image }}</span>
                <button class="text-slate-400 hover:text-red-500 transition-colors p-1 rounded-md hover:bg-slate-200 dark:hover:bg-slate-600" @click="removeImage(image)">
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                    <path d="M6 18L18 6M6 6l12 12" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
              </div>
            </div>

            <div v-if="formErrors.images" class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 text-red-600 dark:text-red-400 text-sm rounded-r">
              {{ formErrors.images }}
            </div>

            <div class="flex gap-3">
              <el-button @click="showLibrarySelector = true">
                <el-icon class="mr-2"><Grid /></el-icon>
                从镜像库选择
              </el-button>
              <el-button @click="showManualInput = true">
                 <el-icon class="mr-2"><Edit /></el-icon>
                手动输入
              </el-button>
            </div>
          </div>
        </div>

        <!-- 任务参数 -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-8 shadow-sm border border-slate-100 dark:border-slate-700/50">
          <div class="flex items-center justify-between text-base font-semibold text-slate-900 dark:text-slate-100 mb-6 pb-3 border-b border-slate-200 dark:border-slate-700 gap-3">
             <div class="flex items-center gap-3">
                <svg class="w-5 h-5 text-cyan-500" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path d="M12 6V4m0 2a2 2 0 100 4 0 2 2 0 000-4zm0 16v-2m0 2a2 2 0 100 4 0 2 2 0 000-4zm8-8h-2m2 0a2 2 0 100 4 0 2 2 0 000-4zM6 12H4m2 0a2 2 0 100 4 0 2 2 0 000-4z" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                任务参数
             </div>
          </div>
          
          <el-form label-width="100px" label-position="top">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
              <el-form-item label="批次大小" :error="formErrors.batchSize">
                <el-input-number v-model="form.batchSize" :min="1" :max="100" class="!w-full" @focus="markFieldTouched('batchSize')" />
              </el-form-item>
              <el-form-item label="优先级" :error="formErrors.priority">
                <el-input-number v-model="form.priority" :min="1" :max="10" class="!w-full" @focus="markFieldTouched('priority')" />
              </el-form-item>
              <el-form-item label="最大重试" :error="formErrors.maxRetries">
                <el-input-number v-model="form.maxRetries" :min="0" :max="5" class="!w-full" @focus="markFieldTouched('maxRetries')" />
              </el-form-item>
            </div>
            
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
               <el-form-item label="重试策略" :error="formErrors.retryStrategy">
                <el-select v-model="form.retryStrategy" class="!w-full" @focus="markFieldTouched('retryStrategy')">
                  <el-option label="线性" value="linear" />
                  <el-option label="指数退避" value="exponential" />
                </el-select>
              </el-form-item>
              <el-form-item label="重试延迟(秒)" :error="formErrors.retryDelay">
                <el-input-number v-model="form.retryDelay" :min="5" :max="300" class="!w-full" @focus="markFieldTouched('retryDelay')" />
              </el-form-item>
            </div>
          </el-form>
        </div>

        <!-- 私有仓库 -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-8 shadow-sm border border-slate-100 dark:border-slate-700/50">
           <div class="flex items-center justify-between text-base font-semibold text-slate-900 dark:text-slate-100 mb-6 pb-3 border-b border-slate-200 dark:border-slate-700 gap-3">
             <div class="flex items-center gap-3">
                <svg class="w-5 h-5 text-cyan-500" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                私有仓库
             </div>
             <el-switch v-model="enableRegistry" />
           </div>

           <transition name="el-zoom-in-top">
            <div v-if="enableRegistry" class="space-y-6">
              <el-form label-width="100px" label-position="top">
                <el-form-item label="认证方式">
                  <el-radio-group v-model="authMode">
                    <el-radio value="manual">手动输入</el-radio>
                    <el-radio value="select">选择已保存</el-radio>
                  </el-radio-group>
                </el-form-item>

                <div v-if="authMode === 'manual'" class="space-y-6">
                  <el-form-item label="仓库地址" required :error="formErrors.registry">
                    <el-input v-model="form.registry" placeholder="harbor.example.com" @focus="markFieldTouched('registry')" />
                  </el-form-item>
                  <el-form-item label="用户名" required :error="formErrors.username">
                    <el-input v-model="form.username" @focus="markFieldTouched('username')" />
                  </el-form-item>
                  <el-form-item label="密码" required :error="formErrors.password">
                    <el-input v-model="form.password" type="password" show-password @focus="markFieldTouched('password')" />
                  </el-form-item>
                </div>

                <div v-if="authMode === 'select'">
                  <el-form-item label="选择认证" required :error="formErrors.secretId">
                    <el-select v-model="form.secretId" placeholder="请选择认证" class="!w-full" @focus="markFieldTouched('secretId')">
                      <el-option
                        v-for="secret in secrets"
                        :key="secret.id"
                        :label="secret.name"
                        :value="secret.id"
                      >
                        <div class="flex justify-between items-center w-full">
                           <span>{{ secret.name }}</span>
                           <span class="text-xs text-slate-400">{{ secret.registry }}</span>
                        </div>
                      </el-option>
                    </el-select>
                  </el-form-item>
                </div>
              </el-form>
            </div>
           </transition>
        </div>
      </div>

      <!-- Right: Reference Panel -->
      <div class="hidden xl:block w-96 bg-white dark:bg-slate-800 rounded-xl border border-slate-100 dark:border-slate-700/50 overflow-hidden sticky top-6 self-start shadow-sm">
        <div class="flex border-b border-slate-200 dark:border-slate-700">
           <button class="flex-1 py-4 text-sm font-medium transition-colors border-b-2"
                   :class="activeTab === 'stats' ? 'text-cyan-600 border-cyan-500 bg-cyan-50/50 dark:bg-cyan-900/10' : 'text-slate-500 dark:text-slate-400 border-transparent hover:text-slate-700 dark:hover:text-slate-200'"
                   @click="activeTab = 'stats'">
             运行统计
           </button>
           <button class="flex-1 py-4 text-sm font-medium transition-colors border-b-2"
                   :class="activeTab === 'images' ? 'text-cyan-600 border-cyan-500 bg-cyan-50/50 dark:bg-cyan-900/10' : 'text-slate-500 dark:text-slate-400 border-transparent hover:text-slate-700 dark:hover:text-slate-200'"
                   @click="activeTab = 'images'">
             常用镜像
           </button>
        </div>

        <div class="p-6 h-[calc(100vh-250px)] overflow-y-auto custom-scrollbar">
           <!-- Stats Tab -->
           <div v-if="activeTab === 'stats'" class="space-y-8">
              <div class="bg-gradient-to-br from-cyan-500 to-blue-600 rounded-xl p-5 text-white shadow-lg shadow-cyan-500/20">
                 <div class="flex items-center gap-2 mb-2 opacity-90">
                    <el-icon><VideoPlay /></el-icon>
                    <span class="text-sm font-medium">运行中任务</span>
                 </div>
                 <div class="text-3xl font-bold">{{ runningTasks.length }}</div>
                 <div class="mt-4 text-xs opacity-80 flex justify-between items-center">
                    <span>实时监控</span>
                    <button class="hover:underline" @click="router.push('/tasks')">查看全部 ></button>
                 </div>
              </div>

              <div v-if="recentFailedTasks.length > 0">
                 <h4 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">最近失败任务</h4>
                 <div class="space-y-2">
                    <div v-for="task in recentFailedTasks" :key="task.taskId" 
                         class="p-3 bg-red-50 dark:bg-red-900/20 border-l-2 border-red-500 rounded-r-lg cursor-pointer hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors group"
                         @click="router.push(`/tasks/${task.taskId}`)">
                       <div class="flex justify-between items-start">
                          <span class="font-mono text-xs text-slate-700 dark:text-slate-300 group-hover:text-red-700 dark:group-hover:text-red-200">{{ task.taskId }}</span>
                          <el-icon class="text-red-400"><ArrowRight /></el-icon>
                       </div>
                       <div class="text-xs text-red-500 mt-1 truncate">{{ task.errorMessage || 'Unknown Error' }}</div>
                    </div>
                 </div>
              </div>

               <div>
                 <h4 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">提示</h4>
                 <ul class="space-y-2 text-sm text-slate-600 dark:text-slate-400 list-disc pl-4 marker:text-cyan-500">
                    <li>批次大小建议10-20</li>
                    <li>优先级越高执行越早</li>
                    <li>私有仓库需配置认证</li>
                 </ul>
              </div>
           </div>

           <!-- Images Tab -->
           <div v-if="activeTab === 'images'" class="space-y-6">
              <div>
                 <h4 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">推荐镜像</h4>
                 <div v-if="popularImages.length > 0" class="space-y-2">
                    <div v-for="image in popularImages" :key="image"
                         class="p-3 rounded-lg border border-slate-100 dark:border-slate-700 hover:border-cyan-500 dark:hover:border-cyan-500 cursor-pointer transition-all group relative bg-slate-50 dark:bg-slate-900"
                         :class="{'ring-2 ring-cyan-500 ring-offset-2 dark:ring-offset-slate-800': selectedImages.includes(image)}"
                         @click="addPopularImage(image)">
                       <div class="font-medium text-slate-700 dark:text-slate-200 text-sm mb-1">{{ getShortImageName(image) }}</div>
                       <div class="text-xs text-slate-500 font-mono truncate">{{ image }}</div>
                       <div v-if="selectedImages.includes(image)" class="absolute top-2 right-2 text-cyan-500">
                          <el-icon><Check /></el-icon>
                       </div>
                    </div>
                 </div>
                 <div v-else class="text-center py-8 text-slate-400 text-sm">暂无推荐数据</div>
              </div>
           </div>
        </div>
      </div>
    </div>

    <!-- Library Selector Dialog -->
    <el-dialog
      v-model="showLibrarySelector"
      title="从镜像库选择"
      width="900px"
      class="rounded-xl overflow-hidden"
      @open="loadLibraryImages"
      append-to-body
    >
      <div class="flex gap-4 mb-4">
        <el-input
          v-model="searchText"
          placeholder="搜索镜像..."
          :prefix-icon="'Search'"
          clearable
          class="flex-1"
        />
        <el-button-group>
           <el-button :type="sortField === 'name' ? 'primary' : ''" @click="sortField = 'name'">名称</el-button>
           <el-button :type="sortField === 'createdAt' ? 'primary' : ''" @click="sortField = 'createdAt'">时间</el-button>
        </el-button-group>
        <el-button @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'">
           {{ sortOrder === 'asc' ? '升序' : '降序' }}
        </el-button>
      </div>
      
      <div v-if="libraryLoading" class="py-12 text-center text-slate-400">加载中...</div>
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 max-h-[400px] overflow-y-auto p-1 custom-scrollbar">
         <div v-for="img in filteredLibraryImages" :key="img.id"
              class="p-3 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer hover:border-cyan-500 transition-all relative"
              :class="selectedImages.includes(img.image) ? 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-500' : 'bg-white dark:bg-slate-800'"
              @click="addImage(img.image)">
             <div class="font-medium text-slate-700 dark:text-slate-200 text-sm truncate">{{ img.name }}</div>
             <div class="text-xs text-slate-500 font-mono truncate mt-1">{{ img.image }}</div>
             <div v-if="selectedImages.includes(img.image)" class="absolute top-2 right-2 text-cyan-600">
                 <el-icon><Check /></el-icon>
             </div>
         </div>
      </div>
      
      <template #footer>
         <div class="flex justify-between items-center">
            <span class="text-slate-500 text-sm">已选择 {{ selectedImages.length }} 个镜像</span>
            <div class="flex gap-2">
               <el-button @click="showLibrarySelector = false">关闭</el-button>
               <el-button type="primary" @click="showLibrarySelector = false">确认</el-button>
            </div>
         </div>
      </template>
    </el-dialog>

    <!-- Manual Input Dialog -->
    <el-dialog v-model="showManualInput" title="手动输入镜像" width="600px" append-to-body class="rounded-xl">
      <el-input
        v-model="manualImageInput"
        type="textarea"
        :rows="8"
        placeholder="每行输入一个镜像地址，例如：&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7"
        class="font-mono text-sm"
      />
      <template #footer>
        <el-button @click="showManualInput = false">取消</el-button>
        <el-button type="primary" @click="addManualImages">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* Custom Scrollbar for inner panels */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #cbd5e1;
  border-radius: 3px;
}
.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #475569;
}
</style>
