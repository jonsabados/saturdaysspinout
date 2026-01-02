import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SentimentBadge from './SentimentBadge.vue'
import { i18n } from '@/i18n'

describe('SentimentBadge', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountComponent(sentiment: 'good' | 'neutral' | 'bad' | null, size?: 'sm' | 'md' | 'lg') {
    return mount(SentimentBadge, {
      props: { sentiment, size },
      global: {
        plugins: [i18n],
      },
    })
  }

  it('renders nothing when sentiment is null', () => {
    const wrapper = mountComponent(null)
    expect(wrapper.find('.sentiment-badge').exists()).toBe(false)
  })

  it('renders good sentiment with + icon', () => {
    const wrapper = mountComponent('good')

    const badge = wrapper.find('.sentiment-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('sentiment-good')
    expect(wrapper.find('.sentiment-icon').text()).toBe('+')
    expect(wrapper.find('.sentiment-text').text()).toBe('Good')
  })

  it('renders neutral sentiment with = icon', () => {
    const wrapper = mountComponent('neutral')

    expect(wrapper.find('.sentiment-badge').classes()).toContain('sentiment-neutral')
    expect(wrapper.find('.sentiment-icon').text()).toBe('=')
    expect(wrapper.find('.sentiment-text').text()).toBe('Okay')
  })

  it('renders bad sentiment with - icon', () => {
    const wrapper = mountComponent('bad')

    expect(wrapper.find('.sentiment-badge').classes()).toContain('sentiment-bad')
    expect(wrapper.find('.sentiment-icon').text()).toBe('-')
    expect(wrapper.find('.sentiment-text').text()).toBe('Rough')
  })

  it('applies size-sm class when size is sm', () => {
    const wrapper = mountComponent('good', 'sm')
    expect(wrapper.find('.sentiment-badge').classes()).toContain('size-sm')
  })

  it('applies size-md class by default', () => {
    const wrapper = mountComponent('good')
    expect(wrapper.find('.sentiment-badge').classes()).toContain('size-md')
  })

  it('applies size-lg class when size is lg', () => {
    const wrapper = mountComponent('good', 'lg')
    expect(wrapper.find('.sentiment-badge').classes()).toContain('size-lg')
  })
})