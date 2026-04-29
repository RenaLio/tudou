<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, watch, shallowRef } from 'vue'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import { useAuthStore } from '@/stores/auth'
import AppButton from '@/components/ui/AppButton.vue'
import AppBadge from '@/components/ui/AppBadge.vue'
import {
  listUserUsageDailyStats,
  listUserUsageHourlyStats,
  getUserStats,
  listChannelStats,
  formatTokens,
  formatCost,
  formatNumber,
  calcSuccessRate,
  type UserUsageDailyStatsResponse,
  type UserUsageHourlyStatsResponse,
  type UserStatsResponse,
  type ChannelStatsResponse,
} from '@/api/stats'
import {
  toRFC3339,
  startOfDay,
  endOfDay,
  addDays,
  toDateKey,
  now,
} from '@/utils/date'

const authStore = useAuthStore()

// ── Time range ──
type TimeRange = 'all' | 'today' | '7d' | '30d'
const timeRange = ref<TimeRange>('today')
const rangeOptions: { value: TimeRange; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'today', label: '当天' },
  { value: '7d', label: '7天' },
  { value: '30d', label: '30天' },
]

const dateRange = computed(() => {
  const today = now()
  switch (timeRange.value) {
    case 'today':
      return { start: startOfDay(today), end: endOfDay(today) }
    case '7d':
      return { start: startOfDay(addDays(today, -6)), end: endOfDay(today) }
    case '30d':
      return { start: startOfDay(addDays(today, -29)), end: endOfDay(today) }
    case 'all':
    default:
      return null
  }
})

// ── Loading states ──
const loading = ref(true)
const loadError = ref('')

// ── Data ──
const userStats = ref<UserStatsResponse | null>(null)
const dailyStats = ref<UserUsageDailyStatsResponse[]>([])
const hourlyStats = ref<UserUsageHourlyStatsResponse[]>([])
const heatmapStats = ref<UserUsageHourlyStatsResponse[]>([])
const channelStats = ref<ChannelStatsResponse[]>([])

function isDateKeyInRange(dateKey: string): boolean {
  const dr = dateRange.value
  if (!dr) return true
  const startKey = toDateKey(dr.start)
  const endKey = toDateKey(dr.end)
  return dateKey >= startKey && dateKey <= endKey
}

// ── Computed: overview ──
const overviewStats = computed(() => {
  if (timeRange.value === 'all' && userStats.value) {
    return {
      requestSuccess: userStats.value.requestSuccess,
      requestFailed: userStats.value.requestFailed,
      inputToken: userStats.value.inputToken,
      outputToken: userStats.value.outputToken,
      cachedReadInputTokens: userStats.value.cachedReadInputTokens,
      cachedCreationInputTokens: userStats.value.cachedCreationInputTokens,
      totalCostMicros: userStats.value.totalCostMicros,
      totalCost: userStats.value.totalCost,
    }
  }
  // Sum from daily stats for today/7d/30d
  const items = dailyStats.value.filter(item => isDateKeyInRange(item.date.slice(0, 10)))
  return {
    requestSuccess: items.reduce((s, d) => s + d.requestSuccess, 0),
    requestFailed: items.reduce((s, d) => s + d.requestFailed, 0),
    inputToken: items.reduce((s, d) => s + d.inputToken, 0),
    outputToken: items.reduce((s, d) => s + d.outputToken, 0),
    cachedReadInputTokens: items.reduce((s, d) => s + d.cachedReadInputTokens, 0),
    cachedCreationInputTokens: items.reduce((s, d) => s + d.cachedCreationInputTokens, 0),
    totalCostMicros: items.reduce((s, d) => s + d.totalCostMicros, 0),
    totalCost: items.reduce((s, d) => s + d.totalCost, 0),
  }
})

const totalRequests = computed(() => {
  return overviewStats.value.requestSuccess + overviewStats.value.requestFailed
})

const totalTokens = computed(() => {
  const o = overviewStats.value
  return o.inputToken + o.outputToken + o.cachedReadInputTokens + o.cachedCreationInputTokens
})

const successRate = computed(() => {
  return calcSuccessRate(overviewStats.value.requestSuccess, overviewStats.value.requestFailed)
})

// ── Computed: channel rankings ──
type ChannelRankSortKey = 'requests' | 'tokens' | 'cost'

const channelRankSort = ref<ChannelRankSortKey>('cost')

const channelRankSortOptions: Array<{ key: ChannelRankSortKey; label: string }> = [
  { key: 'requests', label: '请求量' },
  { key: 'tokens', label: 'Token总量' },
  { key: 'cost', label: '花费' },
]

function inferChannelType(channelName: string): 'openai' | 'claude' | 'azure' | 'custom' {
  const lower = channelName.toLowerCase()
  if (lower.includes('azure')) return 'azure'
  if (lower.includes('claude') || lower.includes('anthropic')) return 'claude'
  if (lower.includes('openai') || lower.includes('gpt') || lower.includes('o1') || lower.includes('o3')) return 'openai'
  return 'custom'
}

