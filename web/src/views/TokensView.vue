<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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
  listTokens,
  createToken,
  updateToken,
  deleteToken,
  setTokenStatus,
  formatToken,
  copyToClipboard,
  TOKEN_STATUS_LABELS,
  type CreateTokenRequest,
  type UpdateTokenRequest,
} from '@/api/token'
import { listChannelGroups, listLoadBalanceStrategies } from '@/api/channel-group'
import type { TokenWithRelations, ChannelGroup, TokenStatus, LoadBalanceStrategy } from '@/types'
import {
  fadeUp,
  slideUp,
  tableRow,
  dialogContent as dialogAnim,
  staggerItem,
  scaleIn,
} from '@/utils/motion'
import CustomSelect from '@/components/CustomSelect.vue'

// Status options for filter
const statusOptions = computed(() => {
  const options = [{ value: '', label: '全部状态' }]
  for (const [key, info] of Object.entries(TOKEN_STATUS_LABELS)) {
    const colors: Record<string, string> = {
      enabled: 'var(--color-success)',
      disabled: 'var(--color-warning)',
      expired: 'var(--color-danger)',
    }
    options.push({ value: key, label: info.label, color: colors[key] })
  }
  return options
})

// List data
const tokens = ref<TokenWithRelations[]>([])
const groups = ref<ChannelGroup[]>([])
const strategies = ref<LoadBalanceStrategy[]>([])

// Group options for filter
const groupOptions = computed(() => {
  const options = [{ value: '', label: '全部分组' }]
  for (const group of groups.value) {
    options.push({ value: group.id, label: group.name })
  }
  return options
})

// Strategy options for form
const strategyOptions = computed(() => {
  return strategies.value.map(s => ({ value: s.value, label: s.label }))
})

const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// Filters
const keyword = ref('')
const filterStatus = ref<TokenStatus | ''>('')
const filterGroupID = ref<string | ''>('')

// Dialog
const dialogOpen = ref(false)
const editingToken = ref<TokenWithRelations | null>(null)
const formLoading = ref(false)
const formError = ref('')
const newlyCreatedToken = ref<string>('')

// Delete confirmation
const deletingToken = ref<TokenWithRelations | null>(null)
const deleteLoading = ref(false)

// Copy feedback
const copiedTokenId = ref<string | null>(null)

// Form data
const formData = ref<CreateTokenRequest>({
  groupID: '',
  name: '',
  limit: undefined,
  expiresAt: undefined,
  loadBalanceStrategy: 'performance',
})

// Edit form status (separate from create)
const editStatus = ref<TokenStatus>('enabled')

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
const statsEnabled = computed(() => tokens.value.filter(t => t.status === 'enabled').length)
const statsDisabled = computed(() => tokens.value.filter(t => t.status === 'disabled').length)

async function loadTokens() {
  loading.value = true
  try {
    const result = await listTokens({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value || undefined,
      status: filterStatus.value || undefined,
      groupID: filterGroupID.value || undefined,
      preloadGroup: true,
      preloadStats: true,
    })
    tokens.value = result.items
    total.value = result.total
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    const result = await listChannelGroups({ pageSize: 100 })
    groups.value = result.items
  } catch {
    // ignore
  }
}

async function loadStrategies() {
  try {
    strategies.value = await listLoadBalanceStrategies()
  } catch {
    // ignore
  }
}

function openCreateDialog() {
  editingToken.value = null
  newlyCreatedToken.value = ''
  formData.value = {
    groupID: groups.value[0]?.id || '',
    name: '',
    limit: undefined,
    expiresAt: undefined,
    loadBalanceStrategy: 'performance',
  }
  formError.value = ''
  dialogOpen.value = true
}

function openEditDialog(token: TokenWithRelations) {
  editingToken.value = token
  newlyCreatedToken.value = ''
  formData.value = {
    groupID: token.groupID,
    name: token.name,
    limit: token.limit || undefined,
    expiresAt: token.expiresAt ? token.expiresAt.slice(0, 16) : undefined,
    loadBalanceStrategy: token.loadBalanceStrategy,
  }
  editStatus.value = token.status
  formError.value = ''
  dialogOpen.value = true
}

