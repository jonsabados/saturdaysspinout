import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ApiClient } from './client'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

// Mock store types
interface MockAuthStore {
  isLoggedIn: boolean
  token: string
  refreshToken: ReturnType<typeof vi.fn>
}

interface MockSessionStore {
  isReady: boolean
  connectionId: string
}

// Create mock stores
function createMockAuthStore(overrides: Partial<MockAuthStore> = {}): MockAuthStore {
  return {
    isLoggedIn: true,
    token: 'test-token',
    refreshToken: vi.fn().mockResolvedValue(true),
    ...overrides,
  }
}

function createMockSessionStore(overrides: Partial<MockSessionStore> = {}): MockSessionStore {
  return {
    isReady: true,
    connectionId: 'test-connection-id',
    ...overrides,
  }
}

function createJsonResponse(data: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    headers: new Headers(),
  }
}

function createErrorResponse(status: number, message?: string) {
  return {
    ok: false,
    status,
    json: () => Promise.resolve({ message: message ?? `Error ${status}`, correlationId: 'test-id' }),
    headers: new Headers(),
  }
}

describe('ApiClient', () => {
  let authStore: ReturnType<typeof createMockAuthStore>
  let sessionStore: ReturnType<typeof createMockSessionStore>
  let client: ApiClient

  beforeEach(() => {
    vi.clearAllMocks()
    authStore = createMockAuthStore()
    sessionStore = createMockSessionStore()
    client = new ApiClient(authStore as never, sessionStore as never)
  })

  describe('fetch', () => {
    it('adds authorization header to requests', async () => {
      mockFetch.mockResolvedValue(createJsonResponse({ data: 'test' }))

      await client.fetch('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/test'),
        expect.objectContaining({
          headers: expect.any(Headers),
        })
      )
      const call = mockFetch.mock.calls[0]
      const headers = call[1].headers as Headers
      expect(headers.get('Authorization')).toBe('Bearer test-token')
    })

    it('returns parsed JSON on success', async () => {
      mockFetch.mockResolvedValue(createJsonResponse({ response: 'test-data' }))

      const result = await client.fetch<{ response: string }>('/test')

      expect(result).toEqual({ response: 'test-data' })
    })

    it('throws error when not authenticated', async () => {
      authStore.isLoggedIn = false

      await expect(client.fetch('/test')).rejects.toThrow('Not authenticated')
      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('throws parsed error on non-2xx response', async () => {
      mockFetch.mockResolvedValue(createErrorResponse(500, 'Server error'))

      await expect(client.fetch('/test')).rejects.toThrow('Server error (Correlation ID: test-id)')
    })

    it('refreshes token on 401 and retries', async () => {
      mockFetch
        .mockResolvedValueOnce(createErrorResponse(401, 'Unauthorized'))
        .mockResolvedValueOnce(createJsonResponse({ data: 'success' }))
      authStore.refreshToken.mockResolvedValue(true)

      const result = await client.fetch<{ data: string }>('/test')

      expect(authStore.refreshToken).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledTimes(2)
      expect(result).toEqual({ data: 'success' })
    })

    it('throws session expired when token refresh fails', async () => {
      mockFetch.mockResolvedValue(createErrorResponse(401, 'Unauthorized'))
      authStore.refreshToken.mockResolvedValue(false)

      await expect(client.fetch('/test')).rejects.toThrow('Session expired - please log in again')
    })

    it('uses refreshed token for retry request', async () => {
      mockFetch
        .mockResolvedValueOnce(createErrorResponse(401, 'Unauthorized'))
        .mockResolvedValueOnce(createJsonResponse({ data: 'success' }))
      authStore.refreshToken.mockImplementation(async () => {
        authStore.token = 'refreshed-token'
        return true
      })

      await client.fetch('/test')

      const secondCall = mockFetch.mock.calls[1]
      const headers = secondCall[1].headers as Headers
      expect(headers.get('Authorization')).toBe('Bearer refreshed-token')
    })
  })

  describe('fetchOrNull', () => {
    it('returns parsed JSON on success', async () => {
      mockFetch.mockResolvedValue(createJsonResponse({ response: 'test-data' }))

      const result = await client.fetchOrNull<{ response: string }>('/test')

      expect(result).toEqual({ response: 'test-data' })
    })

    it('returns null on 404', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ message: 'Not found' }),
      })

      const result = await client.fetchOrNull('/test')

      expect(result).toBeNull()
    })

    it('throws on other errors', async () => {
      mockFetch.mockResolvedValue(createErrorResponse(500, 'Server error'))

      await expect(client.fetchOrNull('/test')).rejects.toThrow('Server error')
    })

    it('refreshes token on 401 and retries', async () => {
      mockFetch
        .mockResolvedValueOnce(createErrorResponse(401, 'Unauthorized'))
        .mockResolvedValueOnce(createJsonResponse({ data: 'success' }))

      const result = await client.fetchOrNull<{ data: string }>('/test')

      expect(authStore.refreshToken).toHaveBeenCalled()
      expect(result).toEqual({ data: 'success' })
    })
  })

  describe('fetchVoid', () => {
    it('completes successfully on 200', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 200 })

      await expect(client.fetchVoid('/test', { method: 'DELETE' })).resolves.toBeUndefined()
    })

    it('completes successfully on 204', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 })

      await expect(client.fetchVoid('/test', { method: 'DELETE' })).resolves.toBeUndefined()
    })

    it('throws on error response', async () => {
      mockFetch.mockResolvedValue(createErrorResponse(500, 'Server error'))

      await expect(client.fetchVoid('/test')).rejects.toThrow('Server error')
    })

    it('refreshes token on 401 and retries', async () => {
      mockFetch
        .mockResolvedValueOnce(createErrorResponse(401, 'Unauthorized'))
        .mockResolvedValueOnce({ ok: true, status: 204 })

      await client.fetchVoid('/test', { method: 'DELETE' })

      expect(authStore.refreshToken).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledTimes(2)
    })
  })

  describe('getJournalEntry', () => {
    it('returns journal entry on success', async () => {
      const journalEntry = {
        raceId: 123,
        notes: 'Great race!',
        tags: ['sentiment:good'],
      }
      mockFetch.mockResolvedValue(createJsonResponse({ response: journalEntry }))

      const result = await client.getJournalEntry(1, 123)

      expect(result).toEqual(journalEntry)
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/driver/1/races/123/journal'),
        expect.any(Object)
      )
    })

    it('returns null when no entry exists (404)', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ message: 'Not found' }),
      })

      const result = await client.getJournalEntry(1, 123)

      expect(result).toBeNull()
    })
  })

  describe('saveJournalEntry', () => {
    it('sends PUT request with entry data', async () => {
      const savedEntry = {
        raceId: 123,
        notes: 'Great race!',
        tags: ['sentiment:good'],
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      }
      mockFetch.mockResolvedValue(createJsonResponse({ response: savedEntry }))

      const result = await client.saveJournalEntry(1, 123, {
        notes: 'Great race!',
        tags: ['sentiment:good'],
      })

      expect(result).toEqual(savedEntry)
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/driver/1/races/123/journal'),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ notes: 'Great race!', tags: ['sentiment:good'] }),
        })
      )
    })
  })

  describe('deleteJournalEntry', () => {
    it('sends DELETE request', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 })

      await client.deleteJournalEntry(1, 123)

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/driver/1/races/123/journal'),
        expect.objectContaining({ method: 'DELETE' })
      )
    })
  })

  describe('deleteDriverRaces', () => {
    it('uses fetchVoid for DELETE', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 })

      await client.deleteDriverRaces(1)

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/driver/1/races'),
        expect.objectContaining({ method: 'DELETE' })
      )
    })
  })

  describe('triggerRaceIngestion', () => {
    it('throws when session not ready', async () => {
      sessionStore.isReady = false

      await expect(client.triggerRaceIngestion()).rejects.toThrow('Session not ready')
    })

    it('returns throttled: false on success', async () => {
      mockFetch.mockResolvedValue(createJsonResponse({}))

      const result = await client.triggerRaceIngestion()

      expect(result).toEqual({ throttled: false })
    })

    it('returns throttled: true with retryAfter on 429', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 429,
        json: () => Promise.resolve({ retryAfter: 600 }),
        headers: new Headers({ 'Retry-After': '600' }),
      })

      const result = await client.triggerRaceIngestion()

      expect(result).toEqual({ throttled: true, retryAfter: 600 })
    })

    it('includes connectionId in request body', async () => {
      mockFetch.mockResolvedValue(createJsonResponse({}))

      await client.triggerRaceIngestion()

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          body: JSON.stringify({ notifyConnectionId: 'test-connection-id' }),
        })
      )
    })
  })
})