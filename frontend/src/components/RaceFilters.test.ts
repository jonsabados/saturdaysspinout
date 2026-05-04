import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RaceFilters, {
  type RaceFiltersState,
  type RaceFiltersDimensions,
} from './RaceFilters.vue'
import { i18n } from '@/i18n'

const mockGetCar = vi.fn()
const mockGetSeries = vi.fn()
const mockGetTrack = vi.fn()

vi.mock('@/stores/cars', () => ({
  useCarsStore: () => ({ getCar: mockGetCar }),
}))
vi.mock('@/stores/series', () => ({
  useSeriesStore: () => ({ getSeries: mockGetSeries }),
}))
vi.mock('@/stores/tracks', () => ({
  useTracksStore: () => ({ getTrack: mockGetTrack }),
}))

describe('RaceFilters', () => {
  const defaultState: RaceFiltersState = {
    from: new Date('2024-01-01'),
    to: new Date('2024-01-31'),
    discipline: null,
    seriesIds: [],
    carIds: [],
    trackIds: [],
  }

  const defaultDimensions: RaceFiltersDimensions = {
    series: [101, 102],
    cars: [1, 2],
    tracks: [501, 502],
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()

    mockGetCar.mockImplementation((id: number) => {
      const cars: Record<number, { name: string; categories: string[] }> = {
        1: { name: 'Mazda MX-5', categories: ['sports_car'] },
        2: { name: 'Sprint Car', categories: ['oval'] },
        3: { name: 'Late Model', categories: ['oval'] },
      }
      return cars[id]
    })
    mockGetSeries.mockImplementation((id: number) => {
      const series: Record<number, { name: string; category: string }> = {
        101: { name: 'MX-5 Cup', category: 'sports_car' },
        102: { name: 'Sprint Series', category: 'oval' },
        103: { name: 'Late Model Series', category: 'oval' },
      }
      return series[id]
    })
    mockGetTrack.mockImplementation((id: number) => {
      const tracks: Record<number, { name: string }> = {
        501: { name: 'Daytona' },
        502: { name: 'Bristol' },
      }
      return tracks[id]
    })
  })

  function mountComponent(
    overrides: Partial<{
      modelValue: RaceFiltersState
      dimensions: RaceFiltersDimensions | null
      disabled: boolean
    }> = {}
  ) {
    return mount(RaceFilters, {
      props: {
        modelValue: overrides.modelValue ?? defaultState,
        dimensions: overrides.dimensions === undefined ? defaultDimensions : overrides.dimensions,
        disabled: overrides.disabled ?? false,
      },
      global: { plugins: [i18n] },
    })
  }

  describe('date inputs', () => {
    it('renders from/to dates with correct values', () => {
      const wrapper = mountComponent()
      const inputs = wrapper.findAll('input[type="date"]')
      expect(inputs).toHaveLength(2)
      expect((inputs[0].element as HTMLInputElement).value).toBe('2024-01-01')
      expect((inputs[1].element as HTMLInputElement).value).toBe('2024-01-31')
    })

    it('emits update when from date changes', async () => {
      const wrapper = mountComponent()
      await wrapper.findAll('input[type="date"]')[0].setValue('2024-02-15')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.from.toISOString()).toContain('2024-02-15')
      expect(next.to).toEqual(defaultState.to)
    })

    it('emits update when to date changes', async () => {
      const wrapper = mountComponent()
      await wrapper.findAll('input[type="date"]')[1].setValue('2024-03-01')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.to.toISOString()).toContain('2024-03-01')
      expect(next.from).toEqual(defaultState.from)
    })
  })

  describe('multi-select filters', () => {
    it('renders all three filter groups when dimensions present', () => {
      const wrapper = mountComponent()
      const labels = wrapper.findAll('.label-text').map((l) => l.text())
      expect(labels).toContain('Series')
      expect(labels).toContain('Car')
      expect(labels).toContain('Track')
    })

    it('hides series filter group when no series in dimensions', () => {
      const wrapper = mountComponent({
        dimensions: { ...defaultDimensions, series: [] },
      })
      const labels = wrapper.findAll('.label-text').map((l) => l.text())
      expect(labels).not.toContain('Series')
      expect(labels).toContain('Car')
      expect(labels).toContain('Track')
    })

    it('hides all filter groups when dimensions is null', () => {
      const wrapper = mountComponent({ dimensions: null })
      const labels = wrapper.findAll('.label-text').map((l) => l.text())
      expect(labels).not.toContain('Series')
      expect(labels).not.toContain('Car')
      expect(labels).not.toContain('Track')
    })

    it('emits update adding series id when option selected', async () => {
      const wrapper = mountComponent()
      // selects in order: discipline, series, car, track
      const seriesSelect = wrapper.findAll('select')[1]
      await seriesSelect.setValue('101')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.seriesIds).toEqual([101])
    })

    it('emits update removing series id when chip clicked', async () => {
      const wrapper = mountComponent({
        modelValue: { ...defaultState, seriesIds: [101, 102] },
      })
      const chips = wrapper.findAll('.filter-chip')
      await chips[0].trigger('click')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.seriesIds).toEqual([102])
    })

    it('shows checkmark in option when id is already selected', () => {
      const wrapper = mountComponent({
        modelValue: { ...defaultState, seriesIds: [101] },
      })
      const seriesOptions = wrapper.findAll('select')[1].findAll('option')
      const selectedOption = seriesOptions.find((o) => o.text().includes('MX-5 Cup'))!
      expect(selectedOption.text()).toContain('✓')
    })

    it('renders chips for active car filter', () => {
      const wrapper = mountComponent({
        modelValue: { ...defaultState, carIds: [1] },
      })
      const chips = wrapper.findAll('.filter-chip')
      expect(chips).toHaveLength(1)
      expect(chips[0].text()).toContain('Mazda MX-5')
    })

    it('emits update toggling track id', async () => {
      const wrapper = mountComponent()
      const selects = wrapper.findAll('select')
      const trackSelect = selects[selects.length - 1]
      await trackSelect.setValue('502')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.trackIds).toEqual([502])
    })

    it('sorts options alphabetically by name', () => {
      const wrapper = mountComponent({
        dimensions: { series: [102, 101], cars: [], tracks: [] },
      })
      const options = wrapper.findAll('select')[0].findAll('option')
      // first option is the placeholder; series options follow
      const seriesNames = options.slice(1).map((o) => o.text().trim())
      expect(seriesNames).toEqual(['MX-5 Cup', 'Sprint Series'])
    })
  })

  describe('discipline filter', () => {
    it('hides discipline select when only one discipline available', () => {
      const wrapper = mountComponent({
        dimensions: { series: [], cars: [1], tracks: [] }, // single car → single discipline
      })
      expect(wrapper.find('.discipline-select').exists()).toBe(false)
    })

    it('shows discipline select when 2+ disciplines available', () => {
      const wrapper = mountComponent()
      // cars 1 (sports_car) + 2 (oval) → 2 disciplines
      expect(wrapper.find('.discipline-select').exists()).toBe(true)
    })

    it('cascades discipline=oval to filtered cars and series, clears trackIds', async () => {
      const wrapper = mountComponent({
        modelValue: { ...defaultState, trackIds: [501] },
        dimensions: { series: [101, 102, 103], cars: [1, 2, 3], tracks: [501] },
      })
      await wrapper.find('.discipline-select').setValue('oval')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.discipline).toBe('oval')
      expect(next.carIds).toEqual([2, 3])
      expect(next.seriesIds).toEqual([102, 103])
      expect(next.trackIds).toEqual([])
    })

    it('clears all filters when discipline cleared back to "all"', async () => {
      const wrapper = mountComponent({
        modelValue: {
          ...defaultState,
          discipline: 'oval',
          seriesIds: [102],
          carIds: [2],
          trackIds: [501],
        },
      })
      await wrapper.find('.discipline-select').setValue('')
      const emitted = wrapper.emitted('update:modelValue')!
      const next = emitted[0][0] as RaceFiltersState
      expect(next.discipline).toBeNull()
      expect(next.seriesIds).toEqual([])
      expect(next.carIds).toEqual([])
      expect(next.trackIds).toEqual([])
    })

    it('reflects current discipline value', () => {
      const wrapper = mountComponent({
        modelValue: { ...defaultState, discipline: 'sports_car' },
      })
      const select = wrapper.find('.discipline-select')
      expect((select.element as HTMLSelectElement).value).toBe('sports_car')
    })
  })

  describe('disabled state', () => {
    it('disables all inputs when disabled prop is true', () => {
      const wrapper = mountComponent({ disabled: true })
      const inputs = [
        ...wrapper.findAll('input[type="date"]').map((i) => i.element as HTMLInputElement),
        ...wrapper.findAll('select').map((s) => s.element as HTMLSelectElement),
      ]
      for (const el of inputs) {
        expect(el.disabled).toBe(true)
      }
    })
  })
})