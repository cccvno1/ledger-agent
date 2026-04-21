<template>
  <el-container class="layout">
    <!-- 顶部导航 -->
    <el-header class="layout-header">
      <div class="header-left" @click="router.push('/')">
        <span class="app-title">记账本</span>
        <span class="app-subtitle">日常流水</span>
      </div>
      <div class="header-right">
        <el-button class="icon-btn" text @click="router.push('/')" title="主页">
          <el-icon :size="20"><House /></el-icon>
        </el-button>
        <el-button class="icon-btn" text @click="router.push('/settings')" title="设置">
          <el-icon :size="22"><Setting /></el-icon>
        </el-button>
        <el-button v-if="!auth.noAuth" class="icon-btn" text @click="handleLogout" title="退出">
          <el-icon :size="22"><SwitchButton /></el-icon>
        </el-button>
      </div>
    </el-header>

    <!-- 主内容 -->
    <el-main class="layout-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { House, Setting, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

async function handleLogout() {
  await ElMessageBox.confirm('确认退出登录？', '提示', {
    confirmButtonText: '退出',
    cancelButtonText: '取消',
    type: 'warning',
  })
  auth.clear()
  router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
}

.layout-header {
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 60px;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08), 0 8px 20px rgba(15, 23, 42, 0.06);
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 10px;
  cursor: pointer;
}

.app-title {
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: 0.5px;
}

.app-subtitle {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

.header-right {
  display: flex;
  gap: 4px;
  align-items: center;
}

.icon-btn {
  width: 40px;
  height: 40px;
  border-radius: 10px;
}

.icon-btn:hover {
  background: #e2e8f0;
}

.layout-main {
  padding: 24px;
  max-width: 1100px;
  margin: 0 auto;
  width: 100%;
}

@media (max-width: 768px) {
  .layout-main {
    padding: 16px;
  }
}
</style>
