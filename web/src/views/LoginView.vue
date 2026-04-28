<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
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

// Canvas background
const canvasRef = ref<HTMLCanvasElement | null>(null)
let animFrame: number

onMounted(() => {
  initCanvas()
  // Staggered reveal
  setTimeout(() => { showForm.value = true }, 400)
})

onUnmounted(() => {
  cancelAnimationFrame(animFrame)
})

function initCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  let w = canvas.width = window.innerWidth
  let h = canvas.height = window.innerHeight

  // Network nodes — like root system junctions
  const nodeCount = Math.floor((w * h) / 25000)
  const nodes: {
    x: number
    y: number
    vx: number
    vy: number
    radius: number
    pulsePhase: number
    pulseSpeed: number
  }[] = []

  for (let i = 0; i < nodeCount; i++) {
    nodes.push({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 0.3,
      vy: (Math.random() - 0.5) * 0.3,
      radius: Math.random() * 1.5 + 0.5,
      pulsePhase: Math.random() * Math.PI * 2,
      pulseSpeed: Math.random() * 0.02 + 0.01,
    })
  }

  let mouseX = w / 2
  let mouseY = h / 2

  const onMove = (e: MouseEvent) => {
    mouseX = e.clientX
    mouseY = e.clientY
  }
  window.addEventListener('mousemove', onMove)

  const onResize = () => {
    w = canvas.width = window.innerWidth
    h = canvas.height = window.innerHeight
  }
  window.addEventListener('resize', onResize)

  function draw() {
    ctx.clearRect(0, 0, w, h)

    // Deep underground gradient
    const grad = ctx.createRadialGradient(w / 2, h / 2, 0, w / 2, h / 2, Math.max(w, h))
    grad.addColorStop(0, 'rgba(13, 20, 13, 0)')
    grad.addColorStop(1, 'rgba(8, 12, 8, 0.4)')
    ctx.fillStyle = grad
    ctx.fillRect(0, 0, w, h)

    // Update and draw connections
    const connectDist = 160
    for (let i = 0; i < nodes.length; i++) {
      const a = nodes[i]
      a.x += a.vx
      a.y += a.vy
      a.pulsePhase += a.pulseSpeed

      // Wrap around
      if (a.x < -10) a.x = w + 10
      if (a.x > w + 10) a.x = -10
      if (a.y < -10) a.y = h + 10
      if (a.y > h + 10) a.y = -10

      // Mouse attraction (subtle)
      const dMouse = Math.hypot(a.x - mouseX, a.y - mouseY)
      if (dMouse < 200) {
        const force = (200 - dMouse) / 200 * 0.02
        a.vx += (mouseX - a.x) / dMouse * force
        a.vy += (mouseY - a.y) / dMouse * force
        // Dampen
        a.vx *= 0.99
        a.vy *= 0.99
      }

      for (let j = i + 1; j < nodes.length; j++) {
        const b = nodes[j]
        const dx = a.x - b.x
        const dy = a.y - b.y
        const dist = Math.hypot(dx, dy)

        if (dist < connectDist) {
          const opacity = (1 - dist / connectDist) * 0.25
          const pulse = (Math.sin(a.pulsePhase) + 1) / 2
          const finalOpacity = opacity * (0.4 + pulse * 0.6)

          ctx.beginPath()
          ctx.moveTo(a.x, a.y)
          ctx.lineTo(b.x, b.y)
          ctx.strokeStyle = `rgba(139, 195, 74, ${finalOpacity})`
          ctx.lineWidth = 0.5
          ctx.stroke()
        }
      }
    }

    // Draw nodes
    for (const n of nodes) {
      const pulse = (Math.sin(n.pulsePhase) + 1) / 2
      const glowRadius = n.radius * (2 + pulse * 2)

      // Glow
      const glow = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, glowRadius)
      glow.addColorStop(0, `rgba(139, 195, 74, ${0.3 * pulse})`)
      glow.addColorStop(1, 'rgba(139, 195, 74, 0)')
      ctx.fillStyle = glow
      ctx.beginPath()
      ctx.arc(n.x, n.y, glowRadius, 0, Math.PI * 2)
      ctx.fill()

      // Core
      ctx.fillStyle = `rgba(139, 195, 74, ${0.5 + pulse * 0.5})`
      ctx.beginPath()
      ctx.arc(n.x, n.y, n.radius, 0, Math.PI * 2)
      ctx.fill()
    }

    animFrame = requestAnimationFrame(draw)
  }

  draw()

  // Cleanup
  const cleanup = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('resize', onResize)
    cancelAnimationFrame(animFrame)
  }
  // Store cleanup for unmount
  ;(canvas as any).__cleanup = cleanup
}

