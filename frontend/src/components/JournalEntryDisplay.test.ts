import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import JournalEntryDisplay from './JournalEntryDisplay.vue'
import type { JournalEntry } from '@/api/client'
import { i18n } from '@/i18n'

const mockEntry: JournalEntry = {
  raceId: 123456789,
  createdAt: '2024-01-15T14:30:00Z',
  updatedAt: '2024-01-15T15:00:00Z',
  notes: 'Great race! Managed to stay clean and work through the field.',
  tags: ['sentiment:good', 'podium'],
  race: {
    id: 123456789,
    subsessionId: 12345678,
    trackId: 167,
    carId: 67,
    seriesId: 231,
    seriesName: 'Advanced Mazda MX-5 Cup Series',
    startTime: '2024-01-15T14:30:00Z',
    startPosition: 5,
    startPositionInClass: 5,
    finishPosition: 2,
    finishPositionInClass: 2,
    incidents: 2,
    reasonOut: 'Running',
    oldIrating: 1850,
    newIrating: 1900,
    oldSubLevel: 385,
    newSubLevel: 399,
    oldLicenseLevel: 8,
    newLicenseLevel: 8,
    oldCpi: 1.42,
    newCpi: 1.45,
  },
}

describe('JournalEntryDisplay', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  function mountComponent(entry: JournalEntry = mockEntry) {
    return mount(JournalEntryDisplay, {
      props: { entry },
      global: {
        plugins: [i18n],
      },
    })
  }

  it('displays journal notes', () => {
    const wrapper = mountComponent()

    expect(wrapper.find('.journal-notes').text()).toBe(
      'Great race! Managed to stay clean and work through the field.'
    )
  })

  it('displays good sentiment badge', () => {
    const wrapper = mountComponent()

    const badge = wrapper.find('.sentiment-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('sentiment-good')
    expect(badge.find('.sentiment-icon').text()).toBe('+')
    expect(badge.find('.sentiment-text').text()).toBe('Good')
  })

  it('displays neutral sentiment badge', () => {
    const entry = { ...mockEntry, tags: ['sentiment:neutral'] }
    const wrapper = mountComponent(entry)

    const badge = wrapper.find('.sentiment-badge')
    expect(badge.classes()).toContain('sentiment-neutral')
    expect(badge.find('.sentiment-icon').text()).toBe('=')
    expect(badge.find('.sentiment-text').text()).toBe('Okay')
  })

  it('displays bad sentiment badge', () => {
    const entry = { ...mockEntry, tags: ['sentiment:bad'] }
    const wrapper = mountComponent(entry)

    const badge = wrapper.find('.sentiment-badge')
    expect(badge.classes()).toContain('sentiment-bad')
    expect(badge.find('.sentiment-icon').text()).toBe('-')
    expect(badge.find('.sentiment-text').text()).toBe('Rough')
  })

  it('does not display sentiment badge when no sentiment tag', () => {
    const entry = { ...mockEntry, tags: ['podium'] }
    const wrapper = mountComponent(entry)

    expect(wrapper.find('.sentiment-badge').exists()).toBe(false)
  })

  it('displays updated date', () => {
    const wrapper = mountComponent()

    const updated = wrapper.find('.journal-updated')
    expect(updated.exists()).toBe(true)
    expect(updated.text()).toContain('Updated')
  })

  it('does not display notes paragraph when notes are empty', () => {
    const entry = { ...mockEntry, notes: '' }
    const wrapper = mountComponent(entry)

    expect(wrapper.find('.journal-notes').exists()).toBe(false)
  })

  it('emits edit event when edit button is clicked', async () => {
    const wrapper = mountComponent()

    await wrapper.find('.btn-secondary').trigger('click')

    expect(wrapper.emitted('edit')).toEqual([[]])
  })

  it('emits delete event when delete is confirmed', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountComponent()

    await wrapper.find('.btn-danger').trigger('click')

    expect(window.confirm).toHaveBeenCalled()
    expect(wrapper.emitted('delete')).toEqual([[]])
  })

  it('does not emit delete event when delete is cancelled', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const wrapper = mountComponent()

    await wrapper.find('.btn-danger').trigger('click')

    expect(window.confirm).toHaveBeenCalled()
    expect(wrapper.emitted('delete')).toBeUndefined()
  })

  it('has edit and delete buttons', () => {
    const wrapper = mountComponent()

    expect(wrapper.find('.btn-secondary').text()).toBe('Edit')
    expect(wrapper.find('.btn-danger').text()).toBe('Delete')
  })
})