<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import {
  DataBoard,
  List,
  Clock,
  Grid,
  Lock,
  Setting,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const { isAdmin } = useAuth()

const collapsed = ref(false)
const hoveredItem = ref<string | null>(null)
const expandedMenus = ref<Set<string>>(new Set())

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

// Menu structure
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
      {
        id: 'tasks-list',
        label: '全部任务',
        path: '/tasks',
      },
      {
        id: 'tasks-create',
        label: '创建任务',
        path: '/tasks/create',
        highlight: true,
      },
    ],
  },
  {
    id: 'scheduled',
    icon: Clock,
    label: '定时任务',
    path: '/scheduled',
    children: [
      {
        id: 'scheduled-list',
        label: '任务列表',
        path: '/scheduled',
      },
      {
        id: 'scheduled-history',
        label: '执行历史',
        path: '/scheduled/history',
      },
    ],
  },
  {
    id: 'library',
    icon: Grid,
    label: '镜像库',
    path: '/library',
    children: [
      {
        id: 'library-list',
        label: '镜像管理',
        path: '/library',
      },
      {
        id: 'library-import',
        label: '批量导入',
        path: '/library/import',
      },
    ],
  },
  {
    id: 'secrets',
    icon: Lock,
    label: '仓库认证',
    path: '/secrets',
    children: [
      {
        id: 'secrets-list',
        label: '认证列表',
        path: '/secrets',
      },
      {
        id: 'secrets-create',
        label: '添加认证',
        path: '/secrets/create',
      },
    ],
  },
] as MenuItem[]

const adminMenuItems = [
  {
    id: 'admin',
    icon: Setting,
    label: '系统设置',
    path: '/admin',
    children: [
      {
        id: 'admin-users',
        label: '用户管理',
        path: '/admin/users',
      },
      {
        id: 'admin-settings',
        label: '系统配置',
        path: '/admin/settings',
      },
      {
        id: 'admin-logs',
        label: '系统日志',
        path: '/admin/logs',
      },
    ] as ChildMenuItem[],
  },
] as MenuItem[]

const allMenuItems = computed(() => {
  if (isAdmin.value) {
    return [...menuItems, ...adminMenuItems]
  }
  return menuItems
})

const activePath = computed(() => route.path)

const isMenuActive = (path: string): boolean => {
  return activePath.value === path || activePath.value.startsWith(path + '/')
}

const toggleCollapse = () => {
  collapsed.value = !collapsed.value
  localStorage.setItem('sidebar_collapsed', String(collapsed.value))
}

const toggleMenu = (menuId: string) => {
  if (expandedMenus.value.has(menuId)) {
    expandedMenus.value.delete(menuId)
  } else {
    expandedMenus.value.add(menuId)
  }
}

const handleNavigation = (path: string) => {
  router.push(path)
}

const handleMouseEnter = (menuId: string) => {
  if (collapsed.value) {
    hoveredItem.value = menuId
  }
}

const handleMouseLeave = () => {
  hoveredItem.value = null
}

const restoreCollapsedState = () => {
  const saved = localStorage.getItem('sidebar_collapsed')
  if (saved !== null) {
    collapsed.value = saved === 'true'
  }
}

const handleResize = () => {
  if (window.innerWidth < 1024) {
    collapsed.value = true
  }
}

