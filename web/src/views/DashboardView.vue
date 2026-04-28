<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, watch, shallowRef } from 'vue'
import * as echarts from 'echarts'
import { useAuthStore } from '@/stores/auth'
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
import { listChannels } from '@/api/channel'
import type { Channel } from '@/types'

const authStore = useAuthStore()

// ── Time helpers ──
function toRFC3339(d: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  const offset = -d.getTimezoneOffset()
  const offsetH = Math.abs(Math.floor(offset / 60))
  const offsetM = Math.abs(offset % 60)
  const offsetSign = offset >= 0 ? '+' : '-'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${offsetSign}${pad(offsetH)}:${pad(offsetM)}`
}

function startOfDay(d: Date): Date {
  const r = new Date(d)
  r.setHours(0, 0, 0, 0)
  return r
}

function endOfDay(d: Date): Date {
  const r = new Date(d)
  r.setHours(23, 59, 59, 999)
  return r
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d)
  r.setDate(r.getDate() + n)
  return r
}

function toDateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function parseUTCDateKey(dateLike: string): Date | null {
  const key = dateLike.slice(0, 10)
  const [yearRaw, monthRaw, dayRaw] = key.split('-')
  const year = Number(yearRaw)
  const month = Number(monthRaw)
  const day = Number(dayRaw)
  if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
    return null
  }
  return new Date(Date.UTC(year, month - 1, day))
}

function parseStatsDateHourToLocal(dateLike: string, hour: number): Date | null {
  const utcDate = parseUTCDateKey(dateLike)
  if (!utcDate) return null
  utcDate.setUTCHours(hour, 0, 0, 0)
  return utcDate
}

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
  const now = new Date()
  switch (timeRange.value) {
    case 'today':
      return { start: startOfDay(now), end: endOfDay(now) }
    case '7d':
      return { start: startOfDay(addDays(now, -6)), end: endOfDay(now) }
    case '30d':
      return { start: startOfDay(addDays(now, -29)), end: endOfDay(now) }
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
const channels = ref<Channel[]>([])

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
  const items = dailyStats.value
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
const channelMap = computed(() => new Map(channels.value.map(c => [c.id, c])))

const rankedChannels = computed(() => {
  return channelStats.value
    .map(s => ({
      ...s,
      name: channelMap.value.get(s.channelID)?.name || s.channelID,
      type: channelMap.value.get(s.channelID)?.type || 'custom',
    }))
    .sort((a, b) => b.totalCost - a.totalCost)
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

const unifiedChartData = computed<ChartPoint[]>(() => {
  if (timeRange.value === 'today') {
    // Hourly data for today, fill 0-23
    const data = hourlyStats.value
    const result: ChartPoint[] = []
    for (let h = 0; h < 24; h++) {
      const found = data.find(d => d.hour === h)
      const reqS = found ? found.requestSuccess : 0
      const reqF = found ? found.requestFailed : 0
      const inp = found ? found.inputToken : 0
      const out = found ? found.outputToken : 0
      const cr = found ? found.cachedReadInputTokens : 0
      const cc = found ? found.cachedCreationInputTokens : 0
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
        cost: found ? found.totalCost : 0,
      })
    }
    return result
  }

  // Daily data for 7d/30d/all
  const items = dailyStats.value.slice().sort((a, b) => a.date.localeCompare(b.date))
  const dr = dateRange.value

  // For 'all', limit to last 30 days of data
  const limited = timeRange.value === 'all' ? items.slice(-30) : items

  if (limited.length === 0) return []

  // Build a map for quick lookup
  const map = new Map<string, typeof limited[0]>()
  for (const d of limited) {
    map.set(d.date.slice(0, 10), d)
  }

  // Generate full date range and fill gaps
  const startDate = new Date(limited[0].date + 'T00:00:00')
  const endDate = new Date(limited[limited.length - 1].date + 'T00:00:00')
  const result: ChartPoint[] = []
  const cur = new Date(startDate)

  while (cur <= endDate) {
    const dateStr = `${cur.getFullYear()}-${String(cur.getMonth() + 1).padStart(2, '0')}-${String(cur.getDate()).padStart(2, '0')}`
    const found = map.get(dateStr)
    const reqS = found ? found.requestSuccess : 0
    const reqF = found ? found.requestFailed : 0
    const inp = found ? found.inputToken : 0
    const out = found ? found.outputToken : 0
    const cr = found ? found.cachedReadInputTokens : 0
    const cc = found ? found.cachedCreationInputTokens : 0
    result.push({
      label: `${cur.getMonth() + 1}/${cur.getDate()}`,
      requests: reqS + reqF,
      requestSuccess: reqS,
      requestFailed: reqF,
      inputToken: inp,
      outputToken: out,
      cachedRead: cr,
      cachedCreate: cc,
      totalTokens: inp + out + cr + cc,
      cost: found ? found.totalCost : 0,
    })
    cur.setDate(cur.getDate() + 1)
  }
  return result
})

// ── ECharts ──
const chartRef = ref<HTMLDivElement>()
const chartInstance = shallowRef<echarts.ECharts>()

function buildChartOption(data: ChartPoint[]): echarts.EChartsOption {
  const labels = data.map(d => d.label)
  const isHourly = timeRange.value === 'today'

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
        html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">费用</span><span style="color:#ffb300;font-weight:600">$${d.cost.toFixed(2)}</span></div>`
        return html
      },
    },
    legend: {
      show: false,
    },
    grid: {
      left: 50,
      right: 50,
      top: 10,
      bottom: 24,
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
    series: [
      {
        name: '输入 Token',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: '#8bc34a', borderRadius: [0, 0, 0, 0] },
        barMaxWidth: 20,
        data: data.map(d => d.inputToken),
      },
      {
        name: '输出 Token',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(139, 195, 74, 0.55)' },
        barMaxWidth: 20,
        data: data.map(d => d.outputToken),
      },
      {
        name: '缓存读',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(255, 179, 0, 0.7)' },
        barMaxWidth: 20,
        data: data.map(d => d.cachedRead),
      },
      {
        name: '缓存创建',
        type: 'bar',
        stack: 'tokens',
        itemStyle: { color: 'rgba(255, 213, 79, 0.6)', borderRadius: [2, 2, 0, 0] },
        barMaxWidth: 20,
        data: data.map(d => d.cachedCreate),
      },
      {
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
      {
        name: '费用',
        type: 'line',
        smooth: true,
        symbol: 'diamond',
        symbolSize: 4,
        lineStyle: { color: '#ffb300', width: 1.5, type: 'dashed' },
        itemStyle: { color: '#ffb300' },
        data: data.map(d => d.cost),
      },
    ],
  }
}