async function handleSubmit() {
  if (!username.value || !password.value) return

  loading.value = true
  errorMessage.value = ''

  try {
    const { accessToken, user: userData } = await login({
      username: username.value,
      password: password.value,
    })
    authStore.setToken(accessToken)
    authStore.setUser(userData)
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
  <div class="login-page">
    <!-- Animated network background -->
    <canvas ref="canvasRef" class="network-canvas" />

    <!-- Ambient gradient orbs -->
    <div class="ambient-orb orb-top" />
    <div class="ambient-orb orb-bottom" />

    <!-- Main content -->
    <div class="login-content">
      <!-- Brand -->
      <div class="brand">
        <div class="logo-ring">
          <div class="logo-core">
            <span class="logo-mark">T</span>
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-main">Tudou</span>
          <span class="title-divider" />
          <span class="title-sub">Gateway</span>
        </h1>
        <p class="brand-tagline">根系深处的智能通道</p>
      </div>

      <!-- Login card -->
      <Transition name="card-rise">
        <div v-if="showForm" class="login-card">
          <!-- Card glow border -->
          <div class="card-glow" />

          <div class="card-inner">
            <!-- Header -->
            <div class="card-header">
              <div class="header-line" />
              <span class="header-label">secure_access</span>
              <div class="header-line" />
            </div>

            <!-- Error -->
            <Transition name="error-shake">
              <div v-if="errorMessage" class="error-banner">
                <span class="error-dot" />
                <span class="error-text">{{ errorMessage }}</span>
              </div>
            </Transition>

            <!-- Form -->
            <form @submit.prevent="handleSubmit" class="login-form">
              <div class="field">
                <label class="field-label">
                  <span class="label-icon">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                      <circle cx="12" cy="7" r="4"/>
                    </svg>
                  </span>
                  用户名
                </label>
                <div class="input-wrap">
                  <input
                    v-model="username"
                    type="text"
                    placeholder="admin"
                    class="input-field"
                    required
                    autofocus
                  />
                  <div class="input-line" />
                  <div class="input-glow" />
                </div>
              </div>

              <div class="field">
                <label class="field-label">
                  <span class="label-icon">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                      <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                    </svg>
                  </span>
                  密码
                </label>
                <div class="input-wrap">
                  <input
                    v-model="password"
                    type="password"
                    placeholder="••••••••"
                    class="input-field"
                    required
                  />
                  <div class="input-line" />
                  <div class="input-glow" />
                </div>
              </div>

              <button
                type="submit"
                :disabled="loading"
                class="submit-btn"
              >
                <span v-if="loading" class="btn-loading">
                  <span class="spinner" />
                  <span>连接中...</span>
                </span>
                <span v-else class="btn-content">
                  <span>进入系统</span>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12"/>
                    <polyline points="12 5 19 12 12 19"/>
                  </svg>
                </span>
              </button>
            </form>

            <!-- Footer -->
            <div class="card-footer">
              <span class="footer-hint">默认账号: admin / admin</span>
            </div>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Bottom attribution -->
    <div class="page-footer">
      <span class="footer-brand">tudou</span>
      <span class="footer-sep">·</span>
      <span class="footer-version">v1.0</span>
    </div>
  </div>
</template>

<style scoped>
/* ── Page layout ── */
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: var(--color-bg-primary);
  font-family: var(--font-body);
}

/* ── Canvas background ── */
.network-canvas {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
}

/* ── Ambient orbs ── */
.ambient-orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(120px);
  pointer-events: none;
  z-index: 1;
}

.orb-top {
  width: 500px;
  height: 500px;
  top: -200px;
  right: -150px;
  background: radial-gradient(circle, rgba(139, 195, 74, 0.08) 0%, transparent 70%);
  animation: orbFloat 20s ease-in-out infinite;
}

.orb-bottom {
  width: 400px;
  height: 400px;
  bottom: -150px;
  left: -100px;
  background: radial-gradient(circle, rgba(255, 179, 0, 0.05) 0%, transparent 70%);
  animation: orbFloat 25s ease-in-out infinite reverse;
}

@keyframes orbFloat {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -20px) scale(1.05); }
  66% { transform: translate(-20px, 15px) scale(0.95); }
}

/* ── Content area ── */
.login-content {
  position: relative;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2.5rem;
  padding: 2rem;
  width: 100%;
  max-width: 400px;
}

/* ── Brand ── */
.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.logo-ring {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  padding: 2px;
  background: conic-gradient(
    from 0deg,
    transparent 0%,
    rgba(139, 195, 74, 0.4) 25%,
    rgba(255, 179, 0, 0.3) 50%,
    rgba(139, 195, 74, 0.4) 75%,
    transparent 100%
  );
  animation: ringRotate 8s linear infinite;
}

