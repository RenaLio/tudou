<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'secondary';
  size?: 'sm' | 'md';
  pulse?: boolean;
}

withDefaults(defineProps<Props>(), {
  variant: 'secondary',
  size: 'md',
});
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 font-medium rounded-full"
    :class="[
      size === 'sm' && 'px-2 py-0.5 text-[10px]',
      size === 'md' && 'px-2.5 py-1 text-xs',
      variant === 'primary' && 'bg-primary-light text-primary',
      variant === 'success' && 'bg-success-light text-success',
      variant === 'warning' && 'bg-warning-light text-warning',
      variant === 'danger' && 'bg-danger-light text-danger',
      variant === 'secondary' && 'bg-bg-tertiary text-text-tertiary border border-border',
    ]"
  >
    <span
      v-if="pulse"
      class="w-1.5 h-1.5 rounded-full"
      :class="[
        variant === 'primary' && 'bg-primary',
        variant === 'success' && 'bg-success',
        variant === 'warning' && 'bg-warning',
        variant === 'danger' && 'bg-danger',
        variant === 'secondary' && 'bg-text-muted',
      ]"
    />
    <slot />
  </span>
</template>
