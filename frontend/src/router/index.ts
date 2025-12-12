import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import AuthCallbackView from '../views/AuthCallbackView.vue'
import ApiExplorerView from '../views/ApiExplorerView.vue'
import RaceHistoryView from '../views/RaceHistoryView.vue'

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
    },
    {
      path: '/race-history',
      name: 'race-history',
      component: RaceHistoryView,
    },
  ],
})

export default router