<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { taskApi } from '@/services/api'
import type { Task } from '@/types/api'

const tasks = ref<Task[]>([])
const loading = ref(false)
const selectedTasks = ref<Task[]>([])

// Pagination state
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})

const refreshTasks = async () => {
  try {
    loading.value = true
    const offset = (pagination.value.page - 1) * pagination.value.pageSize
    const response = await taskApi.list({ 
      limit: pagination.value.pageSize, 
      offset: offset 
    })
    tasks.value = response.tasks
    pagination.value.total = response.total || response.tasks.length
  } catch (error) {
    ElMessage.error('加载任务失败')
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  refreshTasks()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1  // Reset to first page when changing page size
  refreshTasks()
}

const handleDelete = async (task: Task) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这个任务吗？',
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    await taskApi.delete(task.taskId)
    ElMessage.success('任务已删除')
    refreshTasks()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete task error:', error)
      ElMessage.error(`删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
}

const handleBulkDelete = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('请选择要删除的任务')
    return
  }
  
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTasks.value.length} 个任务吗？`,
      '确认批量删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    const deletePromises = selectedTasks.value.map(task => taskApi.delete(task.taskId))
    await Promise.all(deletePromises)
    
    ElMessage.success(`成功删除了 ${deletePromises.length} 个任务`)
    selectedTasks.value = []
    refreshTasks()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Bulk delete error:', error)
      ElMessage.error(`批量删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
}

onMounted(() => {
  refreshTasks()
})

onUnmounted(() => {
  // 清理工作（如果有的话）
})
</script>

<template>
   <div class="tasks">
      <div class="header">
        <h2>任务管理</h2>
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
     <el-table 
        :data="tasks" 
        v-loading="loading" 
        style="width: 100%"
        @selection-change="selectedTasks = $event"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="taskId" label="任务ID" width="180" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'completed' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="images" label="镜像">
          <template #default="{ row }">
            {{ row.images[0] }}
            <span v-if="row.images.length > 1">+{{ row.images.length - 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="progress.percentage" label="进度" width="120">
          <template #default="{ row }">
            {{ row.progress?.percentage?.toFixed(1) || 0 }}%
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.createdAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row: task }">
            <router-link :to="`/tasks/${task.taskId}`">
              <el-button size="small">详情</el-button>
            </router-link>
            <el-button size="small" type="danger" @click="handleDelete(task)">删除</el-button>
          </template>
        </el-table-column>
       </el-table>
   
       <!-- Pagination -->
       <div style="display: flex; justify-content: center; margin-top: 20px;">
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
.tasks {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-right {
  display: flex;
  gap: 12px;
}
</style>
