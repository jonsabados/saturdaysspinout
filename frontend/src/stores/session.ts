import { computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'
import { useWebSocketStore } from './websocket'
import { useTracksStore } from './tracks'
import { useCarsStore } from './cars'
import { useSeriesStore } from './series'

export interface LoadingState {
  id: string
  done: boolean
}

export const useSessionStore = defineStore('session', () => {
  const auth = useAuthStore()
  const ws = useWebSocketStore()
  const tracksStore = useTracksStore()
  const carsStore = useCarsStore()
  const seriesStore = useSeriesStore()

  const isLoggedIn = computed(() => auth.isLoggedIn)

  const isConnected = computed(() =>
    auth.isLoggedIn && ws.status === 'connected' && ws.connectionId !== null
  )

  const loadingStates = computed<LoadingState[]>(() => [
    { id: 'tracks', done: tracksStore.isLoaded },
    { id: 'cars', done: carsStore.isLoaded },
    { id: 'series', done: seriesStore.isLoaded },
  ])

  const pendingStates = computed(() => loadingStates.value.filter((s) => !s.done))

  const isReady = computed(() => isLoggedIn.value && pendingStates.value.length === 0)

  const userId = computed(() => auth.userId)
  const userName = computed(() => auth.userName)
  const connectionId = computed(() => ws.connectionId)

  // Load reference data when logged in
  watch(
    isLoggedIn,
    (loggedIn) => {
      if (loggedIn) {
        tracksStore.fetchTracks().catch((err) => console.error('[Session] Failed to load tracks:', err))
        carsStore.fetchCars().catch((err) => console.error('[Session] Failed to load cars:', err))
        seriesStore.fetchSeries().catch((err) => console.error('[Session] Failed to load series:', err))
      }
    },
    { immediate: true }
  )

  function logout() {
    auth.logout()
  }

  return {
    isLoggedIn,
    isConnected,
    isReady,
    loadingStates,
    pendingStates,
    userId,
    userName,
    connectionId,
    logout,
  }
})