async function handleFormSubmit() {
  if (!formData.value.groupID) {
    formError.value = '请选择渠道分组'
    return
  }
  if (!formData.value.name) {
    formError.value = '请输入令牌名称'
    return
  }

  formLoading.value = true
  formError.value = ''

  try {
    if (editingToken.value) {
      const updateData: UpdateTokenRequest = {
        groupID: formData.value.groupID,
        name: formData.value.name,
        status: editStatus.value,
        loadBalanceStrategy: formData.value.loadBalanceStrategy,
      }
      if (formData.value.limit !== undefined) {
        updateData.limit = formData.value.limit
      }
      await updateToken(editingToken.value.id, updateData)
      dialogOpen.value = false
      await loadTokens()
    } else {
      const result = await createToken(formData.value)
      newlyCreatedToken.value = result.token
      await loadTokens()
    }
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '操作失败'
    formError.value = message
  } finally {
    formLoading.value = false
  }
}

async function handleToggleStatus(token: TokenWithRelations) {
  const newStatus: TokenStatus = token.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await setTokenStatus(token.id, newStatus)
    await loadTokens()
  } catch {
    // ignore
  }
}

async function handleDelete() {
  if (!deletingToken.value) return

  deleteLoading.value = true
  try {
    await deleteToken(deletingToken.value.id)
    deletingToken.value = null
    await loadTokens()
  } catch {
    // ignore
  } finally {
    deleteLoading.value = false
  }
}

async function handleCopyToken(token: TokenWithRelations) {
  const success = await copyToClipboard(token.token)
  if (success) {
    copiedTokenId.value = token.id
    setTimeout(() => {
      copiedTokenId.value = null
    }, 2000)
  }
}

function handleSearch() {
  page.value = 1
  loadTokens()
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadTokens()
}

// Format cost from micros
function formatCost(micros: number): string {
  if (!micros) return '¥0.00'
  return `¥${(micros / 1_000_000).toFixed(4)}`
}

// Format token usage
function formatTokens(num: number): string {
  if (num >= 1_000_000) {
    return `${(num / 1_000_000).toFixed(2)}M`
  }
  if (num >= 1_000) {
    return `${(num / 1_000).toFixed(1)}K`
  }
  return num.toString()
}

onMounted(() => {
  loadTokens()
  loadGroups()
  loadStrategies()
})
</script>

