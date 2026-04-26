import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useAuthStore = defineStore(
  'auth',
  () => {
    const token = ref<string | null>(null)

    const isAuthenticated = computed(() => !!token.value)

    function setToken(newToken: string | null) {
      token.value = newToken
    }

    function logout() {
      token.value = null
    }

    return {
      token,
      isAuthenticated,
      setToken,
      logout,
    }
  },
  {
    persist: {
      key: 'auth',
      storage: localStorage,
      pick: ['token'],
    },
  }
)
