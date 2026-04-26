import {
  defineConfig,
  presetUno,
  presetIcons,
  transformerDirectives,
} from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
    presetIcons({
      scale: 1.2,
      cdn: 'https://esm.sh/',
    }),
  ],
  transformers: [
    transformerDirectives(),
  ],
  shortcuts: {
    // Buttons
    'btn': 'px-4 py-2 rounded-lg inline-flex items-center justify-center gap-2 font-medium transition-all duration-200 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed',
    'btn-primary': 'btn text-white hover:opacity-90 active:opacity-80',
    'btn-secondary': 'btn border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)]',
    'btn-ghost': 'btn text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)]',
    'btn-danger': 'btn text-white hover:opacity-90',

    // Inputs
    'input-base': 'w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-card)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] focus:border-transparent transition-all',

    // Cards
    'card': 'bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)] shadow-sm',
    'card-header': 'px-6 py-4 border-b border-[var(--color-border)]',
    'card-body': 'p-6',
  },
  theme: {
    colors: {
      primary: {
        DEFAULT: 'var(--color-primary)',
        light: 'var(--color-primary-light)',
        dark: 'var(--color-primary-dark)',
      },
    },
  },
  rules: [
    ['bg-base', { 'background-color': 'var(--color-bg-primary)' }],
    ['bg-base-secondary', { 'background-color': 'var(--color-bg-secondary)' }],
    ['bg-card', { 'background-color': 'var(--color-bg-card)' }],
    ['text-base', { color: 'var(--color-text-primary)' }],
    ['text-secondary', { color: 'var(--color-text-secondary)' }],
    ['text-muted', { color: 'var(--color-text-muted)' }],
    ['border-base', { 'border-color': 'var(--color-border)' }],
    ['ring-primary', { '--tw-ring-color': 'var(--color-primary)' }],
  ],
})