<template>
  <div v-motion="fadeUp" class="page-container">
    <!-- Page Header -->
    <header class="page-header">
      <div class="header-left">
        <span class="page-label">令牌管理</span>
        <div v-motion="slideUp" class="stats-bar">
          <div class="stat" :style="{ animationDelay: '0ms' }">
            <span class="stat-dot success"></span>
            <span class="stat-num">{{ statsEnabled }}</span>
            <span class="stat-text">启用中</span>
          </div>
          <div class="stat" :style="{ animationDelay: '100ms' }">
            <span class="stat-dot warning"></span>
            <span class="stat-num">{{ statsDisabled }}</span>
            <span class="stat-text">已禁用</span>
          </div>
          <div class="stat" :style="{ animationDelay: '200ms' }">
            <span class="stat-num">{{ total }}</span>
            <span class="stat-text">总计</span>
          </div>
        </div>
      </div>
      <button
        v-motion
        :initial="{ opacity: 0, scale: 0.9 }"
        :enter="{ opacity: 1, scale: 1, transition: { delay: 150, type: 'spring', stiffness: 400, damping: 25 } }"
        :hovered="{ scale: 1.05, transition: { duration: 150 } }"
        :tapped="{ scale: 0.95 }"
        class="btn-create"
        @click="openCreateDialog"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建令牌
      </button>
    </header>

    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-inputs">
        <input
          v-model="keyword"
          type="text"
          placeholder="搜索令牌名称..."
          class="input-search"
          @keyup.enter="handleSearch"
        />
        <CustomSelect
          v-model="filterStatus"
          :options="statusOptions"
          placeholder="全部状态"
          size="sm"
          @change="handleSearch"
        />
        <CustomSelect
          v-model="filterGroupID"
          :options="groupOptions"
          placeholder="全部分组"
          size="sm"
          @change="handleSearch"
        />
      </div>
      <button class="btn-search" @click="handleSearch">搜索</button>
    </div>

    <!-- Data Table -->
    <div class="table-wrapper">
      <table class="data-table">
        <thead>
          <tr>
            <th>令牌名称</th>
            <th>Token</th>
            <th>所属分组</th>
            <th>状态</th>
            <th>额度使用</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="7" class="cell-loading">
              <div class="spinner"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="tokens.length === 0">
            <td colspan="7" class="cell-empty">
              <div class="empty-icon">○</div>
              <span>暂无令牌数据</span>
            </td>
          </tr>
          <tr v-else v-for="(token, index) in tokens" :key="token.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="data-row">
            <td>
              <div class="token-name">
                <span class="name-text">{{ token.name }}</span>
              </div>
            </td>
            <td>
              <div class="token-value">
                <code class="token-code">{{ formatToken(token.token) }}</code>
                <button
                  class="copy-btn"
                  :class="{ copied: copiedTokenId === token.id }"
                  @click="handleCopyToken(token)"
                >
                  <svg v-if="copiedTokenId !== token.id" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                  </svg>
                  <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                </button>
              </div>
            </td>
            <td>
              <span class="group-tag">{{ token.group?.name || '-' }}</span>
            </td>
            <td>
              <span
                v-motion
                :initial="{ opacity: 0, scale: 0.8 }"
                :enter="{ opacity: 1, scale: 1, transition: { delay: index * 50 + 100, type: 'spring', stiffness: 400, damping: 20 } }"
                class="status-badge"
                :class="token.status"
              >
                {{ TOKEN_STATUS_LABELS[token.status]?.label }}
              </span>
            </td>
            <td>
              <div v-if="token.stats" class="usage-info">
                <div class="usage-row">
                  <span class="usage-label">请求</span>
                  <span class="usage-value">{{ token.stats.requestSuccess + token.stats.requestFailed }}</span>
                </div>
                <div class="usage-row">
                  <span class="usage-label">Token</span>
                  <span class="usage-value">{{ formatTokens(token.stats.inputToken + token.stats.outputToken) }}</span>
                </div>
              </div>
              <span v-else class="no-data">-</span>
            </td>
            <td>
              <span class="date-text">{{ new Date(token.createdAt).toLocaleDateString() }}</span>
            </td>
            <td class="text-right">
              <div class="row-actions">
                <button
                  class="action-link"
                  :class="token.status === 'enabled' ? 'warn' : 'success'"
                  @click="handleToggleStatus(token)"
                >
                  {{ token.status === 'enabled' ? '禁用' : '启用' }}
                </button>
                <button class="action-link" @click="openEditDialog(token)">编辑</button>
                <button class="action-link danger" @click="deletingToken = token">删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="table-footer">
        <span class="total-info">共 {{ total }} 条</span>
        <div class="pagination">
          <button class="page-btn" :disabled="page === 1" @click="handlePageChange(page - 1)">←</button>
          <span class="page-info">{{ page }} / {{ totalPages }}</span>
          <button class="page-btn" :disabled="page === totalPages" @click="handlePageChange(page + 1)">→</button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Dialog -->
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
          class="dlg-content"
        >
          <div class="dlg-header">
            <DialogTitle class="dlg-title">
              {{ editingToken ? '编辑令牌' : '新建令牌' }}
            </DialogTitle>
            <DialogClose class="dlg-close">×</DialogClose>
          </div>

          <DialogDescription class="sr-only">
            {{ editingToken ? '编辑令牌配置' : '创建新的 API 令牌' }}
          </DialogDescription>

          <!-- New Token Display -->
          <div v-if="newlyCreatedToken" v-motion="scaleIn" class="new-token-banner">
            <div class="banner-header">
              <svg v-motion :initial="{ scale: 0, rotate: -45 }" :enter="{ scale: 1, rotate: 0, transition: { delay: 200, type: 'spring', stiffness: 500, damping: 15 } }" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
              </svg>
              <span>令牌创建成功</span>
            </div>
            <div class="banner-body">
              <p class="banner-hint">请立即复制保存，此令牌仅显示一次：</p>
              <div class="token-display">
                <code>{{ newlyCreatedToken }}</code>
                <button class="copy-token-btn" @click="copyToClipboard(newlyCreatedToken)">
                  复制
                </button>
              </div>
            </div>
          </div>

          <form v-else @submit.prevent="handleFormSubmit" class="dlg-body">
            <div v-if="formError" class="form-err">{{ formError }}</div>

            <div class="field">
              <label>令牌名称 <span class="req">*</span></label>
              <input v-model="formData.name" type="text" placeholder="如: 生产环境令牌" required />
            </div>

            <div class="field">
              <label>所属分组 <span class="req">*</span></label>
              <select v-model="formData.groupID" required>
                <option value="">请选择分组</option>
                <option v-for="group in groups" :key="group.id" :value="group.id">
                  {{ group.name }}
                </option>
              </select>
            </div>

            <div class="field-row">
              <div v-if="editingToken" class="field">
                <label>状态</label>
                <select v-model="editStatus">
                  <option value="enabled">启用</option>
                  <option value="disabled">禁用</option>
                </select>
              </div>
              <div class="field">
                <label>负载均衡策略</label>
                <select v-model="formData.loadBalanceStrategy">
                  <option v-for="opt in strategyOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>
            </div>

            <div class="field">
              <label>额度限制 (元)</label>
              <input v-model.number="formData.limit" type="number" step="0.01" min="0" placeholder="不限制" />
              <p class="hint">留空表示不限制</p>
            </div>

            <div class="field">
              <label>过期时间</label>
              <input v-model="formData.expiresAt" type="datetime-local" />
              <p class="hint">留空表示永不过期</p>
            </div>
          </form>

          <div v-if="!newlyCreatedToken" class="dlg-footer">
            <DialogClose as-child>
              <button type="button" class="btn-cancel">取消</button>
            </DialogClose>
            <button type="button" class="btn-save" :disabled="formLoading" @click="handleFormSubmit">
              {{ formLoading ? '处理中...' : (editingToken ? '保存' : '创建') }}
            </button>
          </div>
          <div v-else class="dlg-footer">
            <DialogClose as-child>
              <button type="button" class="btn-save">完成</button>
            </DialogClose>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- Delete Dialog -->
    <DialogRoot :open="!!deletingToken" @update:open="(v) => !v && (deletingToken = null)">
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
            确定要删除令牌「{{ deletingToken?.name }}」吗？此操作无法撤销，使用该令牌的应用将无法继续访问。
          </DialogDescription>
          <div class="dlg-footer dlg-center">
            <button class="btn-cancel" @click="deletingToken = null">取消</button>
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
  max-width: 1400px;
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
  animation: stat-fade-in 0.4s ease-out forwards;
  opacity: 0;
}

