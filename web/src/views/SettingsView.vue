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
import { updatePassword } from '@/api/auth'
import {
  listChannelGroups,
  createChannelGroup,
  updateChannelGroup,
  deleteChannelGroup,
  listLoadBalanceStrategies,
  type CreateChannelGroupRequest,
  type LoadBalanceStrategy,
} from '@/api/channel-group'
import type { ChannelGroup } from '@/types'
import {
  fadeUp,
  scaleIn,
  dialogContent as dialogAnim,
} from '@/utils/motion'

// Strategy options for select
const strategyOptions = computed(() => {
  return strategies.value.map(s => ({ value: s.value, label: s.label }))
})

// Password form
const newPassword = ref('')
const confirmPassword = ref('')
const passwordLoading = ref(false)
const passwordSuccess = ref(false)
const passwordError = ref('')

// Channel groups
const groups = ref<ChannelGroup[]>([])
const groupsLoading = ref(false)
const strategies = ref<LoadBalanceStrategy[]>([])

// Create/Update dialog
const dialogOpen = ref(false)
const editingGroup = ref<ChannelGroup | null>(null)
const formData = ref<CreateChannelGroupRequest>({
  name: '',
  nameRemark: '',
  loadBalanceStrategy: 'performance',
})
const formLoading = ref(false)
const formError = ref('')

// Delete confirmation
const deletingGroup = ref<ChannelGroup | null>(null)
const deleteLoading = ref(false)

async function handlePasswordSubmit() {
  if (!newPassword.value || !confirmPassword.value) {
    passwordError.value = '请填写所有字段'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = '两次输入的密码不一致'
    return
  }
  if (newPassword.value.length < 6) {
    passwordError.value = '密码长度至少6位'
    return
  }

  passwordLoading.value = true
  passwordError.value = ''
  passwordSuccess.value = false

  try {
    await updatePassword(newPassword.value)
    passwordSuccess.value = true
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '修改密码失败'
    passwordError.value = message
  } finally {
    passwordLoading.value = false
  }
}

