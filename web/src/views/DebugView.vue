<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { getRegistry, ENDPOINT_STATUS_LABELS, type RegistryData, type RegistryEndpoint } from '@/api/debug'
import { fadeUp } from '@/utils/motion'
import { useDebugStore } from '@/stores/debug'

const debugStore = useDebugStore()

const data = ref<RegistryData | null>(null)
const loading = ref(false)
const error = ref('')
const lastRefresh = ref<Date | null>(null)

// Model search
const modelSearch = ref('')

let timer: ReturnType<typeof setInterval> | null = null

const channels = computed(() => {
  if (!data.value) return []
  return Object.entries(data.value.channels).map(([id, ch]) => ({ _id: id, ...ch }))
})

const channelNameMap = computed(() => {
  const map: Record<string, string> = {}
  for (const ch of channels.value) {
    map[ch._id] = ch.name
  }
  return map
})

const endpoints = computed(() => {
  if (!data.value) return []
  const result: Array<{ model: string; endpoints: Array<RegistryEndpoint & { _id: string }> }> = []
  for (const [model, channelMap] of Object.entries(data.value.endpoints)) {
    const eps = Object.entries(channelMap).map(([chId, ep]) => ({ _id: chId, ...ep }))
    eps.sort((a, b) => a.status - b.status || b.emaSuccessRate - a.emaSuccessRate)
    result.push({ model, endpoints: eps })
  }
  result.sort((a, b) => a.model.localeCompare(b.model))
  if (!modelSearch.value) return result
  const q = modelSearch.value.toLowerCase()
  return result.filter(g => g.model.toLowerCase().includes(q) || g.endpoints.some(ep => ep.upstreamModel.toLowerCase().includes(q)))
})

const summary = computed(() => {
  const chs = channels.value
  const totalConns = chs.reduce((s, c) => s + (c.activeConns || 0), 0)
  const allEps = endpoints.value.flatMap(g => g.endpoints)
  const healthy = allEps.filter(e => e.status === 0).length
  const degraded = allEps.filter(e => e.status === 1).length
  const circuit = allEps.filter(e => e.status === 2).length
  return { channels: chs.length, totalConns, endpoints: allEps.length, healthy, degraded, circuit }
})

async function fetchData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await getRegistry()
    lastRefresh.value = new Date()
  } catch (e: unknown) {
    error.value = (e as Error).message || '获取数据失败'
  } finally {
    loading.value = false
  }
}

function startAutoRefresh() {
  stopAutoRefresh()
  if (debugStore.autoRefresh) {
    timer = setInterval(fetchData, debugStore.refreshInterval * 1000)
  }
}

function stopAutoRefresh() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

function toggleAutoRefresh() {
  debugStore.setAutoRefresh(!debugStore.autoRefresh)
  if (debugStore.autoRefresh) startAutoRefresh()
  else stopAutoRefresh()
}

function changeInterval(val: number) {
  debugStore.setRefreshInterval(val)
  if (debugStore.autoRefresh) startAutoRefresh()
}

