import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCarsStore } from './cars'
import type { Car } from '@/api/client'

const mockGetCars = vi.fn()
vi.mock('@/api/client', () => ({
  useApiClient: () => ({
    getCars: mockGetCars,
  }),
}))

let mockSessionIsReady = false
let mockSessionIsLoggedIn = true
const sessionWatchers: Array<(value: boolean) => void> = []

vi.mock('./session', () => ({
  useSessionStore: () => ({
    get isReady() {
      return mockSessionIsReady
    },
    get isLoggedIn() {
      return mockSessionIsLoggedIn
    },
  }),
}))

const mockCars: Car[] = [
  {
    id: 1,
    name: 'Mazda MX-5 Miata',
    nameAbbreviated: 'MX-5',
    make: 'Mazda',
    model: 'MX-5 Miata',
    description: 'Perfect starter car',
    weight: 2332,
    hpUnderHood: 155,
    hpActual: 155,
    categories: ['road'],
    logoUrl: 'https://example.com/mazda-logo.png',
    smallImageUrl: 'https://example.com/mazda-small.png',
    largeImageUrl: 'https://example.com/mazda-large.png',
    hasHeadlights: true,
    hasMultipleDryTires: false,
    rainEnabled: true,
    freeWithSubscription: true,
    retired: false,
  },
  {
    id: 2,
    name: 'NASCAR Cup Series Chevrolet Camaro ZL1',
    nameAbbreviated: 'Cup Camaro',
    make: 'Chevrolet',
    model: 'Camaro ZL1',
    description: 'Stock car racing',
    weight: 3200,
    hpUnderHood: 670,
    hpActual: 670,
    categories: ['oval'],
    logoUrl: 'https://example.com/chevy-logo.png',
    smallImageUrl: 'https://example.com/chevy-small.png',
    largeImageUrl: 'https://example.com/chevy-large.png',
    hasHeadlights: false,
    hasMultipleDryTires: true,
    rainEnabled: false,
    freeWithSubscription: false,
    retired: false,
  },
]

describe('cars store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockSessionIsReady = false
    mockSessionIsLoggedIn = true
    sessionWatchers.length = 0
  })

  describe('getCar', () => {
    it('returns undefined when car not found', () => {
      const store = useCarsStore()

      expect(store.getCar(999)).toBeUndefined()
    })

    it('returns car when found', async () => {
      mockGetCars.mockResolvedValue({ response: mockCars, correlationId: 'test' })
      const store = useCarsStore()

      await store.fetchCars()

      expect(store.getCar(1)).toEqual(mockCars[0])
      expect(store.getCar(2)).toEqual(mockCars[1])
    })
  })

  describe('fetchCars', () => {
    it('populates cars map from API response', async () => {
      mockGetCars.mockResolvedValue({ response: mockCars, correlationId: 'test' })
      const store = useCarsStore()

      await store.fetchCars()

      expect(store.isLoaded).toBe(true)
      expect(store.cars.size).toBe(2)
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })

    it('sets error state on API failure', async () => {
      mockGetCars.mockRejectedValue(new Error('API failed'))
      const store = useCarsStore()

      await store.fetchCars()

      expect(store.isLoaded).toBe(false)
      expect(store.error).toBe('API failed')
      expect(store.loading).toBe(false)
    })

    it('handles non-Error rejection', async () => {
      mockGetCars.mockRejectedValue('string error')
      const store = useCarsStore()

      await store.fetchCars()

      expect(store.error).toBe('Failed to fetch cars')
    })

    it('does not fetch if already loading', async () => {
      let resolveFirst: (value: unknown) => void
      const firstCall = new Promise((resolve) => {
        resolveFirst = resolve
      })
      mockGetCars.mockImplementationOnce(() => firstCall)

      const store = useCarsStore()
      const promise1 = store.fetchCars()
      const promise2 = store.fetchCars()

      expect(mockGetCars).toHaveBeenCalledTimes(1)

      resolveFirst!({ response: mockCars, correlationId: 'test' })
      await promise1
      await promise2
    })
  })

  describe('clear', () => {
    it('resets cars and error', async () => {
      mockGetCars.mockResolvedValue({ response: mockCars, correlationId: 'test' })
      const store = useCarsStore()

      await store.fetchCars()
      expect(store.isLoaded).toBe(true)

      store.clear()

      expect(store.isLoaded).toBe(false)
      expect(store.cars.size).toBe(0)
      expect(store.error).toBeNull()
    })
  })

  describe('isLoaded', () => {
    it('returns false when cars map is empty', () => {
      const store = useCarsStore()

      expect(store.isLoaded).toBe(false)
    })

    it('returns true when cars are loaded', async () => {
      mockGetCars.mockResolvedValue({ response: mockCars, correlationId: 'test' })
      const store = useCarsStore()

      await store.fetchCars()

      expect(store.isLoaded).toBe(true)
    })
  })
})