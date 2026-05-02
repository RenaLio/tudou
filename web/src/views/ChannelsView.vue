<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import dayjs from 'dayjs'
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogClose,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
  TooltipPortal,
  TooltipContent,
  TooltipArrow,
} from 'reka-ui'
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  setChannelStatus,
  fetchModels,
  getPlatformOptions,
  getChannelTypeLabel,
  getChannelTypeColor,
  CHANNEL_STATUS_LABELS,
  type CreateChannelRequest,
  type UpdateChannelRequest,
  type PlatformOption,
} from '@/api/channel'
import { listChannelGroups } from '@/api/channel-group'
import { formatTokens, formatNumber, calcSuccessRate } from '@/api/stats'
import type { Channel, ChannelGroup, ChannelType, ChannelStatus, ChannelExtra } from '@/types'
import {
  fadeUp,
  slideUp,
  tableRow,
  dialogContent as dialogAnim,
} from '@/utils/motion'
import AppButton from '@/components/ui/AppButton.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppBadge from '@/components/ui/AppBadge.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppFormField from '@/components/ui/AppFormField.vue'
import AppSelect from '@/components/ui/AppSelect.vue'

// Type options for filter
const typeOptions = computed(() => {
  const options = [{ value: 'all', label: '全部类型' }]
  if (platformOptions.value.length > 0) {
    for (const opt of platformOptions.value) {
      options.push({ value: opt.value, label: opt.key })
    }
  }
  return options
})

// Status options for filter
const statusOptions = computed(() => {
  return [
    { value: 'all', label: '全部状态' },
    { value: 'enabled', label: '启用', color: 'var(--color-success)' },
    { value: 'disabled', label: '禁用', color: 'var(--color-warning)' },
    { value: 'expired', label: '过期', color: 'var(--color-danger)' },
  ]
})

// Form select options - platform options from backend
const platformOptions = ref<PlatformOption[]>([])

const channelTypeOptions = computed(() => {
  return platformOptions.value.map(opt => ({ value: opt.value, label: opt.key }))
})

// Get selected platform extra info
const selectedPlatformExtra = computed(() => {
  const opt = platformOptions.value.find(o => o.value === formData.value.type)
  return opt?.extra
})

const channelStatusOptions = [
  { value: 'enabled', label: '启用' },
  { value: 'disabled', label: '禁用' },
]

// Group options for filter (computed from loaded groups)
const groupOptions = computed(() => {
  const options = [{ value: 'all', label: '全部分组' }]
  for (const group of groups.value) {
    options.push({ value: group.id, label: group.name })
  }
  return options
})

