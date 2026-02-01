<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { taskApi, libraryApi, secretApi } from '@/services/api'
import type { CreateTaskRequest, LibraryImage, Secret } from '@/types/api'

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
// Use a single text area string for images instead of array
const imageInputText = ref('')

const libraryImages = ref<LibraryImage[]>([])
const secrets = ref<Secret[]>([])
const libraryLoading = ref(false)

// Right panel state
const searchText = ref('')
const sortField = ref<'name' | 'createdAt'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')
const showOnlySelected = ref(false)

// Form validation
const formErrors = ref<Record<string, string>>({})
const formTouched = ref<Record<string, boolean>>({})

const validateForm = (): boolean => {
  const errors: Record<string, string> = {}
  
  // Parse images from text area
  const images = imageInputText.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)

  // Validate images
  if (images.length === 0) {
    errors.images = '请至少输入一个镜像'
  } else if (images.length > 50) {
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

// Computeds
const parsedImages = computed(() => {
   return imageInputText.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
})

const filteredLibraryImages = computed(() => {
  let filtered = libraryImages.value

  if (searchText.value) {
    const searchLower = searchText.value.toLowerCase()
    filtered = filtered.filter(img =>
      img.name.toLowerCase().includes(searchLower) ||
      img.image.toLowerCase().includes(searchLower)
    )
  }

  if (showOnlySelected.value) {
     filtered = filtered.filter(img => parsedImages.value.includes(img.image))
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

// Methods
const loadLibraryImages = async () => {
  try {
    libraryLoading.value = true
    const response = await libraryApi.list({ limit: 500 }) // Load more for the side panel
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

const toggleImage = (imageUrl: string) => {
  const currentImages = parsedImages.value
  const index = currentImages.indexOf(imageUrl)
  
  if (index > -1) {
    // Remove image
    const newImages = [...currentImages]
    newImages.splice(index, 1)
    imageInputText.value = newImages.join('\n')
  } else {
    // Add image
    if (imageInputText.value.trim().length > 0) {
      if (!imageInputText.value.endsWith('\n')) {
         imageInputText.value += '\n'
      }
      imageInputText.value += imageUrl
    } else {
      imageInputText.value = imageUrl
    }
  }
}

const isImageSelected = (imageUrl: string) => {
   return parsedImages.value.includes(imageUrl)
}

const submit = async () => {
  if (!validateForm()) {
    ElMessage.error('表单填写有误，请检查')
    return
  }

  try {
    form.value.images = parsedImages.value
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

onMounted(() => {
  loadLibraryImages()
  loadSecrets()
})
</script>

<template>
  <div class="max-w-[1600px] mx-auto p-4 lg:p-6 space-y-6">
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
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
          <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
             <el-icon class="text-cyan-500"><Monitor /></el-icon>
             镜像列表
             <span class="text-xs font-normal text-slate-500 dark:text-slate-400 ml-2">每行一个镜像地址，可从右侧选择</span>
          </div>

          <div class="space-y-2">
             <el-input
                v-model="imageInputText"
                type="textarea"
                :rows="12"
                placeholder="docker.io/library/nginx:latest
registry.cn-hangzhou.aliyuncs.com/library/redis:7"
                class="font-mono text-sm !w-full"
                :class="{'is-error': formErrors.images}"
             />
             <div v-if="formErrors.images" class="text-red-500 text-xs mt-1">{{ formErrors.images }}</div>
             <div class="flex justify-between text-xs text-slate-400 px-1">
                <span>已输入 {{ parsedImages.length }} 个镜像</span>
                <span class="cursor-pointer hover:text-cyan-600" @click="imageInputText = ''" v-if="imageInputText">清空</span>
             </div>
          </div>
        </div>

        <!-- 任务参数 (Compact) -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
           <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
             <el-icon class="text-cyan-500"><Setting /></el-icon>
             参数配置
           </div>
           
           <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">批次大小</label>
                <el-input-number v-model="form.batchSize" :min="1" :max="100" class="!w-full" size="default" controls-position="right" />
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">优先级 (1-10)</label>
                <el-input-number v-model="form.priority" :min="1" :max="10" class="!w-full" size="default" controls-position="right" />
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">最大重试</label>
                <el-input-number v-model="form.maxRetries" :min="0" :max="5" class="!w-full" size="default" controls-position="right" />
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">重试延迟 (秒)</label>
                <el-input-number v-model="form.retryDelay" :min="5" :max="300" class="!w-full" size="default" controls-position="right" />
              </div>
              <div class="space-y-1 lg:col-span-2">
                 <label class="text-xs text-slate-500 dark:text-slate-400">重试策略</label>
                 <el-radio-group v-model="form.retryStrategy" class="!flex">
                    <el-radio-button label="linear" value="linear">线性重试</el-radio-button>
                    <el-radio-button label="exponential" value="exponential">指数退避</el-radio-button>
                 </el-radio-group>
              </div>
           </div>
        </div>

        <!-- 私有仓库 -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
           <div class="flex items-center justify-between mb-4">
             <div class="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
                <el-icon class="text-cyan-500"><Lock /></el-icon>
                私有仓库认证
             </div>
             <el-switch v-model="enableRegistry" inline-prompt active-text="开启" inactive-text="关闭" />
           </div>

           <transition name="el-zoom-in-top">
            <div v-if="enableRegistry" class="pt-2 border-t border-slate-100 dark:border-slate-700/50 mt-2">
               <div class="flex gap-6 mb-4 mt-4">
                  <el-radio-group v-model="authMode" size="small">
                    <el-radio-button label="manual" value="manual">手动输入凭证</el-radio-button>
                    <el-radio-button label="select" value="select">选择已有认证</el-radio-button>
                  </el-radio-group>
               </div>

               <div v-if="authMode === 'manual'" class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <el-input v-model="form.registry" placeholder="仓库地址 (e.g. harbor.example.com)" />
                  <el-input v-model="form.username" placeholder="用户名" />
                  <el-input v-model="form.password" type="password" show-password placeholder="密码 / Token" />
               </div>

               <div v-if="authMode === 'select'">
                  <el-select v-model="form.secretId" placeholder="请选择认证信息" class="!w-full">
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
               </div>
            </div>
           </transition>
        </div>
      </div>

      <!-- Right: Image Library Selector -->
      <div class="hidden xl:flex flex-col w-[380px] bg-white dark:bg-slate-800 rounded-xl border border-slate-100 dark:border-slate-700/50 overflow-hidden sticky top-6 self-start shadow-sm h-[calc(100vh-120px)]">
         <div class="p-4 border-b border-slate-100 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-800/50">
            <div class="font-semibold text-slate-900 dark:text-slate-100 mb-3 flex justify-between items-center">
               <span>镜像库</span>
               <span class="text-xs font-normal text-slate-500 bg-slate-200 dark:bg-slate-700 px-2 py-0.5 rounded-full">{{ libraryImages.length }}</span>
            </div>
            <div class="flex gap-2 mb-2">
               <el-input 
                  v-model="searchText" 
                  size="small" 
                  placeholder="搜索镜像..." 
                  prefix-icon="Search" 
                  clearable
                  class="flex-1"
               />
               <el-tooltip content="仅显示已选">
                  <el-button size="small" :type="showOnlySelected ? 'primary' : ''" @click="showOnlySelected = !showOnlySelected" circle>
                     <el-icon><Filter /></el-icon>
                  </el-button>
               </el-tooltip>
            </div>
            <div class="flex justify-between items-center text-xs">
               <div class="flex gap-1">
                  <span 
                     class="cursor-pointer px-1.5 py-0.5 rounded hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
                     :class="sortField === 'name' ? 'text-cyan-600 font-medium' : 'text-slate-500'"
                     @click="sortField = 'name'"
                  >名称</span>
                  <span class="text-slate-300">|</span>
                  <span 
                     class="cursor-pointer px-1.5 py-0.5 rounded hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
                     :class="sortField === 'createdAt' ? 'text-cyan-600 font-medium' : 'text-slate-500'"
                     @click="sortField = 'createdAt'"
                  >时间</span>
               </div>
               <div 
                  class="cursor-pointer text-slate-500 hover:text-cyan-600 px-1 rounded transition-colors" 
                  @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
               >
                  {{ sortOrder === 'asc' ? '升序' : '降序' }}
               </div>
            </div>
         </div>

         <div class="flex-1 overflow-y-auto p-2 custom-scrollbar space-y-1">
             <div v-if="libraryLoading" class="py-10 text-center text-slate-400 text-sm">加载中...</div>
             <div v-else-if="filteredLibraryImages.length === 0" class="py-10 text-center text-slate-400 text-sm">
                无匹配镜像
             </div>
             
             <div 
               v-for="img in filteredLibraryImages" 
               :key="img.id"
               class="group p-3 rounded-lg border border-transparent hover:border-slate-200 dark:hover:border-slate-700 bg-transparent hover:bg-white dark:hover:bg-slate-700/50 cursor-pointer transition-all relative"
               :class="{'!bg-cyan-50 dark:!bg-cyan-900/20 !border-cyan-200 dark:!border-cyan-800': isImageSelected(img.image)}"
               @click="toggleImage(img.image)"
             >
                <div class="flex items-start gap-2.5">
                   <div 
                     class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 transition-colors"
                     :class="isImageSelected(img.image) ? 'bg-cyan-100 dark:bg-cyan-800 text-cyan-600 dark:text-cyan-300' : 'bg-slate-100 dark:bg-slate-800 text-slate-400 group-hover:text-slate-600'"
                   >
                      <el-icon v-if="isImageSelected(img.image)"><Check /></el-icon>
                      <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                      </svg>
                   </div>
                   <div class="flex-1 min-w-0">
                      <div class="text-sm font-medium text-slate-700 dark:text-slate-200 truncate pr-4">{{ img.name }}</div>
                      <div class="text-xs text-slate-500 font-mono truncate opacity-80" :title="img.image">{{ img.image }}</div>
                   </div>
                </div>
                
                <div v-if="isImageSelected(img.image)" class="absolute top-3 right-3 w-1.5 h-1.5 rounded-full bg-cyan-500"></div>
             </div>
         </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #cbd5e1;
  border-radius: 2px;
}
.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #475569;
}
</style>
