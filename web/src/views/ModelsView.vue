<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from 'reka-ui'
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
  dialogContent as dialogAnim,
} from '@/utils/motion'

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
  <div v-motion="fadeUp" class="page-container">
    <!-- Page Header -->
    <header class="page-header">
      <div class="header-left">
        <span class="page-label">模型管理</span>
        <div class="stats-bar">
          <div class="stat">
            <span class="stat-dot success"></span>
            <span class="stat-num">{{ statsEnabled }}</span>
            <span class="stat-text">已启用</span>
          </div>
          <div class="stat">
            <span class="stat-dot muted"></span>
            <span class="stat-num">{{ statsDisabled }}</span>
            <span class="stat-text">已禁用</span>
          </div>
          <div class="stat">
            <span class="stat-num">{{ total }}</span>
            <span class="stat-text">总计</span>
          </div>
        </div>
      </div>
      <button class="btn-create" @click="openCreateDialog">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        新建模型
      </button>
    </header>

    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-inputs">
        <input
          v-model="keyword"
          type="text"
          placeholder="搜索模型名称或描述..."
          class="input-search"
          @keyup.enter="handleSearch"
        />
      </div>
      <button class="btn-search" @click="handleSearch">搜索</button>
    </div>

    <!-- Data Table -->
    <div class="table-wrapper">
      <table class="data-table">
        <thead>
          <tr>
            <th>模型名称</th>
            <th>计费方式</th>
            <th>定价信息</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="5" class="cell-loading">
              <div class="spinner"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="models.length === 0">
            <td colspan="5" class="cell-empty">
              <div class="empty-icon">◈</div>
              <span>暂无模型数据</span>
            </td>
          </tr>
          <tr v-else v-for="(model, index) in models" :key="model.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="data-row">
            <td>
              <div class="model-info">
                <div class="model-icon">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                    <path d="M2 17l10 5 10-5"/>
                    <path d="M2 12l10 5 10-5"/>
                  </svg>
                </div>
                <div class="model-meta">
                  <span class="model-name">{{ model.name }}</span>
                  <span v-if="model.description" class="model-desc">{{ model.description }}</span>
                </div>
              </div>
            </td>
            <td>
              <span class="pricing-type-badge" :class="model.pricingType">
                {{ PRICING_TYPE_LABELS[model.pricingType] }}
              </span>
            </td>
            <td>
              <div class="pricing-cell">
                <template v-if="model.pricingType === 'tokens'">
                  <div class="price-row">
                    <span class="price-label">输入</span>
                    <span class="price-value">{{ formatPricePerMillion(model.pricing.inputPrice) }}</span>
                  </div>
                  <div class="price-row">
                    <span class="price-label">输出</span>
                    <span class="price-value">{{ formatPricePerMillion(model.pricing.outputPrice) }}</span>
                  </div>
                </template>
                <template v-else>
                  <span class="price-value">{{ model.pricing.perRequestPrice ? `$${model.pricing.perRequestPrice.toFixed(4)}` : '-' }}</span>
                  <span class="price-label">/ 次</span>
                </template>
              </div>
            </td>
            <td><span class="date-text">{{ new Date(model.createdAt).toLocaleDateString() }}</span></td>
            <td class="text-right">
              <div class="row-actions">
                <button class="action-link" @click="openEditDialog(model)">编辑</button>
                <button class="action-link danger" @click="deletingModel = model">删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="table-footer">
        <span class="total-info">共 {{ total }} 条</span>
        <div class="pagination">
          <button class="page-btn" :disabled="page === 1" @click="handlePageChange(page - 1)">
            ←
          </button>
          <span class="page-info">{{ page }} / {{ totalPages }}</span>
          <button class="page-btn" :disabled="page === totalPages" @click="handlePageChange(page + 1)">
            →
          </button>
        </div>
      </div>
    </div>

    <!-- Dialog -->
    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay
          v-motion
          :initial="{ opacity: 0 }"
          :enter="{ opacity: 1, transition: { duration: 200 } }"
          :leave="{ opacity: 0, transition: { duration: 150 } }"
          class="dlg-overlay"
        />
        <DialogContent
          v-motion="dialogAnim"
          class="dlg-content dlg-lg"
        >
          <div class="dlg-header">
            <DialogTitle class="dlg-title">
              {{ editingModel ? '编辑模型' : '新建模型' }}
            </DialogTitle>
            <DialogClose class="dlg-close">×</DialogClose>
          </div>

          <DialogDescription class="sr-only">
            {{ editingModel ? '编辑模型配置' : '创建新模型' }}
          </DialogDescription>

          <form @submit.prevent="handleFormSubmit" class="dlg-body">
            <div v-if="formError" class="form-err">{{ formError }}</div>

            <div class="form-grid">
              <div class="field full">
                <label>模型名称 <span class="req">*</span></label>
                <input v-model="formData.name" type="text" placeholder="如: gpt-4o, claude-3-opus" />
              </div>
              <div class="field full">
                <label>描述</label>
                <textarea v-model="formData.description" rows="2" placeholder="模型描述（可选）"></textarea>
              </div>
              <div class="field">
                <label>计费方式</label>
                <select v-model="formData.pricingType">
                  <option value="tokens">按 Token 计费</option>
                  <option value="request">按次计费</option>
                </select>
              </div>

              <!-- Token-based pricing -->
              <template v-if="formData.pricingType === 'tokens'">
                <div class="field full">
                  <label class="section-label">Token 定价 <span class="unit">(每百万 Token)</span></label>
                </div>
                <div class="field">
                  <label>输入价格</label>
                  <div class="input-with-suffix">
                    <span class="suffix">$</span>
                    <input v-model.number="pricingForm.inputPrice" type="number" step="0.01" min="0" placeholder="0.00" />
                  </div>
                </div>
                <div class="field">
                  <label>输出价格</label>
                  <div class="input-with-suffix">
                    <span class="suffix">$</span>
                    <input v-model.number="pricingForm.outputPrice" type="number" step="0.01" min="0" placeholder="0.00" />
                  </div>
                </div>
                <div class="field">
                  <label>缓存创建价格</label>
                  <div class="input-with-suffix">
                    <span class="suffix">$</span>
                    <input v-model.number="pricingForm.cacheCreatePrice" type="number" step="0.01" min="0" placeholder="0.00" />
                  </div>
                </div>
                <div class="field">
                  <label>缓存读取价格</label>
                  <div class="input-with-suffix">
                    <span class="suffix">$</span>
                    <input v-model.number="pricingForm.cacheReadPrice" type="number" step="0.01" min="0" placeholder="0.00" />
                  </div>
                </div>
              </template>

              <!-- Request-based pricing -->
              <template v-else>
                <div class="field full">
                  <label>每次请求价格</label>
                  <div class="input-with-suffix wide">
                    <span class="suffix">$</span>
                    <input v-model.number="pricingForm.perRequestPrice" type="number" step="0.0001" min="0" placeholder="0.0000" />
                  </div>
                </div>
              </template>
            </div>
          </form>

          <div class="dlg-footer">
            <DialogClose as-child>
              <button type="button" class="btn-cancel">取消</button>
            </DialogClose>
            <button type="button" class="btn-save" :disabled="formLoading" @click="handleFormSubmit">
              {{ formLoading ? '处理中...' : (editingModel ? '保存' : '创建') }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- Delete Dialog -->
    <DialogRoot :open="!!deletingModel" @update:open="(v) => !v && (deletingModel = null)">
      <DialogPortal>
        <DialogOverlay
          v-motion
          :initial="{ opacity: 0 }"
          :enter="{ opacity: 1, transition: { duration: 200 } }"
          :leave="{ opacity: 0, transition: { duration: 150 } }"
          class="dlg-overlay"
        />
        <DialogContent
          v-motion="dialogAnim"
          class="dlg-content dlg-sm"
        >
          <DialogTitle class="dlg-title dlg-center">确认删除</DialogTitle>
          <DialogDescription class="dlg-desc">
            确定要删除「{{ deletingModel?.name }}」吗？此操作无法撤销。
          </DialogDescription>
          <div class="dlg-footer dlg-center">
            <button class="btn-cancel" @click="deletingModel = null">取消</button>
            <button class="btn-danger" :disabled="deleteLoading" @click="handleDelete">
              {{ deleteLoading ? '删除中...' : '删除' }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </div>
</template>

<style scoped>
.page-container {
  max-width: 1200px;
}

/* Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.page-label {
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--color-text-primary);
}

.stats-bar {
  display: flex;
  gap: 1.5rem;
}

.stat {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.stat-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-text-muted);

  &.success {
    background: var(--color-success);
  }

  &.muted {
    background: var(--color-text-muted);
  }
}

.stat-num {
  font-weight: 600;
  color: var(--color-text-primary);
}

.stat-text {
  color: var(--color-text-muted);
}

.btn-create {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }
}

/* Filter Bar */
.filter-bar {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
  padding: 1rem;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.filter-inputs {
  display: flex;
  gap: 0.5rem;
  flex: 1;
}

.input-search {
  flex: 1;
  max-width: 400px;
  padding: 0.5rem 0.875rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-primary);
  transition: border-color 0.15s;

  &::placeholder {
    color: var(--color-text-muted);
  }

  &:focus {
    outline: none;
    border-color: var(--color-primary);
  }
}

.btn-search {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    background: var(--color-bg-tertiary);
    border-color: var(--color-primary);
    color: var(--color-primary);
  }
}

/* Table */
.table-wrapper {
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--color-border);
  overflow: hidden;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  padding: 0.875rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.data-row {
  transition: background 0.15s;

  &:hover {
    background: var(--color-bg-secondary);
  }
}

.text-right {
  text-align: right;
}

.cell-loading,
.cell-empty {
  padding: 3rem !important;
  text-align: center;
  color: var(--color-text-muted);
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto 0.5rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-icon {
  font-size: 2rem;
  margin-bottom: 0.5rem;
  opacity: 0.3;
  color: var(--color-primary);
}

/* Model Info */
.model-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.model-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(139, 92, 246, 0.15));
  color: var(--color-primary);
}