const rankedChannels = computed(() => {
  const items = channelStats.value.map(s => {
    const name = s.channelName?.trim() || s.channelID
    const requestCount = s.requestSuccess + s.requestFailed
    const tokenCount = s.inputToken + s.outputToken + s.cachedReadInputTokens + s.cachedCreationInputTokens
    return {
      ...s,
      name,
      type: inferChannelType(name),
      requestCount,
      tokenCount,
    }
  })

  return items.sort((a, b) => {
    switch (channelRankSort.value) {
      case 'requests':
        return b.requestCount - a.requestCount
      case 'tokens':
        return b.tokenCount - a.tokenCount
      case 'cost':
      default:
        return b.totalCost - a.totalCost
    }
  })
})

// ── Computed: unified chart data ──
interface ChartPoint {
  label: string
  requests: number
  requestSuccess: number
  requestFailed: number
  inputToken: number
  outputToken: number
  cachedRead: number
  cachedCreate: number
  totalTokens: number
  cost: number
}

interface UsageAccumulator {
  requestSuccess: number
  requestFailed: number
  inputToken: number
  outputToken: number
  cachedReadInputTokens: number
  cachedCreationInputTokens: number
  totalCost: number
}

function createUsageAccumulator(): UsageAccumulator {
  return {
    requestSuccess: 0,
    requestFailed: 0,
    inputToken: 0,
    outputToken: 0,
    cachedReadInputTokens: 0,
    cachedCreationInputTokens: 0,
    totalCost: 0,
  }
}

function mergeUsageAccumulator(
  target: UsageAccumulator,
  source: Pick<
    UserUsageDailyStatsResponse | UserUsageHourlyStatsResponse,
    | 'requestSuccess'
    | 'requestFailed'
    | 'inputToken'
    | 'outputToken'
    | 'cachedReadInputTokens'
    | 'cachedCreationInputTokens'
    | 'totalCost'
  >,
) {
  target.requestSuccess += source.requestSuccess
  target.requestFailed += source.requestFailed
  target.inputToken += source.inputToken
  target.outputToken += source.outputToken
  target.cachedReadInputTokens += source.cachedReadInputTokens
  target.cachedCreationInputTokens += source.cachedCreationInputTokens
  target.totalCost += source.totalCost
}

type UnifiedSeriesKey =
  | 'inputToken'
  | 'outputToken'
  | 'cachedRead'
  | 'cachedCreate'
  | 'requests'
  | 'cost'

const unifiedChartData = computed<ChartPoint[]>(() => {
  if (timeRange.value === 'today') {
    const todayKey = toDateKey(now())
    const hourlyMap = new Map<number, UsageAccumulator>()
    for (const item of hourlyStats.value) {
      const dateKey = item.date.slice(0, 10)
      if (dateKey !== todayKey) continue
      const existing = hourlyMap.get(item.hour) ?? createUsageAccumulator()
      mergeUsageAccumulator(existing, item)
      hourlyMap.set(item.hour, existing)
    }

    const result: ChartPoint[] = []
    for (let h = 0; h < 24; h++) {
      const found = hourlyMap.get(h)
      const reqS = found?.requestSuccess ?? 0
      const reqF = found?.requestFailed ?? 0
      const inp = found?.inputToken ?? 0
      const out = found?.outputToken ?? 0
      const cr = found?.cachedReadInputTokens ?? 0
      const cc = found?.cachedCreationInputTokens ?? 0
      result.push({
        label: `${h}`,
        requests: reqS + reqF,
        requestSuccess: reqS,
        requestFailed: reqF,
        inputToken: inp,
        outputToken: out,
        cachedRead: cr,
        cachedCreate: cc,
        totalTokens: inp + out + cr + cc,
        cost: found?.totalCost ?? 0,
      })
    }
    return result
  }

  const dailyMap = new Map<string, UsageAccumulator>()
  for (const item of dailyStats.value) {
    const dayKey = item.date.slice(0, 10)
    const existing = dailyMap.get(dayKey) ?? createUsageAccumulator()
    mergeUsageAccumulator(existing, item)
    dailyMap.set(dayKey, existing)
  }

  const dr = dateRange.value
  const startDate = dr ? startOfDay(dr.start) : startOfDay(addDays(now(), -29))
  const endDate = dr ? endOfDay(dr.end) : endOfDay(now())
  const result: ChartPoint[] = []
  let cur = startDate

  while (cur <= endDate) {
    const dateStr = toDateKey(cur)
    const found = dailyMap.get(dateStr)
    const reqS = found?.requestSuccess ?? 0
    const reqF = found?.requestFailed ?? 0
    const inp = found?.inputToken ?? 0
    const out = found?.outputToken ?? 0
    const cr = found?.cachedReadInputTokens ?? 0
    const cc = found?.cachedCreationInputTokens ?? 0
    result.push({
      label: dayjs(cur).format('M/D'),
      requests: reqS + reqF,
      requestSuccess: reqS,
      requestFailed: reqF,
      inputToken: inp,
      outputToken: out,
      cachedRead: cr,
      cachedCreate: cc,
      totalTokens: inp + out + cr + cc,
      cost: found?.totalCost ?? 0,
    })
    cur = addDays(cur, 1)
  }
  return result
})

