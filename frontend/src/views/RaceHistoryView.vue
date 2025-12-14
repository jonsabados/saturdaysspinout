<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useApiClient } from '@/api/client'
import { useSessionStore } from '@/stores/session'

const apiClient = useApiClient()
const session = useSessionStore()
const status = ref<'idle' | 'loading' | 'success' | 'error'>('idle')
const errorMessage = ref('')

async function triggerIngestion() {
  if (!session.isReady) {
    return
  }

  status.value = 'loading'
  errorMessage.value = ''

  try {
    await apiClient.triggerRaceIngestion()
    status.value = 'success'
  } catch (err) {
    status.value = 'error'
    errorMessage.value = err instanceof Error ? err.message : 'Failed to trigger ingestion'
  }
}

onMounted(() => {
  if (session.isReady) {
    triggerIngestion()
  }
})

watch(() => session.isReady, (ready) => {
  if (ready && status.value === 'idle') {
    triggerIngestion()
  }
})
</script>

<template>
  <div class="race-history">
    <h1>Race History</h1>

    <div v-if="!session.isReady && status === 'idle'" class="status-message">
      Connecting...
    </div>

    <div v-else-if="status === 'loading'" class="status-message">
      Requesting race history ingestion...
    </div>

    <div v-else-if="status === 'success'" class="status-message success">
      Race history ingestion queued. Results will appear here once processing completes.
    </div>

    <div v-else-if="status === 'error'" class="status-message error">
      {{ errorMessage }}
    </div>
  </div>
</template>

<style scoped>
.race-history {
  padding: 2rem;
  max-width: 800px;
  margin: 0 auto;
}

h1 {
  margin-bottom: 1.5rem;
  color: var(--color-text-primary);
}

.status-message {
  padding: 1rem;
  border-radius: 4px;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
}

.status-message.success {
  border-color: var(--color-success, #22c55e);
  background: var(--color-success-subtle, rgba(34, 197, 94, 0.1));
}

.status-message.error {
  border-color: var(--color-error, #ef4444);
  background: var(--color-error-subtle, rgba(239, 68, 68, 0.1));
}
</style>
