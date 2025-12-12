import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import AuthCallbackView from '../views/AuthCallbackView.vue'
import ApiExplorerView from '../views/ApiExplorerView.vue'

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
  ],
})

export default router