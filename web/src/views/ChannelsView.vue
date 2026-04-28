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
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  setChannelStatus,
  fetchModels,
  CHANNEL_TYPE_LABELS,
  CHANNEL_STATUS_LABELS,
  type CreateChannelRequest,
  type UpdateChannelRequest,
} from '@/api/channel'
import { listChannelGroups } from '@/api/channel-group'
import { formatTokens, formatNumber, calcSuccessRate } from '@/api/stats'
import type { Channel, ChannelGroup, ChannelType, ChannelStatus } from '@/types'
import {
  fadeUp,
  slideUp,
  tableRow,
  dialogContent as dialogAnim,
} from '@/utils/motion'
import CustomSelect from '@/components/CustomSelect.vue'

// Type options for filter
const typeOptions = computed(() => {
  const options = [{ value: '', label: '全部类型' }]
  for (const [key, label] of Object.entries(CHANNEL_TYPE_LABELS)) {
    options.push({ value: key, label })
  }
  return options
})

// Status options for filter
const statusOptions = computed(() => {
  return [
    { value: '', label: '全部状态' },
    { value: 'enabled', label: '启用', color: 'var(--color-success)' },
    { value: 'disabled', label: '禁用', color: 'var(--color-warning)' },
    { value: 'expired', label: '过期', color: 'var(--color-danger)' },
  ]
})

// Group options for filter (computed from loaded groups)
const groupOptions = computed(() => {
  const options = [{ value: '', label: '全部分组' }]
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
const filterType = ref<ChannelType | ''>('')
const filterStatus = ref<ChannelStatus | ''>('')
const filterGroupID = ref<string | ''>('')

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
      type: filterType.value || undefined,
      status: filterStatus.value !== '' ? filterStatus.value : undefined,
      groupID: filterGroupID.value || undefined,
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
  }
  availableModels.value = []
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
  }
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

function handlePageChange(newPage: number) {
  page.value = newPage
  loadChannels()
}

onMounted(() => {
  loadChannels()
  loadGroups()
})
</script>