.model-meta {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.model-name {
  font-weight: 500;
  color: var(--color-text-primary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.875rem;
}

.model-desc {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Pricing Type Badge */
.pricing-type-badge {
  display: inline-block;
  padding: 0.25rem 0.625rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;

  &.tokens {
    background: rgba(16, 163, 127, 0.1);
    color: #10a37f;
  }

  &.request {
    background: rgba(217, 119, 6, 0.1);
    color: #d97706;
  }
}

/* Pricing Cell */
.pricing-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.price-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.price-label {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.price-value {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-primary);
  font-family: 'JetBrains Mono', monospace;
}

.date-text {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

/* Row Actions */
.row-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.action-link {
  padding: 0.25rem 0.5rem;
  background: transparent;
  border: none;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;

  &:hover {
    color: var(--color-primary);
    background: var(--color-primary-light);
  }

  &.danger:hover {
    color: var(--color-danger);
    background: var(--color-danger-light);
  }
}

/* Table Footer */
.table-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.875rem 1rem;
  border-top: 1px solid var(--color-border);
}

.total-info {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}

.pagination {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.page-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;

  &:hover:not(:disabled) {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}

.page-info {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}
</style>

<!-- Global Dialog Styles (shared) -->
<style>
.dlg-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  z-index: 40;
}

.dlg-content {
  position: fixed;
  inset: 0;
  margin: auto;
  width: 90%;
  max-width: 480px;
  height: fit-content;
  max-height: 85vh;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  z-index: 50;
  overflow: hidden;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.2), 0 8px 16px rgba(0, 0, 0, 0.1);
}

.dlg-lg {
  max-width: 560px;
}

.dlg-sm {
  max-width: 400px;
}

.dlg-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.dlg-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.dlg-center {
  text-align: center;
  justify-content: center;
}

.dlg-close {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  font-size: 1.25rem;
  color: var(--color-text-muted);
  cursor: pointer;
  border-radius: 6px;

  &:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }
}

.dlg-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.25rem;
}