@keyframes stat-fade-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stat-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-text-muted);
  animation: dot-pulse 2s ease-in-out infinite;

  &.success {
    background: var(--color-success);
    animation-delay: 0s;
  }
  &.warning {
    background: var(--color-warning);
    animation-delay: 0.3s;
  }
}

@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
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
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  box-shadow: 0 2px 8px rgba(var(--color-primary-rgb, 139, 195, 74), 0.3);

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(var(--color-primary-rgb, 139, 195, 74), 0.4);
  }

  &:active {
    transform: translateY(0);
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
  max-width: 280px;
  padding: 0.5rem 0.875rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-primary);

  &::placeholder { color: var(--color-text-muted); }
  &:focus { outline: none; border-color: var(--color-primary); }
}

/* CustomSelect in filter bar */
.filter-inputs .custom-select {
  min-width: 110px;
  max-width: 140px;
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
  transition: background 0.2s, transform 0.2s;

  &:hover {
    background: var(--color-bg-secondary);
    transform: translateX(2px);
  }
}

.text-right { text-align: right; }

.cell-loading,
.cell-empty {
  padding: 3rem !important;
  text-align: center;
  color: var(--color-text-muted);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  margin: 0 auto 0.75rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-icon {
  font-size: 2rem;
  margin-bottom: 0.5rem;
  opacity: 0.3;
}

/* Token Name */
.token-name .name-text {
  font-weight: 500;
  color: var(--color-text-primary);
}

/* Token Value */
.token-value {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.token-code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.8125rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-bg-tertiary);
  border-radius: 4px;
  color: var(--color-text-secondary);
}

