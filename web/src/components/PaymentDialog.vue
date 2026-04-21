<template>
  <el-dialog
    v-model="visible"
    :title="`收款 — ${customerName}`"
    width="420px"
    :close-on-click-modal="false"
    @closed="reset"
  >
    <div class="pending-hint">
      还欠 <span class="pending-amount">¥ {{ fmt(pendingAmount) }}</span>
    </div>

    <el-form :model="form" label-position="top" size="large">
      <el-form-item label="收款金额">
        <div class="amount-input-row">
          <el-input-number
            v-model="form.amount"
            :min="0.01"
            :max="pendingAmount"
            :precision="2"
            :controls="false"
            placeholder="输入收款金额"
            style="flex: 1"
          />
          <el-button @click="form.amount = pendingAmount">全付</el-button>
        </div>
      </el-form-item>

      <el-form-item label="收款日期">
        <el-date-picker
          v-model="form.payment_date"
          type="date"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          :clearable="false"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="备注">
        <el-input v-model="form.notes" placeholder="选填" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button size="large" @click="visible = false">取消</el-button>
      <el-button type="primary" size="large" :loading="saving" @click="handleSave">确认收款</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { paymentApi } from '@/api/payments'

const props = defineProps<{
  modelValue: boolean
  customerId: string
  customerName: string
  pendingAmount: number
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const today = new Date().toISOString().slice(0, 10)
const saving = ref(false)

const form = ref({
  amount: 0,
  payment_date: today,
  notes: '',
})

function fmt(n: number) {
  return n.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function reset() {
  form.value = { amount: 0, payment_date: today, notes: '' }
}

async function handleSave() {
  if (!form.value.amount || form.value.amount <= 0) {
    ElMessage.warning('请输入收款金额')
    return
  }
  if (form.value.amount > props.pendingAmount) {
    ElMessage.warning('收款金额不能超过当前欠款')
    return
  }
  saving.value = true
  try {
    await paymentApi.create({
      customer_id: props.customerId,
      amount: form.value.amount,
      payment_date: form.value.payment_date,
      notes: form.value.notes,
    })
    ElMessage.success('收款已记录')
    emit('saved')
    visible.value = false
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.pending-hint {
  font-size: 17px;
  color: #606266;
  margin-bottom: 20px;
  text-align: center;
}

.pending-amount {
  font-size: 28px;
  font-weight: 700;
  color: #e6462e;
}

.amount-input-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.quick-amounts {
  margin-top: 8px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
