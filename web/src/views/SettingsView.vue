<template>
  <div class="settings-page">
    <div class="settings-head">
      <el-button text class="back-btn" @click="router.push('/')">返回首页</el-button>
      <h2 class="settings-title">设置</h2>
    </div>

    <!-- 微信绑定 -->
    <el-card class="settings-card">
      <template #header>
        <span class="card-title">微信通知</span>
      </template>

      <p class="card-desc">
        绑定微信后，每次有新账目录入时会自动推送消息。
      </p>

      <div v-if="status === 'confirmed'" class="bind-status bind-ok">
        ✅ 微信已绑定
      </div>

      <template v-else>
        <el-button
          type="primary"
          size="large"
          :loading="generating"
          @click="generateQR"
        >
          {{ qrDataUrl ? '重新获取二维码' : '获取绑定二维码' }}
        </el-button>

        <div v-if="qrDataUrl" class="qr-area">
          <div class="qr-panel">
            <img :src="qrDataUrl" class="qr-img" alt="微信二维码" />
          </div>
          <p class="qr-hint">请用微信扫码，然后发送任意消息</p>
          <div v-if="status === 'expired'" class="bind-status bind-expired">
            ⚠️ 二维码已过期，请重新获取
          </div>
          <div v-else-if="status === 'pending'" class="bind-status bind-pending">
            等待扫码...
          </div>
        </div>
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { ElMessage } from 'element-plus'
import { wechatApi } from '@/api/wechat'

type QRStatus = 'idle' | 'pending' | 'confirmed' | 'expired'

const router = useRouter()
const generating = ref(false)
const qrDataUrl = ref('')
const status = ref<QRStatus>('idle')
const currentQRCode = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function generateQR() {
  generating.value = true
  stopPoll()
  status.value = 'idle'
  qrDataUrl.value = ''
  try {
    const res = await wechatApi.generateQR()
    currentQRCode.value = res.qrcode
    qrDataUrl.value = await QRCode.toDataURL(res.img_content, {
      width: 280,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
    status.value = 'pending'
    startPoll()
  } catch {
    ElMessage.error('获取二维码失败')
  } finally {
    generating.value = false
  }
}

function startPoll() {
  pollTimer = setInterval(async () => {
    try {
      const res = await wechatApi.checkStatus(currentQRCode.value)
      if (res.status === 'confirmed') {
        status.value = 'confirmed'
        stopPoll()
        ElMessage.success('微信绑定成功！')
      } else if (res.status === 'expired') {
        status.value = 'expired'
        stopPoll()
      }
    } catch {
      // 静默失败，继续轮询
    }
  }, 3000)
}

onUnmounted(stopPoll)
</script>

<style scoped>
.settings-page {
  max-width: 700px;
}

.settings-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.back-btn {
  border-radius: 8px;
  padding: 0 10px;
  color: #475569;
}

.settings-title {
  font-size: 30px;
  font-weight: 700;
  margin: 0;
  color: #0f172a;
}

.settings-card {
  margin-bottom: 20px;
  border: 1px solid #dbe4ef;
  border-radius: 14px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08), 0 8px 24px rgba(15, 23, 42, 0.06);
}

.card-title {
  font-size: 18px;
  font-weight: 600;
}

.card-desc {
  font-size: 16px;
  color: #475569;
  margin-bottom: 18px;
  line-height: 1.6;
}

.qr-area {
  margin-top: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.qr-panel {
  background: #fff;
  border: 1px solid #dbe4ef;
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08), 0 8px 20px rgba(15, 23, 42, 0.08);
}

.qr-img {
  width: 280px;
  height: 280px;
  display: block;
  border-radius: 10px;
}

.qr-hint {
  font-size: 15px;
  color: #606266;
}

.bind-status {
  font-size: 16px;
  padding: 10px 16px;
  border-radius: 6px;
  margin-top: 8px;
}

.bind-ok {
  background: #f0f9eb;
  color: #67c23a;
  font-weight: 600;
}

.bind-pending {
  background: #f4f4f5;
  color: #909399;
}

.bind-expired {
  background: #fef0f0;
  color: #f56c6c;
}
</style>
