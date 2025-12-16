import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { useApiClient } from '@/api/client'
import { useAuthStore } from './auth'
import { useSessionStore } from './session'
import { useWebSocketStore } from './websocket'

type IngestionStatus = 'idle' | 'loading' | 'success' | 'error'

export const useRaceIngestionStore = defineStore('raceIngestion', () => {
  // Public state
  const status = ref<IngestionStatus>('idle')
  const error = ref<string | null>(null)

  // Private method
  async function handleStaleCredentials() {
    console.log('[RaceIngestion] Received stale credentials signal, refreshing token...')
    const authStore = useAuthStore()
    const sessionStore = useSessionStore()

    const refreshed = await authStore.refreshToken()
    if (refreshed) {
      // Token refresh triggers websocket reconnect - wait for session to be ready
      if (!sessionStore.isReady) {
        console.log('[RaceIngestion] Waiting for session to be ready...')
        await new Promise<void>((resolve) => {
          const unwatch = watch(
            () => sessionStore.isReady,
            (ready) => {
              if (ready) {
                unwatch()
                resolve()
              }
            },
          )
        })
      }
      console.log('[RaceIngestion] Session ready, retrying ingestion')
      await triggerIngestion()
    } else {
      console.error('[RaceIngestion] Token refresh failed, cannot retry ingestion')
      status.value = 'error'
      error.value = 'Session expired. Please log in again.'
    }
  }

  // Public methods
  async function triggerIngestion() {
    const sessionStore = useSessionStore()
    if (!sessionStore.isReady) {
      console.log('[RaceIngestion] Session not ready, skipping ingestion')
      return
    }

    status.value = 'loading'
    error.value = null

    try {
      const apiClient = useApiClient()
      await apiClient.triggerRaceIngestion()
      status.value = 'success'
    } catch (err) {
      status.value = 'error'
      error.value = err instanceof Error ? err.message : 'Failed to trigger ingestion'
    }
  }

  function setupListener() {
    const wsStore = useWebSocketStore()
    wsStore.on('ingestionFailedStaleCredentials', () => handleStaleCredentials())
  }

  return {
    // State
    status,
    error,
    // Actions
    triggerIngestion,
    setupListener,
  }
})

export function setupRaceIngestionListener() {
  const ingestionStore = useRaceIngestionStore()
  ingestionStore.setupListener()
}