// ── ECharts ──
const chartRef = ref<HTMLDivElement>()
const chartInstance = shallowRef<echarts.ECharts>()

function buildChartOption(data: ChartPoint[]): echarts.EChartsOption {
  const labels = data.map(d => d.label)
  const isHourly = timeRange.value === 'today'
  const seriesDefs: Array<{ key: UnifiedSeriesKey; series: echarts.SeriesOption }> = [
    {
      key: 'inputToken',
      series: {
        name: '输入 Token',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: '#8bc34a', borderRadius: [0, 0, 0, 0] },
        barMaxWidth: 20,
        data: data.map(d => d.inputToken),
      },
    },
    {
      key: 'outputToken',
      series: {
        name: '输出 Token',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(139, 195, 74, 0.55)' },
        barMaxWidth: 20,
        data: data.map(d => d.outputToken),
      },
    },
    {
      key: 'cachedRead',
      series: {
        name: '缓存读',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(255, 179, 0, 0.7)' },
        barMaxWidth: 20,
        data: data.map(d => d.cachedRead),
      },
    },
    {
      key: 'cachedCreate',
      series: {
        name: '缓存创建',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(255, 213, 79, 0.6)', borderRadius: [2, 2, 0, 0] },
        barMaxWidth: 20,
        data: data.map(d => d.cachedCreate),
      },
    },
    {
      key: 'requests',
      series: {
        name: '请求数',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { color: '#e57373', width: 1.5 },
        itemStyle: { color: '#e57373' },
        data: data.map(d => d.requests),
      },
    },
    {
      key: 'cost',
      series: {
        name: '费用',
        type: 'line',
        smooth: true,
        symbol: 'diamond',
        symbolSize: 4,
        lineStyle: { color: '#ffb300', width: 1.5, type: 'dashed' },
        itemStyle: { color: '#ffb300' },
        data: data.map(d => d.cost),
      },
    },
  ]
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 30, 30, 0.95)',
      borderColor: 'rgba(139, 195, 74, 0.2)',
      borderWidth: 1,
      textStyle: { color: '#e0e0e0', fontSize: 12 },
      formatter(params: any) {
        const idx = params[0]?.dataIndex
        if (idx == null || !data[idx]) return ''
        const d = data[idx]
        const title = isHourly ? `${d.label}:00` : d.label
        let html = `<div style="font-weight:600;margin-bottom:4px;font-family:var(--font-mono)">${title}</div>`
        html += `<div style="display:flex;justify-content:space-between;gap:16px"><span style="color:#aaa">请求</span><span>${d.requests} <span style="color:#888;font-size:11px">(成功 ${d.requestSuccess} / 失败 ${d.requestFailed})</span></span></div>`
        html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">Token</span><span>${formatTokens(d.totalTokens)}</span></div>`
        html += `<div style="display:grid;grid-template-columns:1fr 1fr;gap:2px 8px;border-top:1px solid rgba(255,255,255,0.1);border-bottom:1px solid rgba(255,255,255,0.1);padding:4px 0;margin:4px 0;font-size:11px;color:#888">`
        html += `<span>读 ${formatTokens(d.inputToken)}</span><span>写 ${formatTokens(d.outputToken)}</span>`
        html += `<span>缓存读 ${formatTokens(d.cachedRead)}</span><span>缓存创建 ${formatTokens(d.cachedCreate)}</span></div>`
        html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">费用</span><span style="color:#ffb300;font-weight:600">$${d.cost.toFixed(4)}</span></div>`
        return html
      },
    },
    legend: {
      show: true,
      bottom: 0,
      left: 'center',
      textStyle: {
        color: '#8f949e',
        fontSize: 11,
      },
      itemWidth: 12,
      itemHeight: 8,
      itemGap: 16,
    },
    grid: {
      left: 50,
      right: 50,
      top: 10,
      bottom: 48,
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      axisTick: { show: false },
      axisLabel: {
        color: '#888',
        fontSize: 10,
        interval: labels.length > 12 ? Math.floor(labels.length / 10) : 0,
      },
    },
    yAxis: [
      {
        type: 'value',
        position: 'left',
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)', type: 'dashed' } },
        axisLabel: {
          color: '#888',
          fontSize: 10,
          formatter(v: number) { return formatTokens(v) },
        },
      },
      {
        type: 'value',
        position: 'right',
        splitLine: { show: false },
        axisLabel: {
          color: '#e57373',
          fontSize: 10,
          formatter(v: number) { return formatNumber(v) },
        },
      },
    ],
    series: seriesDefs.map(item => item.series),
  }
}

function updateChart() {
  if (!chartInstance.value) return
  chartInstance.value.setOption(buildChartOption(unifiedChartData.value), { notMerge: true })
}

function disposeMainChart() {
  chartInstance.value?.dispose()
  chartInstance.value = undefined
}

function handleResize() {
  chartInstance.value?.resize()
  rtChartInstance.value?.resize()
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  disposeMainChart()
  disposeRtChart()
})

