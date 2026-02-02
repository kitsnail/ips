<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import Sidebar from '@/components/Sidebar.vue'
import Breadcrumb from '@/components/Breadcrumb.vue'

const router = useRouter()
const { user, loadUser } = useAuth()

const collapsed = ref(false)
const darkMode = ref(false)
const showShortcutsDialog = ref(false)

// Load sidebar state
const loadSidebarState = () => {
    const saved = localStorage.getItem('sidebar_collapsed')
    if (saved !== null) {
        collapsed.value = saved === 'true'
    }
}

const toggleSidebar = () => {
    collapsed.value = !collapsed.value
    localStorage.setItem('sidebar_collapsed', String(collapsed.value))
}

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

const logout = () => {
  localStorage.removeItem('ips_token')
  localStorage.removeItem('ips_user')
  router.push('/login')
}

// Shortcuts logic preserved
const shortcuts = [
  { key: 'Ctrl/Cmd + N', label: '创建任务', action: () => router.push('/tasks/create'), category: '任务' },
  { key: 'Ctrl/Cmd + Shift + T', label: '任务列表', action: () => router.push('/tasks'), category: '任务' },
  { key: 'Ctrl/Cmd + S', label: '定时任务', action: () => router.push('/scheduled'), category: '定时' },
  { key: 'Ctrl/Cmd + Shift + S', label: '创建定时任务', action: () => router.push('/scheduled/create'), category: '定时' },
  { key: 'Ctrl/Cmd + L', label: '镜像库', action: () => router.push('/library'), category: '镜像' },
  { key: 'Ctrl/Cmd + E', label: '仓库认证', action: () => router.push('/secrets'), category: '镜像' },
  { key: 'Ctrl/Cmd + A', label: '系统设置', action: () => router.push('/admin/settings'), category: '系统' },
  { key: 'Ctrl/Cmd + G', label: '用户管理', action: () => router.push('/admin/users'), category: '系统' },
  { key: 'Ctrl/Cmd + D', label: '系统日志', action: () => router.push('/admin/logs'), category: '系统' },
  { key: 'Ctrl/Cmd + /', label: '切换主题', action: toggleDarkMode, category: '系统' },
  { key: 'Escape', label: '返回上级', action: () => router.back(), category: '导航' },
]

const handleShortcut = (shortcut: any) => {
  shortcut.action()
  showShortcutsDialog.value = false
}

const handleKeydown = (event: KeyboardEvent) => {
  const target = event.target as HTMLElement
  if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') return

  const isCtrlCmd = event.ctrlKey || event.metaKey
  const isShift = event.shiftKey

  for (const shortcut of shortcuts) {
     const keys = shortcut.key.split(' + ')
     const mainKey = keys[keys.length - 1]
     const needsShift = keys.includes('Shift')
     const needsCtrlCmd = keys.some(k => k.includes('Ctrl') || k.includes('Cmd'))
     
     if (mainKey && event.key.toLowerCase() === mainKey.toLowerCase() && 
         isShift === needsShift && 
         isCtrlCmd === needsCtrlCmd) {
        event.preventDefault()
        handleShortcut(shortcut)
        return
     }
  }
}

const groupedShortcuts = computed(() => {
  const grouped: Record<string, typeof shortcuts> = {}
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
  loadSidebarState()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <el-container class="h-screen w-full bg-slate-50 dark:bg-slate-950 transition-colors duration-300">
    <!-- Sidebar -->
    <el-aside 
      :width="collapsed ? '64px' : '260px'" 
      class="border-r border-gray-200 dark:border-slate-800 bg-white dark:bg-slate-900 transition-all duration-300 overflow-hidden z-20"
    >
      <Sidebar :collapsed="collapsed" />
    </el-aside>

    <el-container class="min-w-0 flex flex-col">
      <!-- Top Bar -->
      <el-header class="!h-16 !p-0 bg-white/80 dark:bg-slate-900/80 backdrop-blur-md border-b border-gray-200 dark:border-slate-800 sticky top-0 z-10">
        <div class="h-full px-6 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <!-- Sidebar Toggle (Desktop) -->
            <button 
              class="hidden lg:flex p-2 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              @click="toggleSidebar"
            >
               <svg v-if="collapsed" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7"></path></svg>
               <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7"></path></svg>
            </button>
            
            <!-- Mobile Toggle -->
            <button class="lg:hidden p-2 text-slate-500 hover:bg-slate-100 rounded-lg">
               <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
            </button>
            <Breadcrumb />
          </div>

          <div class="flex items-center gap-4">
            <!-- Theme Toggle -->
            <button 
              class="p-2 text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 rounded-lg transition-colors"
              @click="toggleDarkMode"
              title="切换主题"
            >
              <svg v-if="darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
              <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path></svg>
            </button>

            <!-- User Dropdown -->
            <el-dropdown trigger="click" class="cursor-pointer">
              <div class="flex items-center gap-3 hover:bg-slate-100 dark:hover:bg-slate-800 py-1.5 px-3 rounded-lg transition-colors border border-transparent hover:border-slate-200 dark:hover:border-slate-700">
                <div class="w-8 h-8 rounded-full bg-gradient-to-tr from-blue-500 to-indigo-500 flex items-center justify-center text-white text-sm font-bold shadow-sm">
                  {{ initials }}
                </div>
                <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ user?.username }}</span>
                <svg class="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
              </div>
              <template #dropdown>
                <el-dropdown-menu class="min-w-[160px]">

                  <el-dropdown-item divided @click="logout" class="text-red-500 hover:!text-red-600 hover:!bg-red-50">
                    <div class="flex items-center gap-2">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
                      <span>退出登录</span>
                    </div>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>

      <!-- Main Content -->
      <el-main class="!p-6 overflow-y-auto relative scroll-smooth">
         <router-view v-slot="{ Component }">
            <transition name="fade-slide" mode="out-in">
              <component :is="Component" />
            </transition>
         </router-view>
      </el-main>
    </el-container>

    <!-- Keyboard Shortcuts Dialog -->
    <el-dialog v-model="showShortcutsDialog" title="快捷键速查" width="600px" class="rounded-xl">
      <div class="grid grid-cols-2 gap-6 p-2">
        <div v-for="(group, category) in groupedShortcuts" :key="category">
          <h3 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">{{ category }}</h3>
          <div class="space-y-2">
            <div v-for="(shortcut, idx) in group" :key="idx" 
                 class="flex items-center justify-between p-2 rounded hover:bg-slate-50 dark:hover:bg-slate-800 cursor-pointer group"
                 @click="handleShortcut(shortcut)">
               <span class="text-sm text-slate-700 dark:text-slate-300 group-hover:text-blue-600 transition-colors">{{ shortcut.label }}</span>
               <div class="flex gap-1">
                 <kbd v-for="k in shortcut.key.split('+')" :key="k" class="px-2 py-1 text-xs font-mono bg-slate-100 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded text-slate-500 dark:text-slate-400 min-w-[24px] text-center shadow-sm">
                   {{ k.trim() }}
                 </kbd>
               </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </el-container>
</template>


<style scoped>
/* Transition Utility */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
