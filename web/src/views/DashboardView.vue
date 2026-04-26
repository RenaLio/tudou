<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
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
import { fadeUp, staggerItem } from '@/utils/motion'

// Data
const loading = ref(true)
const userStats = ref<UserStatsResponse | null>(null)
const dailyStats = ref<UserUsageDailyStatsResponse[]>([])
const hourlyStats = ref<UserUsageHourlyStatsResponse[]>([])
const channelStats = ref<ChannelStatsResponse[]>([])
const channels = ref<Channel[]>([])

// Computed
const sortedChannelStats = computed(() => {
  const channelMap = new Map(channels.value.map(c => [c.id, c]))
  return channelStats.value
    .map(s => ({
      ...s,
      channelName: channelMap.get(s.channelID)?.name || s.channelID,
    }))
    .sort((a, b) => b.totalCostMicros - a.totalCostMicros)
})

const successRate = computed(() => {
  if (!userStats.value) return 100
  return calcSuccessRate(userStats.value.requestSuccess, userStats.value.requestFailed)
})

const avgTTFT = computed(() => {
  if (channelStats.value.length === 0) return 0
  const total = channelStats.value.reduce((sum, s) => sum + s.avgTTFT, 0)
  return Math.round(total / channelStats.value.length)
})

const avgTPS = computed(() => {
  if (channelStats.value.length === 0) return 0
  const total = channelStats.value.reduce((sum, s) => sum + s.avgTPS, 0)
  return (total / channelStats.value.length).toFixed(1)
})

// Chart data
const dailyChartData = computed(() => {
  return dailyStats.value.slice().reverse().map(d => ({
    date: d.date.slice(5), // MM-DD
    input: d.inputToken,
    output: d.outputToken,
    cost: d.totalCostMicros / 1_000_000,
  }))
})

const hourlyChartData = computed(() => {
  return hourlyStats.value.slice().reverse().map(h => ({
    hour: `${h.hour}:00`,
    requests: h.requestSuccess,
    tokens: h.inputToken + h.outputToken,
  }))
})

