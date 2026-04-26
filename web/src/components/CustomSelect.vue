<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'

interface Option {
  value: string | number
  label: string
  icon?: string
  color?: string
}

const props = withDefaults(defineProps<{
  modelValue: string | number | ''
  options: Option[]
  placeholder?: string
  disabled?: boolean
  size?: 'sm' | 'md'
}>(), {
  placeholder: '请选择',
  size: 'md',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
  (e: 'change', value: string | number): void
}>()

const isOpen = ref(false)
const selectRef = ref<HTMLElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const highlightedIndex = ref(-1)
const dropdownStyle = ref<Record<string, string>>({})

const selectedOption = computed(() => {
  if (props.modelValue === '' || props.modelValue === undefined) return null
  return props.options.find(opt => opt.value === props.modelValue)
})

const displayLabel = computed(() => {
  return selectedOption.value?.label || props.placeholder
})

function updateDropdownPosition() {
  if (!selectRef.value || !dropdownRef.value) return

  const triggerRect = selectRef.value.getBoundingClientRect()
  const dropdownHeight = dropdownRef.value.offsetHeight || 200
  const viewportHeight = window.innerHeight

  let top = triggerRect.bottom + 4
  let left = triggerRect.left
  let width = triggerRect.width

  // 如果下方空间不足，显示在上方
  if (top + dropdownHeight > viewportHeight - 20) {
    top = triggerRect.top - dropdownHeight - 4
    if (top < 0) {
      top = 20
    }
  }

  // 防止超出右边
  if (left + 300 > window.innerWidth) {
    left = window.innerWidth - 300 - 20
  }

  dropdownStyle.value = {
    position: 'fixed',
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`,
    minWidth: `${Math.max(width, 120)}px`,
    zIndex: '9999',
  }
}

function toggleDropdown() {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    highlightedIndex.value = props.options.findIndex(opt => opt.value === props.modelValue)
    nextTick(() => {
      updateDropdownPosition()
    })
  }
}

function selectOption(option: Option, e: Event) {
  e.preventDefault()
  e.stopPropagation()
  emit('update:modelValue', option.value)
  emit('change', option.value)
  isOpen.value = false
}

function clearSelection(e: Event) {
  e.stopPropagation()
  emit('update:modelValue', '' as '')
  emit('change', '' as '')
  isOpen.value = false
}

function handleKeydown(e: KeyboardEvent) {
  if (!isOpen.value) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      isOpen.value = true
    }
    return
  }

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      highlightedIndex.value = Math.min(highlightedIndex.value + 1, props.options.length - 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
      break
    case 'Enter':
      e.preventDefault()
      if (highlightedIndex.value >= 0) {
        const opt = props.options[highlightedIndex.value]
        emit('update:modelValue', opt.value)
        emit('change', opt.value)
        isOpen.value = false
      }
      break
    case 'Escape':
      isOpen.value = false
      break
  }
}

function handleClickOutside(e: MouseEvent) {
  if (!isOpen.value) return
  const target = e.target as Node

  // 检查是否点击了选择器触发器
  if (selectRef.value && selectRef.value.contains(target)) {
    return
  }

  // 检查是否点击了下拉菜单
  if (dropdownRef.value && dropdownRef.value.contains(target)) {
    return
  }

  // 点击了外部，关闭下拉菜单
  isOpen.value = false
}

function handleScroll() {
  if (isOpen.value) {
    updateDropdownPosition()
  }
}

watch(isOpen, (open) => {
  if (open) {
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
      window.addEventListener('scroll', handleScroll, true)
      window.addEventListener('resize', handleScroll)
    }, 0)
  } else {
    document.removeEventListener('click', handleClickOutside)
    window.removeEventListener('scroll', handleScroll, true)
    window.removeEventListener('resize', handleScroll)
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside, true)
  window.removeEventListener('scroll', handleScroll, true)
  window.removeEventListener('resize', handleScroll)
})
</script>

<template>
  <div
    ref="selectRef"
    class="custom-select"
    :class="[{ open: isOpen, disabled, [`size-${size}`]: true }]"
    tabindex="0"
    @keydown="handleKeydown"
  >
    <button
      type="button"
      class="select-trigger"
      :disabled="disabled"
      @click="toggleDropdown"
    >
      <span v-if="selectedOption?.color" class="color-dot" :style="{ background: selectedOption.color }"></span>
      <span class="select-label" :class="{ placeholder: !selectedOption }">
        {{ displayLabel }}
      </span>
      <span class="select-icons">
        <button
          v-if="selectedOption && modelValue !== ''"
          type="button"
          class="clear-btn"
          @click="clearSelection"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
        <svg class="chevron" :class="{ rotated: isOpen }" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </span>
    </button>

    <Teleport to="body">
      <Transition name="dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="select-dropdown"
          :style="dropdownStyle"
          @mousedown.stop
          @click.stop
        >
          <div class="dropdown-inner">
            <button
              v-for="(option, index) in options"
              :key="option.value"
              type="button"
              class="dropdown-option"
              :class="{
                selected: option.value === modelValue,
                highlighted: index === highlightedIndex,
              }"
              @click="selectOption(option, $event)"
              @mouseenter="highlightedIndex = index"
            >
              <span v-if="option.color" class="color-dot" :style="{ background: option.color }"></span>
              <span class="option-label">{{ option.label }}</span>
              <svg v-if="option.value === modelValue" class="check-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </button>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.custom-select {
  position: relative;
  width: 100%;
  min-width: 120px;
  outline: none;
}

.select-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-primary);
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.custom-select:focus-within .select-trigger {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb, 99, 102, 241), 0.1);
}

.custom-select.open .select-trigger {
  border-color: var(--color-primary);
}

.custom-select.disabled .select-trigger {
  opacity: 0.5;
  cursor: not-allowed;
}

.custom-select.size-sm .select-trigger {
  padding: 0.375rem 0.625rem;
  font-size: 0.8125rem;
}

.select-trigger:hover:not(:disabled) {
  border-color: var(--color-primary);
}

.select-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-label.placeholder {
  color: var(--color-text-muted);
}

.color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.select-icons {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-left: auto;
}

.clear-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  background: var(--color-bg-tertiary);
  border: none;
  border-radius: 4px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.15s;
}

.clear-btn:hover {
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
}

.chevron {
  transition: transform 0.2s ease;
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.chevron.rotated {
  transform: rotate(180deg);
}
</style>

<!-- Global styles for dropdown (rendered in body via Teleport) -->
<style>
.select-dropdown {
  min-width: 120px;
}

.select-dropdown .dropdown-inner {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15), 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 0.25rem;
  max-height: 280px;
  overflow-y: auto;
}

.select-dropdown .dropdown-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.625rem;
  background: transparent;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.select-dropdown .dropdown-option:hover,
.select-dropdown .dropdown-option.highlighted {
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}

.select-dropdown .dropdown-option.selected {
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.select-dropdown .option-label {
  flex: 1;
}

.select-dropdown .color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.select-dropdown .check-icon {
  color: var(--color-primary);
  flex-shrink: 0;
}

/* Transition */
.dropdown-enter-active {
  animation: dropdown-in 0.2s ease-out;
}

.dropdown-leave-active {
  animation: dropdown-in 0.15s ease-in reverse;
}

@keyframes dropdown-in {
  from {
    opacity: 0;
    transform: translateY(-8px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* Scrollbar */
.select-dropdown .dropdown-inner::-webkit-scrollbar {
  width: 6px;
}

.select-dropdown .dropdown-inner::-webkit-scrollbar-track {
  background: transparent;
}

.select-dropdown .dropdown-inner::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

.select-dropdown .dropdown-inner::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-muted);
}
</style>
