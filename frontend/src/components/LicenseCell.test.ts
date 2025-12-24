import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LicenseCell from './LicenseCell.vue'

describe('LicenseCell', () => {
  describe('license level decoding', () => {
    // New encoding: license_level 1-4=R, 5-8=D, 9-12=C, 13-16=B, 17-20=A
    // sub_level is SR * 100 (e.g., 381 = 3.81)

    it('displays Rookie license for levels 1-4', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 2, newLicenseLevel: 3, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.license-class').text()).toBe('R')
      expect(wrapper.find('.license-badge').classes()).toContain('license-r')
    })

    it('displays D license for levels 5-8', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 5, newLicenseLevel: 6, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.license-class').text()).toBe('D')
      expect(wrapper.find('.license-badge').classes()).toContain('license-d')
    })

    it('displays C license for levels 9-12', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 9, newLicenseLevel: 10, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.license-class').text()).toBe('C')
      expect(wrapper.find('.license-badge').classes()).toContain('license-c')
    })

    it('displays B license for levels 13-16', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 13, newLicenseLevel: 14, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.license-class').text()).toBe('B')
      expect(wrapper.find('.license-badge').classes()).toContain('license-b')
    })

    it('displays A license for levels 17-20', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 18, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.license-class').text()).toBe('A')
      expect(wrapper.find('.license-badge').classes()).toContain('license-a')
    })

    it('calculates safety rating from sub_level', () => {
      // sub_level 381 = SR 3.81
      const wrapper1 = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 17, oldSubLevel: 381, newSubLevel: 381, oldCpi: 10, newCpi: 10 },
      })
      expect(wrapper1.find('.license-sr').text()).toBe('3.81')

      // sub_level 47 = SR 0.47
      const wrapper2 = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 17, oldSubLevel: 47, newSubLevel: 47, oldCpi: 10, newCpi: 10 },
      })
      expect(wrapper2.find('.license-sr').text()).toBe('0.47')
    })
  })

  describe('promotion indicator', () => {
    it('shows promotion indicator when license class increases', () => {
      // Level 4 (R) -> Level 5 (D)
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 4, newLicenseLevel: 5, oldSubLevel: 399, newSubLevel: 100, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.promotion-indicator').exists()).toBe(true)
    })

    it('does not show promotion indicator when staying in same class', () => {
      // Level 5 (D) -> Level 6 (D)
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 5, newLicenseLevel: 6, oldSubLevel: 247, newSubLevel: 312, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.promotion-indicator').exists()).toBe(false)
    })

    it('does not show promotion indicator when demoted', () => {
      // Level 5 (D) -> Level 4 (R)
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 5, newLicenseLevel: 4, oldSubLevel: 150, newSubLevel: 399, oldCpi: 10, newCpi: 8 },
      })

      expect(wrapper.find('.promotion-indicator').exists()).toBe(false)
    })
  })

  describe('SR delta display', () => {
    it('shows positive SR delta with gain styling', () => {
      // SR 2.47 -> SR 3.23 (sub_level 247 -> 323)
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 13, newLicenseLevel: 13, oldSubLevel: 247, newSubLevel: 323, oldCpi: 10, newCpi: 12 },
      })

      expect(wrapper.find('.sr-delta').text()).toBe('(+0.76)')
      expect(wrapper.find('.sr-delta').classes()).toContain('stat-gain')
    })

    it('shows negative SR delta with loss styling', () => {
      // SR 3.23 -> SR 2.47 (sub_level 323 -> 247)
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 13, newLicenseLevel: 13, oldSubLevel: 323, newSubLevel: 247, oldCpi: 10, newCpi: 8 },
      })

      expect(wrapper.find('.sr-delta').text()).toBe('(-0.76)')
      expect(wrapper.find('.sr-delta').classes()).toContain('stat-loss')
    })

    it('shows zero SR delta without gain/loss styling', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 13, newLicenseLevel: 13, oldSubLevel: 323, newSubLevel: 323, oldCpi: 10, newCpi: 10 },
      })

      expect(wrapper.find('.sr-delta').text()).toBe('(+0.00)')
      expect(wrapper.find('.sr-delta').classes()).not.toContain('stat-gain')
      expect(wrapper.find('.sr-delta').classes()).not.toContain('stat-loss')
    })
  })

  describe('CPI tooltip', () => {
    it('shows CPI values in tooltip with positive delta', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 18, oldSubLevel: 381, newSubLevel: 399, oldCpi: 10.5, newCpi: 12.75 },
      })

      expect(wrapper.attributes('title')).toBe('CPI: 10.50 → 12.75 (+2.25)')
    })

    it('shows CPI values in tooltip with negative delta', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 17, oldSubLevel: 381, newSubLevel: 350, oldCpi: 15.0, newCpi: 12.5 },
      })

      expect(wrapper.attributes('title')).toBe('CPI: 15.00 → 12.50 (-2.50)')
    })

    it('shows CPI values in tooltip with zero delta', () => {
      const wrapper = mount(LicenseCell, {
        props: { oldLicenseLevel: 17, newLicenseLevel: 17, oldSubLevel: 381, newSubLevel: 381, oldCpi: 10.0, newCpi: 10.0 },
      })

      expect(wrapper.attributes('title')).toBe('CPI: 10.00 → 10.00 (+0.00)')
    })
  })
})