watch(
  [() => chartRef.value, unifiedChartData],
  () => {
    if (!chartRef.value || unifiedChartData.value.length === 0) {
      disposeMainChart()
      return
    }
    if (!chartInstance.value) {
      chartInstance.value = echarts.init(chartRef.value)
    }
    updateChart()
    chartInstance.value.resize()
  },
  { immediate: true, flush: 'post' },
)

// ── Realtime ECharts ──
const rtChartRef = ref<HTMLDivElement>()
const rtChartInstance = shallowRef<echarts.ECharts>()

function buildRtChartOption(data: WindowPoint[]): echarts.EChartsOption {
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(30, 30, 30, 0.95)',
      borderColor: 'rgba(139, 195, 74, 0.2)',
      borderWidth: 1,
      textStyle: { color: '#e0e0e0', fontSize: 12 },
      formatter(params: any) {
        const idx = params[0]?.dataIndex
        if (idx == null || !data[idx]) return ''
        const d = data[idx]
        let html = `<div style="font-weight:600;margin-bottom:4px;font-family:var(--font-mono)">${d.time}</div>`
        html += `<div style="display:flex;justify-content:space-between;gap:16px"><span style="color:#aaa">请求</span><span style="font-family:var(--font-mono)">${d.requests}</span></div>`
        html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">Token</span><span style="font-family:var(--font-mono)">${formatTokens(d.tokens)}</span></div>`
        if (d.ttft > 0) {
          html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">TTFT</span><span style="font-family:var(--font-mono)">${(d.ttft / 1000).toFixed(2)}s</span></div>`
        }
        return html
      },
    },
    legend: {
      show: true,
      bottom: 0,
      left: 'center',
      textStyle: {
        color: '#8f949e',
        fontSize: 11,
      },
      itemWidth: 12,
      itemHeight: 8,
      itemGap: 16,
    },
    grid: { left: 50, right: 50, top: 10, bottom: 48 },
    xAxis: {
      type: 'category',
      data: data.map(d => d.time),
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      axisTick: { show: false },
      axisLabel: {
        color: '#888',
        fontSize: 10,
        interval: data.length > 12 ? 2 : 0,
      },
    },
    yAxis: [
      {
        type: 'value',
        position: 'left',
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)', type: 'dashed' } },
        axisLabel: { color: '#888', fontSize: 10 },
      },
      {
        type: 'value',
        position: 'right',
        splitLine: { show: false },
        axisLabel: {
          color: '#ffb300',
          fontSize: 10,
          formatter(v: number) { return formatTokens(v) },
        },
      },
    ],
    series: [
      {
        name: '请求数',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { color: '#8bc34a', width: 2 },
        itemStyle: { color: '#8bc34a' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(139, 195, 74, 0.25)' },
            { offset: 1, color: 'rgba(139, 195, 74, 0.02)' },
          ]),
        },
        data: data.map(d => d.requests),
      },
      {
        name: 'Token',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbol: 'diamond',
        symbolSize: 4,
        lineStyle: { color: '#ffb300', width: 1.5, type: 'dashed' },
        itemStyle: { color: '#ffb300' },
        data: data.map(d => d.tokens),
      },
    ],
  }
}

function updateRtChart() {
  if (!rtChartInstance.value) return
  rtChartInstance.value.setOption(buildRtChartOption(realtimeData.value), { notMerge: true })
}

function disposeRtChart() {
  rtChartInstance.value?.dispose()
  rtChartInstance.value = undefined
}

// ── Computed: real-time window (aggregate window3h from all channels) ──
interface WindowPoint {
  time: string
  requests: number
  tokens: number
  ttft: number
}

const realtimeData = computed<WindowPoint[]>(() => {
  const buckets = new Map<string, { requests: number; tokens: number; ttftSum: number; ttftCount: number; ts: number }>()

  for (const ch of channelStats.value) {
    if (!ch.window3h?.buckets) continue
    for (const b of ch.window3h.buckets) {
      const key = b.startAt
      const existing = buckets.get(key)
      const reqs = b.requestSuccess + b.requestFailed
      const toks = b.inputToken + b.outputToken
      const parsedTs = dayjs(key).valueOf()
      const ts = Number.isNaN(parsedTs) ? Number.MAX_SAFE_INTEGER : parsedTs
      if (existing) {
        existing.requests += reqs
        existing.tokens += toks
        if (b.avgTTFT > 0) {
          existing.ttftSum += b.avgTTFT
          existing.ttftCount++
        }
      } else {
        buckets.set(key, {
          requests: reqs,
          tokens: toks,
          ttftSum: b.avgTTFT > 0 ? b.avgTTFT : 0,
          ttftCount: b.avgTTFT > 0 ? 1 : 0,
          ts,
        })
      }
    }
  }

  const sorted = Array.from(buckets.entries()).sort((a, b) => a[1].ts - b[1].ts)
  return sorted.map(([time, v]) => ({
    time: (() => {
      const d = dayjs(time)
      if (!d.isValid()) return time.slice(11, 16)
      return d.format('HH:mm')
    })(),
    requests: v.requests,
    tokens: v.tokens,
    ttft: v.ttftCount > 0 ? Math.round(v.ttftSum / v.ttftCount) : 0,
  }))
})

