<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { taskApi } from '@/services/api'
import type { Task } from '@/types/api'
import TaskDetailModal from '@/components/TaskDetailModal.vue'

const tasks = ref<Task[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const showDetailModal = ref(false)
const selectedTask = ref<Task | null>(null)
let refreshInterval: number | null = null

const refreshTasks = async () => {
  try {
    loading.value = true
    const response = await taskApi.list({ limit: 10, offset: 0 })
    tasks.value = response.tasks
  } catch (error) {
    ElMessage.error('加载任务失败')
  } finally {
    loading.value = false
  }
}

const handleCreateSuccess = () => {
  refreshTasks()
}

const showTaskDetail = (task: Task) => {
  selectedTask.value = task
  showDetailModal.value = true
}

onMounted(() => {
  refreshTasks()
  refreshInterval = window.setInterval(() => {
    refreshTasks()
  }, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="tasks">
    <div class="header">
      <h2>任务管理</h2>
      <el-button type="primary" @click="showCreateModal = true">新建任务</el-button>
    </div>
    <el-table :data="tasks" v-loading="loading" style="width: 100%">
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
      <el-table-column label="操作" width="150">
        <template #default="{ row: task }">
          <el-button size="small" @click="showTaskDetail(task)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <CreateTaskModal
      v-model:visible="showCreateModal"
      @success="handleCreateSuccess"
    />

    <TaskDetailModal
      :visible="showDetailModal"
      :task="selectedTask"
      @update:visible="(val) => showDetailModal = val"
    />
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
</style>
