<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Sentiment } from '@/utils/tagHelpers'

defineProps<{
  modelValue: Sentiment | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Sentiment | null]
}>()

const { t } = useI18n()

function select(sentiment: Sentiment) {
  emit('update:modelValue', sentiment)
}
</script>

<template>
  <div class="sentiment-selector" role="radiogroup" :aria-label="t('journal.sentiment.aria')">
    <button
      type="button"
      class="sentiment-btn sentiment-good"
      :class="{ selected: modelValue === 'good' }"
      :disabled="disabled"
      :aria-checked="modelValue === 'good'"
      role="radio"
      @click="select('good')"
    >
      <span class="sentiment-icon">+</span>
      <span class="sentiment-label">{{ t('journal.sentiment.good') }}</span>
    </button>
    <button
      type="button"
      class="sentiment-btn sentiment-neutral"
      :class="{ selected: modelValue === 'neutral' }"
      :disabled="disabled"
      :aria-checked="modelValue === 'neutral'"
      role="radio"
      @click="select('neutral')"
    >
      <span class="sentiment-icon">=</span>
      <span class="sentiment-label">{{ t('journal.sentiment.neutral') }}</span>
    </button>
    <button
      type="button"
      class="sentiment-btn sentiment-bad"
      :class="{ selected: modelValue === 'bad' }"
      :disabled="disabled"
      :aria-checked="modelValue === 'bad'"
      role="radio"
      @click="select('bad')"
    >
      <span class="sentiment-icon">-</span>
      <span class="sentiment-label">{{ t('journal.sentiment.bad') }}</span>
    </button>
  </div>
</template>

<style scoped>
.sentiment-selector {
  display: flex;
  gap: 0.5rem;
}

.sentiment-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.75rem 1rem;
  border: 2px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-surface);
  cursor: pointer;
  transition:
    border-color 0.15s,
    background-color 0.15s,
    transform 0.1s;
  min-width: 70px;
}

.sentiment-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.sentiment-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sentiment-icon {
  font-size: 1.25rem;
  font-weight: bold;
  line-height: 1;
}

.sentiment-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Good sentiment */
.sentiment-good {
  --sentiment-color: #22c55e;
}

.sentiment-good:hover:not(:disabled) {
  border-color: var(--sentiment-color);
  background: rgba(34, 197, 94, 0.1);
}

.sentiment-good.selected {
  border-color: var(--sentiment-color);
  background: rgba(34, 197, 94, 0.15);
}

.sentiment-good .sentiment-icon {
  color: var(--sentiment-color);
}

/* Neutral sentiment */
.sentiment-neutral {
  --sentiment-color: #eab308;
}

.sentiment-neutral:hover:not(:disabled) {
  border-color: var(--sentiment-color);
  background: rgba(234, 179, 8, 0.1);
}

.sentiment-neutral.selected {
  border-color: var(--sentiment-color);
  background: rgba(234, 179, 8, 0.15);
}

.sentiment-neutral .sentiment-icon {
  color: var(--sentiment-color);
}

/* Bad sentiment */
.sentiment-bad {
  --sentiment-color: #ef4444;
}

.sentiment-bad:hover:not(:disabled) {
  border-color: var(--sentiment-color);
  background: rgba(239, 68, 68, 0.1);
}

.sentiment-bad.selected {
  border-color: var(--sentiment-color);
  background: rgba(239, 68, 68, 0.15);
}

.sentiment-bad .sentiment-icon {
  color: var(--sentiment-color);
}
</style>