<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

// Breadcrumb item structure
interface BreadcrumbItem {
  label: string
  path: string
}

// Route configuration for breadcrumb labels
const routeLabels: Record<string, string> = {
  '/dashboard': '概览',
  '/tasks': '任务管理',
  '/tasks/create': '创建任务',
  '/scheduled': '定时任务',
  '/scheduled/history': '执行历史',
  '/library': '镜像库',
  '/library/import': '批量导入',
  '/secrets': '仓库认证',
  '/secrets/create': '添加认证',
  '/admin': '系统设置',
  '/admin/users': '用户管理',
  '/admin/settings': '系统配置',
  '/admin/logs': '系统日志',
}

// Parent routes for breadcrumb building
const routeParents: Record<string, { label: string, path: string }> = {
  '/tasks': { label: '任务管理', path: '/tasks' },
  '/scheduled': { label: '定时任务', path: '/scheduled' },
  '/library': { label: '镜像库', path: '/library' },
  '/secrets': { label: '仓库认证', path: '/secrets' },
  '/admin': { label: '系统设置', path: '/admin' },
}

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const path = route.path
  const items: BreadcrumbItem[] = []

  // Remove leading slash and split
  const segments = path.replace(/^\//, '').split('/')

  // Build breadcrumb path segments
  let currentPath = ''
  for (const segment of segments) {
    if (!segment) continue

    currentPath += '/' + segment

    // Check if this is a subpage (has parent)
    const parent = routeParents[currentPath]
    if (parent) {
      // Add parent first if not already added
      if (!items.find(item => item.path === parent.path)) {
        items.push({
          label: parent.label,
          path: parent.path,
        })
      }
      // Add current page
      const currentLabel = routeLabels[currentPath]
      if (currentLabel) {
        items.push({
          label: currentLabel,
          path: currentPath,
        })
      }
    } else {
      // Direct page (no parent)
      const currentLabel = routeLabels[currentPath]
      if (currentLabel && !items.find(item => item.path === currentPath)) {
        items.push({
          label: currentLabel,
          path: currentPath,
        })
      }
    }
  }

  return items
})

const lastItemIndex = computed(() => {
  return breadcrumbs.value.length > 0 ? breadcrumbs.value.length - 1 : -1
})
</script>

<template>
  <nav v-if="breadcrumbs.length > 0" class="breadcrumb">
    <router-link
      v-for="(item, index) in breadcrumbs"
      :key="item.path"
      :to="item.path"
      class="breadcrumb-item"
      :class="{ last: index === lastItemIndex }"
    >
      {{ item.label }}
    </router-link>
  </nav>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.breadcrumb-item {
  color: #64748b;
  text-decoration: none;
  transition: color 0.15s;
  display: flex;
  align-items: center;
}

.breadcrumb-item:not(.last):hover {
  color: #0891b2;
}

.breadcrumb-item:not(.last)::after {
  content: '/';
  margin-left: 8px;
  color: #cbd5e1;
  font-weight: 300;
}

.breadcrumb-item.last {
  color: #0f172a;
  font-weight: 500;
}

.breadcrumb-item.last:hover {
  color: #0f172a;
  cursor: default;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .breadcrumb-item {
    color: #94a3b8;
  }

  .breadcrumb-item:not(.last):hover {
    color: #22d3ee;
  }

  .breadcrumb-item:not(.last)::after {
    color: #475569;
  }

  .breadcrumb-item.last {
    color: #f8fafc;
  }

  .breadcrumb-item.last:hover {
    color: #f8fafc;
  }
}
</style>
