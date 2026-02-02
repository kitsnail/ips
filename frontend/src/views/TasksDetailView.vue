<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { taskApi } from '@/services/api'
import type { Task } from '@/types/api'

const route = useRoute()
const router = useRouter()

const task = ref<Task | null>(null)
const loading = ref(false)
const refreshInterval = ref<number | null>(null)

const taskId = computed(() => route.params.id as string)

const loadTaskDetail = async () => {
  try {
    loading.value = true
    task.value = await taskApi.get(taskId.value)
  } catch (error) {
    console.error('Load task detail error:', error)
    ElMessage.error('加载任务详情失败')
  } finally {
    loading.value = false
  }
}

const formatTime = (time?: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'completed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'running':
      return 'warning'
    case 'cancelled':
      return 'info'
    default:
      return ''
  }
}

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    pending: '等待中',
    running: '运行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return statusMap[status] || status
}

const refreshData = () => {
  loadTaskDetail()
}

const startAutoRefresh = () => {
  // Auto refresh every 3 seconds when task is running
  if (task.value?.status === 'running' || task.value?.status === 'pending') {
    refreshInterval.value = window.setInterval(() => {
      loadTaskDetail()
    }, 3000)
  }
}

const stopAutoRefresh = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

onMounted(() => {
  loadTaskDetail()
})

onUnmounted(() => {
  stopAutoRefresh()
})

// Watch task status changes to manage auto-refresh
const watchTaskStatus = (newTask: Task | null) => {
  stopAutoRefresh()
  if (newTask && (newTask.status === 'running' || newTask.status === 'pending')) {
    startAutoRefresh()
  }
}

// Watch task status to start/stop auto-refresh
import { watch as vueWatch } from 'vue'
vueWatch(task, (newTask) => {
  watchTaskStatus(newTask)
})
</script>

