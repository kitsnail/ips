<script setup lang="ts">
import { ElMessage } from 'element-plus'

// TODO: Implement secrets create logic
ElMessage.info('添加认证功能开发中...')
</script>

<template>
  <div class="secrets-create">
    <h2>添加仓库认证</h2>
    <el-empty description="功能开发中..." :image-size="120" />
  </div>
</template>

<style scoped>
.secrets-create {
  padding: 0;
}

.secrets-create h2 {
  font-size: 24px;
  font-weight: 600;
  color: #0f172a;
  margin-bottom: 24px;
}

@media (prefers-color-scheme: dark) {
  .secrets-create h2 {
    color: #f8fafc;
  }
}
</style>
