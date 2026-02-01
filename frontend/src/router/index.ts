import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory('/web/'),
  routes: [
    {
      path: '/',
      component: () => import('@/App.vue'),
    },
    {
      path: '/dashboard',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/DashboardView.vue'),
        },
      ],
    },
    {
      path: '/tasks',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/TasksView.vue'),
        },
        {
          path: 'create',
          component: () => import('@/views/TasksCreateView.vue'),
        },
      ],
    },
    {
      path: '/scheduled',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/ScheduledTasksView.vue'),
        },
        {
          path: 'history',
          component: () => import('@/views/ScheduledHistoryView.vue'),
        },
      ],
    },
    {
      path: '/library',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/LibraryView.vue'),
        },
        {
          path: 'import',
          component: () => import('@/views/LibraryImportView.vue'),
        },
      ],
    },
    {
      path: '/secrets',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/SecretsView.vue'),
        },
        {
          path: 'create',
          component: () => import('@/views/SecretsCreateView.vue'),
        },
      ],
    },
    {
      path: '/admin',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          component: () => import('@/views/AdminView.vue'),
        },
        {
          path: 'users',
          component: () => import('@/views/AdminUsersView.vue'),
        },
        {
          path: 'settings',
          component: () => import('@/views/AdminSettingsView.vue'),
        },
        {
          path: 'logs',
          component: () => import('@/views/AdminLogsView.vue'),
        },
      ],
    },
  ],
})

export default router
