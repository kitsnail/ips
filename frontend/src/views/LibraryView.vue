<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { libraryApi } from '@/services/api'
import type { LibraryImage, SaveImageRequest } from '@/types/api'

const loading = ref(false)
const libraryImages = ref<LibraryImage[]>([])
const showAddModal = ref(false)
const selectedImages = ref<number[]>([])
let refreshInterval: number | null = null

const form = ref<SaveImageRequest>({
  name: '',
  image: '',
})

const loadLibraryImages = async () => {
  try {
    loading.value = true
    const response = await libraryApi.list({ limit: 100, offset: 0 })
    libraryImages.value = response.images
  } catch (error) {
    ElMessage.error('加载镜像库失败')
  } finally {
    loading.value = false
  }
}

const handleAdd = async () => {
  try {
    await libraryApi.create(form.value)
    ElMessage.success('镜像添加成功')
    showAddModal.value = false
    form.value = { name: '', image: '' }
    loadLibraryImages()
  } catch (error) {
    ElMessage.error('添加镜像失败')
  }
}

const handleDelete = async (image: LibraryImage) => {
  try {
    await ElMessageBox.confirm('确定要删除这个镜像吗？', '确认删除', {
      type: 'warning',
    })
    await libraryApi.delete(image.id)
    ElMessage.success('镜像已删除')
    loadLibraryImages()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning('请先选择要删除的镜像')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedImages.value.length} 个镜像吗？`,
      '确认删除',
      { type: 'warning' }
    )
    for (const id of selectedImages.value) {
      await libraryApi.delete(id)
    }
    ElMessage.success(`成功删除 ${selectedImages.value.length} 个镜像`)
    selectedImages.value = []
    loadLibraryImages()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

const handleSelectAll = (checked: boolean) => {
  if (checked) {
    selectedImages.value = libraryImages.value.map((img) => img.id)
  } else {
    selectedImages.value = []
  }
}

onMounted(() => {
  loadLibraryImages()
  refreshInterval = window.setInterval(() => {
    loadLibraryImages()
  }, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="library" v-loading="loading">
    <div class="header">
      <h2>镜像库管理</h2>
      <div class="actions">
        <el-button
          type="danger"
          :disabled="selectedImages.length === 0"
          @click="handleBatchDelete"
        >
          批量删除 ({{ selectedImages.length }})
        </el-button>
        <el-button type="primary" @click="showAddModal = true">
          添加镜像
        </el-button>
      </div>
    </div>
    <el-table
      :data="libraryImages"
      @selection-change="selectedImages = $event"
      style="width: 100%"
    >
      <el-table-column type="selection" width="55" @select-all="handleSelectAll" />
      <el-table-column prop="name" label="显示名称" width="200" />
      <el-table-column prop="image" label="镜像地址" min-width="300">
        <template #default="{ row }">
          <span style="font-family: monospace; color: #0891b2;">{{ row.image }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="添加时间" width="180">
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

    <el-dialog v-model="showAddModal" title="添加镜像" width="500px">
      <el-form label-width="100px">
        <el-form-item label="显示名称" required>
          <el-input v-model="form.name" placeholder="例如：Nginx" />
        </el-form-item>
        <el-form-item label="镜像地址" required>
          <el-input v-model="form.image" placeholder="docker.io/library/nginx:latest" />
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
.library {
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