// List data
const channels = ref<Channel[]>([])
const groups = ref<ChannelGroup[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// Filters
const keyword = ref('')
const filterType = ref<ChannelType | 'all'>('all')
const filterStatus = ref<ChannelStatus | 'all'>('all')
const filterGroupID = ref<string | 'all'>('all')

// Dialog
const dialogOpen = ref(false)
const editingChannel = ref<Channel | null>(null)
const formLoading = ref(false)
const formError = ref('')
const fetchingModels = ref(false)

// Form data
const formData = ref<CreateChannelRequest>({
  type: 'openai',
  name: '',
  baseURL: '',
  apiKey: '',
  weight: 100,
  remark: '',
  tag: '',
  model: '',
  customModel: '',
  priceRate: 1,
  groupIDs: [],
  extra: {
    modelMappings: {},
  },
})

// Edit form status (separate from create)
const editStatus = ref<ChannelStatus>('enabled')

// Model management
const availableModels = ref<string[]>([])
const selectedModels = computed({
  get: () => formData.value.model ? formData.value.model.split(',').map(m => m.trim()).filter(Boolean) : [],
  set: (val) => { formData.value.model = val.join(', ') }
})
const customModels = computed({
  get: () => formData.value.customModel ? formData.value.customModel.split(',').map(m => m.trim()).filter(Boolean) : [],
  set: (val) => { formData.value.customModel = val.join(', ') }
})

function toggleModel(model: string) {
  const current = selectedModels.value
  const index = current.indexOf(model)
  if (index > -1) {
    selectedModels.value = [...current.slice(0, index), ...current.slice(index + 1)]
  } else {
    selectedModels.value = [...current, model]
  }
}

function removeCustomModel(model: string) {
  const current = customModels.value
  const index = current.indexOf(model)
  if (index > -1) {
    customModels.value = [...current.slice(0, index), ...current.slice(index + 1)]
  }
}

const newCustomModel = ref('')
function addCustomModel() {
  const model = newCustomModel.value.trim()
  if (model && !customModels.value.includes(model) && !selectedModels.value.includes(model)) {
    customModels.value = [...customModels.value, model]
    newCustomModel.value = ''
  }
}

// Model mappings
const modelMappings = ref<Array<{ key: string; value: string }>>([])

function syncMappingsToForm() {
  const mappings: Record<string, string> = {}
  for (const m of modelMappings.value) {
    if (m.key.trim()) {
      mappings[m.key.trim()] = m.value.trim()
    }
  }
  if (!formData.value.extra) {
    formData.value.extra = { modelMappings: {} }
  }
  formData.value.extra.modelMappings = mappings
}

function addModelMapping() {
  modelMappings.value = [...modelMappings.value, { key: '', value: '' }]
}

function removeModelMapping(index: number) {
  modelMappings.value = [...modelMappings.value.slice(0, index), ...modelMappings.value.slice(index + 1)]
}

function loadModelMappings(extra?: ChannelExtra) {
  const mappings = extra?.modelMappings || {}
  modelMappings.value = Object.entries(mappings).map(([key, value]) => ({ key, value }))
}

// Delete confirmation
const deletingChannel = ref<Channel | null>(null)
const deleteLoading = ref(false)

// Form tabs
const activeTab = ref<'basic' | 'models'>('basic')

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
const statsEnabled = computed(() => channels.value.filter(c => c.status === 'enabled').length)
const statsDisabled = computed(() => channels.value.filter(c => c.status === 'disabled').length)

async function loadChannels() {
  loading.value = true
  try {
    const result = await listChannels({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value || undefined,
      type: filterType.value === 'all' ? undefined : filterType.value,
      status: filterStatus.value === 'all' ? undefined : filterStatus.value,
      groupID: filterGroupID.value === 'all' ? undefined : filterGroupID.value,
      preloadGroups: true,
      preloadStats: true,
    })
    channels.value = result.items
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

async function loadPlatformOptions() {
  try {
    const result = await getPlatformOptions()
    platformOptions.value = result.options
  } catch {
    // ignore
  }
}

function onTypeChange(newType: string) {
  formData.value.type = newType as ChannelType
  // Auto-fill baseURL with exampleBaseUrl from platform extra
  const opt = platformOptions.value.find(o => o.value === newType)
  if (opt?.extra?.exampleBaseUrl && !formData.value.baseURL) {
    formData.value.baseURL = opt.extra.exampleBaseUrl
  }
}

function openCreateDialog() {
  editingChannel.value = null
  formData.value = {
    type: 'openai',
    name: '',
    baseURL: '',
    apiKey: '',
    weight: 100,
    remark: '',
    tag: '',
    model: '',
    customModel: '',
    priceRate: 1,
    groupIDs: [],
    extra: {
      modelMappings: {},
    },
  }
  availableModels.value = []
  loadModelMappings()
  activeTab.value = 'basic'
  formError.value = ''
  dialogOpen.value = true
}

function openEditDialog(channel: Channel) {
  editingChannel.value = channel
  formData.value = {
    type: channel.type,
    name: channel.name,
    baseURL: channel.baseURL,
    apiKey: channel.apiKey || '',
    weight: channel.weight,
    remark: channel.remark || '',
    tag: channel.tag || '',
    model: channel.model || '',
    customModel: channel.customModel || '',
    priceRate: channel.priceRate,
    expiredAt: channel.expiredAt || undefined,
    groupIDs: channel.groupIDs || [],
    extra: {
      modelMappings: channel.extra?.modelMappings ? { ...channel.extra.modelMappings } : {},
    },
  }
  loadModelMappings(channel.extra)
  // 解析已有模型作为可用模型
  const models = channel.model ? channel.model.split(',').map(m => m.trim()).filter(Boolean) : []
  availableModels.value = models
  editStatus.value = channel.status
  activeTab.value = 'basic'
  formError.value = ''
  dialogOpen.value = true
}

async function handleFetchModels() {
  if (!formData.value.baseURL || !formData.value.apiKey) {
    formError.value = '请先填写 Base URL 和 API Key'
    return
  }

  fetchingModels.value = true
  formError.value = ''

  try {
    const models = await fetchModels({
      type: formData.value.type,
      baseURL: formData.value.baseURL,
      apiKey: formData.value.apiKey,
    })
    availableModels.value = models
    // 自动选择所有获取到的模型
    selectedModels.value = models
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '获取模型列表失败'
    formError.value = message
  } finally {
    fetchingModels.value = false
  }
}

async function handleFormSubmit() {
  if (!formData.value.name || !formData.value.baseURL) {
    formError.value = '请填写必填字段'
    return
  }
  if (!editingChannel.value && !formData.value.apiKey) {
    formError.value = '请填写 API Key'
    return
  }

  syncMappingsToForm()

  formLoading.value = true
  formError.value = ''

  try {
    if (editingChannel.value) {
      const updateData: UpdateChannelRequest = {
        ...formData.value,
        status: editStatus.value,
      }
      if (!updateData.apiKey) delete updateData.apiKey
      await updateChannel(editingChannel.value.id, updateData)
    } else {
      await createChannel(formData.value)
    }
    dialogOpen.value = false
    await loadChannels()
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '操作失败'
    formError.value = message
  } finally {
    formLoading.value = false
  }
}

async function handleToggleStatus(channel: Channel) {
  const newStatus: ChannelStatus = channel.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await setChannelStatus(channel.id, newStatus)
    await loadChannels()
  } catch {
    // ignore
  }
}

async function handleDelete() {
  if (!deletingChannel.value) return

  deleteLoading.value = true
  try {
    await deleteChannel(deletingChannel.value.id)
    deletingChannel.value = null
    await loadChannels()
  } catch {
    // ignore
  } finally {
    deleteLoading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadChannels()
}

function getWindow3hOpacity(requests: number, maxRequests: number): number {
  if (requests === 0) return 0.04
  return 0.12 + (requests / maxRequests) * 0.78
}

function getWindow3hMaxRequests(buckets: Array<{ requestSuccess: number; requestFailed: number }>): number {
  return Math.max(...buckets.map(b => b.requestSuccess + b.requestFailed), 1)
}

function getWindow3hColor(bucket: { requestSuccess: number; requestFailed: number }, maxRequests: number): string {
  const total = bucket.requestSuccess + bucket.requestFailed
  const opacity = getWindow3hOpacity(total, maxRequests)
  const failRate = total > 0 ? bucket.requestFailed / total : 0
  if (failRate >= 0.3) {
    return `rgba(239, 83, 80, ${opacity})`
  }
  return `rgba(139, 195, 74, ${opacity})`
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadChannels()
}

function formatCost(cost: number): string {
  if (!cost) return '¥0.00'
  return `¥${cost.toFixed(4)}`
}

function sumWindow3h(buckets: Array<{ requestSuccess: number; requestFailed: number; inputToken: number; outputToken: number; totalCost: number }>) {
  return buckets.reduce(
    (acc, b) => ({
      requestSuccess: acc.requestSuccess + (b.requestSuccess || 0),
      requestFailed: acc.requestFailed + (b.requestFailed || 0),
      inputToken: acc.inputToken + (b.inputToken || 0),
      outputToken: acc.outputToken + (b.outputToken || 0),
      totalCost: acc.totalCost + (b.totalCost || 0),
    }),
    { requestSuccess: 0, requestFailed: 0, inputToken: 0, outputToken: 0, totalCost: 0 },
  )
}

onMounted(() => {
  loadChannels()
  loadGroups()
  loadPlatformOptions()
})
</script>

<template>
  <TooltipProvider>
    <div v-motion="fadeUp" class="max-w-[1600px]">
    <!-- Page Header -->
    <header class="flex items-start justify-between mb-6">
      <div class="flex flex-col gap-3">
        <span class="text-2xl font-semibold tracking-tight text-text-primary">渠道管理</span>
        <div class="flex gap-6">
          <div class="flex items-center gap-2 text-sm">
            <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
            <span class="font-semibold text-text-primary">{{ statsEnabled }}</span>
            <span class="text-text-muted">运行中</span>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <span class="w-1.5 h-1.5 rounded-full bg-warning"></span>
            <span class="font-semibold text-text-primary">{{ statsDisabled }}</span>
            <span class="text-text-muted">已暂停</span>
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
        新建渠道
      </AppButton>
    </header>

    <!-- Filter Bar -->
    <div class="flex gap-3 mb-4 p-4 bg-bg-card rounded-lg border border-border">
      <div class="flex gap-2 flex-1">
        <AppInput
          v-model="keyword"
          type="text"
          placeholder="搜索渠道名称..."
          size="sm"
          class="flex-1 max-w-[280px]"
          @keyup.enter="handleSearch"
        />
        <div class="w-[130px]">
          <AppSelect
            v-model="filterType"
            :options="typeOptions"
            size="sm"
            @update:modelValue="handleSearch"
          />
        </div>
        <div class="w-[130px]">
          <AppSelect
            v-model="filterStatus"
            :options="statusOptions"
            size="sm"
            @update:modelValue="handleSearch"
          />
        </div>
        <div class="w-[130px]">
          <AppSelect
            v-model="filterGroupID"
            :options="groupOptions"
            size="sm"
            @update:modelValue="handleSearch"
          />
        </div>
      </div>
      <AppButton variant="secondary" size="sm" @click="handleSearch">搜索</AppButton>
    </div>

    <!-- Data Table -->
    <div class="bg-bg-card rounded-lg border border-border">
      <table class="w-full border-collapse">
        <thead>
          <tr>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">渠道名称</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">类型</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">状态</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">权重</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">用量统计</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap">近 3h</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">分组</th>
            <th class="px-4 py-3.5 text-left text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">创建时间</th>
            <th class="px-4 py-3.5 text-right text-xs font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="9" class="px-4 py-12 text-center text-text-muted">
              <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin mx-auto mb-2"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="channels.length === 0">
            <td colspan="9" class="px-4 py-12 text-center text-text-muted">
              <div class="text-3xl mb-2 opacity-30">○</div>
              <span>暂无渠道数据</span>
            </td>
          </tr>
          <tr v-else v-for="(channel, index) in channels" :key="channel.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="transition-colors duration-150 hover:bg-bg-secondary">
            <td class="px-4 py-4 border-b border-border align-middle">
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-md flex items-center justify-center text-sm font-semibold text-white" :class="getChannelTypeColor(channel.type).bg">
                  {{ channel.name.charAt(0) }}
                </div>
                <div class="flex flex-col gap-0.5">
                  <span class="font-medium text-text-primary">{{ channel.name }}</span>
                  <span class="text-xs font-mono text-text-muted max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">{{ channel.baseURL }}</span>
                </div>
              </div>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <span class="inline-block px-2 py-0.5 rounded text-xs font-medium" :class="`${getChannelTypeColor(channel.type).badge} ${getChannelTypeColor(channel.type).text}`">
                {{ getChannelTypeLabel(channel.type, platformOptions) }}
              </span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <AppBadge
                :variant="channel.status === 'enabled' ? 'success' : channel.status === 'disabled' ? 'warning' : 'danger'"
                size="md"
                pulse
              >
                {{ CHANNEL_STATUS_LABELS[channel.status]?.label }}
              </AppBadge>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <span class="text-sm text-text-secondary">{{ channel.weight }}</span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <TooltipRoot v-if="channel.stats">
                <TooltipTrigger as-child>
                  <span class="text-sm text-text-secondary font-medium cursor-help">{{ formatCost(channel.stats.totalCost) }}</span>
                </TooltipTrigger>
                <TooltipPortal to="body">
                  <TooltipContent side="bottom" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary whitespace-nowrap">
                    <div class="flex flex-col gap-1">
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">请求</span>
                        <span class="font-medium">{{ formatNumber(channel.stats.requestSuccess + channel.stats.requestFailed) }}</span>
                        <span class="text-[10px] px-1 rounded bg-bg-tertiary" :class="{ 'bg-success-light text-success': calcSuccessRate(channel.stats.requestSuccess, channel.stats.requestFailed) >= 95 }">
                          {{ calcSuccessRate(channel.stats.requestSuccess, channel.stats.requestFailed) }}%
                        </span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">成功</span>
                        <span class="font-medium text-success">{{ formatNumber(channel.stats.requestSuccess) }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">失败</span>
                        <span class="font-medium text-danger">{{ formatNumber(channel.stats.requestFailed) }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">输入</span>
                        <span class="font-medium">{{ formatTokens(channel.stats.inputToken) }}</span>
                      </div>
                      <div class="flex gap-2">
                        <span class="text-text-muted min-w-[40px]">输出</span>
                        <span class="font-medium">{{ formatTokens(channel.stats.outputToken) }}</span>
                      </div>
                    </div>
                    <template v-if="channel.stats.window3h?.buckets?.length">
                      <div class="border-t border-border my-1.5"></div>
                      <div class="text-[10px] text-text-muted uppercase tracking-wider mb-0.5">近 3h</div>
                      <div class="flex flex-col gap-1">
                        <div class="flex gap-2">
                          <span class="text-text-muted min-w-[40px]">请求</span>
                          <span class="font-medium">{{ formatNumber(sumWindow3h(channel.stats.window3h.buckets).requestSuccess + sumWindow3h(channel.stats.window3h.buckets).requestFailed) }}</span>
                        </div>
                        <div class="flex gap-2">
                          <span class="text-text-muted min-w-[40px]">输入</span>
                          <span class="font-medium">{{ formatTokens(sumWindow3h(channel.stats.window3h.buckets).inputToken) }}</span>
                        </div>
                        <div class="flex gap-2">
                          <span class="text-text-muted min-w-[40px]">输出</span>
                          <span class="font-medium">{{ formatTokens(sumWindow3h(channel.stats.window3h.buckets).outputToken) }}</span>
                        </div>
                        <div class="flex gap-2">
                          <span class="text-text-muted min-w-[40px]">费用</span>
                          <span class="font-medium">{{ formatCost(sumWindow3h(channel.stats.window3h.buckets).totalCost) }}</span>
                        </div>
                      </div>
                    </template>
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
              <span v-else class="text-sm text-text-muted">—</span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <div v-if="channel.stats?.window3h?.buckets?.length" class="flex items-center gap-[2px]">
                <div
                  v-for="(bucket, idx) in channel.stats.window3h.buckets"
                  :key="idx"
                  class="w-2 h-4 rounded-[2px] border border-[rgba(139,195,74,0.06)] box-border relative cursor-pointer group z-[1] hover:z-10 transition-transform duration-150 hover:scale-125"
                  :style="{ background: getWindow3hColor(bucket, getWindow3hMaxRequests(channel.stats.window3h.buckets)) }"
                >
                  <div class="hidden group-hover:block absolute bottom-[calc(100%+6px)] left-1/2 -translate-x-1/2 bg-bg-secondary border border-border-hover rounded-md px-3 py-2 min-w-[180px] pointer-events-none shadow-[0_8px_24px_rgba(0,0,0,0.25)] z-[100]">
                    <div class="text-xs font-semibold text-text-primary mb-1 font-mono">{{ dayjs(bucket.startAt).format('HH:mm') }} - {{ dayjs(bucket.endAt).format('HH:mm') }}</div>
                    <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                      <span class="text-text-muted">请求</span>
                      <span class="text-text-primary font-mono font-medium">{{ bucket.requestSuccess + bucket.requestFailed }} <span class="text-text-tertiary font-normal text-[0.625rem]">(成功 {{ bucket.requestSuccess }} / 失败 {{ bucket.requestFailed }})</span></span>
                    </div>
                    <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                      <span class="text-text-muted">Token</span>
                      <span class="text-text-primary font-mono font-medium">{{ formatTokens(bucket.inputToken + bucket.outputToken) }}</span>
                    </div>
                    <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                      <span class="text-text-muted">费用</span>
                      <span class="text-accent font-mono font-semibold">{{ formatCost(bucket.totalCost) }}</span>
                    </div>
                  </div>
                </div>
              </div>
              <span v-else class="text-xs text-text-muted">-</span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <div v-if="channel.groups?.length" class="flex flex-wrap gap-1">
                <span
                  v-for="g in channel.groups"
                  :key="g.id"
                  class="inline-block px-2 py-0.5 bg-bg-tertiary rounded text-[11px] text-text-secondary"
                >
                  {{ g.name }}
                </span>
              </div>
              <span v-else class="text-sm text-text-muted">-</span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle">
              <span class="text-sm text-text-secondary">{{ dayjs(channel.createdAt).format('YYYY-MM-DD') }}</span>
            </td>
            <td class="px-4 py-4 border-b border-border align-middle text-right">
              <div class="flex justify-end gap-2">
                <AppButton
                  :variant="channel.status === 'enabled' ? 'warning' : 'success'"
                  size="sm"
                  @click="handleToggleStatus(channel)"
                >
                  {{ channel.status === 'enabled' ? '暂停' : '启用' }}
                </AppButton>
                <AppButton variant="ghost" size="sm" @click="openEditDialog(channel)">编辑</AppButton>
                <AppButton variant="danger" size="sm" @click="deletingChannel = channel">删除</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3.5 border-t border-border">
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

    <!-- Dialog -->
    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay
          v-motion
          :initial="{ opacity: 0 }"
          :enter="{ opacity: 1, transition: { duration: 200 } }"
          :leave="{ opacity: 0, transition: { duration: 150 } }"
          class="fixed inset-0 bg-black/60 backdrop-blur-sm z-40"
        />
        <DialogContent
          v-motion="dialogAnim"
          class="fixed inset-0 m-auto w-[92%] max-w-[620px] h-fit max-h-[88vh] bg-gradient-to-b from-[var(--color-bg-card)] to-[var(--color-bg-primary)] backdrop-blur-2xl rounded-2xl border border-primary/15 flex flex-col z-50 overflow-hidden shadow-lg ring-1 ring-primary/10"
        >
          <!-- Scanline texture overlay -->
          <div class="pointer-events-none absolute inset-0 opacity-[0.02]" style="background: repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(var(--color-primary-rgb), 0.12) 2px, rgba(var(--color-primary-rgb), 0.12) 4px);"></div>

          <div class="relative flex items-center justify-between px-6 py-5 border-b border-primary/10 bg-gradient-to-r from-primary/5 via-transparent to-transparent">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-xl bg-primary-light flex items-center justify-center text-primary shadow-glow-primary ring-1 ring-primary/20">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                  <path d="M2 17l10 5 10-5"/>
                  <path d="M2 12l10 5 10-5"/>
                </svg>
              </div>
              <div class="flex flex-col">
                <DialogTitle class="text-lg font-semibold text-text-primary tracking-tight">
                  {{ editingChannel ? '编辑渠道' : '新建渠道' }}
                </DialogTitle>
                <span class="text-[11px] text-text-muted font-mono tracking-wider uppercase">{{ editingChannel ? 'CHANNEL EDIT' : 'CHANNEL CREATE' }}</span>
              </div>
            </div>
            <DialogClose class="inline-flex items-center justify-center w-8 h-8 rounded-lg text-text-muted hover:text-text-secondary hover:bg-bg-tertiary transition-all duration-200 hover:rotate-90">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </DialogClose>
          </div>

          <DialogDescription class="sr-only">
            {{ editingChannel ? '编辑渠道配置' : '创建新渠道' }}
          </DialogDescription>

          <div class="relative flex px-6 py-3 bg-bg-secondary/60 border-b border-primary/10">
            <div class="flex gap-1 p-1 bg-bg-tertiary/60 rounded-xl">
              <button
                v-for="tab in ['basic', 'models'] as const"
                :key="tab"
                class="px-5 py-2 rounded-lg text-[13px] font-medium cursor-pointer relative transition-all duration-300"
                :class="activeTab === tab
                  ? 'bg-primary-light text-primary shadow-glow-primary ring-1 ring-primary/20'
                  : 'text-text-muted hover:text-text-secondary hover:bg-bg-secondary/50'"
                @click="activeTab = tab"
              >
                {{ tab === 'basic' ? '基本信息' : '模型配置' }}
              </button>
            </div>
          </div>

          <form @submit.prevent="handleFormSubmit" class="flex-1 overflow-y-auto p-6">
            <!-- Error Banner -->
            <div
              v-if="formError"
              class="px-4 py-3 bg-danger-light/60 text-danger rounded-lg text-[13px] mb-5 border border-danger/20 shadow-[0_0_16px_rgba(229,115,115,0.08)] relative overflow-hidden"
            >
              <div class="absolute left-0 top-0 bottom-0 w-[3px] bg-danger/60"></div>
              <div class="flex items-center gap-2">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                {{ formError }}
              </div>
            </div>

            <div v-show="activeTab === 'basic'" class="flex flex-col gap-5">
              <!-- Section: Identity -->
              <div>
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-1 h-3.5 rounded-full bg-primary/60"></div>
                  <span class="text-[11px] font-mono tracking-wider uppercase text-text-muted">身份标识</span>
                  <div class="flex-1 h-px bg-border/40"></div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <AppFormField label="渠道类型" required>
                    <AppSelect :model-value="formData.type" :options="channelTypeOptions" @update:model-value="onTypeChange" />
                  </AppFormField>
                  <AppFormField label="渠道名称" required>
                    <AppInput v-model="formData.name" type="text" placeholder="如: OpenAI 主账号" />
                  </AppFormField>
                  <AppFormField label="标签" class="col-span-2">
                    <AppInput v-model="formData.tag" type="text" placeholder="用于分类" />
                  </AppFormField>
                </div>
              </div>

              <!-- Section: Connection -->
              <div>
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-1 h-3.5 rounded-full bg-primary/60"></div>
                  <span class="text-[11px] font-mono tracking-wider uppercase text-text-muted">连接配置</span>
                  <div class="flex-1 h-px bg-border/40"></div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <AppFormField label="Base URL" required class="col-span-2">
                    <AppInput v-model="formData.baseURL" type="text" placeholder="https://api.openai.com/v1" autocomplete="off" />
                    <!-- Platform paths info -->
                    <div v-if="selectedPlatformExtra?.paths" class="mt-2 p-2.5 bg-bg-tertiary/50 rounded-lg border border-border/50">
                      <div class="flex items-center gap-1.5 mb-1.5">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-text-muted">
                          <path d="M12 2L2 7l10 5 10-5-10-5z" />
                          <path d="M2 17l10 5 10-5" />
                        </svg>
                        <span class="text-[10px] font-mono text-text-muted uppercase tracking-wider">示例请求路径</span>
                      </div>
                      <div class="flex flex-col gap-1">
                        <div v-for="(path, ability) in selectedPlatformExtra.paths" :key="ability" class="flex items-center gap-2 text-xs">
                          <span class="px-1.5 py-0.5 bg-primary-light text-primary rounded text-[10px] font-mono">{{ ability }}</span>
                          <code class="text-text-secondary font-mono text-[11px]">{{ (formData.baseURL || '').replace(/\/+$/, '') }}{{ path }}</code>
                        </div>
                      </div>
                    </div>
                  </AppFormField>
                  <AppFormField label="API Key" :required="!editingChannel" class="col-span-2">
                    <AppInput
                      v-model="formData.apiKey"
                      type="text"
                      :placeholder="editingChannel ? '留空保持不变' : 'sk-...'"
                      autocomplete="off"
                    />
                  </AppFormField>
                </div>
              </div>

              <!-- Section: Parameters -->
              <div>
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-1 h-3.5 rounded-full bg-primary/60"></div>
                  <span class="text-[11px] font-mono tracking-wider uppercase text-text-muted">参数调节</span>
                  <div class="flex-1 h-px bg-border/40"></div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <AppFormField label="权重">
                    <AppInput v-model.number="formData.weight" type="number" min="0" />
                  </AppFormField>
                  <AppFormField v-if="editingChannel" label="状态">
                    <AppSelect v-model="editStatus" :options="channelStatusOptions" />
                  </AppFormField>
                  <AppFormField label="价格倍率">
                    <AppInput v-model.number="formData.priceRate" type="number" step="0.01" min="0" />
                  </AppFormField>
                </div>
              </div>

              <!-- Section: Extended -->
              <div>
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-1 h-3.5 rounded-full bg-primary/60"></div>
                  <span class="text-[11px] font-mono tracking-wider uppercase text-text-muted">扩展信息</span>
                  <div class="flex-1 h-px bg-border/40"></div>
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <AppFormField label="备注" class="col-span-2">
                    <textarea
                      v-model="formData.remark"
                      rows="2"
                      placeholder="渠道备注"
                      class="w-full px-3 py-2.5 bg-bg-secondary text-text-primary placeholder:text-text-muted border border-border rounded-lg text-sm transition-all duration-200 focus:outline-none focus:border-border-focus focus:shadow-[0_0_12px_rgba(139,195,74,0.08)] focus:ring-1 focus:ring-primary/20 resize-y min-h-[68px]"
                    ></textarea>
                  </AppFormField>

                  <AppFormField label="所属分组" class="col-span-2">
                    <div class="flex flex-wrap gap-2">
                      <label
                        v-for="group in groups"
                        :key="group.id"
                        class="px-3 py-1.5 bg-bg-secondary border border-border/60 rounded-lg text-[13px] text-text-secondary cursor-pointer transition-all duration-200 hover:border-border-hover"
                        :class="formData.groupIDs?.includes(group.id)
                          ? 'bg-primary-light/60 border-primary/50 text-primary shadow-[0_0_10px_rgba(139,195,74,0.1)]'
                          : ''
                        "
                      >
                        <input
                          type="checkbox"
                          :checked="formData.groupIDs?.includes(group.id)"
                          class="sr-only"
                          @change="() => {
                            if (!formData.groupIDs) formData.groupIDs = []
                            const idx = formData.groupIDs.indexOf(group.id)
                            if (idx > -1) formData.groupIDs.splice(idx, 1)
                            else formData.groupIDs.push(group.id)
                          }"
                        />
                        <span class="flex items-center gap-1.5">
                          <span
                            class="w-1.5 h-1.5 rounded-full transition-all duration-200"
                            :class="formData.groupIDs?.includes(group.id) ? 'bg-primary shadow-[0_0_6px_rgba(139,195,74,0.6)]' : 'bg-text-muted/40'"
                          ></span>
                          {{ group.name }}
                        </span>
                      </label>
                      <span v-if="groups.length === 0" class="text-[13px] text-text-muted italic">暂无分组</span>
                    </div>
                  </AppFormField>
                </div>
              </div>
            </div>

            <div v-show="activeTab === 'models'" class="flex flex-col gap-5">
              <div>
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-1 h-3.5 rounded-full bg-primary/60"></div>
                  <span class="text-[11px] font-mono tracking-wider uppercase text-text-muted">模型配置</span>
                  <div class="flex-1 h-px bg-border/40"></div>
                </div>

                <div class="flex items-center justify-between mb-1">
                  <span class="text-sm text-text-secondary">从 API 端点拉取模型列表</span>
                  <AppButton
                    type="button"
                    variant="primary"
                    size="sm"
                    :loading="fetchingModels"
                    :disabled="!formData.baseURL || !formData.apiKey"
                    @click="handleFetchModels"
                  >
                    {{ fetchingModels ? '获取中...' : '自动获取' }}
                  </AppButton>
                </div>
                <p class="text-xs text-text-muted mb-4">点击"自动获取"从 API 拉取模型列表，点击模型芯片选择/取消</p>

                <!-- Unified Model Tags -->
                <div class="p-4 bg-bg-secondary/50 rounded-xl border border-border/50">
                  <div class="flex items-center gap-2 mb-3">
                    <span class="px-2 py-0.5 rounded-md text-[11px] font-medium uppercase bg-primary-light text-primary border border-primary/20">模型列表</span>
                    <span class="text-xs text-text-muted font-mono">{{ selectedModels.length + customModels.length }} 已选</span>
                  </div>

                  <!-- Model tags scrollable area -->
                  <div class="max-h-40 overflow-y-auto flex flex-col gap-2">
                    <!-- Selected tags: system (green) + custom (amber) -->
                    <div class="flex flex-wrap gap-1.5">
                      <!-- System models (selected) -->
                      <button
                        v-for="model in selectedModels"
                        :key="'sel-' + model"
                        type="button"
                        class="inline-flex items-center gap-1 px-2 py-1 rounded-md font-mono text-[11px] cursor-pointer transition-all duration-200 border bg-primary text-[#080a08] border-primary shadow-[0_0_8px_rgba(139,195,74,0.2)] hover:shadow-[0_0_12px_rgba(139,195,74,0.3)] active:scale-[0.96]"
                        @click="toggleModel(model)"
                      >
                        {{ model }}
                        <span class="inline-flex items-center justify-center w-3 h-3 rounded-full bg-[#080a08]/20 text-[#080a08] text-[9px] leading-none">×</span>
                      </button>

                      <!-- Custom models (amber) -->
                      <span
                        v-for="model in customModels"
                        :key="'cust-' + model"
                        class="inline-flex items-center gap-1 px-2 py-1 rounded-md font-mono text-[11px] bg-warning text-[#080a08] border border-warning shadow-[0_0_8px_rgba(255,213,79,0.15)] transition-all duration-200 hover:shadow-[0_0_12px_rgba(255,213,79,0.25)]"
                      >
                        {{ model }}
                        <button
                          type="button"
                          class="inline-flex items-center justify-center w-3 h-3 rounded-full bg-[#080a08]/15 text-[#080a08] text-[9px] leading-none cursor-pointer transition-all duration-150 hover:bg-[#080a08]/30"
                          @click="removeCustomModel(model)"
                        >
                          ×
                        </button>
                      </span>
                    </div>

                    <!-- Unselected available models -->
                    <div
                      v-if="availableModels.filter(m => !selectedModels.includes(m)).length > 0"
                      class="flex flex-wrap gap-1.5 pt-2 border-t border-border/40"
                    >
                      <button
                        v-for="model in availableModels.filter(m => !selectedModels.includes(m))"
                        :key="model"
                        type="button"
                        class="px-2 py-1 rounded-md font-mono text-[11px] cursor-pointer transition-all duration-200 border bg-[rgba(139,195,74,0.06)] text-primary border-[rgba(139,195,74,0.25)] hover:border-primary/60 hover:bg-[rgba(139,195,74,0.1)] active:scale-[0.96]"
                        @click="toggleModel(model)"
                      >
                        {{ model }}
                      </button>
                    </div>

                    <span v-if="availableModels.length === 0 && selectedModels.length === 0 && customModels.length === 0" class="text-[12px] text-text-muted italic py-1">
                      请先填写 Base URL 和 API Key，然后点击"自动获取"
                    </span>
                  </div>

                  <!-- Add custom model input -->
                  <div class="flex gap-2 mt-3 pt-3 border-t border-border/40">
                    <AppInput
                      v-model="newCustomModel"
                      type="text"
                      placeholder="输入模型名称，回车添加"
                      size="sm"
                      @keyup.enter="addCustomModel"
                    />
                    <AppButton type="button" variant="warning" size="sm" @click="addCustomModel">添加</AppButton>
                  </div>
                </div>

                <!-- Model Mappings: Data Pipeline -->
                <div class="mt-4 p-4 bg-bg-secondary/50 rounded-xl border border-border/50">
                  <div class="flex items-center gap-2 mb-3">
                    <span class="px-2 py-0.5 rounded-md text-[11px] font-medium uppercase bg-[rgba(139,195,74,0.08)] text-primary border border-primary/20">模型映射</span>
                    <span class="text-xs text-text-muted font-mono">{{ modelMappings.length }} 条</span>
                  </div>
                  <div class="flex flex-col gap-2.5">
                    <div
                      v-if="modelMappings.length === 0"
                      class="text-[13px] text-text-muted italic py-2 px-3 border border-dashed border-border/50 rounded-lg"
                    >
                      暂无映射，点击下方按钮添加数据流
                    </div>
                    <div
                      v-for="(m, index) in modelMappings"
                      :key="index"
                      class="group flex items-center gap-2 p-2.5 bg-bg-secondary/60 border border-border/60 rounded-lg transition-all duration-200 hover:border-primary/30 hover:shadow-[0_0_12px_rgba(139,195,74,0.06)] relative overflow-hidden"
                    >
                      <div class="absolute left-0 top-0 bottom-0 w-[2px] bg-primary/40 group-hover:bg-primary/70 transition-colors"></div>
                      <AppInput
                        v-model="m.key"
                        type="text"
                        placeholder="请求模型"
                        size="sm"
                        class="flex-1"
                      />
                      <div class="flex items-center justify-center w-6 h-6 rounded-md bg-bg-tertiary border border-border/50 shrink-0">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-text-muted">
                          <path d="M5 12h14M12 5l7 7-7 7" />
                        </svg>
                      </div>
                      <AppInput
                        v-model="m.value"
                        type="text"
                        placeholder="实际请求模型"
                        size="sm"
                        class="flex-1"
                      />
                      <AppButton
                        variant="ghost"
                        size="sm"
                        class="hover:text-danger hover:bg-danger-light shrink-0 opacity-60 group-hover:opacity-100 transition-opacity"
                        @click="removeModelMapping(index)"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <line x1="18" y1="6" x2="6" y2="18" />
                          <line x1="6" y1="6" x2="18" y2="18" />
                        </svg>
                      </AppButton>
                    </div>
                    <AppButton
                      variant="secondary"
                      size="sm"
                      class="self-start border-dashed border-border-hover hover:border-primary/40 hover:bg-primary-light/40 hover:text-primary transition-all duration-200"
                      @click="addModelMapping"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="inline-block mr-1">
                        <line x1="12" y1="5" x2="12" y2="19" />
                        <line x1="5" y1="12" x2="19" y2="12" />
                      </svg>
                      添加映射
                    </AppButton>
                  </div>
                </div>
              </div>
            </div>
          </form>

          <!-- Footer -->
          <div class="flex justify-end gap-3 px-6 py-4 border-t border-primary/10 bg-bg-secondary/30">
            <DialogClose as-child>
              <button
                type="button"
                class="px-5 py-2 rounded-lg text-sm font-medium text-text-secondary bg-bg-tertiary/60 border border-border/60 hover:border-border-hover hover:text-text-primary hover:bg-bg-tertiary transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-primary/20"
              >
                取消
              </button>
            </DialogClose>
            <button
              type="button"
              :disabled="formLoading"
              class="px-5 py-2 rounded-lg text-sm font-medium bg-primary text-[#080a08] hover:bg-primary-dark hover:shadow-[0_0_24px_rgba(139,195,74,0.3)] active:scale-[0.97] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-primary/40 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-1.5"
              @click="handleFormSubmit"
            >
              <span v-if="formLoading" class="inline-block w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
              {{ formLoading ? '处理中...' : (editingChannel ? '保存配置' : '创建渠道') }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- Delete Dialog -->
    <AppDialog
      :open="!!deletingChannel"
      @update:open="(v: boolean) => !v && (deletingChannel = null)"
      title="确认删除"
      description="确定要删除「{{ deletingChannel?.name }}」吗？此操作无法撤销。"
      size="sm"
    >
      <template #footer>
        <AppButton variant="secondary" @click="deletingChannel = null">取消</AppButton>
        <AppButton variant="danger" :loading="deleteLoading" @click="handleDelete">
          {{ deleteLoading ? '删除中...' : '删除' }}
        </AppButton>
      </template>
    </AppDialog>
  </div>
  </TooltipProvider>
</template>
