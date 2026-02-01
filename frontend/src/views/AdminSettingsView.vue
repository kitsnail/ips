<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { healthApi } from '@/services/api'

const settings = ref({
  batchSize: 10,
  priority: 5,
  maxRetries: 3,
  retryStrategy: 'linear' as 'linear' | 'exponential',
  retryDelay: 30,
  maxConcurrentTasks: 3,
  enableWebhook: false,
  webhookUrl: '',
})

const loading = ref(false)
const systemInfo = ref({
  version: 'IPS v1.0.0',
  uptime: '运行中',
  environment: 'Development',
})

const loadSettings = () => {
  // Load settings from localStorage or API
  const savedSettings = localStorage.getItem('ips_settings')
  if (savedSettings) {
    try {
      const parsed = JSON.parse(savedSettings)
      settings.value = { ...settings.value, ...parsed }
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }
}

const saveSettings = async () => {
  try {
    loading.value = true
    localStorage.setItem('ips_settings', JSON.stringify(settings.value))
    await new Promise(resolve => setTimeout(resolve, 500))
    ElMessage.success('设置已保存')
  } catch (error) {
    ElMessage.error('保存设置失败')
  } finally {
    loading.value = false
  }
}

const resetDefaults = () => {
  settings.value = {
    batchSize: 10,
    priority: 5,
    maxRetries: 3,
    retryStrategy: 'linear',
    retryDelay: 30,
    maxConcurrentTasks: 3,
    enableWebhook: false,
    webhookUrl: '',
  }
  ElMessage.info('已恢复默认设置')
}

const checkHealth = async () => {
  try {
    const health = await healthApi.check()
    systemInfo.value = {
      ...systemInfo.value,
      uptime: '运行中',
      environment: health.includes('Production') ? 'Production' : 'Development'
    }
    ElMessage.success(`系统健康检查通过: ${health}`)
  } catch (error) {
    ElMessage.error('系统健康检查失败')
  }
}

onMounted(() => {
  loadSettings()
  // Update system info
  systemInfo.value.uptime = `运行中`
  systemInfo.value.environment = 'Development'
})
</script>

<template>
  <div class="admin-settings">
    <div class="page-header">
      <h1>系统设置</h1>
      <div class="header-actions">
        <el-button @click="resetDefaults">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M4 4v5h.582m15.356-2A3 3 0 00-4.243 0 9.883L13.415 15l-1.766-2.766a3 3 0 00-4.243-9.883 0 0zm-3.5 9a3.5 3.5 0 000-7 0 3.5 3.5 0 007 0z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          恢复默认
        </el-button>
        <el-button type="primary" @click="saveSettings" :loading="loading">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M19 21H5a2 2 0 00-2-2V5a2 2 0 00-2-2h11l-3-3m0 0l3 3m4-4H6m4 4v6"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          保存设置
        </el-button>
      </div>
    </div>

    <div class="content-container">
      <!-- Task Settings -->
      <div class="settings-section">
        <div class="section-header">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <h2>任务默认配置</h2>
        </div>

        <div class="settings-content">
          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">批次大小</div>
              <div class="label-description">默认节点批次处理大小</div>
            </div>
            <el-input-number v-model="settings.batchSize" :min="1" :max="100" />
            <span class="setting-hint">节点: 1-100</span>
          </div>

          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">优先级</div>
              <div class="label-description">默认任务优先级</div>
            </div>
            <el-input-number v-model="settings.priority" :min="1" :max="10" />
            <span class="setting-hint">范围: 1-10</span>
          </div>

          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">最大重试次数</div>
              <div class="label-description">失败时的默认重试上限</div>
            </div>
            <el-input-number v-model="settings.maxRetries" :min="0" :max="5" />
            <span class="setting-hint">范围: 0-5</span>
          </div>

          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">重试策略</div>
              <div class="label-description">重试时的退避算法</div>
            </div>
            <el-select v-model="settings.retryStrategy" style="width: 100%">
              <el-option label="线性" value="linear" />
              <el-option label="指数退避" value="exponential" />
            </el-select>
          </div>

          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">重试延迟(秒)</div>
              <div class="label-description">重试之间的等待时间</div>
            </div>
            <el-input-number v-model="settings.retryDelay" :min="5" :max="300" />
            <span class="setting-hint">范围: 5-300秒</span>
          </div>

          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">最大并发任务数</div>
              <div class="label-description">同时运行的最大任务数</div>
            </div>
            <el-input-number v-model="settings.maxConcurrentTasks" :min="1" :max="10" />
            <span class="setting-hint">范围: 1-10</span>
          </div>
        </div>
      </div>

      <!-- Webhook Settings -->
      <div class="settings-section">
        <div class="section-header">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <h2>通知配置</h2>
        </div>

        <div class="settings-content">
          <div class="setting-item">
            <div class="setting-label">
              <div class="label-text">启用Webhook</div>
              <div class="label-description">任务完成时发送通知</div>
            </div>
            <el-switch v-model="settings.enableWebhook" />
          </div>

          <div v-if="settings.enableWebhook" class="setting-item full-width">
            <div class="setting-label">
              <div class="label-text">Webhook URL</div>
              <div class="label-description">接收POST通知的URL</div>
            </div>
            <el-input
              v-model="settings.webhookUrl"
              placeholder="https://your-webhook-endpoint.com/notify"
              class="webhook-input"
            />
          </div>
        </div>
      </div>

      <!-- System Info -->
      <div class="settings-section">
        <div class="section-header">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0zM13 8h-1V6h1m1 0h.01M21 8a9 9 0 11-18 0 9 9 0 0118 0zM9 16h.01v5H8v-5H8m0-2h2v7H9zm-2 0h10.01M5 8h14v2H5v-2z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <h2>系统信息</h2>
          <div class="header-actions">
            <el-button @click="checkHealth" size="small">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                  d="M22 11.08V12a10 10 0 00-10-10h-4.077C22 10.634 12.099 12 13.309 12 15V3.464C12 1.988 10.828 8.977 8 7.857l1.292-1.292a1 1 0 01.414 1.414l8.586 8.586a1 1 0 011.414-1.414 19.536 11.366 21.504 13.309 22 11.308z"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              健康检查
            </el-button>
          </div>
        </div>

        <div class="system-info">
          <div class="info-item">
            <span class="info-label">版本</span>
            <span class="info-value">{{ systemInfo.version }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">运行时间</span>
            <span class="info-value">{{ new Date().toLocaleString() }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">环境</span>
            <span class="info-value">{{ systemInfo.environment }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-settings {
  padding: 0;
  max-width: 1000px;
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

.header-actions svg {
  width: 18px;
  height: 18px;
}

/* Settings Sections */
.content-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-section {
  background: transparent;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.section-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.section-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.section-header svg {
  width: 20px;
  height: 20px;
  color: #0891b2;
}

.settings-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Setting Items */
.setting-item {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  transition: all 0.2s;
}

.setting-item:hover {
  background: #f1f5f9;
}

.setting-item.full-width {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}

.setting-label {
  flex: 1;
  min-width: 200px;
}

.label-text {
  font-size: 14px;
  font-weight: 500;
  color: #0f172a;
  margin-bottom: 4px;
}

.label-description {
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
}

.setting-hint {
  font-size: 12px;
  color: #94a3b8;
  font-family: monospace;
}

.el-input-number {
  width: 150px;
}

.el-select {
  width: 150px;
}

.webhook-input {
  flex: 1;
}

/* System Info */
.system-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: #0f172a;
  font-family: monospace;
  background: #f1f5f9;
  padding: 6px 12px;
  border-radius: 4px;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h1 {
    color: #f8fafc;
  }

  .settings-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .section-header {
    border-bottom-color: #334155;
  }

  .section-header h2 {
    color: #f8fafc;
  }

  .section-header svg {
    color: #22d3ee;
  }

  .setting-item {
    background: #0f172a;
  }

  .setting-item:hover {
    background: #1e293b;
  }

  .label-text {
    color: #f8fafc;
  }

  .label-description {
    color: #94a3b8;
  }

  .setting-hint {
    color: #64748b;
  }

  .info-label {
    color: #94a3b8;
  }

  .info-value {
    color: #f8fafc;
    background: #334155;
  }
}
</style>