.dlg-desc {
  padding: 1rem 1.25rem;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  text-align: center;
}

.form-err {
  padding: 0.625rem 0.875rem;
  background: var(--color-danger-light);
  color: var(--color-danger);
  border-radius: 6px;
  font-size: 0.8125rem;
  margin-bottom: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;

  &.full {
    grid-column: 1 / -1;
  }

  label {
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .req {
    color: var(--color-danger);
  }

  .section-label {
    font-weight: 600;
    color: var(--color-text-primary);
    margin-top: 0.5rem;

    .unit {
      font-weight: 400;
      font-size: 0.75rem;
      color: var(--color-text-muted);
      margin-left: 0.25rem;
    }
  }

  input,
  select,
  textarea {
    padding: 0.5rem 0.75rem;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    font-size: 0.875rem;
    color: var(--color-text-primary);
    transition: border-color 0.15s;

    &:focus {
      outline: none;
      border-color: var(--color-primary);
    }
  }

  textarea {
    resize: vertical;
    min-height: 60px;
  }

  .input-with-suffix {
    display: flex;
    align-items: center;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    overflow: hidden;
    transition: border-color 0.15s;

    &:focus-within {
      border-color: var(--color-primary);
    }

    .suffix {
      padding: 0.5rem 0 0.5rem 0.75rem;
      font-size: 0.875rem;
      color: var(--color-text-muted);
      font-weight: 500;
    }

    input {
      flex: 1;
      border: none;
      background: transparent;
      padding: 0.5rem 0.75rem 0.5rem 0.25rem;
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.8125rem;

      &:focus {
        outline: none;
      }
    }
  }

  .input-with-suffix.wide {
    max-width: 200px;
  }
}

.dlg-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);

  &.dlg-center {
    justify-content: center;
  }
}

.btn-cancel {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  cursor: pointer;

  &:hover {
    background: var(--color-bg-tertiary);
  }
}

.btn-save {
  padding: 0.5rem 1rem;
  background: var(--color-primary);
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  color: white;
  cursor: pointer;

  &:hover {
    opacity: 0.9;
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.btn-danger {
  padding: 0.5rem 1rem;
  background: var(--color-danger);
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  color: white;
  cursor: pointer;

  &:hover {
    opacity: 0.9;
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}
</style>
