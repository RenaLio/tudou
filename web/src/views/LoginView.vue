<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { login } from '@/api/auth'
import AppButton from '@/components/ui/AppButton.vue'
import AppInput from '@/components/ui/AppInput.vue'
import AppFormField from '@/components/ui/AppFormField.vue'

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
  setTimeout(() => { showForm.value = true }, 400)
})

onUnmounted(() => {
  cancelAnimationFrame(animFrame)
})

function initCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')!
  if (!ctx) return

  let w = canvas.width = window.innerWidth
  let h = canvas.height = window.innerHeight

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

    const grad = ctx.createRadialGradient(w / 2, h / 2, 0, w / 2, h / 2, Math.max(w, h))
    grad.addColorStop(0, 'rgba(13, 20, 13, 0)')
    grad.addColorStop(1, 'rgba(8, 12, 8, 0.4)')
    ctx.fillStyle = grad
    ctx.fillRect(0, 0, w, h)

    const connectDist = 160
    for (let i = 0; i < nodes.length; i++) {
      const a = nodes[i]!
      a.x += a.vx
      a.y += a.vy
      a.pulsePhase += a.pulseSpeed

      if (a.x < -10) a.x = w + 10
      if (a.x > w + 10) a.x = -10
      if (a.y < -10) a.y = h + 10
      if (a.y > h + 10) a.y = -10

      const dMouse = Math.hypot(a.x - mouseX, a.y - mouseY)
      if (dMouse < 200) {
        const force = (200 - dMouse) / 200 * 0.02
        a.vx += (mouseX - a.x) / dMouse * force
        a.vy += (mouseY - a.y) / dMouse * force
        a.vx *= 0.99
        a.vy *= 0.99
      }

      for (let j = i + 1; j < nodes.length; j++) {
        const b = nodes[j]!
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

    for (const n of nodes) {
      const pulse = (Math.sin(n.pulsePhase) + 1) / 2
      const glowRadius = n.radius * (2 + pulse * 2)

      const glow = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, glowRadius)
      glow.addColorStop(0, `rgba(139, 195, 74, ${0.3 * pulse})`)
      glow.addColorStop(1, 'rgba(139, 195, 74, 0)')
      ctx.fillStyle = glow
      ctx.beginPath()
      ctx.arc(n.x, n.y, glowRadius, 0, Math.PI * 2)
      ctx.fill()

      ctx.fillStyle = `rgba(139, 195, 74, ${0.5 + pulse * 0.5})`
      ctx.beginPath()
      ctx.arc(n.x, n.y, n.radius, 0, Math.PI * 2)
      ctx.fill()
    }

    animFrame = requestAnimationFrame(draw)
  }

  draw()

  const cleanup = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('resize', onResize)
    cancelAnimationFrame(animFrame)
  }
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
  <div class="min-h-screen flex flex-col items-center justify-center relative overflow-hidden bg-bg-primary">
    <!-- Animated network background -->
    <canvas ref="canvasRef" class="fixed inset-0 z-0 pointer-events-none" />

    <!-- Ambient gradient orbs -->
    <div class="fixed -top-[200px] -right-[150px] w-[500px] h-[500px] rounded-full blur-[120px] pointer-events-none z-[1] bg-primary/5 animate-pulse" />
    <div class="fixed -bottom-[150px] -left-[100px] w-[400px] h-[400px] rounded-full blur-[120px] pointer-events-none z-[1] bg-accent/5" />

    <!-- Main content -->
    <div class="relative z-10 flex flex-col items-center gap-10 px-8 w-full max-w-[400px]">
      <!-- Brand -->
      <div class="flex flex-col items-center gap-4">
        <div class="w-16 h-16 rounded-full p-0.5 bg-gradient-to-br from-primary/40 via-accent/30 to-primary/40">
          <div class="w-full h-full rounded-full bg-bg-primary flex items-center justify-center">
            <span class="font-display text-[1.75rem] font-bold bg-gradient-to-br from-primary to-accent bg-clip-text text-transparent">T</span>
          </div>
        </div>
        <h1 class="flex items-center gap-3 m-0 font-display">
          <span class="text-[1.875rem] font-bold text-text-primary tracking-tight">Tudou</span>
          <span class="w-px h-5 bg-border-hover"></span>
          <span class="text-lg font-normal text-text-tertiary tracking-widest uppercase">Gateway</span>
        </h1>
        <p class="m-0 text-[13px] text-text-muted tracking-[0.15em]">根系深处的智能通道</p>
      </div>

      <!-- Login card -->
      <Transition
        enter-active-class="transition-all duration-[600ms]"
        enter-from-class="opacity-0 translate-y-8 scale-[0.96]"
        enter-to-class="opacity-100 translate-y-0 scale-100"
      >
        <div v-if="showForm" class="relative w-full">
          <!-- Glow border -->
          <div class="absolute -inset-px rounded-[calc(var(--radius-xl)+1px)] bg-gradient-to-br from-primary/20 via-accent/10 to-primary/20 z-0 animate-pulse" />

          <div class="relative z-[1] bg-bg-card/90 backdrop-blur-xl rounded-xl p-7 border border-border">
            <!-- Header -->
            <div class="flex items-center gap-3 mb-6">
              <div class="flex-1 h-px bg-gradient-to-r from-transparent via-border-hover to-transparent" />
              <span class="font-mono text-[11px] text-text-muted tracking-widest uppercase whitespace-nowrap">secure_access</span>
              <div class="flex-1 h-px bg-gradient-to-r from-transparent via-border-hover to-transparent" />
            </div>

            <!-- Error -->
            <Transition
              enter-active-class="transition-all duration-300"
              enter-from-class="opacity-0 -translate-y-2"
              enter-to-class="opacity-100 translate-y-0"
            >
              <div v-if="errorMessage" class="flex items-center gap-2.5 px-3.5 py-2.5 bg-danger-light border border-danger/15 rounded-lg mb-5"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-danger shrink-0 animate-pulse" />
                <span class="text-sm text-danger">{{ errorMessage }}</span>
              </div>
            </Transition>

            <!-- Form -->
            <form @submit.prevent="handleSubmit" class="flex flex-col gap-5">
              <AppFormField label="用户名">
                <AppInput
                  v-model="username"
                  type="text"
                  placeholder="admin"
                  required
                  autofocus
                />
              </AppFormField>

              <AppFormField label="密码">
                <AppInput
                  v-model="password"
                  type="password"
                  placeholder="••••••••"
                  required
                />
              </AppFormField>

              <AppButton
                type="submit"
                variant="primary"
                size="lg"
                class="w-full mt-2"
                :loading="loading"
              >
                <span v-if="!loading" class="flex items-center justify-center gap-2">
                  进入系统
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12" />
                    <polyline points="12 5 19 12 12 19" />
                  </svg>
                </span>
                <span v-else>连接中...</span>
              </AppButton>
            </form>

            <!-- Footer -->
            <div class="mt-5 text-center">
              <span class="font-mono text-[11px] text-text-muted tracking-wider">默认账号: admin / admin</span>
            </div>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Bottom attribution -->
    <div class="fixed bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-2 z-10 font-mono text-[11px] text-text-muted tracking-wider">
      <span class="text-primary">tudou</span>
      <span class="opacity-40">·</span>
      <span>v1.0</span>
    </div>
  </div>
</template>
