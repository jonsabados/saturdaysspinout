import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useRaceIngestionStore, setupRaceIngestionListener } from './raceIngestion'

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

// Capture the stale credentials callback when registered
let staleCredentialsCallback: (() => void) | null = null
const mockWsOn = vi.fn((event: string, callback: () => void) => {
  if (event === 'ingestionFailedStaleCredentials') {
    staleCredentialsCallback = callback
  }
})

vi.mock('./websocket', () => ({
  useWebSocketStore: () => ({
    on: mockWsOn,
    off: vi.fn(),
  }),
}))

describe('raceIngestion store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockSessionIsReady = true
    staleCredentialsCallback = null
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
      mockTriggerRaceIngestion.mockResolvedValue({ throttled: false })
      const store = useRaceIngestionStore()

      const promise = store.triggerIngestion()
      expect(store.status).toBe('loading')

      await promise
      expect(store.status).toBe('success')
      expect(store.error).toBeNull()
    })

    it('sets status to idle when throttled (429)', async () => {
      mockTriggerRaceIngestion.mockResolvedValue({ throttled: true, retryAfter: 600 })
      const store = useRaceIngestionStore()

      await store.triggerIngestion()

      expect(store.status).toBe('idle')
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

  describe('setupListener', () => {
    it('registers listener for stale credentials event', () => {
      const store = useRaceIngestionStore()
      store.setupListener()

      expect(mockWsOn).toHaveBeenCalledWith('ingestionFailedStaleCredentials', expect.any(Function))
    })
  })

  describe('stale credentials handling (via listener)', () => {
    it('refreshes token and retries ingestion when refresh succeeds', async () => {
      mockRefreshToken.mockResolvedValue(true)
      mockTriggerRaceIngestion.mockResolvedValue({ throttled: false })
      const store = useRaceIngestionStore()

      // Set up the listener to capture the callback
      store.setupListener()
      expect(staleCredentialsCallback).not.toBeNull()

      // Trigger the callback as if websocket received the event
      await staleCredentialsCallback!()

      expect(mockRefreshToken).toHaveBeenCalled()
      expect(mockTriggerRaceIngestion).toHaveBeenCalled()
      expect(store.status).toBe('success')
    })

    it('sets error state when refresh fails', async () => {
      mockRefreshToken.mockResolvedValue(false)
      const store = useRaceIngestionStore()

      store.setupListener()
      await staleCredentialsCallback!()

      expect(mockRefreshToken).toHaveBeenCalled()
      expect(mockTriggerRaceIngestion).not.toHaveBeenCalled()
      expect(store.status).toBe('error')
      expect(store.error).toBe('Session expired. Please log in again.')
    })
  })

  describe('setupRaceIngestionListener', () => {
    it('calls setupListener on the store', () => {
      setupRaceIngestionListener()

      expect(mockWsOn).toHaveBeenCalledWith('ingestionFailedStaleCredentials', expect.any(Function))
    })
  })
})