<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { scheduledTaskApi } from '@/services/api'
import type { ScheduledTask, CreateScheduledTaskRequest } from '@/types/api'

const loading = ref(false)
const scheduledTasks = ref<ScheduledTask[]>([])
const showCreateModal = ref(false)
let refreshInterval: number | null = null

// Use separate field for textarea input
const imagesInput = ref('')

const form = ref<CreateScheduledTaskRequest>({
  name: '',
  description: '',
  cronExpr: '',
  enabled: true,
  taskConfig: {
    images: [],
    batchSize: 10,
    priority: 5,
    maxRetries: 0,
    retryStrategy: 'linear',
    retryDelay: 30,
    webhookUrl: '',
  },
  overlapPolicy: 'skip',
  timeoutSeconds: 0,
})

const loadScheduledTasks = async () => {
  try {
    loading.value = true
    const response = await scheduledTaskApi.list({ limit: 50, offset: 0 })
    scheduledTasks.value = response.tasks
  } catch (error) {
    ElMessage.error('加载定时任务失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  try {
    // Convert textarea images (string) to array
    const imagesArray = imagesInput.value
      .split('\n')
      .map(img => img.trim())
      .filter(img => img.length > 0)

    if (imagesArray.length === 0) {
      ElMessage.warning('请至少输入一个镜像地址')
      return
    }

    const requestData = {
      ...form.value,
      taskConfig: {
        ...form.value.taskConfig,
        images: imagesArray
      }
    }
    await scheduledTaskApi.create(requestData)
    ElMessage.success('定时任务创建成功')
    showCreateModal.value = false
    imagesInput.value = ''
    resetForm()
    loadScheduledTasks()
  } catch (error) {
    ElMessage.error('创建定时任务失败')
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
      ElMessage.error('删除失败')
    }
  }
}

const resetForm = () => {
  form.value = {
    name: '',
    description: '',
    cronExpr: '',
    enabled: true,
    taskConfig: {
      images: [],
      batchSize: 10,
      priority: 5,
      maxRetries: 0,
      retryStrategy: 'linear',
      retryDelay: 30,
      webhookUrl: '',
    },
    overlapPolicy: 'skip',
    timeoutSeconds: 0,
  }
  imagesInput.value = ''
}

onMounted(() => {
  loadScheduledTasks()
  refreshInterval = window.setInterval(() => {
    loadScheduledTasks()
  }, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="scheduled-tasks" v-loading="loading">
    <div class="header">
      <h2>定时任务管理</h2>
      <el-button type="primary" @click="showCreateModal = true">新建定时任务</el-button>
    </div>
    <el-table :data="scheduledTasks" style="width: 100%">
      <el-table-column prop="name" label="任务名称" width="200" />
      <el-table-column prop="cronExpr" label="Cron表达式" width="150" />
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
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleToggleEnable(row)">
            {{ row.enabled ? '禁用' : '启用' }}
          </el-button>
          <el-button size="small" type="primary" @click="handleTrigger(row)">
            触发
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      :model-value="showCreateModal"
      @update:model-value="(val: boolean) => showCreateModal = val"
      title="创建定时任务"
      width="700px"
      @close="resetForm"
    >
      <el-form label-width="120px">
        <el-form-item label="任务名称" required>
          <el-input v-model="form.name" placeholder="例如：每日镜像预热" />
        </el-form-item>
        <el-form-item label="Cron表达式" required>
          <el-input v-model="form.cronExpr" placeholder="0 2 * * *" />
          <div style="font-size: 12px; color: #64748b; margin-top: 4px;">
            示例：每天凌晨2点（0 2 * * *）、每小时（0 * * * *）
          </div>
        </el-form-item>
        <el-form-item label="镜像列表" required>
          <el-input
            v-model="imagesInput"
            type="textarea"
            :rows="4"
            placeholder="每行一个镜像地址"
          />
        </el-form-item>
        <el-form-item label="批次大小">
          <el-input-number v-model="form.taskConfig.batchSize" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="启用任务">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="重叠策略">
          <el-select v-model="form.overlapPolicy">
            <el-option label="跳过（不执行）" value="skip" />
            <el-option label="允许（排队）" value="allow" />
          </el-select>
        </el-form-item>
        <el-form-item label="超时时间（秒）">
          <el-input-number v-model="form.timeoutSeconds" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateModal = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
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
</style>
