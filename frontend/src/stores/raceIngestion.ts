import { defineStore } from 'pinia'
import { watch } from 'vue'
import { useApiClient } from '@/api/client'
import { useAuthStore } from './auth'
import { useSessionStore } from './session'
import { useWebSocketStore } from './websocket'

type IngestionStatus = 'idle' | 'loading' | 'success' | 'error'

interface RaceIngestionState {
  status: IngestionStatus
  error: string | null
}

export const useRaceIngestionStore = defineStore('raceIngestion', {
  state: (): RaceIngestionState => ({
    status: 'idle',
    error: null,
  }),

  actions: {
    async triggerIngestion() {
      const sessionStore = useSessionStore()
      if (!sessionStore.isReady) {
        console.log('[RaceIngestion] Session not ready, skipping ingestion')
        return
      }

      this.status = 'loading'
      this.error = null

      try {
        const apiClient = useApiClient()
        await apiClient.triggerRaceIngestion()
        this.status = 'success'
      } catch (err) {
        this.status = 'error'
        this.error = err instanceof Error ? err.message : 'Failed to trigger ingestion'
      }
    },

    async _handleStaleCredentials() {
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
        await this.triggerIngestion()
      } else {
        console.error('[RaceIngestion] Token refresh failed, cannot retry ingestion')
        this.status = 'error'
        this.error = 'Session expired. Please log in again.'
      }
    },
  },
})

export function setupRaceIngestionListener() {
  const wsStore = useWebSocketStore()
  const ingestionStore = useRaceIngestionStore()

  wsStore.on('ingestionFailedStaleCredentials', () => ingestionStore._handleStaleCredentials())
}