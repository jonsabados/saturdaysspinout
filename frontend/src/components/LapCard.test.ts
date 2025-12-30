import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LapCard from './LapCard.vue'
import type { LapData, Lap } from '@/api/client'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function createLap(overrides: Partial<Lap> = {}): Lap {
  return {
    lapNumber: 1,
    flags: 0,
    incident: false,
    sessionTime: 60000,
    lapTime: 90000, // 1:30.000
    personalBestLap: false,
    lapEvents: [],
    ...overrides,
  }
}

function createLapData(overrides: Partial<LapData> = {}): LapData {
  return {
    bestLapNum: 1,
    bestLapTime: 90000,
    bestNlapsNum: -1,
    bestNlapsTime: -1,
    bestQualLapNum: -1,
    bestQualLapTime: -1,
    bestQualLapAt: '0001-01-01T00:00:00Z',
    custId: 12345,
    name: 'Test Driver',
    carId: 1,
    licenseLevel: 20,
    laps: [createLap()],
    ...overrides,
  }
}

describe('LapCard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('displays driver name and position correctly', () => {
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Max Verstappen',
        finishPosition: 0, // P1
        lapData: createLapData(),
      },
    })

    expect(wrapper.find('.lap-card-title').text()).toBe('P1 - Max Verstappen')
  })

  it('displays correct position for non-first place', () => {
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Lewis Hamilton',
        finishPosition: 4, // P5
        lapData: createLapData(),
      },
    })

    expect(wrapper.find('.lap-card-title').text()).toBe('P5 - Lewis Hamilton')
  })

  it('displays lap count in summary', () => {
    const laps = [createLap({ lapNumber: 1 }), createLap({ lapNumber: 2 }), createLap({ lapNumber: 3 })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    expect(wrapper.find('.lap-card-summary').text()).toContain('3')
  })

  it('renders lap table with correct lap numbers', () => {
    const laps = [
      createLap({ lapNumber: 1, lapTime: 90000 }),
      createLap({ lapNumber: 2, lapTime: 89000 }),
      createLap({ lapNumber: 3, lapTime: 91000 }),
    ]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(3)
    expect(rows[0].find('.col-lap-num').text()).toBe('1')
    expect(rows[1].find('.col-lap-num').text()).toBe('2')
    expect(rows[2].find('.col-lap-num').text()).toBe('3')
  })

  it('shows PB badge for personal best lap', () => {
    const laps = [
      createLap({ lapNumber: 1, personalBestLap: false }),
      createLap({ lapNumber: 2, personalBestLap: true }),
      createLap({ lapNumber: 3, personalBestLap: false }),
    ]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows[0].find('.best-lap-badge').exists()).toBe(false)
    expect(rows[1].find('.best-lap-badge').exists()).toBe(true)
    expect(rows[1].find('.best-lap-badge').text()).toBe('PB')
    expect(rows[2].find('.best-lap-badge').exists()).toBe(false)
  })

  it('applies best-lap class to personal best lap row', () => {
    const laps = [
      createLap({ lapNumber: 1, personalBestLap: false }),
      createLap({ lapNumber: 2, personalBestLap: true }),
    ]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows[0].classes()).not.toContain('best-lap')
    expect(rows[1].classes()).toContain('best-lap')
  })

  it('shows off track event with warning styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['off track'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badge = wrapper.find('.event-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('off track')
    expect(badge.classes()).toContain('event-warning')
  })

  it('shows contact event with danger styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['contact'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badge = wrapper.find('.event-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('contact')
    expect(badge.classes()).toContain('event-danger')
  })

  it('shows car contact event with danger styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['car contact'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badge = wrapper.find('.event-badge')
    expect(badge.text()).toBe('car contact')
    expect(badge.classes()).toContain('event-danger')
  })

  it('shows lost control event with danger styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['lost control'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badge = wrapper.find('.event-badge')
    expect(badge.text()).toBe('lost control')
    expect(badge.classes()).toContain('event-danger')
  })

  it('shows black flag event with info styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['black flag'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badge = wrapper.find('.event-badge')
    expect(badge.text()).toBe('black flag')
    expect(badge.classes()).toContain('event-info')
  })

  it('shows multiple event badges on same lap', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['off track', 'contact', 'tow'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badges = wrapper.findAll('.event-badge')
    expect(badges).toHaveLength(3)
    expect(badges[0].text()).toBe('off track')
    expect(badges[0].classes()).toContain('event-warning')
    expect(badges[1].text()).toBe('contact')
    expect(badges[1].classes()).toContain('event-danger')
    expect(badges[2].text()).toBe('tow')
    expect(badges[2].classes()).toContain('event-info')
  })

  it('filters out non-incident events like pitted and invalid', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['pitted', 'invalid', 'discontinuity', 'off track'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badges = wrapper.findAll('.event-badge')
    expect(badges).toHaveLength(1)
    expect(badges[0].text()).toBe('off track')
  })

  it('applies incident-contact class to row with contact events', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['contact'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const row = wrapper.find('tbody tr')
    expect(row.classes()).toContain('incident-contact')
  })

  it('applies incident-off-track class to row with off track events', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['off track'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const row = wrapper.find('tbody tr')
    expect(row.classes()).toContain('incident-off-track')
  })

  it('contact takes priority over off track for row styling', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['off track', 'contact'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const row = wrapper.find('tbody tr')
    expect(row.classes()).toContain('incident-contact')
    expect(row.classes()).not.toContain('incident-off-track')
  })

  it('personal best takes priority over incidents for row styling', () => {
    const laps = [createLap({ lapNumber: 1, personalBestLap: true, lapEvents: ['contact'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const row = wrapper.find('tbody tr')
    expect(row.classes()).toContain('best-lap')
    expect(row.classes()).not.toContain('incident-contact')
  })

  it('emits remove event when close button is clicked', async () => {
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData(),
      },
    })

    await wrapper.find('.lap-card-close').trigger('click')

    expect(wrapper.emitted('remove')).toHaveLength(1)
  })

  it('applies drag-over class when isDragOver is true', () => {
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData(),
        isDragOver: true,
      },
    })

    expect(wrapper.find('.lap-card').classes()).toContain('drag-over')
  })

  it('does not apply drag-over class when isDragOver is false', () => {
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData(),
        isDragOver: false,
      },
    })

    expect(wrapper.find('.lap-card').classes()).not.toContain('drag-over')
  })

  it('handles case-insensitive event matching', () => {
    const laps = [createLap({ lapNumber: 1, lapEvents: ['OFF TRACK', 'CONTACT'] })]
    const wrapper = mount(LapCard, {
      props: {
        driverName: 'Test Driver',
        finishPosition: 0,
        lapData: createLapData({ laps }),
      },
    })

    const badges = wrapper.findAll('.event-badge')
    expect(badges).toHaveLength(2)
    // Row should have contact styling (case insensitive)
    expect(wrapper.find('tbody tr').classes()).toContain('incident-contact')
  })
})