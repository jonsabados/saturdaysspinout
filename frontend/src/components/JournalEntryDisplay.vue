<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { JournalEntry } from '@/api/client'
import { getSentiment, type Sentiment } from '@/utils/tagHelpers'

const props = defineProps<{
  entry: JournalEntry
}>()

const emit = defineEmits<{
  edit: []
  delete: []
}>()

const { t, d } = useI18n()

const sentiment = computed<Sentiment | null>(() => {
  return getSentiment(props.entry.tags)
})

const sentimentLabel = computed(() => {
  if (!sentiment.value) return null
  return t(`journal.sentiment.${sentiment.value}`)
})

const updatedDate = computed(() => {
  return d(new Date(props.entry.updatedAt), 'short')
})

function handleEdit() {
  emit('edit')
}

function handleDelete() {
  if (confirm(t('journal.confirmDelete'))) {
    emit('delete')
  }
}
</script>

<template>
  <div class="journal-display">
    <div class="journal-header">
      <div v-if="sentiment" class="sentiment-badge" :class="`sentiment-${sentiment}`">
        <span class="sentiment-icon">
          {{ sentiment === 'good' ? '+' : sentiment === 'neutral' ? '=' : '-' }}
        </span>
        <span class="sentiment-text">{{ sentimentLabel }}</span>
      </div>
      <span class="journal-updated">{{ t('journal.updated', { date: updatedDate }) }}</span>
    </div>

    <p v-if="entry.notes" class="journal-notes">{{ entry.notes }}</p>

    <div class="journal-actions">
      <button type="button" class="btn btn-secondary btn-sm" @click="handleEdit">
        {{ t('journal.actions.edit') }}
      </button>
      <button type="button" class="btn btn-danger btn-sm" @click="handleDelete">
        {{ t('journal.actions.delete') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.journal-display {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.journal-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.sentiment-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.8125rem;
  font-weight: 500;
}

.sentiment-icon {
  font-weight: bold;
}

.sentiment-good {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.sentiment-neutral {
  background: rgba(234, 179, 8, 0.15);
  color: #ca8a04;
}

.sentiment-bad {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.journal-updated {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}

.journal-notes {
  margin: 0;
  line-height: 1.6;
  color: var(--color-text-primary);
  white-space: pre-wrap;
}

.journal-actions {
  display: flex;
  gap: 0.5rem;
}
</style>