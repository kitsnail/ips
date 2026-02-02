<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { scheduledTaskApi, libraryApi, secretApi } from '@/services/api'
import type { CreateScheduledTaskRequest, LibraryImage, Secret } from '@/types/api'

const router = useRouter()

const loading = ref(false)

// Form state
const form = ref<CreateScheduledTaskRequest>({
  name: '',
  description: '',
  cronExpr: '',
  enabled: true,
  taskConfig: {
    images: [],
    batchSize: 10,
    priority:5,
    maxRetries: 0,
    retryStrategy: 'linear',
    retryDelay: 30,
    secretId: undefined,
    registry: '',
    username: '',
    password: '',
  },
  overlapPolicy: 'skip',
  timeoutSeconds: 0,
})

// Use separate field for textarea input
const imagesInput = ref('')

// Private registry state
const enableRegistry = ref(false)
const authMode = ref<'manual' | 'select'>('manual')

// Image library state
const libraryImages = ref<LibraryImage[]>([])
const secrets = ref<Secret[]>([])
const libraryLoading = ref(false)
const secretsLoading = ref(false)

// Right panel state
const searchText = ref('')
const sortField = ref<'name' | 'createdAt'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')
const showOnlySelected = ref(false)

// Form validation
const formErrors = ref<Record<string, string>>({})

// Cron helper
const showCronHelper = ref(false)
const cronPresets = [
  { label: '每小时', expr: '0 * * * *', desc: '每小时的第0分钟执行' },
  { label: '每天凌晨2点', expr: '0 2 * * *', desc: '每天凌晨2点执行' },
  { label: '每周一凌晨2点', expr: '0 2 * * 1', desc: '每周一凌晨2点执行' },
  { label: '每月1号凌晨2点', expr: '0 2 1 * *', desc: '每月1号凌晨2点执行' },
  { label: '工作日早上9点', expr: '0 9 * * 1-5', desc: '周一到周五早上9点执行' },
  { label: '每30分钟', expr: '*/30 * * * *', desc: '每30分钟执行一次' },
]

