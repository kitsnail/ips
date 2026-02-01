<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi } from '@/services/api'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)

const handleLogin = async () => {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  
  loading.value = true
  try {
    const { token } = await authApi.login({ username: username.value, password: password.value })
    localStorage.setItem('ips_token', token)
    // Optional: store user info if returned
    localStorage.setItem('ips_user', JSON.stringify({ username: username.value }))
    
    ElMessage.success('登录成功')
    router.replace('/dashboard')
  } catch (error) {
    ElMessage.error('登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-500 to-purple-600">
    <div class="bg-white dark:bg-slate-800 rounded-2xl shadow-2xl w-full max-w-md p-8 transform transition-all duration-300 hover:scale-[1.01]">
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-gray-800 dark:text-white mb-2">IPS Console</h1>
        <p class="text-gray-500 dark:text-gray-400">镜像预热控制台</p>
      </div>

      <el-form @submit.prevent="handleLogin" label-position="top" size="large">
        <el-form-item label="用户名" class="mb-6">
          <el-input 
            v-model="username" 
            placeholder="请输入用户名" 
            :prefix-icon="'User'"
            class="!h-12"
          />
        </el-form-item>
        
        <el-form-item label="密码" class="mb-8">
          <el-input 
            v-model="password" 
            type="password" 
            placeholder="请输入密码" 
            :prefix-icon="'Lock'" 
            show-password
            class="!h-12"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-button 
          type="primary" 
          :loading="loading" 
          class="w-full !h-12 !text-lg !rounded-lg !font-semibold !bg-gradient-to-r from-blue-600 to-indigo-600 hover:!from-blue-700 hover:!to-indigo-700 !border-0 shadow-lg hover:shadow-xl transition-all duration-300"
          @click="handleLogin"
        >
          登录
        </el-button>
      </el-form>
      
      <div class="mt-8 text-center text-sm text-gray-400">
        &copy; 2026 IPS System. All rights reserved.
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.el-form-item__label) {
  @apply text-gray-700 dark:text-gray-300 font-medium;
}
:deep(.el-input__wrapper) {
  @apply rounded-lg shadow-sm border-gray-200 dark:bg-slate-700 dark:border-slate-600 box-border;
}
</style>