function updateChart() {
  if (!chartInstance.value) return
  chartInstance.value.setOption(buildChartOption(unifiedChartData.value), { notMerge: true })
}

function handleResize() {
  chartInstance.value?.resize()
  rtChartInstance.value?.resize()
}

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    updateChart()
  }
  if (rtChartRef.value && hasRealtimeData.value) {
    rtChartInstance.value = echarts.init(rtChartRef.value)
    updateRtChart()
  }
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance.value?.dispose()
  rtChartInstance.value?.dispose()
})

watch([unifiedChartData, timeRange], () => {
  updateChart()
})

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
          html += `<div style="display:flex;justify-content:space-between"><span style="color:#aaa">TTFT</span><span style="font-family:var(--font-mono)">${d.ttft}ms</span></div>`
        }
        return html
      },
    },
    legend: { show: false },
    grid: { left: 50, right: 50, top: 10, bottom: 24 },
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

// ── Computed: real-time window (aggregate window3h from all channels) ──
interface WindowPoint {
  time: string
  requests: number
  tokens: number
  ttft: number
}

const realtimeData = computed<WindowPoint[]>(() => {
  const buckets = new Map<string, { requests: number; tokens: number; ttftSum: number; ttftCount: number }>()

  for (const ch of channelStats.value) {
    if (!ch.window3h?.buckets) continue
    for (const b of ch.window3h.buckets) {
      const key = b.startAt
      const existing = buckets.get(key)
      const reqs = b.requestSuccess + b.requestFailed
      const toks = b.inputToken + b.outputToken
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
        })
      }
    }
  }

  const sorted = Array.from(buckets.entries()).sort((a, b) => a[0].localeCompare(b[0]))
  return sorted.map(([time, v]) => ({
    time: time.slice(11, 16),
    requests: v.requests,
    tokens: v.tokens,
    ttft: v.ttftCount > 0 ? Math.round(v.ttftSum / v.ttftCount) : 0,
  }))
})