const hasRealtimeData = computed(() => realtimeData.value.length > 0)

watch(
  [() => rtChartRef.value, realtimeData],
  () => {
    if (!rtChartRef.value || !hasRealtimeData.value) {
      disposeRtChart()
      return
    }
    if (!rtChartInstance.value) {
      rtChartInstance.value = echarts.init(rtChartRef.value)
    }
    updateRtChart()
    rtChartInstance.value.resize()
  },
  { immediate: true, flush: 'post' },
)

// ── Heatmap helpers ──
function formatHeatmapDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = dayjs(dateStr)
  const today = dayjs().startOf('day')
  const diff = today.diff(d.startOf('day'), 'day')
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  return d.format('M/D')
}

interface HeatmapCell {
  date: string
  hour: number
  requests: number
  requestSuccess: number
  requestFailed: number
  inputToken: number
  outputToken: number
  cachedReadInputTokens: number
  cachedCreationInputTokens: number
  totalCostMicros: number
  totalCost: number
}

const heatmapCells = computed<HeatmapCell[]>(() => {
  const today = now()
  const cells: HeatmapCell[] = []
  const localHourlyMap = new Map<string, UsageAccumulator & { totalCostMicros: number }>()

  for (const item of heatmapStats.value) {
    const key = `${item.date.slice(0, 10)}-${item.hour}`
    const existing = localHourlyMap.get(key) ?? {
      ...createUsageAccumulator(),
      totalCostMicros: 0,
    }
    mergeUsageAccumulator(existing, item)
    existing.totalCostMicros += item.totalCostMicros
    localHourlyMap.set(key, existing)
  }

  for (let dayOffset = 6; dayOffset >= 0; dayOffset--) {
    const d = addDays(now(), -dayOffset)
    const dateStr = toDateKey(d)
    for (let h = 0; h < 24; h++) {
      const found = localHourlyMap.get(`${dateStr}-${h}`)
      cells.push({
        date: dateStr,
        hour: h,
        requests: found ? found.requestSuccess + found.requestFailed : 0,
        requestSuccess: found?.requestSuccess ?? 0,
        requestFailed: found?.requestFailed ?? 0,
        inputToken: found?.inputToken ?? 0,
        outputToken: found?.outputToken ?? 0,
        cachedReadInputTokens: found?.cachedReadInputTokens ?? 0,
        cachedCreationInputTokens: found?.cachedCreationInputTokens ?? 0,
        totalCostMicros: found?.totalCostMicros ?? 0,
        totalCost: found?.totalCost ?? 0,
      })
    }
  }
  return cells
})

const heatmapRows = computed<HeatmapCell[][]>(() => {
  const rows: HeatmapCell[][] = []
  for (let i = 0; i < heatmapCells.value.length; i += 24) {
    rows.push(heatmapCells.value.slice(i, i + 24))
  }
  return rows
})

const maxHeatmapRequests = computed(() => {
  return Math.max(...heatmapCells.value.map(c => c.requests), 1)
})

function getHeatmapOpacity(requests: number): number {
  if (requests === 0) return 0.04
  return 0.12 + (requests / maxHeatmapRequests.value) * 0.78
}

// ── Load data ──
async function loadData() {
  loading.value = true
  loadError.value = ''

  try {
    const userId = authStore.user?.id
    if (!userId) {
      loadError.value = '未获取到用户信息，请重新登录'
      loading.value = false
      return
    }

    const dr = dateRange.value
    const dailyParams: Parameters<typeof listUserUsageDailyStats>[0] = {
      userID: userId,
      pageSize: 365,
    }
    if (dr) {
      // Query with a one-day buffer to avoid edge clipping around day boundaries.
      dailyParams.startTime = toRFC3339(startOfDay(addDays(dr.start, -1)))
      dailyParams.endTime = toRFC3339(endOfDay(addDays(dr.end, 1)))
    }

    // Hourly query also keeps a small buffer; rendering will filter by selected range.
    const hourlyStart = dr ? startOfDay(addDays(dr.start, -1)) : startOfDay(addDays(now(), -31))
    const hourlyEnd = dr ? endOfDay(addDays(dr.end, 1)) : endOfDay(now())
    const hourlyPageSize = Math.max(
      72,
      Math.ceil(dayjs(hourlyEnd).diff(hourlyStart, 'hour')),
    )
    const hourlyParams: Parameters<typeof listUserUsageHourlyStats>[0] = {
      userID: userId,
      startTime: toRFC3339(hourlyStart),
      endTime: toRFC3339(hourlyEnd),
      pageSize: hourlyPageSize,
    }

    const heatmapEnd = now()
    const heatmapStart = addDays(heatmapEnd, -6)
    const heatmapQueryStart = startOfDay(addDays(heatmapStart, -1))
    const heatmapQueryEnd = endOfDay(addDays(heatmapEnd, 1))

    const tasks: Promise<any>[] = [
      timeRange.value === 'all' ? getUserStats(userId) : Promise.resolve(null),
      listUserUsageDailyStats(dailyParams),
      listUserUsageHourlyStats(hourlyParams),
      listUserUsageHourlyStats({
        userID: userId,
        startTime: toRFC3339(heatmapQueryStart),
        endTime: toRFC3339(heatmapQueryEnd),
        pageSize: 24 * 10,
      }),
    ]

    const [userStatsRes, dailyRes, hourlyRes, heatmapRes] = await Promise.all(tasks)

    if (userStatsRes) userStats.value = userStatsRes
    dailyStats.value = dailyRes.items
    hourlyStats.value = hourlyRes.items
    heatmapStats.value = heatmapRes.items

    channelStats.value = await listChannelStats()
  } catch (err: any) {
    loadError.value = err?.response?.data?.message || '加载统计数据失败'
    console.error('Failed to load stats:', err)
  } finally {
    loading.value = false
  }
}

