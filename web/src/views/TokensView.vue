<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import dayjs from 'dayjs'
import {
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
  TooltipPortal,
  TooltipContent,
  TooltipArrow,
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
import { listChannelGroups, listLoadBalanceStrategies, type LoadBalanceStrategyItem } from '@/api/channel-group'
import type { TokenWithRelations, ChannelGroup, TokenStatus } from '@/types'
import {
  fadeUp,
  slideUp,
  tableRow,
  dialogContent as dialogAnim,
  staggerItem,
  scaleIn,
} from '@/utils/motion'
import AppButton from '@/components/ui/AppButton.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppBadge from '@/components/ui/AppBadge.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppFormField from '@/components/ui/AppFormField.vue'
import AppSelect from '@/components/ui/AppSelect.vue'

// Status options for filter
const statusOptions = computed(() => {
  const options: Array<{ value: string; label: string }> = [{ value: 'all', label: '全部状态' }]
  for (const [key, info] of Object.entries(TOKEN_STATUS_LABELS)) {
    options.push({ value: key, label: info.label })
  }
  return options
})

// List data
const tokens = ref<TokenWithRelations[]>([])
const groups = ref<ChannelGroup[]>([])
const strategies = ref<LoadBalanceStrategyItem[]>([])

// Group options for filter
const groupOptions = computed(() => {
  const options = [{ value: 'all', label: '全部分组' }]
  for (const group of groups.value) {
    options.push({ value: group.id, label: group.name })
  }
  return options
})

// Strategy options for form
const strategyOptions = computed(() => {
  return strategies.value.map(s => ({ value: s.value, label: s.label }))
})

const formGroupOptions = computed(() => {
  return groups.value.map(g => ({ value: g.id, label: g.name }))
})

const editStatusOptions = [
  { value: 'enabled', label: '启用' },
  { value: 'disabled', label: '禁用' },
]

const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// Filters
const keyword = ref('')
const filterStatus = ref<TokenStatus | 'all'>('all')
const filterGroupID = ref<string>('all')

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
      status: filterStatus.value === 'all' ? undefined : filterStatus.value,
      groupID: filterGroupID.value === 'all' ? undefined : filterGroupID.value,
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
  <TooltipProvider>
    <div v-motion="fadeUp" class="max-w-[1600px] min-w-[1100px]">
    <!-- Page Header -->
    <header class="flex justify-between items-start mb-6">
      <div class="flex flex-col gap-3">
        <span class="text-2xl font-semibold tracking-tight text-text-primary">令牌管理</span>
        <div v-motion="slideUp" class="flex gap-6">
          <div class="flex items-center gap-2 text-sm" :style="{ animationDelay: '0ms' }">
            <span class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></span>
            <span class="font-semibold text-text-primary">{{ statsEnabled }}</span>
            <span class="text-text-muted">启用中</span>
          </div>
          <div class="flex items-center gap-2 text-sm" :style="{ animationDelay: '100ms' }">
            <span class="w-1.5 h-1.5 rounded-full bg-warning"></span>
            <span class="font-semibold text-text-primary">{{ statsDisabled }}</span>
            <span class="text-text-muted">已禁用</span>
          </div>
          <div class="flex items-center gap-2 text-sm" :style="{ animationDelay: '200ms' }">
            <span class="font-semibold text-text-primary">{{ total }}</span>
            <span class="text-text-muted">总计</span>
          </div>
        </div>
      </div>
      <AppButton
        v-motion
        :initial="{ opacity: 0, scale: 0.9 }"
        :enter="{ opacity: 1, scale: 1, transition: { delay: 150, type: 'spring', stiffness: 400, damping: 25 } }"
        :hovered="{ scale: 1.05, transition: { duration: 150 } }"
        :tapped="{ scale: 0.95 }"
        variant="primary"
        @click="openCreateDialog"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建令牌
      </AppButton>
    </header>

    <!-- Filter Bar -->
    <div class="flex items-center gap-3 mb-4 p-4 bg-bg-card rounded-lg border border-border">
      <div class="flex items-center gap-2 flex-wrap flex-1">
        <div class="flex-1 max-w-[280px]">
          <AppInput
            v-model="keyword"
            type="text"
            placeholder="搜索令牌名称..."
            size="sm"
            @keyup.enter="handleSearch"
          />
        </div>
        <div class="w-[130px]">
          <AppSelect
            v-model="filterStatus"
            :options="statusOptions"
            size="sm"
          />
        </div>
        <div class="w-[130px]">
          <AppSelect
            v-model="filterGroupID"
            :options="groupOptions"
            size="sm"
          />
        </div>
      </div>
      <AppButton variant="secondary" size="sm" @click="handleSearch">搜索</AppButton>
    </div>

    <!-- Data Table -->
    <div class="bg-bg-card rounded-lg border border-border min-h-[400px]">
      <table class="w-full border-collapse min-w-[800px]">
        <thead>
          <tr>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">令牌名称</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">Token</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">所属分组</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">状态</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">额度使用</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">创建时间</th>
            <th class="px-4 py-3.5 text-right text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="7" class="px-4 py-12 text-center text-text-muted">
              <div class="w-8 h-8 border-[3px] border-border border-t-primary rounded-full animate-spin mx-auto mb-3"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="tokens.length === 0">
            <td colspan="7" class="px-4 py-12 text-center text-text-muted">
              <div class="text-3xl mb-2 opacity-30">○</div>
              <span>暂无令牌数据</span>
            </td>
          </tr>
          <tr v-else v-for="(token, index) in tokens" :key="token.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="border-b border-border transition-all duration-200 hover:bg-bg-secondary hover:translate-x-0.5">
            <td class="px-4 py-4 align-middle">
              <div class="font-medium text-text-primary">{{ token.name }}</div>
            </td>
            <td class="px-4 py-4 align-middle">
              <div class="flex items-center gap-2">
                <code class="font-mono text-[13px] px-2 py-1 bg-bg-tertiary rounded text-text-secondary">{{ formatToken(token.token) }}</code>
                <button
                  class="w-7 h-7 inline-flex items-center justify-center bg-transparent border border-border rounded text-text-muted transition-all duration-200 hover:bg-bg-tertiary hover:text-text-secondary hover:scale-110 active:scale-95"
                  :class="{ 'bg-success-light border-success text-success': copiedTokenId === token.id }"
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
            <td class="px-4 py-4 align-middle">
              <span class="inline-block px-2.5 py-1 bg-bg-tertiary rounded text-[13px] text-text-secondary">{{ token.group?.name || '-' }}</span>
            </td>
            <td class="px-4 py-4 align-middle">
              <AppBadge
                v-motion
                :initial="{ opacity: 0, scale: 0.8 }"
                :enter="{ opacity: 1, scale: 1, transition: { delay: index * 50 + 100, type: 'spring', stiffness: 400, damping: 20 } }"
                :variant="token.status === 'enabled' ? 'success' : token.status === 'disabled' ? 'warning' : 'danger'"
                :pulse="token.status === 'enabled'"
              >
                {{ TOKEN_STATUS_LABELS[token.status]?.label }}
              </AppBadge>
            </td>
            <td class="px-4 py-4 align-middle">
              <TooltipRoot v-if="token.stats">
                <TooltipTrigger as-child>
                  <span class="text-sm text-text-secondary font-medium cursor-help">{{ formatCost(token.stats.totalCostMicros) }}/{{ token.limit ? `¥${token.limit.toFixed(2)}` : '∞' }}</span>
                </TooltipTrigger>
                <TooltipPortal to="body">
                  <TooltipContent side="bottom" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary whitespace-nowrap">
                    <div class="flex flex-col gap-1">
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">请求</span>
                        <span class="font-medium">{{ token.stats.requestSuccess + token.stats.requestFailed }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">成功</span>
                        <span class="font-medium text-success">{{ token.stats.requestSuccess }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">失败</span>
                        <span class="font-medium text-danger">{{ token.stats.requestFailed }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">Token</span>
                        <span class="font-medium">{{ formatTokens(token.stats.inputToken + token.stats.outputToken) }}</span>
                      </div>
                    </div>
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
              <span v-else class="text-text-muted">-</span>
            </td>
            <td class="px-4 py-4 align-middle">
              <span class="text-sm text-text-secondary">{{ dayjs(token.createdAt).format('YYYY-MM-DD') }}</span>
            </td>
            <td class="px-4 py-4 align-middle text-right">
              <div class="flex justify-end gap-2">
                <AppButton
                  :variant="token.status === 'enabled' ? 'warning' : 'success'"
                  size="sm"
                  @click="handleToggleStatus(token)"
                >
                  {{ token.status === 'enabled' ? '禁用' : '启用' }}
                </AppButton>
                <AppButton variant="ghost" size="sm" @click="openEditDialog(token)">编辑</AppButton>
                <AppButton variant="danger" size="sm" @click="deletingToken = token">删除</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex justify-between items-center px-4 py-3.5 border-t border-border">
        <span class="text-[13px] text-text-muted">共 {{ total }} 条</span>
        <div class="flex items-center gap-3">
          <AppButton
            variant="secondary"
            size="sm"
            :disabled="page === 1"
            @click="handlePageChange(page - 1)"
          >
            ←
          </AppButton>
          <span class="text-[13px] text-text-secondary">{{ page }} / {{ totalPages }}</span>
          <AppButton
            variant="secondary"
            size="sm"
            :disabled="page === totalPages"
            @click="handlePageChange(page + 1)"
          >
            →
          </AppButton>
        </div>
      </div>
    </div>

    <!-- Create/Edit Dialog -->
    <AppDialog
      v-model:open="dialogOpen"
      :title="editingToken ? '编辑令牌' : '新建令牌'"
      :description="editingToken ? '编辑令牌配置' : '创建新的 API 令牌'"
      size="md"
    >
      <!-- New Token Display -->
      <div v-if="newlyCreatedToken" v-motion="scaleIn" class="p-5 bg-gradient-to-br from-success-light to-success/15 border-b border-success animate-[banner-slide-in_0.4s_ease-out]">
        <div class="flex items-center gap-2 font-semibold text-success mb-3">
          <svg v-motion :initial="{ scale: 0, rotate: -45 }" :enter="{ scale: 1, rotate: 0, transition: { delay: 200, type: 'spring', stiffness: 500, damping: 15 } }" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
          <span>令牌创建成功</span>
        </div>
        <div>
          <p class="text-[13px] text-text-secondary mb-2">请立即复制保存，此令牌仅显示一次：</p>
          <div class="flex items-center gap-2 p-3 bg-bg-card rounded-md border border-border">
            <code class="flex-1 font-mono text-sm text-text-primary break-all">{{ newlyCreatedToken }}</code>
            <AppButton variant="primary" size="sm" @click="copyToClipboard(newlyCreatedToken)">复制</AppButton>
          </div>
        </div>
      </div>

      <form v-else @submit.prevent="handleFormSubmit">
        <div v-if="formError" class="px-5 py-2.5 bg-danger-light text-danger rounded-md text-[13px] mb-4">
          {{ formError }}
        </div>

        <div class="px-5 py-4 space-y-4">
          <AppFormField label="令牌名称" required>
            <AppInput v-model="formData.name" type="text" placeholder="如: 生产环境令牌" />
          </AppFormField>

          <AppFormField label="所属分组" required>
            <AppSelect v-model="formData.groupID" :options="formGroupOptions" size="md" />
          </AppFormField>

          <div class="grid grid-cols-2 gap-4">
            <AppFormField v-if="editingToken" label="状态">
              <AppSelect v-model="editStatus" :options="editStatusOptions" size="md" />
            </AppFormField>
            <AppFormField label="负载均衡策略">
              <AppSelect v-model="formData.loadBalanceStrategy" :options="strategyOptions" size="md" />
            </AppFormField>
          </div>

          <AppFormField label="额度限制 (元)" hint="留空表示不限制">
            <AppInput v-model.number="formData.limit" type="number" step="0.01" min="0" placeholder="不限制" />
          </AppFormField>

          <AppFormField label="过期时间" hint="留空表示永不过期">
            <AppInput v-model="formData.expiresAt" type="datetime-local" />
          </AppFormField>
        </div>
      </form>

      <template #footer>
        <div v-if="!newlyCreatedToken" class="flex justify-end gap-2">
          <AppButton variant="secondary" @click="dialogOpen = false">取消</AppButton>
          <AppButton variant="primary" :loading="formLoading" @click="handleFormSubmit">
            {{ formLoading ? '处理中...' : (editingToken ? '保存' : '创建') }}
          </AppButton>
        </div>
        <div v-else class="flex justify-end">
          <AppButton variant="primary" @click="dialogOpen = false">完成</AppButton>
        </div>
      </template>
    </AppDialog>

    <!-- Delete Dialog -->
    <AppDialog
      :open="!!deletingToken"
      @update:open="(v: boolean) => !v && (deletingToken = null)"
      title="确认删除"
      size="sm"
    >
      <p class="text-sm text-text-secondary text-center px-5 py-4">
        确定要删除令牌「{{ deletingToken?.name }}」吗？此操作无法撤销，使用该令牌的应用将无法继续访问。
      </p>
      <template #footer>
        <div class="flex justify-center gap-2">
          <AppButton variant="secondary" @click="deletingToken = null">取消</AppButton>
          <AppButton variant="danger" :loading="deleteLoading" @click="handleDelete">
            {{ deleteLoading ? '删除中...' : '删除' }}
          </AppButton>
        </div>
      </template>
    </AppDialog>
  </div>
  </TooltipProvider>
</template>
