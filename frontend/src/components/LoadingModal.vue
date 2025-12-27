<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'

const { t } = useI18n()
const sessionStore = useSessionStore()

const shouldShow = computed(() => sessionStore.isLoggedIn && !sessionStore.isReady)

const labelKeys: Record<string, string> = {
  tracks: 'loading.loadingTracks',
  cars: 'loading.loadingCars',
}

function getLabel(id: string): string {
  const key = labelKeys[id]
  return key ? t(key) : id
}
</script>

<template>
  <Teleport to="body">
    <div v-if="shouldShow" class="loading-backdrop">
      <div class="loading-modal" role="dialog" aria-modal="true" aria-label="Loading">
        <h2 class="loading-title">{{ t('loading.title') }}</h2>
        <ul class="loading-list">
          <li
            v-for="state in sessionStore.loadingStates"
            :key="state.id"
            :class="['loading-item', { done: state.done }]"
          >
            <span class="loading-icon">
              <span v-if="state.done" class="checkmark">&#10003;</span>
              <span v-else class="spinner"></span>
            </span>
            <span class="loading-label">{{ getLabel(state.id) }}</span>
          </li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.loading-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.loading-modal {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1.5rem 2rem;
  min-width: 280px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
}

.loading-title {
  margin: 0 0 1.25rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text-primary);
  text-align: center;
}

.loading-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.loading-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--color-text-secondary);
  transition: color 0.2s;
}

.loading-item.done {
  color: var(--color-text-primary);
}

.loading-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.checkmark {
  color: var(--color-accent);
  font-size: 1rem;
  font-weight: bold;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.loading-label {
  font-size: 0.9375rem;
}
</style>