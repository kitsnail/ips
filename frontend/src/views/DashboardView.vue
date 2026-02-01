<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { taskApi, scheduledTaskApi } from '@/services/api'
import type { Task } from '@/types/api'

const loading = ref(false)
const runningTasks = ref(0)
const successRate = ref(0)
const nodes = ref('0/0')
const scheduledTasks = ref(0)
const scheduledActive = ref(0)
const recentTasks = ref<Task[]>([])



const refreshDashboardStats = async () => {
  try {
    loading.value = true

    const [tasksResponse, scheduledResponse] = await Promise.all([
      taskApi.list({ limit: 1000 }),
      scheduledTaskApi.list({ limit: 100 }),
    ])

    const tasks = tasksResponse?.tasks || []
    const scheduled = scheduledResponse?.tasks || []

    const running = tasks.filter((t) => t.status === 'running').length
    const pending = tasks.filter((t) => t.status === 'pending').length
    runningTasks.value = running + pending

    const today = new Date().toDateString()
    const todayTasks = tasks.filter((t) => new Date(t.createdAt).toDateString() === today)
    const completed = todayTasks.filter((t) => t.status === 'completed').length
    successRate.value = todayTasks.length > 0 ? Math.round((completed / todayTasks.length) * 100) : 100

    scheduledTasks.value = scheduled.length
    scheduledActive.value = scheduled.filter((t) => t.enabled).length

    recentTasks.value = tasks.slice(0, 5)
  } catch (error) {
    console.error('Dashboard loading error:', error)
    ElMessage.error('加载Dashboard失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshDashboardStats()
})

onUnmounted(() => {
  // 清理工作（如果有的话）
})
</script>

<template>
  <div class="dashboard" v-loading="loading">
    <div class="dashboard-grid">
      <div class="stat-card primary">
        <div class="stat-label">运行中任务</div>
        <div class="stat-value">{{ runningTasks }}</div>
        <div class="stat-trend neutral">
          <el-icon><Clock /></el-icon>
          实时更新
        </div>
      </div>
      <div class="stat-card success">
        <div class="stat-label">今日成功率</div>
        <div class="stat-value">{{ successRate }}%</div>
        <div class="stat-trend">
          <el-icon><TrendCharts /></el-icon>
          良好
        </div>
      </div>
      <div class="stat-card info">
        <div class="stat-label">节点覆盖</div>
        <div class="stat-value">{{ nodes }}</div>
        <div class="stat-trend neutral">可用/总数</div>
      </div>
      <div class="stat-card warning">
        <div class="stat-label">定时任务</div>
        <div class="stat-value">{{ scheduledTasks }}</div>
        <div class="stat-trend neutral">
          {{ scheduledActive }} 个已启用
        </div>
      </div>
    </div>

     <div class="card">
       <div class="section-header">
         <h3 class="section-title">最近任务</h3>
       </div>
      <div class="table-container">
        <el-table :data="recentTasks" style="width: 100%">
          <el-table-column prop="taskId" label="任务ID" width="180" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'completed' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="images" label="镜像">
            <template #default="{ row }">
              {{ row.images[0] }}
              <span v-if="row.images.length > 1">+{{ row.images.length - 1 }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="progress.percentage" label="进度" width="120">
            <template #default="{ row }">
              {{ row.progress?.percentage?.toFixed(1) || 0 }}%
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="180">
            <template #default="{ row }">
              {{ new Date(row.createdAt).toLocaleString() }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 0;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  background: transparent;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  transition: box-shadow 0.2s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.stat-card.primary {
  background: linear-gradient(135deg, #0891b2 0%, #0e7490 100%);
  color: white;
}

.stat-card.primary .stat-label,
.stat-card.primary .stat-value {
  color: white;
}

.stat-card.success {
  border-left: 3px solid #22c55e;
}

.stat-card.warning {
  border-left: 3px solid #f59e0b;
}

.stat-card.info {
  border-left: 3px solid #0891b2;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1;
  margin-bottom: 4px;
}

.stat-trend {
  font-size: 12px;
  color: #22c55e;
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat-trend.neutral {
  color: #94a3b8;
}

.card {
  background: transparent;
  border-radius: 16px;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  padding: 0;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid #e2e8f0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.section-actions {
  display: flex;
  gap: 8px;
}

.table-container {
  padding: 0 24px 24px 24px;
}

@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
