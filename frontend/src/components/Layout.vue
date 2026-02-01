<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import Sidebar from '@/components/Sidebar.vue'
import Breadcrumb from '@/components/Breadcrumb.vue'

const router = useRouter()
const { user, isAdmin, loadUser } = useAuth()

const showMobileMenu = ref(false)
const darkMode = ref(false)
const showShortcutsDialog = ref(false)

// Load dark mode preference
const loadDarkMode = () => {
  const saved = localStorage.getItem('dark_mode')
  if (saved !== null) {
    darkMode.value = saved === 'true'
  } else {
    darkMode.value = window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  applyDarkMode()
}

const toggleDarkMode = () => {
  darkMode.value = !darkMode.value
  localStorage.setItem('dark_mode', String(darkMode.value))
  applyDarkMode()
}

const applyDarkMode = () => {
  if (darkMode.value) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

const toggleMobileMenu = () => {
  showMobileMenu.value = !showMobileMenu.value
}

const logout = () => {
  localStorage.removeItem('ips_token')
  localStorage.removeItem('ips_user')
  router.push('/')
}

// Keyboard shortcuts
const shortcuts = [
  {
    key: 'Ctrl/Cmd + N',
    label: '创建任务',
    action: () => router.push('/tasks/create'),
    category: '任务',
  },
  {
    key: 'Ctrl/Cmd + Shift + T',
    label: '任务列表',
    action: () => router.push('/tasks'),
    category: '任务',
  },
  {
    key: 'Ctrl/Cmd + S',
    label: '定时任务',
    action: () => router.push('/scheduled'),
    category: '定时',
  },
  {
    key: 'Ctrl/Cmd + L',
    label: '镜像库',
    action: () => router.push('/library'),
    category: '镜像',
  },
  {
    key: 'Ctrl/Cmd + E',
    label: '仓库认证',
    action: () => router.push('/secrets'),
    category: '镜像',
  },
  {
    key: 'Ctrl/Cmd + A',
    label: '系统设置',
    action: () => router.push('/admin/settings'),
    category: '系统',
  },
  {
    key: 'Ctrl/Cmd + G',
    label: '用户管理',
    action: () => router.push('/admin/users'),
    category: '系统',
  },
  {
    key: 'Ctrl/Cmd + D',
    label: '系统日志',
    action: () => router.push('/admin/logs'),
    category: '系统',
  },
  {
    key: 'Ctrl/Cmd + /',
    label: '切换主题',
    action: toggleDarkMode,
    category: '系统',
  },
  {
    key: 'Escape',
    label: '返回上级',
    action: () => router.back(),
    category: '导航',
  },
]

const handleShortcut = (shortcut: typeof shortcuts[0]) => {
  shortcut.action()
  showShortcutsDialog.value = false
}

const handleKeydown = (event: KeyboardEvent) => {
  // Ignore if user is typing in input field
  const target = event.target as HTMLElement
  if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
    return
  }

  // Check for modifier keys
  const isCtrlCmd = event.ctrlKey || event.metaKey
  const isShift = event.shiftKey

  // Find matching shortcut
  for (const shortcut of shortcuts) {
    const [baseKey, ...modifiers] = shortcut.key.split('+')
    const keyMatches = event.key === baseKey
    const modifiersMatch = modifiers.every(mod => {
      if (mod === 'Ctrl') return isCtrlCmd
      if (mod === 'Cmd') return isCtrlCmd
      if (mod === 'Shift') return isShift
      return false
    })

    if (keyMatches && modifiersMatch) {
      event.preventDefault()
      handleShortcut(shortcut)
      return
    }
  }
}

const groupedShortcuts = computed(() => {
  const grouped: Record<string, Array<{ key: string; label: string; action: string | (() => void); category: string }>> = {}
  shortcuts.forEach(shortcut => {
    const category = shortcut.category
    if (!grouped[category]) {
      grouped[category] = []
    }
    grouped[category]!.push(shortcut)
  })
  return grouped
})

const initials = computed(() => {
  return user.value && user.value.username ? user.value.username.charAt(0).toUpperCase() : 'U'
})

onMounted(() => {
  loadUser()
  loadDarkMode()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

defineExpose({
  toggleDarkMode,
})
</script>

<template>
  <div class="layout" :class="{ dark: darkMode }">
    <!-- Sidebar (Desktop) -->
    <Sidebar class="sidebar-desktop" />

    <!-- Mobile Menu Overlay -->
    <div v-if="showMobileMenu" class="mobile-overlay" @click="toggleMobileMenu"></div>

    <!-- Mobile Menu -->
    <div v-if="showMobileMenu" class="mobile-menu">
      <div class="mobile-menu-header">
        <div class="logo">IPS</div>
        <button class="close-btn" @click="toggleMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M6 18L18 6M6 6l12 12M6 18M6 6l12 12M6 18l6M6 6l-6-6M6l-6 6M12 12M6 12z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
      <nav class="mobile-nav">
        <router-link to="/dashboard" @click="toggleMobileMenu">概览</router-link>
        <router-link to="/tasks" @click="toggleMobileMenu">任务管理</router-link>
        <router-link to="/scheduled" @click="toggleMobileMenu">定时任务</router-link>
        <router-link to="/library" @click="toggleMobileMenu">镜像库</router-link>
        <router-link to="/secrets" @click="toggleMobileMenu">仓库认证</router-link>
        <router-link
          v-if="isAdmin"
          to="/admin/settings"
          @click="toggleMobileMenu"
          class="admin-link"
        >
          系统设置
        </router-link>
        <router-link
          v-if="isAdmin"
          to="/admin/users"
          @click="toggleMobileMenu"
          class="admin-link"
        >
          用户管理
        </router-link>
        <router-link
          v-if="isAdmin"
          to="/admin/logs"
          @click="toggleMobileMenu"
          class="admin-link"
        >
          系统日志
        </router-link>
         </nav>
    </div>

    <!-- Main Content -->
    <div class="main-content">
      <!-- Top Bar -->
      <header class="top-bar">
        <div class="top-bar-left">
          <button class="mobile-menu-btn" @click="toggleMobileMenu">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M4 6h16M4 12h.01M21 12a10 10H5a10 10 0 00 11h-4M15 11H4l-10 10 0 000-1z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <Breadcrumb />
        </div>

        <div class="top-bar-right">
          <button
            class="theme-toggle"
            @click="toggleDarkMode"
            :title="darkMode ? '切换到浅色模式' : '切换到深色模式'"
          >
            <svg v-if="darkMode" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M12 3v1m0 2a2 2 0 00-2v1m1.42 10 0 01.58 0 11.42 0 011 0zm-1 13.42 0 000-2-2h13a2 2 0 00-2v1.58a.2 2 0 011.0zm-1 13.42 0 000-2-2h13a2 2 0 0 0-2v1.58a.2 2 0 00-2v1.58a.2 2 0 011.0z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M20 7h-9M20 11h-9M20 15h-9M3 7h2v10H3V7zm0 0l2-2M3 17l2 2 0 00-2v-1M19 9 10.01 10.01 0 2 9.9.0 00-10.01-2v1.9 10 02 0z"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <el-dropdown trigger="click" class="user-dropdown">
            <div class="user-info">
              <div class="user-avatar">{{ initials }}</div>
              <span class="username">{{ user?.username }}</span>
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                  d="M6 9l6 6m0 0l-6 6m6-6H6m12 0H12m-6 6h.01M18 12h.01M12 6h.01M18 6h.01M6 12h12"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="showShortcutsDialog = true">
                  <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path
                      d="M13 10V3L4 14h7v7l9-11h-7z"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                  <span>快捷键</span>
                </el-dropdown-item>
                <el-dropdown-item divided @click="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <!-- Page Content -->
      <main class="page-content">
        <router-view></router-view>
      </main>
    </div>

    <!-- Keyboard Shortcuts Dialog -->
    <el-dialog v-model="showShortcutsDialog" title="快捷键" width="800px">
      <div class="shortcuts-container">
        <div
          v-for="(group, category) in Object.entries(groupedShortcuts)"
          :key="category"
          class="shortcuts-group"
        >
          <div class="group-title">{{ category }}</div>
          <div class="shortcuts-list">
            <div
              v-for="(shortcut, index) in group"
              :key="index"
              class="shortcut-item"
              @click="handleShortcut(shortcut as any)"
            >
              <div class="shortcut-keys">
                <span
                  v-for="(key, keyIndex) in (shortcut as any).key.split('+')"
                  :key="keyIndex"
                  class="key-badge"
                >
                  {{ key }}
                </span>
              </div>
              <span class="shortcut-label">{{ (shortcut as any).label }}</span>
              <span class="shortcut-description"></span>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  background: #f8fafc;
  background-image: radial-gradient(#cffafe 1px, transparent 1px);
  background-size: 24px 24px;
  transition: background-color 0.3s;
}

.layout.dark {
  background: #020617;
  background-image: radial-gradient(rgba(34, 211, 238, 0.05) 1px, transparent 1px);
}

/* Sidebar (Desktop) */
.sidebar-desktop {
  display: none;
}

@media (min-width: 1024px) {
  .sidebar-desktop {
    display: block;
  }
}

/* Mobile Menu Overlay */
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1000;
}

.mobile-menu {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #ffffff;
  z-index: 1001;
  display: flex;
  flex-direction: column;
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(0%);
  }
}

.mobile-menu-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 20px;
  border-bottom: 1px solid #e2e8f0;
}

