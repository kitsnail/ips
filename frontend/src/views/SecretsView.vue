<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { secretApi } from '@/services/api'
import type { Secret, CreateSecretRequest } from '@/types/api'

const loading = ref(false)
const secrets = ref<Secret[]>([])

// Pagination state
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})
const showAddModal = ref(false)
const selectedSecrets = ref<number[]>([])


const form = ref<CreateSecretRequest>({
  name: '',
  registry: '',
  username: '',
  password: '',
})

const loadSecrets = async () => {
  try {
    loading.value = true
    // Use page and page size parameters for consistency with other views 
    const response = await secretApi.list({ 
      page: pagination.value.page, 
      pageSize: pagination.value.pageSize 
    })
    secrets.value = response.secrets
    pagination.value.total = response.total || response.secrets.length
  } catch (error) {
    ElMessage.error('加载认证信息失败')
  } finally {
    loading.value = false
  }
}

const handleAdd = async () => {
  try {
    await secretApi.create(form.value)
    ElMessage.success('认证信息添加成功')
    showAddModal.value = false
    form.value = { name: '', registry: '', username: '', password: '' }
    loadSecrets()
  } catch (error) {
    ElMessage.error('添加认证失败')
  }
}

const handleDelete = async (secret: Secret) => {
  try {
    await ElMessageBox.confirm('确定要删除这个认证信息吗？', '确认删除', {
      type: 'warning',
    })
    await secretApi.delete(secret.id)
    ElMessage.success('认证信息已删除')
    loadSecrets()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedSecrets.value.length === 0) {
    ElMessage.warning('请先选择要删除的认证信息')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedSecrets.value.length} 个认证信息吗？`,
      '确认删除',
      { type: 'warning' }
    )
    for (const item of selectedSecrets.value) {
      // @ts-ignore
      await secretApi.delete(item.id)
    }
    ElMessage.success(`成功删除 ${selectedSecrets.value.length} 个认证信息`)
    selectedSecrets.value = []
    loadSecrets()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadSecrets()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1  // Reset to first page when changing page size
  loadSecrets()
}

const handleSelectAll = (checked: boolean) => {
  if (checked) {
    selectedSecrets.value = secrets.value.map((s) => s.id)
  } else {
    selectedSecrets.value = []
  }
}

onMounted(() => {
  loadSecrets()
})

onUnmounted(() => {
  // 清理工作（如果有的话）
})
</script>

<template>
  <div class="secrets" v-loading="loading">
    <div class="header">
      <h2>仓库认证管理</h2>
      <div class="actions">
        <el-button
          type="danger"
          :disabled="selectedSecrets.length === 0"
          @click="handleBatchDelete"
        >
          批量删除 ({{ selectedSecrets.length }})
        </el-button>

      </div>
    </div>
    <el-table
      :data="secrets"
      @selection-change="selectedSecrets = $event"
      style="width: 100%"
    >
      <el-table-column type="selection" width="55" @select-all="handleSelectAll" />
      <el-table-column prop="name" label="名称" width="200" />
      <el-table-column prop="registry" label="镜像仓库地址" min-width="250">
        <template #default="{ row }">
          <span style="font-family: monospace; color: #0891b2;">{{ row.registry }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="username" label="用户名" width="150" />
      <el-table-column prop="createdAt" label="创建时间" width="180">
        <template #default="{ row }">
          {{ new Date(row.createdAt).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
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

     <el-dialog v-model="showAddModal" title="添加仓库认证" width="500px">
      <el-form label-width="120px">
        <el-form-item label="认证名称" required>
          <el-input v-model="form.name" placeholder="例如：Harbor私有仓库" />
        </el-form-item>
        <el-form-item label="仓库地址" required>
          <el-input v-model="form.registry" placeholder="harbor.example.com" />
        </el-form-item>
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" placeholder="your-username" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="form.password" type="password" placeholder="your-password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddModal = false">取消</el-button>
        <el-button type="primary" @click="handleAdd">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.secrets {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.actions {
  display: flex;
  gap: 12px;
}
</style>