.copy-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);

  &:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
    transform: scale(1.1);
  }

  &:active {
    transform: scale(0.95);
  }

  &.copied {
    background: var(--color-success-light);
    border-color: var(--color-success);
    color: var(--color-success);
    animation: success-bounce 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
}

@keyframes success-bounce {
  0% { transform: scale(1); }
  30% { transform: scale(1.2); }
  60% { transform: scale(0.9); }
  100% { transform: scale(1); }
}

/* Group Tag */
.group-tag {
  display: inline-block;
  padding: 0.25rem 0.625rem;
  background: var(--color-bg-tertiary);
  border-radius: 4px;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

/* Status Badge */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 500;
  transition: transform 0.2s, box-shadow 0.2s;

  &::before {
    content: '';
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: currentColor;
  }

  &.enabled {
    background: var(--color-success-light);
    color: var(--color-success);
    animation: pulse-glow 2s ease-in-out infinite;
  }
  &.disabled {
    background: var(--color-warning-light);
    color: var(--color-warning);
  }
  &.expired {
    background: var(--color-danger-light);
    color: var(--color-danger);
  }
}

@keyframes pulse-glow {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(var(--color-success-rgb, 34, 197, 94), 0.4);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(var(--color-success-rgb, 34, 197, 94), 0);
  }
}

/* Usage Info */
.usage-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.usage-row {
  display: flex;
  gap: 0.5rem;
  font-size: 0.8125rem;
}

.usage-label {
  color: var(--color-text-muted);
  min-width: 40px;
}

.usage-value {
  color: var(--color-text-secondary);
  font-weight: 500;
}

.no-data {
  color: var(--color-text-muted);
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

  &.success:hover {
    color: var(--color-success);
    background: var(--color-success-light);
  }

  &.warn:hover {
    color: var(--color-warning);
    background: var(--color-warning-light);
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

<!-- Global Dialog Styles -->
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

.dlg-sm { max-width: 400px; }

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

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-bottom: 1rem;

  label {
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .req { color: var(--color-danger); }

  input, select {
    padding: 0.5rem 0.75rem;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    font-size: 0.875rem;
    color: var(--color-text-primary);

    &:focus { outline: none; border-color: var(--color-primary); }
  }

  .hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    margin: 0;
  }
}

.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.dlg-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);

  &.dlg-center { justify-content: center; }
}

.btn-cancel {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  cursor: pointer;

  &:hover { background: var(--color-bg-tertiary); }
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

  &:hover { opacity: 0.9; }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
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

  &:hover { opacity: 0.9; }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
}

/* New Token Banner */
.new-token-banner {
  padding: 1.25rem;
  background: linear-gradient(135deg, var(--color-success-light) 0%, rgba(34, 197, 94, 0.15) 100%);
  border-bottom: 1px solid var(--color-success);
  animation: banner-slide-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes banner-slide-in {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.banner-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: var(--color-success);
  margin-bottom: 0.75rem;
  animation: header-fade-in 0.3s ease-out 0.2s both;
}

@keyframes header-fade-in {
  from { opacity: 0; transform: translateX(-10px); }
  to { opacity: 1; transform: translateX(0); }
}

.banner-hint {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.5rem;
}

.token-display {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--color-bg-card);
  border-radius: 6px;
  border: 1px solid var(--color-border);

  code {
    flex: 1;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.875rem;
    color: var(--color-text-primary);
    word-break: break-all;
  }
}

.copy-token-btn {
  padding: 0.375rem 0.75rem;
  background: var(--color-primary);
  border: none;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  color: white;
  cursor: pointer;
  white-space: nowrap;

  &:hover { opacity: 0.9; }
}
</style>