const parsedImages = computed(() => {
  return imagesInput.value
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

const validateForm = (): boolean => {
  const errors: Record<string, string> = {}

  // Validate name
  if (!form.value.name || form.value.name.trim() === '') {
    errors.name = '请输入任务名称'
  } else if (form.value.name.length > 100) {
    errors.name = '任务名称不能超过100个字符'
  } else {
    errors.name = ''
  }

  // Validate cronExpr
  if (!form.value.cronExpr || form.value.cronExpr.trim() === '') {
    errors.cronExpr = '请输入Cron表达式'
  } else if (!/^\S+\s+\S+\s+\S+\s+\S+\s+\S+$/.test(form.value.cronExpr.trim())) {
    errors.cronExpr = 'Cron表达式格式不正确'
  } else {
    errors.cronExpr = ''
  }

  // Validate images
  const images = parsedImages.value
  if (images.length === 0) {
    errors.images = '请至少输入一个镜像'
  } else if (images.length > 100) {
    errors.images = '镜像数量不能超过100个'
  } else {
    errors.images = ''
  }

  // Validate batchSize
  if (form.value.taskConfig.batchSize < 1 || form.value.taskConfig.batchSize > 100) {
    errors.batchSize = '批次大小必须在1-100之间'
  } else {
    errors.batchSize = ''
  }

  // Validate priority
  if (form.value.taskConfig.priority < 1 || form.value.taskConfig.priority > 10) {
    errors.priority = '优先级必须在1-10之间'
  } else {
    errors.priority = ''
  }

  // Validate maxRetries
  if (form.value.taskConfig.maxRetries < 0 || form.value.taskConfig.maxRetries > 5) {
    errors.maxRetries = '最大重试次数必须在0-5之间'
  } else {
    errors.maxRetries = ''
  }

  // Validate retryDelay
  if (!form.value.taskConfig.retryDelay || form.value.taskConfig.retryDelay < 5 || form.value.taskConfig.retryDelay > 300) {
    errors.retryDelay = '重试延迟必须在5-300秒之间'
  } else {
    errors.retryDelay = ''
  }

  // Validate retryStrategy
  if (!['linear', 'exponential'].includes(form.value.taskConfig.retryStrategy)) {
    errors.retryStrategy = '请选择有效的重试策略'
  } else {
    errors.retryStrategy = ''
  }

  // Validate timeoutSeconds
  if (form.value.timeoutSeconds < 0) {
    errors.timeoutSeconds = '超时时间不能为负数'
  } else {
    errors.timeoutSeconds = ''
  }

  // Validate overlapPolicy
  if (!['skip', 'allow', 'queue'].includes(form.value.overlapPolicy)) {
    errors.overlapPolicy = '请选择有效的重叠策略'
  } else {
    errors.overlapPolicy = ''
  }

  // Validate private registry
  if (enableRegistry.value) {
    if (authMode.value === 'manual') {
      if (!form.value.taskConfig.registry || form.value.taskConfig.registry.trim() === '') {
        errors.registry = '请输入仓库地址'
      } else {
        errors.registry = ''
      }

      if (!form.value.taskConfig.username || form.value.taskConfig.username.trim() === '') {
        errors.username = '请输入用户名'
      } else {
        errors.username = ''
      }

      if (!form.value.taskConfig.password || form.value.taskConfig.password.trim() === '') {
        errors.password = '请输入密码'
      } else if (form.value.taskConfig.password.length < 6) {
        errors.password = '密码至少6个字符'
      } else {
        errors.password = ''
      }
    } else if (authMode.value === 'select') {
      if (!form.value.taskConfig.secretId) {
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

const loadLibraryImages = async () => {
  try {
    libraryLoading.value = true
    const response = await libraryApi.list({ limit: 500 })
    libraryImages.value = response.images
  } catch (error) {
    ElMessage.error('加载镜像库失败')
  } finally {
    libraryLoading.value = false
  }
}

const loadSecrets = async () => {
  try {
    secretsLoading.value = true
    const response = await secretApi.list({ pageSize: 100 })
    secrets.value = response.secrets
  } catch (error) {
    ElMessage.error('加载认证信息失败')
  } finally {
    secretsLoading.value = false
  }
}

const toggleImage = (imageUrl: string) => {
  const currentImages = parsedImages.value
  const index = currentImages.indexOf(imageUrl)

  if (index > -1) {
    // Remove image
    const newImages = [...currentImages]
    newImages.splice(index, 1)
    imagesInput.value = newImages.join('\n')
  } else {
    // Add image
    if (imagesInput.value.trim().length > 0) {
      if (!imagesInput.value.endsWith('\n')) {
        imagesInput.value += '\n'
      }
      imagesInput.value += imageUrl
    } else {
      imagesInput.value = imageUrl
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
    form.value.taskConfig.images = parsedImages.value
    loading.value = true
    await scheduledTaskApi.create(form.value)
    ElMessage.success('定时任务创建成功')
    router.push('/scheduled')
  } catch (error) {
    ElMessage.error('创建定时任务失败')
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
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">创建定时任务</h1>
      <div class="flex gap-3">
        <el-button type="primary" @click="submit" :loading="loading">
          创建任务
        </el-button>
      </div>
    </div>

    <div class="flex flex-col xl:flex-row gap-6 items-start">
      <!-- Left: Form Section -->
      <div class="flex-1 w-full space-y-6">

        <!-- Basic Info -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
          <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
             <el-icon class="text-cyan-500"><Document /></el-icon>
             基本信息
          </div>

          <div class="space-y-4">
             <div class="space-y-1">
               <label class="text-xs text-slate-500 dark:text-slate-400 font-medium">任务名称 <span class="text-red-500">*</span></label>
               <el-input
                  v-model="form.name"
                  placeholder="例如：每日镜像预热"
                  :class="{'is-error': formErrors.name}"
                  class="!w-full"
               />
               <div v-if="formErrors.name" class="text-red-500 text-xs">{{ formErrors.name }}</div>
             </div>

             <div class="space-y-1">
               <label class="text-xs text-slate-500 dark:text-slate-400 font-medium">任务描述</label>
               <el-input
                  v-model="form.description"
                  type="textarea"
                  :rows="2"
                  placeholder="请输入任务描述"
                  class="!w-full"
               />
             </div>

             <div class="space-y-1">
               <label class="text-xs text-slate-500 dark:text-slate-400 font-medium">Cron 表达式 <span class="text-red-500">*</span></label>
               <div class="flex gap-2">
                 <el-input
                    v-model="form.cronExpr"
                    placeholder="0 2 * * *"
                    :class="{'is-error': formErrors.cronExpr}"
                    class="!flex-1"
                 />
                 <el-button
                   type="primary"
                   text
                   size="small"
                   @click="showCronHelper = !showCronHelper"
                 >
                   帮助
                 </el-button>
               </div>
               <div v-if="formErrors.cronExpr" class="text-red-500 text-xs">{{ formErrors.cronExpr }}</div>

               <div v-if="showCronHelper" class="mt-3 p-3 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
                 <div class="text-xs font-semibold text-slate-600 dark:text-slate-300 mb-2">Cron 表达式示例</div>
                 <div class="grid grid-cols-2 gap-2">
                   <div
                     v-for="preset in cronPresets"
                     :key="preset.expr"
                     class="p-2 bg-white dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded cursor-pointer hover:border-cyan-500 dark:hover:border-cyan-500 hover:bg-cyan-50 dark:hover:bg-cyan-900/20 transition-all"
                     @click="form.cronExpr = preset.expr"
                   >
                     <div class="text-xs font-semibold text-slate-700 dark:text-slate-200">{{ preset.label }}</div>
                     <div class="text-xs font-mono text-cyan-600 dark:text-cyan-400">{{ preset.expr }}</div>
                     <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">{{ preset.desc }}</div>
                   </div>
                 </div>
               </div>
             </div>

             <div class="flex items-center gap-2">
               <el-switch v-model="form.enabled" inline-prompt active-text="已启用" inactive-text="已禁用" />
               <span class="text-sm text-slate-600 dark:text-slate-400">启用任务</span>
             </div>
          </div>
        </div>

        <!-- Image Configuration -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
          <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
             <el-icon class="text-cyan-500"><Monitor /></el-icon>
             镜像列表
             <span class="text-xs font-normal text-slate-500 dark:text-slate-400 ml-2">每行一个镜像地址，可从右侧选择</span>
          </div>

          <div class="space-y-2">
             <el-input
                v-model="imagesInput"
                type="textarea"
                :rows="10"
                placeholder="docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7"
                class="font-mono text-sm !w-full"
                :class="{'is-error': formErrors.images}"
             />
             <div v-if="formErrors.images" class="text-red-500 text-xs">{{ formErrors.images }}</div>
             <div class="flex justify-between text-xs text-slate-400 px-1">
                <span>已输入 {{ parsedImages.length }} 个镜像</span>
                <span class="cursor-pointer hover:text-cyan-600" @click="imagesInput = ''" v-if="imagesInput">清空</span>
             </div>
          </div>
        </div>

        <!-- Task Parameters -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
           <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
             <el-icon class="text-cyan-500"><Setting /></el-icon>
             参数配置
           </div>

           <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">批次大小</label>
                <el-input-number v-model="form.taskConfig.batchSize" :min="1" :max="100" class="!w-full" size="default" controls-position="right" />
                <div v-if="formErrors.batchSize" class="text-red-500 text-xs">{{ formErrors.batchSize }}</div>
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">优先级 (1-10)</label>
                <el-input-number v-model="form.taskConfig.priority" :min="1" :max="10" class="!w-full" size="default" controls-position="right" />
                <div v-if="formErrors.priority" class="text-red-500 text-xs">{{ formErrors.priority }}</div>
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">最大重试</label>
                <el-input-number v-model="form.taskConfig.maxRetries" :min="0" :max="5" class="!w-full" size="default" controls-position="right" />
                <div v-if="formErrors.maxRetries" class="text-red-500 text-xs">{{ formErrors.maxRetries }}</div>
              </div>
              <div class="space-y-1">
                <label class="text-xs text-slate-500 dark:text-slate-400">重试延迟 (秒)</label>
                <el-input-number v-model="form.taskConfig.retryDelay" :min="5" :max="300" class="!w-full" size="default" controls-position="right" />
                <div v-if="formErrors.retryDelay" class="text-red-500 text-xs">{{ formErrors.retryDelay }}</div>
              </div>
           </div>

           <div class="mt-4 space-y-4">
             <div class="space-y-1">
               <label class="text-xs text-slate-500 dark:text-slate-400">重试策略</label>
               <el-radio-group v-model="form.taskConfig.retryStrategy" class="!flex">
                  <el-radio-button label="linear" value="linear">线性重试</el-radio-button>
                  <el-radio-button label="exponential" value="exponential">指数退避</el-radio-button>
               </el-radio-group>
               <div v-if="formErrors.retryStrategy" class="text-red-500 text-xs">{{ formErrors.retryStrategy }}</div>
             </div>

             <div class="grid grid-cols-2 gap-4">
               <div class="space-y-1">
                 <label class="text-xs text-slate-500 dark:text-slate-400">重叠策略</label>
                 <el-select v-model="form.overlapPolicy" class="!w-full">
                   <el-option label="跳过（不执行）" value="skip" />
                   <el-option label="允许（排队）" value="allow" />
                   <el-option label="队列执行" value="queue" />
                 </el-select>
                 <div v-if="formErrors.overlapPolicy" class="text-red-500 text-xs">{{ formErrors.overlapPolicy }}</div>
               </div>
               <div class="space-y-1">
                 <label class="text-xs text-slate-500 dark:text-slate-400">超时时间（秒）</label>
                 <el-input-number
                   v-model="form.timeoutSeconds"
                   :min="0"
                   class="!w-full"
                   controls-position="right"
                 />
                 <div v-if="formErrors.timeoutSeconds" class="text-red-500 text-xs">{{ formErrors.timeoutSeconds }}</div>
               </div>
             </div>
           </div>
         </div>

         <!-- Private Registry -->
        <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
           <div class="flex items-center justify-between mb-4">
             <div class="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
               <el-icon class="text-cyan-500"><Lock /></el-icon>
               私有仓库认证
             </div>
             <el-switch v-model="enableRegistry" inline-prompt active-text="开启" inactive-text="关闭" />
           </div>

           <transition name="el-zoom-in-top">
             <div v-if="enableRegistry" class="pt-4 border-t border-slate-100 dark:border-slate-700/50 mt-2">
                <div class="flex gap-4 mb-4">
                   <el-radio-group v-model="authMode" size="small">
                     <el-radio-button label="manual" value="manual">手动输入凭证</el-radio-button>
                     <el-radio-button label="select" value="select">选择已有认证</el-radio-button>
                   </el-radio-group>
                </div>

                <div v-if="authMode === 'manual'" class="grid grid-cols-1 md:grid-cols-3 gap-4">
                   <div class="space-y-1">
                     <label class="text-xs text-slate-500 dark:text-slate-400">仓库地址 <span class="text-red-500">*</span></label>
                     <el-input
                       v-model="form.taskConfig.registry"
                       placeholder="harbor.example.com"
                       :class="{'is-error': formErrors.registry}"
                     />
                     <div v-if="formErrors.registry" class="text-red-500 text-xs">{{ formErrors.registry }}</div>
                   </div>
                   <div class="space-y-1">
                     <label class="text-xs text-slate-500 dark:text-slate-400">用户名 <span class="text-red-500">*</span></label>
                     <el-input
                       v-model="form.taskConfig.username"
                       :class="{'is-error': formErrors.username}"
                     />
                     <div v-if="formErrors.username" class="text-red-500 text-xs">{{ formErrors.username }}</div>
                   </div>
                   <div class="space-y-1">
                     <label class="text-xs text-slate-500 dark:text-slate-400">密码 / Token <span class="text-red-500">*</span></label>
                     <el-input
                       v-model="form.taskConfig.password"
                       type="password"
                       show-password
                       placeholder="密码 / Token"
                       :class="{'is-error': formErrors.password}"
                     />
                     <div v-if="formErrors.password" class="text-red-500 text-xs">{{ formErrors.password }}</div>
                   </div>
                </div>

                <div v-if="authMode === 'select'" class="space-y-1">
                   <label class="text-xs text-slate-500 dark:text-slate-400">选择认证 <span class="text-red-500">*</span></label>
                   <el-select v-model="form.taskConfig.secretId" placeholder="请选择认证信息" class="!w-full">
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
                   <div v-if="formErrors.secretId" class="text-red-500 text-xs">{{ formErrors.secretId }}</div>
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