const hasRealtimeData = computed(() => realtimeData.value.length > 0)

watch(realtimeData, () => {
  if (rtChartInstance.value) {
    updateRtChart()
  } else if (rtChartRef.value && hasRealtimeData.value) {
    rtChartInstance.value = echarts.init(rtChartRef.value)
    updateRtChart()
  }
})

// ── Heatmap helpers ──
function formatHeatmapDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const cellDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const diff = Math.floor((today.getTime() - cellDay.getTime()) / (1000 * 60 * 60 * 24))
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  return `${d.getMonth() + 1}/${d.getDate()}`
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
  const now = new Date()
  const cells: HeatmapCell[] = []

  for (let dayOffset = 6; dayOffset >= 0; dayOffset--) {
    const d = addDays(now, -dayOffset)
    const dateStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    for (let h = 0; h < 24; h++) {
      const found = heatmapStats.value.find(item =>
        item.date.slice(0, 10) === dateStr && item.hour === h,
      )
      cells.push({
        date: dateStr,
        hour: h,
        requests: found ? found.requestSuccess + found.requestFailed : 0,
        requestSuccess: found ? found.requestSuccess : 0,
        requestFailed: found ? found.requestFailed : 0,
        inputToken: found ? found.inputToken : 0,
        outputToken: found ? found.outputToken : 0,
        cachedReadInputTokens: found ? found.cachedReadInputTokens : 0,
        cachedCreationInputTokens: found ? found.cachedCreationInputTokens : 0,
        totalCostMicros: found ? found.totalCostMicros : 0,
        totalCost: found ? found.totalCost : 0,
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
      dailyParams.startTime = toRFC3339(dr.start)
      dailyParams.endTime = toRFC3339(dr.end)
    }

    // Hourly: for today use today's range; otherwise use the last day of the range
    const hourlyEnd = dr ? dr.end : new Date()
    const hourlyParams: Parameters<typeof listUserUsageHourlyStats>[0] = {
      userID: userId,
      startTime: toRFC3339(startOfDay(hourlyEnd)),
      endTime: toRFC3339(endOfDay(hourlyEnd)),
      pageSize: 24,
    }

    const heatmapEnd = new Date()
    const heatmapStart = addDays(heatmapEnd, -6)

    const tasks: Promise<any>[] = [
      timeRange.value === 'all' ? getUserStats(userId) : Promise.resolve(null),
      listUserUsageDailyStats(dailyParams),
      listUserUsageHourlyStats(hourlyParams),
      listChannels({ pageSize: 100 }),
      listUserUsageHourlyStats({
        userID: userId,
        startTime: toRFC3339(startOfDay(heatmapStart)),
        endTime: toRFC3339(endOfDay(heatmapEnd)),
        pageSize: 24 * 7,
      }),
    ]

    const [userStatsRes, dailyRes, hourlyRes, channelsRes, heatmapRes] = await Promise.all(tasks)

    if (userStatsRes) userStats.value = userStatsRes
    dailyStats.value = dailyRes.items
    hourlyStats.value = hourlyRes.items
    heatmapStats.value = heatmapRes.items
    channels.value = channelsRes.items

    if (channels.value.length > 0) {
      const channelIDs = channels.value.map(c => c.id)
      channelStats.value = await listChannelStats(channelIDs)
    } else {
      channelStats.value = []
    }
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
  <div class="dashboard">
    <!-- Header with time range -->
    <header class="page-header">
      <div>
        <h1 class="page-title">统计看板</h1>
        <p class="page-subtitle">实时监控您的 API 使用情况</p>
      </div>
      <div class="header-actions">
        <div class="range-tabs">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            class="range-tab"
            :class="{ active: timeRange === opt.value }"
            @click="timeRange = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
        <button class="refresh-btn" :class="{ spinning: loading }" @click="loadData" title="刷新数据">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10" />
            <polyline points="1 20 1 14 7 14" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
        </button>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading && !userStats" class="loading-state">
      <div class="spinner" />
      <span>加载统计数据...</span>
    </div>

    <!-- Error -->
    <div v-else-if="loadError" class="error-state">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <p>{{ loadError }}</p>
      <button class="retry-btn" @click="loadData">重试</button>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Overview Cards: 3 columns -->
      <div class="metrics-grid">
        <!-- Total Requests -->
        <div class="metric-card">
          <div class="metric-icon primary">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          </div>
          <div class="metric-info">
            <span class="metric-label">总请求数</span>
            <span class="metric-value">{{ formatNumber(totalRequests) }}</span>
            <div class="metric-breakdown">
              <span class="bd-item good">
                <span class="bd-dot" />
                成功 {{ formatNumber(overviewStats.requestSuccess) }}
              </span>
              <span class="bd-item bad">
                <span class="bd-dot" />
                失败 {{ formatNumber(overviewStats.requestFailed) }}
              </span>
              <span class="bd-item warn">
                <span class="bd-dot" />
                成功率 {{ successRate }}%
              </span>
            </div>
          </div>
        </div>

        <!-- Total Tokens -->
        <div class="metric-card">
          <div class="metric-icon accent">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          <div class="metric-info">
            <span class="metric-label">总 Token 数</span>
            <span class="metric-value">{{ formatTokens(totalTokens) }}</span>
            <div class="metric-breakdown">
              <span class="bd-item">
                <span class="bd-dot primary" />
                输入 {{ formatTokens(overviewStats.inputToken) }}
              </span>
              <span class="bd-item">
                <span class="bd-dot accent" />
                输出 {{ formatTokens(overviewStats.outputToken) }}
              </span>
              <span class="bd-item">
                <span class="bd-dot success" />
                缓存读 {{ formatTokens(overviewStats.cachedReadInputTokens) }}
              </span>
              <span class="bd-item">
                <span class="bd-dot warning" />
                缓存创建 {{ formatTokens(overviewStats.cachedCreationInputTokens) }}
              </span>
            </div>
          </div>
        </div>

        <!-- Total Cost -->
        <div class="metric-card">
          <div class="metric-icon warning">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <line x1="12" y1="1" x2="12" y2="23" />
              <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
            </svg>
          </div>
          <div class="metric-info">
            <span class="metric-label">总成本</span>
            <span class="metric-value">${{ overviewStats.totalCost.toFixed(2) }}</span>
            <div class="metric-breakdown">
              <span class="bd-item">
                <span class="bd-dot warning" />
                按实际调用计费
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Heatmap -->
      <div class="heatmap-section">
        <div class="section-header">
          <div class="section-title">使用活跃度</div>
          <span class="section-subtitle">最近 7 天 · 每小时</span>
        </div>
        <div class="heatmap-card">
          <div class="heatmap-grid">
            <template v-for="(row, rowIndex) in heatmapRows" :key="rowIndex">
              <span class="heatmap-label">{{ formatHeatmapDate(row[0]?.date || '') }}</span>
              <div
                v-for="cell in row"
                :key="`${cell.date}-${cell.hour}`"
                class="heatmap-cell"
                :style="{ background: `rgba(139, 195, 74, ${getHeatmapOpacity(cell.requests)})` }"
              >
                <div class="heatmap-tip">
                  <div class="tip-title">{{ cell.date }} {{ String(cell.hour).padStart(2, '0') }}:00</div>
                  <div class="tip-row">
                    <span class="tip-label">请求</span>
                    <span class="tip-val">{{ cell.requests }} <span class="tip-sub">(成功 {{ cell.requestSuccess }} / 失败 {{ cell.requestFailed }})</span></span>
                  </div>
                  <div class="tip-row">
                    <span class="tip-label">Token</span>
                    <span class="tip-val">{{ formatTokens(cell.inputToken + cell.outputToken + cell.cachedReadInputTokens + cell.cachedCreationInputTokens) }}</span>
                  </div>
                  <div class="tip-grid">
                    <span>读 {{ formatTokens(cell.inputToken) }}</span>
                    <span>写 {{ formatTokens(cell.outputToken) }}</span>
                    <span>缓存读 {{ formatTokens(cell.cachedReadInputTokens) }}</span>
                    <span>缓存创建 {{ formatTokens(cell.cachedCreationInputTokens) }}</span>
                  </div>
                  <div class="tip-row">
                    <span class="tip-label">费用</span>
                    <span class="tip-val tip-cost">${{ cell.totalCost.toFixed(2) }}</span>
                  </div>
                </div>
              </div>
            </template>
            <!-- X-axis labels: placeholder + 24 labels -->
            <span class="heatmap-label heatmap-label-axis" />
            <span
              v-for="h in 24"
              :key="`lbl-${h}`"
              class="heatmap-x-label"
            >
              {{ [1, 7, 13, 19, 24].includes(h) ? (h === 1 ? '0' : h === 7 ? '6' : h === 13 ? '12' : h === 19 ? '18' : '23') : '' }}
            </span>
          </div>
          <div class="heatmap-legend">
            <span class="heatmap-legend-label">少</span>
            <div class="heatmap-legend-scale">
              <div class="heatmap-legend-block" style="background: rgba(139, 195, 74, 0.04)" />
              <div class="heatmap-legend-block" style="background: rgba(139, 195, 74, 0.2)" />
              <div class="heatmap-legend-block" style="background: rgba(139, 195, 74, 0.4)" />
              <div class="heatmap-legend-block" style="background: rgba(139, 195, 74, 0.6)" />
              <div class="heatmap-legend-block" style="background: rgba(139, 195, 74, 0.85)" />
            </div>
            <span class="heatmap-legend-label">多</span>
          </div>
        </div>
      </div>

      <!-- Real-time Window (if data available) -->
      <div v-if="hasRealtimeData" class="rt-section">
        <div class="section-header">
          <div class="section-title">
            <span class="live-dot" />
            <span>实时观测</span>
          </div>
          <span class="section-subtitle">最近 3 小时 · 15 分钟粒度</span>
        </div>
        <div class="rt-card">
          <div ref="rtChartRef" class="echarts-container" />
          <div class="rt-legend">
            <span class="legend-item">
              <span class="legend-dot" style="background: #8bc34a" />
              请求数
            </span>
            <span class="legend-item">
              <span class="legend-line" style="background: #ffb300" />
              Token
            </span>
          </div>
        </div>
      </div>

      <!-- Unified Chart -->
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">用量趋势</h3>
          <span class="chart-badge">{{ timeRange === 'today' ? '当天 · 小时' : timeRange === '7d' ? '7 天 · 日' : timeRange === '30d' ? '30 天 · 日' : '近 30 天 · 日' }}</span>
        </div>
        <div class="chart-body">
          <div v-if="unifiedChartData.length > 0" ref="chartRef" class="echarts-container" />
          <div v-else class="chart-empty">暂无数据</div>

          <!-- Legend -->
          <div class="unified-legend">
            <span class="legend-item"><span class="legend-dot" style="background: #8bc34a" />输入 Token</span>
            <span class="legend-item"><span class="legend-dot" style="background: rgba(139, 195, 74, 0.55)" />输出 Token</span>
            <span class="legend-item"><span class="legend-dot" style="background: rgba(255, 179, 0, 0.7)" />缓存读</span>
            <span class="legend-item"><span class="legend-dot" style="background: rgba(255, 213, 79, 0.6)" />缓存创建</span>
            <span class="legend-item"><span class="legend-line" style="background: #e57373" />请求数</span>
            <span class="legend-item"><span class="legend-line" style="background: #ffb300" />费用</span>
          </div>
        </div>
      </div>

      <!-- Channel Rankings -->
      <div class="table-card">
        <div class="table-header">
          <h3 class="table-title">渠道排行</h3>
          <span class="table-count">{{ rankedChannels.length }} 个渠道</span>
        </div>
        <div class="table-body">
          <table class="data-table">
            <thead>
              <tr>
                <th>渠道</th>
                <th class="num">请求</th>
                <th class="num">输入 Token</th>
                <th class="num">输出 Token</th>
                <th class="num">成功率</th>
                <th class="num">平均 TTFT</th>
                <th class="num">成本</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ch in rankedChannels" :key="ch.channelID" class="data-row">
                <td>
                  <div class="channel-info">
                    <span class="channel-avatar" :class="ch.type">{{ ch.name.charAt(0) }}</span>
                    <div class="channel-meta">
                      <span class="channel-name">{{ ch.name }}</span>
                      <span class="channel-id">{{ ch.channelID.slice(0, 8) }}...</span>
                    </div>
                  </div>
                </td>
                <td class="num">{{ formatNumber(ch.requestSuccess + ch.requestFailed) }}</td>
                <td class="num">{{ formatTokens(ch.inputToken) }}</td>
                <td class="num">{{ formatTokens(ch.outputToken) }}</td>
                <td class="num">
                  <span class="rate-badge" :class="{
                    high: calcSuccessRate(ch.requestSuccess, ch.requestFailed) >= 95,
                    medium: calcSuccessRate(ch.requestSuccess, ch.requestFailed) >= 85,
                    low: calcSuccessRate(ch.requestSuccess, ch.requestFailed) < 85,
                  }">
                    {{ calcSuccessRate(ch.requestSuccess, ch.requestFailed) }}%
                  </span>
                </td>
                <td class="num">{{ ch.avgTTFT }}ms</td>
                <td class="num">${{ ch.totalCost.toFixed(2) }}</td>
              </tr>
              <tr v-if="rankedChannels.length === 0">
                <td colspan="7" class="cell-empty">暂无渠道数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1400px;
}

/* ── Header with time range ── */
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 1.5rem;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  font-family: var(--font-display);
  letter-spacing: -0.02em;
}

