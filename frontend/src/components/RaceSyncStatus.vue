<script setup lang="ts">
import { computed } from 'vue'
import { useRaceIngestionStore } from '@/stores/raceIngestion'
import { useSessionStore } from '@/stores/session'

const ingestionStore = useRaceIngestionStore()
const sessionStore = useSessionStore()

const isLoading = computed(() => ingestionStore.status === 'loading')
const isDisabled = computed(() => !sessionStore.isReady || isLoading.value)

function handleClick() {
  if (!isDisabled.value) {
    ingestionStore.triggerIngestion()
  }
}
</script>

<template>
  <button
    class="sync-button"
    :class="{ loading: isLoading, disabled: isDisabled }"
    :disabled="isDisabled"
    @click="handleClick"
    title="Sync race history"
  >
    <span class="sync-icon" :class="{ spinning: isLoading }">&#x21bb;</span>
    <span class="sync-label">Sync</span>
  </button>
</template>

<style scoped>
.sync-button {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
}

.sync-button:hover:not(.disabled) {
  color: var(--color-text-primary);
  border-color: var(--color-border-light);
  background: var(--color-accent-subtle);
}

.sync-button.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sync-icon {
  font-size: 1rem;
  line-height: 1;
}

.sync-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.sync-label {
  font-weight: 500;
}

/* Hide label on smaller screens, keep icon */
@media (max-width: 480px) {
  .sync-label {
    display: none;
  }

  .sync-button {
    padding: 0.375rem;
  }
}
</style>