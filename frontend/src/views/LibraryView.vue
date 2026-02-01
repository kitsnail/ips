<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { libraryApi } from '@/services/api'
import type { LibraryImage } from '@/types/api'

const loading = ref(false)
const libraryImages = ref<LibraryImage[]>([])

// Pagination state
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})
const showAddModal = ref(false)
const selectedImages = ref<number[]>([])


const imageInput = ref('')

const loadLibraryImages = async () => {
  try {
    loading.value = true
    const offset = (pagination.value.page - 1) * pagination.value.pageSize
    const response = await libraryApi.list({
      limit: pagination.value.pageSize,
      offset: offset
    })
    libraryImages.value = response.images
    pagination.value.total = response.total || response.images.length
  } catch (error) {
    ElMessage.error('加载镜像库失败')
  } finally {
    loading.value = false
  }
}

const extractDisplayName = (image: string): string => {
  const trimmed = image.trim()
  if (!trimmed) return ''
  const parts = trimmed.split('/')
  const lastPart = parts[parts.length - 1]
  return lastPart || ''
}

const handleAdd = async () => {
  const imageLines = imageInput.value.split('\n').map(line => line.trim()).filter(line => line.length > 0)

  if (imageLines.length === 0) {
    ElMessage.warning('请输入至少一个镜像地址')
    return
  }

  try {
    let successCount = 0
    let failCount = 0

    for (const image of imageLines) {
      try {
        await libraryApi.create({
          name: extractDisplayName(image),
          image: image
        })
        successCount++
      } catch (error) {
        failCount++
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功添加 ${successCount} 个镜像${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
      showAddModal.value = false
      imageInput.value = ''
      loadLibraryImages()
    } else {
      ElMessage.error('所有镜像添加失败')
    }
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
    for (const item of selectedImages.value) {
      // @ts-ignore
      await libraryApi.delete(item.id)
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

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadLibraryImages()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1  // Reset to first page when changing page size
  loadLibraryImages()
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
})

onUnmounted(() => {
  // 清理工作（如果有的话）
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

     <el-dialog v-model="showAddModal" title="添加镜像" width="700px">
      <el-form label-width="100px">
        <el-form-item label="镜像地址" required>
          <el-input
            v-model="imageInput"
            type="textarea"
            :rows="8"
            placeholder="每行输入一个镜像地址，例如：&#10;192.168.3.81/ips/spec-b1:v0&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7"
          />
        </el-form-item>
        <el-form-item label="提示">
          <span style="color: #909399; font-size: 12px;">
            • 每行一个镜像地址&#10;
            • 显示名称将自动从镜像地址中提取（例如：spec-b1:v0）&#10;
            • 支持批量添加多个镜像
          </span>
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
