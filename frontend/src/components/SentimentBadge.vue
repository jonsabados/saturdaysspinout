<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Sentiment } from '@/utils/tagHelpers'

const props = withDefaults(
  defineProps<{
    sentiment: Sentiment | null
    size?: 'sm' | 'md' | 'lg'
  }>(),
  {
    size: 'md',
  }
)

const { t } = useI18n()

const label = computed(() => {
  if (!props.sentiment) return null
  return t(`journal.sentiment.${props.sentiment}`)
})

const icon = computed(() => {
  if (!props.sentiment) return null
  return props.sentiment === 'good' ? '+' : props.sentiment === 'neutral' ? '=' : '-'
})
</script>

<template>
  <span
    v-if="sentiment"
    class="sentiment-badge"
    :class="[`sentiment-${sentiment}`, `size-${size}`]"
  >
    <span class="sentiment-icon">{{ icon }}</span>
    <span class="sentiment-text">{{ label }}</span>
  </span>
</template>

<style scoped>
.sentiment-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  border-radius: 9999px;
  font-weight: 500;
}

.size-sm {
  padding: 0.125rem 0.5rem;
  font-size: 0.75rem;
}

.size-md {
  padding: 0.25rem 0.75rem;
  font-size: 0.8125rem;
}

.size-lg {
  padding: 0.375rem 1rem;
  font-size: 0.875rem;
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
</style>