watch(timeRange, () => {
  loadData()
})

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="max-w-[1600px]">
    <!-- Header with time range -->
    <header class="flex items-start justify-between mb-6 gap-4 flex-wrap">
      <div>
        <h1 class="text-2xl font-semibold text-text-primary m-0 font-display tracking-tight">统计看板</h1>
        <p class="text-text-muted text-[0.8125rem] mt-1">实时监控您的 API 使用情况</p>
      </div>
      <div class="flex items-center gap-3">
        <div class="flex bg-bg-tertiary rounded-md p-[3px] gap-[2px]">
          <AppButton
            v-for="opt in rangeOptions"
            :key="opt.value"
            :variant="timeRange === opt.value ? 'secondary' : 'ghost'"
            size="sm"
            @click="timeRange = opt.value"
          >
            {{ opt.label }}
          </AppButton>
        </div>
        <AppButton variant="secondary" size="sm" :loading="loading" @click="loadData" title="刷新数据">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10" />
            <polyline points="1 20 1 14 7 14" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
        </AppButton>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading && !userStats" class="flex flex-col items-center justify-center p-16 text-text-muted gap-3">
      <div class="w-7 h-7 border-2 border-border border-t-primary rounded-full animate-spin"></div>
      <span>加载统计数据...</span>
    </div>

    <!-- Error -->
    <div v-else-if="loadError" class="flex flex-col items-center justify-center p-16 text-text-muted gap-3">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-danger">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <p>{{ loadError }}</p>
      <AppButton variant="primary" size="sm" @click="loadData">重试</AppButton>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Overview Cards: 3 columns -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <!-- Total Requests -->
        <div class="bg-bg-card border border-border rounded-lg p-5 flex items-start gap-3.5 transition-colors duration-200 hover:border-border-hover hover:shadow-md">
          <div class="w-10 h-10 rounded-[10px] flex items-center justify-center shrink-0 bg-gradient-to-br from-primary/15 to-primary/[0.08] text-primary">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          </div>
          <div class="flex flex-col min-w-0 flex-1">
            <span class="text-[0.6875rem] font-medium text-text-muted uppercase tracking-widest">总请求数</span>
            <span class="text-2xl font-bold text-text-primary leading-tight mt-1 font-mono">{{ formatNumber(totalRequests) }}</span>
            <div class="flex flex-wrap gap-x-3.5 gap-y-2 mt-2">
              <span class="flex items-center gap-1.5 text-xs text-success">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                成功 {{ formatNumber(overviewStats.requestSuccess) }}
              </span>
              <span class="flex items-center gap-1.5 text-xs text-danger">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                失败 {{ formatNumber(overviewStats.requestFailed) }}
              </span>
              <span class="flex items-center gap-1.5 text-xs text-warning">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                成功率 {{ successRate }}%
              </span>
            </div>
          </div>
        </div>

        <!-- Total Tokens -->
        <div class="bg-bg-card border border-border rounded-lg p-5 flex items-start gap-3.5 transition-colors duration-200 hover:border-border-hover hover:shadow-md">
          <div class="w-10 h-10 rounded-[10px] flex items-center justify-center shrink-0 bg-gradient-to-br from-accent/12 to-accent/[0.06] text-accent">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          <div class="flex flex-col min-w-0 flex-1">
            <span class="text-[0.6875rem] font-medium text-text-muted uppercase tracking-widest">总 Token 数</span>
            <span class="text-2xl font-bold text-text-primary leading-tight mt-1 font-mono">{{ formatTokens(totalTokens) }}</span>
            <div class="flex flex-wrap gap-x-3.5 gap-y-2 mt-2">
              <span class="flex items-center gap-1.5 text-xs text-text-tertiary">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                输入 {{ formatTokens(overviewStats.inputToken) }}
              </span>
              <span class="flex items-center gap-1.5 text-xs text-text-tertiary">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                输出 {{ formatTokens(overviewStats.outputToken) }}
              </span>
              <span class="flex items-center gap-1.5 text-xs text-text-tertiary">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                缓存读 {{ formatTokens(overviewStats.cachedReadInputTokens) }}
              </span>
              <span class="flex items-center gap-1.5 text-xs text-text-tertiary">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                缓存创建 {{ formatTokens(overviewStats.cachedCreationInputTokens) }}
              </span>
            </div>
          </div>
        </div>

        <!-- Total Cost -->
        <div class="bg-bg-card border border-border rounded-lg p-5 flex items-start gap-3.5 transition-colors duration-200 hover:border-border-hover hover:shadow-md">
          <div class="w-10 h-10 rounded-[10px] flex items-center justify-center shrink-0 bg-gradient-to-br from-warning/10 to-warning/[0.05] text-warning">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <line x1="12" y1="1" x2="12" y2="23" />
              <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
            </svg>
          </div>
          <div class="flex flex-col min-w-0 flex-1">
            <span class="text-[0.6875rem] font-medium text-text-muted uppercase tracking-widest">总成本</span>
            <span class="text-2xl font-bold text-text-primary leading-tight mt-1 font-mono">${{ overviewStats.totalCost.toFixed(4) }}</span>
            <div class="flex flex-wrap gap-x-3.5 gap-y-2 mt-2">
              <span class="flex items-center gap-1.5 text-xs text-text-tertiary">
                <span class="w-[5px] h-[5px] rounded-full bg-current shrink-0"></span>
                按实际调用计费
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Heatmap -->
      <div class="mb-6">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-text-primary">使用活跃度</div>
          <span class="text-xs text-text-muted">最近 7 天 · 每小时</span>
        </div>
        <div class="bg-bg-card border border-border rounded-lg p-5">
          <div class="grid gap-[3px]" style="grid-template-columns: 40px repeat(24, 1fr);">
            <template v-for="(row, rowIndex) in heatmapRows" :key="rowIndex">
              <span class="text-[0.6875rem] text-text-muted text-right pr-1.5 font-mono self-center">{{ formatHeatmapDate(row[0]?.date || '') }}</span>
              <div
                v-for="cell in row"
                :key="`${cell.date}-${cell.hour}`"
                class="h-[18px] rounded-[2px] border border-[rgba(139,195,74,0.06)] box-border transition-all duration-[120ms] cursor-pointer relative z-[1] hover:z-10 hover:outline-[1.5px] hover:outline-primary hover:outline-offset-[-1px] hover:scale-125 group"
                :style="{ background: `rgba(139, 195, 74, ${getHeatmapOpacity(cell.requests)})` }"
              >
                <div class="hidden group-hover:block absolute bottom-[calc(100%+8px)] left-1/2 -translate-x-1/2 bg-bg-secondary border border-border-hover rounded-md px-3 py-2.5 min-w-[200px] pointer-events-none shadow-[0_8px_24px_rgba(0,0,0,0.25)] z-[100]">
                  <div class="text-xs font-semibold text-text-primary mb-1 font-mono">{{ cell.date }} {{ String(cell.hour).padStart(2, '0') }}:00</div>
                  <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                    <span class="text-text-muted">请求</span>
                    <span class="text-text-primary font-mono font-medium">{{ cell.requests }} <span class="text-text-tertiary font-normal text-[0.625rem]">(成功 {{ cell.requestSuccess }} / 失败 {{ cell.requestFailed }})</span></span>
                  </div>
                  <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                    <span class="text-text-muted">Token</span>
                    <span class="text-text-primary font-mono font-medium">{{ formatTokens(cell.inputToken + cell.outputToken + cell.cachedReadInputTokens + cell.cachedCreationInputTokens) }}</span>
                  </div>
                  <div class="grid grid-cols-2 gap-x-2 gap-y-0.5 text-[0.625rem] text-text-tertiary py-1 border-t border-b border-border my-1">
                    <span>读 {{ formatTokens(cell.inputToken) }}</span>
                    <span>写 {{ formatTokens(cell.outputToken) }}</span>
                    <span>缓存读 {{ formatTokens(cell.cachedReadInputTokens) }}</span>
                    <span>缓存创建 {{ formatTokens(cell.cachedCreationInputTokens) }}</span>
                  </div>
                  <div class="flex justify-between items-baseline text-[0.6875rem] py-0.5">
                    <span class="text-text-muted">费用</span>
                    <span class="text-accent font-mono font-semibold">${{ cell.totalCost.toFixed(4) }}</span>
                  </div>
                </div>
              </div>
            </template>
            <!-- X-axis labels: placeholder + 24 labels -->
            <span></span>
            <span
              v-for="h in 24"
              :key="`lbl-${h}`"
              class="text-center text-[0.625rem] text-text-muted font-mono"
            >
              {{ [1, 7, 13, 19, 24].includes(h) ? (h === 1 ? '0' : h === 7 ? '6' : h === 13 ? '12' : h === 19 ? '18' : '23') : '' }}
            </span>
          </div>
          <div class="flex items-center gap-2 mt-3 justify-end">
            <span class="text-[0.6875rem] text-text-muted">少</span>
            <div class="flex gap-[2px]">
              <div class="w-2.5 h-2.5 rounded-[2px]" style="background: rgba(139, 195, 74, 0.12);"></div>
              <div class="w-2.5 h-2.5 rounded-[2px]" style="background: rgba(139, 195, 74, 0.32);"></div>
              <div class="w-2.5 h-2.5 rounded-[2px]" style="background: rgba(139, 195, 74, 0.52);"></div>
              <div class="w-2.5 h-2.5 rounded-[2px]" style="background: rgba(139, 195, 74, 0.72);"></div>
              <div class="w-2.5 h-2.5 rounded-[2px]" style="background: rgba(139, 195, 74, 0.92);"></div>
            </div>
            <span class="text-[0.6875rem] text-text-muted">多</span>
          </div>
        </div>
      </div>

      <!-- Real-time Window (if data available) -->
      <div v-if="hasRealtimeData" class="mb-6">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-text-primary">
            <span class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></span>
            <span>实时观测</span>
          </div>
          <span class="text-xs text-text-muted">最近 3 小时 · 15 分钟粒度</span>
        </div>
        <div class="bg-bg-card border border-border rounded-lg p-5">
          <div ref="rtChartRef" class="w-full h-[280px]"></div>
        </div>
      </div>

      <!-- Unified Chart -->
      <div class="bg-bg-card border border-border rounded-lg overflow-hidden mb-6">
        <div class="flex items-center justify-between px-5 py-4 border-b border-border">
          <h3 class="text-sm font-semibold text-text-primary m-0">用量趋势</h3>
          <span class="text-[0.6875rem] text-text-muted bg-bg-tertiary px-2 py-0.5 rounded">{{ timeRange === 'today' ? '当天 · 小时' : timeRange === '7d' ? '7 天 · 日' : timeRange === '30d' ? '30 天 · 日' : '近 30 天 · 日' }}</span>
        </div>
        <div class="p-5">
          <div v-if="unifiedChartData.length > 0" ref="chartRef" class="w-full h-[280px]"></div>
          <div v-else class="h-[280px] flex items-center justify-center text-text-muted text-sm">暂无数据</div>
        </div>
      </div>

      <!-- Channel Rankings -->
      <div class="bg-bg-card border border-border rounded-lg overflow-hidden">
        <div class="flex items-center justify-between px-5 py-4 border-b border-border max-sm:flex-col max-sm:items-start max-sm:gap-2">
          <h3 class="text-sm font-semibold text-text-primary m-0">渠道排行</h3>
          <div class="flex items-center gap-3 flex-wrap max-sm:w-full max-sm:justify-between">
            <span class="text-xs text-text-muted">{{ rankedChannels.length }} 个渠道</span>
            <div class="inline-flex items-center gap-[2px] bg-bg-tertiary rounded-sm p-[2px]">
              <AppButton
                v-for="opt in channelRankSortOptions"
                :key="opt.key"
                :variant="channelRankSort === opt.key ? 'secondary' : 'ghost'"
                size="sm"
                @click="channelRankSort = opt.key"
              >
                {{ opt.label }}
              </AppButton>
            </div>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr>
                <th class="px-4 py-3 text-left text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap">渠道</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">请求</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">输入 Token</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">输出 Token</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">成功率</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">平均 TTFT</th>
                <th class="px-4 py-3 text-right text-[0.6875rem] font-semibold uppercase tracking-wider text-text-muted bg-bg-secondary border-b border-border whitespace-nowrap font-mono">成本</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ch in rankedChannels" :key="ch.channelID" class="transition-colors duration-150 hover:bg-bg-secondary">
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary">
                  <div class="flex items-center gap-2.5">
                    <span
                      class="w-7 h-7 rounded-md flex items-center justify-center text-xs font-semibold text-bg-primary shrink-0"
                      :class="{
                        'bg-[#10a37f]': ch.type === 'openai',
                        'bg-[#d97706]': ch.type === 'claude',
                        'bg-[#0078d4]': ch.type === 'azure',
                        'bg-text-tertiary': ch.type === 'custom',
                      }"
                    >
                      {{ ch.name.charAt(0) }}
                    </span>
                    <div class="flex flex-col gap-0.5">
                      <span class="font-medium text-text-primary">{{ ch.name }}</span>
                      <span class="text-[0.6875rem] text-text-muted font-mono">{{ ch.channelID.slice(0, 8) }}...</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">{{ formatNumber(ch.requestCount) }}</td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">{{ formatTokens(ch.inputToken) }}</td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">{{ formatTokens(ch.outputToken) }}</td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">
                  <AppBadge
                    :variant="
                      calcSuccessRate(ch.requestSuccess, ch.requestFailed) >= 95 ? 'success' :
                      calcSuccessRate(ch.requestSuccess, ch.requestFailed) >= 85 ? 'warning' :
                      'danger'
                    "
                    size="sm"
                  >
                    {{ calcSuccessRate(ch.requestSuccess, ch.requestFailed) }}%
                  </AppBadge>
                </td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">{{ ch.avgTTFT }}ms</td>
                <td class="px-4 py-3.5 border-b border-border text-[0.8125rem] text-text-secondary text-right font-mono whitespace-nowrap">${{ ch.totalCost.toFixed(4) }}</td>
              </tr>
              <tr v-if="rankedChannels.length === 0">
                <td colspan="7" class="p-12 text-center text-text-muted">暂无渠道数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>
