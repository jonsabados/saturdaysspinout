import { describe, it, expect } from 'vitest'
import { formatLapTime, formatInterval } from './raceFormatters'

describe('formatLapTime', () => {
  it('returns dash for zero', () => {
    expect(formatLapTime(0)).toBe('-')
  })

  it('returns dash for negative values', () => {
    expect(formatLapTime(-1000)).toBe('-')
    expect(formatLapTime(-1)).toBe('-')
  })

  it('formats sub-minute times as seconds only', () => {
    // 30.123 seconds = 301230 in iRacing format
    expect(formatLapTime(301230)).toBe('30.123')
  })

  it('formats times under 10 seconds', () => {
    // 5.456 seconds = 54560
    expect(formatLapTime(54560)).toBe('5.456')
  })

  it('formats multi-minute times with colon separator', () => {
    // 1:30.456 = 90.456 seconds = 904560
    expect(formatLapTime(904560)).toBe('1:30.456')
  })

  it('pads seconds correctly for multi-minute times', () => {
    // 1:05.123 = 65.123 seconds = 651230
    expect(formatLapTime(651230)).toBe('1:05.123')
  })

  it('handles exactly one minute', () => {
    // 1:00.000 = 60 seconds = 600000
    expect(formatLapTime(600000)).toBe('1:00.000')
  })

  it('handles multi-minute times correctly', () => {
    // 2:15.789 = 135.789 seconds = 1357890
    expect(formatLapTime(1357890)).toBe('2:15.789')
  })

  it('handles very small positive values', () => {
    // 0.001 seconds = 10
    expect(formatLapTime(10)).toBe('0.001')
  })
})

describe('formatInterval', () => {
  describe('leader detection', () => {
    it('returns dash for leader', () => {
      expect(formatInterval(0, true, 50, 50)).toBe('-')
    })

    it('returns dash for leader even with interval value', () => {
      expect(formatInterval(100000, true, 50, 50)).toBe('-')
    })
  })

  describe('laps down detection', () => {
    it('returns dash when driver has fewer laps than leader', () => {
      expect(formatInterval(50000, false, 49, 50)).toBe('-')
    })

    it('returns dash for multiple laps down', () => {
      expect(formatInterval(50000, false, 45, 50)).toBe('-')
    })
  })

  describe('small interval handling', () => {
    it('returns dash for zero interval', () => {
      expect(formatInterval(0, false, 50, 50)).toBe('-')
    })

    it('returns dash for interval less than 1 (< 0.0001s)', () => {
      expect(formatInterval(0.5, false, 50, 50)).toBe('-')
    })

    it('shows interval of exactly 1', () => {
      expect(formatInterval(1, false, 50, 50)).toBe('+0.000')
    })
  })

  describe('valid intervals', () => {
    it('formats positive intervals with plus sign', () => {
      // 5.123 seconds = 51230
      expect(formatInterval(51230, false, 50, 50)).toBe('+5.123')
    })

    it('formats multi-second intervals', () => {
      // 12.456 seconds = 124560
      expect(formatInterval(124560, false, 50, 50)).toBe('+12.456')
    })

    it('formats multi-minute intervals', () => {
      // 1:30.000 = 90 seconds = 900000
      expect(formatInterval(900000, false, 50, 50)).toBe('+1:30.000')
    })
  })
})