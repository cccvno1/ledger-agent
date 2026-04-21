<template>
  <div v-if="loading" class="center-tip">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <div v-else>
    <!-- 顶部：返回 + 客户名 -->
    <div class="detail-header">
      <el-button text :icon="ArrowLeft" @click="router.back()">返回</el-button>
      <h2 class="customer-name">{{ customer?.name }}</h2>
    </div>

    <!-- 金额汇总卡 -->
    <el-card class="summary-card">
      <div class="summary-row">
        <div class="summary-item">
          <div class="summary-label">累计出货</div>
          <div class="summary-value">¥ {{ fmt(summary?.total_amount ?? 0) }}</div>
        </div>
        <div class="summary-item">
          <div class="summary-label">累计收款</div>
          <div class="summary-value amount-settled">¥ {{ fmt(summary?.settled_amount ?? 0) }}</div>
        </div>
        <div class="summary-item">
          <div class="summary-label">当前欠款</div>
          <div class="summary-value amount-pending amount-large">
            ¥ {{ fmt(summary?.pending_amount ?? 0) }}
          </div>
        </div>
      </div>
      <div class="summary-tip">账目按全历史累计计算，当前欠款 = 累计出货 - 累计收款</div>
    </el-card>

    <!-- 操作按钮 -->
    <div class="action-bar">
      <el-button type="primary" :icon="Plus" size="large" @click="openEntry">记一笔账</el-button>
      <el-button type="success" :icon="Wallet" size="large" @click="openPayment">收款</el-button>
      <el-button type="warning" size="large" @click="openReceipt">出账单</el-button>
    </div>

    <!-- 流水时间轴 -->
    <div class="timeline-section">
      <div class="filter-bar">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          size="default"
          style="max-width: 320px"
          @change="load"
        />
      </div>

      <el-empty v-if="timelineGroups.length === 0" description="暂无记录" />

      <div v-for="group in timelineGroups" :key="group.date" class="date-group">
        <div class="date-label">{{ group.date }}</div>
        <div v-for="item in group.items" :key="item.id" class="timeline-item">
          <div class="item-icon" :class="item.type === 'entry' ? 'icon-entry' : 'icon-payment'">
            {{ item.type === 'entry' ? '📤' : '💰' }}
          </div>
          <div class="item-body">
            <div class="item-main">
              <span v-if="item.type === 'entry'" class="item-desc">
                {{ item.product_name }}
                <span class="item-qty">{{ item.quantity }}{{ item.unit }}</span>
                <span class="item-unit-price">×{{ item.unit_price }}</span>
              </span>
              <span v-else class="item-desc item-payment-label">收款</span>
              <span class="item-amount" :class="item.type === 'entry' ? '' : 'amount-settled'">
                {{ item.type === 'entry' ? '' : '+' }}¥ {{ fmt(item.amount) }}
              </span>
            </div>
            <div v-if="item.notes" class="item-notes">{{ item.notes }}</div>
          </div>
          <div class="item-actions">
            <el-button text size="small" @click="editItem(item)">改</el-button>
            <el-button text size="small" type="danger" @click="deleteItem(item)">删</el-button>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 弹层 -->
  <EntryDrawer
    v-model="entryDrawerVisible"
    :customer-id="customerId"
    :customer-name="customer?.name ?? ''"
    :editing-entry="editingEntry"
    @saved="load"
  />
  <PaymentDialog
    v-model="paymentDialogVisible"
    :customer-id="customerId"
    :customer-name="customer?.name ?? ''"
    :pending-amount="summary?.pending_amount ?? 0"
    @saved="load"
  />
  <ReceiptDialog
    v-model="receiptDialogVisible"
    :customer-id="customerId"
    :customer-name="customer?.name ?? ''"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Plus, Wallet } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { customerApi, type CustomerSummary } from '@/api/customers'
import { entryApi, type Entry } from '@/api/entries'
import { paymentApi, type Payment } from '@/api/payments'
import EntryDrawer from '@/components/EntryDrawer.vue'
import PaymentDialog from '@/components/PaymentDialog.vue'
import ReceiptDialog from '@/components/ReceiptDialog.vue'

interface TimelineItem {
  id: string
  type: 'entry' | 'payment'
  date: string
  amount: number
  product_name?: string
  quantity?: number
  unit?: string
  unit_price?: number
  notes?: string
  raw: Entry | Payment
}

interface DateGroup {
  date: string
  items: TimelineItem[]
}

const route = useRoute()
const router = useRouter()
const customerId = route.params.id as string

