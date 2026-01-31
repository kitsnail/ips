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
      ],
    },
  ],
})

export default router
