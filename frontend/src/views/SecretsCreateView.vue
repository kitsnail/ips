<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { secretApi } from '@/services/api'
import type { CreateSecretRequest } from '@/types/api'

const router = useRouter()
const loading = ref(false)

const form = ref<CreateSecretRequest>({
  name: '',
  registry: '',
  username: '',
  password: '',
})

const formErrors = ref<Record<string, string>>({})
const formTouched = ref<Record<string, boolean>>({})

const validateForm = (): boolean => {
  const errors: Record<string, string> = {}
  
  if (!form.value.name.trim()) {
    errors.name = '请输入认证名称'
  }
  
  if (!form.value.registry.trim()) {
    errors.registry = '请输入仓库地址'
  }
  
  if (!form.value.username.trim()) {
    errors.username = '请输入用户名'
  }
  
  if (!form.value.password) {
    errors.password = '请输入密码'
  }

  formErrors.value = errors
  return Object.keys(errors).length === 0
}

const markFieldTouched = (field: string) => {
  formTouched.value[field] = true
}

const submit = async () => {
  markFieldTouched('name')
  markFieldTouched('registry')
  markFieldTouched('username')
  markFieldTouched('password')

  if (!validateForm()) {
    return
  }

  try {
    loading.value = true
    await secretApi.create(form.value)
    ElMessage.success('仓库认证添加成功')
    router.push('/secrets')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '添加认证失败')
  } finally {
    loading.value = false
  }
}


</script>

<template>
  <div class="max-w-[1000px] mx-auto p-6 space-y-8">
    <!-- Page Header -->
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">添加仓库认证</h1>
      <div class="flex gap-3">
        <el-button type="primary" @click="submit" :loading="loading">
          保存
        </el-button>
      </div>
    </div>

    <div class="bg-white dark:bg-slate-800 rounded-xl p-8 shadow-sm border border-slate-100 dark:border-slate-700/50 max-w-3xl">
      <div class="flex items-center gap-3 mb-8 pb-4 border-b border-slate-100 dark:border-slate-700">
        <div class="w-10 h-10 rounded-lg bg-cyan-50 dark:bg-cyan-900/20 flex items-center justify-center text-cyan-600 dark:text-cyan-400">
          <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        </div>
        <div>
          <h3 class="text-lg font-semibold text-slate-900 dark:text-white">认证详情</h3>
          <p class="text-sm text-slate-500 dark:text-slate-400">配置私有镜像仓库的访问凭证</p>
        </div>
      </div>

      <el-form label-position="top" class="max-w-xl">
        <el-form-item label="认证名称" required :error="formErrors.name">
          <el-input 
            v-model="form.name" 
            placeholder="例如：Harbor Production" 
            @blur="markFieldTouched('name')"
            class="!w-full"
          />
          <div class="text-xs text-slate-400 mt-1">用于识别该认证信息的唯一名称</div>
        </el-form-item>

        <el-form-item label="仓库地址" required :error="formErrors.registry">
          <el-input 
            v-model="form.registry" 
            placeholder="例如：harbor.example.com" 
            @blur="markFieldTouched('registry')"
          >
            <template #prefix>
              <el-icon class="text-slate-400"><Link /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="用户名" required :error="formErrors.username">
          <el-input 
            v-model="form.username" 
            placeholder="仓库访问用户名" 
            @blur="markFieldTouched('username')"
          >
            <template #prefix>
              <el-icon class="text-slate-400"><User /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="密码 / Token" required :error="formErrors.password">
          <el-input 
            v-model="form.password" 
            type="password" 
            show-password 
            placeholder="请输入密码或访问令牌" 
            @blur="markFieldTouched('password')"
          />
        </el-form-item>
      </el-form>
      
      <div class="mt-8 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-slate-100 dark:border-slate-700 text-sm text-slate-600 dark:text-slate-400">
        <div class="flex items-center gap-2 font-medium mb-2 text-slate-800 dark:text-slate-200">
          <el-icon><InfoFilled /></el-icon>
          <span>安全提示</span>
        </div>
        <ul class="list-disc pl-5 space-y-1">
          <li>您的密码将以加密方式存储。</li>
          <li>建议使用具有最小权限的 Robot 账号或访问令牌。</li>
          <li>该认证信息将用于拉取私有镜像。</li>
        </ul>
      </div>
    </div>
  </div>
</template>