// Load data
async function loadData() {
  loading.value = true
  try {
    const today = new Date()
    const dateTo = today.toISOString().slice(0, 10)
    const dateFrom = new Date(today.getTime() - 14 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)

    const [userStatsRes, dailyRes, hourlyRes, channelsRes] = await Promise.all([
      getUserStats('3347227672726999040'), // admin user
      listUserUsageDailyStats({ dateFrom, dateTo, pageSize: 30 }),
      listUserUsageHourlyStats({ date: dateTo, pageSize: 24 }),
      listChannels({ pageSize: 100 }),
    ])

    userStats.value = userStatsRes
    dailyStats.value = dailyRes.items
    hourlyStats.value = hourlyRes.items
    channels.value = channelsRes.items

    // Load channel stats
    if (channels.value.length > 0) {
      const channelIDs = channels.value.map(c => c.id)
      channelStats.value = await listChannelStats(channelIDs)
    }
  } catch (err) {
    console.error('Failed to load stats:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div v-motion="fadeUp" class="dashboard">
    <!-- Header -->
    <header class="page-header">
      <h1 class="page-title">统计看板</h1>
      <p class="page-subtitle">实时监控您的 API 使用情况</p>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>加载统计数据...</span>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Key Metrics -->
      <div class="metrics-grid">
        <div v-motion="staggerItem" class="metric-card primary">
          <div class="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 2L2 7l10 5 10-5-10-5z"/>
              <path d="M2 17l10 5 10-5"/>
              <path d="M2 12l10 5 10-5"/>
            </svg>
          </div>
          <div class="metric-content">
            <span class="metric-label">总 Token</span>
            <span class="metric-value">{{ formatTokens((userStats?.inputToken || 0) + (userStats?.outputToken || 0)) }}</span>
            <span class="metric-detail">输入 {{ formatTokens(userStats?.inputToken || 0) }} · 输出 {{ formatTokens(userStats?.outputToken || 0) }}</span>
          </div>
        </div>

        <div v-motion="staggerItem" class="metric-card success">
          <div class="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
          </div>
          <div class="metric-content">
            <span class="metric-label">成功率</span>
            <span class="metric-value">{{ successRate }}%</span>
            <span class="metric-detail">{{ formatNumber(userStats?.requestSuccess || 0) }} 成功 · {{ formatNumber(userStats?.requestFailed || 0) }} 失败</span>
          </div>
        </div>

        <div v-motion="staggerItem" class="metric-card warning">
          <div class="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
          </div>
          <div class="metric-content">
            <span class="metric-label">平均 TTFT</span>
            <span class="metric-value">{{ avgTTFT }}<span class="metric-unit">ms</span></span>
            <span class="metric-detail">首字响应延迟</span>
          </div>
        </div>

        <div v-motion="staggerItem" class="metric-card info">
          <div class="metric-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <line x1="12" y1="1" x2="12" y2="23"/>
              <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
            </svg>
          </div>
          <div class="metric-content">
            <span class="metric-label">总成本</span>
            <span class="metric-value">{{ formatCost(userStats?.totalCostMicros || 0) }}</span>
            <span class="metric-detail">平均 TPS: {{ avgTPS }}</span>
          </div>
        </div>
      </div>

      <!-- Charts Row -->
      <div class="charts-row">
        <!-- Daily Trend -->
        <div class="chart-card">
          <div class="chart-header">
            <h3 class="chart-title">每日用量趋势</h3>
            <span class="chart-range">最近 14 天</span>
          </div>
          <div class="chart-body">
            <div class="bar-chart">
              <div class="chart-y-axis">
                <span>{{ formatTokens(Math.max(...dailyChartData.map(d => d.input + d.output))) }}</span>
                <span>{{ formatTokens(Math.max(...dailyChartData.map(d => d.input + d.output)) * 0.5) }}</span>
                <span>0</span>
              </div>
              <div class="chart-bars">
                <div
                  v-for="(d, i) in dailyChartData"
                  :key="d.date"
                  class="bar-group"
                  :style="{ animationDelay: `${i * 30}ms` }"
                >
                  <div class="bar-stack-wrapper">
                    <div class="bar-stack">
                      <div
                        class="bar input"
                        :style="{ height: `${(d.input / (Math.max(...dailyChartData.map(x => x.input + x.output)) || 1)) * 100}%` }"
                      ></div>
                      <div
                        class="bar output"
                        :style="{ height: `${(d.output / (Math.max(...dailyChartData.map(x => x.input + x.output)) || 1)) * 100}%` }"
                      ></div>
                    </div>
                    <div class="bar-tooltip">
                      <div class="tooltip-header">{{ d.date }}</div>
                      <div class="tooltip-row">
                        <span class="tooltip-dot input"></span>
                        <span>输入</span>
                        <span class="tooltip-value">{{ formatTokens(d.input) }}</span>
                      </div>
                      <div class="tooltip-row">
                        <span class="tooltip-dot output"></span>
                        <span>输出</span>
                        <span class="tooltip-value">{{ formatTokens(d.output) }}</span>
                      </div>
                      <div class="tooltip-row total">
                        <span>成本</span>
                        <span class="tooltip-value">{{ formatCost(d.cost * 1000000) }}</span>
                      </div>
                    </div>
                  </div>
                  <span class="bar-label">{{ d.date }}</span>
                </div>
              </div>
            </div>
            <div class="chart-legend">
              <span class="legend-item"><span class="dot input"></span>输入</span>
              <span class="legend-item"><span class="dot output"></span>输出</span>
            </div>
          </div>
        </div>

        <!-- Hourly Distribution -->
        <div class="chart-card">
          <div class="chart-header">
            <h3 class="chart-title">今日用量分布</h3>
            <span class="chart-range">24 小时</span>
          </div>
          <div class="chart-body">
            <div class="hourly-chart">
              <div
                v-for="(h, i) in hourlyChartData"
                :key="h.hour"
                class="hour-bar"
                :style="{ animationDelay: `${i * 20}ms` }"
              >
                <div class="hour-fill-wrapper">
                  <div
                    class="hour-fill"
                    :style="{ height: `${(h.requests / (Math.max(...hourlyChartData.map(x => x.requests)) || 1)) * 100}%` }"
                  ></div>
                  <div class="hour-tooltip">
                    <div class="tooltip-header">{{ h.hour }}</div>
                    <div class="tooltip-row">
                      <span>请求</span>
                      <span class="tooltip-value">{{ formatNumber(h.requests) }}</span>
                    </div>
                    <div class="tooltip-row">
                      <span>Token</span>
                      <span class="tooltip-value">{{ formatTokens(h.tokens) }}</span>
                    </div>
                  </div>
                </div>
                <span v-if="i % 4 === 0" class="hour-label">{{ h.hour }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Channel Stats Table -->
      <div class="table-card">
        <div class="table-header">
          <h3 class="table-title">渠道统计</h3>
          <span class="table-count">{{ channelStats.length }} 个渠道</span>
        </div>
        <div class="table-body">
          <table class="stats-table">
            <thead>
              <tr>
                <th>渠道</th>
                <th class="text-right">输入 Token</th>
                <th class="text-right">输出 Token</th>
                <th class="text-right">请求数</th>
                <th class="text-right">成功率</th>
                <th class="text-right">成本</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="stat in sortedChannelStats" :key="stat.channelID" class="stats-row">
                <td>
                  <div class="channel-cell">
                    <span class="channel-dot"></span>
                    <span class="channel-name">{{ stat.channelName }}</span>
                  </div>
                </td>
                <td class="text-right mono">{{ formatTokens(stat.inputToken) }}</td>
                <td class="text-right mono">{{ formatTokens(stat.outputToken) }}</td>
                <td class="text-right mono">{{ formatNumber(stat.requestSuccess) }}</td>
                <td class="text-right">
                  <span class="rate-badge" :class="{ high: calcSuccessRate(stat.requestSuccess, stat.requestFailed) >= 95, medium: calcSuccessRate(stat.requestSuccess, stat.requestFailed) >= 90 && calcSuccessRate(stat.requestSuccess, stat.requestFailed) < 95, low: calcSuccessRate(stat.requestSuccess, stat.requestFailed) < 90 }">
                    {{ calcSuccessRate(stat.requestSuccess, stat.requestFailed) }}%
                  </span>
                </td>
                <td class="text-right mono">{{ formatCost(stat.totalCostMicros) }}</td>
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

/* Header */
.page-header {
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--color-text-primary);
  margin: 0;
}

.page-subtitle {
  color: var(--color-text-muted);
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  color: var(--color-text-muted);
  gap: 1rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.metric-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  transition: transform 0.2s, box-shadow 0.2s;
}

.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.metric-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.metric-card.primary .metric-icon {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(139, 92, 246, 0.2));
  color: var(--color-primary);
}

.metric-card.success .metric-icon {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.2), rgba(52, 211, 153, 0.2));
  color: #10b981;
}

