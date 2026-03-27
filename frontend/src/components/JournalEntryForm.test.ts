import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import JournalEntryForm from './JournalEntryForm.vue'
import SentimentSelector from './SentimentSelector.vue'
import { i18n } from '@/i18n'

describe('JournalEntryForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountComponent(props: {
    initialNotes?: string
    initialTags?: string[]
    initialReplayVideo?: string
    saving?: boolean
    fieldErrors?: Record<string, string>
  } = {}) {
    return mount(JournalEntryForm, {
      props,
      global: {
        plugins: [i18n],
      },
    })
  }

  it('renders form with sentiment selector, textarea, and replay video input', () => {
    const wrapper = mountComponent()

    expect(wrapper.findComponent(SentimentSelector).exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('input[type="url"]').exists()).toBe(true)
    expect(wrapper.find('.btn-primary').exists()).toBe(true)
    expect(wrapper.find('.btn-secondary').exists()).toBe(true)
  })

  it('initializes with empty values by default', () => {
    const wrapper = mountComponent()

    const textarea = wrapper.find('textarea')
    expect((textarea.element as HTMLTextAreaElement).value).toBe('')

    const urlInput = wrapper.find('input[type="url"]')
    expect((urlInput.element as HTMLInputElement).value).toBe('')

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    expect(sentimentSelector.props('modelValue')).toBeNull()
  })

  it('initializes with provided notes', () => {
    const wrapper = mountComponent({ initialNotes: 'Great race!' })

    const textarea = wrapper.find('textarea')
    expect((textarea.element as HTMLTextAreaElement).value).toBe('Great race!')
  })

  it('initializes sentiment from tags', () => {
    const wrapper = mountComponent({ initialTags: ['sentiment:good', 'podium'] })

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    expect(sentimentSelector.props('modelValue')).toBe('good')
  })

  it('disables save button when form is empty', () => {
    const wrapper = mountComponent()

    const saveBtn = wrapper.find('.btn-primary')
    expect(saveBtn.attributes('disabled')).toBeDefined()
  })

  it('enables save button when notes are entered', async () => {
    const wrapper = mountComponent()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('Some notes')

    const saveBtn = wrapper.find('.btn-primary')
    expect(saveBtn.attributes('disabled')).toBeUndefined()
  })

  it('enables save button when sentiment is selected', async () => {
    const wrapper = mountComponent()

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    await sentimentSelector.vm.$emit('update:modelValue', 'good')

    const saveBtn = wrapper.find('.btn-primary')
    expect(saveBtn.attributes('disabled')).toBeUndefined()
  })

  it('emits save event with notes, tags, and replayVideo when form is submitted', async () => {
    const wrapper = mountComponent()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('Great race!')

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    await sentimentSelector.vm.$emit('update:modelValue', 'good')

    const urlInput = wrapper.find('input[type="url"]')
    await urlInput.setValue('https://www.youtube.com/watch?v=dQw4w9WgXcQ')

    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('save')).toEqual([
      [{ notes: 'Great race!', tags: ['sentiment:good'], replayVideo: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ' }],
    ])
  })

  it('preserves existing non-sentiment tags when saving', async () => {
    const wrapper = mountComponent({
      initialNotes: 'Original notes',
      initialTags: ['podium', 'sentiment:neutral', 'clean-race'],
    })

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    await sentimentSelector.vm.$emit('update:modelValue', 'good')

    await wrapper.find('form').trigger('submit')

    const emitted = wrapper.emitted('save')
    expect(emitted).toBeDefined()
    const savedData = emitted![0][0] as { notes: string; tags: string[]; replayVideo: string }
    expect(savedData.tags).toContain('podium')
    expect(savedData.tags).toContain('clean-race')
    expect(savedData.tags).toContain('sentiment:good')
    expect(savedData.tags).not.toContain('sentiment:neutral')
  })

  it('trims notes and replayVideo before emitting save', async () => {
    const wrapper = mountComponent()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('  Trimmed notes  ')

    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('save')).toEqual([
      [{ notes: 'Trimmed notes', tags: [], replayVideo: '' }],
    ])
  })

  it('initializes replayVideo from prop', () => {
    const wrapper = mountComponent({ initialReplayVideo: 'https://youtu.be/abc123' })

    const urlInput = wrapper.find('input[type="url"]')
    expect((urlInput.element as HTMLInputElement).value).toBe('https://youtu.be/abc123')
  })

  it('enables save button when only replayVideo is set', async () => {
    const wrapper = mountComponent()

    const urlInput = wrapper.find('input[type="url"]')
    await urlInput.setValue('https://youtu.be/abc123')

    const saveBtn = wrapper.find('.btn-primary')
    expect(saveBtn.attributes('disabled')).toBeUndefined()
  })

  it('displays field error for replayVideo when provided', () => {
    const wrapper = mountComponent({ fieldErrors: { replayVideo: 'Please enter a valid URL (e.g. https://...)' } })

    const errorMsg = wrapper.find('.field-error')
    expect(errorMsg.exists()).toBe(true)
    expect(errorMsg.text()).toBe('Please enter a valid URL (e.g. https://...)')

    const urlInput = wrapper.find('input[type="url"]')
    expect(urlInput.classes()).toContain('form-input-error')
  })

  it('does not display field error when fieldErrors is empty', () => {
    const wrapper = mountComponent({ fieldErrors: {} })

    expect(wrapper.find('.field-error').exists()).toBe(false)
  })

  it('emits cancel event when cancel button is clicked', async () => {
    const wrapper = mountComponent()

    await wrapper.find('.btn-secondary').trigger('click')

    expect(wrapper.emitted('cancel')).toEqual([[]])
  })

  it('disables form elements when saving', () => {
    const wrapper = mountComponent({ saving: true })

    expect(wrapper.find('textarea').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.btn-primary').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.btn-secondary').attributes('disabled')).toBeDefined()

    const sentimentSelector = wrapper.findComponent(SentimentSelector)
    expect(sentimentSelector.props('disabled')).toBe(true)
  })

  it('shows saving text on save button when saving', () => {
    const wrapper = mountComponent({ saving: true, initialNotes: 'test' })

    const saveBtn = wrapper.find('.btn-primary')
    expect(saveBtn.text()).toBe('Saving...')
  })

  it('does not emit save when saving is in progress', async () => {
    const wrapper = mountComponent({ saving: true, initialNotes: 'test' })

    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('save')).toBeUndefined()
  })
})