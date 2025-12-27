import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CarCell from './CarCell.vue'
import type { Car } from '@/api/client'

const mockCar: Car = {
  id: 1,
  name: 'Porsche 911 GT3 R',
  nameAbbreviated: 'Porsche 911',
  make: 'Porsche',
  model: '911 GT3 R',
  description: 'A fast GT3 car',
  weight: 2866,
  hpUnderHood: 550,
  hpActual: 520,
  categories: ['road', 'gt3'],
  logoUrl: 'https://example.com/logo.png',
  smallImageUrl: 'https://example.com/small.png',
  largeImageUrl: 'https://example.com/large.png',
  hasHeadlights: true,
  hasMultipleDryTires: true,
  rainEnabled: true,
  freeWithSubscription: false,
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

  it('displays car name with make/model when both exist', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(CarCell, {
      props: { carId: 1 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.car-name').text()).toBe('Porsche 911 GT3 R')
    expect(wrapper.find('.car-make-model').text()).toBe('Porsche 911 GT3 R')
    expect(wrapper.attributes('title')).toBe('Porsche 911 GT3 R')
  })

  it('displays car name only when no make/model', () => {
    mockGetCar.mockReturnValue({ ...mockCar, make: '', model: '' })

    const wrapper = mount(CarCell, {
      props: { carId: 1 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.car-name').text()).toBe('Porsche 911 GT3 R')
    expect(wrapper.find('.car-make-model').exists()).toBe(false)
  })

  it('displays fallback when car not found', () => {
    mockGetCar.mockReturnValue(undefined)

    const wrapper = mount(CarCell, {
      props: { carId: 999 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.car-name').text()).toBe('Car 999')
    expect(wrapper.find('.car-make-model').exists()).toBe(false)
    expect(wrapper.attributes('title')).toBe('Car 999')
  })

  it('calls getCar with correct carId', () => {
    mockGetCar.mockReturnValue(mockCar)

    mount(CarCell, {
      props: { carId: 42 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(mockGetCar).toHaveBeenCalledWith(42)
  })

  it('links to car details page with correct id', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(CarCell, {
      props: { carId: 123 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    const link = wrapper.findComponent(RouterLinkStub)
    expect(link.props('to')).toEqual({ name: 'car-details', params: { id: 123 } })
  })

  it('uses abbreviated name on mobile view', () => {
    mockGetCar.mockReturnValue(mockCar)

    const wrapper = mount(CarCell, {
      props: { carId: 1 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.car-text-abbrev').text()).toBe('Porsche 911')
  })

  it('falls back to full name when abbreviated name is empty', () => {
    mockGetCar.mockReturnValue({ ...mockCar, nameAbbreviated: '' })

    const wrapper = mount(CarCell, {
      props: { carId: 1 },
      global: {
        stubs: { RouterLink: RouterLinkStub },
      },
    })

    expect(wrapper.find('.car-text-abbrev').text()).toBe('Porsche 911 GT3 R')
  })
})