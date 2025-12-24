import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TrackCell from './TrackCell.vue'
import type { Track } from '@/api/client'

const mockTrack: Track = {
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
}

const mockGetTrack = vi.fn()
vi.mock('@/stores/tracks', () => ({
  useTracksStore: () => ({
    getTrack: mockGetTrack,
  }),
}))

describe('TrackCell', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('displays track name with config when both exist', () => {
    mockGetTrack.mockReturnValue(mockTrack)

    const wrapper = mount(TrackCell, {
      props: { trackId: 1 },
    })

    expect(wrapper.text()).toBe('Daytona International Speedway - Oval')
    expect(wrapper.attributes('title')).toBe('Daytona International Speedway - Oval')
  })

  it('displays track name only when no config', () => {
    mockGetTrack.mockReturnValue({ ...mockTrack, configName: '' })

    const wrapper = mount(TrackCell, {
      props: { trackId: 1 },
    })

    expect(wrapper.text()).toBe('Daytona International Speedway')
  })

  it('displays fallback when track not found', () => {
    mockGetTrack.mockReturnValue(undefined)

    const wrapper = mount(TrackCell, {
      props: { trackId: 999 },
    })

    expect(wrapper.text()).toBe('Track 999')
    expect(wrapper.attributes('title')).toBe('Track 999')
  })

  it('calls getTrack with correct trackId', () => {
    mockGetTrack.mockReturnValue(mockTrack)

    mount(TrackCell, {
      props: { trackId: 42 },
    })

    expect(mockGetTrack).toHaveBeenCalledWith(42)
  })
})