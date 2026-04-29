<script setup lang="ts">
interface Props {
  label?: string;
  required?: boolean;
  error?: string;
  hint?: string;
}

defineProps<Props>();
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <label v-if="label" class="text-sm font-medium text-text-secondary flex items-center gap-1">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <slot />
    <p v-if="hint && !error" class="text-xs text-text-muted">{{ hint }}</p>
    <p v-if="error" class="text-xs text-danger">{{ error }}</p>
  </div>
</template>
