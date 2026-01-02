import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import JournalFilters, { type JournalFiltersState } from './JournalFilters.vue'
import { i18n } from '@/i18n'

describe('JournalFilters', () => {
  const defaultFilters: JournalFiltersState = {
    from: new Date('2024-01-01'),
    to: new Date('2024-01-31'),
    sentiment: null,
    showDNFOnly: false,
  }

  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountComponent(modelValue: JournalFiltersState = defaultFilters) {
    return mount(JournalFilters, {
      props: { modelValue },
      global: {
        plugins: [i18n],
      },
    })
  }

  it('displays date inputs with correct values', () => {
    const wrapper = mountComponent()

    const inputs = wrapper.findAll('input[type="date"]')
    expect(inputs).toHaveLength(2)
    expect((inputs[0].element as HTMLInputElement).value).toBe('2024-01-01')
    expect((inputs[1].element as HTMLInputElement).value).toBe('2024-01-31')
  })

  it('emits update when from date changes', async () => {
    const wrapper = mountComponent()

    const fromInput = wrapper.findAll('input[type="date"]')[0]
    await fromInput.setValue('2024-02-01')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.from.toISOString()).toContain('2024-02-01')
  })

  it('emits update when to date changes', async () => {
    const wrapper = mountComponent()

    const toInput = wrapper.findAll('input[type="date"]')[1]
    await toInput.setValue('2024-02-28')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.to.toISOString()).toContain('2024-02-28')
  })

  it('displays sentiment chips', () => {
    const wrapper = mountComponent()

    const chips = wrapper.findAll('.sentiment-chip')
    expect(chips).toHaveLength(3)
    expect(chips[0].text()).toBe('Good')
    expect(chips[1].text()).toBe('Okay')
    expect(chips[2].text()).toBe('Rough')
  })

  it('sentiment chips are not selected by default', () => {
    const wrapper = mountComponent()

    const chips = wrapper.findAll('.sentiment-chip')
    chips.forEach(chip => {
      expect(chip.classes()).not.toContain('selected')
    })
  })

  it('clicking sentiment chip selects it', async () => {
    const wrapper = mountComponent()

    const goodChip = wrapper.findAll('.sentiment-chip')[0]
    await goodChip.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.sentiment).toEqual(['good'])
  })

  it('clicking selected sentiment chip deselects it', async () => {
    const filters = { ...defaultFilters, sentiment: ['good'] as ('good' | 'neutral' | 'bad')[] }
    const wrapper = mountComponent(filters)

    const goodChip = wrapper.findAll('.sentiment-chip')[0]
    await goodChip.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.sentiment).toBeNull()
  })

  it('can select multiple sentiments', async () => {
    const filters = { ...defaultFilters, sentiment: ['good'] as ('good' | 'neutral' | 'bad')[] }
    const wrapper = mountComponent(filters)

    const badChip = wrapper.findAll('.sentiment-chip')[2]
    await badChip.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.sentiment).toEqual(['good', 'bad'])
  })

  it('DNF checkbox is unchecked by default', () => {
    const wrapper = mountComponent()

    const checkbox = wrapper.find('input[type="checkbox"]')
    expect((checkbox.element as HTMLInputElement).checked).toBe(false)
  })

  it('clicking DNF checkbox emits update', async () => {
    const wrapper = mountComponent()

    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(true)

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    const newState = emitted![0][0] as JournalFiltersState
    expect(newState.showDNFOnly).toBe(true)
  })

  it('shows selected state for sentiment chips', () => {
    const filters = { ...defaultFilters, sentiment: ['good', 'bad'] as ('good' | 'neutral' | 'bad')[] }
    const wrapper = mountComponent(filters)

    const chips = wrapper.findAll('.sentiment-chip')
    expect(chips[0].classes()).toContain('selected') // good
    expect(chips[1].classes()).not.toContain('selected') // neutral
    expect(chips[2].classes()).toContain('selected') // bad
  })
})