.mobile-menu-header .logo {
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
}

.close-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 8px;
  color: #64748b;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.close-btn svg {
  width: 24px;
  height: 24px;
}

.mobile-nav {
  flex: 1;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mobile-nav a {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;
  text-decoration: none;
  font-size: 14px;
  padding: 8px 12px;
  border-radius: 8px;
  transition: color 0.2s;
  border: 1px solid transparent;
}

.mobile-nav a:hover {
  background: #e0f2fe;
  color: #0f172a;
}

.mobile-nav a.router-link-active {
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.mobile-nav .admin-link {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
}

.mobile-nav .admin-link.router-link-active {
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

/* Main Content */
.main-content {
  flex: 1;
  min-width: 0;
}

/* Top Bar */
.top-bar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  transition: background-color 0.3s;
}

.top-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mobile-menu-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 8px;
  color: #64748b;
  transition: all 0.2s;
}

.mobile-menu-btn:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.mobile-menu-btn svg {
  width: 24px;
  height: 24px;
}

/* User Dropdown */
.user-dropdown {
  cursor: pointer;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 32px;
  height: 32px;
  background: #3b82f6;
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
}

.username {
  font-size: 14px;
  font-weight: 500;
  color: #0f172a;
  white-space: nowrap;
}

.dropdown-arrow {
  width: 16px;
  height: 16px;
  color: #64748b;
}

/* Keyboard Shortcuts Dialog */
.shortcuts-container {
  max-height: 600px;
  overflow-y: auto;
}

.shortcuts-group {
  margin-bottom: 24px;
}

.shortcuts-group:last-child {
  margin-bottom: 0;
}

.group-title {
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.shortcuts-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.shortcut-item:hover {
  background: #e0f2fe;
  border-color: #0891b2;
}

.shortcut-keys {
  display: flex;
  gap: 4px;
}

.key-badge {
  padding: 4px 8px;
  background: #0891b2;
  color: white;
  border-radius: 4px;
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  font-weight: 600;
  min-width: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.shortcut-label {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
  color: #0f172a;
}

.shortcut-description {
  flex: 1;
  font-size: 13px;
  color: #64748b;
  text-align: right;
}

.menu-icon {
  width: 16px;
  height: 16px;
  color: #0891b2;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .top-bar {
    background: rgba(15, 23, 42, 0.9);
    border-bottom-color: rgba(255, 255, 255, 0.1);
  }

  .mobile-menu {
    background: #0f172a;
  }

  .mobile-menu-header {
    border-bottom-color: #334155;
  }

  .close-btn:hover {
    background: #334155;
    color: #f8fafc;
  }

  .mobile-nav a {
    color: #94a3b8;
  }

  .mobile-nav a:hover {
    background: #1e293b;
    color: #f8fafc;
  }

  .mobile-nav a.router-link-active {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .mobile-nav .admin-link.router-link-active {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .user-avatar {
    background: #3b82f6;
  }

  .username {
    color: #f8fafc;
  }

  .dropdown-arrow {
    color: #94a3b8;
  }

  .shortcuts-container {
    background: #1e293b;
    border: 1px solid #334155;
  }

  .group-title {
    color: #f8fafc;
    border-bottom-color: #334155;
  }

  .shortcut-item {
    background: #0f172a;
    border-color: #334155;
  }

  .shortcut-item:hover {
    background: #1e293b;
    border-color: #22d3ee;
  }

  .key-badge {
    background: #22d3ee;
  }

  .shortcut-label {
    color: #f8fafc;
  }

  .shortcut-description {
    color: #94a3b8;
  }

  .menu-icon {
    color: #22d3ee;
  }
}
</style>
