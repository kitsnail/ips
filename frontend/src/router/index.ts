import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory('/web/'),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/dashboard',
      component: () => import('@/components/Layout.vue'),
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('ips_token')

  if (to.matched.some(record => record.meta.requiresAuth)) {
    if (!token) {
      next({
        path: '/login',
        query: { redirect: to.fullPath },
      })
    } else {
      next()
    }
  } else {
    // Prevent logged-in users from visiting login page
    if (to.path === '/login' && token) {
      next('/dashboard')
    } else {
      next()
    }
  }
})

export default router
