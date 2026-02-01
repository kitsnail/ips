<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { scheduledTaskApi, taskApi } from '@/services/api'
import type { ScheduledExecution, Task } from '@/types/api'

const route = useRoute()

const loading = ref(false)
const executions = ref<ScheduledExecution[]>([])
const executionTasks = ref<Map<string, Task>>(new Map())

const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0
})

const statusFilter = ref<string>('all')
const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '跳过', value: 'skipped' },
  { label: '超时', value: 'timeout' },
]

const loadExecutions = async () => {
  const scheduledTaskId = route.query.scheduledTaskId as string
  if (!scheduledTaskId) {
    ElMessage.warning('缺少定时任务ID')
    return
  }

  try {
    loading.value = true
    const offset = (pagination.value.page - 1) * pagination.value.pageSize
    const response = await scheduledTaskApi.listExecutions({
      scheduledTaskId,
      limit: pagination.value.pageSize,
      offset,
    })

    executions.value = response.executions
    pagination.value.total = response.total

    // Load related tasks
    const taskIds = [...new Set(response.executions.map(e => e.taskId))]
    for (const taskId of taskIds) {
      if (!executionTasks.value.has(taskId)) {
        try {
          const task = await taskApi.get(taskId)
          executionTasks.value.set(taskId, task)
        } catch (error) {
          console.error(`Failed to load task ${taskId}:`, error)
        }
      }
    }
  } catch (error) {
    ElMessage.error('加载执行历史失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadExecutions()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadExecutions()
}

const handleStatusFilter = (status: string) => {
  statusFilter.value = status
  pagination.value.page = 1
  loadExecutions()
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'skipped':
      return 'info'
    case 'timeout':
      return 'warning'
    default:
      return 'info'
  }
}

const getStatusLabel = (status: string) => {
  const option = statusOptions.find(opt => opt.value === status)
  return option?.label || status
}

const formatDuration = (seconds: number): string => {
  if (!seconds || seconds === 0) return '0s'
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return minutes > 0 ? `${minutes}m ${remainingSeconds}s` : `${remainingSeconds}s`
}

const getTaskName = (taskId: string): string => {
  const task = executionTasks.value.get(taskId)
  return task?.taskId || taskId
}

const viewTaskDetail = (executionId: number) => {
  const execution = executions.value.find(e => e.id === executionId)
  if (execution) {
    window.open(`/web/tasks/${execution.taskId}`, '_blank')
  }
}

onMounted(() => {
  loadExecutions()
})
</script>

<template>
  <div class="max-w-[1200px] mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold text-slate-900 dark:text-white">定时任务执行历史</h2>
      <div class="flex gap-3">

      </div>
    </div>

    <!-- Filters -->
    <div class="mb-4 flex gap-4 items-center">
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-slate-600 dark:text-slate-400">状态:</span>
        <el-select v-model="statusFilter" class="w-[200px]" @change="handleStatusFilter">
          <el-option
            v-for="option in statusOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
      </div>
    </div>

    <!-- Executions Table -->
    <div class="bg-white dark:bg-slate-800 rounded-xl p-6 shadow-sm border border-slate-200 dark:border-slate-700">
      <el-table
        :data="executions"
        v-loading="loading"
        style="width: 100%"
        :header-cell-style="{ background: 'transparent', color: 'inherit' }"
      >
        <el-table-column prop="id" label="执行ID" width="100" />
        <el-table-column prop="taskId" label="任务ID" width="180">
          <template #default="{ row }">
            <span class="text-cyan-600 dark:text-cyan-400 hover:text-cyan-700 dark:hover:text-cyan-300 font-mono cursor-pointer underline decoration-cyan-500/30 hover:decoration-cyan-500 transition-all" @click="viewTaskDetail(row.id)">
              {{ getTaskName(row.taskId) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" effect="light" class="!border-0">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="durationSeconds" label="耗时" width="120">
          <template #default="{ row }">
            <el-tooltip :content="`${row.durationSeconds}秒`">
              <span class="font-mono text-slate-600 dark:text-slate-400">{{ formatDuration(row.durationSeconds) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="startedAt" label="开始时间" width="180">
          <template #default="{ row }">
            <span class="text-slate-600 dark:text-slate-400 text-sm">{{ new Date(row.startedAt).toLocaleString() }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="finishedAt" label="结束时间" width="180">
          <template #default="{ row }">
            <span class="text-slate-600 dark:text-slate-400 text-sm">{{ row.finishedAt ? new Date(row.finishedAt).toLocaleString() : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="errorMessage" label="错误信息" min-width="200">
          <template #default="{ row }">
            <span class="text-xs text-red-500 dark:text-red-400 font-mono" v-if="row.errorMessage">{{ row.errorMessage }}</span>
            <span class="text-slate-400" v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              size="small"
              @click="viewTaskDetail(row.id)"
            >
              查看任务
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="mt-6 flex justify-center">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="sizes, prev, pager, next, total"
          :background="true"
          @size-change="handlePageSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && executions.length === 0" class="mt-8 flex justify-center">
       <el-empty
         description="暂无执行历史记录"
         :image-size="120"
       />
    </div>
  </div>
</template>

<style scoped>
/* Scoped styles replaced by Tailwind */
</style>
