import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTracksStore } from './tracks'
import type { Track } from '@/api/client'

const mockGetTracks = vi.fn()
vi.mock('@/api/client', () => ({
  useApiClient: () => ({
    getTracks: mockGetTracks,
  }),
}))

let mockIsLoggedIn = true

vi.mock('./auth', () => ({
  useAuthStore: () => ({
    get isLoggedIn() {
      return mockIsLoggedIn
    },
  }),
}))

const mockTracks: Track[] = [
  {
    id: 1,
    name: 'Daytona International Speedway',
    configName: 'Oval',
    category: 'oval',
    location: 'USA',
    cornersPerLap: 4,
    lengthMiles: 2.5,
    description: 'Famous speedway',
    logoUrl: 'https://example.com/logo.png',
    smallImageUrl: 'https://example.com/small.png',
    largeImageUrl: 'https://example.com/large.png',
    trackMapUrl: 'https://example.com/map.png',
    isDirt: false,
    isOval: true,
    hasNightLighting: true,
    rainEnabled: false,
    freeWithSubscription: true,
    retired: false,
    pitRoadSpeedLimit: 55,
  },
  {
    id: 2,
    name: 'Spa-Francorchamps',
    configName: '',
    category: 'road',
    location: 'Belgium',
    cornersPerLap: 19,
    lengthMiles: 4.35,
    description: 'Classic circuit',
    logoUrl: 'https://example.com/spa-logo.png',
    smallImageUrl: 'https://example.com/spa-small.png',
    largeImageUrl: 'https://example.com/spa-large.png',
    trackMapUrl: 'https://example.com/spa-map.png',
    isDirt: false,
    isOval: false,
    hasNightLighting: false,
    rainEnabled: true,
    freeWithSubscription: false,
    retired: false,
    pitRoadSpeedLimit: 60,
  },
]

describe('tracks store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockIsLoggedIn = true
  })

  describe('getTrack', () => {
    it('returns undefined when track not found', () => {
      const store = useTracksStore()

      expect(store.getTrack(999)).toBeUndefined()
    })

    it('returns track when found', async () => {
      mockGetTracks.mockResolvedValue({ response: mockTracks, correlationId: 'test' })
      const store = useTracksStore()

      await store.fetchTracks()

      expect(store.getTrack(1)).toEqual(mockTracks[0])
      expect(store.getTrack(2)).toEqual(mockTracks[1])
    })
  })

  describe('fetchTracks', () => {
    it('populates tracks map from API response', async () => {
      mockGetTracks.mockResolvedValue({ response: mockTracks, correlationId: 'test' })
      const store = useTracksStore()

      await store.fetchTracks()

      expect(store.isLoaded).toBe(true)
      expect(store.tracks.size).toBe(2)
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })

    it('sets error state on API failure', async () => {
      mockGetTracks.mockRejectedValue(new Error('API failed'))
      const store = useTracksStore()

      await store.fetchTracks()

      expect(store.isLoaded).toBe(false)
      expect(store.error).toBe('API failed')
      expect(store.loading).toBe(false)
    })

    it('handles non-Error rejection', async () => {
      mockGetTracks.mockRejectedValue('string error')
      const store = useTracksStore()

      await store.fetchTracks()

      expect(store.error).toBe('Failed to fetch tracks')
    })

    it('does not fetch if already loading', async () => {
      let resolveFirst: (value: unknown) => void
      const firstCall = new Promise((resolve) => {
        resolveFirst = resolve
      })
      mockGetTracks.mockImplementationOnce(() => firstCall)

      const store = useTracksStore()
      const promise1 = store.fetchTracks()
      const promise2 = store.fetchTracks()

      expect(mockGetTracks).toHaveBeenCalledTimes(1)

      resolveFirst!({ response: mockTracks, correlationId: 'test' })
      await promise1
      await promise2
    })
  })

  describe('clear', () => {
    it('resets tracks and error', async () => {
      mockGetTracks.mockResolvedValue({ response: mockTracks, correlationId: 'test' })
      const store = useTracksStore()

      await store.fetchTracks()
      expect(store.isLoaded).toBe(true)

      store.clear()

      expect(store.isLoaded).toBe(false)
      expect(store.tracks.size).toBe(0)
      expect(store.error).toBeNull()
    })
  })

  describe('isLoaded', () => {
    it('returns false when tracks map is empty', () => {
      const store = useTracksStore()

      expect(store.isLoaded).toBe(false)
    })

    it('returns true when tracks are loaded', async () => {
      mockGetTracks.mockResolvedValue({ response: mockTracks, correlationId: 'test' })
      const store = useTracksStore()

      await store.fetchTracks()

      expect(store.isLoaded).toBe(true)
    })
  })
})