const loading = ref(false)
const customer = ref<{ name: string } | null>(null)
const summary = ref<CustomerSummary | null>(null)
const entries = ref<Entry[]>([])
const payments = ref<Payment[]>([])
const dateRange = ref<[string, string] | null>(null)
const entryDrawerVisible = ref(false)
const paymentDialogVisible = ref(false)
const receiptDialogVisible = ref(false)
const editingEntry = ref<Entry | null>(null)

const timelineGroups = computed<DateGroup[]>(() => {
  const items: TimelineItem[] = [
    ...entries.value.map((e): TimelineItem => ({
      id: `e-${e.id}`,
      type: 'entry',
      date: e.entry_date,
      amount: e.amount,
      product_name: e.product_name,
      quantity: e.quantity,
      unit: e.unit,
      unit_price: e.unit_price,
      notes: e.notes,
      raw: e,
    })),
    ...payments.value.map((p): TimelineItem => ({
      id: `p-${p.id}`,
      type: 'payment',
      date: p.payment_date,
      amount: p.amount,
      notes: p.notes,
      raw: p,
    })),
  ]

  items.sort((a, b) => (a.date < b.date ? 1 : -1))

  const groups: Record<string, TimelineItem[]> = {}
  for (const item of items) {
    if (!groups[item.date]) groups[item.date] = []
    groups[item.date].push(item)
  }

  return Object.entries(groups)
    .sort(([a], [b]) => (a < b ? 1 : -1))
    .map(([date, grpItems]) => ({ date, items: grpItems }))
})

function fmt(n: number) {
  return n.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

async function load() {
  loading.value = true
  try {
    const [cust, summaries, entryList, payList] = await Promise.all([
      customerApi.get(customerId),
      customerApi.summary(customerId),
      entryApi.list({
        customer_id: customerId,
        date_from: dateRange.value?.[0],
        date_to: dateRange.value?.[1],
      }),
      paymentApi.list(customerId),
    ])
    customer.value = cust
    summary.value = summaries[0] ?? null
    entries.value = entryList
    payments.value = payList
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function openEntry() {
  editingEntry.value = null
  entryDrawerVisible.value = true
}

function openPayment() {
  if ((summary.value?.pending_amount ?? 0) <= 0) {
    ElMessage.info('当前没有欠款，不需要收款')
    return
  }
  paymentDialogVisible.value = true
}

function openReceipt() {
  receiptDialogVisible.value = true
}

function editItem(item: TimelineItem) {
  if (item.type === 'entry') {
    editingEntry.value = item.raw as Entry
    entryDrawerVisible.value = true
  }
  // payment 编辑暂不支持（直接删了重录）
}

async function deleteItem(item: TimelineItem) {
  const label = item.type === 'entry'
    ? `${item.product_name} ¥${fmt(item.amount)}`
    : `收款 ¥${fmt(item.amount)}`
  await ElMessageBox.confirm(`确认删除「${label}」？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
  })
  try {
    if (item.type === 'entry') {
      await entryApi.delete((item.raw as Entry).id)
    }
    // payment 无删除接口，暂跳过
    ElMessage.success('已删除')
    await load()
  } catch {
    ElMessage.error('删除失败')
  }
}

onMounted(load)
</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.customer-name {
  font-size: 26px;
  font-weight: 700;
  color: #303133;
  margin: 0;
}

.summary-card {
  margin-bottom: 20px;
}

.summary-row {
  display: flex;
  gap: 0;
}

.summary-tip {
  margin-top: 14px;
  font-size: 14px;
  color: #64748b;
  text-align: center;
}

.summary-item {
  flex: 1;
  text-align: center;
  padding: 8px 0;
  border-right: 1px solid #e4e7ed;
}

.summary-item:last-child {
  border-right: none;
}

.summary-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 6px;
}

.summary-value {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.action-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.timeline-section {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
}

.filter-bar {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.date-group {
  margin-bottom: 16px;
}

.date-label {
  font-size: 15px;
  font-weight: 600;
  color: #909399;
  padding: 4px 0 8px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 8px;
}

.timeline-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid #f9f9f9;
}

.timeline-item:last-child {
  border-bottom: none;
}

.item-icon {
  font-size: 20px;
  width: 28px;
  flex-shrink: 0;
  text-align: center;
}

.item-body {
  flex: 1;
  min-width: 0;
}

.item-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.item-desc {
  font-size: 17px;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-payment-label {
  color: #67c23a;
  font-weight: 600;
}

.item-qty {
  color: #606266;
  margin-left: 6px;
}

.item-unit-price {
  color: #909399;
  font-size: 14px;
  margin-left: 4px;
}

.item-amount {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
}

.item-notes {
  font-size: 13px;
  color: #909399;
  margin-top: 2px;
}

.item-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
</style>
