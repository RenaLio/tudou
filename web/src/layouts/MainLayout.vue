<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const menuItems = [
  { path: '/dashboard', title: '统计看板', icon: 'dashboard' },
  { path: '/tokens', title: '令牌管理', icon: 'token' },
  { path: '/channels', title: '渠道管理', icon: 'channel' },
  { path: '/models', title: '模型管理', icon: 'model' },
  { path: '/settings', title: '系统配置', icon: 'settings' },
]

const currentPageTitle = computed(() =>
  menuItems.find(item => item.path === route.path)?.title || 'Tudou'
)

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="app-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <!-- Logo -->
      <RouterLink to="/" class="logo-link">
        <div class="logo-mark">
          <span>T</span>
        </div>
        <div class="logo-text">
          <span class="logo-name">Tudou</span>
          <span class="logo-tag">Gateway</span>
        </div>
      </RouterLink>

      <!-- Navigation -->
      <nav class="nav-menu">
        <RouterLink
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <span class="nav-icon">
            <!-- Dashboard -->
            <svg v-if="item.icon === 'dashboard'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
            <!-- Token -->
            <svg v-else-if="item.icon === 'token'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M12 2L2 7l10 5 10-5-10-5z"/>
              <path d="M2 17l10 5 10-5"/>
              <path d="M2 12l10 5 10-5"/>
            </svg>
            <!-- Channel -->
            <svg v-else-if="item.icon === 'channel'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
            </svg>
            <!-- Model -->
            <svg v-else-if="item.icon === 'model'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
            <!-- Settings -->
            <svg v-else-if="item.icon === 'settings'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.67 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.67 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
          </span>
          <span class="nav-label">{{ item.title }}</span>
        </RouterLink>
      </nav>

      <!-- Bottom Actions -->
      <div class="sidebar-footer">
        <button class="footer-btn" @click="themeStore.toggleTheme">
          <span class="btn-icon">
            <svg v-if="themeStore.theme === 'dark'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="4"/>
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
            </svg>
          </span>
          <span class="btn-label">{{ themeStore.theme === 'dark' ? '浅色' : '深色' }}</span>
        </button>

        <button class="footer-btn logout" @click="handleLogout">
          <span class="btn-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
          </span>
          <span class="btn-label">退出</span>
        </button>
      </div>
    </aside>

    <!-- Main Area -->
    <div class="main-area">
      <!-- Top Bar -->
      <header class="topbar">
        <div class="topbar-left">
          <h1 class="page-title">{{ currentPageTitle }}</h1>
        </div>
        <div class="topbar-right">
          <div class="status-indicator">
            <span class="status-pulse" />
            <span class="status-text">运行中</span>
          </div>
        </div>
      </header>

      <!-- Content -->
      <main class="content-area">
        <RouterView v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
  background: var(--color-bg-primary);
}

/* ── Sidebar ── */
.sidebar {
  width: 220px;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 100;
}

/* Logo */
.logo-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.25rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
  text-decoration: none;
}

.logo-mark {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--color-primary-dark) 0%, var(--color-primary) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-bg-primary);
  font-weight: 700;
  font-size: 1rem;
  font-family: var(--font-display);
  box-shadow: 0 2px 8px rgba(139, 195, 74, 0.2);
}

.logo-text {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.logo-name {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  font-family: var(--font-display);
  line-height: 1.2;
}

.logo-tag {
  font-size: 0.625rem;
  color: var(--color-text-muted);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

/* Navigation */
.nav-menu {
  flex: 1;
  padding: 0.875rem 0.625rem;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 0.875rem;
  margin-bottom: 0.25rem;
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text-tertiary);
  font-size: 0.8125rem;
  font-weight: 500;
  transition: all 0.2s ease;
  position: relative;
}

.nav-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 2px;
  height: 0;
  background: var(--color-primary);
  border-radius: 0 2px 2px 0;
  transition: height 0.2s ease;
}

.nav-item:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
}

.nav-item.active {
  background: var(--color-primary-light);
  color: var(--color-primary);
}

.nav-item.active::before {
  height: 50%;
}

.nav-icon {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.nav-icon svg {
  width: 16px;
  height: 16px;
}

.nav-label {
  flex: 1;
}

/* Sidebar Footer */
.sidebar-footer {
  padding: 0.625rem;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 0.25rem;
}

.footer-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  flex: 1;
  padding: 0.5rem 0.625rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-tertiary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.footer-btn:hover {
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
}

.footer-btn.logout:hover {
  background: var(--color-danger-light);
  color: var(--color-danger);
}

.btn-icon {
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-icon svg {
  width: 14px;
  height: 14px;
}

/* ── Main Area ── */
.main-area {
  flex: 1;
  margin-left: 220px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* Top Bar */
.topbar {
  height: 52px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  position: sticky;
  top: 0;
  z-index: 50;
}

.page-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  font-family: var(--font-display);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success);
  animation: statusPulse 2s ease-in-out infinite;
}

@keyframes statusPulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(129, 199, 132, 0.4); }
  50% { opacity: 0.7; box-shadow: 0 0 0 4px rgba(129, 199, 132, 0); }
}

.status-text {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

/* Content Area */
.content-area {
  flex: 1;
  padding: 1.5rem;
  overflow-y: auto;
}

/* Page transition */
.page-fade-enter-active {
  transition: opacity 0.2s ease-out, transform 0.2s ease-out;
}

.page-fade-leave-active {
  transition: opacity 0.15s ease-in, transform 0.15s ease-in;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* Responsive */
@media (max-width: 768px) {
  .sidebar {
    width: 60px;
  }

  .logo-text,
  .nav-label,
  .btn-label {
    display: none;
  }

  .logo-link {
    justify-content: center;
    padding: 1rem;
  }

  .nav-item {
    justify-content: center;
    padding: 0.75rem;
  }

  .footer-btn {
    padding: 0.625rem;
  }

  .main-area {
    margin-left: 60px;
  }
}
</style>
