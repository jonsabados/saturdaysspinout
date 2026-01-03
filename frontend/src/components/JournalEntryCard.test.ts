import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import JournalEntryCard from './JournalEntryCard.vue'
import SentimentBadge from './SentimentBadge.vue'
import type { JournalEntry } from '@/api/client'
import { i18n } from '@/i18n'
import { useTracksStore } from '@/stores/tracks'

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

describe('JournalEntryCard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // Mock tracks store
    const tracksStore = useTracksStore()
    vi.spyOn(tracksStore, 'getTrack').mockReturnValue({
      id: 167,
      name: 'Road Atlanta',
      configName: 'Full Course',
      category: 'road',
      location: 'Georgia, USA',
      cornersPerLap: 12,
      lengthMiles: 2.54,
      description: '',
      logoUrl: '',
      smallImageUrl: '',
      largeImageUrl: '',
      trackMapUrl: '',
      trackMapLayers: {
        background: '',
        inactive: '',
        active: '',
        pitroad: '',
        startFinish: '',
        turns: '',
      },
      isDirt: false,
      isOval: false,
      hasNightLighting: false,
      rainEnabled: false,
      freeWithSubscription: true,
      retired: false,
      pitRoadSpeedLimit: 45,
    })
  })

  function mountComponent(entry: JournalEntry = mockEntry) {
    return mount(JournalEntryCard, {
      props: { entry },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
    })
  }

  it('displays series name', () => {
    const wrapper = mountComponent()
    expect(wrapper.find('.series-name').text()).toBe('Advanced Mazda MX-5 Cup Series')
  })

  it('displays track name from store', () => {
    const wrapper = mountComponent()
    expect(wrapper.find('.track-name').text()).toBe('@ Road Atlanta - Full Course')
  })

  it('displays sentiment badge', () => {
    const wrapper = mountComponent()
    const badge = wrapper.findComponent(SentimentBadge)
    expect(badge.exists()).toBe(true)
    expect(badge.props('sentiment')).toBe('good')
  })

  it('displays position change (1-indexed from 0-indexed API data)', () => {
    const wrapper = mountComponent()
    // API returns 0-indexed positions (5, 2), display adds 1 for human-readable (P6, P3)
    expect(wrapper.find('.stat.position').text()).toBe('P6 → P3')
  })

  it('displays incidents', () => {
    const wrapper = mountComponent()
    expect(wrapper.find('.stat.incidents').text()).toBe('2x')
  })

  it('displays iRating change with positive styling', () => {
    const wrapper = mountComponent()
    const irating = wrapper.find('.stat.irating')
    expect(irating.text()).toBe('+50 iR')
    expect(irating.classes()).toContain('positive')
  })

  it('displays SR change with positive styling', () => {
    const wrapper = mountComponent()
    const sr = wrapper.find('.stat.sr')
    expect(sr.text()).toBe('+0.14 SR')
    expect(sr.classes()).toContain('positive')
  })

  it('displays full notes without truncation', () => {
    const wrapper = mountComponent()
    expect(wrapper.find('.notes').text()).toBe(
      'Great race! Managed to stay clean and work through the field.'
    )
  })

  it('displays long notes in full', () => {
    const longNotes = 'A'.repeat(200)
    const entry = { ...mockEntry, notes: longNotes }
    const wrapper = mountComponent(entry)
    expect(wrapper.find('.notes').text()).toBe(longNotes)
  })

  it('shows DNF badge when reasonOut is not Running', () => {
    const entry = {
      ...mockEntry,
      race: { ...mockEntry.race, reasonOut: 'Disqualified' },
    }
    const wrapper = mountComponent(entry)
    expect(wrapper.find('.badge-dnf').exists()).toBe(true)
    expect(wrapper.find('.badge-dnf').text()).toBe('DNF')
  })

  it('does not show DNF badge when reasonOut is Running', () => {
    const wrapper = mountComponent()
    expect(wrapper.find('.badge-dnf').exists()).toBe(false)
  })

  it('shows promoted badge when license level increased', () => {
    const entry = {
      ...mockEntry,
      race: { ...mockEntry.race, oldLicenseLevel: 8, newLicenseLevel: 9 },
    }
    const wrapper = mountComponent(entry)
    expect(wrapper.find('.badge-promoted').exists()).toBe(true)
  })

  it('shows demoted badge when license level decreased', () => {
    const entry = {
      ...mockEntry,
      race: { ...mockEntry.race, oldLicenseLevel: 9, newLicenseLevel: 8 },
    }
    const wrapper = mountComponent(entry)
    expect(wrapper.find('.badge-demoted').exists()).toBe(true)
  })

  it('displays negative iRating change with negative styling', () => {
    const entry = {
      ...mockEntry,
      race: { ...mockEntry.race, oldIrating: 1900, newIrating: 1850 },
    }
    const wrapper = mountComponent(entry)
    const irating = wrapper.find('.stat.irating')
    expect(irating.text()).toBe('-50 iR')
    expect(irating.classes()).toContain('negative')
  })
})