<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { taskApi } from '@/services/api'
import type { Task } from '@/types/api'

interface Props {
  visible: boolean
  task: Task | null
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'cancel'): void
}

const emit = defineEmits<Emits>()
const props = defineProps<Props>()

const loading = ref(false)

const handleCancel = async () => {
  if (!props.task) return

  try {
    loading.value = true
    await taskApi.delete(props.task.taskId)
    ElMessage.success('任务已取消')
    emit('cancel')
  } catch (error) {
    ElMessage.error('取消任务失败')
  } finally {
    loading.value = false
  }
}

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    pending: '等待中',
    running: '运行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return statusMap[status] || status
}

const getStatusType = (status: string) => {
  const typeMap: Record<string, 'success' | 'danger' | 'warning' | 'info'> = {
    pending: 'info',
    running: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
  }
  return typeMap[status] || 'info'
}

const formatTime = (timestamp?: string) => {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString()
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="任务详情"
    width="900px"
  >
    <div v-if="task" v-loading="loading">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="任务ID">
          <span style="font-family: monospace; color: #0891b2;">{{ task.taskId }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(task.status)">{{ getStatusText(task.status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="优先级">
          {{ task.priority }}
        </el-descriptions-item>
        <el-descriptions-item label="批次大小">
          {{ task.batchSize }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formatTime(task.createdAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="开始时间" v-if="task.startedAt">
          {{ formatTime(task.startedAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="完成时间" v-if="task.finishedAt">
          {{ formatTime(task.finishedAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="重试次数">
          {{ task.retryCount }} / {{ task.maxRetries }}
        </el-descriptions-item>
        <el-descriptions-item label="重试策略">
          {{ task.retryStrategy === 'linear' ? '线性' : '指数退避' }}
        </el-descriptions-item>
        <el-descriptions-item label="重试延迟" v-if="task.retryDelay">
          {{ task.retryDelay }} 秒
        </el-descriptions-item>
        <el-descriptions-item label="Webhook URL" v-if="task.webhookUrl">
          <span style="font-family: monospace;">{{ task.webhookUrl }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">进度</el-divider>
      <el-descriptions :column="3" border v-if="task.progress">
        <el-descriptions-item label="总节点数">
          {{ task.progress.totalNodes }}
        </el-descriptions-item>
        <el-descriptions-item label="已完成">
          {{ task.progress.completedNodes }}
        </el-descriptions-item>
        <el-descriptions-item label="失败">
          {{ task.progress.failedNodes }}
        </el-descriptions-item>
        <el-descriptions-item label="当前批次">
          {{ task.progress.currentBatch }} / {{ task.progress.totalBatches }}
        </el-descriptions-item>
        <el-descriptions-item label="完成率">
          {{ task.progress.percentage.toFixed(1) }}%
        </el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">镜像列表</el-divider>
      <el-scrollbar max-height="200px">
        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px;">
          <el-tag
            v-for="(img, index) in task.images"
            :key="index"
            style="font-family: monospace; font-size: 13px;"
          >
            {{ img }}
          </el-tag>
        </div>
      </el-scrollbar>

      <el-divider content-position="left" v-if="task.secretName">私有仓库认证</el-divider>
      <el-alert
        v-if="task.secretName"
        type="info"
        :closable="false"
        style="margin-bottom: 24px;"
      >
        <template #title>
          <span style="font-weight: 600;">已启用私有仓库认证</span>
        </template>
        <div style="margin-top: 8px;">
          <div style="display: flex; align-items: center; gap: 8px;">
            <span>Secret 名称：</span>
            <span style="font-family: monospace; color: #0891b2; font-weight: 500;">{{ task.secretName }}</span>
          </div>
          <div style="font-size: 12px; color: #64748b; margin-top: 4px;">
            临时 Secret 会在任务完成后自动清理
          </div>
        </div>
      </el-alert>

      <el-divider content-position="left" v-if="task.errorMessage">错误信息</el-divider>
      <el-alert
        v-if="task.errorMessage"
        type="error"
        :closable="false"
      >
        {{ task.errorMessage }}
      </el-alert>

      <el-divider content-position="left" v-if="task.failedNodeDetails && task.failedNodeDetails.length > 0">失败节点详情</el-divider>
      <el-table
        v-if="task.failedNodeDetails && task.failedNodeDetails.length > 0"
        :data="task.failedNodeDetails"
        style="width: 100%; margin-top: 8px;"
        max-height="200px"
      >
        <el-table-column prop="nodeName" label="节点名称" width="200" />
        <el-table-column prop="image" label="镜像" min-width="250">
          <template #default="{ row }">
            <span style="font-family: monospace;">{{ row.image }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" width="120" />
        <el-table-column prop="message" label="消息">
          <template #default="{ row }">
            {{ row.message || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="timestamp" label="时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.timestamp).toLocaleString() }}
          </template>
        </el-table-column>
      </el-table>
    </div>

    <template #footer>
      <el-button @click="emit('update:visible', false)">关闭</el-button>
      <el-button
        v-if="task && (task.status === 'pending' || task.status === 'running')"
        type="danger"
        :loading="loading"
        @click="handleCancel"
      >
        取消任务
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.el-descriptions {
  margin-bottom: 24px;
}

.el-divider {
  margin: 24px 0;
}
</style>
