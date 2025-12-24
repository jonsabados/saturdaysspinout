import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CarCell from './CarCell.vue'
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

describe('CarCell', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('displays car name when found', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(CarCell, {
      props: { carId: 1 },
    })

    expect(wrapper.find('.car-text-full').text()).toBe('Mazda MX-5 Miata')
    expect(wrapper.find('.car-text-abbrev').text()).toBe('MX-5')
    expect(wrapper.attributes('title')).toBe('Mazda MX-5 Miata')
  })

  it('displays fallback when car not found', () => {
    mockGetCar.mockReturnValue(undefined)

    const wrapper = mount(CarCell, {
      props: { carId: 999 },
    })

    expect(wrapper.find('.car-text-full').text()).toBe('Car 999')
    expect(wrapper.find('.car-text-abbrev').text()).toBe('Car 999')
    expect(wrapper.attributes('title')).toBe('Car 999')
  })

  it('calls getCar with correct carId', () => {
    mockGetCar.mockReturnValue(mockCar)

    mount(CarCell, {
      props: { carId: 42 },
    })

    expect(mockGetCar).toHaveBeenCalledWith(42)
  })
})