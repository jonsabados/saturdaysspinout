import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'
import { useWebSocketStore } from './websocket'

export const useSessionStore = defineStore('session', () => {
  const auth = useAuthStore()
  const ws = useWebSocketStore()

  const isLoggedIn = computed(() => auth.isLoggedIn)

  const isReady = computed(() =>
    auth.isLoggedIn && ws.status === 'connected' && ws.connectionId !== null
  )

  const userId = computed(() => auth.userId)
  const userName = computed(() => auth.userName)
  const connectionId = computed(() => ws.connectionId)

  function logout() {
    auth.logout()
  }

  return {
    isLoggedIn,
    isReady,
    userId,
    userName,
    connectionId,
    logout,
  }
})