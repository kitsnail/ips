<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { scheduledTaskApi } from '@/services/api'
import type { ScheduledExecution, ScheduledTask } from '@/types/api'

const route = useRoute()
const router = useRouter()

// Get scheduled task ID from route params
const scheduledTaskId = computed(() => route.params.id as string)

const loading = ref(false)
const taskLoading = ref(false)
const executions = ref<ScheduledExecution[]>([])
const scheduledTask = ref<ScheduledTask | null>(null)

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

const loadScheduledTask = async () => {
  try {
    taskLoading.value = true
    const task = await scheduledTaskApi.get(scheduledTaskId.value)
    scheduledTask.value = task
  } catch (error) {
    ElMessage.error('加载定时任务失败')
    console.error(error)
  } finally {
    taskLoading.value = false
  }
}

const loadExecutions = async () => {
  try {
    loading.value = true
    const offset = (pagination.value.page - 1) * pagination.value.pageSize
    const response = await scheduledTaskApi.listExecutions({
      scheduledTaskId: scheduledTaskId.value,
      limit: pagination.value.pageSize,
      offset,
    })

    executions.value = response.executions
    pagination.value.total = response.total
  } catch (error) {
    ElMessage.error('加载执行历史失败')
    console.error('Load executions error:', error)
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

const viewTaskDetail = (taskId: string) => {
  window.open(`/web/tasks/${taskId}`, '_blank')
}

const goBack = () => {
  router.push('/scheduled')
}

onMounted(() => {
  loadScheduledTask()
  loadExecutions()
})
</script>

<template>
  <div class="scheduled-executions" v-loading="loading">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-left">
        <el-button @click="goBack" :icon="'ArrowLeft'" circle class="back-button" />
        <div class="header-title">
          <h2>定时任务执行历史</h2>
          <div v-if="scheduledTask" class="task-info">
            <span class="task-name">{{ scheduledTask.name }}</span>
            <el-tag :type="scheduledTask.enabled ? 'success' : 'info'" size="small">
              {{ scheduledTask.enabled ? '已启用' : '已禁用' }}
            </el-tag>
            <span class="task-cron">{{ scheduledTask.cronExpr }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Task Detail Card -->
    <div v-if="scheduledTask" class="task-detail-card">
      <div class="detail-item">
        <div class="detail-label">任务名称</div>
        <div class="detail-value">{{ scheduledTask.name }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">Cron 表达式</div>
        <div class="detail-value font-mono">{{ scheduledTask.cronExpr }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">状态</div>
        <div class="detail-value">
          <el-tag :type="scheduledTask.enabled ? 'success' : 'info'" size="small">
            {{ scheduledTask.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </div>
      </div>
      <div class="detail-item">
        <div class="detail-label">描述</div>
        <div class="detail-value">{{ scheduledTask.description || '-' }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">批次大小</div>
        <div class="detail-value">{{ scheduledTask.taskConfig.batchSize }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">优先级</div>
        <div class="detail-value">{{ scheduledTask.taskConfig.priority }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">镜像数量</div>
        <div class="detail-value">{{ scheduledTask.taskConfig.images.length }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">创建时间</div>
        <div class="detail-value">{{ new Date(scheduledTask.createdAt).toLocaleString() }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">最后执行时间</div>
        <div class="detail-value">{{ scheduledTask.lastExecutionAt ? new Date(scheduledTask.lastExecutionAt).toLocaleString() : '从未执行过' }}</div>
      </div>
      <div class="detail-item">
        <div class="detail-label">下次执行时间</div>
        <div class="detail-value">{{ scheduledTask.nextExecutionAt ? new Date(scheduledTask.nextExecutionAt).toLocaleString() : '无' }}</div>
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
      <div class="filter-item">
        <span class="filter-label">执行次数:</span>
        <span class="filter-value">{{ pagination.total }}</span>
      </div>
    </div>

    <!-- Executions Table -->
    <div class="executions-card">
      <el-table
        :data="executions"
        v-loading="loading"
        style="width: 100%"
      >
        <el-table-column prop="id" label="执行ID" width="80" />
        <el-table-column prop="taskId" label="任务ID" width="180">
          <template #default="{ row }">
            <span class="task-link">
              {{ row.taskId }}
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
        <el-table-column prop="triggeredAt" label="触发时间" width="180">
          <template #default="{ row }">
            <span class="text-slate-600 dark:text-slate-400 text-sm">{{ new Date(row.triggeredAt).toLocaleString() }}</span>
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
              @click="viewTaskDetail(row.taskId)"
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
    <div v-if="!loading && executions.length === 0" class="empty-state">
       <el-empty
         description="暂无执行历史记录"
         :image-size="120"
       />
    </div>
  </div>
</template>

<style scoped>
.scheduled-executions {
  padding: 0;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-button {
  flex-shrink: 0;
}

.header-title h2 {
  font-size: 20px;
  font-weight: 600;
  color: #0f172a;
  margin: 0 0 8px 0;
}

.task-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.task-name {
  font-size: 14px;
  font-weight: 500;
  color: #475569;
}

.task-cron {
  font-family: monospace;
  font-size: 13px;
  color: #64748b;
  background: #f1f5f9;
  padding: 2px 8px;
  border-radius: 4px;
}

/* Task Detail Card */
.task-detail-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 24px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.detail-value {
  font-size: 14px;
  color: #334155;
  font-weight: 400;
}

.detail-value.font-mono {
  font-family: monospace;
}

/* Filters Section */
.filters-section {
  display: flex;
  gap: 24px;
  align-items: center;
  margin-bottom: 16px;
  padding: 16px;
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  font-size: 13px;
  font-weight: 500;
  color: #475569;
}

.filter-select {
  width: 150px;
}

.filter-value {
  font-size: 13px;
  color: #334155;
  font-weight: 500;
}

/* Executions Card */
.executions-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
}

.task-link {
  color: #0891b2;
  font-family: monospace;
  font-size: 13px;
  cursor: pointer;
  text-decoration: underline;
  text-decoration-color: #0891b2;
  transition: all 0.2s;
}

.task-link:hover {
  color: #0e7490;
  text-decoration-color: #0e7490;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

/* Empty State */
.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 60px 0;
}

/* Dark Mode */
.dark .header-title h2 {
  color: #f1f5f9;
}

.dark .task-name {
  color: #cbd5e1;
}

.dark .task-cron {
  color: #94a3b8;
  background: #334155;
}

.dark .task-detail-card,
.dark .filters-section,
.dark .executions-card {
  background: #1e293b;
  border-color: #334155;
}

.dark .detail-label {
  color: #94a3b8;
}

.dark .detail-value {
  color: #e2e8f0;
}

.dark .filter-label {
  color: #cbd5e1;
}

.dark .filter-value {
  color: #e2e8f0;
}

.dark .task-link {
  color: #22d3ee;
}

.dark .task-link:hover {
  color: #67e8f9;
}

/* Responsive */
@media (max-width: 768px) {
  .task-detail-card {
    grid-template-columns: repeat(2, 1fr);
  }

  .filters-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-left {
    flex-wrap: wrap;
  }

  .task-info {
    flex-wrap: wrap;
  }
}
</style>
