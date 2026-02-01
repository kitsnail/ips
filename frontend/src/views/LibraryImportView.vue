<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { libraryApi } from '@/services/api'

const router = useRouter()

const importInput = ref('')
const loading = ref(false)
const successCount = ref(0)
const failCount = ref(0)
const importedImages = ref<string[]>([])
const parsingErrors = ref<Array<{ line: number; error: string }>>([])

const extractDisplayName = (image: string): string => {
  const trimmed = image.trim()
  if (!trimmed) return ''
  const parts = trimmed.split('/')
  const lastPart = parts[parts.length - 1]
  return lastPart || trimmed
}

const validateImageLine = (line: string): { valid: boolean; displayName: string; imageUrl: string } => {
  const trimmed = line.trim()
  if (!trimmed) {
    return { valid: false, displayName: '', imageUrl: '' }
  }

  // Check if it's a valid image URL format
  const imageRegex = /^[\w.-]+(:\S+)?$/
  if (!imageRegex.test(trimmed)) {
    return { valid: false, displayName: '', imageUrl: trimmed }
  }

  return {
    valid: true,
    displayName: extractDisplayName(trimmed),
    imageUrl: trimmed,
  }
}

const parseImport = () => {
  const lines = importInput.value.split('\n')
  const errors: Array<{ line: number; error: string }> = []
  const validImages: string[] = []

  lines.forEach((line, index) => {
    const result = validateImageLine(line)

    if (result.valid) {
      validImages.push(result.imageUrl)
    } else {
      errors.push({
        line: index + 1,
        error: result.imageUrl ? `格式错误: ${result.imageUrl}` : '空行',
      })
    }
  })

  parsingErrors.value = errors
  importedImages.value = validImages
}

const handleImport = async () => {
  if (importedImages.value.length === 0) {
    ElMessage.warning('请输入要导入的镜像地址')
    return
  }

  if (parsingErrors.value.length > 0) {
    ElMessage.warning(`发现 ${parsingErrors.value.length} 个格式错误，请检查后重试`)
    return
  }

  try {
    loading.value = true
    successCount.value = 0
    failCount.value = 0

    for (const image of importedImages.value) {
      try {
        await libraryApi.create({
          name: extractDisplayName(image),
          image: image,
        })
        successCount.value++
      } catch (error) {
        failCount.value++
        console.error(`Failed to import image: ${image}`, error)
      }
    }

    if (successCount.value > 0) {
      ElMessage.success(
        `成功导入 ${successCount.value} 个镜像${failCount.value > 0 ? `，失败 ${failCount.value} 个` : ''}`
      )
    } else {
      ElMessage.error('所有镜像导入失败')
    }

    importInput.value = ''
    importedImages.value = []
    parsingErrors.value = []
    successCount.value = 0
    failCount.value = 0
  } catch (error) {
    ElMessage.error('导入失败，请稍后重试')
    console.error('Import error:', error)
  } finally {
    loading.value = false
  }
}

const handleClear = () => {
  importInput.value = ''
  parsingErrors.value = []
  importedImages.value = []
}

const handleBack = () => {
  router.back()
}
</script>