.metric-card.warning .metric-icon {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.2), rgba(251, 191, 36, 0.2));
  color: #f59e0b;
}

.metric-card.info .metric-icon {
  background: linear-gradient(135deg, rgba(6, 182, 212, 0.2), rgba(34, 211, 238, 0.2));
  color: #06b6d4;
}

.metric-content {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.metric-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.metric-value {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
  margin-top: 0.25rem;
}

.metric-unit {
  font-size: 1rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin-left: 0.125rem;
}

.metric-detail {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  margin-top: 0.25rem;
}

/* Charts Row */
.charts-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.chart-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  overflow: hidden;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.chart-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.chart-range {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  background: var(--color-bg-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.chart-body {
  padding: 1.25rem;
}

/* Bar Chart */
.bar-chart {
  display: flex;
  gap: 0.5rem;
  height: 200px;
}

.chart-y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  padding: 0.25rem 0;
  width: 40px;
  text-align: right;
}

.chart-bars {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  border-bottom: 1px solid var(--color-border);
  border-left: 1px solid var(--color-border);
  padding: 0 0.25rem;
}

.bar-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  animation: barGrow 0.5s ease-out backwards;
}

@keyframes barGrow {
  from {
    opacity: 0;
    transform: scaleY(0);
  }
  to {
    opacity: 1;
    transform: scaleY(1);
  }
}

.bar-stack-wrapper {
  position: relative;
  width: 100%;
  height: 160px;
}

.bar-stack {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column-reverse;
  border-radius: 2px 2px 0 0;
  overflow: hidden;
  transition: filter 0.2s;
}

.bar-group:hover .bar-stack {
  filter: brightness(1.1);
}

.bar {
  width: 100%;
  transition: height 0.3s ease-out;
}

.bar.input {
  background: var(--color-primary);
}

.bar.output {
  background: rgba(99, 102, 241, 0.4);
}

.bar-tooltip {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0.5rem 0.625rem;
  min-width: 100px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s;
  margin-bottom: 0.5rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 10;
}

.bar-group:hover .bar-tooltip {
  opacity: 1;
}

.tooltip-header {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-primary);
  padding-bottom: 0.375rem;
  margin-bottom: 0.375rem;
  border-bottom: 1px solid var(--color-border);
}