.page-subtitle {
  color: var(--color-text-muted);
  font-size: 0.8125rem;
  margin: 0.25rem 0 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.range-tabs {
  display: flex;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-md);
  padding: 3px;
  gap: 2px;
}

.range-tab {
  padding: 0.375rem 0.875rem;
  background: transparent;
  border: none;
  border-radius: calc(var(--radius-md) - 2px);
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.range-tab:hover {
  color: var(--color-text-secondary);
}

.range-tab.active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: var(--shadow-sm);
}

.refresh-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-tertiary);
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.refresh-btn:hover {
  border-color: var(--color-border-hover);
  color: var(--color-text-secondary);
}

.refresh-btn.spinning svg {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Loading / Error ── */
.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  color: var(--color-text-muted);
  gap: 0.75rem;
}

.spinner {
  width: 28px;
  height: 28px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.error-state svg {
  color: var(--color-danger);
}

.retry-btn {
  margin-top: 0.5rem;
  padding: 0.375rem 1rem;
  background: var(--color-primary-light);
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-sm);
  color: var(--color-primary);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all 0.15s;
}

.retry-btn:hover {
  background: var(--color-primary);
  color: var(--color-bg-primary);
}

/* ── Metrics Grid: 3 columns ── */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.metric-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  align-items: flex-start;
  gap: 0.875rem;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.metric-card:hover {
  border-color: var(--color-border-hover);
  box-shadow: var(--shadow-md);
}

