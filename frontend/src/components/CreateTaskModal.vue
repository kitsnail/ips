<script setup lang="ts">
import { ref } from 'vue'
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
const selectedLibraryImages = ref<number[]>([])

const libraryImages = ref<LibraryImage[]>([])
const secrets = ref<Secret[]>([])
const loading = ref(false)
const libraryLoading = ref(false)
const secretsLoading = ref(false)

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
}

const handleClose = () => {
  emit('update:visible', false)
}

const handleImageToggle = (imageId: number) => {
  const index = selectedLibraryImages.value.indexOf(imageId)
  if (index > -1) {
    selectedLibraryImages.value.splice(index, 1)
  } else {
    selectedLibraryImages.value.push(imageId)
  }
}

const submit = async () => {
  try {
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
    width="800px"
    @open="handleOpen"
  >
    <el-form label-width="120px">
      <el-form-item label="镜像列表">
        <el-checkbox-group v-model="selectedLibraryImages">
          <div class="image-picker">
            <el-checkbox
              v-for="img in libraryImages"
              :key="img.id"
              :label="img.id"
              border
              @change="handleImageToggle(img.id)"
            >
              <div class="image-item">
                <div class="image-name">{{ img.name }}</div>
                <div class="image-url">{{ img.image }}</div>
              </div>
            </el-checkbox>
          </div>
          <div v-if="libraryLoading" class="loading-text">加载中...</div>
          <el-empty v-else-if="libraryImages.length === 0" description="暂无镜像" :image-size="80" />
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="已选镜像">
        <div class="selected-images">
          <el-tag
            v-for="imgId in selectedLibraryImages"
            :key="imgId"
            closable
            @close="handleImageToggle(imgId)"
          >
            {{ libraryImages.find((i) => i.id === imgId)?.name }}
          </el-tag>
        </div>
      </el-form-item>

      <el-form-item label="批次大小">
        <el-input-number v-model="form.batchSize" :min="1" :max="100" />
      </el-form-item>

      <el-form-item label="优先级">
        <el-input-number v-model="form.priority" :min="1" :max="10" />
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

      <el-form-item label="最大重试">
        <el-input-number v-model="form.maxRetries" :min="0" :max="5" />
      </el-form-item>

      <el-form-item label="重试策略">
        <el-select v-model="form.retryStrategy">
          <el-option label="线性" value="linear" />
          <el-option label="指数退避" value="exponential" />
        </el-select>
      </el-form-item>
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
.image-picker {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.image-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
}

.image-name {
  font-weight: 500;
  color: #0f172a;
}

.image-url {
  font-size: 12px;
  color: #64748b;
  font-family: monospace;
}

.selected-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 32px;
}

.loading-text {
  text-align: center;
  color: #64748b;
  padding: 40px 0;
}

@media (max-width: 640px) {
  .image-picker {
    grid-template-columns: 1fr;
  }
}
</style>
