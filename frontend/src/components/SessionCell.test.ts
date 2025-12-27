import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SessionCell from './SessionCell.vue'
import type { Car } from '@/api/client'

const mockCar: Car = {
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
  logoUrl: 'https://example.com/logo.png',
  smallImageUrl: 'https://example.com/small.png',
  largeImageUrl: 'https://example.com/large.png',
  hasHeadlights: true,
  hasMultipleDryTires: false,
  rainEnabled: true,
  freeWithSubscription: true,
  retired: false,
}

const mockGetCar = vi.fn()
vi.mock('@/stores/cars', () => ({
  useCarsStore: () => ({
    getCar: mockGetCar,
  }),
}))

describe('SessionCell', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('displays series name and car name when car is found', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(SessionCell, {
      props: { seriesName: 'Advanced Mazda MX-5 Cup Series', carId: 1 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.series-name').text()).toBe('Advanced Mazda MX-5 Cup Series')
    expect(wrapper.find('.car-name').text()).toBe('Mazda MX-5 Miata')
    expect(wrapper.find('.session-text-abbrev').text()).toBe('MX-5')
    expect(wrapper.attributes('title')).toBe('Advanced Mazda MX-5 Cup Series - Mazda MX-5 Miata')
  })

  it('displays fallback car name when car not found', () => {
    mockGetCar.mockReturnValue(undefined)

    const wrapper = mount(SessionCell, {
      props: { seriesName: 'Some Series', carId: 999 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.series-name').text()).toBe('Some Series')
    expect(wrapper.find('.car-name').text()).toBe('Car 999')
    expect(wrapper.find('.session-text-abbrev').text()).toBe('Car 999')
    expect(wrapper.attributes('title')).toBe('Some Series - Car 999')
  })

  it('calls getCar with correct carId', () => {
    mockGetCar.mockReturnValue(mockCar)

    mount(SessionCell, {
      props: { seriesName: 'Test Series', carId: 42 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(mockGetCar).toHaveBeenCalledWith(42)
  })

  it('links car name to car details page', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(SessionCell, {
      props: { seriesName: 'Test Series', carId: 123 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    const links = wrapper.findAllComponents(RouterLinkStub)
    expect(links.length).toBe(2) // One for .car-name, one for .session-text-abbrev
    expect(links[0].props('to')).toEqual({ name: 'car-details', params: { id: 123 } })
  })
})