.metric-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.metric-icon.primary {
  background: linear-gradient(135deg, rgba(139, 195, 74, 0.15), rgba(124, 179, 66, 0.08));
  color: var(--color-primary);
}

.metric-icon.accent {
  background: linear-gradient(135deg, rgba(255, 179, 0, 0.12), rgba(255, 204, 77, 0.06));
  color: var(--color-accent);
}

.metric-icon.warning {
  background: linear-gradient(135deg, rgba(255, 213, 79, 0.1), rgba(255, 224, 130, 0.05));
  color: var(--color-warning);
}

.metric-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.metric-label {
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.metric-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
  margin-top: 0.25rem;
  font-family: var(--font-mono);
}

.metric-breakdown {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 0.875rem;
  margin-top: 0.5rem;
}

.bd-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.bd-item.good {
  color: var(--color-success);
}

.bd-item.bad {
  color: var(--color-danger);
}

.bd-item.warn {
  color: var(--color-warning);
}

.bd-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.bd-dot.primary {
  background: var(--color-primary);
}

.bd-dot.accent {
  background: var(--color-accent);
}

.bd-dot.success {
  background: var(--color-success);
}

.bd-dot.warning {
  background: var(--color-warning);
}

/* ── Real-time Section ── */
.rt-section {
  margin-bottom: 1.5rem;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success);
  animation: livePulse 2s ease-in-out infinite;
}

