<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import {
  DataBoard,
  List,
  Clock,
  Grid,
  Lock,
  Setting,
} from '@element-plus/icons-vue'

const props = defineProps<{
  collapsed: boolean
}>()

const route = useRoute()
const { isAdmin } = useAuth()

const activeMenu = computed(() => {
  return route.path
})

interface MenuItem {
  id: string
  icon?: any
  label: string
  path: string
  children?: ChildMenuItem[]
}

interface ChildMenuItem {
  id: string
  label: string
  path: string
  highlight?: boolean
}

const menuItems: MenuItem[] = [
  {
    id: 'dashboard',
    icon: DataBoard,
    label: '概览',
    path: '/dashboard',
  },
  {
    id: 'tasks',
    icon: List,
    label: '任务管理',
    path: '/tasks',
    children: [
      { id: 'tasks-list', label: '全部任务', path: '/tasks' },
      { id: 'tasks-create', label: '创建任务', path: '/tasks/create' },
    ],
  },
  {
    id: 'scheduled',
    icon: Clock,
    label: '定时任务',
    path: '/scheduled',
    children: [
      { id: 'scheduled-list', label: '任务列表', path: '/scheduled' },
      { id: 'scheduled-create', label: '创建任务', path: '/scheduled/create' },
      { id: 'scheduled-history', label: '执行历史', path: '/scheduled/history' },
    ],
  },
  {
    id: 'library',
    icon: Grid,
    label: '镜像库',
    path: '/library',
    children: [
      { id: 'library-list', label: '镜像管理', path: '/library' },
      { id: 'library-import', label: '批量导入', path: '/library/import' },
    ],
  },
  {
    id: 'secrets',
    icon: Lock,
    label: '仓库认证',
    path: '/secrets',
    children: [
      { id: 'secrets-list', label: '认证列表', path: '/secrets' },
      { id: 'secrets-create', label: '添加认证', path: '/secrets/create' },
    ],
  },
]

const adminMenuItems: MenuItem[] = [
  {
    id: 'admin',
    icon: Setting,
    label: '系统设置',
    path: '/admin',
    children: [
      { id: 'admin-users', label: '用户管理', path: '/admin/users' },
      { id: 'admin-settings', label: '系统配置', path: '/admin/settings' },
      { id: 'admin-logs', label: '系统日志', path: '/admin/logs' },
    ],
  },
]

const allMenuItems = computed(() => {
  return isAdmin.value ? [...menuItems, ...adminMenuItems] : menuItems
})
</script>

<template>
  <el-menu
    :default-active="activeMenu"
    :collapse="collapsed"
    :collapse-transition="false"
    router
    class="border-r-0 h-full bg-white dark:bg-slate-900"
    active-text-color="#0891b2"
  >
    <!-- Logo Area -->
    <div class="h-16 flex items-center justify-center border-b border-gray-100 dark:border-slate-800 mb-2 overflow-hidden">
        <div class="flex items-center gap-3">
             <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-cyan-600 to-cyan-400 flex items-center justify-center text-white font-bold shadow-lg shadow-cyan-500/30">
               <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                 <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
               </svg>
             </div>
             <span v-show="!collapsed" class="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-slate-800 to-slate-600 dark:from-white dark:to-slate-300 transition-all duration-300">
               IPS
             </span>
        </div>
    </div>

    <template v-for="item in allMenuItems" :key="item.id">
      <el-sub-menu v-if="item.children" :index="item.id">
        <template #title>
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </template>
        <el-menu-item 
          v-for="child in item.children" 
          :key="child.path" 
          :index="child.path"
        >
          <span>{{ child.label }}</span>
          <span v-if="child.highlight" class="ml-auto px-1.5 py-0.5 text-xs font-medium bg-green-500 text-white rounded transform scale-90">NEW</span>
        </el-menu-item>
      </el-sub-menu>
      
      <el-menu-item v-else :index="item.path">
        <el-icon><component :is="item.icon" /></el-icon>
        <template #title>{{ item.label }}</template>
      </el-menu-item>
    </template>
  </el-menu>
</template>

<style scoped>
:deep(.el-menu) {
  @apply bg-transparent;
}
:deep(.el-menu-item) {
  @apply text-slate-500 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-lg mx-2 my-1 h-10 leading-10;
}
:deep(.el-menu-item.is-active) {
  @apply bg-cyan-50 dark:bg-cyan-900/20 text-cyan-600 dark:text-cyan-400 font-medium;
}
:deep(.el-sub-menu__title) {
  @apply text-slate-500 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-lg mx-2 my-1 h-10 leading-10;
}
</style>
