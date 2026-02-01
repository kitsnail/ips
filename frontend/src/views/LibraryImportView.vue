<script setup lang="ts">
import { ref } from 'vue'

import { ElMessage } from 'element-plus'
import { libraryApi } from '@/services/api'



const importInput = ref('')
const loading = ref(false)
const successCount = ref(0)
const failCount = ref(0)
const importedImages = ref<string[]>([])
const parsingErrors = ref<Array<{ line: number; error: string }>>([])

const extractDisplayName = (image: string): string => {
  const trimmed = image.trim()
  if (!trimmed) return ''
  const parts = trimmed.split('/')
  const lastPart = parts[parts.length - 1]
  return lastPart || trimmed
}

const validateImageLine = (line: string): { valid: boolean; displayName: string; imageUrl: string } => {
  const trimmed = line.trim()
  if (!trimmed) {
    return { valid: false, displayName: '', imageUrl: '' }
  }

  // Check if it's a valid image URL format
  const imageRegex = /^[\w.-]+(:\S+)?$/
  if (!imageRegex.test(trimmed)) {
    return { valid: false, displayName: '', imageUrl: trimmed }
  }

  return {
    valid: true,
    displayName: extractDisplayName(trimmed),
    imageUrl: trimmed,
  }
}

const parseImport = () => {
  const lines = importInput.value.split('\n')
  const errors: Array<{ line: number; error: string }> = []
  const validImages: string[] = []

  lines.forEach((line, index) => {
    const result = validateImageLine(line)

    if (result.valid) {
      validImages.push(result.imageUrl)
    } else {
      errors.push({
        line: index + 1,
        error: result.imageUrl ? `格式错误: ${result.imageUrl}` : '空行',
      })
    }
  })

  parsingErrors.value = errors
  importedImages.value = validImages
}

const handleImport = async () => {
  if (importedImages.value.length === 0) {
    ElMessage.warning('请输入要导入的镜像地址')
    return
  }

  if (parsingErrors.value.length > 0) {
    ElMessage.warning(`发现 ${parsingErrors.value.length} 个格式错误，请检查后重试`)
    return
  }

  try {
    loading.value = true
    successCount.value = 0
    failCount.value = 0

    for (const image of importedImages.value) {
      try {
        await libraryApi.create({
          name: extractDisplayName(image),
          image: image,
        })
        successCount.value++
      } catch (error) {
        failCount.value++
        console.error(`Failed to import image: ${image}`, error)
      }
    }

    if (successCount.value > 0) {
      ElMessage.success(
        `成功导入 ${successCount.value} 个镜像${failCount.value > 0 ? `，失败 ${failCount.value} 个` : ''}`
      )
    } else {
      ElMessage.error('所有镜像导入失败')
    }

    importInput.value = ''
    importedImages.value = []
    parsingErrors.value = []
    successCount.value = 0
    failCount.value = 0
  } catch (error) {
    ElMessage.error('导入失败，请稍后重试')
    console.error('Import error:', error)
  } finally {
    loading.value = false
  }
}

const handleClear = () => {
  importInput.value = ''
  parsingErrors.value = []
  importedImages.value = []
}


</script>

