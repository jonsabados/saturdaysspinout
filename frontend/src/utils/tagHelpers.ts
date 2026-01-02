export type Sentiment = 'good' | 'neutral' | 'bad'

/**
 * Extract sentiment value from a tags array.
 * Looks for tags in the format "sentiment:good", "sentiment:neutral", or "sentiment:bad".
 */
export function getSentiment(tags: string[]): Sentiment | null {
  const sentimentTag = tags.find((t) => t.startsWith('sentiment:'))
  if (!sentimentTag) return null
  const value = sentimentTag.split(':')[1]
  if (value === 'good' || value === 'neutral' || value === 'bad') {
    return value
  }
  return null
}

/**
 * Set or replace the sentiment tag in a tags array.
 * Removes any existing sentiment tag and adds the new one (if provided).
 * Returns a new array, does not mutate the input.
 */
export function setSentiment(tags: string[], sentiment: Sentiment | null): string[] {
  const filtered = tags.filter((t) => !t.startsWith('sentiment:'))
  if (sentiment) {
    filtered.push(`sentiment:${sentiment}`)
  }
  return filtered
}

/**
 * Check if any tag with the given prefix exists in the tags array.
 * For example, hasTagPrefix(tags, 'focus') returns true if tags contains 'focus:braking'.
 */
export function hasTagPrefix(tags: string[], prefix: string): boolean {
  return tags.some((t) => t.startsWith(`${prefix}:`))
}

/**
 * Get all values for a given tag prefix.
 * For example, getTagValues(['focus:braking', 'focus:racecraft', 'podium'], 'focus')
 * returns ['braking', 'racecraft'].
 */
export function getTagValues(tags: string[], prefix: string): string[] {
  return tags.filter((t) => t.startsWith(`${prefix}:`)).map((t) => t.split(':')[1])
}

/**
 * Get all plain tags (tags without a colon/prefix).
 * For example, getPlainTags(['sentiment:good', 'podium', 'clean-race'])
 * returns ['podium', 'clean-race'].
 */
export function getPlainTags(tags: string[]): string[] {
  return tags.filter((t) => !t.includes(':'))
}

/**
 * Add a tag to the array if it doesn't already exist.
 * Returns a new array, does not mutate the input.
 */
export function addTag(tags: string[], tag: string): string[] {
  if (tags.includes(tag)) return tags
  return [...tags, tag]
}

/**
 * Remove a tag from the array.
 * Returns a new array, does not mutate the input.
 */
export function removeTag(tags: string[], tag: string): string[] {
  return tags.filter((t) => t !== tag)
}

/**
 * Toggle a plain tag (add if absent, remove if present).
 * Returns a new array, does not mutate the input.
 */
export function toggleTag(tags: string[], tag: string): string[] {
  if (tags.includes(tag)) {
    return removeTag(tags, tag)
  }
  return addTag(tags, tag)
}