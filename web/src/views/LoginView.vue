<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { login } from '@/api/auth'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')
const showForm = ref(false)
const typedTitle = ref('')
const fullTitle = 'Tudou Gateway'
const cursorVisible = ref(true)

onMounted(() => {
  // Typewriter effect
  let index = 0
  const typeInterval = setInterval(() => {
    if (index <= fullTitle.length) {
      typedTitle.value = fullTitle.slice(0, index)
      index++
    } else {
      clearInterval(typeInterval)
      setTimeout(() => {
        showForm.value = true
      }, 300)
    }
  }, 80)

  // Cursor blink
  setInterval(() => {
    cursorVisible.value = !cursorVisible.value
  }, 530)
})

async function handleSubmit() {
  if (!username.value || !password.value) return

  loading.value = true
  errorMessage.value = ''

  try {
    const { accessToken } = await login({
      username: username.value,
      password: password.value,
    })
    authStore.setToken(accessToken)
    router.push('/')
  } catch (error: unknown) {
    const message =
      (error as { response?: { data?: { message?: string } } })?.response?.data?.message
      || '登录失败，请检查用户名和密码'
    errorMessage.value = message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <!-- Animated Grid Background -->
    <div class="grid-bg">
      <div class="grid-lines"></div>
      <div class="glow-orb glow-orb-1"></div>
      <div class="glow-orb glow-orb-2"></div>
    </div>

    <!-- Main Content -->
    <div class="login-content">
      <!-- Logo & Title -->
      <div class="header-section">
        <div class="logo-container">
          <div class="logo-border"></div>
          <div class="logo-inner">
            <span class="logo-text">T</span>
          </div>
        </div>

        <h1 class="title">
          <span class="typed-text">{{ typedTitle }}</span>
          <span class="cursor" :class="{ 'cursor-visible': cursorVisible }">_</span>
        </h1>

        <p class="subtitle">Personal LLM Infrastructure</p>
      </div>

      <!-- Login Form -->
      <Transition name="form-appear">
        <form v-if="showForm" @submit.prevent="handleSubmit" class="login-form">
          <div class="form-header">
            <span class="form-indicator"></span>
            <span class="form-title">secure_access</span>
          </div>

          <!-- Error Message -->
          <div v-if="errorMessage" class="error-message">
            <div class="i-lucide:alert-triangle error-icon"></div>
            <span>{{ errorMessage }}</span>
          </div>

          <!-- Username -->
          <div class="input-group">
            <label for="username" class="input-label">
              <span class="label-text">username</span>
            </label>
            <div class="input-wrapper">
              <div class="i-lucide:user input-icon"></div>
              <input
                id="username"
                v-model="username"
                type="text"
                placeholder="admin"
                class="input-field"
                required
                autofocus
              />
            </div>
          </div>

          <!-- Password -->
          <div class="input-group">
            <label for="password" class="input-label">
              <span class="label-text">password</span>
            </label>
            <div class="input-wrapper">
              <div class="i-lucide:lock input-icon"></div>
              <input
                id="password"
                v-model="password"
                type="password"
                placeholder="••••••••"
                class="input-field"
                required
              />
            </div>
          </div>

          <!-- Submit Button -->
          <button type="submit" :disabled="loading" class="submit-btn">
            <span v-if="loading" class="loading-content">
              <div class="i-lucide:loader-2 spin-icon"></div>
              <span>authenticating...</span>
            </span>
            <span v-else class="btn-content">
              <span>connect</span>
              <div class="i-lucide:arrow-right"></div>
            </span>
          </button>

          <div class="form-footer">
            <span class="hint">default: admin / admin</span>
          </div>
        </form>
      </Transition>
    </div>

    <!-- Footer -->
    <div class="footer-text">
      <span>powered_by</span>
      <span class="highlight">tudou</span>
      <span class="version">v1.0.0</span>
    </div>
  </div>
</template>

<style scoped>
/* Container */
.login-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: var(--color-bg-primary);
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  color: var(--color-text-primary);
}

/* Grid Background */
.grid-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.grid-lines {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--grid-color) 1px, transparent 1px),
    linear-gradient(90deg, var(--grid-color) 1px, transparent 1px);
  background-size: 60px 60px;
  mask-image: radial-gradient(ellipse at center, black 20%, transparent 70%);
}

