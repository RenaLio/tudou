<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'warning';
  size?: 'sm' | 'md' | 'lg';
  disabled?: boolean;
  loading?: boolean;
  type?: 'button' | 'submit' | 'reset';
}

withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
});
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    class="inline-flex items-center justify-center gap-1.5 font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-0 focus:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed rounded-md"
    :class="[
      // Size
      size === 'sm' && 'px-2.5 py-1.5 text-xs',
      size === 'md' && 'px-4 py-2 text-sm',
      size === 'lg' && 'px-5 py-2.5 text-base',
      // Variant
      variant === 'primary' && 'bg-primary text-[#080a08] hover:bg-primary-dark hover:shadow-glow-primary active:scale-[0.98]',
      variant === 'secondary' && 'bg-bg-tertiary text-text-secondary border border-border hover:border-border-hover hover:text-text-primary active:scale-[0.98]',
      variant === 'ghost' && 'bg-transparent text-text-tertiary hover:bg-bg-tertiary hover:text-text-secondary active:scale-[0.98]',
      variant === 'danger' && 'bg-danger/10 text-danger border border-danger/20 hover:bg-danger/20 hover:shadow-[0_0_16px_rgba(229,115,115,0.15)] active:scale-[0.98]',
      variant === 'success' && 'bg-success/10 text-success border border-success/20 hover:bg-success/20 active:scale-[0.98]',
      variant === 'warning' && 'bg-warning/10 text-warning border border-warning/20 hover:bg-warning/20 active:scale-[0.98]',
    ]"
  >
    <span v-if="loading" class="inline-block w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
    <slot />
  </button>
</template>