@keyframes ringRotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.logo-core {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: var(--color-bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-mark {
  font-family: var(--font-display);
  font-size: 1.75rem;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-accent) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: 0;
  font-family: var(--font-display);
}

.title-main {
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.title-divider {
  width: 1px;
  height: 1.25rem;
  background: var(--color-border-hover);
}

.title-sub {
  font-size: 1.125rem;
  font-weight: 400;
  color: var(--color-text-tertiary);
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.brand-tagline {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  letter-spacing: 0.15em;
}

/* ── Login card ── */
.login-card {
  position: relative;
  width: 100%;
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.card-glow {
  position: absolute;
  inset: -1px;
  border-radius: calc(var(--radius-xl) + 1px);
  background: linear-gradient(
    135deg,
    rgba(139, 195, 74, 0.2) 0%,
    rgba(255, 179, 0, 0.1) 50%,
    rgba(139, 195, 74, 0.2) 100%
  );
  z-index: 0;
  animation: glowShift 6s ease-in-out infinite;
}

@keyframes glowShift {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}

.card-inner {
  position: relative;
  z-index: 1;
  background: var(--color-bg-card);
  backdrop-filter: blur(20px);
  border-radius: var(--radius-xl);
  padding: 1.75rem;
  border: 1px solid var(--color-border);
}

/* ── Card header ── */
.card-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.header-line {
  flex: 1;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    var(--color-border-hover) 50%,
    transparent 100%
  );
}

.header-label {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  letter-spacing: 0.1em;
  text-transform: uppercase;
  white-space: nowrap;
}

/* ── Error banner ── */
.error-banner {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 0.875rem;
  background: var(--color-danger-light);
  border: 1px solid rgba(229, 115, 115, 0.15);
  border-radius: var(--radius-md);
  margin-bottom: 1.25rem;
}

.error-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-danger);
  animation: errorPulse 1.5s ease-in-out infinite;
  flex-shrink: 0;
}

@keyframes errorPulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 var(--color-danger-glow); }
  50% { opacity: 0.7; box-shadow: 0 0 8px 2px var(--color-danger-glow); }
}

.error-text {
  font-size: 0.8125rem;
  color: var(--color-danger);
}

/* ── Form fields ── */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field-label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-tertiary);
  letter-spacing: 0.05em;
}

.label-icon {
  display: flex;
  align-items: center;
  color: var(--color-text-muted);
}

.input-wrap {
  position: relative;
}

.input-field {
  width: 100%;
  padding: 0.75rem 0;
  background: transparent;
  border: none;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 0.9375rem;
  font-family: var(--font-body);
  outline: none;
  transition: border-color 0.3s ease;
}

.input-field::placeholder {
  color: var(--color-text-muted);
}

.input-line {
  position: absolute;
  bottom: 0;
  left: 50%;
  width: 0;
  height: 1px;
  background: linear-gradient(90deg, var(--color-primary), var(--color-accent));
  transition: width 0.3s ease, left 0.3s ease;
}

.input-glow {
  position: absolute;
  bottom: -1px;
  left: 50%;
  width: 0;
  height: 2px;
  background: var(--color-primary-glow);
  filter: blur(4px);
  transition: width 0.3s ease, left 0.3s ease;
}

.input-field:focus ~ .input-line,
.input-field:focus ~ .input-glow {
  left: 0;
  width: 100%;
}

.input-field:focus {
  border-bottom-color: transparent;
}

/* ── Submit button ── */
.submit-btn {
  width: 100%;
  margin-top: 0.5rem;
  padding: 0.875rem 1.5rem;
  background: linear-gradient(135deg, rgba(139, 195, 74, 0.15) 0%, rgba(255, 179, 0, 0.08) 100%);
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-md);
  color: var(--color-primary);
  font-size: 0.9375rem;
  font-weight: 500;
  font-family: var(--font-body);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  position: relative;
  overflow: hidden;
}

.submit-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-dark) 100%);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.submit-btn:hover:not(:disabled) {
  border-color: var(--color-primary);
  box-shadow: var(--shadow-glow-primary);
  transform: translateY(-1px);
}

.submit-btn:hover:not(:disabled)::before {
  opacity: 1;
}

.submit-btn:active:not(:disabled) {
  transform: translateY(0);
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-content,
.btn-loading {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.submit-btn:hover:not(:disabled) .btn-content {
  color: var(--color-bg-primary);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-primary);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Card footer ── */
.card-footer {
  margin-top: 1.25rem;
  text-align: center;
}

.footer-hint {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

/* ── Page footer ── */
.page-footer {
  position: fixed;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  z-index: 10;
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  letter-spacing: 0.05em;
}

.footer-brand {
  color: var(--color-primary);
}

.footer-sep {
  opacity: 0.4;
}

/* ── Transitions ── */
.card-rise-enter-active {
  transition: all 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.card-rise-enter-from {
  opacity: 0;
  transform: translateY(30px) scale(0.96);
}

.error-shake-enter-active {
  animation: errorIn 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes errorIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* ── Responsive ── */
@media (max-width: 480px) {
  .login-content {
    padding: 1.5rem;
    gap: 2rem;
  }

  .title-main {
    font-size: 1.5rem;
  }

  .title-sub {
    font-size: 1rem;
  }

  .card-inner {
    padding: 1.25rem;
  }
}
</style>