:root {
  --grid-color: rgba(16, 185, 129, 0.03);
}

[data-theme='dark'] {
  --grid-color: rgba(16, 185, 129, 0.03);
}

[data-theme='light'] {
  --grid-color: rgba(16, 185, 129, 0.08);
}

.glow-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.4;
}

.glow-orb-1 {
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, rgba(16, 185, 129, 0.3) 0%, transparent 70%);
  top: -150px;
  right: -100px;
  animation: float 20s ease-in-out infinite;
}

.glow-orb-2 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(6, 182, 212, 0.2) 0%, transparent 70%);
  bottom: -100px;
  left: -100px;
  animation: float 25s ease-in-out infinite reverse;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -30px); }
}

/* Content */
.login-content {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 400px;
  padding: 2rem;
}

/* Header */
.header-section {
  text-align: center;
  margin-bottom: 2.5rem;
}

.logo-container {
  position: relative;
  width: 72px;
  height: 72px;
  margin: 0 auto 1.5rem;
}

.logo-border {
  position: absolute;
  inset: 0;
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 16px;
  animation: pulse-border 3s ease-in-out infinite;
}

@keyframes pulse-border {
  0%, 100% { opacity: 0.3; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(1.02); }
}

.logo-inner {
  position: absolute;
  inset: 4px;
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-text {
  font-size: 32px;
  font-weight: 700;
  color: white;
  font-family: 'Outfit', sans-serif;
}

.title {
  font-size: 1.75rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin-bottom: 0.5rem;
  font-family: 'Outfit', sans-serif;
}

.typed-text {
  color: var(--color-text-primary);
}

.cursor {
  color: var(--color-primary);
  opacity: 0;
  transition: opacity 0.1s;
}

.cursor-visible {
  opacity: 1;
}

.subtitle {
  color: var(--color-text-muted);
  font-size: 0.875rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

/* Form */
.login-form {
  background: var(--form-bg);
  backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  padding: 1.75rem;
}

:root {
  --form-bg: rgba(24, 24, 27, 0.6);
}

[data-theme='light'] {
  --form-bg: rgba(255, 255, 255, 0.8);
}

.form-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--color-border);
}

.form-indicator {
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
  animation: blink 2s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.form-title {
  font-size: 0.75rem;
  color: var(--color-primary);
  letter-spacing: 0.05em;
}

/* Input Group */
.input-group {
  margin-bottom: 1.25rem;
}

.input-label {
  display: block;
  margin-bottom: 0.5rem;
}

.label-text {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 1rem;
  color: var(--color-text-muted);
  font-size: 1rem;
  pointer-events: none;
}

.input-field {
  width: 100%;
  padding: 0.875rem 1rem 0.875rem 2.75rem;
  background: var(--input-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  color: var(--color-text-primary);
  font-size: 0.9375rem;
  font-family: inherit;
  transition: all 0.2s;
}

:root {
  --input-bg: rgba(9, 9, 11, 0.6);
}

[data-theme='light'] {
  --input-bg: rgba(243, 244, 246, 0.8);
}

.input-field::placeholder {
  color: var(--color-text-muted);
}

.input-field:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

/* Error */
.error-message {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-danger-light);
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  margin-bottom: 1.25rem;
  font-size: 0.8125rem;
  color: var(--color-danger);
}

.error-icon {
  color: var(--color-danger);
}

/* Submit Button */
.submit-btn {
  width: 100%;
  padding: 0.875rem 1.5rem;
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  border: none;
  border-radius: 8px;
  color: white;
  font-size: 0.9375rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
  margin-top: 0.5rem;
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px rgba(16, 185, 129, 0.3);
}

.submit-btn:active:not(:disabled) {
  transform: translateY(0);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.btn-content, .loading-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Form Footer */
.form-footer {
  margin-top: 1rem;
  text-align: center;
}

.hint {
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

/* Footer Text */
.footer-text {
  position: absolute;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

.highlight {
  color: var(--color-primary);
}

.version {
  color: var(--color-text-tertiary);
  margin-left: 0.5rem;
}

/* Transition */
.form-appear-enter-active {
  animation: form-appear 0.5s ease-out;
}

@keyframes form-appear {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
