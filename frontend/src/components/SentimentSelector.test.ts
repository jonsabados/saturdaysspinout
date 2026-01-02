import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SentimentSelector from './SentimentSelector.vue'
import { i18n } from '@/i18n'

describe('SentimentSelector', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountComponent(props: { modelValue: 'good' | 'neutral' | 'bad' | null; disabled?: boolean }) {
    return mount(SentimentSelector, {
      props,
      global: {
        plugins: [i18n],
      },
    })
  }

  it('renders three sentiment buttons', () => {
    const wrapper = mountComponent({ modelValue: null })

    const buttons = wrapper.findAll('.sentiment-btn')
    expect(buttons).toHaveLength(3)
  })

  it('displays translated labels', () => {
    const wrapper = mountComponent({ modelValue: null })

    const labels = wrapper.findAll('.sentiment-label')
    expect(labels[0].text()).toBe('Good')
    expect(labels[1].text()).toBe('Okay')
    expect(labels[2].text()).toBe('Rough')
  })

  it('marks good button as selected when modelValue is good', () => {
    const wrapper = mountComponent({ modelValue: 'good' })

    const goodBtn = wrapper.find('.sentiment-good')
    expect(goodBtn.classes()).toContain('selected')
    expect(goodBtn.attributes('aria-checked')).toBe('true')

    const neutralBtn = wrapper.find('.sentiment-neutral')
    expect(neutralBtn.classes()).not.toContain('selected')

    const badBtn = wrapper.find('.sentiment-bad')
    expect(badBtn.classes()).not.toContain('selected')
  })

  it('marks neutral button as selected when modelValue is neutral', () => {
    const wrapper = mountComponent({ modelValue: 'neutral' })

    const neutralBtn = wrapper.find('.sentiment-neutral')
    expect(neutralBtn.classes()).toContain('selected')
    expect(neutralBtn.attributes('aria-checked')).toBe('true')
  })

  it('marks bad button as selected when modelValue is bad', () => {
    const wrapper = mountComponent({ modelValue: 'bad' })

    const badBtn = wrapper.find('.sentiment-bad')
    expect(badBtn.classes()).toContain('selected')
    expect(badBtn.attributes('aria-checked')).toBe('true')
  })

  it('emits update:modelValue when good button is clicked', async () => {
    const wrapper = mountComponent({ modelValue: null })

    await wrapper.find('.sentiment-good').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([['good']])
  })

  it('emits update:modelValue when neutral button is clicked', async () => {
    const wrapper = mountComponent({ modelValue: null })

    await wrapper.find('.sentiment-neutral').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([['neutral']])
  })

  it('emits update:modelValue when bad button is clicked', async () => {
    const wrapper = mountComponent({ modelValue: null })

    await wrapper.find('.sentiment-bad').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([['bad']])
  })

  it('disables all buttons when disabled prop is true', () => {
    const wrapper = mountComponent({ modelValue: null, disabled: true })

    const buttons = wrapper.findAll('.sentiment-btn')
    buttons.forEach((btn) => {
      expect(btn.attributes('disabled')).toBeDefined()
    })
  })

  it('does not emit when clicking disabled button', async () => {
    const wrapper = mountComponent({ modelValue: null, disabled: true })

    await wrapper.find('.sentiment-good').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('has correct aria-label on the container', () => {
    const wrapper = mountComponent({ modelValue: null })

    const container = wrapper.find('.sentiment-selector')
    expect(container.attributes('role')).toBe('radiogroup')
    expect(container.attributes('aria-label')).toBe('How did this race feel?')
  })
})