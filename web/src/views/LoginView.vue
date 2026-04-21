<template>
  <div class="login-page">
    <el-card class="login-card">
      <div class="login-logo">📒</div>
      <h1 class="login-title">记账本</h1>
      <p class="login-desc">请输入访问密码登录</p>

      <el-form @submit.prevent="handleLogin">
        <el-form-item>
          <el-input
            v-model="token"
            type="password"
            placeholder="输入访问密码"
            size="large"
            show-password
            @keyup.enter="handleLogin"
            autofocus
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            style="width: 100%"
            @click="handleLogin"
          >
            进入
          </el-button>
        </el-form-item>
      </el-form>

      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import axios from 'axios'

const router = useRouter()
const auth = useAuthStore()

const token = ref('')
const loading = ref(false)
const errorMsg = ref('')

async function handleLogin() {
  if (!token.value.trim()) {
    errorMsg.value = '请输入访问密码'
    return
  }
  loading.value = true
  errorMsg.value = ''

  try {
    // 用 health 接口验证 token（携带 Bearer 头）
    await axios.get('/api/v1/customers', {
      headers: { Authorization: `Bearer ${token.value.trim()}` },
    })
    auth.setToken(token.value.trim())
    router.push('/')
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      errorMsg.value = '密码不正确，请重试'
    } else {
      errorMsg.value = '连接服务器失败，请检查网络'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  background: #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  padding: 16px 8px;
  border-radius: 12px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
}

.login-logo {
  font-size: 56px;
  text-align: center;
  margin-bottom: 8px;
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  text-align: center;
  color: #303133;
  margin: 0 0 8px;
}

.login-desc {
  font-size: 16px;
  color: #909399;
  text-align: center;
  margin: 0 0 28px;
}

.error-msg {
  color: #e6462e;
  font-size: 15px;
  text-align: center;
  margin-top: 8px;
}
</style>
