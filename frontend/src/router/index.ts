import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import AuthCallbackView from '../views/AuthCallbackView.vue'
import ApiExplorerView from '../views/ApiExplorerView.vue'
import RaceHistoryView from '../views/RaceHistoryView.vue'
import RaceDetailsView from '../views/RaceDetailsView.vue'
import TrackDetailsView from '../views/TrackDetailsView.vue'
import CarDetailsView from '../views/CarDetailsView.vue'
import { useSessionStore } from '@/stores/session'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/auth/ir/callback',
      name: 'auth-callback',
      component: AuthCallbackView,
    },
    {
      path: '/iracing-api',
      name: 'iracing-api',
      component: ApiExplorerView,
      meta: { requiresAuth: true, requiresEntitlement: 'developer' },
    },
    {
      path: '/race-history',
      name: 'race-history',
      component: RaceHistoryView,
      meta: { requiresAuth: true, keepAlive: true },
    },
    {
      path: '/race/:subsessionId',
      name: 'race-details',
      component: RaceDetailsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/tracks/:id',
      name: 'track-details',
      component: TrackDetailsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/cars/:id',
      name: 'car-details',
      component: CarDetailsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/journal',
      name: 'journal',
      component: () => import('@/views/JournalView.vue'),
      meta: { requiresAuth: true, keepAlive: true },
    },
  ],
})

router.beforeEach((to) => {
  const session = useSessionStore()
  const auth = useAuthStore()

  // Redirect logged-in users from home to race-history
  if (to.name === 'home' && session.isLoggedIn) {
    return { name: 'race-history' }
  }

  // Redirect unauthenticated users away from protected routes
  if (to.meta.requiresAuth && !session.isLoggedIn) {
    return { name: 'home' }
  }

  // Check entitlement requirements
  if (to.meta.requiresEntitlement && typeof to.meta.requiresEntitlement === 'string') {
    if (!auth.hasEntitlement(to.meta.requiresEntitlement)) {
      return { name: 'race-history' }
    }
  }
})

export default router