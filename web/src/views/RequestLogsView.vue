<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import {
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
  TooltipPortal,
  TooltipContent,
  TooltipArrow,
} from 'reka-ui'
import {
  listRequestLogs,
  formatMicrosCost,
  type RequestLogResponse,
  type RequestLogStatus,
} from '@/api/request-log'
import { fadeUp, staggerItem } from '@/utils/motion'
import AppButton from '@/components/ui/AppButton.vue'
import AppBadge from '@/components/ui/AppBadge.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppSelect from '@/components/ui/AppSelect.vue'

// ── Data ──
const logs = ref<RequestLogResponse[]>([])
const loading = ref(false)
const loadError = ref('')

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'fail' },
]

const streamOptions = [
  { label: '全部流式', value: 'all' },
  { label: '仅流式', value: 'true' },
  { label: '仅非流式', value: 'false' },
]

const orderOptions = [
  { label: '时间倒序', value: 'created_desc' },
  { label: '时间正序', value: 'created_asc' },
]

const keyword = ref('')
const status = ref<RequestLogStatus | 'all'>('all')
const streamMode = ref<'all' | 'true' | 'false'>('all')
const dateFrom = ref('')
const dateTo = ref('')
const orderMode = ref<'created_desc' | 'created_asc'>('created_desc')

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const expandedId = ref<string | null>(null)
const copiedId = ref<string | null>(null)

// ── Modal ──
const selectedLog = ref<RequestLogResponse | null>(null)
const showModal = ref(false)

// ── ID Search ──
const idSearch = ref('')
const idSearchError = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

// ── Helpers ──
function mapOrderBy() {
  return orderMode.value === 'created_asc'
    ? 'created_at ASC, id ASC'
    : 'created_at DESC, id DESC'
}

import dayjs from 'dayjs'

function toRFC3339(value: string) {
  if (!value) return undefined
  const d = dayjs(value)
  if (!d.isValid()) return undefined
  return d.toISOString()
}

function formatStatus(s: RequestLogStatus) {
  return s === 'success' ? '成功' : '失败'
}

function formatTokens(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toString()
}

function truncate(str: string | undefined, len: number) {
  if (!str) return '-'
  return str.length > len ? str.slice(0, len) + '…' : str
}

function fmtSeconds(ms: number | undefined) {
  if (ms == null) return '-'
  return (ms / 1000).toFixed(2)
}

function totalMs(a: number | undefined, b: number | undefined): number | undefined {
  if (a == null || b == null) return undefined
  return a + b
}

function formatDate(value: string): { date: string; time: string; full: string } {
  const d = dayjs(value)
  if (!d.isValid()) return { date: value, time: '', full: value }
  const date = d.format('YYYY-MM-DD')
  const time = d.format('HH:mm:ss')
  return { date, time, full: `${date} ${time}` }
}

function safeJSONStringify(data: unknown) {
  return JSON.stringify(data ?? {}, null, 2)
}

function tokenBarSegments(log: RequestLogResponse) {
  const total = log.inputToken + log.outputToken + log.cachedReadInputTokens + log.cachedCreationInputTokens
  if (total === 0) {
    return { input: 0, output: 0, cachedRead: 0, cachedCreate: 0, total: 0 }
  }
  return {
    input: (log.inputToken / total) * 100,
    output: (log.outputToken / total) * 100,
    cachedRead: (log.cachedReadInputTokens / total) * 100,
    cachedCreate: (log.cachedCreationInputTokens / total) * 100,
    total,
  }
}

async function copyText(text: string, id: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedId.value = id
    setTimeout(() => { copiedId.value = null }, 1500)
  } catch { /* noop */ }
}

function toggleDetail(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

function openModal(log: RequestLogResponse) {
  selectedLog.value = log
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  selectedLog.value = null
}

function searchById() {
  const id = idSearch.value.trim()
  if (!id) {
    idSearchError.value = '请输入日志 ID'
    return
  }
  const found = logs.value.find(log => log.id === id)
  if (found) {
    idSearchError.value = ''
    openModal(found)
  } else {
    idSearchError.value = '当前页面未找到该日志，请检查 ID 或切换分页'
  }
}

// ── Load ──
async function loadLogs() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await listRequestLogs({
      page: page.value,
      pageSize: pageSize.value,
      orderBy: mapOrderBy(),
      keyword: keyword.value.trim() || undefined,
      status: status.value === 'all' ? undefined : status.value,
      isStream: streamMode.value === 'all' ? undefined : streamMode.value === 'true',
      dateFrom: toRFC3339(dateFrom.value),
      dateTo: toRFC3339(dateTo.value),
    })
    logs.value = res.items
    total.value = res.total
  } catch (err: any) {
    loadError.value = err?.response?.data?.message || '加载请求日志失败'
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadLogs()
}

function handleReset() {
  keyword.value = ''
  status.value = 'all'
  streamMode.value = 'all'
  dateFrom.value = ''
  dateTo.value = ''
  orderMode.value = 'created_desc'
  page.value = 1
  loadLogs()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  loadLogs()
}

onMounted(() => {
  loadLogs()
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})
</script>

