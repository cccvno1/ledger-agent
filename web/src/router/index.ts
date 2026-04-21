import { createRouter, createWebHistory } from 'vue-router'
import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/components/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'customers',
          component: () => import('@/views/CustomersView.vue'),
        },
        {
          path: 'customers/:id',
          name: 'customer-detail',
          component: () => import('@/views/CustomerDetailView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
        },
      ],
    },
  ],
})

// 路由守卫：未登录时探测服务端是否启用了鉴权
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public || auth.isLoggedIn) return

  // 没有存储的 token，用裸 axios 探测（不携带 Authorization 头）
  try {
    await axios.get('/api/v1/customers')
    // 服务端未配置 AUTH_TOKEN，直接进入无鉴权模式
    auth.setNoAuth()
  } catch {
    // 返回 401 或网络错误 → 需要登录
    return { name: 'login' }
  }
})

export default router
