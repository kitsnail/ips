<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'

const logs = ref<Array<{
  timestamp: string
  level: 'info' | 'warning' | 'error' | 'success'
  message: string
  details?: string
}>>([])

const loading = ref(false)
const autoScroll = ref(false)
const selectedLog = ref<typeof logs.value[0] | null>(null)
const logContainer = ref<HTMLElement | null>(null)

const logLevels = ['all', 'info', 'warning', 'error', 'success']
const levelFilter = ref('all')
const searchText = ref('')

const filteredLogs = computed(() => {
  let filtered = logs.value

  // Apply level filter
  if (levelFilter.value !== 'all') {
    filtered = filtered.filter(log => log.level === levelFilter.value)
  }

  // Apply search filter
  if (searchText.value) {
    const searchLower = searchText.value.toLowerCase()
    filtered = filtered.filter(log =>
      log.message.toLowerCase().includes(searchLower) ||
      (log.details && log.details.toLowerCase().includes(searchLower))
    )
  }

  return filtered
})

const getLevelType = (level: string) => {
  switch (level) {
    case 'error':
      return 'danger'
    case 'warning':
      return 'warning'
    case 'success':
      return 'success'
    default:
      return 'info'
  }
}

const getLevelLabel = (level: string) => {
  const levelMap: Record<string, string> = {
    all: '全部',
    info: '信息',
    warning: '警告',
    error: '错误',
    success: '成功',
  }
  return levelMap[level] || level
}

const loadLogs = async () => {
  // In a real implementation, this would fetch from an API
  // For now, we'll simulate with mock data
  loading.value = true

  await new Promise(resolve => setTimeout(resolve, 500))

  logs.value = [
    {
      timestamp: new Date(Date.now() - 60000).toISOString(),
      level: 'info',
      message: '系统启动完成，监听端口 8080',
      details: 'PID: 12345',
    },
    {
      timestamp: new Date(Date.now() - 50000).toISOString(),
      level: 'success',
      message: '用户 admin 登录成功',
      details: 'IP: 192.168.1.100, User-Agent: Mozilla/5.0',
    },
    {
      timestamp: new Date(Date.now() - 40000).toISOString(),
      level: 'warning',
      message: '任务 Task-001 执行超时',
      details: 'TaskID: Task-001, 超时时间: 300s',
    },
    {
      timestamp: new Date(Date.now() - 30000).toISOString(),
      level: 'error',
      message: '拉取镜像 docker.io/library/nginx:latest 失败: connection timeout',
      details: 'Node: node-01, Error: dial tcp i/o timeout',
    },
  ]

  loading.value = false
}