<template>
  <div class="task-detail">
    <!-- Page Header -->
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-center gap-3">
        <el-button link @click="router.back()">
          <el-icon class="mr-1"><ArrowLeft /></el-icon>
          返回
        </el-button>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">任务详情</h1>
        <el-tag v-if="task" :type="getStatusTagType(task.status)" size="large">
          {{ getStatusText(task.status) }}
        </el-tag>
      </div>
      <el-button @click="refreshData" :loading="loading">
        <el-icon class="mr-1"><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <div v-if="loading && !task" class="flex justify-center items-center py-20">
      <el-icon class="is-loading text-4xl text-cyan-500"><Loading /></el-icon>
    </div>

    <div v-else-if="task" class="space-y-6">
      <!-- Task Basic Info -->
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
          <el-icon class="text-cyan-500"><Document /></el-icon>
          基本信息
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">任务ID</label>
            <div class="text-sm font-mono text-slate-700 dark:text-slate-300 mt-1 break-all">{{ task.taskId }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">优先级</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.priority }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">批次大小</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.batchSize }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">创建时间</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ formatTime(task.createdAt) }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">开始时间</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ formatTime(task.startedAt) }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">完成时间</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ formatTime(task.finishedAt) }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">最大重试次数</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.maxRetries }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">已重试次数</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.retryCount }}</div>
          </div>
          <div>
            <label class="text-xs text-slate-500 dark:text-slate-400">重试策略</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">
              {{ task.retryStrategy === 'linear' ? '线性重试' : '指数退避' }}
            </div>
          </div>
        </div>
      </div>

      <!-- Progress Card -->
      <div v-if="task.progress" class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
          <el-icon class="text-cyan-500"><DataLine /></el-icon>
          执行进度
        </div>
        <div class="space-y-4">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm text-slate-600 dark:text-slate-400">总体进度</span>
            <span class="text-2xl font-bold text-cyan-600">{{ task.progress.percentage.toFixed(1) }}%</span>
          </div>
          <el-progress :percentage="task.progress.percentage" :stroke-width="20" :show-text="false" />

          <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mt-6">
            <div class="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-4 text-center">
              <div class="text-2xl font-bold text-slate-700 dark:text-slate-300">{{ task.progress.totalNodes }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">总节点数</div>
            </div>
            <div class="bg-green-50 dark:bg-green-900/20 rounded-lg p-4 text-center">
              <div class="text-2xl font-bold text-green-600">{{ task.progress.completedNodes }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">已完成</div>
            </div>
            <div class="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 text-center">
              <div class="text-2xl font-bold text-red-600">{{ task.progress.failedNodes }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">失败</div>
            </div>
            <div class="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4 text-center">
              <div class="text-2xl font-bold text-blue-600">
                {{ task.progress.currentBatch }} / {{ task.progress.totalBatches }}
              </div>
              <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">批次进度</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Images List -->
      <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
          <el-icon class="text-cyan-500"><Picture /></el-icon>
          镜像列表
          <span class="text-xs font-normal text-slate-500 dark:text-slate-400 ml-2">共 {{ task.images.length }} 个镜像</span>
        </div>
        <div class="space-y-2">
          <div
            v-for="(image, index) in task.images"
            :key="index"
            class="flex items-center gap-2 p-3 bg-slate-50 dark:bg-slate-900/50 rounded-lg"
          >
            <el-icon class="text-slate-400"><CopyDocument /></el-icon>
            <code class="text-sm font-mono text-slate-700 dark:text-slate-300 flex-1">{{ image }}</code>
          </div>
        </div>
      </div>

      <!-- Node Selector -->
      <div v-if="task.nodeSelector && Object.keys(task.nodeSelector).length > 0" class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
          <el-icon class="text-cyan-500"><Filter /></el-icon>
          节点选择器
        </div>
        <div class="space-y-2">
          <div
            v-for="(value, key) in task.nodeSelector"
            :key="key"
            class="flex items-center gap-2"
          >
            <el-tag type="info" size="small">{{ key }}</el-tag>
            <span class="text-sm text-slate-700 dark:text-slate-300">{{ value }}</span>
          </div>
        </div>
      </div>

      <!-- Failed Nodes -->
      <div v-if="task.failedNodeDetails && task.failedNodeDetails.length > 0" class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-red-200 dark:border-red-800/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-red-600">
          <el-icon class="text-red-500"><CircleClose /></el-icon>
          失败节点详情
          <span class="text-xs font-normal text-red-400 ml-2">共 {{ task.failedNodeDetails.length }} 个失败</span>
        </div>
        <el-table :data="task.failedNodeDetails" stripe style="width: 100%" max-height="400">
          <el-table-column prop="nodeName" label="节点名称" width="200" />
          <el-table-column prop="image" label="镜像" min-width="250">
            <template #default="{ row }">
              <code class="text-xs">{{ row.image }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="失败原因" width="150" />
          <el-table-column prop="message" label="详细信息" min-width="250" show-overflow-tooltip />
          <el-table-column prop="timestamp" label="时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.timestamp) }}
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Error Message -->
      <div v-if="task.errorMessage" class="bg-red-50 dark:bg-red-900/20 rounded-xl p-6 border border-red-200 dark:border-red-800/50">
        <div class="flex items-center gap-2 mb-3 text-base font-semibold text-red-600">
          <el-icon><CircleClose /></el-icon>
          错误信息
        </div>
        <div class="text-sm text-red-700 dark:text-red-300 whitespace-pre-wrap">{{ task.errorMessage }}</div>
      </div>

      <!-- Additional Config -->
      <div v-if="task.webhookUrl || task.registry" class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-100 dark:border-slate-700/50">
        <div class="flex items-center gap-2 mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">
          <el-icon class="text-cyan-500"><Setting /></el-icon>
          其他配置
        </div>
        <div class="space-y-3">
          <div v-if="task.webhookUrl">
            <label class="text-xs text-slate-500 dark:text-slate-400">Webhook URL</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1 break-all">{{ task.webhookUrl }}</div>
          </div>
          <div v-if="task.registry">
            <label class="text-xs text-slate-500 dark:text-slate-400">私有仓库</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.registry }}</div>
          </div>
          <div v-if="task.username">
            <label class="text-xs text-slate-500 dark:text-slate-400">仓库用户名</label>
            <div class="text-sm text-slate-700 dark:text-slate-300 mt-1">{{ task.username }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex flex-col items-center justify-center py-20">
      <el-icon class="text-6xl text-slate-300 dark:text-slate-600 mb-4"><Warning /></el-icon>
      <div class="text-slate-500 dark:text-slate-400">未找到任务详情</div>
    </div>
  </div>
</template>

<style scoped>
.task-detail {
  max-width: 1400px;
  margin: 0 auto;
}
</style>