<template>
  <div class="max-w-[1200px] mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <div class="flex justify-between items-center mb-8">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-white">批量导入镜像</h1>
      <div class="flex gap-3">

        <el-button @click="handleClear" :disabled="!importInput && parsingErrors.length === 0">
          清空
        </el-button>
        <el-button
          type="primary"
          @click="handleImport"
          :loading="loading"
          :disabled="importedImages.length === 0 || parsingErrors.length > 0"
        >
          导入
        </el-button>
      </div>
    </div>

    <div class="space-y-6">
      <!-- Import Input -->
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 sm:p-8 shadow-sm border border-slate-200 dark:border-slate-700">
        <div class="flex justify-between items-center mb-5 pb-4 border-b border-slate-100 dark:border-slate-700">
          <h3 class="text-lg font-semibold text-slate-900 dark:text-white">镜像列表</h3>
          <div class="text-sm text-slate-500 dark:text-slate-400">
            每行输入一个镜像地址
          </div>
        </div>

        <div class="mt-4">
          <el-input
            v-model="importInput"
            type="textarea"
            :rows="15"
            placeholder="例如：&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7&#10;192.168.1.100:5000/myapp:v1.0.0"
            @input="parseImport"
            class="font-mono"
            :input-style="{ fontSize: '14px', lineHeight: '1.6' }"
          />
        </div>

        <!-- Parsing Errors -->
        <div v-if="parsingErrors.length > 0" class="mt-5 p-4 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded-r-lg">
          <div class="flex items-center gap-2 text-sm font-semibold text-red-700 dark:text-red-300 mb-3">
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 8v4m0 6h.01M12 21v-4m0-6H12m0-6h6"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            格式错误 ({{ parsingErrors.length }})
          </div>
          <div class="flex flex-col gap-2">
            <div v-for="error in parsingErrors" :key="error.line" class="flex items-start gap-3 text-[13px]">
              <span class="text-red-600 dark:text-red-400 font-mono font-bold min-w-[60px]">行 {{ error.line }}:</span>
              <span class="text-red-800 dark:text-red-200">{{ error.error }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Import Preview -->
      <div v-if="importedImages.length > 0 && parsingErrors.length === 0" class="bg-white dark:bg-slate-800 rounded-xl p-6 sm:p-8 shadow-sm border border-slate-200 dark:border-slate-700">
        <div class="flex justify-between items-center mb-5">
          <h3 class="text-lg font-semibold text-slate-900 dark:text-white">待导入镜像</h3>
          <div class="px-3 py-1 bg-cyan-50 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400 rounded-full text-sm font-medium">
            {{ importedImages.length }} 个
          </div>
        </div>

        <div class="flex flex-col gap-2 max-h-[400px] overflow-y-auto">
          <div v-for="(image, index) in importedImages" :key="index" class="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-cyan-50 hover:border-cyan-500 dark:hover:bg-slate-800 dark:hover:border-cyan-500 transition-all duration-200 group">
            <div class="w-8 h-8 flex items-center justify-center bg-cyan-600 text-white rounded-full font-bold text-sm shrink-0">
              {{ index + 1 }}
            </div>
            <div class="flex-1 font-medium text-slate-900 dark:text-slate-200 truncate">
              {{ extractDisplayName(image) }}
            </div>
            <div class="flex-[2] font-mono text-xs text-slate-500 dark:text-slate-400 truncate">
              {{ image }}
            </div>
            <el-button
              type="danger"
              size="small"
              text
              @click="importedImages.splice(index, 1); parseImport()"
              class="opacity-0 group-hover:opacity-100 transition-opacity"
            >
              移除
            </el-button>
          </div>
        </div>
      </div>

      <!-- Instructions -->
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 sm:p-8 shadow-sm border border-slate-200 dark:border-slate-700">
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white mb-5">使用说明</h3>
        <ul class="space-y-3 m-0 p-0 list-none">
          <li class="relative pl-6 text-sm text-slate-600 dark:text-slate-400 leading-relaxed before:content-['•'] before:absolute before:left-0 before:text-cyan-600 before:font-bold before:text-lg">
            每行输入一个完整的镜像地址(包含仓库和tag)
          </li>
          <li class="relative pl-6 text-sm text-slate-600 dark:text-slate-400 leading-relaxed before:content-['•'] before:absolute before:left-0 before:text-cyan-600 before:font-bold before:text-lg">
            格式示例: <code class="bg-slate-100 dark:bg-slate-700 px-2 py-0.5 rounded text-slate-800 dark:text-slate-200 font-mono text-xs">docker.io/library/nginx:latest</code>
          </li>
          <li class="relative pl-6 text-sm text-slate-600 dark:text-slate-400 leading-relaxed before:content-['•'] before:absolute before:left-0 before:text-cyan-600 before:font-bold before:text-lg">
            支持私有仓库镜像和公共镜像
          </li>
          <li class="relative pl-6 text-sm text-slate-600 dark:text-slate-400 leading-relaxed before:content-['•'] before:absolute before:left-0 before:text-cyan-600 before:font-bold before:text-lg">
            导入时会自动从镜像地址提取显示名称
          </li>
          <li class="relative pl-6 text-sm text-slate-600 dark:text-slate-400 leading-relaxed before:content-['•'] before:absolute before:left-0 before:text-cyan-600 before:font-bold before:text-lg">
            重复的镜像名称会自动添加序号区分
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Minimal scoped styles if needed, mostly handled by Tailwind now */
</style>