<template>
  <TooltipProvider>
    <div v-motion="fadeUp" class="relative max-w-[1600px]">
      <!-- Noise texture overlay -->
      <div class="fixed inset-0 pointer-events-none opacity-[0.025] z-0 bg-repeat bg-[length:256px_256px]" style="background-image: url(&quot;data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E&quot;);"></div>

      <!-- Header -->
      <header class="flex items-start justify-between gap-6 mb-6 flex-wrap">
        <div class="flex flex-col gap-1.5">
          <h1 class="m-0 text-[1.625rem] font-bold text-text-primary tracking-tight flex items-center gap-2">
            <span class="text-primary font-mono text-xl font-normal" style="text-shadow: 0 0 12px var(--color-primary-glow);">›</span>
            请求日志
          </h1>
          <div class="flex items-center gap-2 text-xs">
            <span class="flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full bg-success shadow-[0_0_8px_var(--color-success)] animate-[pulse_2.5s_ease-in-out_infinite]"></span>
              <span class="text-success font-medium font-mono">实时流</span>
            </span>
            <span class="text-text-muted font-mono">/</span>
            <span class="text-text-muted font-mono">{{ total }} 条记录</span>
          </div>
        </div>

        <!-- ID Search -->
        <div class="flex flex-col gap-1.5">
          <div class="flex items-center bg-bg-card border border-border rounded-md overflow-hidden transition-colors duration-200 focus-within:border-border-focus focus-within:shadow-[0_0_0_3px_rgba(139,195,74,0.08),var(--shadow-glow-primary)]">
            <span class="pl-3 pr-2 text-primary font-mono text-sm font-semibold">#</span>
            <input
              v-model="idSearch"
              type="text"
              class="flex-1 min-w-[180px] h-[38px] bg-transparent border-none text-text-primary font-mono text-[0.8125rem] outline-none px-2 placeholder:text-text-muted"
              placeholder="日志 ID 精确查询"
              @keyup.enter="searchById"
            >
            <AppButton variant="ghost" size="sm" class="h-[38px] w-[38px] rounded-none border-l border-border bg-primary-light text-primary hover:bg-[rgba(139,195,74,0.2)]" @click="searchById">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <circle cx="11" cy="11" r="8" />
                <line x1="21" y1="21" x2="16.65" y2="16.65" />
              </svg>
            </AppButton>
          </div>
          <p v-if="idSearchError" class="m-0 text-[0.6875rem] text-danger pl-1">{{ idSearchError }}</p>
        </div>
      </header>

      <!-- Detail Modal -->
      <AppDialog
        :open="!!selectedLog"
        title="请求日志详情"
        :description="selectedLog ? `日志 ${selectedLog.id} 的详细信息` : ''"
        size="xl"
        show-close
        @update:open="(v: boolean) => !v && (selectedLog = null)"
      >
        <div v-if="selectedLog">
          <!-- Modal Header -->
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <span class="text-[0.625rem] font-bold text-primary tracking-widest font-mono">LOG</span>
              <code class="font-mono text-[0.8125rem] text-text-primary bg-bg-secondary px-2 py-0.5 rounded border border-border">{{ selectedLog.id }}</code>
              <button
                class="w-6 h-6 inline-flex items-center justify-center bg-transparent border border-border rounded text-text-muted transition-all duration-150 hover:border-primary hover:text-primary"
                :class="{ 'border-success text-success': copiedId === selectedLog.id }"
                @click="copyText(selectedLog.id, selectedLog.id)"
              >
                <svg v-if="copiedId !== selectedLog.id" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="9" y="9" width="13" height="13" rx="2" />
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
                <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Modal Body -->
          <div class="overflow-auto">
            <!-- Info bar: status + time + stream + transfer -->
            <div class="flex items-center gap-2 mb-4 flex-wrap">
              <AppBadge :variant="selectedLog.status === 'success' ? 'success' : 'danger'" size="sm">
                {{ formatStatus(selectedLog.status) }}
              </AppBadge>
              <span class="w-px h-3.5 bg-border"></span>
              <TooltipRoot>
                <TooltipTrigger as-child>
                  <span class="text-xs text-text-muted font-mono cursor-help">{{ formatDate(selectedLog.createdAt).full }}</span>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                    {{ selectedLog.createdAt }}
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
              <span class="w-px h-3.5 bg-border"></span>
              <span class="inline-flex items-center gap-1 text-[0.6875rem] text-text-muted font-mono" :class="{ 'text-primary': selectedLog.isStream }">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M4 14h6v6H4z" />
                  <path d="M4 4h6v6H4z" opacity="0.5" />
                  <path d="M14 8h6v6h-6z" opacity="0.5" />
                  <path d="M14 18h6v-6h-6z" opacity="0.3" />
                </svg>
                {{ selectedLog.isStream ? '流式' : '非流式' }}
              </span>
              <span class="w-px h-3.5 bg-border"></span>
              <span class="inline-flex items-center gap-1 text-[0.6875rem] text-text-muted font-mono">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <polyline points="12 6 12 12 16 14" />
                </svg>
                {{ fmtSeconds(selectedLog.transferTime) }}s
              </span>
            </div>

            <!-- Model highlight -->
            <div class="flex items-center gap-3 p-4 rounded-md mb-4 border border-border-hover" style="background: linear-gradient(135deg, rgba(139,195,74,0.06), rgba(255,179,0,0.04));">
              <div class="flex-1 flex flex-col gap-1.5 min-w-0">
                <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">请求模型</span>
                <code class="text-sm text-text-primary font-mono font-semibold break-all">{{ selectedLog.model || '-' }}</code>
              </div>
              <div class="text-primary flex-shrink-0 flex items-center opacity-60">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <line x1="5" y1="12" x2="19" y2="12" />
                  <polyline points="12 5 19 12 12 19" />
                </svg>
              </div>
              <div class="flex-1 flex flex-col gap-1.5 min-w-0">
                <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">上游模型</span>
                <code class="text-sm text-text-primary font-mono font-semibold break-all">{{ selectedLog.upstreamModel || '-' }}</code>
              </div>
            </div>

            <!-- Provider card -->
            <TooltipRoot v-if="selectedLog.providerDetail?.provider">
              <TooltipTrigger as-child>
                <div class="mb-4 px-3.5 py-2.5 bg-bg-secondary border border-border rounded-md flex items-center justify-between gap-3 relative">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="w-7 h-7 rounded-md bg-primary-light text-primary inline-flex items-center justify-center flex-shrink-0">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
                        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
                        <line x1="12" y1="22.08" x2="12" y2="12" />
                      </svg>
                    </span>
                    <div class="flex flex-col gap-0.5 min-w-0">
                      <span class="text-[0.8125rem] font-semibold text-text-primary">{{ selectedLog.providerDetail.provider }}</span>
                      <span v-if="selectedLog.providerDetail.requestFormat || selectedLog.providerDetail.transFormat" class="text-[0.6875rem] text-text-muted font-mono">
                        {{ selectedLog.providerDetail.requestFormat || '—' }} → {{ selectedLog.providerDetail.transFormat || '—' }}
                      </span>
                    </div>
                  </div>
                </div>
              </TooltipTrigger>
              <TooltipPortal>
                <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                  <div v-if="selectedLog.providerDetail.requestFormat" class="flex gap-2 py-0.5 items-center">
                    <span class="text-text-muted flex-shrink-0">请求格式</span>
                    <span>{{ selectedLog.providerDetail.requestFormat }}</span>
                  </div>
                  <div v-if="selectedLog.providerDetail.transFormat" class="flex gap-2 py-0.5 items-center">
                    <span class="text-text-muted flex-shrink-0">转换格式</span>
                    <span>{{ selectedLog.providerDetail.transFormat }}</span>
                  </div>
                  <TooltipArrow class="fill-bg-secondary" />
                </TooltipContent>
              </TooltipPortal>
            </TooltipRoot>

            <!-- Core metrics -->
            <div class="grid grid-cols-4 gap-2 mb-4">
              <div class="flex flex-col items-center gap-1 px-2.5 py-2.5 bg-bg-secondary border border-border rounded-sm text-center">
                <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">费用</span>
                <span class="text-sm text-accent font-mono font-semibold">{{ formatMicrosCost(selectedLog.costMicros) }}</span>
              </div>
              <div class="flex flex-col items-center gap-1 px-2.5 py-2.5 bg-bg-secondary border border-border rounded-sm text-center">
                <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">TTFT</span>
                <span class="text-sm text-text-primary font-mono font-semibold">{{ fmtSeconds(selectedLog.ttft) }}<span class="text-[0.625rem] text-text-muted font-normal ml-0.5">s</span></span>
              </div>
              <div class="flex flex-col items-center gap-1 px-2.5 py-2.5 bg-bg-secondary border border-border rounded-sm text-center">
                <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">总耗时</span>
                <span class="text-sm text-text-primary font-mono font-semibold">{{ fmtSeconds(totalMs(selectedLog.ttft, selectedLog.transferTime)) }}<span class="text-[0.625rem] text-text-muted font-normal ml-0.5">s</span></span>
              </div>
              <TooltipRoot>
                <TooltipTrigger as-child>
                  <div class="flex flex-col items-center gap-1 px-2.5 py-2.5 bg-bg-secondary border border-border rounded-sm text-center cursor-help">
                    <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono">渠道</span>
                    <span class="text-sm text-text-primary font-mono font-semibold">{{ selectedLog.channelName || selectedLog.channelId }}</span>
                  </div>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent v-if="selectedLog.channelPriceRate !== undefined" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                    <span>费率 {{ selectedLog.channelPriceRate }}x</span>
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
            </div>

            <!-- IDs row with hover tooltips -->
            <div class="flex gap-2 mb-4 flex-wrap">
              <TooltipRoot>
                <TooltipTrigger as-child>
                  <div class="flex flex-col gap-0.5 px-2.5 py-1.5 bg-bg-secondary border border-border rounded-sm min-w-0">
                    <span class="text-[0.5625rem] text-text-muted uppercase tracking-wider font-mono">请求ID</span>
                    <code class="text-[0.6875rem] text-text-secondary font-mono break-all">{{ selectedLog.requestId || '-' }}</code>
                  </div>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent v-if="selectedLog.extra?.requestPath" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                    <span class="text-text-muted">请求路径</span>
                    <span>{{ selectedLog.extra.requestPath }}</span>
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
              <TooltipRoot>
                <TooltipTrigger as-child>
                  <div class="flex flex-col gap-0.5 px-2.5 py-1.5 bg-bg-secondary border border-border rounded-sm min-w-0">
                    <span class="text-[0.5625rem] text-text-muted uppercase tracking-wider font-mono">用户ID</span>
                    <code class="text-[0.6875rem] text-text-secondary font-mono break-all">{{ selectedLog.userId }}</code>
                  </div>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent v-if="selectedLog.extra?.ip || selectedLog.extra?.userAgent" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                    <span v-if="selectedLog.extra.ip" class="flex gap-2 py-0.5 items-center">
                      <span class="text-text-muted">IP</span>
                      <span>{{ selectedLog.extra.ip }}</span>
                    </span>
                    <span v-if="selectedLog.extra.userAgent" class="flex gap-2 py-0.5 items-center">
                      <span class="text-text-muted">UA</span>
                      <span>{{ selectedLog.extra.userAgent }}</span>
                    </span>
                    <TooltipArrow class="fill-bg-secondary" />
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
              <div class="flex flex-col gap-0.5 px-2.5 py-1.5 bg-bg-secondary border border-border rounded-sm min-w-0">
                <span class="text-[0.5625rem] text-text-muted uppercase tracking-wider font-mono">令牌ID</span>
                <code class="text-[0.6875rem] text-text-secondary font-mono break-all">{{ selectedLog.tokenId }}</code>
              </div>
              <div class="flex flex-col gap-0.5 px-2.5 py-1.5 bg-bg-secondary border border-border rounded-sm min-w-0">
                <span class="text-[0.5625rem] text-text-muted uppercase tracking-wider font-mono">分组ID</span>
                <code class="text-[0.6875rem] text-text-secondary font-mono break-all">{{ selectedLog.groupId || '-' }}</code>
              </div>
            </div>

            <!-- Token breakdown -->
            <div class="mb-4 px-4 py-3 bg-bg-secondary border border-border rounded-md">
              <div class="text-[0.625rem] font-bold text-text-muted uppercase tracking-wider font-mono mb-2.5">Token 用量明细</div>
              <div class="mb-3">
                <div v-if="tokenBarSegments(selectedLog).total > 0" class="flex h-1.5 rounded overflow-hidden bg-bg-primary border border-border">
                  <div class="h-full transition-[width] duration-300 ease-out bg-primary" :style="{ width: tokenBarSegments(selectedLog).input + '%' }"></div>
                  <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(139,195,74,0.55);" :style="{ width: tokenBarSegments(selectedLog).output + '%' }"></div>
                  <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(255,179,0,0.7);" :style="{ width: tokenBarSegments(selectedLog).cachedRead + '%' }"></div>
                  <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(255,213,79,0.6);" :style="{ width: tokenBarSegments(selectedLog).cachedCreate + '%' }"></div>
                </div>
                <div v-else class="h-1.5 rounded bg-bg-primary border border-border opacity-30"></div>
              </div>
              <div class="grid grid-cols-5 gap-2">
                <div class="flex flex-col items-center gap-1 p-2 bg-bg-primary border border-border rounded-sm">
                  <span class="w-2 h-2 rounded-full bg-primary"></span>
                  <span class="text-[0.625rem] text-text-muted font-mono">输入</span>
                  <span class="text-xs text-text-primary font-mono font-semibold">{{ formatTokens(selectedLog.inputToken) }}</span>
                </div>
                <div class="flex flex-col items-center gap-1 p-2 bg-bg-primary border border-border rounded-sm">
                  <span class="w-2 h-2 rounded-full" style="background: rgba(139,195,74,0.55);"></span>
                  <span class="text-[0.625rem] text-text-muted font-mono">输出</span>
                  <span class="text-xs text-text-primary font-mono font-semibold">{{ formatTokens(selectedLog.outputToken) }}</span>
                </div>
                <div class="flex flex-col items-center gap-1 p-2 bg-bg-primary border border-border rounded-sm">
                  <span class="w-2 h-2 rounded-full" style="background: rgba(255,179,0,0.7);"></span>
                  <span class="text-[0.625rem] text-text-muted font-mono">缓存读</span>
                  <span class="text-xs text-text-primary font-mono font-semibold">{{ formatTokens(selectedLog.cachedReadInputTokens) }}</span>
                </div>
                <div class="flex flex-col items-center gap-1 p-2 bg-bg-primary border border-border rounded-sm">
                  <span class="w-2 h-2 rounded-full" style="background: rgba(255,213,79,0.6);"></span>
                  <span class="text-[0.625rem] text-text-muted font-mono">缓存创建</span>
                  <span class="text-xs text-text-primary font-mono font-semibold">{{ formatTokens(selectedLog.cachedCreationInputTokens) }}</span>
                </div>
                <div class="flex flex-col items-center gap-1 p-2 bg-bg-primary border border-border-hover rounded-sm">
                  <span class="text-[0.625rem] text-text-muted font-mono">总计</span>
                  <span class="text-xs text-accent font-mono font-semibold">{{ formatTokens(selectedLog.inputToken + selectedLog.outputToken + selectedLog.cachedReadInputTokens + selectedLog.cachedCreationInputTokens) }}</span>
                </div>
              </div>
            </div>

            <!-- Retry trace -->
            <div v-if="selectedLog.extra?.retryTrace?.length" class="mb-4 px-4 py-3 bg-bg-secondary border border-border rounded-md">
              <div class="flex items-center gap-2 text-xs font-semibold text-text-secondary mb-2.5">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="17 1 21 5 17 9" />
                  <path d="M3 11V9a4 4 0 0 1 4-4h14" />
                  <polyline points="7 23 3 19 7 15" />
                  <path d="M21 13v2a4 4 0 0 1-4 4H3" />
                </svg>
                重试链路
                <span class="text-[0.625rem] text-text-muted font-mono font-normal">{{ selectedLog.extra.retryTrace.length }} 次</span>
              </div>
              <div class="flex flex-col gap-1.5">
                <div
                  v-for="(trace, i) in selectedLog.extra.retryTrace"
                  :key="i"
                  class="grid grid-cols-[28px_1fr_1fr_50px_1fr] gap-2 items-center px-2 py-1.5 bg-bg-primary border border-border rounded-sm text-[0.6875rem] font-mono"
                  :class="{ 'border-danger/30 bg-danger/5': trace.statusCode && trace.statusCode >= 400 }"
                >
                  <span class="text-text-muted text-[0.625rem]">#{{ i + 1 }}</span>
                  <span class="text-text-secondary truncate" :title="trace.channelName || trace.channelID">{{ trace.channelName || trace.channelID || '-' }}</span>
                  <code class="text-text-tertiary text-[0.625rem] truncate">{{ trace.upstreamModel || '-' }}</code>
                  <span v-if="trace.statusCode" class="text-center px-1 py-0.5 rounded text-[0.625rem] font-semibold" :class="trace.statusCode < 400 ? 'text-success bg-success-light' : 'text-danger bg-danger-light'">{{ trace.statusCode }}</span>
                  <TooltipRoot v-if="trace.statusBody">
                    <TooltipTrigger as-child>
                      <span class="text-text-muted text-[0.625rem] truncate">{{ trace.statusBody }}</span>
                    </TooltipTrigger>
                    <TooltipPortal>
                      <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                        {{ trace.statusBody }}
                        <TooltipArrow class="fill-bg-secondary" />
                      </TooltipContent>
                    </TooltipPortal>
                  </TooltipRoot>
                </div>
              </div>
            </div>

            <!-- JSON panels -->
            <div class="grid grid-cols-2 gap-3">
              <div class="border border-border rounded-sm overflow-hidden">
                <div class="flex items-center gap-2 px-2.5 py-2 bg-bg-secondary border-b border-border">
                  <span class="w-1.5 h-1.5 rounded-full bg-primary shadow-[0_0_6px_var(--color-primary-glow)]"></span>
                  <span class="text-[0.6875rem] font-semibold text-text-secondary font-mono uppercase tracking-wider">extra</span>
                </div>
                <pre class="m-0 px-2.5 py-2 max-h-[200px] overflow-auto text-[0.6875rem] leading-relaxed text-text-tertiary font-mono bg-bg-primary">{{ safeJSONStringify(selectedLog.extra) }}</pre>
              </div>
              <div class="border border-border rounded-sm overflow-hidden">
                <div class="flex items-center gap-2 px-2.5 py-2 bg-bg-secondary border-b border-border">
                  <span class="w-1.5 h-1.5 rounded-full bg-accent shadow-[0_0_6px_var(--color-accent-glow)]"></span>
                  <span class="text-[0.6875rem] font-semibold text-text-secondary font-mono uppercase tracking-wider">providerDetail</span>
                </div>
                <pre class="m-0 px-2.5 py-2 max-h-[200px] overflow-auto text-[0.6875rem] leading-relaxed text-text-tertiary font-mono bg-bg-primary">{{ safeJSONStringify(selectedLog.providerDetail) }}</pre>
              </div>
            </div>
          </div>
        </div>
      </AppDialog>

      <!-- Command Bar (Filters) -->
      <div class="flex items-center gap-3 flex-wrap mb-5 px-4 py-3 bg-bg-card border border-border rounded-lg backdrop-blur-sm">
        <div class="flex items-center gap-1.5 flex-shrink-0">
          <span class="text-primary font-mono text-sm font-bold">$</span>
          <span class="text-text-muted text-[0.6875rem] font-mono uppercase tracking-wider">筛选</span>
        </div>
        <div class="flex items-center gap-2 flex-wrap flex-1">
          <div class="min-w-[160px] flex-1">
            <AppInput v-model="keyword" placeholder="关键词..." size="sm" @keyup.enter="handleSearch" />
          </div>
          <div class="min-w-[110px]">
            <AppSelect v-model="status" :options="statusOptions" size="sm" />
          </div>
          <div class="min-w-[110px]">
            <AppSelect v-model="streamMode" :options="streamOptions" size="sm" />
          </div>
          <div class="min-w-[110px]">
            <AppSelect v-model="orderMode" :options="orderOptions" size="sm" />
          </div>
          <div class="min-w-[180px] flex-initial">
            <AppInput v-model="dateFrom" type="datetime-local" size="sm" />
          </div>
          <div class="min-w-[180px] flex-initial">
            <AppInput v-model="dateTo" type="datetime-local" size="sm" />
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <AppButton variant="primary" size="sm" @click="handleSearch">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
            <span>执行</span>
          </AppButton>
          <AppButton variant="ghost" size="sm" @click="handleReset">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="1 4 1 10 7 10" />
              <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10" />
            </svg>
            <span>重置</span>
          </AppButton>
        </div>
      </div>

      <!-- Error -->
      <Transition name="fade">
        <div v-if="loadError" class="flex items-center gap-2 mb-4 px-3.5 py-2.5 bg-danger-light border border-danger/20 rounded-md text-danger text-xs">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>{{ loadError }}</span>
        </div>
      </Transition>

      <!-- Log Stream -->
      <div class="bg-bg-card border border-border rounded-lg overflow-hidden mb-5">
        <!-- Stream Header -->
        <div class="grid grid-cols-[90px_75px_75px_minmax(220px,1fr)_70px_120px_70px_48px_80px_44px] gap-2 items-center px-4 py-2.5 bg-bg-secondary border-b border-border text-[0.625rem] font-bold text-text-muted uppercase tracking-wider font-mono">
          <div>时间</div>
          <div>令牌</div>
          <div>分组</div>
          <div>渠道 / 模型映射</div>
          <div>状态</div>
          <div>Token</div>
          <div>费用</div>
          <div>流式</div>
          <div>TTFT/Total</div>
          <div></div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="px-4 py-2">
          <div v-for="i in 5" :key="i" class="grid grid-cols-[90px_75px_75px_minmax(220px,1fr)_70px_120px_70px_48px_80px_44px] gap-2 items-center py-3 border-b border-border">
            <div class="h-3 rounded bg-gradient-to-r from-bg-secondary via-bg-tertiary to-bg-secondary bg-[length:200%_100%] animate-[shimmer_1.5s_ease-in-out_infinite]"></div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else-if="logs.length === 0" class="flex flex-col items-center justify-center py-16 px-8 text-text-muted">
          <div class="text-text-muted opacity-30 mb-4">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
              <rect x="2" y="3" width="20" height="14" rx="2" />
              <line x1="8" y1="21" x2="16" y2="21" />
              <line x1="12" y1="17" x2="12" y2="21" />
            </svg>
          </div>
          <p class="m-0 mb-1.5 text-base font-semibold text-text-secondary">暂无数据包</p>
          <p class="m-0 text-[0.8125rem] text-text-muted">当前筛选条件下没有找到请求日志</p>
        </div>

        <!-- Log Packets -->
        <template v-else>
          <div
            v-for="(log, index) in logs"
            :key="log.id"
            v-motion="staggerItem"
            :style="{ transitionDelay: `${index * 40}ms` }"
            class="border-b border-border transition-colors duration-150 hover:bg-primary/[0.02]"
            :class="{ 'bg-primary/[0.03] border-border-hover': expandedId === log.id, 'bg-danger/[0.02] hover:bg-danger/[0.04]': log.status === 'fail' }"
          >
            <!-- Main Row -->
            <div class="grid grid-cols-[90px_75px_75px_minmax(220px,1fr)_70px_120px_70px_48px_80px_44px] gap-2 items-center px-4 py-2.5 cursor-pointer transition-[padding] duration-200" :class="{ 'pb-3': expandedId === log.id }" @click="toggleDetail(log.id)">
              <!-- Time -->
              <div class="flex flex-col gap-0.5">
                <span class="text-[0.6875rem] text-text-muted font-mono">{{ formatDate(log.createdAt).date }}</span>
                <span class="text-xs text-text-secondary font-mono font-medium">{{ formatDate(log.createdAt).time }}</span>
              </div>

              <!-- Token -->
              <div class="min-w-0">
                <TooltipRoot>
                  <TooltipTrigger as-child>
                    <span class="inline-block max-w-full text-[0.625rem] font-mono text-primary bg-primary-light border border-border rounded px-1.5 py-0.5 whitespace-nowrap truncate cursor-help">{{ log.tokenName || log.tokenId }}</span>
                  </TooltipTrigger>
                  <TooltipPortal>
                    <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                      {{ log.tokenName || log.tokenId }}
                      <TooltipArrow class="fill-bg-secondary" />
                    </TooltipContent>
                  </TooltipPortal>
                </TooltipRoot>
              </div>

              <!-- Group -->
              <div class="min-w-0">
                <TooltipRoot>
                  <TooltipTrigger as-child>
                    <span class="inline-block max-w-full text-[0.625rem] font-mono text-text-secondary bg-bg-secondary border border-border rounded px-1.5 py-0.5 whitespace-nowrap truncate cursor-help">{{ log.groupName || log.groupId || '-' }}</span>
                  </TooltipTrigger>
                  <TooltipPortal>
                    <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                      {{ log.groupName || log.groupId || '-' }}
                      <TooltipArrow class="fill-bg-secondary" />
                    </TooltipContent>
                  </TooltipPortal>
                </TooltipRoot>
              </div>

              <!-- Channel / Model / Provider -->
              <div class="flex flex-col gap-1 min-w-0">
                <div class="flex items-center gap-1.5 min-w-0 flex-wrap">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <span class="w-5 h-5 rounded-[5px] flex items-center justify-center text-[0.625rem] font-bold text-bg-primary flex-shrink-0" style="background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));">{{ (log.channelName || log.channelId || '?').charAt(0) }}</span>
                    <TooltipRoot>
                      <TooltipTrigger as-child>
                        <span class="text-xs text-text-primary font-medium truncate">{{ log.channelName || log.channelId }}</span>
                      </TooltipTrigger>
                      <TooltipPortal>
                        <TooltipContent v-if="log.extra?.ip" class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                          <span class="text-text-muted">IP</span>
                          <span>{{ log.extra.ip }}</span>
                          <TooltipArrow class="fill-bg-secondary" />
                        </TooltipContent>
                      </TooltipPortal>
                    </TooltipRoot>
                  </div>
                  <TooltipRoot v-if="log.providerDetail?.provider">
                    <TooltipTrigger as-child>
                      <span class="text-[0.625rem] font-mono text-primary bg-primary-light border border-border rounded px-1 leading-[18px] whitespace-nowrap flex-shrink-0 cursor-help">{{ log.providerDetail.provider }}</span>
                    </TooltipTrigger>
                    <TooltipPortal>
                      <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] whitespace-nowrap max-w-[320px]">
                        <span class="flex gap-2 py-0.5 items-center"><span class="text-text-muted">请求格式</span><span>{{ log.providerDetail.requestFormat || '—' }}</span></span>
                        <span class="flex gap-2 py-0.5 items-center"><span class="text-text-muted">转换格式</span><span>{{ log.providerDetail.transFormat || '—' }}</span></span>
                        <TooltipArrow class="fill-bg-secondary" />
                      </TooltipContent>
                    </TooltipPortal>
                  </TooltipRoot>
                  <TooltipRoot v-if="log.extra?.retryTrace?.length">
                    <TooltipTrigger as-child>
                      <span class="text-[0.625rem] font-mono text-accent bg-accent/10 border border-accent/20 rounded px-1 leading-[18px] whitespace-nowrap flex-shrink-0 cursor-help">重试 {{ log.extra.retryTrace.length }}</span>
                    </TooltipTrigger>
                    <TooltipPortal>
                      <TooltipContent class="bg-bg-secondary border border-border-hover rounded-md px-3 py-2.5 shadow-[0_8px_24px_rgba(0,0,0,0.3)] text-[0.6875rem] font-mono text-text-primary z-[100] max-w-[360px] flex flex-col">
                        <div
                          v-for="(t, i) in log.extra.retryTrace"
                          :key="i"
                          class="grid grid-cols-[22px_1fr_1fr_auto] gap-2 items-center py-1 border-b border-border last:border-b-0"
                        >
                          <span class="text-text-muted text-[0.625rem]">#{{ i + 1 }}</span>
                          <span class="truncate" :title="t.channelName || t.channelID">{{ truncate(t.channelName || t.channelID, 6) }}</span>
                          <span class="text-text-muted truncate">{{ truncate(t.upstreamModel, 8) }}</span>
                          <span
                            v-if="t.statusCode"
                            class="text-[0.625rem] font-semibold px-1 py-px rounded leading-4 text-center min-w-[28px]"
                            :class="t.statusCode < 400 ? 'text-success bg-success-light' : 'text-danger bg-danger-light'"
                          >{{ t.statusCode }}</span>
                        </div>
                        <TooltipArrow class="fill-bg-secondary" />
                      </TooltipContent>
                    </TooltipPortal>
                  </TooltipRoot>
                </div>
                <span class="flex items-center gap-1.5 min-w-0">
                  <code class="text-[0.6875rem] text-text-muted font-mono truncate bg-bg-secondary px-1.5 py-px rounded border border-border">{{ log.model || '-' }}</code>
                  <span v-if="log.upstreamModel && log.upstreamModel !== log.model" class="text-[0.6875rem] text-primary font-mono opacity-70 flex-shrink-0">→</span>
                  <code v-if="log.upstreamModel && log.upstreamModel !== log.model" class="text-[0.6875rem] text-text-secondary font-mono truncate bg-bg-secondary px-1.5 py-px rounded border border-border">{{ log.upstreamModel }}</code>
                  <span v-else-if="!log.upstreamModel" class="text-[0.6875rem] text-text-muted opacity-50 bg-transparent border-transparent font-mono">-</span>
                </span>
              </div>

              <!-- Status -->
              <div>
                <span class="inline-flex items-center gap-1.5 text-[0.6875rem] font-semibold" :class="log.status === 'success' ? 'text-success' : 'text-danger'">
                  <span class="w-1.5 h-1.5 rounded-full flex-shrink-0" :class="log.status === 'success' ? 'bg-success shadow-[0_0_8px_var(--color-success),0_0_16px_rgba(129,199,132,0.3)] animate-[pulse_2.5s_ease-in-out_infinite]' : 'bg-danger shadow-[0_0_8px_var(--color-danger),0_0_16px_rgba(229,115,115,0.3)]'"></span>
                  <span class="whitespace-nowrap">{{ formatStatus(log.status) }}</span>
                </span>
              </div>

              <!-- Tokens -->
              <div>
                <div class="flex items-center gap-2">
                  <div v-if="tokenBarSegments(log).total > 0" class="flex w-20 h-[5px] rounded overflow-hidden bg-bg-secondary border border-border">
                    <div class="h-full transition-[width] duration-300 ease-out bg-primary" :style="{ width: tokenBarSegments(log).input + '%' }"></div>
                    <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(139,195,74,0.55);" :style="{ width: tokenBarSegments(log).output + '%' }"></div>
                    <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(255,179,0,0.7);" :style="{ width: tokenBarSegments(log).cachedRead + '%' }"></div>
                    <div class="h-full transition-[width] duration-300 ease-out" style="background: rgba(255,213,79,0.6);" :style="{ width: tokenBarSegments(log).cachedCreate + '%' }"></div>
                  </div>
                  <div v-else class="w-20 h-[5px] rounded bg-bg-secondary border border-border opacity-30"></div>
                  <span class="text-[0.6875rem] text-text-muted font-mono whitespace-nowrap">{{ formatTokens(tokenBarSegments(log).total) }}</span>
                </div>
              </div>

              <!-- Cost -->
              <div>
                <span class="text-xs text-accent font-mono font-semibold">{{ formatMicrosCost(log.costMicros) }}</span>
              </div>

              <!-- Stream -->
              <div>
                <span v-if="log.isStream" class="inline-flex items-center justify-center w-[22px] h-[22px] rounded-[5px] text-primary bg-primary-light border border-border" title="流式">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M4 14h6v6H4z" />
                    <path d="M4 4h6v6H4z" opacity="0.5" />
                    <path d="M14 8h6v6h-6z" opacity="0.5" />
                    <path d="M14 18h6v-6h-6z" opacity="0.3" />
                  </svg>
                </span>
                <span v-else class="inline-flex items-center justify-center w-[22px] h-[22px] rounded-[5px] text-text-muted bg-bg-secondary border border-border" title="非流式">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="18" height="18" rx="2" />
                  </svg>
                </span>
              </div>

              <!-- TTFT -->
              <div>
                <span class="text-xs font-mono">
                  <span class="text-primary font-semibold">{{ fmtSeconds(log.ttft) }}</span>
                  <span class="text-[0.625rem] text-text-muted mx-0.5">/</span>
                  <span class="text-text-secondary">{{ fmtSeconds(totalMs(log.ttft, log.transferTime)) }}</span>
                  <span class="text-[0.625rem] text-text-muted ml-0.5">s</span>
                </span>
              </div>
            </div>

            <!-- Expanded Detail -->
            <Transition name="expand">
              <div v-if="expandedId === log.id" class="px-4 pb-4 overflow-hidden">
                <div class="grid grid-cols-3 gap-3 p-3 bg-bg-secondary border border-border rounded-md mb-3">
                  <div class="flex flex-col gap-2">
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">请求ID</span>
                      <code class="text-text-secondary font-mono break-all text-right">{{ log.requestId || '-' }}</code>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">用户ID</span>
                      <code class="text-text-secondary font-mono break-all text-right">{{ log.userId }}</code>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">令牌ID</span>
                      <code class="text-text-secondary font-mono break-all text-right">{{ log.tokenId }}</code>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">渠道费率</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ log.channelPriceRate }}x</span>
                    </div>
                  </div>
                  <div class="flex flex-col gap-2">
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">输入 Token</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ formatTokens(log.inputToken) }}</span>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">输出 Token</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ formatTokens(log.outputToken) }}</span>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">缓存读</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ formatTokens(log.cachedReadInputTokens) }}</span>
                    </div>
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">缓存创建</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ formatTokens(log.cachedCreationInputTokens) }}</span>
                    </div>
                  </div>
                  <div class="flex flex-col gap-2">
                    <div class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">传输时间</span>
                      <span class="text-text-secondary font-mono break-all text-right">{{ fmtSeconds(log.transferTime) }}s</span>
                    </div>
                    <div v-if="log.errorCode" class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1 border-b border-dashed border-border">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">错误码</span>
                      <code class="text-danger font-mono break-all text-right">{{ log.errorCode }}</code>
                    </div>
                    <div v-if="log.errorMsg" class="flex justify-between items-baseline gap-2 text-[0.6875rem] py-1">
                      <span class="text-text-muted font-mono uppercase tracking-wider text-[0.625rem]">错误信息</span>
                      <span class="text-danger font-mono break-all text-right">{{ log.errorMsg }}</span>
                    </div>
                  </div>
                </div>
                <div v-if="log.extra?.requestPath" class="mb-3 px-3 py-2 bg-bg-secondary border border-border rounded-md flex items-center gap-2">
                  <span class="text-[0.625rem] text-text-muted uppercase tracking-wider font-mono flex-shrink-0">请求路径</span>
                  <code class="text-[0.6875rem] text-text-secondary font-mono break-all">{{ log.extra.requestPath }}</code>
                </div>
                <div class="flex justify-end">
                  <AppButton variant="secondary" size="sm" @click="openModal(log)">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="2" y="3" width="20" height="14" rx="2" />
                      <line x1="8" y1="21" x2="16" y2="21" />
                      <line x1="12" y1="17" x2="12" y2="21" />
                    </svg>
                    查看完整详情
                  </AppButton>
                </div>
              </div>
            </Transition>
          </div>
        </template>
      </div>

      <!-- Pagination -->
      <div v-if="!loading && logs.length > 0" class="flex items-center justify-between px-4 py-3 bg-bg-card border border-border rounded-lg">
        <span class="text-xs text-text-muted font-mono">
          第 <strong class="text-text-secondary">{{ page }}</strong> / {{ totalPages }} 页 · 共 <strong class="text-text-secondary">{{ total }}</strong> 条
        </span>
        <div class="flex items-center gap-1.5">
          <AppButton
            variant="secondary"
            size="sm"
            class="w-8 h-8 p-0"
            :disabled="page <= 1 || loading"
            @click="handlePageChange(page - 1)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="15 18 9 12 15 6" />
            </svg>
          </AppButton>
          <div class="flex items-center gap-1">
            <AppButton
              v-for="p in Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                const start = Math.max(1, Math.min(page - 2, totalPages - 4))
                return start + i
              })"
              :key="p"
              variant="ghost"
              size="sm"
              class="min-w-[32px] h-8 p-0 font-mono text-xs"
              :class="{ 'bg-primary-light border-border-hover text-primary font-semibold': p === page }"
              @click="handlePageChange(p)"
            >
              {{ p }}
            </AppButton>
          </div>
          <AppButton
            variant="secondary"
            size="sm"
            class="w-8 h-8 p-0"
            :disabled="page >= totalPages || loading"
            @click="handlePageChange(page + 1)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </AppButton>
        </div>
      </div>
    </div>
  </TooltipProvider>
</template>

<style>
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.7); }
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.25s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.expand-enter-to,
.expand-leave-from {
  opacity: 1;
  max-height: 300px;
}
</style>
