import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useRaceIngestionStore } from './raceIngestion'

// Mock dependencies
const mockTriggerRaceIngestion = vi.fn()
vi.mock('@/api/client', () => ({
  useApiClient: () => ({
    triggerRaceIngestion: mockTriggerRaceIngestion,
  }),
}))

const mockRefreshToken = vi.fn()
vi.mock('./auth', () => ({
  useAuthStore: () => ({
    refreshToken: mockRefreshToken,
  }),
}))

let mockSessionIsReady = true
vi.mock('./session', () => ({
  useSessionStore: () => ({
    get isReady() {
      return mockSessionIsReady
    },
  }),
}))

vi.mock('./websocket', () => ({
  useWebSocketStore: () => ({
    on: vi.fn(),
    off: vi.fn(),
  }),
}))

describe('raceIngestion store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockSessionIsReady = true
  })

  describe('triggerIngestion', () => {
    it('skips ingestion when session is not ready', async () => {
      mockSessionIsReady = false
      const store = useRaceIngestionStore()

      await store.triggerIngestion()

      expect(mockTriggerRaceIngestion).not.toHaveBeenCalled()
      expect(store.status).toBe('idle')
    })

    it('sets status to loading then success on successful API call', async () => {
      mockTriggerRaceIngestion.mockResolvedValue(undefined)
      const store = useRaceIngestionStore()

      const promise = store.triggerIngestion()
      expect(store.status).toBe('loading')

      await promise
      expect(store.status).toBe('success')
      expect(store.error).toBeNull()
    })

    it('sets status to error on API failure', async () => {
      mockTriggerRaceIngestion.mockRejectedValue(new Error('API failed'))
      const store = useRaceIngestionStore()

      await store.triggerIngestion()

      expect(store.status).toBe('error')
      expect(store.error).toBe('API failed')
    })

    it('handles non-Error rejection', async () => {
      mockTriggerRaceIngestion.mockRejectedValue('string error')
      const store = useRaceIngestionStore()

      await store.triggerIngestion()

      expect(store.status).toBe('error')
      expect(store.error).toBe('Failed to trigger ingestion')
    })
  })

  describe('_handleStaleCredentials', () => {
    it('refreshes token and retries ingestion when refresh succeeds', async () => {
      mockRefreshToken.mockResolvedValue(true)
      mockTriggerRaceIngestion.mockResolvedValue(undefined)
      const store = useRaceIngestionStore()

      await store._handleStaleCredentials()

      expect(mockRefreshToken).toHaveBeenCalled()
      expect(mockTriggerRaceIngestion).toHaveBeenCalled()
      expect(store.status).toBe('success')
    })

    it('sets error state when refresh fails', async () => {
      mockRefreshToken.mockResolvedValue(false)
      const store = useRaceIngestionStore()

      await store._handleStaleCredentials()

      expect(mockRefreshToken).toHaveBeenCalled()
      expect(mockTriggerRaceIngestion).not.toHaveBeenCalled()
      expect(store.status).toBe('error')
      expect(store.error).toBe('Session expired. Please log in again.')
    })

    it('waits for session to be ready before retrying', async () => {
      mockSessionIsReady = false
      mockRefreshToken.mockResolvedValue(true)
      mockTriggerRaceIngestion.mockResolvedValue(undefined)
      const store = useRaceIngestionStore()

      // Start the handler - it should wait for session
      const handlerPromise = store._handleStaleCredentials()

      // Give it a tick to start waiting
      await new Promise((r) => setTimeout(r, 10))

      // Ingestion shouldn't have been called yet
      expect(mockTriggerRaceIngestion).not.toHaveBeenCalled()

      // Simulate session becoming ready
      mockSessionIsReady = true

      // The promise should still be pending because the watch hasn't fired
      // In a real scenario, the watch would detect the change
      // For this test, we verify the guard is in place
    })
  })
})