@keyframes livePulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(129, 199, 132, 0.4); }
  50% { opacity: 0.6; box-shadow: 0 0 0 4px rgba(129, 199, 132, 0); }
}

.section-subtitle {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.rt-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
}

.rt-legend {
  display: flex;
  gap: 1rem;
  margin-top: 0.75rem;
  justify-content: center;
}

/* ── Chart Card ── */
.chart-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: 1.5rem;
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.chart-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.chart-badge {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  background: var(--color-bg-tertiary);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.chart-body {
  padding: 1.25rem;
}

/* ── Unified Chart ── */
.echarts-container {
  width: 100%;
  height: 280px;
}

.chart-empty {
  height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.unified-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.5rem 1.25rem;
  margin-top: 0.75rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
  flex-shrink: 0;
}

.legend-line {
  width: 16px;
  height: 2px;
  border-radius: 1px;
  flex-shrink: 0;
}

/* ── Table Card ── */
.table-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.table-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.table-count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.table-body {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}

.data-table td {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--color-border);
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.data-row {
  transition: background 0.15s;
}

.data-row:hover {
  background: var(--color-bg-secondary);
}

.data-table .num {
  text-align: right;
  font-family: var(--font-mono);
  white-space: nowrap;
}

.cell-empty {
  padding: 3rem !important;
  text-align: center;
  color: var(--color-text-muted);
}

/* Channel info in table */
.channel-info {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.channel-avatar {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-bg-primary);
  flex-shrink: 0;
}

.channel-avatar.openai {
  background: #10a37f;
}

.channel-avatar.claude {
  background: #d97706;
}

.channel-avatar.azure {
  background: #0078d4;
}

.channel-avatar.custom {
  background: var(--color-text-tertiary);
}

.channel-meta {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.channel-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.channel-id {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
}

/* Rate badge */
.rate-badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.rate-badge.high {
  background: rgba(129, 199, 132, 0.1);
  color: var(--color-success);
}

.rate-badge.medium {
  background: rgba(255, 213, 79, 0.1);
  color: var(--color-warning);
}

.rate-badge.low {
  background: rgba(229, 115, 115, 0.1);
  color: var(--color-danger);
}

/* ── Heatmap ── */
.heatmap-section {
  margin-bottom: 1.5rem;
}

.heatmap-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
}

.heatmap-grid {
  display: grid;
  grid-template-columns: 40px repeat(24, 1fr);
  gap: 3px;
}

.heatmap-label {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  text-align: right;
  padding-right: 6px;
  font-family: var(--font-mono);
  align-self: center;
}

.heatmap-label-axis {
  visibility: hidden;
}

.heatmap-cell {
  height: 18px;
  border-radius: 2px;
  background: rgba(139, 195, 74, 0.04);
  border: 1px solid rgba(139, 195, 74, 0.06);
  box-sizing: border-box;
  transition: outline 0.12s ease;
  cursor: pointer;
  position: relative;
  z-index: 1;
}

.heatmap-cell:hover {
  z-index: 10;
  outline: 1.5px solid var(--color-primary);
  outline-offset: -1px;
}

/* ── Tooltip ── */
.heatmap-tip {
  display: none;
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-md);
  padding: 0.625rem 0.75rem;
  min-width: 200px;
  pointer-events: none;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  z-index: 100;
}