async function loadGroups() {
  groupsLoading.value = true
  try {
    const result = await listChannelGroups()
    groups.value = result.items
  } catch {
    // ignore
  } finally {
    groupsLoading.value = false
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
  editingGroup.value = null
  formData.value = { name: '', nameRemark: '', loadBalanceStrategy: 'performance' }
  formError.value = ''
  dialogOpen.value = true
}

function openEditDialog(group: ChannelGroup) {
  editingGroup.value = group
  formData.value = {
    name: group.name,
    nameRemark: group.nameRemark || '',
    loadBalanceStrategy: group.loadBalanceStrategy,
  }
  formError.value = ''
  dialogOpen.value = true
}

async function handleFormSubmit() {
  if (!formData.value.name) {
    formError.value = '请输入组名称'
    return
  }

  formLoading.value = true
  formError.value = ''

  try {
    if (editingGroup.value) {
      await updateChannelGroup(editingGroup.value.id, formData.value)
    } else {
      await createChannelGroup(formData.value)
    }
    dialogOpen.value = false
    await loadGroups()
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
  if (!deletingGroup.value) return

  deleteLoading.value = true
  try {
    await deleteChannelGroup(deletingGroup.value.id)
    deletingGroup.value = null
    await loadGroups()
  } catch {
    // ignore
  } finally {
    deleteLoading.value = false
  }
}

onMounted(() => {
  loadGroups()
  loadStrategies()
})
</script>

<template>
  <div v-motion="fadeUp" class="settings-page">
    <!-- Page Header -->
    <header class="page-header">
      <h1 class="page-title">系统配置</h1>
      <p class="page-desc">管理渠道分组和账户安全设置</p>
    </header>

    <!-- Settings Grid -->
    <div class="settings-grid">
      <!-- Channel Groups Panel -->
      <section
        v-motion
        :initial="{ opacity: 0, y: 30, scale: 0.98 }"
        :enter="{ opacity: 1, y: 0, scale: 1, transition: { delay: 100, type: 'spring', stiffness: 200, damping: 25 } }"
        class="settings-panel"
      >
        <div class="panel-header">
          <div class="panel-icon groups">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
          </div>
          <div class="panel-info">
            <h2 class="panel-title">渠道分组</h2>
            <p class="panel-desc">用于令牌关联和负载均衡策略配置</p>
          </div>
          <button class="btn-add" @click="openCreateDialog">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            新建
          </button>
        </div>

        <div class="panel-body">
          <!-- Loading -->
          <div v-if="groupsLoading" class="loading-state">
            <div class="spinner"></div>
          </div>

          <!-- Empty -->
          <div v-else-if="groups.length === 0" class="empty-state">
            <div class="empty-visual">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect x="3" y="3" width="7" height="7" rx="1"/>
                <rect x="14" y="3" width="7" height="7" rx="1"/>
                <rect x="3" y="14" width="7" height="7" rx="1"/>
                <line x1="17" y1="17" x2="17" y2="21"/>
                <line x1="15" y1="19" x2="19" y2="19"/>
              </svg>
            </div>
            <p class="empty-text">暂无分组</p>
            <p class="empty-hint">创建分组后可在令牌中关联使用</p>
          </div>

          <!-- Group List -->
          <div v-else class="group-list">
            <div
              v-for="(group, index) in groups"
              :key="group.id"
              v-motion="scaleIn"
              :style="{ transitionDelay: `${index * 60}ms` }"
              class="group-item"
            >
              <div class="group-main">
                <div class="group-name">
                  {{ group.name }}
                  <span v-if="group.nameRemark" class="group-remark">{{ group.nameRemark }}</span>
                </div>
              </div>
              <div class="group-actions">
                <button class="icon-btn" @click="openEditDialog(group)">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="icon-btn danger" @click="deletingGroup = group">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Account Security Panel -->
      <section
        v-motion
        :initial="{ opacity: 0, y: 30, scale: 0.98 }"
        :enter="{ opacity: 1, y: 0, scale: 1, transition: { delay: 200, type: 'spring', stiffness: 200, damping: 25 } }"
        class="settings-panel"
      >
        <div class="panel-header">
          <div class="panel-icon security">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div class="panel-info">
            <h2 class="panel-title">账户安全</h2>
            <p class="panel-desc">修改登录密码和访问凭证</p>
          </div>
        </div>

        <div class="panel-body">
          <form @submit.prevent="handlePasswordSubmit" class="password-form">
            <!-- Success -->
            <div v-if="passwordSuccess" class="alert success">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
              </svg>
              密码已更新
            </div>

            <!-- Error -->
            <div v-if="passwordError" class="alert error">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              {{ passwordError }}
            </div>

            <div class="form-field">
              <label>新密码</label>
              <input
                v-model="newPassword"
                type="password"
                placeholder="至少 6 位字符"
                required
              />
            </div>

            <div class="form-field">
              <label>确认密码</label>
              <input
                v-model="confirmPassword"
                type="password"
                placeholder="再次输入新密码"
                required
              />
            </div>

            <button type="submit" class="btn-submit" :disabled="passwordLoading">
              <span v-if="passwordLoading">处理中...</span>
              <span v-else>更新密码</span>
            </button>
          </form>
        </div>
      </section>
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
              {{ editingGroup ? '编辑分组' : '新建分组' }}
            </DialogTitle>
            <DialogClose class="dlg-close">×</DialogClose>
          </div>

          <DialogDescription class="sr-only">
            {{ editingGroup ? '编辑渠道分组信息' : '创建新的渠道分组' }}
          </DialogDescription>

          <form @submit.prevent="handleFormSubmit" class="dlg-body">
            <div v-if="formError" class="form-err">{{ formError }}</div>

            <div class="field">
              <label>分组名称 <span class="req">*</span></label>
              <input v-model="formData.name" type="text" placeholder="请输入分组名称" required />
              <p class="hint">分组名称必须唯一</p>
            </div>

            <div class="field">
              <label>备注名</label>
              <input v-model="formData.nameRemark" type="text" placeholder="可选" />
            </div>

            <div class="field">
              <label>负载均衡策略</label>
              <select v-model="formData.loadBalanceStrategy">
                <option v-for="opt in strategyOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </form>

          <div class="dlg-footer">
            <DialogClose as-child>
              <button type="button" class="btn-cancel">取消</button>
            </DialogClose>
            <button type="button" class="btn-save" :disabled="formLoading" @click="handleFormSubmit">
              {{ formLoading ? '处理中...' : (editingGroup ? '保存' : '创建') }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <!-- Delete Dialog -->
    <DialogRoot :open="!!deletingGroup" @update:open="(v) => !v && (deletingGroup = null)">
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
            确定要删除「{{ deletingGroup?.name }}」吗？此操作无法撤销，关联渠道也将解除绑定。
          </DialogDescription>
          <div class="dlg-footer dlg-center">
            <button class="btn-cancel" @click="deletingGroup = null">取消</button>
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
.settings-page {
  max-width: 1200px;
}

/* Page Header */
.page-header {
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--color-text-primary);
  margin: 0 0 0.25rem;
}

.page-desc {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  margin: 0;
}

/* Settings Grid */
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
}

/* Settings Panel */
.settings-panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.panel-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);

  &.groups {
    background: rgba(16, 185, 129, 0.1);
    color: var(--color-primary);
  }

  &.security {
    background: rgba(245, 158, 11, 0.1);
    color: var(--color-warning);
  }
}

