<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { scheduledTaskApi } from '@/services/api'
import type { ScheduledTask, CreateScheduledTaskRequest } from '@/types/api'

const loading = ref(false)
const scheduledTasks = ref<ScheduledTask[]>([])

// Pagination state
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})
const showCreateModal = ref(false)
const showEditModal = ref(false)
const selectedTasks = ref<ScheduledTask[]>([])
const editingTask = ref<ScheduledTask | null>(null)


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
      console.error('Delete scheduled task error:', error)
      ElMessage.error(`删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
}

const handleEdit = async (task: ScheduledTask) => {
  editingTask.value = { ...task };
  // Populate form for editing
  form.value = {
    name: task.name,
    description: task.description || '',
    cronExpr: task.cronExpr,
    enabled: task.enabled,
    taskConfig: { ...task.taskConfig },
    overlapPolicy: task.overlapPolicy,
    timeoutSeconds: task.timeoutSeconds,
  };
  // Set images input
  imagesInput.value = task.taskConfig.images.join('\n');
  showEditModal.value = true;
};

const handleUpdate = async () => {
  try {
    if (!editingTask.value) return;

    // Convert textarea images (string) to array
    const imagesArray = imagesInput.value
      .split('\n')
      .map(img => img.trim())
      .filter(img => img.length > 0);

    if (imagesArray.length === 0) {
      ElMessage.warning('请至少输入一个镜像地址');
      return;
    }

    const requestData = {
      ...form.value,
      taskConfig: {
        ...form.value.taskConfig,
        images: imagesArray
      }
    };

    await scheduledTaskApi.update(editingTask.value.id, requestData);
    ElMessage.success('定时任务更新成功');
    showEditModal.value = false;
    editingTask.value = null;
    resetForm();
    loadScheduledTasks();
  } catch (error) {
    ElMessage.error('更新定时任务失败');
  }
};

const handleBulkDelete = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('请选择要删除的定时任务');
    return;
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
    );

    const deletePromises = selectedTasks.value.map(task => scheduledTaskApi.delete(task.id));
    await Promise.all(deletePromises);
    
    ElMessage.success(`成功删除了 ${deletePromises.length} 个定时任务`);
    selectedTasks.value = [];
    loadScheduledTasks();
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Bulk delete scheduled tasks error:', error)
      ElMessage.error(`批量删除失败: ${error instanceof Error ? error.message : '未知错误'}`)
    }
  }
};

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadScheduledTasks()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1  // Reset to first page when changing page size
  loadScheduledTasks()
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
  editingTask.value = null;
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
         <el-button type="primary" @click="showCreateModal = true">新建定时任务</el-button>
       </div>
     </div>
     <el-table 
       :data="scheduledTasks" 
       style="width: 100%"
       @selection-change="selectedTasks = $event"
     >
       <el-table-column type="selection" width="55" />
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
           <el-button size="small" type="danger" @click="handleDelete(row)">
             删除
           </el-button>
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

      <!-- Create Dialog -->
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

     <!-- Edit Dialog -->
     <el-dialog
       :model-value="showEditModal"
       @update:model-value="(val: boolean) => showEditModal = val"
       title="编辑定时任务"
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
         <el-button @click="showEditModal = false">取消</el-button>
         <el-button type="primary" @click="handleUpdate">更新</el-button>
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

.header-right {
  display: flex;
  gap: 12px;
}
</style>
