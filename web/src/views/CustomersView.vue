<template>
  <div>
    <!-- 页面标题 + 操作区 -->
    <div class="page-header">
      <el-input
        v-model="search"
        placeholder="搜索客户姓名..."
        clearable
        class="search-input"
        prefix-icon="Search"
      />
      <el-button type="primary" :icon="Plus" @click="openCreate">新增客户</el-button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="center-tip">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
    </div>

    <!-- 客户列表 -->
    <div v-else>
      <!-- 有欠款 -->
      <template v-if="pendingCustomers.length > 0">
        <div class="section-title">欠款未结（{{ pendingCustomers.length }} 人）</div>
        <div class="customer-grid">
          <div
            v-for="c in pendingCustomers"
            :key="c.id"
            class="customer-card pending"
            @click="goDetail(c.id)"
          >
            <div class="customer-name">{{ c.name }}</div>
            <div class="customer-debt amount-pending amount-large">
              ¥ {{ fmt(summaryMap[c.id]?.pending_amount ?? 0) }}
            </div>
            <div class="customer-sub">欠款未结</div>
          </div>
        </div>
      </template>

      <!-- 已结清 -->
      <template v-if="settledCustomers.length > 0">
        <div class="section-title settled-title">
          已结清（{{ settledCustomers.length }} 人）
          <el-button text size="small" @click="showSettled = !showSettled">
            {{ showSettled ? '收起' : '展开' }}
          </el-button>
        </div>
        <div v-if="showSettled" class="customer-grid">
          <div
            v-for="c in settledCustomers"
            :key="c.id"
            class="customer-card settled"
            @click="goDetail(c.id)"
          >
            <div class="customer-name">{{ c.name }}</div>
            <div class="customer-debt amount-settled" style="font-size: 20px; font-weight: 600">
              已结清
            </div>
          </div>
        </div>
      </template>

      <!-- 无数据 -->
      <el-empty v-if="filtered.length === 0 && !loading" description="没有找到客户" />
    </div>

    <!-- 新增/编辑客户 -->
    <CustomerDialog v-model="dialogVisible" :customer="editingCustomer" @saved="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { customerApi, type Customer, type CustomerSummary } from '@/api/customers'
import CustomerDialog from '@/components/CustomerDialog.vue'

const router = useRouter()
const search = ref('')
const loading = ref(false)
const showSettled = ref(false)
const customers = ref<Customer[]>([])
const summaryMap = ref<Record<string, CustomerSummary>>({})
const dialogVisible = ref(false)
const editingCustomer = ref<Customer | null>(null)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return customers.value
  return customers.value.filter(
    (c) =>
      c.name.toLowerCase().includes(q) ||
      c.aliases.some((a) => a.toLowerCase().includes(q)),
  )
})

const pendingCustomers = computed(() =>
  filtered.value.filter((c) => (summaryMap.value[c.id]?.pending_amount ?? 0) > 0),
)
const settledCustomers = computed(() =>
  filtered.value.filter((c) => (summaryMap.value[c.id]?.pending_amount ?? 0) <= 0),
)

function fmt(n: number) {
  return n.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

async function load() {
  loading.value = true
  try {
    customers.value = await customerApi.list()
    // 并发拉取每个客户的汇总
    const results = await Promise.allSettled(
      customers.value.map((c) => customerApi.summary(c.id)),
    )
    results.forEach((r, i) => {
      if (r.status === 'fulfilled' && r.value.length > 0) {
        summaryMap.value[customers.value[i].id] = r.value[0]
      }
    })
  } catch {
    ElMessage.error('加载客户列表失败')
  } finally {
    loading.value = false
  }
}

function goDetail(id: string) {
  router.push(`/customers/${id}`)
}

function openCreate() {
  editingCustomer.value = null
  dialogVisible.value = true
}

onMounted(load)
</script>

<style scoped>
.page-header {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  align-items: center;
}

.search-input {
  flex: 1;
  max-width: 400px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #334155;
  margin: 0 0 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.settled-title {
  margin-top: 28px;
}

.customer-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 8px;
}

.customer-card {
  background: #fff;
  border-radius: 16px;
  padding: 20px 16px;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.2s ease;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -2px rgba(0, 0, 0, 0.03);
}

.customer-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -4px rgba(0, 0, 0, 0.05);
}

.customer-card.pending {
  border-color: #fee2e2;
  background: linear-gradient(135deg, #fff 0%, #fef6f6 100%);
}

.customer-card.settled {
  border-color: #dcfce7;
  background: linear-gradient(135deg, #fff 0%, #f3fcf5 100%);
}

.customer-name {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.customer-debt {
  margin-bottom: 4px;
}

.customer-sub {
  font-size: 13px;
  color: #909399;
}
</style>