<template>
  <div v-motion="fadeUp" class="page-container">
    <!-- Page Header -->
    <header class="page-header">
      <div class="header-left">
        <span class="page-label">渠道管理</span>
        <div class="stats-bar">
          <div class="stat">
            <span class="stat-dot success"></span>
            <span class="stat-num">{{ statsEnabled }}</span>
            <span class="stat-text">运行中</span>
          </div>
          <div class="stat">
            <span class="stat-dot warning"></span>
            <span class="stat-num">{{ statsDisabled }}</span>
            <span class="stat-text">已暂停</span>
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
        新建渠道
      </button>
    </header>

    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-inputs">
        <input
          v-model="keyword"
          type="text"
          placeholder="搜索渠道名称..."
          class="input-search"
          @keyup.enter="handleSearch"
        />
        <CustomSelect
          v-model="filterType"
          :options="typeOptions"
          placeholder="全部类型"
          size="sm"
          @change="handleSearch"
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
            <th>渠道名称</th>
            <th>类型</th>
            <th>状态</th>
            <th>权重</th>
            <th>用量统计</th>
            <th>分组</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="8" class="cell-loading">
              <div class="spinner"></div>
              <span>加载中...</span>
            </td>
          </tr>
          <tr v-else-if="channels.length === 0">
            <td colspan="8" class="cell-empty">
              <div class="empty-icon">○</div>
              <span>暂无渠道数据</span>
            </td>
          </tr>
          <tr v-else v-for="(channel, index) in channels" :key="channel.id" v-motion="tableRow" :style="{ transitionDelay: `${index * 50}ms` }" class="data-row">
            <td>
              <div class="channel-info">
                <div class="channel-avatar" :class="channel.type">
                  {{ channel.name.charAt(0) }}
                </div>
                <div class="channel-meta">
                  <span class="channel-name">{{ channel.name }}</span>
                  <span class="channel-url">{{ channel.baseURL }}</span>
                </div>
              </div>
            </td>
            <td>
              <span class="type-tag" :class="channel.type">
                {{ CHANNEL_TYPE_LABELS[channel.type] }}
              </span>
            </td>
            <td>
              <span class="status-badge" :class="{ active: channel.status === 'enabled', paused: channel.status === 'disabled', expired: channel.status === 'expired' }">
                {{ CHANNEL_STATUS_LABELS[channel.status]?.label }}
              </span>
            </td>
            <td><span class="weight-value">{{ channel.weight }}</span></td>
            <td>
              <div v-if="channel.stats" class="stats-cell">
                <div class="stat-row">
                  <span class="stat-label">请求</span>
                  <span class="stat-val">{{ formatNumber(channel.stats.requestSuccess + channel.stats.requestFailed) }}</span>
                  <span class="stat-rate" :class="{ good: calcSuccessRate(channel.stats.requestSuccess, channel.stats.requestFailed) >= 95 }">
                    {{ calcSuccessRate(channel.stats.requestSuccess, channel.stats.requestFailed) }}%
                  </span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">输入</span>
                  <span class="stat-val">{{ formatTokens(channel.stats.inputToken) }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">输出</span>
                  <span class="stat-val">{{ formatTokens(channel.stats.outputToken) }}</span>
                </div>
              </div>
              <span v-else class="no-stats">—</span>
            </td>
            <td><span class="groups-count">{{ channel.groupIDs?.length || 0 }} 个</span></td>
            <td><span class="date-text">{{ new Date(channel.createdAt).toLocaleDateString() }}</span></td>
            <td class="text-right">
              <div class="row-actions">
                <button
                  class="action-link"
                  :class="channel.status === 'enabled' ? 'warn' : 'success'"
                  @click="handleToggleStatus(channel)"
                >
                  {{ channel.status === 'enabled' ? '暂停' : '启用' }}
                </button>
                <button class="action-link" @click="openEditDialog(channel)">编辑</button>
                <button class="action-link danger" @click="deletingChannel = channel">删除</button>
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
          class="dlg-content"
        >
          <div class="dlg-header">
            <DialogTitle class="dlg-title">
              {{ editingChannel ? '编辑渠道' : '新建渠道' }}
            </DialogTitle>
            <DialogClose class="dlg-close">×</DialogClose>
          </div>

          <DialogDescription class="sr-only">
            {{ editingChannel ? '编辑渠道配置' : '创建新渠道' }}
          </DialogDescription>

          <div class="dlg-tabs">
            <button
              v-for="tab in ['basic', 'models'] as const"
              :key="tab"
              class="dlg-tab"
              :class="{ active: activeTab === tab }"
              @click="activeTab = tab"
            >
              {{ tab === 'basic' ? '基本信息' : '模型配置' }}
            </button>
          </div>

          <form @submit.prevent="handleFormSubmit" class="dlg-body">
            <div v-if="formError" class="form-err">{{ formError }}</div>

            <div v-show="activeTab === 'basic'" class="form-grid">
              <div class="field">
                <label>渠道类型 <span class="req">*</span></label>
                <select v-model="formData.type">
                  <option v-for="(label, key) in CHANNEL_TYPE_LABELS" :key="key" :value="key">
                    {{ label }}
                  </option>
                </select>
              </div>
              <div class="field">
                <label>渠道名称 <span class="req">*</span></label>
                <input v-model="formData.name" type="text" placeholder="如: OpenAI 主账号" />
              </div>
              <div class="field full">
                <label>Base URL <span class="req">*</span></label>
                <input v-model="formData.baseURL" type="text" placeholder="https://api.openai.com/v1" autocomplete="off" />
              </div>
              <div class="field full">
                <label>API Key <span v-if="!editingChannel" class="req">*</span></label>
                <input
                  v-model="formData.apiKey"
                  type="text"
                  :placeholder="editingChannel ? '留空保持不变' : 'sk-...'"
                  autocomplete="off"
                />
              </div>
              <div class="field">
                <label>权重</label>
                <input v-model.number="formData.weight" type="number" min="0" />
              </div>
              <div v-if="editingChannel" class="field">
                <label>状态</label>
                <select v-model="editStatus">
                  <option value="enabled">启用</option>
                  <option value="disabled">禁用</option>
                </select>
              </div>
              <div class="field">
                <label>价格倍率</label>
                <input v-model.number="formData.priceRate" type="number" step="0.01" min="0" />
              </div>
              <div class="field full">
                <label>标签</label>
                <input v-model="formData.tag" type="text" placeholder="用于分类" />
              </div>
              <div class="field full">
                <label>备注</label>
                <textarea v-model="formData.remark" rows="2" placeholder="渠道备注"></textarea>
              </div>
              <div class="field full">
                <label>所属分组</label>
                <div class="chip-group">
                  <label
                    v-for="group in groups"
                    :key="group.id"
                    class="chip"
                    :class="{ on: formData.groupIDs?.includes(group.id) }"
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
                    {{ group.name }}
                  </label>
                  <span v-if="groups.length === 0" class="no-data">暂无分组</span>
                </div>
              </div>
            </div>

            <div v-show="activeTab === 'models'" class="form-grid">
              <div class="field full">
                <div class="field-header">
                  <label>模型配置</label>
                  <button
                    type="button"
                    class="btn-fetch"
                    :disabled="fetchingModels || !formData.baseURL || !formData.apiKey"
                    @click="handleFetchModels"
                  >
                    <span v-if="fetchingModels">获取中...</span>
                    <span v-else>自动获取</span>
                  </button>
                </div>
                <p class="hint">点击"自动获取"从 API 拉取模型列表，点击模型芯片选择/取消</p>

                <div class="model-section">
                  <div class="model-section-header">
                    <span class="model-tag system">系统模型</span>
                    <span class="model-count">{{ selectedModels.length }} 个已选</span>
                  </div>
                  <div class="chip-group models">
                    <button
                      v-for="model in availableModels"
                      :key="model"
                      type="button"
                      class="chip model system"
                      :class="{ on: selectedModels.includes(model) }"
                      @click="toggleModel(model)"
                    >
                      {{ model }}
                    </button>
                    <span v-if="availableModels.length === 0" class="no-models">
                      请先填写 Base URL 和 API Key，然后点击"自动获取"
                    </span>
                  </div>
                </div>

                <div class="model-section">
                  <div class="model-section-header">
                    <span class="model-tag custom">自定义模型</span>
                    <span class="model-count">{{ customModels.length }} 个</span>
                  </div>
                  <div class="chip-group models">
                    <span
                      v-for="model in customModels"
                      :key="model"
                      class="chip model custom"
                    >
                      {{ model }}
                      <button type="button" class="chip-remove" @click="removeCustomModel(model)">×</button>
                    </span>
                  </div>
                  <div class="add-model-row">
                    <input
                      v-model="newCustomModel"
                      type="text"
                      placeholder="输入模型名称"
                      @keyup.enter="addCustomModel"
                    />
                    <button type="button" class="btn-add-model" @click="addCustomModel">添加</button>
                  </div>
                </div>
              </div>
            </div>
          </form>

          <div class="dlg-footer">
            <DialogClose as-child>
              <button type="button" class="btn-cancel">取消</button>
            </DialogClose>
            <button type="button" class="btn-save" :disabled="formLoading" @click="handleFormSubmit">
              {{ formLoading ? '处理中...' : (editingChannel ? '保存' : '创建') }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- Delete Dialog -->
    <DialogRoot :open="!!deletingChannel" @update:open="(v) => !v && (deletingChannel = null)">
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
            确定要删除「{{ deletingChannel?.name }}」吗？此操作无法撤销。
          </DialogDescription>
          <div class="dlg-footer dlg-center">
            <button class="btn-cancel" @click="deletingChannel = null">取消</button>
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
}

.stat-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-text-muted);

  &.success {
    background: var(--color-success);
  }

  &.warning {
    background: var(--color-warning);
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
  max-width: 280px;
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
}