<template>
  <div class="library-import">
    <div class="page-header">
      <h1>批量导入镜像</h1>
      <div class="header-actions">
        <el-button @click="handleBack">返回</el-button>
        <el-button @click="handleClear" :disabled="!importInput && parsingErrors.length === 0">
          清空
        </el-button>
        <el-button
          type="primary"
          @click="handleImport"
          :loading="loading"
          :disabled="importedImages.length === 0 || parsingErrors.length > 0"
        >
          导入
        </el-button>
      </div>
    </div>

    <div class="content-container">
      <!-- Import Input -->
      <div class="import-section">
        <div class="section-header">
          <h3>镜像列表</h3>
          <div class="section-info">
            每行输入一个镜像地址
          </div>
        </div>

        <div class="input-container">
          <el-input
            v-model="importInput"
            type="textarea"
            :rows="15"
            placeholder="例如：&#10;docker.io/library/nginx:latest&#10;registry.cn-hangzhou.aliyuncs.com/library/redis:7&#10;192.168.1.100:5000/myapp:v1.0.0"
            @input="parseImport"
            class="import-textarea"
          />
        </div>

        <!-- Parsing Errors -->
        <div v-if="parsingErrors.length > 0" class="errors-section">
          <div class="errors-title">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 8v4m0 6h.01M12 21v-4m0-6H12m0-6h6"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            格式错误 ({{ parsingErrors.length }})
          </div>
          <div class="errors-list">
            <div v-for="error in parsingErrors" :key="error.line" class="error-item">
              <span class="error-line">行 {{ error.line }}:</span>
              <span class="error-text">{{ error.error }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Import Preview -->
      <div v-if="importedImages.length > 0 && parsingErrors.length === 0" class="preview-section">
        <div class="preview-header">
          <h3>待导入镜像</h3>
          <div class="preview-count">{{ importedImages.length }} 个</div>
        </div>

        <div class="preview-list">
          <div v-for="(image, index) in importedImages" :key="index" class="preview-item">
            <div class="preview-index">{{ index + 1 }}.</div>
            <div class="preview-name">{{ extractDisplayName(image) }}</div>
            <div class="preview-url">{{ image }}</div>
            <el-button
              type="danger"
              size="small"
              text
              @click="importedImages.splice(index, 1); parseImport()"
            >
              移除
            </el-button>
          </div>
        </div>
      </div>

      <!-- Instructions -->
      <div class="instructions-section">
        <h3>使用说明</h3>
        <ul class="instructions-list">
          <li>每行输入一个完整的镜像地址(包含仓库和tag)</li>
          <li>格式示例: <code>docker.io/library/nginx:latest</code></li>
          <li>支持私有仓库镜像和公共镜像</li>
          <li>导入时会自动从镜像地址提取显示名称</li>
          <li>重复的镜像名称会自动添加序号区分</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.library-import {
  padding: 0;
  max-width: 1200px;
  margin: 0 auto;
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

.header-actions {
  display: flex;
  gap: 12px;
}

/* Import Section */
.import-section {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.section-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.section-info {
  font-size: 14px;
  color: #64748b;
}

.input-container {
  margin-top: 16px;
}

.import-textarea {
  font-family: 'Monaco', 'Consolas', 'Monospace', monospace;
}

.import-textarea :deep(textarea) {
  font-size: 14px;
  line-height: 1.6;
}

/* Errors Section */
.errors-section {
  margin-top: 20px;
  padding: 16px;
  background: #fef2f2;
  border-left: 4px solid #ef4444;
  border-radius: 8px;
}

.errors-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #991b1b;
  margin-bottom: 12px;
}

.errors-title svg {
  width: 20px;
  height: 20px;
}

.errors-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.error-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  font-size: 13px;
}

.error-line {
  color: #dc2626;
  font-weight: 600;
  font-family: monospace;
  min-width: 60px;
}

.error-text {
  color: #991b1b;
}

/* Preview Section */
.preview-section {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.preview-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.preview-count {
  font-size: 14px;
  font-weight: 500;
  color: #0891b2;
  background: #e0f2fe;
  padding: 4px 12px;
  border-radius: 16px;
}

.preview-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.preview-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  transition: all 0.2s;
}

.preview-item:hover {
  background: #e0f2fe;
  border-color: #0891b2;
  transform: translateX(2px);
}

.preview-index {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0891b2;
  color: white;
  border-radius: 50%;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.preview-name {
  flex: 1;
  font-weight: 500;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-url {
  flex: 2;
  font-family: monospace;
  font-size: 12px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Instructions Section */
.instructions-section {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.instructions-section h3 {
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  margin: 0 0 20px;
}

.instructions-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.instructions-list li {
  position: relative;
  padding-left: 24px;
  padding-bottom: 12px;
  font-size: 14px;
  line-height: 1.6;
  color: #475569;
}

.instructions-list li:last-child {
  padding-bottom: 0;
}

.instructions-list li::before {
  content: '•';
  position: absolute;
  left: 0;
  color: #0891b2;
  font-weight: 600;
  font-size: 18px;
}

.instructions-list code {
  background: #f1f5f9;
  color: #0f172a;
  padding: 2px 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Consolas', 'Monospace', monospace;
  font-size: 13px;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h1 {
    color: #f8fafc;
  }

  .section-header {
    border-bottom-color: #334155;
  }

  .section-header h3 {
    color: #f8fafc;
  }

  .section-info {
    color: #94a3b8;
  }

  .import-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .errors-section {
    background: rgba(239, 68, 68, 0.15);
    border-left-color: #f87171;
  }

  .errors-title {
    color: #fecaca;
  }

  .error-line {
    color: #fca5a5;
  }

  .error-text {
    color: #fecaca;
  }

  .preview-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .preview-header h3 {
    color: #f8fafc;
  }

  .preview-count {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .preview-item {
    background: #0f172a;
    border-color: #334155;
  }

  .preview-item:hover {
    background: #1e293b;
    border-color: #22d3ee;
  }

  .preview-name {
    color: #f8fafc;
  }

  .preview-url {
    color: #94a3b8;
  }

  .instructions-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .instructions-section h3 {
    color: #f8fafc;
  }

  .instructions-list li {
    color: #cbd5e1;
  }

  .instructions-list li::before {
    color: #22d3ee;
  }

  .instructions-list code {
    background: #334155;
    color: #f8fafc;
  }
}
</style>
