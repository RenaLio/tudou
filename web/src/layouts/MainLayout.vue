<script setup lang="ts">
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import AppButton from '@/components/ui/AppButton.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const menuItems = [
  { path: '/dashboard', title: '统计看板', icon: 'dashboard' },
  { path: '/request-logs', title: '请求日志', icon: 'request-log' },
  { path: '/tokens', title: '令牌管理', icon: 'token' },
  { path: '/channels', title: '渠道管理', icon: 'channel' },
  { path: '/models', title: '模型管理', icon: 'model' },
  { path: '/settings', title: '系统配置', icon: 'settings' },
]

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex min-h-screen bg-bg-primary">
    <!-- Sidebar -->
    <aside class="w-[220px] bg-bg-secondary border-r border-border flex flex-col fixed top-0 left-0 bottom-0 z-[100] max-md:w-[60px]">
      <!-- Logo -->
      <RouterLink to="/" class="flex items-center gap-3 px-5 py-5 border-b border-border no-underline max-md:justify-center max-md:p-4">
        <div class="w-[34px] h-[34px] rounded-[10px] bg-gradient-to-br from-primary-dark to-primary flex items-center justify-center text-bg-primary font-bold text-base font-display shadow-[0_2px_8px_rgba(139,195,74,0.2)]">
          <span>T</span>
        </div>
        <div class="flex flex-col gap-0.5 max-md:hidden">
          <span class="text-base font-semibold text-text-primary font-display leading-tight">Tudou</span>
          <span class="text-[0.625rem] text-text-muted tracking-[0.08em] uppercase">Gateway</span>
        </div>
      </RouterLink>

      <!-- Navigation -->
      <nav class="flex-1 py-3.5 px-2.5 overflow-y-auto">
        <RouterLink
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-2.5 px-3.5 py-2.5 mb-1 rounded-md no-underline text-text-tertiary text-[0.8125rem] font-medium transition-all duration-200 ease-out relative max-md:justify-center max-md:p-3"
          :class="route.path === item.path ? 'bg-primary-light text-primary before:h-1/2' : 'hover:bg-bg-tertiary hover:text-text-secondary before:h-0'"
        >
          <span class="w-[18px] h-[18px] flex items-center justify-center shrink-0">
            <!-- Dashboard -->
            <svg v-if="item.icon === 'dashboard'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
            <!-- Request Logs -->
            <svg v-else-if="item.icon === 'request-log'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
              <line x1="8" y1="13" x2="16" y2="13"/>
              <line x1="8" y1="17" x2="13" y2="17"/>
            </svg>
            <!-- Token -->
            <svg v-else-if="item.icon === 'token'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 2L2 7l10 5 10-5-10-5z"/>
              <path d="M2 17l10 5 10-5"/>
              <path d="M2 12l10 5 10-5"/>
            </svg>
            <!-- Channel -->
            <svg v-else-if="item.icon === 'channel'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
            </svg>
            <!-- Model -->
            <svg v-else-if="item.icon === 'model'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
            <!-- Settings -->
            <svg v-else-if="item.icon === 'settings'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.67 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.67 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
          </span>
          <span class="flex-1 max-md:hidden">{{ item.title }}</span>
        </RouterLink>
      </nav>

      <!-- Bottom Actions -->
      <div class="p-2.5 border-t border-border flex gap-1 max-md:hidden">
        <AppButton variant="ghost" size="sm" class="flex-1" @click="themeStore.toggleTheme">
          <span class="w-3.5 h-3.5 flex items-center justify-center">
            <svg v-if="themeStore.theme === 'dark'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="4"/>
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
            </svg>
          </span>
          {{ themeStore.theme === 'dark' ? '浅色' : '深色' }}
        </AppButton>

        <AppButton variant="ghost" size="sm" class="flex-1 hover:bg-danger-light hover:text-danger" @click="handleLogout">
          <span class="w-3.5 h-3.5 flex items-center justify-center">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
          </span>
          退出
        </AppButton>
      </div>

      <!-- Mobile Bottom Actions (icon only) -->
      <div class="p-2.5 border-t border-border flex gap-1 md:hidden">
        <button class="flex-1 flex items-center justify-center p-2.5 bg-transparent border-none rounded-md text-text-tertiary transition-all duration-150 hover:bg-bg-tertiary hover:text-text-secondary" @click="themeStore.toggleTheme">
          <span class="w-3.5 h-3.5 flex items-center justify-center">
            <svg v-if="themeStore.theme === 'dark'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="4"/>
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
            </svg>
          </span>
        </button>

        <button class="flex-1 flex items-center justify-center p-2.5 bg-transparent border-none rounded-md text-text-tertiary transition-all duration-150 hover:bg-danger-light hover:text-danger" @click="handleLogout">
          <span class="w-3.5 h-3.5 flex items-center justify-center">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
          </span>
        </button>
      </div>
    </aside>

    <!-- Main Area -->
    <div class="flex-1 ml-[220px] flex flex-col min-h-screen max-md:ml-[60px]">
      <!-- Content -->
      <main class="flex-1 p-6 overflow-y-auto flex flex-col items-center">
        <RouterView v-slot="{ Component }">
          <transition enter-active-class="transition-all duration-200 ease-out" leave-active-class="transition-all duration-150 ease-in" enter-from-class="opacity-0 translate-y-1.5" leave-to-class="opacity-0 -translate-y-1">
            <component :is="Component" />
          </transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>