onMounted(() => {
  restoreCollapsedState()
  window.addEventListener('resize', handleResize)
  handleResize() // Initial check
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

defineExpose({
  collapsed,
  toggleCollapse,
})
</script>

<template>
  <aside
    class="sidebar"
    :class="{ collapsed }"
    @mouseleave="handleMouseLeave"
  >
    <!-- Logo Area -->
    <div class="sidebar-header">
      <div class="logo">
        <div class="logo-icon">
          <svg viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect width="32" height="32" rx="6" fill="url(#gradient)" />
            <defs>
              <linearGradient id="gradient" x1="0" y1="0" x2="32" y2="32">
                <stop offset="0%" stop-color="#0891b2" />
                <stop offset="100%" stop-color="#22d3ee" />
              </linearGradient>
            </defs>
            <path d="M8 10h16M8 16h16M8 22h8" stroke="white" stroke-width="2" stroke-linecap="round" />
          </svg>
        </div>
        <span v-show="!collapsed" class="logo-text">IPS</span>
      </div>
      <button
        class="collapse-btn"
        @click="toggleCollapse"
        :title="collapsed ? '展开侧边栏' : '折叠侧边栏'"
      >
        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" class="icon">
          <path
            :d="collapsed ? 'M13 7l5 5-5 5M6 12h12' : 'M11 7l-5 5 5 5M18 12H6'"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>

    <!-- Navigation Menu -->
    <nav class="sidebar-nav">
      <template v-for="item in allMenuItems" :key="item.id">
        <!-- Menu with children -->
        <div v-if="item.children" class="menu-group">
          <button
            class="menu-item parent-item"
            :class="{
              active: isMenuActive(item.path),
              expanded: expandedMenus.has(item.id)
            }"
            @click="toggleMenu(item.id)"
            @mouseenter="handleMouseEnter(item.id)"
            :title="item.label"
          >
            <component :is="item.icon" class="menu-icon" />
            <span v-show="!collapsed" class="menu-label">{{ item.label }}</span>
            <svg
              v-show="!collapsed"
              viewBox="0 0 24 24"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              class="arrow-icon"
              :class="{ expanded: expandedMenus.has(item.id) }"
            >
              <path
                d="M6 9l6 6 6-6"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>

            <!-- Tooltip for collapsed state -->
            <div v-if="collapsed" class="tooltip">
              {{ item.label }}
            </div>
          </button>

          <!-- Submenu -->
          <div v-if="!collapsed && expandedMenus.has(item.id)" class="submenu">
            <button
              v-for="child in item.children"
              :key="child.id"
              class="menu-item child-item"
              :class="{
                active: activePath === (child as any).path,
                highlight: (child as any).highlight
              }"
              @click="handleNavigation((child as any).path)"
            >
              <span class="menu-dot"></span>
              <span class="menu-label">{{ (child as any).label }}</span>
              <span v-if="(child as any).highlight" class="badge">新建</span>
            </button>
          </div>

          <!-- Flyout for collapsed state -->
          <div
            v-if="collapsed && hoveredItem === item.id"
            class="flyout-menu"
          >
            <button
              v-for="child in item.children"
              :key="child.id"
              class="flyout-item"
              :class="{ active: activePath === (child as any).path }"
              @click="handleNavigation((child as any).path)"
            >
              {{ (child as any).label }}
              <span v-if="(child as any).highlight" class="badge">新建</span>
            </button>
          </div>
        </div>

        <!-- Simple menu item (no children) -->
        <button
          v-else
          class="menu-item"
          :class="{ active: isMenuActive(item.path) }"
          @click="handleNavigation(item.path)"
          @mouseenter="handleMouseEnter(item.id)"
          :title="item.label"
        >
          <component :is="item.icon" class="menu-icon" />
          <span v-show="!collapsed" class="menu-label">{{ item.label }}</span>

          <!-- Tooltip for collapsed state -->
          <div v-if="collapsed" class="tooltip">
            {{ item.label }}
          </div>
        </button>
      </template>
    </nav>

    <!-- Footer -->
    <div class="sidebar-footer">
      <div v-if="!collapsed" class="footer-info">
        <div class="version">IPS v1.0.0</div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  width: 260px;
  height: 100vh;
  background: #ffffff;
  border-right: 1px solid #e2e8f0;
  transition: width 300ms cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.sidebar.collapsed {
  width: 64px;
}

/* Header */
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 20px;
  border-bottom: 1px solid #e2e8f0;
  flex-shrink: 0;
}

.sidebar.collapsed .sidebar-header {
  justify-content: center;
  padding: 0 20px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.logo-icon {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
}

.logo-icon svg {
  width: 100%;
  height: 100%;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  opacity: 1;
  transition: opacity 0.2s;
}

.sidebar.collapsed .logo-text {
  opacity: 0;
  width: 0;
}

.collapse-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: #64748b;
  transition: all 0.2s;
  flex-shrink: 0;
}

.collapse-btn:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.collapse-btn .icon {
  width: 20px;
  height: 20px;
}

