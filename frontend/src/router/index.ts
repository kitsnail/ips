import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory('/web/'),
  routes: [
    {
      path: '/',
      component: () => import('@/components/Layout.vue'),
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        {
          path: 'tasks',
          component: () => import('@/views/TasksView.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to, _, next) => {
  const token = localStorage.getItem('ips_token')
  if (!token && to.path !== '/') {
    next('/')
  } else {
    next()
  }
})

export default router
