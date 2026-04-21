<template>
  <el-drawer
    v-model="visible"
    :title="isEdit ? '修改账目' : `给 ${customerName} 出货`"
    direction="rtl"
    size="780px"
    :close-on-click-modal="false"
    @closed="reset"
  >
    <!-- 日期 -->
    <div class="drawer-date">
      <span class="date-prefix">日期</span>
      <el-date-picker
        v-model="entryDate"
        type="date"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        :clearable="false"
        size="default"
        style="width: 160px"
      />
    </div>

    <!-- 出货单表格 -->
    <div class="table-wrap">
      <div class="table-header">
        <div class="col-name">商品</div>
        <div class="col-qty">数量</div>
        <div class="col-unit">单位</div>
        <div class="col-price">单价</div>
        <div class="col-amount">金额</div>
        <div class="col-notes">备注</div>
        <div class="col-del"></div>
      </div>

      <div v-for="(row, idx) in displayRows" :key="idx" class="table-row">
        <div class="col-name">
          <el-input
            v-model="row.product_name"
            placeholder="商品名"
            size="default"
            class="cell-input"
          />
        </div>
        <div class="col-qty">
          <el-input-number
            v-model="row.quantity"
            :min="0"
            :precision="2"
            :controls="false"
            size="default"
            class="cell-input"
          />
        </div>
        <div class="col-unit">
          <el-input
            v-model="row.unit"
            placeholder=""
            size="default"
            class="cell-input"
          />
        </div>
        <div class="col-price">
          <el-input-number
            v-model="row.unit_price"
            :min="0"
            :precision="2"
            :controls="false"
            size="default"
            class="cell-input"
          />
        </div>
        <div class="col-amount amount-cell">
          ¥ {{ fmt(row.quantity * row.unit_price) }}
        </div>
        <div class="col-notes">
          <el-input
            v-model="row.notes"
            placeholder="选填"
            size="default"
            class="cell-input"
          />
        </div>
        <div class="col-del">
          <el-button
            v-if="!isEdit && rows.length > 1"
            text
            type="danger"
            class="remove-btn"
            @click="removeRow(idx)"
          >×</el-button>
        </div>
      </div>
    </div>

    <template v-if="!isEdit">
      <el-button text :icon="Plus" @click="addRow" class="add-btn">再加一行</el-button>

      <div v-if="validRows.length > 0" class="total-bar">
        共 {{ validRows.length }} 项，合计
        <span class="total-amount">¥ {{ fmt(totalAmount) }}</span>
      </div>
    </template>

    <template #footer>
      <el-button size="large" @click="visible = false">取消</el-button>
      <el-button type="primary" size="large" :loading="saving" @click="handleSave">
        {{ isEdit ? '保存' : '记账' }}
      </el-button>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { entryApi, type Entry } from '@/api/entries'

interface Row {
  product_name: string
  quantity: number
  unit: string
  unit_price: number
  notes: string
}

