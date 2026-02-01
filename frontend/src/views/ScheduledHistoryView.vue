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
  <div class="scheduled-history">
    <div class="page-header">
      <h2>定时任务执行历史</h2>
      <div class="header-actions">
        <router-link to="/scheduled" class="back-link">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M19 12H5m7 7-7 7m0-14l1.5 1.5M13 11l1.5 1.5"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          返回定时任务
        </router-link>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-section">
      <div class="filter-item">
        <span class="filter-label">状态:</span>
        <el-select v-model="statusFilter" class="filter-select" @change="handleStatusFilter">
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
    <div class="table-container">
      <el-table
        :data="executions"
        v-loading="loading"
        style="width: 100%"
      >
        <el-table-column prop="id" label="执行ID" width="100" />
        <el-table-column prop="taskId" label="任务ID" width="180">
          <template #default="{ row }">
            <span class="task-link" @click="viewTaskDetail(row.id)">
              {{ getTaskName(row.taskId) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="durationSeconds" label="耗时" width="120">
          <template #default="{ row }">
            <el-tooltip :content="`${row.durationSeconds}秒`">
              {{ formatDuration(row.durationSeconds) }}
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="startedAt" label="开始时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.startedAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="finishedAt" label="结束时间" width="180">
          <template #default="{ row }">
            {{ row.finishedAt ? new Date(row.finishedAt).toLocaleString() : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="errorMessage" label="错误信息" min-width="200">
          <template #default="{ row }">
            <span class="error-message">{{ row.errorMessage || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              @click="viewTaskDetail(row.id)"
            >
              查看任务
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-wrapper">
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
    <el-empty
      v-if="!loading && executions.length === 0"
      description="暂无执行历史记录"
      :image-size="120"
    />
  </div>
</template>

<style scoped>
.scheduled-history {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h2 {
  font-size: 24px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.back-link {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.back-link:hover {
  color: #0891b2;
}

.back-link svg {
  width: 18px;
  height: 18px;
}

.filters-section {
  margin-bottom: 16px;
  display: flex;
  gap: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
}

.filter-select {
  width: 200px;
}

.table-container {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.pagination-wrapper {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}

.task-link {
  color: #0891b2;
  cursor: pointer;
  font-family: monospace;
  font-weight: 500;
  transition: color 0.2s;
}

.task-link:hover {
  color: #0e7490;
  text-decoration: underline;
}

.error-message {
  color: #64748b;
  font-size: 13px;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h2 {
    color: #f8fafc;
  }

  .back-link {
    color: #94a3b8;
  }

  .back-link:hover {
    color: #22d3ee;
  }

  .filter-label {
    color: #94a3b8;
  }

  .table-container {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .task-link {
    color: #22d3ee;
  }

  .task-link:hover {
    color: #06b6d4;
  }

  .error-message {
    color: #94a3b8;
  }
}
</style>
