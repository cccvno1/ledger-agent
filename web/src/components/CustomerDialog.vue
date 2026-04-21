<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑客户' : '新增客户'"
    width="400px"
    :close-on-click-modal="false"
    @closed="reset"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="80px" size="large">
      <el-form-item label="客户姓名" prop="name">
        <el-input v-model="form.name" placeholder="输入姓名" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button size="large" @click="visible = false">取消</el-button>
      <el-button type="primary" size="large" :loading="saving" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import type { Customer } from '@/api/customers'
import { customerApi } from '@/api/customers'

const props = defineProps<{
  modelValue: boolean
  customer?: Customer | null
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const isEdit = computed(() => !!props.customer)

const formRef = ref<FormInstance>()
const saving = ref(false)
const form = ref({ name: '' })

const rules: FormRules = {
  name: [{ required: true, message: '请输入客户姓名', trigger: 'blur' }],
}

watch(
  () => props.customer,
  (c) => {
    form.value.name = c?.name ?? ''
  },
)

function reset() {
  form.value.name = ''
  formRef.value?.resetFields()
}

async function handleSave() {
  await formRef.value?.validate()
  saving.value = true
  try {
    await customerApi.create(form.value.name.trim())
    ElMessage.success('客户已添加')
    emit('saved')
    visible.value = false
  } catch {
    ElMessage.error('添加失败，请重试')
  } finally {
    saving.value = false
  }
}
</script>
