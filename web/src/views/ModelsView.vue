<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import dayjs from 'dayjs'
import {
  listAIModels,
  createAIModel,
  updateAIModel,
  deleteAIModel,
  PRICING_TYPE_LABELS,
  formatPricePerMillion,
  type CreateAIModelRequest,
  type UpdateAIModelRequest,
} from '@/api/model'
import type { AIModel, ModelPricing, ModelPricingType } from '@/types'
import {
  fadeUp,
  tableRow,
} from '@/utils/motion'
import AppButton from '@/components/ui/AppButton.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppBadge from '@/components/ui/AppBadge.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppFormField from '@/components/ui/AppFormField.vue'

// List data
const models = ref<AIModel[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// Filters
const keyword = ref('')

// Dialog
const dialogOpen = ref(false)
const editingModel = ref<AIModel | null>(null)
const formLoading = ref(false)
const formError = ref('')

// Form data
const formData = ref<CreateAIModelRequest>({
  name: '',
  description: '',
  pricing: {},
  pricingType: 'tokens',
})

// Pricing form (separate for easier manipulation)
const pricingForm = ref<ModelPricing>({
  inputPrice: undefined,
  outputPrice: undefined,
  cacheCreatePrice: undefined,
  cacheReadPrice: undefined,
  perRequestPrice: undefined,
})

// Delete confirmation
const deletingModel = ref<AIModel | null>(null)
const deleteLoading = ref(false)

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
const statsEnabled = computed(() => models.value.filter(m => m.isEnabled).length)
const statsDisabled = computed(() => models.value.filter(m => !m.isEnabled).length)

async function loadModels() {
  loading.value = true
  try {
    const result = await listAIModels({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value || undefined,
    })
    models.value = result.items
    total.value = result.total
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingModel.value = null
  formData.value = {
    name: '',
    description: '',
    pricing: {},
    pricingType: 'tokens',
  }
  pricingForm.value = {
    inputPrice: undefined,
    outputPrice: undefined,
    cacheCreatePrice: undefined,
    cacheReadPrice: undefined,
    perRequestPrice: undefined,
  }
  formError.value = ''
  dialogOpen.value = true
}

function openEditDialog(model: AIModel) {
  editingModel.value = model
  formData.value = {
    name: model.name,
    description: model.description || '',
    pricing: { ...model.pricing },
    pricingType: model.pricingType,
  }
  pricingForm.value = {
    inputPrice: model.pricing.inputPrice,
    outputPrice: model.pricing.outputPrice,
    cacheCreatePrice: model.pricing.cacheCreatePrice,
    cacheReadPrice: model.pricing.cacheReadPrice,
    perRequestPrice: model.pricing.perRequestPrice,
  }
  formError.value = ''
  dialogOpen.value = true
}

function updatePricingFromForm() {
  formData.value.pricing = {
    inputPrice: pricingForm.value.inputPrice || undefined,
    outputPrice: pricingForm.value.outputPrice || undefined,
    cacheCreatePrice: pricingForm.value.cacheCreatePrice || undefined,
    cacheReadPrice: pricingForm.value.cacheReadPrice || undefined,
    perRequestPrice: pricingForm.value.perRequestPrice || undefined,
  }
}

async function handleFormSubmit() {
  if (!formData.value.name) {
    formError.value = '请填写模型名称'
    return
  }

  updatePricingFromForm()

  formLoading.value = true
  formError.value = ''

  try {
    if (editingModel.value) {
      const updateData: UpdateAIModelRequest = {
        name: formData.value.name,
        description: formData.value.description || undefined,
        pricing: formData.value.pricing,
        pricingType: formData.value.pricingType,
      }
      await updateAIModel(editingModel.value.id, updateData)
    } else {
      await createAIModel(formData.value)
    }
    dialogOpen.value = false
    await loadModels()
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '操作失败'
    formError.value = message
  } finally {
    formLoading.value = false
  }
}

async function handleDelete() {
  if (!deletingModel.value) return

  deleteLoading.value = true
  try {
    await deleteAIModel(deletingModel.value.id)
    deletingModel.value = null
    await loadModels()
  } catch {
    // ignore
  } finally {
    deleteLoading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadModels()
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadModels()
}

onMounted(() => {
  loadModels()
})
</script>

<template>
  <div v-motion="fadeUp" class="max-w-[1400px]">
    <!-- Page Header -->
    <header class="flex justify-between items-start mb-6">
      <div class="flex flex-col gap-3">
        <span class="text-2xl font-semibold tracking-tight text-text-primary">模型管理</span>
        <div class="flex gap-6">
          <div class="flex items-center gap-2 text-sm">
            <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
            <span class="font-semibold text-text-primary">{{ statsEnabled }}</span>
            <span class="text-text-muted">已启用</span>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <span class="w-1.5 h-1.5 rounded-full bg-text-muted"></span>
            <span class="font-semibold text-text-primary">{{ statsDisabled }}</span>
            <span class="text-text-muted">已禁用</span>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <span class="font-semibold text-text-primary">{{ total }}</span>
            <span class="text-text-muted">总计</span>
          </div>
        </div>
      </div>
      <AppButton variant="primary" @click="openCreateDialog">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        新建模型
      </AppButton>
    </header>

    <!-- Filter Bar -->
    <div class="flex gap-3 mb-4 p-4 bg-bg-card rounded-lg border border-border">
      <div class="flex gap-2 flex-1">
        <AppInput
          v-model="keyword"
          type="text"
          placeholder="搜索模型名称或描述..."
          class="flex-1 max-w-[400px]"
          @keyup.enter="handleSearch"
        />
      </div>
      <AppButton variant="secondary" @click="handleSearch">搜索</AppButton>
    </div>

    <!-- Data Table -->
    <div class="bg-bg-card rounded-lg border border-border overflow-hidden">
      <table class="w-full border-collapse">
        <thead>
          <tr>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">模型名称</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">计费方式</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">定价信息</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">创建时间</th>
            <th class="px-4 py-3.5 text-right text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="5" class="px-4 py-12 text-center text-text-muted">
              <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin mx-auto mb-2"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="models.length === 0">
            <td colspan="5" class="px-4 py-12 text-center text-text-muted">
              <div class="text-4xl mb-2 opacity-30 text-primary">&#9672;</div>
              <span>暂无模型数据</span>
            </td>
          </tr>
          <tr v-else v-for="(model, index) in models" :key="model.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="border-b border-border transition-colors duration-150 hover:bg-bg-secondary">
            <td class="px-4 py-4 align-middle">
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-lg flex items-center justify-center bg-gradient-to-br from-primary/15 to-primary/10 text-primary">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                    <path d="M2 17l10 5 10-5"/>
                    <path d="M2 12l10 5 10-5"/>
                  </svg>
                </div>
                <div class="flex flex-col gap-0.5">
                  <span class="font-medium text-text-primary font-mono text-sm">{{ model.name }}</span>
                  <span v-if="model.description" class="text-xs text-text-muted max-w-[300px] overflow-hidden text-ellipsis whitespace-nowrap">{{ model.description }}</span>
                </div>
              </div>
            </td>
            <td class="px-4 py-4 align-middle">
              <AppBadge :variant="model.pricingType === 'tokens' ? 'success' : 'warning'">
                {{ PRICING_TYPE_LABELS[model.pricingType] }}
              </AppBadge>
            </td>
            <td class="px-4 py-4 align-middle">
              <div class="flex flex-col gap-1">
                <template v-if="model.pricingType === 'tokens'">
                  <div class="flex items-center gap-2">
                    <span class="text-[11px] text-text-muted uppercase tracking-wider">输入</span>
                    <span class="text-[13px] font-medium text-text-primary font-mono">{{ formatPricePerMillion(model.pricing.inputPrice) }}</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="text-[11px] text-text-muted uppercase tracking-wider">输出</span>
                    <span class="text-[13px] font-medium text-text-primary font-mono">{{ formatPricePerMillion(model.pricing.outputPrice) }}</span>
                  </div>
                </template>
                <template v-else>
                  <span class="text-[13px] font-medium text-text-primary font-mono">{{ model.pricing.perRequestPrice ? `$${model.pricing.perRequestPrice.toFixed(4)}` : '-' }}</span>
                  <span class="text-[11px] text-text-muted uppercase tracking-wider">/ 次</span>
                </template>
              </div>
            </td>
            <td class="px-4 py-4 align-middle">
              <span class="text-sm text-text-secondary">{{ dayjs(model.createdAt).format('YYYY-MM-DD') }}</span>
            </td>
            <td class="px-4 py-4 align-middle text-right">
              <div class="flex justify-end gap-2">
                <AppButton variant="ghost" size="sm" @click="openEditDialog(model)">编辑</AppButton>
                <AppButton variant="ghost" size="sm" class="hover:text-danger hover:bg-danger-light" @click="deletingModel = model">删除</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex justify-between items-center px-4 py-3.5 border-t border-border">
        <span class="text-[13px] text-text-muted">共 {{ total }} 条</span>
        <div class="flex items-center gap-3">
          <AppButton variant="secondary" size="sm" :disabled="page === 1" @click="handlePageChange(page - 1)">
            &#8592;
          </AppButton>
          <span class="text-[13px] text-text-secondary">{{ page }} / {{ totalPages }}</span>
          <AppButton variant="secondary" size="sm" :disabled="page === totalPages" @click="handlePageChange(page + 1)">
            &#8594;
          </AppButton>
        </div>
      </div>
    </div>

    <!-- Dialog -->
    <AppDialog v-model:open="dialogOpen" :title="editingModel ? '编辑模型' : '新建模型'" :description="editingModel ? '编辑模型配置' : '创建新模型'" size="lg">
      <form @submit.prevent="handleFormSubmit">
        <div v-if="formError" class="px-3.5 py-2.5 bg-danger-light text-danger rounded-md text-[13px] mb-4">{{ formError }}</div>

        <div class="grid grid-cols-2 gap-4">
          <AppFormField label="模型名称" required class="col-span-2">
            <AppInput v-model="formData.name" type="text" placeholder="如: gpt-4o, claude-3-opus" />
          </AppFormField>
          <AppFormField label="描述" class="col-span-2">
            <textarea v-model="formData.description" rows="2" placeholder="模型描述（可选）" class="w-full px-3 py-2 bg-bg-secondary text-text-primary placeholder:text-text-muted border border-border rounded-md text-sm transition-all duration-200 focus:outline-none focus:border-border-focus focus:ring-1 focus:ring-primary/30 resize-y min-h-[60px]"></textarea>
          </AppFormField>
          <AppFormField label="计费方式">
            <select v-model="formData.pricingType" class="w-full px-3 py-2 bg-bg-secondary text-text-primary border border-border rounded-md text-sm transition-all duration-200 focus:outline-none focus:border-border-focus focus:ring-1 focus:ring-primary/30">
              <option value="tokens">按 Token 计费</option>
              <option value="request">按次计费</option>
            </select>
          </AppFormField>

          <!-- Token-based pricing -->
          <template v-if="formData.pricingType === 'tokens'">
            <AppFormField class="col-span-2">
              <template #label>
                <span class="text-sm font-semibold text-text-primary mt-2">Token 定价 <span class="font-normal text-xs text-text-muted ml-1">(每百万 Token)</span></span>
              </template>
            </AppFormField>
            <AppFormField label="输入价格">
              <div class="flex items-center bg-bg-secondary border border-border rounded-md overflow-hidden transition-colors duration-150 focus-within:border-border-focus focus-within:ring-1 focus-within:ring-primary/30">
                <span class="pl-3 pr-0 py-2 text-sm text-text-muted font-medium">$</span>
                <input v-model.number="pricingForm.inputPrice" type="number" step="0.01" min="0" placeholder="0.00" class="flex-1 bg-transparent border-none px-3 py-2 text-[13px] font-mono text-text-primary focus:outline-none" />
              </div>
            </AppFormField>
            <AppFormField label="输出价格">
              <div class="flex items-center bg-bg-secondary border border-border rounded-md overflow-hidden transition-colors duration-150 focus-within:border-border-focus focus-within:ring-1 focus-within:ring-primary/30">
                <span class="pl-3 pr-0 py-2 text-sm text-text-muted font-medium">$</span>
                <input v-model.number="pricingForm.outputPrice" type="number" step="0.01" min="0" placeholder="0.00" class="flex-1 bg-transparent border-none px-3 py-2 text-[13px] font-mono text-text-primary focus:outline-none" />
              </div>
            </AppFormField>
            <AppFormField label="缓存创建价格">
              <div class="flex items-center bg-bg-secondary border border-border rounded-md overflow-hidden transition-colors duration-150 focus-within:border-border-focus focus-within:ring-1 focus-within:ring-primary/30">
                <span class="pl-3 pr-0 py-2 text-sm text-text-muted font-medium">$</span>
                <input v-model.number="pricingForm.cacheCreatePrice" type="number" step="0.01" min="0" placeholder="0.00" class="flex-1 bg-transparent border-none px-3 py-2 text-[13px] font-mono text-text-primary focus:outline-none" />
              </div>
            </AppFormField>
            <AppFormField label="缓存读取价格">
              <div class="flex items-center bg-bg-secondary border border-border rounded-md overflow-hidden transition-colors duration-150 focus-within:border-border-focus focus-within:ring-1 focus-within:ring-primary/30">
                <span class="pl-3 pr-0 py-2 text-sm text-text-muted font-medium">$</span>
                <input v-model.number="pricingForm.cacheReadPrice" type="number" step="0.01" min="0" placeholder="0.00" class="flex-1 bg-transparent border-none px-3 py-2 text-[13px] font-mono text-text-primary focus:outline-none" />
              </div>
            </AppFormField>
          </template>

          <!-- Request-based pricing -->
          <template v-else>
            <AppFormField label="每次请求价格" class="col-span-2">
              <div class="flex items-center bg-bg-secondary border border-border rounded-md overflow-hidden transition-colors duration-150 focus-within:border-border-focus focus-within:ring-1 focus-within:ring-primary/30 max-w-[200px]">
                <span class="pl-3 pr-0 py-2 text-sm text-text-muted font-medium">$</span>
                <input v-model.number="pricingForm.perRequestPrice" type="number" step="0.0001" min="0" placeholder="0.0000" class="flex-1 bg-transparent border-none px-3 py-2 text-[13px] font-mono text-text-primary focus:outline-none" />
              </div>
            </AppFormField>
          </template>
        </div>
      </form>
      <template #footer>
        <AppButton variant="secondary" @click="dialogOpen = false">取消</AppButton>
        <AppButton variant="primary" :loading="formLoading" @click="handleFormSubmit">
          {{ formLoading ? '处理中...' : (editingModel ? '保存' : '创建') }}
        </AppButton>
      </template>
    </AppDialog>

    <!-- Delete Dialog -->
    <AppDialog :open="!!deletingModel" @update:open="(v: boolean) => !v && (deletingModel = null)" title="确认删除" size="sm">
      <p class="text-sm text-text-secondary text-center">确定要删除「{{ deletingModel?.name }}」吗？此操作无法撤销。</p>
      <template #footer>
        <div class="flex justify-center gap-2 w-full">
          <AppButton variant="secondary" @click="deletingModel = null">取消</AppButton>
          <AppButton variant="danger" :loading="deleteLoading" @click="handleDelete">
            {{ deleteLoading ? '删除中...' : '删除' }}
          </AppButton>
        </div>
      </template>
    </AppDialog>
  </div>
</template>
