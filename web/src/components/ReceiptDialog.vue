<template>
  <el-dialog
    v-model="visible"
    :title="`${customerName} — 对账单`"
    width="680px"
    :close-on-click-modal="false"
  >
    <!-- 时间范围筛选 -->
    <div class="receipt-filter">
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        @change="load"
      />
      <el-button @click="dateRange = null; load()">全部</el-button>
    </div>

    <!-- 对账单主体（供截图） -->
    <div ref="receiptRef" class="receipt-body">
      <div class="receipt-title">{{ customerName }} 对账单</div>
      <div class="receipt-date">打印日期：{{ today }}</div>

      <div v-if="loading" class="center-tip">加载中...</div>

      <template v-else>
        <!-- 出货明细 -->
        <table class="receipt-table">
          <thead>
            <tr>
              <th>日期</th>
              <th>商品</th>
              <th>数量</th>
              <th>单价</th>
              <th>金额</th>
              <th>备注</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in entries" :key="e.id">
              <td>{{ e.entry_date }}</td>
              <td>{{ e.product_name }}</td>
              <td>{{ e.quantity }}{{ e.unit }}</td>
              <td>{{ e.unit_price }}</td>
              <td>{{ fmt(e.amount) }}</td>
              <td class="notes-cell">{{ e.notes || '' }}</td>
            </tr>
            <tr v-if="entries.length === 0">
              <td colspan="6" style="text-align: center; color: #909399">无出货记录</td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="total-row">
              <td colspan="4">出货合计</td>
              <td>¥ {{ fmt(totalEntry) }}</td>
              <td></td>
            </tr>
          </tfoot>
        </table>

        <!-- 收款明细 -->
        <table class="receipt-table" style="margin-top: 16px">
          <thead>
            <tr>
              <th>日期</th>
              <th colspan="3">收款</th>
              <th>金额</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in payments" :key="p.id">
              <td>{{ p.payment_date }}</td>
              <td colspan="3">{{ p.notes || '收款' }}</td>
              <td>{{ fmt(p.amount) }}</td>
            </tr>
            <tr v-if="payments.length === 0">
              <td colspan="5" style="text-align: center; color: #909399">无收款记录</td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="total-row">
              <td colspan="4">收款合计</td>
              <td>¥ {{ fmt(totalPayment) }}</td>
            </tr>
          </tfoot>
        </table>

        <!-- 余欠 -->
        <div class="receipt-balance">
          <span>余欠</span>
          <span class="balance-amount">¥ {{ fmt(Math.max(0, totalEntry - totalPayment)) }}</span>
        </div>
      </template>
    </div>

    <template #footer>
      <el-button size="large" @click="visible = false">关闭</el-button>
      <el-button type="primary" size="large" :loading="screenshotting" @click="takeScreenshot">
        📷 截图保存
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { entryApi, type Entry } from '@/api/entries'
import { paymentApi, type Payment } from '@/api/payments'

const props = defineProps<{
  modelValue: boolean
  customerId: string
  customerName: string
}>()

const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const today = new Date().toLocaleDateString('zh-CN')
const loading = ref(false)
const screenshotting = ref(false)
const dateRange = ref<[string, string] | null>(null)
const entries = ref<Entry[]>([])
const payments = ref<Payment[]>([])
const receiptRef = ref<HTMLElement>()

const totalEntry = computed(() => entries.value.reduce((s, e) => s + e.amount, 0))
const totalPayment = computed(() => payments.value.reduce((s, p) => s + p.amount, 0))

function fmt(n: number) {
  return n.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

async function load() {
  if (!props.customerId) return
  loading.value = true
  try {
    const [entryList, payList] = await Promise.all([
      entryApi.list({
        customer_id: props.customerId,
        date_from: dateRange.value?.[0],
        date_to: dateRange.value?.[1],
      }),
      paymentApi.list(props.customerId),
    ])
    entries.value = entryList
    if (dateRange.value) {
      const from = dateRange.value[0]
      const to = dateRange.value[1]
      payments.value = payList.filter((p) => p.payment_date >= from && p.payment_date <= to)
    } else {
      payments.value = payList
    }
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function takeScreenshot() {
  if (!receiptRef.value) return
  screenshotting.value = true
  try {
    const html2canvas = (await import('html2canvas')).default
    const canvas = await html2canvas(receiptRef.value, { scale: 2, backgroundColor: '#fff' })
    const link = document.createElement('a')
    link.download = `${props.customerName}-对账单-${today}.png`
    link.href = canvas.toDataURL('image/png')
    link.click()
    ElMessage.success('截图已保存')
  } catch {
    ElMessage.error('截图失败，请手动截屏')
  } finally {
    screenshotting.value = false
  }
}

watch(visible, (v) => { if (v) load() })
</script>

<style scoped>
.receipt-filter {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.receipt-body {
  background: #fff;
  padding: 24px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

.receipt-title {
  font-size: 22px;
  font-weight: 700;
  text-align: center;
  margin-bottom: 4px;
  color: #303133;
}

.receipt-date {
  font-size: 14px;
  color: #909399;
  text-align: center;
  margin-bottom: 16px;
}

.receipt-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 15px;
}

.receipt-table th,
.receipt-table td {
  border: 1px solid #dcdfe6;
  padding: 8px 10px;
  text-align: right;
}

.receipt-table th:first-child,
.receipt-table td:first-child {
  text-align: left;
}

.receipt-table th:nth-child(2),
.receipt-table td:nth-child(2) {
  text-align: left;
}

.receipt-table th {
  background: #f5f7fa;
  font-weight: 600;
  color: #606266;
}

.total-row td {
  font-weight: 700;
  background: #fafafa;
}

.receipt-balance {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 16px;
  padding: 12px 10px;
  background: #fff3f0;
  border-radius: 6px;
  font-size: 18px;
  font-weight: 700;
}

.balance-amount {
  font-size: 26px;
  color: #e6462e;
}

.notes-cell {
  color: #909399;
  font-size: 12px;
  text-align: left;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
