<script setup lang="ts">
import {
  SelectContent,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport,
} from 'reka-ui'

interface Option {
  label: string
  value: string
}

interface Props {
  modelValue?: string
  options: Option[]
  placeholder?: string
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
}

withDefaults(defineProps<Props>(), {
  size: 'md',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()
</script>

<template>
  <SelectRoot
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  >
    <SelectTrigger
      class="w-full bg-bg-secondary text-text-primary border border-border rounded-md transition-all duration-200 focus:outline-none focus:border-border-focus focus:ring-1 focus:ring-primary/30 disabled:opacity-50 disabled:cursor-not-allowed hover:border-border-hover text-left flex items-center justify-between"
      :class="[
        size === 'sm' && 'px-2.5 py-1.5 text-xs',
        size === 'md' && 'px-3 py-2 text-sm',
        size === 'lg' && 'px-4 py-2.5 text-base',
      ]"
    >
      <SelectValue :placeholder="placeholder" />
      <span class="text-text-muted ml-1.5">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </span>
    </SelectTrigger>

    <SelectPortal>
      <SelectContent
        position="popper"
        :side-offset="4"
        class="bg-bg-card border border-border rounded-md shadow-[0_8px_24px_rgba(0,0,0,0.2)] z-[100] min-w-[var(--reka-select-trigger-width)]"
      >
        <SelectViewport class="p-1 max-h-[260px] overflow-y-auto">
          <SelectItem
            v-for="opt in options"
            :key="opt.value"
            :value="opt.value"
            class="relative flex items-center gap-2 px-2.5 py-1.5 text-xs text-text-primary rounded-sm cursor-pointer outline-none transition-colors duration-150 hover:bg-primary-light hover:text-primary data-[state=checked]:bg-primary-light data-[state=checked]:text-primary select-none"
          >
            <SelectItemText>{{ opt.label }}</SelectItemText>
            <SelectItemIndicator class="ml-auto">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
