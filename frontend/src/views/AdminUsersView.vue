<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi } from '@/services/api'
import type { User, CreateUserRequest } from '@/types/api'

const users = ref<User[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const editingUser = ref<User | null>(null)

const form = ref<CreateUserRequest>({
  username: '',
  password: '',
  role: 'viewer',
})

const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
})

const loadUsers = async () => {
  try {
    loading.value = true
    const response = await userApi.list()
    users.value = response
    pagination.value.total = response.length
  } catch (error) {
    ElMessage.error('加载用户列表失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  try {
    loading.value = true
    await userApi.create(form.value)
    ElMessage.success('用户创建成功')
    showCreateDialog.value = false
    form.value = { username: '', password: '', role: 'viewer' }
    loadUsers()
  } catch (error) {
    ElMessage.error('创建用户失败')
  } finally {
    loading.value = false
  }
}

const handleEdit = async () => {
  if (!editingUser.value) return

  try {
    loading.value = true
    await userApi.update(editingUser.value.id, {
      role: form.value.role,
    })
    ElMessage.success('用户更新成功')
    showEditDialog.value = false
    editingUser.value = null
    loadUsers()
  } catch (error) {
    ElMessage.error('更新用户失败')
  } finally {
    loading.value = false
  }
}

const openEditDialog = (user: User) => {
  editingUser.value = user
  form.value = {
    username: user.username,
    password: '',
    role: user.role,
  }
  showEditDialog.value = true
}

const handleDelete = async (user: User) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 "${user.username}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    loading.value = true
    await userApi.delete(user.id)
    ElMessage.success('用户删除成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除用户失败')
    }
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadUsers()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadUsers()
}

const resetForm = () => {
  form.value = { username: '', password: '', role: 'viewer' }
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <div class="admin-users">
    <div class="page-header">
      <h1>用户管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M12 4v16m8-8H4m0 0l3-3m0 3v12h2m0 0l-3 3"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        创建用户
      </el-button>
    </div>

    <div class="table-container">
      <el-table :data="users" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
              {{ row.role === 'admin' ? '管理员' : '查看者' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.createdAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row: user }">
            <el-button size="small" @click="openEditDialog(user)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(user)">删除</el-button>
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

    <!-- Create User Dialog -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建用户"
      width="500px"
      @close="resetForm"
    >
      <el-form label-width="100px">
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" placeholder="输入用户名" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="form.password" type="password" placeholder="输入密码" show-password />
        </el-form-item>
        <el-form-item label="角色" required>
          <el-radio-group v-model="form.role">
            <el-radio value="viewer">查看者</el-radio>
            <el-radio value="admin">管理员</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="loading">创建</el-button>
      </template>
    </el-dialog>

    <!-- Edit User Dialog -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑用户"
      width="500px"
      @close="resetForm"
    >
      <el-form label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" disabled placeholder="用户名" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="form.password" type="password" placeholder="留空则不修改" show-password />
        </el-form-item>
        <el-form-item label="角色" required>
          <el-radio-group v-model="form.role">
            <el-radio value="viewer">查看者</el-radio>
            <el-radio value="admin">管理员</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleEdit" :loading="loading">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.admin-users {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  margin: 0;
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

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h1 {
    color: #f8fafc;
  }

  .table-container {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }
}
</style>