const handleExport = () => {
  const logText = logs.value.map(log =>
    `[${new Date(log.timestamp).toLocaleString()}] [${log.level.toUpperCase()}] ${log.message}${log.details ? ` | ${log.details}` : ''}`
  ).join('\n')

  const blob = new Blob([logText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `ips-logs-${new Date().toISOString().split('T')[0]}.txt`
  a.click()
  URL.revokeObjectURL(url)

  ElMessage.success('日志导出成功')
}

const clearLogs = () => {
  logs.value = []
  ElMessage.info('日志已清空')
}

const handleScroll = () => {
  if (autoScroll.value && logContainer.value) {
    const container = logContainer.value
    container.scrollTop = container.scrollHeight
  }
}

let scrollInterval: number | null = null

const toggleAutoScroll = () => {
  autoScroll.value = !autoScroll.value
  if (autoScroll.value) {
    handleScroll()
    scrollInterval = window.setInterval(() => {
      if (logContainer.value) {
        const container = logContainer.value
        container.scrollTop = container.scrollHeight
      }
    }, 100)
  } else {
    if (scrollInterval) {
      clearInterval(scrollInterval)
      scrollInterval = null
    }
  }
}

const selectLog = (log: typeof logs.value[0]) => {
  selectedLog.value = selectedLog.value === log ? null : log
}

const refreshLogs = () => {
  loadLogs()
}

const formatTime = (timestamp: string): string => {
  return new Date(timestamp).toLocaleString('zh-CN', {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

onMounted(() => {
  loadLogs()
})

onUnmounted(() => {
  if (scrollInterval) {
    clearInterval(scrollInterval)
  }
})
</script>

<template>
  <div class="admin-logs">
    <div class="page-header">
      <h1>系统日志</h1>
      <div class="header-actions">
        <el-button @click="refreshLogs" :loading="loading">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M4 4v5a.5.5 0 00-1 0v-5h4l-4 4m0-5h4V4a.5.5 0 011-1 0v5zm0 7a.5.5 0 000-1v1h6a.5.5 0 001 1v-1H4a.5.5 0 000-1zm6-5a.5.5 0 000-1v5a.5.5 0 000-1 0 5l-3-3a.5.5 0 000-1zm5 2h4a.5.5 0 00-1V5a.5.5 0 001 1V8a.5.5 0 001-1z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          刷新
        </el-button>
        <el-button @click="handleExport">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M21 15v4a2 2 0 00-2 2v-4h2a2 2 0 00-4 4-6a.5.5 0 000-4zm-2 4v14a2 2 0 002 2v-4h2a2 2 0 002 2v-4zm0 0h2v1a.5.5 0 000-1v1h1a.5.5 0 000-1v1H4a.5.5 0 000-1zm1 11h3a2 2 0 00-1-1.5V15h-4a2 2 0 00-4 4-9a.5.5 0 000-1z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          导出
        </el-button>
        <el-button @click="clearLogs">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M19 7l-.867 5.5-.867-5.5a.5.5 0 01-.708 0-1.292-.708 1.292l.708.708 1.708L19 12z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          清空
        </el-button>
        <el-button @click="toggleAutoScroll" :type="autoScroll ? 'primary' : 'default'" size="small">
          {{ autoScroll ? '停止' : '开始' }}滚动
        </el-button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-section">
      <div class="filter-group">
        <span class="filter-label">日志级别:</span>
        <el-select v-model="levelFilter" size="small" class="filter-select">
          <el-option
            v-for="level in logLevels"
            :key="level"
            :label="getLevelLabel(level)"
            :value="level"
          />
        </el-select>
      </div>
      <div class="filter-group">
        <el-input
          v-model="searchText"
          placeholder="搜索日志..."
          :prefix-icon="'Search'"
          size="small"
          class="search-input"
          clearable
        />
        <span class="log-count">{{ filteredLogs.length }} 条</span>
      </div>
    </div>

    <!-- Logs Container -->
    <div class="logs-container" ref="logContainer">
      <el-empty
        v-if="!loading && filteredLogs.length === 0"
        description="暂无日志记录"
        :image-size="120"
      />
      <div v-else class="logs-list">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="log-item"
          :class="`log-${log.level}`"
          @click="selectLog(log)"
        >
          <div class="log-header">
            <span class="log-timestamp">{{ formatTime(log.timestamp) }}</span>
            <el-tag :type="getLevelType(log.level)" size="small" class="log-level">
              {{ getLevelLabel(log.level) }}
            </el-tag>
            <span class="log-level-icon" v-if="log.level === 'error'">⚠️</span>
            <span class="log-level-icon" v-if="log.level === 'warning'">⚡️</span>
            <span class="log-level-icon" v-if="log.level === 'success'">✅</span>
          </div>
          <div class="log-message">{{ log.message }}</div>
          <div v-if="log.details" class="log-details">{{ log.details }}</div>
        </div>
      </div>
    </div>

    <!-- Log Detail Panel -->
    <div v-if="selectedLog" class="log-detail-panel">
      <div class="detail-header">
        <h3>日志详情</h3>
        <el-button size="small" @click="selectedLog = null" text>关闭</el-button>
      </div>
      <div class="detail-content">
        <div class="detail-row">
          <span class="detail-label">时间:</span>
          <span class="detail-value">{{ formatTime(selectedLog.timestamp) }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">级别:</span>
          <el-tag :type="getLevelType(selectedLog.level)" size="small">
            {{ getLevelLabel(selectedLog.level) }}
          </el-tag>
        </div>
        <div class="detail-row">
          <span class="detail-label">消息:</span>
          <span class="detail-message">{{ selectedLog.message }}</span>
        </div>
        <div v-if="selectedLog.details" class="detail-row">
          <span class="detail-label">详情:</span>
          <span class="detail-value">{{ selectedLog.details }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-logs {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
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

/* Filters Section */
.filters-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 16px 24px;
  background: white;
  border-radius: 12px;
  margin-bottom: 24px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-label {
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
}

.filter-select {
  width: 140px;
}

.search-input {
  width: 250px;
}

.log-count {
  font-size: 13px;
  font-weight: 500;
  color: #64748b;
}

/* Logs Container */
.logs-container {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  min-height: 600px;
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-item {
  padding: 12px 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  border-left: 3px solid transparent;
  transition: all 0.2s;
  cursor: pointer;
}

.log-item:hover {
  border-left-color: #0891b2;
  background: #f1f5f9;
  transform: translateX(4px);
}

.log-item.selected {
  background: #e0f2fe;
  border-left-color: #0891b2;
}

.log-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.log-timestamp {
  font-family: monospace;
  font-size: 12px;
  color: #94a3b8;
  flex-shrink: 0;
}

.log-level-icon {
  font-size: 14px;
}

.log-message {
  flex: 1;
  font-size: 14px;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-details {
  font-size: 13px;
  color: #64748b;
  font-family: monospace;
  margin-top: 4px;
}

/* Log Levels */
.log-info {
  border-left-color: #0891b2;
}

.log-warning {
  border-left-color: #f59e0b;
}

.log-error {
  border-left-color: #ef4444;
}

.log-success {
  border-left-color: #22c55e;
}

/* Detail Panel */
.log-detail-panel {
  margin-left: 24px;
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.detail-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-row {
  display: flex;
  gap: 12px;
}

.detail-label {
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
  min-width: 80px;
}

.detail-value {
  font-size: 14px;
  color: #0f172a;
  flex: 1;
  font-family: monospace;
}

.detail-message {
  background: #fef2f2;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 13px;
  color: #0f172a;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .page-header h1 {
    color: #f8fafc;
  }

  .filters-section {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .logs-container {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .log-item {
    background: #0f172a;
    border-color: #334155;
  }

  .log-item:hover {
    background: #1e293b;
    border-color: #22d3ee;
  }

  .log-item.selected {
    background: rgba(34, 211, 238, 0.15);
    border-color: #22d3ee;
  }

  .log-header {
    margin-bottom: 8px;
  }

  .log-message {
    color: #f8fafc;
  }

  .log-details {
    color: #94a3b8;
  }

  .log-info {
    border-left-color: #22d3ee;
  }

  .log-warning {
    border-left-color: #f59e0b;
  }

  .log-error {
    border-left-color: #ef4444;
  }

  .log-success {
    border-left-color: #22c55e;
  }

  .log-detail-panel {
    background: #1e293b;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
  }

  .detail-header {
    border-bottom-color: #334155;
  }

  .detail-header h3 {
    color: #f8fafc;
  }

  .detail-label {
    color: #94a3b8;
  }

  .detail-value {
    color: #f8fafc;
  }

  .detail-message {
    background: rgba(239, 68, 68, 0.15);
    color: #f8fafc;
  }
}
</style>
