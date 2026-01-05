<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Sentiment } from '@/utils/tagHelpers'

export interface JournalFiltersState {
  from: Date
  to: Date
  sentiment: Sentiment[] | null  // null = all, array = selected sentiments
  showDNFOnly: boolean
}

const props = defineProps<{
  modelValue: JournalFiltersState
}>()

const emit = defineEmits<{
  'update:modelValue': [value: JournalFiltersState]
}>()

const { t } = useI18n()

const sentimentOptions: Sentiment[] = ['good', 'neutral', 'bad']

// Format date for input[type="date"]
function formatDateForInput(date: Date): string {
  return date.toISOString().split('T')[0]
}

// Parse date from input[type="date"]
function parseDateFromInput(value: string): Date {
  return new Date(value + 'T00:00:00')
}

const fromDate = computed({
  get: () => formatDateForInput(props.modelValue.from),
  set: (value: string) => {
    emit('update:modelValue', {
      ...props.modelValue,
      from: parseDateFromInput(value),
    })
  },
})

const toDate = computed({
  get: () => formatDateForInput(props.modelValue.to),
  set: (value: string) => {
    emit('update:modelValue', {
      ...props.modelValue,
      to: parseDateFromInput(value),
    })
  },
})

function isSentimentSelected(sentiment: Sentiment): boolean {
  if (props.modelValue.sentiment === null) return false
  return props.modelValue.sentiment.includes(sentiment)
}

function toggleSentiment(sentiment: Sentiment) {
  const current = props.modelValue.sentiment

  if (current === null) {
    // Nothing selected, select just this one
    emit('update:modelValue', {
      ...props.modelValue,
      sentiment: [sentiment],
    })
  } else if (current.includes(sentiment)) {
    // Already selected, remove it
    const remaining = current.filter(s => s !== sentiment)
    emit('update:modelValue', {
      ...props.modelValue,
      sentiment: remaining.length === 0 ? null : remaining,
    })
  } else {
    // Not selected, add it
    emit('update:modelValue', {
      ...props.modelValue,
      sentiment: [...current, sentiment],
    })
  }
}

function toggleDNFOnly() {
  emit('update:modelValue', {
    ...props.modelValue,
    showDNFOnly: !props.modelValue.showDNFOnly,
  })
}

function getSentimentLabel(sentiment: Sentiment): string {
  return t(`journal.sentiment.${sentiment}`)
}
</script>

<template>
  <div class="filter-bar journal-filters">
    <div class="filter-group date-filters">
      <label class="filter-label">
        <span class="label-text">{{ t('journal.filters.from') }}</span>
        <input type="date" v-model="fromDate" class="date-input" />
      </label>
      <label class="filter-label">
        <span class="label-text">{{ t('journal.filters.to') }}</span>
        <input type="date" v-model="toDate" class="date-input" />
      </label>
    </div>

    <div class="filter-group sentiment-filters">
      <span class="filter-section-label">{{ t('journal.filters.sentiment') }}</span>
      <div class="sentiment-chips">
        <button
          v-for="sentiment in sentimentOptions"
          :key="sentiment"
          type="button"
          class="sentiment-chip"
          :class="[`sentiment-${sentiment}`, { selected: isSentimentSelected(sentiment) }]"
          @click="toggleSentiment(sentiment)"
        >
          {{ getSentimentLabel(sentiment) }}
        </button>
      </div>
    </div>

    <div class="filter-group">
      <label class="checkbox-label">
        <input
          type="checkbox"
          :checked="modelValue.showDNFOnly"
          @change="toggleDNFOnly"
          class="checkbox-input"
        />
        <span class="checkbox-text">{{ t('journal.filters.dnfOnly') }}</span>
      </label>
    </div>
  </div>
</template>

<style scoped>
.journal-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
  align-items: flex-end;
  padding: 1rem;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  margin-bottom: 1.5rem;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.date-filters {
  flex-direction: row;
  gap: 1rem;
}

.filter-label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.label-text,
.filter-section-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.date-input {
  padding: 0.5rem 0.75rem;
  background: var(--color-bg-deep);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-primary);
  font-size: 0.875rem;
  font-family: inherit;
}

.date-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.sentiment-filters {
  gap: 0.5rem;
}

.sentiment-chips {
  display: flex;
  gap: 0.5rem;
}

.sentiment-chip {
  padding: 0.375rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s;
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
}

.sentiment-chip:hover {
  background: var(--color-bg-hover);
}

.sentiment-chip.selected.sentiment-good {
  background: rgba(34, 197, 94, 0.15);
  border-color: rgba(34, 197, 94, 0.3);
  color: #22c55e;
}

.sentiment-chip.selected.sentiment-neutral {
  background: rgba(234, 179, 8, 0.15);
  border-color: rgba(234, 179, 8, 0.3);
  color: #ca8a04;
}

.sentiment-chip.selected.sentiment-bad {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-input {
  width: 1rem;
  height: 1rem;
  accent-color: var(--color-accent);
}

.checkbox-text {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

@media (max-width: 768px) {
  .journal-filters {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
}
</style>