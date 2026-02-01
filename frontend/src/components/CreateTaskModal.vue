<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { taskApi, libraryApi, secretApi } from '@/services/api'
import type { CreateTaskRequest, LibraryImage, Secret } from '@/types/api'

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const props = defineProps<{
  visible: boolean
}>()

const form = ref<CreateTaskRequest>({
  images: [],
  batchSize: 10,
  priority: 5,
  maxRetries: 0,
  retryStrategy: 'linear',
  retryDelay: 30,
  nodeSelector: {},
})

const enableRegistry = ref(false)
const authMode = ref<'manual' | 'select'>('manual')
const manualImageInput = ref('')

const libraryImages = ref<LibraryImage[]>([])
const secrets = ref<Secret[]>([])
const loading = ref(false)
const libraryLoading = ref(false)
const secretsLoading = ref(false)

const searchText = ref('')
const sortField = ref<'name' | 'createdAt'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')

const loadLibraryImages = async () => {
  try {
    libraryLoading.value = true
    const response = await libraryApi.list({ limit: 100 })
    libraryImages.value = response.images
  } catch (error) {
    ElMessage.error('加载镜像库失败')
  } finally {
    libraryLoading.value = false
  }
}

const loadSecrets = async () => {
  try {
    secretsLoading.value = true
    const response = await secretApi.list({ pageSize: 100 })
    secrets.value = response.secrets
  } catch (error) {
    ElMessage.error('加载认证信息失败')
  } finally {
    secretsLoading.value = false
  }
}

const handleOpen = () => {
  loadLibraryImages()
  loadSecrets()
  manualImageInput.value = ''
  searchText.value = ''
}

const handleClose = () => {
  emit('update:visible', false)
}

const isImageAdded = (imageUrl: string): boolean => {
  const manualImages = manualImageInput.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
  return manualImages.includes(imageUrl)
}

const filteredImages = computed(() => {
  let filtered = libraryImages.value

  if (searchText.value) {
    const searchLower = searchText.value.toLowerCase()
    filtered = filtered.filter(img =>
      img.name.toLowerCase().includes(searchLower) ||
      img.image.toLowerCase().includes(searchLower)
    )
  }

  if (sortField.value === 'name') {
    filtered = [...filtered].sort((a, b) => {
      const comparison = a.name.localeCompare(b.name)
      return sortOrder.value === 'asc' ? comparison : -comparison
    })
  } else {
    filtered = [...filtered].sort((a, b) => {
      const comparison = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      return sortOrder.value === 'asc' ? comparison : -comparison
    })
  }

  return filtered
})

const addToManualInput = (imageUrl: string) => {
  if (!isImageAdded(imageUrl)) {
    const currentImages = manualImageInput.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)
    currentImages.push(imageUrl)
    manualImageInput.value = currentImages.join('\n')
  }
}

const removeFromManualInput = (imageUrl: string) => {
  const lines = manualImageInput.value.split('\n')
  const newLines = lines.filter(line => line.trim() !== imageUrl)
  manualImageInput.value = newLines.join('\n')
}

const submit = async () => {
  try {
    const selectedImages = manualImageInput.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    if (selectedImages.length === 0) {
      ElMessage.warning('请至少添加一个镜像')
      return
    }

    form.value.images = selectedImages

    loading.value = true
    await taskApi.create(form.value)
    ElMessage.success('任务创建成功')
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '创建任务失败')
  } finally {
    loading.value = false
  }
}

defineExpose({
  handleOpen,
})
</script>

