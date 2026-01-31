<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi } from '@/services/api'
import type { User, CreateUserRequest } from '@/types/api'

const loading = ref(false)
const users = ref<User[]>([])
const showCreateModal = ref(false)

const form = ref<CreateUserRequest>({
  username: '',
  password: '',
  role: 'viewer',
})

const loadUsers = async () => {
  try {
    loading.value = true
    users.value = await userApi.list()
  } catch (error) {
    ElMessage.error('加载用户列表失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  try {
    if (!form.value.username || !form.value.password) {
      ElMessage.warning('请填写用户名和密码')
      return
    }
    await userApi.create(form.value)
    ElMessage.success('用户创建成功')
    showCreateModal.value = false
    resetForm()
    loadUsers()
  } catch (error) {
    ElMessage.error('创建用户失败')
  }
}

const handleDelete = async (user: User) => {
  if (user.username === 'admin') {
    ElMessage.warning('不能删除 admin 用户')
    return
  }

  try {
    await ElMessageBox.confirm(`确定要删除用户 ${user.username} 吗？`, '确认删除', {
      type: 'warning',
    })
    await userApi.delete(user.id)
    ElMessage.success('用户已删除')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除用户失败')
    }
  }
}

const resetForm = () => {
  form.value = {
    username: '',
    password: '',
    role: 'viewer',
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <div class="admin-view" v-loading="loading">
    <div class="header">
      <h2>系统管理</h2>
      <el-button type="primary" @click="showCreateModal = true">新建用户</el-button>
    </div>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户列表</span>
        </div>
      </template>

      <el-table :data="users" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="role" label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">
              {{ row.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.createdAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row: user }">
            <el-button
              v-if="user.username !== 'admin'"
              size="small"
              type="danger"
              @click="handleDelete(user)"
            >
              删除
            </el-button>
            <span v-else style="color: #999; font-size: 13px">不可删除</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="showCreateModal"
      title="创建用户"
      width="500px"
      @close="resetForm"
    >
      <el-form label-width="100px">
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="角色" required>
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="普通用户" value="viewer" />
            <el-option label="管理员" value="admin" />
          </el-select>
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
.admin-view {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.card-header {
  font-weight: 600;
  color: #0f172a;
}
</style>
