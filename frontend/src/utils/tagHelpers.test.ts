import { describe, it, expect } from 'vitest'
import {
  getSentiment,
  setSentiment,
  hasTagPrefix,
  getTagValues,
  getPlainTags,
  addTag,
  removeTag,
  toggleTag
} from './tagHelpers'

describe('getSentiment', () => {
  it('returns null for empty array', () => {
    expect(getSentiment([])).toBe(null)
  })

  it('returns null when no sentiment tag exists', () => {
    expect(getSentiment(['podium', 'clean-race'])).toBe(null)
  })

  it('extracts good sentiment', () => {
    expect(getSentiment(['sentiment:good', 'podium'])).toBe('good')
  })

  it('extracts neutral sentiment', () => {
    expect(getSentiment(['sentiment:neutral'])).toBe('neutral')
  })

  it('extracts bad sentiment', () => {
    expect(getSentiment(['podium', 'sentiment:bad', 'clean-race'])).toBe('bad')
  })

  it('returns null for invalid sentiment value', () => {
    expect(getSentiment(['sentiment:invalid'])).toBe(null)
  })
})

describe('setSentiment', () => {
  it('adds sentiment to empty array', () => {
    expect(setSentiment([], 'good')).toEqual(['sentiment:good'])
  })

  it('adds sentiment to array with other tags', () => {
    const result = setSentiment(['podium', 'clean-race'], 'neutral')
    expect(result).toContain('sentiment:neutral')
    expect(result).toContain('podium')
    expect(result).toContain('clean-race')
  })

  it('replaces existing sentiment', () => {
    const result = setSentiment(['sentiment:bad', 'podium'], 'good')
    expect(result).toContain('sentiment:good')
    expect(result).not.toContain('sentiment:bad')
    expect(result).toContain('podium')
  })

  it('removes sentiment when null is passed', () => {
    const result = setSentiment(['sentiment:good', 'podium'], null)
    expect(result).toEqual(['podium'])
  })

  it('does not mutate original array', () => {
    const original = ['sentiment:bad', 'podium']
    setSentiment(original, 'good')
    expect(original).toEqual(['sentiment:bad', 'podium'])
  })
})

describe('hasTagPrefix', () => {
  it('returns false for empty array', () => {
    expect(hasTagPrefix([], 'focus')).toBe(false)
  })

  it('returns false when prefix not present', () => {
    expect(hasTagPrefix(['podium', 'sentiment:good'], 'focus')).toBe(false)
  })

  it('returns true when prefix is present', () => {
    expect(hasTagPrefix(['focus:braking', 'podium'], 'focus')).toBe(true)
  })

  it('returns true with multiple values for same prefix', () => {
    expect(hasTagPrefix(['focus:braking', 'focus:racecraft'], 'focus')).toBe(true)
  })
})

describe('getTagValues', () => {
  it('returns empty array for empty tags', () => {
    expect(getTagValues([], 'focus')).toEqual([])
  })

  it('returns empty array when prefix not present', () => {
    expect(getTagValues(['podium', 'sentiment:good'], 'focus')).toEqual([])
  })

  it('returns single value', () => {
    expect(getTagValues(['focus:braking', 'podium'], 'focus')).toEqual(['braking'])
  })

  it('returns multiple values for same prefix', () => {
    const result = getTagValues(['focus:braking', 'focus:racecraft', 'podium'], 'focus')
    expect(result).toEqual(['braking', 'racecraft'])
  })
})

describe('getPlainTags', () => {
  it('returns empty array for empty tags', () => {
    expect(getPlainTags([])).toEqual([])
  })

  it('returns empty array when all tags have prefixes', () => {
    expect(getPlainTags(['sentiment:good', 'focus:braking'])).toEqual([])
  })

  it('returns only plain tags', () => {
    expect(getPlainTags(['podium', 'sentiment:good', 'clean-race'])).toEqual([
      'podium',
      'clean-race'
    ])
  })
})

describe('addTag', () => {
  it('adds tag to empty array', () => {
    expect(addTag([], 'podium')).toEqual(['podium'])
  })

  it('adds tag to existing array', () => {
    expect(addTag(['podium'], 'clean-race')).toEqual(['podium', 'clean-race'])
  })

  it('does not add duplicate tag', () => {
    expect(addTag(['podium', 'clean-race'], 'podium')).toEqual(['podium', 'clean-race'])
  })

  it('does not mutate original array', () => {
    const original = ['podium']
    addTag(original, 'clean-race')
    expect(original).toEqual(['podium'])
  })
})

describe('removeTag', () => {
  it('returns empty array when removing from empty array', () => {
    expect(removeTag([], 'podium')).toEqual([])
  })

  it('removes existing tag', () => {
    expect(removeTag(['podium', 'clean-race'], 'podium')).toEqual(['clean-race'])
  })

  it('returns same elements when tag not present', () => {
    expect(removeTag(['podium'], 'clean-race')).toEqual(['podium'])
  })

  it('does not mutate original array', () => {
    const original = ['podium', 'clean-race']
    removeTag(original, 'podium')
    expect(original).toEqual(['podium', 'clean-race'])
  })
})

describe('toggleTag', () => {
  it('adds tag when not present', () => {
    expect(toggleTag(['podium'], 'clean-race')).toEqual(['podium', 'clean-race'])
  })

  it('removes tag when present', () => {
    expect(toggleTag(['podium', 'clean-race'], 'podium')).toEqual(['clean-race'])
  })

  it('works on empty array', () => {
    expect(toggleTag([], 'podium')).toEqual(['podium'])
  })

  it('does not mutate original array', () => {
    const original = ['podium']
    toggleTag(original, 'podium')
    expect(original).toEqual(['podium'])
  })
})