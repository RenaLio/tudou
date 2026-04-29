<script setup lang="ts">
interface Props {
  modelValue?: string | number;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  readonly?: boolean;
  error?: string;
  size?: 'sm' | 'md' | 'lg';
}

withDefaults(defineProps<Props>(), {
  type: 'text',
  size: 'md',
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value);
}
</script>

<template>
  <div class="w-full">
    <input
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      class="w-full bg-bg-secondary text-text-primary placeholder:text-text-muted border rounded-md transition-all duration-200 focus:outline-none focus:border-border-focus focus:ring-1 focus:ring-primary/30 disabled:opacity-50 disabled:cursor-not-allowed"
      :class="[
        size === 'sm' && 'px-2.5 py-1.5 text-xs',
        size === 'md' && 'px-3 py-2 text-sm',
        size === 'lg' && 'px-4 py-2.5 text-base',
        error ? 'border-danger/50 focus:border-danger focus:ring-danger/20' : 'border-border hover:border-border-hover',
      ]"
      @input="onInput"
    />
    <p v-if="error" class="mt-1 text-xs text-danger">{{ error }}</p>
  </div>
</template>
