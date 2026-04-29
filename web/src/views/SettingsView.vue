<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { updatePassword } from '@/api/auth'
import {
  listChannelGroups,
  createChannelGroup,
  updateChannelGroup,
  deleteChannelGroup,
  listLoadBalanceStrategies,
  type CreateChannelGroupRequest,
  type LoadBalanceStrategyItem,
} from '@/api/channel-group'
import type { ChannelGroup } from '@/types'
import { fadeUp, scaleIn } from '@/utils/motion'
import AppButton from '@/components/ui/AppButton.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppFormField from '@/components/ui/AppFormField.vue'

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
const strategies = ref<LoadBalanceStrategyItem[]>([])

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
  <div v-motion="fadeUp" class="max-w-[1400px]">
    <!-- Page Header -->
    <header class="mb-8">
      <h1 class="text-2xl font-semibold tracking-tight text-text-primary mb-1">系统配置</h1>
      <p class="text-sm text-text-muted">管理渠道分组和账户安全设置</p>
    </header>

    <!-- Settings Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Channel Groups Panel -->
      <section
        v-motion
        :initial="{ opacity: 0, y: 30, scale: 0.98 }"
        :enter="{ opacity: 1, y: 0, scale: 1, transition: { delay: 100, type: 'spring', stiffness: 200, damping: 25 } }"
        class="bg-bg-card border border-border rounded-xl overflow-hidden"
      >
        <div class="flex items-start gap-4 px-6 py-5 border-b border-border bg-bg-secondary">
          <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 bg-success/10 text-success transition-transform duration-300 group-hover:scale-110 group-hover:rotate-[5deg]"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
          </div>
          <div class="flex-1 min-w-0">
            <h2 class="text-base font-semibold text-text-primary mb-0.5">渠道分组</h2>
            <p class="text-[13px] text-text-muted">用于令牌关联和负载均衡策略配置</p>
          </div>
          <AppButton size="sm" @click="openCreateDialog">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="inline-block mr-1">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            新建
          </AppButton>
        </div>

        <div class="px-6 py-5">
          <!-- Loading -->
          <div v-if="groupsLoading" class="flex justify-center py-8">
            <div class="w-8 h-8 border-[3px] border-border border-t-primary rounded-full animate-spin" />
          </div>

          <!-- Empty -->
          <div v-else-if="groups.length === 0" class="text-center py-8 px-4">
            <div class="inline-flex items-center justify-center w-14 h-14 bg-bg-tertiary rounded-xl text-text-muted mb-4 animate-bounce"
            >
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect x="3" y="3" width="7" height="7" rx="1"/>
                <rect x="14" y="3" width="7" height="7" rx="1"/>
                <rect x="3" y="14" width="7" height="7" rx="1"/>
                <line x1="17" y1="17" x2="17" y2="21"/>
                <line x1="15" y1="19" x2="19" y2="19"/>
              </svg>
            </div>
            <p class="text-[15px] font-medium text-text-secondary mb-1">暂无分组</p>
            <p class="text-[13px] text-text-muted">创建分组后可在令牌中关联使用</p>
          </div>

          <!-- Group List -->
          <div v-else class="flex flex-col gap-2">
            <div
              v-for="(group, index) in groups"
              :key="group.id"
              v-motion="scaleIn"
              :style="{ transitionDelay: `${index * 60}ms` }"
              class="flex items-center justify-between px-4 py-3.5 bg-bg-secondary rounded-lg border border-transparent transition-all duration-200 hover:bg-bg-tertiary hover:translate-x-1 hover:border-border"
            >
              <div class="flex-1 min-w-0">
                <div class="font-medium text-text-primary">
                  {{ group.name }}
                  <span v-if="group.nameRemark" class="text-[13px] font-normal text-text-muted ml-2">{{ group.nameRemark }}</span>
                </div>
              </div>
              <div class="flex gap-1 ml-4">
                <AppButton variant="ghost" size="sm" @click="openEditDialog(group)">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </AppButton>
                <AppButton variant="ghost" size="sm" class="hover:bg-danger-light hover:text-danger" @click="deletingGroup = group">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </AppButton>
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
        class="bg-bg-card border border-border rounded-xl overflow-hidden"
      >
        <div class="flex items-start gap-4 px-6 py-5 border-b border-border bg-bg-secondary">
          <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 bg-warning/10 text-warning"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div class="flex-1 min-w-0">
            <h2 class="text-base font-semibold text-text-primary mb-0.5">账户安全</h2>
            <p class="text-[13px] text-text-muted">修改登录密码和访问凭证</p>
          </div>
        </div>

        <div class="px-6 py-5">
          <form @submit.prevent="handlePasswordSubmit" class="flex flex-col gap-4">
            <!-- Success -->
            <div v-if="passwordSuccess" class="flex items-center gap-2 px-4 py-3 rounded-lg text-sm bg-success-light text-success"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
              </svg>
              密码已更新
            </div>

            <!-- Error -->
            <div v-if="passwordError" class="flex items-center gap-2 px-4 py-3 rounded-lg text-sm bg-danger-light text-danger"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              {{ passwordError }}
            </div>

            <AppFormField label="新密码" required>
              <AppInput
                v-model="newPassword"
                type="password"
                placeholder="至少 6 位字符"
              />
            </AppFormField>

            <AppFormField label="确认密码" required>
              <AppInput
                v-model="confirmPassword"
                type="password"
                placeholder="再次输入新密码"
              />
            </AppFormField>

            <AppButton type="submit" :loading="passwordLoading" class="self-start">
              更新密码
            </AppButton>
          </form>
        </div>
      </section>
    </div>

    <!-- Create/Edit Dialog -->
    <AppDialog
      v-model:open="dialogOpen"
      :title="editingGroup ? '编辑分组' : '新建分组'"
      :description="editingGroup ? '编辑渠道分组信息' : '创建新的渠道分组'"
      size="md"
    >
      <form @submit.prevent="handleFormSubmit" class="flex flex-col gap-4">
        <div v-if="formError" class="px-3.5 py-2.5 rounded-lg text-sm bg-danger-light text-danger">{{ formError }}</div>

        <AppFormField label="分组名称" required hint="分组名称必须唯一">
          <AppInput v-model="formData.name" placeholder="请输入分组名称" />
        </AppFormField>

        <AppFormField label="备注名">
          <AppInput v-model="formData.nameRemark" placeholder="可选" />
        </AppFormField>

        <AppFormField label="负载均衡策略">
          <select
            v-model="formData.loadBalanceStrategy"
            class="w-full px-3 py-2 bg-bg-secondary text-text-primary border border-border rounded-md text-sm focus:outline-none focus:border-border-focus focus:ring-1 focus:ring-primary/30 appearance-none cursor-pointer"
            style="background-image: url(&quot;data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E&quot;); background-repeat: no-repeat; background-position: right 0.75rem center; background-size: 14px; padding-right: 2rem;"
          >
            <option v-for="opt in strategyOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </AppFormField>
      </form>

      <template #footer>
        <AppButton variant="secondary" @click="dialogOpen = false">取消</AppButton>
        <AppButton :loading="formLoading" @click="handleFormSubmit">
          {{ editingGroup ? '保存' : '创建' }}
        </AppButton>
      </template>
    </AppDialog>

    <!-- Delete Dialog -->
    <AppDialog
      :open="!!deletingGroup"
      @update:open="(v: boolean) => !v && (deletingGroup = null)"
      title="确认删除"
      size="sm"
    >
      <p class="text-sm text-text-secondary text-center">
        确定要删除「{{ deletingGroup?.name }}」吗？此操作无法撤销，关联渠道也将解除绑定。
      </p>

      <template #footer>
        <AppButton variant="secondary" @click="deletingGroup = null">取消</AppButton>
        <AppButton variant="danger" :loading="deleteLoading" @click="handleDelete">删除</AppButton>
      </template>
    </AppDialog>
  </div>
</template>