const props = defineProps<{
  modelValue: boolean
  customerId: string
  customerName: string
  editingEntry?: Entry | null
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const isEdit = computed(() => !!props.editingEntry)

const today = new Date().toISOString().slice(0, 10)
const entryDate = ref(today)
const saving = ref(false)

function newRow(): Row {
  return { product_name: '', quantity: 1, unit: '', unit_price: 0, notes: '' }
}

const rows = ref<Row[]>([newRow()])
const singleRow = ref<Row>(newRow())

const displayRows = computed<Row[]>(() =>
  isEdit.value ? [singleRow.value] : rows.value,
)

function addRow() {
  rows.value.push(newRow())
}
function removeRow(idx: number) {
  rows.value.splice(idx, 1)
}

const validRows = computed(() =>
  rows.value.filter((r) => r.quantity > 0 && r.unit_price > 0),
)
const totalAmount = computed(() =>
  validRows.value.reduce((s, r) => s + r.quantity * r.unit_price, 0),
)

watch(
  () => props.editingEntry,
  (e) => {
    if (e) {
      entryDate.value = e.entry_date
      singleRow.value = {
        product_name: e.product_name,
        quantity: e.quantity || 1,
        unit: e.unit || '',
        unit_price: e.unit_price || 0,
        notes: e.notes || '',
      }
    }
  },
  { immediate: true },
)

function fmt(n: number) {
  return n.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function reset() {
  rows.value = [newRow()]
  entryDate.value = today
  singleRow.value = newRow()
}

function rowToPayload(row: Row) {
  return {
    product_name: row.product_name.trim(),
    quantity: row.quantity,
    unit: row.unit.trim(),
    unit_price: row.unit_price,
    entry_date: entryDate.value,
    notes: row.notes,
  }
}

function rowValid(row: Row): string | null {
  if (!row.product_name.trim()) return '请填商品名'
  if (!(row.quantity > 0)) return '数量须大于 0'
  if (!(row.unit_price > 0)) return '单价须大于 0'
  return null
}

async function handleSave() {
  if (isEdit.value && props.editingEntry) {
    const err = rowValid(singleRow.value)
    if (err) {
      ElMessage.warning(err)
      return
    }
    saving.value = true
    try {
      await entryApi.update(props.editingEntry.id, rowToPayload(singleRow.value))
      ElMessage.success('已更新')
      emit('saved')
      visible.value = false
    } catch {
      ElMessage.error('保存失败')
    } finally {
      saving.value = false
    }
    return
  }

  if (validRows.value.length === 0) {
    ElMessage.warning('请填写商品、数量、单价')
    return
  }
  for (const r of validRows.value) {
    const err = rowValid(r)
    if (err) {
      ElMessage.warning(err)
      return
    }
  }

  saving.value = true
  const errors: string[] = []

  for (const row of validRows.value) {
    try {
      await entryApi.create({
        customer_id: props.customerId,
        customer_name: props.customerName,
        ...rowToPayload(row),
      })
    } catch {
      errors.push(row.product_name || `¥${row.quantity * row.unit_price}`)
    }
  }

  saving.value = false

  if (errors.length === 0) {
    ElMessage.success(validRows.value.length > 1 ? `已记 ${validRows.value.length} 项` : '已记账')
    emit('saved')
    visible.value = false
  } else {
    ElMessage.error(`部分记账失败：${errors.join('、')}`)
  }
}
</script>

<style scoped>
.drawer-date {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  color: #606266;
  font-size: 15px;
}

.date-prefix {
  font-weight: 500;
}

.table-wrap {
  width: 100%;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

.table-header {
  display: flex;
  align-items: center;
  background: #f5f7fa;
  padding: 8px 10px;
  font-size: 13px;
  color: #909399;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
  gap: 6px;
}

.table-row {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  gap: 6px;
  border-bottom: 1px solid #f0f0f0;
}

.table-row:last-child {
  border-bottom: none;
}

.col-name {
  flex: 1.6;
  min-width: 0;
}

.col-qty {
  width: 72px;
  flex-shrink: 0;
}

.col-unit {
  width: 60px;
  flex-shrink: 0;
}

.col-price {
  width: 84px;
  flex-shrink: 0;
}

.col-amount {
  width: 92px;
  flex-shrink: 0;
}

.col-notes {
  flex: 1.2;
  min-width: 0;
}

.col-del {
  width: 28px;
  flex-shrink: 0;
  display: flex;
  justify-content: center;
}

.cell-input {
  width: 100%;
}

.col-qty .cell-input :deep(.el-input__inner),
.col-price .cell-input :deep(.el-input__inner) {
  text-align: right;
}

.amount-cell {
  font-weight: 600;
  color: #e6462e;
  text-align: right;
  font-size: 15px;
}

.remove-btn {
  font-size: 18px;
  padding: 0;
  min-width: 28px;
}

.add-btn {
  margin-top: 10px;
  font-size: 15px;
}

.total-bar {
  margin-top: 10px;
  padding: 10px 14px;
  background: #f5f7fa;
  border-radius: 6px;
  font-size: 15px;
  color: #606266;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
}

.total-amount {
  font-size: 22px;
  font-weight: 700;
  color: #303133;
}
</style>