function formatTimestamp(ts: number): string {
  if (!ts) return '—'
  const d = new Date(ts * 1000)
  const now = Date.now()
  const diff = Math.floor((now - d.getTime()) / 1000)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

function statusColor(status: number): string {
  if (status === 0) return 'text-success'
  if (status === 1) return 'text-warning'
  return 'text-danger'
}

function statusBg(status: number): string {
  if (status === 0) return 'bg-success/10 border-success/20'
  if (status === 1) return 'bg-warning/10 border-warning/20'
  return 'bg-danger/10 border-danger/20'
}

function rateColor(val: number): string {
  if (val >= 0.95) return 'text-success'
  if (val >= 0.8) return 'text-warning'
  return 'text-danger'
}

onMounted(() => {
  fetchData()
  startAutoRefresh()
})

onBeforeUnmount(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div v-motion="fadeUp" class="max-w-[1600px] min-w-[1100px]">
    <!-- Header -->
    <header class="relative mb-6">
      <!-- Scanline overlay -->
      <div class="absolute inset-0 pointer-events-none opacity-[0.03] bg-[repeating-linear-gradient(0deg,transparent,transparent_2px,rgba(255,255,255,0.05)_2px,rgba(255,255,255,0.05)_4px)] rounded-lg"></div>
      <div class="relative bg-bg-card border border-border rounded-lg px-5 py-4">
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-4">
            <div class="flex items-center gap-2.5">
              <span
                class="w-2 h-2 rounded-full transition-all duration-300"
                :class="debugStore.autoRefresh ? 'bg-success shadow-[0_0_6px_rgba(139,195,74,0.6)] animate-pulse' : 'bg-text-muted'"
              ></span>
              <span class="text-lg font-semibold tracking-tight text-text-primary font-mono">MONITOR</span>
            </div>
            <div class="h-4 w-px bg-border"></div>
            <div class="flex items-center gap-1.5 text-xs text-text-muted font-mono">
              <span v-if="lastRefresh">{{ lastRefresh.toLocaleTimeString() }}</span>
              <span v-if="loading" class="text-primary animate-pulse">SYNC</span>
              <span v-else-if="debugStore.autoRefresh" class="text-success/70">LIVE</span>
              <span v-else class="text-warning/70">PAUSED</span>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex items-center bg-bg-primary border border-border rounded-md overflow-hidden">
              <button
                v-for="sec in [15, 30, 60]"
                :key="sec"
                class="px-3 py-1.5 text-xs font-mono transition-all duration-150 border-r border-border last:border-r-0"
                :class="debugStore.refreshInterval === sec ? 'bg-primary/15 text-primary font-medium' : 'text-text-muted hover:text-text-secondary hover:bg-bg-secondary'"
                @click="changeInterval(sec)"
              >
                {{ sec }}s
              </button>
            </div>
            <button
              class="px-3 py-1.5 text-xs font-mono rounded-md border transition-all duration-150"
              :class="debugStore.autoRefresh
                ? 'border-warning/30 text-warning bg-warning/10 hover:bg-warning/20'
                : 'border-primary/30 text-primary bg-primary/10 hover:bg-primary/20'"
              @click="toggleAutoRefresh"
            >
              {{ debugStore.autoRefresh ? 'PAUSE' : 'RESUME' }}
            </button>
            <button
              class="px-3 py-1.5 text-xs font-mono rounded-md border border-border text-text-muted hover:text-text-secondary hover:bg-bg-secondary transition-all duration-150"
              :class="loading && 'text-primary border-primary/30'"
              @click="fetchData"
            >
              {{ loading ? '...' : 'REFRESH' }}
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Error -->
    <div v-if="error" class="mb-4 px-4 py-3 bg-danger-light/60 text-danger rounded-lg text-xs font-mono border border-danger/20">
      ERROR: {{ error }}
    </div>

    <!-- Summary Cards -->
    <div class="grid grid-cols-6 gap-2.5 mb-5">
      <div class="bg-bg-card border border-border rounded-lg px-3.5 py-3 group hover:border-border-hover transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">CHANNELS</div>
        <div class="text-xl font-mono font-bold text-text-primary">{{ summary.channels }}</div>
      </div>
      <div class="bg-bg-card border border-border rounded-lg px-3.5 py-3 group hover:border-border-hover transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">CONNS</div>
        <div class="text-xl font-mono font-bold" :class="summary.totalConns > 0 ? 'text-primary' : 'text-text-muted'">{{ summary.totalConns }}</div>
      </div>
      <div class="bg-bg-card border border-border rounded-lg px-3.5 py-3 group hover:border-border-hover transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">ENDPOINTS</div>
        <div class="text-xl font-mono font-bold text-text-primary">{{ summary.endpoints }}</div>
      </div>
      <div class="bg-bg-card border border-success/15 rounded-lg px-3.5 py-3 hover:border-success/30 transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">HEALTHY</div>
        <div class="text-xl font-mono font-bold text-success">{{ summary.healthy }}</div>
      </div>
      <div class="bg-bg-card border border-warning/15 rounded-lg px-3.5 py-3 hover:border-warning/30 transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">DEGRADED</div>
        <div class="text-xl font-mono font-bold text-warning">{{ summary.degraded }}</div>
      </div>
      <div class="bg-bg-card border border-danger/15 rounded-lg px-3.5 py-3 hover:border-danger/30 transition-colors duration-200">
        <div class="text-[10px] font-mono text-text-muted uppercase tracking-widest mb-1.5">CIRCUIT</div>
        <div class="text-xl font-mono font-bold text-danger">{{ summary.circuit }}</div>
      </div>
    </div>

    <!-- Channels Table -->
    <div class="bg-bg-card rounded-lg border border-border mb-5 overflow-hidden">
      <button
        class="w-full px-4 py-2.5 flex items-center gap-2 bg-bg-secondary/50 cursor-pointer border-none text-left transition-colors duration-150 hover:bg-bg-secondary/80"
        @click="debugStore.setChannelsCollapsed(!debugStore.channelsCollapsed)"
      >
        <svg
          class="w-3 h-3 text-text-muted/50 transition-transform duration-200"
          :class="debugStore.channelsCollapsed && '-rotate-90'"
          viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
        >
          <polyline points="6 9 12 15 18 9"/>
        </svg>
        <span class="text-[10px] font-mono text-text-muted uppercase tracking-widest">CHANNELS</span>
        <div class="flex-1 h-px bg-border/40"></div>
        <span class="text-[10px] font-mono text-text-muted">{{ channels.length }}</span>
      </button>
      <div v-show="!debugStore.channelsCollapsed" class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">ID</th>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">NAME</th>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">TYPE</th>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">STATUS</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">CONNS</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">RATE</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">WEIGHT</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/70 bg-bg-secondary/30 border-b border-border">LAST</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="channels.length === 0">
              <td colspan="8" class="px-4 py-8 text-center text-text-muted text-xs font-mono">
                {{ loading ? 'LOADING...' : 'NO DATA' }}
              </td>
            </tr>
            <tr
              v-for="ch in channels"
              :key="ch._id"
              class="border-b border-border/50 transition-colors duration-100 hover:bg-primary/[0.03]"
            >
              <td class="px-3 py-2 text-[11px] font-mono text-text-muted/60">{{ ch._id }}</td>
              <td class="px-3 py-2 text-[13px] text-text-primary font-medium">{{ ch.name }}</td>
              <td class="px-3 py-2">
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-bg-tertiary/60 text-text-secondary/80">{{ ch.type }}</span>
              </td>
              <td class="px-3 py-2">
                <span
                  class="inline-flex items-center gap-1.5 text-[10px] font-mono"
                  :class="ch.status === 'enabled' ? 'text-success' : 'text-warning'"
                >
                  <span
                    class="w-1.5 h-1.5 rounded-full"
                    :class="ch.status === 'enabled' ? 'bg-success shadow-[0_0_4px_rgba(139,195,74,0.5)]' : 'bg-warning shadow-[0_0_4px_rgba(255,213,79,0.5)]'"
                  ></span>
                  {{ ch.status === 'enabled' ? 'ON' : 'OFF' }}
                </span>
              </td>
              <td class="px-3 py-2 text-right">
                <span
                  class="text-[13px] font-mono font-semibold tabular-nums"
                  :class="ch.activeConns > 0 ? 'text-primary' : 'text-text-muted/40'"
                >{{ ch.activeConns }}</span>
              </td>
              <td class="px-3 py-2 text-right">
                <span
                  class="text-[13px] font-mono font-semibold tabular-nums"
                  :class="rateColor(ch.successRate || 0)"
                >{{ ((ch.successRate || 0) * 100).toFixed(1) }}%</span>
              </td>
              <td class="px-3 py-2 text-right text-[12px] font-mono text-text-secondary/70 tabular-nums">{{ ch.weight }}</td>
              <td class="px-3 py-2 text-right text-[11px] font-mono text-text-muted/50 tabular-nums">{{ formatTimestamp(ch.lastUsedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Endpoints Header -->
    <div class="flex items-center gap-3 mb-4">
      <div class="flex items-center gap-2">
        <span class="text-[10px] font-mono text-text-muted uppercase tracking-widest">ENDPOINTS</span>
        <div class="h-px w-8 bg-border/40"></div>
      </div>
      <div class="relative flex-1 max-w-xs">
        <svg class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-text-muted/50" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="M21 21l-4.35-4.35"/>
        </svg>
        <input
          v-model="modelSearch"
          type="text"
          placeholder="filter models..."
          class="w-full pl-8 pr-3 py-1.5 bg-bg-primary text-text-primary placeholder:text-text-muted/40 border border-border rounded-md text-xs font-mono transition-all duration-200 focus:outline-none focus:border-primary/40 focus:shadow-[0_0_8px_rgba(139,195,74,0.08)]"
        />
      </div>
      <span class="text-[10px] font-mono text-text-muted/50 ml-auto">{{ endpoints.length }} models</span>
    </div>

    <!-- Endpoints by Model -->
    <div v-for="group in endpoints" :key="group.model" class="bg-bg-card rounded-lg border border-border mb-3 overflow-hidden">
      <div class="px-4 py-2.5 border-b border-border flex items-center gap-2.5 bg-bg-secondary/30">
        <span class="w-1 h-3.5 rounded-full bg-accent/50"></span>
        <span class="text-[13px] font-mono font-medium text-text-primary">{{ group.model }}</span>
        <div class="flex-1"></div>
        <span class="text-[10px] font-mono text-text-muted/50">{{ group.endpoints.length }} eps</span>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">CHANNEL</th>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">TYPE</th>
              <th class="px-3 py-2 text-left text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">UPSTREAM</th>
              <th class="px-3 py-2 text-center text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">STATUS</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">TTFT</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">TPS</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">RATE</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">FAIL</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">WT</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">COST</th>
              <th class="px-3 py-2 text-right text-[10px] font-mono uppercase tracking-wider text-text-muted/50 bg-bg-secondary/20 border-b border-border/50">LAST</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="ep in group.endpoints"
              :key="ep._id"
              class="border-b border-border/30 transition-colors duration-100 hover:bg-primary/[0.03]"
            >
              <td class="px-3 py-2">
                <span class="text-[12px] text-text-secondary/80">{{ channelNameMap[ep.channelId] || ep.channelId }}</span>
                <span class="text-[10px] font-mono text-text-muted/40 ml-1.5">#{{ ep.channelId }}</span>
              </td>
              <td class="px-3 py-2">
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-bg-tertiary/50 text-text-secondary/70">{{ ep.channelType }}</span>
              </td>
              <td class="px-3 py-2 text-[11px] font-mono text-text-secondary/80">{{ ep.upstreamModel }}</td>
              <td class="px-3 py-2 text-center">
                <span
                  class="inline-flex items-center gap-1 text-[10px] font-mono px-1.5 py-0.5 rounded-sm border"
                  :class="statusBg(ep.status)"
                >
                  <span
                    class="w-1.5 h-1.5 rounded-full"
                    :class="[
                      statusColor(ep.status).replace('text-', 'bg-'),
                      ep.status === 0 ? 'shadow-[0_0_4px_rgba(139,195,74,0.5)]' : ep.status === 1 ? 'shadow-[0_0_4px_rgba(255,213,79,0.5)]' : 'shadow-[0_0_4px_rgba(229,115,115,0.5)]'
                    ]"
                  ></span>
                  <span :class="statusColor(ep.status)">{{ ENDPOINT_STATUS_LABELS[ep.status]?.label || '?' }}</span>
                </span>
              </td>
              <td class="px-3 py-2 text-right text-[12px] font-mono text-text-secondary/70 tabular-nums">
                {{ ep.emaTTFT.toFixed(0) }}<span class="text-text-muted/40 text-[10px]">ms</span>
              </td>
              <td class="px-3 py-2 text-right text-[12px] font-mono text-text-secondary/70 tabular-nums">
                {{ ep.emaTPS.toFixed(1) }}
              </td>
              <td class="px-3 py-2 text-right">
                <span class="text-[12px] font-mono font-semibold tabular-nums" :class="rateColor(ep.emaSuccessRate)">{{ (ep.emaSuccessRate * 100).toFixed(1) }}%</span>
              </td>
              <td class="px-3 py-2 text-right">
                <span
                  class="text-[12px] font-mono font-semibold tabular-nums"
                  :class="ep.consecutiveFails === 0 ? 'text-text-muted/30' : ep.consecutiveFails < 3 ? 'text-warning' : 'text-danger'"
                >{{ ep.consecutiveFails }}</span>
              </td>
              <td class="px-3 py-2 text-right text-[11px] font-mono text-text-secondary/60 tabular-nums">{{ ep.baseWeight }}</td>
              <td class="px-3 py-2 text-right text-[11px] font-mono text-text-secondary/60 tabular-nums">{{ ep.costRate }}x</td>
              <td class="px-3 py-2 text-right text-[11px] font-mono text-text-muted/40 tabular-nums">{{ formatTimestamp(ep.lastUsedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!loading && endpoints.length === 0 && !error" class="text-center py-20">
      <div class="text-text-muted/20 text-4xl font-mono mb-3">---</div>
      <span class="text-xs font-mono text-text-muted/40">NO ENDPOINTS</span>
    </div>
  </div>
</template>