/* Navigation */
.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 12px 8px;
}

.sidebar-nav::-webkit-scrollbar {
  width: 6px;
}

.sidebar-nav::-webkit-scrollbar-track {
  background: transparent;
}

.sidebar-nav::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}

.sidebar-nav::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

.menu-group {
  margin-bottom: 4px;
}

.menu-item {
  width: 100%;
  display: flex;
  align-items: center;
  padding: 0 12px;
  height: 40px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 8px;
  color: #64748b;
  font-size: 14px;
  transition: all 0.15s ease;
  position: relative;
  text-decoration: none;
}

.menu-item:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.menu-item.active {
  background: #e0f2fe;
  color: #0891b2;
  font-weight: 500;
}

.menu-item.parent-item {
  justify-content: space-between;
}

.menu-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  stroke-width: 1.5;
}

.menu-label {
  margin-left: 12px;
  white-space: nowrap;
}

.menu-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
}

.menu-item.active .menu-dot {
  background: #0891b2;
}

.arrow-icon {
  width: 16px;
  height: 16px;
  color: #94a3b8;
  transition: transform 0.2s;
}

.arrow-icon.expanded {
  transform: rotate(90deg);
}

/* Submenu */
.submenu {
  margin-left: 44px;
  margin-top: 4px;
  margin-bottom: 8px;
  border-left: 2px solid #e2e8f0;
  padding-left: 8px;
}

.child-item {
  padding-left: 8px;
  height: 36px;
  font-size: 13px;
}

.child-item.highlight {
  color: #0891b2;
}

.badge {
  margin-left: auto;
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 500;
  background: #22c55e;
  color: white;
  border-radius: 4px;
  white-space: nowrap;
}

/* Flyout menu (collapsed state) */
.flyout-menu {
  position: fixed;
  left: 68px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  padding: 8px;
  min-width: 200px;
  z-index: 1001;
}

.flyout-item {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  font-size: 14px;
  color: #64748b;
  transition: all 0.15s ease;
}

.flyout-item:hover {
  background: #f1f5f9;
  color: #0f172a;
}

.flyout-item.active {
  background: #e0f2fe;
  color: #0891b2;
  font-weight: 500;
}

/* Tooltip */
.tooltip {
  position: absolute;
  left: 100%;
  top: 50%;
  transform: translateY(-50%);
  margin-left: 12px;
  padding: 6px 10px;
  background: #1e293b;
  color: white;
  font-size: 13px;
  border-radius: 6px;
  white-space: nowrap;
  z-index: 1002;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.2s;
}

.menu-item:hover .tooltip {
  opacity: 1;
}

/* Footer */
.sidebar-footer {
  padding: 12px 20px;
  border-top: 1px solid #e2e8f0;
  flex-shrink: 0;
}

.footer-info {
  text-align: center;
}

.version {
  font-size: 12px;
  color: #94a3b8;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .sidebar {
    background: #0f172a;
    border-right-color: #1e293b;
  }

  .sidebar-header {
    border-bottom-color: #1e293b;
  }

  .logo-text {
    color: #f8fafc;
  }

  .collapse-btn {
    color: #94a3b8;
  }

  .collapse-btn:hover {
    background: #1e293b;
    color: #f8fafc;
  }

  .menu-item {
    color: #cbd5e1;
  }

  .menu-item:hover {
    background: #1e293b;
    color: #f8fafc;
  }

  .menu-item.active {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .menu-dot {
    background: #475569;
  }

  .menu-item.active .menu-dot {
    background: #22d3ee;
  }

  .arrow-icon {
    color: #64748b;
  }

  .submenu {
    border-left-color: #1e293b;
  }

  .flyout-menu {
    background: #1e293b;
    border-color: #334155;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5);
  }

  .flyout-item {
    color: #cbd5e1;
  }

  .flyout-item:hover {
    background: #334155;
    color: #f8fafc;
  }

  .flyout-item.active {
    background: rgba(34, 211, 238, 0.15);
    color: #22d3ee;
  }

  .sidebar-footer {
    border-top-color: #1e293b;
  }

  .version {
    color: #64748b;
  }
}
</style>
