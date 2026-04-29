<script setup lang="ts">
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from 'reka-ui';

interface Props {
  open: boolean;
  title?: string;
  description?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showClose?: boolean;
}

withDefaults(defineProps<Props>(), {
  size: 'md',
  showClose: true,
});

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void;
}>();
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-[300] bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=open]:fade-in data-[state=closed]:animate-out data-[state=closed]:fade-out"
      />
      <DialogContent
        class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-[400] bg-bg-card border border-border rounded-xl shadow-lg backdrop-blur-xl data-[state=open]:animate-in data-[state=open]:zoom-in-95 data-[state=open]:fade-in data-[state=closed]:animate-out data-[state=closed]:zoom-out-95 data-[state=closed]:fade-out"
        :class="[
          size === 'sm' && 'w-full max-w-sm',
          size === 'md' && 'w-full max-w-md',
          size === 'lg' && 'w-full max-w-lg',
          size === 'xl' && 'w-full max-w-xl',
        ]"
      >
        <div class="flex items-center justify-between px-5 py-4 border-b border-border">
          <div class="flex flex-col gap-0.5">
            <DialogTitle class="text-base font-semibold text-text-primary">{{ title }}</DialogTitle>
            <DialogDescription v-if="description" class="text-xs text-text-muted">
              {{ description }}
            </DialogDescription>
          </div>
          <DialogClose v-if="showClose" class="inline-flex items-center justify-center w-7 h-7 rounded-md text-text-muted hover:text-text-secondary hover:bg-bg-tertiary transition-colors">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </DialogClose>
        </div>
        <div class="px-5 py-4">
          <slot />
        </div>
        <div v-if="$slots.footer" class="flex items-center justify-end gap-2 px-5 py-3 border-t border-border">
          <slot name="footer" />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
