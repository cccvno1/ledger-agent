import axios from 'axios'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// 请求拦截：自动携带 Bearer Token
client.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

// 响应拦截：统一解包 {success, data} 外层，401 跳登录页
client.interceptors.response.use(
  (res) => {
    // Backend wraps all responses as {success: true, data: ...}
    // Unwrap so callers receive the inner data directly.
    if (res.data && typeof res.data === 'object' && 'success' in res.data) {
      res.data = res.data.data
    }
    return res
  },
  (err) => {
    if (err.response?.status === 401) {
      const auth = useAuthStore()
      auth.clear()
      router.push('/login')
    }
    return Promise.reject(err)
  }
)

export default client