.heatmap-cell:hover .heatmap-tip {
  display: block;
}

.tip-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 0.375rem;
  font-family: var(--font-mono);
}

.tip-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 0.6875rem;
  padding: 0.125rem 0;
}

.tip-label {
  color: var(--color-text-muted);
}

.tip-val {
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  font-weight: 500;
}

.tip-sub {
  color: var(--color-text-tertiary);
  font-weight: 400;
  font-size: 0.625rem;
}

.tip-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.125rem 0.5rem;
  font-size: 0.625rem;
  color: var(--color-text-tertiary);
  padding: 0.25rem 0;
  border-top: 1px solid var(--color-border);
  border-bottom: 1px solid var(--color-border);
  margin: 0.25rem 0;
}

.tip-cost {
  color: var(--color-accent);
  font-weight: 600;
}

.heatmap-x-label {
  text-align: center;
  font-size: 0.625rem;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
}

.heatmap-legend {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  justify-content: flex-end;
}

.heatmap-legend-label {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
}

.heatmap-legend-scale {
  display: flex;
  gap: 2px;
}

.heatmap-legend-block {
  width: 10px;
  height: 10px;
  border-radius: 2px;
}

/* ── Responsive ── */
@media (max-width: 1024px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .header-actions {
    width: 100%;
    justify-content: space-between;
  }
}

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
  }

  .range-tabs {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .metric-breakdown {
    flex-direction: column;
    gap: 0.375rem;
  }
}
</style>
