<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

// Map of route names/paths to friendly labels where meta.title isn't available
const labelMap: Record<string, string> = {
  '/dashboard': '概览',
  '/tasks': '任务管理',
  '/tasks/create': '创建任务',
  '/scheduled': '定时任务',
  '/library': '镜像库',
  '/library/import': '批量导入',
  '/secrets': '仓库认证',
  '/secrets/create': '添加认证',
  '/admin': '系统设置',
  '/admin/users': '用户管理',
  '/admin/settings': '系统配置',
  '/admin/logs': '系统日志',
}

const breadcrumbs = computed(() => {

  const items: Array<{ label: string, path: string }> = []

  // If we are essentially at the root (just dashboard), show nothing or just Dashboard
  // But generally matched array gives us the hierarchy.
  
  // Custom handling: We want to show the full path based on the URL segments because
  // route.matched might be nested in a way that doesn't strictly follow the menu hierarchy visually
  // OR we can stick to the labelMap which seemed to define the desired hierarchy.
  
  // Let's use the labelMap approach essentially as "Source of Truth" for labels, 
  // but generate the path segments dynamically to avoid the "double add" bug.
  // The previous bug was adding Parent AND Current for the same segment.
  
  const path = route.path
  const segments = path.replace(/^\//, '').split('/')
  
  let currentPath = ''
  for (const segment of segments) {
    if (!segment) continue
    currentPath += '/' + segment
    
    // Only add if we have a label for this path
    if (labelMap[currentPath]) {
      items.push({
        label: labelMap[currentPath]!,
        path: currentPath
      })
    }
  }

  // Handle root/dashboard case if needed, but based on the map, /dashboard is handled.
  
  return items
})

const lastItemIndex = computed(() => {
  return breadcrumbs.value.length > 0 ? breadcrumbs.value.length - 1 : -1
})
</script>

<template>
  <nav v-if="breadcrumbs.length > 0" class="flex items-center gap-2 text-sm">
    <template v-for="(item, index) in breadcrumbs" :key="item.path">
       <router-link
        v-if="index !== lastItemIndex"
        :to="item.path"
        class="text-slate-500 hover:text-cyan-600 dark:text-slate-400 dark:hover:text-cyan-400 transition-colors flex items-center"
      >
        {{ item.label }}
      </router-link>
      <span v-else class="text-slate-900 dark:text-slate-200 font-medium">
        {{ item.label }}
      </span>
      
      <!-- Separator -->
      <span v-if="index !== lastItemIndex" class="text-slate-300 dark:text-slate-600 font-light">/</span>
    </template>
  </nav>
</template>
