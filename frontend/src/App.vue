<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '@/services/api'

const username = ref('')
const password = ref('')
const isLoggedIn = computed(() => !!window.localStorage.getItem('ips_token'))

const login = async () => {
  try {
    await authApi.login({ username: username.value, password: password.value })
    ElMessage.success('登录成功')
    window.location.href = '/web/dashboard'
  } catch (error) {
    ElMessage.error('登录失败，请检查用户名和密码')
  }
}
</script>

<template>
  <router-view v-if="isLoggedIn" />
  <div v-else class="login-container">
    <h1>镜像预热控制台 (IPS)</h1>
    <div class="login-box">
       <el-form @submit.prevent="login" label-width="auto">
         <el-form-item label="用户名">
           <el-input v-model="username" placeholder="请输入用户名" />
         </el-form-item>
         <el-form-item label="密码">
           <el-input v-model="password" type="password" placeholder="请输入密码" />
         </el-form-item>
         <el-form-item>
           <el-button type="primary" native-type="submit" style="width: 100%">
             登录
           </el-button>
         </el-form-item>
       </el-form>
    </div>
  </div>
</template>

<style>
.login-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  z-index: 9999;
}

.login-container h1 {
  color: white;
  margin-bottom: 40px;
  font-size: 32px;
}

.login-box {
  background: white;
  padding: 40px;
  border-radius: 16px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  width: 400px;
}
</style>