/* Channel Info */
.channel-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.channel-avatar {
  width: 36px;
  height: 36px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  font-weight: 600;
  color: white;

  &.openai { background: #10a37f; }
  &.claude { background: #d97706; }
  &.azure { background: #0078d4; }
  &.custom { background: #6b7280; }
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

.channel-url {
  font-size: 0.75rem;
  font-family: 'JetBrains Mono', monospace;
  color: var(--color-text-muted);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Type & Status */
.type-tag {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;

  &.openai {
    background: rgba(16, 163, 127, 0.1);
    color: #10a37f;
  }
  &.claude {
    background: rgba(217, 119, 6, 0.1);
    color: #d97706;
  }
  &.azure {
    background: rgba(0, 120, 212, 0.1);
    color: #0078d4;
  }
  &.custom {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 500;

  &::before {
    content: '';
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: currentColor;
  }

  &.active {
    background: var(--color-success-light);
    color: var(--color-success);
  }
  &.paused {
    background: var(--color-warning-light);
    color: var(--color-warning);
  }
  &.expired {
    background: var(--color-danger-light);
    color: var(--color-danger);
  }
}

.weight-value,
.groups-count,
.date-text {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

/* Stats Cell */
.stats-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.75rem;
}

.stat-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.stat-label {
  width: 28px;
  color: var(--color-text-muted);
}

.stat-val {
  font-family: 'JetBrains Mono', monospace;
  color: var(--color-text-secondary);
  min-width: 48px;
}

.stat-rate {
  font-size: 0.6875rem;
  padding: 0.125rem 0.375rem;
  background: var(--color-bg-tertiary);
  border-radius: 3px;
  color: var(--color-text-muted);

  &.good {
    background: var(--color-success-light);
    color: var(--color-success);
  }
}

.no-stats {
  color: var(--color-text-muted);
  font-size: 0.875rem;
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
  max-width: 560px;
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

.dlg-tabs {
  display: flex;
  gap: 0.25rem;
  padding: 0 1.25rem;
  background: var(--color-bg-secondary);
}

.dlg-tab {
  padding: 0.75rem 1rem;
  background: transparent;
  border: none;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
  position: relative;
  transition: color 0.15s;

  &:hover {
    color: var(--color-text-secondary);
  }

  &.active {
    color: var(--color-primary);

    &::after {
      content: '';
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      height: 2px;
      background: var(--color-primary);
    }
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

  .hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    margin-top: -0.125rem;
  }

  .field-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    label {
      margin: 0;
    }
  }

  .btn-fetch {
    padding: 0.25rem 0.625rem;
    background: var(--color-primary-light);
    border: 1px solid var(--color-primary);
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--color-primary);
    cursor: pointer;
    transition: all 0.15s;

    &:hover:not(:disabled) {
      background: var(--color-primary);
      color: white;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
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

    &.mono {
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.8125rem;
    }

    &:focus {
      outline: none;
      border-color: var(--color-primary);
    }
  }

  textarea {
    resize: vertical;
    min-height: 60px;
  }
}

.chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;

  &.models {
    min-height: 2rem;
  }
}

.chip {
  padding: 0.375rem 0.75rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    border-color: var(--color-primary);
  }

  &.on {
    background: var(--color-primary-light);
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &.model {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;

    &.system {
      background: rgba(139, 195, 74, 0.08);
      border-color: rgba(139, 195, 74, 0.3);
      color: var(--color-primary);

      &:hover {
        border-color: var(--color-primary);
      }

      &.on {
        background: var(--color-primary);
        color: white;
      }
    }

    &.custom {
      background: rgba(255, 213, 79, 0.08);
      border-color: rgba(255, 213, 79, 0.3);
      color: var(--color-warning);
      display: flex;
      align-items: center;
      gap: 0.25rem;
    }
  }
}

.chip-remove {
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 213, 79, 0.2);
  border: none;
  border-radius: 50%;
  color: var(--color-warning);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    background: var(--color-warning);
    color: white;
  }
}

.model-section {
  margin-top: 1rem;
  padding: 0.75rem;
  background: var(--color-bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.model-section-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.model-tag {
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.6875rem;
  font-weight: 500;
  text-transform: uppercase;

  &.system {
    background: var(--color-primary-light);
    color: var(--color-primary);
  }

  &.custom {
    background: rgba(255, 213, 79, 0.15);
    color: var(--color-warning);
  }
}

.model-count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.no-models {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  font-style: italic;
}

.add-model-row {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;

  input {
    flex: 1;
    padding: 0.375rem 0.5rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    font-size: 0.8125rem;
    color: var(--color-text-primary);
    font-family: 'JetBrains Mono', monospace;

    &:focus {
      outline: none;
      border-color: var(--color-warning);
    }
  }

  .btn-add-model {
    padding: 0.375rem 0.75rem;
    background: var(--color-warning);
    border: none;
    border-radius: 4px;
    font-size: 0.8125rem;
    font-weight: 500;
    color: white;
    cursor: pointer;
    transition: all 0.15s;

    &:hover {
      opacity: 0.9;
    }
  }
}

.no-data {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  font-style: italic;
}

.check-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;

  input {
    width: 16px;
    height: 16px;
    accent-color: var(--color-primary);
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