.settings-panel:hover .panel-icon {
  transform: scale(1.1) rotate(5deg);
}

.panel-info {
  flex: 1;
}

.panel-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 0.125rem;
}

.panel-desc {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  margin: 0;
}

.btn-add {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.875rem;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  box-shadow: 0 2px 8px rgba(var(--color-primary-rgb, 99, 102, 241), 0.3);

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(var(--color-primary-rgb, 99, 102, 241), 0.4);
  }

  &:active {
    transform: translateY(0);
  }
}

.panel-body {
  padding: 1.25rem 1.5rem;
}

/* Loading State */
.loading-state {
  display: flex;
  justify-content: center;
  padding: 2rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s cubic-bezier(0.4, 0, 0.2, 1) infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 2rem 1rem;
}

.empty-visual {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  background: var(--color-bg-tertiary);
  border-radius: 12px;
  color: var(--color-text-muted);
  margin-bottom: 1rem;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.empty-text {
  font-size: 0.9375rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin: 0 0 0.25rem;
}

.empty-hint {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  margin: 0;
}

/* Group List */
.group-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.group-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.875rem 1rem;
  background: var(--color-bg-secondary);
  border-radius: 8px;
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  border: 1px solid transparent;

  &:hover {
    background: var(--color-bg-tertiary);
    transform: translateX(4px);
    border-color: var(--color-border);
  }
}

.group-main {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: 500;
  color: var(--color-text-primary);
  margin-bottom: 0.125rem;
}

.group-remark {
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--color-text-muted);
  margin-left: 0.5rem;
}

.group-actions {
  display: flex;
  gap: 0.25rem;
  margin-left: 1rem;
}

.icon-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    background: var(--color-bg-card);
    color: var(--color-text-secondary);
  }

  &.danger:hover {
    background: var(--color-danger-light);
    color: var(--color-danger);
  }
}

/* Password Form */
.password-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.alert {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-size: 0.8125rem;
  animation: alert-slide-in 0.3s ease-out;

  &.success {
    background: var(--color-success-light);
    color: var(--color-success);
  }

  &.error {
    background: var(--color-danger-light);
    color: var(--color-danger);
  }
}

@keyframes alert-slide-in {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;

  label {
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  input {
    padding: 0.625rem 0.875rem;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
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
}

.btn-submit {
  padding: 0.625rem 1.25rem;
  background: var(--color-primary);
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  color: white;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  align-self: flex-start;
  box-shadow: 0 2px 8px rgba(var(--color-primary-rgb, 99, 102, 241), 0.3);

  &:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(var(--color-primary-rgb, 99, 102, 241), 0.4);
  }

  &:active:not(:disabled) {
    transform: translateY(0);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

/* Responsive */
@media (max-width: 768px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .panel-header {
    flex-wrap: wrap;
  }

  .btn-add {
    width: 100%;
    justify-content: center;
    margin-top: 0.75rem;
  }
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
  max-width: 420px;
  height: fit-content;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--color-border);
  z-index: 50;
  overflow: hidden;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.2), 0 8px 16px rgba(0, 0, 0, 0.1);
}

.dlg-sm {
  max-width: 380px;
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
  border-radius: 8px;
  font-size: 0.8125rem;
  margin-bottom: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-bottom: 1rem;

  &:last-child {
    margin-bottom: 0;
  }

  label {
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .req {
    color: var(--color-danger);
  }

  input,
  select {
    padding: 0.5rem 0.75rem;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    font-size: 0.875rem;
    color: var(--color-text-primary);

    &:focus {
      outline: none;
      border-color: var(--color-primary);
    }
  }

  select {
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 14px;
    padding-right: 2rem;
  }

  .hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    margin: 0;
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
  border-radius: 8px;
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
  border-radius: 8px;
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
  border-radius: 8px;
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
