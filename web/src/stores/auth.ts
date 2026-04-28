import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { User } from '@/types'

export const useAuthStore = defineStore(
  'auth',
  () => {
    const token = ref<string | null>(null)
    const user = ref<User | null>(null)

    const isAuthenticated = computed(() => !!token.value)

    function setToken(newToken: string | null) {
      token.value = newToken
    }

    function setUser(newUser: User | null) {
      user.value = newUser
    }

    function logout() {
      token.value = null
      user.value = null
    }

    return {
      token,
      user,
      isAuthenticated,
      setToken,
      setUser,
      logout,
    }
  },
  {
    persist: {
      key: 'auth',
      storage: localStorage,
      pick: ['token', 'user'],
    },
  }
)
