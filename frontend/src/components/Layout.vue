<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const useRouterInstance = useRouter()
const route = useRoute()
const { user, isAdmin } = useAuth()

const activeTab = computed(() => route.path)

const menuItems = [
  { name: 'Dashboard', path: '/web/dashboard', label: '概览' },
  { name: 'Tasks', path: '/web/tasks', label: '任务管理' },
  { name: 'ScheduledTasks', path: '/web/scheduled', label: '定时任务' },
  { name: 'Library', path: '/web/library', label: '镜像库' },
  { name: 'Secrets', path: '/web/secrets', label: '仓库认证' },
]

const adminMenuItems = [
  { name: 'Admin', path: '/web/admin', label: '系统设置' },
]

const logout = () => {
  localStorage.removeItem('ips_token')
  localStorage.removeItem('ips_user')
  useRouterInstance.push('/')
}
</script>

<template>
  <div class="layout">
    <div class="header">
      <div class="header-content">
        <h1 class="logo">
          <span class="logo-icon"></span>
          镜像预热控制台（IPS）
        </h1>
        <nav class="nav-tabs">
          <router-link
            v-for="item in menuItems"
            :key="item.path"
            :to="item.path"
            class="nav-link"
            :class="{ active: activeTab === item.path }"
          >
            {{ item.label }}
          </router-link>
          <router-link
            v-for="item in adminMenuItems"
            :key="item.path"
            :to="item.path"
            class="nav-link"
            :class="{ active: activeTab === item.path, 'admin-only': !isAdmin }"
            v-if="isAdmin"
          >
            {{ item.label }}
          </router-link>
        </nav>
        <div class="user-info">
          <div class="user-avatar">{{ user?.username?.charAt(0).toUpperCase() }}</div>
          <span class="username">{{ user?.username }}</span>
          <el-button text @click="logout">退出登录</el-button>
        </div>
      </div>
    </div>
    <div class="main-content">
      <router-view></router-view>
    </div>
  </div>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  background: #f8fafc;
  background-image: radial-gradient(#cffafe 1px, transparent 1px);
  background-size: 24px 24px;
}

.header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255,255,255, 0.9);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid #e2e8f0;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  height: 64px;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 24px;
  max-width: 1440px;
  margin: 0 auto;
}

.logo {
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  margin-right: 48px;
  display: flex;
  align-items: center;
  gap: 12px;
  letter-spacing: -0.025em;
}

.logo-icon {
  display: block;
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #0891b2 0%, #22d3ee 100%);
  border-radius: 8px;
  box-shadow: 0 0 0 1px rgba(8, 145, 178, 0.1), 0 4px 6px -1px rgba(8, 145, 178, 0.2);
}

.nav-tabs {
  display: flex;
  gap: 8px;
  margin-right: auto;
  height: 100%;
  align-items: center;
}

.nav-link {
  color: #64748b;
  font-weight: 500;
  font-size: 14px;
  padding: 6px 16px;
  height: 36px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  transition: all 0.2s ease;
  cursor: pointer;
  border: none;
  text-decoration: none;
}

.nav-link:hover {
  color: #0f172a;
  background: rgba(15, 23, 42, 0.05);
}

.nav-link.active {
  color: #0891b2;
  background: #eef2ff;
  font-weight: 600;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
}

.user-avatar {
  width: 32px;
  height: 32px;
  background: #bfdbfe;
  color: #1e40af;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.username {
  color: #0f172a;
  font-weight: 500;
}

.main-content {
  max-width: 1440px;
  margin: 0 auto;
  padding: 32px 24px;
}
</style>
