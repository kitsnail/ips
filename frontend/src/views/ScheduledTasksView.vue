<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { scheduledTaskApi } from '@/services/api'
import type { ScheduledTask } from '@/types/api'

const router = useRouter()

const loading = ref(false)
const scheduledTasks = ref<ScheduledTask[]>([])

// Pagination state
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})

const selectedTasks = ref<ScheduledTask[]>([])

const loadScheduledTasks = async () => {
  try {
    loading.value = true
    const offset = (pagination.value.page - 1) * pagination.value.pageSize
    const response = await scheduledTaskApi.list({
      limit: pagination.value.pageSize,
      offset: offset
    })
    scheduledTasks.value = response.tasks
    pagination.value.total = response.total || response.tasks.length
  } catch (error) {
    ElMessage.error('加载定时任务失败')
  } finally {
    loading.value = false
  }
}

const handleToggleEnable = async (task: ScheduledTask) => {
  try {
    if (task.enabled) {
      await scheduledTaskApi.disable(task.id)
      ElMessage.success('定时任务已禁用')
    } else {
      await scheduledTaskApi.enable(task.id)
      ElMessage.success('定时任务已启用')
    }
    loadScheduledTasks()
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

const handleTrigger = async (task: ScheduledTask) => {
  try {
    await scheduledTaskApi.trigger(task.id)
    ElMessage.success('定时任务已触发')
  } catch (error) {
    ElMessage.error('触发失败')
  }
}

const handleDelete = async (task: ScheduledTask) => {
  try {
    await ElMessageBox.confirm('确定要删除这个定时任务吗？', '确认删除', {
      type: 'warning',
    })
    await scheduledTaskApi.delete(task.id)
    ElMessage.success('定时任务已删除')
    loadScheduledTasks()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete scheduled task error:', error)
      ElMessage.error(`删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
}

const handleEdit = (task: ScheduledTask) => {
  // Navigate to edit page (can create a separate edit view or reuse create view)
  router.push(`/scheduled/${task.id}/edit`)
}

const handleViewHistory = (task: ScheduledTask) => {
  router.push(`/scheduled/${task.id}/executions`)
}

const handleBulkDelete = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('请选择要删除的定时任务')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTasks.value.length} 个定时任务吗？`,
      '确认批量删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const deletePromises = selectedTasks.value.map(task => scheduledTaskApi.delete(task.id))
    await Promise.all(deletePromises)

    ElMessage.success(`成功删除了 ${deletePromises.length} 个定时任务`)
    selectedTasks.value = []
    loadScheduledTasks()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Bulk delete scheduled tasks error:', error)
      ElMessage.error(`批量删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadScheduledTasks()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadScheduledTasks()
}

onMounted(() => {
  loadScheduledTasks()
})

onUnmounted(() => {
  // 清理工作（如果有的话）
})
</script>

<template>
  <div class="scheduled-tasks" v-loading="loading">
    <!-- Page Header -->
    <div class="header">
      <h2>定时任务管理</h2>
      <div class="header-right">
        <el-button
          v-if="selectedTasks.length > 0"
          type="danger"
          @click="handleBulkDelete"
          :disabled="selectedTasks.length === 0"
        >
          批量删除 ({{ selectedTasks.length }})
        </el-button>
      </div>
    </div>

    <!-- Table -->
    <el-table
      :data="scheduledTasks"
      style="width: 100%"
      @selection-change="selectedTasks = $event"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="name" label="任务名称" width="200" />
      <el-table-column prop="cronExpr" label="Cron表达式" width="150">
        <template #default="{ row }">
          <span class="font-mono text-sm">{{ row.cronExpr }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="taskConfig.images" label="镜像" min-width="200">
        <template #default="{ row }">
          {{ row.taskConfig.images[0] }}
          <span v-if="row.taskConfig.images.length > 1">+{{ row.taskConfig.images.length - 1 }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">
            {{ row.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="180">
        <template #default="{ row }">
          {{ new Date(row.createdAt).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button size="small" @click="handleToggleEnable(row)">
            {{ row.enabled ? '禁用' : '启用' }}
          </el-button>
          <el-button size="small" type="primary" @click="handleTrigger(row)">
            触发
          </el-button>
          <el-button size="small" @click="handleViewHistory(row)">
            历史
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :page-sizes="[10, 20, 50]"
        :total="pagination.total"
        layout="sizes, prev, pager, next, total"
        :background="true"
        @size-change="handlePageSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<style scoped>
.scheduled-tasks {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header h2 {
  font-size: 20px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.header-right {
  display: flex;
  gap: 12px;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

/* Dark Mode */
.dark .header h2 {
  color: #f1f5f9;
}
</style>
