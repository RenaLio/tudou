import { defineStore, } from 'pinia';
import { ref, watch, } from 'vue';

export type Theme = 'light' | 'dark';

export const useThemeStore = defineStore(
  'theme',
  () => {
    const theme = ref<Theme>('dark',);

    function setTheme(newTheme: Theme,) {
      theme.value = newTheme;
    }

    function toggleTheme() {
      theme.value = theme.value === 'dark' ? 'light' : 'dark';
    }

    // Watch and apply theme to document
    watch(
      theme,
      val => {
        document.documentElement.setAttribute('data-theme', val,);
      },
      { immediate: true, },
    );

    return {
      theme,
      setTheme,
      toggleTheme,
    };
  },
  {
    persist: {
      key: 'theme',
      storage: localStorage,
      pick: ['theme',],
    },
  },
);
