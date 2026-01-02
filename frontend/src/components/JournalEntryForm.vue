<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SentimentSelector from './SentimentSelector.vue'
import { getSentiment, setSentiment, type Sentiment } from '@/utils/tagHelpers'

const props = defineProps<{
  initialNotes?: string
  initialTags?: string[]
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [data: { notes: string; tags: string[] }]
  cancel: []
}>()

const { t } = useI18n()

const notes = ref(props.initialNotes ?? '')
const sentiment = ref<Sentiment | null>(null)

// Initialize sentiment from tags
watch(
  () => props.initialTags,
  (tags) => {
    sentiment.value = tags ? getSentiment(tags) : null
  },
  { immediate: true }
)

const canSave = computed(() => {
  return notes.value.trim().length > 0 || sentiment.value !== null
})

function handleSave() {
  if (!canSave.value || props.saving) return

  const tags = setSentiment(props.initialTags ?? [], sentiment.value)
  emit('save', {
    notes: notes.value.trim(),
    tags,
  })
}

function handleCancel() {
  emit('cancel')
}
</script>

<template>
  <form class="journal-form" @submit.prevent="handleSave">
    <div class="form-group">
      <label class="form-label">{{ t('journal.howDidItGo') }}</label>
      <SentimentSelector v-model="sentiment" :disabled="saving" />
    </div>

    <div class="form-group">
      <label for="journal-notes" class="form-label">{{ t('journal.notes.label') }}</label>
      <textarea
        id="journal-notes"
        v-model="notes"
        class="form-textarea"
        :placeholder="t('journal.notes.placeholder')"
        :disabled="saving"
        rows="4"
      />
    </div>

    <div class="form-actions">
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="handleCancel">
        {{ t('journal.actions.cancel') }}
      </button>
      <button type="submit" class="btn btn-primary" :disabled="!canSave || saving">
        {{ saving ? t('journal.saving') : t('journal.actions.save') }}
      </button>
    </div>
  </form>
</template>

<style scoped>
.journal-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.form-textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-surface);
  color: var(--color-text-primary);
  font-family: inherit;
  font-size: 0.9375rem;
  line-height: 1.5;
  resize: vertical;
  min-height: 100px;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-subtle);
}

.form-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-textarea::placeholder {
  color: var(--color-text-muted);
}

.form-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
}
</style>