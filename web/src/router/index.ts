import { useAuthStore, } from '@/stores/auth';
import { createRouter, createWebHistory, } from 'vue-router';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL,),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false, },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true, },
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        {
          path: 'request-logs',
          name: 'request-logs',
          component: () => import('@/views/RequestLogsView.vue'),
        },
        {
          path: 'tokens',
          name: 'tokens',
          component: () => import('@/views/TokensView.vue'),
        },
        {
          path: 'channels',
          name: 'channels',
          component: () => import('@/views/ChannelsView.vue'),
        },
        {
          path: 'models',
          name: 'models',
          component: () => import('@/views/ModelsView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
        },
      ],
    },
  ],
},);

// Navigation guard
router.beforeEach((to, _from, next,) => {
  const authStore = useAuthStore();

  if (to.meta.requiresAuth !== false && !authStore.isAuthenticated) {
    next('/login',);
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/dashboard',);
  } else {
    next();
  }
},);

export default router;