.tooltip-row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.tooltip-row.total {
  margin-top: 0.375rem;
  padding-top: 0.375rem;
  border-top: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-weight: 500;
}

.tooltip-dot {
  width: 6px;
  height: 6px;
  border-radius: 1px;
}

.tooltip-dot.input { background: var(--color-primary); }
.tooltip-dot.output { background: rgba(99, 102, 241, 0.6); }

.tooltip-value {
  margin-left: auto;
  font-family: 'JetBrains Mono', monospace;
  color: var(--color-text-primary);
}

.bar-label {
  font-size: 0.5625rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.chart-legend {
  display: flex;
  justify-content: center;
  gap: 1.5rem;
  margin-top: 0.75rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}

.dot.input { background: var(--color-primary); }
.dot.output { background: rgba(99, 102, 241, 0.4); }

/* Hourly Chart */
.hourly-chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 180px;
  padding: 0.5rem 0;
}

.hour-bar {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  animation: hourGrow 0.4s ease-out backwards;
}

@keyframes hourGrow {
  from {
    opacity: 0;
    transform: scaleY(0);
  }
  to {
    opacity: 1;
    transform: scaleY(1);
  }
}

.hour-fill-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.hour-fill {
  width: 100%;
  min-height: 4px;
  background: linear-gradient(to top, var(--color-primary), rgba(99, 102, 241, 0.6));
  border-radius: 2px 2px 0 0;
  transition: background 0.2s, filter 0.2s;
  flex: 1;
}

.hour-bar:hover .hour-fill {
  background: var(--color-primary);
  filter: brightness(1.1);
}

.hour-tooltip {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0.5rem 0.625rem;
  min-width: 80px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s;
  margin-bottom: 0.5rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 10;
}

.hour-bar:hover .hour-tooltip {
  opacity: 1;
}

.hour-label {
  font-size: 0.5625rem;
  color: var(--color-text-muted);
  margin-top: 0.25rem;
}

/* Table Card */
.table-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  overflow: hidden;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.stats-table {
  width: 100%;
  border-collapse: collapse;
}

.stats-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.stats-table td {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--color-border);
}

.stats-row {
  transition: background 0.15s;
}

.stats-row:hover {
  background: var(--color-bg-secondary);
}

.text-right {
  text-align: right;
}

.mono {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.8125rem;
}

.channel-cell {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.channel-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}

.channel-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.rate-badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.rate-badge.high {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.rate-badge.medium {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.rate-badge.low {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

/* Responsive */
@media (max-width: 1024px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-row {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }
}
</style>