<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="创建镜像预热任务"
    width="1200px"
    @open="handleOpen"
  >
    <el-form label-width="120px">
      <el-form-item label="镜像列表" required>
        <div class="image-section">
          <div class="image-input-section">
            <div class="section-title">手动输入</div>
            <el-input
              v-model="manualImageInput"
              type="textarea"
              :rows="12"
              placeholder="每行输入一个镜像地址，例如：&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7"
              class="manual-input"
            />
          </div>

          <div class="image-library-section">
            <div class="section-header">
              <div class="section-title">从镜像库选择</div>
              <div class="controls">
                <el-input
                  v-model="searchText"
                  placeholder="搜索镜像..."
                  :prefix-icon="'Search'"
                  class="search-input"
                  clearable
                />
                <el-select v-model="sortField" class="sort-select">
                  <el-option label="按名称" value="name" />
                  <el-option label="按添加时间" value="createdAt" />
                </el-select>
                <el-button
                  :icon="sortOrder === 'asc' ? 'ArrowUp' : 'ArrowDown'"
                  @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
                  size="small"
                />
              </div>
            </div>
            <div v-if="libraryLoading" class="loading-text">加载中...</div>
            <el-empty v-else-if="filteredImages.length === 0" description="暂无镜像" :image-size="60" />
            <div v-else class="table-container">
              <table class="image-table">
                <thead>
                  <tr>
                    <th style="width: 60%">显示名称</th>
                    <th style="width: 30%">镜像地址</th>
                    <th style="width: 10%">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="img in filteredImages" :key="img.id" class="table-row">
                    <td>{{ img.name }}</td>
                    <td class="url-cell">{{ img.image }}</td>
                    <td>
                      <el-button
                        v-if="isImageAdded(img.image)"
                        type="danger"
                        size="small"
                        @click="removeFromManualInput(img.image)"
                      >
                        已添加
                      </el-button>
                      <el-button
                        v-else
                        type="primary"
                        size="small"
                        @click="addToManualInput(img.image)"
                      >
                        添加
                      </el-button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </el-form-item>

      <el-form-item label="批次大小">
        <el-input-number v-model="form.batchSize" :min="1" :max="100" />
      </el-form-item>

      <el-form-item label="优先级">
        <el-input-number v-model="form.priority" :min="1" :max="10" />
      </el-form-item>

      <el-form-item label="最大重试">
        <el-input-number v-model="form.maxRetries" :min="0" :max="5" />
      </el-form-item>

      <el-form-item label="重试策略">
        <el-select v-model="form.retryStrategy">
          <el-option label="线性" value="linear" />
          <el-option label="指数退避" value="exponential" />
        </el-select>
      </el-form-item>

      <el-form-item label="私有仓库">
        <el-switch v-model="enableRegistry" />
      </el-form-item>

      <template v-if="enableRegistry">
        <el-form-item label="认证方式">
          <el-radio-group v-model="authMode">
            <el-radio value="manual">手动输入</el-radio>
            <el-radio value="select">选择已保存</el-radio>
          </el-radio-group>
        </el-form-item>

        <template v-if="authMode === 'manual'">
          <el-form-item label="仓库地址" required>
            <el-input v-model="form.registry" placeholder="harbor.example.com" />
          </el-form-item>
          <el-form-item label="用户名" required>
            <el-input v-model="form.username" />
          </el-form-item>
          <el-form-item label="密码" required>
            <el-input v-model="form.password" type="password" show-password />
          </el-form-item>
        </template>

        <template v-if="authMode === 'select'">
          <el-form-item label="选择认证" required>
            <el-select v-model="form.secretId" placeholder="请选择认证" style="width: 100%">
              <el-option
                v-for="secret in secrets"
                :key="secret.id"
                :label="secret.name"
                :value="secret.id"
              >
                <span style="float: left">{{ secret.name }}</span>
                <span style="float: right; color: #8492a6; font-size: 12px">
                  {{ secret.registry }}
                </span>
              </el-option>
            </el-select>
          </el-form-item>
        </template>
      </template>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="submit" :loading="loading">
        创建任务
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.image-section {
  display: flex;
  gap: 24px;
  width: 100%;
}

.image-input-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.image-library-section {
  flex: 1.5;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 500px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}

.section-title {
  font-weight: 500;
  color: #0f172a;
  font-size: 14px;
}

.controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.search-input {
  width: 200px;
}

.sort-select {
  width: 120px;
}

.manual-input {
  width: 100%;
}

.table-container {
  flex: 1;
  overflow-y: auto;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
}

.image-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.image-table thead {
  position: sticky;
  top: 0;
  background: #f8fafc;
  z-index: 1;
}

.image-table th {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: #475569;
  border-bottom: 1px solid #e2e8f0;
  background: #f8fafc;
}

.image-table td {
  padding: 10px 16px;
  border-bottom: 1px solid #f1f5f9;
  color: #334155;
}

.table-row:hover {
  background-color: #f8fafc;
}

.url-cell {
  font-family: monospace;
  font-size: 11px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 300px;
}

.loading-text {
  text-align: center;
  color: #64748b;
  padding: 60px 0;
}

@media (max-width: 768px) {
  .image-section {
    flex-direction: column;
  }

  .image-library-section {
    max-height: 400px;
  }

  .section-header {
    flex-direction: column;
    align-items: stretch;
  }

  .search-input {
    width: 100%;
  }

  .sort-select {
    width: 100%;
  }

  .url-cell {
    max-width: 150px;
  }
}
</style>
